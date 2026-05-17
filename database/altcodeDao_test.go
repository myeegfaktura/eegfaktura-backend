package database

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// --- database.go ---

func TestGetTariff(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "name", "billingPeriod", "useVat", "vatInPercent",
		"accountNetAmount", "accountGrossAmount", "participantFee", "baseFee",
		"businessNr", "version", "type", "centPerKWh", "discount", "freeKWh",
	}).
		AddRow("bd427cac-c6a7-49b1-915e-c7eeb215bb5d", "Sepp", "monthly", false, 0, 0, 0, 0, 0, 0, 1, "", 12, 0, 100)

	mockDb.Mock.ExpectQuery(`SELECT .* FROM base\.activetariff WHERE tenant = \$1`).
		WithArgs("TE100100").
		WillReturnRows(rows)

	got, err := GetTariff(mockDb.OpenMockDb, "TE100100")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Sepp", got[0].Name)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestArchiveTariff(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	tariffId := "bd427cac-c6a7-49b1-915e-c7eeb215bb5d"

	// In-use check 1: participants with this tariffId — must return 0 rows.
	mockDb.Mock.ExpectQuery(`SELECT "id" FROM "base"\."participant" WHERE \("tariffId" = '` + tariffId + `'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	// In-use check 2: meteringpoints with this tariffId — also empty.
	mockDb.Mock.ExpectQuery(`SELECT "id" FROM "base"\."meteringpoint" WHERE .*tariffId.*` + tariffId).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mockDb.Mock.ExpectExec(`UPDATE base\.tariff SET status = 'ARCHIVED' WHERE tenant = \$1 AND id = \$2`).
		WithArgs("TE100100", tariffId).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = ArchiveTariff(mockDb.OpenMockDb, "TE100100", tariffId)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

// --- eegDao.go ---

func TestGetEeg(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	row := sqlmock.NewRows([]string{
		"name", "businessNr", "legal", "gridoperator_name", "communityId",
		"gridoperator_code", "rcNumber", "allocationMode", "settlementInterval",
		"providerBusinessNr", "street", "streetNumber", "zip", "city",
		"phone", "email", "website", "iban", "owner", "sepa",
		"taxNumber", "vatNumber", "online", "contactPerson",
	}).AddRow(
		"T-VIERE", 123456789, "verein", "Netz OOE", "AT00300000000TC100100000000000001",
		"AT003000", "TE100100", "DYNAMIC", "MONTHLY",
		nil, "Solarstraße", "9", "1111", "Solarcity",
		"0043-664-1234567", "test-eeg@gmx.at", "test-eeg.at",
		"AT011234000000321321", "T-VIERE", false,
		"11 123/4567", nil, false, "Max Sonnenmann",
	)
	mockDb.Mock.ExpectQuery(`SELECT name, .* FROM base\.eeg WHERE tenant = \$1`).
		WithArgs("TE100100").
		WillReturnRows(row)

	got, err := GetEeg(mockDb.OpenMockDb, "TE100100")
	require.NoError(t, err)
	assert.Equal(t, "T-VIERE", got.Name)
	assert.Equal(t, "TE100100", got.Id, "Id should be set to the tenant string")
	assert.Equal(t, "AT00300000000TC100100000000000001", got.CommunityId)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestGetEeg_NoRowsReturnsEmpty(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	// QueryRow without a row triggers sql.ErrNoRows in Scan; the function
	// returns &eeg, nil for that case (NOT the wrapped error).
	mockDb.Mock.ExpectQuery(`SELECT name, .* FROM base\.eeg WHERE tenant = \$1`).
		WithArgs("TE-MISSING").
		WillReturnRows(sqlmock.NewRows([]string{"name"})) // empty

	got, err := GetEeg(mockDb.OpenMockDb, "TE-MISSING")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got.Name)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestSaveNotification(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`INSERT INTO base\.notification \(tenant, notification, date, type, role\) VALUES \(\$1, \$2, NOW\(\), \$3, \$4\)`).
		WithArgs("TE100100", `{"foo":"bar"}`, "CR_MSG", "USER").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = SaveNotification(mockDb.OpenMockDb, "TE100100", `{"foo":"bar"}`, "CR_MSG", "USER")
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

// --- notificationDao.go ---

func TestSaveEdaHistory(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	history := &model.EdaProcessHistory{
		Tenant:         "TE100100",
		ConversationId: "conv-1",
		ProcessType:    model.EBMS_ONLINE_REG_APPROVAL,
		Protocol:       null.StringFrom("CR_MSG"),
		Issuer:         "ADMIN",
		MessageByte:    []byte(`{}`),
		Direction:      "CONSUMPTION",
	}

	mockDb.Mock.ExpectExec(`INSERT INTO "base"\."processhistory" .* VALUES .*'TE100100'.*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = SaveEdaHistory(mockDb.OpenMockDb, history)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

// --- participantDao.go ---

func TestQueryParticipant(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	participantId := uuid.New()

	// Main row from base.participant.
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."participant" WHERE \("id" = '` + participantId + `'\)`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "firstname", "lastname", "role", "businessRole", "titleBefore", "titleAfter",
			"participantSince", "vatNumber", "taxNumber", "companyRegisterNumber", "status", "createdBy",
			"version", "tariffId", "participantNumber",
		}).AddRow(participantId, "Sepp", "Huber", "EEG_USER", "EEG_PRIVATE", "", "",
			time.Now(), "", "", "", "NEW", "admin",
			1, "", "001"))

	// CompleteParticipant follow-up queries: contact, bank, billing addr, residence addr, meters.
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."contactdetail"`).
		WillReturnRows(sqlmock.NewRows([]string{"email", "phone"}).AddRow("a@b.c", "+43"))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."bankaccount"`).
		WillReturnRows(sqlmock.NewRows([]string{"iban", "owner"}))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."address" WHERE .*"type" = 'BILLING'`).
		WillReturnRows(sqlmock.NewRows([]string{"city", "street", "streetNumber", "type", "zip"}))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."address" WHERE .*"type" = 'RESIDENCE'`).
		WillReturnRows(sqlmock.NewRows([]string{"city", "street", "streetNumber", "type", "zip"}))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."meteringpoint"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"city", "direction", "equipmentName", "equipmentNumber", "inverterid",
			"metering_point_id", "modifiedAt", "modifiedBy", "registeredSince", "status",
			"street", "streetNumber", "tariff_id", "transformer", "zip",
		}))

	got, err := QueryParticipant(mockDb.OpenMockDb, participantId)
	require.NoError(t, err)
	assert.Equal(t, "Sepp", got.FirstName)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestCompleteParticipant(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	pid := uuid.New()
	p := &model.EegParticipant{Id: uuid.Parse(pid)}

	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."contactdetail"`).
		WillReturnRows(sqlmock.NewRows([]string{"email", "phone"}).AddRow("a@b.c", "+43"))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."bankaccount"`).
		WillReturnRows(sqlmock.NewRows([]string{"iban", "owner"}))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."address" WHERE .*"type" = 'BILLING'`).
		WillReturnRows(sqlmock.NewRows([]string{"city", "street", "streetNumber", "type", "zip"}))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."address" WHERE .*"type" = 'RESIDENCE'`).
		WillReturnRows(sqlmock.NewRows([]string{"city", "street", "streetNumber", "type", "zip"}))
	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"\."meteringpoint"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"city", "direction", "equipmentName", "equipmentNumber", "inverterid",
			"metering_point_id", "modifiedAt", "modifiedBy", "registeredSince", "status",
			"street", "streetNumber", "tariff_id", "transformer", "zip",
		}))

	db, _ := mockDb.OpenMockDb()
	err = CompleteParticipant(db, p)
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", p.Contact.Email.String)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestImportParticipant_ExistingParticipantAttachesMeters(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	p := &model.EegParticipant{
		FirstName: "Sepp",
		LastName:  "Huber",
		MeteringPoint: []*model.MeteringPoint{
			{MeteringPoint: "AT00100000000000000000010001", Direction: model.CONSUMPTION},
		},
	}

	// Lookup-Query findet einen bestehenden Participant.
	mockDb.Mock.ExpectQuery(`SELECT "id" FROM "base"\."participant" WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("existing-pid"))
	// ImportMeteringPoints in a Tx.
	mockDb.Mock.ExpectBegin()
	mockDb.Mock.ExpectExec(`INSERT INTO "base"\."meteringpoint"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectCommit()

	err = ImportParticipant(mockDb.OpenMockDb, "TE100100", "admin", p)
	require.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestConfirmParticipant(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	pid := uuid.New()

	mockDb.Mock.ExpectExec(`UPDATE base\.participant SET status = 'ACTIVE'.*WHERE id = \$2`).
		WithArgs("admin", pid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = ConfirmParticipant(mockDb.OpenMockDb, "TE100100", "admin", pid)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestArchiveParticipant(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	pid := uuid.New()

	mockDb.Mock.ExpectExec(`UPDATE "base"\."participant" SET .*"status"='ARCHIVED'.* WHERE \("id" = '` + pid + `'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = ArchiveParticipant(mockDb.OpenMockDb, "admin", pid)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}
