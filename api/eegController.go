package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eegfaktura/eegfaktura-backend/api/middleware"
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/model"
	mqttclient "github.com/eegfaktura/eegfaktura-backend/mqtt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func InitEegRouter(r *mux.Router, jwtWrapper middleware.JWTWrapperFunc) *mux.Router {
	s := r.PathPrefix("/eeg").Subrouter()

	s.HandleFunc("", jwtWrapper(getEEG())).Methods("GET")
	s.HandleFunc("", jwtWrapper(updateEEG())).Methods("POST")
	s.HandleFunc("/tariff", jwtWrapper(getTariff())).Methods("GET")
	s.HandleFunc("/tariff", jwtWrapper(addTariff())).Methods("POST")
	s.HandleFunc("/tariff/{id}", jwtWrapper(fetchTariffHistory())).Methods("GET")
	s.HandleFunc("/tariff/{id}", jwtWrapper(archiveTariff())).Methods("DELETE")
	s.HandleFunc("/sync/participants", jwtWrapper(syncParticipantsEda())).Methods("POST")
	s.HandleFunc("/sync/meterpoint", jwtWrapper(syncMeterpointEda())).Methods("POST")
	s.HandleFunc("/import/masterdata", jwtWrapper(uploadMasterData())).Methods("POST")
	s.HandleFunc("/notifications/{id}", jwtWrapper(notifications())).Methods("GET")

	return r
}

func getEEG() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		log.Infof("Query EEG with TENANT: %s", tenant)
		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, 200, eeg)
	}
}

func updateEEG() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		var e map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&e)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err = database.UpdateEegPartial(database.GetDBXConnection, tenant, e); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, 200, eeg)
	}
}

func getTariff() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		tariff, err := database.GetTariff(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, 200, tariff)
	}
}

func addTariff() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		var t model.Tariff
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("ADD TARIF: %+v Tenant: %+v", t, tenant)

		if err = database.AddTariff(database.GetDBXConnection, tenant, &t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusCreated, t)
	}
}

func fetchTariffHistory() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		history, err := database.GetTariffHistory(database.GetDBXConnection, tenant, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusOK, history)
	}
}

func archiveTariff() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		if err := database.ArchiveTariff(database.GetDBXConnection, tenant, idStr); err != nil {
			if errors.Is(err, database.ErrTariffUtilized) {
				respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{"id": 900, "error": err.Error()})
				return
			}
			respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{"id": 500, "error": err.Error()})
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]interface{}{"status": "ok"})
	}
}

func syncParticipantsEda() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			log.WithField("error", err).Error("Query EEG")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		day := time.Now()
		ebmsMessage := model.EbmsMessage{
			Sender:      strings.ToUpper(tenant),
			Receiver:    strings.ToUpper(eeg.GridOperator),
			MessageCode: model.EBMS_ZP_LIST,
			Meter:       &model.Meter{MeteringPoint: eeg.CommunityId},
			Timeline: &model.Timeline{
				From: time.Date(day.Year(), day.Month(), day.Day()-1, 0, 0, 0, 0, day.Location()).UnixMilli(),
				To:   time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli()},
		}

		log.WithField("tenant", tenant).Info("Start Participant sync")
		if err = mqttclient.SendEbmsMessage(ebmsMessage); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithStatus(w, http.StatusNoContent)
	}
}

func syncMeterpointEda() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		var m model.MeteringPoint
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			log.Errorf("Body Parsing. %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			log.WithField("error", err).Error("Query EEG")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		day := time.Now()
		ebmsMessage := model.EbmsMessage{
			Sender:      strings.ToUpper(tenant),
			Receiver:    strings.ToUpper(eeg.GridOperator),
			MessageCode: model.EBMS_ZP_SYNC,
			Meter:       &model.Meter{MeteringPoint: m.MeteringPoint},
			Timeline: &model.Timeline{
				From: time.Date(day.Year(), day.Month(), day.Day()-3, 0, 0, 0, 0, day.Location()).UnixMilli(),
				To:   time.Date(day.Year(), day.Month(), day.Day()-2, 0, 0, 0, 0, day.Location()).UnixMilli()},
		}

		log.WithField("tenant", tenant).Info("Start Metering sync")
		if err = mqttclient.SendEbmsMessage(ebmsMessage); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithStatus(w, http.StatusNoContent)
	}
}

func uploadMasterData() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		var err error = r.ParseMultipartForm(10 << 20)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		sheet := r.FormValue("sheet")

		file, handler, err := r.FormFile("masterdatafile")
		if err != nil {
			glog.Error(err)
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		defer file.Close()
		glog.Infof("--- Upload File: %s, %s, %s\n", sheet, handler.Filename, tenant)

		if err = database.ImportMasterdataFromExcel(database.GetDBXConnection, file, handler.Filename, sheet, tenant); err != nil {
			glog.Error(err)
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else {
			glog.Infof("Import File %s successful", handler.Filename)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func notifications() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		isAdmin := func() bool {
			for _, a := range claims.AccessGroups {
				if a == "/EEG_ADMIN" {
					return true
				}
			}
			return false
		}
		//tenant = "RC100181"
		notifications, err := database.GetNotification(database.GetDBXConnection, tenant, id, isAdmin())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, 200, notifications)
	}
}
