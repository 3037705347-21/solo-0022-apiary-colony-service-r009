package inspection

import "example.com/apiary-colony-service/internal/domain"

// 关注项编码在 domain 中统一定义，这里保留导出别名以兼容既有引用。
const (
	ConcernMiteLoad    = domain.ConcernMiteLoad
	ConcernBroodGap    = domain.ConcernBroodGap
	ConcernTemperRisk  = domain.ConcernTemperRisk
	ConcernLowStrength = domain.ConcernLowStrength
)

const (
	MiteLoadThreshold  = 9
	LowStrengthCeiling = 2
)

// DeriveConcerns 按固定阈值从一次巡检派生出关注项，顺序稳定。
func DeriveConcerns(record domain.Inspection) []string {
	concerns := make([]string, 0, 4)
	if record.MiteCount >= MiteLoadThreshold {
		concerns = append(concerns, domain.ConcernMiteLoad)
	}
	if record.BroodPattern == domain.BroodNone {
		concerns = append(concerns, domain.ConcernBroodGap)
	}
	if record.Temper == domain.TemperAggressive {
		concerns = append(concerns, domain.ConcernTemperRisk)
	}
	if record.ColonyStrength <= LowStrengthCeiling {
		concerns = append(concerns, domain.ConcernLowStrength)
	}
	return concerns
}
