//go:build e2e

package e2e

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScalarDocsAndOpenAPI(t *testing.T) {
	client := &http.Client{}
	for _, path := range []string{"/docs", "/openapi/v1.yaml"} {
		resp, err := client.Get(gatewayURL() + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, body)
		if path == "/docs" {
			require.Contains(t, string(body), "scalar")
		}
		if path == "/openapi/v1.yaml" {
			require.True(t, strings.Contains(string(body), "openapi:"))
		}
	}
}

func TestProblemsAndProductsEndpoints(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/v1/problems", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), `/v1/problems/validation-error`)

	req, err = http.NewRequest(http.MethodGet, gatewayURL()+"/v1/regions/ir-thr-1/products", nil)
	require.NoError(t, err)
	req.Header.Set("X-Customer-ID", "demo")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)
	require.Contains(t, string(body), `"products"`)
	require.Contains(t, string(body), `"vm"`)
	require.Contains(t, string(body), `"min_amount":`)
	require.NotRegexp(t, `"min_amount":"[0-9]+"`, string(body))
}