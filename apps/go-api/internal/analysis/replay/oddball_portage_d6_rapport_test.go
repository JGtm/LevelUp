package replay

// oddball_portage_d6_rapport_test.go — CE QUE D6 PUBLIE : les classes, les distances, ce que fait
// la mort, et le verdict contre l'oracle API.
//
// Il vit a part de la reconstruction pour la meme raison que partout ailleurs dans ce paquet : le
// fichier de mesure garde la CHAINE, celui-ci porte ce qu'elle DIT. Les deux sous le seuil du
// depot.

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// d6PublieClasses imprime la repartition des trous par classe.
//
// LES QUATRE CLASSES SE COMPTENT SEPAREMENT, et c'est le coeur de la reponse a ma reserve
// d'hier : « un trou n'est pas forcement un portage ». Un RETOUR n'est pas un echec
// d'attribution, c'est un cas correctement reconnu — les melanger ferait payer a la
// reconstruction une erreur qu'elle ne commet pas.
func d6PublieClasses(t *testing.T, id string, trous []d6Trou) {
	t.Helper()
	n := map[string]int{}
	var secondes float64
	for _, tr := range trous {
		n[tr.classe]++
		secondes += tr.dureeS()
	}
	t.Logf("%s : %d trou(s) — %d PORTE, %d ambigu, %d retour, %d inexplique ; "+
		"%.0f s de portage attribuees", id, len(trous), n["porte"], n["ambigu"],
		n["retour"], n["inexplique"], secondes)
}

// d6PublieDistances imprime la DISTRIBUTION des distances au plus proche.
//
// ELLE EST PUBLIEE PARCE QUE LE SEUIL NE VAUT QUE PAR ELLE. Si les deux populations — « quelqu'un
// ramasse » et « personne n'est la » — ne se separent pas, aucun seuil ne les separera, et il
// faudra le dire plutot que de s'en servir. C'est la meme exigence qui a valide
// `originDropMaxDist` des deux cotes.
func d6PublieDistances(t *testing.T, id string, trous []d6Trou) {
	t.Helper()
	var ds []float64
	for _, tr := range trous {
		if tr.distM != math.MaxFloat64 {
			ds = append(ds, tr.distM)
		}
	}
	if len(ds) == 0 {
		t.Logf("DISTANCES %s : aucune distance confrontable", id)
		return
	}
	sort.Float64s(ds)
	q := func(p float64) float64 { return ds[int(float64(len(ds)-1)*p)] }
	sous := 0
	for _, d := range ds {
		if d <= d6RayonRamassageM {
			sous++
		}
	}
	t.Logf("DISTANCES %s : %d mesurees — min %.2f m, q25 %.2f, MEDIANE %.2f, q75 %.2f, "+
		"q90 %.2f, max %.1f ; %d sous le seuil de %.1f m (%.1f %%)",
		id, len(ds), ds[0], q(0.25), q(0.50), q(0.75), q(0.90), ds[len(ds)-1],
		sous, d6RayonRamassageM, 100*float64(sous)/float64(len(ds)))
}

// d6PublieMort mesure CE QUE FAIT LA MORT (volet D6.2) : un porteur qui meurt lache-t-il ?
//
// L'HYPOTHESE EST TESTEE, PAS SUPPOSEE. La regle de cloture du protocole (« le portage s'arrete
// a la mort du porteur ») n'est fondee que si une vie libre nait effectivement pres du lieu de la
// mort. Ce chiffre ne conditionne pas le gate : il dit si cette regle repose sur quelque chose.
func d6PublieMort(t *testing.T, id string, vies []flagFreeLife, trous []d6Trou,
	tracks map[uint32]slotTrack, pont objBridge,
) {
	t.Helper()
	confrontables, suivies := 0, 0
	for _, tr := range trous {
		if tr.classe != "porte" || tr.finPortageUS >= tr.finUS {
			continue // aucune mort n'a ferme ce portage
		}
		confrontables++
		if d6VieApres(vies, tr, tracks, pont) {
			suivies++
		}
	}
	t.Logf("MORT %s : %d portage(s) ferme(s) par la mort du porteur ; %s suivis d'une naissance "+
		"de vie libre a <= %.0f m du lieu de la mort dans les %d ms",
		id, confrontables, d6Part(suivies, confrontables), d6MortRayonM, d6MortFenetreMS)
}

