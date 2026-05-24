package model

import (
	"github.com/pborman/uuid"
	"gopkg.in/guregu/null.v4"
)

type BillingPeriod string

const (
	ANNUAL     BillingPeriod = "annual"
	MONTHLY    BillingPeriod = "monthly"
	SEMIANNUAL BillingPeriod = "semiannual"
	QUARTERLY  BillingPeriod = "quarterly"
)

type TariffModelType string

const (
	EEG    TariffModelType = "EEG"
	VZP    TariffModelType = "VZP"
	EZP    TariffModelType = "EZP"
	AKONTO TariffModelType = "AKONTO"
)

type Tariff struct {
	Id                  uuid.UUID       `json:"id" goqu:"defaultifempty"`
	Version             int             `json:"version" db:"version"`
	Type                TariffModelType `json:"type"`
	Name                string          `json:"name"`
	BillingPeriod       string          `json:"billingPeriod" db:"billingPeriod"`
	UseVat              bool            `json:"useVat" db:"useVat"`
	VatInPercent        int             `json:"vatInPercent" db:"vatInPercent"`
	AccountNetAmount    int             `json:"accountNetAmount" db:"accountNetAmount"`
	AccountGrossAmount  int             `json:"accountGrossAmount"  db:"accountGrossAmount"`
	ParticipantFee      int             `json:"participantFee" db:"participantFee"`
	BaseFee             int             `json:"baseFee" db:"baseFee"`
	BusinessNr          null.Int        `json:"businessNr" db:"businessNr"`
	CentPerKWh          int             `json:"centPerKWh" db:"centPerKWh"`
	// FreeKWh and Discount exist in base.tariff but prod-vfeeg-backend
	// does not emit them in the GET response. omitempty drops them when
	// zero — typical state for newly-imported tariffs.
	FreeKWh             int             `json:"freeKWh,omitempty" db:"freeKWh"`
	Discount            int             `json:"discount,omitempty"`
	// Fields added to match prod-image v0.3.05 response shape (#45).
	UseMeteringPointFee bool            `json:"useMeteringPointFee" db:"useMeteringPointFee"`
	MeteringPointFee    null.Float      `json:"meteringPointFee" db:"meteringPointFee"`
	MeteringPointVat    null.Int        `json:"meteringPointVat" db:"meteringPointVat"`
	InactiveSince       null.Time       `json:"inactiveSince" db:"inactiveSince"`
}


//func (t Tariff) PrepareType() Tariff {
//	switch t.Type {
//	case "EEG":
//		t.AccountNetAmount = 0
//		t.AccountGrossAmount = 0
//		t.CentPerKWh = 0
//		t.FreeKWH = 0
//		break
//	case "VZP":
//		t.AccountNetAmount = 0
//		t.AccountGrossAmount = 0
//		t.ParticipantFee = 0
//		break
//	case "EZP":
//		t.AccountNetAmount = 0
//		t.AccountGrossAmount = 0
//		t.ParticipantFee = 0
//	}
//	return t
//}
