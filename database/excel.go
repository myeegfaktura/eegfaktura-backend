package database

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gopkg.in/guregu/null.v4"
)

var netOperatorMatch = regexp.MustCompile(`^[A-Z]{2}[0-9]*$`)

func openReader(r io.Reader, filename string, opt ...excelize.Options) (*excelize.File, error) {
	f, err := excelize.OpenReader(r, opt...)
	if err != nil {
		return nil, err
	}
	f.Path = filename
	return f, nil
}

func ImportMasterdataFromExcel(dbConn OpenDbXConnection, r io.Reader, filename, sheet, tenant string) error {
	var f *excelize.File
	var err error

	if f, err = openReader(r, filename); err != nil {
		return err
	}

	defer f.Close()
	log.Debug("Successfully open stream")

	rows, err := f.Rows(sheet)
	if err != nil {
		glog.Error(err)
		return err
	}
	participants := transformExcelData(rows)
	log.Debugf("Rows: %+v\n", rows)
	log.Debugf("LEN _ Import participants: %+v\n", len(participants))

	for _, p := range participants {
		err = ImportParticipant(dbConn, strings.ToUpper(tenant), "excel", p)
		if err != nil {
			log.Errorf("Error Import Participant from Excel: %s", err.Error())
		}
	}

	return nil
}

func findParticipant(participants []*model.EegParticipant, firstname, lastname string) (*model.EegParticipant, bool) {
	for _, p := range participants {
		if p.FirstName == firstname && p.LastName == lastname {
			return p, true
		}
	}
	return nil, false
}

func getColumValue(cols []string, values map[string]int, deName, enName string, defaultValue *string) string {
	idx := -1
	if _, ok := values[strings.ToLower(deName)]; ok {
		idx = values[strings.ToLower(deName)]
	} else if _, ok := values[strings.ToLower(enName)]; ok {
		idx = values[strings.ToLower(enName)]
	}

	if idx < 0 {
		if defaultValue != nil {
			return *defaultValue
		}
		return ""
	}
	if idx >= len(cols) {
		if defaultValue != nil {
			return *defaultValue
		}
		return ""
	}
	return cols[idx]
}

var numberPattern = regexp.MustCompile(`^[0-9\\.,]+$`)

func isDate(cell string) bool {
	if len(cell) > 0 && numberPattern.MatchString(cell) {
		return true
	}
	println(cell)
	return false
}

func parseExcelDate(cell string) time.Time {
	if isDate(cell) {
		var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)
		var days, _ = strconv.ParseFloat(cell, 64)
		return excelEpoch.Add(time.Second * time.Duration(days*86400))
	}
	return time.Now()
}

