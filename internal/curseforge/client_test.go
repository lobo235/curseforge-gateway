package curseforge_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lobo235/curseforge-gateway/internal/curseforge"
)

// newTestClient returns a Client pointed at a test server and the test server itself.
func newTestClient(t *testing.T, handler http.Handler) (*curseforge.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client := curseforge.NewClientWithBaseURL("test-api-key", srv.URL)
	return client, srv
}

func TestPing_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/games", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Errorf("missing x-api-key header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPing_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/games", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetModpack_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"id":      123,
				"name":    "All the Mods 10",
				"summary": "A big modpack",
				"classId": 4471,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	p, err := client.GetModpack(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "All the Mods 10" {
		t.Errorf("name = %q, want All the Mods 10", p.Name)
	}
	if p.ClassID != 4471 {
		t.Errorf("classId = %d, want 4471", p.ClassID)
	}
}

func TestGetModpack_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/999", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := client.GetModpack(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetModpack_WrongClass(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/456", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"id":      456,
				"name":    "Some Mod",
				"summary": "A mod, not a modpack",
				"classId": 6,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := client.GetModpack(context.Background(), 456)
	if err == nil {
		t.Fatal("expected error for wrong classId")
	}
}

func TestGetMod_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/789", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"id":      789,
				"name":    "JEI",
				"summary": "Just Enough Items",
				"classId": 6,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	p, err := client.GetMod(context.Background(), 789)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "JEI" {
		t.Errorf("name = %q, want JEI", p.Name)
	}
}

func TestGetFiles_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123/files", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{
					"id":           1001,
					"displayName":  "ATM10-Server-1.0.zip",
					"fileName":     "ATM10-Server-1.0.zip",
					"gameVersions": []string{"1.20.1"},
					"isServerPack": true,
					"releaseType":  1,
				},
				{
					"id":           1002,
					"displayName":  "ATM10-Client-1.0.zip",
					"fileName":     "ATM10-Client-1.0.zip",
					"gameVersions": []string{"1.20.1"},
					"isServerPack": false,
					"releaseType":  1,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	files, err := client.GetFiles(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].DisplayName != "ATM10-Server-1.0.zip" {
		t.Errorf("files[0].DisplayName = %q", files[0].DisplayName)
	}
}

func TestGetFiles_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/999/files", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := client.GetFiles(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetFiles_UpstreamError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123/files", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := client.GetFiles(context.Background(), 123)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestCache_ProjectHit(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123", func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		resp := map[string]any{
			"data": map[string]any{
				"id":      123,
				"name":    "Cached Pack",
				"summary": "Should be cached",
				"classId": 4471,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	// First call — hits upstream
	_, err := client.GetModpack(context.Background(), 123)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — should be cached
	p, err := client.GetModpack(context.Background(), 123)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if p.Name != "Cached Pack" {
		t.Errorf("name = %q", p.Name)
	}
	if callCount != 1 {
		t.Errorf("upstream called %d times, want 1", callCount)
	}
}

func TestCache_FilesHit(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123/files", func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 1, "displayName": "file1.jar"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, _ = client.GetFiles(context.Background(), 123)
	_, _ = client.GetFiles(context.Background(), 123)
	if callCount != 1 {
		t.Errorf("upstream called %d times, want 1", callCount)
	}
}
