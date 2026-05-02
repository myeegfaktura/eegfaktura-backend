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
	}
}

func protocolCrMsgHandler(msg model.SubscribeMessage, recorder EdaRecording) {
	logrus.Printf("Handle Subscriptions: %+v", msg.Protocol)

	if msg.Payload.Meter != nil && msg.Payload.Energy != nil {
		historyValue := map[string]interface{}{"meter": msg.Payload.Meter.MeteringPoint, "from": msg.Payload.Energy.Start, "to": msg.Payload.Energy.End}
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
		if err := database.MeteringPointsSetStatus(recorder.databaseConnect, msg.Tenant, status, meters); err != nil {
			logrus.WithField("error", err.Error()).Errorf("can not change metering point status %+v", meters)
			return
		}
		if status == model.ACTIVE {
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
	eeg, err := database.GetEeg(tenant)
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
		if err := database.MeteringPointsSetStatus(recorder.databaseConnect, msg.Tenant, status, meters); err != nil {
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
