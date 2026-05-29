package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestMapPqError_DuplicateMeteringpoint(t *testing.T) {
	in := &pq.Error{
		Code:       "23505",
		Constraint: "idx_unique_meteringpoint_active",
		Message:    "duplicate key value violates unique constraint",
	}
	payload, status := mapPqError(in)
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, 1201, payload.Code)
	assert.Equal(t, "duplicate_meteringpoint", payload.Error)
	assert.Contains(t, payload.Message, "duplicate key")
}

func TestMapPqError_GenericUniqueViolation(t *testing.T) {
	in := &pq.Error{Code: "23505", Constraint: "some_other_uq", Message: "x"}
	payload, status := mapPqError(in)
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, 1202, payload.Code)
	assert.Equal(t, "duplicate_value", payload.Error)
}

func TestMapPqError_ForeignKey(t *testing.T) {
	in := &pq.Error{Code: "23503", Message: "x"}
	payload, status := mapPqError(in)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, 1203, payload.Code)
}

func TestMapPqError_NotNull(t *testing.T) {
	in := &pq.Error{Code: "23502", Message: "x"}
	payload, status := mapPqError(in)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, 1204, payload.Code)
}

func TestMapPqError_UnknownPqCode(t *testing.T) {
	in := &pq.Error{Code: "99999", Message: "x"}
	payload, status := mapPqError(in)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, 1100, payload.Code)
}

func TestTryRespondPqError_NonPqError_ReturnsFalse(t *testing.T) {
	w := &noopResponseWriter{header: http.Header{}}
	handled := tryRespondPqError(w, errors.New("plain go error"))
	assert.False(t, handled)
}

type noopResponseWriter struct{ header http.Header }

func (n *noopResponseWriter) Header() http.Header        { return n.header }
func (n *noopResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (n *noopResponseWriter) WriteHeader(int)            {}
