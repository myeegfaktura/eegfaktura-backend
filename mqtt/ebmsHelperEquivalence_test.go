// Equivalence tests for the EBMS-helper refactor (class B + class C).
//
// For each call-site that previously built `model.EbmsMessage{...}`
// inline and is now (or will be) routed through a helper, we capture
// both the inline-form and the helper-form for the same input and
// assert that they produce the same wire-relevant fields. Conversation/
// RequestId differ by design (uuid.New per call), so they are ignored.
//
// The "wire-relevant" fields are those that eda-comm reads when serialising
// the JSON EbmsMessage to EBMS XML. For ZP_SYNC + ZP_LIST (CPRequest_01p12),
// eda-comm's XML mapper (CPRequestMeteringValueXMLMessage, CPRequestZPListXMLMessage)
// only consumes:
//   - Sender, Receiver        (RoutingHeader)
//   - MessageCode             (MarketParticipantDirectory.MessageCode)
//   - Meter.MeteringPoint     (ProcessDirectory.MeteringPoint — string field)
//   - Timeline.From, Timeline.To  (Extension.DateTimeFrom/To)
// Direction is read by eda-comm only on the inbound ECMPList response,
// never on outbound CPRequest — so propagating it or not on the way out
// is wire-invisible.
package mqttclient

import (
	"strings"
	"testing"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/model"
)

// stripVolatile clears the per-call random fields so two messages built
// for the same logical event compare equal on their wire-relevant content.
func stripVolatile(m model.EbmsMessage) model.EbmsMessage {
	m.ConversationId = ""
	m.RequestId = ""
	return m
}

// ----- class B: meteringPointController.go:500 (registerMeterPoint loop)

// Inline-form recreates the exact `model.EbmsMessage{...}` literal that
// was at meteringPointController.go:500 before the refactor.
func inlineZPSyncFromMeterRequest(eeg *model.Eeg, meterPoint string, direction model.DirectionType, fromDate, toDate int64) model.EbmsMessage {
	return model.EbmsMessage{
		Sender:      strings.ToUpper(eeg.Id),
		Receiver:    strings.ToUpper(eeg.GridOperator),
		MessageCode: model.EBMS_ZP_SYNC,
		Meter:       &model.Meter{MeteringPoint: meterPoint, Direction: direction},
		Timeline:    &model.Timeline{From: fromDate, To: toDate},
	}
}

func TestZPSyncFromMeterRequest_HelperMatchesInline(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	eeg := sampleEeg()
	from := int64(1700000000000)
	to := int64(1700086400000)
	meter := &model.MeteringPoint{
		MeteringPoint: "AT001000000000000000000000123456",
		Direction:     model.CONSUMPTION,
	}

	if err := RequestingEnergyData(eeg, meter, from, to); err != nil {
		t.Fatalf("RequestingEnergyData: %v", err)
	}
	got := stripVolatile(*cap.msg)
	want := stripVolatile(inlineZPSyncFromMeterRequest(
		eeg, meter.MeteringPoint, meter.Direction, from, to))

	// The Sender field: inline used eeg.Id (= tenant header value),
	// helper uses eeg.RcNumber. Per DB-check in PR #23 these are
	// equal in both stacks (TE100200 has tenant == rcNumber). For
	// the equivalence test we assert the precondition explicitly.
	if eeg.Id != eeg.RcNumber {
		t.Fatalf("test precondition broken: eeg.Id (%q) != eeg.RcNumber (%q)", eeg.Id, eeg.RcNumber)
	}

	if got.Sender != want.Sender {
		t.Errorf("Sender mismatch: got %q, want %q", got.Sender, want.Sender)
	}
	if got.Receiver != want.Receiver {
		t.Errorf("Receiver mismatch: got %q, want %q", got.Receiver, want.Receiver)
	}
	if got.MessageCode != want.MessageCode {
		t.Errorf("MessageCode mismatch: got %q, want %q", got.MessageCode, want.MessageCode)
	}
	if got.Meter.MeteringPoint != want.Meter.MeteringPoint {
		t.Errorf("Meter.MeteringPoint mismatch: got %q, want %q",
			got.Meter.MeteringPoint, want.Meter.MeteringPoint)
	}
	if got.Timeline.From != want.Timeline.From || got.Timeline.To != want.Timeline.To {
		t.Errorf("Timeline mismatch: got %+v, want %+v", got.Timeline, want.Timeline)
	}
	// Direction: helper propagates it, inline kept it too at this call-site.
	// eda-comm ignores it, but the JSON field is observable to a tester
	// capturing the MQTT message — assert the helper preserves it.
	if got.Meter.Direction != want.Meter.Direction {
		t.Errorf("Meter.Direction mismatch: got %q, want %q",
			got.Meter.Direction, want.Meter.Direction)
	}
}

// ----- class B: eegController.go:265 (syncMeterpointEda)

