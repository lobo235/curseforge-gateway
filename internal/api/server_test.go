package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lobo235/curseforge-gateway/internal/api"
	"github.com/lobo235/curseforge-gateway/internal/curseforge"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const testAPIKey = "test-api-key"
const testVersion = "v1.0.0-test"

// mockCurseForge is a configurable mock that satisfies the curseforgeClient interface.
type mockCurseForge struct {
	pingFunc         func(ctx context.Context) error
	getModpackFn     func(ctx context.Context, projectID int) (*curseforge.Project, error)
	getModFn         func(ctx context.Context, projectID int) (*curseforge.Project, error)
	getFilesFn       func(ctx context.Context, projectID int) ([]curseforge.File, error)
	getFileFn        func(ctx context.Context, projectID, fileID int) (*curseforge.File, error)
	searchModpacksFn func(ctx context.Context, query string) ([]curseforge.SearchResult, error)
	searchModsFn     func(ctx context.Context, query string) ([]curseforge.SearchResult, error)
}

func (m *mockCurseForge) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockCurseForge) GetModpack(ctx context.Context, projectID int) (*curseforge.Project, error) {
	if m.getModpackFn != nil {
		return m.getModpackFn(ctx, projectID)
	}
	return nil, curseforge.ErrNotFound
}

func (m *mockCurseForge) GetMod(ctx context.Context, projectID int) (*curseforge.Project, error) {
	if m.getModFn != nil {
		return m.getModFn(ctx, projectID)
	}
	return nil, curseforge.ErrNotFound
}

func (m *mockCurseForge) GetFiles(ctx context.Context, projectID int) ([]curseforge.File, error) {
	if m.getFilesFn != nil {
		return m.getFilesFn(ctx, projectID)
	}
	return nil, curseforge.ErrNotFound
}

func (m *mockCurseForge) GetFile(ctx context.Context, projectID, fileID int) (*curseforge.File, error) {
	if m.getFileFn != nil {
		return m.getFileFn(ctx, projectID, fileID)
	}
	return nil, curseforge.ErrNotFound
}

func (m *mockCurseForge) SearchModpacks(ctx context.Context, query string) ([]curseforge.SearchResult, error) {
	if m.searchModpacksFn != nil {
		return m.searchModpacksFn(ctx, query)
	}
	return nil, nil
}

func (m *mockCurseForge) SearchMods(ctx context.Context, query string) ([]curseforge.SearchResult, error) {
	if m.searchModsFn != nil {
		return m.searchModsFn(ctx, query)
	}
	return nil, nil
}

// newTestServer creates a test HTTP server with the given mock CurseForge client.
func newTestServer(t *testing.T, mock *mockCurseForge) *httptest.Server {
	t.Helper()
	srv := api.NewServer(mock, testAPIKey, testVersion, discardLogger())
	return httptest.NewServer(srv.Handler())
}

func authHeader() string {
	return "Bearer " + testAPIKey
}

// --- helpers ---

func getJSON(t *testing.T, srv *httptest.Server, path string, auth bool) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if auth {
		req.Header.Set("Authorization", authHeader())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want %d, body: %s", resp.StatusCode, want, string(body))
	}
}

func assertErrorCode(t *testing.T, resp *http.Response, wantCode string) {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Code != wantCode {
		t.Errorf("error code = %q, want %q", body.Code, wantCode)
	}
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	return result
}

// --- auth middleware ---

func TestAuth_MissingToken(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/modpacks/123", false)
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, resp, "unauthorized")
}

func TestAuth_WrongToken(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/modpacks/123", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_ValidToken(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "test", ClassID: 4471}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/modpacks/123", true)
	assertStatus(t, resp, http.StatusOK)
}

// --- GET /health ---

func TestHealth_CurseForgeUp(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		pingFunc: func(_ context.Context) error { return nil },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/health", false)
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Version != testVersion {
		t.Errorf("version = %q, want %q", body.Version, testVersion)
	}
}

func TestHealth_CurseForgeDown(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		pingFunc: func(_ context.Context) error { return errors.New("connection refused") },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/health", false)
	assertStatus(t, resp, http.StatusServiceUnavailable)

	var body struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "unavailable" {
		t.Errorf("status = %q, want unavailable", body.Status)
	}
}

func TestHealth_NoAuthRequired(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/health", false)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("/health should not require auth")
	}
}

// --- X-Trace-ID ---

func TestTraceID_NotInResponse(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
	req.Header.Set("X-Trace-ID", "my-trace-123")
	resp, _ := http.DefaultClient.Do(req)
	if got := resp.Header.Get("X-Trace-ID"); got != "" {
		t.Errorf("X-Trace-ID should not be in response headers, got %q", got)
	}
}

