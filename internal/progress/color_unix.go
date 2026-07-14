//go:build !windows

package progress

import "os"

func vtEnabled() bool {
	return os.Getenv("NO_COLOR") == ""
}
