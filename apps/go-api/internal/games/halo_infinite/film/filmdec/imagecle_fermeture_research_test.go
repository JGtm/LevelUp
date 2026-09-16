package filmdec

// imagecle_fermeture_research_test.go — PHASE 5a, OBJECTIFS 1 ET 2 : CE QUE LA BONNE FORME
// DU RECORD D'IMAGE-CLE CHANGE, ARCHETYPE PAR ARCHETYPE.
//
// # CE QUE LA PHASE 4 A ETABLI, ET QUI EST LE POINT DE DEPART
//
// Le record d'image-cle est lu par le lecteur d'ETAT COMPLET `FUN_142e2bfd0`, PAS par le
// lecteur de record NEW. Sa forme :
//
//	108 bits d'en-tete PAR ENTITE   R(32) id · R(32) typeIndex · R(32) · R(4) · R(8)
//	 32 bits  n1                    taille de tampon ; si > 0 l'etat par defaut suit
//	  w bits  etat par defaut       vtable[0x60] de l'archetype
//	 32 bits  n2                    taille de tampon ; si > 0 la boucle de composants suit
//	         les composants          DANS L'ORDRE DU REGISTRE, SANS AUCUN MASQUE DE PRESENCE
//
// soit `debut des composants(ti) = 172 + largeurEtatParDefaut(ti)` — 186 pour ti=9, verifie
// sur 2 424 records sur 2 424 et derive a l'identique sur cinq builds
// (`NOTE_PROFIL_PAR_BUILD_2026-09-12.md`, sections A.1 a A.6).
//
// Le modele de la PRODUCTION (`keyframe_record_walk.go` : en-tete de 64 bits, etat par
// defaut, porte R(1), MASQUE de presence, composants) est REFUTE pour cette table : sa borne
// arithmetique est `64 + 22 + 65 = 151`, soit 35 bits trop court pour 186.
//
// # CE QUE CE FICHIER MESURE, ET QU'AUCUN INSTRUMENT DU DEPOT NE MESURAIT
//
// La phase 4 a prouve ou commence le PREMIER composant. Elle n'a jamais mesure si le record
// SE FERME — c'est-a-dire si, une fois les 64 composants deroules, la marche atterrit
// exactement sur le debut du record suivant. Et elle ne l'a jamais mesure PAR ARCHETYPE.
// R7-e l'avait mesure, mais sur le seul `ti=35` (bipede), toutes options confondues, et son
// verdict (0,51 % d'atterrissage) porte sur l'archetype dont l'etat par defaut n'est PAS
// bit-exact et dont la moitie des composants n'est pas portee.
//
// CE FICHIER REND LE TABLEAU `archetype x modele` : records fermes / total, sur les memes
// records, sous les deux lectures, a COUVERTURE DE COMPOSANTS CONSTANTE (aucun bouchon, aucune
// largeur de carte installee, bascules du process aux valeurs par defaut). C'est le chiffre
// que le pilote attend pour decider d'un lot de production : ce que le correctif GAGNE.
//
// # LES CONTROLES, ECRITS AVANT LA MESURE
//
//	T+  TEMOIN POSITIF   ti=9 doit fermer massivement sous l'etat complet : c'est l'archetype
//	                     dont la phase 4 a prouve la derivation (186) et dont `n1` et `n2` sont
//	                     constants sur 2 424 records.
//	T-  TEMOIN NEGATIF   le MEME comptage sous le modele de la production doit etre PIRE la ou
//	                     la phase 4 l'a dit — a commencer par ti=9, ou le modele place le
//	                     premier composant a 82 et lit un masque qui dit « aucun composant ».
//	T0  DENOMINATEUR     tout est publie sur le meme denominateur : les records BORNES, ceux
//	                     dont le balayeur rend un voisin suivant. Un record non borne n'a pas
//	                     de frontiere a atteindre et ne compte nulle part.
//
// # CE QUE « FERMER » VEUT DIRE, EXACTEMENT
//
// La frontiere visee est l'ancre du record suivant rendue par `WalkKeyframeWorld`. Sous le
// modele d'ETAT COMPLET le filtre de ce balayeur n'est PAS arbitraire : il exige que le mot
// de 32 bits a `+32` vaille moins de 50, ce qui est EXACTEMENT le `typeIndex` teste par
// `FUN_142e2bfd0`. Le balayeur est donc, sous cette lecture, un filtre du format lui-meme.
//
// Quatre issues, exclusives :
//
//	FERME    marche complete (aucun composant non porte) et fin == frontiere visee ;
//	DESYNC   marche arretee sur un composant non porte : c'est la COUVERTURE du dispatch qui
//	         manque, pas forcement le cadre — et c'est la raison pour laquelle ce compte est
//	         publie a part et jamais fondu dans « ne ferme pas » ;
//	SOUS     marche complete, fin AVANT la frontiere ;
//	SUR      marche complete, fin APRES la frontiere.
//
// Garde CHUNK00_FILMS. Aucun code de production modifie ; le modele de production est rejoue
// PAR SON PROPRE CODE (`readKeyframeHeader` + `walkOneKeyframeRecord`).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Les deux modeles compares. Ce sont des cles de tableau, pas des reglages.
const (
	imcEtatComplet = "etat-complet"
	// imcEtatDecale est le TEMOIN DE HASARD : la MEME lecture, l'en-tete par entite decale
	// d'UN SEUL BIT. Sans lui, un taux de fermeture ne se compare a rien — la regle 4 de
	// `METHODE_RETRO_INGENIERIE_FILM.md` interdit de calculer ce plancher, il se MESURE.
	imcEtatDecale = "etat-complet+1bit"
	imcProduction = "production"
)

