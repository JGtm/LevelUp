package replay

// attachement_phase0_ctf_test.go — ITEM 0.2 : LE DRAPEAU PORTÉ.
//
// L'ORACLE EST EXACT ET IL EXISTE DÉJÀ. `FlagGrabs` / `FlagSteals` / `FlagCaptures` sont
// répliqués par le statborg, datés à la milliseconde et nommés par slot ; l'item 4 de la
// phase 0 des objectifs les a assemblés en FENÊTRES DE PORTAGE (`objPortageWindows`) et a
// construit le pont slot de bipède -> xuid (`objBridgeOf`). Rien de tout cela n'est réécrit
// ici : l'item 0.2 pose une SEULE question nouvelle — quand un joueur prend le drapeau,
// est-ce qu'un composant i10 passe à porte ouverte avec un champ qui désigne ce joueur ?
//
// L'HYPOTHÈSE DE CHAMP N'EST PAS UN CHOIX ESTHÉTIQUE, ELLE EST STRUCTURELLE. `Quant16` est
// lu par `readQuantStat(1, 13)`, qui rend `(top2 << 30) | (base + R(13))` — exactement la
// forme que `readRecordID` donne à un IDENTIFIANT DE RECORD : 13 bits de slot en bas, 2 bits
// de génération en haut. Les valeurs observées au recensement le confirment de vue
// (0x40001B20, 0xC000040D, 0x800003C2 : des tags 1..3 sur des slots de 3 à 4 chiffres). Le
// candidat principal est donc `Quant16 & 0x3FFFFFFF`. Les deux autres hypothèses du plan
// (`Word16`, et la branche libre) sont mesurées À CÔTÉ, sur le même dénominateur — sans quoi
// « le premier essai a marché » ne se distinguerait pas de « n'importe quoi marche ».
//
// LE TÉMOIN EST OBLIGATOIRE, et il est construit pour être COMPARABLE : à chaque fenêtre de
// portage on substitue au porteur un AUTRE joueur du même match, tiré de façon déterministe.
// Même nombre de fenêtres, même corpus de lectures, même tolérance : seule l'identité change.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// attTolerancesMS — les tolérances d'appariement publiées, de la plus stricte à la plus
// large. La PREMIÈRE est celle du plan (« dans les 2 images » : deux paquets delta, dont
// l'écart médian est mesuré et publié à côté) ; les suivantes disent à quelle vitesse le
// taux monte, ce qui distingue un signal décalé d'un signal absent.
var attTolerancesMS = []int64{50, 250, 1000, 2000}

// attSeuilTaux / attSeuilTemoin — les seuils du plan (décision 4(a)), écrits avant mesure.
const (
	attSeuilTaux   = 0.90
	attSeuilTemoin = 0.05
)

// attHypothese nomme une façon de lire un slot de parent dans une lecture d'i10.
type attHypothese struct {
	Nom string
	// Slot rend le slot désigné, et false quand l'hypothèse ne s'applique pas à la lecture.
	Slot func(attI10) (uint32, bool)
}

// attHypotheses — les trois lectures candidates du plan, plus la décomposition basse de
// `Quant16` (le champ R(13) seul, sans la génération) qui est la variante la plus proche.
func attHypotheses() []attHypothese {
	return []attHypothese{
		{"quant16 & 0x3FFFFFFF (slot+gen)", func(l attI10) (uint32, bool) {
			if !l.St.Attached {
				return 0, false
			}
			return l.St.Quant16 & 0x3FFFFFFF, true
		}},
		{"quant16 & 0x1FFF (R(13) seul)", func(l attI10) (uint32, bool) {
			if !l.St.Attached {
				return 0, false
			}
			return l.St.Quant16 & 0x1FFF, true
		}},
		{"word16", func(l attI10) (uint32, bool) {
			if !l.St.Attached {
				return 0, false
			}
			return l.St.Word16, true
		}},
		{"opt16", func(l attI10) (uint32, bool) {
			if !l.St.Attached || !l.St.HasOpt16 {
				return 0, false
			}
			return l.St.Opt16, true
		}},
	}
}

