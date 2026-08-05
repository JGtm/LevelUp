package persist

// kill_events_film_pass.go — RELIRE L ENRICHISSEMENT DEJA EN BASE, sans re-decoder un seul film.
//
// C EST CE QUI REND L INVERSION DE PRESEANCE BON MARCHE. L enrichissement des 949 matchs decodes
// est DEJA en base, indexe par `(match_id, time_ms)` : recomposer une passe credit-base ne demande
// ni les 23 Go de films en cache, ni les 3 a 11 heures de decodage, ni un token. La recomposition
// est SQL -> memoire -> SQL.
//
// CE QUE CETTE LECTURE REND : les lignes de la passe COURANTE du match dont la voie est une voie de
// FILM (`persist.FilmReadPaths`). Elle ne rend jamais les lignes credit de la meme passe — sur une
// passe deja fusionnee elles seraient la base, pas l enrichissement, et les relire comme film ferait
// tourner la fusion sur elle-meme.
//
// ⚠ LECTURE PAR LA VUE `_latest`, JAMAIS PAR LA TABLE (ADR 0026). Une lecture brute servirait les
// lignes de passes precedentes en plus de la courante : la fusion enrichirait alors une mort avec
// une mesure perimee, et l ambiguite d instant ferait taire l enrichissement sans que rien ne le
// dise.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// rowQuerier : ce dont la lecture a besoin, et rien de plus. `*sql.DB` comme `*sql.Tx` le
// satisfont — le producteur live lit dans la transaction du match, les producteurs hors ligne
// lisent sur leur handle.
type rowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// filmPassColumns : les colonnes de l enrichissement, dans l ordre du scan.
//
// `decoder_rev` et `publishable` sont des attributs de la PASSE, pas de la ligne : ils sont lus
// sur chaque ligne et la derniere gagne, ce qui est exact puisque la vue `_latest` ne rend qu une
// passe et qu une passe porte la meme valeur sur toutes ses lignes.
const filmPassColumns = `
	decoder_rev, publishable, time_ms,
	victim_gamertag, victim_xuid,
	feed_killer_gamertag, feed_killer_xuid, feed_present,
	assist_gamertag, assist_xuid, assist_known, assist_index, assist_rejected, assist_extra_count,
	source_tag, source_category, killer_damage_pct, assist_damage_pct,
	diverges, read_path, read_origin`

// FilmPassForMatch rend, sous forme de [KillSourceBatch], les lignes de FILM de la passe courante
// du match. Batch vide (aucune mort) quand le match n en porte aucune — le cas majoritaire.
//
// La passe rendue n est PAS ecrivable telle quelle : elle sert d ENTREE a [MergeCreditAndFilm],
// qui la pose sur une base credit. Son `CreditBaseCount` reste donc a zero, et c est correct :
// elle ne porte aucune base.
func FilmPassForMatch(ctx context.Context, q rowQuerier, matchID string) (KillSourceBatch, error) {
	marques := make([]string, 0, len(FilmReadPaths))
	args := make([]any, 0, len(FilmReadPaths)+1)
	args = append(args, matchID)
	for _, p := range FilmReadPaths {
		marques = append(marques, "?")
		args = append(args, p)
	}

	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s FROM match_kill_events_latest
		WHERE match_id = ? AND read_path IN (%s)
		ORDER BY time_ms, id`, filmPassColumns, strings.Join(marques, ", ")), args...)
	if err != nil {
		return KillSourceBatch{}, fmt.Errorf("persist: passe film %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	out := KillSourceBatch{MatchID: matchID}
	for rows.Next() {
		d, rev, publiable, errScan := scanFilmPassRow(rows)
		if errScan != nil {
			return KillSourceBatch{}, fmt.Errorf("persist: passe film %s (scan): %w", matchID, errScan)
		}
		out.DecoderRev, out.Publishable = rev, publiable
		out.Deaths = append(out.Deaths, d)
	}
	if err := rows.Err(); err != nil {
		return KillSourceBatch{}, fmt.Errorf("persist: passe film %s (rows): %w", matchID, err)
	}
	return out, nil
}

// scanFilmPassRow : une ligne de film relue.
//
// TOUTES LES COLONNES NULLABLES PASSENT PAR UN `sql.Null*`, ET C EST LE POINT DU FICHIER. Scanner
// `source_tag` dans un `uint32` non nullable ecrirait 0 sur un NULL — et 0 veut dire « source non
// mesuree » : la fusion recopierait alors une non-mesure comme si c en etait une. Idem pour les
// parts de degats (nil = non mesure, jamais zero) et pour `diverges` (NULL = non mesurable).
func scanFilmPassRow(rows *sql.Rows) (KillEventInsert, string, bool, error) {
	var (
		d                             KillEventInsert
		rev                           string
		publiable                     bool
		victimXUID, killerGT, killerX sql.NullString
		assistGT, assistX, rejected   sql.NullString
		assistIndex                   sql.NullInt64
		assistExtra                   sql.NullInt64
		sourceTag                     sql.NullInt64
		sourceCat                     sql.NullString
		killerPct, assistPct          sql.NullInt64
		diverges                      sql.NullBool
	)
	err := rows.Scan(
		&rev, &publiable, &d.TimeMS,
		&d.VictimGamertag, &victimXUID,
		&killerGT, &killerX, &d.FeedPresent,
		&assistGT, &assistX, &d.AssistKnown, &assistIndex, &rejected, &assistExtra,
		&sourceTag, &sourceCat, &killerPct, &assistPct,
		&diverges, &d.ReadPath, &d.ReadOrigin)
	if err != nil {
		return d, "", false, err
	}
	d.VictimXUID, d.FeedKillerGamertag, d.FeedKillerXUID = victimXUID.String, killerGT.String, killerX.String
	d.AssistGamertag, d.AssistXUID, d.AssistRejected = assistGT.String, assistX.String, rejected.String
	if assistIndex.Valid {
		idx := int(assistIndex.Int64)
		d.AssistIndex = &idx
	}
	d.AssistExtra = int(assistExtra.Int64)
	d.SourceTag, d.SourceCategory = uint32(sourceTag.Int64), sourceCat.String
	d.KillerDamagePct, d.AssistDamagePct = pctFromNull(killerPct), pctFromNull(assistPct)
	d.Diverges = diverges.Bool
	return d, rev, publiable, nil
}

// pctFromNull : NULL -> nil. « Non mesure » n est jamais zero — et un zero mesure existe.
func pctFromNull(v sql.NullInt64) *uint8 {
	if !v.Valid {
		return nil
	}
	p := uint8(v.Int64)
	return &p
}
