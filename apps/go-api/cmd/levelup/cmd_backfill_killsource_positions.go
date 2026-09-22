package main

// cmd_backfill_killsource_positions.go — LE CABLAGE DE LA CAPTURE DES POSITIONS (G.2bis) pour
// `levelup backfill-killsource`.
//
// Fichier dedie extrait de `cmd_backfill_killsource.go` le 2026-09-22 (tour de revue du
// lot 5.24), quand l extraction de `jouerLesDeuxPasses` a laisse le fichier a 520 lignes.
// Deplacement pur, aucun changement de comportement. La coupure suit une frontiere nette :
// LA-BAS ce que la passe fait, ICI ce qu il faut lui brancher pour qu elle place ses morts.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/killcollector"
)

// positionCaptureDeps construit les dependances de la capture des positions (G.2bis) :
// resolution de carte (`port.ReplayMapNameRepo`, implementee par `duckdb.ReplayMapRepo`) et
// catalogue de bornes de dequantification (le meme que `replaybuild`, DONNEE DE REFERENCE
// VERSIONNEE — data/titles/{slug}/reference/map_quant_bounds.json, pas une sortie de sync).
//
// BEST-EFFORT PAR CONCEPTION : un catalogue illisible ou une metadata indisponible degrade en
// « positions desactivees », jamais une erreur fatale — le backfill des morts et des tirs, la
// raison d etre de cette commande, ne doit pas dependre d une brique tierce. `cleanup` ferme le
// handle metadata ouvert ici ; elle est TOUJOURS non-nil (no-op si rien n a ete ouvert), donc
// l appelant peut la `defer` inconditionnellement.
func positionCaptureDeps(
	cfg *config.AppConfig, titleSlug string, sharedDB *sql.DB, porte *killcollector.PorteDeLaBase,
) (killcollector.DepsCapture, func()) {
	noop := func() {}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)

	// OpenReadOnly (pas OpenReadForQuery) : la precondition de CETTE commande est le SERVEUR
	// ARRETE (comme pour le handle RW de shared), donc aucun autre process ne tient metadata en
	// ecriture pendant la passe — la garde « different configuration » d OpenReadForQuery n a
	// rien a proteger ici, et NewReplayMapRepo demande le type *duckdb.DB, pas *sql.DB brut.
	metaDB, err := duckdb.OpenReadOnly(pr.MetadataDBPath(titleSlug))
	if err != nil {
		fmt.Printf("metadata illisible (%v) — positions desactivees pour cette passe\n", err)
		return killcollector.DepsCapture{}, noop
	}
	fermer := func() { _ = metaDB.Close() }

	// LE CATALOGUE ET SA POLITIQUE DE DEGRADATION VIVENT DANS `killcollector` (lot 7C.8) :
	// les TROIS chemins de collecte (post-sync du serveur, --online, ce backfill) les
	// partagent. Le defaut P0-1 etait que seul celui-ci cablait la capture.
	// LA RESOLUTION DE CARTE PASSE PAR LA PORTE (lot 5.24.2) : elle lit `match_registry` sur le
	// handle partage, donc elle est l un des trois chemins par lesquels cette passe parle a la
	// base — et il n y a aucune raison d en laisser un dehors.
	capture, err := killcollector.CaptureDepuisCatalogue(cfg.RepoRoot, titleSlug,
		porte.GarderLesCartes(duckdb.NewReplayMapRepo(staticSharedReader{db: sharedDB}, metaDB)))
	if err != nil {
		fmt.Printf("%v — positions desactivees pour cette passe\n", err)
		return killcollector.DepsCapture{}, fermer
	}
	return capture, fermer
}

// staticSharedReader adapte le handle shared DEJA OUVERT (le lease de cette commande) en
// duckdb.SharedReader. Meme raison que writerDeja : le process est seul (serveur arrete), donc
// release est un no-op — il n y a pas de second lease a poser par-dessus celui deja tenu.
type staticSharedReader struct{ db *sql.DB }

func (s staticSharedReader) Get(context.Context) (*sql.DB, func(), error) {
	return s.db, func() {}, nil
}
