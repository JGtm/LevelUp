package killsource

// health.go — LA METRIQUE DE SANTE : LES CANDIDATS QUE RIEN NE PUBLIE.
//
// CE QU ELLE SURVEILLE. Le decodeur repose sur des hypotheses qui peuvent cesser d etre vraies
// SANS PRODUIRE LA MOINDRE ERREUR : un catalogue de tags perime par une saison neuve, un roster
// plus grand que prevu, un mode de jeu inconnu. Aucune de ces situations ne leve d exception —
// elles se paient en COUVERTURE SILENCIEUSE.
//
// LES CANDIDATS INEXPLIQUES NE SORTENT PAS et ne coutent rien au consommateur. C est leur TAUX
// qui informe, par comparaison avec la distribution observee : 7.0 / 9.4 / 11.6 / 17.8 % sur les
// quatre films de reference, 27.0 % en BTB. Jamais leur existence.
//
// LE COMPTEUR PRINCIPAL EST << TAG HORS CATALOGUE, VU PAR LA MARCHE >>. Distribution observee :
// ZERO sur cinq films. Controle positif : 20 ablations d un tag reel sur 20 le font sonner, dont
// 8 fois SANS que la couverture ne bouge — il previent donc AVANT la perte, pas apres.
//
// DEUX PORTEES A CITER AVEC CES CHIFFRES, et elles ont ete mesurees contre la formulation trop
// large qui les precedait (RE_LOG 7ter.73 (4a)(4d)) :
//
//	SON BRUIT EST MESURE NUL SUR CINQ FILMS, pas << nul par construction >>. L argument
//	structurel est fort — la marche n a pas de porte de catalogue et l appariement par couple
//	exact a deja prouve que l enregistrement est reel — mais sur le corpus de reference AUCUN
//	enregistrement de la marche ne porte un tag hors catalogue : la porte n a jamais eu a
//	rejeter quoi que ce soit. Nul PAR ABSENCE DE CAS.
//
//	LE 20/20 VAUT SUR LES TAGS QUE LA MARCHE PORTE (les cinq plus frequents de chaque film), pas
//	sur << un catalogue perime >> en general. La 21e ablation, sur un tag servi par le SCAN SEUL,
//	rend un compteur a ZERO et aucune alerte. Dans ce mode, c est le plancher de couverture a
//	1.00 qui est le filet — voir `filmdec.CoverageWarnRatio`.
//
// LA SONDE A PORTE RELACHEE EST PUBLIEE MAIS EXCLUE DES ALERTES, et c est mesure : son rapport
// signal/hasard vaut 1.10 a 1.72 selon le film. Elle n est lisible que restreinte aux morts NON
// COUVERTES — ou elle vaut zero par construction quand la couverture est complete. Elle coute
// aussi 6 min 40 s sur un BTB : elle n a pas sa place dans un pipeline de production, d ou
// [Options] absent et [decodeCtx.relaxedProbe] non appele par [Decode].
//
// PIEGE ATTRAPE EN MESURE, et il valait une section : sans exclusion du couple FANTOME la sonde
// alerte sur une mort QUI N EXISTE PAS. Le fantome est compte comme couvert.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// health : la mesure complete.
func (c *decodeCtx) health(p *pass, cov Coverage) filmdec.KillSourceHealth {
	nWalk, _ := c.walkOutOfCatalogue()
	return filmdec.KillSourceHealth{
		Film:                  c.name,
		Candidates:            len(p.all),
		Published:             len(p.byTime),
		UnexplainedPair:       p.unexpPair,
		UnexplainedSelf:       p.unexpSelf,
		UnexplainedBotIdx:     p.unexpBot,
		OutOfRoster:           c.outOfRoster(),
		TagOutOfCatalogueWalk: nWalk,
		DeathsReal:            cov.RealPairs,
		DeathsCovered:         cov.Covered,
	}
}

