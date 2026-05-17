package database

import (
	dbsql "database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

const TABLE_EEG = "base.eeg"
const TABLE_EEG_ADDRESS = "base.address"

func GetEeg(tenant string) (*model.Eeg, error) {

	db, err := GetDBXConnection()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var eeg model.Eeg
	err = db.QueryRow(""+
		"SELECT name, \"businessNr\", legal, gridoperator_name, \"communityId\", gridoperator_code, \"rcNumber\", \"allocationMode\", "+
		"\"settlementInterval\", \"providerBusinessNr\", street, \"streetNumber\", zip, city, phone, email, website, iban, owner, sepa, "+
		"\"taxNumber\", \"vatNumber\", online, \"contactPerson\" FROM base.eeg WHERE tenant = $1", tenant).
		Scan(&eeg.Name, &eeg.BusinessNr, &eeg.Legal, &eeg.OperatorName,
			&eeg.CommunityId, &eeg.GridOperator, &eeg.RcNumber,
			&eeg.AllocationMode, &eeg.SettlementInterval, &eeg.ProviderBusinessNr,
			&eeg.Street, &eeg.StreetNumber, &eeg.Zip, &eeg.City, &eeg.Contact.Phone, &eeg.Contact.Email,
			&eeg.Optionals.Website, &eeg.AccountInfo.Iban, &eeg.AccountInfo.Owner, &eeg.AccountInfo.Sepa,
			&eeg.TaxNumber, &eeg.VatNumber, &eeg.Online, &eeg.ContactPerson,
		)
	if err == dbsql.ErrNoRows {
		return &eeg, nil
	}
	eeg.Id = tenant
	return &eeg, err
}

func UpdateEeg(db *sqlx.DB, tenant string, eeg *model.Eeg) error {

	//db, err := GetDBXConnection()
	//if err != nil {
	//	return err
	//}
	//defer db.Close()

	sql, _, err := pgDialect.Insert("base.eeg").Rows(eeg).ToSQL()
	fmt.Printf("Stmt: %s\n", sql)
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	return err
}

func UpdateEegPartial(dbOpen OpenDbXConnection, tenant string, fields map[string]interface{}) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := pgDialect.Update(TABLE_EEG).Set(fields).Where(goqu.Ex{"tenant": goqu.V(tenant)}).ToSQL()

	log.Debugf("Update EEG VALUES: %s\n", statement)

	_, err = db.Exec(statement)
	return err
}

func UpdateEegAddressPartial(dbOpen OpenDbXConnection, tenant string, fields map[string]interface{}) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	statement, _, _ := pgDialect.Update(TABLE_EEG_ADDRESS).Set(fields).Where(goqu.Ex{"tenant": goqu.V(tenant)}).ToSQL()

	log.Debugf("Update EEG VALUES: %s\n", statement)

	_, err = db.Exec(statement)
	return err
}

func GetCommunityId(tenant string) (string, error) {

	db, err := GetDBConnection()
	if err != nil {
		return "", err
	}
	defer db.Close()

	communityId := ""
	err = db.QueryRow(`SELECT "communityId" FROM base.eeg WHERE tenant = $1`, tenant).Scan(&communityId)

	return communityId, err
}

//func fetchEegAddressInfo(db sqlx.DB, tenant string)

func SaveNotification(dbOpen OpenDbXConnection, tenant string, notification string, msgType, role string) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO base.notification (tenant, notification, date, type, role) VALUES ($1, $2, NOW(), $3, $4)", tenant, notification, msgType, role)
	return err
}

func GetNotification(dbOpen OpenDbXConnection, tenant string, start int64, isAdmin bool) ([]model.EegNotification, error) {
	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	n := []model.EegNotification{}

	statement := pgDialect.From("base.notification").Select(&n).
		Where(goqu.C("tenant").Eq(tenant), goqu.C("id").Gt(start))
	if !isAdmin {
		statement = statement.Where(goqu.C("role").Eq("USER"))
	}

	sql, _, err := statement.Order(goqu.I("id").Desc()).Limit(30).ToSQL()
	if err != nil {
		return nil, err
	}
	err = db.Select(&n, sql)
	if err != nil && err != dbsql.ErrNoRows {
		return nil, err
	}

	return n, err
}
