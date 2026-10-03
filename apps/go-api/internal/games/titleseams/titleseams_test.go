package titleseams

// titleseams_test.go — RegisterAll pose bien les seams qu'un binaire de sync attend.
//
// Ce que ce test protège (2026-09-16) : avant RegisterAll, seul cmd/server posait les huit
// seams ; GetLUSRChain panique (fail-loud MT-15) tant que le classifier par défaut n'est pas
// posé, et le post-sync CLI rendait perf_scores=0 lusr=0 citations=0 dominance=0.

import (
	"path/filepath"
	"testing"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncskill "levelup/go-api/internal/sync/skill"
)

// TestRegisterAllCableLesClassifiers — après RegisterAll, aucun appel ne panique et chaque
// classifier rend la valeur du titre attendu.
func TestRegisterAllCableLesClassifiers(t *testing.T) {
	RegisterAll("")

	// Classifier par défaut (Halo Infinite) : un pair_name BTB donne la chaîne btb.
	if got := syncskill.GetLUSRChain("BTB:Slayer"); got != "btb" {
		t.Errorf("GetLUSRChain(\"BTB:Slayer\") = %q, attendu \"btb\" — classifier Infinite non câblé", got)
	}
	// Un pair_name inconnu retombe sur arena_slayer : la chaîne n'est jamais vide par défaut.
	if got := syncskill.GetLUSRChain("Prefixe Inconnu:Slayer"); got == "" {
		t.Error("GetLUSRChain sur un pair_name inconnu rend une chaîne vide — le fallback arena_slayer est perdu")
	}
	// Route title-aware : Halo 5 a son propre classifier (pas de pair_name).
	if got := syncskill.GetLUSRChainForTitle(halo5.TitleSlug, ""); got != halo5.LUSRChainArena {
		t.Errorf("GetLUSRChainForTitle(halo_5) = %q, attendu %q — classifier h5 non routé", got, halo5.LUSRChainArena)
	}
	// Validation de boot : le serveur refuse de démarrer sans ce câblage.
	if err := syncskill.ValidateLUSRChainClassifierWired(); err != nil {
		t.Errorf("ValidateLUSRChainClassifierWired après RegisterAll : %v", err)
	}
	// Seam famille objectif : Infinite reconnaît un sous-mode objectif, h5 répond toujours false.
	if !syncskill.IsObjectiveFamilyForTitle("", "Ranked Arena:CTF") {
		t.Error("IsObjectiveFamilyForTitle(\"\", \"Ranked Arena:CTF\") = false — classifier famille objectif Infinite non câblé")
	}
	if syncskill.IsObjectiveFamilyForTitle(halo5.TitleSlug, "peu importe") {
		t.Error("IsObjectiveFamilyForTitle(halo_5) = true — classifier h5 non routé")
	}
}

// TestRegisterAllEstIdempotent — un second appel ne casse rien (les CLI l'appellent au
// démarrage, le serveur aussi, et les TestMain des paquets qui atteignent GetLUSRChain).
func TestRegisterAllEstIdempotent(t *testing.T) {
	RegisterAll("")
	RegisterAll("")
	if got := syncskill.GetLUSRChain("BTB:Slayer"); got != "btb" {
		t.Errorf("GetLUSRChain après double RegisterAll = %q, attendu \"btb\"", got)
	}
}

// TestPrestigeConfigDir — la racine config/titles/{slug} est calculée comme dans cmd/server.
func TestPrestigeConfigDir(t *testing.T) {
	if got := PrestigeConfigDir(""); got != "" {
		t.Errorf("PrestigeConfigDir(\"\") = %q, attendu \"\"", got)
	}
	got := PrestigeConfigDir("/repo")
	want := filepath.Join("/repo", "config", "titles", "halo_infinite")
	if got != want {
		t.Errorf("PrestigeConfigDir(\"/repo\") = %q, attendu %q", got, want)
	}
}
