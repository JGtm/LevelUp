package killcollector

// positions.go — LE PRODUCTEUR DE `shared.kill_positions` POUR HALO INFINITE : les coordonnées
// monde (x,y,z) du tueur et de la victime par kill, décodées du MÊME film que les morts et les
// tirs (G.2bis — dernière pièce de la conception G.0/G.2 : la dette bloquante « table pas
// append-only » est fermée, ce fichier câble enfin la CAPTURE).
//
// # POURQUOI UN PONT DISQUE, ET PAS UNE VARIANTE MÉMOIRE
//
// `analysis/replay`/`analysis/filmdec` exposent QUATRE lectures « hors ligne » (I/O disque,
// jamais depuis un chemin de requête, cf. leurs propres en-têtes) : ScanFilmBipedPositions,
// ScanFilmClockOrigin, ScanFilmPlayerIndices, ScanFilmDeaths. Le collecteur, lui, décode
// aujourd'hui les chunks d'un film EN MÉMOIRE PURE (`killsource.Decode(..., ChunkSourceOf(chunks),
// nil)`) — il ne les a jamais écrits sur disque.
//
// Deux ponts étaient possibles (cf. plan G.2, §3bis) : (a) écrire les chunks déjà en mémoire dans
// un répertoire temporaire et appeler les quatre fonctions exportées telles quelles ; (b) des
// variantes mémoire de chacune. (b) est écarté : ce serait un DEUXIÈME décodeur des mêmes octets
// pour chacune des quatre lectures — la règle qui gouverne tout ce chantier (« deux décodeurs du
// même fait divergeraient », répétée dans killpos.go/deaths_source.go/identity.go) l'interdit. (a)
// ne décode RIEN : il recopie des octets déjà téléchargés vers le nom de fichier attendu
// (`chunk_NN.bin`, identique au cache film hérité — cf. writeChunksToTempDir). La seule façon
// dont il peut échouer est un disque plein ou un chemin illisible, jamais une position fausse.
//
// # LE PONT SLOT -> XUID EST CELUI DU REJEU 2D, PAS UNE RÉSOLUTION LOCALE
//
// `replay.BuildKillPositions` a besoin d'un `slotXUID map[uint32]uint64` (un joueur change de
// slot à chaque réapparition). Ce pont existe déjà, LU et pas voté, dans `analysis/replay`
// (owners.go : fil des morts + index de joueur, mesuré à 90/105 vies nommées avec 0 collision sur
// le film témoin). Une résolution locale à ce paquet serait, encore, un second décodeur du même
// fait — c'est pourquoi `replay.ResolveSlotXUID` a été exportée (killpos_bridge.go) plutôt que
// réimplémentée ici.
//
// # LA CAPABILITY, ET CE QU'ELLE NE GARANTIT PAS SEULE
//
// `games.CapFilmKillPositions` dit que le titre EXPOSE la capture (Infinite) ; elle ne dit rien de
// savoir si CE collecteur a été câblé avec une résolution de carte (WithPositionCapture). Les deux
// conditions sont nécessaires : un titre qui n'a pas la capability n'essaie jamais ; un titre qui
// l'a mais dont le collecteur n'a pas reçu WithPositionCapture (CLI qui ne l'appelle pas, test) se
// dégrade silencieusement AU NIVEAU CONFIGURATION — situation journalisée en Debug, jamais en
// erreur : ce n'est pas une panne, c'est une passe qui n'a pas demandé cette donnée.
//
// # BEST-EFFORT ASSUMÉ, MÊME DOCTRINE QUE shots.go
//
// Les morts sont la donnée qui motive tout le chantier ; les positions sont un enrichissement
// TROISIÈME (après le crédit, après la source du dégât). Un échec ICI ne doit JAMAIS faire
// échouer la passe de morts, déjà écrite quand ce code s'exécute (cf. l'appel dans collector.go,
// après c.write). Chaque sortie anticipée journalise sa cause et compte un métrique dédié — un
// zéro qu'on ne peut interroger qu'en relisant le code n'alerte personne.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/haloclient"
)

