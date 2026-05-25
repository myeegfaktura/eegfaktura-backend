package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchEdaHistory_NoFilters(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	stmt := "SELECT \"conversationId\", \"date\", \"direction\", \"issuer\", \"message\", \"protocol\", \"tenant\", \"type\" FROM \"base\".\"processhistory\" WHERE \\(\\(\"tenant\" = 'RC100298'\\) AND \\(\"protocol\" IS NOT NULL\\)\\)"

	rows := sqlmock.NewRows([]string{"conversationId", "date", "direction", "issuer", "message", "protocol", "tenant", "type"}).
		AddRow("1", time.Now(), "CONSUMPTION", "ADMIN", "{}", "CR_MSG", "SEPP", model.EBMS_ONLINE_REG_APPROVAL)
	mockDb.Mock.ExpectQuery(stmt).WillReturnRows(rows)
	res, err := FetchEdaHistory(mockDb.OpenMockDb, "RC100298", 0, 0, nil)
	require.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())

	for k, v := range res {
		fmt.Printf("K: %v\n", k)
		for _, e := range v {
			fmt.Printf("    V: %v\n", e)
		}
	}
}

func TestFetchEdaHistory_ProtocolFilter(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	// IN-list replaces IS NOT NULL when protocols are passed.
	stmt := `SELECT .* FROM "base"."processhistory" WHERE \(\("tenant" = 'RC100298'\) AND \("protocol" IN \('CR_MSG', 'EC_REQ_ONL'\)\)\)`

	rows := sqlmock.NewRows([]string{"conversationId", "date", "direction", "issuer", "message", "protocol", "tenant", "type"})
	mockDb.Mock.ExpectQuery(stmt).WillReturnRows(rows)

	_, err = FetchEdaHistory(mockDb.OpenMockDb, "RC100298", 0, 0, []string{"CR_MSG", "EC_REQ_ONL"})
	require.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}

func TestFetchEdaHistory_TimeRange(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	// Both bounds present -> WHERE includes date >= start AND date <= end.
	stmt := `SELECT .* FROM "base"."processhistory" WHERE .*"date" >= .* AND \("date" <= .*\)`

	rows := sqlmock.NewRows([]string{"conversationId", "date", "direction", "issuer", "message", "protocol", "tenant", "type"})
	mockDb.Mock.ExpectQuery(stmt).WillReturnRows(rows)

	startMs := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	endMs := time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC).UnixMilli()
	_, err = FetchEdaHistory(mockDb.OpenMockDb, "RC100298", startMs, endMs, nil)
	require.NoError(t, err)
	assert.NoError(t, mockDb.Mock.ExpectationsWereMet())
}
