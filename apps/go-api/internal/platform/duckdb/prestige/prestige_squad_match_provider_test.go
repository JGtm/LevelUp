package prestige

import (
	"context"
	"errors"
	"testing"
)

// TestTopNByFreq couvre la sélection des labels dominants de l'indice escouade :
// fréquence décroissante, départage alphabétique déterministe, bornage n.
func TestTopNByFreq(t *testing.T) {
	counts := map[string]int{"Ranked": 5, "BTB": 3, "Fiesta": 5, "Slayer": 1}
	got := topNByFreq(counts, 2)
	// Top 2 par fréquence (Fiesta=5, Ranked=5) ; tie cassé par ordre alpha.
	if len(got) != 2 || got[0] != "Fiesta" || got[1] != "Ranked" {
		t.Fatalf("topNByFreq = %v, want [Fiesta Ranked]", got)
	}

	// n > len → toutes les clés.
	all := topNByFreq(map[string]int{"a": 2}, 5)
	if len(all) != 1 || all[0] != "a" {
		t.Errorf("topNByFreq(n>len) = %v, want [a]", all)
	}

	// Map vide → slice vide.
	if empty := topNByFreq(map[string]int{}, 3); len(empty) != 0 {
		t.Errorf("topNByFreq(empty) = %v, want []", empty)
	}
}

// TestUUIDLabelRE vérifie que l'indice écarte les labels non résolus (UUID brut)
// mais garde les libellés normaux (y compris accentués).
func TestUUIDLabelRE(t *testing.T) {
	if !uuidLabelRE.MatchString("0123abcd-0123-0123-0123-0123456789ab") {
		t.Error("UUID non détecté (devrait être écarté de l'indice)")
	}
	if uuidLabelRE.MatchString("Classé") || uuidLabelRE.MatchString("Big Team Battle") {
		t.Error("libellé normal détecté à tort comme UUID")
	}
}

// TestApplyModeTranslationsFR couvre la résolution FR des modes de l'indice
// escouade (V72-10.1) : EN → FR quand mode_name_tr fournit une entrée, EN
// conservé sinon (trou de couverture, jamais vidé), et dégradation gracieuse
// (traducteur nil / en erreur / renvoyant une chaîne vide → indice en EN).
func TestApplyModeTranslationsFR(t *testing.T) {
	ctx := context.Background()

	// Nominal : Team Slayer traduit, CTF absent de la table → EN conservé.
	tr := func(_ context.Context, _ []string) (map[string]string, error) {
		return map[string]string{"Slayer": "Assassin", "Team Slayer": "Assassin par équipe"}, nil
	}
	p := (&PrestigeSquadMatchProvider{}).WithModeTranslatorFR(tr)
	got := p.applyModeTranslationsFR(ctx, []string{"Team Slayer", "CTF"})
	if len(got) != 2 || got[0] != "Assassin par équipe" || got[1] != "CTF" {
		t.Fatalf("applyModeTranslationsFR = %v, want [Assassin par équipe CTF]", got)
	}

	// Traducteur nil → modes inchangés (indice EN plutôt qu'absent).
	if got := (&PrestigeSquadMatchProvider{}).applyModeTranslationsFR(ctx, []string{"Slayer"}); len(got) != 1 || got[0] != "Slayer" {
		t.Errorf("nil translator = %v, want [Slayer]", got)
	}

	// Erreur du traducteur → modes inchangés (best-effort, jamais vide).
	pErr := (&PrestigeSquadMatchProvider{}).WithModeTranslatorFR(
		func(context.Context, []string) (map[string]string, error) { return nil, errors.New("metadata absente") },
	)
	if got := pErr.applyModeTranslationsFR(ctx, []string{"Slayer"}); len(got) != 1 || got[0] != "Slayer" {
		t.Errorf("translator error = %v, want [Slayer]", got)
	}

	// Traduction blanche → EN conservé (un libellé n'est jamais vidé).
	pBlank := (&PrestigeSquadMatchProvider{}).WithModeTranslatorFR(
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Slayer": "  "}, nil
		},
	)
	if got := pBlank.applyModeTranslationsFR(ctx, []string{"Slayer"}); len(got) != 1 || got[0] != "Slayer" {
		t.Errorf("blank FR = %v, want [Slayer]", got)
	}
}
