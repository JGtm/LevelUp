package replay

// flag_grabs_net_bridge.go — LE PONT entre le calque de drapeau PUBLIE et la regle des prises
// nettes (`objectives.NetFlagGrabs`).
//
// # POURQUOI ON LIT LE DOCUMENT, ET PAS LE FILM
//
// C'est l'inverse du choix fait pour l'Assaut (`sync/replayartifacts/bombstats.go` : « le
// document publie le portage en FRAMES, sans les periodes non pontees ni la distinction
// lacher/mort — reconstruire a partir de lui ferait un SECOND decodeur »). Ici, trois faits le
// renversent :
//
//	LE DOCUMENT PORTE DEJA LA MESURE EXACTE. `flagCarries[].spans` EST la chronologie de
//	  portage, bornee par les evenements nommes du statborg, avec le porteur resolu en xuid par
//	  le pont par instants de mort. Rien n'est reconstruit : la regle ne fait que LIRE.
//	LA PRECISION SUFFIT, ET ON PEUT LE DIRE. La grille du document est `frameIntervalMs`
//	  (100 ms sur tout le parc) ; la fenetre mesuree vaut 1,5 s, soit QUINZE pas. La coupure
//	  mesuree est large de 200 ms (1,4 s replie / 1,6 s compte) : deux pas de grille.
//	TOUT LE PARC DEVIENT LISIBLE SANS RECUISSON. Les artefacts publient `flagCarries` depuis le
//	  schema 14 ; calculer a la cuisson aurait exige de RE-CUIRE le parc pour une grandeur que
//	  les octets deja ranges contiennent — et la cuisson en lot est interdite (bombe RAM).
//
// # CE QUI N'EST PAS PUBLIE DANS LE DOCUMENT, ET CE QUE CA COUTE
//
// L'artefact ne porte pas les evenements nommes de la table DRAPEAU : le compte de l'oracle
// (`Openings`) arrive donc par `coverage.flagCarries.openings`, qui EST ce compte, deja
// fusionne. C'est pour cela que [FlagTracksOf] le rend a part.

import (
	"levelup/go-api/internal/games/halo_infinite/film/facts/objectives"
)

// FlagTracksOf projette le calque de drapeau d'un document vers l'entree de
// [objectives.NetFlagGrabs], en MILLISECONDES.
//
// Rend `ok` faux quand le document n'est PAS exploitable pour cette grandeur :
//
//	pas de couverture de drapeau      l'appelant n'a rien fourni a lire a la cuisson
//	verdict `flagFilm` faux           le film n'est pas du CTF (le cas nominal de tous les
//	                                  autres modes) — le calque de drapeau y est vide ou
//	                                  trompeur, cf. objectives/flagfilm.go
//	`frameIntervalMs` absent          l'axe de temps n'a pas d'echelle : aucun ecart ne se
//	                                  convertit en secondes, donc aucune fenetre ne s'applique
//
// Dans les trois cas la grandeur est NON MESUREE — jamais des zeros.
func FlagTracksOf(doc *ReplayDocument) (tracks []objectives.FlagTrack, openings int, ok bool) {
	if doc == nil || doc.Coverage == nil || doc.Coverage.FlagCarries == nil {
		return nil, 0, false
	}
	cov := doc.Coverage.FlagCarries
	if !cov.FlagFilm || doc.FrameIntervalMS <= 0 {
		return nil, 0, false
	}
	iv := doc.FrameIntervalMS
	out := make([]objectives.FlagTrack, 0, len(doc.FlagCarries))
	for _, fc := range doc.FlagCarries {
		tr := objectives.FlagTrack{Team: fc.Team, Spans: make([]objectives.FlagSpan, 0, len(fc.Spans))}
		for _, sp := range fc.Spans {
			x := ""
			if sp.XUID != nil {
				x = *sp.XUID
			}
			tr.Spans = append(tr.Spans, objectives.FlagSpan{
				State: sp.State, StartMS: sp.T0 * iv, EndMS: sp.T1 * iv, XUID: x,
			})
		}
		out = append(out, tr)
	}
	return out, cov.Openings, true
}
