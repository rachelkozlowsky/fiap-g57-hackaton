package security

import (
"testing"
"auth-service/domain"
"github.com/stretchr/testify/assert"
)

func TestGenerateAccessToken_Success(t *testing.T) {
	user := &domain.User{
		ID:    "user123",
		Email: "user@example.com",
		Role:  "user",
	}

	token, err := GenerateAccessToken(user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	user := &domain.User{
		ID:    "user123",
		Email: "user@example.com",
		Role:  "user",
	}

	token, err := GenerateRefreshToken(user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateAccessToken_Success(t *testing.T) {
	user := &domain.User{
		ID:    "user123",
		Email: "user@example.com",
		Role:  "admin",
	}

	token, err := GenerateAccessToken(user)
	assert.NoError(t, err)

	claims, err := ValidateAccessToken(token)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "g57-auth-service", claims.Issuer)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	claims, err := ValidateAccessToken("invalid.token.string")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_EmptyToken(t *testing.T) {
	claims, err := ValidateAccessToken("")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_TamperedToken(t *testing.T) {
	user := &domain.User{ID: "user123", Email: "user@example.com", Role: "user"}
	token, _ := GenerateAccessToken(user)

	tampered := token[:len(token)-5] + "XXXXX"
	claims, err := ValidateAccessToken(tampered)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_Success(t *testing.T) {
	user := &domain.User{
		ID:    "user456",
		Email: "refresh@example.com",
		Role:  "user",
	}

	token, err := GenerateRefreshToken(user)
	assert.NoError(t, err)

	claims, err := ValidateRefreshToken(token)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user456", claims.UserID)
	assert.Equal(t, "refresh@example.com", claims.Email)
}

func TestValidateRefreshToken_InvalidToken(t *testing.T) {
	claims, err := ValidateRefreshToken("not.a.token")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestGenerateAndValidate_DifferentUsers(t *testing.T) {
	users := []*domain.User{
		{ID: "1", Email: "a@example.com", Role: "user"},
		{ID: "2", Email: "b@example.com", Role: "admin"},
		{ID: "3", Email: "c@example.com", Role: "user"},
	}

	for _, user := range users {
		token, err := GenerateAccessToken(user)
		assert.NoError(t, err)

		claims, err := ValidateAccessToken(token)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	}
}

func TestTokensAreNotEmpty(t *testing.T) {
	user := &domain.User{ID: "user123", Email: "user@example.com", Role: "user"}

	accessToken, errA := GenerateAccessToken(user)
	refreshToken, errR := GenerateRefreshToken(user)

	assert.NoError(t, errA)
	assert.NoError(t, errR)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
}

// TestValidateAccessToken_WrongSigningMethod exercises the "unexpected signing method" branch
// by passing a JWT that was signed with RS256 instead of HS256.
// eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9 = {"alg":"RS256","typ":"JWT"}
func TestValidateAccessToken_WrongSigningMethod(t *testing.T) {
	rs256Token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.e30.c2lnbmF0dXJl"

	claims, err := ValidateAccessToken(rs256Token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestValidateRefreshToken_WrongSigningMethod(t *testing.T) {
	rs256Token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.e30.c2lnbmF0dXJl"

	claims, err := ValidateRefreshToken(rs256Token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}