package version

import (
	"fmt"
)

// Version holds the current version.
var Version = "source"

// String returns version string.
func String() string {
	return fmt.Sprintf("CRSM Operator version: %s", Version)
}
