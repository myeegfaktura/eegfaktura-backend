package database

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/golang/glog"
	"github.com/jjeffery/civil"
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

// ExportMasterdataToExcel builds an .xlsx workbook with two sheets:
// the EEG master sheet (sheet name = RcNumber) and the participant
// roster sheet ("Mitglieder", one row per (participant × metering
// point) tuple). Mirrors prod's vfeeg-backend output so the customer
// SPA's masterdata download keeps the same shape across stacks.
func ExportMasterdataToExcel(participants []model.EegParticipant, eeg *model.Eeg, tariffMap map[string]string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.WithField("tenant", eeg.Id).WithError(err).Error("Error while closing file")
		}
	}()

	if err := generateEegMastersheet(f, eeg); err != nil {
		return nil, err
	}
	if err := generateParticipantMastersheet(f, participants, tariffMap); err != nil {
		return nil, err
	}

	_ = f.DeleteSheet("Sheet1")
	return f.WriteToBuffer()
}

func generateEegMastersheet(f *excelize.File, eeg *model.Eeg) error {
	styleId, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10.0}})
	styleIdHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10.0},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	styleIdHeaderTop, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11.0},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#cccccc"},
			Shading: 0,
		},
	})

	line := 1
	sheet := eeg.RcNumber
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{"EEG"})
	_ = f.SetRowStyle(sheet, 1, 1, styleIdHeaderTop)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		"Kurzname", "Bezeichnung", "Gemeinschafts-ID", "Ponton",
	})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeader)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		eeg.Name, eeg.Description, eeg.CommunityId, eeg.Online,
	})
	_ = f.SetRowStyle(sheet, line, line, styleId)

	line += 2
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{"Netz"})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeaderTop)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		"Netzbetreiber", "Netzbetreiber Name", "Verteilung",
	})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeader)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		eeg.GridOperator, eeg.OperatorName, eeg.AllocationMode,
	})
	_ = f.SetRowStyle(sheet, line, line, styleId)

	line += 2
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{"Kontakt"})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeaderTop)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		"Kontaktperson", "E-Mail", "TelefonNr.", "PLZ", "Wohnort", "Straße", "StraßenNr.", "Web Seite",
	})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeader)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		eeg.ContactPerson.String, eeg.Email.String, eeg.Phone.String, eeg.Zip, eeg.City, eeg.Street, eeg.StreetNumber, eeg.Website.String,
	})
	_ = f.SetRowStyle(sheet, line, line, styleId)

	line += 2
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{"Bankdaten"})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeaderTop)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		"Kontoinhaber", "IBAN", "SEPA",
	})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeader)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		eeg.Owner.String, eeg.Iban.String, eeg.Sepa,
	})
	_ = f.SetRowStyle(sheet, line, line, styleId)

	line += 2
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{"Geschäftliches"})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeaderTop)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		"Rechtsform", "Geschäftsnummer", "Verrechnungsinterval", "Ust.", "SteuerNr.",
	})
	_ = f.SetRowStyle(sheet, line, line, styleIdHeader)
	line += 1
	_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", line), &[]interface{}{
		eeg.Legal, eeg.BusinessNr.String, eeg.SettlementInterval, eeg.VatNumber.String, eeg.TaxNumber.String,
	})
	_ = f.SetRowStyle(sheet, line, line, styleId)

	_ = f.SetColWidth(sheet, "A", "B", 25.0)
	_ = f.SetColWidth(sheet, "C", "C", 35.0)
	_ = f.SetColWidth(sheet, "D", "H", 20.0)

	return nil
}

