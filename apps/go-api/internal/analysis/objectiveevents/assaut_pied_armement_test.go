package objectiveevents

// assaut_pied_armement_test.go — LE PIED DE FILM D'ASSAUT : RECENSEMENT DES INDICES, puis la
// FENETRE DE MECHE, avec son PLANCHER.
//
// # CE QUI EST DEJA ETABLI, ET QU'ON NE REFAIT PAS
//
//	- `assaut_pied_ancre_test.go` : le pied d'Assaut n'est PAS vide ; l'octet 47, que la
//	  production lit comme « indice de type », est le PETIT OCTET de la valeur de recompense
//	  (u32 grand-boutien aux octets 44-47). Les « indices » d'Assaut sont donc des VALEURS.
//	- `assaut_pied_classement_test.go` : negatif net sur la POSE cherchee comme recompense a
//	  delai constant AU MEME ACTEUR (dispersions 54 a 60 %).
//
// # CE QUE CET INSTRUMENT AJOUTE, ET POURQUOI IL FALLAIT L'AJOUTER
//
// Le negatif du classement conditionnait la recherche a l'ACTEUR de la recompense la plus
// proche de l'explosion — un acteur dont le meme test a REFUTE qu'il soit le poseur (c'est
// l'auteur du dernier frag). Si le poseur est un autre joueur, sa recompense de pose n'a jamais
// ete regardee. Le trou est reel, et il se ferme en retirant le filtre d'acteur.
//
// Second manque : aucune sonde du pied n'a jamais interroge l'instant d'ARMEMENT. Toutes ont
// mesure le delai au plus proche bloc PRECEDANT une explosion dans une fenetre de 120 s — une
// mesure que les recompenses de combat saturent. L'oracle d'armement existe desormais : la
// MECHE de 4 930 ms des variantes A MECHE FIXE, mesuree par l'anneau `ti=12 i14` et gardee en
// production comme meche de REFERENCE (`replay.BombFuseMS`, gate `TestAssautArmementGate`,
// critere (b) a +-600 ms). Depuis le 2026-09-04 la production MESURE la meche de chaque film ;
// la tolerance de 600 ms ci-dessous reste celle de CE protocole, qui ne juge que la meche fixe.
//
// # LE PROTOCOLE, FIGE AVANT LA PREMIERE MESURE
//
// ORACLE. L'instant d'armement d'une explosion des variantes A MECHE FIXE (Neutral Bomb, Husky
// Raid) vaut `explosion - 4930 ms`, a +-600 ms. Les trois films One Bomb sont EXCLUS du critere :
// leur meche (16,2 s) est PAUSABLE, donc l'instant d'armement n'y est pas une soustraction. Ils
// sont recenses, jamais juges. Denominateur du critere : 17 explosions sur 6 films.
//
// CRITERE DE RETENUE d'une valeur de recompense, les DEUX exiges :
//
//	COUVERTURE    au moins un bloc de cette valeur dans la fenetre de meche de CHACUNE des
//	              17 explosions a meche fixe ;
//	SELECTIVITE   le taux de presence en fenetre de meche doit valoir au moins TROIS FOIS le
//	              PLANCHER de la meme valeur — la part d'instants QUELCONQUES du film qui ont
//	              eux aussi un bloc de cette valeur a +-600 ms.
//
// LE PLANCHER SE MESURE AVANT D'INTERPRETER LE MOINDRE TAUX. C'est le piege n. 1 du chantier :
// une famille qui emet mille blocs sur un film de dix minutes couvre n'importe quelle fenetre de
// 1,2 s par pure densite. Le plancher est tire de 500 instants ALEATOIRES par film (graine fixe,
// donc la mesure est reproductible), dans l'intervalle borne par le premier et le dernier bloc
// du meme pied.
//
// CE QUE L'INSTRUMENT NE PEUT PAS FAIRE, dit avant de lire son verdict : il cherche une
// RECOMPENSE DE SCORE. Si l'armement d'Assaut ne rapporte aucun point (mode SCRIPTE, cf.
// `primitive_carriable_arming_base`), il n'y a rien a trouver dans ce flux et le negatif se lit
// « la pose n'est pas recompensee », jamais « la pose n'est pas dans le film ».
//
// REGIME : garde `ASSAUT_CACHE`, lecture du seul cache film. Aucune base, aucun reseau, aucun
// decodage de paquet FRAME (le pied est un chunk a part) — sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedArmement -v -timeout 30m

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

