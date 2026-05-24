package database

import (
	dbsql "database/sql"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/jmoiron/sqlx"
	"github.com/pborman/uuid"
)

func GetParticipant(dbConn OpenDbXConnection, tenant string) ([]model.EegParticipant, error) {
	var participants []model.EegParticipant = []model.EegParticipant{}
	db, err := dbConn()
	if err != nil {
		return []model.EegParticipant{}, err
	}
	defer db.Close()

	sql, _, err := pgDialect.From("base.participant").Select(&participants).
		Where(goqu.Ex{
			"tenant": tenant, "status": goqu.Op{"neq": "ARCHIVED"}}).Order(goqu.I("lastname").Asc()).ToSQL()
	if err != nil {
		return []model.EegParticipant{}, err
	}

	err = db.Select(&participants, sql)
	if err != nil {
		return []model.EegParticipant{}, err
	}

	for i, p := range participants {
		sql, _, err = pgDialect.From("base.contactdetail").Select(&p.Contact).Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
		if err != nil {
			return []model.EegParticipant{}, err
		}
		err = db.Get(&(participants[i].Contact), sql)
		if err != nil && err != dbsql.ErrNoRows {
			return []model.EegParticipant{}, err
		}

		sql, _, err = pgDialect.From("base.bankaccount").Select(&p.BankAccount).Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
		if err != nil {
			return []model.EegParticipant{}, err
		}
		err = db.Get(&(participants[i].BankAccount), sql)
		if err != nil && err != dbsql.ErrNoRows {
			return []model.EegParticipant{}, err
		}

		sql, _, err = pgDialect.From("base.address").Select(&p.BillingAddress).
			Where(goqu.C("participant_id").Eq(p.Id.String()), goqu.C("type").Eq("BILLING")).ToSQL()
		if err != nil {
			return []model.EegParticipant{}, err
		}
		err = db.Get(&(participants[i].BillingAddress), sql)
		if err != nil && err != dbsql.ErrNoRows {
			return []model.EegParticipant{}, err
		}

		sql, _, err = pgDialect.From("base.address").Select(&p.ResidentAddress).
			Where(goqu.C("participant_id").Eq(p.Id.String()), goqu.C("type").Eq("RESIDENCE")).ToSQL()
		if err != nil {
			return []model.EegParticipant{}, err
		}
		//fmt.Printf("SQL: %+v\n", sql)
		err = db.Get(&(participants[i].ResidentAddress), sql)
		if err != nil && err != dbsql.ErrNoRows {
			return []model.EegParticipant{}, err
		}
		//fmt.Printf("ADDRESS: %+v\n", p.ResidentAddress)

		sql, _, err = pgDialect.From("base.meteringpoint").Select(&p.MeteringPoint).
			Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
		if err != nil {
			return []model.EegParticipant{}, err
		}
		err = db.Select(&(participants[i].MeteringPoint), sql)
		if err != nil && err != dbsql.ErrNoRows {
			return []model.EegParticipant{}, err
		}
		if participants[i].MeteringPoint == nil {
			participants[i].MeteringPoint = []*model.MeteringPoint{}
		}
	}

	return participants, nil
}

func QueryParticipant(dbOpen OpenDbXConnection, participantId string) (*model.EegParticipant, error) {
	var participant model.EegParticipant = model.EegParticipant{}
	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sql, _, err := pgDialect.From("base.participant").Select(&participant).Where(goqu.C("id").Eq(participantId)).ToSQL()
	if err != nil {
		return nil, err
	}
	err = db.Get(&participant, sql)
	if err != nil {
		return nil, err
	}

	err = CompleteParticipant(db, &participant)
	if err != nil {
		return nil, err
	}

	return &participant, nil
}

func CompleteParticipant(db *sqlx.DB, p *model.EegParticipant) error {
	sql, _, err := pgDialect.From("base.contactdetail").Select(&p.Contact).Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
	if err != nil {
		return err
	}
	err = db.Get(&(p.Contact), sql)
	if err != nil && err != dbsql.ErrNoRows {
		return err
	}

	sql, _, err = pgDialect.From("base.bankaccount").Select(&p.BankAccount).Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
	if err != nil {
		return err
	}
	err = db.Get(&(p.BankAccount), sql)
	if err != nil && err != dbsql.ErrNoRows {
		return err
	}

	sql, _, err = pgDialect.From("base.address").Select(&p.BillingAddress).
		Where(goqu.C("participant_id").Eq(p.Id.String()), goqu.C("type").Eq("BILLING")).ToSQL()
	if err != nil {
		return err
	}
	err = db.Get(&(p.BillingAddress), sql)
	if err != nil && err != dbsql.ErrNoRows {
		return err
	}

	sql, _, err = pgDialect.From("base.address").Select(&p.ResidentAddress).
		Where(goqu.C("participant_id").Eq(p.Id.String()), goqu.C("type").Eq("RESIDENCE")).ToSQL()
	if err != nil {
		return err
	}
	//fmt.Printf("SQL: %+v\n", sql)
	err = db.Get(&(p.ResidentAddress), sql)
	if err != nil && err != dbsql.ErrNoRows {
		return err
	}
	//fmt.Printf("ADDRESS: %+v\n", p.ResidentAddress)

	sql, _, err = pgDialect.From("base.meteringpoint").Select(&p.MeteringPoint).
		Where(goqu.C("participant_id").Eq(p.Id.String())).ToSQL()
	if err != nil {
		return err
	}
	err = db.Select(&(p.MeteringPoint), sql)
	if err != nil && err != dbsql.ErrNoRows {
		return err
	}
	return nil
}

