package main

// cmd_backfill_killsource_carte.go — LES DEUX PASSES DE `backfill-killsource` RETIRENT LES MATCHS
// SANS CARTE AVANT DE LES DECODER (2026-09-27).
//
// Un match dont la carte n est pas resolue ne se decode pas (« le flux du film est la seule source
// fiable. Pas de repli. »). La passe `--online` le telechargeait quand meme, et les deux passes lui
// donnaient une place de `--limit`. La selection se fait donc SANS borne, les matchs sans carte en
// sont retires par le collecteur (`RetenirLesMatchsAvecCarte`, le meme filtre que le post-sync),
// et `--limit` s applique APRES : la place d un match sans carte va au suivant.
//
// LE `--dry-run` ANNONCE LA SELECTION DE LA PASSE QUI DECODE : il passe par la MEME fonction
// ([candidatsDeLaPasse], [idsDeLaPasseEnLigne]), retrait des matchs sans carte compris. Un plan qui
// ne resoudrait pas les cartes annoncerait des matchs que la passe retire aussitot.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/killcollector"
)

// selectionSansBorne : les options de SELECTION — sans `--limit`, parce que la borne s applique
// apres le retrait des matchs sans carte.
func selectionSansBorne(o killsourceOptions) killsourceOptions {
	o.limit = 0
	return o
}

// candidatsDeLaPasse : LA selection de la passe hors ligne, pour le plan (`--dry-run`) comme pour
// la passe qui decode — les films du cache a decoder ([filmsACollecter], sans borne), les matchs
// sans carte retires par le collecteur, puis `--limit`.
func candidatsDeLaPasse(
	ctx context.Context, db *sql.DB, cacheRoot string, o killsourceOptions, col *killcollector.KillSourceCollector,
) ([]filmCandidat, bilanDeSelection, error) {
	candidats, bilan, err := filmsACollecter(ctx, db, cacheRoot, selectionSansBorne(o))
	if err != nil {
		return nil, bilanDeSelection{}, err
	}
	return candidatsAvecCarte(ctx, col, candidats, o.limit), bilan, nil
}

// idsDeLaPasseEnLigne : LA selection de la passe `--online`, pour le plan comme pour la passe — les
// matchs sans passe de film ([matchsSansPasseDeFilm], sans borne), les matchs sans carte retires,
// puis `--limit`. Aucun aller-retour reseau : la carte se resout en base.
func idsDeLaPasseEnLigne(
	ctx context.Context, db *sql.DB, o killsourceOptions, col *killcollector.KillSourceCollector,
) ([]string, error) {
	ids, err := matchsSansPasseDeFilm(ctx, db, selectionSansBorne(o))
	if err != nil {
		return nil, err
	}
	return idsAvecCarte(ctx, col, ids, o.limit), nil
}

// idsAvecCarte : les identifiants dont la carte se resout, dans l ordre, au plus `limit` (0 : tous).
func idsAvecCarte(ctx context.Context, col *killcollector.KillSourceCollector, ids []string, limit int) []string {
	retenus, ecartes := col.RetenirLesMatchsAvecCarte(ctx, ids, limit)
	if len(ecartes) > 0 {
		fmt.Printf("films ecartes faute de carte resolue : %d (aucun telechargement ni decodage ; ils "+
			"restent candidats et reviendront quand leur carte se resoudra)\n", len(ecartes))
	}
	return retenus
}

// candidatsAvecCarte : [idsAvecCarte] sur les candidats de la passe hors ligne, leur ordre garde.
func candidatsAvecCarte(
	ctx context.Context, col *killcollector.KillSourceCollector, candidats []filmCandidat, limit int,
) []filmCandidat {
	ids := make([]string, 0, len(candidats))
	for _, c := range candidats {
		ids = append(ids, c.matchID)
	}
	garde := make(map[string]bool, len(candidats))
	for _, id := range idsAvecCarte(ctx, col, ids, limit) {
		garde[id] = true
	}
	out := make([]filmCandidat, 0, len(garde))
	for _, c := range candidats {
		if garde[c.matchID] {
			out = append(out, c)
		}
	}
	return out
}

// collecteurHorsLigne : le collecteur de la passe hors ligne — le cache local, la porte de la base
// (lot 5.24.2) et la capture. Sorti de `passeDesFilms` quand le retrait des matchs sans carte l a
// portee au-dela des 80 lignes du depot : aucune ligne de cablage n a change.
func collecteurHorsLigne(
	cache *haloclient.LocalFilmCache, db *sql.DB, porte *killcollector.PorteDeLaBase,
	caps games.CapabilityMap, capture killcollector.DepsCapture,
) *killcollector.KillSourceCollector {
	return killcollector.NewKillSourceCollector(
		killcollector.NewLocalCacheFilms(cache),
		// L ANNUAIRE DE PASSE (lot 5.24.2) : `v_gamertag_lookup` lue UNE FOIS, pas par match.
		// C est la seule lecture repetee que la decomposition 5.24.1 ait trouvee — et depuis que
		// la passe a des ouvriers, elle serait SERIALISEE derriere la porte, donc un plafond.
		porte.GarderLeRoster(killcollector.NewSharedRoster(db).AvecAnnuaireDePasse()),
		porte.GarderLeWriter(writerDeja(db)),
		caps,
		0, // limite par match : le defaut du collecteur (45 min)
	).AvecCapture(capture)
}