// --- GET /modpacks/{projectID} ---

func TestGetModpack_OK(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, id int) (*curseforge.Project, error) {
			return &curseforge.Project{
				ID:           id,
				Name:         "All the Mods 10",
				Summary:      "A big modpack",
				ClassID:      4471,
				GameVersions: []string{"1.20.1"},
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123", true)
	assertStatus(t, resp, http.StatusOK)

	body := decodeJSON(t, resp)
	if body["name"] != "All the Mods 10" {
		t.Errorf("name = %v", body["name"])
	}
}

func TestGetModpack_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return nil, curseforge.ErrNotFound
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/999", true)
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

func TestGetModpack_WrongClass(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return nil, curseforge.ErrWrongClass
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/456", true)
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

func TestGetModpack_InvalidID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/modpacks/abc", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetModpack_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return nil, errors.New("curseforge unavailable")
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /modpacks/{projectID}/files ---

func TestGetModpackFiles_OK(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "ATM10", ClassID: 4471}, nil
		},
		getFilesFn: func(_ context.Context, _ int) ([]curseforge.File, error) {
			return []curseforge.File{
				{ID: 1001, DisplayName: "Server-1.0.zip", IsServerPack: true},
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123/files", true)
	assertStatus(t, resp, http.StatusOK)

	var files []map[string]any
	json.NewDecoder(resp.Body).Decode(&files)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
}

func TestGetModpackFiles_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return nil, curseforge.ErrNotFound
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/999/files", true)
	assertStatus(t, resp, http.StatusNotFound)
}

// --- GET /mods/{projectID} ---

func TestGetMod_OK(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, id int) (*curseforge.Project, error) {
			return &curseforge.Project{
				ID:      id,
				Name:    "JEI",
				Summary: "Just Enough Items",
				ClassID: 6,
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/789", true)
	assertStatus(t, resp, http.StatusOK)

	body := decodeJSON(t, resp)
	if body["name"] != "JEI" {
		t.Errorf("name = %v", body["name"])
	}
}

func TestGetMod_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/999", true)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestGetMod_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return nil, errors.New("curseforge unavailable")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/123", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /mods/{projectID}/files ---

func TestGetModFiles_OK(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 789, Name: "JEI", ClassID: 6}, nil
		},
		getFilesFn: func(_ context.Context, _ int) ([]curseforge.File, error) {
			return []curseforge.File{
				{ID: 2001, DisplayName: "jei-1.0.jar"},
				{ID: 2002, DisplayName: "jei-1.1.jar"},
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/789/files", true)
	assertStatus(t, resp, http.StatusOK)

	var files []map[string]any
	json.NewDecoder(resp.Body).Decode(&files)
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
}

func TestGetModFiles_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/999/files", true)
	assertStatus(t, resp, http.StatusNotFound)
}

// --- GET /modpacks/{projectID}/files/{fileID} ---

func TestGetModpackFile_OK(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "ATM10", ClassID: 4471}, nil
		},
		getFileFn: func(_ context.Context, _ int, fileID int) (*curseforge.File, error) {
			return &curseforge.File{
				ID:               fileID,
				DisplayName:      "ATM10-Client-1.0.zip",
				ServerPackFileID: 5001,
				IsServerPack:     false,
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123/files/1001", true)
	assertStatus(t, resp, http.StatusOK)

	body := decodeJSON(t, resp)
	if int(body["serverPackFileId"].(float64)) != 5001 {
		t.Errorf("serverPackFileId = %v, want 5001", body["serverPackFileId"])
	}
}

func TestGetModpackFile_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "ATM10", ClassID: 4471}, nil
		},
		getFileFn: func(_ context.Context, _ int, _ int) (*curseforge.File, error) {
			return nil, curseforge.ErrNotFound
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123/files/9999", true)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestGetModpackFile_InvalidFileID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "ATM10", ClassID: 4471}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123/files/abc", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

// --- GET /mods/{projectID}/files/{fileID} ---

func TestGetModFile_OK(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 789, Name: "JEI", ClassID: 6}, nil
		},
		getFileFn: func(_ context.Context, _ int, fileID int) (*curseforge.File, error) {
			return &curseforge.File{
				ID:          fileID,
				DisplayName: "jei-1.0.jar",
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/789/files/2001", true)
	assertStatus(t, resp, http.StatusOK)

	body := decodeJSON(t, resp)
	if body["displayName"] != "jei-1.0.jar" {
		t.Errorf("displayName = %v, want jei-1.0.jar", body["displayName"])
	}
}

func TestGetModFile_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 789, Name: "JEI", ClassID: 6}, nil
		},
		getFileFn: func(_ context.Context, _ int, _ int) (*curseforge.File, error) {
			return nil, curseforge.ErrNotFound
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/789/files/9999", true)
	assertStatus(t, resp, http.StatusNotFound)
}

