package config

import (
	"fmt"
	"net/url"
	"strconv"
)

// API is what the API server needs: its database, where it listens and is
// reached, and its quiz provider.
type API struct {
	Database Database
	Port     int
	URL      string
	AI       AI
}

// LoadAPI reads the DB_*, APP_*, and AI_* variables. DB_PASSWORD may be
// empty; every other DB_* and APP_* variable is required.
func LoadAPI(getenv func(string) string) (API, error) {
	db, err := LoadDatabase(getenv)
	if err != nil {
		return API{}, err
	}
	for _, key := range []string{"APP_PORT", "APP_URL"} {
		if getenv(key) == "" {
			return API{}, fmt.Errorf("%s is not set; copy .env.example to .env", key)
		}
	}
	port, err := strconv.Atoi(getenv("APP_PORT"))
	if err != nil || port < 1 || port > 65535 {
		return API{}, fmt.Errorf("APP_PORT %q is not a port number", getenv("APP_PORT"))
	}
	appURL := getenv("APP_URL")
	u, err := url.Parse(appURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return API{}, fmt.Errorf("APP_URL %q is not an absolute http or https URL", appURL)
	}
	// A URL that names a port is this server itself, so it must be the port
	// the server listens on; one without a port is a proxy in front of it.
	if p := u.Port(); p != "" && p != strconv.Itoa(port) {
		return API{}, fmt.Errorf("APP_URL %q names port %s, but APP_PORT is %d", appURL, p, port)
	}
	ai, err := LoadAI(getenv)
	if err != nil {
		return API{}, err
	}
	return API{Database: db, Port: port, URL: appURL, AI: ai}, nil
}