// walkOutOfCatalogue : LE COMPTEUR PRINCIPAL. Enregistrements de la MARCHE apparies au kill-feed
// par le COUPLE EXACT et dont le tag est ABSENT du catalogue. Rend aussi les tags fautifs, pour
// que la regeneration sache quoi ajouter.
func (c *decodeCtx) walkOutOfCatalogue() (int, []uint32) {
	cands, _ := c.walkRes.candidates()
	var tags []uint32
	seen := map[uint32]bool{}
	n := 0
	for _, cd := range cands {
		if isCatalogued(cd.tag) || c.matchExact(cd) == nil {
			continue
		}
		n++
		if !seen[cd.tag] {
			seen[cd.tag] = true
			tags = append(tags, cd.tag)
		}
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i] < tags[j] })
	return n, tags
}

// outOfRoster : dead-states de la marche, dans la plage bipede, dont un indice DEPASSE le roster
// retenu. C est le signal << il y a un participant que nous ne comptons pas >>.
//
// LE COMPTEUR A ETE DURCI PAR SA PROPRE MESURE. La premiere version comptait TOUT dead-state hors
// roster : elle rendait 2 sur un film NOMINAL et declenchait une alerte fausse. Les deux lignes
// portaient `tag=ffffffff` et un indice a -1 — du DECHET de decodage, pas un participant non
// compte. Le compteur exige donc que le TAG SOIT AU CATALOGUE : le meme test que le scan, employe
// ici comme test de plausibilite. Apres durcissement : 0 sur les quatre films de reference, et
// TROIS VRAIS sur le BTB (indices 28/29/30 pour 36 participants) — sa premiere detection positive.
func (c *decodeCtx) outOfRoster() int {
	n := 0
	for _, d := range c.walkRes.deads {
		if d.slot < c.walkRes.bipLo || d.slot > c.walkRes.bipHi {
			continue
		}
		if int(d.dead.EnumA) < c.roster.nPlay && int(d.dead.EnumB) < c.roster.nPlay {
			continue
		}
		if !isCatalogued(d.dead.SrcTag0) {
			continue
		}
		n++
	}
	return n
}

// RelaxedProbe : le resultat de la sonde a porte de catalogue RELACHEE.
type RelaxedProbe struct {
	// Candidates : candidats structurels, tout tag confondu.
	Candidates int
	// OutOfCatalogue : dont le tag est hors du catalogue.
	OutOfCatalogue int
	// Paired : hors catalogue ET couple exact au feed dans la fenetre. BRUITE — rapport
	// signal/hasard 1.10 a 1.72. Ne pas alerter dessus.
	Paired int
	// Uncovered : ... et la mort n est couverte par personne. Le seul signal exploitable, et il
	// vaut zero par construction quand la couverture est complete.
	Uncovered int
	// Tags : les tags distincts de `Uncovered`.
	Tags []uint32
}

// relaxedProbe : le meme gabarit que le scan, sans la porte du catalogue, sur les paquets A
// EVENTS (les morts y sont toutes).
//
// ELLE NE TOURNE QUE SI LA COUVERTURE EST INCOMPLETE — c est exactement le seul regime ou elle
// porte de l information, et cela lui evite de couter quoi que ce soit sur un film nominal.
func (c *decodeCtx) relaxedProbe(covered map[int]bool) RelaxedProbe {
	var st RelaxedProbe
	seen := map[uint32]bool{}
	for i := range c.film.t0 {
		p := &c.film.t0[i]
		if !hasEvents(p) {
			continue
		}
		ms := c.film.ms(p)
		for _, cd := range scanRelaxedPayload(p.payload, c.roster.nPlay) {
			st.Candidates++
			if isCatalogued(cd.tag) {
				continue
			}
			st.OutOfCatalogue++
			cd.chunk, cd.pidx, cd.ms = p.chunk, p.idx, ms
			e := c.matchExact(cd)
			if e == nil {
				continue
			}
			st.Paired++
			if covered[e.timeMS] {
				continue
			}
			st.Uncovered++
			if !seen[cd.tag] {
				seen[cd.tag] = true
				st.Tags = append(st.Tags, cd.tag)
			}
		}
	}
	sort.Slice(st.Tags, func(i, j int) bool { return st.Tags[i] < st.Tags[j] })
	return st
}
