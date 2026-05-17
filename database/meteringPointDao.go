package database

import (
	"database/sql"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/eegfaktura/eegfaktura-backend/model"
	log "github.com/sirupsen/logrus"
)

const TABLE_METERINGPOINT = "base.meteringpoint"
const TABLE_PARTITION_FACT = "base.metering_partition_factor"

type meteringEntryType struct {
	model.MeteringPoint
	Participant_id string
	Tenant         string
}

func RegisterMeteringPoints(tx *sql.Tx, tenant, participantId string, point []*model.MeteringPoint) error {
	meteringEntry := []meteringEntryType{}
	for _, p := range point {
		p.Status = model.NEW
		meteringEntry = append(meteringEntry, meteringEntryType{*p, participantId, tenant})
	}
	return saveMeteringPoint(tx, meteringEntry)
}

func ImportMeteringPoints(tx *sql.Tx, tenant, participantId string, point []*model.MeteringPoint) error {
	meteringEntry := []meteringEntryType{}
	for _, p := range point {
		meteringEntry = append(meteringEntry, meteringEntryType{*p, participantId, tenant})
	}
	return saveMeteringPoint(tx, meteringEntry)
}

func saveMeteringPoint(tx *sql.Tx, meteringEntry []meteringEntryType) error {
	statement, _, _ := pgDialect.Insert(TABLE_METERINGPOINT).Rows(meteringEntry).ToSQL()
	log.Debugf("Register Meterings: %+v", statement)
	_, err := tx.Exec(statement)
	return err
}

func RegisterMeteringPoint(tenant, participantId string, point *model.MeteringPoint) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	type meteringEntryType struct {
		*model.MeteringPoint
		ParticipantId string `db:"participant_id"`
		Tenant        string
	}
	meteringEntry := meteringEntryType{point, participantId, tenant}

	statement, _, _ := pgDialect.Insert(TABLE_METERINGPOINT).Rows(meteringEntry).ToSQL()
	_, err = db.Exec(statement)
	return err
}

func UpdateMeteringPoint(tenant, participantId, meterId string, meteringPoint *model.MeteringPoint) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := goqu.Update(TABLE_METERINGPOINT).
		Set(meteringPoint).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
			"participant_id":    goqu.Op{"eq": participantId},
		}).
		ToSQL()
	_, err = db.Exec(statement)

	return err
}

func RemoveMeteringPoint(dbOpen OpenDbXConnection, tenant, participantId, meterId string) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := goqu.Delete(TABLE_METERINGPOINT).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
			"participant_id":    goqu.Op{"eq": participantId},
			"status":            goqu.Op{"eq": "INVALID"},
		}).
		ToSQL()
	_, err = db.Exec(statement)

	return err
}

func ActivateMeteringPoints(tenant string, meterId []string) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := goqu.Update(TABLE_METERINGPOINT).
		Set(goqu.Record{"status": "ACTIVE"}).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
		}).
		ToSQL()
	_, err = db.Exec(statement)

	return err
}

