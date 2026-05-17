package mqttclient

import (
	"errors"
	"testing"

	"github.com/eegfaktura/eegfaktura-backend/model"
)

// captureDispatch replaces the package-level dispatch var for the
// duration of a test. The returned cleanup restores the previous value
// and yields the captured message via the *EbmsMessage.
type captured struct {
	msg *model.EbmsMessage
}

func captureDispatch(t *testing.T) (*captured, func()) {
	t.Helper()
	orig := dispatch
	cap := &captured{}
	dispatch = func(m model.EbmsMessage) error {
		cap.msg = &m
		return nil
	}
	cleanup := func() { dispatch = orig }
	return cap, cleanup
}

func sampleEeg() *model.Eeg {
	return &model.Eeg{
		Id:           "TE100200",
		RcNumber:     "TE100200",
		CommunityId:  "AT00999900000TC100200000000000002",
		GridOperator: "NB-OP-001",
	}
}

func sampleMeter() *model.MeteringPoint {
	return &model.MeteringPoint{
		MeteringPoint: "AT001000000000000000000000123456",
		Direction:     model.CONSUMPTION,
	}
}

func TestRegistrationForParticipationDispatchesEbmsMessage(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	from := int64(1700000000000)
	if err := RegistrationForParticipation(sampleEeg(), sampleMeter(), &from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg == nil {
		t.Fatal("dispatch was not called")
	}
	if got := cap.msg.MessageCode; got != model.EBMS_ONLINE_REG_INIT {
		t.Errorf("MessageCode = %q, want %q", got, model.EBMS_ONLINE_REG_INIT)
	}
	if cap.msg.Timeline == nil || cap.msg.Timeline.From != from {
		t.Errorf("Timeline.From not propagated: %+v", cap.msg.Timeline)
	}
	if cap.msg.Meter == nil || cap.msg.Meter.MeteringPoint == "" {
		t.Errorf("Meter not set: %+v", cap.msg.Meter)
	}
	if cap.msg.Sender != "TE100200" {
		t.Errorf("Sender = %q, want uppercase RcNumber", cap.msg.Sender)
	}
	if cap.msg.Receiver != "NB-OP-001" {
		t.Errorf("Receiver = %q, want uppercase GridOperator", cap.msg.Receiver)
	}
	if cap.msg.ConversationId == "" || cap.msg.RequestId == "" {
		t.Errorf("ConversationId/RequestId must be generated; got %+v", cap.msg)
	}
}

func TestOfflineRegistrationUsesOfflineCode(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	if err := OfflineRegistrationForParticipation(sampleEeg(), sampleMeter(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg.MessageCode != model.EBMS_OFFLINE_REG_INIT {
		t.Errorf("MessageCode = %q, want %q", cap.msg.MessageCode, model.EBMS_OFFLINE_REG_INIT)
	}
	if cap.msg.Timeline != nil {
		t.Errorf("Timeline should be nil when from is nil; got %+v", cap.msg.Timeline)
	}
}

func TestRequestingEnergyDataCarriesTimeline(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	from := int64(1700000000000)
	to := int64(1700086400000)
	if err := RequestingEnergyData(sampleEeg(), sampleMeter(), from, to); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg.MessageCode != model.EBMS_ZP_SYNC {
		t.Errorf("MessageCode = %q, want %q", cap.msg.MessageCode, model.EBMS_ZP_SYNC)
	}
	if cap.msg.Timeline == nil || cap.msg.Timeline.From != from || cap.msg.Timeline.To != to {
		t.Errorf("Timeline: %+v", cap.msg.Timeline)
	}
}

func TestRevokeMeteringPointCarriesReason(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	reason := "Mitglied gekündigt"
	consentEnd := int64(1700086400000)
	if err := RevokeMeteringPoint(sampleEeg(), sampleMeter(), consentEnd, &reason); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg.MessageCode != model.EBMS_AUFHEBUNG_CCMS {
		t.Errorf("MessageCode = %q, want %q", cap.msg.MessageCode, model.EBMS_AUFHEBUNG_CCMS)
	}
	if cap.msg.ErrorMessage != reason {
		t.Errorf("ErrorMessage = %q, want %q (reason propagation)", cap.msg.ErrorMessage, reason)
	}
	if cap.msg.Timeline == nil || cap.msg.Timeline.To != consentEnd {
		t.Errorf("Timeline.To not set: %+v", cap.msg.Timeline)
	}
}

func TestRevokeMeteringPointNilReason(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	if err := RevokeMeteringPoint(sampleEeg(), sampleMeter(), 0, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg.ErrorMessage != "" {
		t.Errorf("ErrorMessage should be empty when reason is nil; got %q", cap.msg.ErrorMessage)
	}
}

func TestRequestingMeteringPointListOverridesReceiver(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	if err := RequestingMeteringPointList(sampleEeg(), "OTHER-RECEIVER", 100, 200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg.MessageCode != model.EBMS_ZP_LIST {
		t.Errorf("MessageCode = %q, want %q", cap.msg.MessageCode, model.EBMS_ZP_LIST)
	}
	if cap.msg.Receiver != "OTHER-RECEIVER" {
		t.Errorf("Receiver should be overridden to %q; got %q", "OTHER-RECEIVER", cap.msg.Receiver)
	}
	if cap.msg.Meter != nil {
		t.Errorf("Meter should be nil for list request; got %+v", cap.msg.Meter)
	}
	if cap.msg.Timeline == nil || cap.msg.Timeline.From != 100 || cap.msg.Timeline.To != 200 {
		t.Errorf("Timeline: %+v", cap.msg.Timeline)
	}
}

func TestRequestingMeteringPointListFallsBackToGridOperator(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	if err := RequestingMeteringPointList(sampleEeg(), "", 100, 200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.msg.Receiver != "NB-OP-001" {
		t.Errorf("Receiver fallback to GridOperator failed: got %q", cap.msg.Receiver)
	}
}

func TestDispatchErrorPropagates(t *testing.T) {
	orig := dispatch
	defer func() { dispatch = orig }()

	wantErr := errors.New("mqtt down")
	dispatch = func(model.EbmsMessage) error { return wantErr }

	err := RegistrationForParticipation(sampleEeg(), sampleMeter(), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestNewEbmsMessageSetsEcId(t *testing.T) {
	msg := newEbmsMessage(sampleEeg(), sampleMeter(), model.EBMS_ZP_LIST)
	if msg.EcId != "AT00999900000TC100200000000000002" {
		t.Errorf("EcId not propagated: %q", msg.EcId)
	}
}
