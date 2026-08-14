// dump.go — LE DETAIL PAR ACTION, pour confronter l'attribution a un releve terrain.
//
// Les tableaux agreges disent COMBIEN d'actions ont ete rattachees a une zone. Ils ne
// disent jamais si c'est LA BONNE — et c'est le chiffre qui manque a tout ce chantier.
// Le seul temoin externe existant est le releve a l'oeil du 2026-08-02 sur Vagabond
// (`.ai/V7.5/film_re/RELEVE_TERRAIN_CAPTURES_2026-07-31.md`), qui date quatre instants
// d'un film precis. Confronter l'un a l'autre exige de sortir les actions UNE PAR UNE,
// avec leur auteur nomme et leur instant en secondes de film.
//
// CE QUE CE MODE NE PEUT PAS FAIRE, et il faut le dire ici plutot que le decouvrir en
// lisant la sortie : le releve nomme « la base B ». La lettre affichee en jeu n'existe
// dans aucune donnee decodee (etat de l'art Forge/zones, § NOMS DE ZONE) et le rang
// spatial publie ici n'en est PAS une (objectives_catalog.go). La confrontation porte donc
// sur ce que le releve dit d'autre — l'auteur, l'instant, et le NOMBRE de zones en jeu —
// jamais sur une egalite de lettre qu'aucune des deux sources ne porte.
package main

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/replay"
)

// printActionDump imprime chaque action d'objectif du match avec son attribution APRES
// correction d'horloge, triee par instant.
//
// L'instant est donne en SECONDES DE FILM (l'axe du releve terrain, qui compte depuis le
// debut du film) et non en frames : le releve dit « 0:48 », pas « frame 489 ».
func printActionDump(res []result, tune runTuning) {
	for _, r := range res {
		if r.err != nil || !r.hasOrigin {
			continue
		}
		fmt.Printf("DETAIL PAR ACTION — %s (%s), origine %d ms\n", r.m.short, r.m.mapName, r.originMS)
		fmt.Printf("%9s %-18s %-16s %8s %6s %9s\n",
			"t film", "joueur", "statistique", "attribue", "rang", "distance")
		att, _ := r.attribute(r.corrected, r.zones, tune.maxGap, 0)
		rows := append([]replay.ZoneAttribution(nil), att...)
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Action.TimeMS < rows[j].Action.TimeMS })
		names := namesByXUID(r.roster)
		for _, a := range rows {
			verdict, rank := "hors zone", "-"
			if a.Attributed {
				verdict, rank = "DEDANS", fmt.Sprintf("%d", a.SpatialRank)
			} else if !a.HasSample {
				verdict = "sans pos"
			}
			name := names[a.Action.XUID]
			if name == "" {
				name = a.Action.XUID
			}
			fmt.Printf("%8.1fs %-18s %-16s %8s %6s %8.1fm\n",
				float64(a.Action.TimeMS)/1000, name, a.Action.Stat, verdict, rank, a.DistanceM)
		}
		fmt.Println()
		printZoneSpread(rows)
	}
}

