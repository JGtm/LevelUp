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
// # LE PONT FilmIndex -> xuid, RESERVE LEVEE (mesuree 2026-09-01)
//
// Les chunks de replication ne portent aucun xuid : l identite se resout par l indice. On REUTILISE
// le resolveur des tirs (resolvePlayerIndices : indice de replication -> xuid, valide a 77 % contre
// l oracle killsource).
//
// ⚠ CE RESOLVEUR EST UNE INFERENCE, ET LE FILM ECRIT LA REPONSE. `chunk_00` porte la table des
// joueurs (`decfilm.ReadPlayerTable`, lot 1.5) : le lien `FilmIndex -> XUID` s y lit directement,
// la ou `resolvePlayerIndices` le CHERCHE (motif du xuid dans le flux, 5 bits qui precedent).
// Le lot 1.8 a bascule le decodeur de morts sur cette lecture ; les TIRS et les TOUCHES ne l ont
// PAS ete — ils portent deux revisions distinctes (`WeaponShotsDecoderRev`,
// `migration.WeaponHitDistanceDecoderRev`), donc deux backlogs de redecodage separes, hors du
// perimetre de ce lot. Decouverte D4 (1.8) du PLAN_DECODEUR_FILM, §4. La RESERVE historique etait que decfilm.WeaponHitStats.FilmIndex venait d un
// AUTRE champ du record de tir (decodeFireEvent, bits 36-40 >>1 = 4 bits) que l indice que
// resolvePlayerIndices indexe (5 bits). VERDICT MESURE (TestWeaponIndexNumDenomEquivalence, package
// analysis) : le 4 bits n etait que la MOITIE BASSE du champ. La cle est desormais
// decfilm.FireEvent.ShooterIndex5 (bits 35-39, R(5)), ALIGNEE au bit pres sur
// weaponscan.FireEvent.FilmIndex5 (le denominateur match_weapon_shots) : mismatch 0 sur 4342 records
// BTB correles + tous les records arene. Num et denom keyent DESORMAIS IDENTIQUE. Sous 17 joueurs les
// deux lectures coincidaient deja (arene) ; au-dela (BTB 4f77afc1, lobby 24), le 4 bits saturait a 15
// et fusionnait 8 paires de joueurs — d ou une precision fausse. Corrige au Lot 3 (ScanFilmWeaponShots
// key sur ShooterIndex5). Le compteur `killsource_hits_indices_non_resolus` reste le garde-fou en prod.
//
// BEST-EFFORT ASSUME : tout echec (film absent du disque, scan casse, carte inconnue) se journalise
// et se compte, jamais il n avale la passe des morts (deja ecrites, justes).

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
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
	// metricHitsNoMapEntry : la carte du film n a pas d entree au catalogue de bornes — donc ni
	// bornes monde ni decoupage d i0. D-4 d ADR 0034 : une carte inconnue se COMPTE, elle ne se
	// devine pas. Les touches restent comptees, seule leur distance manque.
	metricHitsNoMapEntry = "killsource_hits_cartes_hors_catalogue"
	// metricHitsNoMapName : le match n a AUCUN nom de carte (base muette sur ce match) — donc
	// aucune identite de carte, donc aucune borne. DISTINCT du precedent (lot 1.9.4) : « je ne
	// sais pas quelle carte » et « je sais quelle carte, elle n est pas au catalogue » appellent
	// deux gestes differents, et un seul compteur pour les deux les rendrait indiscernables.
	metricHitsNoMapName = "killsource_hits_matchs_sans_nom_de_carte"
	// metricHitsNoMapWiring : la capability est la, mais le collecteur n a pas recu
	// `WithPositionCapture` — il n a donc ni resolveur de nom ni catalogue. Regression de
	// CABLAGE, pas etat des donnees : meme motif que `metricPositionsNotWired`.
	metricHitsNoMapWiring = "killsource_hits_carte_non_cablee"
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

	raw, err := decfilm.ReadFilmChunk(dir, 0)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "chunk_00 illisible", err)
	}
	reg, err := decfilm.ParseRegistryChunk(raw)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "registre illisible", err)
	}
	n := decfilm.CountFilmChunks(dir)
	shots, err := decfilm.ScanFilmWeaponShots(dir, n)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "collecte des tirs", err)
	}
	damages, _, err := decfilm.ScanFilmWeaponDamages(dir, reg, n)
	if err != nil {
		return c.hitsScanFailed(ctx, matchID, "collecte des degats", err)
	}

	// Distance : optionnelle. Bornes de carte inconnues -> distFn nil, les touches restent
	// comptees (reserve #5 du plan) ; pas un echec de passe.
	distFn := c.resolveHitDistanceFunc(ctx, matchID, dir, damages, n)

	stats := decfilm.PairWeaponHits(shots, damages, decfilm.WeaponHitPairWindowUS, distFn)

	// LE PONT FilmIndex -> xuid : le resolveur EXISTANT (indice de replication -> xuid), sur les
	// memes chunks de replication que la ventilation des tirs. Voir la reserve en tete de fichier.
	piToXUID := resolvePlayerIndices(parts.XUIDs, ReplicationChunks(chunks))
	resolveXUID := func(filmIndex int) string { return piToXUID[filmIndex] }

	accuracy, distance := ingest.MapWeaponAccuracyFilm(matchID, stats, resolveXUID, migration.WeaponHitDistanceDecoderRev)
	return accuracy, distance, true
}

