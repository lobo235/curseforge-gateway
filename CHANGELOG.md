# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.0.0] - 2026-03-23

### Added

- Initial project scaffold: go.mod, cmd/server/main.go, internal/ layout
- Config loading from environment variables with validation
- CurseForge API client with in-memory caching (30min project, 5min files)
- HTTP handlers for modpack and mod validation endpoints
- Bearer token authentication middleware
- Request logging middleware with X-Trace-ID propagation
- Health check endpoint (unauthenticated)
- Graceful shutdown on SIGINT/SIGTERM
- Makefile with standard targets (build, test, cover, lint, run, hooks, clean)
- Dockerfile (multi-stage: golang:1.24-alpine -> alpine:3.21)
- golangci-lint configuration
- Pre-commit hook (lint + test)
- Full test suite for config, client, and handlers
