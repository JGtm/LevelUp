package persist

// kill_events_merge_adossement.go — TOUTE MORT D HUMAIN SERVIE S ADOSSE A UN EVENEMENT DEATH DE L API.
//
// Le kill-feed de l API porte un evenement `death` par mort d humain (le xuid de la victime, un
// instant). Une mort servie deux fois — la ligne du credit et une ligne de film decalee de 1 ms,
// ou deux lignes de film — s adosse deux fois au MEME evenement. La regle :
//
//   - chaque ligne servie dont la victime a un xuid (un humain ; un bot n en a pas, et sa mort ne
//     produit aucun evenement) prend l evenement `death` LIBRE de ce xuid le plus proche de son
//     instant, a [analysis.KillDeathPairToleranceMS] pres ; un evenement ne sert qu une ligne ;
//   - les morts de credit passent d abord (elles viennent de ces evenements, une par evenement) ;
//     une mort de credit sans evenement est conservee et comptee — le credit fait foi ;
//   - une ligne de film ORPHELINE a victime humaine qui ne trouve pas d evenement libre n est PAS
//     servie : elle est refusee et comptee ([MergeStats.OrphansSansMortAPI]).
//
// Sans preuve lue ([MergeEvidence.Known] faux), aucun adossement n est applique.

import (
	"sort"

	"levelup/go-api/internal/analysis"
)

// adossement : les evenements `death` de l API d un match, par xuid, et ceux deja pris.
type adossement struct {
	morts map[string][]evenementMort
}

// evenementMort : un evenement `death` de l API.
type evenementMort struct {
	ms   int
	pris bool
}

// nouvelAdossement : les evenements `death` de `ev`, tries par instant pour chaque xuid.
func nouvelAdossement(ev MergeEvidence) *adossement {
	a := &adossement{morts: make(map[string][]evenementMort, len(ev.APIDeaths))}
	for x, instants := range ev.APIDeaths {
		es := make([]evenementMort, 0, len(instants))
		for _, t := range instants {
			es = append(es, evenementMort{ms: t})
		}
		sort.Slice(es, func(i, j int) bool { return es[i].ms < es[j].ms })
		a.morts[x] = es
	}
	return a
}

// prendre consomme l evenement `death` libre de `xuid` le plus proche de `ms` (a egalite, le plus
// ancien), dans la fenetre d appariement. Faux quand il n y en a aucun.
func (a *adossement) prendre(xuid string, ms int) bool {
	tol := int(analysis.KillDeathPairToleranceMS)
	es := a.morts[xuid]
	meilleur, ecart := -1, 0
	for i := range es {
		d := es[i].ms - ms
		if d < 0 {
			d = -d
		}
		if es[i].pris || d > tol {
			continue
		}
		if meilleur < 0 || d < ecart {
			meilleur, ecart = i, d
		}
	}
	if meilleur < 0 {
		return false
	}
	es[meilleur].pris = true
	return true
}

// adosserLeCredit : chaque mort de credit a victime humaine prend son evenement ; rend le nombre
// de celles qui n en trouvent pas (conservees).
func (a *adossement) adosserLeCredit(deaths []KillEventInsert) int {
	sans := 0
	for i := range deaths {
		if v := deaths[i].VictimXUID; v != "" && !a.prendre(v, deaths[i].TimeMS) {
			sans++
		}
	}
	return sans
}

// orphelineServie : une ligne de film orpheline peut-elle etre servie ? Une victime sans xuid (un
// bot) toujours ; une victime humaine si un evenement libre l adosse (consomme alors).
func (a *adossement) orphelineServie(d KillEventInsert) bool {
	return d.VictimXUID == "" || a.prendre(d.VictimXUID, d.TimeMS)
}
