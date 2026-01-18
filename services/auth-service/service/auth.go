package service

import (
	"time"
	"auth-service/domain"
	"auth-service/security"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db    domain.DatabaseInterface
	redis domain.RedisClientInterface
}

func NewAuthService(db domain.DatabaseInterface, redis domain.RedisClientInterface) *AuthService {
	return &AuthService{
		db:    db,
		redis: redis,
	}
}

func (s *AuthService) Register(email, password, name, ip, userAgent string) (*domain.User, string, string, error) {
	existingUser, _ := s.db.GetUserByEmail(email)
	if existingUser != nil {
		return nil, "", "", domain.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", err
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		Role:         "user",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.db.CreateUser(user); err != nil {
		return nil, "", "", err
	}

	accessToken, err := security.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := security.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", err
	}

	session := &domain.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IPAddress:    ip,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
	}

	if err := s.db.CreateSession(session); err != nil {
		return nil, "", "", err
	}

	s.redis.SetUser(user.ID, user, 24*time.Hour)

	s.db.CreateAuditLog(&domain.AuditLog{
		UserID:     &user.ID,
		Action:     "user.register",
		EntityType: "user",
		EntityID:   &user.ID,
		IPAddress:  ip,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	})

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) Login(email, password, ip, userAgent string) (*domain.User, string, string, error) {
	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		return nil, "", "", domain.ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, "", "", domain.ErrAccountDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", domain.ErrInvalidCredentials
	}

	accessToken, err := security.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := security.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", err
	}

	session := &domain.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IPAddress:    ip,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
	}

	if err := s.db.CreateSession(session); err != nil {
		return nil, "", "", err
	}

	now := time.Now()
	user.LastLoginAt = &now
	s.db.UpdateUser(user)

	s.redis.SetUser(user.ID, user, 24*time.Hour)

	s.db.CreateAuditLog(&domain.AuditLog{
		UserID:     &user.ID,
		Action:     "user.login",
		EntityType: "user",
		EntityID:   &user.ID,
		IPAddress:  ip,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	})

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) RefreshToken(token string) (string, error) {
	claims, err := security.ValidateRefreshToken(token)
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	session, err := s.db.GetSessionByRefreshToken(token)
	if err != nil {
		return "", domain.ErrSessionNotFound
	}

	if session.ExpiresAt.Before(time.Now()) {
		return "", domain.ErrSessionExpired
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		return "", domain.ErrUserNotFound
	}

	accessToken, err := security.GenerateAccessToken(user)
	if err != nil {
		return "", err
	}

	session.LastUsedAt = time.Now()
	s.db.UpdateSession(session)

	return accessToken, nil
}

func (s *AuthService) Logout(refreshToken, userID, ip, userAgent string) error {
	if err := s.db.DeleteSessionByRefreshToken(refreshToken); err != nil {
		return err
	}

	s.redis.DeleteUser(userID)

	s.db.CreateAuditLog(&domain.AuditLog{
		UserID:     &userID,
		Action:     "user.logout",
		EntityType: "user",
		EntityID:   &userID,
		IPAddress:  ip,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	})

	return nil
}

func (s *AuthService) GetCurrentUser(userID string) (*domain.User, error) {
	user, err := s.redis.GetUser(userID)
	if err != nil {
		user, err = s.db.GetUserByID(userID)
		if err != nil {
			return nil, domain.ErrUserNotFound
		}
		s.redis.SetUser(userID, user, 24*time.Hour)
	}
	return user, nil
}
