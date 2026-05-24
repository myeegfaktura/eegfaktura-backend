package model

import (
	"github.com/jmoiron/sqlx/types"
	"gopkg.in/guregu/null.v4"
	"time"
)

type Eeg struct {
	Id                 string      `json:"id" db:"tenant"`
	Name               string      `json:"name,omitempty"`
	Description        string      `json:"description,omitempty"`
	BusinessNr         null.String `json:"businessNr,omitempty" db:"businessNr"`
	Area               AreaType    `json:"area"` /* LOCAL | REGIONAL*/
	Legal              string      `json:"legal,omitempty"`
	OperatorName       string      `json:"operatorName,omitempty" db:"gridoperator_name"`
	CommunityId        string      `json:"communityId,omitempty" db:"communityId"`
	GridOperator       string      `json:"gridOperator,omitempty" db:"gridoperator_code"`
	RcNumber           string      `json:"rcNumber" db:"rcNumber"`
	AllocationMode     string      `json:"allocationMode,omitempty" db:"allocationMode"`
	SettlementInterval string      `json:"settlementInterval,omitempty" db:"settlementInterval"`
	ProviderBusinessNr null.Int    `json:"providerBusinessNr,omitempty" db:"providerBusinessNr"`
	TaxNumber          null.String `json:"taxNumber,omitempty" db:"taxNumber"`
	VatNumber          null.String `json:"vatNumber" db:"vatNumber"`
	ContactPerson      null.String `json:"contactPerson" db:"contactPerson"`
	EegAddress         `json:"address,omitempty"`
	AccountInfo        `json:"accountInfo,omitempty"`
	Contact            `json:"contact,omitempty"`
	Optionals          `json:"optionals,omitempty"`
	Periods            []int16 `json:"periods" goqu:"skipinsert,defaultifempty"`
	Online             bool    `json:"online"`
}

type AreaType string

const (
	LOCAL    AreaType = "LOCAL"
	REGIONAL AreaType = "REGIONAL"
)

type AddressType string

const (
	BILLING   AddressType = "BILLING"
	RESIDENCE AddressType = "RESIDENCE"
)

type Address struct {
	Type         AddressType `json:"type, omitempty" goqu:"skipupdate"`
	Street       string      `json:"street,omitempty"`
	StreetNumber string      `json:"streetNumber,omitempty" db:"streetNumber"`
	Zip          string      `json:"zip,omitempty"`
	City         string      `json:"city,omitempty"`
}

type EegAddress struct {
	Street       string `json:"street,omitempty"`
	StreetNumber string `json:"streetNumber,omitempty" db:"streetNumber"`
	Zip          string `json:"zip,omitempty"`
	City         string `json:"city,omitempty"`
}

type Contact struct {
	Phone null.String `json:"phone,omitempty"`
	Email null.String `json:"email,omitempty"`
}

type AccountInfo struct {
	Iban  null.String `json:"iban"`
	Owner null.String `json:"owner"`
	Sepa  bool        `json:"sepa"`
}

type Optionals struct {
	Website null.String `json:"website,omitempty"`
}
type EegNotification struct {
	Id      int16          `json:"id"`
	MsgType string         `json:"type" db:"type"`
	Message types.JSONText `json:"message" db:"notification"`
	Date    time.Time      `json:"date"`
}
