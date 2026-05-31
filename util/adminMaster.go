package util

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/database"
	protobuf "github.com/eegfaktura/eegfaktura-backend/proto"
	log "github.com/sirupsen/logrus"
)

// AdminService implementiert proto.AdminEegService. Die Methode updateValue
// wird vom Scala-registration-backend per gRPC aufgerufen, sobald das
// admin-frontend POST /admin/master/update[/eeg|/participant] sendet.
//
// Recovery-Hintergrund: Service war in der prod-Variante (vfeeg-backend
// v0.3.05) vorhanden, ist beim Phase-5-Source-Backfill aber als „skip"
// klassifiziert worden — was sich erst beim ersten Settlement-Klick im
// Rewrite-admin-web als Backend-UNIMPLEMENTED gezeigt hat. Proto-Layout
// rekonstruiert aus dem deployten registration-backend-Jar (siehe
// proto/master.proto).
type AdminService struct {
	protobuf.UnimplementedAdminEegServiceServer
}

func (s *AdminService) UpdateValue(
	ctx context.Context,
	req *protobuf.UpdateEegRequest,
) (*protobuf.UpdateEegReply, error) {
	tenant := req.GetTenant()
	if tenant == "" {
		return reply(400, "tenant fehlt"), nil
	}
	switch req.GetUpdateClass() {
	case protobuf.UpdateEegRequest_EEG:
		return s.updateEeg(tenant, req.GetValue())
	case protobuf.UpdateEegRequest_PARTICIPANT:
		return s.updateParticipant(tenant, req.GetParticipantId(), req.GetValue())
	case protobuf.UpdateEegRequest_PROCESSSTATUS,
		protobuf.UpdateEegRequest_ACTIVESINCE,
		protobuf.UpdateEegRequest_INACTIVESINCE:
		return s.updateMeter(tenant, req.GetParticipantId(), req.GetMeteringPoint(), req.GetValue())
	default:
		return reply(400, fmt.Sprintf("unbekannte updateClass %v", req.GetUpdateClass())), nil
	}
}

func (s *AdminService) updateEeg(tenant string, value map[string]string) (*protobuf.UpdateEegReply, error) {
	fields := map[string]interface{}{}
	for k, v := range value {
		fields[k] = v
	}
	if err := database.UpdateEegPartial(database.GetDBXConnection, tenant, fields); err != nil {
		log.Errorf("AdminService.updateEeg: %v", err)
		return reply(500, err.Error()), nil
	}
	return reply(200, "ok"), nil
}

func (s *AdminService) updateParticipant(
	tenant, participantId string,
	value map[string]string,
) (*protobuf.UpdateEegReply, error) {
	if participantId == "" {
		return reply(400, "participantId fehlt"), nil
	}
	// Frontend kann mehrere Felder gleichzeitig senden (firstname, lastname,
	// businessRole). UpdateParticipantPartial nimmt nur ein Feld pro Call,
	// also iterieren wir. Fehler beim ersten Feld bricht ab.
	for key, val := range value {
		if err := database.UpdateParticipantPartial(database.GetDBXConnection, tenant, participantId, key, val); err != nil {
			log.Errorf("AdminService.updateParticipant key=%s: %v", key, err)
			return reply(500, err.Error()), nil
		}
	}
	return reply(200, "ok"), nil
}

func (s *AdminService) updateMeter(
	tenant, participantId, meterId string,
	value map[string]string,
) (*protobuf.UpdateEegReply, error) {
	if participantId == "" || meterId == "" {
		return reply(400, "participantId oder meteringPoint fehlt"), nil
	}
	// Date-Felder werden vom Frontend als Unix-Ms-String gesendet
	// (moment(date).unix()*1000). Wir konvertieren sie zu time.Time, damit
	// der Postgres-Driver sie korrekt als timestamp persistiert. Andere
	// Felder wie processState bleiben string.
	converted := map[string]interface{}{}
	for k, v := range value {
		switch k {
		case "activeSince", "inactiveSince":
			t, err := parseUnixMsString(v)
			if err != nil {
				log.Errorf("AdminService.updateMeter parse %s=%q: %v", k, v, err)
				return reply(400, fmt.Sprintf("ungueltiges Datum fuer %s: %s", k, v)), nil
			}
			converted[k] = t
		default:
			converted[k] = v
		}
	}
	if err := database.UpdateMeteringPointPartial(
		database.GetDBXConnection,
		tenant,
		"admin-grpc",
		participantId,
		meterId,
		converted,
	); err != nil {
		log.Errorf("AdminService.updateMeter: %v", err)
		return reply(500, err.Error()), nil
	}
	return reply(200, "ok"), nil
}

func parseUnixMsString(s string) (time.Time, error) {
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms).UTC(), nil
}

func reply(status int32, message string) *protobuf.UpdateEegReply {
	return &protobuf.UpdateEegReply{Status: status, Message: message}
}
