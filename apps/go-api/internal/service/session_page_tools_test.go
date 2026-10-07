package service

// session_page_tools_test.go — « OUTILS DE DESTRUCTION » DE LA SESSION (plan
// `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, S2.6) : le builder de l'Escouade sur le seul joueur de
// la page, avec les catégories de source du film quand le lecteur d'armes les sert.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

const keyOutilBobine = "hinf_" + "coil_plasma"

// armesDeSession — le lecteur d'armes de la session, avec (ou sans) l'interface optionnelle des
// catégories de source du film.
type armesDeSession struct {
	rows []port.WeaponKillRow
}

func (a armesDeSession) LoadWeaponKillsAggregated(context.Context, string, port.WeaponKillFilters) ([]port.WeaponKillRow, error) {
	return a.rows, nil
}

type armesEtCategories struct {
	armesDeSession
	categories []port.KillSourceCategoryRow
	err        error
	filtres    port.WeaponKillFilters
}

func (a *armesEtCategories) LoadKillSourceCategoryKills(_ context.Context, _ string, f port.WeaponKillFilters) ([]port.KillSourceCategoryRow, error) {
	a.filtres = f
	return a.categories, a.err
}

// lignesArmesDeSession : 4 frags au BR75 et une bobine (objet explosif) lus au film.
func lignesArmesDeSession() []port.WeaponKillRow {
	return []port.WeaponKillRow{
		{XUID: "P", WeaponKey: "hinf_br75", Label: "BR75", LabelEN: "BR75", Class: "shoulder", Kills: 4, FromDamageSource: true},
		{XUID: "P", WeaponKey: keyOutilBobine, Label: "Bobine à plasma", Class: "environmental", Kills: 1, FromDamageSource: true},
	}
}

// canonDeSession : un match, 10 frags dont 1 en mêlée (feuille de match).
func canonDeSession() []canonical.PlayerMatchRow {
	kills, melee := 10, 1
	return []canonical.PlayerMatchRow{{
		Summary: canonical.MatchSummary{MatchID: "m1"},
		Self:    canonical.MatchParticipant{Kills: &kills, MeleeKills: &melee},
	}}
}

func outilsParNature(tools *domain.SquadWeaponTools) map[string]int {
	out := map[string]int{}
	for _, l := range tools.Lines {
		name := l.Kind
		if l.Kind == domain.SquadToolKindWeapon {
			name = l.Label
		}
		out[name] = l.KillsByPlayer["GT"]
	}
	return out
}

func TestAttachSessionFragDistribution_OutilsDeDestruction(t *testing.T) {
	repo := &armesEtCategories{
		armesDeSession: armesDeSession{rows: lignesArmesDeSession()},
		categories: []port.KillSourceCategoryRow{
			{XUID: "P", Category: domain.KillSourceCategoryExplosiveObject, WeaponKey: keyOutilBobine, Kills: 1},
		},
	}
	svc := &SessionPageService{titleSlug: "halo_infinite", gamertag: "GT", weaponKillsRepo: repo}
	entry := &domain.SessionCompareEntry{}
	svc.attachSessionFragDistribution(context.Background(), entry, canonDeSession(), []string{"m1"})

	if entry.WeaponTools == nil || len(entry.WeaponTools.Players) != 1 || entry.WeaponTools.Players[0] != "GT" {
		t.Fatalf("outils = %+v, attendu le seul joueur de la page", entry.WeaponTools)
	}
	got := outilsParNature(entry.WeaponTools)
	want := map[string]int{
		"BR75": 4, domain.SquadToolKindExplosiveObject: 1, domain.SquadToolKindMelee: 1,
		domain.SquadToolKindUnattributed: 4,
	}
	if len(got) != len(want) {
		t.Errorf("lignes = %v, attendu %v (la bobine reprise par l'objet explosif)", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %d, attendu %d", k, got[k], v)
		}
	}
	last := entry.WeaponTools.Lines[len(entry.WeaponTools.Lines)-1]
	if last.Kind != domain.SquadToolKindUnattributed {
		t.Errorf("dernière ligne = %+v, attendu « Non attribué »", last)
	}
	if repo.filtres.Gamertag != "GT" || len(repo.filtres.MatchIDs) != 1 {
		t.Errorf("filtres des catégories = %+v, attendu le scope de la session et le joueur", repo.filtres)
	}
}

// Sans l'interface des catégories (titre sans film) ou catégories non supportées / en échec :
// la bobine reste une ligne d'arme, rien n'est perdu (le reliquat reste juste).
func TestAttachSessionFragDistribution_OutilsSansCategories(t *testing.T) {
	for _, repo := range []port.WeaponKillsRepository{
		armesDeSession{rows: lignesArmesDeSession()},
		&armesEtCategories{armesDeSession: armesDeSession{rows: lignesArmesDeSession()}, err: games.ErrCapabilityNotSupported},
		&armesEtCategories{armesDeSession: armesDeSession{rows: lignesArmesDeSession()}, err: errors.New("vue absente")},
	} {
		svc := &SessionPageService{titleSlug: "halo_infinite", gamertag: "GT", weaponKillsRepo: repo}
		entry := &domain.SessionCompareEntry{}
		svc.attachSessionFragDistribution(context.Background(), entry, canonDeSession(), []string{"m1"})
		if entry.WeaponTools == nil {
			t.Fatalf("%T : outils absents", repo)
		}
		got := outilsParNature(entry.WeaponTools)
		if got["Bobine à plasma"] != 1 || got[domain.SquadToolKindExplosiveObject] != 0 || got[domain.SquadToolKindUnattributed] != 4 {
			t.Errorf("%T : lignes = %v, attendu la bobine en ligne d'arme et 4 non attribués", repo, got)
		}
	}
}
