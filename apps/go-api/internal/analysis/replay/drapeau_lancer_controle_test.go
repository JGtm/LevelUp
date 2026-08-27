package replay

// drapeau_lancer_controle_test.go — LE CONTROLE ELARGI : ET SI ON LANCAIT LE DRAPEAU ?
//
// D'OU VIENT LA QUESTION. Le controle 3 du plan `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md` a
// REFUSE la piste : 149/197 = 75,6 % des vies libres naissent a moins de 1,5 m d'un socle ou du
// porteur qui vient de finir, contre 90 % exiges. Le temoin, lui, tient (12,8 %) : la piste
// discrimine, elle n'est pas du bruit — mais 48 vies restent sans explication, et le diagnostic
// a ecarte la re-creation sur place (3 vies seulement).
//
// L'HYPOTHESE MESUREE ICI, ET ELLE EST DU JEU, PAS DU CODE : on ne LACHE pas seulement le
// drapeau a ses pieds, on peut aussi le LANCER quelques metres devant soi. Si c'est vrai, les 48
// residuelles ne sont pas des objets etrangers : ce sont des drapeaux nes AU BOUT d'un lancer,
// c'est-a-dire a quelques metres du porteur et non a 1,5 m. Un lancer a une PORTEE ; la mesurer,
// c'est balayer le rayon et regarder ou la courbe se pose.
//
// LA REGLE ELARGIE, ECRITE AVANT LA MESURE (c'est la condition de sa valeur) :
//
//	une vie libre est EXPLIQUEE si elle nait a moins de 1,5 m d'un `flag_spawn`  (branche SOCLE,
//	  rayon INCHANGE — un drapeau rendu apparait SUR son socle, pas a huit metres),
//	OU a MOINS DE R metres de la position du porteur, DANS LES 2 s de la fin de son portage
//	  (branche PORTEUR, la seule elargie : c'est elle qui porte le lancer).
//
//	R balaye 1,5 / 3 / 5 / 8 / 10 m. UN SEUL R sera retenu — celui ou la courbe se STABILISE,
//	pas celui qui maximise le score : un rayon qui gagne encore des vies a 10 m ne mesure plus
//	un lancer, il ratisse la carte.
//
// LES DEUX SEUILS DU PLAN SONT INCHANGES : >= 90 % des vies expliquees, et le TEMOIN — la MEME
// regle elargie, au MEME R, appliquee aux creations `ti=42` d'ARMES ORDINAIRES — doit rester
// <= 20 %. LE TEMOIN EST CE QUI REND LE BALAYAGE HONNETE : elargir un rayon fait monter TOUS les
// taux. Si les armes ordinaires montent aussi vite que le drapeau, R ne mesure pas une portee de
// lancer, il mesure la densite des joueurs sur la carte — et le controle ne conclut rien.
//
// CE QUI CHANGE PAR RAPPORT AU CONTROLE 3, ET CE QUI NE CHANGE PAS. Changent : le rayon de la
// branche porteur (balaye) et la fenetre de lacher (1 s -> 2 s, parce qu'un objet lance MET du
// temps a se poser et que sa premiere replication peut suivre la fin du portage). Ne changent
// pas : les seuils, la branche socle, la reference du porteur (sa position A L'INSTANT DE LA
// NAISSANCE, pour tout portage qui couvre cet instant ou vient de s'achever — correction du
// 2026-08-18 sans laquelle le lacher VOLONTAIRE, qui survient au milieu d'un portage, est exclu
// par construction), et le jeu de socles (TOUS les points du role `flag_spawn`).
//
// GARDE : `OBJ_FILM` (racine du cache film) et `OBJ_REPO` (racine du depot). Lecture seule,
// aucune base.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache OBJ_REPO=<depot> \
//	  go test ./internal/analysis/replay/ -run DrapeauLancerControle -v -timeout 60m
//
// ────────────────────────────────────────────────────────────────────────────────────────────
// CE QUE LA MESURE A DONNE (2026-08-18, trois films CTF : 197 vies libres, 950 armes) —
// L'HYPOTHESE DU LANCER EST REFUTEE, ET DEUX FOIS PLUTOT QU'UNE.
//
//	R = 1,5 m   162/197 = 82,2 %   temoin 136/950 = 14,3 %
//	R = 3   m   162/197 = 82,2 %   temoin 151/950 = 15,9 %
//	R = 5   m   162/197 = 82,2 %   temoin 167/950 = 17,6 %
//	R = 8   m   163/197 = 82,7 %   temoin 211/950 = 22,2 %   (le temoin creve son plafond)
//	R = 10  m   164/197 = 83,2 %   temoin 227/950 = 23,9 %
//
// AUCUN R N'ATTEINT 90 %, ET LE RAPPORT EST INVERSE : de 1,5 a 10 m le drapeau gagne DEUX vies,
// le temoin en gagne QUATRE-VINGT-ONZE. Elargir profite dix fois plus aux armes ordinaires qu'au
// drapeau — le rayon ne mesure pas une portee de lancer, il mesure la densite des joueurs.
//
// LA DISTRIBUTION REFUTE SANS MEME PASSER PAR LES SEUILS : sur les 35 residuelles a 1,5 m, 26
// (74 %) n'ont AUCUNE reference porteur dans les 2 s — pas meme un lanceur candidat. Les 9
// mesurables sont a mediane 20,6 m (p90 et max 43,1 m), et les tranches ]1,5-3 m] et ]3-5 m]
// sont VIDES. Un lancer de drapeau porte quelques metres ; a quelques metres, il n'y a rien.
//
// CE QUE LA MESURE APPREND QUAND MEME, ET QUI N'EST PAS TRAITE : a rayon INCHANGE, la fenetre a
// 2 s fait passer le controle 3 de 75,6 % a 82,2 % (treize vies) pour +1,5 point de temoin. Ce
// qui manquait au lacher est le DELAI, pas la distance. Non transporte en production : la
// fenetre de `flag_objects.go` FERME des portages, et sa seconde est bornee pour ne jamais
// rattraper le portage precedent. Piste au registre, pas correctif.
//
// CE TEST RESTE, ET IL RESTE VERT : il ne juge pas, il PUBLIE le balayage. Le refus est ecrit au
// plan `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md` (phase 4), la ou un verdict se conteste.
// ────────────────────────────────────────────────────────────────────────────────────────────

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
)

