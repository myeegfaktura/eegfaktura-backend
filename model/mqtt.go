package model

type EbMsMessageType string

const (
	EBMS_ENERGY_FILE_RESPONSE  EbMsMessageType = "DATEN_CRMSG"
	EBMS_ONLINE_REG_INIT       EbMsMessageType = "ANFORDERUNG_ECON"
	EBMS_OFFLINE_REG_INIT      EbMsMessageType = "ANFORDERUNG_ECOF"
	EBMS_REQ_CHANGE_PARTFACT   EbMsMessageType = "ANFORDERUNG_CPF"
	EBMS_ONLINE_REG_ANSWER     EbMsMessageType = "ANTWORT_ECON"
	EBMS_ONLINE_REG_REJECTION  EbMsMessageType = "ABLEHNUNG_ECON"
	EBMS_ONLINE_REG_APPROVAL   EbMsMessageType = "ZUSTIMMUNG_ECON"
	EBMS_ONLINE_REG_COMPLETION EbMsMessageType = "ABSCHLUSS_ECON"
	EBMS_ZP_LIST               EbMsMessageType = "ANFORDERUNG_ECP"
	EBMS_ZP_SYNC               EbMsMessageType = "ANFORDERUNG_PT"
	EBMS_ZP_RES                EbMsMessageType = "ANTWORT_PT"
	EBMS_ZP_REJ                EbMsMessageType = "ABLEHNUNG_PT"
	EBMS_ZP_LIST_RESPONSE      EbMsMessageType = "SENDEN_ECP"
	EBMS_AUFHEBUNG_CCMI        EbMsMessageType = "AUFHEBUNG_CCMI"
	EBMS_AUFHEBUNG_CCMS        EbMsMessageType = "AUFHEBUNG_CCMS"
	EBMS_AUFHEBUNG_CCMC        EbMsMessageType = "AUFHEBUNG_CCMC"
	EBMS_ABLEHNUNG_CCMS        EbMsMessageType = "ABLEHNUNG_CCMS"
	EBMS_ANTWORT_CCMS          EbMsMessageType = "ANTWORT_CCMS"
	EBMS_EEG_BASE_DATA         EbMsMessageType = "ANFORDERUNG_GN"
	EBMS_ERROR_MESSAGE         EbMsMessageType = "ERROR_MESSAGE"
)

type EdaProtocol string

const (
	CR_MSG     EdaProtocol = "CR_MSG"
	CR_REQ_PT  EdaProtocol = "CR_REQ_PT"
	EC_PODLIST EdaProtocol = "EC_PODLIST"
	EC_REQ_ONL EdaProtocol = "EC_REQ_ONL"
	CM_REV_IMP EdaProtocol = "CM_REV_IMP"
	CM_REV_CUS EdaProtocol = "CM_REV_CUS"
	ERROR      EdaProtocol = "ERROR"
)

type Timeline struct {
	From int64 `json:"from"` // Date
	To   int64 `json:"to"`   // Date
}

type EnergyValue struct {
	From   int64   `json:"from"`
	To     int64   `json:"to,omitempty"`
	Method string  `json:"method,omitempty"`
	Value  float64 `json:"value"`
}

type EnergyData struct {
	MeterCode string        `json:"meterCode"`
	Value     []EnergyValue `json:"value"`
}

type Energy struct {
	Start     int64        `json:"start"`
	End       int64        `json:"end"`
	Interval  string       `json:"interval"`
	NInterval int64        `json:"NInterval"`
	Data      []EnergyData `json:"data"`
}

type Meter struct {
	MeteringPoint string        `json:"meteringPoint"`
	Direction     DirectionType `json:"direction,omitempty"`
	// Activation is the epoch-millis timestamp from which a new
	// partition factor takes effect. Only populated for EBMS_REQ_CHANGE_PARTFACT
	// payloads; omitted in other flows.
	Activation int64 `json:"activation,omitempty"`
	// PartFact is the integer partition factor (0–100) requested for
	// the meter. Only populated for EBMS_REQ_CHANGE_PARTFACT payloads.
	PartFact int `json:"partFact,omitempty"`
}

type ResponseData struct {
	MeteringPoint string  `json:"meteringPoint,omitempty"`
	ResponseCode  []int16 `json:"responseCode"`
}

type EbmsMessage struct {
	ConversationId string          `json:"conversationId"`
	MessageId      string          `json:"messageId,omitempty"`
	Sender         string          `json:"sender"`
	Receiver       string          `json:"receiver"`
	MessageCode    EbMsMessageType `json:"messageCode"`
	// MessageCodeVersion lets the receiver (eda-comm / xp-adapter) pick
	// a specific EBMS schema version for the outgoing XML. Populated
	// from `eda-process-versions.<MessageCode>` in viper; absent means
	// the receiver falls back to its hard-coded default version.
	MessageCodeVersion string          `json:"messageCodeVersion,omitempty"`
	RequestId          string          `json:"requestId,omitempty"`
	Meter              *Meter          `json:"meter,omitempty"`
	EcId               string          `json:"ecId,omitempty"` // Community ID
	ResponseData       []ResponseData  `json:"responseData,omitempty"`
	Energy             *Energy         `json:"energy,omitempty"`
	Timeline           *Timeline       `json:"timeline,omitempty"`
	MeterList          []Meter         `json:"meterList,omitempty"`
	ErrorMessage       string          `json:"errorMessage,omitempty"`
}

func (ebms EbmsMessage) Meters() []string {
	if ebms.Meter != nil {
		return []string{ebms.Meter.MeteringPoint}
	}
	return []string{}
}

//type EdaMessage struct {
//	Message EbmsMessage `json:"message"`
//}

// SubscribeMessage aggregates the result from subscribing.
type SubscribeMessage struct {
	// Reports the index of corresponding SubscribeTopic.
	MessageCode EbMsMessageType

	Protocol EdaProtocol

	// Determine the tenantId.
	Tenant string

	// Reports the payload content.
	Payload EbmsMessage
}

type SubscribeHandler func(msg SubscribeMessage)

type Subscriptions struct {
	Protocol EdaProtocol
	Handler  SubscribeHandler
}
