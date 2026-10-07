package replay

// zone_colline_nom_temoin_test.go — LE TEMOIN SUR FILM REEL DU PROPRIETAIRE DE COLLINE DESIGNE PAR
// LE NOM.
//
// Un KOTH du corpus de la phase 2a : le calque des zones est assemble DEUX FOIS par le chemin de
// production (`BuildFromPositions`, calque du drapeau exclu, cf. `zone_etat_initial_temoin_test.go`),
// sans puis avec les lectures d image-cle :
//
//	sans   aucun nom : le proprietaire est le slot voisin du designateur (repli compte) ;
//	avec   le nom du designateur designe le proprietaire de son bloc.
//
// Les deux rendent LE MEME calque : le nom designe le canal que le voisinage designait, et la
// couverture le dit (`ownerNamed` 1, `ownerVoteDisagreed` 0).
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_FILM`), un film par processus, avant-plan :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks/01e1f945"
//	go test -count=1 -run TestCollineProprietaireNomTemoin -v -timeout 30m ./internal/games/halo_infinite/film/replay/

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// TestCollineProprietaireNomTemoin assemble le calque des collines sans puis avec les noms.
func TestCollineProprietaireNomTemoin(t *testing.T) {
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	if film.Mode != "KOTH" {
		t.Skipf("film %s hors KOTH", short)
	}
	sc := p2bScan(t, dir)
	zone := ZoneInput{Scanned: true, Reads: sc.Reads, Zones: p2aZones(t, film.MapID, p2aRolesDuMode(film)...),
		Roles: p2bRoles(film), TeamByXUID: film.p2aTeams(), Hill: true}
	cuire := etatInitialCuisson(t, dir, short, p2aQuant(t, film.Carte))
	caps := p2aCaptures(p2aBobine(t, dir), film)
	avant, _ := cuire(zone, caps)
	zone.KeyReads = sc.KeyReads
	apres, _ := cuire(zone, caps)
	ca, cp := avant.Coverage.Zones, apres.Coverage.Zones
	t.Logf("FILM %s — avant : %d collines, ownerNamed %d, repli voisin %d ; apres : %d collines, "+
		"ownerNamed %d, ownerVoteDisagreed %d, repli voisin %d", short, len(avant.ZoneStates), ca.OwnerNamed,
		repliVoisin(avant), len(apres.ZoneStates), cp.OwnerNamed, cp.OwnerVoteDisagreed, repliVoisin(apres))
	if ca.Method != ZoneMethodDesignator || cp.Method != ZoneMethodDesignator {
		t.Fatalf("methodes %q / %q : le temoin exige la voie du designateur", ca.Method, cp.Method)
	}
	if repliVoisin(avant) != 1 || ca.OwnerNamed != 0 {
		t.Errorf("sans nom : repli voisin %d, nommees %d — attendu 1 et 0", repliVoisin(avant), ca.OwnerNamed)
	}
	if repliVoisin(apres) != 0 || cp.OwnerNamed != 1 || cp.OwnerVoteDisagreed != 0 {
		t.Errorf("avec les noms : repli voisin %d, nommees %d, discordances %d — attendu 0, 1, 0",
			repliVoisin(apres), cp.OwnerNamed, cp.OwnerVoteDisagreed)
	}
	if len(apres.ZoneStates) == 0 || !reflect.DeepEqual(avant.ZoneStates, apres.ZoneStates) {
		t.Errorf("le calque des collines change avec les noms (%d -> %d etats)", len(avant.ZoneStates),
			len(apres.ZoneStates))
	}
}

// repliVoisin rend le nombre de declenchements du repli « voisin du designateur » d un document.
func repliVoisin(doc ReplayDocument) int {
	for _, f := range doc.Coverage.Fallbacks {
		if f.Name == string(fallback.NomCollineProprietaireVoisinDuDesignateur) {
			return f.Hits
		}
	}
	return 0
}
