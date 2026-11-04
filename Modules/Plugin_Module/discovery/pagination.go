package discovery

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"pride-plugin-module/domain"
)

const (
	defaultPageSize int32 = 50
	maximumPageSize int32 = 100
)

type pageToken struct {
	Version int    `json:"v"`
	Digest  string `json:"d"`
	Offset  int    `json:"o"`
}

type pageSnapshot struct {
	Filter  string
	Plugins []pageSnapshotPlugin
}

type pageSnapshotPlugin struct {
	ID                  domain.PluginID
	Manifest            domain.Manifest
	Type                domain.PluginType
	State               domain.PluginState
	Capabilities        []domain.Capability
	Available           bool
	CompatibilityStatus string
	StatusMessage       string
}

func normalizePageSize(value int32) (int, error) {
	if value < 0 {
		return 0, fmt.Errorf("%w: page size must not be negative", ErrInvalidQuery)
	}
	if value == 0 {
		value = defaultPageSize
	}
	if value > maximumPageSize {
		value = maximumPageSize
	}
	return int(value), nil
}

func snapshotDigest(plugins []domain.Plugin, filter string) (string, error) {
	snapshot := pageSnapshot{
		Filter:  filter,
		Plugins: make([]pageSnapshotPlugin, 0, len(plugins)),
	}
	for _, plugin := range plugins {
		snapshot.Plugins = append(snapshot.Plugins, pageSnapshotPlugin{
			ID:                  plugin.ID,
			Manifest:            plugin.Manifest,
			Type:                plugin.Type,
			State:               plugin.State,
			Capabilities:        plugin.Protected.Capabilities,
			Available:           plugin.Available,
			CompatibilityStatus: plugin.Protected.CompatibilityStatus,
			StatusMessage:       plugin.StatusMessage,
		})
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("build discovery snapshot: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func decodePageToken(token, digest string, total int) (int, error) {
	if token == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("%w: malformed token", ErrInvalidPageToken)
	}
	var value pageToken
	if err := json.Unmarshal(decoded, &value); err != nil {
		return 0, fmt.Errorf("%w: malformed token", ErrInvalidPageToken)
	}
	if value.Version != 1 || value.Digest != digest || value.Offset <= 0 || value.Offset >= total {
		return 0, ErrInvalidPageToken
	}
	return value.Offset, nil
}

func encodePageToken(digest string, offset int) string {
	encoded, _ := json.Marshal(pageToken{Version: 1, Digest: digest, Offset: offset})
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func paginate(plugins []domain.Plugin, pageSize int, offset int, digest string) ([]domain.Plugin, string) {
	end := offset + pageSize
	if end > len(plugins) {
		end = len(plugins)
	}
	page := make([]domain.Plugin, 0, end-offset)
	for _, plugin := range plugins[offset:end] {
		page = append(page, plugin.Clone())
	}
	next := ""
	if end < len(plugins) {
		next = encodePageToken(digest, end)
	}
	return page, next
}
