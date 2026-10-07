package domain

import "time"

// Disposition 是协调人对一次巡检派生关注项的处置登记。
// 处置绑定到具体巡检而非蜂箱：新巡检派生的关注项不受旧处置影响，
// 处置本身也不改写原始巡检记录。
type Disposition struct {
	ID           string    `json:"id"`
	HiveID       string    `json:"hiveId"`
	InspectionID string    `json:"inspectionId"`
	Concern      string    `json:"concern"`
	DisposedBy   string    `json:"disposedBy"`
	Note         string    `json:"note"`
	DisposedAt   time.Time `json:"disposedAt"`
}

func NewDisposition(hiveID, concern, disposedBy, note string, disposedAt time.Time) (Disposition, error) {
	if err := ValidateIdentifier("hiveId", hiveID, 3, 40); err != nil {
		return Disposition{}, err
	}
	if err := ValidateText("concern", concern, 1, 40); err != nil {
		return Disposition{}, err
	}
	if err := ValidateText("disposedBy", disposedBy, 1, 40); err != nil {
		return Disposition{}, err
	}
	if err := ValidateText("note", note, 0, MaxNoteRunes); err != nil {
		return Disposition{}, err
	}
	return Disposition{
		HiveID:     TrimText(hiveID),
		Concern:    TrimText(concern),
		DisposedBy: TrimText(disposedBy),
		Note:       TrimText(note),
		DisposedAt: disposedAt.UTC(),
	}, nil
}
