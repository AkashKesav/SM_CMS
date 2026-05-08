package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cms-backend/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// JWKS cache for Supabase JWT verification
var (
	jwksCache         *jwt.MapClaims
	jwksPublicKey     interface{}
	jwksOnce          sync.Once
	jwksMu            sync.RWMutex
	jwksExpireAt      time.Time
	supabaseJWTSecret string
)

func init() {
	// Supabase uses the JWT_SECRET from your project to sign tokens
	// The JWT secret is available in Supabase Dashboard > Settings > API > JWT Secret
	supabaseJWTSecret = os.Getenv("JWT_SECRET")
}

// JWTAuth middleware validates JWT tokens from Supabase Auth
func JWTAuth(jwtSecret string) fiber.Handler {
	secret := jwtSecret
	if secret == "" {
		secret = supabaseJWTSecret
	}

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Allow demo tokens only in local/demo test modes.
		if strings.HasPrefix(tokenString, "demo-token") && allowDemoToken() {
			log.Debug().Msg("Using demo token for testing")
			c.Locals("user_id", "demo-user-id")
			c.Locals("user_email", "demo@example.com")
			c.Locals("user_role", "admin")
			return c.Next()
		}

		// Parse and validate the JWT using Supabase's JWT secret (HS256)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Supabase uses HS256 by default
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil {
			if userID, email, role, verifyErr := verifyWithSupabaseAuth(tokenString); verifyErr == nil {
				c.Locals("user_id", userID)
				c.Locals("user_email", email)
				c.Locals("user_role", role)
				return c.Next()
			}

			// If real JWT verification fails and we're in demo mode, allow with mock user
			if isDemoMode() {
				log.Debug().Msg("JWT validation failed, but running in demo mode — allowing request")
				c.Locals("user_id", "demo-user-id")
				c.Locals("user_email", "demo@example.com")
				c.Locals("user_role", "admin")
				return c.Next()
			}
			log.Warn().Err(err).Msg("JWT validation failed")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid or expired token",
				"details": err.Error(),
			})
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, _ := claims["sub"].(string)
			email, _ := claims["email"].(string)

			// Extract role from app_metadata or default
			role := "authenticated"
			if appMeta, ok := claims["app_metadata"].(map[string]interface{}); ok {
				if r, ok := appMeta["role"].(string); ok {
					role = r
				}
			}
			if roleMeta, ok := claims["role"].(string); ok && roleMeta != "" {
				role = roleMeta
			}

			c.Locals("user_id", userID)
			c.Locals("user_email", email)
			c.Locals("user_role", role)
		}

		return c.Next()
	}
}

func isDemoMode() bool {
	return os.Getenv("DATABASE_URL") == "" || os.Getenv("SUPABASE_URL") == ""
}

func verifyWithSupabaseAuth(tokenString string) (string, string, string, error) {
	supabaseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	apiKey := os.Getenv("SUPABASE_ANON_KEY")
	if supabaseURL == "" || apiKey == "" {
		return "", "", "", fmt.Errorf("Supabase URL or anon key is not configured")
	}

	req, err := http.NewRequest("GET", supabaseURL+"/auth/v1/user", nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", "", "", fmt.Errorf("Supabase token verification failed with status %d", resp.StatusCode)
	}

	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", "", "", err
	}

	userID, _ := user["id"].(string)
	email, _ := user["email"].(string)
	role := "authenticated"
	if appMeta, ok := user["app_metadata"].(map[string]interface{}); ok {
		if value, ok := appMeta["role"].(string); ok && value != "" {
			role = value
		}
	}
	if userMeta, ok := user["user_metadata"].(map[string]interface{}); ok {
		if value, ok := userMeta["role"].(string); ok && value != "" {
			role = value
		}
	}
	if userID == "" {
		return "", "", "", fmt.Errorf("Supabase user response did not include an id")
	}
	return userID, email, role, nil
}

func allowDemoToken() bool {
	if isDemoMode() {
		return true
	}

	switch strings.ToLower(os.Getenv("ALLOW_DEMO_AUTH")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

type ProfileResolver interface {
	GetProfile(ctx context.Context, userID, email string) (model.UserProfile, error)
}

func LoadProfile(resolver ProfileResolver) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if resolver == nil {
			return c.Next()
		}
		userID, _ := c.Locals("user_id").(string)
		email, _ := c.Locals("user_email").(string)
		if userID == "" {
			return c.Next()
		}

		profile, err := resolver.GetProfile(c.Context(), userID, email)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to load user profile")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unable to load user profile",
			})
		}

		c.Locals("profile", profile)
		c.Locals("user_role", profile.Role)
		if profile.StudentID != nil {
			c.Locals("student_id", *profile.StudentID)
		}
		return c.Next()
	}
}

func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("user_role").(string)
		if role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}

// ErrorHandler provides consistent error responses
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	log.Error().Err(err).Int("status", code).Str("method", c.Method()).Str("path", c.Path()).Msg("Request error")

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   http.StatusText(code),
		"details": err.Error(),
	})
}
