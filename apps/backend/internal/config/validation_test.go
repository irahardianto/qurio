package config_test

import (
	"errors"
	"testing"

	"qurio/apps/backend/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
		errIs   error
	}{
		{
			name: "Valid Config",
			config: config.Config{
				DBHost: "localhost",
				DBUser: "user",
				DBName: "db",
			},
			wantErr: false,
		},
		{
			name: "Missing DBHost",
			config: config.Config{
				DBHost: "",
				DBUser: "user",
				DBName: "db",
			},
			wantErr: true,
			errIs:   config.ErrMissingRequired,
		},
		{
			name: "Missing DBUser",
			config: config.Config{
				DBHost: "localhost",
				DBUser: "",
				DBName: "db",
			},
			wantErr: true,
			errIs:   config.ErrMissingRequired,
		},
		{
			name: "Missing DBName",
			config: config.Config{
				DBHost: "localhost",
				DBUser: "user",
				DBName: "",
			},
			wantErr: true,
			errIs:   config.ErrMissingRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.True(t, errors.Is(err, tt.errIs))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
