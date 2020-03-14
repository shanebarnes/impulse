package version

import (
	"fmt"
)

const (
	verMajor = 1
	verMinor = 0
	verPatch = 0
)

func String() string {
	return fmt.Sprintf("%d.%d.%d", verMajor, verMinor, verPatch)
}
