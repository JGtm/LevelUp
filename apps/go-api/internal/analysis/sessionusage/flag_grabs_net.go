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
// Elle se publie donc À PART, AVEC SES PROPRES DÉNOMINATEURS.
//
// # LES TROIS DÉNOMINATEURS, ET POURQUOI ILS SONT TROIS
//
//	MatchesFlagFamily   les matchs de la famille DRAPEAU de la session. C'est LE dénominateur
//	                    de couverture, et il a été corrigé en revue : compter les matchs à
//	                    objectif TOUTES FAMILLES confondues attribuait au film l'absence des
//	                    matchs de zones ou de crâne — des matchs qui n'ont pas de drapeau, et
//	                    dont l'absence ne dit rien de la couverture du film.
//	MatchesMeasured     ceux d'entre eux dont le calque de drapeau a été lu.
//	MatchesTeamKnown    ceux d'entre eux où le camp du joueur suivi est connu — le SEUL
//	                    périmètre sur lequel une part d'équipe a un sens.
//
// # LA PART SE CALCULE SUR UN PÉRIMÈTRE UNIQUE (règle de computeMetric, usage.go)
//
// Numérateur ET dénominateur sur le sous-ensemble à camp connu. Compter le joueur sur tous
// les matchs et son camp sur les seuls matchs à camp connu ferait dépasser 100 % dès qu'un
// match du scope a un camp inconnu — c'est exactement le défaut relevé en revue.
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
	// Raw / Net : les prises brutes LUES DU FILM, et les mêmes jonglage replié. Le « brut »
	// est ici celui du film — le compteur de l'API (`match_objective_stats.flag_grabs`) est
	// une AUTRE chaîne, et il n'entre jamais dans cette table.
	Raw, Net int
	// Openings : les ouvertures de portage comptées par l'oracle du film sur CE match. Valeur
	// de match, identique sur toutes les lignes du match. C'est le dénominateur de Raw.
	Openings int
	// WindowMS : la fenêtre sous laquelle Net a été calculé.
	WindowMS int
}

// FlagGrabsNetInput — tout ce que l'agrégat demande.
type FlagGrabsNetInput struct {
	Rows       []FlagGrabsNetRow
	PlayerXUID string
	// MatchesFlagFamily : les matchs de la famille DRAPEAU de la session (dénominateur de
	// couverture). Zéro = la session n'a aucun match à drapeau.
	MatchesFlagFamily int
	// PlayerTeam : matchID -> camp du joueur suivi (clé absente = camp inconnu).
	PlayerTeam map[string]int
	// TeamOf : matchID -> (xuid -> camp).
	TeamOf map[string]map[string]int
}

// ComputeFlagGrabsNet agrège les prises nettes du scope. nil si aucune ligne — le sous-bloc est
// alors OMIS, jamais servi à zéro (un scope sans CTF, ou sans film lu, n'a pas de prises nettes
// à montrer).
func ComputeFlagGrabsNet(in FlagGrabsNetInput) *domain.SessionFlagGrabsNetBlock {
	if len(in.Rows) == 0 {
		return nil
	}
	out := &domain.SessionFlagGrabsNetBlock{MatchesWithFlagFamily: in.MatchesFlagFamily}
	mesures := map[string]bool{}
	campConnus := map[string]bool{}
	fenetres := map[int]bool{}
	ouvertures := map[string]int{}

	for i := range in.Rows {
		r := &in.Rows[i]
		mesures[r.MatchID] = true
		fenetres[r.WindowMS] = true
		ouvertures[r.MatchID] = r.Openings
		out.LobbyTotal += r.Net
		out.LobbyRawTotal += r.Raw
		if r.XUID == in.PlayerXUID {
			out.PlayerTotal += r.Net
			out.PlayerRawTotal += r.Raw
		}
		accumulerPerimetreEquipe(out, in, r, campConnus)
	}

	out.MatchesMeasured = len(mesures)
	out.MatchesTeamKnown = len(campConnus)
	for _, n := range ouvertures {
		out.OpeningsTotal += n
	}
	if len(fenetres) == 1 {
		for ms := range fenetres {
			out.WindowSeconds = float64(ms) / 1000
		}
	}
	// PAS DE PART SUR UN DÉNOMINATEUR NUL, et pas de 0 % non plus : un camp qui n'a pris aucun
	// drapeau ne donne pas « 0 % de participation », il ne donne PAS de part.
	if out.TeamTotal > 0 {
		p := 100 * float64(out.PlayerTeamScopeTotal) / float64(out.TeamTotal)
		out.PlayerShareOfTeamPct = &p
	}
	return out
}

// accumulerPerimetreEquipe ajoute UNE ligne aux totaux du périmètre à camp connu — les seuls
// qui se comparent entre eux.
//
// DEUX PRÉSENCES SE TESTENT, PAS UNE. Le camp du joueur suivi sur ce match, et le camp de
// CETTE ligne dans la table du match. Comparer `teamOf[m][x]` sans tester la présence de la
// clé ferait passer un xuid ABSENT (valeur zéro d'une map) pour un joueur de l'équipe 0 — et
// zéro est un camp parfaitement légitime.
func accumulerPerimetreEquipe(
	out *domain.SessionFlagGrabsNetBlock, in FlagGrabsNetInput, r *FlagGrabsNetRow,
	campConnus map[string]bool,
) {
	camp, campSu := in.PlayerTeam[r.MatchID]
	if !campSu {
		return
	}
	campConnus[r.MatchID] = true
	if r.XUID == in.PlayerXUID {
		out.PlayerTeamScopeTotal += r.Net
		out.PlayerTeamScopeRawTotal += r.Raw
	}
	campDeLaLigne, connu := in.TeamOf[r.MatchID][r.XUID]
	if !connu || campDeLaLigne != camp {
		return
	}
	out.TeamTotal += r.Net
	out.TeamRawTotal += r.Raw
}