// imcModeles est l'ordre de publication des colonnes.
func imcModeles() []string { return []string{imcEtatComplet, imcEtatDecale, imcProduction} }

// imcCompte est le comptage d'UN archetype sous UN modele.
type imcCompte struct {
	Total, Ferme, Desync, Sous, Sur int
	// Voisin* restreint aux paires de slots CONSECUTIFS (`slotSuivant == slot + 1`), ou le
	// balayeur ne peut avoir saute aucun record entre les deux ancres.
	VoisinTotal, VoisinFerme int
	// Desyncs nomme les composants qui arretent la marche, pour que « DESYNC » ne soit pas
	// un mot mais une liste.
	Desyncs map[string]int
}

// imcTable est le tableau `modele -> archetype -> comptage`.
type imcTable map[string]map[int]*imcCompte

// imcNouvelleTable rend une table vide pour les deux modeles.
func imcNouvelleTable() imcTable {
	t := imcTable{}
	for _, m := range imcModeles() {
		t[m] = map[int]*imcCompte{}
	}
	return t
}

// cellule rend (en la creant au besoin) le comptage d'un archetype sous un modele.
func (t imcTable) cellule(modele string, ti int) *imcCompte {
	c := t[modele][ti]
	if c == nil {
		c = &imcCompte{Desyncs: map[string]int{}}
		t[modele][ti] = c
	}
	return c
}

// imcFilm porte ce qu'un film offre a la mesure.
type imcFilm struct {
	Nom, Build string
	Reg        *Registry
	Pays       [][]byte
}

// imcCharger lit le registre et TOUS les payloads d'image-cle d'un film, plus sa chaine de
// build (lue dans l'en-tete de `chunk_00`, le meme champ que la table de profil de la
// phase 4).
func imcCharger(t *testing.T, dir string) (imcFilm, bool) {
	t.Helper()
	f := imcFilm{Nom: filepath.Base(dir)}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Logf("%s : ECARTE (aucun chunk)", f.Nom)
		return f, false
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Logf("%s : ECARTE (chunk_00 illisible : %v)", f.Nom, err)
		return f, false
	}
	if f.Reg, err = ParseRegistryChunk(raw); err != nil {
		t.Logf("%s : ECARTE (registre illisible : %v)", f.Nom, err)
		return f, false
	}
	if _, d := readChunk00(t, dir); len(d) > 0 {
		f.Build, _ = s3bBuild(d)
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeKeyframe {
				f.Pays = append(f.Pays, pk.Payload(data))
			}
		}
	}
	return f, len(f.Pays) > 0
}

