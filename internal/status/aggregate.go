package status

import (
	"sort"
	"time"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/store"
)

const InspectionWindow = 14 * 24 * time.Hour

// ConcernStatus 是汇总中单个关注项相对于处置登记的状态。
const (
	ConcernOpen     = "open"
	ConcernResolved = "resolved"
)

type ConcernItem struct {
	Code         string     `json:"code"`
	Status       string     `json:"status"`
	ResolutionID string     `json:"resolutionId,omitempty"`
	ResolvedBy   string     `json:"resolvedBy,omitempty"`
	Resolution   string     `json:"resolution,omitempty"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
}

type HiveSummary struct {
	HiveID               string        `json:"hiveId"`
	State                string        `json:"state"`
	LastInspectionAt     *time.Time    `json:"lastInspectionAt"`
	LatestInspectionID   string        `json:"latestInspectionId,omitempty"`
	LatestConcerns       []ConcernItem `json:"latestConcerns"`
	OpenConcernCount     int           `json:"openConcernCount"`
	ResolvedConcernCount int           `json:"resolvedConcernCount"`
	QueenID              string        `json:"queenId"`
	QueenMarkColor       string        `json:"queenMarkColor"`
}

type Report struct {
	ApiaryID        string         `json:"apiaryId"`
	Name            string         `json:"name"`
	AsOf            time.Time      `json:"asOf"`
	HiveCount       int            `json:"hiveCount"`
	StateCounts     map[string]int `json:"stateCounts"`
	NeedsInspection []string       `json:"needsInspection"`
	// ConcernHives 仍指「最近一次巡检存在未处置关注项」的蜂箱，保持既有汇总口径。
	ConcernHives []string `json:"concernHives"`
	// ResolvedConcernHives 是最近一次巡检关注项已全部处置的蜂箱。
	ResolvedConcernHives []string      `json:"resolvedConcernHives"`
	OpenConcernCount     int           `json:"openConcernCount"`
	ResolvedConcernCount int           `json:"resolvedConcernCount"`
	LayingQueens         int           `json:"layingQueens"`
	Hives                []HiveSummary `json:"hives"`
}

// BuildReport 在单份快照上完成聚合，保证计数与逐蜂箱摘要来自同一视图。
func BuildReport(snapshot store.Snapshot, apiary domain.Apiary, asOf time.Time) Report {
	asOf = asOf.UTC()
	report := Report{
		ApiaryID:             apiary.ID,
		Name:                 apiary.Name,
		AsOf:                 asOf,
		StateCounts:          initialStateCounts(),
		NeedsInspection:      []string{},
		ConcernHives:         []string{},
		ResolvedConcernHives: []string{},
		Hives:                []HiveSummary{},
	}
	hives := snapshot.ApiaryHives(apiary.ID)
	report.HiveCount = len(hives)
	for _, hive := range hives {
		report.StateCounts[string(hive.State)]++
		summary := HiveSummary{
			HiveID:           hive.ID,
			State:            string(hive.State),
			LastInspectionAt: hive.LastInspectionAt,
			LatestConcerns:   []ConcernItem{},
		}
		if queen, ok := snapshot.LayingQueen(hive.ID); ok {
			summary.QueenID = queen.ID
			summary.QueenMarkColor = queen.MarkColor
			report.LayingQueens++
		}
		if latest, ok := snapshot.LatestInspection(hive.ID); ok {
			observedAt := latest.ObservedAt
			summary.LastInspectionAt = &observedAt
			summary.LatestInspectionID = latest.ID
			if len(latest.Concerns) > 0 {
				// 只按「最新巡检的编号」关联处置：旧巡检上的处置
				// 不会被算到新巡检头上，新关注项一律以 open 出现。
				resolutions := snapshot.ResolutionMap(hive.ID, latest.ID)
				for _, code := range latest.Concerns {
					item := ConcernItem{Code: code, Status: ConcernOpen}
					if resolved, ok := resolutions[code]; ok {
						item.Status = ConcernResolved
						item.ResolutionID = resolved.ID
						item.ResolvedBy = resolved.ResolvedBy
						item.Resolution = resolved.Resolution
						resolvedAt := resolved.ResolvedAt
						item.ResolvedAt = &resolvedAt
						summary.ResolvedConcernCount++
						report.ResolvedConcernCount++
					} else {
						summary.OpenConcernCount++
						report.OpenConcernCount++
					}
					summary.LatestConcerns = append(summary.LatestConcerns, item)
				}
				switch {
				case summary.OpenConcernCount > 0:
					report.ConcernHives = append(report.ConcernHives, hive.ID)
				case summary.ResolvedConcernCount > 0:
					report.ResolvedConcernHives = append(report.ResolvedConcernHives, hive.ID)
				}
			}
		}
		if NeedsInspection(hive, asOf) {
			report.NeedsInspection = append(report.NeedsInspection, hive.ID)
		}
		report.Hives = append(report.Hives, summary)
	}
	sort.Strings(report.NeedsInspection)
	sort.Strings(report.ConcernHives)
	sort.Strings(report.ResolvedConcernHives)
	return report
}

// NeedsInspection 判定蜂箱是否需要巡检：从未巡检，或距 asOf 超过巡检窗口。
func NeedsInspection(hive domain.Hive, asOf time.Time) bool {
	if hive.LastInspectionAt == nil {
		return true
	}
	return asOf.Sub(hive.LastInspectionAt.UTC()) > InspectionWindow
}

func initialStateCounts() map[string]int {
	return map[string]int{
		string(domain.HiveColonized):  0,
		string(domain.HiveProduction): 0,
		string(domain.HiveWintering):  0,
		string(domain.HiveRetired):    0,
	}
}
