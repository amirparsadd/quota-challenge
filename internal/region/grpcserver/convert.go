package grpcserver

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"quota-challenge/internal/capacity"
	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func denyGateway(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	vals := md.Get("x-caller")
	for _, v := range vals {
		if strings.EqualFold(v, callerGateway) {
			return status.Error(codes.PermissionDenied, "admin denied for gateway")
		}
	}
	return nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, capacity.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, capacity.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, capacity.ErrValidation), errors.Is(err, capacity.ErrIdempotencyMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, capacity.ErrQuotaExceeded), errors.Is(err, capacity.ErrCapacityExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, capacity.ErrIdempotencyConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, capacity.ErrReservationExpired):
		return status.Error(codes.FailedPrecondition, "reservation-expired")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toProtoService(svc *capacity.Service) *regionquotav1.Service {
	items := make([]*regionquotav1.ResourceLine, len(svc.Items))
	for i, it := range svc.Items {
		items[i] = &regionquotav1.ResourceLine{Resource: it.Resource, Amount: it.Amount, Unit: it.Unit}
	}
	out := &regionquotav1.Service{
		Id: svc.ID.String(), CustomerId: svc.CustomerID, Product: svc.Product, State: string(svc.State),
		Items: items, CreatedAt: svc.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: svc.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if svc.ExpiresAt != nil {
		u := svc.ExpiresAt.Unix()
		out.ExpiresAtUnix = &u
	}
	return out
}