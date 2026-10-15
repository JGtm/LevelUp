package persist

// kill_events_merge_evidence.go — CE QUE L API PROUVE D UN INSTANT, ET CE QU ELLE COMPTE OFFICIELLEMENT.
//
// La fusion credit/film ne voyait que deux listes de morts. Or une ligne de film qui nomme une
// AUTRE victime que la mort de credit du meme instant n est pas une mort de plus tant que rien
// d independant du film ne la prouve : mesure du 2026-10-15, 11 morts fantomes sur 12 groupes
// (une victime servie 10 fois pour 7 morts officielles) ajoutees par ce seul raisonnement.
//
// Trois preuves independantes du film accompagnent donc la base credit :
//
//	APIKills        les kills du kill-feed de l API, par instant. Deux morts distinctes a la meme
//	                milliseconde laissent deux kills (un par mort) ; un seul kill ne porte qu une mort.
//	OfficialDeaths  les morts officielles de chaque joueur (`match_participants.deaths`), le
//	                controle de qualite des morts servies (jamais une troncature).
//	APIDeaths       les evenements `death` de chaque joueur : toute mort d humain servie s adosse a
//	                l un d eux, un evenement par ligne (kill_events_merge_adossement.go).
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
	// controle applique.
	Known bool
	// APIKills : time_ms -> nombre de tueurs distincts ayant un kill a cet instant dans
	// `highlight_events`. DISTINCT sur le xuid : la table porte des groupes en double exact.
	APIKills map[int]int
	// OfficialDeaths : xuid -> morts officielles du joueur dans le match.
	OfficialDeaths map[string]int
	// APIDeaths : xuid -> instants DISTINCTS de ses evenements `death` dans `highlight_events` — ce a
	// quoi chaque mort d humain servie s adosse (kill_events_merge_adossement.go).
	APIDeaths map[string][]int
}

// LoadMergeEvidence lit les preuves d UN match.
func LoadMergeEvidence(ctx context.Context, q rowQuerier, matchID string) (MergeEvidence, error) {
	ev := MergeEvidence{Known: true, APIKills: map[int]int{}, OfficialDeaths: map[string]int{},
		APIDeaths: map[string][]int{}}

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
	if err := chargerMortsAPI(ctx, q, matchID, ev.APIDeaths); err != nil {
		return MergeEvidence{}, err
	}
	return ev, nil
}

// chargerMortsAPI lit dans `out` les instants distincts des evenements `death` de chaque xuid du
// match (la table porte des groupes en double exact : DISTINCT).
func chargerMortsAPI(ctx context.Context, q rowQuerier, matchID string, out map[string][]int) error {
	rows, err := q.QueryContext(ctx, `
		SELECT DISTINCT xuid, time_ms FROM highlight_events
		WHERE match_id = ? AND event_type = 'death' AND time_ms IS NOT NULL
		  AND xuid IS NOT NULL AND xuid <> ''
		ORDER BY 1, 2`, matchID)
	if err != nil {
		return fmt.Errorf("persist: preuves de fusion %s (morts de l API): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var x string
		var t int
		if err := rows.Scan(&x, &t); err != nil {
			return fmt.Errorf("persist: preuves de fusion %s (morts de l API, scan): %w", matchID, err)
		}
		out[x] = append(out[x], t)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("persist: preuves de fusion %s (morts de l API, rows): %w", matchID, err)
	}
	return nil
}

// autreMortDeclarable : l API compte-t-elle autant de kills que de morts a cet instant, la mort
// declaree comprise ? `deja` = morts de credit de l instant + autres morts deja declarees.
func (e MergeEvidence) autreMortDeclarable(instant, deja int) bool {
	return e.Known && e.APIKills[instant] >= deja+1
}

// OverOfficialPlayer : un joueur servi plus de fois qu il n est mort officiellement.
type OverOfficialPlayer struct {
	XUID     string
	Official int
	Served   int
}

// joueursAuDessusDesMortsOfficielles : CONTROLE DE QUALITE, JAMAIS UN COUPERET. Rien n est retire ici :
// retirer une ligne pour ramener un joueur a son compte, c est choisir sans preuve laquelle est
// fausse (on peut garder la fausse et perdre la juste). Les lignes sans preuve sont deja refusees
// en amont (passe 2, [MergeEvidence.autreMortDeclarable] ; adossement, [adossement.orphelineServie]) ;
// ce qui depasse encore se compte et se journalise. Ordre : par xuid, pour un rendu reproductible.
//
// LE PLAFOND EST LE PLUS GRAND DES DEUX COMPTES DE L API, les morts officielles et les evenements
// `death` : un joueur parti puis revenu en cours de match n a de morts officielles que sur une
// partie du match, alors que le kill-feed porte toutes ses morts (ecart legitime, pas une mort de
// trop).
func joueursAuDessusDesMortsOfficielles(deaths []KillEventInsert, ev MergeEvidence) []OverOfficialPlayer {
	servies := map[string]int{}
	for i := range deaths {
		if v := deaths[i].VictimXUID; v != "" {
			servies[v]++
		}
	}
	var out []OverOfficialPlayer
	for x, n := range servies {
		if o, connu := ev.OfficialDeaths[x]; connu && n > max(o, len(ev.APIDeaths[x])) {
			out = append(out, OverOfficialPlayer{XUID: x, Official: o, Served: n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].XUID < out[j].XUID })
	return out
}
