package disposition

import (
	"time"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/store"
)

// Service 登记协调人对最近一次巡检派生关注项的处置。
type Service struct {
	store *store.Memory
	clock func() time.Time
}

func New(memory *store.Memory, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: memory, clock: clock}
}

type RecordInput struct {
	Concern    string
	DisposedBy string
	Note       string
}

func (s *Service) Record(hiveID string, input RecordInput) (domain.Disposition, error) {
	record, err := domain.NewDisposition(hiveID, input.Concern, input.DisposedBy, input.Note, s.clock())
	if err != nil {
		return domain.Disposition{}, err
	}
	return s.store.AddDisposition(hiveID, record)
}

func (s *Service) List(hiveID string) ([]domain.Disposition, error) {
	return s.store.ListDispositions(hiveID)
}
