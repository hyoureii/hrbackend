//go:build ignore

package main

import (
	"encoding/json"
	"maps"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	conv "github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

const help string = `Usage:
convertOpenApi.go	<input1> <input2> ... <output>
`

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		panic(help)
	}

	inputs := args[:len(args)-1]
	output := args[len(args)-1]

	var merged openapi2.T

	for i, input := range inputs {
		data, err := os.ReadFile(input)
		if err != nil { panic(err) }

		var doc openapi2.T
		if err := json.Unmarshal(data, &doc); err != nil { panic(err) }
		if i == 0 {
			merged = doc
		} else {
			mergeSpec(&merged, &doc)
		}
	}

	doc, err := conv.ToV3(&merged)
	if err != nil { panic(err) }

	doc.Info = &openapi3.Info{
		Title: "HRConnect REST Gateway to gRPC",
		Version: "v1",
	}
	doc.AddServer(&openapi3.Server{ URL: "/api/v1", })
	authHeader := openapi3.Parameter{
		Name: "authorization",
		In: "header",
		Schema: openapi3.NewSchemaRef("string", &openapi3.Schema{
			Type: &openapi3.Types{"string"},
		}),
	}
	if doc.Components == nil {
		doc.Components = &openapi3.Components{}
	}
	if doc.Components.Parameters == nil {
		doc.Components.Parameters = make(map[string]*openapi3.ParameterRef)
	}

	doc.Components.Parameters["authHeader"] = &openapi3.ParameterRef{
		Value: &authHeader,
	}

	for _, pathItem := range doc.Paths.Map() {
		for _, operation := range pathItem.Operations() {
			operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{
				Ref: "#/components/parameters/authHeader",
			})
		}
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil { panic(err) }

	err = os.WriteFile(output, out, 0644)
	if err != nil { panic(err) }
}

func mergeSpec(target, source *openapi2.T) {
	maps.Copy(target.Paths, source.Paths)

	for k, v := range source.Definitions {
		if _, exists := target.Definitions[k]; !exists {
			target.Definitions[k] = v
		}
	}

	target.Tags = append(target.Tags, source.Tags...)
}