// objLancerRayons — le balayage de la branche PORTEUR, en metres. Le premier terme est
// `originDropMaxDist` (1,5 m), le rayon du controle 3 : la premiere colonne du tableau REJOUE
// donc la mesure deja publiee, et c'est voulu — sans cette ancre, on ne saurait pas si le
// balayage mesure le lancer ou une derive de l'instrument.
var objLancerRayons = []float64{originDropMaxDist, 3, 5, 8, 10}

// objLancerFenetreUS — la fenetre de la branche porteur, en microsecondes : 2 s.
//
// POURQUOI PAS `flagFreeDropWindowMS` (1 s), LA CONSTANTE DE PRODUCTION. Un drapeau LACHE nait
// aux pieds du porteur, tout de suite : une seconde suffit. Un drapeau LANCE vole, rebondit, puis
// se pose — sa premiere position repliquee peut arriver sensiblement apres la fin du portage. La
// fenetre elargie est donc une consequence de l'hypothese mesuree, pas un seuil de confort ; et
// comme tout elargissement, elle passe par le TEMOIN, qui en profite autant que le drapeau.
const objLancerFenetreUS = 2_000_000

// objLancerItem est UNE naissance mesuree — vie libre du drapeau, ou creation d'arme du temoin.
//
// LA MESURE EST FAITE UNE FOIS, LE BALAYAGE LA RELIT : la distance minimale a une reference
// porteur ne depend pas de R. Recalculer par rayon donnerait cinq fois la meme chose, et cinq
// occasions de diverger.
type objLancerItem struct {
	// socle dit que la naissance est a moins de 1,5 m d'un `flag_spawn` — branche NON elargie.
	socle bool
	// dist est la distance minimale a la position d'un porteur eligible, en metres, ou +Inf
	// quand AUCUN portage ne couvre l'instant ni ne vient de s'achever. Cette distinction
	// compte : une naissance sans aucune reference porteur n'est pas un lancer trop long,
	// c'est une naissance dont le film ne dit rien.
	dist float64
	// delai est l'ecart, en microsecondes, entre la naissance et la FIN du portage retenu
	// (celui de `dist`). Zero quand le portage couvre encore l'instant : c'est le cas du
	// lacher — ou du lancer — VOLONTAIRE, que le film ne date par aucun evenement.
	delai int64
}

