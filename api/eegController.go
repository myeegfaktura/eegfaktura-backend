package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	s.HandleFunc("/sync/participants/{oid}", jwtWrapper(syncParticipantsByOperatorEda())).Methods("POST")
	s.HandleFunc("/sync/meterpoint", jwtWrapper(syncMeterpointEda())).Methods("POST")
	s.HandleFunc("/import/masterdata", jwtWrapper(uploadMasterData())).Methods("POST")
	s.HandleFunc("/export/masterdata", jwtWrapper(exportMasterdata())).Methods("GET")
	s.HandleFunc("/notifications/{id}", jwtWrapper(notifications())).Methods("GET")
	s.HandleFunc("/gridoperators", jwtWrapper(gridOperators())).Methods("GET")
	s.HandleFunc("/user/get-user", jwtWrapper(getUser())).Methods("GET")

	return r
}

// getUser returns the list of {tenant, name} entries the caller can see
// for the tenant they currently selected. The JWT middleware has already
// asserted that the tenant header value is in the claim's tenants[] list,
// so we trust `tenant` as-is and read the EEG name from base.eeg.
//
// Prod (vfeeg-backend:v0.3.05) emits an array even though there is always
// one entry — keeps the wire shape forward-compatible with a future
// multi-tenant-per-call API. We match that shape so the frontend's
// RTK-Query type `{tenant: string, name: string}[]` works unchanged.
func getUser() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusOK, []map[string]string{
			{"tenant": tenant, "name": eeg.Name},
		})
	}
}

// gridOperators returns the AT grid-operator lookup table as `{id: name}`
// (e.g. `{"AT420001": "EHA Energie-Handels-Gesellschaft mbH & Co. KG", ...}`).
// Frontend consumes it for the EEG-creation grid-operator dropdown.
// Wire shape matches prod (vfeeg-backend v0.3.05).
func gridOperators() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		operators, err := database.GetGridOperators(database.GetDBXConnection)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusOK, operators)
	}
}

// exportMasterdata streams an .xlsx workbook with two sheets — EEG master
// data (sheet name = RcNumber) and the participant + meter list (sheet
// name "Mitglieder"). Wire shape matches prod (vfeeg-backend v0.3.05):
// Content-Type spreadsheetml.sheet, Content-Disposition + filename header
// carrying "<tenant>-EEG-Masterdata-<YYYYMMDD>". Frontend consumes via
// handleDownload in eeg.service.ts:exportMasterdata.
func exportMasterdata() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		participants, err := database.GetParticipant(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tariffMap, err := database.GetTariffNameMap(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		b, err := database.ExportMasterdataToExcel(participants, eeg, tariffMap)
		if err != nil {
			log.Errorf("Export masterdata: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("%s-EEG-Masterdata-%s", tenant, time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))
		w.Header().Set("filename", filename)

		if _, err := b.WriteTo(w); err != nil {
			log.Errorf("Write masterdata export: %v", err)
		}
	}
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
		from := time.Date(day.Year(), day.Month(), day.Day()-1, 0, 0, 0, 0, day.Location()).UnixMilli()
		to := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli()

		log.WithField("tenant", tenant).Info("Start Participant sync")
		if err = mqttclient.RequestingMeteringPointListForCommunity(eeg, from, to); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithStatus(w, http.StatusNoContent)
	}
}

// syncParticipantsByOperatorEda is the per-grid-operator variant of
// syncParticipantsEda — the {oid} path-param is forwarded as the EBMS
// receiver of the CR_PODLIST request, so the response only lists
// metering points of that one operator. Used by Customer-Web's
// "Stammdaten synchronisieren" action when an EEG has meters across
// multiple grid operators and the admin wants to query them one at
// a time. The Fork's `/sync/participants` (no oid) still works and
// uses RequestingMeteringPointListForCommunity for community-wide
// pull.
func syncParticipantsByOperatorEda() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		operatorId := mux.Vars(r)["oid"]

		eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
		if err != nil {
			log.WithField("error", err).Error("Query EEG")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		day := time.Now()
		from := time.Date(day.Year(), day.Month(), day.Day()-1, 0, 0, 0, 0, day.Location()).UnixMilli()
		to := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli()

		log.WithField("tenant", tenant).WithField("operator", operatorId).Info("Start Participant sync (per operator)")
		if err = mqttclient.RequestingMeteringPointList(eeg, operatorId, from, to); err != nil {
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
		from := time.Date(day.Year(), day.Month(), day.Day()-3, 0, 0, 0, 0, day.Location()).UnixMilli()
		to := time.Date(day.Year(), day.Month(), day.Day()-2, 0, 0, 0, 0, day.Location()).UnixMilli()

		log.WithField("tenant", tenant).Info("Start Metering sync")
		if err = mqttclient.RequestingEnergyData(eeg, &m, from, to); err != nil {
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