// Compteurs de sante de la passe de positions (ADR 0009 : entiers, snake_case, aucun ratio).
const (
	metricPositionsMatches      = "killsource_positions_matchs_couverts"
	metricPositionsRows         = "killsource_positions_lignes_ecrites"
	metricPositionsNoMap        = "killsource_positions_sans_carte"
	metricPositionsNoOrigin     = "killsource_positions_sans_horloge"
	metricPositionsBridgeFail   = "killsource_positions_pont_echec"
	metricPositionsAmbiguous    = "killsource_positions_index_ambigus"
	metricPositionsNoBridge     = "killsource_positions_sans_pont_identite"
	metricPositionsWriteFail    = "killsource_positions_erreurs_ecriture"
	metricPositionsKillsDropped = "killsource_positions_morts_sans_position"
)

// collectPositions : la TROISIÈME écriture de la passe — `shared.kill_positions`.
//
// `deaths` est le SOUS-ENSEMBLE PRÉ-FUSION (batch.Deaths dans collect(), avant
// MergeCreditAndFilm) : les positions n'existent QUE pour les kills que CE FILM a lui-même
// décodés — un kill récupéré par le producteur crédit-seul (highlight_events, sans film) n'a
// structurellement aucune position à offrir. Utiliser la liste pré-fusion n'est donc pas une
// approximation, c'est la population exacte qui peut avoir une position.
func (c *KillSourceCollector) collectPositions(
	ctx context.Context, matchID string, chunks []haloclient.FilmChunk,
	ids MatchIdentities, deaths []persist.KillEventInsert,
) {
	if !c.caps.Has(games.CapFilmKillPositions) {
		slog.DebugContext(ctx, "killsource: positions — capability absente, passe ignoree",
			"match_id", matchID, "capability", string(games.CapFilmKillPositions),
			"err", games.ErrCapabilityNotSupported)
		return
	}
	if c.mapNames == nil || c.mapBounds == nil {
		slog.DebugContext(ctx, "killsource: positions — collecteur non cable (WithPositionCapture "+
			"absent), passe ignoree", "match_id", matchID)
		return
	}

	kills := killRefsFromDeaths(deaths)
	if len(kills) == 0 {
		slog.DebugContext(ctx, "killsource: positions — aucune mort avec tueur ET victime resolus, "+
			"rien a placer", "match_id", matchID)
		return
	}

	entry, err := c.resolveMapBounds(ctx, matchID)
	if err != nil {
		observability.AddInt(metricPositionsNoMap, 1)
		slog.InfoContext(ctx, "killsource: positions — carte hors catalogue de bornes, passe ignoree",
			"match_id", matchID, "err", err)
		return
	}

	dir, cleanup, err := writeChunksToTempDir(chunks)
	if err != nil {
		observability.AddInt(metricPositionsBridgeFail, 1)
		slog.ErrorContext(ctx, "killsource: positions — pont disque echoue",
			"match_id", matchID, "err", err)
		return
	}
	defer cleanup()

	rep, rows, err := buildPositionRows(dir, entry, ids, kills, matchID)
	if err != nil {
		slog.WarnContext(ctx, "killsource: positions — passe ignoree", "match_id", matchID, "err", err)
		return
	}

	if err := c.writePositions(ctx, matchID, rows); err != nil {
		observability.AddInt(metricPositionsWriteFail, 1)
		slog.ErrorContext(ctx, "killsource: positions — ecriture echouee",
			"match_id", matchID, "err", err)
		return
	}
	publishPositionsPass(ctx, matchID, rep, len(rows))
}

// resolveMapBounds : les identites de carte candidates du match (base), puis leurs bornes de
// dequantification (catalogue). Deux echecs distincts, une seule cause utile a l appelant :
// « pas de bornes », qu il vienne de la base ou du catalogue.
func (c *KillSourceCollector) resolveMapBounds(ctx context.Context, matchID string) (filmdec.MapQuantEntry, error) {
	keys, err := c.mapNames.MapKeysForMatch(ctx, matchID)
	if err != nil {
		return filmdec.MapQuantEntry{}, fmt.Errorf("identite de carte: %w", err)
	}
	for _, name := range keys.Names {
		if name == "" {
			continue
		}
		if entry, err := c.mapBounds.Lookup(name); err == nil {
			return entry, nil
		}
	}
	return filmdec.MapQuantEntry{}, fmt.Errorf("%w (candidats: %v)", filmdec.ErrUnknownMapBounds, keys.Names)
}