// explique dit si l'item est explique par la regle elargie au rayon R.
func (it objLancerItem) explique(r float64) bool { return it.socle || it.dist <= r }

// TestDrapeauLancerControle — la regle elargie sur les trois films CTF du corpus.
func TestDrapeauLancerControle(t *testing.T) {
	root := objRequireRoot(t)
	cat := goldenCatalog(t)
	if len(cat.ObjectiveObjects) == 0 {
		t.Fatal("le manifeste du titre ne declare aucun drapeau — le controle n'a rien a mesurer")
	}
	var vies, temoins []objLancerItem
	joues := 0
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		joues++
		v, w := objLancerFilm(t, root, id, src, cat)
		vies, temoins = append(vies, v...), append(temoins, w...)
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", objFilmEnv, root)
	}
	objLancerTableau(t, "CUMUL DES TROIS FILMS", vies, temoins)
	objLancerDistribution(t, "CUMUL DES TROIS FILMS", vies)
}

// objLancerFilm mesure UN film : ses vies libres, puis son temoin, et publie son tableau.
func objLancerFilm(t *testing.T, root, id string, src *objDiskFilm,
	cat LabelCatalog) (vies, temoins []objLancerItem) {
	t.Helper()
	b := objBridgeOf(t, root, id)
	d := objDocumentDe(t, root, id, b, src)
	step := uint64(d.doc.FrameIntervalMS) * 1000
	if step == 0 {
		t.Fatalf("%s : axe de temps sans echelle — les instants ne se comparent pas", id)
	}
	refs := objDrapeauRefs(t, id, d, step)
	for _, l := range flagFreeLives(d.gw, cat.ObjectiveObjects) {
		x, y := l.First()
		vies = append(vies, objLancerMesure(refs, x, y, l.T0US, step, d.originUS))
	}
	known := loadoutFamilies()
	for _, c := range d.gw.Creations {
		w, res := gwPadsIdentity(c)
		if !res || !known[w] {
			continue
		}
		temoins = append(temoins, objLancerMesure(refs, c.X, c.Y, c.TimestampUS, step, d.originUS))
	}
	objLancerTableau(t, id, vies, temoins)
	objLancerDistribution(t, id, vies)
	return vies, temoins
}

// objLancerMesure calcule, pour UNE naissance, sa branche socle et sa distance minimale a un
// porteur eligible — la mesure que le balayage relit ensuite a chaque rayon.
func objLancerMesure(refs []objDrapeauRef, x, y float32, atUS, step, originUS uint64) objLancerItem {
	it := objLancerItem{dist: math.Inf(1)}
	for _, r := range refs {
		if len(r.porteur) == 0 {
			if math.Hypot(float64(x-r.x), float64(y-r.y)) < originDropMaxDist {
				it.socle = true
			}
			continue
		}
		if atUS+objLancerFenetreUS < r.t0US || atUS > r.t1US+objLancerFenetreUS {
			continue
		}
		if atUS < originUS {
			continue // la creation precede la premiere frame publiee : rien a comparer
		}
		p, ok := pointOfXUIDAt(r.porteur, int((atUS-originUS)/step))
		if !ok {
			continue
		}
		if dd := math.Hypot(float64(x-p.X), float64(y-p.Y)); dd < it.dist {
			it.dist, it.delai = dd, objLancerDelai(atUS, r.t1US)
		}
	}
	return it
}

// objLancerDelai rend l'ecart entre la naissance et la fin du portage, ou zero quand le portage
// couvre encore l'instant (lacher volontaire, que le film ne date pas).
func objLancerDelai(atUS, finUS uint64) int64 {
	if atUS <= finUS {
		return 0
	}
	return int64(atUS - finUS)
}

// objLancerTableau publie le balayage : pour chaque rayon, la part expliquee et son temoin.
//
// LES DEUX COLONNES SE LISENT ENSEMBLE, JAMAIS SEPAREMENT. Un rayon n'est retenu que si sa part
// atteint 90 % ET que son temoin reste sous 20 % : une part qui monte en trainant le temoin avec
// elle mesure la proximite des joueurs, pas la portee d'un lancer.
func objLancerTableau(t *testing.T, titre string, vies, temoins []objLancerItem) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "REGLE ELARGIE — %s : %d vies libres, %d creations d'armes (temoin)\n",
		titre, len(vies), len(temoins))
	fmt.Fprintf(&b, "  %-8s %-22s %-22s %s\n", "R (m)", "vies expliquees", "temoin armes", "verdict")
	for _, r := range objLancerRayons {
		vOK, tOK := objLancerCompte(vies, r), objLancerCompte(temoins, r)
		pv, pt := objPart(vOK, len(vies)), objPart(tOK, len(temoins))
		fmt.Fprintf(&b, "  %-8.1f %5d/%-5d = %5.1f %%   %5d/%-5d = %5.1f %%   %s\n",
			r, vOK, len(vies), 100*pv, tOK, len(temoins), 100*pt,
			objTenu(pv >= objDrapeauSeuilVies && pt <= objDrapeauSeuilTemoin))
	}
	t.Log(b.String())
}

