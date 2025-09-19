package config

import "os"

type Config struct {
	Port      string
	JWTSecret string
	TTSBase   string
}

func Load() Config {
	return Config{
		Port:      getenv("PORT", "8080"),
		JWTSecret: getenv("JWT_SECRET", "devsecret"),
		TTSBase:   getenv("TTS_BASE_URL", ""),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
