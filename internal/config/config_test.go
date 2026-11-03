package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("PUBLIC_URL", "https://starter.example.org/")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicURL != "https://starter.example.org" || c.AuthURL != "https://auth.example.org" {
		t.Fatalf("%+v", c)
	}
}

func TestRefusals(t *testing.T) {
	cases := map[string]map[string]string{
		"no public url": {"PUBLIC_URL": ""},
		"placeholder":   {"PUBLIC_URL": "https://x.example.org", "EMAIL_BACKEND": "smtp", "SMTP_HOST": "h", "SMTP_USER": "u", "SMTP_PASS": "CHANGE_ME"},
		"smtp missing":  {"PUBLIC_URL": "https://x.example.org", "EMAIL_BACKEND": "smtp"},
		"bad backend":   {"PUBLIC_URL": "https://x.example.org", "EMAIL_BACKEND": "carrier-pigeon"},
		"bad from":      {"PUBLIC_URL": "https://x.example.org", "EMAIL_BACKEND": "log", "EMAIL_FROM": "not an address"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Fatal("accepted")
			} else if name == "placeholder" && !strings.Contains(err.Error(), "SMTP_PASS") {
				t.Fatalf("error does not name the setting: %v", err)
			}
		})
	}
}
