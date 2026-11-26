package handles

import (
	"net/http"
	"strconv"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	statsDays       = 7
)

// listMyAnalyses: GET /analyses. As analises de todas as fazendas do usuario logado, da mais nova para a
// mais antiga, em paginas (?limit=20&offset=0).
func (h *Handles) listMyAnalyses(w http.ResponseWriter, r *http.Request, p *Principal) {
	limit, err := intParam(r, "limit", defaultPageSize, 1, maxPageSize)

	if err != nil {
		h.fail(w, err)

		return
	}

	offset, err := intParam(r, "offset", 0, 0, 1_000_000)

	if err != nil {
		h.fail(w, err)

		return
	}

	items, total, err := h.db.Analyses.ListByUser(r.Context(), p.User.ID, limit, offset)

	if err != nil {
		h.fail(w, err)

		return
	}

	response := schema.AnalysisListResponse{Items: make([]schema.AnalysisResponse, 0, len(items)), Total: total}

	for i := range items {
		response.Items = append(response.Items, schema.UserAnalysisFromModel(&items[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

// stats: GET /stats. O resumo do usuario logado para a pagina de inicio. Os "dias" seguem o fuso de quem pede:
// ?offset= e quantos minutos o fuso esta a leste de UTC (Brasilia: -180).
func (h *Handles) stats(w http.ResponseWriter, r *http.Request, p *Principal) {
	offset, err := intParam(r, "offset", 0, -12*60, 14*60)

	if err != nil {
		h.fail(w, err)

		return
	}

	totals, err := h.db.Analyses.Totals(r.Context(), p.User.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	zone := time.FixedZone("user", offset*60)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	first := today.AddDate(0, 0, -(statsDays - 1))

	points, err := h.db.Analyses.Activity(r.Context(), p.User.ID, first)

	if err != nil {
		h.fail(w, err)

		return
	}

	days := make([]schema.DayStats, statsDays)
	index := make(map[string]int, statsDays)

	for i := range days {
		date := first.AddDate(0, 0, i).Format("2006-01-02")

		days[i].Date = date
		index[date] = i
	}

	for _, point := range points {
		if i, found := index[point.CreatedAt.In(zone).Format("2006-01-02")]; found {
			days[i].Analyses++
			days[i].Approved += point.Approved
			days[i].Rejected += point.Rejected
		}
	}

	writeJSON(w, http.StatusOK, schema.StatsResponse{
		Farms:    totals.Farms,
		Tanks:    totals.Tanks,
		Analyses: totals.Analyses,
		Approved: totals.Approved,
		Rejected: totals.Rejected,
		Days:     days,
	})
}

// intParam le um parametro numerico da URL, com valor padrao e limites. Fora dos limites e erro 422.
func intParam(r *http.Request, name string, fallback, min, max int) (int, error) {
	raw := r.URL.Query().Get(name)

	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)

	if err != nil || value < min || value > max {
		return 0, bodyError(name, "is out of the valid range")
	}

	return value, nil
}
