package front_service

import (
	"errors"
	"log/slog"
	"net/url"
)

func (s *FrontService) RegisterChunkServer(serverURL string) error {
	if serverURL == "" {
		return errors.New("URL not provided")
	}

	parsedURL, err := url.ParseRequestURI(serverURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.New("invalid URL")
	}

	s.logger.Info("Registering chunk server", slog.String("url", serverURL))
	err = s.registry.AddChunkServer(serverURL)
	if err != nil {
		return err
	}

	return nil
}
