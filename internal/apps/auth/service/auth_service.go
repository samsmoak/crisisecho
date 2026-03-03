package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"crisisecho/internal/apps/user/model"
	userSvc "crisisecho/internal/apps/user/service"
	"crisisecho/internal/firebase"
)

var (
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	jwtExpiry = 72 * time.Hour
)

// AuthService defines the public contract for the auth domain.
type AuthService interface {
	// GoogleAuth authenticates a user via a Firebase Google ID token.
	// On first login it creates the user; on subsequent logins it returns the existing user.
	// Returns a signed JWT and the user document.
	GoogleAuth(ctx context.Context, idToken string) (string, *model.User, error)

	// AppleAuth authenticates a user via an Apple Sign-In ID token (placeholder).
	AppleAuth(ctx context.Context, idToken string) (string, *model.User, error)

	// GenerateToken creates a signed JWT for the given user.
	GenerateToken(user *model.User) (string, error)

	// VerifyToken parses and validates a JWT string.
	VerifyToken(tokenStr string) (*jwt.Token, error)
}

type authService struct {
	userSvc userSvc.UserService
}

// NewAuthService constructs an AuthService with the given user service dependency.
func NewAuthService(uSvc userSvc.UserService) AuthService {
	return &authService{userSvc: uSvc}
}

// ── Google Auth ──────────────────────────────────────────────────────────────

func (s *authService) GoogleAuth(ctx context.Context, idToken string) (string, *model.User, error) {
	if idToken == "" {
		return "", nil, errors.New("id token is required")
	}

	if !firebase.IsAvailable() {
		return "", nil, errors.New("google authentication is currently unavailable")
	}

	// Verify the Firebase ID token once and extract all claims.
	firebaseToken, err := firebase.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", nil, fmt.Errorf("google auth failed: invalid token: %w", err)
	}

	email, _ := firebaseToken.Claims["email"].(string)
	if email == "" {
		return "", nil, errors.New("google auth failed: email not found in token")
	}

	emailVerified, _ := firebaseToken.Claims["email_verified"].(bool)
	if !emailVerified {
		return "", nil, errors.New("google auth failed: email not verified")
	}

	name, _ := firebaseToken.Claims["name"].(string)

	// Find or create the user in our database.
	user, err := s.userSvc.GetOrCreateUser(ctx, firebaseToken.UID, email, name, "")
	if err != nil {
		return "", nil, fmt.Errorf("google auth: user upsert failed: %w", err)
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("google auth: token generation failed: %w", err)
	}

	return token, user, nil
}

// ── Apple Auth (placeholder) ─────────────────────────────────────────────────

func (s *authService) AppleAuth(ctx context.Context, idToken string) (string, *model.User, error) {
	return "", nil, errors.New("apple sign-in is not yet implemented")
}

// ── JWT helpers ──────────────────────────────────────────────────────────────

func (s *authService) GenerateToken(user *model.User) (string, error) {
	if user == nil {
		return "", errors.New("user is nil")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":      user.ID.Hex(),
		"firebase_uid": user.FirebaseUID,
		"role":         user.Role,
		"exp":          time.Now().Add(jwtExpiry).Unix(),
	})
	return token.SignedString(jwtSecret)
}

func (s *authService) VerifyToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
}
