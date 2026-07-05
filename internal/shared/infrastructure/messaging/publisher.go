package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// WatermillEventPublisher implements ports.EventPublisher by marshaling each
// domain event to JSON and publishing it on the topic named by EventName().
type WatermillEventPublisher struct {
	publisher message.Publisher
}

func NewWatermillEventPublisher(publisher message.Publisher) *WatermillEventPublisher {
	return &WatermillEventPublisher{publisher: publisher}
}

func (p *WatermillEventPublisher) Publish(_ context.Context, events ...domain.Event) error {
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", event.EventName(), err)
		}
		msg := message.NewMessage(watermill.NewUUID(), payload)
		if err := p.publisher.Publish(event.EventName(), msg); err != nil {
			return fmt.Errorf("publish event %s: %w", event.EventName(), err)
		}
	}
	return nil
}