// attOracle porte tout ce que l'oracle CTF d'un film fournit.
type attOracle struct {
	Bridge   objBridge
	Fenetres []objWindow
	// SlotsDe est l'inverse du pont : xuid -> TOUS ses slots de bipède.
	//
	// TOUS, ET C'EST LE POINT. Un slot de bipède vaut UNE VIE, pas un joueur : le pont en
	// nomme 92 à 122 pour huit joueurs. Ne garder qu'un slot par joueur — ce que fait
	// naturellement une table `xuid -> slot` — reviendrait à chercher le drapeau dans une
	// vie sur onze, tirée au hasard de l'ordre d'itération d'une map. Le test échouerait
	// pour une raison qui n'a rien à voir avec la question posée.
	SlotsDe map[uint64]map[uint32]bool
}

// attOracleCTF construit l'oracle d'un film CTF.
func attOracleCTF(t *testing.T, root, id string) attOracle {
	t.Helper()
	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache", id)
	}
	b := objBridgeOf(t, root, id)
	identity := objectiveevents.SlotIdentityFromDeaths(src, objDeathInstants(b.Deaths))
	evs := objectiveevents.IdentifyNamedEvents(
		objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeFlag), identity)
	wins, _ := objPortageWindows(evs, b.Deaths, objFinMatch(evs, b.Deaths))
	inv := map[uint64]map[uint32]bool{}
	for slot, x := range b.SlotXUID {
		if inv[x] == nil {
			inv[x] = map[uint32]bool{}
		}
		inv[x][slot] = true
	}
	return attOracle{Bridge: b, Fenetres: wins, SlotsDe: inv}
}

// attMatchMS convertit l'instant FILM d'une lecture en instant MATCH.
func attMatchMS(l attI10, b objBridge) int64 { return int64(l.TS/1000) - b.OffsetMS }