// objLancerCompte rend le nombre d'items expliques au rayon R.
func objLancerCompte(items []objLancerItem, r float64) int {
	n := 0
	for _, it := range items {
		n += objBool(it.explique(r))
	}
	return n
}

// objLancerDistribution decrit les RESIDUELLES du controle 3 — les vies que le rayon d'origine
// (1,5 m) n'explique pas — par leur distance au porteur et leur delai.
//
// C'EST LA MESURE QUI DIT SI L'HYPOTHESE DU LANCER TIENT, AVANT MEME LE TABLEAU. Un lancer a une
// portee bornee : si les residuelles se pressent entre 2 et 6 metres, l'hypothese est bonne et le
// rayon retenu se lit dans la distribution. Si elles s'etalent jusqu'aux confins de la carte, ce
// ne sont pas des lancers, et aucun R ne les rachetera honnetement.
//
// LES SANS-REFERENCE SONT COMPTEES A PART, ET C'EST LE POINT DE PROBITE DE CE RAPPORT : une
// naissance qu'AUCUN portage ne couvre n'a pas de distance — l'agreger aux autres avec une
// distance infinie fausserait la mediane, et la passer sous silence ferait croire que tout le
// residu est une question de rayon.
func objLancerDistribution(t *testing.T, titre string, vies []objLancerItem) {
	t.Helper()
	var dists []float64
	var delais []float64
	sans, socles := 0, 0
	for _, it := range vies {
		if it.explique(originDropMaxDist) {
			socles++
			continue
		}
		if math.IsInf(it.dist, 1) {
			sans++
			continue
		}
		dists, delais = append(dists, it.dist), append(delais, float64(it.delai)/1e6)
	}
	res := len(vies) - socles
	t.Logf("RESIDUELLES — %s : %d vies non expliquees a %.1f m, dont %d SANS AUCUNE reference "+
		"porteur dans les %.0f s (ni lancer ni lacher : le film n'en dit rien) et %d mesurables",
		titre, res, originDropMaxDist, sans, float64(objLancerFenetreUS)/1e6, len(dists))
	if len(dists) == 0 {
		return
	}
	sort.Float64s(dists)
	sort.Float64s(delais)
	t.Logf("RESIDUELLES — %s : distance au porteur (m) mediane %.1f · p90 %.1f · max %.1f ; "+
		"delai depuis la fin du portage (s) mediane %.2f · p90 %.2f · max %.2f",
		titre, objLancerQuantile(dists, 0.5), objLancerQuantile(dists, 0.9), dists[len(dists)-1],
		objLancerQuantile(delais, 0.5), objLancerQuantile(delais, 0.9), delais[len(delais)-1])
	t.Logf("RESIDUELLES — %s : par tranche de distance %s", titre, objLancerTranches(dists))
}

// objLancerQuantile rend le quantile d'une serie DEJA TRIEE, par le rang le plus proche.
func objLancerQuantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	i := int(math.Ceil(q*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

// objLancerTranches compte les residuelles par tranche de distance — la forme de la distribution
// se lit mieux la que dans trois quantiles.
func objLancerTranches(sorted []float64) string {
	bornes := []float64{3, 5, 8, 10, 20, math.Inf(1)}
	prec, parts := originDropMaxDist, make([]string, 0, len(bornes))
	for _, b := range bornes {
		n := 0
		for _, d := range sorted {
			if d > prec && d <= b {
				n++
			}
		}
		if math.IsInf(b, 1) {
			parts = append(parts, fmt.Sprintf("> %.0f m : %d", prec, n))
		} else {
			parts = append(parts, fmt.Sprintf("]%.1f-%.0f m] : %d", prec, b, n))
		}
		prec = b
	}
	return strings.Join(parts, " · ")
}
