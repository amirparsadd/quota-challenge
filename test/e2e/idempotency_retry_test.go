//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Proves design fault row: after an apparent failure, client retries same Idempotency-Key → stored outcome.
func TestCreateRetrySameIdempotencyKeyReturnsSameService(t *testing.T) {
	key := uuid.NewString()
	body := map[string]any{
		"product": "vm",
		"items": []map[string]any{
			{"resource": "cpu", "amount": 2},
			{"resource": "ram", "amount": 4},
			{"resource": "disk", "amount": 40},
			{"resource": "public_ip", "amount": 1},
		},
	}
	b, _ := json.Marshal(body)

	post := func() (status int, idempotentReplay string, serviceID string) {
		req, err := http.NewRequest(http.MethodPost, gatewayURL()+"/v1/regions/ir-thr-1/services", bytes.NewReader(b))
		require.NoError(t, err)
		req.Header.Set("X-Customer-ID", "demo")
		req.Header.Set("Idempotency-Key", key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		replay := resp.Header.Get("Idempotent-Replayed")
		raw, _ := io.ReadAll(resp.Body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		svc, _ := out["service"].(map[string]any)
		id, _ := svc["id"].(string)
		return resp.StatusCode, replay, id
	}

	st1, _, id1 := post()
	require.Equal(t, http.StatusCreated, st1)
	require.NotEmpty(t, id1)

	st2, replay2, id2 := post()
	require.Equal(t, http.StatusCreated, st2)
	require.Equal(t, "true", replay2)
	require.Equal(t, id1, id2)
}