const (
	// apMecheMS : la meche des variantes a meche fixe, recopiee de `replay.BombFuseMS`. Le
	// paquet `replay` importe celui-ci ; la constante est donc recopiee, pas importee.
	apMecheMS = 4930
	// apToleranceMS : la demi-fenetre de CE protocole, celle sous laquelle la meche fixe a ete
	// mesuree (ecart-type ~80 ms). La production, elle, ne suppose plus de meche : elle la
	// mesure par film (2026-09-04).
	apToleranceMS = 600
	// apTiragesPlancher : le nombre d'instants aleatoires par film qui donnent le plancher.
	apTiragesPlancher = 500
	// apGrainePlancher fige le tirage : la mesure doit se rejouer a l'identique.
	apGrainePlancher = 20260904
	// apFacteurSelectivite : de combien le taux en fenetre de meche doit depasser le plancher.
	apFacteurSelectivite = 3.0
)

// apVariante dit, pour chaque film du corpus, si sa meche est FIXE (donc si l'oracle
// `explosion - meche` s'y applique). Source : `.ai/V7.5/PLAN_ASSAUT_LOT_A_2026-08-27.md` §0.
var apVariante = map[string]struct {
	nom       string
	mecheFixe bool
}{
	"35b75a31": {"Neutral Bomb", true},
	"ce083875": {"Neutral Bomb", true},
	"69b16f5d": {"Neutral Bomb", true},
	"3d58eb37": {"Neutral Bomb", true},
	"34bb3bc8": {"Neutral Bomb", true},
	"1c01e34f": {"Husky Raid", true},
	"df8fcbef": {"One Bomb", false},
	"c75f33b8": {"One Bomb", false},
	"9f57c612": {"One Bomb", false},
}

