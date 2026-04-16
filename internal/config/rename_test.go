package config_test

import (
	"testing"

	"github.com/TruyLabs/rias/internal/config"
)

func TestDefaultAgentNameIsRias(t *testing.T) {
	if config.DefaultAgentName != "rias" {
		t.Errorf("expected DefaultAgentName=rias, got %q", config.DefaultAgentName)
	}
}

func TestDefaultPathsUseRias(t *testing.T) {
	if config.DefaultBrainPath != "~/.rias/brain" {
		t.Errorf("expected DefaultBrainPath=~/.rias/brain, got %q", config.DefaultBrainPath)
	}
	if config.DefaultSessionsPath != "~/.rias/sessions" {
		t.Errorf("expected DefaultSessionsPath=~/.rias/sessions, got %q", config.DefaultSessionsPath)
	}
}

func TestDefaultUserNameIsUser(t *testing.T) {
	if config.DefaultUserName != "User" {
		t.Errorf("expected DefaultUserName=User, got %q", config.DefaultUserName)
	}
}
