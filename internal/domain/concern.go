package domain

import "time"

// 关注项编码与巡检派生规则保持一一对应，顺序即汇总中的稳定顺序。
const (
	ConcernMiteLoad    = "mite_load"
	ConcernBroodGap    = "brood_gap"
	ConcernTemperRisk  = "temper_risk"
	ConcernLowStrength = "low_strength"
)

// ConcernOrder 是关注项派生与汇总展示的稳定顺序。
var ConcernOrder = []string{
	ConcernMiteLoad,
	ConcernBroodGap,
	ConcernTemperRisk,
	ConcernLowStrength,
}

// ValidConcern 判断编码是否为已知的巡检派生关注项。
func ValidConcern(code string) bool {
	switch code {
	case ConcernMiteLoad, ConcernBroodGap, ConcernTemperRisk, ConcernLowStrength:
		return true
	default:
		return false
	}
}

// ConcernResolution 是协调人对「某一次巡检派生出的某个关注项」登记的处置。
// 它只追加、不可改写：处置不回写巡检与其派生关注项，而是通过 InspectionID
// 与 ConcernCode 关联到具体来源。新巡检得到新的 InspectionID，旧处置因此
// 不会掩盖新巡检派生出的新关注项。
type ConcernResolution struct {
	ID           string    `json:"id"`
	HiveID       string    `json:"hiveId"`
	InspectionID string    `json:"inspectionId"`
	ConcernCode  string    `json:"concernCode"`
	ResolvedBy   string    `json:"resolvedBy"`
	Resolution   string    `json:"resolution"`
	ResolvedAt   time.Time `json:"resolvedAt"`
}

// NewConcernResolution 在不接触仓储的前提下固化一条处置登记。
func NewConcernResolution(
	hiveID, inspectionID, code, resolvedBy, resolution string,
	resolvedAt time.Time,
) (ConcernResolution, error) {
	if err := ValidateIdentifier("hiveId", hiveID, 3, 40); err != nil {
		return ConcernResolution{}, err
	}
	// InspectionID 不在此处校验：它由仓储在写锁内按「最近一次巡检」回填，
	// 调用方不能自行指定要处置哪一次巡检。
	if !ValidConcern(code) {
		return ConcernResolution{}, Invalid("concernCode",
			"concernCode 必须是 %s、%s、%s 或 %s",
			ConcernMiteLoad, ConcernBroodGap, ConcernTemperRisk, ConcernLowStrength)
	}
	if err := ValidateText("resolvedBy", resolvedBy, 1, MaxNoteRunes); err != nil {
		return ConcernResolution{}, err
	}
	if err := ValidateText("resolution", resolution, 1, MaxNoteRunes); err != nil {
		return ConcernResolution{}, err
	}
	if resolvedAt.IsZero() {
		return ConcernResolution{}, Invalid("resolvedAt", "resolvedAt 不能为空")
	}
	return ConcernResolution{
		HiveID:       TrimText(hiveID),
		InspectionID: inspectionID,
		ConcernCode:  code,
		ResolvedBy:   TrimText(resolvedBy),
		Resolution:   TrimText(resolution),
		ResolvedAt:   resolvedAt.UTC(),
	}, nil
}
