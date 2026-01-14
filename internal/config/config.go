package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Eightfold
	EightfoldBaseURL   string
	EightfoldBasicAuth string
	EightfoldUser      string
	EightfoldPass      string
	// Optional: if provided, we skip password-grant auth and use this directly.
	EightfoldBearerToken string

	// Udemy
	UdemyBaseURL      string
	UdemyClientID     string
	UdemyClientSecret string

	// Pluralsight
	PluralsightBaseURL string
	PluralsightToken   string

	// SFTP (para subir el CSV)
	SFTPHost                 string
	SFTPPort                 int
	SFTPUser                 string
	SFTPPass                 string
	SFTPDir                  string
	SFTPInsecureIgnoreHostKey bool
}

func Load() Config {
	return Config{
		// Eightfold
		EightfoldBaseURL:   os.Getenv("EIGHTFOLD_BASE_URL"),
		EightfoldBasicAuth: os.Getenv("EIGHTFOLD_BASIC_AUTH"),
		EightfoldUser:      os.Getenv("EIGHTFOLD_USERNAME"),
		EightfoldPass:      os.Getenv("EIGHTFOLD_PASSWORD"),
		EightfoldBearerToken: os.Getenv("EIGHTFOLD_BEARER_TOKEN"),

		// Udemy
		UdemyBaseURL:      os.Getenv("UDEMY_BASE_URL"),
		UdemyClientID:     os.Getenv("UDEMY_CLIENT_ID"),
		UdemyClientSecret: os.Getenv("UDEMY_CLIENT_SECRET"),

		// Pluralsight
		PluralsightBaseURL: os.Getenv("PLURALSIGHT_GQL_URL"),
		PluralsightToken:   os.Getenv("PLURALSIGHT_TOKEN"),

		// SFTP
		SFTPHost:                 getenv("SFTP_HOST", ""),
		SFTPPort:                 getenvInt("SFTP_PORT", 22),
		SFTPUser:                 getenv("SFTP_USER", ""),
		SFTPPass:                 getenv("SFTP_PASS", ""),
		SFTPDir:                  getenv("SFTP_DIR", "/upload"),
		SFTPInsecureIgnoreHostKey: getenvBool("SFTP_INSECURE_IGNORE_HOSTKEY", true),
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}