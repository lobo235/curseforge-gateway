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
	mux.HandleFunc("GET /v1/mods/123/files", func(w http.ResponseWriter, r *http.Request) {
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
			"pagination": map[string]any{
				"index":       0,
				"pageSize":    50,
				"resultCount": 2,
				"totalCount":  2,
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

func TestGetFiles_Pagination(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/100/files", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		index := r.URL.Query().Get("index")
		var resp map[string]any
		if index == "" || index == "0" {
			// First page: return pageSize items to signal more pages
			files := make([]map[string]any, 50)
			for i := range 50 {
				files[i] = map[string]any{
					"id":          i + 1,
					"displayName": "file-page1",
				}
			}
			resp = map[string]any{
				"data": files,
				"pagination": map[string]any{
					"index":       0,
					"pageSize":    50,
					"resultCount": 50,
					"totalCount":  75,
				},
			}
		} else {
			// Second page: return remaining items
			files := make([]map[string]any, 25)
			for i := range 25 {
				files[i] = map[string]any{
					"id":          i + 51,
					"displayName": "file-page2",
				}
			}
			resp = map[string]any{
				"data": files,
				"pagination": map[string]any{
					"index":       50,
					"pageSize":    50,
					"resultCount": 25,
					"totalCount":  75,
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	files, err := client.GetFiles(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 75 {
		t.Fatalf("got %d files, want 75", len(files))
	}
	if callCount != 2 {
		t.Errorf("upstream called %d times, want 2", callCount)
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

func TestGetFile_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123/files/1001", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"id":               1001,
				"displayName":      "ATM10-Client-1.0.zip",
				"fileName":         "ATM10-Client-1.0.zip",
				"gameVersions":     []string{"1.20.1"},
				"isServerPack":     false,
				"serverPackFileId": 2001,
				"releaseType":      1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	file, err := client.GetFile(context.Background(), 123, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ID != 1001 {
		t.Errorf("file.ID = %d, want 1001", file.ID)
	}
	if file.ServerPackFileID != 2001 {
		t.Errorf("file.ServerPackFileID = %d, want 2001", file.ServerPackFileID)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123/files/9999", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := client.GetFile(context.Background(), 123, 9999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestCache_FileHit(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/mods/123/files/1001", func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		resp := map[string]any{
			"data": map[string]any{
				"id":          1001,
				"displayName": "cached-file.zip",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	client, srv := newTestClient(t, mux)
	defer srv.Close()

	_, _ = client.GetFile(context.Background(), 123, 1001)
	f, _ := client.GetFile(context.Background(), 123, 1001)
	if f.DisplayName != "cached-file.zip" {
		t.Errorf("displayName = %q", f.DisplayName)
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
			"pagination": map[string]any{
				"index":       0,
				"pageSize":    50,
				"resultCount": 1,
				"totalCount":  1,
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
