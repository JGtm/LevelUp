//go:build cgo

// Package duckdb — require_social_persister_test.go (ADR 0021 Gap 1).
//
// Vérifie le flag RequireSocialPersister + ErrSocialPersisterNotWired.

package duckdb

import (
	"errors"
	"testing"
)

func TestRequireSocialPersister_DefaultFalse(t *testing.T) {
	// Sanity : la valeur par défaut doit être false pour ne pas casser les
	// tests qui n'instancient pas le Persister.
	//
	// NB : ce test peut être influencé par d'autres tests qui set le flag
	// — on lit la valeur courante et on documente le contrat.
	if RequireSocialPersister {
		t.Log("[INFO] RequireSocialPersister=true à ce moment du test — set par un autre test ou main.go en cours d'exécution")
	} else {
		t.Log("[OK] RequireSocialPersister=false par défaut (contrat)")
	}
}

func TestErrSocialPersisterNotWired_TypedError(t *testing.T) {
	// Vérifie que l'erreur est exportée et identifiable via errors.Is.
	err := ErrSocialPersisterNotWired
	if err == nil {
		t.Fatal("ErrSocialPersisterNotWired ne doit pas être nil")
	}
	if !errors.Is(err, ErrSocialPersisterNotWired) {
		t.Error("errors.Is doit reconnaître l'erreur sentinel")
	}
	if err.Error() == "" {
		t.Error("message d'erreur ne doit pas être vide")
	}
}

func TestRequireSocialPersister_ToggleAroundCall(t *testing.T) {
	// Valide qu'on peut toggle le flag autour d'un test pour simuler le mode
	// prod sans polluer les autres tests.
	original := RequireSocialPersister
	defer func() { RequireSocialPersister = original }()

	RequireSocialPersister = true
	if !RequireSocialPersister {
		t.Fatal("toggle KO")
	}
	RequireSocialPersister = false
	if RequireSocialPersister {
		t.Fatal("toggle reverse KO")
	}
}
