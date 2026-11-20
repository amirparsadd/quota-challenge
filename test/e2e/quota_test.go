//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDemoQuotaShowsProvisionedLimits(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/regions/ir-thr-1/quota", nil)
	require.NoError(t, err)
	req.Header.Set("X-Customer-ID", "demo")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	entries, ok := out["entries"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, entries)
	var foundCPU bool
	for _, raw := range entries {
		e, _ := raw.(map[string]any)
		if e["resource"] == "cpu" {
			foundCPU = true
			require.Equal(t, true, e["provisioned"])
			require.NotNil(t, e["limit"])
			require.NotNil(t, e["remaining"])
		}
	}
	require.True(t, foundCPU)
}