package curseforge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL = "https://api.curseforge.com/v1"

	// ClassID constants for CurseForge project types.
	classIDModpacks = 4471
	classIDMods     = 6

	// Cache TTLs.
	projectCacheTTL = 30 * time.Minute
	fileCacheTTL    = 5 * time.Minute
)

// Project represents a CurseForge project (modpack or mod).
type Project struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Summary         string   `json:"summary"`
	ClassID         int      `json:"classId"`
	GameVersions    []string `json:"gameVersions,omitempty"`
	LatestFileIndex []struct {
		GameVersion string `json:"gameVersion"`
		FileID      int    `json:"fileId"`
	} `json:"latestFilesIndexes,omitempty"`
}

// File represents a CurseForge file (server pack or mod file).
type File struct {
	ID           int      `json:"id"`
	DisplayName  string   `json:"displayName"`
	FileName     string   `json:"fileName"`
	GameVersions []string `json:"gameVersions"`
	IsServerPack bool     `json:"isServerPack"`
	DownloadURL  string   `json:"downloadUrl"`
	FileLength   int64    `json:"fileLength"`
	ReleaseType  int      `json:"releaseType"`
}

// cacheEntry holds a cached value with an expiry time.
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// Client wraps the CurseForge REST API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	cache      sync.Map
}

// NewClient creates a new CurseForge API client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewClientWithBaseURL creates a CurseForge API client with a custom base URL (for testing).
func NewClientWithBaseURL(apiKey, base string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: base + "/v1",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// cacheGet returns the cached value and true if found and not expired.
func (c *Client) cacheGet(key string) (any, bool) {
	val, ok := c.cache.Load(key)
	if !ok {
		return nil, false
	}
	entry := val.(cacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.cache.Delete(key)
		return nil, false
	}
	return entry.value, true
}

// cacheSet stores a value in the cache with the given TTL.
func (c *Client) cacheSet(key string, value any, ttl time.Duration) {
	c.cache.Store(key, cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	})
}

// Ping verifies connectivity to the CurseForge API.
func (c *Client) Ping(ctx context.Context) error {
	// Use the games endpoint as a lightweight health check.
	req, err := c.newRequest(ctx, http.MethodGet, "/games")
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("curseforge ping failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("curseforge ping returned status %d", resp.StatusCode)
	}
	return nil
}

// GetProject fetches a CurseForge project by ID and validates its classId.
// Returns ErrNotFound if the project doesn't exist or has the wrong classId.
func (c *Client) GetProject(ctx context.Context, projectID int, expectedClassID int) (*Project, error) {
	cacheKey := fmt.Sprintf("project:%d", projectID)
	if cached, ok := c.cacheGet(cacheKey); ok {
		p := cached.(*Project)
		if p.ClassID != expectedClassID {
			return nil, ErrWrongClass
		}
		return p, nil
	}

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/mods/%d", projectID))
	if err != nil {
		return nil, fmt.Errorf("creating project request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("curseforge returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope struct {
		Data Project `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decoding project response: %w", err)
	}

	c.cacheSet(cacheKey, &envelope.Data, projectCacheTTL)

	if envelope.Data.ClassID != expectedClassID {
		return nil, ErrWrongClass
	}
	return &envelope.Data, nil
}

// GetModpack fetches a modpack project (classId=4471).
func (c *Client) GetModpack(ctx context.Context, projectID int) (*Project, error) {
	return c.GetProject(ctx, projectID, classIDModpacks)
}

// GetMod fetches a mod project (classId=6).
func (c *Client) GetMod(ctx context.Context, projectID int) (*Project, error) {
	return c.GetProject(ctx, projectID, classIDMods)
}

// GetFiles fetches the file list for a CurseForge project.
func (c *Client) GetFiles(ctx context.Context, projectID int) ([]File, error) {
	cacheKey := fmt.Sprintf("files:%d", projectID)
	if cached, ok := c.cacheGet(cacheKey); ok {
		return cached.([]File), nil
	}

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/mods/%d/files", projectID))
	if err != nil {
		return nil, fmt.Errorf("creating files request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("curseforge returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope struct {
		Data []File `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decoding files response: %w", err)
	}

	c.cacheSet(cacheKey, envelope.Data, fileCacheTTL)
	return envelope.Data, nil
}

// ClassIDModpacks returns the classId for modpacks.
func ClassIDModpacks() int { return classIDModpacks }

// ClassIDMods returns the classId for mods.
func ClassIDMods() int { return classIDMods }
