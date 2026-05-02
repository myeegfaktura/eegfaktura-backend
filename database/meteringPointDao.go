package database

import (
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	"github.com/eegfaktura/eegfaktura-backend/model"
	log "github.com/sirupsen/logrus"
)

const TABLE_METERINGPOINT = "base.meteringpoint"

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

	var p model.EegParticipant
	err = db.QueryRowx(`
		SELECT p.id, p.firstname, p.lastname, p."participantNumber", p."businessRole", p.role,
		       COALESCE(p."titleBefore", '') AS "titleBefore", COALESCE(p."titleAfter", '') AS "titleAfter",
		       p."participantSince", COALESCE(p."vatNumber", '') AS "vatNumber", COALESCE(p."taxNumber", '') AS "taxNumber",
		       p."companyRegisterNumber", p."tariffId", p.status, p.version, p."createdBy"
		FROM base.participant p
		JOIN base.meteringpoint mp ON mp.participant_id = p.id
		WHERE mp.metering_point_id = $1 AND mp.tenant = $2
	`, meteringPointId, tenant).StructScan(&p)
	if err != nil {
		return nil, err
	}

	contactSQL, _, err := pgDialect.From("base.contactdetail").Select(&p.Contact).Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
	if err != nil {
		return nil, err
	}
	if err = db.Get(&p.Contact, contactSQL); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &p, nil
}

func MeteringPointsSetStatus(dbOpen OpenDbXConnection, tenant string, status model.StatusType, meterId []string) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := goqu.Update(TABLE_METERINGPOINT).
		Set(goqu.Record{"status": status}).
		Where(goqu.Ex{
			"tenant":            goqu.Op{"eq": tenant},
			"metering_point_id": goqu.Op{"eq": meterId},
		}).
		ToSQL()
	_, err = db.Exec(statement)

	return err
}
