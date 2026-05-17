// EBMS outbound message helpers for the mqttclient package.
//
// These helpers build the EBMS message envelope for the most common
// outbound flows and dispatch it through SendEbmsMessage on the MQTT
// streamer. They exist as a thin convenience layer over directly
// constructing model.EbmsMessage in the controllers; using them keeps
// the receiver/sender derivation, conversation/request-id generation
// and message-code mapping in one place.
//
// The helpers are declared as package-level variables holding func
// values rather than plain functions on purpose: tests can substitute
// a no-op or capturing implementation without touching the MQTT
// transport. The default values delegate to the real implementations
// at the bottom of this file.
package mqttclient

import (
	"strings"

	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

// RegistrationForParticipation sends an online-registration request
// for the given metering point to the grid operator. If a non-nil
// `from` timestamp is supplied it is propagated as the Timeline.From
// value (registration start).
var RegistrationForParticipation = func(eeg *model.Eeg, meter *model.MeteringPoint, from *int64) error {
	return sendRegistration(eeg, meter, from, model.EBMS_ONLINE_REG_INIT)
}

// OfflineRegistrationForParticipation sends the offline-variant of the
// participation registration. Behaves like RegistrationForParticipation
// but uses the offline message code.
var OfflineRegistrationForParticipation = func(eeg *model.Eeg, meter *model.MeteringPoint, from *int64) error {
	return sendRegistration(eeg, meter, from, model.EBMS_OFFLINE_REG_INIT)
}

// RequestingEnergyData asks the grid operator to deliver energy data
// for the metering point in the given time window. fromDate/toDate are
// inclusive epoch-milliseconds.
var RequestingEnergyData = func(eeg *model.Eeg, meter *model.MeteringPoint, fromDate, toDate int64) error {
	msg := newEbmsMessage(eeg, meter, model.EBMS_ZP_SYNC)
	msg.Timeline = &model.Timeline{From: fromDate, To: toDate}
	return dispatch(msg)
}

// RevokeMeteringPoint revokes the participation of the metering point.
// The optional `reason` is propagated as ErrorMessage (per the EBMS
// spec; "reason" is encoded into the existing freeform field).
func RevokeMeteringPoint(eeg *model.Eeg, meter *model.MeteringPoint, consentEnd int64, reason *string) error {
	msg := newEbmsMessage(eeg, meter, model.EBMS_AUFHEBUNG_CCMS)
	msg.Timeline = &model.Timeline{To: consentEnd}
	if reason != nil {
		msg.ErrorMessage = *reason
	}
	return dispatch(msg)
}

// RequestingMeteringPointList asks the receiver (typically a grid
// operator) for the current list of metering points within the
// supplied time window. Meter is intentionally nil — the receiver is
// the addressable grid operator, not a specific meter.
func RequestingMeteringPointList(eeg *model.Eeg, receiver string, from, to int64) error {
	msg := newEbmsMessage(eeg, nil, model.EBMS_ZP_LIST)
	if receiver != "" {
		msg.Receiver = strings.ToUpper(receiver)
	}
	msg.Timeline = &model.Timeline{From: from, To: to}
	return dispatch(msg)
}

// sendRegistration is the common path for online and offline
// participation registration.
func sendRegistration(eeg *model.Eeg, meter *model.MeteringPoint, from *int64, code model.EbMsMessageType) error {
	msg := newEbmsMessage(eeg, meter, code)
	if from != nil {
		msg.Timeline = &model.Timeline{From: *from}
	}
	return dispatch(msg)
}

// newEbmsMessage prepares a fresh EbmsMessage with sender/receiver
// derived from the EEG plus the message code; conversation and
// request identifiers are generated. Meter may be nil for messages
// that target the receiver as a whole rather than a specific point.
func newEbmsMessage(eeg *model.Eeg, meter *model.MeteringPoint, code model.EbMsMessageType) model.EbmsMessage {
	msg := model.EbmsMessage{
		ConversationId: uuid.New(),
		RequestId:      uuid.New(),
		Sender:         strings.ToUpper(eeg.RcNumber),
		Receiver:       receiverFor(eeg, meter),
		MessageCode:    code,
		EcId:           eeg.CommunityId,
	}
	if meter != nil {
		msg.Meter = &model.Meter{
			MeteringPoint: meter.MeteringPoint,
			Direction:     meter.Direction,
		}
	}
	return msg
}

// receiverFor derives the EBMS receiver from the EEG (and, when
// relevant, the metering point). For the LOCAL/REGIONAL areas of the
// public model the receiver is always the configured grid operator of
// the EEG.
func receiverFor(eeg *model.Eeg, _ *model.MeteringPoint) string {
	return strings.ToUpper(eeg.GridOperator)
}

// dispatch is the single point at which all helpers hand off to the
// MQTT transport. Indirected through a package-level variable so tests
// can capture or discard the message.
var dispatch = func(msg model.EbmsMessage) error {
	if err := SendEbmsMessage(msg); err != nil {
		log.WithError(err).WithField("code", msg.MessageCode).Error("EBMS dispatch failed")
		return err
	}
	return nil
}

