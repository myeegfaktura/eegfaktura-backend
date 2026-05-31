package eda

import (
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/model"
	mqttclient "github.com/eegfaktura/eegfaktura-backend/mqtt"
	"github.com/eegfaktura/eegfaktura-backend/parser"
	"github.com/eegfaktura/eegfaktura-backend/util"
	"github.com/sirupsen/logrus"
)

var (
	ECON_RESPONSE_CODES = map[int16]string{
		99:  "Meldung erhalten",
		182: "Noch kein fernauslesbarer Zähler eingebaut",
		183: "Summe der gemeldeten Aufteilungsschlüssel übersteigt 100%",

		175: "Zustimmung erteilt",

		56:  "Zählpunkt nicht gefunden",
		184: "Kunde hat optiert",
		177: "Keine Datenfreigabe vorhanden",
		160: "Verteilmodell entspricht nicht der Vereinbarung",
		159: "Zu Prozessdatum ZP inaktiv bzw. noch kein Gerät eingebaut",
		158: "ZP ist nicht teilnahmeberechtigt",
		157: "ZP bereits einem Betreiber zugeordnet",
		156: "ZP bereits zugeordnet",
		86:  "konkurrierende Prozesse",
		181: "Gemeinschafts-ID nicht vorhanden",
		178: "Consent existiert bereits",
		174: "Angefragte Daten nicht lieferbar",
		173: "Kunde hat auf Datenfreigabe nicht reagiert (Timeout)",
		172: "Kunde hat Datenfreigabe abgelehnt",
		76:  "Ungültige Anforderungsdaten",
		57:  "Zählpunkt nicht versorgt",
		185: "Zählpunkt befindet sich nicht im Bereich der Energiegemeinschaft",
		37:  "Stornierung nicht möglich",

		55: "Zählpunkt nicht dem Lieferanten zugeordnet",
		70: "Änderung/Anforderung akzeptiert",
		82: "Prozessdatum falsch",
		90: "Kein Smart Meter",
		94: "Keine Daten im angeforderten Zeitraum vorhanden",
	}
	REJECTED_INVALID_CODES = []int16{56, 184, 177, 159, 158, 156, 86}
)

func InitEdaSubscription() {
	mqttclient.Subscribe(getSubsriptions()...)
}

func getSubsriptions() []model.Subscriptions {
	recorder := NewEdaRecorder()
	return []model.Subscriptions{
		{
			Protocol: model.CR_MSG,
			Handler: func(msg model.SubscribeMessage) {
				protocolCrMsgHandler(msg, recorder)
			},
		},
		{
			Protocol: model.CR_REQ_PT,
			Handler: func(msg model.SubscribeMessage) {
				protocolCrReqPtHandler(msg, recorder)
			},
		},
		{
			Protocol: model.EC_REQ_ONL,
			Handler: func(msg model.SubscribeMessage) {
				protocolEcReqOnlHandler(msg, recorder)
			},
		},
		{
			Protocol: model.CM_REV_IMP,
			Handler: func(msg model.SubscribeMessage) {
				protocolCmRevImpHandler(msg, recorder)
			},
		},
		{
			Protocol: model.CM_REV_CUS,
			Handler: func(msg model.SubscribeMessage) {
				protocolCmRevImpHandler(msg, recorder)
			},
		},
		{
			Protocol: model.CM_REV_SP,
			Handler: func(msg model.SubscribeMessage) {
				protocolCmRevImpHandler(msg, recorder)
			},
		},
		{
			Protocol: model.EC_PRTFACT_CHANGE,
			Handler: func(msg model.SubscribeMessage) {
				protocolEcPrtChangeHandler(msg, recorder)
			},
		},
	}
}