// apFilms rend les neuf films dans un ordre stable.
func apFilms() []string {
	out := make([]string, 0, len(afExplosions))
	for id := range afExplosions {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// apBloc est un bloc du pied avec ce qu'il PORTE : instant, valeur de recompense (u32 BE aux
// octets 44-47), petit octet de cette valeur (ce que la production appelle `th`), gamertag en
// clair et slot d'acteur.
type apBloc struct {
	t      int
	valeur int
	th     int
	slot   int
	tag    string
}

// apBlocs rend les blocs dedoublonnes d'un pied, enrichis. La geometrie est celle de la
// production (marqueur de fin `[00 00 2e e0]`, bloc de 60 octets en arriere), reprise de
// `paBlocsOctets` — la seule difference est ce qu'on en LIT.
func apBlocs(data []byte) []apBloc {
	octets := paBlocsOctets(data)
	out := make([]apBloc, 0, len(octets))
	for _, b := range octets {
		out = append(out, apBloc{
			t:      b.t,
			valeur: paValeur(b.oct[:]),
			th:     int(b.oct[47]),
			slot:   int(b.oct[36]),
			tag:    paTag(b.oct[:]),
		})
	}
	return out
}

// TestAssautPiedArmementRecensement — T3 : QUELS INDICES DE TYPE VIVENT VRAIMENT DANS LE PIED
// DES NEUF FILMS, ET CE QU'ILS PORTENT.
//
// Le recensement se fait sur des blocs DEDOUBLONNES PAR MARQUEUR DE FIN (un bloc = un compte),
// pas par candidat XUID — le premier releve comptait par candidat et gonflait d'un facteur
// variable. Les blocs de `th=10` — le SEUL indice que la production sait lire, et celui dont le
// releve precedent disait qu'il est « quasi absent » — sont imprimes UN A UN, en entier.
func TestAssautPiedArmementRecensement(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedArmementRecensement", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — recensement interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	global := map[int]int{}
	globalTh := map[int]int{}
	blocsTotal := 0
	for _, id := range apFilms() {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			t.Errorf("%s : film absent du cache (%s)", id, cache)
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			t.Errorf("%s : AUCUN PIED lisible", id)
			continue
		}
		blocs := apBlocs(footer)
		blocsTotal += len(blocs)
		parVal := map[int]int{}
		parTh := map[int]int{}
		for _, b := range blocs {
			parVal[b.valeur]++
			parTh[b.th]++
			global[b.valeur]++
			globalTh[b.th]++
		}
		v := apVariante[id]
		t.Logf("%-9s %-13s pied %7d o, %5d bloc(s) ; VALEURS %s ; octet 47 %s",
			id, v.nom, len(footer), len(blocs), apHisto(parVal), apHisto(parTh))
		// Les blocs de th=10, en entier : c'est le seul indice que la production sait lire.
		for _, b := range blocs {
			if b.th != 10 {
				continue
			}
			t.Logf("           th=10  t=%7d  valeur +%-4d  slot %3d  tag %-20s  %s",
				b.t, b.valeur, b.slot, b.tag, apContexte(b.t, afExplosions[id], v.mecheFixe))
		}
	}
	t.Logf("########## TOTAL 9 films : %d bloc(s) dedoublonnes", blocsTotal)
	t.Logf("  VALEURS de recompense (enumeration complete) : %s", apHisto(global))
	t.Logf("  OCTET 47 (« indice de type » de la production) : %s", apHisto(globalTh))
}

// TestAssautPiedArmementFenetre — T3 : LA FENETRE DE MECHE, AVEC SON PLANCHER.
//
// Pour chaque valeur de recompense : couverture des 17 instants d'armement des films a meche
// fixe, plancher tire de 500 instants aleatoires par film, et selectivite = le rapport des deux.
// Les criteres sont ceux de l'en-tete, ecrits avant le premier run.
func TestAssautPiedArmementFenetre(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedArmementFenetre", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	// couverts[valeur] = armements de l'oracle ayant au moins un bloc de cette valeur en fenetre.
	couverts := map[int]int{}
	// tiragesTouches[valeur] = instants ALEATOIRES ayant au moins un bloc de cette valeur en
	// fenetre ; tiragesTotal = leur denominateur commun.
	tiragesTouches := map[int]int{}
	tiragesTotal := 0
	armements := 0
	// tags[valeur] = les gamertags rencontres en fenetre de meche, pour lire QUI si une valeur
	// tient les criteres.
	tags := map[int]map[string]int{}
	rng := rand.New(rand.NewSource(apGrainePlancher)) //nolint:gosec // tirage de mesure, pas de crypto

	for _, id := range apFilms() {
		v := apVariante[id]
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		blocs := apBlocs(footer)
		if len(blocs) == 0 {
			t.Logf("%-9s %-13s AUCUN BLOC — film ecarte des deux passes", id, v.nom)
			continue
		}
		// PASSE 1 — le plancher, sur des instants quelconques du meme pied. Il se mesure sur
		// TOUS les films, y compris One Bomb : c'est une propriete de densite du flux.
		tmin, tmax := blocs[0].t, blocs[len(blocs)-1].t
		for i := 0; i < apTiragesPlancher; i++ {
			if tmax <= tmin {
				break
			}
			instant := tmin + rng.Intn(tmax-tmin)
			tiragesTotal++
			for val := range apValeursEnFenetre(blocs, instant, nil) {
				tiragesTouches[val]++
			}
		}
		// PASSE 2 — la fenetre de meche, sur les seuls films a meche fixe.
		if !v.mecheFixe {
			t.Logf("%-9s %-13s %5d bloc(s) — EXCLU du critere (meche pausable), plancher seul",
				id, v.nom, len(blocs))
			continue
		}
		t.Logf("%-9s %-13s %5d bloc(s), %d explosion(s) jugee(s)", id, v.nom, len(blocs), len(afExplosions[id]))
		for _, det := range afExplosions[id] {
			arm := det - apMecheMS
			armements++
			vus := map[int][]string{}
			for val, tg := range apValeursEnFenetre(blocs, arm, vus) {
				couverts[val]++
				if tags[val] == nil {
					tags[val] = map[string]int{}
				}
				for _, x := range tg {
					tags[val][x]++
				}
			}
			t.Logf("    explosion %7d ms -> armement %7d ms +-%d : %s",
				det, arm, apToleranceMS, apDetail(vus))
		}
	}

	t.Logf("########## VERDICT — %d armement(s) de l'oracle, plancher sur %d instant(s) aleatoire(s)",
		armements, tiragesTotal)
	if armements == 0 || tiragesTotal == 0 {
		t.Fatalf("mesure vide (armements=%d, tirages=%d) — l'instrument n'a rien lu",
			armements, tiragesTotal)
	}
	vals := make([]int, 0, len(couverts)+len(tiragesTouches))
	vu := map[int]bool{}
	for val := range couverts {
		if !vu[val] {
			vals, vu[val] = append(vals, val), true
		}
	}
	for val := range tiragesTouches {
		if !vu[val] {
			vals, vu[val] = append(vals, val), true
		}
	}
	sort.Ints(vals)
	retenus := 0
	for _, val := range vals {
		tauxFenetre := float64(couverts[val]) / float64(armements)
		plancher := float64(tiragesTouches[val]) / float64(tiragesTotal)
		selectivite := 0.0
		if plancher > 0 {
			selectivite = tauxFenetre / plancher
		}
		verdict := ""
		if couverts[val] == armements && (plancher == 0 || selectivite >= apFacteurSelectivite) {
			verdict = "   *** TIENT COUVERTURE ET SELECTIVITE ***"
			retenus++
		}
		t.Logf("  valeur +%-4d : fenetre %2d/%d (%.0f %%), plancher %.0f %%, selectivite %.2fx%s",
			val, couverts[val], armements, 100*tauxFenetre, 100*plancher, selectivite, verdict)
		if verdict != "" {
			t.Logf("      acteurs en fenetre : %s", apTags(tags[val]))
		}
	}
	if retenus == 0 {
		t.Logf("AUCUNE VALEUR NE TIENT LES DEUX CRITERES : le pied de film ne porte pas " +
			"l'armement d'Assaut, ni sous forme de recompense, ni sous une autre. Le negatif " +
			"est BORNE par le plancher ci-dessus — sans lui, les taux de fenetre ne voudraient rien.")
	}
}

// apValeursEnFenetre rend les valeurs de recompense presentes a +-[apToleranceMS] d'un instant.
// Si `detail` est non nil, il recoit les gamertags par valeur.
func apValeursEnFenetre(blocs []apBloc, instant int, detail map[int][]string) map[int][]string {
	out := map[int][]string{}
	for _, b := range blocs {
		d := b.t - instant
		if d < -apToleranceMS || d > apToleranceMS {
			continue
		}
		out[b.valeur] = append(out[b.valeur], b.tag)
		if detail != nil {
			detail[b.valeur] = append(detail[b.valeur], fmt.Sprintf("%s@%+d", b.tag, d))
		}
	}
	return out
}

// apDetail rend le contenu d'une fenetre en une ligne lisible.
func apDetail(m map[int][]string) string {
	if len(m) == 0 {
		return "(fenetre VIDE)"
	}
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := ""
	for _, k := range ks {
		if out != "" {
			out += " ; "
		}
		out += fmt.Sprintf("+%d x%d [%s]", k, len(m[k]), apJoindre(m[k], 4))
	}
	return out
}

// apJoindre concatene au plus `max` elements.
func apJoindre(xs []string, max int) string {
	out := ""
	for i, x := range xs {
		if i >= max {
			out += fmt.Sprintf(", … (+%d)", len(xs)-max)
			break
		}
		if out != "" {
			out += ", "
		}
		out += x
	}
	return out
}

// apHisto rend un histogramme trie par cle.
func apHisto(m map[int]int) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := ""
	for _, k := range ks {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d:x%d", k, m[k])
	}
	if out == "" {
		return "(aucune)"
	}
	return out
}

// apTags rend les acteurs tries par frequence decroissante.
func apTags(m map[string]int) string {
	type l struct {
		tag string
		n   int
	}
	ls := make([]l, 0, len(m))
	for tg, n := range m {
		ls = append(ls, l{tg, n})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].n != ls[j].n {
			return ls[i].n > ls[j].n
		}
		return ls[i].tag < ls[j].tag
	})
	out := ""
	for _, x := range ls {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%s x%d", x.tag, x.n)
	}
	if out == "" {
		return "(aucun)"
	}
	return out
}

// apContexte situe un instant par rapport aux explosions ET aux armements deduits d'un film.
func apContexte(t int, exps []int, mecheFixe bool) string {
	if len(exps) == 0 {
		return "(aucune explosion au releve)"
	}
	meilleur, delta := 0, 1<<30
	for _, ms := range exps {
		if d := apAbs(t - ms); d < delta {
			meilleur, delta = ms, d
		}
	}
	out := fmt.Sprintf("%+d ms de l'explosion %d", t-meilleur, meilleur)
	if mecheFixe {
		out += fmt.Sprintf(" ; %+d ms de son armement", t-(meilleur-apMecheMS))
	}
	return out
}

// apAbs rend la valeur absolue d'un entier.
func apAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