func transformExcelData(rows *excelize.Rows) []*model.EegParticipant {
	colMap := map[string]int{}
	participants := []*model.EegParticipant{}

	businessRole := func(cols []string, values map[string]int) string {
		val := getColumValue(cols, colMap, "BusinessRole", "BusinessRole", nil)
		if strings.ToLower(val) == "business" {
			return "EEG_BUSINESS"
		}
		return "EEG_PRIVATE"
	}

	equipmentName := func(cols []string, values map[string]int) null.String {
		val := getColumValue(cols, colMap, "ObjektName", "ObjectName", nil)
		if len(val) > 0 {
			return null.StringFrom(val)
		}
		return null.String{}
	}

	equipmentNumber := func(cols []string, values map[string]int) null.String {
		val := getColumValue(cols, colMap, "EquipmentNr", "EquipmentNr", nil)
		if len(val) > 0 {
			return null.StringFrom(val)
		}
		return null.String{}
	}

	for rows.Next() {
		if cols, err := rows.Columns(excelize.Options{RawCellValue: true}); err == nil && len(cols) > 0 {
			switch cols[0] {
			case "[### Leerzeile für Importer ###]":
				continue
			case "Netzbetreiber", "Grid Operator":
				for i, c := range cols {
					colMap[strings.ToLower(c)] = i
				}

				continue
			default:
				switch {
				case netOperatorMatch.MatchString(cols[0]):
					var firstname string
					var lastname string

					excelName1 := getColumValue(cols, colMap, "Name 2", "Name2", nil)
					excelName2 := getColumValue(cols, colMap, "Name 1", "Name1", nil)

					if len(excelName2) == 0 || len(excelName2) < 2 {
						if _, err := fmt.Sscanf(getColumValue(cols, colMap, "Name 2", "Name2", nil), "%s %s", &lastname, &firstname); err != nil {
							fmt.Printf("Error Name extracting: %s (%s)\n", err, getColumValue(cols, colMap, "Name 1", "Name1", nil))
							continue
						}
					} else {
						firstname = excelName2
						lastname = excelName1
					}

					role := model.UNKNOWN
					switch getColumValue(cols, colMap, "Energierichtung", "Energy Direction", nil) {
					case "GENERATION":
						role = model.GENERATOR
					case "CONSUMPTION":
						role = model.CONSUMPTION
					default:
						role = model.CONSUMPTION
					}

					streetNumber := getColumValue(cols, colMap, "Hausnummer", "Street Number", nil)
					var participantSince time.Time
					docSignedAt := getColumValue(cols, colMap, "Dokument unterschrieben", "Document Signature Date", nil)
					if len(docSignedAt) > 0 {
						participantSince = parseExcelDate(docSignedAt)
					} else {
						participantSince = time.Now()
					}

					cpStatus := getColumValue(cols, colMap, "Zählpunktstatus", "Metering Point State", nil)
					if cpStatus == "ACTIVATED" || cpStatus == "REGISTERED" || len(cpStatus) == 0 {
						var participant *model.EegParticipant
						if p, ok := findParticipant(participants, firstname, lastname); ok {
							participant = p
						} else {
							participant = &model.EegParticipant{
								ParticipantNumber: null.StringFrom(getColumValue(cols, colMap, "MitgliedsNr", "ParticipantNr", nil)),
								FirstName:         firstname,
								LastName:          lastname,
								TitleBefore:       getColumValue(cols, colMap, "TitelVor", "TitleBefor", nil),
								TitleAfter:        getColumValue(cols, colMap, "TitelNach", "TitleAfter", nil),
								BusinessRole:      businessRole(cols, colMap),
								ResidentAddress: model.Address{
									Type:         model.RESIDENCE,
									Street:       null.StringFrom(getColumValue(cols, colMap, "Straße", "Street", nil)),
									StreetNumber: null.StringFrom(streetNumber),
									Zip:          null.StringFrom(getColumValue(cols, colMap, "PLZ", "ZIP", nil)),
									City:         null.StringFrom(getColumValue(cols, colMap, "Ort", "City", nil)),
								},
								BillingAddress: model.Address{
									Type:         model.BILLING,
									Street:       null.StringFrom(getColumValue(cols, colMap, "Straße", "Street", nil)),
									StreetNumber: null.StringFrom(streetNumber),
									Zip:          null.StringFrom(getColumValue(cols, colMap, "PLZ", "ZIP", nil)),
									City:         null.StringFrom(getColumValue(cols, colMap, "Ort", "City", nil)),
								},
								Status:           model.ACTIVE,
								ParticipantSince: participantSince,
								MeteringPoint:    []*model.MeteringPoint{},
								BankAccount: model.BankInfo{
									Iban:  null.StringFrom(getColumValue(cols, colMap, "IBAN", "IBAN", nil)),
									Owner: null.StringFrom(getColumValue(cols, colMap, "Kontoinhaber", "Accountname", nil))},
								Contact:   model.ContactInfo{Email: null.StringFrom(getColumValue(cols, colMap, "email", "email", nil))},
								TaxNumber: getColumValue(cols, colMap, "SteuerNr", "taxNumber", nil),
								Version:   0,
							}
							participants = append(participants, participant)
						}
						participant.MeteringPoint = append(participant.MeteringPoint, &model.MeteringPoint{
							MeteringPoint:   getColumValue(cols, colMap, "Zählpunkt", "MeteringPoint Id", nil),
							Transformer:     null.String{},
							Direction:       role,
							Status:          model.ACTIVE,
							TariffId:        null.String{},
							EquipmentNumber: equipmentNumber(cols, colMap),
							EquipmentName:   equipmentName(cols, colMap),
							InverterId:      null.String{},
							Street:          null.StringFrom(getColumValue(cols, colMap, "Straße", "Street", nil)),
							StreetNumber:    null.StringFrom(getColumValue(cols, colMap, "Hausnummer", "Street Number", nil)),
							City:            null.StringFrom(getColumValue(cols, colMap, "Ort", "City", nil)),
							Zip:             null.StringFrom(getColumValue(cols, colMap, "PLZ", "ZIP", nil)),
						})
					}
				}
			}
		}
	}
	return participants
}
