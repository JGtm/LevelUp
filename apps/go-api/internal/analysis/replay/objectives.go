package replay

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// objectives.go — LE CALQUE DES ACTIONS D'OBJECTIF : ce que chaque joueur a FAIT, nomme et
// pose sur l'axe de temps du rejeu.
//
// # Ce qu'il apporte, et que les autres calques n'ont pas
//
// Les tirs, lancers et positions disent OU les joueurs etaient et vers ou ils visaient.
// Celui-ci dit ce qu'ils ont ACCOMPLI : une capture de drapeau, un retour, une prise de
// zone, un porteur stoppe — nomme, date a la milliseconde, attribue a un xuid.
//
// # L'horloge ne demande aucun recalage
//
// Le TimeMS des evenements est sur l'horloge du manifeste, la meme que celle des positions
// (etat de l'art §15 : les deux concordent a moins de 4 ms sur 573 s de film). La seule
// conversion est donc une DIVISION par le pas de la grille — pas d'appariement, pas de
// fenetre de tolerance, pas de seuil.
//
// # Le pont d'identite est deja fait, PAR MANCHE, et il n'est pas le numero de slot
//
// Les evenements arrivent deja identifies par xuid (`objectiveevents.IdentifiedEvent`), resolus
// PAR MANCHE par les seuls INSTANTS DE MORT ([objectiveevents.ResolveRoundIdentity]) chez
// l'appelant hors ligne — comme la couronne VIP, le drapeau vivant et le porteur du crane. Ce
// n'est pas un detail de commodite : le slot d'entite statborg et le slot de biped du rejeu sont
// DEUX ESPACES DIFFERENTS, et le slot d'entite est REATTRIBUE d'une manche a l'autre — les
// confondre, ou resoudre par les TOTAUX du match, poserait les actions sur les mauvais joueurs
// apres la bascule de manche sans que rien ne le signale (etat de l'art §20.1).

// ObjectiveAction est une action d'objectif posee sur l'axe de temps du rejeu.
type ObjectiveAction struct {
	// T est l'index de frame, sur le meme axe que Point.T et Shot.T.
	T int `json:"t"`
	// XUID du joueur auteur, en decimal — la cle sur laquelle le client joint la Track.
	//
	// POURQUOI LE XUID ET PAS LE SLOT : le meme raisonnement que Track.XUID. Un slot est un
	// ORDRE, et il n'a pas le meme sens dans le film et dans le statborg.
	XUID string `json:"xuid"`
	// Stat est le nom canonique de la statistique (`flag_captures`, `zone_secures`,
	// `flag_grabs`...), tel que `match_objective_stats` le nomme.
	//
	// CE SONT DES NOMS DE STATISTIQUE, PAS DE RECOMPENSE, et la distinction a ete payee :
	// `flag_grabs` compte les ramassages, la recompense `flag_taken` n'en recompense
	// qu'une partie (etat de l'art §22.2).
	Stat string `json:"stat"`
	// TimeMS est l'instant exact sur l'horloge du film. Conserve a cote de T parce que la
	// grille de frames PERD de la precision (100 ms par defaut) : le client qui veut
	// ordonner deux actions de la meme frame en a besoin.
	TimeMS int `json:"timeMs"`
}

// buildObjectiveActions pose les evenements identifies sur la grille de frames du rejeu, et
// rend la couverture du calque.
//
// L'HORLOGE DEMANDE UNE SOUSTRACTION, ET C'EST TOUT — mais elle n'etait pas faite (report
// `:123` du registre, corrige le 2026-08-18). Le TimeMS d'un evenement est date depuis le
// PREMIER PAQUET DU FILM ; la grille de frames, elle, compte depuis le premier paquet de
// POSITION. L'ecart entre les deux zeros est exactement `originMs`, et il vaut de 3,6 s a
// 50,8 s selon le match. Sans le retrancher, les pulses d'action s'allumaient trop tard et se
// posaient sur l'element le plus proche du joueur au MAUVAIS instant : l'appartenance stricte
// mesuree passe de 9,9 % a 40,9 % quand la correction est appliquee.
//
// La fenetre borne l'axe des DEUX cotes : un evenement anterieur a la frame 0 (mise en place
// du match) ou posterieur a la derniere frame publiee est compte hors fenetre plutot que
// rattache a une frame qui n'existe pas. La correction RAMENE dans la fenetre les actions de
// fin de match que l'ancien calcul rejetait (+11 sur 525 au corpus d'origine).
// LE DENOMINATEUR COMPTE AUSSI CE QUI N'EST JAMAIS ARRIVE ICI (`unnamed`). Les evenements que
// le pont d'identite de l'appelant n'a pas su attribuer ne figurent pas dans `evs` : les ignorer
// ferait de `Available` un compte de RESCAPES, et le rapport rattache/disponible se lirait
// ~100 % sur un calque partiel. Ils entrent donc au denominateur ET sous `NoSlot`, la categorie
// que le contrat public reserve exactement a ce cas (« le pont ne couvre pas ce joueur »).
// Mesure du depot sur `c0a82e88` : 17 actions nommees, 12 identifiees — la couverture annoncait
// 12/12 et 0 `noSlot`.
func buildObjectiveActions(evs []objectiveevents.IdentifiedEvent, unnamed int,
	c scoreClock) ([]ObjectiveAction, LayerCoverage) {
	cov := LayerCoverage{Available: len(evs) + unnamed, NoSlot: unnamed}
	if c.intervalMS <= 0 || c.frames <= 0 {
		cov.OutOfWindow = len(evs)
		return nil, cov
	}
	out := make([]ObjectiveAction, 0, len(evs))
	for _, e := range evs {
		if e.XUID == "" {
			// Un evenement sans identite n'est pas posable : le rattacher a un slot
			// arbitraire serait exactement l'erreur que le pont existe pour eviter. La
			// population de production de CETTE branche est nulle (les deux ponts
			// d'`objectiveevents` ecartent deja le xuid vide) : elle garde l'invariant du
			// champ publie `ObjectiveAction.XUID`, jamais vide, contre une entree malformee.
			// Le vrai peuplement de `NoSlot` vient d'`unnamed`, ci-dessus.
			cov.NoSlot++
			continue
		}
		t, ok := c.frameOf(e.TimeMS)
		if !ok {
			cov.OutOfWindow++
			continue
		}
		out = append(out, ObjectiveAction{T: t, XUID: e.XUID, Stat: e.Stat, TimeMS: e.TimeMS})
		cov.Attached++
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		if out[i].XUID != out[j].XUID {
			return out[i].XUID < out[j].XUID
		}
		return out[i].Stat < out[j].Stat
	})
	return out, cov
}

