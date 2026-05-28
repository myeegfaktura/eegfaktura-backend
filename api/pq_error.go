package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lib/pq"
)

// pqErrorPayload mirrors the shape the frontend expects in
// `IErrorResponse` (service/error.text.ts:4-6): `{error:{code,error,message}}`.
// The `error` string is the second component of the i18n key the
// frontend builds (`E_<code>_<error>`) — keep it underscored-snake-case
// and stable, because the i18n keys live in locales/de/error.json and
// changes break the lookup.
type pqErrorPayload struct {
	Code    int    `json:"code"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// tryRespondPqError inspects err for a lib/pq driver error and emits the
// structured JSON the frontend can map to an i18n key. Returns true if
// the response has been written. Falls through (returns false) for
// non-pq errors so the caller can keep its existing http.Error path.
func tryRespondPqError(w http.ResponseWriter, err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}

	payload, status := mapPqError(pqErr)
	respondWithJSON(w, status, map[string]pqErrorPayload{"error": payload})
	return true
}

// mapPqError converts a libpq error into the structured payload + HTTP
// status. The code/error pairs become i18n keys `E_<code>_<error>` on
// the frontend (de/error.json). New entries on the backend side MUST
// have a matching i18n key, otherwise the frontend falls back to
// showing the raw key as text.
func mapPqError(pqErr *pq.Error) (pqErrorPayload, int) {
	switch pqErr.Code {
	case "23505":
		// unique_violation — narrow to known constraints so each gets a
		// dedicated user-facing message; fall back to a generic 1202.
		if strings.Contains(pqErr.Constraint, "idx_unique_meteringpoint_active") {
			return pqErrorPayload{
				Code:    1201,
				Error:   "duplicate_meteringpoint",
				Message: pqErr.Message,
			}, http.StatusConflict
		}
		return pqErrorPayload{
			Code:    1202,
			Error:   "duplicate_value",
			Message: pqErr.Message,
		}, http.StatusConflict
	case "23503":
		return pqErrorPayload{
			Code:    1203,
			Error:   "foreign_key_violation",
			Message: pqErr.Message,
		}, http.StatusBadRequest
	case "23502":
		return pqErrorPayload{
			Code:    1204,
			Error:   "missing_required_field",
			Message: pqErr.Message,
		}, http.StatusBadRequest
	}
	return pqErrorPayload{
		Code:    1100,
		Error:   "db_error",
		Message: pqErr.Message,
	}, http.StatusInternalServerError
}
