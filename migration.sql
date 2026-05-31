CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS base;

CREATE TABLE IF NOT EXISTS base.EEG
(
    tenant               VARCHAR PRIMARY KEY,
    name                 TEXT    NOT NULL,
    description          TEXT,
    periods              JSON             DEFAULT ('[]'),
    "rcNumber"           TEXT    NOT NULL,
    area                 TEXT    NOT NULL, /* Ortsgebiet (LOCAL | REGIONAL) */
    legal                TEXT    NOT NULL DEFAULT 'verein', /* Unternehmensform ("verein" | "genossenschaft" | "geselschaft") */
    gridoperator_code    TEXT    NOT NULL,
    gridoperator_name    TEXT    NOT NULL,
    "communityId"        TEXT    NOT NULL,
    "businessNr"         INTEGER,
    "allocationMode"     TEXT    NOT NULL DEFAULT 'DYNAMIC', /* "DYNAMIC" | "STATIC" */
    "settlementInterval" TEXT    NOT NULL DEFAULT 'MONTHLY', /* "MONTHLY" | "ANNUAL" | BIANNUAL | QUARTER*/
    "providerBusinessNr" INTEGER,
    "taxNumber"          TEXT,
    "vatNumber"          TEXT,
    subjecttovat         BOOLEAN,
    "contactPerson"      TEXT,
    -- Address Info
    street               TEXT    NOT NULL,
    "streetNumber"       TEXT    NOT NULL,
    city                 TEXT    NOT NULL,
    zip                  TEXT    NOT NULL,
    -- Account Info
    iban                 TEXT,
    owner                TEXT,
    sepa                 BOOLEAN NOT NULL DEFAULT false,
    -- Contact Info
    phone                TEXT,
    email                TEXT    NOT NULL,
    website              TEXT,

    online               BOOLEAN NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_eeg ON base.EEG (tenant, name, "rcNumber");

CREATE TABLE IF NOT EXISTS base.tariff
(
    id                   UUID    NOT NULL DEFAULT uuid_generate_v4(),
    tenant               VARCHAR NOT NULL,
    type                 VARCHAR NOT NULL, /* 'tariff type like EEG, VZP, EZP, AKONTO' */
    name                 TEXT    NOT NULL,
    "billingPeriod"      TEXT             DEFAULT 'monthly',
    "useVat"             BOOLEAN          DEFAULT FALSE,
    "vatInPercent"       NUMERIC,
    "accountNetAmount"   NUMERIC,
    "accountGrossAmount" NUMERIC,
    "participantFee"     NUMERIC,
    "baseFee"            FLOAT   NOT NULL,
    "freeKWh"            INTEGER,
    "businessNr"         INTEGER,
    "createdBy"          TEXT,
    "createdDate"        DATE,
    "lastModifiedDate"   DATE,
    version              INTEGER,
    "centPerKWh"         FLOAT,
    discount             INTEGER,
    status               TEXT    NOT NULL DEFAULT 'ACTIVE', /* ACTIVE | INACTIVE */
    "inactiveSince"      DATE,
    CONSTRAINT TariffPK PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tariff ON base.tariff (id, tenant, name, type, version);

CREATE TABLE IF NOT EXISTS base.participant
(
    id                      UUID    NOT NULL DEFAULT uuid_generate_v4(),
    "participantNumber"     VARCHAR,
    tenant                  VARCHAR NOT NULL,
    firstname               VARCHAR NOT NULL,
    lastname                VARCHAR NOT NULL,
    role                    VARCHAR NOT NULL DEFAULT 'EEG_USER', /* 'EEG_USER' | 'EEG_ADMIN' */
    "businessRole"          VARCHAR NOT NULL DEFAULT 'EEG_PRIVATE', /* 'EEG_PRIVATE' | 'EEG_BUSINESS' */
    "titleBefore"           VARCHAR,
    "titleAfter"            VARCHAR,
    "participantSince"      DATE    NOT NULL DEFAULT now(),
    "vatNumber"             VARCHAR,
    "taxNumber"             VARCHAR,
    "companyRegisterNumber" VARCHAR,
    status                  VARCHAR NOT NULL DEFAULT 'NEW', /* 'NEW' | 'PENDING' | 'ACCEPTED' | 'ACTIVE' | 'INACTIVE' */
    "createdBy"             VARCHAR NOT NULL,
    "createdDate"           DATE             DEFAULT now(),
    "lastModifiedBy"        VARCHAR NOT NULL,
    "lastModifiedDate"      DATE             DEFAULT now(),
    version                 INTEGER          DEFAULT 1,
    "tariffId"              uuid,
    CONSTRAINT ParticipantPK PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_participant_tenant ON base.participant (id, tenant, version);

CREATE TABLE IF NOT EXISTS base.contactdetail
(
    id             UUID NOT NULL DEFAULT uuid_generate_v4(),
    participant_id UUID NOT NULL,
    email          TEXT,
    phone          TEXT,
    CONSTRAINT contactdetailsPK PRIMARY KEY (id),
    CONSTRAINT FK_ParticipantDetail FOREIGN KEY (participant_id) REFERENCES base.participant (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS base.address
(
    id             UUID NOT NULL DEFAULT uuid_generate_v4(),
    participant_id UUID NOT NULL,
    type           TEXT NOT NULL DEFAULT 'RESIDENCE', /*Address-Types: 'RESIDENCE' | 'BILLING' */
    street         TEXT,
    "streetNumber" TEXT,
    city           TEXT,
    zip            TEXT,
    CONSTRAINT addressPK PRIMARY KEY (id),
    CONSTRAINT FK_ParticipantAddress FOREIGN KEY (participant_id) REFERENCES base.participant (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS base.bankaccount
(
    id             UUID NOT NULL DEFAULT uuid_generate_v4(),
    participant_id UUID NOT NULL,
    iban           TEXT NOT NULL,
    owner          TEXT,
    CONSTRAINT bankaccountPK PRIMARY KEY (id),
    CONSTRAINT FK_ParticipantBankaccount FOREIGN KEY (participant_id) REFERENCES base.participant (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS base.meteringpoint
(
    metering_point_id  TEXT      NOT NULL,
    consent_id         TEXT,
    participant_id     UUID      NOT NULL,
    tenant             TEXT      NOT NULL,
    grid_operator_name VARCHAR,
    grid_operator_id   VARCHAR,
    allocation_factor  FLOAT,
    transformer        TEXT,
    direction          TEXT      NOT NULL DEFAULT 'CONSUMPTION', /* 'GENERATION' | 'CONSUMPTION' */
    status             TEXT      NOT NULL DEFAULT 'NEW', /* "NEW" | "PENDING" | "ACCEPTED" | "ACTIVE" | "INACTIVE" */
    process_state      TEXT      NOT NULL DEFAULT 'NEW', /* roundtrip state, updated by EDA-inbound handlers */
    last_process_state TEXT,
    "statusCode"       INTEGER,
    tariff_id          UUID,
    inverterid         TEXT,
    "equipmentNumber"  TEXT,
    "equipmentName"    TEXT,
    street             TEXT,
    "streetNumber"     TEXT,
    city               TEXT,
    zip                TEXT,
    "registeredSince"  DATE      NOT NULL DEFAULT now(),
    "modifiedAt"       TIMESTAMP NOT NULL DEFAULT now(),
    "modifiedBy"       TEXT,
    activeSince        DATE,
    inactiveSince      DATE,
    active             INT       NOT NULL DEFAULT 1,
    flag               INT       NOT NULL DEFAULT 1,
    CONSTRAINT meteringpointPK PRIMARY KEY (metering_point_id, tenant),
    CONSTRAINT FK_ParticipantMeteringpoint FOREIGN KEY (participant_id) REFERENCES base.participant (id) ON DELETE CASCADE
--     CONSTRAINT FK_TariffMeteringpoint FOREIGN KEY (tariff_id) REFERENCES base.tariff (id)
);
-- NOTE: this file was historically a snapshot of the original public-
-- fork schema. The src-stack DB is initialized via pg_dump from prod
-- (see eegfaktura-platform/scripts/bootstrap-poc-data.sh), so the
-- live DB has the columns above. The EDA-inbound handlers
-- (protocolEcPodListHandler, protocolEcPrtChangeHandler) require
-- consent_id, grid_operator_id/name, process_state, activeSince,
-- inactiveSince, flag. Greenfield deployments without pg_dump
-- restore need to apply this DDL before the backend starts — see
-- memory pilot-greenfield-schema-init.

CREATE TABLE IF NOT EXISTS base.notification
(
    id           SERIAL PRIMARY KEY,
    tenant       TEXT      NOT NULL,
    type         TEXT      NOT NULL DEFAULT 'MESSAGE',/* MESSAGE TYPE DESCRIBE 'ERROR' | 'MESSAGE' | 'NOTIFICATION' */
    notification json      NOT NULL DEFAULT '{}',
    date         TIMESTAMP NOT NULL DEFAULT now(),
    role         VARCHAR   NOT NULL DEFAULT 'ADMIN' /* 'USER' | 'ADMIN' */
);

CREATE TABLE IF NOT EXISTS base.processhistory
(
    id               UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant           TEXT NOT NULL,
    "conversationId" TEXT NOT NULL,
    type             TEXT NOT NULL,
    date             TIMESTAMP NOT NULL             DEFAULT now(),
    issuer           TEXT NOT NULL,
    message          json NOT NULL             DEFAULT '{}',
    direction        TEXT NOT NULL             DEFAULT 'OUT', /* MESSAGE DIRECTION 'OUT' | 'IN' */
    protocol         VARCHAR
);


CREATE VIEW base.activeTariff AS
SELECT id,
       name,
       tenant,
       "billingPeriod",
       "useVat",
       "vatInPercent",
       "accountNetAmount",
       "accountGrossAmount",
       "participantFee",
       "baseFee",
       "businessNr",
       version,
       type,
       "centPerKWh",
       discount,
       "freeKWh"
FROM base.tariff,
     (SELECT id as tid, MAX(version) as tversion FROM base.tariff GROUP BY id) as x
WHERE id = x.tid
  AND version = x.tversion AND status != 'ARCHIVED';



CREATE VIEW
    base.billing_masterdata AS
SELECT p.id                                                      participant_id,
       p."titleBefore"                                           participant_title_before,
       p.firstname                                               participant_firstname,
       p."participantNumber"                                     participant_number,
       p.lastname                                                participant_lastname,
       p."titleAfter"                                            participant_title_after,
       p."vatNumber"                                             participant_vat_id,
       p."taxNumber"                                             participant_tax_id,
       p."companyRegisterNumber"                                 participant_company_register_number,
       pm.metering_point_id                                      metering_point_id,
       pm."equipmentNumber"                                      equipment_number,
       pm."equipmentName"                                        metering_equipment_name,
       (CASE WHEN pm.direction = 'GENERATION' THEN 0 ELSE 1 END) metering_point_type,
       c.tenant                                                  eec_id,
       c."rcNumber"                                              tenant_id,
       c.name                                                    eec_name,
       c."vatNumber"                                             eec_vat_id,
       c."taxNumber"                                             eec_tax_id,
       c."businessNr"                                            eec_company_register_number,
       c.subjecttovat                                            eec_subject_to_vat,
       c.phone                                                   eec_phone,
       c.email                                                   eec_email,
       c.website                                                 eec_website,
       concat(c.street, ' ', c."streetNumber")                   eec_street,
       c.zip                                                     eec_zip_code,
       c.city                                                    eec_city,
       concat(p_address.street, ' ', p_address."streetNumber")   participant_street,
       p_address.zip                                             participant_zip_code,
       p_address.city                                            participant_city,
       t.type                                                    tariff_type,
       t.name                                                    tariff_name,
       t."billingPeriod"                                         tariff_billing_period,
       t."useVat"                                                tariff_use_vat,
       t."vatInPercent"                                          tariff_vat_in_percent,
       t."participantFee"                                        tariff_participant_fee,
       t."baseFee"                                               tariff_basic_fee,
       t.discount                                                tariff_discount,
       t."centPerKWh"                                            tariff_working_fee_per_consumedkwh,
       t."centPerKWh"                                            tariff_credit_amount_per_producedkwh,
       t."freeKWh"                                               tariff_freekwh,
       'Bank Name'                                               participant_bank_name,
       b.iban                                                    participant_bank_iban,
       b.owner                                                   participant_bank_owner,
       o.email                                                   participant_email,
       'Bank Name'                                               eec_bank_name,
       c.iban                                                    eec_bank_iban,
       c.owner                                                   eec_bank_owner,
       'SEPA Mandat'                                             participant_sepa_mandate_reference
FROM base.participant p
         LEFT JOIN base.eeg c ON c.tenant = p.tenant
         LEFT JOIN base.meteringpoint pm ON pm.participant_id = p.id
         LEFT JOIN base.address p_address ON p.id = p_address.participant_id AND p_address.type = 'BILLING'
         LEFT JOIN base.activetariff t ON t.id = pm.tariff_id
         LEFT JOIN base.bankaccount b ON b.participant_id = p.id
         LEFT JOIN base.contactdetail o ON o.participant_id = p.id;



create table alembic_version
(
    version_num varchar(32) not null
        constraint alembic_version_pkc
            primary key
);

alter table alembic_version
    owner to vfeeg;

create table file_categories
(
    id   uuid default uuid_generate_v4() not null
        primary key,
    name varchar(128)
);

alter table file_categories
    owner to vfeeg;

create table storages
(
    id            uuid default uuid_generate_v4() not null
        primary key,
    community_id  varchar(8)                      not null,
    name          varchar(128),
    configuration json
);

alter table storages
    owner to vfeeg;

create table file_containers
(
    id               uuid default uuid_generate_v4() not null
        primary key,
    name             varchar(128),
    configuration    json,
    file_category_id uuid                            not null
        references file_categories,
    community_id     varchar(8)                      not null,
    storage_id       uuid                            not null
        references storages
);

alter table file_containers
    owner to vfeeg;

create table files
(
    id                uuid      default uuid_generate_v4() not null
        primary key,
    name              varchar(128),
    file_container_id uuid                                 not null
        references file_containers,
    community_id      varchar(8)                           not null,
    user_id           uuid,
    created_at        timestamp default now()              not null
);

alter table files
    owner to vfeeg;

create table file_attributes
(
    file_id uuid         not null
        references files,
    key     varchar(128) not null,
    value   varchar(512) not null,
    primary key (file_id, key)
);

alter table file_attributes
    owner to vfeeg;

-- ----------------------------------------------------------------------
-- Metering-Point Partition Factor history
--
-- Stores the per-meter partition factor versioned by SERIAL. The
-- activeMeteringPartition view exposes only the latest version per
-- metering point. New partition-factor values are appended (INSERT
-- with the next version) rather than updated, so the history is
-- preserved for audit / billing.
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS base.metering_partition_factor
(
    metering_point_id TEXT    NOT NULL,
    version           SERIAL,
    participant_id    UUID    NOT NULL,
    tenant            TEXT    NOT NULL,
    "partFact"        INTEGER NOT NULL,
    "createdAt"       DATE    NOT NULL DEFAULT now(),
    "createdBy"       VARCHAR NOT NULL,
    CONSTRAINT meteringpointPartitionPK PRIMARY KEY (metering_point_id, version),
    CONSTRAINT FK_MeteringpointPartition FOREIGN KEY (metering_point_id, tenant)
        REFERENCES base.meteringpoint (metering_point_id, tenant) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE OR REPLACE VIEW base.activeMeteringPartition AS
SELECT *
FROM (
    SELECT *, ROW_NUMBER() OVER (
        PARTITION BY metering_point_id, participant_id
        ORDER BY version DESC
    ) AS rowid
    FROM base.metering_partition_factor
) AS partp
WHERE rowid = 1;