// countActionsWithoutTrack compte — SANS LES JETER — les actions dont l'auteur n'a aucune
// trajectoire publiee.
//
// # UNE ACTION EST UNE ACTION (doctrine du plan v2, lot R1, 2026-09-08)
//
// Cette fonction s'appelait `dropUnpublishedActions` et elle SUPPRIMAIT ces actions. Le
// raisonnement etait « le client ne pourrait pas la dessiner, il n'a pas de trajectoire ou
// l'accrocher » — mais une action d'objectif est un FAIT LU DU FILM, date a la milliseconde et
// nomme par le pont d'identite. Le jeter parce que le calque des positions est incomplet fait
// disparaitre une lecture vraie pour une raison de RENDU, et le compte publie n'a plus de
// rapport avec ce que le film porte. Mesure : `3372e7eb`, 35 actions sur 76 supprimees,
// toutes celles de deux joueurs sans aucune piste dans le film.
//
// LE CLIENT SAIT DEJA S'EN ABSTENIR : `buildObjectivePulses` ecarte une action dont il ne peut
// pas relire la position (« Une action sans position relue est ECARTEE — un pulse pose au hasard
// designerait la mauvaise zone »). L'abstention est donc au bon endroit — a l'affichage, pas
// dans la donnee servie.
//
// LE COMPTE RESTE, dans le journal : une action sans piste signale un DEFAUT du calque des
// positions, et le taire serait l'anti-patron que coverage.go interdit. Il n'entre plus dans
// `Unpublished`, qui est une categorie de REJET : plus rien n'est rejete ici.
func countActionsWithoutTrack(actions []ObjectiveAction, tracks []Track,
	slotXUID map[uint32]uint64) int {
	published := publishedXUIDs(tracks, slotXUID)
	n := 0
	for _, a := range actions {
		if !published[a.XUID] {
			n++
		}
	}
	return n
}

// attachObjectiveActions pose le calque des actions d'objectif sur le document et rend sa
// couverture.
//
// LES ACTIONS ARRIVENT DEJA IDENTIFIEES PAR XUID, ET PAR MANCHE : leur pont passe desormais par les
// seuls INSTANTS DE MORT ([objectiveevents.ResolveRoundIdentity]), resolu chez l'appelant hors
// ligne (cf. Options.Objectives) — aucune base, et JUSTE en multi-manche (le slot d'entite est
// reattribue d'une manche a l'autre). L'horloge, elle, demande une soustraction — celle de
// l'origine (cf. buildObjectiveActions et build_score.go).
func attachObjectiveActions(doc *ReplayDocument, opt Options, reg IdentityRegistry,
	c scoreClock) LayerCoverage {
	actions, cov := buildObjectiveActions(opt.Objectives, opt.ObjectivesUnnamed, c)
	doc.Objectives = actions
	if n := countActionsWithoutTrack(actions, doc.Tracks, reg.PontEpure()); n > 0 {
		// PUBLIEES QUAND MEME, ET SIGNALEES : le defaut est dans le calque des POSITIONS, pas
		// dans l'action. Le taire ferait disparaitre une lecture vraie sans laisser de trace.
		slog.Warn("rejeu : actions d'objectif dont l'auteur n'a aucune trajectoire publiee",
			"match_id", doc.MatchID, "actions", n, "publiees", len(actions))
	}
	cov.warnIfLossy("objectifs")
	return cov
}