// attEcartPaquetsMS mesure l'écart médian entre deux paquets delta consécutifs — c'est la
// durée d'une « image » au sens du plan, et elle se MESURE plutôt qu'elle ne se suppose.
func attEcartPaquetsMS(lectures []attI10) int64 {
	var ts []int64
	vus := map[uint64]bool{}
	for _, l := range lectures {
		if vus[l.TS] {
			continue
		}
		vus[l.TS] = true
		ts = append(ts, int64(l.TS/1000))
	}
	if len(ts) < 2 {
		return 0
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
	var ecarts []int64
	for i := 1; i < len(ts); i++ {
		ecarts = append(ecarts, ts[i]-ts[i-1])
	}
	sort.Slice(ecarts, func(i, j int) bool { return ecarts[i] < ecarts[j] })
	return ecarts[len(ecarts)/2]
}

// attResultat porte le verdict d'une hypothèse sur un corpus.
type attResultat struct {
	Nom string
	// Appariees / Fenetres : fenêtres de portage dont la prise est suivie d'une lecture
	// attachée désignant le porteur, à la tolérance donnée.
	Appariees, Fenetres int
	// Temoin : le même compte avec un AUTRE joueur du match à la place du porteur.
	Temoin int
	// Delais : l'écart signé prise -> lecture, pour les fenêtres appariées.
	Delais []int64
	// Applicables : lectures auxquelles l'hypothèse s'applique (dénominateur du champ).
	Applicables int
	// VersBipede : parmi elles, celles dont le slot désigné est un slot de bipède NOMMÉ.
	VersBipede int
}

// attEvalue confronte une hypothèse à l'oracle d'un film, à une tolérance donnée.
//
// LA RECHERCHE EST BORNÉE DES DEUX CÔTÉS de la prise, et c'est délibéré : rien ne garantit
// que le film réplique l'attachement APRÈS l'événement de statistique plutôt qu'avant (les
// deux voyagent par des canaux différents, et le calage d'horloge du pont a lui-même une
// précision de l'ordre de 150 ms). Le délai signé est publié pour que le sens se lise.
func attEvalue(h attHypothese, o attOracle, lectures []attI10, tolMS int64, propres bool) attResultat {
	r := attResultat{Nom: h.Nom, Fenetres: len(o.Fenetres)}
	if propres {
		r.Nom += " [bit-exactes]"
	}
	// Index : slot désigné -> instants MATCH des lectures attachées qui le désignent.
	parSlot := map[uint32][]int64{}
	for _, l := range lectures {
		if propres && !l.Propre {
			continue
		}
		slot, ok := h.Slot(l)
		if !ok {
			continue
		}
		r.Applicables++
		if _, nomme := o.Bridge.SlotXUID[slot]; nomme {
			r.VersBipede++
		}
		parSlot[slot] = append(parSlot[slot], attMatchMS(l, o.Bridge))
	}
	for s := range parSlot {
		sort.Slice(parSlot[s], func(i, j int) bool { return parSlot[s][i] < parSlot[s][j] })
	}
	autres := attAutresJoueurs(o)
	for i, w := range o.Fenetres {
		if d, found := attPlusProcheParmi(parSlot, o.SlotsDe[w.XUID], w.T0, tolMS); found {
			r.Appariees++
			r.Delais = append(r.Delais, d)
		}
		if len(autres) == 0 {
			continue
		}
		// TÉMOIN DÉTERMINISTE : le i-ème autre joueur du match, jamais le porteur.
		t := autres[i%len(autres)]
		if t == w.XUID {
			t = autres[(i+1)%len(autres)]
		}
		if t == w.XUID {
			continue
		}
		if _, found := attPlusProcheParmi(parSlot, o.SlotsDe[t], w.T0, tolMS); found {
			r.Temoin++
		}
	}
	return r
}

// attPlusProcheParmi cherche la lecture la plus proche de t0 parmi TOUTES les vies d'un
// joueur — un slot de bipède est une vie, un joueur en a une dizaine par match.
func attPlusProcheParmi(parSlot map[uint32][]int64, slots map[uint32]bool,
	t0, tolMS int64) (int64, bool) {
	best, found := int64(0), false
	for s := range slots {
		d, ok := attPlusProche(parSlot[s], t0, tolMS)
		if !ok {
			continue
		}
		if !found || attAbs(d) < attAbs(best) {
			best, found = d, true
		}
	}
	return best, found
}

// attAutresJoueurs rend les xuid du match, dans un ordre stable.
func attAutresJoueurs(o attOracle) []uint64 {
	out := make([]uint64, 0, len(o.SlotsDe))
	for x := range o.SlotsDe {
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// attPlusProche rend le délai signé (lecture - prise) le plus petit en valeur absolue dans
// la tolérance, et s'il existe.
func attPlusProche(instants []int64, t0, tolMS int64) (int64, bool) {
	best, found := int64(0), false
	for _, at := range instants {
		d := at - t0
		if d < -tolMS || d > tolMS {
			continue
		}
		if !found || attAbs(d) < attAbs(best) {
			best, found = d, true
		}
	}
	return best, found
}

func attAbs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// TestAttachementPhase0CTF — ITEM 0.2 : les quatre hypothèses de champ contre l'oracle CTF.
func TestAttachementPhase0CTF(t *testing.T) {
	root := attRequireRoot(t)
	joues := 0
	cumul := map[string]map[int64]*attResultat{}
	for _, id := range objCTFFilms {
		if _, ok := objOpenFilm(t, root, id); !ok {
			t.Logf("%s : film absent du cache — sauté", id)
			continue
		}
		joues++
		o := attOracleCTF(t, root, id)
		lectures, st := attScanOf(t, root, id)
		attachees := 0
		for _, l := range lectures {
			if l.St.Attached {
				attachees++
			}
		}
		t.Logf("%s : %d fenêtres de portage · %d slots de bipède nommés · calage %d ms · "+
			"%d lectures d'i10 (%d attachées) · %d orphelines · écart médian entre paquets %d ms",
			id, len(o.Fenetres), len(o.Bridge.SlotXUID), o.Bridge.OffsetMS, len(lectures),
			attachees, st.Orphelines, attEcartPaquetsMS(lectures))
		// QUALITÉ DU PONT, publiée avec le résultat : un verdict négatif ne vaut que si la
		// chaîne qui nomme les slots tenait. Sans ces quatre nombres, « 0 % » pourrait aussi
		// bien dire « le pont n'a nommé personne ».
		t.Logf("%s : qualité du pont bipède — %d vies, %d morts nommées, %d coïncidences de "+
			"calage, %d collisions ; %d joueurs distincts portés par le pont",
			id, o.Bridge.LivesTotal, o.Bridge.DeathsNamed, o.Bridge.OffsetMatches,
			o.Bridge.Collisions, len(o.SlotsDe))
		for _, h := range attHypotheses() {
			for _, propres := range []bool{false, true} {
				for _, tol := range attTolerancesMS {
					r := attEvalue(h, o, lectures, tol, propres)
					attCumule(cumul, r.Nom, tol, r)
					t.Logf("%s :   %-46s tol %4d ms — %d/%d appariées (%.1f %%), témoin %d (%.1f %%) ; "+
						"champ applicable %d fois, dont %d vers un bipède nommé",
						id, r.Nom, tol, r.Appariees, r.Fenetres, 100*attPart(r.Appariees, r.Fenetres),
						r.Temoin, 100*attPart(r.Temoin, r.Fenetres), r.Applicables, r.VersBipede)
				}
			}
		}
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", attFilmEnv, root)
	}
	attVerdict(t, "ITEM 0.2 (CTF)", cumul)
}

// attCumule accumule un résultat de film dans le cumul du corpus.
func attCumule(cumul map[string]map[int64]*attResultat, nom string, tol int64, r attResultat) {
	if cumul[nom] == nil {
		cumul[nom] = map[int64]*attResultat{}
	}
	c := cumul[nom][tol]
	if c == nil {
		c = &attResultat{Nom: nom}
		cumul[nom][tol] = c
	}
	c.Appariees += r.Appariees
	c.Fenetres += r.Fenetres
	c.Temoin += r.Temoin
	c.Applicables += r.Applicables
	c.VersBipede += r.VersBipede
	c.Delais = append(c.Delais, r.Delais...)
}

// attVerdict publie le cumul du corpus et tranche sur les seuils du plan.
func attVerdict(t *testing.T, titre string, cumul map[string]map[int64]*attResultat) {
	t.Helper()
	noms := make([]string, 0, len(cumul))
	for n := range cumul {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	tenu := ""
	for _, n := range noms {
		tols := make([]int64, 0, len(cumul[n]))
		for tol := range cumul[n] {
			tols = append(tols, tol)
		}
		sort.Slice(tols, func(i, j int) bool { return tols[i] < tols[j] })
		for _, tol := range tols {
			c := cumul[n][tol]
			taux, tem := attPart(c.Appariees, c.Fenetres), attPart(c.Temoin, c.Fenetres)
			t.Logf("%s CUMUL — %-32s tol %4d ms : %d/%d = %.1f %% (seuil %.0f %%), "+
				"témoin %d/%d = %.1f %% (seuil %.0f %%) ; délai médian %d ms",
				titre, n, tol, c.Appariees, c.Fenetres, 100*taux, 100*attSeuilTaux,
				c.Temoin, c.Fenetres, 100*tem, 100*attSeuilTemoin, attMedianeI64(c.Delais))
			if taux >= attSeuilTaux && tem <= attSeuilTemoin && tenu == "" {
				tenu = fmt.Sprintf("%s à %d ms", n, tol)
			}
		}
	}
	if tenu == "" {
		t.Logf("%s VERDICT : aucune hypothèse ne tient les deux seuils — NÉGATIF", titre)
		return
	}
	t.Logf("%s VERDICT : hypothèse TENUE — %s", titre, tenu)
}

// TestAttachementPhase0PorteursGate — ITEM 0.2 : la PORTE d'i10 CHEZ LE PORTEUR.
//
// LA QUESTION EST L'AUTRE MOITIE DU MODELE PARENT-ENFANT, et elle se pose separement du
// handle. Si le lien etait porte par le bipede plutot que par l'objet — hypothese que le plan
// laisse ouverte (« ou du porteur : sens a etablir ») — alors la PORTE d'i10 d'un joueur qui
// tient le drapeau devrait etre ouverte pendant qu'il le tient, et fermee sinon. Cette
// mesure-la ne suppose RIEN du sens des champs : elle ne lit qu'un bit, et le confronte a la
// frontiere de portage de l'oracle.
//
// LES DEUX TAUX SE PUBLIENT ENSEMBLE. Un taux d'ouverture pendant le portage ne prouve rien
// tant qu'on ne sait pas ce qu'il vaut en dehors : la porte d'i10 est ouverte dans un tiers
// des lectures TOUS archetypes confondus, donc c'est l'ECART entre les deux qui est le signal.
func TestAttachementPhase0PorteursGate(t *testing.T) {
	root := attRequireRoot(t)
	joues := 0
	var cumDedans, cumOuvDedans, cumDehors, cumOuvDehors int
	for _, id := range objCTFFilms {
		if _, ok := objOpenFilm(t, root, id); !ok {
			continue
		}
		joues++
		o := attOracleCTF(t, root, id)
		lectures, _ := attScanOf(t, root, id)
		// QUALITE DU PONT, publiee avec le resultat : un verdict negatif ne vaut que si la
		// chaine qui nomme les slots tenait. Sans ces nombres, « 0 % » pourrait aussi bien
		// vouloir dire « le pont n'a nomme personne ».
		t.Logf("%s : qualite du pont bipede — %d vies, %d morts nommees, %d coincidences de "+
			"calage, %d collisions ; %d joueurs distincts, %d slots nommes, %d fenetres",
			id, o.Bridge.LivesTotal, o.Bridge.DeathsNamed, o.Bridge.OffsetMatches,
			o.Bridge.Collisions, len(o.SlotsDe), len(o.Bridge.SlotXUID), len(o.Fenetres))
		dedans, ouvDedans, dehors, ouvDehors := attPorteursGate(o, lectures)
		cumDedans, cumOuvDedans = cumDedans+dedans, cumOuvDedans+ouvDedans
		cumDehors, cumOuvDehors = cumDehors+dehors, cumOuvDehors+ouvDehors
		t.Logf("%s : i10 des BIPEDES nommes — pendant un portage %d lectures dont %d a porte "+
			"ouverte (%.1f %%) ; hors portage %d lectures dont %d ouvertes (%.1f %%)",
			id, dedans, ouvDedans, 100*attPart(ouvDedans, dedans),
			dehors, ouvDehors, 100*attPart(ouvDehors, dehors))
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", attFilmEnv, root)
	}
	t.Logf("ITEM 0.2 CUMUL porte du PORTEUR — portage %d/%d = %.1f %% ouvertes ; hors portage "+
		"%d/%d = %.1f %% ; ecart %.1f points",
		cumOuvDedans, cumDedans, 100*attPart(cumOuvDedans, cumDedans),
		cumOuvDehors, cumDehors, 100*attPart(cumOuvDehors, cumDehors),
		100*(attPart(cumOuvDedans, cumDedans)-attPart(cumOuvDehors, cumDehors)))
}

// attPorteursGate compte, sur les lectures d'i10 des bipedes que le pont NOMME, celles qui
// tombent pendant une fenetre de portage de LEUR joueur et celles qui tombent en dehors, avec
// dans chaque camp le nombre de portes ouvertes.
func attPorteursGate(o attOracle, lectures []attI10) (dedans, ouvDedans, dehors, ouvDehors int) {
	parXUID := map[uint64][]objWindow{}
	for _, w := range o.Fenetres {
		parXUID[w.XUID] = append(parXUID[w.XUID], w)
	}
	for _, l := range lectures {
		if l.TI != uint32(filmdec.BipedTypeIndex) {
			continue
		}
		x, nomme := o.Bridge.SlotXUID[l.Slot]
		if !nomme {
			continue
		}
		if objDansFenetre(parXUID[x], attMatchMS(l, o.Bridge)) {
			dedans++
			if l.St.Attached {
				ouvDedans++
			}
			continue
		}
		dehors++
		if l.St.Attached {
			ouvDehors++
		}
	}
	return dedans, ouvDedans, dehors, ouvDehors
}

// attMedianeI64 rend la mediane d'une serie (0 si vide). Copie locale de l'ancien
// `objMedianeI64`, supprime avec l'instrument du canal delta (item 4 phase 1.0b) : cet
// instrument est le seul lecteur restant, il porte donc le helper.
func attMedianeI64(v []int64) int64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]int64{}, v...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)/2]
}
