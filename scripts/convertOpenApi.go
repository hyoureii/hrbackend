//go:build ignore
package main

import (
	"encoding/json"
	"os"

	conv "github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi2"
)

func main() {
	input, output := os.Args[1], os.Args[2]

	data, err := os.ReadFile(input)
	if err != nil {
		panic(err)
	}

	var doc openapi2.T
	if err := json.Unmarshal(data, &doc); err != nil {
		panic(err)
	}

	doc3, err := conv.ToV3(&doc)
	if err != nil {
		panic(err)
	}

	out, err := json.MarshalIndent(doc3, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(output, out, 0644)
	if err != nil {
		panic(err)
	}
}
