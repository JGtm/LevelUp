package duckdb

// squad_v2_adapter_weapons_test.go — L'ESCOUADE LIT LE MEME ARME QUE LES AUTRES SURFACES.
//
// POURQUOI CE TEST EXISTE. Jusqu'au 2026-09-01, `LoadWeaponKills` construisait
// `NewWeaponKillsRepo` SANS condition : c'etait le dernier appelant de production hors du
// gate de capability. Sur un titre a decodeur de film, `weapon_kills` a ete SUPPRIMEE
// (`shared_drop_weapon_kills_v1`) — la page servait donc des series VIDES pendant que
// toutes les autres surfaces lisaient la source de degat. Le defaut etait muet : aucune
// erreur, aucune ligne, un graphe blanc.
//
// Le test verrouille les deux sens du branchement, parce que les deux comptent : la
// fabrique injectee DOIT etre suivie (sinon la regression revient), et son absence DOIT
// rendre le lecteur historique (les appelants hors HTTP et les titres sans decodeur de
// film n'ont pas d'autre voie).

import (
	"context"
	"testing"

	"levelup/go-api/internal/port"
)

// stubWeaponKillsRepo : un lecteur reconnaissable, pour prouver QUI a ete choisi.
type stubWeaponKillsRepo struct{ marque string }

func (s *stubWeaponKillsRepo) LoadWeaponKillsAggregated(
	context.Context, string, port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	return nil, nil
}

func TestSquadV2LoaderAdapter_SuitLaFabriqueInjectee(t *testing.T) {
	attendu := &stubWeaponKillsRepo{marque: "source-de-degat"}
	var recu *PlayerDB
	appels := 0

	a := NewSquadV2LoaderAdapter(nil)
	a.SetWeaponKillsRepoFactory(func(pdb *PlayerDB) port.WeaponKillsRepository {
		appels++
		recu = pdb
		return attendu
	})

	obtenu := a.weaponKillsRepo(nil)
	if obtenu != port.WeaponKillsRepository(attendu) {
		t.Fatalf("lecteur = %T, attendu le stub injecte — l'Escouade ignore la fabrique du "+
			"cablage et retombe sur le lecteur historique", obtenu)
	}
	if appels != 1 {
		t.Errorf("fabrique appelee %d fois, attendu 1", appels)
	}
	if recu != nil {
		t.Errorf("PlayerDB transmis = %v, attendu celui passe a weaponKillsRepo", recu)
	}
}

func TestSquadV2LoaderAdapter_SansFabriqueRendLeLecteurHistorique(t *testing.T) {
	a := NewSquadV2LoaderAdapter(nil)
	if _, ok := a.weaponKillsRepo(nil).(*WeaponKillsRepo); !ok {
		t.Fatal("sans fabrique injectee, le repli DOIT etre le lecteur historique " +
			"(appelants hors HTTP, titres sans decodeur de film)")
	}
}

// TestSquadV2LoaderAdapter_FabriqueNilNeCasseRien : une fabrique qui rend nil (titre sans
// PlayerDB resolu) ne doit pas propager un lecteur nil — sinon l'appel suivant panique.
func TestSquadV2LoaderAdapter_FabriqueNilNeCasseRien(t *testing.T) {
	a := NewSquadV2LoaderAdapter(nil)
	a.SetWeaponKillsRepoFactory(func(*PlayerDB) port.WeaponKillsRepository { return nil })
	if _, ok := a.weaponKillsRepo(nil).(*WeaponKillsRepo); !ok {
		t.Fatal("une fabrique qui rend nil doit retomber sur le lecteur historique, " +
			"jamais rendre nil")
	}
}