// imcBornes : LA MEME POPULATION QUE LA PRODUCTION. Depuis le lot 1.4 (2026-09-14),
// l appariement « record i, frontiere i+1 » vit dans `keyframeBornes` (keyframe_closure.go) et
// les trois lecteurs — la mesure, cet oracle et les deux balayages de production — comptent sur
// exactement les memes bornes.
var imcBornes = keyframeBornes

// imcMarcher rejoue UN record sous le modele demande et rend la position de fin et l'index
// du premier composant non porte (-1 si la marche va au bout).
//
// LA COLONNE `imcProduction` EST REJOUEE PAR LE CADRE DELTA (`readKeyframeHeader` puis
// `walkOneKeyframeRecord`) — depuis le lot 1.4 (2026-09-14) ce n'est plus ce que la production
// lit : les deux balayages de production sont passes au cadre d'etat complet. La colonne garde
// son nom de cle de tableau, mais elle mesure desormais L'ANCIENNE lecture, et c'est a ce titre
// qu'elle reste publiee : sans elle, le gain du lot 1.4 ne se compare a rien.
func imcMarcher(modele string, pay []byte, reg *Registry, bit int) (end, desync int) {
	switch modele {
	case imcProduction:
		h, ok := readKeyframeHeader(pay, bit, len(pay)*8)
		if !ok {
			return bit, 0
		}
		rec, _, _ := walkOneKeyframeRecord(pay, reg, bit, h, profilDInstrument)
		return rec.BitEnd, rec.DesyncAt
	case imcEtatDecale:
		tr := walkKeyframeFullState(pay, bit, reg, profilDInstrument,
			keyframeFullStateTemoin{EnTeteBits: keyframeFullStateHeaderBits + 1})
		return tr.EndBit, tr.DesyncAt
	}
	tr := WalkKeyframeFullState(pay, bit, reg, profilDInstrument)
	return tr.EndBit, tr.DesyncAt
}

// imcClasser range UNE marche dans le comptage de son archetype.
func imcClasser(c *imcCompte, b keyframeBorne, end, desync int, nom string) {
	c.Total++
	if b.Voisin {
		c.VoisinTotal++
	}
	if desync >= 0 {
		c.Desync++
		c.Desyncs[nom]++
		return
	}
	switch {
	case end == b.Want:
		c.Ferme++
		if b.Voisin {
			c.VoisinFerme++
		}
	case end < b.Want:
		c.Sous++
	default:
		c.Sur++
	}
}

// imcNomComposant nomme le composant qui a arrete la marche, ou l'index nu si le registre
// ne le porte pas.
func imcNomComposant(reg *Registry, ti, idx int) string {
	if arch, ok := reg.Archetype(ti); ok && idx >= 0 && idx < len(arch.Components) {
		return fmt.Sprintf("i%d %s", idx, arch.Components[idx])
	}
	return fmt.Sprintf("i%d (hors registre)", idx)
}

// imcMesurerFilm accumule, pour un film, les deux modeles sur les memes records bornes.
func imcMesurerFilm(f imcFilm, t imcTable) {
	for _, pay := range f.Pays {
		for _, b := range imcBornes(pay) {
			for _, m := range imcModeles() {
				end, desync := imcMarcher(m, pay, f.Reg, b.Bit)
				nom := ""
				if desync >= 0 {
					nom = imcNomComposant(f.Reg, b.TI, desync)
				}
				imcClasser(t.cellule(m, b.TI), b, end, desync, nom)
			}
		}
	}
}

// imcTIs rend les archetypes rencontres, tries.
func imcTIs(t imcTable) []int {
	vus := map[int]bool{}
	for _, m := range imcModeles() {
		for ti := range t[m] {
			vus[ti] = true
		}
	}
	out := make([]int, 0, len(vus))
	for ti := range vus {
		out = append(out, ti)
	}
	sort.Ints(out)
	return out
}

