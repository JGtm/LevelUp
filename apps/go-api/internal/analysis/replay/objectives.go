package replay

import (
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
// # Le pont d'identite est deja fait, et il n'est pas le numero de slot
//
// Les evenements arrivent deja identifies par xuid (`objectiveevents.IdentifiedEvent`). Ce
// n'est pas un detail de commodite : le slot d'entite statborg et le slot de biped du rejeu
// sont DEUX ESPACES DIFFERENTS, et les confondre poserait les actions sur les mauvais
// joueurs sans que rien ne le signale (etat de l'art §20.1).

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
func buildObjectiveActions(evs []objectiveevents.IdentifiedEvent, c scoreClock) ([]ObjectiveAction, LayerCoverage) {
	cov := LayerCoverage{Available: len(evs)}
	if c.intervalMS <= 0 || c.frames <= 0 {
		cov.OutOfWindow = len(evs)
		return nil, cov
	}
	out := make([]ObjectiveAction, 0, len(evs))
	for _, e := range evs {
		if e.XUID == "" {
			// Un evenement sans identite n'est pas posable : le rattacher a un slot
			// arbitraire serait exactement l'erreur que le pont existe pour eviter.
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

// dropUnpublishedActions retire les actions dont le joueur n'a AUCUNE track publiee, et les
// compte a part.
//
// Ce n'est pas un echec de rattachement — l'action est bien identifiee — mais le client ne
// pourrait pas la dessiner : il n'a pas de trajectoire ou l'accrocher. La compter sous
// `Unpublished` plutot que sous `Attached` evite d'annoncer une couverture que l'ecran ne
// tiendra pas.
func dropUnpublishedActions(actions []ObjectiveAction, tracks []Track, cov LayerCoverage) ([]ObjectiveAction, LayerCoverage) {
	published := map[string]bool{}
	for _, tr := range tracks {
		if tr.XUID != "" {
			published[tr.XUID] = true
		}
	}
	out := actions[:0:0]
	for _, a := range actions {
		if !published[a.XUID] {
			cov.Unpublished++
			cov.Attached--
			continue
		}
		out = append(out, a)
	}
	return out, cov
}

// attachObjectiveActions pose le calque des actions d'objectif sur le document et rend sa
// couverture.
//
// LES ACTIONS ARRIVENT DEJA IDENTIFIEES PAR XUID : leur pont passe par les lignes de match, donc
// par la base, que ce paquet n'ouvre pas (cf. Options.Objectives). L'horloge, elle, demande une
// soustraction — celle de l'origine (cf. buildObjectiveActions et build_score.go).
func attachObjectiveActions(doc *ReplayDocument, evs []objectiveevents.IdentifiedEvent, c scoreClock) LayerCoverage {
	actions, cov := buildObjectiveActions(evs, c)
	doc.Objectives, cov = dropUnpublishedActions(actions, doc.Tracks, cov)
	cov.warnIfLossy("objectifs")
	return cov
}
