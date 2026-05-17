package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterMeteringPoint(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	point := &model.MeteringPoint{
		MeteringPoint: "AT00100000000000000000010001",
		Direction:     model.CONSUMPTION,
		Status:        model.NEW,
	}

	mockDb.Mock.ExpectExec(`INSERT INTO "base"."meteringpoint" .* VALUES .*'AT00100000000000000000010001'.*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = RegisterMeteringPoint(mockDb.OpenMockDb, "TE100100", "p-1", point)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestUpdateMeteringPoint(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	point := &model.MeteringPoint{
		MeteringPoint: "AT00100000000000000000010001",
		Direction:     model.CONSUMPTION,
		Status:        model.ACTIVE,
	}

	mockDb.Mock.ExpectExec(`UPDATE "base"."meteringpoint" SET .* WHERE \(\("metering_point_id" = 'AT00100000000000000000010001'\) AND \("participant_id" = 'p-1'\) AND \("tenant" = 'TE100100'\)\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateMeteringPoint(mockDb.OpenMockDb, "TE100100", "p-1", "AT00100000000000000000010001", point)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

// Tx-based DAOs: caller owns the transaction. Tests use sqlmock's
// transaction primitives (ExpectBegin/ExpectExec/ExpectCommit) and pass
// the raw *sql.Tx the production code expects.
func TestRegisterMeteringPoints_SetsStatusNew(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	points := []*model.MeteringPoint{
		{MeteringPoint: "AT00100000000000000000010001", Direction: model.CONSUMPTION, Status: model.ACTIVE},
	}

	mockDb.Mock.ExpectBegin()
	// 'NEW' must appear in the VALUES clause even though the input had ACTIVE —
	// RegisterMeteringPoints overrides status to NEW before saving.
	mockDb.Mock.ExpectExec(`INSERT INTO "base"."meteringpoint" .* VALUES .*'NEW'.*`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectCommit()

	db, _ := mockDb.OpenMockDb()
	tx, err := db.Begin()
	require.NoError(t, err)

	err = RegisterMeteringPoints(tx, "TE100100", "p-1", points)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// Input mutation is part of the contract — verify the caller's slice was updated.
	assert.Equal(t, model.NEW, points[0].Status)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestImportMeteringPoints_PreservesStatus(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	points := []*model.MeteringPoint{
		{MeteringPoint: "AT00100000000000000000010001", Direction: model.CONSUMPTION, Status: model.ACTIVE},
	}

	mockDb.Mock.ExpectBegin()
	// Status passes through unchanged for the Import variant.
	mockDb.Mock.ExpectExec(`INSERT INTO "base"."meteringpoint" .* VALUES .*'ACTIVE'.*`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectCommit()

	db, _ := mockDb.OpenMockDb()
	tx, err := db.Begin()
	require.NoError(t, err)

	err = ImportMeteringPoints(tx, "TE100100", "p-1", points)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Equal(t, model.ACTIVE, points[0].Status)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}
