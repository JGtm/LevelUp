package main

// empreinte_axes.go — LA PASSE SPECIALISEE : ce que la taille d'un calque ne dit pas.
//
// La passe generique compte les elements. Elle ne dirait pas qu'un calque a GARDE ses 17
// actions mais que cinq d'entre elles ont change de joueur, ni qu'un fil de score existe
// toujours mais rend zero frag pour tout le monde. Ce fichier descend donc dans les calques
// dont la valeur PRODUIT depend du detail — et il le fait, autant que possible, sans nommer de
// champ : la mesure par element est generique (`mesurerTableau`), seules les jointures qui
// croisent deux champs (joueur x famille d'action, serie de score) sont ecrites a la main.

import (
	"sort"
	"strconv"
)

// champsFamille sont les champs dont la VALEUR nomme une famille : les compter par valeur est
// ce qui distingue « 17 actions » de « 17 actions dont 3 captures ». Un champ d'IDENTITE (xuid)
// est traite a part ; tout autre champ textuel n'est compte que par presence — repartir un
// document sur des valeurs libres ferait exploser l'empreinte sans rien apprendre.
var champsFamille = map[string]bool{
	"stat": true, "state": true, "kind": true, "family": true, "end": true,
	"actorSource": true, "type": true, "mode": true, "reason": true, "verdict": true,
	"chassis": true, "slotKind": true, "source": true, "oracle": true, "cause": true,
}

// axesParCle donne l'axe de rapport de chaque calque connu. Un calque INCONNU prend son propre
// nom pour axe : c'est ce qui fait entrer un calque neuf dans le rapport sans qu'on l'y inscrive.
var axesParCle = map[string]string{
	"schemaVersion": "entete", "matchId": "entete", "titleSlug": "entete",
	"frameCount": "horloges", "frameIntervalMs": "horloges", "durationMs": "horloges",
	"originMs": "horloges", "t0FilmMs": "horloges",
	"bounds": "carte", "geometry": "carte", "geometryBounds": "carte",
	"structure": "carte", "structureBounds": "carte",
	"tracks": "pistes", "roster": "roster",
	"scoreTimeline": "score",
	"objectives":    "objectifs",
	"flagCarries":   "ports", "skullCarries": "ports", "bombCarries": "ports",
	"vipCrown": "ports", "flagReturnZone": "objets-objectif",
	"objectiveObjects": "objets-objectif", "mapObjectives": "objets-objectif",
	"zoneStates": "objets-objectif",
	"vehicles":   "vehicules", "vehicleLabels": "vehicules",
	"equipmentPlacements": "equipement", "equipmentChanges": "equipement",
	"equipmentEpisodes": "equipement", "abilityCharges": "equipement",
	"abilityImpulses": "equipement", "abilities": "equipement",
	"abilityLabels": "equipement", "grappleLines": "equipement",
	"translocations": "equipement", "inventory": "equipement",
	"shots": "armes", "loadouts": "armes", "weaponChanges": "armes", "pickups": "armes",
	"groundWeapons": "armes", "weaponPads": "armes", "padPickups": "armes",
	"weaponLabels": "armes", "killEffects": "armes", "mapWeaponPads": "armes",
	"grenades": "grenades", "grenadeReads": "grenades", "grenadeLabels": "grenades",
	"projectiles": "grenades",
	"bombStats":   "assaut", "bombArmings": "assaut", "bombEvents": "assaut",
	"coverage": "couverture", "neutralDeaths": "morts",
}

// axeDe rend l'axe de rapport d'un calque.
func axeDe(k string) string {
	if a, ok := axesParCle[k]; ok {
		return a
	}
	return "autres:" + k
}

