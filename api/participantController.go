package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/eegfaktura/eegfaktura-backend/api/middleware"
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/model"
	mqttclient "github.com/eegfaktura/eegfaktura-backend/mqtt"
	"github.com/eegfaktura/eegfaktura-backend/parser"
	"github.com/eegfaktura/eegfaktura-backend/util"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func InitParticipantRouter(r *mux.Router, jwtWrapper middleware.JWTWrapperFunc) *mux.Router {
	s := r.PathPrefix("/participant").Subrouter()

	s.HandleFunc("", jwtWrapper(fetchParticipant())).Methods("GET")
	s.HandleFunc("", jwtWrapper(registerParticipant())).Methods("POST")
	s.HandleFunc("/{id}", jwtWrapper(updateParticipant())).Methods("PUT")
	s.HandleFunc("/{id}", jwtWrapper(archiveParticipant())).Methods("DELETE")
	s.HandleFunc("/{id}/confirm", jwtWrapper(confirmParticipant())).Methods("POST")

	return r
}

func fetchParticipant() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		participant, err := database.GetParticipant(database.GetDBXConnection, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, 200, participant)
	}
}

func updateParticipant() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		//vars := mux.Vars(r)
		//participantId := vars["id"]

		var t model.EegParticipant
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = database.UpdateParticipant(tenant, claims.Username, &t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusAccepted, t)
	}
}

func registerParticipant() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		var t model.EegParticipant
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = database.RegisterParticipant(database.GetDBXConnection, tenant, claims.Username, &t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondWithJSON(w, http.StatusCreated, t)
	}
}

func confirmParticipant() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {

		vars := mux.Vars(r)
		participantId := vars["id"]

		eeg, err := database.GetEeg(tenant)
		if err != nil {
			log.WithField("error", err).Error("Query EEG")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		participant, err := database.QueryParticipant(database.GetDBXConnection, participantId)
		if err != nil {
			log.WithField("error", err).Error("Query Participant")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//// Parse our multipart form, 10 << 20 specifies a maximum
		//// upload of 10 MB files.
		//err = r.ParseMultipartForm(10 << 20)
		//if err != nil {
		//	respondWithError(w, http.StatusBadRequest, err.Error())
		//	return
		//}
		//
		//formdata := r.MultipartForm // ok, no problem so far, read the Form data
		//
		////get the *fileheaders
		//files := formdata.File["docfiles"] // grab the filenames
		//
		//for i, _ := range files { // loop through the files one by one
		//	file, err := files[i].Open()
		//	defer file.Close()
		//	if err != nil {
		//		http.Error(w, err.Error(), http.StatusBadRequest)
		//		return
		//	}
		//
		//	outputPath := filepath.Join(viper.GetString("file-content.basedir"), tenant)
		//	err = os.MkdirAll(outputPath, os.ModePerm)
		//	if err != nil {
		//		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege %s", err.Error())
		//		return
		//	}
		//	out, err := os.Create(filepath.Join(outputPath, files[i].Filename))
		//
		//	defer out.Close()
		//	if err != nil {
		//		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege %s", err.Error())
		//		return
		//	}
		//
		//	_, err = io.Copy(out, file)
		//
		//	if err != nil {
		//		fmt.Fprintln(w, err)
		//		return
		//	}
		//
		//	log.Debug("Files uploaded successfully : ")
		//}
		if err = database.ConfirmParticipant(database.GetDBXConnection, tenant, claims.Username, participantId); err != nil {
			fmt.Fprint(w, err.Error())
			return
		}
		participant.Status = model.ACTIVE

		if eeg.Online {
			for _, m := range participant.MeteringPoint {
				ebmsMessage := model.EbmsMessage{
					Sender: strings.ToUpper(tenant),
					//Sender: strings.ToUpper("sepp.gaug"),
					Receiver: strings.ToUpper(eeg.GridOperator),
					//Receiver:    strings.ToUpper("obermueller.peter"),
					MessageCode: model.EBMS_ONLINE_REG_INIT,
					EcId:        eeg.CommunityId,
					Meter:       &model.Meter{MeteringPoint: m.MeteringPoint, Direction: m.Direction},
				}

				log.WithField("tenant", tenant).Infof("Start Meteringpoint %s registration", m.MeteringPoint)
				if err = mqttclient.SendEbmsMessage(ebmsMessage); err != nil {
					respondWithError(w, http.StatusInternalServerError, err.Error())
					return
				}
			}

			if err == nil && participant.Contact.Email.Valid {
				if err = parser.SendActivationMailFromTemplate(util.SendMail,
					tenant, "Aktivierung im Serviceportal", eeg, participant); err != nil {
					log.Errorf("Error Sending Mail: %+v", err.Error())
					//http.Error(w, err.Error(), http.StatusBadRequest)
					//return
				}
			}
		} else {
			meterIds := []string{}
			for _, m := range participant.MeteringPoint {
				meterIds = append(meterIds, m.MeteringPoint)
				m.Status = model.ACTIVE
			}
			_, err := database.MeteringPointsSetStatus(database.GetDBXConnection, tenant, model.ACTIVE, meterIds)
			if err != nil {
				log.Errorf("Error SET PARTICIPANT ACTIVE: %+v", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		respondWithJSON(w, http.StatusCreated, participant)
	}
}

func archiveParticipant() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		if err := database.ArchiveParticipant(database.GetDBXConnection, claims.Username, idStr); err != nil {
			respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{"id": 500, "error": err.Error()})
			return
		}
		respondWithJSON(w, http.StatusAccepted, map[string]interface{}{"status": "ok"})
	}
}
