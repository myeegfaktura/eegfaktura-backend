package model

import (
	"github.com/pborman/uuid"
	"gopkg.in/guregu/null.v4"
	"time"
)

type EegParticipant struct {
	Id                    uuid.UUID        `json:"id" goqu:"skipupdate"`
	ParticipantNumber     null.String      `json:"participantNumber" db:"participantNumber"`
	BusinessRole          string           `json:"businessRole" db:"businessRole"`
	Role                  string           `json:"role" db:"role"`
	FirstName             string           `json:"firstname"`
	LastName              string           `json:"lastname"`
	TitleBefore           string           `json:"titleBefore" db:"titleBefore"`
	TitleAfter            string           `json:"titleAfter" db:"titleAfter"`
	ParticipantSince      time.Time        `json:"participantSince" db:"participantSince" goqu:"defaultifempty"`
	VatNumber             string           `json:"vatNumber" db:"vatNumber"`
	TaxNumber             string           `json:"taxNumber" db:"taxNumber"`
	CompanyRegisterNumber null.String      `json:"companyRegisterNumber" db:"companyRegisterNumber"`
	Contact               ContactInfo      `json:"contact" db:"-" goqu:"skipinsert"`
	BillingAddress        Address          `json:"billingAddress" db:"-" goqu:"skipinsert"`
	ResidentAddress       Address          `json:"residentAddress" db:"-" goqu:"skipinsert"`
	BankAccount           BankInfo         `json:"accountInfo" db:"-" goqu:"skipinsert"`
	MeteringPoint         []*MeteringPoint `json:"meters" db:"-" goqu:"skipinsert"`
	TariffId              null.String      `json:"tariffId" db:"tariffId" goqu:"skipinsert"`
	Status                StatusType       `json:"status" goqu:"defaultifempty"`
	Version               int              `json:"version,omitempty" goqu:"defaultifempty"`
	CreatedBy             string           `json:"createdBy,omitempty" db:"createdBy"`
}

type ContactInfo struct {
	Phone null.String `json:"phone" db:"phone"`
	Email null.String `json:"email" db:"email"`
}

type BankInfo struct {
	Iban  null.String `json:"iban"`
	Owner null.String `json:"owner"`
}

type DirectionType string

const (
	CONSUMPTION DirectionType = "CONSUMPTION"
	GENERATOR   DirectionType = "GENERATION"
	UNKNOWN     DirectionType = "UNKNOWN"
)

type StatusType string

const (
	NEW      StatusType = "NEW"
	INIT     StatusType = "INIT"     // request initiated, before grid-provider answer
	PENDING  StatusType = "PENDING"  // answer message from grid-provider was received
	APPROVED StatusType = "APPROVED" // participant accepted in the grid-operator portal
	ACTIVE   StatusType = "ACTIVE"
	INACTIVE StatusType = "INACTIVE"
	REJECTED StatusType = "REJECTED"
	REVOKED  StatusType = "REVOKED"
	INVALID  StatusType = "INVALID"
	ARCHIVED StatusType = "ARCHIVED"
	ABORTED  StatusType = "ABORTED"
	RESTORE  StatusType = "RESTORE"
)

type MeteringPoint struct {
	MeteringPoint   string        `json:"meteringPoint" db:"metering_point_id"`
	Transformer     null.String   `json:"transformer,omitempty"`
	Direction       DirectionType `json:"direction,omitempty"`
	Status          StatusType    `json:"status,omitempty"`
	TariffId        null.String   `json:"tariffId" db:"tariff_id"`
	EquipmentNumber null.String   `json:"equipmentNumber,omitempty" db:"equipmentNumber"`
	EquipmentName   null.String   `json:"equipmentName,omitempty" db:"equipmentName"`
	InverterId      null.String   `json:"inverterId,omitempty" db:"inverterid"`
	Street          null.String   `json:"street,omitempty"`
	StreetNumber    null.String   `json:"streetNumber,omitempty" db:"streetNumber"`
	City            null.String   `json:"city,omitempty"`
	Zip             null.String   `json:"zip,omitempty"`
	RegisteredSince time.Time     `json:"registeredSince" db:"registeredSince"`
	ModifiedAt      time.Time     `json:"modifiedAt" db:"modifiedAt"`
	ModifiedBy      null.String   `json:"modifiedBy" db:"modifiedBy"`
}

// ChangePartitionFactorRequest is one entry in the payload of the
// POST /meteringpoint/changepartitionfactor endpoint. The route
// accepts a list of these (one per metering point whose partition
// factor changes), and dispatches one EBMS_REQ_CHANGE_PARTFACT
// message per receiving grid operator.
//
// GridOperatorId can override the EEG's default grid operator for a
// single metering point (useful when meters live across operator
// boundaries). Activation is the timestamp from which the new
// partition factor takes effect.
type ChangePartitionFactorRequest struct {
	MeteringPoint  string        `json:"meter"`
	Direction      DirectionType `json:"direction"`
	GridOperatorId null.String   `json:"gridOperatorId,omitempty"`
	Activation     time.Time     `json:"activation"`
	PartFact       int           `json:"partFact"`
}
