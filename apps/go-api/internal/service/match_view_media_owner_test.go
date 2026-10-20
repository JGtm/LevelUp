package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// L'onglet Médias porte le propriétaire de chaque média (le front n'offre la réassociation
// que sur les siens) ; un propriétaire inconnu reste absent.
func TestBuildMediaTab_CarriesTheOwner(t *testing.T) {
	tab := buildMediaTab([]domain.MediaAssocRaw{
		{FileID: "a", OwnerSlug: "Moi"},
		{FileID: "b"},
	})
	if got := tab.MediaItems[0].OwnerGamertag; got == nil || *got != "Moi" {
		t.Fatalf("propriétaire = %v, attendu Moi", got)
	}
	if got := tab.MediaItems[1].OwnerGamertag; got != nil {
		t.Fatalf("propriétaire inconnu = %q, attendu absent", *got)
	}
}
