package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
)

const maxConfigBytes = 16 << 20

var ErrSecretStorage = errors.New("protected configuration cannot be read or written")

type protectedValue struct {
	Scheme     string `json:"scheme"`
	Ciphertext []byte `json:"ciphertext"`
}
type diskConfig struct {
	Config
	Protected *protectedValue `json:"protected_secrets,omitempty"`
}

func hasSecrets(c *Config) bool {
	return len(c.Keys) > 0 || len(c.KeyEntries) > 0 || c.Passphrase != "" || c.ImageKey != "" || c.Key != "" || c.ImageXORKey != nil
}
func clearSecretFields(c *Config) {
	c.Keys = nil
	c.KeyEntries = nil
	c.Passphrase = ""
	c.ImageKey = ""
	c.Key = ""
	c.ImageXORKey = nil
}
func copySecretFields(dst, src *Config) {
	dst.Keys = src.Keys
	dst.KeyEntries = src.KeyEntries
	dst.Passphrase = src.Passphrase
	dst.ImageKey = src.ImageKey
	dst.Key = src.Key
	dst.ImageXORKey = src.ImageXORKey
}

func encodeConfig(c *Config) ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil config")
	}
	copyCfg := *c
	copyCfg.Keys = maps.Clone(c.Keys)
	copyCfg.KeyEntries = maps.Clone(c.KeyEntries)
	copyCfg.normalize()
	copyCfg.SchemaVersion = CurrentSchemaVersion
	disk := diskConfig{Config: copyCfg}
	if hasSecrets(&copyCfg) {
		raw, err := json.Marshal(&copyCfg)
		if err != nil {
			return nil, err
		}
		defer clear(raw)
		protected, err := protectConfigBytes(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrSecretStorage, err)
		}
		// Verify storage before allowing the atomic file replacement.
		check, err := unprotectConfigBytes(protected)
		defer clear(check)
		if err != nil || !bytes.Equal(raw, check) {
			return nil, fmt.Errorf("%w: protection round-trip validation failed", ErrSecretStorage)
		}
		clearSecretFields(&disk.Config)
		disk.Protected = protected
	}
	raw, err := json.MarshalIndent(disk, "", "  ")
	if len(raw) >= maxConfigBytes { // include the trailing newline in the read limit
		return nil, errors.New("config exceeds size limit")
	}
	return append(raw, '\n'), err
}

func decodeConfig(raw []byte) (*Config, error) {
	if len(raw) > maxConfigBytes {
		return nil, errors.New("config exceeds size limit")
	}
	var disk diskConfig
	if err := json.Unmarshal(raw, &disk); err != nil {
		return nil, err
	}
	if disk.SchemaVersion > CurrentSchemaVersion {
		return nil, errors.New("configuration was written by a newer unsupported schema")
	}
	if disk.Protected != nil {
		if hasSecrets(&disk.Config) {
			return nil, fmt.Errorf("%w: mixed plaintext and protected material", ErrSecretStorage)
		}
		plain, err := unprotectConfigBytes(disk.Protected)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrSecretStorage, err)
		}
		defer clear(plain)
		var secret Config
		if err := json.Unmarshal(plain, &secret); err != nil {
			return nil, fmt.Errorf("%w: invalid protected payload", ErrSecretStorage)
		}
		if secret.Wxid != disk.Wxid || secret.DBRoot != disk.DBRoot {
			return nil, fmt.Errorf("%w: account metadata differs from protected payload", ErrSecretStorage)
		}
		copySecretFields(&disk.Config, &secret)
	} else if disk.SchemaVersion >= 5 && hasSecrets(&disk.Config) {
		return nil, fmt.Errorf("%w: schema 5 material must be protected", ErrSecretStorage)
	}
	disk.Config.normalize()
	return &disk.Config, nil
}

func readConfigBytes(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxConfigBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxConfigBytes {
		return nil, errors.New("config exceeds size limit")
	}
	return b, nil
}