func generateParticipantMastersheet(f *excelize.File, participants []model.EegParticipant, tariffMap map[string]string) error {
	getTariffName := func(id string) string {
		name, ok := tariffMap[id]
		if !ok {
			return ""
		}
		return name
	}

	getNullDate := func(d civil.NullDate) string {
		if !d.Valid {
			return ""
		}
		return d.Date.String()
	}

	styleId, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10.0}})
	styleDateId, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10.0}, NumFmt: 14})
	styleIdHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10.0},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	styleIdDate, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 10.0},
		NumFmt: 14,
	})

	sheet := "Mitglieder"
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}

	_ = sw.SetColWidth(1, 1, 5.0)
	_ = sw.SetColWidth(2, 3, 30.0)
	colNr, _ := excelize.ColumnNameToNumber("F")
	_ = sw.SetColWidth(colNr, colNr, 12.0)
	_ = sw.SetColWidth(colNr+1, colNr+1, 25.0)
	_ = sw.SetColWidth(colNr+2, colNr+7, 20.0)
	colNr, _ = excelize.ColumnNameToNumber("O")
	_ = sw.SetColWidth(colNr, colNr, 20.0)
	_ = sw.SetColWidth(colNr+1, colNr+1, 12.0)
	colNr, _ = excelize.ColumnNameToNumber("R")
	_ = sw.SetColWidth(colNr, colNr+1, 20.0)
	colNr, _ = excelize.ColumnNameToNumber("Y")
	_ = sw.SetColWidth(colNr, colNr+1, 32.0)
	_ = sw.SetColWidth(colNr+3, colNr+3, 8.0)
	_ = sw.SetColWidth(colNr+4, colNr+4, 20.0)
	_ = sw.SetColWidth(colNr+6, colNr+6, 18.0)
	colNr, _ = excelize.ColumnNameToNumber("AI")
	_ = sw.SetColWidth(colNr, colNr+1, 20.0)
	_ = sw.SetColWidth(colNr+3, colNr+4, 12.0)
	_ = sw.SetColWidth(colNr+5, colNr+5, 30.0)

	line := 1
	_ = sw.SetRow(fmt.Sprintf("A%d", line),
		[]interface{}{
			excelize.Cell{Value: "Mit. Nr."},
			excelize.Cell{Value: "Name 1"},
			excelize.Cell{Value: "Name 2"},
			excelize.Cell{Value: "Titel"},
			excelize.Cell{Value: "Status"},
			excelize.Cell{Value: "Mitglied seit."},
			excelize.Cell{Value: "E-Mail"},
			excelize.Cell{Value: "Telefonnummer"},
			excelize.Cell{Value: "SteuerNr."},
			excelize.Cell{Value: "Ust."},
			excelize.Cell{Value: "IBAN."},
			excelize.Cell{Value: "Kontoinhaber"},
			excelize.Cell{Value: "Bankname"},
			excelize.Cell{Value: "DebitType"},
			excelize.Cell{Value: "Mandat-Ref."},
			excelize.Cell{Value: "Mandat-Dat."},
			excelize.Cell{Value: "PLZ"},
			excelize.Cell{Value: "Ort"},
			excelize.Cell{Value: "Straße"},
			excelize.Cell{Value: "HausNr."},
			excelize.Cell{Value: ""},
			excelize.Cell{Value: "EEG-Role"},
			excelize.Cell{Value: "teilnahme als"},
			excelize.Cell{Value: "Status"},
			excelize.Cell{Value: "Mitgliedstarif"},
			excelize.Cell{Value: "Zählpunkt"},
			excelize.Cell{Value: "ZP-Status"},
			excelize.Cell{Value: "ZpNr."},
			excelize.Cell{Value: "Zählpunktname"},
			excelize.Cell{Value: "registriert"},
			excelize.Cell{Value: "Bezugsrichtung"},
			excelize.Cell{Value: "Teilnahme Fkt."},
			excelize.Cell{Value: "WechselrichterNr."},
			excelize.Cell{Value: "PLZ"},
			excelize.Cell{Value: "Ort"},
			excelize.Cell{Value: "Straße"},
			excelize.Cell{Value: "HausNr."},
			excelize.Cell{Value: "aktiviert"},
			excelize.Cell{Value: "deaktiviert"},
			excelize.Cell{Value: "Zp. Tarifname"},
			excelize.Cell{Value: "Umspannwerk"},
		}, excelize.RowOpts{StyleID: styleIdHeader, Height: 0.42 * 72})
	for _, c := range participants {
		for _, m := range c.MeteringPoint {
			line = line + 1
			activeSince, inactiveSince := civil.NullDate{}, civil.NullDate{}
			if m.State != nil {
				activeSince = m.State.ActiveSince
				inactiveSince = m.State.InactiveSince
			}
			var mandateDateCell interface{} = ""
			if c.BankAccount.MandateDate.Valid {
				mandateDateCell = c.BankAccount.MandateDate.Date
			}
			_ = sw.SetRow(fmt.Sprintf("A%d", line),
				[]interface{}{
					excelize.Cell{Value: c.ParticipantNumber.String},
					excelize.Cell{Value: c.FirstName},
					excelize.Cell{Value: c.LastName},
					excelize.Cell{Value: joinTitles(c.TitleBefore, c.TitleAfter)},
					excelize.Cell{Value: string(c.Status)},
					excelize.Cell{Value: c.ParticipantSince.Format("2006-01-02"), StyleID: styleIdDate},
					excelize.Cell{Value: c.Contact.Email.String},
					excelize.Cell{Value: c.Contact.Phone.String},
					excelize.Cell{Value: c.TaxNumber},
					excelize.Cell{Value: c.VatNumber},
					excelize.Cell{Value: c.BankAccount.Iban.String},
					excelize.Cell{Value: c.BankAccount.Owner.String},
					excelize.Cell{Value: c.BankAccount.BankName.String},
					excelize.Cell{Value: c.BankAccount.SepaDirectDebit.String},
					excelize.Cell{Value: c.BankAccount.MandateReference.String},
					excelize.Cell{Value: mandateDateCell, StyleID: styleDateId},
					excelize.Cell{Value: c.BillingAddress.Zip.String},
					excelize.Cell{Value: c.BillingAddress.City.String},
					excelize.Cell{Value: c.BillingAddress.Street.String},
					excelize.Cell{Value: c.BillingAddress.StreetNumber.String},
					excelize.Cell{Value: c.CompanyRegisterNumber.String},
					excelize.Cell{Value: c.Role},
					excelize.Cell{Value: businessRoleLabel(c.BusinessRole)},
					excelize.Cell{Value: string(c.Status)},
					excelize.Cell{Value: getTariffName(c.TariffId.String), StyleID: styleDateId},
					excelize.Cell{Value: m.MeteringPoint},
					excelize.Cell{Value: m.ProcessState},
					excelize.Cell{Value: m.EquipmentNumber.String},
					excelize.Cell{Value: m.EquipmentName.String},
					excelize.Cell{Value: m.RegisteredSince, StyleID: styleDateId},
					excelize.Cell{Value: string(m.Direction)},
					excelize.Cell{Value: fmt.Sprintf("%d %%", m.PartFact)},
					excelize.Cell{Value: m.InverterId.String},
					excelize.Cell{Value: m.Zip.String},
					excelize.Cell{Value: m.City.String},
					excelize.Cell{Value: m.Street.String},
					excelize.Cell{Value: m.StreetNumber.String},
					excelize.Cell{Value: getNullDate(activeSince), StyleID: styleDateId},
					excelize.Cell{Value: getNullDate(inactiveSince), StyleID: styleDateId},
					excelize.Cell{Value: getTariffName(m.TariffId.String), StyleID: styleDateId},
					excelize.Cell{Value: m.Transformer.String},
				}, excelize.RowOpts{StyleID: styleId})
		}
	}

	_ = f.AutoFilter(sheet, "A1:AH10", nil)
	return sw.Flush()
}

