package httpapi

import (
	"net/http"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/resolution"
)

type resolveConcernsRequest struct {
	Concerns   []string `json:"concerns"`
	ResolvedBy string   `json:"resolvedBy"`
	Resolution string   `json:"resolution"`
	ResolvedAt string   `json:"resolvedAt"`
}

func (s *Server) handleResolveConcerns(w http.ResponseWriter, r *http.Request) {
	var request resolveConcernsRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}
	resolvedAt, err := parseOptionalTime("resolvedAt", request.ResolvedAt)
	if err != nil {
		writeError(w, err)
		return
	}
	input := resolution.ResolveInput{
		Concerns:   request.Concerns,
		ResolvedBy: request.ResolvedBy,
		Resolution: request.Resolution,
	}
	if resolvedAt != nil {
		input.ResolvedAt = *resolvedAt
	}
	outcome, err := s.deps.Resolution.Resolve(r.PathValue("hiveId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, outcome)
}

func (s *Server) handleListResolutions(w http.ResponseWriter, r *http.Request) {
	items, err := s.deps.Resolution.List(r.PathValue("hiveId"))
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []domain.ConcernResolution{}
	}
	writeJSON(w, http.StatusOK, items)
}
