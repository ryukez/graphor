package graphor

import (
	"fmt"
	"strings"
)

type Facet struct {
	Edge string
}

type Boolean struct {
	Edge   string
	Filter string
}

type RelationSchema struct {
	Edge           string
	HasMany        bool
	Include        bool
	IncludeOptions string
	CountField     string
	Facets         map[string]Facet
	SchemaFunc     func() Schema
}

type Schema struct {
	Tag       int
	Fields    []string
	Booleans  map[string]Boolean
	Relations map[string]RelationSchema
}

func EmptySchema() Schema {
	return Schema{}
}

func (schema Schema) Build() string {
	edges := append(schema.Fields, "uid", "created_at", "updated_at", "deleted_at")

	for name, b := range schema.Booleans {
		filter := b.Filter

		if Auth().IsLogin() {
			filter = strings.Replace(filter, "#{login_uid}", Auth().GetLoginUid(), -1)
		} else {
			continue
		}

		if filter != "" {
			edges = append(edges, fmt.Sprintf("%s: %s @filter(%s) { uid }", name, b.Edge, filter))
		} else {
			edges = append(edges, fmt.Sprintf("%s: %s { uid }", name, b.Edge))
		}
	}

	for name, r := range schema.Relations {
		if r.CountField != "" {
			edges = append(edges, fmt.Sprintf("%s: count(%s) @filter(not has(deleted_at))", r.CountField, r.Edge))
		}

		if r.Include {
			edges = append(edges, fmt.Sprintf("%s: %s %s {\n%s\n}", name, r.Edge, r.IncludeOptions, r.SchemaFunc().Build()))
		}
	}

	return strings.Join(edges, "\n")
}

func (schema Schema) Decode(src interface{}) QueryData {
	hash := src.(map[string]interface{})

	for name := range schema.Booleans {
		_, ok := hash[name]
		hash[name] = ok
	}

	for name, r := range schema.Relations {
		if r.Include {
			if edges, ok := hash[name]; ok {
				children := edges.([]interface{})
				schema := r.SchemaFunc()
				if r.HasMany {
					res := []interface{}{}
					for _, child := range children {
						res = append(res, schema.Decode(child))
					}
					hash[name] = res
				} else {
					hash[name] = schema.Decode(children[0])
				}
			} else if r.HasMany {
				hash[name] = []interface{}{}
			}
		}
	}

	return hash
}
