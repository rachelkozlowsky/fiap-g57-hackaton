package utils

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_Exists(t *testing.T) {
	os.Setenv("TEST_KEY", "hello")
	defer os.Unsetenv("TEST_KEY")
	assert.Equal(t, "hello", GetEnv("TEST_KEY", "default"))
}

func TestGetEnv_Default(t *testing.T) {
	os.Unsetenv("MISSING_KEY")
	assert.Equal(t, "default", GetEnv("MISSING_KEY", "default"))
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	p := TimePtr(now)
	assert.NotNil(t, p)
	assert.Equal(t, now, *p)
}

func TestStringPtr(t *testing.T) {
	p := StringPtr("hello")
	assert.NotNil(t, p)
	assert.Equal(t, "hello", *p)
}

func TestIntPtr(t *testing.T) {
	p := IntPtr(42)
	assert.NotNil(t, p)
	assert.Equal(t, 42, *p)
}

func TestInt64Ptr(t *testing.T) {
	var v int64 = 1024
	p := Int64Ptr(v)
	assert.NotNil(t, p)
	assert.Equal(t, v, *p)
}