// Inline-form for eegController.go:265 — the body-parsed MeteringPoint
// did NOT carry Direction in the inline form (only MeteringPoint).
func inlineZPSyncFromBody(eeg *model.Eeg, meterPoint string, day time.Time) model.EbmsMessage {
	return model.EbmsMessage{
		Sender:      strings.ToUpper(eeg.Id),
		Receiver:    strings.ToUpper(eeg.GridOperator),
		MessageCode: model.EBMS_ZP_SYNC,
		Meter:       &model.Meter{MeteringPoint: meterPoint},
		Timeline: &model.Timeline{
			From: time.Date(day.Year(), day.Month(), day.Day()-3, 0, 0, 0, 0, day.Location()).UnixMilli(),
			To:   time.Date(day.Year(), day.Month(), day.Day()-2, 0, 0, 0, 0, day.Location()).UnixMilli(),
		},
	}
}

func TestZPSyncFromBody_HelperMatchesInline_WireFields(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	eeg := sampleEeg()
	day := time.Date(2026, 5, 30, 14, 0, 0, 0, time.Local)
	meter := &model.MeteringPoint{
		MeteringPoint: "AT001000000000000000000000654321",
		// inline form did NOT set Direction here.
		Direction: model.DirectionType(""),
	}
	from := time.Date(day.Year(), day.Month(), day.Day()-3, 0, 0, 0, 0, day.Location()).UnixMilli()
	to := time.Date(day.Year(), day.Month(), day.Day()-2, 0, 0, 0, 0, day.Location()).UnixMilli()

	if err := RequestingEnergyData(eeg, meter, from, to); err != nil {
		t.Fatalf("RequestingEnergyData: %v", err)
	}
	got := stripVolatile(*cap.msg)
	want := stripVolatile(inlineZPSyncFromBody(eeg, meter.MeteringPoint, day))

	if got.Sender != want.Sender {
		t.Errorf("Sender mismatch: got %q, want %q", got.Sender, want.Sender)
	}
	if got.Receiver != want.Receiver {
		t.Errorf("Receiver mismatch: got %q, want %q", got.Receiver, want.Receiver)
	}
	if got.MessageCode != want.MessageCode {
		t.Errorf("MessageCode mismatch: got %q, want %q", got.MessageCode, want.MessageCode)
	}
	if got.Meter.MeteringPoint != want.Meter.MeteringPoint {
		t.Errorf("Meter.MeteringPoint mismatch: got %q, want %q",
			got.Meter.MeteringPoint, want.Meter.MeteringPoint)
	}
	if got.Timeline.From != want.Timeline.From || got.Timeline.To != want.Timeline.To {
		t.Errorf("Timeline mismatch: got %+v, want %+v", got.Timeline, want.Timeline)
	}
	// Meter.Direction: helper propagates `meter.Direction` which is "" here,
	// inline form did not set it (zero value also "") — they match.
}

// ----- class C: eegController.go:228 (syncParticipantsEda)

// Inline-form for eegController.go:228 — the trick is that MeteringPoint
// is set to eeg.CommunityId rather than an actual metering point.
func inlineZPListFromCommunity(eeg *model.Eeg, day time.Time) model.EbmsMessage {
	return model.EbmsMessage{
		Sender:      strings.ToUpper(eeg.Id),
		Receiver:    strings.ToUpper(eeg.GridOperator),
		MessageCode: model.EBMS_ZP_LIST,
		Meter:       &model.Meter{MeteringPoint: eeg.CommunityId},
		Timeline: &model.Timeline{
			From: time.Date(day.Year(), day.Month(), day.Day()-1, 0, 0, 0, 0, day.Location()).UnixMilli(),
			To:   time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli(),
		},
	}
}

func TestZPListForCommunity_HelperMatchesInline(t *testing.T) {
	cap, cleanup := captureDispatch(t)
	defer cleanup()

	eeg := sampleEeg()
	day := time.Date(2026, 5, 30, 14, 0, 0, 0, time.Local)
	from := time.Date(day.Year(), day.Month(), day.Day()-1, 0, 0, 0, 0, day.Location()).UnixMilli()
	to := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli()

	if err := RequestingMeteringPointListForCommunity(eeg, from, to); err != nil {
		t.Fatalf("RequestingMeteringPointListForCommunity: %v", err)
	}
	got := stripVolatile(*cap.msg)
	want := stripVolatile(inlineZPListFromCommunity(eeg, day))

	if got.Sender != want.Sender {
		t.Errorf("Sender mismatch: got %q, want %q", got.Sender, want.Sender)
	}
	if got.Receiver != want.Receiver {
		t.Errorf("Receiver mismatch: got %q, want %q", got.Receiver, want.Receiver)
	}
	if got.MessageCode != want.MessageCode {
		t.Errorf("MessageCode mismatch: got %q, want %q", got.MessageCode, want.MessageCode)
	}
	// The crucial check: MeteringPoint must contain CommunityId, not "".
	if got.Meter == nil {
		t.Fatal("Meter is nil — helper would have dropped CommunityId on wire")
	}
	if got.Meter.MeteringPoint != want.Meter.MeteringPoint {
		t.Errorf("Meter.MeteringPoint mismatch: got %q, want %q (= CommunityId)",
			got.Meter.MeteringPoint, want.Meter.MeteringPoint)
	}
	if got.Timeline.From != want.Timeline.From || got.Timeline.To != want.Timeline.To {
		t.Errorf("Timeline mismatch: got %+v, want %+v", got.Timeline, want.Timeline)
	}
}
