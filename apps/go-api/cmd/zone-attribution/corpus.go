// corpus.go — QUELS matchs sont mesurables, et pourquoi les autres ne le sont pas.
//
// La chaine a quatre maillons, et chacun elimine du monde. Les compter separement est
// tout l'interet de ce fichier : « 8 matchs mesurables » ne dit rien, « 8 sur 76, dont
// 26 sans film et 30 sans bornes de carte » dit ou porter l'effort.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// candidate est un match du registre, avant tout filtrage.
type candidate struct {
	full    string
	short   string
	mapID   string
	mapName string
	variant string
	// pairName est le libelle de mode DU REGISTRE (« Arena:Total Control on Streets »). Il
	// sert au RECENSEMENT (census.go), qui le normalise comme le service — la mesure, elle,
	// classe par `game_variant_name` (cf. selectEligible). Les deux libelles ne portent pas le
	// mode dans le meme ordre, et c'est mesure : `Strongholds:Arena` contre `Arena:Strongholds`.
	pairName string
}

// rejects compte les maillons manquants, par motif.
type rejects struct {
	pasLeBonMode int
	sansFilm     int
	sansBornes   int
	sansFormes   int
}

// loadCandidates lit le registre. matchArg vide = tout le registre.
func loadCandidates(ctx context.Context, db *sql.DB, matchArg string) ([]candidate, error) {
	q := `SELECT match_id, COALESCE(map_id,''), COALESCE(map_name,''), COALESCE(game_variant_name,''),
	             COALESCE(pair_name,'')
	      FROM match_registry`
	var args []any
	if matchArg != "" {
		q += ` WHERE match_id = ? OR match_id LIKE ?`
		args = append(args, matchArg, matchArg+"%")
	}
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("lecture de match_registry : %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.full, &c.mapID, &c.mapName, &c.variant, &c.pairName); err != nil {
			return nil, err
		}
		c.short = title.FilmShortMatchID(c.full)
		out = append(out, c)
	}
	return out, rows.Err()
}

// selectEligible garde les matchs dont les QUATRE maillons sont la, et compte les autres.
//
// Le mode est classe par objectiveevents.ObjectiveTypeOf : le scan par mots-cles sur le
// nom de variante vit la-bas et nulle part ailleurs (« Arena:Strongholds »,
// « Strongholds:Arena », « Land Grab », « Total Control »...).
func (r *runner) selectEligible(all []candidate) []eligible {
	var out []eligible
	for _, c := range all {
		if objectiveevents.ObjectiveTypeOf(c.variant) != objectiveevents.ObjectiveTypeZone {
			r.rejects.pasLeBonMode++
			continue
		}
		if st, err := os.Stat(filmcache.ChunkDir(r.cacheDir, c.short)); err != nil || !st.IsDir() {
			r.rejects.sansFilm++
			continue
		}
		entry, err := r.bounds.Lookup(c.mapName)
		if err != nil {
			r.rejects.sansBornes++
			continue
		}
		catEntry, err := r.zones.Lookup(c.mapID)
		if err != nil {
			r.rejects.sansFormes++
			continue
		}
		set := catEntry.ZonesOfRole(r.role)
		if len(set.Zones) == 0 {
			r.rejects.sansFormes++
			continue
		}
		out = append(out, eligible{candidate: c, quant: &entry, zones: set})
	}
	return out
}

// eligible est un match dont les quatre maillons sont reunis.
type eligible struct {
	candidate
	// quant porte l'entree de catalogue ENTIERE : bornes ET largeurs d'axe. Les dissocier
	// laisserait armer les unes sans les autres (cf. replay.Options.MapQuant).
	quant *filmdec.MapQuantEntry
	zones replay.ZoneSet
}

// loadPlayerLines lit les lignes de match qui fondent le pont slot -> xuid.
//
// Le triplet (frags, morts, assistances) est la clé d'appariement : un slot dont le
// triplet ne designe pas UNE seule ligne n'est pas apparie (objectiveevents/slotidentity.go).
func loadPlayerLines(ctx context.Context, db *sql.DB, matchID string) ([]objectiveevents.PlayerLine, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT xuid, COALESCE(kills,0), COALESCE(deaths,0), COALESCE(assists,0)
		 FROM match_participants WHERE match_id = ?`, matchID)
	if err != nil {
		return nil, fmt.Errorf("lecture de match_participants : %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []objectiveevents.PlayerLine
	for rows.Next() {
		var l objectiveevents.PlayerLine
		if err := rows.Scan(&l.XUID, &l.Kills, &l.Deaths, &l.Assists); err != nil {
			return nil, err
		}
		l.XUID = strings.TrimSpace(l.XUID)
		out = append(out, l)
	}
	return out, rows.Err()
}