// d6VieApres dit si une vie libre nait pres du LIEU de la mort qui a ferme ce portage.
//
// LE LIEU DE LA MORT N'EST PAS DANS LE FIL DES MORTS — il ne porte que le xuid et l'instant. Il
// se lit donc sur la PISTE DE LA VICTIME a cet instant, la meme source que la proximite du
// ramassage : une seule geometrie pour les deux bouts du portage, jamais deux.
func d6VieApres(vies []flagFreeLife, tr d6Trou, tracks map[uint32]slotTrack, pont objBridge) bool {
	mx, my, trouve := float32(0), float32(0), false
	for slot, track := range tracks {
		if pont.SlotXUID[slot] != tr.xuid {
			continue
		}
		p, ecart := track.at(tr.finPortageUS)
		if ecart <= d6EcartMaxMS*1000 {
			mx, my, trouve = p.X, p.Y, true
			break
		}
	}
	if !trouve {
		return false
	}
	for _, l := range vies {
		if l.T0US < tr.finPortageUS || l.T0US > tr.finPortageUS+d6MortFenetreMS*1000 {
			continue
		}
		x, y := l.First()
		if math.Hypot(float64(x)-float64(mx), float64(y)-float64(my)) <= d6MortRayonM {
			return true
		}
	}
	return false
}

// d6Verdict confronte la reconstruction a l'oracle API, avec son temoin.
func d6Verdict(t *testing.T, id string, rec, tem, oracle map[string]float64) {
	t.Helper()
	var totalAPI float64
	xuids := make([]string, 0, len(oracle))
	for x, s := range oracle {
		totalAPI += s
		xuids = append(xuids, x)
	}
	sort.Strings(xuids)
	if totalAPI <= 0 {
		t.Logf("NON EXPLOITABLE %s : temps de portage API nul. NI POUR NI CONTRE.", id)
		return
	}
	for _, x := range xuids {
		t.Logf("  %s : API %.0f s · reconstruit %.0f s · temoin %.0f s",
			x[len(x)-4:], oracle[x], rec[x], tem[x])
	}
	recouv := d6Recouvrement(rec, oracle) / totalAPI
	recTem := d6Recouvrement(tem, oracle) / totalAPI
	principal := d6MemePrincipal(rec, oracle)
	t.Logf("SIGNAL %s : recouvrement %.1f %% (seuil %.0f %%) ; temoin %.1f %% (seuil <= %.0f %%) ; "+
		"porteur principal %s", id, 100*recouv, 100*d6SeuilRecouvrement, 100*recTem,
		100*d6SeuilTemoin, d6Oui(principal))
	switch {
	case recouv >= d6SeuilRecouvrement && recTem <= d6SeuilTemoin:
		t.Logf("VERDICT %s : les DEUX bornes du recouvrement sont tenues sur ce film.", id)
	case recouv >= d6SeuilRecouvrement:
		t.Logf("VERDICT %s : le recouvrement tient mais le TEMOIN aussi — une attribution au "+
			"hasard rend le meme signal, donc la proximite n'explique rien.", id)
	default:
		t.Logf("VERDICT %s : le recouvrement est SOUS le seuil.", id)
	}
}

// d6Recouvrement somme, par joueur, le minimum du reconstruit et de l'oracle.
//
// LE MINIMUM PLUTOT QU'UN ECART RELATIF : c'est une mesure de RECOUVREMENT bornee dans [0, total],
// insensible au decoupage et qui ne recompense jamais une sur-attribution — attribuer mille
// secondes a un joueur qui en a dix n'en fait gagner que dix.
func d6Recouvrement(rec, oracle map[string]float64) float64 {
	var s float64
	for x, api := range oracle {
		if r := rec[x]; r < api {
			s += r
		} else {
			s += api
		}
	}
	return s
}

// d6MemePrincipal dit si le plus gros porteur de l'oracle est aussi celui du reconstruit.
// CRITERE SANS SEUIL REGLABLE : il ne se negocie pas a la lecture du chiffre.
func d6MemePrincipal(rec, oracle map[string]float64) bool {
	return d6Max(oracle) != "" && d6Max(oracle) == d6Max(rec)
}

// d6Max rend la cle de plus grande valeur, vide si la carte l'est. Ordre total sur egalite.
func d6Max(m map[string]float64) string {
	best, bestV := "", math.Inf(-1)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if m[k] > bestV {
			best, bestV = k, m[k]
		}
	}
	return best
}

// d6Part rend un taux lisible. UN TAUX SANS DENOMINATEUR N'EST PAS ZERO.
func d6Part(n, d int) string {
	if d == 0 {
		return "pas de denominateur"
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}

// d6Oui rend un booleen lisible.
func d6Oui(b bool) string {
	if b {
		return "IDENTIFIE"
	}
	return "MANQUE"
}
