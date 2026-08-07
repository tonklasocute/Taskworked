// Package docs embeds the OpenAPI spec so it's baked into the compiled
// binary and available regardless of deployment method — the Docker
// image's final stage only copies the binary, not the source tree (see
// backend/Dockerfile), so a plain os.ReadFile at request time wouldn't
// find it there.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
