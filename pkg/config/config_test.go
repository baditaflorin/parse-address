package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Clear any existing env vars
	os.Clearenv()

	// Test with defaults
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Default port: got %d, want 8080", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Default host: got %s, want 0.0.0.0", cfg.Server.Host)
	}

	if cfg.Security.MaxInputLength != 10000 {
		t.Errorf("Default max input: got %d, want 10000", cfg.Security.MaxInputLength)
	}
}

func TestLoadWithCustomValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("SERVER_PORT", "9000")
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SECURITY_MAX_INPUT_LENGTH", "5000")
	os.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 9000 {
		t.Errorf("Custom port: got %d, want 9000", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Custom host: got %s, want 127.0.0.1", cfg.Server.Host)
	}

	if cfg.Security.MaxInputLength != 5000 {
		t.Errorf("Custom max input: got %d, want 5000", cfg.Security.MaxInputLength)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Custom log level: got %s, want debug", cfg.Logging.Level)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "Valid config",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
				},
				Security: SecurityConfig{
					MaxInputLength: 1000,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: false,
		},
		{
			name: "Invalid port - too low",
			config: Config{
				Server: ServerConfig{
					Port:         0,
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
				},
				Security: SecurityConfig{
					MaxInputLength: 1000,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: true,
		},
		{
			name: "Invalid port - too high",
			config: Config{
				Server: ServerConfig{
					Port:         99999,
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
				},
				Security: SecurityConfig{
					MaxInputLength: 1000,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: true,
		},
		{
			name: "Invalid timeout",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  0,
					WriteTimeout: 10 * time.Second,
				},
				Security: SecurityConfig{
					MaxInputLength: 1000,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: true,
		},
		{
			name: "Invalid max input length - too low",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
				},
				Security: SecurityConfig{
					MaxInputLength: 50,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: true,
		},
		{
			name: "Invalid log level",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
				},
				Security: SecurityConfig{
					MaxInputLength: 1000,
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	os.Clearenv()

	// Test default value
	result := getEnvAsInt("NONEXISTENT", 42)
	if result != 42 {
		t.Errorf("Default value: got %d, want 42", result)
	}

	// Test valid integer
	os.Setenv("TEST_INT", "100")
	result = getEnvAsInt("TEST_INT", 42)
	if result != 100 {
		t.Errorf("Valid integer: got %d, want 100", result)
	}

	// Test invalid integer (should return default)
	os.Setenv("TEST_INT", "not_a_number")
	result = getEnvAsInt("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Invalid integer fallback: got %d, want 42", result)
	}
}

func TestGetEnvAsBool(t *testing.T) {
	os.Clearenv()

	// Test default value
	result := getEnvAsBool("NONEXISTENT", true)
	if result != true {
		t.Errorf("Default value: got %v, want true", result)
	}

	// Test valid boolean
	os.Setenv("TEST_BOOL", "false")
	result = getEnvAsBool("TEST_BOOL", true)
	if result != false {
		t.Errorf("Valid boolean: got %v, want false", result)
	}

	// Test invalid boolean (should return default)
	os.Setenv("TEST_BOOL", "not_a_bool")
	result = getEnvAsBool("TEST_BOOL", true)
	if result != true {
		t.Errorf("Invalid boolean fallback: got %v, want true", result)
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	os.Clearenv()

	// Test default value
	result := getEnvAsDuration("NONEXISTENT", 5*time.Second)
	if result != 5*time.Second {
		t.Errorf("Default value: got %v, want 5s", result)
	}

	// Test valid duration
	os.Setenv("TEST_DURATION", "10s")
	result = getEnvAsDuration("TEST_DURATION", 5*time.Second)
	if result != 10*time.Second {
		t.Errorf("Valid duration: got %v, want 10s", result)
	}

	// Test invalid duration (should return default)
	os.Setenv("TEST_DURATION", "not_a_duration")
	result = getEnvAsDuration("TEST_DURATION", 5*time.Second)
	if result != 5*time.Second {
		t.Errorf("Invalid duration fallback: got %v, want 5s", result)
	}
}
