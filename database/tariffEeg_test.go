package database

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTariffHistory(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "name", "billingPeriod", "useVat", "vatInPercent",
		"accountNetAmount", "accountGrossAmount", "participantFee", "baseFee",
		"businessNr", "version", "type", "centPerKWh", "discount", "freeKWh",
	}).
		AddRow("bd427cac-c6a7-49b1-915e-c7eeb215bb5d", "Sepp", "monthly", false, 0, 0, 0, 0, 0, 0, 2, "", 12, 0, 100).
		AddRow("bd427cac-c6a7-49b1-915e-c7eeb215bb5d", "Sepp", "monthly", false, 0, 0, 0, 0, 0, 0, 1, "", 10, 0, 50)

	mockDb.Mock.ExpectQuery(`SELECT .* FROM base\.tariff WHERE tenant = \$1 AND id = \$2 ORDER BY version DESC`).
		WithArgs("TE100100", "bd427cac-c6a7-49b1-915e-c7eeb215bb5d").
		WillReturnRows(rows)

	got, err := GetTariffHistory(mockDb.OpenMockDb, "TE100100", "bd427cac-c6a7-49b1-915e-c7eeb215bb5d")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, 2, got[0].Version)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestGetTariffNameMap(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow("t-1", "Sepp").
		AddRow("t-2", "Default")

	mockDb.Mock.ExpectQuery(`SELECT id, name FROM base\.activetariff WHERE tenant = \$1`).
		WithArgs("TE100100").
		WillReturnRows(rows)

	got, err := GetTariffNameMap(mockDb.OpenMockDb, "TE100100")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"t-1": "Sepp", "t-2": "Default"}, got)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestUpdateEegPartial(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"."eeg" SET .* WHERE \("tenant" = 'TE100100'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateEegPartial(mockDb.OpenMockDb, "TE100100",
		map[string]interface{}{"name": "T-VIERE-NEU"})
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestUpdateEegAddressPartial(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"."address" SET .* WHERE \("tenant" = 'TE100100'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateEegAddressPartial(mockDb.OpenMockDb, "TE100100",
		map[string]interface{}{"street": "Solarstraße 9"})
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestGetNotification_AdminSeesAll(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "type", "notification", "date"}).
		AddRow(int16(2), "CR_MSG", `{"foo":"bar"}`, time.Now())

	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"."notification" WHERE \(\("tenant" = 'TE100100'\) AND \("id" > 0\)\) ORDER BY "id" DESC LIMIT 30`).
		WillReturnRows(rows)

	got, err := GetNotification(mockDb.OpenMockDb, "TE100100", 0, true)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestGetNotification_NonAdminFiltersByUserRole(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "type", "notification", "date"})

	mockDb.Mock.ExpectQuery(`SELECT .* FROM "base"."notification" WHERE \(\("tenant" = 'TE100100'\) AND \("id" > 0\) AND \("role" = 'USER'\)\) ORDER BY "id" DESC LIMIT 30`).
		WillReturnRows(rows)

	_, err = GetNotification(mockDb.OpenMockDb, "TE100100", 0, false)
	require.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}
