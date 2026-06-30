package plugin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	QueryType string `json:"queryType"`
	Component string `json:"component"`
	Channel   string `json:"channel"`
	Source    string `json:"source"`
	Key       string `json:"key,omitempty"`
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	switch qm.QueryType {
	case "events":
		return d.queryEvents(ctx, pCtx, qm, query)
	case "telemetry":
		return d.queryTelemetry(ctx, pCtx, qm, query)
	}
	return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("invalid query type: %s", qm.QueryType))
}

func (d *Datasource) queryEvents(ctx context.Context, _ backend.PluginContext, qm queryModel, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	queryArgs := []interface{}{
		qm.Component,
		qm.Channel,
		qm.Source,
		query.TimeRange.From,
		query.TimeRange.To,
	}

	eventSQL := `
		SELECT 
			e.time,
			d.severity,
			e.message,
			e.source,
			e.args::text AS arguments
		FROM eventDefs d
		JOIN events e ON e.eventDefId = d.id
		WHERE ($1 = '' OR d.component = $1)
		  AND ($2 = '' OR d.name = $2)
		  AND ($3 = '' OR e.source = $3)
		  AND e.time >= $4
		  AND e.time <= $5
		ORDER BY e.time ASC;`

	rows, err := d.db.QueryContext(ctx, eventSQL, queryArgs...)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("events query execution failed: %v", err.Error()))
	}
	defer rows.Close()

	frame := data.NewFrame("Events: " + qm.Channel)
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{}),
		data.NewField("severity", nil, []int64{}),
		data.NewField("message", nil, []string{}),
		data.NewField("source", nil, []string{}),
		data.NewField("args", nil, []string{}),
	)

	for rows.Next() {
		var t time.Time
		var severity int64
		var message, source, args string

		if err := rows.Scan(&t, &severity, &message, &source, &args); err != nil {
			return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("events row scan failure: %v", err.Error()))
		}

		frame.AppendRow(t, severity, message, source, args)
	}

	response.Frames = append(response.Frames, frame)
	return response
}

func (d *Datasource) queryTelemetry(ctx context.Context, _ backend.PluginContext, qm queryModel, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse
	if qm.Component == "" || qm.Channel == "" {
		return response
	}

	// Resolve telemetry def id
	var valueType string
	var defID int64
	// TODO: switch to a single query instead of two
	// TODO: also consider having valueType in telemetryDefs instead of telemetry
	defQuery := `
		SELECT d.id, t.valueType
		FROM telemetryDefs d
		JOIN telemetry t ON t.telemetryDefId = d.id
		WHERE d.component = $1 AND d.name = $2
		LIMIT 1;`
	err := d.db.QueryRowContext(ctx, defQuery, qm.Component, qm.Channel).Scan(&defID, &valueType)
	if err != nil {
		if err == sql.ErrNoRows {
			return backend.ErrDataResponse(backend.StatusNotFound, fmt.Sprintf("telemetry channel '%s.%s' not found", qm.Component, qm.Channel))
		}
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("telemetry metadata registry failure: %v", err.Error()))
	}

	// Resolve data column
	rawSQL, queryArgs := buildSQLQuery(valueType, defID, qm, query)

	// Execute the query
	rows, err := d.db.QueryContext(ctx, rawSQL, queryArgs...)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("telemetry query execution failed: %v", err.Error()))
	}
	defer rows.Close()

	return buildResponse(qm, rows, response)
}

func buildSQLQuery(valueType string, defID int64, qm queryModel, query backend.DataQuery) (string, []interface{}) {
	targetColumn := "floating"
	switch valueType {
	case "int", "uint":
		targetColumn = "integral"
	case "bool":
		targetColumn = "boolval"
	case "string", "enum":
		targetColumn = "string"
	case "float":
		targetColumn = "floating"
	}

	// Configure query
	var queryArgs []interface{}
	queryArgs = append(queryArgs, defID, qm.Source, query.TimeRange.From, query.TimeRange.To, qm.Key)

	// Set time grouping interval
	intervalStr := fmt.Sprintf("%d seconds", int(query.Interval.Seconds()))
	if query.Interval.Seconds() < 1 {
		intervalStr = "1 second"
	}
	queryArgs = append(queryArgs, intervalStr)

	rawSQL := fmt.Sprintf(`
		SELECT 
			time_bucket($6::interval, t.time) AS time_bucket,
			AVG(t.%s::double precision) AS value
		FROM telemetry t
		WHERE t.telemetryDefId = $1
		  AND ($2 = '' OR t.source = $2)
		  AND t.time >= $3 
		  AND t.time <= $4
		  AND ($5 = '' OR t.key = $5)
		GROUP BY time_bucket
		ORDER BY time_bucket ASC;
	`, targetColumn)

	return rawSQL, queryArgs
}

func buildResponse(qm queryModel, rows *sql.Rows, response backend.DataResponse) backend.DataResponse {
	// Create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame(qm.Channel)
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{}),
		data.NewField("values", nil, []*float64{}),
	)

	for rows.Next() {
		var t time.Time
		var v sql.NullFloat64
		if err := rows.Scan(&t, &v); err != nil {
			return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("telemetry row scan failure: %v", err.Error()))
		}

		var valPtr *float64
		if v.Valid {
			valPtr = &v.Float64
		}
		frame.AppendRow(t, valPtr)
	}

	response.Frames = append(response.Frames, frame)
	return response
}
