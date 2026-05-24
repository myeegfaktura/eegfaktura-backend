package factory

import (
	"testing"

	protobuf "github.com/eegfaktura/eegfaktura-backend/proto"
)

func strPtr(s string) *string { return &s }

func TestGetEegFromRegisterEeg_AllFields(t *testing.T) {
	phone := "+43 660 1234567"
	web := "https://example.test"
	req := &protobuf.RegisterEegRequest{
		RcNumber:           "TE100200",
		Name:               "T-VIERE",
		Description:        "Test EEG",
		Area:               protobuf.RegisterEegRequest_LOCAL,
		Legal:              protobuf.RegisterEegRequest_VEREIN,
		GridName:           "Netz OÖ",
		CommunityId:        "AT00999900000TC100200000000000002",
		GridId:             "AT003000",
		Allocation:         protobuf.RegisterEegRequest_DYNAMIC,
		SettelmentInterval: protobuf.RegisterEegRequest_MONTHLY,
		TaxNumber:          "11 123/4567",
		VatNumber:          "ATU12345678",
		Street:             "Solarstraße",
		Iban:               "AT011234000000321321",
		Owner:              "T-VIERE",
		Sepa:               true,
		Email:              "test@example.test",
		Phone:              &phone,
		Web:                &web,
		Online:             true,
	}

	got := GetEegFromRegisterEeg(req)

	if got.Id != "TE100200" {
		t.Errorf("Id = %q, want TE100200", got.Id)
	}
	if got.RcNumber != "TE100200" {
		t.Errorf("RcNumber = %q, want TE100200 (tenant ≡ rcNumber invariant)", got.RcNumber)
	}
	if got.Name != "T-VIERE" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.CommunityId != "AT00999900000TC100200000000000002" {
		t.Errorf("CommunityId = %q", got.CommunityId)
	}
	if got.GridOperator != "AT003000" {
		t.Errorf("GridOperator = %q, want AT003000 (mapped from GridId)", got.GridOperator)
	}
	if got.OperatorName != "Netz OÖ" {
		t.Errorf("OperatorName = %q, want 'Netz OÖ' (mapped from GridName)", got.OperatorName)
	}
	if !got.Contact.Phone.Valid || got.Contact.Phone.String != phone {
		t.Errorf("Phone not propagated: %+v", got.Contact.Phone)
	}
	if !got.Optionals.Website.Valid || got.Optionals.Website.String != web {
		t.Errorf("Website not propagated: %+v", got.Optionals.Website)
	}
	if !got.Online {
		t.Errorf("Online = false, want true")
	}

	// Quirk pinned: Street value is reused for StreetNumber/Zip/City —
	// preserves the inline-form behaviour. Documented in eegFactory.go.
	if got.EegAddress.Street != "Solarstraße" {
		t.Errorf("Street = %q", got.EegAddress.Street)
	}
	if got.EegAddress.StreetNumber != "Solarstraße" {
		t.Errorf("StreetNumber = %q, expected Street value due to inline-form quirk", got.EegAddress.StreetNumber)
	}
}

func TestGetEegFromRegisterEeg_NilOptionalFields(t *testing.T) {
	req := &protobuf.RegisterEegRequest{
		RcNumber: "TE100200",
		Name:     "T-VIERE",
		Email:    "test@example.test",
		// Phone, Web intentionally left nil
	}

	got := GetEegFromRegisterEeg(req)

	if got.Contact.Phone.Valid {
		t.Errorf("Phone should be NULL when nil pointer; got %+v", got.Contact.Phone)
	}
	if got.Optionals.Website.Valid {
		t.Errorf("Website should be NULL when nil pointer; got %+v", got.Optionals.Website)
	}
}
