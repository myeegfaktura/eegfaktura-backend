package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTariff(t *testing.T) {
	tariff := model.Tariff{Version: 1, Name: "Sepp", UseVat: false, BillingPeriod: "monthly", FreeKWh: 100, CentPerKWh: 12}
	var mockDb, err = GetDatabaseMock()
	require.NoError(t, err)

	// id-Spalte wird seit AddTariff() den UUID Go-seitig setzt als
	// String-Literal eingefügt (war früher DEFAULT/serverseitig).
	stmt := "INSERT INTO (.+) VALUES \\(0, 0, 0, 'monthly', NULL, 12, 0, 100, '[0-9a-fA-F-]+', 'Sepp', 0, 'sepp', '', FALSE, 0, 1\\)"

	mockDb.Mock.ExpectExec(stmt).WillReturnResult(sqlmock.NewResult(1, 1))

	err = AddTariff(mockDb.OpenMockDb, "sepp", &tariff)
	assert.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}