// imcPublier ecrit le tableau `archetype x modele`.
func imcPublier(t *testing.T, titre string, tab imcTable) {
	t.Helper()
	t.Logf("---- %s ----", titre)
	t.Logf("%-5s | %-30s | %-30s | %-30s", "ti",
		"ETAT COMPLET (fermes/total)", "TEMOIN +1 bit", "PRODUCTION")
	tot := map[string]*imcCompte{}
	for _, m := range imcModeles() {
		tot[m] = &imcCompte{Desyncs: map[string]int{}}
	}
	for _, ti := range imcTIs(tab) {
		t.Logf("ti=%-3d | %-30s | %-30s | %-30s", ti,
			imcCellule(tab[imcEtatComplet][ti]), imcCellule(tab[imcEtatDecale][ti]),
			imcCellule(tab[imcProduction][ti]))
		for _, m := range imcModeles() {
			imcAjouter(tot[m], tab[m][ti])
		}
	}
	t.Logf("%-5s | %-30s | %-30s | %-30s", "TOTAL",
		imcCellule(tot[imcEtatComplet]), imcCellule(tot[imcEtatDecale]),
		imcCellule(tot[imcProduction]))
}

// imcAjouter additionne un comptage dans un autre (les compteurs, pas les libelles).
func imcAjouter(d, c *imcCompte) {
	if c == nil {
		return
	}
	d.Total += c.Total
	d.Ferme += c.Ferme
	d.Desync += c.Desync
	d.Sous += c.Sous
	d.Sur += c.Sur
	d.VoisinTotal += c.VoisinTotal
	d.VoisinFerme += c.VoisinFerme
}

// imcCellule rend une cellule du tableau : fermes/total, part, et la ventilation du reste.
func imcCellule(c *imcCompte) string {
	if c == nil || c.Total == 0 {
		return "-"
	}
	return fmt.Sprintf("%5d/%-5d %5.1f%% d%d s%d u%d",
		c.Ferme, c.Total, 100*float64(c.Ferme)/float64(c.Total), c.Desync, c.Sous, c.Sur)
}

// imcPublierDesyncs publie, pour un archetype et un modele, les composants qui arretent la
// marche — sans quoi « DESYNC » ne veut rien dire.
func imcPublierDesyncs(t *testing.T, modele string, ti int, c *imcCompte, n int) {
	t.Helper()
	if c == nil || len(c.Desyncs) == 0 {
		return
	}
	type kv struct {
		k string
		n int
	}
	xs := make([]kv, 0, len(c.Desyncs))
	for k, v := range c.Desyncs {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].n > xs[j].n })
	if len(xs) > n {
		xs = xs[:n]
	}
	for _, x := range xs {
		t.Logf("      [%s ti=%d] %-58s %5d fois", modele, ti, x.k, x.n)
	}
}

// TestImageCleFermetureParArchetype EST LA MESURE : le tableau `archetype x modele`, par
// build puis tous films confondus, avec les temoins T+ et T-.
func TestImageCleFermetureParArchetype(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	global := imcNouvelleTable()
	parBuild := map[string]imcTable{}
	films := 0
	for _, dir := range dirs {
		f, ok := imcCharger(t, dir)
		if !ok {
			continue
		}
		films++
		local := imcNouvelleTable()
		imcMesurerFilm(f, local)
		imcFusionner(global, local)
		if parBuild[f.Build] == nil {
			parBuild[f.Build] = imcNouvelleTable()
		}
		imcFusionner(parBuild[f.Build], local)
		t.Logf("== film %s (build %q, %d images-cles) ==", f.Nom, f.Build, len(f.Pays))
		imcPublier(t, "film "+f.Nom, local)
	}
	if films == 0 {
		t.Skip("aucun film exploitable")
	}
	for _, b := range imcBuilds(parBuild) {
		imcPublier(t, "BUILD "+b, parBuild[b])
	}
	imcPublier(t, fmt.Sprintf("TOUS FILMS (%d)", films), global)
	imcPublierCauses(t, global)
	imcTemoins(t, global)
}

