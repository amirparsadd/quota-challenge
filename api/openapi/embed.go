package openapi

import _ "embed"

// V1 is the gateway OpenAPI 3 document served at /openapi/v1.yaml and Scalar /docs.
//
//go:embed v1.yaml
var V1 []byte