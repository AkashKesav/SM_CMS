package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SupabaseURL        string
	SupabaseAnonKey    string
	SupabaseServiceKey string
	DatabaseURL        string
	BackendPort        string
	BackendHost        string
	JWTSecret          string
	SessionSecret      string
	StorageBucket      string
	Environment        string
	AllowedOrigins     []string
}

func Load() (*Config, error) {
	// Load root-level env when the backend is run from either repo root or backend/.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := &Config{
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:    getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseServiceKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		BackendPort:        getEnv("BACKEND_PORT", "8080"),
		BackendHost:        getEnv("BACKEND_HOST", "localhost"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		SessionSecret:      getEnv("SESSION_SECRET", ""),
		StorageBucket:      getEnv("SUPABASE_STORAGE_BUCKET", "cms-media"),
		Environment:        getEnv("BACKEND_ENV", "development"),
		AllowedOrigins:     []string{"http://localhost:*", "http://tauri.localhost"},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
