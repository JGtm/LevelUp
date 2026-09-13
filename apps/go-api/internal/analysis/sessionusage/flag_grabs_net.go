package sessionusage

// flag_grabs_net.go — LES PRISES NETTES DE DRAPEAU dans le bloc objectifs de la session.
//
// # POURQUOI UNE LIGNE A ELLE, ET PAS UN AJOUT AU ROLE « PRENDRE »
//
// La grandeur entre bien dans le rôle « prendre » au niveau de la TABLE DES RÔLES
// (`narrative.ObjectiveRoleGrandeurs`) : c'est elle qui dit ce que « prendre » veut dire, et
// c'est sur cette table que les grilles d'objectif construisent leurs colonnes.
//
// Mais le bloc de session, lui, publie des SOMMES PAR RÔLE et leurs PARTS. Or cette grandeur
// n'est mesurée que sur les matchs dont le film a été lu — les autres n'ont AUCUNE valeur. La
// verser dans la somme « prendre » ferait deux dégâts silencieux : un match sans film y
// compterait comme un match sans prise (le faux zéro que tout ce dépôt refuse), et la part
// joueur/camp changerait de dénominateur d'un scope à l'autre selon la couverture du film.
//
// Elle se publie donc À PART, AVEC SES PROPRES DÉNOMINATEURS : combien de matchs à objectif, et
// combien d'entre eux sont mesurés. Une part qui dit sur quoi elle porte est lisible ; une somme
// qui mélange mesuré et non mesuré ne l'est pas.
//
// # LA FENÊTRE VOYAGE AVEC LA MESURE
//
// Chaque ligne porte la fenêtre de jonglage sous laquelle elle a été calculée. Le bloc publie
// celle du scope — et seulement si elle est UNIQUE : un scope qui mêle deux fenêtres (parc
// partiellement re-projeté après un changement de règle) n'a pas UNE fenêtre, et publier l'une
// des deux mentirait sur l'autre.

import "levelup/go-api/internal/domain"

// FlagGrabsNetRow — une ligne (match, joueur) de `match_flag_grabs_net_latest`.
type FlagGrabsNetRow struct {
	MatchID string
	XUID    string
	// Raw / Net : les prises brutes lues du film, et les mêmes jonglage replié.
	Raw, Net int
	// WindowMS : la fenêtre sous laquelle Net a été calculé.
	WindowMS int
}

// ComputeFlagGrabsNet agrège les prises nettes du scope. nil si aucune ligne — le sous-bloc est
// alors OMIS, jamais servi à zéro (un scope sans CTF, ou sans film lu, n'a pas de prises nettes
// à montrer).
func ComputeFlagGrabsNet(
	rows []FlagGrabsNetRow, playerXUID string, matchsObjectif int,
	playerTeam map[string]int, teamOf map[string]map[string]int,
) *domain.SessionFlagGrabsNetBlock {
	if len(rows) == 0 {
		return nil
	}
	out := &domain.SessionFlagGrabsNetBlock{MatchesWithObjectives: matchsObjectif}
	mesures := map[string]bool{}
	fenetres := map[int]bool{}
	for i := range rows {
		r := &rows[i]
		mesures[r.MatchID] = true
		fenetres[r.WindowMS] = true
		out.LobbyTotal += r.Net
		out.LobbyRawTotal += r.Raw
		camp, campConnu := playerTeam[r.MatchID]
		if campConnu && teamOf[r.MatchID][r.XUID] == camp {
			out.TeamTotal += r.Net
			out.TeamRawTotal += r.Raw
		}
		if r.XUID == playerXUID {
			out.PlayerTotal += r.Net
			out.PlayerRawTotal += r.Raw
		}
	}
	out.MatchesMeasured = len(mesures)
	if len(fenetres) == 1 {
		for ms := range fenetres {
			out.WindowSeconds = float64(ms) / 1000
		}
	}
	// PAS DE PART SUR UN DÉNOMINATEUR NUL, et pas de 0 % non plus : un camp qui n'a pris aucun
	// drapeau ne donne pas « 0 % de participation », il ne donne PAS de part.
	if out.TeamTotal > 0 {
		p := 100 * float64(out.PlayerTotal) / float64(out.TeamTotal)
		out.PlayerShareOfTeamPct = &p
	}
	return out
}
