package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hsuanlee/watch-compare/backend/internal/alerts"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

// SubscriberHeader carries the client's random subscriber id, which is also its OneSignal
// external_id. It is the only credential for managing that device's alerts, so it travels in a
// header rather than the URL (request logs record path and query).
const SubscriberHeader = "X-Subscriber-ID"

func subscriber(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(r.Header.Get(SubscriberHeader))
	if !alerts.ValidSubscriberID(id) {
		writeError(w, http.StatusUnauthorized, "missing or invalid "+SubscriberHeader)
		return "", false
	}
	return id, true
}

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	sub, ok := subscriber(w, r)
	if !ok {
		return
	}
	items, err := s.repo.AlertsFor(r.Context(), sub)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createAlertRequest struct {
	Query string `json:"query"` // same parameters as GET /api/v1/listings, incl. currency
	Name  string `json:"name"`  // optional; derived from the query when empty
}

func (s *Server) createAlert(w http.ResponseWriter, r *http.Request) {
	sub, ok := subscriber(w, r)
	if !ok {
		return
	}
	var req createAlertRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	conv, _, err := s.rates.get(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	query, err := alerts.Normalize(req.Query, conv)
	if errors.Is(err, alerts.ErrNoCriteria) {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	} else if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		name = alerts.DefaultName(query)
	}
	a, ok, err := s.repo.CreateAlert(r.Context(), sub, name, query, alerts.MaxPerSubscriber)
	if err != nil {
		s.fail(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusConflict, "alert limit reached")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) deleteAlert(w http.ResponseWriter, r *http.Request) {
	sub, ok := subscriber(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.repo.DeleteAlert(r.Context(), sub, id); errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "alert not found")
		return
	} else if err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
