package database

import (
	dbsql "database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/doug-martin/goqu/v9"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/jmoiron/sqlx"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateParticipant(t *testing.T) {
	var tests = []struct {
		name     string
		line     func(table string, param interface{}) (sql string, params []interface{}, err error)
		params   interface{}
		database string
		result   []float64
	}{
		{
			name: "Test One",
			line: func(table string, param interface{}) (sql string, params []interface{}, err error) {
				sql, params, err = goqu.Insert("base.participant").Rows(param).ToSQL()
				return
			},
			params:   map[string]interface{}{"firstname": "hans"},
			database: "participant",
			result:   []float64{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, _ := tt.line(tt.database, tt.params)
			println(sql)
		})
	}
}

func TestRegisterParticipant(t *testing.T) {

	mockDb, err := GetDatabaseMock()

	participantJson := `{"businessRole":"EEG_PRIVATE","firstname":"Peter","lastname":"Obermüller","residentAddress":{"street":"Lambacherstraße","streetNumber":"39","zip":"4680","city":"Haag am Hausruck","type":"RESIDENCE"},"contact":{"phone":"06603611758","email":"obermueller.peter@gmail.com"},"accountInfo":{},"optionals":{},"status":"NEW","id":"e98b8619-7b6a-4836-baff-5489fb539535","role":"EEG_USER","billingAddress":{"street":"Lambacherstraße","streetNumber":"39","zip":"4680","city":"Haag am Hausruck","type":"BILLING"},"meters":[{"direction":"CONSUMPTION","status":"NEW","meteringPoint":"AT48124817243712897412","participantId":"e98b8619-7b6a-4836-baff-5489fb539535","tariffId":"a48d1990-a5a2-40c9-8d0a-77bed8e7dbcd","street":"Lambacherstraße","streetNumber":"39","zip":"4680","city":"Haag am Hausruck"}]}`

	var p model.EegParticipant
	err = json.NewDecoder(strings.NewReader(participantJson)).Decode(&p)
	assert.NoError(t, err)

	fmt.Printf("Participant: %+v\n", p)

	mockDb.Mock.ExpectBegin()
	mockDb.Mock.ExpectQuery("INSERT (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).FromCSVString("1")) //.WillReturnResult(sqlmock.NewResult(1, 1)) //.WithArgs("firstname", "lastname")
	mockDb.Mock.ExpectExec("INSERT (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectExec("INSERT (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectExec("INSERT (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectExec("INSERT (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mockDb.Mock.ExpectCommit()

	err = RegisterParticipant(mockDb.OpenMockDb, "RC200200", "petero", &p)
	assert.NoError(t, err)
}

func TestGetParticipant(t *testing.T) {
	mockDb, err := GetDatabaseMock()

	participantRows := sqlmock.NewRows([]string{
		"id", "firstname", "lastname", "role", "businessRole", "titleBefore", "titleAfter", "participantSince",
		"vatNumber", "taxNumber", "companyRegisterNumber", "status", "createdBy", //"createdDate", "lastModifiedBy", "lastModifiedDate",
		"version", "tariffId", "participantNumber"}).
		AddRow(uuid.New(), "Sepp", "Huber", "EEG_USER", "EEG_PRIVATE", "", "", time.Now(),
			"", "", "", "NEW", "admin", //time.Now(), "petero", time.Now(),
			1, uuid.New(), "001")
	mockDb.Mock.ExpectQuery("SELECT (.+) FROM \"base\".\"participant\" (.+)").WillReturnRows(participantRows)

	contactDetailsRows := sqlmock.NewRows([]string{"email", "phone"}).AddRow("mail@test.com", "+4325622 232311 32323")
	mockDb.Mock.ExpectQuery("SELECT (.+) FROM \"base\".\"contactdetail\" (.+)").WillReturnRows(contactDetailsRows)

	bankaccountRows := sqlmock.NewRows([]string{"iban", "owner"}).AddRow("AT12 3456 7987 9887 7765", "Sepp Huber")
	mockDb.Mock.ExpectQuery("SELECT (.+) FROM \"base\".\"bankaccount\" (.+)").WillReturnRows(bankaccountRows)

	addressRows := sqlmock.NewRows([]string{"city", "street", "streetNumber", "type", "zip"}).
		AddRow("Solarcity", "Energieweg", "12a", "BILLING", "1234")
	mockDb.Mock.ExpectQuery("SELECT (.+) FROM \"base\".\"address\" (.+)").WillReturnRows(addressRows)

	addressResidenceRows := sqlmock.NewRows([]string{"city", "street", "streetNumber", "type", "zip"}).
		AddRow("Solarcity", "Energieweg", "12a", "RESIDENCE", "1234")
	mockDb.Mock.ExpectQuery("SELECT (.+) FROM \"base\".\"address\" (.+)").WillReturnRows(addressResidenceRows)

	meterRows := sqlmock.NewRows([]string{"city", "direction", "equipmentName", "equipmentNumber", "inverterid", "metering_point_id",
		"modifiedAt", "modifiedBy", "registeredSince", "status", "street", "streetNumber", "tariff_id", "transformer", "zip"}).
		AddRow("Solarcity", "GENERATOR", "", "", "", "AT0020001110000010011111001",
			time.Now(), "admin", time.Now(), "NEW", "Energieweg", "12a", uuid.New(), "", "1234")
	mockDb.Mock.ExpectQuery("SELECT (.+) FROM \"base\".\"meteringpoint\" (.+)").WillReturnRows(meterRows)

	// Follow-up SELECT by populateMeterStates for the activation window.
	meterStateRows := sqlmock.NewRows([]string{"metering_point_id", "activesince", "inactivesince"}).
		AddRow("AT0020001110000010011111001", nil, nil)
	mockDb.Mock.ExpectQuery("SELECT \"metering_point_id\", \"activesince\", \"inactivesince\" FROM \"base\".\"meteringpoint\" (.+)").WillReturnRows(meterStateRows)

	participants, err := GetParticipant(mockDb.OpenMockDb, "RC100298")
	assert.NoError(t, err)

	assert.NotEmpty(t, participants)
	fmt.Printf("Participants: %+v\n", participants)
}

func Test_saveParticipant(t *testing.T) {
	type args struct {
		db                         *sqlx.DB
		tenant                     string
		username                   string
		participant                *model.EegParticipant
		registerMeteringPointsFunc func(*dbsql.Tx, string, string, []*model.MeteringPoint) error
	}

	mDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	participantJson := `{"businessRole":"EEG_PRIVATE","firstname":"Peter","lastname":"Obermüller","residentAddress":{"street":"Lambacherstraße","streetNumber":"39","zip":"4680","city":"Haag am Hausruck","type":"RESIDENCE"},"contact":{"phone":"06603611758","email":"obermueller.peter@gmail.com"},"accountInfo":{},"optionals":{},"status":"NEW","id":"e98b8619-7b6a-4836-baff-5489fb539535","role":"EEG_USER","billingAddress":{"street":"Lambacherstraße","streetNumber":"39","zip":"4680","city":"Haag am Hausruck","type":"BILLING"},"meters":[{"direction":"CONSUMPTION","status":"NEW","meteringPoint":"AT48124817243712897412","participantId":"e98b8619-7b6a-4836-baff-5489fb539535","tariffId":"a48d1990-a5a2-40c9-8d0a-77bed8e7dbcd","street":"Lambacherstraße","streetNumber":"39","zip":"4680","city":"Haag am Hausruck"}]}`

	var p model.EegParticipant
	err = json.NewDecoder(strings.NewReader(participantJson)).Decode(&p)
	assert.NoError(t, err)

	mdb := sqlx.NewDb(mDB, "mock")

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT (.+) \"base\".\"participant\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("11"))
	mock.ExpectExec("INSERT (.+) \"base\".\"contactdetail\"").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT (.+) \"base\".\"bankaccount\"").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT (.+) \"base\".\"address\"").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT (.+) \"base\".\"meteringpoint\"").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "Save Participant", // TODO: Add test cases.
			args:    args{db: mdb, tenant: "te100001", username: "tester", participant: &p, registerMeteringPointsFunc: ImportMeteringPoints},
			wantErr: assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := saveParticipant(tt.args.db, tt.args.tenant, tt.args.username, tt.args.participant, tt.args.registerMeteringPointsFunc)
			assert.NoError(t, mock.ExpectationsWereMet())
			require.NoError(t, err)

		})
	}
}
