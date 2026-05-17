package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/api/middleware"
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/model"
	mqttclient "github.com/eegfaktura/eegfaktura-backend/mqtt"
	"github.com/eegfaktura/eegfaktura-backend/parser"
	"github.com/eegfaktura/eegfaktura-backend/util"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v4"
)

func InitMeteringRouter(r *mux.Router, jwtWrapper middleware.JWTWrapperFunc) *mux.Router {
	s := r.PathPrefix("/meteringpoint").Subrouter()

	s.HandleFunc("/{pid}/update/{mid}", jwtWrapper(updateMeteringPoint())).Methods("PUT")
	s.HandleFunc("/{pid}/remove/{mid}", jwtWrapper(removeMeteringPoint())).Methods("DELETE")
	s.HandleFunc("/{pid}/archive/{mid}", jwtWrapper(archiveMeteringPoint())).Methods("PUT")
	s.HandleFunc("/{pid}/create", jwtWrapper(createMeteringPoint())).Methods("PUT")
	s.HandleFunc("/{pid}/register", jwtWrapper(registerMeteringPoint())).Methods("POST")
	s.HandleFunc("/{pid}/syncenergy", jwtWrapper(requestMeteringPointValues())).Methods("POST")
	s.HandleFunc("/{pid}/revokemeters", jwtWrapper(requestRevokeMeteringPoint())).Methods("POST")
	s.HandleFunc("/{pid}/updateid/{mid}", jwtWrapper(updateMeteringPointId())).Methods("PUT")
	s.HandleFunc("/v2/{pid}/update/{mid}", jwtWrapper(updateMeteringPointPartial())).Methods("PUT")
	s.HandleFunc("/{spid}/{dpid}/move/{mid}", jwtWrapper(moveMeteringPoint())).Methods("PUT")
	s.HandleFunc("/changepartitionfactor", jwtWrapper(requestChangePartitionFactor())).Methods("POST")
	s.HandleFunc("/{pid}/update/{mid}/partfact", jwtWrapper(updateMeteringPointPartFact())).Methods("PUT")

	return r
}

// updateMeteringPointPartFactRequest is the JSON body for the
// /update/{mid}/partfact route — a single integer.
type updateMeteringPointPartFactRequest struct {
	PartFact int `json:"partFact"`
}

func updateMeteringPointPartFact() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]
		meterId := vars["mid"]

		var req updateMeteringPointPartFactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.PartFact < 0 || req.PartFact > 100 {
			http.Error(w, "partFact must be in 0..100", http.StatusBadRequest)
			return
		}

		if err := database.UpdateMeteringPointPartFact(database.GetDBXConnection, tenant, claims.Username, participantId, meterId, req.PartFact); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]interface{}{
			"status":   "ok",
			"partFact": req.PartFact,
		})
	}
}

// changePartitionFactorRequestBody is the JSON body accepted by the
// /changepartitionfactor route — a flat list of per-meter partition
// factor change requests.
type changePartitionFactorRequestBody struct {
	MeteringPoints []*model.ChangePartitionFactorRequest `json:"meteringPoints"`
}

func requestChangePartitionFactor() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		var body changePartitionFactorRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body.MeteringPoints) == 0 {
			http.Error(w, "no metering points provided", http.StatusBadRequest)
			return
		}

		eeg, err := database.GetEeg(tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Only the EBMS-online path is implemented. The offline path
		// would write directly to a partition-factor table that the
		// public stand does not carry; tracked as a Phase-7 followup.
		if !eeg.Online {
			respondWithJSON(w, http.StatusNotImplemented, map[string]string{
				"status": "offline path not supported (no partition_fact schema in this stand)",
			})
			return
		}

		if err := mqttclient.ChangePartitionFactor(eeg, body.MeteringPoints); err != nil {
			log.WithField("tenant", tenant).WithError(err).Error("ChangePartitionFactor dispatch failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
	}
}

func moveMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		sourceParticipantId := vars["spid"]
		destParticipantId := vars["dpid"]
		meterId := vars["mid"]

		if sourceParticipantId == destParticipantId {
			http.Error(w, "source and destination participant ids are identical", http.StatusBadRequest)
			return
		}

		if err := database.MoveMeteringPoint(database.GetDBXConnection, tenant, claims.Username, sourceParticipantId, destParticipantId, meterId); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
	}
}

// updateMeteringPointPartialRequest is the JSON body accepted by the
// /v2/{pid}/update/{mid} route. It carries a single column name +
// value pair to apply to the row.
type updateMeteringPointPartialRequest struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func updateMeteringPointPartial() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]
		meterId := vars["mid"]

		var req updateMeteringPointPartialRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Path) == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}

		err := database.UpdateMeteringPointPartial(
			database.GetDBXConnection,
			tenant, claims.Username, participantId, meterId,
			map[string]interface{}{req.Path: req.Value},
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
	}
}

