package gateway

import (
	"errors"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"quota-challenge/internal/capacity"
)

func HTTPFromGRPC(err error) (code int, slug, detail string, retryAfter int) {
	if err == nil {
		return http.StatusOK, "", "", 0
	}
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, "internal-error", err.Error(), 0
	}
	detail = st.Message()
	switch st.Code() {
	case codes.NotFound:
		return http.StatusNotFound, "not-found", detail, 0
	case codes.InvalidArgument:
		return http.StatusUnprocessableEntity, "validation-error", detail, 0
	case codes.FailedPrecondition:
		if strings.Contains(detail, "reservation-expired") {
			return http.StatusConflict, "reservation-expired", detail, 0
		}
		return http.StatusConflict, "invalid-state", detail, 0
	case codes.ResourceExhausted:
		if strings.Contains(strings.ToLower(detail), "quota") {
			return http.StatusConflict, "quota-exceeded", detail, 0
		}
		return http.StatusConflict, "capacity-exceeded", detail, 0
	case codes.Aborted:
		return http.StatusConflict, "idempotency-in-progress", detail, 5
	case codes.Unavailable:
		return http.StatusServiceUnavailable, "region-unavailable", detail, 30
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout, "outcome-unknown", "upstream deadline exceeded; retry with same Idempotency-Key", 0
	case codes.PermissionDenied:
		return http.StatusForbidden, "forbidden", detail, 0
	default:
		return http.StatusInternalServerError, "internal-error", detail, 0
	}
}

func WriteGRPCError(w http.ResponseWriter, err error) {
	code, slug, detail, retry := HTTPFromGRPC(err)
	if retry > 0 {
		WriteProblemWithRetry(w, code, slug, detail, retry)
		return
	}
	WriteProblem(w, code, slug, detail)
}

func MapDomain(err error) (int, string) {
	switch {
	case errors.Is(err, capacity.ErrValidation):
		return http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, capacity.ErrNotFound):
		return http.StatusNotFound, err.Error()
	default:
		return http.StatusInternalServerError, err.Error()
	}
}