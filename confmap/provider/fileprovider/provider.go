// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fileprovider // import "go.opentelemetry.io/collector/confmap/provider/fileprovider"

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v3"
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
	return decryptRegInfoValue(apiKey), decryptRegInfoValue(apiSecret), nil
}

// decryptRegInfoValue decrypts an AES-GCM encrypted regInfo secret. If the value
// is not encrypted (or decryption fails) the original value is returned unchanged.
func decryptRegInfoValue(value string) string {
	if value == "" {
		return value
	}
	cipherText, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	key := sha256.Sum256([]byte("_"))
	block, err := aes.NewCipher(key[:16])
	if err != nil {
		return value
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return value
	}
	nonce := make([]byte, gcm.NonceSize())
	for index := range nonce {
		nonce[index] = byte(index)
	}
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return value
	}
	return string(plainText)
}

func replaceConfig(config []byte, clientID, clientSecret string) ([]byte, error) {
	// 1. Parse the YAML content into a generic map
	var configMap map[string]interface{}
	err := yaml.Unmarshal(config, &configMap)
	if err != nil {
		// If YAML parsing fails, return original config
		return config, fmt.Errorf("error parsing YAML: %w", err)
	}

	// 2. Navigate and replace credentials
	// Path: exporters -> keys containing "opsrampotlp" -> security -> client_id/client_secret
	if exporters, ok := configMap["exporters"].(map[string]interface{}); ok {
		for exporterName, exporterConfig := range exporters {
			// Check if the key contains "opsrampotlp"
			if strings.Contains(exporterName, "opsrampotlp") {
				if exporterMap, ok := exporterConfig.(map[string]interface{}); ok {
					if security, ok := exporterMap["security"].(map[string]interface{}); ok {
						// Update the values
						security["client_id"] = clientID
						security["client_secret"] = clientSecret
					} else {
						return config, fmt.Errorf("security section not found or invalid for exporter %q in YAML", exporterName)
					}
				}
			}
		}
	} else {
		return config, fmt.Errorf("exporters section not found or invalid in YAML")
	}

	// 3. Marshal back to YAML
	updatedYAML, err := yaml.Marshal(configMap)
	if err != nil {
		// If marshaling fails, return original config
		return config, fmt.Errorf("error marshaling updated config to YAML: %w", err)
	}

	return updatedYAML, nil
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

	apiKey, apiSecret, err := getCredentials("/etc/nsg/regInfo")
	if err != nil {
		return nil, fmt.Errorf("error reading credentials file: %w", err)
	}
	updatedConfig, err := replaceConfig(content, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("error replacing credentials in config: %w", err)
	}
	return confmap.NewRetrievedFromYAML(updatedConfig)
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
