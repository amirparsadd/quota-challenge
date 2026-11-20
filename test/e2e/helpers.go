//go:build e2e

package e2e

import (
	"net/http"
	"os/exec"
	"testing"
	"time"
)

const regionBContainer = "quota-challenge-region-quota-service-b-1"

func ensureAllRegionsUp(t *testing.T) {
	t.Helper()
	_ = exec.Command("docker", "start", regionBContainer).Run()
	deadline := time.Now().Add(45 * time.Second)
	client := &http.Client{Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/regions/ir-thr-2/availability", nil)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		req.Header.Set("X-Customer-ID", "demo")
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("region ir-thr-2 not healthy")
}