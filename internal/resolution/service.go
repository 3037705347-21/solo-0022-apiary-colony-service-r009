package resolution

import (
	"time"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/store"
)

// Service 处理协调人对最近一次巡检派生关注项的处置登记。
// 处置只追加，不改写巡检与其派生关注项；关联到具体巡检由仓储在写锁内完成。
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

type ResolveInput struct {
	Concerns   []string
	ResolvedBy string
	Resolution string
	ResolvedAt time.Time
}

// Outcome 是登记成功后返回给调用方的结果。
type Outcome struct {
	HiveID       string                     `json:"hiveId"`
	InspectionID string                     `json:"inspectionId"`
	Resolutions  []domain.ConcernResolution `json:"resolutions"`
}

// Resolve 针对蜂箱最近一次巡检登记一个或多个关注项的处置。
func (s *Service) Resolve(hiveID string, input ResolveInput) (Outcome, error) {
	if err := domain.ValidateIdentifier("hiveId", hiveID, 3, 40); err != nil {
		return Outcome{}, err
	}
	if len(input.Concerns) == 0 {
		return Outcome{}, domain.Invalid("concerns", "concerns 至少包含一个关注项")
	}
	resolvedAt := input.ResolvedAt
	if resolvedAt.IsZero() {
		resolvedAt = s.clock()
	}
	// 去重与字段校验在进入仓储前完成，避免同一批请求里夹带重复或非法项。
	seen := make(map[string]bool, len(input.Concerns))
	candidates := make([]domain.ConcernResolution, 0, len(input.Concerns))
	for _, raw := range input.Concerns {
		code := domain.TrimText(raw)
		if seen[code] {
			return Outcome{}, domain.Invalid("concerns", "关注项 %q 在同一批处置中重复出现", code)
		}
		seen[code] = true
		record, err := domain.NewConcernResolution(
			domain.TrimText(hiveID), "", code, input.ResolvedBy, input.Resolution, resolvedAt)
		if err != nil {
			return Outcome{}, err
		}
		candidates = append(candidates, record)
	}
	outcome, err := s.store.ResolveConcerns(candidates)
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{
		HiveID:       domain.TrimText(hiveID),
		InspectionID: outcome.InspectionID,
		Resolutions:  outcome.Resolutions,
	}, nil
}

func (s *Service) List(hiveID string) ([]domain.ConcernResolution, error) {
	return s.store.ListResolutions(hiveID)
}
