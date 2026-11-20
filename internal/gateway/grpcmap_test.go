package gateway

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHTTPFromGRPCValidation(t *testing.T) {
	err := status.Error(codes.InvalidArgument, "bad")
	code, _, _, _ := HTTPFromGRPC(err)
	require.Equal(t, http.StatusUnprocessableEntity, code)
}

func TestHTTPFromGRPCExhausted(t *testing.T) {
	err := status.Error(codes.ResourceExhausted, capacityQuotaMsg())
	code, _, _, _ := HTTPFromGRPC(err)
	require.Equal(t, http.StatusConflict, code)
}

func TestHTTPFromGRPCDeadlineIsOutcomeUnknown(t *testing.T) {
	err := status.Error(codes.DeadlineExceeded, "timeout")
	code, slug, detail, retry := HTTPFromGRPC(err)
	require.Equal(t, http.StatusGatewayTimeout, code)
	require.Equal(t, "outcome-unknown", slug)
	require.Contains(t, detail, "Idempotency-Key")
	require.Equal(t, 0, retry)
}

func capacityQuotaMsg() string { return "quota exceeded" }