package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateParticipantPartial_TopLevel(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"\."participant" SET "firstname"='NewName' WHERE .*"id" = '11111111-1111-1111-1111-111111111111'.*"tenant" = 'TE100300'`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "11111111-1111-1111-1111-111111111111", "firstname", "NewName")
	assert.NoError(t, err)
}

func TestUpdateParticipantPartial_BankAccountMandateDate(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"\."bankaccount" SET "mandate_date"='2026-05-28' WHERE \("participant_id" = '22222222-2222-2222-2222-222222222222'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "22222222-2222-2222-2222-222222222222", "accountInfo.mandateDate", "2026-05-28")
	assert.NoError(t, err)
}

func TestUpdateParticipantPartial_BillingAddressCity(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"\."address" SET "city"='Linz' WHERE .*"participant_id" = '33333333-3333-3333-3333-333333333333'.*"type" = 'BILLING'`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "33333333-3333-3333-3333-333333333333", "billingAddress.city", "Linz")
	assert.NoError(t, err)
}

func TestUpdateParticipantPartial_ContactEmail(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"\."contactdetail" SET "email"='new@x.de' WHERE \("participant_id" = '44444444-4444-4444-4444-444444444444'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "44444444-4444-4444-4444-444444444444", "contact.email", "new@x.de")
	assert.NoError(t, err)
}

func TestUpdateParticipantPartial_UnknownPathReturns1102(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "55555555-5555-5555-5555-555555555555", "doesNotExist", "value")
	require.Error(t, err)

	pe, ok := err.(*PartialUpdateError)
	require.True(t, ok, "expected PartialUpdateError, got %T", err)
	assert.Equal(t, 1102, pe.Code)
	assert.Contains(t, pe.Message, "doesNotExist")
}

func TestUpdateParticipantPartial_UnknownSubPathReturns1102(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "66666666-6666-6666-6666-666666666666", "accountInfo.nopeNotAColumn", "v")
	require.Error(t, err)

	pe, ok := err.(*PartialUpdateError)
	require.True(t, ok)
	assert.Equal(t, 1102, pe.Code)
}

func TestUpdateParticipantPartial_NoRowAffectedReturns1103(t *testing.T) {
	mockDb, err := GetDatabaseMock()
	require.NoError(t, err)

	mockDb.Mock.ExpectExec(`UPDATE "base"\."participant"`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = UpdateParticipantPartial(mockDb.OpenMockDb, "TE100300", "77777777-7777-7777-7777-777777777777", "firstname", "X")
	require.Error(t, err)

	pe, ok := err.(*PartialUpdateError)
	require.True(t, ok)
	assert.Equal(t, 1103, pe.Code)
}
