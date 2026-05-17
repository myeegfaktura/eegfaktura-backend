package database

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeteringPointRevoke(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	inactiveSince := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

	mockDb.Mock.ExpectExec(`UPDATE "base"."meteringpoint" SET .* WHERE \(\("metering_point_id" = 'AT00100000000000000000010001'\) AND \("tenant" = 'TE100100'\)\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = MeteringPointRevoke(mockDb.OpenMockDb, "TE100100", "AT00100000000000000000010001", inactiveSince)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestUpdateMeteringPointPartial(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"."meteringpoint" SET .* WHERE \(\("metering_point_id" = 'AT00100000000000000000010001'\) AND \("participant_id" = 'p-1'\) AND \("tenant" = 'TE100100'\)\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateMeteringPointPartial(
		mockDb.OpenMockDb,
		"TE100100", "admin", "p-1", "AT00100000000000000000010001",
		map[string]interface{}{"direction": "CONSUMPTION"},
	)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestFindActiveMeteringByIds(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"city", "direction", "equipmentName", "equipmentNumber", "inverterid",
		"metering_point_id", "modifiedAt", "modifiedBy", "registeredSince", "status",
		"street", "streetNumber", "tariff_id", "transformer", "zip",
	}).AddRow(
		"Solarcity", model.CONSUMPTION, "", "", "",
		"AT00100000000000000000010001", time.Now(), "admin", time.Now(), model.ACTIVE,
		"Energieweg", "12a", "", "", "1234",
	)
	mockDb.Mock.ExpectQuery(`SELECT \* FROM "base"."meteringpoint" WHERE .*"status" = 'ACTIVE'`).
		WillReturnRows(rows)

	got, err := FindActiveMeteringByIds(mockDb.OpenMockDb, "TE100100",
		[]string{"AT00100000000000000000010001"})
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "AT00100000000000000000010001", got[0].MeteringPoint)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestUpdateMeteringPointPartFact(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`INSERT INTO "base"."metering_partition_factor" .* VALUES \('admin', 'AT00100000000000000000010001', 42, 'p-1', 'TE100100'\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = UpdateMeteringPointPartFact(mockDb.OpenMockDb,
		"TE100100", "admin", "p-1", "AT00100000000000000000010001", 42)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestMoveMeteringPoint(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectBegin()
	mockDb.Mock.ExpectExec(`UPDATE "base"."meteringpoint" SET .*"participant_id"='dest'.* WHERE \(\("metering_point_id" = 'AT00100000000000000000010001'\) AND \("participant_id" = 'src'\) AND \("tenant" = 'TE100100'\)\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mockDb.Mock.ExpectCommit()

	err = MoveMeteringPoint(mockDb.OpenMockDb,
		"TE100100", "admin", "src", "dest", "AT00100000000000000000010001")
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}
