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

// TestApplyPlaylistTranslationsFR couvre la résolution FR des playlists par
// IDENTIFIANT de l'indice escouade (V72-10 suite) : une playlist dont le libellé
// COALESCE(playlist_name_fr, playlist_name) est resté en EN — playlist_name_fr
// vide dans match_registry, trou de données réel pour « Quick Play » / « Big
// Team Battle » — est néanmoins servie en FR quand son playlist_id est traduit
// (metadata.asset_translations, même résolveur — ResolveAssetNamesBulk — que la
// page Carrière). Priorité : traduction par id > libellé COALESCE existant
// (jamais vide).
func TestApplyPlaylistTranslationsFR(t *testing.T) {
	ctx := context.Background()
	idByLabel := map[string]string{
		"Quick Play":      "pl-quickplay-uuid",
		"Big Team Battle": "pl-btb-uuid",
	}

	// Nominal : "Quick Play" (name_fr vide, trou de données) traduit par id ;
	// "Big Team Battle" absent de la réponse du traducteur → COALESCE conservé.
	tr := func(_ context.Context, ids []string) (map[string]string, error) {
		out := map[string]string{}
		for _, id := range ids {
			if id == "pl-quickplay-uuid" {
				out[id] = "Partie rapide"
			}
		}
		return out, nil
	}
	p := (&PrestigeSquadMatchProvider{}).WithPlaylistTranslatorFR(tr)
	got := p.applyPlaylistTranslationsFR(ctx, []string{"Quick Play", "Big Team Battle"}, idByLabel)
	if len(got) != 2 || got[0] != "Partie rapide" || got[1] != "Big Team Battle" {
		t.Fatalf("applyPlaylistTranslationsFR = %v, want [Partie rapide Big Team Battle]", got)
	}

	// Traducteur nil → libellés inchangés (indice servi tel quel plutôt qu'absent).
	if got := (&PrestigeSquadMatchProvider{}).applyPlaylistTranslationsFR(ctx, []string{"Quick Play"}, idByLabel); len(got) != 1 || got[0] != "Quick Play" {
		t.Errorf("nil translator = %v, want [Quick Play]", got)
	}

	// Erreur du traducteur → libellés inchangés (best-effort, jamais vide).
	pErr := (&PrestigeSquadMatchProvider{}).WithPlaylistTranslatorFR(
		func(context.Context, []string) (map[string]string, error) { return nil, errors.New("metadata absente") },
	)
	if got := pErr.applyPlaylistTranslationsFR(ctx, []string{"Quick Play"}, idByLabel); len(got) != 1 || got[0] != "Quick Play" {
		t.Errorf("translator error = %v, want [Quick Play]", got)
	}

	// Traduction blanche → libellé COALESCE conservé (jamais vidé).
	pBlank := (&PrestigeSquadMatchProvider{}).WithPlaylistTranslatorFR(
		func(_ context.Context, ids []string) (map[string]string, error) {
			out := map[string]string{}
			for _, id := range ids {
				out[id] = "  "
			}
			return out, nil
		},
	)
	if got := pBlank.applyPlaylistTranslationsFR(ctx, []string{"Quick Play"}, idByLabel); len(got) != 1 || got[0] != "Quick Play" {
		t.Errorf("blank FR = %v, want [Quick Play]", got)
	}

	// Aucun id connu pour le libellé (idByLabel vide) → libellé conservé, pas de
	// traduction inutile demandée (aucun id à résoudre).
	if got := p.applyPlaylistTranslationsFR(ctx, []string{"Ranked Arena"}, map[string]string{}); len(got) != 1 || got[0] != "Ranked Arena" {
		t.Errorf("no known id = %v, want [Ranked Arena]", got)
	}
}

// TestPlaylistTally_CountsByIDNotLabel : régression contre-revue V7.2. Le top 2
// des playlists de l'indice escouade doit être compté par playlist_id, pas par
// libellé — sinon deux libellés désignant la MÊME playlist (renommage saisonnier,
// name_fr renseigné sur une partie des lignes seulement) produisent deux entrées
// sous-comptées, et la playlist réellement dominante peut sortir du top 2.
func TestPlaylistTally_CountsByIDNotLabel(t *testing.T) {
	const (
		idQuickPlay = "aaaaaaaa-0000-0000-0000-000000000001"
		idBTB       = "bbbbbbbb-0000-0000-0000-000000000002"
	)
	tally := newPlaylistTally()
	// Même playlist (même id) sous 2 libellés : 3 + 2 = 5 occurrences.
	for i := 0; i < 3; i++ {
		tally.add(idQuickPlay, "Quick Play")
	}
	for i := 0; i < 2; i++ {
		tally.add(idQuickPlay, "Partie rapide")
	}
	// Playlist concurrente : 4 occurrences — devancerait chaque moitié éclatée.
	for i := 0; i < 4; i++ {
		tally.add(idBTB, "Big Team Battle")
	}

	labels, idByLabel := tally.top(2)
	if len(labels) != 2 {
		t.Fatalf("top(2) = %v, want 2 entrées", labels)
	}
	if labels[0] != "Quick Play" {
		t.Errorf("playlist dominante = %q, want \"Quick Play\" (5 occurrences agrégées par id, "+
			"pas 3 et 2 éclatées sous deux libellés)", labels[0])
	}
	if labels[1] != "Big Team Battle" {
		t.Errorf("2e playlist = %q, want \"Big Team Battle\"", labels[1])
	}
	// Le libellé représentatif est le PREMIER vu, et il porte son id pour la
	// traduction FR par identifiant.
	if idByLabel["Quick Play"] != idQuickPlay {
		t.Errorf("idByLabel[Quick Play] = %q, want %q", idByLabel["Quick Play"], idQuickPlay)
	}
	if idByLabel["Big Team Battle"] != idBTB {
		t.Errorf("idByLabel[Big Team Battle] = %q, want %q", idByLabel["Big Team Battle"], idBTB)
	}
}

// TestPlaylistTally_FallbackOnLabelWithoutID : sans playlist_id (registre
// incomplet), le libellé reste la seule identité disponible — comptage par
// libellé, et aucun id exposé pour la traduction (le COALESCE existant est alors
// conservé tel quel par applyPlaylistTranslationsFR).
func TestPlaylistTally_FallbackOnLabelWithoutID(t *testing.T) {
	tally := newPlaylistTally()
	tally.add("", "Fiesta")
	tally.add("", "Fiesta")
	tally.add("", "Husky Raid")

	labels, idByLabel := tally.top(2)
	if len(labels) != 2 || labels[0] != "Fiesta" || labels[1] != "Husky Raid" {
		t.Fatalf("top(2) = %v, want [Fiesta Husky Raid]", labels)
	}
	if len(idByLabel) != 0 {
		t.Errorf("aucun id connu → idByLabel doit rester vide, got %v", idByLabel)
	}
}