func UpdateParticipant(tenant, user string, participant *model.EegParticipant) error {
	db, err := GetDBXConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	sql, _, _ := goqu.Update("base.participant").
		Set(participant).
		Where(goqu.Ex{
			"tenant": goqu.Op{"eq": tenant},
			"id":     goqu.Op{"eq": participant.Id.String()},
		}).
		ToSQL()
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql, _, _ = goqu.Update("base.contactdetail").
		Set(participant.Contact).
		Where(goqu.Ex{
			"participant_id": participant.Id.String(),
		}).
		ToSQL()
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql, _, _ = goqu.Update("base.address").
		Set(participant.ResidentAddress).
		Where(goqu.Ex{
			"type":           model.RESIDENCE,
			"participant_id": participant.Id.String(),
		}).
		ToSQL()
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql, _, _ = goqu.Update("base.address").
		Set(participant.BillingAddress).
		Where(goqu.Ex{
			"type":           model.BILLING,
			"participant_id": participant.Id.String(),
		}).
		ToSQL()
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	sql, _, _ = goqu.Update("base.bankaccount").
		Set(participant.BankAccount).
		Where(goqu.Ex{
			"participant_id": participant.Id.String(),
		}).
		ToSQL()
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}
	return err
}

type ParticipantWithMeta struct {
	*model.EegParticipant
	Tenant         string `db:"tenant"`
	CreatedBy      string `db:"createdBy"`
	LastmodifiedBy string `db:"lastModifiedBy"`
}

func RegisterParticipant(dbConn OpenDbXConnection, tenant, username string, participant *model.EegParticipant) error {
	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	participant.Status = model.PENDING
	participant.Id = uuid.NewUUID()
	participant.ParticipantSince = time.Now()
	// Reflect createdBy on the model so the HTTP response carries it
	// (matches prod-image's POST /api/participant body shape, parity
	// gap surfaced by ADR-0007 test 04-participant-create). With
	// `json:"createdBy,omitempty"` an empty value would be omitted —
	// prod always emits it because the value is set.
	participant.CreatedBy = username
	return saveParticipant(db, tenant, username, participant, RegisterMeteringPoints)
}

func ImportParticipant(dbConn OpenDbXConnection, tenant, username string, participant *model.EegParticipant) error {
	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	// check if User already exists
	sql, _, err := pgDialect.From("base.participant").
		Select("id").
		Where(goqu.C("firstname").Eq(participant.FirstName),
			goqu.C("lastname").Eq(participant.LastName)).ToSQL()
	if err == nil {
		participantId := ""
		err = db.Get(&participantId, sql)
		if err == nil {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Commit()
			return ImportMeteringPoints(tx, tenant, participantId, participant.MeteringPoint)
		}
	}

	participant.Id = uuid.NewUUID()
	return saveParticipant(db, tenant, username, participant, ImportMeteringPoints)
}

func ConfirmParticipant(dbOpen OpenDbXConnection, tenant, username, participantId string) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("UPDATE base.participant SET status = 'ACTIVE', \"lastModifiedDate\" = 'now()', \"lastModifiedBy\" = $1 WHERE id = $2", username, participantId)

	return err
}

func saveParticipant(db *sqlx.DB, tenant, username string, participant *model.EegParticipant,
	registerMeteringPointsFunc func(*dbsql.Tx, string, string, []*model.MeteringPoint) error) error {

	registeredParticipant := ParticipantWithMeta{
		participant, tenant, username, username,
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	participantId := ""
	sql, _, _ := pgDialect.Insert("base.participant").Rows(registeredParticipant).Returning("id").
		//OnConflict(goqu.DoUpdate("lastmodifieddate", goqu.L("NOW()"))).
		ToSQL()
	err = tx.QueryRow(sql).Scan(&participantId)
	if err != nil {
		return err
	}

	contactEntry := struct {
		model.ContactInfo
		Participant_id string
	}{participant.Contact, participantId}
	sql, _, _ = pgDialect.Insert("base.contactdetail").Rows(contactEntry).ToSQL()
	_, err = tx.Exec(sql)
	if err != nil {
		return err
	}

	bankInfoEntry := struct {
		model.BankInfo
		Participant_id string
	}{participant.BankAccount, participantId}

	sql, _, _ = pgDialect.Insert("base.bankaccount").Rows(bankInfoEntry).ToSQL()
	_, err = tx.Exec(sql)
	if err != nil {
		return err
	}

	billingAddrEntry := struct {
		model.Address
		Participant_id string
	}{participant.BillingAddress, participantId}
	residenceAddrEntry := struct {
		model.Address
		Participant_id string
	}{participant.ResidentAddress, participantId}
	sql, _, _ = pgDialect.Insert("base.address").Rows(billingAddrEntry, residenceAddrEntry).ToSQL()
	_, err = tx.Exec(sql)
	if err != nil {
		return err
	}

	err = registerMeteringPointsFunc(tx, tenant, participantId, participant.MeteringPoint)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func ArchiveParticipant(dbConn OpenDbXConnection, user string, id string) error {

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, _, err := pgDialect.Update("base.participant").
		Set(goqu.Record{"status": "ARCHIVED", "lastModifiedDate": time.Now(), "lastModifiedBy": user}).
		Where(goqu.Ex{"id": id}).ToSQL()
	if err != nil {
		return err
	}
	_, err = db.Exec(stmt)
	return err
}

func InsertParticipant(tenant string, participant *model.EegParticipant) error {
	return nil
}