// imcPublierCauses publie, archetype par archetype, POURQUOI la marche ne ferme pas sous
// l'etat complet : le composant qui l'arrete (couverture du dispatch) ou la part de slots
// CONSECUTIFS, qui dit si la frontiere visee etait bien celle du voisin immediat.
func imcPublierCauses(t *testing.T, tab imcTable) {
	t.Helper()
	t.Logf("---- CAUSES, sous l'ETAT COMPLET ----")
	for _, ti := range imcTIs(tab) {
		c := tab[imcEtatComplet][ti]
		if c == nil || c.Total == 0 {
			continue
		}
		t.Logf("ti=%-2d | slots consecutifs : %d/%d fermes sur %d bornes consecutives",
			ti, c.VoisinFerme, c.VoisinTotal, c.VoisinTotal)
		imcPublierDesyncs(t, imcEtatComplet, ti, c, 2)
	}
}

// imcBuilds rend les builds rencontres, tries.
func imcBuilds(m map[string]imcTable) []string {
	out := make([]string, 0, len(m))
	for b := range m {
		out = append(out, b)
	}
	sort.Strings(out)
	return out
}

// imcFusionner ajoute `src` dans `dst`.
func imcFusionner(dst, src imcTable) {
	for _, m := range imcModeles() {
		for ti, c := range src[m] {
			d := dst.cellule(m, ti)
			imcAjouter(d, c)
			for k, v := range c.Desyncs {
				d.Desyncs[k] += v
			}
		}
	}
}

// imcTemoins joue T+ et T- sur le cumul, et publie le RESUME que le pilote lit.
func imcTemoins(t *testing.T, tab imcTable) {
	t.Helper()
	ec, pr := tab[imcEtatComplet][equipeTI], tab[imcProduction][equipeTI]
	t.Logf("T+ temoin positif  ti=%d etat complet : %s", equipeTI, imcCellule(ec))
	t.Logf("T- temoin negatif  ti=%d production   : %s", equipeTI, imcCellule(pr))
	t.Logf("Th temoin hasard   ti=%d +1 bit       : %s", equipeTI,
		imcCellule(tab[imcEtatDecale][equipeTI]))
	imcPublierDesyncs(t, imcEtatComplet, equipeTI, ec, 3)
	imcPublierDesyncs(t, imcProduction, equipeTI, pr, 3)
	gagnes, perdus, egaux := 0, 0, 0
	for _, ti := range imcTIs(tab) {
		a, b := tab[imcEtatComplet][ti], tab[imcProduction][ti]
		switch {
		case a == nil || b == nil:
		case a.Ferme > b.Ferme:
			gagnes++
		case a.Ferme < b.Ferme:
			perdus++
		default:
			egaux++
		}
	}
	t.Logf("BILAN par archetype : etat complet GAGNE sur %d, PERD sur %d, EGALITE sur %d",
		gagnes, perdus, egaux)
	if ec == nil || ec.Total == 0 {
		t.Errorf("T+ IMPOSSIBLE : aucun record ti=%d borne", equipeTI)
		return
	}
	if os.Getenv("CHUNK00_TEMOINS_DURS") == "" {
		return // par defaut l'instrument PUBLIE ; les temoins durs sont opt-in
	}
	if ec.Ferme*100 < ec.Total*99 {
		t.Errorf("T+ ECHOUE : ti=%d ferme %d/%d sous l'etat complet (desyncs : %v)",
			equipeTI, ec.Ferme, ec.Total, ec.Desyncs)
	}
	// T- se juge sur le TOTAL, pas sur ti=9 : la phase 4 a refute le modele de production pour
	// la TABLE, et sur ti=9 les deux modeles valent zero pour des raisons differentes (couverture
	// d'un cote, cadre de l'autre). Comparer la seule cellule ti=9 mesurerait une egalite sans
	// contenu.
	totEC, totPR := &imcCompte{}, &imcCompte{}
	for _, ti := range imcTIs(tab) {
		imcAjouter(totEC, tab[imcEtatComplet][ti])
		imcAjouter(totPR, tab[imcProduction][ti])
	}
	if totPR.Ferme >= totEC.Ferme {
		t.Errorf("T- ECHOUE : la production ferme %d/%d, l'etat complet %d/%d",
			totPR.Ferme, totPR.Total, totEC.Ferme, totEC.Total)
	}
}