// buildPositionRows : les QUATRE lectures hors ligne + la composition pure, une fois le pont
// disque en place. Découpée de [collectPositions] pour rester sous le plafond de longueur du
// dépôt (80 lignes) — chaque refus reste journalisable par l appelant, jamais avalé ici.
func buildPositionRows(
	dir string, entry filmdec.MapQuantEntry, ids MatchIdentities, kills []replay.KillRef, matchID string,
) (replay.KillPosReport, []persist.KillPositionInsert, error) {
	bipedOpt := filmdec.DefaultScanFilmOptions()
	rng := entry.Range()
	bipedOpt.WorldRange = &rng
	positions, err := filmdec.ScanFilmBipedPositions(dir, bipedOpt)
	if err != nil {
		return replay.KillPosReport{}, nil, fmt.Errorf("positions bipeds: %w", err)
	}

	originUS, err := replay.ScanFilmClockOrigin(dir)
	if err != nil {
		observability.AddInt(metricPositionsNoOrigin, 1)
		return replay.KillPosReport{}, nil, fmt.Errorf("horloge du film: %w", err)
	}

	deathsFilm, err := replay.ScanFilmDeaths(dir)
	if err != nil {
		return replay.KillPosReport{}, nil, fmt.Errorf("fil des morts (rejeu): %w", err)
	}

	idx, err := replay.ScanFilmPlayerIndices(dir, rosterUint64(ids.XUIDs))
	if err != nil {
		return replay.KillPosReport{}, nil, fmt.Errorf("index de joueur: %w", err)
	}
	if idx.Disagreements > 0 {
		observability.AddInt(metricPositionsAmbiguous, int64(idx.Disagreements))
	}

	slotXUID, owners := replay.ResolveSlotXUID(positions, deathsFilm, idx)
	if len(slotXUID) == 0 {
		observability.AddInt(metricPositionsNoBridge, 1)
		return replay.KillPosReport{}, nil, fmt.Errorf(
			"pont slot->xuid vide (vies=%d nommees=%d lectures_index=%d)",
			owners.LivesTotal, owners.DeathsNamed, owners.IndexReadings)
	}

	posOut, rep := replay.BuildKillPositions(positions, slotXUID, kills, int64(originUS))
	return rep, toKillPositionRows(matchID, posOut), nil
}

// writePositions : l ecriture, sous son PROPRE lease court — meme raison que writeShots (le
// lease RW de shared est la ressource la plus disputee du process, ADR 0013).
func (c *KillSourceCollector) writePositions(ctx context.Context, matchID string, rows []persist.KillPositionInsert) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", matchID, err)
	}
	defer release()
	return persist.NewKillPositionPersister(db).PersistPass(ctx, matchID, rows)
}

// killRefsFromDeaths ne garde que les morts dont LES DEUX identites sont resolues — un xuid vide
// (bot, nom non resolu) n a pas de position a chercher, et BuildKillPositions ne sait rien faire
// d un KillRef a zero.
func killRefsFromDeaths(deaths []persist.KillEventInsert) []replay.KillRef {
	out := make([]replay.KillRef, 0, len(deaths))
	for i := range deaths {
		d := &deaths[i]
		killerXUID, ok1 := parseXUID(d.FeedKillerXUID)
		victimXUID, ok2 := parseXUID(d.VictimXUID)
		if !ok1 || !ok2 {
			continue
		}
		out = append(out, replay.KillRef{KillerXUID: killerXUID, VictimXUID: victimXUID, TimeMS: int64(d.TimeMS)})
	}
	return out
}

// rosterUint64 traduit les xuids texte de MatchIdentities pour ScanFilmPlayerIndices, qui lit un
// roster numerique (le film ne porte que des motifs de bits, jamais une chaine).
func rosterUint64(xuids []string) []uint64 {
	out := make([]uint64, 0, len(xuids))
	for _, s := range xuids {
		if v, ok := parseXUID(s); ok {
			out = append(out, v)
		}
	}
	return out
}