// resolveHitDistanceFunc construit la WeaponHitDistanceFunc (distance tireur<->victime) si
// L ENTREE DE CATALOGUE de la carte se resout ; nil sinon (distances desactivees, touches
// comptees — `repli_distances_de_touche_desactivees` au registre).
//
// L ENTREE ENTIERE, PAS SES SEULES BORNES (lot 1.9.2) : elle porte AUSSI le decoupage d i0 de la
// carte, que le balayage impose desormais au lieu de le laisser detecter
// (`decfilm.BuildBipedTracks`). D-3 d ADR 0034.
func (c *KillSourceCollector) resolveHitDistanceFunc(
	ctx context.Context, matchID, dir string, damages []decfilm.WeaponDamage, n int,
) decfilm.WeaponHitDistanceFunc {
	entry, ok := c.entreeDeCarteDesTouches(ctx, matchID)
	if !ok {
		return nil
	}
	distFn, base, err := decfilm.FilmWeaponHitDistance(dir, entry, damages, n)
	if err != nil {
		slog.DebugContext(ctx, "killsource: precision par arme — positions bipedes indisponibles, distances desactivees",
			"match_id", matchID, "err", err)
		return nil
	}
	slog.DebugContext(ctx, "killsource: precision par arme — distances actives",
		"match_id", matchID, "base_positions", base)
	return distFn
}

// entreeDeCarteDesTouches rend l entree de catalogue de la carte du match — LUE A SON NOM, JAMAIS
// DEVINEE (lot 1.9.4, D13). ok=false desactive les distances ; les touches restent comptees.
//
// # CE QUE CE LOT A RETIRE, ET CE QUE LA MESURE EN DIT
//
// Ce site appelait `grammar.DetectFilmMapEntry(dir, c.mapBoundsPath, "")` : la carte s y
// reconnaissait a la SIGNATURE des largeurs d axe du decoupage d i0, lu dans le film, croisee au
// catalogue — alors que le MEME collecteur resolvait deja le nom de carte du match par la base
// pour la passe des positions, et que le parametre `mapNameOverride` existait et etait passe VIDE.
//
// La signature n est pas seulement ambigue, elle est FAUSSE (mesure du lot, §5 du plan) : sur les
// 79 cartes du catalogue, 68 tombent dans 5 classes de meme signature dont une de 59 cartes, et
// sur les 17 films mesures la signature rend 2 accords, 13 ambiguites et 2 DESACCORDS — les deux
// films Live Fire, ou elle designe `aquarius` avec un seul candidat. Les distances y auraient ete
// calculees dans l AABB d une autre carte, sans un mot. Elle n a donc pas ete retrogradee en
// repli : un repli qui se declenche a tort corrompt un fait que la lecture aurait donne juste
// (D14 d).
//
// # TROIS SORTIES, TROIS COMPTEURS, ET C EST VOULU
//
// Le CABLAGE absent (pas de resolveur de nom), le NOM absent (base muette sur ce match) et la
// carte HORS CATALOGUE sont trois causes distinctes qui appellent trois gestes distincts. Un
// compteur unique les rendrait indiscernables — et un titre entier pourrait perdre ses distances
// en silence, ce que D-4 d ADR 0034 interdit.
func (c *KillSourceCollector) entreeDeCarteDesTouches(
	ctx context.Context, matchID string,
) (decfilm.MapQuantEntry, bool) {
	if c.mapNames == nil || c.mapBounds == nil {
		observability.AddInt(metricHitsNoMapWiring, 1)
		slog.WarnContext(ctx, "killsource: precision par arme — collecteur sans resolution de carte "+
			"(WithPositionCapture absent), distances desactivees", "match_id", matchID)
		return decfilm.MapQuantEntry{}, false
	}
	noms, err := c.nomsDeCarteDuMatch(ctx, matchID)
	if err != nil {
		observability.AddInt(metricHitsNoMapName, 1)
		slog.InfoContext(ctx, "killsource: precision par arme — match sans nom de carte, distances desactivees",
			"match_id", matchID, "err", err)
		return decfilm.MapQuantEntry{}, false
	}
	entry, err := c.entreeDeCatalogueParNom(noms)
	if err != nil {
		observability.AddInt(metricHitsNoMapEntry, 1)
		slog.InfoContext(ctx, "killsource: precision par arme — carte hors catalogue de bornes, distances desactivees",
			"match_id", matchID, "err", err)
		return decfilm.MapQuantEntry{}, false
	}
	return entry, true
}

// hitsScanFailed compte et journalise un echec de scan, et rend le triplet d abandon (best-effort).
//
// Le premier resultat est TOUJOURS nil et c est le point : un abandon rend une passe VIDE, et le
// triplet epouse la signature des appelants (`return c.hitsScanFailed(...)`) pour que chaque site
// d echec tienne en une ligne. Exemption unparam datee 2026-09-02 (ratchet CI) — la scinder
// forcerait quatre `return nil, ...` sans rien clarifier.
//
//nolint:unparam // resultat 0 toujours nil par construction (triplet d abandon), cf. ci-dessus.
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
