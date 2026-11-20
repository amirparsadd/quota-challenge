package gateway

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func TestRESTJSON_IntegersAreNumbersNotStrings(t *testing.T) {
	resp := &regionquotav1.ListProductsResponse{
		Products: []*regionquotav1.Product{{
			Name: "vm", DefaultTtlSeconds: 600, MaxTtlSeconds: 3600,
			Resources: []*regionquotav1.ProductResource{{
				Resource: "cpu", Unit: "cores", MinAmount: 1, StepAmount: 1,
			}},
		}},
	}
	b, err := json.Marshal(encodeListProductsResponse(resp))
	require.NoError(t, err)
	require.Contains(t, string(b), `"min_amount":1`)
	require.NotContains(t, string(b), `"min_amount":"1"`)

	svc := encodeServiceResponse(&regionquotav1.ServiceResponse{
		Service: &regionquotav1.Service{
			Id: "00000000-0000-0000-0000-000000000001", CustomerId: "demo", Product: "vm", State: "reserved",
			Items: []*regionquotav1.ResourceLine{{Resource: "ram", Amount: 4, Unit: "GiB"}},
			CreatedAt: "2020-01-01T00:00:00Z", UpdatedAt: "2020-01-01T00:00:00Z",
		},
	})
	b, err = json.Marshal(svc)
	require.NoError(t, err)
	require.Contains(t, string(b), `"amount":4`)
	require.NotContains(t, string(b), `"amount":"4"`)
}