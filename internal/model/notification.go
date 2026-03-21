package model

import (
	"time"

	"github.com/google/uuid"
)

type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

func (c Channel) IsValid() bool {
	switch c {
	case ChannelSMS, ChannelEmail, ChannelPush:
		return true
	}
	return false
}

type Priority int

const (
	PriorityHigh   Priority = 0
	PriorityNormal Priority = 1
	PriorityLow    Priority = 2
)

func (p Priority) IsValid() bool {
	return p >= PriorityHigh && p <= PriorityLow
}

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	}
	return "normal"
}

// QueueName returns the Redis key for a given priority and channel.
func QueueName(priority Priority, channel Channel) string {
	return "queue:" + priority.String() + ":" + string(channel)
}

// AllQueueNames returns all 9 queue names in priority order.
func AllQueueNames() []string {
	var names []string
	for _, p := range []Priority{PriorityHigh, PriorityNormal, PriorityLow} {
		for _, c := range []Channel{ChannelSMS, ChannelEmail, ChannelPush} {
			names = append(names, QueueName(p, c))
		}
	}
	return names
}

type Status string

const (
	StatusPending    Status = "pending"
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusSent       Status = "sent"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

type Notification struct {
	ID           uuid.UUID  `json:"id"`
	BatchID      *uuid.UUID `json:"batch_id,omitempty"`
	Channel      Channel    `json:"channel"`
	Recipient    string     `json:"recipient"`
	Content      string     `json:"content"`
	Priority     Priority   `json:"priority"`
	Status       Status     `json:"status"`
	AttemptCount int        `json:"attempt_count"`
	MaxAttempts  int        `json:"max_attempts"`
	LastError    *string    `json:"last_error,omitempty"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
