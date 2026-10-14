package duckdb

// tactical_repo_coequipiers.go — LES COEQUIPIERS A LA MORT, pour le detail d'une zone du plan
// tactique : etat et position de chaque coequipier de la victime a l'instant de sa mort.
//
// Lecture de `match_death_mates_latest` SEULE (derniere passe entiere par match, ADR 0026). LA LISTE
// BLANCHE EST EXIGEE ET LIEE EN CONSTANTES sur le `match_id` de la vue : c'est la seule forme que
// DuckDB pousse sous la fenetre de la derniere passe (ADR 0036 I2, garde-rail
// `TestTacticalRepo_MatesDesMorts_BorneEtListe`). Les victimes demandees bornent en plus la lecture
// aux morts de la zone ; l'appariement exact (match, victime, instant) est fait ICI, sur la cle de
// la ligne de contexte que l'appelant a deja appariee a la mort (a ± 1,5 s) : la base rend les
// lignes de cette cle-la, elle ne rapproche aucune horloge.
//
// LE LEFT JOIN SUR LES PARTICIPANTS EST DELIBERE : il ne sert qu'a nommer le coequipier. Un
// coequipier que le registre du match ne nomme pas garde sa ligne (nom vide) — l'INNER JOIN le
// ferait disparaitre et la somme des lignes ne retomberait plus sur `teammates_total`.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// QTacticalMates : %s = la liste des matchs, puis celle des victimes, en constantes liees.
const QTacticalMates = `
SELECT m.match_id, m.victim_xuid, m.time_ms, m.mate_xuid, COALESCE(p.gamertag, '') AS gamertag,
       m.state, m.x, m.y, m.distance_m
FROM match_death_mates_latest m
LEFT JOIN match_participants p ON p.match_id = m.match_id AND p.xuid = m.mate_xuid
WHERE m.match_id IN (%s)
  AND m.victim_xuid IN (%s)
ORDER BY m.match_id, m.victim_xuid, m.time_ms, m.mate_xuid`

// MatesDesMorts rend les coequipiers des morts `morts` (liste blanche `q.Matchs` obligatoire ; aucune
// mort = aucune requete). Seules les lignes dont la cle exacte (match, victime, instant) est
// demandee sont rendues. Table absente -> ErrCapabilityNotSupported.
func (r *TacticalRepo) MatesDesMorts(ctx context.Context, q domain.TacticalQuery,
	morts []domain.CleDeMort) ([]domain.MateDeMort, error) {
	if err := exigerLaListe(q, "MatesDesMorts"); err != nil {
		return nil, err
	}
	ids := q.Matchs.IDs()
	victimes := victimesDistinctes(morts)
	if len(ids) == 0 || len(victimes) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()
	db, release, err := r.ouvrir(ctx, q, "MatesDesMorts")
	if err != nil {
		return nil, err
	}
	defer release()

	args := append(ToAnySlice(ids), ToAnySlice(victimes)...)
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(QTacticalMates, Placeholders(len(ids)), Placeholders(len(victimes))), args...)
	if err != nil {
		return nil, r.degrader(ctx, "MatesDesMorts", err)
	}
	demandees := make(map[domain.CleDeMort]bool, len(morts))
	for _, m := range morts {
		demandees[m] = true
	}
	var out []domain.MateDeMort
	err = scanRows(ctx, rows, "TacticalRepo.MatesDesMorts", func(sc rowScanner) error {
		var m domain.MateDeMort
		var x, y, d sql.NullFloat64
		if err := sc.Scan(&m.MatchID, &m.VictimXUID, &m.TimeMs, &m.MateXUID, &m.Gamertag,
			&m.Etat, &x, &y, &d); err != nil {
			return err
		}
		if !demandees[domain.CleDeMort{MatchID: m.MatchID, VictimXUID: m.VictimXUID, TimeMs: m.TimeMs}] {
			return nil
		}
		m.X, m.Y, m.DistanceM = nullFloatPtr(x), nullFloatPtr(y), nullFloatPtr(d)
		out = append(out, m)
		return nil
	})
	return out, err
}

// victimesDistinctes rend les xuids de victime distincts des morts demandees, dans l'ordre de
// premiere apparition.
func victimesDistinctes(morts []domain.CleDeMort) []string {
	vus := make(map[string]bool, len(morts))
	out := make([]string, 0, len(morts))
	for _, m := range morts {
		if m.VictimXUID == "" || vus[m.VictimXUID] {
			continue
		}
		vus[m.VictimXUID] = true
		out = append(out, m.VictimXUID)
	}
	return out
}