// protocolEcPrtChangeHandler processes the grid-operator response to a
// partition-factor change request (CR_PARTITIONFACTORCHANGE, MQTT
// protocol EC_PRTFACT_CHANGE). Three message variants:
//   - EBMS_ANS_CHANGE_PARTFACT (ANTWORT_CPF) — change accepted, the new
//     partition factors are appended to base.metering_partition_factor
//     via MeteringPointChangePartFactor.
//   - EBMS_REJ_CHANGE_PARTFACT (ABLEHNUNG_CPF) — change rejected, a
//     notification with the error code is saved for the EEG admin.
//   - EBMS_REQ_CHANGE_PARTFACT (ANFORDERUNG_CPF) — outbound echo,
//     recorded for history but no DB side-effect.
//
// Tenant lookup goes via EcId (communityId) because EDA inbound
// payloads identify the EEG by community id, not tenant.
func protocolEcPrtChangeHandler(msg model.SubscribeMessage, recorder EdaRecording) {
	logrus.WithField("tenant", msg.Tenant).Printf("Handle Subscriptions: %+v Code: %s", msg.Protocol, msg.MessageCode)

	var meters []model.Meter
	var errCode int16
	switch msg.MessageCode {
	case model.EBMS_REJ_CHANGE_PARTFACT:
		if len(msg.Payload.ResponseData) > 0 && len(msg.Payload.ResponseData[0].ResponseCode) > 0 {
			errCode = msg.Payload.ResponseData[0].ResponseCode[0]
		} else {
			errCode = 1000
		}
	case model.EBMS_ANS_CHANGE_PARTFACT:
		meters = msg.Payload.MeterList
		errCode = 0
	case model.EBMS_REQ_CHANGE_PARTFACT:
		meters = nil
	default:
		logrus.WithField("tenant", msg.Tenant).Warnf("Unknown Messagecode: %v", msg)
		return
	}

	db, err := recorder.databaseConnect()
	if err != nil {
		logrus.WithField("tenant", msg.Tenant).Errorf("can not open db: %v", err)
		return
	}
	defer db.Close()

	eeg, err := database.GetEegByEcIdDB(db, msg.Payload.EcId)
	if err != nil {
		logrus.WithField("tenant", msg.Tenant).Errorf("can not fetch eeg with EcId %q: %v", msg.Payload.EcId, err)
		return
	}

	if len(meters) > 0 && errCode == 0 {
		if err := database.MeteringPointChangePartFactorDB(db, eeg.Id, meters); err != nil {
			logrus.WithField("tenant", eeg.Id).Errorf("can not change partition factor: %v", err)
			return
		}
	}

	if errCode > 0 {
		meterIds := make([]string, 0, len(meters))
		for _, m := range meters {
			meterIds = append(meterIds, m.MeteringPoint)
		}
		_ = recorder.saveNotification(map[string]interface{}{
			"type":           msg.MessageCode,
			"meteringPoints": meterIds,
			"responseCodes":  convertCodes2Strings([]int16{errCode}),
		}, eeg.Id, "NOTIFICATION", "ADMIN")
	}
	_ = recorder.saveHistory(eeg.Id, msg.MessageCode, msg.Payload.ConversationId, "ADMIN", "IN", msg.Protocol, msg.Payload)
}

func protocolCrMsgHandler(msg model.SubscribeMessage, recorder EdaRecording) {
	logrus.Printf("Handle Subscriptions: %+v", msg.Protocol)

	if msg.Payload.Meter != nil && len(msg.Payload.Energy) > 0 {
		from, to := msg.Payload.Energy[0].Start, msg.Payload.Energy[0].End
		for _, e := range msg.Payload.Energy[1:] {
			if e.Start < from {
				from = e.Start
			}
			if e.End > to {
				to = e.End
			}
		}
		historyValue := map[string]interface{}{"meter": msg.Payload.Meter.MeteringPoint, "from": from, "to": to}
		_ = recorder.saveHistory(msg.Tenant, msg.MessageCode, msg.Payload.ConversationId, "ADMIN", "IN", msg.Protocol, historyValue)
	}
	return
}

func protocolCrReqPtHandler(msg model.SubscribeMessage, recorder EdaRecording) {
	var err error
	logrus.Printf("Handle Subscriptions: %+v", msg.Protocol)

	codes := []int16{}

	switch msg.MessageCode {
	case model.EBMS_ZP_RES, model.EBMS_ZP_REJ, model.EBMS_ZP_SYNC:
		codes, _, _ = extractResponseCodeAndMeteringPoint(&msg.Payload)
	default:
		return
	}

	if err = recorder.saveNotification(map[string]interface{}{
		"type":           msg.MessageCode,
		"meteringPoints": msg.Payload.Meters(),
		"responseCodes":  convertCodes2Strings(codes),
	}, msg.Tenant, "NOTIFICATION", "ADMIN"); err != nil {
		logrus.WithField("PROTOCOL", msg.Protocol).Error(err)
	}
	_ = recorder.saveHistory(msg.Tenant, msg.MessageCode, msg.Payload.ConversationId, "ADMIN", "IN", msg.Protocol, msg.Payload)
}

