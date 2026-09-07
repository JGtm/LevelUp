package duckdb

// tactical_repo_isolement.go — LES MORTS DE L'UNIVERS, AVEC CE QUI LES ENTOURAIT.
//
// # TROIS TABLES, UNE SEULE REQUETE
//
//	match_kill_events_latest      QUI est mort, QUAND. Le journal fait foi sur les morts.
//	match_death_context_latest    QUI etait la, dans quel etat, a quelle distance. Ecrit AU
//	                              SYNC par le collecteur de kills (lot 7C).
//	kill_positions_latest         OU la mort a eu lieu — pour la peindre sur la carte.
//
// Les trois se joignent sans rapprocher aucune horloge : le contexte porte exactement la cle du
// journal (`match_id`, `victim_xuid`, `time_ms`), et les positions portent la meme `time_ms`.
// Un rapprochement TOLERANT aurait transforme un defaut de calage en resultat plausible.
//
// # LA JOINTURE DES POSITIONS PASSE PAR LE TUEUR, ET CE N'EST PAS UN DETOUR
//
// `kill_positions` a pour cle (match_id, killer_xuid, time_ms) : c'est une ligne PAR KILL, qui
// porte les deux positions. La position de la VICTIME s'y lit donc en joignant sur le TUEUR
// credite de sa mort. Une mort que personne ne revendique (chute, hors-limites) n'a pas de ligne
// et sort de la lecture : elle a bien eu lieu, mais on ne sait pas ou la peindre.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// QTacticalIsolement : les morts de l'univers, avec leur lieu et leur voisinage.
//
// `INNER JOIN` SUR LE CONTEXTE ET SUR LES POSITIONS, delibere. Une mort sans contexte est une
// mort d'un match dont le film n'a pas ete decode ; une mort sans position est une mort dont on
// ignore le lieu. Dans les deux cas, la lecture n'a rien a en dire — et un `LEFT JOIN` les
// ferait entrer avec des colonnes nulles que le service devrait interpreter, c'est-a-dire
// deviner.
//
// LE FILTRE `publishable` EST CELUI DU JOURNAL : une passe non publiable est juste en AGREGAT et
// fausse ligne par ligne (marge de bijection nulle, cas BTB). Une lecture qui nomme une mort
// exige donc `publishable = TRUE`, comme le kill feed et la timeline.
const QTacticalIsolement = `
SELECT e.match_id, e.victim_xuid,
       p.victim_x, p.victim_y,
       c.nearest_teammate_m, c.teammates_visible, c.teammates_out_of_sight
FROM match_kill_events_latest e
JOIN match_death_context_latest c
  ON c.match_id = e.match_id AND c.victim_xuid = e.victim_xuid AND c.time_ms = e.time_ms
JOIN kill_positions_latest p
  ON p.match_id = e.match_id AND p.killer_xuid = e.feed_killer_xuid AND p.time_ms = e.time_ms
WHERE e.match_id IN (SELECT u.match_id FROM (%s) u)
  AND e.publishable
  AND e.victim_xuid IS NOT NULL AND e.victim_xuid <> ''
  AND p.victim_x IS NOT NULL AND p.victim_y IS NOT NULL
ORDER BY e.match_id, e.time_ms, e.victim_xuid`

// MortsAvecContexte rend l'univers ET les morts localisees de ses matchs, avec le voisinage que
// le collecteur a mesure au sync.
//
// TOUS LES JOUEURS SONT RENDUS : l'axe « qui » (moi / escouade / adversaires) se tranche dans le
// service, a partir des equipes de l'univers — une requete par axe multiplierait les scans de la
// meme fenetre.
func (r *TacticalRepo) MortsAvecContexte(ctx context.Context, q domain.TacticalQuery) (domain.TacticalMortsContexte, error) {
	var out domain.TacticalMortsContexte
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()
	db, release, err := r.ouvrir(ctx, q, "MortsAvecContexte")
	if err != nil {
		return out, err
	}
	defer release()

	univ, err := r.chargerUnivers(ctx, db, q)
	if err != nil {
		return out, r.degrader(ctx, "MortsAvecContexte", err)
	}
	out.Univers = univ
	if len(univ.Matchs) == 0 {
		return out, nil
	}

	selectSQL, args := r.universSQL(q)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(QTacticalIsolement, selectSQL), args...)
	if err != nil {
		return out, r.degrader(ctx, "MortsAvecContexte", err)
	}
	err = scanRows(ctx, rows, "TacticalRepo.MortsAvecContexte", func(sc rowScanner) error {
		m, err := scanMortContexte(sc)
		if err != nil {
			return err
		}
		out.Morts = append(out.Morts, m)
		return nil
	})
	return out, err
}

// scanMortContexte lit une ligne. `nearest_teammate_m` est NULLABLE, et sa nullite PORTE UN SENS
// (aucun coequipier visible) : la traduire en zero ferait lire « a portee » une absence de
// mesure, c'est-a-dire l'exact inverse.
func scanMortContexte(sc rowScanner) (domain.MortContexte, error) {
	var m domain.MortContexte
	var proche sql.NullFloat64
	if err := sc.Scan(&m.MatchID, &m.VictimXUID, &m.X, &m.Y,
		&proche, &m.Visibles, &m.HorsDeVue); err != nil {
		return m, err
	}
	if proche.Valid {
		v := proche.Float64
		m.PlusProcheM = &v
	}
	return m, nil
}
