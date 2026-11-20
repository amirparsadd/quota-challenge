//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func gatewayURL() string {
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func TestCreateServiceCurlParity(t *testing.T) {
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
	req, err := http.NewRequest(http.MethodPost, gatewayURL()+"/v1/regions/ir-thr-1/services", bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("X-Customer-ID", "demo")
	req.Header.Set("Idempotency-Key", uuid.NewString())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Contains(t, []int{http.StatusCreated, http.StatusOK}, resp.StatusCode)
}

func TestMultiregionFanoutPartial(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/availability", nil)
	require.NoError(t, err)
	req.Header.Set("X-Customer-ID", "demo")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	_, hasPartial := out["partial"]
	require.True(t, hasPartial)
	regions, _ := out["regions"].([]any)
	require.GreaterOrEqual(t, len(regions), 1)
	fmt.Printf("fan-out regions=%d partial=%v\n", len(regions), out["partial"])
}