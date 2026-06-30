package plugin

import (
	"context"
	"encoding/json"
	"testing"

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
		if res.Message != "Host is missing" {
			t.Errorf("expected 'Host is missing', got '%s'", res.Message)
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
		if res.Message != "Database is missing" {
			t.Errorf("expected 'Database is missing', got '%s'", res.Message)
		}
	})

	t.Run("returns ok when host and database are provided", func(t *testing.T) {
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
		if res.Status != backend.HealthStatusOk {
			t.Errorf("expected HealthStatusOk, got %v", res.Status)
		}
		if res.Message != "Data source is working" {
			t.Errorf("expected 'Data source is working', got '%s'", res.Message)
		}
	})
}
