package main

import (
	"strings"
	"testing"
)

func TestJWTSecretsFromEnv(t *testing.T) {
	for _, tc := range []struct{ name, access, refresh, badKey string }{
		{"valid", "test-access-key", "test-refresh-key", ""},
		{"missing access", "", "test-refresh-key", "ACCESS_SECRET"},
		{"missing refresh", "test-access-key", "", "REFRESH_SECRET"},
		{"blank access", " \t\n", "test-refresh-key", "ACCESS_SECRET"},
		{"blank refresh", "test-access-key", " \t", "REFRESH_SECRET"},
		{"old access default", "access-secret", "test-refresh-key", "ACCESS_SECRET"},
		{"old refresh default", "test-access-key", "refresh-secret", "REFRESH_SECRET"},
		{"swapped default", "refresh-secret", "test-refresh-key", "ACCESS_SECRET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ACCESS_SECRET", tc.access)
			t.Setenv("REFRESH_SECRET", tc.refresh)
			access, refresh, err := jwtSecretsFromEnv()
			if tc.badKey != "" {
				if err == nil || !strings.Contains(err.Error(), tc.badKey) {
					t.Fatalf("expected configuration error naming %s", tc.badKey)
				}
				if access != "" || refresh != "" {
					t.Fatal("invalid configuration returned secrets")
				}
				for _, value := range []string{tc.access, tc.refresh} {
					if strings.TrimSpace(value) != "" && strings.Contains(err.Error(), value) {
						t.Fatal("error exposed a secret value")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if access != tc.access || refresh != tc.refresh {
				t.Fatal("configured secrets were changed")
			}
		})
	}
}
