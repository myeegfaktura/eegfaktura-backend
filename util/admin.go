package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/factory"
	protobuf "github.com/eegfaktura/eegfaktura-backend/proto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type SendMailFunc func(tenant, to, subject string, body *bytes.Buffer, attachments []*Attachment) error

type Attachment struct {
	Type        string
	Filename    string
	Filecontent *bytes.Buffer
	MimeType    string
	ContentId   *string
}

func SendMail(tenant, to, subject string, body *bytes.Buffer, attachments []*Attachment) error {
	//fmt.Printf("GRPC SERVER: %v\n", viper.GetString("services.mail-server"))
	//conn, err := grpc.Dial(viper.GetString("services.mail-server"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	//if err != nil {
	//	return err
	//}
	//defer conn.Close()
	//c := protobuf.NewSendMailServiceClient(conn)
	//
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//defer cancel()

	//filterInlineAttachments := func() ([]*Attachment, []*Attachment) {
	//	in, at := []*Attachment{}, []*Attachment{}
	//	for i := range attachments {
	//		if attachments[i].Type == "INLINE" {
	//			in = append(in, attachments[i])
	//		} else {
	//			at = append(at, attachments[i])
	//		}
	//	}
	//	return in, at
	//}

	//if body != nil {
	//	request.Body = body.Bytes()
	//}

	//if attachments != nil {
	return sendHtmlInlineAttachment(tenant, to, subject, body, attachments)
	//}

	//if fileName != nil && fileContent != nil {
	//	request.Content = fileContent.Bytes()
	//	request.Filename = fileName
	//}
	//
	//r, err := c.SendExcel(ctx, request)
	//log.Infof("Response from MAIL-SERVER: %v", r)
	//if r == nil {
	//	return errors.New("error Send Mail")
	//}
	//return err
}

func sendHtmlInlineAttachment(sender, recipient, subject string, htmlBody *bytes.Buffer, attachments []*Attachment) error {
	conn, err := grpc.Dial(viper.GetString("services.mail-server"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	c := protobuf.NewSendMailServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_attachments := []*protobuf.Attachement{}
	for i := range attachments {
		_attachments = append(_attachments, &protobuf.Attachement{
			MimeType:  attachments[i].MimeType,
			Filename:  attachments[i].Filename,
			Content:   attachments[i].Filecontent.Bytes(),
			ContentId: attachments[i].ContentId,
		})
	}
	request := &protobuf.SendMailWithInlineAttachmentsRequest{
		Sender:      sender,
		Recipient:   recipient,
		Subject:     subject,
		HtmlBody:    htmlBody.String(),
		Attachments: _attachments,
	}
	r, err := c.SendMailWithInlineAttachment(ctx, request)
	log.Infof("Response from MAIL-SERVER: %v", r)
	if r == nil {
		return errors.New("error Send Mail")
	}
	if r.Status != 200 {
		return errors.New(*r.Message)
	}
	return err
}

type RegisterService struct {
	protobuf.UnimplementedRegisterEegServiceServer
}

func (r *RegisterService) Register(ctx context.Context, eeg *protobuf.RegisterEegRequest) (*protobuf.RegisteredEegReply, error) {

	newEeg := factory.GetEegFromRegisterEeg(eeg)

	//fmt.Printf("Register EEG: %+v\n", newEeg)
	db, err := database.GetDBXConnection()
	if err != nil {
		log.Errorf("Database Error: %v", err)
		return &protobuf.RegisteredEegReply{Status: 500}, err
	}
	defer db.Close()

	err = database.UpdateEeg(db, eeg.RcNumber, &newEeg)
	if err != nil {
		log.Errorf("Could not create an EEG! %v", err.Error())
		return &protobuf.RegisteredEegReply{Status: 500},
			status.Errorf(codes.NotFound, "unknown service %v", err)
	}

	return &protobuf.RegisteredEegReply{Status: 201}, nil
}

func StartGRPCServer() {
	port := viper.GetInt("grpc-provider.port")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	defer func() {
		listener.Close()
		log.Info("gRPC Server stops")
	}()
	log.Infof("gRPC Server listen on %s", fmt.Sprintf(":%d", port))
	grpcServer := grpc.NewServer()
	protobuf.RegisterRegisterEegServiceServer(grpcServer, &RegisterService{})
	protobuf.RegisterAdminEegServiceServer(grpcServer, &AdminService{})
	grpcServer.Serve(listener)
}
