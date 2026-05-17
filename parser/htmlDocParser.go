package parser

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"os"
	"path/filepath"

	"github.com/eegfaktura/eegfaktura-backend/config"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/eegfaktura/eegfaktura-backend/util"
	"github.com/gabriel-vasile/mimetype"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//go:embed templates
var templates embed.FS

func ParseTemplate(templateFileName string, data interface{}) (*bytes.Buffer, error) {

	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return nil, err
	}
	return buf, nil
}

func SendActivationMailFromTemplate(sendMail util.SendMailFunc,
	tenant, subject string, eeg *model.Eeg, participant *model.EegParticipant) error {

	templateConfigDir := filepath.Join(viper.GetString("file-content.templates"), tenant, "templates")
	_, statErr := os.Stat(templateConfigDir)
	if errors.Is(statErr, os.ErrNotExist) {
		templateConfigDir = filepath.Join(viper.GetString("file-content.templates"), "templates")
	}

	templateConfig, err := config.ReadActivationMailTemplateConfig(filepath.Join(templateConfigDir, "activation-mail-template.toml"))
	if err != nil {
		return err
	}

	return sendMailFromTemplate(sendMail, tenant, subject, templateConfigDir, templateConfig, eeg, participant)
}

func SendMeteringPointActiveMailFromTemplate(sendMail util.SendMailFunc,
	tenant, subject, meteringPointId string, eeg *model.Eeg, participant *model.EegParticipant) error {

	templateConfigDir := filepath.Join(viper.GetString("file-content.templates"), tenant, "templates")
	if _, statErr := os.Stat(templateConfigDir); errors.Is(statErr, os.ErrNotExist) {
		templateConfigDir = filepath.Join(viper.GetString("file-content.templates"), "templates")
	}

	templateConfig, err := config.ReadActivationMailTemplateConfig(filepath.Join(templateConfigDir, "zp-active-mail-template.toml"))
	if err != nil {
		return err
	}

	if !participant.Contact.Email.Valid {
		log.Warnf("Participant without email contact: %s (%s)", participant.LastName, participant.Id)
		return nil
	}

	templateData := struct {
		Eeg           *model.Eeg
		Participant   *model.EegParticipant
		MeteringPoint string
	}{eeg, participant, meteringPointId}

	buf, err := ParseTemplate(filepath.Join(templateConfigDir, templateConfig.TemplateFile), templateData)
	if err != nil {
		return err
	}

	return sendMail(tenant, participant.Contact.Email.String, subject, buf, buildAttachments(templateConfigDir, templateConfig.InlinePictures))
}

func sendMailFromTemplate(sendMail util.SendMailFunc, tenant, subject, templatePath string, templateConfig *model.ActivationMailTemplate, eeg *model.Eeg, participant *model.EegParticipant) error {
	meterIds := []string{}
	for i := range participant.MeteringPoint {
		meterIds = append(meterIds, participant.MeteringPoint[i].MeteringPoint)
	}

	templateData := struct {
		Eeg            *model.Eeg
		Participant    *model.EegParticipant
		Meteringpoints []string
	}{eeg, participant, meterIds}

	if !participant.Contact.Email.Valid {
		log.Warnf("Participant without email contact: %s (%s)", participant.LastName, participant.Id)
		return nil
	}

	tmpPath := filepath.Join(templatePath, templateConfig.TemplateFile)
	buf, err := ParseTemplate(tmpPath, templateData)
	if err != nil {
		return err
	}

	return sendMail(tenant, participant.Contact.Email.String,
		subject, buf, buildAttachments(templatePath, templateConfig.InlinePictures))
}

func GetTemplateFor(templateType, tenant string) (string, error) {

	path := filepath.Join(viper.GetString("file-content.templates"), tenant, "templates")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join("../public/templates")
	}

	switch templateType {
	case "ACTIVATION":
		// filepath.ToSlash makes the returned template path consistent
		// across operating systems — file.Open on Windows accepts
		// forward slashes, and downstream callers (and tests) compare
		// against forward-slash form.
		return filepath.ToSlash(filepath.Join(path, "AktivierungsEmail-templates.html")), nil
	}
	return "", errors.New("template not found")
}

func buildAttachments(templatePath string, a []model.InlinePicture) []*util.Attachment {
	attachments := []*util.Attachment{}
	for i := range a {
		att := a[i]
		data, err := os.ReadFile(filepath.Join(templatePath, att.Filepath))
		if err != nil {
			log.Errorf("Read Attachment. Reason: %+v", err)
			continue
		}
		mime := mimetype.Detect(data)
		attachments = append(attachments, &util.Attachment{
			Type:        "INLINE",
			Filename:    filepath.Base(att.Filepath),
			Filecontent: bytes.NewBuffer(data),
			MimeType:    mime.String(),
			ContentId:   &att.ContentId,
		})
	}
	return attachments
}
