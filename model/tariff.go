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
	Id                 uuid.UUID       `json:"id" goqu:"defaultifempty"`
	Version            int             `json:"version" db:"version"`
	Type               TariffModelType `json:"type"`
	Name               string          `json:"name"`
	BillingPeriod      string          `json:"billingPeriod" db:"billingPeriod"`
	UseVat             bool            `json:"useVat" db:"useVat"`
	VatInPercent       int             `json:"vatInPercent,string" db:"vatInPercent"`
	AccountNetAmount   int             `json:"accountNetAmount,string" db:"accountNetAmount"`
	AccountGrossAmount int             `json:"accountGrossAmount,string"  db:"accountGrossAmount"`
	ParticipantFee     int             `json:"participantFee,string" db:"participantFee"`
	BaseFee            int             `json:"baseFee,string" db:"baseFee"`
	BusinessNr         null.Int        `json:"businessNr" db:"businessNr"`
	CentPerKWh         int             `json:"centPerKWh,string" db:"centPerKWh"`
	FreeKWh            int             `json:"freeKWh,string" db:"freeKWh"`
	Discount           int             `json:"discount,string"`
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
