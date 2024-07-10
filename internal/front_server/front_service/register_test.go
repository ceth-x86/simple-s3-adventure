package front_service

import (
	"errors"
	"log/slog"
	"os"
	"simple-s3-adventure/internal/front_server/registry_service"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterChunkServer(t *testing.T) {
	tests := []struct {
		name              string
		serverURL         string
		addChunkServerErr error
		expectedErr       error
	}{
		{
			name:        "empty URL",
			serverURL:   "",
			expectedErr: errors.New("URL not provided"),
		},
		{
			name:        "invalid URL",
			serverURL:   "invalid-url",
			expectedErr: errors.New("invalid URL"),
		},
		{
			name:              "valid URL",
			serverURL:         "http://example.com",
			addChunkServerErr: nil,
			expectedErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &FrontService{
				logger:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
				registry: registry_service.NewChunkServerRegistry(),
			}

			err := service.RegisterChunkServer(tt.serverURL)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
