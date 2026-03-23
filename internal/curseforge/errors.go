package curseforge

import "errors"

// ErrNotFound indicates the requested project or file does not exist.
var ErrNotFound = errors.New("not found")

// ErrWrongClass indicates the project exists but has the wrong classId
// (e.g., looking up a mod ID on the modpacks endpoint).
var ErrWrongClass = errors.New("wrong class")
