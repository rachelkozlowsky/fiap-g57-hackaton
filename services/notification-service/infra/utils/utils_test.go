package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_ReturnsDefault_WhenNotSet(t *testing.T) {
	os.Unsetenv("TEST_GETENV_KEY")
	result := GetEnv("TEST_GETENV_KEY", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnv_ReturnsEnvValue_WhenSet(t *testing.T) {
	os.Setenv("TEST_GETENV_KEY", "actual_value")
	defer os.Unsetenv("TEST_GETENV_KEY")
	result := GetEnv("TEST_GETENV_KEY", "default_value")
	assert.Equal(t, "actual_value", result)
}

func TestGetEnv_ReturnsEmptyString_WhenSetToEmpty(t *testing.T) {
	// LookupEnv: empty string is still a valid set value → returns "" not default
	os.Setenv("TEST_GETENV_KEY_EMPTY", "")
	defer os.Unsetenv("TEST_GETENV_KEY_EMPTY")
	result := GetEnv("TEST_GETENV_KEY_EMPTY", "default_value")
	assert.Equal(t, "", result)
}

func TestGetEnv_DifferentDefaults(t *testing.T) {
	os.Unsetenv("TEST_GETENV_MISSING")
	assert.Equal(t, "foo", GetEnv("TEST_GETENV_MISSING", "foo"))
	assert.Equal(t, "bar", GetEnv("TEST_GETENV_MISSING", "bar"))
	assert.Equal(t, "", GetEnv("TEST_GETENV_MISSING", ""))
}
