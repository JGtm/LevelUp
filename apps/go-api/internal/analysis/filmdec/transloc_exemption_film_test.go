package filmdec

// transloc_exemption_film_test.go — VALIDATION SUR PIÈCES de l'exemption D2 (P1.4) : sur le
// film Dynasty, les échantillons que le filtre de vitesse rejetait autour des téléportations
// datées par l'événement 117 sont ré-acceptés, et TOUTE ré-acceptation tombe dans une
// fenêtre ±200 ms d'un événement du même slot ; sur un film SANS tête 117, le décodage est
// bit à bit identique à l'actuel.
//
// Gardés par P1_FILM / P1_FILM_SANS117 (les films ne sont pas versionnés), skip par défaut.
//
// LE SCAN Y PASSE `nil` COMME ENTRÉE DE CATALOGUE, et c'est délibéré : l'exemption ne consomme
// que l'INSTANT et le SLOT des événements. Les positions de la charge sont validées à part
// (transloc_positions_film_test.go), sur le film et la carte du cas index.
//
//	CGO_ENABLED=0 P1_FILM=<depot>/data/cache/film_chunks/1b2d9e08 \
//	  go test ./internal/analysis/filmdec/ -run '^TestP1Exemption' -v -timeout 30m
//	CGO_ENABLED=0 P1_FILM_SANS117=<depot>/data/cache/film_chunks/7344d24f \
//	  go test ./internal/analysis/filmdec/ -run '^TestP1Invariance' -v -timeout 30m

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

const p1FilmSans117Env = "P1_FILM_SANS117"

// p1PseudoWorld projette des quanta en pseudo-mètres à l'échelle du canevas Forge
// (axisWidths 15/15/17 sur les bornes fo08 : ~1,4 cm par quantum). Un filtre en m/s
// comparable à la production sans dépendre du catalogue : l'ÉCART entre deux politiques du
// filtre ne dépend que des instants exemptés, pas de l'échelle exacte.
func p1PseudoWorld(raw []BipedPosition) []BipedPosition {
	pos := make([]BipedPosition, len(raw))
	for i, p := range raw {
		pos[i] = p
		pos[i].HasWorld = true
		pos[i].X = float32(p.Q[0]) * 0.0141
		pos[i].Y = float32(p.Q[1]) * 0.0138
		pos[i].Z = float32(p.Q[2]) * 0.0091
	}
	return pos
}

// p1RawPositions décode les positions SANS filtre de vitesse (QuantaOnly : aucune borne de
// carte requise, les filtres en m/s y sont inopérants par contrat).
func p1RawPositions(t *testing.T, dir string) []BipedPosition {
	t.Helper()
	scan := DefaultScanFilmOptions()
	scan.QuantaOnly = true
	raw, err := ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	return p1PseudoWorld(raw)
}

// TestP1ExemptionVitesseDynasty mesure l'effet de l'exemption sur les positions BRUTES du
// cas index : des échantillons sont ré-acceptés, et TOUS tombent à ±200 ms d'un événement
// 117 de leur slot — la seule porte que D2 ouvre.
func TestP1ExemptionVitesseDynasty(t *testing.T) {
	dir := os.Getenv(p1FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : validation sur pièces sautée", p1FilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	evts := ScanFilmTranslocatorTeleports(dir, nil)
	for _, e := range evts {
		t.Logf("EVENEMENT 117 slot %d @%dus", e.Slot, e.TimestampUS)
	}
	if strings.Contains(dir, "1b2d9e08") && len(evts) != 3 {
		t.Fatalf("%d tête(s) 117, attendu 3 sur le cas index (R1 §4.1)", len(evts))
	}
	pos := p1RawPositions(t, dir)
	sans := DropTeleports(pos, DefaultMaxSpeedMPS)
	exempt := TeleportExemptionsOf(evts)
	avec := DropTeleportsExcept(pos, DefaultMaxSpeedMPS, exempt)
	t.Logf("FILTRE : %d échantillons bruts, %d acceptés sans exemption, %d avec (%+d)",
		len(pos), len(sans), len(avec), len(avec)-len(sans))
	if len(avec) < len(sans) {
		t.Fatalf("l'exemption ne peut que ré-accepter des échantillons : %d -> %d", len(sans), len(avec))
	}
	dans := map[BipedPosition]bool{}
	for _, p := range sans {
		dans[p] = true
	}
	nouveaux, horsFenetre := 0, 0
	for _, p := range avec {
		if dans[p] {
			continue
		}
		nouveaux++
		if exempt.covers(p.Slot, p.TimestampUS) {
			t.Logf("RE-ACCEPTE slot %d @%dus (dans une fenêtre ±200 ms)", p.Slot, p.TimestampUS)
			continue
		}
		horsFenetre++
		t.Errorf("échantillon ré-accepté HORS fenêtre : slot %d @%dus — l'exemption a débordé",
			p.Slot, p.TimestampUS)
	}
	t.Logf("EXEMPTION : %d échantillon(s) ré-accepté(s), %d hors fenêtre ±200 ms", nouveaux, horsFenetre)
	if strings.Contains(dir, "1b2d9e08") && nouveaux == 0 {
		// R3 §2 : 3 + 1 + 3 = 7 rejets à tort mesurés sur les trois téléportations du film.
		t.Error("aucun échantillon ré-accepté : l'exemption n'a pas joué sur le cas index")
	}
}

// TestP1InvarianceSansTete117 prouve l'invariance D2 sur pièces : zéro tête 117, donc zéro
// fenêtre, donc un décodage IDENTIQUE BIT À BIT à la sémantique du schéma 37 — l'oracle est
// l'implémentation de RÉFÉRENCE figée (dropTeleportsReference, offline_filters_test.go),
// jamais la production : depuis le lot P1, DropTeleports délègue à DropTeleportsExcept, et
// se comparer à soi-même ne prouverait rien (revue ronde 1, F6).
func TestP1InvarianceSansTete117(t *testing.T) {
	dir := os.Getenv(p1FilmSans117Env)
	if dir == "" {
		t.Skipf("%s absent : invariance sur pièces sautée", p1FilmSans117Env)
	}
	release := LockProcessDecode()
	defer release()
	evts := ScanFilmTranslocatorTeleports(dir, nil)
	if len(evts) != 0 {
		t.Fatalf("%d tête(s) 117 sur le film témoin — choisir un film SANS translocateur"+
			" (R3 : 696a9d7c, 7344d24f)", len(evts))
	}
	pos := p1RawPositions(t, dir)
	reference := dropTeleportsReference(pos, DefaultMaxSpeedMPS)
	avec := DropTeleportsExcept(pos, DefaultMaxSpeedMPS, TeleportExemptionsOf(evts))
	if !reflect.DeepEqual(reference, avec) {
		t.Fatalf("le filtre exempté diffère de la sémantique de RÉFÉRENCE (schéma 37) sur un"+
			" film sans tête 117 : %d contre %d échantillons — l'invariance D2 est rompue",
			len(avec), len(reference))
	}
	t.Logf("INVARIANCE : %d échantillons, décodage bit à bit identique à la référence figée"+
		" (0 tête 117)", len(reference))
}
