package eda

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type RecorderMock struct {
	mock.Mock
	dbOpen database.OpenDbXConnection
}

func (_m *RecorderMock) saveNotification(notificationValue map[string]interface{}, tenant, notificationType, role string) error {
	_ = _m.Called(notificationValue, tenant, notificationType, role)
	//var r0 error
	//if rf, ok := ret.Get(0).(func(string, string) error); ok {
	//	r0 = rf(paperSize, content)
	//} else {
	//	r0 = ret.Error(0)
	//}
	//
	//return r0

	return nil
}
func (_m *RecorderMock) saveHistory(tenant string, messageCode model.EbMsMessageType, conversationId, role, dir string, protocol model.EdaProtocol, msg interface{}) error {
	_ = _m.Called(tenant, messageCode, conversationId, role, dir, protocol, msg)
	//var r0 error
	//if rf, ok := ret.Get(0).(func(string, string) error); ok {
	//	r0 = rf(paperSize, content)
	//} else {
	//	r0 = ret.Error(0)
	//}
	//
	//return r0
	return nil
}

func (_m *RecorderMock) databaseConnect() (*sqlx.DB, error) {
	return _m.dbOpen()
}

func TestProtcolCrMsgHandler(t *testing.T) {
	var mockDb, err = database.GetDatabaseMock()
	require.NoError(t, err)
	recorder := &RecorderMock{dbOpen: mockDb.OpenMockDb}

	jsonString := `{"messageId":"AT003000202208201421374610104995950","conversationId":"AT003000202208191420233640008300242","sender":"AT003000","receiver":"RC100130","messageCode":"DATEN_CRMSG","meter":{"meteringPoint":"AT0030000000000000000000000200959"},"energy":{"start":1660773600000,"end":1660860000000,"interval":"QH","nInterval":288,"data":[{"meterCode":"1-1:1.9.0 G.01","value":[{"from":1660773600000,"to":1660774500000,"method":"L1","value":0.00525},{"from":1660774500000,"to":1660775400000,"method":"L1","value":0.0055},{"from":1660775400000,"to":1660776300000,"method":"L1","value":0.0055},{"from":1660776300000,"to":1660777200000,"method":"L1","value":0.00925},{"from":1660777200000,"to":1660778100000,"method":"L1","value":0.0075},{"from":1660778100000,"to":1660779000000,"method":"L1","value":0.005},{"from":1660779000000,"to":1660779900000,"method":"L1","value":0.006},{"from":1660779900000,"to":1660780800000,"method":"L1","value":0.0055},{"from":1660780800000,"to":1660781700000,"method":"L1","value":0.006},{"from":1660781700000,"to":1660782600000,"method":"L1","value":0.00525},{"from":1660782600000,"to":1660783500000,"method":"L1","value":0.00625},{"from":1660783500000,"to":1660784400000,"method":"L1","value":0.00625},{"from":1660784400000,"to":1660785300000,"method":"L1","value":0.0065},{"from":1660785300000,"to":1660786200000,"method":"L1","value":0.006},{"from":1660786200000,"to":1660787100000,"method":"L1","value":0.006},{"from":1660787100000,"to":1660788000000,"method":"L1","value":0.0085},{"from":1660788000000,"to":1660788900000,"method":"L1","value":0.00875},{"from":1660788900000,"to":1660789800000,"method":"L1","value":0.00975},{"from":1660789800000,"to":1660790700000,"method":"L1","value":0.01},{"from":1660790700000,"to":1660791600000,"method":"L1","value":0.009},{"from":1660791600000,"to":1660792500000,"method":"L1","value":0.008},{"from":1660792500000,"to":1660793400000,"method":"L1","value":0.0065},{"from":1660793400000,"to":1660794300000,"method":"L1","value":0.007},{"from":1660794300000,"to":1660795200000,"method":"L1","value":0.0065},{"from":1660795200000,"to":1660796100000,"method":"L1","value":0.00725},{"from":1660796100000,"to":1660797000000,"method":"L1","value":0.00725},{"from":1660797000000,"to":1660797900000,"method":"L1","value":0.00625},{"from":1660797900000,"to":1660798800000,"method":"L1","value":0.006},{"from":1660798800000,"to":1660799700000,"method":"L1","value":0},{"from":1660799700000,"to":1660800600000,"method":"L1","value":0.00025},{"from":1660800600000,"to":1660801500000,"method":"L1","value":0.00175},{"from":1660801500000,"to":1660802400000,"method":"L1","value":0.00075},{"from":1660802400000,"to":1660803300000,"method":"L1","value":0.00325},{"from":1660803300000,"to":1660804200000,"method":"L1","value":0.00725},{"from":1660804200000,"to":1660805100000,"method":"L1","value":0.01675},{"from":1660805100000,"to":1660806000000,"method":"L1","value":0.0155},{"from":1660806000000,"to":1660806900000,"method":"L1","value":0},{"from":1660806900000,"to":1660807800000,"method":"L1","value":0},{"from":1660807800000,"to":1660808700000,"method":"L1","value":0},{"from":1660808700000,"to":1660809600000,"method":"L1","value":0},{"from":1660809600000,"to":1660810500000,"method":"L1","value":0},{"from":1660810500000,"to":1660811400000,"method":"L1","value":0},{"from":1660811400000,"to":1660812300000,"method":"L1","value":0},{"from":1660812300000,"to":1660813200000,"method":"L1","value":0},{"from":1660813200000,"to":1660814100000,"method":"L1","value":0},{"from":1660814100000,"to":1660815000000,"method":"L1","value":0},{"from":1660815000000,"to":1660815900000,"method":"L1","value":0},{"from":1660815900000,"to":1660816800000,"method":"L1","value":0},{"from":1660816800000,"to":1660817700000,"method":"L1","value":0},{"from":1660817700000,"to":1660818600000,"method":"L1","value":0},{"from":1660818600000,"to":1660819500000,"method":"L1","value":0},{"from":1660819500000,"to":1660820400000,"method":"L1","value":0},{"from":1660820400000,"to":1660821300000,"method":"L1","value":0},{"from":1660821300000,"to":1660822200000,"method":"L1","value":0},{"from":1660822200000,"to":1660823100000,"method":"L1","value":0},{"from":1660823100000,"to":1660824000000,"method":"L1","value":0},{"from":1660824000000,"to":1660824900000,"method":"L1","value":0},{"from":1660824900000,"to":1660825800000,"method":"L1","value":0},{"from":1660825800000,"to":1660826700000,"method":"L1","value":0},{"from":1660826700000,"to":1660827600000,"method":"L1","value":0},{"from":1660827600000,"to":1660828500000,"method":"L1","value":0},{"from":1660828500000,"to":1660829400000,"method":"L1","value":0},{"from":1660829400000,"to":1660830300000,"method":"L1","value":0},{"from":1660830300000,"to":1660831200000,"method":"L1","value":0},{"from":1660831200000,"to":1660832100000,"method":"L1","value":0},{"from":1660832100000,"to":1660833000000,"method":"L1","value":0},{"from":1660833000000,"to":1660833900000,"method":"L1","value":0.0035},{"from":1660833900000,"to":1660834800000,"method":"L1","value":0.00275},{"from":1660834800000,"to":1660835700000,"method":"L1","value":0},{"from":1660835700000,"to":1660836600000,"method":"L1","value":0},{"from":1660836600000,"to":1660837500000,"method":"L1","value":0},{"from":1660837500000,"to":1660838400000,"method":"L1","value":0},{"from":1660838400000,"to":1660839300000,"method":"L1","value":0},{"from":1660839300000,"to":1660840200000,"method":"L1","value":0},{"from":1660840200000,"to":1660841100000,"method":"L1","value":0},{"from":1660841100000,"to":1660842000000,"method":"L1","value":0},{"from":1660842000000,"to":1660842900000,"method":"L1","value":0},{"from":1660842900000,"to":1660843800000,"method":"L1","value":0},{"from":1660843800000,"to":1660844700000,"method":"L1","value":0.0015},{"from":1660844700000,"to":1660845600000,"method":"L1","value":0.00825},{"from":1660845600000,"to":1660846500000,"method":"L1","value":0.0075},{"from":1660846500000,"to":1660847400000,"method":"L1","value":0.00725},{"from":1660847400000,"to":1660848300000,"method":"L1","value":0.00675},{"from":1660848300000,"to":1660849200000,"method":"L1","value":0.0065},{"from":1660849200000,"to":1660850100000,"method":"L1","value":0.0075},{"from":1660850100000,"to":1660851000000,"method":"L1","value":0.006},{"from":1660851000000,"to":1660851900000,"method":"L1","value":0.008},{"from":1660851900000,"to":1660852800000,"method":"L1","value":0.0095},{"from":1660852800000,"to":1660853700000,"method":"L1","value":0.00975},{"from":1660853700000,"to":1660854600000,"method":"L1","value":0.00825},{"from":1660854600000,"to":1660855500000,"method":"L1","value":0.01},{"from":1660855500000,"to":1660856400000,"method":"L1","value":0.009},{"from":1660856400000,"to":1660857300000,"method":"L1","value":0.00625},{"from":1660857300000,"to":1660858200000,"method":"L1","value":0.00575},{"from":1660858200000,"to":1660859100000,"method":"L1","value":0.00625},{"from":1660859100000,"to":1660860000000,"method":"L1","value":0.006}]},{"meterCode":"1-1:2.9.0 G.02","value":[{"from":1660773600000,"to":1660774500000,"method":"L1","value":0},{"from":1660774500000,"to":1660775400000,"method":"L1","value":0},{"from":1660775400000,"to":1660776300000,"method":"L1","value":0},{"from":1660776300000,"to":1660777200000,"method":"L1","value":0},{"from":1660777200000,"to":1660778100000,"method":"L1","value":0},{"from":1660778100000,"to":1660779000000,"method":"L1","value":0},{"from":1660779000000,"to":1660779900000,"method":"L1","value":0},{"from":1660779900000,"to":1660780800000,"method":"L1","value":0},{"from":1660780800000,"to":1660781700000,"method":"L1","value":0},{"from":1660781700000,"to":1660782600000,"method":"L1","value":0},{"from":1660782600000,"to":1660783500000,"method":"L1","value":0},{"from":1660783500000,"to":1660784400000,"method":"L1","value":0},{"from":1660784400000,"to":1660785300000,"method":"L1","value":0},{"from":1660785300000,"to":1660786200000,"method":"L1","value":0},{"from":1660786200000,"to":1660787100000,"method":"L1","value":0},{"from":1660787100000,"to":1660788000000,"method":"L1","value":0},{"from":1660788000000,"to":1660788900000,"method":"L1","value":0},{"from":1660788900000,"to":1660789800000,"method":"L1","value":0},{"from":1660789800000,"to":1660790700000,"method":"L1","value":0},{"from":1660790700000,"to":1660791600000,"method":"L1","value":0},{"from":1660791600000,"to":1660792500000,"method":"L1","value":0},{"from":1660792500000,"to":1660793400000,"method":"L1","value":0},{"from":1660793400000,"to":1660794300000,"method":"L1","value":0},{"from":1660794300000,"to":1660795200000,"method":"L1","value":0},{"from":1660795200000,"to":1660796100000,"method":"L1","value":0},{"from":1660796100000,"to":1660797000000,"method":"L1","value":0},{"from":1660797000000,"to":1660797900000,"method":"L1","value":0},{"from":1660797900000,"to":1660798800000,"method":"L1","value":0},{"from":1660798800000,"to":1660799700000,"method":"L1","value":0},{"from":1660799700000,"to":1660800600000,"method":"L1","value":0},{"from":1660800600000,"to":1660801500000,"method":"L1","value":0},{"from":1660801500000,"to":1660802400000,"method":"L1","value":0},{"from":1660802400000,"to":1660803300000,"method":"L1","value":0},{"from":1660803300000,"to":1660804200000,"method":"L1","value":0},{"from":1660804200000,"to":1660805100000,"method":"L1","value":0},{"from":1660805100000,"to":1660806000000,"method":"L1","value":0.0005},{"from":1660806000000,"to":1660806900000,"method":"L1","value":0},{"from":1660806900000,"to":1660807800000,"method":"L1","value":0},{"from":1660807800000,"to":1660808700000,"method":"L1","value":0},{"from":1660808700000,"to":1660809600000,"method":"L1","value":0},{"from":1660809600000,"to":1660810500000,"method":"L1","value":0},{"from":1660810500000,"to":1660811400000,"method":"L1","value":0},{"from":1660811400000,"to":1660812300000,"method":"L1","value":0},{"from":1660812300000,"to":1660813200000,"method":"L1","value":0},{"from":1660813200000,"to":1660814100000,"method":"L1","value":0},{"from":1660814100000,"to":1660815000000,"method":"L1","value":0},{"from":1660815000000,"to":1660815900000,"method":"L1","value":0},{"from":1660815900000,"to":1660816800000,"method":"L1","value":0},{"from":1660816800000,"to":1660817700000,"method":"L1","value":0},{"from":1660817700000,"to":1660818600000,"method":"L1","value":0},{"from":1660818600000,"to":1660819500000,"method":"L1","value":0},{"from":1660819500000,"to":1660820400000,"method":"L1","value":0},{"from":1660820400000,"to":1660821300000,"method":"L1","value":0},{"from":1660821300000,"to":1660822200000,"method":"L1","value":0},{"from":1660822200000,"to":1660823100000,"method":"L1","value":0},{"from":1660823100000,"to":1660824000000,"method":"L1","value":0},{"from":1660824000000,"to":1660824900000,"method":"L1","value":0},{"from":1660824900000,"to":1660825800000,"method":"L1","value":0},{"from":1660825800000,"to":1660826700000,"method":"L1","value":0},{"from":1660826700000,"to":1660827600000,"method":"L1","value":0},{"from":1660827600000,"to":1660828500000,"method":"L1","value":0},{"from":1660828500000,"to":1660829400000,"method":"L1","value":0},{"from":1660829400000,"to":1660830300000,"method":"L1","value":0},{"from":1660830300000,"to":1660831200000,"method":"L1","value":0},{"from":1660831200000,"to":1660832100000,"method":"L1","value":0},{"from":1660832100000,"to":1660833000000,"method":"L1","value":0},{"from":1660833000000,"to":1660833900000,"method":"L1","value":0},{"from":1660833900000,"to":1660834800000,"method":"L1","value":0},{"from":1660834800000,"to":1660835700000,"method":"L1","value":0},{"from":1660835700000,"to":1660836600000,"method":"L1","value":0},{"from":1660836600000,"to":1660837500000,"method":"L1","value":0},{"from":1660837500000,"to":1660838400000,"method":"L1","value":0},{"from":1660838400000,"to":1660839300000,"method":"L1","value":0},{"from":1660839300000,"to":1660840200000,"method":"L1","value":0},{"from":1660840200000,"to":1660841100000,"method":"L1","value":0},{"from":1660841100000,"to":1660842000000,"method":"L1","value":0},{"from":1660842000000,"to":1660842900000,"method":"L1","value":0},{"from":1660842900000,"to":1660843800000,"method":"L1","value":0},{"from":1660843800000,"to":1660844700000,"method":"L1","value":0},{"from":1660844700000,"to":1660845600000,"method":"L1","value":0},{"from":1660845600000,"to":1660846500000,"method":"L1","value":0},{"from":1660846500000,"to":1660847400000,"method":"L1","value":0},{"from":1660847400000,"to":1660848300000,"method":"L1","value":0},{"from":1660848300000,"to":1660849200000,"method":"L1","value":0},{"from":1660849200000,"to":1660850100000,"method":"L1","value":0},{"from":1660850100000,"to":1660851000000,"method":"L1","value":0},{"from":1660851000000,"to":1660851900000,"method":"L1","value":0},{"from":1660851900000,"to":1660852800000,"method":"L1","value":0},{"from":1660852800000,"to":1660853700000,"method":"L1","value":0},{"from":1660853700000,"to":1660854600000,"method":"L1","value":0},{"from":1660854600000,"to":1660855500000,"method":"L1","value":0},{"from":1660855500000,"to":1660856400000,"method":"L1","value":0},{"from":1660856400000,"to":1660857300000,"method":"L1","value":0},{"from":1660857300000,"to":1660858200000,"method":"L1","value":0},{"from":1660858200000,"to":1660859100000,"method":"L1","value":0},{"from":1660859100000,"to":1660860000000,"method":"L1","value":0}]},{"meterCode":"1-1:2.9.0 G.03","value":[{"from":1660773600000,"to":1660774500000,"method":"L1","value":0},{"from":1660774500000,"to":1660775400000,"method":"L1","value":0},{"from":1660775400000,"to":1660776300000,"method":"L1","value":0},{"from":1660776300000,"to":1660777200000,"method":"L1","value":0},{"from":1660777200000,"to":1660778100000,"method":"L1","value":0},{"from":1660778100000,"to":1660779000000,"method":"L1","value":0},{"from":1660779000000,"to":1660779900000,"method":"L1","value":0},{"from":1660779900000,"to":1660780800000,"method":"L1","value":0},{"from":1660780800000,"to":1660781700000,"method":"L1","value":0},{"from":1660781700000,"to":1660782600000,"method":"L1","value":0},{"from":1660782600000,"to":1660783500000,"method":"L1","value":0},{"from":1660783500000,"to":1660784400000,"method":"L1","value":0},{"from":1660784400000,"to":1660785300000,"method":"L1","value":0},{"from":1660785300000,"to":1660786200000,"method":"L1","value":0},{"from":1660786200000,"to":1660787100000,"method":"L1","value":0},{"from":1660787100000,"to":1660788000000,"method":"L1","value":0},{"from":1660788000000,"to":1660788900000,"method":"L1","value":0},{"from":1660788900000,"to":1660789800000,"method":"L1","value":0},{"from":1660789800000,"to":1660790700000,"method":"L1","value":0},{"from":1660790700000,"to":1660791600000,"method":"L1","value":0},{"from":1660791600000,"to":1660792500000,"method":"L1","value":0},{"from":1660792500000,"to":1660793400000,"method":"L1","value":0},{"from":1660793400000,"to":1660794300000,"method":"L1","value":0},{"from":1660794300000,"to":1660795200000,"method":"L1","value":0},{"from":1660795200000,"to":1660796100000,"method":"L1","value":0},{"from":1660796100000,"to":1660797000000,"method":"L1","value":0},{"from":1660797000000,"to":1660797900000,"method":"L1","value":0},{"from":1660797900000,"to":1660798800000,"method":"L1","value":0},{"from":1660798800000,"to":1660799700000,"method":"L1","value":0},{"from":1660799700000,"to":1660800600000,"method":"L1","value":0},{"from":1660800600000,"to":1660801500000,"method":"L1","value":0},{"from":1660801500000,"to":1660802400000,"method":"L1","value":0},{"from":1660802400000,"to":1660803300000,"method":"L1","value":0},{"from":1660803300000,"to":1660804200000,"method":"L1","value":0},{"from":1660804200000,"to":1660805100000,"method":"L1","value":0},{"from":1660805100000,"to":1660806000000,"method":"L1","value":0.0005},{"from":1660806000000,"to":1660806900000,"method":"L1","value":0},{"from":1660806900000,"to":1660807800000,"method":"L1","value":0},{"from":1660807800000,"to":1660808700000,"method":"L1","value":0},{"from":1660808700000,"to":1660809600000,"method":"L1","value":0},{"from":1660809600000,"to":1660810500000,"method":"L1","value":0},{"from":1660810500000,"to":1660811400000,"method":"L1","value":0},{"from":1660811400000,"to":1660812300000,"method":"L1","value":0},{"from":1660812300000,"to":1660813200000,"method":"L1","value":0},{"from":1660813200000,"to":1660814100000,"method":"L1","value":0},{"from":1660814100000,"to":1660815000000,"method":"L1","value":0},{"from":1660815000000,"to":1660815900000,"method":"L1","value":0},{"from":1660815900000,"to":1660816800000,"method":"L1","value":0},{"from":1660816800000,"to":1660817700000,"method":"L1","value":0},{"from":1660817700000,"to":1660818600000,"method":"L1","value":0},{"from":1660818600000,"to":1660819500000,"method":"L1","value":0},{"from":1660819500000,"to":1660820400000,"method":"L1","value":0},{"from":1660820400000,"to":1660821300000,"method":"L1","value":0},{"from":1660821300000,"to":1660822200000,"method":"L1","value":0},{"from":1660822200000,"to":1660823100000,"method":"L1","value":0},{"from":1660823100000,"to":1660824000000,"method":"L1","value":0},{"from":1660824000000,"to":1660824900000,"method":"L1","value":0},{"from":1660824900000,"to":1660825800000,"method":"L1","value":0},{"from":1660825800000,"to":1660826700000,"method":"L1","value":0},{"from":1660826700000,"to":1660827600000,"method":"L1","value":0},{"from":1660827600000,"to":1660828500000,"method":"L1","value":0},{"from":1660828500000,"to":1660829400000,"method":"L1","value":0},{"from":1660829400000,"to":1660830300000,"method":"L1","value":0},{"from":1660830300000,"to":1660831200000,"method":"L1","value":0},{"from":1660831200000,"to":1660832100000,"method":"L1","value":0},{"from":1660832100000,"to":1660833000000,"method":"L1","value":0},{"from":1660833000000,"to":1660833900000,"method":"L1","value":0},{"from":1660833900000,"to":1660834800000,"method":"L1","value":0},{"from":1660834800000,"to":1660835700000,"method":"L1","value":0},{"from":1660835700000,"to":1660836600000,"method":"L1","value":0},{"from":1660836600000,"to":1660837500000,"method":"L1","value":0},{"from":1660837500000,"to":1660838400000,"method":"L1","value":0},{"from":1660838400000,"to":1660839300000,"method":"L1","value":0},{"from":1660839300000,"to":1660840200000,"method":"L1","value":0},{"from":1660840200000,"to":1660841100000,"method":"L1","value":0},{"from":1660841100000,"to":1660842000000,"method":"L1","value":0},{"from":1660842000000,"to":1660842900000,"method":"L1","value":0},{"from":1660842900000,"to":1660843800000,"method":"L1","value":0},{"from":1660843800000,"to":1660844700000,"method":"L1","value":0},{"from":1660844700000,"to":1660845600000,"method":"L1","value":0},{"from":1660845600000,"to":1660846500000,"method":"L1","value":0},{"from":1660846500000,"to":1660847400000,"method":"L1","value":0},{"from":1660847400000,"to":1660848300000,"method":"L1","value":0},{"from":1660848300000,"to":1660849200000,"method":"L1","value":0},{"from":1660849200000,"to":1660850100000,"method":"L1","value":0},{"from":1660850100000,"to":1660851000000,"method":"L1","value":0},{"from":1660851000000,"to":1660851900000,"method":"L1","value":0},{"from":1660851900000,"to":1660852800000,"method":"L1","value":0},{"from":1660852800000,"to":1660853700000,"method":"L1","value":0},{"from":1660853700000,"to":1660854600000,"method":"L1","value":0},{"from":1660854600000,"to":1660855500000,"method":"L1","value":0},{"from":1660855500000,"to":1660856400000,"method":"L1","value":0},{"from":1660856400000,"to":1660857300000,"method":"L1","value":0},{"from":1660857300000,"to":1660858200000,"method":"L1","value":0},{"from":1660858200000,"to":1660859100000,"method":"L1","value":0},{"from":1660859100000,"to":1660860000000,"method":"L1","value":0}]}]}}`
	msg := model.SubscribeMessage{
		MessageCode: model.EBMS_ENERGY_FILE_RESPONSE,
		Protocol:    model.CR_MSG,
		Tenant:      "TE1000001",
		Payload:     model.EbmsMessage{},
	}
	err = json.Unmarshal([]byte(jsonString), &msg.Payload)
	require.NoError(t, err)

	historyValue := map[string]interface{}{"meter": msg.Payload.Meter.MeteringPoint, "from": msg.Payload.Energy[0].Start, "to": msg.Payload.Energy[0].End}
	recorder.Mock.On("saveHistory", "TE1000001", model.EBMS_ENERGY_FILE_RESPONSE, "AT003000202208191420233640008300242", "ADMIN", "IN", model.CR_MSG, historyValue)

	protocolCrMsgHandler(msg, recorder)
	recorder.AssertExpectations(t)
}

