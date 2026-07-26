package wire

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestAppReleaseTargets_ExcludesAuthOnly — les profils auth_only (pool de tokens,
// aucune player DB) sont écartés AVANT toute tentative d'émission : c'est la source
// des 5 WARN « app_release: emit » par redémarrage post-release en production.
func TestAppReleaseTargets_ExcludesAuthOnly(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: "JGtm", TitleSlug: "halo_infinite"},
		{PlayerSlug: "QuiteSiren", TitleSlug: "halo_infinite", AuthOnly: true},
		{PlayerSlug: "UppedJoker", TitleSlug: "halo_infinite", AuthOnly: true},
		{PlayerSlug: "Madina97294", TitleSlug: "halo_infinite"},
	}

	targets, skipped := appReleaseTargets(players)

	if want := []string{"JGtm", "Madina97294"}; !reflect.DeepEqual(targets, want) {
		t.Errorf("targets = %v, attendu %v", targets, want)
	}
	if want := []string{"QuiteSiren", "UppedJoker"}; !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped = %v, attendu %v", skipped, want)
	}
}

// TestAppReleaseTargets_DedupesAcrossTitles — LoadPlayers() sans filtre renvoie une
// entrée par (titre, joueur) ; la notification est per-JOUEUR, donc un joueur
// déclaré sur 2 titres ne doit être traité qu'une fois.
func TestAppReleaseTargets_DedupesAcrossTitles(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: "JGtm", TitleSlug: "halo_infinite"},
		{PlayerSlug: "JGtm", TitleSlug: "halo_5"},
		{PlayerSlug: "Chocoboflor", TitleSlug: "halo_5"},
		{PlayerSlug: "Chocoboflor", TitleSlug: "halo_infinite"},
	}

	targets, skipped := appReleaseTargets(players)

	if want := []string{"Chocoboflor", "JGtm"}; !reflect.DeepEqual(targets, want) {
		t.Errorf("targets = %v, attendu %v", targets, want)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, attendu vide", skipped)
	}
}

// TestAppReleaseTargets_IgnoresEmptySlug — un profil sans slug n'est ni cible ni
// « ignoré auth_only » (il n'a rien à voir avec le bruit auth_only).
func TestAppReleaseTargets_IgnoresEmptySlug(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: ""},
		{PlayerSlug: "", AuthOnly: true},
		{PlayerSlug: "JGtm"},
	}

	targets, skipped := appReleaseTargets(players)

	if want := []string{"JGtm"}; !reflect.DeepEqual(targets, want) {
		t.Errorf("targets = %v, attendu %v", targets, want)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, attendu vide", skipped)
	}
}

// TestAppReleaseTargets_AuthOnlyDedupedAcrossTitles — un compte auth_only déclaré
// sur plusieurs titres n'apparaît qu'une fois dans la trace DEBUG.
func TestAppReleaseTargets_AuthOnlyDedupedAcrossTitles(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: "DankerGlue", TitleSlug: "halo_infinite", AuthOnly: true},
		{PlayerSlug: "DankerGlue", TitleSlug: "halo_5", AuthOnly: true},
	}

	targets, skipped := appReleaseTargets(players)

	if len(targets) != 0 {
		t.Errorf("targets = %v, attendu vide", targets)
	}
	if want := []string{"DankerGlue"}; !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped = %v, attendu %v", skipped, want)
	}
}

// TestAppReleaseTargets_Empty — aucun profil : aucune cible, aucun ignoré, aucun nil-panic.
func TestAppReleaseTargets_Empty(t *testing.T) {
	targets, skipped := appReleaseTargets(nil)
	if len(targets) != 0 || len(skipped) != 0 {
		t.Errorf("targets=%v skipped=%v, attendu vides", targets, skipped)
	}
}
