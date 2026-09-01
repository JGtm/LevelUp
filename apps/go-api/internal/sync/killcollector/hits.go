package killcollector

// hits.go — LE PRODUCTEUR du NUMERATEUR de precision par arme d Infinite : weapon_accuracy
// (shots_fired/shots_landed) + match_weapon_hit_distance (histogramme de distance des touches),
// reconstruits DU FILM par l appariement tir<->degat (methode PAR LE TIR,
// NOTE_ATTRIBUTION_ARME_TIR_2026-08-31). Pendant, cote precision, de la ventilation des tirs
// (shots.go) : la meme passe de fond, une famille de donnees de plus.
//
// # POURQUOI DIR-BASE (et pas sur les chunks en memoire comme shots.go)
//
// Le numerateur exige les POSITIONS bipedes (distance tireur<->victime) et les damage_aftermath,
// decodes par les scanners filmdec (ScanFilmBipedPositions, ScanFilmWeaponDamages, decouverte du
// decoupage i0 de la carte). Ces scanners sont DIR-BASE (ils lisent chunk_NN.bin + inflatent) et
// productionises au Lot 2 : les rejouer en memoire dupliquerait le decodeur (interdit). La passe
// se greffe donc sur le film DEJA sur disque (cache local, data/cache/film_chunks/{matchID}) —
// pas un re-telechargement, la seule ressource chere du chantier. Sans repertoire (chemin live),
// elle se saute proprement (ConfigureFilmAccuracy non appele -> filmDir nil).
//
// # LE PONT FilmIndex -> xuid, ET SA RESERVE
//
// Le film ne porte aucun xuid cote replication : l identite se resout par l indice. On REUTILISE le
// resolveur des tirs (resolvePlayerIndices : indice de replication -> xuid, valide a 77 % contre
// l oracle killsource). RESERVE : filmdec.WeaponHitStats.FilmIndex vient d un AUTRE champ du record
// de tir (decodeFireEvent, bits 36-40 >>1) que l indice que resolvePlayerIndices indexe (5 bits
// AVANT le motif xuid). Leur equivalence n est PAS encore validee sur film ; un decalage
// systematique se verrait au compteur `killsource_hits_indices_non_resolus` (indices non rattaches)
// et au gate visuel du Lot 4. On instrumente le risque, on ne le masque pas.
//
// BEST-EFFORT ASSUME : tout echec (film absent du disque, scan casse, carte inconnue) se journalise
// et se compte, jamais il n avale la passe des morts (deja ecrites, justes).

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/ingest"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/haloclient"
)

// hitsMetric* : les compteurs de sante du numerateur film (ADR 0009 — entiers, snake_case).
const (
	metricHitsMatches     = "killsource_hits_matchs"
	metricHitsAccuracyRow = "killsource_hits_accuracy_lignes"
	metricHitsDistanceRow = "killsource_hits_distance_lignes"
	metricHitsNoIndex     = "killsource_hits_indices_non_resolus"
	metricHitsNoFilmDir   = "killsource_hits_films_absents_disque"
	metricHitsScanFail    = "killsource_hits_erreurs_scan"
	metricHitsWriteFail   = "killsource_hits_erreurs_ecriture"
)

// collectHits : la troisieme ecriture de la passe — weapon_accuracy + match_weapon_hit_distance.
//
// GATE DATA-LEVEL `match.weapon.accuracy` (games.CapWeaponAccuracy) : un titre qui n expose pas la
// precision par arme ne se voit pas reconstruire un numerateur film. Degradation gracieuse (ni
// panic, ni erreur remontee), title-agnostic (jamais slug ==).
func (c *KillSourceCollector) collectHits(
	ctx context.Context, matchID string, chunks []haloclient.FilmChunk, parts MatchIdentities,
) {
	if !c.caps.Has(games.CapWeaponAccuracy) {
		slog.DebugContext(ctx, "killsource: precision par arme — capability absente, passe ignoree",
			"match_id", matchID, "capability", string(games.CapWeaponAccuracy),
			"err", games.ErrCapabilityNotSupported)
		return
	}
	if c.filmDir == nil {
		return // numerateur non configure (chemin live sans cache disque) : best-effort silencieux
	}
	dir := c.filmDir(matchID)
	if dir == "" {
		observability.AddInt(metricHitsNoFilmDir, 1)
		slog.InfoContext(ctx, "killsource: precision par arme — film absent du disque, passe sautee",
			"match_id", matchID)
		return
	}

	accuracy, distance, ok := c.buildHitsBatches(ctx, matchID, dir, chunks, parts)
	if !ok {
		return
	}
	if err := c.writeHits(ctx, accuracy, distance); err != nil {
		observability.AddInt(metricHitsWriteFail, 1)
		slog.ErrorContext(ctx, "killsource: ecriture du numerateur de precision echouee",
			"match_id", matchID, "err", err)
		return
	}
	publishHitsPass(ctx, matchID, accuracy, distance, len(parts.XUIDs))
}