// updateMeteringPointIdRequest is the JSON body accepted by the
// /updateid/{mid} route. It contains the new metering-point ID that
// should replace the existing one identified in the URL path.
type updateMeteringPointIdRequest struct {
	NewId string `json:"newId"`
}

func updateMeteringPointId() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]
		meterId := vars["mid"]

		var req updateMeteringPointIdRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.NewId) == "" {
			http.Error(w, "newId is required", http.StatusBadRequest)
			return
		}

		err := database.UpdateMeteringPointPartial(
			database.GetDBXConnection,
			tenant, claims.Username, participantId, meterId,
			map[string]interface{}{"metering_point_id": req.NewId},
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]string{
			"status": "ok",
			"newId":  req.NewId,
		})
	}
}

// revokeMeteringPointRequest is the JSON body accepted by the
// /revokemeters route. `From` is the consent-end timestamp in epoch
// milliseconds. `Reason` is optional and propagated to the EBMS
// envelope (online path) and ignored otherwise.
type revokeMeteringPointRequest struct {
	MeteringPoints []struct {
		Meter string `json:"meter"`
	} `json:"meteringPoints"`
	From   int64  `json:"from"`
	Reason string `json:"reason"`
}

func requestRevokeMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]

		var req revokeMeteringPointRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(req.MeteringPoints) == 0 {
			http.Error(w, "no metering points selected", http.StatusBadRequest)
			return
		}

		eeg, err := database.GetEeg(tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := database.QueryParticipant(participantId); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fromDate := util.TruncateToStartOfDay(time.UnixMilli(req.From))
		var reason *string
		if s := strings.TrimSpace(req.Reason); s != "" {
			reason = &s
		}

		meterIds := make([]string, 0, len(req.MeteringPoints))
		for _, m := range req.MeteringPoints {
			meterIds = append(meterIds, m.Meter)
		}

		log.WithField("tenant", tenant).
			WithField("participant", participantId).
			Infof("revoke meters %v at %s", meterIds, fromDate.Format("2006-01-02"))

		// When the EEG is online, the revoke is routed via EBMS; otherwise
		// the DB row is updated directly.
		if eeg.Online {
			meters, err := database.FindActiveMeteringByIds(database.GetDBXConnection, tenant, meterIds)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			var errorList []string
			for _, m := range meters {
				if err := mqttclient.RevokeMeteringPoint(eeg, m, fromDate.UnixMilli(), reason); err != nil {
					log.WithField("tenant", tenant).Errorf("EBMS revoke meter %s: %v", m.MeteringPoint, err)
					errorList = append(errorList, m.MeteringPoint+": "+err.Error())
				}
			}
			if len(errorList) > 0 {
				respondWithJSON(w, http.StatusPartialContent,
					map[string]interface{}{"errors": errorList})
				return
			}
			respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
			return
		}

		// Offline path: just record the revoke in the database.
		for _, meterId := range meterIds {
			if err := database.MeteringPointRevoke(database.GetDBXConnection, tenant, meterId, fromDate); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
	}
}

func createMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]

		var m model.MeteringPoint
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		m.ModifiedAt = time.Now()
		m.RegisteredSince = time.Now()
		m.ModifiedBy = null.StringFrom(claims.Username)

		err = database.RegisterMeteringPoint(database.GetDBXConnection, tenant, participantId, &m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if m.Status == model.NEW {
			log.WithField("tenant", tenant).Infof("register Meter:  %v ", m)
			eeg, err := database.GetEeg(tenant)
			if err != nil {
				log.WithField("error", err).Error("Query EEG")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			participant, err := database.QueryParticipant(participantId)
			if err != nil {
				log.WithField("error", err).Error("Query Participant")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if eeg.Online {
				ebmsMessage := model.EbmsMessage{
					Sender:      strings.ToUpper(tenant),
					Receiver:    strings.ToUpper(eeg.GridOperator),
					MessageCode: model.EBMS_ONLINE_REG_INIT,
					EcId:        eeg.CommunityId,
					Meter:       &model.Meter{MeteringPoint: m.MeteringPoint, Direction: m.Direction},
				}

				log.WithField("tenant", tenant).Infof("Start Meteringpoint %s registration", m.MeteringPoint)
				if err = mqttclient.SendEbmsMessage(ebmsMessage); err != nil {
					respondWithError(w, http.StatusInternalServerError, err.Error())
					return
				}

				if err = parser.SendActivationMailFromTemplate(util.SendMail, tenant,
					"Aktivierung im Serviceportal", eeg, participant); err != nil {
					log.Errorf("Error Sending Mail: %+v", err.Error())
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
		}
		respondWithJSON(w, http.StatusCreated, m)
	}
}

func updateMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]
		meterId := vars["mid"]

		m := model.MeteringPoint{}
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			log.WithField("error", err).Error("Decode UpdateMessage Json")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		m.ModifiedAt = time.Now()
		m.ModifiedBy = null.StringFrom(claims.Username)
		err = database.UpdateMeteringPoint(database.GetDBXConnection, tenant, participantId, meterId, &m)
		if err != nil {
			log.WithField("error", err).Error("Update Meteringpoint")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, m)
	}
}

