package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_ReturnsDefault_WhenNotSet(t *testing.T) {
	os.Unsetenv("TEST_PROC_KEY")
	assert.Equal(t, "default", GetEnv("TEST_PROC_KEY", "default"))
}

func TestGetEnv_ReturnsEnvValue_WhenSet(t *testing.T) {
	os.Setenv("TEST_PROC_KEY", "actual")
	defer os.Unsetenv("TEST_PROC_KEY")
	assert.Equal(t, "actual", GetEnv("TEST_PROC_KEY", "default"))
}

func TestGetEnv_ReturnsEmptyString_WhenSetToEmpty(t *testing.T) {
	// LookupEnv: empty string env var is valid → returns "" not default
	os.Setenv("TEST_PROC_KEY_EMPTY", "")
	defer os.Unsetenv("TEST_PROC_KEY_EMPTY")
	assert.Equal(t, "", GetEnv("TEST_PROC_KEY_EMPTY", "default"))
}

func TestGetEnv_DifferentDefaults(t *testing.T) {
	os.Unsetenv("TEST_PROC_MISSING")
	assert.Equal(t, "foo", GetEnv("TEST_PROC_MISSING", "foo"))
	assert.Equal(t, "bar", GetEnv("TEST_PROC_MISSING", "bar"))
	assert.Equal(t, "", GetEnv("TEST_PROC_MISSING", ""))
}

func TestGetEnv_FFMPEG_FPS_Default(t *testing.T) {
	os.Unsetenv("FFMPEG_FPS")
	assert.Equal(t, "1", GetEnv("FFMPEG_FPS", "1"))
}

func TestGetEnv_FFMPEG_FPS_Override(t *testing.T) {
	os.Setenv("FFMPEG_FPS", "25")
	defer os.Unsetenv("FFMPEG_FPS")
	assert.Equal(t, "25", GetEnv("FFMPEG_FPS", "1"))
}
