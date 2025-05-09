package auth

import (
	"errors"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Service provides authentication functionality
type Service struct {
	apiToken string
}

// NewService creates a new authentication service that uses API token
func NewService() *Service {
	// Attempt to load token from env
	_ = godotenv.Load("env/.env")

	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		// Use a default token only for development
		apiToken = "your-default-api-token"
	}

	return &Service{
		apiToken: apiToken,
	}
}

// ExtractTokenFromHeader extracts the API token from the Authorization header
func ExtractTokenFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("authorization header format must be Bearer {token}")
	}

	return parts[1], nil
}

// ValidateToken validates the provided API token
func (s *Service) ValidateToken(token string) error {
	if token != s.apiToken {
		return errors.New("invalid API token")
	}
	return nil
}
