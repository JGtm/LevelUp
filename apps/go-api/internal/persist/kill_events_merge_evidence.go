package persist

// kill_events_merge_evidence.go — CE QUE L API PROUVE D UN INSTANT, ET CE QU ELLE COMPTE OFFICIELLEMENT.
//
// La fusion credit/film ne voyait que deux listes de morts. Or une ligne de film qui nomme une
// AUTRE victime que la mort de credit du meme instant n est pas une mort de plus tant que rien
// d independant du film ne la prouve : mesure du 2026-10-15, 11 morts fantomes sur 12 groupes
// (une victime servie 10 fois pour 7 morts officielles) ajoutees par ce seul raisonnement.
//
// Deux preuves independantes du film accompagnent donc la base credit :
//
//	APIKills        les kills du kill-feed de l API, par instant. Deux morts distinctes a la meme
//	                milliseconde laissent deux kills (un par mort) ; un seul kill ne porte qu une mort.
//	OfficialDeaths  les morts officielles de chaque joueur (`match_participants.deaths`), le
//	                plafond qu aucun nombre de morts servies ne depasse.
//
// Sans preuve (`Known` faux), la fusion ne declare AUCUNE autre mort : refuser coute une ligne de
// film, en ajouter une a tort invente une mort.

import (
	"context"
	"fmt"
	"sort"
)

// MergeEvidence : les preuves qu une base credit porte au moment de la fusion. La valeur zero
// signifie « inconnu » et rend la fusion prudente (cf. ci-dessus).
type MergeEvidence struct {
	// Known : les preuves ont ete lues pour ce match. Faux = aucune autre mort declarable, aucun
	// plafond applique.
	Known bool
	// APIKills : time_ms -> nombre de tueurs distincts ayant un kill a cet instant dans
	// `highlight_events`. DISTINCT sur le xuid : la table porte des groupes en double exact.
	APIKills map[int]int
	// OfficialDeaths : xuid -> morts officielles du joueur dans le match.
	OfficialDeaths map[string]int
}

// LoadMergeEvidence lit les preuves d UN match.
func LoadMergeEvidence(ctx context.Context, q rowQuerier, matchID string) (MergeEvidence, error) {
	ev := MergeEvidence{Known: true, APIKills: map[int]int{}, OfficialDeaths: map[string]int{}}

	rows, err := q.QueryContext(ctx, `
		SELECT time_ms, COUNT(DISTINCT xuid) FROM highlight_events
		WHERE match_id = ? AND event_type = 'kill' AND time_ms IS NOT NULL
		GROUP BY time_ms`, matchID)
	if err != nil {
		return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (kills): %w", matchID, err)
	}
	for rows.Next() {
		var t, n int
		if err := rows.Scan(&t, &n); err != nil {
			_ = rows.Close()
			return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (kills, scan): %w", matchID, err)
		}
		ev.APIKills[t] = n
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (kills, rows): %w", matchID, err)
	}
	if err := rows.Close(); err != nil {
		return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (kills, close): %w", matchID, err)
	}

	rows, err = q.QueryContext(ctx, `
		SELECT xuid, deaths FROM match_participants
		WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> '' AND deaths IS NOT NULL`, matchID)
	if err != nil {
		return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (morts officielles): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var x string
		var d int
		if err := rows.Scan(&x, &d); err != nil {
			return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (morts officielles, scan): %w", matchID, err)
		}
		ev.OfficialDeaths[x] = d
	}
	if err := rows.Err(); err != nil {
		return MergeEvidence{}, fmt.Errorf("persist: preuves de fusion %s (morts officielles, rows): %w", matchID, err)
	}
	return ev, nil
}

// autreMortDeclarable : l API compte-t-elle autant de kills que de morts a cet instant, la mort
// declaree comprise ? `deja` = morts de credit de l instant + autres morts deja declarees.
func (e MergeEvidence) autreMortDeclarable(instant, deja int) bool {
	return e.Known && e.APIKills[instant] >= deja+1
}

// orphanRef : une ligne de film conservee en orpheline, pour le plafond des morts officielles.
type orphanRef struct {
	outIdx  int
	victim  string
	timeMS  int
	partage bool
}

// plafonnerAuxMortsOfficielles : AUCUN joueur n est servi plus de fois qu il n est mort
// officiellement. Le surplus ne peut venir que des orphelines de film (la base credit ne se
// retranche jamais : cf. `CreditBaseCount`) ; on retire d abord celles qui PARTAGENT leur instant
// avec une mort de credit, puis les plus tardives. Rend la sortie filtree, le nombre de lignes
// retirees et le nombre de joueurs encore au-dessus du plafond (surplus porte par le credit seul).
func plafonnerAuxMortsOfficielles(deaths []KillEventInsert, orphans []orphanRef, official map[string]int,
) (out []KillEventInsert, retirees, restants int) {
	servies := map[string]int{}
	for i := range deaths {
		if v := deaths[i].VictimXUID; v != "" {
			servies[v]++
		}
	}
	retire := map[int]bool{}
	for victim, n := range servies {
		plafond, connu := official[victim]
		if !connu || n <= plafond {
			continue
		}
		surplus := n - plafond
		for _, c := range orphelinesDe(orphans, victim) {
			if surplus == 0 {
				break
			}
			retire[c.outIdx] = true
			surplus--
		}
		if surplus > 0 {
			restants++
		}
	}
	if len(retire) == 0 {
		return deaths, 0, restants
	}
	out = make([]KillEventInsert, 0, len(deaths)-len(retire))
	for i := range deaths {
		if !retire[i] {
			out = append(out, deaths[i])
		}
	}
	return out, len(retire), restants
}

// orphelinesDe : les orphelines d une victime, dans l ordre de retrait (partagees d abord, puis
// les plus tardives, puis l indice pour un ordre total).
func orphelinesDe(orphans []orphanRef, victim string) []orphanRef {
	var c []orphanRef
	for _, o := range orphans {
		if o.victim == victim {
			c = append(c, o)
		}
	}
	sort.Slice(c, func(i, j int) bool {
		if c[i].partage != c[j].partage {
			return c[i].partage
		}
		if c[i].timeMS != c[j].timeMS {
			return c[i].timeMS > c[j].timeMS
		}
		return c[i].outIdx > c[j].outIdx
	})
	return c
}
