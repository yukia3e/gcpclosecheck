//go:build tools

package main

// This file declares dependencies that are used when running go generate, or used
// during the development process but not otherwise depended on by built code.

import (
	_ "cloud.google.com/go/spanner" // Used in tests for type checking and analysis
)
