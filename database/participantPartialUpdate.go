package database

import (
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
)

// PartialUpdateError carries the structured error prod returns from
// PUT /participant/v2/{id} so the HTTP layer can echo `{error:{code,
// error, message}}`. Code 1102 is "unknown path", 1103 is "DB rejected
// the update" (constraint violations etc.); we mirror that split so the
// frontend's error-key construction in store.ts:44 sees the same shape
// it sees from prod.
type PartialUpdateError struct {
	Code    int
	Message string
}

func (e *PartialUpdateError) Error() string { return e.Message }

// participantColumns maps the JSON path the frontend emits to the
// physical column name on base.participant. Per-table whitelisting
// keeps the column name out of user-controlled SQL emission.
var participantColumns = map[string]string{
	"participantNumber":     "participantNumber",
	"firstname":             "firstname",
	"lastname":              "lastname",
	"role":                  "role",
	"businessRole":          "businessRole",
	"titleBefore":           "titleBefore",
	"titleAfter":            "titleAfter",
	"participantSince":      "participantSince",
	"vatNumber":             "vatNumber",
	"taxNumber":             "taxNumber",
	"companyRegisterNumber": "companyRegisterNumber",
	"status":                "status",
	"tariffId":              "tariffId",
}

var addressColumns = map[string]string{
	"street":       "street",
	"streetNumber": "streetNumber",
	"city":         "city",
	"zip":          "zip",
}

var contactColumns = map[string]string{
	"email": "email",
	"phone": "phone",
}

// bankAccountColumns maps frontend dotted-path tails to the actual DB
// column. Note the camelCase→snake_case rename for the SEPA-mandate
// triplet — base.bankaccount is one of the older tables in the schema.
var bankAccountColumns = map[string]string{
	"iban":             "iban",
	"owner":            "owner",
	"bankName":         "bankName",
	"mandateReference": "mandate_reference",
	"mandateDate":      "mandate_date",
	"sepaDirectDebit":  "sepa_direct_debit",
}

// UpdateParticipantPartial applies a single {path, value} change to one
// of the rows tied to a participant — base.participant or one of its
// child rows. Path uses the same dotted style the frontend's
// participant.service.ts sends ("billingAddress.city", "contact.email",
// "accountInfo.mandateDate", "tariffId", ...).
//
// The frontend's MemberForm.component.tsx fires one of these per field
// change, so this endpoint runs hot — a single targeted UPDATE per call
// is the right shape.
func UpdateParticipantPartial(dbConn OpenDbXConnection, tenant, participantId, path string, value interface{}) error {
	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	parts := strings.SplitN(path, ".", 2)
	root := parts[0]

	switch {
	case len(parts) == 1:
		col, ok := participantColumns[root]
		if !ok {
			return unknownPath(path)
		}
		return updateColumn(db, "base.participant",
			goqu.Ex{"id": participantId, "tenant": tenant}, col, value)
	case root == "billingAddress":
		col, ok := addressColumns[parts[1]]
		if !ok {
			return unknownPath(path)
		}
		return updateOrInsertAddressColumn(db, participantId, "BILLING", col, value)
	case root == "residentAddress", root == "residenceAddress":
		col, ok := addressColumns[parts[1]]
		if !ok {
			return unknownPath(path)
		}
		return updateOrInsertAddressColumn(db, participantId, "RESIDENCE", col, value)
	case root == "contact":
		col, ok := contactColumns[parts[1]]
		if !ok {
			return unknownPath(path)
		}
		return updateColumn(db, "base.contactdetail",
			goqu.Ex{"participant_id": participantId}, col, value)
	case root == "accountInfo", root == "bankAccount":
		col, ok := bankAccountColumns[parts[1]]
		if !ok {
			return unknownPath(path)
		}
		return updateColumn(db, "base.bankaccount",
			goqu.Ex{"participant_id": participantId}, col, value)
	}
	return unknownPath(path)
}

func unknownPath(path string) error {
	return &PartialUpdateError{
		Code:    1102,
		Message: fmt.Sprintf("Can not update structure of %s", path),
	}
}

// updateOrInsertAddressColumn handles partial updates to base.address, which
// (unlike participant/contactdetail/bankaccount) does not always have a row
// per (participant, type). Members imported via the initial bootstrap don't
// get RESIDENCE/BILLING address rows — the create-flow inserts them only
// for newly registered members. Without this auto-insert, the first
// street/city/zip-edit of an imported member fails with 1103.
func updateOrInsertAddressColumn(db *sqlx.DB, participantId, addressType, col string, value interface{}) error {
	updateStmt, _, err := pgDialect.Update("base.address").
		Set(goqu.Record{col: value}).
		Where(goqu.Ex{"participant_id": participantId, "type": addressType}).
		ToSQL()
	if err != nil {
		return &PartialUpdateError{Code: 1103, Message: err.Error()}
	}
	res, err := db.Exec(updateStmt)
	if err != nil {
		return &PartialUpdateError{Code: 1103, Message: err.Error()}
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	insertStmt, _, err := pgDialect.Insert("base.address").
		Rows(goqu.Record{
			"participant_id": participantId,
			"type":           addressType,
			col:              value,
		}).
		ToSQL()
	if err != nil {
		return &PartialUpdateError{Code: 1103, Message: err.Error()}
	}
	if _, err := db.Exec(insertStmt); err != nil {
		return &PartialUpdateError{Code: 1103, Message: err.Error()}
	}
	return nil
}

func updateColumn(db *sqlx.DB, table string, where goqu.Ex, col string, value interface{}) error {
	stmt, _, err := pgDialect.Update(table).
		Set(goqu.Record{col: value}).
		Where(where).
		ToSQL()
	if err != nil {
		return &PartialUpdateError{Code: 1103, Message: err.Error()}
	}
	res, err := db.Exec(stmt)
	if err != nil {
		return &PartialUpdateError{Code: 1103, Message: err.Error()}
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &PartialUpdateError{
			Code:    1103,
			Message: fmt.Sprintf("No matching row in %s for the given participant", table),
		}
	}
	return nil
}

