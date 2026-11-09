// Package api — post_sync_medal_first_earned_test.go : tests de l'émission
// « médaille inédite » (V72-20) via un résolveur de noms factice + recordingEmitter.
package wire

import (
	"context"
	"encoding/json"
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// fakeMedalNamer : résolveur factice. Capture l'appel (ids) et renvoie les noms
// programmés — permet d'asserter que le récap ne résout PAS les noms et que les
// ids inconnus retombent sur une paire vide (médaille sautée).
type fakeMedalNamer struct {
	names  map[int64]i18n.Text
	called bool
	gotIDs []int64
}

func (f *fakeMedalNamer) MedalNames(_ context.Context, ids []int64) map[int64]i18n.Text {
	f.called = true
	f.gotIDs = ids
	out := make(map[int64]i18n.Text, len(ids))
	for _, id := range ids {
		if p, ok := f.names[id]; ok {
			out[id] = p
		}
	}
	return out
}

// medalSet construit un ensemble de medal_name_id pour un PlayerSnapshot.
func medalSet(ids ...int64) map[int64]struct{} {
	m := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func TestEmitMedalFirstEarned_NewMedalEmitsNamedNotification(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	namer := &fakeMedalNamer{names: map[int64]i18n.Text{3: i18n.FromColumns("Perfect", "Tir de précision")}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "player-slug", before, after)

	if len(em.emitted) != 1 {
		t.Fatalf("attendu 1 notification émise, obtenu %d", len(em.emitted))
	}
	in := em.emitted[0]
	if in.Category != notifications.CategoryMedalFirstEarned {
		t.Errorf("catégorie attendue %q, obtenue %q", notifications.CategoryMedalFirstEarned, in.Category)
	}
	if in.Severity != notifications.SeveritySuccess {
		t.Errorf("médaille inédite → severity success attendue, obtenue %q", in.Severity)
	}
	if in.TitleKey != "notif.medal_first_earned.title" {
		t.Errorf("title_key nominatif attendu, obtenu %q", in.TitleKey)
	}
	if in.Params[medalNameKey(i18n.LocaleFR)] != "Tir de précision" {
		t.Errorf("medal_name_fr attendu, obtenu %v", in.Params[medalNameKey(i18n.LocaleFR)])
	}
	if in.Params[medalNameKey(i18n.LocaleEN)] != "Perfect" {
		t.Errorf("medal_name_en attendu, obtenu %v", in.Params[medalNameKey(i18n.LocaleEN)])
	}
	if in.TargetRoute != "/t/halo_infinite/players/player-slug/career/medals" {
		t.Errorf("target_route page Médailles attendue, obtenue %q", in.TargetRoute)
	}
	if !targetRouteIsValid(in.TargetRoute) {
		t.Errorf("target_route hors whitelist : %q", in.TargetRoute)
	}
	if in.Source != postSyncSource {
		t.Errorf("source attendue %q, obtenue %q", postSyncSource, in.Source)
	}
}

// Garde cold-start : before sans aucune médaille → seed silencieux, aucune notif,
// et le résolveur de noms n'est jamais appelé (pas de burst historique).
func TestEmitMedalFirstEarned_ColdStartSeedsSilently(t *testing.T) {
	before := &PlayerSnapshot{} // EarnedMedalIDs nil → froid
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3, 4, 5)}
	namer := &fakeMedalNamer{}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("cold-start doit semer sans notifier, obtenu %d", len(em.emitted))
	}
	if namer.called {
		t.Error("cold-start ne doit pas résoudre de noms de médailles")
	}
}

// Anti-rafale : plus de 3 médailles inédites dans un cycle → 1 seule notification
// récapitulative avec le compte, sans résolution de noms individuels.
func TestEmitMedalFirstEarned_RecapAboveThreshold(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3, 4, 5)} // 4 nouvelles > 3
	namer := &fakeMedalNamer{}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 1 {
		t.Fatalf("anti-rafale → 1 notification récapitulative attendue, obtenu %d", len(em.emitted))
	}
	in := em.emitted[0]
	if in.TitleKey != "notif.medal_first_earned.recap.title" {
		t.Errorf("title_key récap attendu, obtenu %q", in.TitleKey)
	}
	if in.Params[paramKeyCount] != 4 {
		t.Errorf("count récap attendu 4, obtenu %v", in.Params[paramKeyCount])
	}
	if namer.called {
		t.Error("récap ne doit pas résoudre les noms individuels")
	}
}