// printAgreement publie la CONCORDANCE INTER-JOUEURS : quand plusieurs joueurs agissent
// sur la meme zone au meme instant, leur attribution designe-t-elle la MEME zone ?
//
// # Pourquoi c'est le seul oracle d'identite disponible
//
// Tout le reste de ce rapport mesure un TAUX : « une zone a-t-elle ete rattachee ». Le
// releve terrain du 2026-08-02 est le seul temoin externe, et il nomme sa base par une
// LETTRE que rien dans la donnee decodee ne porte (cf. entete de ce fichier) : il ne peut
// donc pas arbitrer une egalite de zone. La concordance, elle, le peut — sans lettre.
//
// Une prise de Bastion est un evenement DE ZONE : les coequipiers qui la portent au meme
// instant sont sur la MEME base. Leurs positions, elles, sont decodees independamment les
// unes des autres — rien dans le croisement ne les force a tomber ensemble. Si
// l'attribution etait du bruit reparti sur les trois zones d'une carte, deux joueurs
// concorderaient une fois sur trois.
//
// LE TEMOIN N'EST PAS OPTIONNEL : les memes groupes, decales de 30 s. Il repond a
// l'objection « les coequipiers sont de toute facon groupes » — si c'etait la seule cause,
// le temoin concorderait autant.
func printAgreement(res []result, tune runTuning) {
	fmt.Println("CONCORDANCE INTER-JOUEURS — deux joueurs qui agissent au meme instant")
	fmt.Println("designent-ils la MEME zone ? (hasard attendu sur 3 zones : 33 %)")
	fmt.Printf("%-9s %-14s %8s %8s %8s %8s %8s %8s\n",
		"film", "carte", "groupes", "concord", "taux", "tGroupes", "tConcord", "tTaux")
	var okTot, grpTot, okNull, grpNull int
	for _, r := range res {
		if r.err != nil || !r.hasOrigin {
			continue
		}
		att, _ := r.attribute(r.corrected, r.zones, tune.maxGap, 0)
		nul, _ := r.attribute(r.correctedNull, r.zones, tune.maxGap, 0)
		ok, grp := agreementOf(att)
		okN, grpN := agreementOf(nul)
		okTot, grpTot, okNull, grpNull = okTot+ok, grpTot+grp, okNull+okN, grpNull+grpN
		fmt.Printf("%-9s %-14s %8d %8d %8s %8d %8d %8s\n",
			r.m.short, r.m.mapName, grp, ok, rateOf(ok, grp), grpN, okN, rateOf(okN, grpN))
	}
	fmt.Printf("%-9s %-14s %8d %8d %8s %8d %8d %8s\n",
		"TOTAL", "", grpTot, okTot, rateOf(okTot, grpTot), grpNull, okNull, rateOf(okNull, grpNull))
	fmt.Println()
}

// agreementOf compte les groupes d'actions SIMULTANEES comptant au moins deux attributions,
// et ceux dont toutes les attributions designent la meme zone.
//
// Le groupe est l'instant EXACT (TimeMS) : c'est une seule emission du statborg, donc un
// seul fait de jeu. Elargir la fenetre melangerait deux prises voisines et fabriquerait
// des desaccords qui n'en sont pas.
func agreementOf(att []replay.ZoneAttribution) (agreed, groups int) {
	byTime := map[int][]int{}
	for _, a := range att {
		if !a.Attributed {
			continue
		}
		byTime[a.Action.TimeMS] = append(byTime[a.Action.TimeMS], a.SpatialRank)
	}
	for _, ranks := range byTime {
		if len(ranks) < 2 {
			continue
		}
		groups++
		same := true
		for _, r := range ranks[1:] {
			if r != ranks[0] {
				same = false
				break
			}
		}
		if same {
			agreed++
		}
	}
	return agreed, groups
}

// rateOf formate un taux, ou « - » quand le denominateur est nul : un pourcentage sur zero
// groupe se lirait comme une mesure alors qu'il n'y en a pas.
func rateOf(n, d int) string {
	if d == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(n)/float64(d))
}

// namesByXUID indexe le roster du film par xuid. Le nom vient du film lui-meme (aucune
// resolution externe), ce qui le rend confrontable a un releve fait a l'oeil devant l'ecran.
func namesByXUID(roster []replay.RosterEntry) map[string]string {
	out := map[string]string{}
	for _, e := range roster {
		if e.XUID != "" && e.Name != "" {
			out[e.XUID] = e.Name
		}
	}
	return out
}

// printZoneSpread compte les actions attribuees PAR RANG DE ZONE.
//
// C'est le seul controle d'identite que la donnee autorise sans lettre : une carte de
// Bastion a trois zones jouees, et une attribution qui en concentrerait la quasi-totalite
// sur une seule dirait qu'elle designe un lieu plutot qu'une zone. La repartition
// attendue n'est pas l'uniformite — les equipes ne se disputent pas les trois bases a
// egalite — mais les trois rangs doivent etre representes.
func printZoneSpread(rows []replay.ZoneAttribution) {
	byRank := map[int]int{}
	total := 0
	for _, a := range rows {
		if !a.Attributed {
			continue
		}
		byRank[a.SpatialRank]++
		total++
	}
	ranks := make([]int, 0, len(byRank))
	for k := range byRank {
		ranks = append(ranks, k)
	}
	sort.Ints(ranks)
	fmt.Printf("  repartition des %d attributions par rang de zone :", total)
	for _, k := range ranks {
		fmt.Printf("  rang %d = %d", k, byRank[k])
	}
	fmt.Println()
	fmt.Println()
}