type registerMeterRequestType struct {
	MeteringPoint string              `json:"meteringPoint"`
	Direction     model.DirectionType `json:"direction"`
	From          int64               `json:"from"`
	To            int64               `json:"to"`
}

// registerMeteringPoint activates existing meter at the net operator
//
// Here the registration only perform an online EDA communication
func registerMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]

		request := registerMeterRequestType{}
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			log.WithField("error", err).Error("Decode Metering Request (Register) Json")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		eeg, err := database.GetEeg(tenant)
		if err != nil {
			log.WithField("error", err).Error("Query EEG")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		participant, err := database.QueryParticipant(participantId)
		if err != nil {
			log.WithField("error", err).Error("Query Participant")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check Meter available in Participant
		meterExistsInParticipant := false
		for _, p := range participant.MeteringPoint {
			if p.MeteringPoint == request.MeteringPoint {
				meterExistsInParticipant = true
				break
			}
		}

		log.WithField("tenant", tenant).Infof("register Meter:  %v ", request)

		if eeg.Online && meterExistsInParticipant {
			ebmsMessage := model.EbmsMessage{
				Sender:      strings.ToUpper(tenant),
				Receiver:    strings.ToUpper(eeg.GridOperator),
				MessageCode: model.EBMS_ONLINE_REG_INIT,
				EcId:        eeg.CommunityId,
				Meter:       &model.Meter{MeteringPoint: request.MeteringPoint, Direction: request.Direction},
			}

			log.WithField("tenant", tenant).Infof("Start Meteringpoint %s registration", request.MeteringPoint)
			if err = mqttclient.SendEbmsMessage(ebmsMessage); err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}

			if err = parser.SendActivationMailFromTemplate(util.SendMail, tenant,
				"Aktivierung im Serviceportal", eeg, participant); err != nil {
				log.Errorf("Error Sending Mail: %+v", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		respondWithJSON(w, http.StatusCreated, participant)
	}
}

func requestMeteringPointValues() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]

		request := struct {
			MeteringPoints []struct {
				Meter     string              `json:"meter"`
				Direction model.DirectionType `json:"direction"`
			} `json:"meteringPoints"`
			From int64 `json:"from"`
			To   int64 `json:"to"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			log.WithField("error", err).Error("Decode Metering Request (Sync) Message Json")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		eeg, err := database.GetEeg(tenant)
		if err != nil {
			log.WithField("error", err).Error("Query EEG")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		participant, err := database.QueryParticipant(participantId)
		if err != nil {
			log.WithField("error", err).Error("Query Participant")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check Meter available in Participant
		//meterExistsInParticipant := false
		//for _, p := range participant.MeteringPoint {
		//	if p.MeteringPoint == request.MeteringPoints.Meter {
		//		meterExistsInParticipant = true
		//		break
		//	}
		//}
		meterExistsInParticipant := true

		fromDate := util.TruncateToStartOfDay(time.UnixMilli(request.From)).UnixMilli()
		toDate := util.TruncateToEndOfDay(time.UnixMilli(request.To)).UnixMilli()

		log.WithField("tenant", tenant).Infof("request Metering values %v (%d - %d)", request, fromDate, toDate)
		if eeg.Online && meterExistsInParticipant {
			for _, m := range request.MeteringPoints {
				ebmsMessage := model.EbmsMessage{
					Sender: strings.ToUpper(tenant),
					//Sender: strings.ToUpper("SEPP.GAUG"),
					Receiver: strings.ToUpper(eeg.GridOperator),
					//Receiver:    strings.ToUpper("OBERMUELLER.PETER"),
					MessageCode: model.EBMS_ZP_SYNC,
					Meter:       &model.Meter{MeteringPoint: m.Meter, Direction: m.Direction},
					Timeline: &model.Timeline{
						From: fromDate,
						To:   toDate},
				}
				log.WithField("tenant", tenant).Infof("Start Meteringpoint (%v) value request", request.MeteringPoints)
				if err = mqttclient.SendEbmsMessage(ebmsMessage); err != nil {
					respondWithError(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
		}
		respondWithJSON(w, http.StatusCreated, participant)
	}
}

func removeMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		participantId := vars["pid"]
		meterId := vars["mid"]

		err := database.RemoveMeteringPoint(database.GetDBXConnection, tenant, participantId, meterId)
		if err != nil {
			log.WithField("error", err).Errorf("Remove Meteringpoint %s", meterId)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]interface{}{"meteringpoint": meterId})
	}
}

func archiveMeteringPoint() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		meterId := vars["mid"]

		_, err := database.MeteringPointsSetStatus(database.GetDBXConnection, tenant, model.ARCHIVED, []string{meterId})
		if err != nil {
			log.WithField("error", err).Errorf("Remove Meteringpoint %s", meterId)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]interface{}{"meteringpoint": meterId})
	}
}
