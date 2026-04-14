package static

import _ "embed"

//go:embed index.html
var ScalarHtml []byte

//go:embed openapi.json
var OpenApiSpec []byte

//go:embed scalar.js
var ScalarJS []byte
