package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_String(t *testing.T) {
	assert.Equal(t, "1.0.0", String())
}
