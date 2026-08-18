package replay

import (
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// flag_carries_marker.go — LE CONTROLE INDEPENDANT, et les comptes qu'il alimente.
//
// POURQUOI UN CONTROLE, ALORS QUE LES BORNES SONT DEJA EXACTES. Les bornes d'un portage viennent
// des COMPTEURS DE STATISTIQUE du statborg : elles disent qu'un joueur a pris le drapeau, pas
// qu'il le tenait encore trois secondes plus tard. Le LACHER VOLONTAIRE, en particulier, n'est
// date par rien (cf. flag_carries.go) — un portage peut donc etre trop long, et rien dans sa
// propre chaine ne le dirait.
//
// LE MARQUEUR EST UNE SECONDE CHAINE, ENTIEREMENT DISJOINTE : une suite de 32 bits presente dans
// le record de bipede d'une image-cle quand ce joueur porte quelque chose
// (`filmdec.ScanFilmCarrierMarks`). Elle ne partage avec la premiere ni sa source (des bits du
// dump d'etat monde contre des compteurs repliques), ni son espace de slots (bipede contre
// statborg), ni son horloge (film contre match). Leur accord est donc une preuve.
//
// CE QU'IL NE PEUT PAS FAIRE : confirmer un portage pendant lequel AUCUNE image-cle ne tombe.
// Les images-cles sont espacees de ~20 s et beaucoup de portages durent moins. C'est pourquoi le
// denominateur publie n'est pas « tous les portages » mais « les portages OBSERVABLES » —
// `MarkerObserved` — et que les deux comptes voyagent ensemble.

// markFlagCarries pose le CONTROLE du marqueur sur chaque portage : y a-t-il eu une image-cle
// pendant qu'il durait, et cette image-cle portait-elle le marqueur sur le slot du porteur ?
func markFlagCarries(raws []flagCarryRaw, scan filmdec.CarrierMarkScan, ctx flagCarryCtx) {
	marked := map[string]map[int64]bool{}
	for _, m := range scan.Marks {
		x, ok := ctx.slotXUID[m.Slot]
		if !ok {
			continue
		}
		key := strconv.FormatUint(x, 10)
		if marked[key] == nil {
			marked[key] = map[int64]bool{}
		}
		marked[key][int64(m.TimestampUS/1000)-ctx.deathOffsetMS] = true
	}
	kfMatchMS := make([]int64, 0, len(scan.KeyframeUS))
	for _, us := range scan.KeyframeUS {
		kfMatchMS = append(kfMatchMS, int64(us/1000)-ctx.deathOffsetMS)
	}
	for i := range raws {
		for _, at := range kfMatchMS {
			if at < raws[i].t0 || at > raws[i].t1 {
				continue
			}
			raws[i].observable = true
			if marked[raws[i].xuid][at] {
				raws[i].confirmed = true
				break
			}
		}
	}
}

// tallyFlagCarries porte les comptes du calque dans la couverture.
//
// LE CONTROLE DU MARQUEUR SE COMPTE DEUX FOIS, ET C'EST LE POINT. Les portages FERMES et les
// portages OUVERTS ne mesurent pas la meme chose : un portage ouvert est trop long par
// construction, ses images-cles tardives tombent apres le lacher, et aucune ne porte le marqueur.
// Les additionner ferait baisser un taux qui juge la JUSTESSE DES BORNES. Les deux populations
// sont donc comptees separement, et publiees toutes les deux — le taux melange reste calculable.
func tallyFlagCarries(raws []flagCarryRaw, cov *FlagCarriesCoverage) {
	cov.Carries = len(raws)
	for _, r := range raws {
		switch {
		case r.closed:
			cov.Closed++
			if r.observable {
				cov.MarkerObserved++
			}
			if r.confirmed {
				cov.MarkerConfirmed++
			}
		default:
			cov.Open++
			if r.observable {
				cov.OpenObserved++
			}
			if r.confirmed {
				cov.OpenConfirmed++
			}
		}
	}
}

// countFlagOverlaps compte les prises pour lesquelles plus de deux portages sont ouverts. En CTF
// il y a deux drapeaux : au-dela, la lecture se contredit, et on le publie.
//
// Rend DEUX comptes : sur tous les portages, puis sur les seuls FERMES. Le second est celui qui
// juge — un depassement porte par des portages que rien ne ferme est explique par leur duree
// (incertitude deja publiee en [FlagStateCarriedOpen]), la ou un depassement entre portages
// FERMES serait une contradiction entre faits dates.
func countFlagOverlaps(raws []flagCarryRaw) (all, closed int) {
	for i := range raws {
		open, openClosed := 0, 0
		for j := range raws {
			if raws[j].t0 > raws[i].t0 || raws[i].t0 >= raws[j].t1 {
				continue
			}
			open++
			if raws[j].closed {
				openClosed++
			}
		}
		if open > 2 {
			all++
		}
		if openClosed > 2 && raws[i].closed {
			closed++
		}
	}
	return all, closed
}