// passeSpecialisee enrichit l'empreinte des mesures fines.
func passeSpecialisee(e *Empreinte, doc map[string]any) {
	for k, v := range doc {
		if items := tableau(v); items != nil {
			mesurerTableau(e, axeDe(k), k, items, 0)
		}
	}
	mesurerPistes(e, tableau(doc["tracks"]))
	mesurerRoster(e, tableau(doc["roster"]))
	mesurerObjectifs(e, tableau(doc["objectives"]))
	mesurerScore(e, objet(doc["scoreTimeline"]))
	aplatir(e, "couverture", "coverage", doc["coverage"], 0)
	aplatir(e, "assaut", "bombStats", doc["bombStats"], 0)
	aplatir(e, "carte", "bounds", doc["bounds"], 0)
}

// mesurerTableau mesure UN calque element par element, sans connaitre sa forme : la longueur de
// chaque sous-tableau, la presence de chaque champ, la repartition des champs de famille, et le
// compte par xuid. Elle descend d'un niveau dans les sous-tableaux (les intervalles d'un
// drapeau, les echantillons d'un vehicule) et s'arrete la : plus bas, la mesure ne porterait
// plus de sens produit.
func mesurerTableau(e *Empreinte, axe, prefixe string, items []any, profondeur int) {
	e.num(axe, prefixe+"/n", float64(len(items)))
	if profondeur > 1 {
		return
	}
	for _, it := range items {
		obj := objet(it)
		if obj == nil {
			continue
		}
		for champ, v := range obj {
			mesurerChamp(e, axe, prefixe, champ, v, profondeur)
		}
	}
}

// mesurerChamp pose les mesures d'UN champ d'UN element.
func mesurerChamp(e *Empreinte, axe, prefixe, champ string, v any, profondeur int) {
	if v == nil {
		return
	}
	if sous := tableau(v); sous != nil {
		e.incr(axe, prefixe+"."+champ+"/total", float64(len(sous)))
		mesurerTableau(e, axe, prefixe+"."+champ, sous, profondeur+1)
		return
	}
	s, estTexte := v.(string)
	if estTexte && s == "" {
		return
	}
	e.incr(axe, prefixe+"."+champ+"/presents", 1)
	if !estTexte {
		return
	}
	switch {
	case champ == "xuid":
		e.incr(axe, prefixe+"/par-xuid/"+s, 1)
	case champsFamille[champ]:
		e.incr(axe, prefixe+"/par-"+champ+"/"+s, 1)
	}
}

// mesurerPistes : les trajectoires. Le nombre de vies, leurs points, et surtout COMBIEN de vies
// portent une identite — le pont d'identite est ce qui rend un rejeu lisible, et sa perte ne se
// voit dans aucun compte d'elements.
func mesurerPistes(e *Empreinte, items []any) {
	if len(items) == 0 {
		return
	}
	const axe = "pistes"
	e.num(axe, "tracks/points", compteLongueurs(items, "points"))
	e.num(axe, "tracks/vies-nommees", comptePresents(items, "xuid"))
	e.num(axe, "tracks/vies-bot", comptePresents(items, "bot"))
	xuids := distincts(items, "xuid")
	e.num(axe, "tracks/xuids-distincts", float64(len(xuids)))
	for _, x := range xuids {
		var vies float64
		for _, it := range items {
			if chaine(objet(it)["xuid"]) == x {
				vies++
			}
		}
		e.num(axe, "tracks/vies-par-xuid/"+x, vies)
	}
}

// mesurerRoster : le pont d'identite lui-meme.
func mesurerRoster(e *Empreinte, items []any) {
	if len(items) == 0 {
		return
	}
	const axe = "roster"
	e.num(axe, "roster/nommes", comptePresents(items, "name"))
	e.num(axe, "roster/bots", comptePresents(items, "bot"))
	e.num(axe, "roster/xuids-distincts", float64(len(distincts(items, "xuid"))))
	for _, x := range distincts(items, "xuid") {
		e.num(axe, "roster/present/"+x, 1)
	}
}

