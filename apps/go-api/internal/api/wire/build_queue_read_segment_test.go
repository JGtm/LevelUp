package wire

// build_queue_read_segment_test.go — LE DEPOT D'OUVRIER TRANSMET UN SEGMENT DE LECTURE AUX
// DERIVATIONS (lot L4.1 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
//
// Sans lui, les niveaux d'armes ne lisaient jamais l'identite des matchs (carte, mode) et
// n'ecrivaient rien sur ce chemin. Ce qui est tenu ici : le segment emprunte le provider DU
// TITRE et le relache ; sans provider il n'y a pas de segment (les niveaux le journalisent en
// erreur et ne marquent pas le match) ; un provider en panne saute l'etape sans la faire croire
// jouee.

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

func TestSharedReadDepot_SansProviderPasDeSegment(t *testing.T) {
	if sharedReadDepot(nil, "halo_infinite") != nil {
		t.Fatal("segment de lecture rendu sans provider : les derivations liraient le shared d'un " +
			"autre titre, ou rien")
	}
}

func TestSharedReadDepot_EmprunteLeProviderDuTitre(t *testing.T) {
	var sentinelle sql.DB
	lire := sharedReadDepot(sharedprovider.FromInMemoryDB(&sentinelle, "shared.duckdb"), "halo_infinite")
	if lire == nil {
		t.Fatal("aucun segment de lecture alors qu'un provider est cable")
	}
	var vu *sql.DB
	lire(context.Background(), "niveaux d'armes", func(db *sql.DB) { vu = db })
	if vu != &sentinelle {
		t.Error("le segment n'a pas rendu le handle du provider du titre")
	}
}

func TestSharedReadDepot_ProviderFermeSauteLEtape(t *testing.T) {
	var sentinelle sql.DB
	p := sharedprovider.FromInMemoryDB(&sentinelle, "shared.duckdb")
	if err := p.Close(); err != nil {
		t.Fatalf("fermeture du provider: %v", err)
	}
	appele := false
	sharedReadDepot(p, "halo_infinite")(context.Background(), "niveaux d'armes",
		func(*sql.DB) { appele = true })
	if appele {
		t.Error("l'etape a ete jouee sur un provider ferme : la lecture doit etre sautee, et la " +
			"famille qui l'attendait doit le savoir (elle n'est pas appelee)")
	}
}
