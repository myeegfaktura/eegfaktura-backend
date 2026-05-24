// Package factory builds domain entities from external request types.
// Mirrors the factory/-Package in prod-vfeeg-backend v0.3.05 (ADR-0006
// parity gap). Today only contains the EEG construction from gRPC
// RegisterEegRequest, extracted from util/admin.go for reusability.
package factory

import (
	"github.com/eegfaktura/eegfaktura-backend/model"
	protobuf "github.com/eegfaktura/eegfaktura-backend/proto"
	"gopkg.in/guregu/null.v4"
)

// GetEegFromRegisterEeg maps a protobuf RegisterEegRequest into the
// domain `model.Eeg`. The mapping preserves the in-place behaviour
// from the previous inline construction in util/admin.go:
//
//   - Optional fields (Phone, Web) become null.String via the
//     getOptionalField helper.
//   - Mandatory fields (Email, Iban, Owner, TaxNumber, VatNumber)
//     are wrapped with null.StringFrom even when callers pass an
//     empty string — matches prior behaviour.
//   - `eeg.Street` is intentionally reused for StreetNumber, Zip
//     and City fields. That looks like a copy-paste oversight from
//     the original inline form but is preserved here verbatim;
//     fixing it is out of scope for this parity backport.
//   - `Id` and `RcNumber` both receive `eeg.RcNumber` (tenant ≡
//     rcNumber convention).
func GetEegFromRegisterEeg(eeg *protobuf.RegisterEegRequest) model.Eeg {
	return model.Eeg{
		Id:                 eeg.RcNumber,
		Name:               eeg.Name,
		Description:        eeg.Description,
		BusinessNr:         null.Int{},
		Area:               model.AreaType(eeg.Area.String()),
		Legal:              eeg.Legal.String(),
		OperatorName:       eeg.GridName,
		CommunityId:        eeg.CommunityId,
		GridOperator:       eeg.GridId,
		RcNumber:           eeg.RcNumber,
		AllocationMode:     eeg.Allocation.String(),
		SettlementInterval: eeg.SettelmentInterval.String(),
		ProviderBusinessNr: null.Int{},
		TaxNumber:          null.StringFrom(eeg.TaxNumber),
		VatNumber:          null.StringFrom(eeg.VatNumber),
		EegAddress: model.EegAddress{
			Street:       eeg.Street,
			StreetNumber: eeg.Street,
			Zip:          eeg.Street,
			City:         eeg.Street,
		},
		AccountInfo: model.AccountInfo{
			Iban:  null.StringFrom(eeg.Iban),
			Owner: null.StringFrom(eeg.Owner),
			Sepa:  eeg.Sepa,
		},
		Contact: model.Contact{
			Phone: getOptionalField(eeg.Phone),
			Email: null.StringFrom(eeg.Email),
		},
		Optionals: model.Optionals{
			Website: getOptionalField(eeg.Web),
		},
		Periods: nil,
		Online:  eeg.Online,
	}
}

// getOptionalField wraps a nullable *string into a null.String. Empty
// (nil) pointer becomes a SQL NULL.
func getOptionalField(field *string) null.String {
	if field == nil {
		return null.String{}
	}
	return null.StringFrom(*field)
}
