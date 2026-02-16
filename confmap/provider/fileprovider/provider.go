// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fileprovider // import "go.opentelemetry.io/collector/confmap/provider/fileprovider"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

const schemeName = "file"

type provider struct{}

// NewFactory returns a factory for a confmap.Provider that reads the configuration from a file.
//
// This Provider supports "file" scheme, and can be called with a "uri" that follows:
//
//	file-uri		= "file:" local-path
//	local-path		= [ drive-letter ] file-path
//	drive-letter	= ALPHA ":"
//
// The "file-path" can be relative or absolute, and it can be any OS supported format.
//
// Examples:
// `file:path/to/file` - relative path (unix, windows)
// `file:/path/to/file` - absolute path (unix, windows)
// `file:c:/path/to/file` - absolute path including drive-letter (windows)
// `file:c:\path\to\file` - absolute path including drive-letter (windows)
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(confmap.ProviderSettings) confmap.Provider {
	return &provider{}
}

func getCredentials(filePath string) (string, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}
	var apiKey, apiSecret string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "API_KEY=") {
			apiKey = strings.TrimPrefix(line, "API_KEY=")
		}
		if strings.HasPrefix(line, "API_SECRET=") {
			apiSecret = strings.TrimPrefix(line, "API_SECRET=")
		}
	}
	return apiKey, apiSecret, nil
}

func replaceConfig(config, clientID, clientSecret string) string {

	// Use simple string replacement approach instead of regex for better control
	lines := strings.Split(config, "\n")
	var result []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if this line is exactly "client_id:" (with possible leading whitespace)
		if strings.HasSuffix(trimmedLine, "client_id:") && !strings.Contains(trimmedLine, "client_secret") {
			// Extract leading whitespace
			leadingSpaces := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			newLine := leadingSpaces + `client_id: "` + clientID + `"`
			result = append(result, newLine)
		} else if strings.HasSuffix(trimmedLine, "client_secret:") && !strings.Contains(trimmedLine, "client_id") {
			// Extract leading whitespace
			leadingSpaces := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			newLine := leadingSpaces + `client_secret: "` + clientSecret + `"`
			result = append(result, newLine)
		} else {
			result = append(result, line)
		}
	}

	config = strings.Join(result, "\n")
	return config
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	// Clean the path before using it.
	content, err := os.ReadFile(filepath.Clean(uri[len(schemeName)+1:]))
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", uri, err)
	}
	configStr := string(content)
	apiKey, apiSecret, err := getCredentials("/etc/nsg/regInfo")
	if err != nil {
		return nil, fmt.Errorf("error reading credentials file: %w", err)
	}
	updatedConfig := replaceConfig(configStr, apiKey, apiSecret)
	return confmap.NewRetrievedFromYAML([]byte(updatedConfig))
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
