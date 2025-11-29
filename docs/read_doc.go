package docs

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ReadDoc returns the swagger document as a JSON string by reading the
// canonical docs/swagger.yaml and converting it to JSON.
func ReadDoc() string {
	data, err := os.ReadFile("docs/swagger.yaml")
	if err != nil {
		return "{}"
	}
	var v interface{}
	if err := yaml.Unmarshal(data, &v); err != nil {
		return "{}"
	}

	var convert func(interface{}) interface{}
	convert = func(in interface{}) interface{} {
		switch x := in.(type) {
		case map[string]interface{}:
			m := make(map[string]interface{}, len(x))
			for k, v := range x {
				m[k] = convert(v)
			}
			return m
		case map[interface{}]interface{}:
			m := make(map[string]interface{}, len(x))
			for k, v := range x {
				m[fmt.Sprint(k)] = convert(v)
			}
			return m
		case []interface{}:
			a := make([]interface{}, len(x))
			for i, v := range x {
				a[i] = convert(v)
			}
			return a
		default:
			return x
		}
	}

	out := convert(v)
	b, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(b)
}
