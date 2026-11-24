package persist

// events_completion_surplus.go — QUELLES LIGNES DE LA COMPLÉTION S'INSÈRENT : LE SURPLUS SEULEMENT.
//
// La complétion (et le rejeu des événements, qui la rappelle sur un match déjà rempli) doit être
// idempotente : rejouer N fois ne change pas le nombre de lignes. Mais deux lignes IDENTIQUES d'un même
// lot sont parfois deux faits (deux médailles identiques au même instant, deux temps forts « mode »
// identiques : 224 groupes de médailles et 28 groupes « mode » légitimes sur la base du poste le
// 2026-11-24). Une clé « existe déjà ? » les confondrait. La règle est donc un COMPTE : par clé, le
// nombre de lignes du lot moins le nombre déjà en base ; on n'insère que ce qui dépasse.
//
// LA CLÉ
//   - hors médaille (élimination, mort, « mode », assistance...) : (type, instant, joueur). Ni le code
//     ni `raw_json` n'y entrent : un match écrit sous la forme « document » (raw_json plein sur les
//     éliminations et les morts), sous l'ancienne forme sans code, ou sous la forme de la complétion
//     est le même fait, et le rejeu ne le redouble pas ;
//   - médaille : (type, instant, joueur, code, nom de la médaille). Deux médailles distinctes du même
//     joueur au même instant sont deux faits. Le nom se lit dans `raw_json.medal_name` quelle que soit
//     la richesse du document ; absent, il vaut « » (médaille sans nom).
//
// UNE LIGNE DE L'ANCIENNE FORME (code et `raw_json` NULL tous deux, écrite avant le 2026-09-02) tient
// lieu de la ligne du film pour son (type, instant, joueur) : elle couvre, une fois chacune, les lignes
// de médaille de ce groupe qui n'ont pas de ligne identique, sinon un rejeu doublerait ces matchs.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"levelup/go-api/internal/domain/highlightevent"
)

// highlightGroup : (type, instant, joueur), la clé des faits hors médaille et le groupe des médailles.
type highlightGroup struct {
	eventType string
	timeMS    int
	xuid      string
}

// highlightMedal : la clé d'une médaille dans son groupe.
type highlightMedal struct {
	group    highlightGroup
	typeHint int
	name     string
}

// existingHighlight : ce que la base porte déjà pour le match.
type existingHighlight struct {
	groups map[highlightGroup]int // lignes par groupe, toute forme confondue (faits hors médaille)
	medals map[highlightMedal]int // médailles par (groupe, code, nom)
	bare   map[highlightGroup]int // lignes de l'ancienne forme, par groupe
}

// medalName rend le nom de médaille d'un raw_json ; « » quand il est absent.
func medalName(raw *string) string {
	if raw == nil || *raw == "" {
		return ""
	}
	var doc struct {
		MedalName string `json:"medal_name"`
	}
	if err := json.Unmarshal([]byte(*raw), &doc); err != nil {
		return *raw
	}
	return doc.MedalName
}

// loadExistingHighlights lit les lignes du match, dans la transaction du writer.
func loadExistingHighlights(ctx context.Context, tx *sql.Tx, matchID string) (existingHighlight, error) {
	ex := existingHighlight{
		groups: map[highlightGroup]int{}, medals: map[highlightMedal]int{}, bare: map[highlightGroup]int{},
	}
	rows, err := tx.QueryContext(ctx, `SELECT event_type, COALESCE(time_ms, -1), COALESCE(xuid, ''), type_hint, raw_json
		FROM highlight_events WHERE match_id = ?`, matchID)
	if err != nil {
		return ex, fmt.Errorf("persist: EventsCompletion lecture highlight_events %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var g highlightGroup
		var hint sql.NullInt64
		var raw sql.NullString
		if err := rows.Scan(&g.eventType, &g.timeMS, &g.xuid, &hint, &raw); err != nil {
			return ex, fmt.Errorf("persist: EventsCompletion scan highlight_events %s: %w", matchID, err)
		}
		ex.groups[g]++
		if !hint.Valid && !raw.Valid {
			ex.bare[g]++
			continue
		}
		if g.eventType == highlightevent.EventTypeMedal {
			var rawP *string
			if raw.Valid {
				rawP = &raw.String
			}
			ex.medals[highlightMedal{g, int(hint.Int64), medalName(rawP)}]++
		}
	}
	return ex, rows.Err()
}

// highlightSurplus rend, dans l'ordre du lot, les lignes à insérer : celles qui dépassent ce que la base
// porte déjà pour leur clé. Fonction pure.
func highlightSurplus(events []HLEventCompletion, ex existingHighlight) []HLEventCompletion {
	groups := make(map[highlightGroup]int, len(ex.groups))
	medals := make(map[highlightMedal]int, len(ex.medals))
	bare := make(map[highlightGroup]int, len(ex.bare))
	for k, v := range ex.groups {
		groups[k] = v
	}
	for k, v := range ex.medals {
		medals[k] = v
	}
	for k, v := range ex.bare {
		bare[k] = v
	}
	var out []HLEventCompletion
	for _, e := range events {
		g := highlightGroup{e.EventType, e.TimeMS, e.XUID}
		if e.EventType != highlightevent.EventTypeMedal {
			if groups[g] > 0 {
				groups[g]--
				continue
			}
			out = append(out, e)
			continue
		}
		m := highlightMedal{g, e.TypeHint, medalName(e.RawJSON)}
		switch {
		case medals[m] > 0:
			medals[m]--
		case bare[g] > 0:
			bare[g]--
		default:
			out = append(out, e)
		}
	}
	return out
}
