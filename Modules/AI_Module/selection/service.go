package selection

import (
	"context"
	"errors"
	"fmt"

	selectioncontract "pride-ai-module/contracts/selection"
)

// Service manages the currently selected AI configuration.
type Service struct {
	store selectioncontract.SelectionStore
}

// NewService creates a configuration selection service.
func NewService(
	store selectioncontract.SelectionStore,
) *Service {
	return &Service{
		store: store,
	}
}

// GetSelected returns the currently selected configuration.
func (s *Service) GetSelected(
	ctx context.Context,
) (selectioncontract.ConfigurationSelection, error) {
	if err := ctx.Err(); err != nil {
		return selectioncontract.ConfigurationSelection{}, err
	}

	if s == nil {
		return selectioncontract.ConfigurationSelection{}, errors.New(
			"selection service must not be nil",
		)
	}

	if s.store == nil {
		return selectioncontract.ConfigurationSelection{}, errors.New(
			"selection store must not be nil",
		)
	}

	selection, err := s.store.Get(ctx)
	if err != nil {
		return selectioncontract.ConfigurationSelection{}, fmt.Errorf(
			"get selected configuration: %w",
			err,
		)
	}

	return selection, nil
}

// Select changes the currently selected configuration.
func (s *Service) Select(
	ctx context.Context,
	configurationRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil {
		return errors.New(
			"selection service must not be nil",
		)
	}

	if s.store == nil {
		return errors.New(
			"selection store must not be nil",
		)
	}

	if configurationRef == "" {
		return ErrConfigurationRefEmpty
	}

	selection := selectioncontract.ConfigurationSelection{
		ConfigurationRef: configurationRef,
	}

	if err := s.store.Set(ctx, selection); err != nil {
		return fmt.Errorf(
			"set selected configuration: %w",
			err,
		)
	}

	return nil
}

// Compile-time contract verification.
var _ selectioncontract.ConfigurationSelector = (*Service)(nil)
