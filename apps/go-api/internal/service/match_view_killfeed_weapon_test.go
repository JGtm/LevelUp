package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ptrS(s string) *string  { return &s }
func ptrI64k(v int64) *int64 { return &v }
func ptrIk(v int) *int       { return &v }

// feedFixture : deux tueurs de deux équipes, trois kills et un event non-kill.
func feedFixture() []domain.MatchHighlightEvent {
	return []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(1000), ActorXUID: ptrS("A")},
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(2000), ActorXUID: ptrS("B")},
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(3000), ActorXUID: ptrS("A")},
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(4000), ActorXUID: ptrS("A")},
	}
}

func feedScoreboard() []domain.ScoreboardRaw {
	return []domain.ScoreboardRaw{
		{XUID: "A", TeamID: ptrIk(0)},
		{XUID: "B", TeamID: ptrIk(1)},
		{XUID: "C"}, // sans team_id : ne doit rien casser
	}
}

// TestDecorateKillFeed_PoseArmeEtEquipe : le chemin nominal. L'arme n'arrive que sur les
// kills appariés, l'équipe arrive sur TOUS les events dont l'acteur est au scoreboard.
func TestDecorateKillFeed_PoseArmeEtEquipe(t *testing.T) {
	events := feedFixture()
	sources := []domain.KillSourceRaw{
		{XUID: "A", TimeMS: 1000, SourceTag: 0x11},
		{XUID: "B", TimeMS: 2000, SourceTag: 0x22},
		// pas de source pour (A, 3000) : trou assumé
	}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{
		0x11: {WeaponKey: "hinf_br75", Label: "BR75", ImageURL: "/static/x/killfeed-00.png", Tinted: true},
		0x22: {Label: "", ImageURL: "/static/x/killfeed-65.png", Tinted: true}, // melee : sans nom propre
	}}

	decorateKillFeed(context.Background(), events, sources, feedScoreboard(), adapter)

	if events[0].WeaponKey != "hinf_br75" || events[0].WeaponLabel != "BR75" {
		t.Errorf("kill 0 : arme = %q/%q, attendu hinf_br75/BR75", events[0].WeaponKey, events[0].WeaponLabel)
	}
	if !events[0].WeaponImageTinted || events[0].WeaponImageURL == "" {
		t.Errorf("kill 0 : image = %q tinted=%v", events[0].WeaponImageURL, events[0].WeaponImageTinted)
	}
	if events[1].WeaponImageURL == "" || events[1].WeaponLabel != "" {
		t.Errorf("kill 1 (melee) : doit avoir une image sans nom propre, got %q/%q",
			events[1].WeaponImageURL, events[1].WeaponLabel)
	}
	if events[2].WeaponImageURL != "" {
		t.Errorf("kill 2 : sans source appariée, il ne doit PAS avoir d'image (got %q)", events[2].WeaponImageURL)
	}
	// L'équipe est posée sur tous les events, kill ou non — c'est elle qui colore le nom.
	for i, want := range []int{0, 1, 0, 0} {
		if events[i].ActorTeamID == nil || *events[i].ActorTeamID != want {
			t.Errorf("event %d : team_id = %v, attendu %d", i, events[i].ActorTeamID, want)
		}
	}
}

// TestDecorateKillFeed_AucuneArmeSurUnEventNonKill verrouille la règle qui évite le
// non-sens : une médaille ou un event de mode n'a pas d'arme, même si son acteur a tué à
// cet instant précis.
func TestDecorateKillFeed_AucuneArmeSurUnEventNonKill(t *testing.T) {
	events := []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(1000), ActorXUID: ptrS("A")},
	}
	sources := []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 0x11}}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{
		0x11: {ImageURL: "/static/x/killfeed-00.png"},
	}}

	decorateKillFeed(context.Background(), events, sources, feedScoreboard(), adapter)

	if events[0].WeaponImageURL != "" {
		t.Errorf("event medal : arme posée (%q) alors qu'il n'en a pas", events[0].WeaponImageURL)
	}
}

// TestDecorateKillFeed_SourceSansIconeResteSansArme : le cas des sources identifiées mais
// non traduisibles en image (véhicule, bidon, chute, nom alternatif contradictoire). Le
// pont rend faux, et RIEN ne doit être posé — surtout pas une icône par défaut.
func TestDecorateKillFeed_SourceSansIconeResteSansArme(t *testing.T) {
	events := feedFixture()
	sources := []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 0xdead}}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{}} // le pont ne connaît rien

	decorateKillFeed(context.Background(), events, sources, feedScoreboard(), adapter)

	for i, e := range events {
		if e.WeaponImageURL != "" || e.WeaponKey != "" || e.WeaponLabel != "" {
			t.Errorf("event %d : décoré (%q/%q/%q) alors que le pont n'a rien rendu",
				i, e.WeaponKey, e.WeaponLabel, e.WeaponImageURL)
		}
	}
}

// TestDecorateKillFeed_DegradationsGracieuses : chaque entrée peut manquer sans que la
// carte Dominance en souffre. C'est l'état NOMINAL d'un titre sans décodeur de film
// (Halo 5) et d'un match jamais passé au décodeur.
func TestDecorateKillFeed_DegradationsGracieuses(t *testing.T) {
	cas := []struct {
		nom        string
		sources    []domain.KillSourceRaw
		scoreboard []domain.ScoreboardRaw
		adapter    *stubAssetURL
	}{
		{"aucune source", nil, feedScoreboard(), &stubAssetURL{}},
		{"aucun scoreboard", []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 1}}, nil, &stubAssetURL{}},
		{"adapter nil", []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 1}}, feedScoreboard(), nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			events := feedFixture()
			if c.adapter == nil {
				decorateKillFeed(context.Background(), events, c.sources, c.scoreboard, nil)
			} else {
				decorateKillFeed(context.Background(), events, c.sources, c.scoreboard, c.adapter)
			}
			for i, e := range events {
				if e.WeaponImageURL != "" {
					t.Errorf("event %d : arme posée sans adapter/source valide", i)
				}
			}
		})
	}
	// Tranche vide : pas de panique, pas d'allocation.
	decorateKillFeed(context.Background(), nil, nil, nil, nil)
}

// TestKillFeedWeaponCoverage : le compteur ne regarde QUE les kills. Un feed de 4 events
// dont 3 kills et 1 médaille compte 3, jamais 4 — sinon le taux publié serait faux.
func TestKillFeedWeaponCoverage(t *testing.T) {
	events := feedFixture()
	events[0].WeaponImageURL = "/x.png"
	events[3].WeaponImageURL = "/y.png" // médaille : ne doit compter ni au numérateur ni au dénominateur
	avec, total := killFeedWeaponCoverage(events)
	if avec != 1 || total != 3 {
		t.Errorf("couverture = %d/%d, attendu 1/3", avec, total)
	}
}
