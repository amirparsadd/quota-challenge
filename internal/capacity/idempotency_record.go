package capacity

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyRecord is the persisted idempotency state for reserve/create.
type IdempotencyRecord struct {
	Status         string
	RequestHash    string
	LeaseExpires   *time.Time
	LeaseToken     *uuid.UUID
	ResponseStatus *int
	ResponseBody   []byte
}