package salary

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/turfaa/apotek-hris/pkg/httpx"
)

// postJournalWriteTimeout is how long a post may take to answer. It sends
// several entries to vmedis-proxy one by one, so it outlasts the server's
// default write timeout.
const postJournalWriteTimeout = 10 * time.Minute

// GetSnapshotJournal returns a snapshot's Vmedis journal entries: the ones it
// can post and the ones posted.
func (h *Handler) GetSnapshotJournal(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	journal, err := h.service.GetSnapshotJournal(r.Context(), id, journalPosterFrom(r))
	if err != nil {
		journalError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpx.Ok(w, journal)
}

// PostSnapshotJournal posts a snapshot to the Vmedis journal, or sends its
// failed entries again.
func (h *Handler) PostSnapshotJournal(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	var req PostSnapshotJournalRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(postJournalWriteTimeout))

	journal, err := h.service.PostSnapshotJournal(r.Context(), id, req, journalPosterFrom(r))
	if err != nil {
		journalError(w, err)
		return
	}

	httpx.Ok(w, journal)
}

// journalPosterFrom returns the dashboard user making the request: X-Email,
// and X-Name, which the dashboard sends percent-encoded since names may not be
// ASCII.
func journalPosterFrom(r *http.Request) JournalPoster {
	name := r.Header.Get("X-Name")
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}

	return JournalPoster{
		Email: strings.TrimSpace(r.Header.Get("X-Email")),
		Name:  strings.TrimSpace(name),
	}
}

type journalConflictResponse struct {
	Message         string `json:"message"`
	OtherSnapshotID *int64 `json:"otherSnapshotID,omitempty"`
}

func journalError(w http.ResponseWriter, err error) {
	var requestErr *JournalRequestError
	var conflictErr *JournalConflictError

	switch {
	case errors.As(err, &requestErr):
		httpx.Error(w, requestErr, http.StatusBadRequest)
	case errors.As(err, &conflictErr):
		httpx.Status(w, journalConflictResponse{Message: conflictErr.Message, OtherSnapshotID: conflictErr.OtherSnapshotID}, http.StatusConflict)
	case errors.Is(err, ErrJournalNotConfigured):
		httpx.Error(w, ErrJournalNotConfigured, http.StatusServiceUnavailable)
	case errors.Is(err, sql.ErrNoRows):
		httpx.Error(w, errors.New("snapshot not found"), http.StatusNotFound)
	default:
		httpx.Error(w, err, http.StatusInternalServerError)
	}
}
