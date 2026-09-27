//go:build integration

// Package duckdb — killsource_category_kills_repo_test.go : les frags par CATEGORIE de
// source (objet explosif, chute et environnement) du lecteur adosse au film.
//
// Ce qu'ils verrouillent : un objet explosif SANS cle de registre est compte (c'est la
// raison d'etre de la lecture) ; un objet avec cle porte sa cle (l'appelant retire ces
// frags de la ligne par arme) ; une arme ordinaire n'a pas de categorie ; un titre sans
// categoriseur degrade en games.ErrCapabilityNotSupported.
//
//	go test -tags=integration -run KillSourceCategory ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// kscTagDecor : objet explosif du decor SANS cle de registre (modele destructible).
const kscTagDecor = uint32(0x0d2035aa)

// fakeKillSourceCategorizer ajoute port.KillSourceCategorizer au double de classificateur.
type fakeKillSourceCategorizer struct{ fakeKillSourceClassifier }

func (fakeKillSourceCategorizer) KillSourceCategory(tag uint32) (string, bool) {
	switch tag {
	case kscTagCoil, kscTagFusion, kscTagDecor:
		return domain.KillSourceCategoryExplosiveObject, true
	case kscTagFall:
		return domain.KillSourceCategoryEnvironment, true
	}
	return "", false
}

func TestKillSourceCategoryKills_ObjetsEtEnvironnement(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagDecor, 2000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagDecor, 3000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagFall, 4000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 5000)

	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceCategorizer{})
	got, err := repo.LoadKillSourceCategoryKills(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}})
	if err != nil {
		t.Fatalf("LoadKillSourceCategoryKills: %v", err)
	}
	want := []port.KillSourceCategoryRow{
		{XUID: kscXUID, Category: domain.KillSourceCategoryEnvironment, WeaponKey: "hinf_environment", Kills: 1},
		{XUID: kscXUID, Category: domain.KillSourceCategoryExplosiveObject, WeaponKey: "", Kills: 2},
		{XUID: kscXUID, Category: domain.KillSourceCategoryExplosiveObject, WeaponKey: "hinf_coil_plasma", Kills: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ligne %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestKillSourceCategoryKills_SansCategoriseur : un classificateur qui ne range pas les
// sources (titre sans cette table) degrade proprement.
func TestKillSourceCategoryKills_SansCategoriseur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceClassifier{})
	_, err := repo.LoadKillSourceCategoryKills(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want games.ErrCapabilityNotSupported", err)
	}
}

// TestKillSourceCategoryKills_FiltresInvalides : garde-fou anti-scan-complet.
func TestKillSourceCategoryKills_FiltresInvalides(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceCategorizer{})
	if _, err := repo.LoadKillSourceCategoryKills(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{}); err == nil {
		t.Error("filtres vides acceptes")
	}
}