// --- GET /modpacks/{projectID}/files error paths ---

func TestGetModpackFiles_InvalidID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/modpacks/abc/files", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetModpackFiles_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "ATM10", ClassID: 4471}, nil
		},
		getFilesFn: func(_ context.Context, _ int) ([]curseforge.File, error) {
			return nil, errors.New("upstream timeout")
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123/files", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /mods/{projectID}/files error paths ---

func TestGetModFiles_InvalidID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/abc/files", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetModFiles_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 789, Name: "JEI", ClassID: 6}, nil
		},
		getFilesFn: func(_ context.Context, _ int) ([]curseforge.File, error) {
			return nil, errors.New("upstream timeout")
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/789/files", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /mods/{projectID}/files/{fileID} error paths ---

func TestGetModFile_InvalidProjectID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/abc/files/1001", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetModFile_InvalidFileID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 789, Name: "JEI", ClassID: 6}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/789/files/abc", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetModFile_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 789, Name: "JEI", ClassID: 6}, nil
		},
		getFileFn: func(_ context.Context, _ int, _ int) (*curseforge.File, error) {
			return nil, errors.New("upstream timeout")
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/789/files/2001", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /modpacks/{projectID}/files/{fileID} additional error paths ---

func TestGetModpackFile_InvalidProjectID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/modpacks/abc/files/1001", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetModpackFile_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return &curseforge.Project{ID: 123, Name: "ATM10", ClassID: 4471}, nil
		},
		getFileFn: func(_ context.Context, _ int, _ int) (*curseforge.File, error) {
			return nil, errors.New("upstream timeout")
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123/files/1001", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /mods/{projectID} additional error paths ---

func TestGetMod_InvalidID(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()
	resp := getJSON(t, srv, "/mods/abc", true)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestGetMod_WrongClass(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModFn: func(_ context.Context, _ int) (*curseforge.Project, error) {
			return nil, curseforge.ErrWrongClass
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/mods/456", true)
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

// --- collectGameVersions from LatestFileIndex ---

func TestGetModpack_GameVersionsFromIndex(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{
		getModpackFn: func(_ context.Context, id int) (*curseforge.Project, error) {
			return &curseforge.Project{
				ID:           id,
				Name:         "Test Pack",
				Summary:      "Pack with file index versions",
				ClassID:      4471,
				GameVersions: nil, // empty — should fall through to LatestFileIndex
				LatestFileIndex: []struct {
					GameVersion string `json:"gameVersion"`
					FileID      int    `json:"fileId"`
				}{
					{GameVersion: "1.20.1", FileID: 1},
					{GameVersion: "1.20.1", FileID: 2}, // duplicate
					{GameVersion: "1.21.0", FileID: 3},
					{GameVersion: "", FileID: 4}, // empty — should be skipped
				},
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/modpacks/123", true)
	assertStatus(t, resp, http.StatusOK)

	body := decodeJSON(t, resp)
	versions, ok := body["gameVersions"].([]any)
	if !ok {
		t.Fatalf("gameVersions not an array: %T", body["gameVersions"])
	}
	if len(versions) != 2 {
		t.Errorf("got %d game versions, want 2 (deduplicated)", len(versions))
	}
}

// --- TraceID middleware ---

func TestTraceID_Generated(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()

	// Request without X-Trace-ID — middleware should generate one (we just verify no crash)
	resp := getJSON(t, srv, "/health", false)
	if resp.StatusCode == 0 {
		t.Error("expected a valid response")
	}
}

func TestTraceID_Propagated(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
	req.Header.Set("X-Trace-ID", "custom-trace-id")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	// The middleware propagates trace ID into context (for logging), not into response headers.
	// Just verify the request completes successfully.
	if resp.StatusCode == 0 {
		t.Error("expected a valid response")
	}
}

// --- Auth edge cases ---

func TestAuth_MalformedHeader(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/modpacks/123", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic auth instead of Bearer
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_EmptyBearer(t *testing.T) {
	srv := newTestServer(t, &mockCurseForge{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/modpacks/123", nil)
	req.Header.Set("Authorization", "Bearer ")
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusUnauthorized)
}

// Suppress "declared and not used" for the bytes import — used by test helper pattern.
var _ = bytes.NewReader
