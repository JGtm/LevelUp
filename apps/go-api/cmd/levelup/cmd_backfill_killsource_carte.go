package main

// cmd_backfill_killsource_carte.go — LES DEUX PASSES DE `backfill-killsource` RETIRENT LES MATCHS
// SANS CARTE AVANT DE LES DECODER (2026-09-27).
//
// Un match dont la carte n est pas resolue ne se decode pas (« le flux du film est la seule source
// fiable. Pas de repli. »). La passe `--online` le telechargeait quand meme, et les deux passes lui
// donnaient une place de `--limit`. La selection se fait donc SANS borne, les matchs sans carte en
// sont retires par le collecteur (`RetenirLesMatchsAvecCarte`, le meme filtre que le post-sync),
// et `--limit` s applique APRES : la place d un match sans carte va au suivant. Le `--dry-run`
// garde la borne a la selection — il n ouvre pas la base de metadonnees qui resout les cartes.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/killcollector"
)

// selectionSansBorne : les options de SELECTION — sans `--limit` hors `--dry-run`, parce que la
// borne s applique apres le retrait des matchs sans carte.
func selectionSansBorne(o killsourceOptions) killsourceOptions {
	if !o.dryRun {
		o.limit = 0
	}
	return o
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
