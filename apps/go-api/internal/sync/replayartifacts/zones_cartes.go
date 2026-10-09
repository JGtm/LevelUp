package replayartifacts

// zones_cartes.go — LES CARTES JOUÉES DU REGISTRE, pour le premier passage du rattrapage des
// zones nommées (`levelup backfill-map-callouts`).
//
// Le fil de l'eau (zones_rattrapage.go) ne voit que les cartes des films qu'il cuit ; les
// cartes de l'historique passent par cette liste. Les noms candidats d'une carte sont ceux
// du cycle (`nomsDeCarte` : nom anglais du catalogue d'assets, puis libellé brut du registre) :
// la CLI et le fil de l'eau interrogent la cascade avec les mêmes identités.
//
// LE PÉRIMÈTRE EST CELUI DES MATCHS JcJ. Les matchs Firefight sont écartés : la demande porte
// sur les cartes que le rejeu et l'onglet Tactique montrent. Une carte Firefight dont un film
// serait cuit est rattrapée au fil de l'eau comme les autres.

import (
	"context"
	"database/sql"
	"fmt"
)

// CarteJouee : une carte du registre, ses noms candidats et le nombre de ses matchs.
type CarteJouee struct {
	MapID  string
	Noms   []string
	Matchs int
}

// requeteCartesJouees : les cartes JcJ du registre, les plus jouées d'abord.
const requeteCartesJouees = `
	SELECT map_id, max(map_name), count(*) AS matchs
	FROM match_registry
	WHERE map_id IS NOT NULL AND map_id <> '' AND NOT COALESCE(is_firefight, FALSE)
	GROUP BY map_id
	ORDER BY matchs DESC, map_id`

// CartesJoueesJcJ lit les cartes JcJ du registre partagé. `metaDB` peut être nil : les noms se
// réduisent alors au libellé brut du registre.
func CartesJoueesJcJ(ctx context.Context, sharedDB, metaDB *sql.DB) ([]CarteJouee, error) {
	rows, err := sharedDB.QueryContext(ctx, requeteCartesJouees)
	if err != nil {
		return nil, fmt.Errorf("cartes jouées du registre : %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []CarteJouee
	for rows.Next() {
		var mapID string
		var nom sql.NullString
		var n int
		if err := rows.Scan(&mapID, &nom, &n); err != nil {
			return nil, fmt.Errorf("cartes jouées du registre (scan) : %w", err)
		}
		out = append(out, CarteJouee{
			MapID:  mapID,
			Noms:   nomsDeCarte(ctx, metaDB, candidatARattraper{mapID: mapID, rawName: nom.String}),
			Matchs: n,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cartes jouées du registre (rows) : %w", err)
	}
	return out, nil
}