func TestProtocolCrReqPtHandler(t *testing.T) {
	type test struct {
		name        string
		message     string
		codes       []string
		messageType model.EbMsMessageType
	}

	tests := []test{
		{
			name:        "Antwort",
			message:     `{"conversationId":"AT003000202208191420233640008300242","messageId":"AT003000202308140722134490185248575","sender":"AT003000","receiver":"RC100298","messageCode":"ANTWORT_PT","meter":{"meteringPoint":"AT0030000000000000000000000446232","direction":"CONSUMPTION"},"responseData":[{"responseCode":[70]}]}`,
			codes:       []string{"Änderung/Anforderung akzeptiert"},
			messageType: model.EBMS_ZP_RES,
		},
		{
			name:        "Ablehnung",
			message:     `{"conversationId":"AT003000202208191420233640008300242","messageId":"AT003000202308140722134490185248575","sender":"AT003000","receiver":"RC100298","messageCode":"ABLEHNUNG_PT","meter":{"meteringPoint":"AT0030000000000000000000000446232","direction":"CONSUMPTION"},"responseData":[{"responseCode":[56]}]}`,
			codes:       []string{"Zählpunkt nicht gefunden"},
			messageType: model.EBMS_ZP_REJ,
		},
		{
			name:        "Anforderung",
			message:     `{"conversationId":"AT003000202208191420233640008300242","messageId":"RC100298202308141691990530000000319","sender":"RC100298","receiver":"AT003000","messageCode":"ANFORDERUNG_PT","requestId":"JOVM6US5","meter":{"meteringPoint":"AT0030000000000000000000000446232","direction":"CONSUMPTION"},"timeline":{"from":1691445600000,"to":1691703900000}}`,
			codes:       []string{},
			messageType: model.EBMS_ZP_SYNC,
		},
	}

	for _, m := range tests {
		t.Run(m.name, func(t *testing.T) {
			var mockDb, err = database.GetDatabaseMock()
			require.NoError(t, err)
			recorder := &RecorderMock{dbOpen: mockDb.OpenMockDb}

			msg := model.SubscribeMessage{
				MessageCode: m.messageType,
				Protocol:    model.CR_REQ_PT,
				Tenant:      "TE1000001",
				Payload:     model.EbmsMessage{},
			}
			err = json.Unmarshal([]byte(m.message), &msg.Payload)
			require.NoError(t, err)

			recorder.Mock.On("saveNotification", map[string]interface{}{
				"type":           msg.MessageCode,
				"meteringPoints": msg.Payload.Meters(),
				"responseCodes":  m.codes,
			}, msg.Tenant, "NOTIFICATION", "ADMIN")
			recorder.Mock.On("saveHistory", "TE1000001", msg.MessageCode, "AT003000202208191420233640008300242", "ADMIN", "IN", msg.Protocol, msg.Payload)

			protocolCrReqPtHandler(msg, recorder)
			recorder.AssertExpectations(t)
		})
	}
}