func joinTitles(before, after string) string {
	titles := []string{}
	if before != "" {
		titles = append(titles, before)
	}
	if after != "" {
		titles = append(titles, after)
	}
	return strings.Join(titles, ", ")
}

func businessRoleLabel(role string) string {
	if role == "EEG_PRIVATE" {
		return "Privat"
	}
	return "Business"
}

// ExportZPListToExcel renders the meter list from an inbound EBMS
// ZP-list response (CR_PODLIST → SENDEN_ECP) into a single-sheet xlsx
// workbook. The bytes.Buffer is suitable as a mail attachment payload.
//
// Used by the EDA POD-list handler so the EEG admin gets a human-
// readable copy of what the grid operator reported. The same data also
// flows into base.meteringpoint via SyncActiveMeteringPoints; the xlsx
// is the operator-friendly side.
//
// Wire-format parity with prod (obpeter/vfeeg-backend@e1755a1
// database/excel.go:730).
func ExportZPListToExcel(ebmsMsg *model.EbmsMessage) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.WithError(err).Error("ExportZPListToExcel: close")
		}
	}()

	if err := generateZPListMastersheet(f, ebmsMsg); err != nil {
		return nil, err
	}

	_ = f.DeleteSheet("Sheet1")
	return f.WriteToBuffer()
}

func generateZPListMastersheet(f *excelize.File, ebmsMsg *model.EbmsMessage) error {
	styleId, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10.0}})
	styleIdHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10.0},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	styleIdDate, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 10.0},
		NumFmt: 14,
	})

	sheet := "ZP-List"
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}

	_ = sw.SetColWidth(1, 1, 5.0)
	_ = sw.SetColWidth(2, 3, 30.0)
	_ = sw.SetColWidth(4, 4, 20.0)
	_ = sw.SetColWidth(5, 5, 9.5)
	colNr, _ := excelize.ColumnNameToNumber("G")
	_ = sw.SetColWidth(colNr, colNr+3, 12.0)

	line := 1
	_ = sw.SetRow(fmt.Sprintf("A%d", line),
		[]interface{}{
			excelize.Cell{Value: "Nr."},
			excelize.Cell{Value: "Zählpunktname"},
			excelize.Cell{Value: "ConsentID"},
			excelize.Cell{Value: "Bezugsrichtung"},
			excelize.Cell{Value: "Teilnahme-faktor"},
			excelize.Cell{Value: "statische Aufteilung"},
			excelize.Cell{Value: "aktiviert"},
			excelize.Cell{Value: "aktiv seit"},
			excelize.Cell{Value: "aktiv bis"},
		}, excelize.RowOpts{StyleID: styleIdHeader, Height: 0.42 * 72})

	for idx, m := range ebmsMsg.MeterList {
		line++
		_ = sw.SetRow(fmt.Sprintf("A%d", line),
			[]interface{}{
				excelize.Cell{Value: idx + 1},
				excelize.Cell{Value: m.MeteringPoint},
				excelize.Cell{Value: m.ConsentID},
				excelize.Cell{Value: string(m.Direction)},
				excelize.Cell{Value: m.PartFact},
				excelize.Cell{Value: m.Share},
				excelize.Cell{Value: time.UnixMilli(m.Activation), StyleID: styleIdDate},
				excelize.Cell{Value: time.UnixMilli(m.From), StyleID: styleIdDate},
				excelize.Cell{Value: time.UnixMilli(m.To), StyleID: styleIdDate},
			}, excelize.RowOpts{StyleID: styleId})
	}

	return sw.Flush()
}
