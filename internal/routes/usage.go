package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/storage"
)

func MountUsageReal(r chi.Router, _ *config.Config, usage *storage.UsageStore) {
	r.Get("/critique", func(w http.ResponseWriter, req *http.Request) {
		records := usage.ByType("critique")
		daily := aggregateDaily(records)
		writeJSON(w, 200, map[string]any{"daily": daily, "total": len(records)})
	})

	r.Get("/ui-reverse", func(w http.ResponseWriter, req *http.Request) {
		records := usage.ByType("ui-reverse")
		daily := aggregateDaily(records)
		writeJSON(w, 200, map[string]any{"daily": daily, "total": len(records)})
	})

	r.Get("/all", func(w http.ResponseWriter, req *http.Request) {
		records := usage.All()
		byType := map[string]int{}
		totalCost := 0.0
		for _, r := range records {
			byType[r.Type]++
			totalCost += r.CostUSD
		}
		writeJSON(w, 200, map[string]any{
			"total":     len(records),
			"by_type":   byType,
			"total_cost": totalCost,
		})
	})
}

func aggregateDaily(records []storage.UsageRecord) []map[string]any {
	dm := map[string]map[string]any{}
	for _, r := range records {
		date := r.CreatedAt[:10] // YYYY-MM-DD
		if _, ok := dm[date]; !ok {
			dm[date] = map[string]any{"date": date, "requests": 0, "cost": 0.0}
		}
		dm[date]["requests"] = dm[date]["requests"].(int) + 1
		dm[date]["cost"] = dm[date]["cost"].(float64) + r.CostUSD
	}
	out := make([]map[string]any, 0, len(dm))
	for _, v := range dm {
		out = append(out, v)
	}
	return out
}
