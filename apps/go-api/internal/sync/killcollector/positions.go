package killcollector

// positions.go — LE PRODUCTEUR DE `shared.kill_positions` POUR HALO INFINITE : les coordonnées
// monde (x,y,z) du tueur et de la victime par kill, décodées du MÊME film que les morts et les
// tirs (G.2bis — dernière pièce de la conception G.0/G.2 : la dette bloquante « table pas
// append-only » est fermée, ce fichier câble enfin la CAPTURE).
//
// # LE FILM EST CHARGÉ UNE FOIS, ET LE PONT DISQUE A DISPARU
//
// `analysis/replay`/`analysis/filmdec` exposent QUATRE lectures du film : ScanBipedPositions,
// ScanClockOrigin, ScanPlayerIndices, ScanDeaths. Le collecteur, lui, tient les chunks EN MÉMOIRE
// PURE (téléchargés par `FilmChunksForMatch`) — il ne les a jamais écrits sur disque.
//
// HISTORIQUE, ET CE QUI L'A FERMÉ. Deux ponts étaient possibles (plan G.2, §3bis) : (a) écrire les
// chunks déjà en mémoire dans un répertoire temporaire et appeler les quatre fonctions `ScanFilm*`
// telles quelles ; (b) des variantes mémoire de chacune. (b) était écarté parce que ç'aurait été un
// DEUXIÈME décodeur des mêmes octets pour chacune des quatre lectures — la règle qui gouverne tout
// ce chantier (« deux décodeurs du même fait divergeraient », répétée dans
// killpos.go/deaths_source.go/identity.go) l'interdit. (a) a donc tenu jusqu'au lot 1 de
// PLAN_CUISSON_PERF (item 1.6, 2026-09-02) : quatre écritures de fichiers, puis QUATRE relectures
// et QUATRE décompressions du film entier.
//
// Le lot 1 a rendu (b) possible SANS second décodeur : `internal/analysis/filmsource` est la source
// unique du film (une décompression, un découpage en paquets, une grammaire), et les quatre
// balayages prennent désormais un `*filmsource.Film`. Le collecteur charge donc le film UNE fois
// pour les morts (`killsource.Decode`) et le repasse tel quel ici. Plus de répertoire temporaire,
// plus de disque plein possible — et le seul refus qui reste est celui qui protégeait d'une
// position fausse : la séquence trouée (cf. refuserSequenceTrouee).
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
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// Compteurs de sante de la passe de positions (ADR 0009 : entiers, snake_case, aucun ratio).
const (
	metricPositionsMatches  = "killsource_positions_matchs_couverts"
	metricPositionsRows     = "killsource_positions_lignes_ecrites"
	metricPositionsNoMap    = "killsource_positions_sans_carte"
	metricPositionsNoOrigin = "killsource_positions_sans_horloge"
	// metricPositionsBridgeFail : le film n'a pas pu être présenté aux balayages. Le nom date du
	// pont disque (supprimé au lot 1) ; il reste INCHANGÉ parce qu'un compteur publié est une
	// interface, et sa seule cause aujourd'hui est la séquence trouée.
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
	ctx context.Context, matchID string, film *filmsource.Film,
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

	if err := refuserSequenceTrouee(film); err != nil {
		observability.AddInt(metricPositionsBridgeFail, 1)
		slog.ErrorContext(ctx, "killsource: positions — film inexploitable",
			"match_id", matchID, "err", err)
		return
	}

	rep, rows, err := buildPositionRows(film, entry, ids, kills, matchID)
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

// buildPositionRows : les QUATRE lectures du film + la composition pure. Découpée de
// [collectPositions] pour rester sous le plafond de longueur du dépôt (80 lignes) — chaque refus
// reste journalisable par l appelant, jamais avalé ici.
//
// LES QUATRE BALAYAGES PARTAGENT LE FILM DÉJÀ CHARGÉ (lot 1, item 1.6) : ils prenaient chacun un
// répertoire et relisaient le film entier depuis le disque, décompression comprise.
//
// TIENT LE VERROU DE DÉCODAGE DU PROCESS, comme son frère `buildHitsBatches` (hits.go) et comme
// le contrat de `filmdec/decode_gate.go:16-18` l'exige : « tout chemin qui enchaîne les balayages
// de ce paquet acquiert ce verrou pour TOUTE la durée du décodage d'un film ». Ce chemin-ci
// enchaîne QUATRE balayages sur des globaux de paquet et ne le prenait pas — asymétrie relevée
// au registre (E5) et corrigée le 2026-09-05 (lot E, item E.5).
//
// PAS DE RÉ-ENTRANCE : le mutex n'est pas réentrant, et `killsource.Decode` — le seul autre
// preneur du chemin `collect()` — le relâche AVANT de rendre (`killsource/decode.go:78-79`,
// `defer release()` sur une fonction qui retourne). `collectPositions` est appelé après lui,
// jamais dedans.
func buildPositionRows(
	film *filmsource.Film, entry filmdec.MapQuantEntry, ids MatchIdentities,
	kills []replay.KillRef, matchID string,
) (replay.KillPosReport, []persist.KillPositionInsert, error) {
	release := filmdec.LockProcessDecode()
	defer release()

	bipedOpt := filmdec.DefaultScanFilmOptions()
	rng := entry.Range()
	bipedOpt.WorldRange = &rng
	positions, err := filmdec.ScanBipedPositions(film, bipedOpt)
	if err != nil {
		return replay.KillPosReport{}, nil, fmt.Errorf("positions bipeds: %w", err)
	}

	originUS, err := replay.ScanClockOrigin(film)
	if err != nil {
		observability.AddInt(metricPositionsNoOrigin, 1)
		return replay.KillPosReport{}, nil, fmt.Errorf("horloge du film: %w", err)
	}

	deathsFilm, err := replay.ScanDeaths(film)
	if err != nil {
		return replay.KillPosReport{}, nil, fmt.Errorf("fil des morts (rejeu): %w", err)
	}

	idx, err := replay.ScanPlayerIndices(film, rosterUint64(ids.XUIDs))
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

// refuserSequenceTrouee : REFUS SUR TROU DE SEQUENCE, le seul controle du disparu pont disque
// qui protegeait d une position FAUSSE — et il survit tel quel, sans disque.
//
// `FilmOf` (bridge.go) tolere des index non contigus (les trous restent des chunks VIDES,
// `killsource.Decode` fait de l acces direct par index) ; les QUATRE balayages, eux, parcourent
// les chunks de donnees par numero (`filmdec.FilmChunkNumbers`) et un chunk vide ne rend aucun
// paquet — un film troue leur ferait donc lire un film AMPUTE, en silence, jamais une erreur. Le
// controle ci-dessous refuse ce cas au lieu de le laisser produire une lecture partielle
// plausible : le critere de ce chantier est qu aucune position fausse ne soit possible, un film
// incomplet perd donc SES positions plutot que d en risquer de fausses.
//
// LA REGLE EST CELLE D AVANT, A L IDENTIQUE : le controle porte sur les chunks de DONNEES
// (numeros 1..N), jamais sur l en-tete — c est ce que faisait `filmdec.CountFilmChunks`, qui
// comptait a partir de `chunk_01.bin`. Un film reduit au seul chunk 0, ou vide, passe donc ici
// et se fait refuser par les balayages eux-memes (`ErrNoFilmChunk`).
func refuserSequenceTrouee(film *filmsource.Film) error {
	if film == nil {
		return fmt.Errorf("film absent")
	}
	for i := 1; i < film.NumChunks(); i++ {
		if len(film.Chunk(i)) == 0 {
			return fmt.Errorf("sequence de chunks trouee (chunk %d absent sur %d attendus depuis "+
				"l index 1) — positions ignorees plutot que lues sur un film ampute",
				i, film.NumChunks()-1)
		}
	}
	return nil
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
