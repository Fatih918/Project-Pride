package selection

import (
	"context"
	"errors"
	"testing"

	selectioncontract "pride-ai-module/contracts/selection"
)

func TestServiceSelectAndGetSelected(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)

	err := service.Select(
		ctx,
		"config_openai",
	)
	if err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	selected, err := service.GetSelected(ctx)
	if err != nil {
		t.Fatalf(
			"GetSelected() failed: %v",
			err,
		)
	}

	if selected.ConfigurationRef != "config_openai" {
		t.Fatalf(
			"unexpected configuration reference: got %q, want %q",
			selected.ConfigurationRef,
			"config_openai",
		)
	}
}

func TestServiceCanChangeSelection(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)

	if err := service.Select(ctx, "config_openai"); err != nil {
		t.Fatalf(
			"first Select() failed: %v",
			err,
		)
	}

	if err := service.Select(ctx, "config_anthropic"); err != nil {
		t.Fatalf(
			"second Select() failed: %v",
			err,
		)
	}

	selected, err := service.GetSelected(ctx)
	if err != nil {
		t.Fatalf(
			"GetSelected() failed: %v",
			err,
		)
	}

	if selected.ConfigurationRef != "config_anthropic" {
		t.Fatalf(
			"selection was not updated: got %q, want %q",
			selected.ConfigurationRef,
			"config_anthropic",
		)
	}
}

func TestServiceRejectsEmptyConfigurationRef(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)

	err := service.Select(ctx, "")

	if !errors.Is(err, ErrConfigurationRefEmpty) {
		t.Fatalf(
			"expected ErrConfigurationRefEmpty, got %v",
			err,
		)
	}
}

func TestServicePropagatesStoreError(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)

	_, err := service.GetSelected(ctx)

	if !errors.Is(err, ErrConfigurationNotSelected) {
		t.Fatalf(
			"expected ErrConfigurationNotSelected, got %v",
			err,
		)
	}
}

func TestServiceRejectsNilStore(t *testing.T) {
	service := NewService(nil)

	_, err := service.GetSelected(context.Background())

	if err == nil {
		t.Fatal(
			"expected nil selection store error",
		)
	}
}

func TestServiceImplementsContract(t *testing.T) {
	var _ selectioncontract.ConfigurationSelector = (*Service)(nil)
}
