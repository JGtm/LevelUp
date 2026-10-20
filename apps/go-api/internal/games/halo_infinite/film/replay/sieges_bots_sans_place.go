package replay

// sieges_bots_sans_place.go — LE BOT DECLARE QUI N'A AUCUNE VIE ET NE TROUVE AUCUNE PLACE N'ENTRE PAS
// AU ROSTER PUBLIE (regle des places : une equipe a exactement ses places, une tuile de plus est une
// erreur de lecture).
//
// # LE CAS
//
// BOT_METADATA declare un bot dans une equipe dont aucune place ne se libere : ni le film (aucune
// suppression d'entite, aucune image-cle apres l'arrivee) ni la base (aucun depart) ne montrent de
// partant. Le bot n'a pas de place a reprendre, et sans aucune vie il n'a rien a montrer : sa fiche
// serait une tuile de trop. Il sort du roster publie, et cela se COMPTE (expvar) et se DIT
// (AVERTISSEMENT). Aucun partant n'est deduit.
//
// # CE QUE LA REGLE NE MASQUE PAS
//
// Un bot sans place qui A une vie reste au roster, sans place (`index`) : c'est un defaut de lecture
// des places, compte dans `sansPlace` et journalise en ERREUR. Un arrivant sans equipe lue reste lui
// aussi sans place : le defaut d'equipe se lit dans `sansEquipe`.

import (
	"context"
	"log/slog"
	"slices"

	"levelup/go-api/internal/observability"
)

// metriqueBotsSansVieNiPlace : le compteur expvar des bots ecartes du roster publie par cette regle.
const metriqueBotsSansVieNiPlace = "rejeu_bots_sans_vie_ni_place_ecartes"

// botsSansPlace : ce que la pose des places a decide des bots d'equipe lue qui ne trouvent aucune place.
type botsSansPlace struct {
	// ecartes : les indices, dans le roster, des bots sans vie ecartes du roster publie.
	ecartes []int
	// avecVie : les bots sans place qui ont au moins une vie — un defaut, journalise en ERREUR.
	avecVie int
}

// ecarterLeBotSansVie statue sur l'arrivant `i` d'equipe lue qui ne trouve aucune place : un bot sans
// AUCUNE vie est ecarte (sa presence est videe, il n'est ni affiche, ni compte sans place) ; vrai alors.
// Faux pour tout autre arrivant, qui reste sans place — et un bot qui a une vie est compte a part.
func (pp *poseDesPlaces) ecarterLeBotSansVie(i int) bool {
	if !pp.roster[i].Bot {
		return false
	}
	if len(pp.occ.parEntree[i].vies) > 0 {
		pp.sansPlace.avecVie++
		return false
	}
	pp.occ.parEntree[i].presence = nil
	pp.sansPlace.ecartes = append(pp.sansPlace.ecartes, i)
	return true
}

// ecarte dit que l'entree `i` est un bot ecarte du roster publie.
func (pp *poseDesPlaces) ecarte(i int) bool { return slices.Contains(pp.sansPlace.ecartes, i) }

// sansLesBotsEcartes rend le roster publie : celui de la pose, prive des bots que la pose a ecartes
// ([SeatCoverage] porte leurs indices). L ordre du roster est conserve.
func sansLesBotsEcartes(roster []RosterEntry, cov SeatCoverage) []RosterEntry {
	if len(cov.botsEcartes) == 0 {
		return roster
	}
	out := make([]RosterEntry, 0, len(roster)-len(cov.botsEcartes))
	for i, e := range roster {
		if !slices.Contains(cov.botsEcartes, i) {
			out = append(out, e)
		}
	}
	return out
}

// journaliserLesBotsSansPlace dit, au compteur et en AVERTISSEMENT, les bots ecartes ; en ERREUR, les
// bots sans place qui ont une vie — jamais masques.
func journaliserLesBotsSansPlace(ctx context.Context, matchID string, ecartes, avecVie int) {
	if ecartes > 0 {
		observability.AddInt(metriqueBotsSansVieNiPlace, int64(ecartes))
		slog.WarnContext(ctx, "rejeu : bot declare sans aucune vie ni place libre dans son equipe — hors du "+
			"roster publie, aucun partant deduit", "match_id", matchID, "bots", ecartes)
	}
	if avecVie > 0 {
		slog.ErrorContext(ctx, "rejeu : bot sans place qui a une vie — sa place n'est pas lue", "match_id", matchID,
			"bots", avecVie)
	}
}
