package httpapi

import (
	"net/http"

	"example.com/apiary-colony-service/internal/disposition"
	"example.com/apiary-colony-service/internal/domain"
)

type recordDispositionRequest struct {
	Concern    string `json:"concern"`
	DisposedBy string `json:"disposedBy"`
	Note       string `json:"note"`
}

func (s *Server) handleRecordDisposition(w http.ResponseWriter, r *http.Request) {
	var request recordDispositionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}
	input := disposition.RecordInput{
		Concern:    request.Concern,
		DisposedBy: request.DisposedBy,
		Note:       request.Note,
	}
	record, err := s.deps.Disposition.Record(r.PathValue("hiveId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleListDispositions(w http.ResponseWriter, r *http.Request) {
	items, err := s.deps.Disposition.List(r.PathValue("hiveId"))
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []domain.Disposition{}
	}
	writeJSON(w, http.StatusOK, items)
}
