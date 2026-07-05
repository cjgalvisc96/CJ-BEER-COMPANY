package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

const (
	BatchStartedTopic   = "brewing.batch_started"
	BatchCompletedTopic = "brewing.batch_completed"
)

type BatchStarted struct {
	shared.BaseEvent
	BatchID string `json:"batch_id"`
	BeerID  string `json:"beer_id"`
	Units   int    `json:"units"`
}

func (BatchStarted) EventName() string { return BatchStartedTopic }

type BatchCompleted struct {
	shared.BaseEvent
	BatchID string `json:"batch_id"`
	BeerID  string `json:"beer_id"`
	Units   int    `json:"units"`
}

func (BatchCompleted) EventName() string { return BatchCompletedTopic }
