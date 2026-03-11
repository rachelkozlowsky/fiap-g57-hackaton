package utils

import (
	"os"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGetEnv_ReturnsDefault_WhenNotSet(t *testing.T) {
	os.Unsetenv("TEST_VAR_MISSING")

	result := GetEnv("TEST_VAR_MISSING", "default_value")

	assert.Equal(t, "default_value", result)
}

func TestGetEnv_ReturnsEnvValue_WhenSet(t *testing.T) {
	os.Setenv("TEST_VAR_SET", "env_value")
	defer os.Unsetenv("TEST_VAR_SET")

	result := GetEnv("TEST_VAR_SET", "default_value")

	assert.Equal(t, "env_value", result)
}

func TestGetEnv_ReturnsDefault_WhenEnvIsEmpty(t *testing.T) {
	os.Setenv("TEST_VAR_EMPTY", "")
	defer os.Unsetenv("TEST_VAR_EMPTY")

	result := GetEnv("TEST_VAR_EMPTY", "fallback")

	assert.Equal(t, "fallback", result)
}

func TestGetEnv_DifferentDefaults(t *testing.T) {
	os.Unsetenv("NOT_SET_VAR")

	assert.Equal(t, "a", GetEnv("NOT_SET_VAR", "a"))
	assert.Equal(t, "b", GetEnv("NOT_SET_VAR", "b"))
	assert.Equal(t, "", GetEnv("NOT_SET_VAR", ""))
}
