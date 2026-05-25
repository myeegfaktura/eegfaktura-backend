package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/eegfaktura/eegfaktura-backend/api/middleware"
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/gorilla/mux"
)

func InitProcessRouter(r *mux.Router, jwtWrapper middleware.JWTWrapperFunc) *mux.Router {
	s := r.PathPrefix("/process").Subrouter()

	s.HandleFunc("/history", jwtWrapper(fetchProcessHistory())).Methods("GET")

	return r
}

// fetchProcessHistory honours optional query params (start, end, protocol, ps)
// to filter base.processhistory. When the `ps` param is present the response
// is wrapped in prod-paritätischer Shape `{History: {next: ...}, data: ...}`
// (customer-web getHistories1 path); without `ps` the raw map is returned
// (getHistories path, backwards-compatible).
//
// Actual page-size limiting / cursor advancement is intentionally not
// implemented here — the wrapper carries an empty NextType. Cursor
// pagination is a separate follow-up.
func fetchProcessHistory() middleware.JWTHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, claims *middleware.PlatformClaims, tenant string) {
		q := r.URL.Query()
		startMs, _ := strconv.ParseInt(q.Get("start"), 10, 64)
		endMs, _ := strconv.ParseInt(q.Get("end"), 10, 64)

		var protocols []string
		if p := q.Get("protocol"); p != "" {
			for _, s := range strings.Split(p, ";") {
				if s = strings.TrimSpace(s); s != "" {
					protocols = append(protocols, s)
				}
			}
		}

		history, err := database.FetchEdaHistory(database.GetDBXConnection, tenant, startMs, endMs, protocols)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, hasPs := q["ps"]
		if hasPs {
			resp := struct {
				History struct {
					Next database.NextType `json:"next,omitempty"`
				}
				Data map[string]map[string][]model.EdaProcessHistory `json:"data"`
			}{Data: history}
			respondWithJSON(w, 200, resp)
			return
		}
		respondWithJSON(w, 200, history)
	}
}