// parseXUID : un xuid est une suite de chiffres decimale, et rien d autre — meme regle que
// identities.go (estDecimal), reappliquee ici parce que cette lecture est numerique alors que
// MatchIdentities.Resoudre rend des chaines.
func parseXUID(s string) (uint64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// toKillPositionRows traduit le resultat pur en lignes ecrivables. AUCUNE ligne pour une mort
// dont ni le tueur ni la victime n ont ete localises — BuildKillPositions ne les rend deja pas.
func toKillPositionRows(matchID string, positions []replay.KillPosition) []persist.KillPositionInsert {
	out := make([]persist.KillPositionInsert, 0, len(positions))
	for i := range positions {
		p := &positions[i]
		row := persist.KillPositionInsert{
			MatchID:    matchID,
			KillerXUID: strconv.FormatUint(p.KillerXUID, 10),
			TimeMS:     int(p.TimeMS),
		}
		if p.Killer != nil {
			row.KillerX, row.KillerY, row.KillerZ = &p.Killer.X, &p.Killer.Y, &p.Killer.Z
		}
		if p.Victim != nil {
			row.VictimX, row.VictimY, row.VictimZ = &p.Victim.X, &p.Victim.Y, &p.Victim.Z
		}
		out = append(out, row)
	}
	return out
}

// writeChunksToTempDir ecrit les chunks DEJA EN MEMOIRE dans un repertoire temporaire, au format
// chunk_NN.bin attendu par les scanners disque (identique au cache film herite,
// haloclient.LocalFilmCache — memes octets, zlib ou non : ReadFilmChunk decompresse a la lecture,
// et l operation est idempotente sur des octets deja clairs). AUCUN DECODAGE : un pont, rien de
// plus — la seule facon d echouer est un disque plein ou un chemin illisible.
//
// REFUS SUR TROU DE SEQUENCE. `ChunkSourceOf` (bridge.go) tolere des index non contigus (les
// trous restent nil, `killsource.Decode` fait de l acces direct par index) ; les QUATRE scanners
// disque, eux, comptent les chunks par PREFIXE CONTIGU (filmdec.CountFilmChunks s arrete au
// premier index manquant — c est cette meme fonction que ScanFilmDeaths utilise pour trouver le
// DERNIER chunk, cense etre le kill-feed). Un film troue verrait donc ces quatre lectures
// s arreter prematurement, en silence, sur un PREFIXE du film — jamais une erreur. Le controle
// ci-dessous refuse ce cas au lieu de le laisser produire une lecture partielle plausible : le
// critere de ce chantier est qu aucune position fausse ne soit possible, un film incomplet perd
// donc SES positions plutot que d en risquer de fausses.
func writeChunksToTempDir(chunks []haloclient.FilmChunk) (string, func(), error) {
	maxIdx := 0
	for _, c := range chunks {
		if c.Index > maxIdx {
			maxIdx = c.Index
		}
	}
	dir, err := os.MkdirTemp("", "levelup-killpos-")
	if err != nil {
		return "", nil, fmt.Errorf("mkdir temporaire: %w", err)
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.Warn("killsource: positions — nettoyage du repertoire temporaire echoue",
				"dir", dir, "err", err)
		}
	}

	for _, c := range chunks {
		if c.Index < 0 || len(c.Data) == 0 {
			continue
		}
		path := filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", c.Index))
		if err := os.WriteFile(path, c.Data, 0o600); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("ecriture %s: %w", path, err)
		}
	}

	if got := filmdec.CountFilmChunks(dir); got != maxIdx {
		cleanup()
		return "", nil, fmt.Errorf("sequence de chunks trouee (attendu %d chunks contigus depuis "+
			"l index 1, filmdec en compte %d) — positions ignorees plutot que lues sur un prefixe "+
			"du film", maxIdx, got)
	}
	return dir, cleanup, nil
}

// publishPositionsPass : les compteurs de sante (ADR 0009) et la trace de la passe.
func publishPositionsPass(ctx context.Context, matchID string, rep replay.KillPosReport, rowsWritten int) {
	observability.AddInt(metricPositionsMatches, 1)
	observability.AddInt(metricPositionsRows, int64(rowsWritten))
	if rep.Dropped > 0 {
		observability.AddInt(metricPositionsKillsDropped, int64(rep.Dropped))
	}
	slog.InfoContext(ctx, "killsource: positions decodees",
		"match_id", matchID, "kills", rep.Kills, "deux_cotes", rep.Both,
		"tueur_seul", rep.KillerOnly, "victime_seule", rep.VictimOnly,
		"sans_position", rep.Dropped, "sans_pont_identite", rep.NoBridge, "lignes", rowsWritten)
}