func GetParticipantByMeteringPoint(dbOpen OpenDbXConnection, tenant, meteringPointId string) (*model.EegParticipant, error) {
	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	mpSQL, _, err := pgDialect.From("base.meteringpoint").
		Select(goqu.C("participant_id")).
		Where(goqu.Ex{
			"metering_point_id": goqu.Op{"eq": meteringPointId},
			"tenant":            goqu.Op{"eq": tenant},
		}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	var participantId string
	if err = db.Get(&participantId, mpSQL); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var p model.EegParticipant
	pSQL, _, err := pgDialect.From("base.participant").
		Select(&p).
		Where(goqu.Ex{
			"id":     goqu.Op{"eq": participantId},
			"tenant": goqu.Op{"eq": tenant},
		}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	if err = db.Get(&p, pSQL); err != nil {
		return nil, err
	}
	if err = CompleteParticipant(db, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func MeteringPointsSetStatus(dbOpen OpenDbXConnection, tenant string, status model.StatusType, meterId []string) (int64, error) {
	db, err := dbOpen()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	statement, _, _ := goqu.Update(TABLE_METERINGPOINT).
		Set(goqu.Record{"status": status}).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
		}).
		ToSQL()
	result, err := db.Exec(statement)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// FindActiveMeteringByIds returns the metering points for the given
// tenant whose IDs are in meterIds and whose status is ACTIVE. Order
// of the returned slice is not guaranteed.
func FindActiveMeteringByIds(tenant string, meterIds []string) ([]*model.MeteringPoint, error) {
	db, err := GetDBXConnection()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	statement, _, err := pgDialect.From(TABLE_METERINGPOINT).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"status":            goqu.Op{"eq": model.ACTIVE},
			"metering_point_id": goqu.Op{"in": meterIds},
		}).ToSQL()
	if err != nil {
		return nil, err
	}

	var points []*model.MeteringPoint
	if err := db.Select(&points, statement); err != nil {
		log.WithField("SQL", "SELECT").Errorf("Stmt: %v", statement)
		return nil, err
	}
	return points, nil
}

// UpdateMeteringPointPartFact appends a new partition-factor row to
// the metering_partition_factor history table. The SERIAL version
// column ensures monotonic ordering; the activeMeteringPartition view
// exposes only the latest version per metering point.
//
// Use case: a participant's share of an EEG meter changes (e.g. via
// the /v2/{pid}/update/{mid}/partfact route). Old factors stay in the
// history table for audit and billing purposes.
func UpdateMeteringPointPartFact(tenant, username, participantId, meterId string, partFact int) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, err := pgDialect.Insert(TABLE_PARTITION_FACT).
		Rows(goqu.Record{
			"metering_point_id": meterId,
			"participant_id":    participantId,
			"tenant":            tenant,
			"partFact":          partFact,
			"createdBy":         username,
		}).
		ToSQL()
	if err != nil {
		return err
	}

	if _, err = db.Exec(statement); err != nil {
		log.WithField("SQL", "INSERT").Errorf("Stmt: %v", statement)
		return err
	}
	return nil
}

// MoveMeteringPoint re-parents a metering point from one participant
// to another within the same tenant. The change is wrapped in a
// transaction; modifiedBy/modifiedAt are stamped to track the operation.
//
// Use case: a metering point was wired to the wrong participant on
// import and needs to be re-assigned without disturbing its history.
func MoveMeteringPoint(tenant, username, sourceParticipantId, destParticipantId, meterId string) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	statement, _, err := pgDialect.Update(TABLE_METERINGPOINT).
		Set(goqu.Record{
			"participant_id": destParticipantId,
			"modifiedBy":     username,
			"modifiedAt":     time.Now(),
		}).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
			"participant_id":    goqu.Op{"eq": sourceParticipantId},
		}).
		ToSQL()
	if err != nil {
		return err
	}

	if _, err = tx.Exec(statement); err != nil {
		log.WithField("SQL", "UPDATE").Errorf("Stmt: %v", statement)
		return err
	}
	return nil
}

// MeteringPointRevoke marks a metering point as revoked for the given
// tenant. inactiveSince records the consent end date; status is set
// to model.REVOKED. Returns nil on success or a wrapped error if the
// update failed.
func MeteringPointRevoke(tenant, meterId string, inactiveSince time.Time) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := goqu.Update(TABLE_METERINGPOINT).
		Set(goqu.Record{
			"status":        model.REVOKED,
			"inactiveSince": inactiveSince,
		}).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
		}).
		ToSQL()

	if _, err = db.Exec(statement); err != nil {
		log.WithField("SQL", "UPDATE").Errorf("Stmt: %v", statement)
		return err
	}
	return nil
}

// UpdateMeteringPointPartial applies a partial update to a metering
// point row. The values map carries the columns to update (already
// in their DB-column names). modifiedBy and modifiedAt are added
// automatically.
func UpdateMeteringPointPartial(tenant, username, participantId, meterId string, values map[string]interface{}) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	values["modifiedBy"] = username
	values["modifiedAt"] = time.Now()

	statement, _, err := pgDialect.Update(TABLE_METERINGPOINT).Set(values).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
			"participant_id":    goqu.Op{"eq": participantId},
		}).
		ToSQL()
	if err != nil {
		return err
	}

	if _, err = db.Exec(statement); err != nil {
		log.WithField("SQL", "UPDATE").Errorf("Stmt: %v", statement)
		return err
	}
	return nil
}
