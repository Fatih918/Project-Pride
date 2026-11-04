package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidPlugin = errors.New("invalid plugin")
	ErrInvalidID     = errors.New("plugin ID is required")
)

func (p Plugin) Validate() error {
	if strings.TrimSpace(string(p.ID)) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidPlugin, ErrInvalidID)
	}
	if p.Manifest.PluginID != p.ID {
		return fmt.Errorf("%w: manifest plugin ID must match aggregate ID", ErrInvalidPlugin)
	}
	required := map[string]string{
		"name": p.Manifest.Name, "version": p.Manifest.Version,
		"description": p.Manifest.Description, "author": p.Manifest.Author,
		"license": p.Manifest.License, "entry_point": p.Manifest.EntryPoint,
		"minimum_pride_version": p.Manifest.MinimumPRIDEVersion,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: manifest %s is required", ErrInvalidPlugin, field)
		}
	}
	if p.Type != PluginTypeActive && p.Type != PluginTypePassive {
		return fmt.Errorf("%w: plugin type is unspecified or unknown", ErrInvalidPlugin)
	}
	if p.State < PluginStateInstalled || p.State > PluginStateError {
		return fmt.Errorf("%w: plugin state is unspecified or unknown", ErrInvalidPlugin)
	}
	seenCapabilities := make(map[string]struct{}, len(p.Protected.Capabilities))
	for _, capability := range p.Protected.Capabilities {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			return fmt.Errorf("%w: capability ID is required", ErrInvalidPlugin)
		}
		if _, exists := seenCapabilities[id]; exists {
			return fmt.Errorf("%w: duplicate capability ID %q", ErrInvalidPlugin, id)
		}
		seenCapabilities[id] = struct{}{}
	}
	seenPermissions := make(map[string]struct{}, len(p.Protected.PermissionIDs))
	for _, permissionID := range p.Protected.PermissionIDs {
		id := strings.TrimSpace(permissionID)
		if id == "" {
			return fmt.Errorf("%w: permission ID must not be empty", ErrInvalidPlugin)
		}
		if _, exists := seenPermissions[id]; exists {
			return fmt.Errorf("%w: duplicate permission ID %q", ErrInvalidPlugin, id)
		}
		seenPermissions[id] = struct{}{}
	}
	return nil
}
