package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (d *Datasource) handleGetComponents(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.QueryContext(r.Context(), "SELECT DISTINCT component FROM telemetryDefs ORDER BY component;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

func (d *Datasource) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	component := r.URL.Query().Get("component")
	rows, err := d.db.QueryContext(r.Context(), "SELECT name FROM telemetryDefs WHERE component = $1 ORDER BY name;", component)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

func (d *Datasource) handleGetSources(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.QueryContext(r.Context(), "SELECT DISTINCT source FROM telemetry WHERE time >= NOW() - INTERVAL '24 hours' LIMIT 100;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []string{}
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

func (d *Datasource) handleGetKeys(w http.ResponseWriter, r *http.Request) {
	component := r.URL.Query().Get("component")
	channel := r.URL.Query().Get("channel")

	query := `
		SELECT DISTINCT t.key 
		FROM telemetry t
		JOIN telemetryDefs d ON t.telemetryDefId = d.id
		WHERE d.component = $1 AND d.name = $2 AND t.key IS NOT NULL
		LIMIT 200;`

	rows, err := d.db.QueryContext(r.Context(), query, component, channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

func (d *Datasource) handleGetEventComponents(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.QueryContext(r.Context(), "SELECT DISTINCT component FROM eventDefs ORDER BY component;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

func (d *Datasource) handleGetEventNames(w http.ResponseWriter, r *http.Request) {
	component := r.URL.Query().Get("component")
	rows, err := d.db.QueryContext(r.Context(), "SELECT name FROM eventDefs WHERE component = $1 ORDER BY name;", component)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

func (d *Datasource) handleGetEventSources(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.QueryContext(r.Context(), "SELECT DISTINCT source FROM events WHERE time >= NOW() - INTERVAL '24 hours' LIMIT 100;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []string{}
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	writeJSONResponse(w, items)
}

var severityLabels = map[int64]string{
	0: "DIAGNOSTIC",
	1: "ACTIVITY_LOW",
	2: "ACTIVITY_HIGH",
	3: "WARNING_LOW",
	4: "WARNING_HIGH",
	5: "COMMAND",
	6: "FATAL",
}

type severityOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func (d *Datasource) handleGetEventSeverities(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.QueryContext(r.Context(), "SELECT DISTINCT severity FROM eventDefs ORDER BY severity;")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []severityOption
	for rows.Next() {
		var sev int64
		if err := rows.Scan(&sev); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		label, ok := severityLabels[sev]
		if !ok {
			label = fmt.Sprintf("UNKNOWN(%d)", sev)
		}
		items = append(items, severityOption{Value: fmt.Sprintf("%d", sev), Label: label})
	}
	writeJSONResponse(w, items)
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}
