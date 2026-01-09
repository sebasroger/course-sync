package config

import "os"

type Config struct {
	// Eightfold
	EightfoldBaseURL   string
	EightfoldBasicAuth string
	EightfoldUser      string
	EightfoldPass      string

	// Udemy
	UdemyBaseURL      string
	UdemyClientID     string
	UdemyClientSecret string

	// Pluralsight
	PluralsightBaseURL string
	PluralsightToken   string
}

func Load() Config {
	return Config{
		// Eightfold
		EightfoldBaseURL:   getenv("EIGHTFOLD_BASE_URL", "https://apiv2.eightfold.ai"),
		EightfoldBasicAuth: os.Getenv("EIGHTFOLD_BASIC_AUTH"),
		EightfoldUser:      os.Getenv("EIGHTFOLD_USERNAME"),
		EightfoldPass:      os.Getenv("EIGHTFOLD_PASSWORD"),

		// Udemy
		UdemyBaseURL:      getenv("UDEMY_BASE_URL", "https://femsa.udemy.com/api-2.0/organizations/243186"),
		UdemyClientID:     os.Getenv("UDEMY_CLIENT_ID"),
		UdemyClientSecret: os.Getenv("UDEMY_CLIENT_SECRET"),

		// Pluralsight
		PluralsightBaseURL: getenv("PLURALSIGHT_GQL_URL", "https://api.pluralsight.com/api"),
		PluralsightToken:   os.Getenv("PLURALSIGHT_TOKEN"),
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
