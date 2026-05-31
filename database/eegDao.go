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

func GetEeg(dbOpen OpenDbXConnection, tenant string) (*model.Eeg, error) {

	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var eeg model.Eeg
	err = db.QueryRow(""+
		"SELECT name, description, \"businessNr\", legal, gridoperator_name, \"communityId\", gridoperator_code, \"rcNumber\", \"allocationMode\", "+
		"\"settlementInterval\", \"providerBusinessNr\", street, \"streetNumber\", zip, city, phone, email, website, iban, owner, sepa, "+
		"\"bankName\", creditor_id, bic, \"bankPurpose\", "+
		"\"taxNumber\", \"vatNumber\", online, \"contactPerson\" FROM base.eeg WHERE tenant = $1", tenant).
		Scan(&eeg.Name, &eeg.Description, &eeg.BusinessNr, &eeg.Legal, &eeg.OperatorName,
			&eeg.CommunityId, &eeg.GridOperator, &eeg.RcNumber,
			&eeg.AllocationMode, &eeg.SettlementInterval, &eeg.ProviderBusinessNr,
			&eeg.Street, &eeg.StreetNumber, &eeg.Zip, &eeg.City, &eeg.Contact.Phone, &eeg.Contact.Email,
			&eeg.Optionals.Website, &eeg.AccountInfo.Iban, &eeg.AccountInfo.Owner, &eeg.AccountInfo.Sepa,
			&eeg.AccountInfo.BankName, &eeg.AccountInfo.CreditorId, &eeg.AccountInfo.Bic, &eeg.AccountInfo.BankPurpose,
			&eeg.TaxNumber, &eeg.VatNumber, &eeg.Online, &eeg.ContactPerson,
		)
	if err == dbsql.ErrNoRows {
		return &eeg, nil
	}
	eeg.Id = tenant
	return &eeg, err
}

// GetEegByEcId fetches an EEG row by its communityId (the EDA-side
// identifier used in inbound EBMS payloads). Returns the same shape
// as GetEeg.
func GetEegByEcId(dbOpen OpenDbXConnection, ecId string) (*model.Eeg, error) {

	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return GetEegByEcIdDB(db, ecId)
}

// GetEegByEcIdDB is the connection-bound variant used by handlers that
// need to chain multiple DAO calls against a single opened connection.
func GetEegByEcIdDB(db *sqlx.DB, ecId string) (*model.Eeg, error) {
	var eeg model.Eeg
	err := db.QueryRow(""+
		"SELECT tenant, name, description, \"businessNr\", legal, gridoperator_name, \"communityId\", gridoperator_code, \"rcNumber\", \"allocationMode\", "+
		"\"settlementInterval\", \"providerBusinessNr\", street, \"streetNumber\", zip, city, phone, email, website, iban, owner, sepa, "+
		"\"bankName\", creditor_id, bic, \"bankPurpose\", "+
		"\"taxNumber\", \"vatNumber\", online, \"contactPerson\" FROM base.eeg WHERE \"communityId\" = $1", ecId).
		Scan(&eeg.Id, &eeg.Name, &eeg.Description, &eeg.BusinessNr, &eeg.Legal, &eeg.OperatorName,
			&eeg.CommunityId, &eeg.GridOperator, &eeg.RcNumber,
			&eeg.AllocationMode, &eeg.SettlementInterval, &eeg.ProviderBusinessNr,
			&eeg.Street, &eeg.StreetNumber, &eeg.Zip, &eeg.City, &eeg.Contact.Phone, &eeg.Contact.Email,
			&eeg.Optionals.Website, &eeg.AccountInfo.Iban, &eeg.AccountInfo.Owner, &eeg.AccountInfo.Sepa,
			&eeg.AccountInfo.BankName, &eeg.AccountInfo.CreditorId, &eeg.AccountInfo.Bic, &eeg.AccountInfo.BankPurpose,
			&eeg.TaxNumber, &eeg.VatNumber, &eeg.Online, &eeg.ContactPerson,
		)
	if err == dbsql.ErrNoRows {
		return nil, err
	}
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

// GetGridOperators returns the AT grid-operator lookup table as a
// `{id: name}` map. Source: `base.gridoperators` (id, name) — seeded
// at deploy time from the public ECP-AT regulator list.
func GetGridOperators(dbOpen OpenDbXConnection) (map[string]string, error) {

	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name FROM base.gridoperators`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id, name string
	result := map[string]string{}
	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}
	return result, nil
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
