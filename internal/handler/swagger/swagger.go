package swagger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"log/slog"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"gopkg.in/yaml.v3"
)

func RegisterSwagger(router chi.Router, logger *slog.Logger) {
	router.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, err := os.ReadFile("docs/swagger.yaml")
		if err != nil {
			http.Error(w, "спецификация swagger не найдена", http.StatusInternalServerError)
			return
		}

		var v interface{}
		if err := yaml.Unmarshal(data, &v); err != nil {
			http.Error(w, "не удалось распарсить swagger yaml", http.StatusInternalServerError)
			return
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
			http.Error(w, "не удалось закодировать swagger в json", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	})

	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
}
