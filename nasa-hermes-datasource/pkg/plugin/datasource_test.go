package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

func TestSeverityLabel(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "DIAGNOSTIC"},
		{1, "ACTIVITY_LOW"},
		{2, "ACTIVITY_HIGH"},
		{3, "WARNING_LOW"},
		{4, "WARNING_HIGH"},
		{5, "COMMAND"},
		{6, "FATAL"},
		{99, "UNKNOWN(99)"},
		{-1, "UNKNOWN(-1)"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("severity_%d", tt.input), func(t *testing.T) {
			result := severityLabel(tt.input)
			if result != tt.expected {
				t.Errorf("severityLabel(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQueryDispatch(t *testing.T) {
	ds := Datasource{}

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: []byte(`{invalid`)},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Responses["A"].Status != backend.StatusBadRequest {
			t.Errorf("expected StatusBadRequest, got %v", resp.Responses["A"].Status)
		}
	})

	t.Run("returns error on unknown query type", func(t *testing.T) {
		qJSON, _ := json.Marshal(queryModel{QueryType: "unknown"})
		resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: qJSON},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Responses["A"].Status != backend.StatusBadRequest {
			t.Errorf("expected StatusBadRequest, got %v", resp.Responses["A"].Status)
		}
	})

	t.Run("returns empty response for telemetry with missing component", func(t *testing.T) {
		qJSON, _ := json.Marshal(queryModel{QueryType: "telemetry", Channel: "ch1"})
		resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: qJSON},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Responses["A"].Frames) != 0 {
			t.Errorf("expected no frames for missing component, got %d", len(resp.Responses["A"].Frames))
		}
	})

	t.Run("returns empty response for telemetry with missing channel", func(t *testing.T) {
		qJSON, _ := json.Marshal(queryModel{QueryType: "telemetry", Component: "comp1"})
		resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: qJSON},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Responses["A"].Frames) != 0 {
			t.Errorf("expected no frames for missing channel, got %d", len(resp.Responses["A"].Frames))
		}
	})
}

func TestQueryDataMultipleQueries(t *testing.T) {
	ds := Datasource{}

	q1, _ := json.Marshal(queryModel{QueryType: "unknown"})
	q2, _ := json.Marshal(queryModel{QueryType: "telemetry", Component: "comp"})

	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: q1},
			{RefID: "B", JSON: q2},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(resp.Responses))
	}
	if _, ok := resp.Responses["A"]; !ok {
		t.Error("missing response for RefID A")
	}
	if _, ok := resp.Responses["B"]; !ok {
		t.Error("missing response for RefID B")
	}
}

func TestQueryTimeOverrides(t *testing.T) {
	ds := Datasource{}

	// Telemetry with empty component returns early before DB, so time overrides are
	// parsed but don't cause errors — this verifies the parse path doesn't panic.
	overrideFrom := "2024-01-01T00:00:00Z"
	overrideJSON, _ := json.Marshal(queryModel{
		QueryType:        "telemetry",
		TimeOverrideFrom: overrideFrom,
		TimeOverrideTo:   "2024-12-31T23:59:59Z",
	})

	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: overrideJSON, TimeRange: backend.TimeRange{
				From: time.Now().Add(-1 * time.Hour),
				To:   time.Now(),
			}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty (no component), no error
	if resp.Responses["A"].Status != 0 {
		t.Errorf("expected no error status, got %v", resp.Responses["A"].Status)
	}
}

func TestBuildResponseIntType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "int", 42.0, nil, nil, nil).
		AddRow(now.Add(time.Second), "int", 100.0, nil, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	resultRows, _ := db.Query("SELECT")
	qm := queryModel{Component: "comp", Channel: "ch"}
	resp := buildResponse(qm, resultRows, backend.DataResponse{})

	if len(resp.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(resp.Frames))
	}
	frame := resp.Frames[0]
	if frame.Name != "ch" {
		t.Errorf("expected frame name 'ch', got %q", frame.Name)
	}
	if len(frame.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(frame.Fields))
	}
	if frame.Fields[1].Name != "comp.ch" {
		t.Errorf("expected value field 'comp.ch', got %q", frame.Fields[1].Name)
	}
	if frame.Fields[0].Len() != 2 {
		t.Errorf("expected 2 rows, got %d", frame.Fields[0].Len())
	}
	val := frame.Fields[1].At(0).(*float64)
	if *val != 42.0 {
		t.Errorf("expected 42.0, got %f", *val)
	}
}

