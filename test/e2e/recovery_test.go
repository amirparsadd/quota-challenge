//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegionRestartStillAcceptsCreates(t *testing.T) {
	ctr := "quota-challenge-region-quota-service-1"
	require.NoError(t, exec.Command("docker", "restart", ctr).Run())
	time.Sleep(4 * time.Second)

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
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}