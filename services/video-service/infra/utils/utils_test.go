package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_Exists(t *testing.T) {
	os.Setenv("TEST_VIDEO_KEY", "myvalue")
	defer os.Unsetenv("TEST_VIDEO_KEY")
	assert.Equal(t, "myvalue", GetEnv("TEST_VIDEO_KEY", "default"))
}

func TestGetEnv_Default(t *testing.T) {
	os.Unsetenv("MISSING_VIDEO_KEY")
	assert.Equal(t, "fallback", GetEnv("MISSING_VIDEO_KEY", "fallback"))
}
