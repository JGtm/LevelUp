// Package duckdb — career_repo_friends.go : les xuids des amis du joueur, en UNE lecture sans
// v_gamertag_lookup (lot perf L9-go, 2026-09-23, revue adversariale D, P1).
//
// Les rencontres de la Carrière (« joueurs les plus croisés, hors amis ») excluent les amis
// configurés du joueur, qui sont des GAMERTAGS. Jusqu'ici chacun se résolvait par
// ExplorerRepo.ResolveXUIDByGamertag, qui matérialise la vue des noms EN ENTIER : 1,8 à 2,8 s
// PAR AMI (mesure de la revue D), avant même la lecture des rencontres. Le service résout
// d'abord les amis SUIVIS par le registre des profils (db_profiles.json : leur xuid est connu,
// aucune lecture) ; ResolveFriendXUIDs lit les autres, tous à la fois.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/observability/timing"
)

// qAmisToken : la place des VALUES des gamertags cherchés dans QAmisParGamertagTpl.
const qAmisToken = "/*__AMIS__*/"

// QAmisParGamertagTpl : les xuids de gamertags donnés, sans la vue des noms. Mêmes niveaux que
// la vue, dans son ordre, sur ce qui peut nommer un ami du joueur :
//
//  1. xuid_aliases — le nom d'affichage de la vue quand il existe ;
//  2. les participants de l'historique du joueur (QMatchsDuJoueurTpl) — un ami est quelqu'un
//     avec qui il a joué ; ce niveau reconnaît aussi un ancien nom porté dans ces matchs.
//
// Comparaison insensible à la casse (ILIKE), bots écartés, comme la lecture d'avant. Un nom
// porté par plusieurs xuids : le niveau le plus fort, puis le plus petit xuid — la vue en
// prenait un au hasard (LIMIT 1 sans ORDER BY). Paramètres : les gamertags (VALUES), puis
// ?1 = xuid du joueur.
var QAmisParGamertagTpl = `
WITH amis(nom) AS (VALUES ` + qAmisToken + `),
par_alias AS (
    SELECT a.nom, xa.xuid, 1 AS niveau
    FROM amis a
    JOIN xuid_aliases xa ON xa.gamertag ILIKE a.nom
),
historique AS (` + QMatchsDuJoueurTpl + `),
par_participant AS (
    SELECT DISTINCT a.nom, mp.xuid, 2 AS niveau
    FROM amis a
    JOIN match_participants mp ON mp.gamertag ILIKE a.nom
    WHERE mp.match_id IN (SELECT match_id FROM historique)
)
SELECT nom, xuid
FROM (SELECT * FROM par_alias UNION ALL SELECT * FROM par_participant) c
WHERE ` + analysis.SQLIsNotBotCol("xuid") + `
QUALIFY ROW_NUMBER() OVER (PARTITION BY nom ORDER BY niveau, xuid) = 1`

// careerFriendsTimeout : plafond de la lecture des amis (une lecture, quelques ms mesurées).
const careerFriendsTimeout = 10 * time.Second

// ResolveFriendXUIDs rend, pour chaque gamertag résolu, son xuid (clé : le gamertag tel que
// demandé ; absent = aucun niveau ne le connaît). Une seule lecture pour tous
// (QAmisParGamertagTpl). Section de durée `career_friends`.
func (r *CareerRepo) ResolveFriendXUIDs(ctx context.Context, gamertags []string) (map[string]string, error) {
	if len(gamertags) == 0 {
		return map[string]string{}, nil
	}
	defer timing.FromContext(ctx).Section("career_friends")()
	ctx, cancel := context.WithTimeout(ctx, careerFriendsTimeout)
	defer cancel()

	valeurs := strings.TrimSuffix(strings.Repeat("(?), ", len(gamertags)), ", ")
	q := resolveCampaignExclusionByMatchID(strings.Replace(QAmisParGamertagTpl, qAmisToken, valeurs, 1), r.pdb.TitleSlug, "match_id")
	args := make([]any, 0, len(gamertags)+1)
	for _, gt := range gamertags {
		args = append(args, gt)
	}
	args = append(args, r.pdb.XUID)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.ResolveFriendXUIDs: shared reader: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.ResolveFriendXUIDs: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string, len(gamertags))
	for rows.Next() {
		var nom, xuid string
		if err := rows.Scan(&nom, &xuid); err != nil {
			return nil, fmt.Errorf("CareerRepo.ResolveFriendXUIDs scan: %w", err)
		}
		out[nom] = xuid
	}
	return out, rows.Err()
}