// buildHitsBatches rejoue le film (dir-base), apparie tirs<->degats et rend les deux batches
// (accuracy + distance). ok=false sur tout echec de scan (best-effort, journalise). Tient le verrou
// de decode du process (les parametres de replication de filmdec sont des globaux de paquet).
func (c *KillSourceCollector) buildHitsBatches(
	ctx context.Context, matchID, dir string, chunks []haloclient.FilmChunk, parts MatchIdentities,
) ([]persist.WeaponAccuracyInsert, persist.WeaponHitDistanceBatch, bool) {
	release := filmdec.LockProcessDecode()
	defer release()

	raw, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "chunk_00 illisible", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "registre illisible", err)
	}
	n := filmdec.CountFilmChunks(dir)
	shots, err := filmdec.ScanFilmWeaponShots(dir, n)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "collecte des tirs", err)
	}
	damages, _, err := filmdec.ScanFilmWeaponDamages(dir, reg, n)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "collecte des degats", err)
	}

	// Distance : optionnelle. Bornes de carte inconnues -> distFn nil, les touches restent
	// comptees (reserve #5 du plan) ; pas un echec de passe.
	distFn := c.resolveHitDistanceFunc(ctx, matchID, dir, damages, n)

	stats := filmdec.PairWeaponHits(shots, damages, filmdec.WeaponHitPairWindowUS, distFn)

	// LE PONT FilmIndex -> xuid : le resolveur EXISTANT (indice de replication -> xuid), sur les
	// memes chunks de replication que la ventilation des tirs. Voir la reserve en tete de fichier.
	piToXUID := resolvePlayerIndices(parts.XUIDs, ReplicationChunks(chunks))
	resolveXUID := func(filmIndex int) string { return piToXUID[filmIndex] }

	accuracy, distance := ingest.MapWeaponAccuracyFilm(matchID, stats, resolveXUID, migration.WeaponHitDistanceDecoderRev)
	return accuracy, distance, true
}

// resolveHitDistanceFunc construit la WeaponHitDistanceFunc (distance tireur<->victime) si les
// bornes de la carte se resolvent ; nil sinon (distances desactivees, touches comptees). Le
// catalogue de bornes non configure (mapBoundsPath vide) est un cas NORMAL, pas une erreur.
func (c *KillSourceCollector) resolveHitDistanceFunc(
	ctx context.Context, matchID, dir string, damages []filmdec.WeaponDamage, n int,
) filmdec.WeaponHitDistanceFunc {
	if c.mapBoundsPath == "" {
		return nil
	}
	wr, err := filmdec.DetectFilmWorldRange(dir, c.mapBoundsPath, "")
	if err != nil {
		slog.DebugContext(ctx, "killsource: precision par arme — bornes de carte inconnues, distances desactivees",
			"match_id", matchID, "err", err)
		return nil
	}
	distFn, base, err := filmdec.FilmWeaponHitDistance(dir, wr, damages, n)
	if err != nil {
		slog.DebugContext(ctx, "killsource: precision par arme — positions bipedes indisponibles, distances desactivees",
			"match_id", matchID, "err", err)
		return nil
	}
	slog.DebugContext(ctx, "killsource: precision par arme — distances actives",
		"match_id", matchID, "base_positions", base)
	return distFn
}

// hitsScanFailed compte et journalise un echec de scan, et rend le triplet d abandon (best-effort).
func (c *KillSourceCollector) hitsScanFailed(
	ctx context.Context, matchID, etape string, err error,
) ([]persist.WeaponAccuracyInsert, persist.WeaponHitDistanceBatch, bool) {
	observability.AddInt(metricHitsScanFail, 1)
	slog.InfoContext(ctx, "killsource: precision par arme — scan du film echoue, passe sautee",
		"match_id", matchID, "etape", etape, "err", err)
	return nil, persist.WeaponHitDistanceBatch{}, false
}

// writeHits ecrit le numerateur (accuracy + distance) sous le lease RW de shared (ADR 0013).
//
// UN LEASE COURT, DEDIE, comme writeShots : les deux tables sont append-only/independantes de la
// passe des morts, rien n exige la meme transaction. Le lease RW de shared est la ressource la plus
// disputee du process — on ne le tient pas pendant le scan du film.
func (c *KillSourceCollector) writeHits(
	ctx context.Context, accuracy []persist.WeaponAccuracyInsert, distance persist.WeaponHitDistanceBatch,
) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", distance.MatchID, err)
	}
	defer release()
	return persist.NewWeaponHitDistancePersister(db).PersistPass(ctx, accuracy, distance)
}

// publishHitsPass : les compteurs de sante (ADR 0009) et la trace de la passe.
//
// `killsource_hits_indices_non_resolus` est celui qui informe : un tireur dont l indice film n a
// pas ete rattache a un xuid n a AUCUNE ligne. Un decalage systematique du pont FilmIndex->xuid
// (reserve en tete) se verrait ici — un echec total serait sinon silencieux dans les tables.
func publishHitsPass(
	ctx context.Context, matchID string,
	accuracy []persist.WeaponAccuracyInsert, distance persist.WeaponHitDistanceBatch, rosterSize int,
) {
	observability.AddInt(metricHitsMatches, 1)
	observability.AddInt(metricHitsAccuracyRow, int64(len(accuracy)))
	observability.AddInt(metricHitsDistanceRow, int64(len(distance.Rows)))

	// Joueurs distincts effectivement attribues (accuracy) — un roster plus grand que ce compte
	// signale des indices non resolus (bots compris, qui n ont legitimement pas de xuid).
	named := map[string]struct{}{}
	for i := range accuracy {
		named[accuracy[i].XUID] = struct{}{}
	}
	if manquants := rosterSize - len(named); manquants > 0 {
		observability.AddInt(metricHitsNoIndex, int64(manquants))
	}
	slog.InfoContext(ctx, "killsource: numerateur de precision par arme",
		"match_id", matchID, "accuracy_lignes", len(accuracy),
		"distance_lignes", len(distance.Rows), "joueurs_attribues", len(named), "roster", rosterSize)
}
