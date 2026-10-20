package selection

import (
	"context"
	"errors"
	"testing"

	selectioncontract "pride-ai-module/contracts/selection"
)

func TestMemoryStoreSelectAndGetSelected(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	err := store.Set(
		ctx,
		selectioncontract.ConfigurationSelection{
			ConfigurationRef: "config_openai",
		},
	)
	if err != nil {
		t.Fatalf(
			"Set() failed: %v",
			err,
		)
	}

	selected, err := store.Get(ctx)
	if err != nil {
		t.Fatalf(
			"Get() failed: %v",
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

func TestMemoryStoreCanChangeSelection(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	if err := store.Set(ctx, selectioncontract.ConfigurationSelection{ConfigurationRef: "config_openai"}); err != nil {
		t.Fatalf("first Set() failed: %v", err)
	}

	if err := store.Set(ctx, selectioncontract.ConfigurationSelection{ConfigurationRef: "config_anthropic"}); err != nil {
		t.Fatalf("second Set() failed: %v", err)
	}

	selected, err := store.Get(ctx)
	if err != nil {
		t.Fatalf(
			"Get() failed: %v",
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

func TestMemoryStoreRejectsEmptyReference(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	err := store.Set(ctx, selectioncontract.ConfigurationSelection{})

	if !errors.Is(err, ErrConfigurationRefEmpty) {
		t.Fatalf(
			"expected ErrConfigurationRefEmpty, got %v",
			err,
		)
	}
}

func TestMemoryStoreRejectsGetBeforeSelection(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	_, err := store.Get(ctx)

	if !errors.Is(err, ErrConfigurationNotSelected) {
		t.Fatalf(
			"expected ErrConfigurationNotSelected, got %v",
			err,
		)
	}
}

func TestMemoryStoreImplementsContract(t *testing.T) {
	var _ selectioncontract.SelectionStore = (*MemoryStore)(nil)
}