func TestProtocolEcReqOnlHandler(t *testing.T) {

	type test struct {
		name        string
		message     string
		codes       []string
		messageType model.EbMsMessageType
	}

	tests := []test{
		{
			name:        "Zustimmung",
			message:     `{"conversationId":"RC100298202308171692252620000000321","messageId":"AT003000202308170810324070187796715","sender":"AT003000","receiver":"RC100298","messageCode":"ZUSTIMMUNG_ECON","requestId":"XV3VFJN2","responseData":[{"meteringPoint":"AT0030000000000000000000000459143","responseCode":[175]}]}`,
			codes:       []string{"Zustimmung erteilt"},
			messageType: model.EBMS_ONLINE_REG_APPROVAL,
		},
		{
			name:        "Antwort",
			message:     `{"conversationId":"RC100298202308171692252620000000321","messageId":"AT003000202307070957427130168201034","sender":"AT003000","receiver":"RC100298","messageCode":"ANTWORT_ECON","requestId":"6P2EU64Z","responseData":[{"meteringPoint":"AT0030000000000000000000000410702","responseCode":[99]}]}`,
			codes:       []string{"Meldung erhalten"},
			messageType: model.EBMS_ONLINE_REG_ANSWER,
		},
		{
			name:        "Abschluss",
			message:     `{"conversationId":"RC100298202308171692252620000000321","messageId":"AT003000202308180842215740187694787","sender":"AT003000","receiver":"RC100298","messageCode":"ABSCHLUSS_ECON","meterList":[{"meteringPoint":"AT0030000000000000000000000519928","direction":"CONSUMPTION"}]}`,
			codes:       []string{},
			messageType: model.EBMS_ONLINE_REG_COMPLETION,
		},
	}
	for _, m := range tests {
		t.Run(m.name, func(t *testing.T) {
			var mockDb, err = database.GetDatabaseMock()
			require.NoError(t, err)
			recorder := &RecorderMock{dbOpen: mockDb.OpenMockDb}

			//jsonString := `{"conversationId":"RC100298202308171692252620000000321","messageId":"AT003000202308170810324070187796715","sender":"AT003000","receiver":"RC100298","messageCode":"ZUSTIMMUNG_ECON","requestId":"XV3VFJN2","responseData":[{"meteringPoint":"AT0030000000000000000000000459143","responseCode":[175]}]}`
			msg := model.SubscribeMessage{
				MessageCode: m.messageType,
				Protocol:    model.EC_REQ_ONL,
				Tenant:      "TE1000001",
				Payload:     model.EbmsMessage{},
			}
			err = json.Unmarshal([]byte(m.message), &msg.Payload)
			require.NoError(t, err)

			mockDb.Mock.ExpectExec("UPDATE (.+)").WillReturnResult(sqlmock.NewResult(1, 1))

			recorder.Mock.On("saveNotification", map[string]interface{}{
				"type":           msg.MessageCode,
				"meteringPoints": msg.Payload.Meters(),
				"responseCodes":  m.codes,
			}, msg.Tenant, "NOTIFICATION", "ADMIN")

			recorder.Mock.On("saveHistory", "TE1000001", msg.MessageCode, "RC100298202308171692252620000000321", "ADMIN", "IN", model.EC_REQ_ONL, msg.Payload)

			protocolEcReqOnlHandler(msg, recorder)
			recorder.AssertExpectations(t)

		})
	}
}
