package duckdb

// tactical_repo_ripostes.go — LE JOURNAL PUBLIABLE DES MATCHS D'UNE ZONE, pour savoir si le tueur d'une
// mort a ete abattu par un coequipier de la victime, et si l'auteur d'un frag a ete abattu dans la
// foulee (`analysis/coordination.TueurAbattu` / `AbattuEnRetour`).
//
// Deux vues `_latest` : `match_kill_events_latest` (qui a tue qui, quand — le tueur d'un bot est
// identifie par son gamertag, `feed_killer_xuid` NULL) et `kill_positions_latest` (ou tirait le
// tueur). LE JOURNAL EST CELUI QUI FAIT FOI SUR LES MORTS, filtre `publishable` compris (une passe
// non publiable est juste en agregat et fausse ligne par ligne).
//
// LE LEFT JOIN SUR LES POSITIONS EST DELIBERE : un coup sans position mesuree (victime bot, film
// sans position) reste un coup — le vengeur existe, seul son point manque. L'INNER JOIN de
// `QTacticalIsolement` ecarterait ce coup et ferait perdre une riposte. La jointure passe par le
// tueur credite (cle de `kill_positions`) et le meme instant ; `GROUP BY` sur l'identite du kill
// evite qu'une position en double ne duplique la ligne.
//
// LA LISTE BLANCHE EST EXIGEE ET LIEE EN CONSTANTES, deux fois (une par vue), pour que la fenetre
// de chaque `_latest` ne voie que les matchs de la zone (ADR 0036 I2).

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// QTacticalJournalRipostes : %s = la table de positions, puis la liste des matchs pour chaque vue.
const QTacticalJournalRipostes = `
SELECT e.match_id, e.time_ms, COALESCE(e.victim_xuid, '') AS victim_xuid, e.victim_gamertag,
       COALESCE(e.feed_killer_xuid, '') AS killer_xuid, COALESCE(e.feed_killer_gamertag, '') AS killer_gamertag,
       min(p.killer_x) AS killer_x, min(p.killer_y) AS killer_y
FROM match_kill_events_latest e
LEFT JOIN %s p
  ON p.match_id = e.match_id AND p.killer_xuid = e.feed_killer_xuid AND p.time_ms = e.time_ms
 AND p.match_id IN (%s)
WHERE e.match_id IN (%s)
  AND e.publishable
GROUP BY e.id, e.match_id, e.time_ms, e.victim_xuid, e.victim_gamertag, e.feed_killer_xuid, e.feed_killer_gamertag
ORDER BY e.match_id, e.time_ms, e.id`

// JournalDesKills rend les kills publiables des matchs de `q.Matchs` (liste blanche obligatoire ;
// vide = aucune requete), avec la position du tueur au coup quand elle est mesuree. Table absente ->
// ErrCapabilityNotSupported.
func (r *TacticalRepo) JournalDesKills(ctx context.Context, q domain.TacticalQuery) ([]domain.KillDuJournal, error) {
	if err := exigerLaListe(q, "JournalDesKills"); err != nil {
		return nil, err
	}
	ids := q.Matchs.IDs()
	if len(ids) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()
	db, release, err := r.ouvrir(ctx, q, "JournalDesKills")
	if err != nil {
		return nil, err
	}
	defer release()

	args := append(ToAnySlice(ids), ToAnySlice(ids)...)
	// Le nom de la table de positions passe par la constante de kill_measured.go, comme
	// QTacticalPositions et QTacticalIsolement.
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(QTacticalJournalRipostes, positionsAtKill, Placeholders(len(ids)), Placeholders(len(ids))), args...)
	if err != nil {
		return nil, r.degrader(ctx, "JournalDesKills", err)
	}
	var out []domain.KillDuJournal
	err = scanRows(ctx, rows, "TacticalRepo.JournalDesKills", func(sc rowScanner) error {
		var k domain.KillDuJournal
		var x, y sql.NullFloat64
		if err := sc.Scan(&k.MatchID, &k.TimeMs, &k.VictimXUID, &k.VictimGamertag,
			&k.KillerXUID, &k.KillerGamertag, &x, &y); err != nil {
			return err
		}
		k.KillerX, k.KillerY = nullFloatPtr(x), nullFloatPtr(y)
		out = append(out, k)
		return nil
	})
	return out, err
}
