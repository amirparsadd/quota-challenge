package capacity

import (
	"time"

	"github.com/google/uuid"
)

type ServiceState string

const (
	StateReserved  ServiceState = "reserved"
	StateActive    ServiceState = "active"
	StateCancelled ServiceState = "cancelled"
	StateExpired   ServiceState = "expired"
	StateReleased  ServiceState = "released"
	StateRejected  ServiceState = "rejected"
)

type ResourceLine struct {
	Resource string `json:"resource"`
	Amount   int64  `json:"amount"`
	Unit     string `json:"unit,omitempty"`
}

type Service struct {
	ID         uuid.UUID      `json:"id"`
	CustomerID string         `json:"customer_id"`
	Product    string         `json:"product"`
	State      ServiceState   `json:"state"`
	Items      []ResourceLine `json:"items"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

const TTLDefaultUnset = int64(-1)

type ReserveRequest struct {
	CustomerID     string
	IdempotencyKey string
	RequestHash    string
	Product        string
	Items          []ResourceLine
	TTLSeconds     int64 // TTLDefaultUnset => product default; 0 => active immediately
}

type Pool struct {
	Resource           string
	Total              int64
	BaselineAllocated  int64
	Allocated          int64
	Reserved           int64
}

type QuotaEntry struct {
	Resource    string
	Unit        string
	Limit       int64 // valid when Provisioned
	Allocated   int64
	Reserved    int64
	Provisioned bool
}

type AvailabilityEntry struct {
	Resource        string
	Unit            string
	PoolRemaining   int64
	QuotaRemaining  int64
}

type ProductResource struct {
	Resource    string
	Unit        string
	MinAmount   int64
	StepAmount  int64
}

type Product struct {
	Name              string
	DefaultTTLSeconds int
	MaxTTLSeconds     int
	Resources         []ProductResource
}

type ServiceEvent struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"`
	At        time.Time `json:"at"`
}

type IdempotencyStatus string

const (
	IdempotencyInProgress IdempotencyStatus = "in_progress"
	IdempotencyCompleted  IdempotencyStatus = "completed"
	IdempotencyRejected   IdempotencyStatus = "rejected"
)