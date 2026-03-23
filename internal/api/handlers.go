package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/lobo235/curseforge-gateway/internal/curseforge"
)

// modpackResponse is the JSON response for GET /modpacks/{projectID}.
type modpackResponse struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Summary      string   `json:"summary"`
	GameVersions []string `json:"gameVersions"`
}

// modResponse is the JSON response for GET /mods/{projectID}.
type modResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// fileResponse is the JSON response for individual files in a file list.
type fileResponse struct {
	ID               int      `json:"id"`
	DisplayName      string   `json:"displayName"`
	FileName         string   `json:"fileName"`
	GameVersions     []string `json:"gameVersions"`
	IsServerPack     bool     `json:"isServerPack"`
	ServerPackFileID int      `json:"serverPackFileId"`
	DownloadURL      string   `json:"downloadUrl"`
	FileLength       int64    `json:"fileLength"`
	ReleaseType      int      `json:"releaseType"`
}

// getModpackHandler handles GET /modpacks/{projectID}.
func (s *Server) getModpackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.Atoi(r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "projectID must be an integer")
			return
		}

		project, err := s.curseforge.GetModpack(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "get modpack", projectID, err)
			return
		}

		writeJSON(w, http.StatusOK, modpackResponse{
			ID:           project.ID,
			Name:         project.Name,
			Summary:      project.Summary,
			GameVersions: collectGameVersions(project),
		})
	}
}

// getModpackFilesHandler handles GET /modpacks/{projectID}/files.
func (s *Server) getModpackFilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.Atoi(r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "projectID must be an integer")
			return
		}

		// Validate the project exists and is a modpack before listing files.
		_, err = s.curseforge.GetModpack(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "validate modpack", projectID, err)
			return
		}

		files, err := s.curseforge.GetFiles(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "get modpack files", projectID, err)
			return
		}

		writeJSON(w, http.StatusOK, toFileResponses(files))
	}
}

// getModHandler handles GET /mods/{projectID}.
func (s *Server) getModHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.Atoi(r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "projectID must be an integer")
			return
		}

		project, err := s.curseforge.GetMod(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "get mod", projectID, err)
			return
		}

		writeJSON(w, http.StatusOK, modResponse{
			ID:      project.ID,
			Name:    project.Name,
			Summary: project.Summary,
		})
	}
}

// getModFilesHandler handles GET /mods/{projectID}/files.
func (s *Server) getModFilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.Atoi(r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "projectID must be an integer")
			return
		}

		// Validate the project exists and is a mod before listing files.
		_, err = s.curseforge.GetMod(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "validate mod", projectID, err)
			return
		}

		files, err := s.curseforge.GetFiles(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "get mod files", projectID, err)
			return
		}

		writeJSON(w, http.StatusOK, toFileResponses(files))
	}
}

// getModpackFileHandler handles GET /modpacks/{projectID}/files/{fileID}.
func (s *Server) getModpackFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.Atoi(r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "projectID must be an integer")
			return
		}

		fileID, err := strconv.Atoi(r.PathValue("fileID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "fileID must be an integer")
			return
		}

		// Validate the project exists and is a modpack.
		_, err = s.curseforge.GetModpack(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "validate modpack", projectID, err)
			return
		}

		file, err := s.curseforge.GetFile(r.Context(), projectID, fileID)
		if err != nil {
			s.handleUpstreamError(w, "get modpack file", projectID, err)
			return
		}

		writeJSON(w, http.StatusOK, toFileResponse(file))
	}
}

// getModFileHandler handles GET /mods/{projectID}/files/{fileID}.
func (s *Server) getModFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.Atoi(r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "projectID must be an integer")
			return
		}

		fileID, err := strconv.Atoi(r.PathValue("fileID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "fileID must be an integer")
			return
		}

		// Validate the project exists and is a mod.
		_, err = s.curseforge.GetMod(r.Context(), projectID)
		if err != nil {
			s.handleUpstreamError(w, "validate mod", projectID, err)
			return
		}

		file, err := s.curseforge.GetFile(r.Context(), projectID, fileID)
		if err != nil {
			s.handleUpstreamError(w, "get mod file", projectID, err)
			return
		}

		writeJSON(w, http.StatusOK, toFileResponse(file))
	}
}

// handleUpstreamError maps curseforge client errors to HTTP responses.
func (s *Server) handleUpstreamError(w http.ResponseWriter, op string, projectID int, err error) {
	if errors.Is(err, curseforge.ErrNotFound) || errors.Is(err, curseforge.ErrWrongClass) {
		writeError(w, http.StatusNotFound, "not_found", "project not found")
		return
	}
	s.log.Error("upstream error", "operation", op, "project_id", projectID, "error", err)
	writeError(w, http.StatusBadGateway, "upstream_error", "failed to fetch data from CurseForge")
}

// collectGameVersions deduplicates game versions from the project's latest file index.
func collectGameVersions(p *curseforge.Project) []string {
	if len(p.GameVersions) > 0 {
		return p.GameVersions
	}
	seen := make(map[string]bool)
	var versions []string
	for _, idx := range p.LatestFileIndex {
		if idx.GameVersion != "" && !seen[idx.GameVersion] {
			seen[idx.GameVersion] = true
			versions = append(versions, idx.GameVersion)
		}
	}
	return versions
}

// toFileResponse converts a single curseforge.File to fileResponse.
func toFileResponse(f *curseforge.File) fileResponse {
	return fileResponse{
		ID:               f.ID,
		DisplayName:      f.DisplayName,
		FileName:         f.FileName,
		GameVersions:     f.GameVersions,
		IsServerPack:     f.IsServerPack,
		ServerPackFileID: f.ServerPackFileID,
		DownloadURL:      f.DownloadURL,
		FileLength:       f.FileLength,
		ReleaseType:      f.ReleaseType,
	}
}

// toFileResponses converts a slice of curseforge.File to fileResponse.
func toFileResponses(files []curseforge.File) []fileResponse {
	result := make([]fileResponse, len(files))
	for i, f := range files {
		result[i] = fileResponse{
			ID:               f.ID,
			DisplayName:      f.DisplayName,
			FileName:         f.FileName,
			GameVersions:     f.GameVersions,
			IsServerPack:     f.IsServerPack,
			ServerPackFileID: f.ServerPackFileID,
			DownloadURL:      f.DownloadURL,
			FileLength:       f.FileLength,
			ReleaseType:      f.ReleaseType,
		}
	}
	return result
}
