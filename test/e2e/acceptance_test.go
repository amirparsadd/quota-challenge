//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateSucceedsWhenOtherRegionStopped(t *testing.T) {
	_ = exec.Command("docker", "stop", regionBContainer).Run()
	t.Cleanup(func() { ensureAllRegionsUp(t) })

	time.Sleep(2 * time.Second)

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

	availReq, _ := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/availability", nil)
	availReq.Header.Set("X-Customer-ID", "demo")
	avail, err := http.DefaultClient.Do(availReq)
	require.NoError(t, err)
	defer avail.Body.Close()
	require.Equal(t, http.StatusOK, avail.StatusCode)
	var out map[string]any
	require.NoError(t, json.NewDecoder(avail.Body).Decode(&out))
	require.Equal(t, true, out["partial"])
	unhealthy, _ := out["unavailable_regions"].([]any)
	require.Contains(t, unhealthy, "ir-thr-2")
}

func TestPerRegionDataIsolation(t *testing.T) {
	ensureAllRegionsUp(t)
	key1 := uuid.NewString()
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
	var idThr1 string
	for _, region := range []string{"ir-thr-1", "ir-thr-2"} {
		req, _ := http.NewRequest(http.MethodPost, gatewayURL()+"/v1/regions/"+region+"/services", bytes.NewReader(b))
		req.Header.Set("X-Customer-ID", "demo")
		req.Header.Set("Idempotency-Key", key1+"-"+region)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		loc := resp.Header.Get("Location")
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		if region == "ir-thr-1" {
			idThr1 = loc[strings.LastIndex(loc, "/")+1:]
		}
	}
	listReq, _ := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/regions/ir-thr-2/services", nil)
	listReq.Header.Set("X-Customer-ID", "demo")
	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	var listed map[string]any
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listed))
	services, _ := listed["services"].([]any)
	for _, raw := range services {
		svc, _ := raw.(map[string]any)
		require.NotEqual(t, idThr1, svc["id"])
	}
	getReq, _ := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/regions/ir-thr-2/services/"+idThr1, nil)
	getReq.Header.Set("X-Customer-ID", "demo")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	getResp.Body.Close()
	require.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestCreateValidationHTTP422(t *testing.T) {
	ensureAllRegionsUp(t)
	body := map[string]any{
		"product": "vm",
		"items": []map[string]any{
			{"resource": "cpu", "amount": 2},
			{"resource": "ram", "amount": 4},
			{"resource": "disk", "amount": 15},
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
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	require.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestAvailabilityProductQuery(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/regions/ir-thr-1/availability?product=vm", nil)
	require.NoError(t, err)
	req.Header.Set("X-Customer-ID", "demo")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	entries, _ := out["entries"].([]any)
	require.Len(t, entries, 4)
}

func TestConfirmAfterCancelReturns409(t *testing.T) {
	ensureAllRegionsUp(t)
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
	post, _ := http.NewRequest(http.MethodPost, gatewayURL()+"/v1/regions/ir-thr-1/services", bytes.NewReader(b))
	post.Header.Set("X-Customer-ID", "demo")
	post.Header.Set("Idempotency-Key", uuid.NewString())
	post.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(post)
	require.NoError(t, err)
	loc := resp.Header.Get("Location")
	bb, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(bb))
	id := loc[strings.LastIndex(loc, "/")+1:]

	del, _ := http.NewRequest(http.MethodDelete, gatewayURL()+"/v1/regions/ir-thr-1/services/"+id, nil)
	del.Header.Set("X-Customer-ID", "demo")
	delResp, err := http.DefaultClient.Do(del)
	require.NoError(t, err)
	delResp.Body.Close()
	require.Equal(t, http.StatusOK, delResp.StatusCode)

	patchBody := bytes.NewBufferString(`{"state":"active"}`)
	patch, _ := http.NewRequest(http.MethodPatch, gatewayURL()+"/v1/regions/ir-thr-1/services/"+id, patchBody)
	patch.Header.Set("X-Customer-ID", "demo")
	patch.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patch)
	require.NoError(t, err)
	defer patchResp.Body.Close()
	require.Equal(t, http.StatusConflict, patchResp.StatusCode)
}