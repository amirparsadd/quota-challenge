package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"quota-challenge/internal/capacity"
)

// QuotaRepository is the outbound port for region quota persistence (hexagonal driven adapter).
type QuotaRepository interface {
	AssertRegion(ctx context.Context) error

	ResourceByName(ctx context.Context, name string) (capacity.ResourceType, error)
	ProductTTLs(ctx context.Context, product string) (productID, defaultTTL, maxTTL int, err error)
	ListProducts(ctx context.Context) ([]capacity.Product, error)

	GetIdempotency(ctx context.Context, customerID, key string) (*capacity.IdempotencyRecord, error)
	ClaimIdempotency(ctx context.Context, customerID, key, hash string, lease time.Duration) (uuid.UUID, error)
	IdempotencyReplay(row *capacity.IdempotencyRecord) (*capacity.Service, int, error)
	IdempotencyHTTPReplay(row *capacity.IdempotencyRecord) (int, []byte, error)
	CompleteRejected(ctx context.Context, customerID, key string, token uuid.UUID, problem any) error
	SaveCompletedReserve(ctx context.Context, customerID, key string, token uuid.UUID, svc *capacity.Service) error

	RecordRejectedService(ctx context.Context, customerID string, productID int, reason string) error
	ReserveCreate(ctx context.Context, customerID string, productID int, productName string, lines []capacity.ReserveLine, ttl int64, token uuid.UUID, idemKey string) (*capacity.Service, error)
	ReclaimExpiredInScope(ctx context.Context, rtIDs []int) error

	GetService(ctx context.Context, customerID string, id uuid.UUID) (*capacity.Service, error)
	ListServices(ctx context.Context, customerID string) ([]capacity.Service, error)
	ConfirmService(ctx context.Context, customerID string, id uuid.UUID) (*capacity.Service, error)
	CancelService(ctx context.Context, customerID string, id uuid.UUID) (*capacity.Service, error)
	ReleaseService(ctx context.Context, id uuid.UUID) (*capacity.Service, error)
	ListServiceEvents(ctx context.Context, customerID string, serviceID uuid.UUID) ([]capacity.ServiceEvent, error)

	GetAvailability(ctx context.Context, customerID, product string) ([]capacity.AvailabilityEntry, error)
	GetQuota(ctx context.Context, customerID string) ([]capacity.QuotaEntry, error)

	SetPoolTotal(ctx context.Context, resource string, total int64) (*capacity.Pool, error)
	SetCustomerQuota(ctx context.Context, customerID, resource string, limit int64) error
	GetPool(ctx context.Context, resource string) (*capacity.Pool, error)

	ReapExpired(ctx context.Context, limit int) (int64, time.Duration, error)
	ResyncCounters(ctx context.Context, apply bool) (int64, error)
}