package pgclient

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		config   ConnConfig
		expected string
	}{
		{
			name: "Basic configuration",
			config: ConnConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				Database: "postgres",
				SSLMode:  "disable",
			},
			expected: "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable",
		},
		{
			name: "Configuration with special characters",
			config: ConnConfig{
				Host:     "db.example.com",
				Port:     5432,
				Username: "user@domain",
				Password: "p@ssw0rd!",
				Database: "test_db",
				SSLMode:  "require",
			},
			expected: "postgresql://user@domain:p@ssw0rd!@db.example.com:5432/test_db?sslmode=require",
		},
		{
			name: "Configuration with different SSL mode",
			config: ConnConfig{
				Host:     "localhost",
				Port:     5433,
				Username: "admin",
				Password: "secure",
				Database: "production",
				SSLMode:  "verify-full",
			},
			expected: "postgresql://admin:secure@localhost:5433/production?sslmode=verify-full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnConfig_ID(t *testing.T) {
	tests := []struct {
		name     string
		config   ConnConfig
		expected string
	}{
		{
			name: "Basic configuration",
			config: ConnConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				Database: "postgres",
				SSLMode:  "disable",
			},
			expected: "localhost:postgres@postgres",
		},
		{
			name: "Different host and database",
			config: ConnConfig{
				Host:     "db.example.com",
				Port:     5432,
				Username: "user",
				Password: "password",
				Database: "test_db",
				SSLMode:  "require",
			},
			expected: "db.example.com:user@test_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ID()
			assert.Equal(t, tt.expected, result)
		})
	}
}
