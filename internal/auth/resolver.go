package auth

import "os"

func ResolveAPIKey(store *Store, provider, envVar string) string {
	if store != nil {
		if key, err := store.Get(provider); err == nil && key != "" {
			return key
		}
	}

	if envVar != "" {
		if key := os.Getenv(envVar); key != "" {
			return key
		}
	}

	return ""
}

type KeySource string

const (
	SourceAuthStore KeySource = "auth.json"
	SourceEnvVar    KeySource = "env var"
	SourceNone      KeySource = "not configured"
)

func ResolveAPIKeySource(store *Store, provider, envVar string) (string, KeySource) {
	if store != nil {
		if key, err := store.Get(provider); err == nil && key != "" {
			return key, SourceAuthStore
		}
	}

	if envVar != "" {
		if key := os.Getenv(envVar); key != "" {
			return key, SourceEnvVar
		}
	}

	return "", SourceNone
}