func TestBuildResponseUintType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "uint", 255.0, nil, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	qm := queryModel{Component: "c", Channel: "ch"}
	resp := buildResponse(qm, resultRows, backend.DataResponse{})

	if len(resp.Frames) != 1 || resp.Frames[0].Fields[0].Len() != 1 {
		t.Fatal("expected 1 frame with 1 row")
	}
	val := resp.Frames[0].Fields[1].At(0).(*float64)
	if *val != 255.0 {
		t.Errorf("expected 255.0, got %f", *val)
	}
}

func TestBuildResponseFloatType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "float", nil, 3.14, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	resp := buildResponse(queryModel{Component: "c", Channel: "ch"}, resultRows, backend.DataResponse{})

	val := resp.Frames[0].Fields[1].At(0).(*float64)
	if *val != 3.14 {
		t.Errorf("expected 3.14, got %f", *val)
	}
}

func TestBuildResponseBoolType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "bool", nil, nil, 1.0, nil).
		AddRow(now.Add(time.Second), "bool", nil, nil, 0.0, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	resp := buildResponse(queryModel{Component: "c", Channel: "ch"}, resultRows, backend.DataResponse{})

	v1 := resp.Frames[0].Fields[1].At(0).(*bool)
	v2 := resp.Frames[0].Fields[1].At(1).(*bool)
	if *v1 != true {
		t.Error("expected first bool row to be true")
	}
	if *v2 != false {
		t.Error("expected second bool row to be false")
	}
}

func TestBuildResponseStringType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "string", nil, nil, nil, "hello")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	resp := buildResponse(queryModel{Component: "c", Channel: "ch"}, resultRows, backend.DataResponse{})

	val := resp.Frames[0].Fields[1].At(0).(*string)
	if *val != "hello" {
		t.Errorf("expected 'hello', got %q", *val)
	}
}

func TestBuildResponseEnumType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	// enum falls through to default (string) branch in buildResponse
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "enum", nil, nil, nil, "MY_ENUM_VAL")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	resp := buildResponse(queryModel{Component: "c", Channel: "ch"}, resultRows, backend.DataResponse{})

	val := resp.Frames[0].Fields[1].At(0).(*string)
	if *val != "MY_ENUM_VAL" {
		t.Errorf("expected 'MY_ENUM_VAL', got %q", *val)
	}
}

func TestBuildResponseNullValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Truncate(time.Second)
	// All value columns are NULL
	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "int", nil, nil, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	resp := buildResponse(queryModel{Component: "c", Channel: "ch"}, resultRows, backend.DataResponse{})

	val := resp.Frames[0].Fields[1].At(0)
	if val != (*float64)(nil) {
		t.Errorf("expected nil *float64 for null int, got %v", val)
	}
}

func TestBuildResponseEmptyRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"})

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resultRows, _ := db.Query("SELECT")
	resp := buildResponse(queryModel{Component: "c", Channel: "ch"}, resultRows, backend.DataResponse{})

	if len(resp.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(resp.Frames))
	}
	// Only the time field, no value field added
	if len(resp.Frames[0].Fields) != 1 {
		t.Errorf("expected 1 field (time only) for empty result, got %d", len(resp.Frames[0].Fields))
	}
}

func TestQueryEventsWithMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	ds := Datasource{db: db}
	now := time.Now().Truncate(time.Second)

	eventRows := sqlmock.NewRows([]string{"time", "component", "name", "severity", "message", "source", "arguments"}).
		AddRow(now, "comp1", "evt1", int64(3), "something happened", "src1", `{"key":"val"}`).
		AddRow(now.Add(time.Second), "comp1", "evt2", int64(6), "fatal error", "src1", `{}`)

	mock.ExpectQuery("SELECT").WillReturnRows(eventRows)

	qJSON, _ := json.Marshal(queryModel{QueryType: "events", Source: "src1"})
	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A", JSON: qJSON, TimeRange: backend.TimeRange{From: now.Add(-time.Hour), To: now.Add(time.Hour)}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dr := resp.Responses["A"]
	if dr.Status != 0 {
		t.Fatalf("expected success, got status %v: %s", dr.Status, dr.Error)
	}
	if len(dr.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(dr.Frames))
	}
	frame := dr.Frames[0]
	if frame.Name != "Events" {
		t.Errorf("expected frame name 'Events', got %q", frame.Name)
	}
	if frame.Fields[0].Len() != 2 {
		t.Errorf("expected 2 rows, got %d", frame.Fields[0].Len())
	}
	// severity labels
	if frame.Fields[3].At(0) != "WARNING_LOW" {
		t.Errorf("expected WARNING_LOW, got %v", frame.Fields[3].At(0))
	}
	if frame.Fields[3].At(1) != "FATAL" {
		t.Errorf("expected FATAL, got %v", frame.Fields[3].At(1))
	}
}

func TestQueryTelemetryWithMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	ds := Datasource{db: db}
	now := time.Now().Truncate(time.Second)

	telemetryRows := sqlmock.NewRows([]string{"time_bucket", "valueType", "val_int", "val_float", "val_bool", "val_str"}).
		AddRow(now, "float", nil, 1.5, nil, nil).
		AddRow(now.Add(time.Second), "float", nil, 2.5, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(telemetryRows)

	qJSON, _ := json.Marshal(queryModel{QueryType: "telemetry", Component: "comp1", Channel: "ch1", Source: "src1"})
	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{
				RefID:    "A",
				JSON:     qJSON,
				Interval: 10 * time.Second,
				TimeRange: backend.TimeRange{
					From: now.Add(-time.Hour),
					To:   now.Add(time.Hour),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dr := resp.Responses["A"]
	if dr.Status != 0 {
		t.Fatalf("expected success, got status %v: %s", dr.Status, dr.Error)
	}
	if len(dr.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(dr.Frames))
	}
	frame := dr.Frames[0]
	if frame.Name != "ch1" {
		t.Errorf("expected frame name 'ch1', got %q", frame.Name)
	}
	if frame.Fields[1].Name != "comp1.ch1" {
		t.Errorf("expected field name 'comp1.ch1', got %q", frame.Fields[1].Name)
	}
	if frame.Fields[0].Len() != 2 {
		t.Errorf("expected 2 rows, got %d", frame.Fields[0].Len())
	}
	v0 := frame.Fields[1].At(0).(*float64)
	v1 := frame.Fields[1].At(1).(*float64)
	if *v0 != 1.5 {
		t.Errorf("expected 1.5, got %f", *v0)
	}
	if *v1 != 2.5 {
		t.Errorf("expected 2.5, got %f", *v1)
	}
}

func TestCheckHealth(t *testing.T) {
	ds := Datasource{}

	t.Run("returns error when host is missing", func(t *testing.T) {
		jsonData, _ := json.Marshal(map[string]interface{}{
			"host":     "",
			"database": "hermes",
		})

		res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{
			PluginContext: backend.PluginContext{
				DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
					JSONData:                jsonData,
					DecryptedSecureJSONData: map[string]string{},
				},
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != backend.HealthStatusError {
			t.Errorf("expected HealthStatusError, got %v", res.Status)
		}
		if res.Message != "Host configuration parameter is missing" {
			t.Errorf("expected 'Host configuration parameter is missing', got '%s'", res.Message)
		}
	})

	t.Run("returns error when database is missing", func(t *testing.T) {
		jsonData, _ := json.Marshal(map[string]interface{}{
			"host":     "localhost:5432",
			"database": "",
		})

		res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{
			PluginContext: backend.PluginContext{
				DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
					JSONData:                jsonData,
					DecryptedSecureJSONData: map[string]string{},
				},
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != backend.HealthStatusError {
			t.Errorf("expected HealthStatusError, got %v", res.Status)
		}
		if res.Message != "Database configuration parameter is missing" {
			t.Errorf("expected 'Database configuration parameter is missing', got '%s'", res.Message)
		}
	})

	t.Run("returns error when db is nil", func(t *testing.T) {
		jsonData, _ := json.Marshal(map[string]interface{}{
			"host":     "localhost:5432",
			"database": "hermes",
		})

		res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{
			PluginContext: backend.PluginContext{
				DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
					JSONData:                jsonData,
					DecryptedSecureJSONData: map[string]string{},
				},
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != backend.HealthStatusError {
			t.Errorf("expected HealthStatusError, got %v", res.Status)
		}
		if res.Message != "Internal database connection is null" {
			t.Errorf("expected 'Internal database connection is null', got '%s'", res.Message)
		}
	})
}