// Au seuil exact (3 inédites) → une notification nominative par médaille.
func TestEmitMedalFirstEarned_AtThresholdEmitsPerMedal(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3, 4)} // 3 nouvelles = seuil
	namer := &fakeMedalNamer{names: map[int64]i18n.Text{
		2: i18n.FromColumns("A", "A"), 3: i18n.FromColumns("B", "B"), 4: i18n.FromColumns("C", "C"),
	}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if n := countCategory(em.emitted, notifications.CategoryMedalFirstEarned); n != 3 {
		t.Fatalf("3 inédites (= seuil) → 3 notifs nominatives, obtenu %d", n)
	}
}

func TestEmitMedalFirstEarned_NoChangeNoEmit(t *testing.T) {
	snap := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, &fakeMedalNamer{}, "halo_infinite", "p", snap, snap)

	if len(em.emitted) != 0 {
		t.Fatalf("aucune médaille nouvelle → 0 émission, obtenu %d", len(em.emitted))
	}
}

// Nom introuvable dans les DEUX langues → médaille sautée (jamais d'id brut), les
// autres médailles nommées du même cycle passent.
func TestEmitMedalFirstEarned_UnresolvedNameSkipped(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)} // 2 nouvelles ≤ 3
	namer := &fakeMedalNamer{names: map[int64]i18n.Text{2: i18n.FromColumns("Resolved", "Résolue")}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 1 {
		t.Fatalf("seule la médaille nommée doit émettre, obtenu %d", len(em.emitted))
	}
	if em.emitted[0].Params[medalNameKey(i18n.LocaleFR)] != "Résolue" {
		t.Errorf("médaille nommée attendue, obtenu %v", em.emitted[0].Params[medalNameKey(i18n.LocaleFR)])
	}
}

// Un nom présent dans une seule langue est écrit tel quel (l'autre langue reste vide) ; la lecture
// (notifications.LocalizeParams) le réutilise pour toute langue sans nom : dégradation propre, pour fr, en
// et toute langue ajoutée plus tard.
func TestEmitMedalFirstEarned_SingleLangNameServedInEveryLanguage(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2)}
	namer := &fakeMedalNamer{names: map[int64]i18n.Text{2: i18n.FromColumns("", "Étoile")}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 1 {
		t.Fatalf("attendu 1 notification, obtenu %d", len(em.emitted))
	}
	raw, err := json.Marshal(em.emitted[0].Params)
	if err != nil {
		t.Fatalf("Marshal : %v", err)
	}
	for _, l := range i18n.Locales() {
		items := []notifications.Notification{{Params: raw}}
		notifications.LocalizeParams(context.Background(), items, l)
		var got map[string]any
		if err := json.Unmarshal(items[0].Params, &got); err != nil {
			t.Fatalf("Unmarshal : %v", err)
		}
		if got[notifications.ParamMedalName] != "Étoile" {
			t.Errorf("langue %s : medal_name = %v, attendu Étoile", l, got[notifications.ParamMedalName])
		}
		for k := range got {
			if k != notifications.ParamMedalName {
				t.Errorf("langue %s : clé inattendue %q (aucune paire par langue ne doit sortir)", l, k)
			}
		}
	}
}

// medalNameKey est la clé de stockage du nom de la médaille dans la langue l.
func medalNameKey(l i18n.Locale) string {
	return i18n.ParamKey(notifications.ParamMedalName, l)
}

// Émetteur en échec sur la catégorie médaille → best-effort, aucune panique, la
// boucle poursuit (rien d'enregistré car les deux émissions échouent).
func TestEmitMedalFirstEarned_EmitErrorIsBestEffort(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	namer := &fakeMedalNamer{names: map[int64]i18n.Text{
		2: i18n.FromColumns("A", "A"), 3: i18n.FromColumns("B", "B"),
	}}
	em := &recordingEmitter{failOn: notifications.CategoryMedalFirstEarned}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("émetteur en échec → 0 notification enregistrée, obtenu %d", len(em.emitted))
	}
}

func TestNewMedalNamerForPDB_NilReturnsNil(t *testing.T) {
	if n := newMedalNamerForPDB(nil); n != nil {
		t.Error("pdb nil → résolveur nil attendu")
	}
	if n := newMedalNamerForPDB(&duckdb.PlayerDB{}); n != nil {
		t.Error("pdb.Metadata nil → résolveur nil attendu")
	}
}

func TestEmitMedalFirstEarned_NilGuards(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2)}

	// Résolveur nil (pdb invalide) → no-op, aucune panique.
	emitMedalFirstEarned(context.Background(), em, nil, "halo_infinite", "p", before, after)
	// Émetteur nil → no-op.
	emitMedalFirstEarned(context.Background(), nil, &fakeMedalNamer{}, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("gardes nil → 0 émission, obtenu %d", len(em.emitted))
	}
}