func protocolEcReqOnlHandler(msg model.SubscribeMessage, recorder EdaRecording) {
	var err error
	logrus.Printf("Handle Subscriptions: %+v", msg.Protocol)

	codes, meters, _ := extractResponseCodeAndMeteringPoint(&msg.Payload)
	var status model.StatusType

	switch msg.MessageCode {
	case model.EBMS_ONLINE_REG_COMPLETION:
		codes = []int16{}
		meters = extractMeterList(&msg.Payload)
		status = model.ACTIVE
	case model.EBMS_ONLINE_REG_REJECTION:
		if codesContains(REJECTED_INVALID_CODES, codes) {
			status = model.INVALID
		} else {
			status = model.REJECTED
		}
	case model.EBMS_ONLINE_REG_APPROVAL:
		for _, c := range codes {
			if c == 175 {
				status = model.APPROVED
			}
		}
	case model.EBMS_ONLINE_REG_ANSWER:
		for _, c := range codes {
			if c == 99 {
				status = model.PENDING
			}
		}
	case model.EBMS_ONLINE_REG_INIT:
		codes = []int16{}
	default:
		return
	}

	if len(meters) > 0 && len(status) > 0 {
		rowsAffected, err := database.MeteringPointsSetStatus(recorder.databaseConnect, msg.Tenant, status, meters)
		if err != nil {
			logrus.WithField("error", err.Error()).Errorf("can not change metering point status %+v", meters)
			return
		}
		if status == model.ACTIVE && rowsAffected > 0 {
			sendMeteringPointActiveMails(msg.Tenant, meters, recorder)
		}
	}

	if err = recorder.saveNotification(map[string]interface{}{
		"type":           msg.MessageCode,
		"meteringPoints": msg.Payload.Meters(),
		"responseCodes":  convertCodes2Strings(codes),
	}, msg.Tenant, "NOTIFICATION", "ADMIN"); err != nil {
		logrus.WithField("PROTOCOL", msg.Protocol).Error(err)
	}
	_ = recorder.saveHistory(msg.Tenant, msg.MessageCode, msg.Payload.ConversationId, "ADMIN", "IN", msg.Protocol, msg.Payload)
}

func sendMeteringPointActiveMails(tenant string, meteringPointIds []string, recorder EdaRecording) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("activation mail: recovered from panic: %v", r)
		}
	}()

	eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
	if err != nil {
		logrus.WithField("error", err.Error()).Errorf("activation mail: cannot load EEG for tenant %s", tenant)
		return
	}

	for _, mpId := range meteringPointIds {
		participant, err := database.GetParticipantByMeteringPoint(recorder.databaseConnect, tenant, mpId)
		if err != nil {
			logrus.WithField("error", err.Error()).Errorf("activation mail: cannot find participant for metering point %s", mpId)
			continue
		}
		if participant == nil {
			logrus.Warnf("activation mail: no participant found for metering point %s", mpId)
			continue
		}

		if err = parser.SendMeteringPointActiveMailFromTemplate(util.SendMail, tenant, "Ihr Zählpunkt ist aktiv", mpId, eeg, participant); err != nil {
			logrus.WithField("error", err.Error()).Errorf("activation mail: send failed for participant %s, metering point %s", participant.Id, mpId)
		}
	}
}

func protocolCmRevImpHandler(msg model.SubscribeMessage, recorder EdaRecording) {
	var err error
	logrus.Printf("Handle Subscriptions: %+v", msg.Protocol)

	codes, meters, _ := extractResponseCodeAndMeteringPoint(&msg.Payload)
	var status model.StatusType

	switch msg.MessageCode {
	case model.EBMS_AUFHEBUNG_CCMI, model.EBMS_AUFHEBUNG_CCMS, model.EBMS_AUFHEBUNG_CCMC:
		status = model.REVOKED
	default:
		return
	}

	if len(meters) > 0 && len(status) > 0 {
		if _, err := database.MeteringPointsSetStatus(recorder.databaseConnect, msg.Tenant, status, meters); err != nil {
			logrus.WithField("error", err.Error()).Errorf("can not change metering point status %+v", meters)
			return
		}
	}

	if err = recorder.saveNotification(map[string]interface{}{
		"type":           msg.MessageCode,
		"meteringPoints": msg.Payload.Meters(),
		"responseCodes":  convertCodes2Strings(codes),
	}, msg.Tenant, "NOTIFICATION", "ADMIN"); err != nil {
		logrus.WithField("PROTOCOL", msg.Protocol).Error(err)
	}
	_ = recorder.saveHistory(msg.Tenant, msg.MessageCode, msg.Payload.ConversationId, "ADMIN", "IN", msg.Protocol, msg.Payload)
}