// mesurerObjectifs croise le JOUEUR et la FAMILLE d'action : c'est le seul croisement que la
// mesure generique ne fait pas, et c'est celui qui a revele la perte des captures de drapeau
// (les comptes par famille et par joueur pris separement restaient plausibles).
func mesurerObjectifs(e *Empreinte, items []any) {
	if len(items) == 0 {
		return
	}
	const axe = "objectifs"
	for _, it := range items {
		obj := objet(it)
		stat, xuid := chaine(obj["stat"]), chaine(obj["xuid"])
		if stat == "" {
			continue
		}
		if xuid == "" {
			xuid = "sans-identite"
		}
		e.incr(axe, "objectives/par-joueur/"+xuid+"/"+stat, 1)
	}
}

// mesurerScore : les series de score. La mesure retenue est la DERNIERE valeur de chaque serie
// — le compteur de fin de match, celui qu'un joueur reconnait sur sa feuille.
func mesurerScore(e *Empreinte, st map[string]any) {
	if st == nil {
		return
	}
	const axe = "score"
	aplatir(e, axe, "scoreTimeline", map[string]any{
		"targetScore": st["targetScore"], "teamIdentity": st["teamIdentity"],
		"rounds": st["rounds"], "modeSupported": st["modeSupported"],
		"truncated": st["truncated"], "oracle": st["oracle"],
	}, 0)
	for i, t := range tableau(st["teams"]) {
		obj := objet(t)
		nom := strconv.Itoa(i)
		if n, ok := nombre(obj["teamId"]); ok {
			nom = strconv.Itoa(int(n))
		}
		if v, ok := derniereValeur(obj); ok {
			e.num(axe, "score/equipe/"+nom+"/final", v)
		}
		e.num(axe, "score/equipe/"+nom+"/manches", float64(len(tableau(obj["rounds"]))))
	}
	mesurerJoueursScore(e, tableau(st["players"]))
}

// mesurerJoueursScore pose les compteurs de fin de match par joueur : frags, morts,
// assistances, points.
func mesurerJoueursScore(e *Empreinte, joueurs []any) {
	const axe = "joueurs"
	series := []string{"score", "kills", "deaths", "assists"}
	for _, p := range joueurs {
		obj := objet(p)
		xuid := chaine(obj["xuid"])
		if xuid == "" {
			continue
		}
		e.num(axe, "joueur/"+xuid+"/present", 1)
		if n, ok := nombre(obj["teamId"]); ok {
			e.num(axe, "joueur/"+xuid+"/equipe", n)
		}
		for _, s := range series {
			if v, ok := derniereValeur(objet(obj[s])); ok {
				e.num(axe, "joueur/"+xuid+"/"+s, v)
			}
		}
	}
}

// derniereValeur rend la derniere valeur d'une serie de score : le dernier point de `total`
// quand il existe, sinon celui de la derniere manche. Une serie vide ne rend RIEN — un zero
// affirme un compteur nul la ou il n'y a pas de mesure.
func derniereValeur(serie map[string]any) (float64, bool) {
	if serie == nil {
		return 0, false
	}
	if v, ok := dernierTick(tableau(serie["total"])); ok {
		return v, true
	}
	manches := tableau(serie["rounds"])
	var max float64
	var vu bool
	for _, m := range manches {
		if v, ok := dernierTick(tableau(objet(m)["points"])); ok {
			max += v
			vu = true
		}
	}
	return max, vu
}

// dernierTick rend la valeur `v` du dernier point d'une serie, l'axe `t` faisant foi pour
// l'ordre (un tableau relu d'un JSON n'est pas garanti trie).
func dernierTick(points []any) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	idx := make([]int, len(points))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ta, _ := nombre(objet(points[idx[a]])["t"])
		tb, _ := nombre(objet(points[idx[b]])["t"])
		return ta < tb
	})
	v, ok := nombre(objet(points[idx[len(idx)-1]])["v"])
	return v, ok
}
