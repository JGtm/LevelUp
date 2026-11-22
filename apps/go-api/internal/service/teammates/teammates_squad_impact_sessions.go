package teammates

// teammates_squad_impact_sessions.go — l'entrée du calcul de l'impact par session
// (squadimpact.BuildSessions) tirée de la page : les joueurs présents dans l'équipe alliée de
// chaque match, et le compte des matchs que le tableau laisse de côté faute de libellé de session.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squadimpact"
	"levelup/go-api/internal/domain"
)

// sessionsDImpact rend le bloc de l'impact par session : les sessions publiées (in.evenings), les
// rôles tombés sur l'escouade et la présence de chaque joueur dans l'équipe alliée du joueur
// principal. Nil quand rien ne se publie (voir squadimpact.BuildSessions).
func sessionsDImpact(
	in impactEscouade, players []string, squad map[string]string, roles map[string][]squadimpact.Attribution,
) *domain.SquadImpactSessions {
	return squadimpact.BuildSessions(squadimpact.SessionsInput{
		Evenings: in.evenings, Players: players, Roles: roles, Allied: joueursDansLEquipeAlliee(in, squad),
	})
}

// joueursDansLEquipeAlliee rend, par match, les noms des membres de l'escouade présents dans
// l'équipe alliée du joueur principal (Q32b). Sans xuid du principal aucune équipe n'est
// attribuable : aucun joueur n'est présent par cette voie.
func joueursDansLEquipeAlliee(in impactEscouade, squad map[string]string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	if in.mainXUID == "" {
		return out
	}
	for _, a := range in.allies {
		name, ok := squad[a.XUID]
		if !ok {
			continue
		}
		if out[a.MatchID] == nil {
			out[a.MatchID] = map[string]bool{}
		}
		out[a.MatchID][name] = true
	}
	return out
}

// matchsSansSession compte les matchs de la population sans libellé de session : le tableau n'a
// pas de ligne pour eux.
func matchsSansSession(rows []domain.SquadMatchRow) int {
	seen := map[string]bool{}
	for _, r := range rows {
		if r.SessionLabel == nil || *r.SessionLabel == "" {
			seen[r.MatchID] = true
		}
	}
	return len(seen)
}

// journaliserMatchsSansSession journalise, une fois par requête, le nombre de matchs de la
// population que le tableau par session exclut faute de libellé de session.
func journaliserMatchsSansSession(ctx context.Context, rows []domain.SquadMatchRow) {
	if n := matchsSansSession(rows); n > 0 {
		slog.InfoContext(ctx, "teammates_squad_impact_sessions_unlabelled_matches", "matches", n)
	}
}
