//go:build research

package grammar

// ri27b_canaux_marche_research_test.go — LA MESURE QUI DECIDE 2.7.b (plan de l etape 2 de la
// representation intermediaire) : pour les neuf lecteurs qui ancrent encore les records bipedes des
// trames delta ([FilmContext.parcourirLesAncresBipedes]), ce que la marche des trames lit des memes
// composants, record par record. Un instrument : aucun fichier de production n est touche.
//
// # LES DEUX COTES
//
// L ancrage : chaque record ancre est marche comme les lecteurs de production le marchent
// ([walkRecordComponents], grammaire du contexte), crochets des neuf lecteurs poses ; chaque appel
// est range sous son record (paquet, slot). La position est le bit d i0 du record.
//
// La marche : [Distribuer] avec un canal des trames. A chaque trame, les records que la marche a lus
// ([MarcheDistribuee.recordsDeLaTrame]) sont relus composant par composant, a l etendue que leur
// trace donne, sous le cadre de la marche, crochets poses : c est ce qu un canal de la marche
// recevrait, attribue au record qui le porte (les crochets ne portent pas le slot). La position est
// le debut d i0 dans la trace. Un composant traverse a une largeur calibree ou bouchon n est pas
// relu : aucun deserialiseur ne l a lu. La trame est comparee a l ancrage des qu elle est rendue.
//
// # LA FIDELITE DE LA RELECTURE
//
// Les crochets de la marche elle-meme sont recueillis trame par trame, tous archetypes : leur
// multiensemble, compare a celui de la relecture de TOUS les records lus, dit ce que la marche publie
// en plus de ses records (lectures d essai, NEW refuses) et ce que la relecture manquerait.
//
// # LES LISTES QUE LA MARCHE NE LOCALISE PAS
//
// Pour une trame dont la liste d evenements n est pas localisee, la vue B est lue comme le canal des
// morts la recupere ([MarcheDistribuee.recupererLaListe]) : ses records sont relus de meme, sous la
// classe `recuperee_signature` ou `recuperee_largeur_libre` ; la classe `non_localisee` ne garde que
// les trames que ce localisateur ne recupere pas non plus.
//
// # LA SORTIE (TSV)
//
// Par film, crochet et classe de trame (fermee, refusee, queue opaque par cause, non localisee ou
// recuperee, sans verdict, hors marche) : lus par l ancrage, lus par la marche, communs (meme record,
// meme valeur), divergents (meme record, valeur differente), ancrage seul, marche seule, puis l ancrage
// seul reparti selon que la marche a lu ou non un record du meme slot dans le paquet. Le pseudo-crochet
// `_records` compte les records eux-memes. La classe `nouveau` compte les lectures des NEW bipedes,
// que l ancrage ne voit jamais. Les classes `_fidelite_tete` et `_fidelite_liste` (trame partie de sa
// tete, ou d un debut de liste que le localisateur a cherche) reprennent les colonnes autrement :
// crochets de la marche, crochets de la relecture, communs.
//
// Contexte : celui de la cuisson ([ri27bContexte]) ; les deux cotes lisent sous le meme. Une mesure
// des morts d objet faite sans la generation stricte de la cuisson s etait revelee fausse (plan de
// l etape 2, mesure avant 2.7.a).
//
//	RI27B_FILMS=084a804d,111fa685 RI27B_RACINE=<film_chunks> RI27B_OUT=<tsv> \
//	  go test -tags=research -count=1 -run '^TestRI27bCanauxMarche$' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"encoding/json"
	"fmt"
	"hash/maphash"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// Les crochets mesures, et les deux pseudo-crochets.
var ri27bCrochets = []string{"charges", "camouflage", "capacite_non_predite", "capacite_spartan", "rang",
	"grenades_choix", "grenades_comptes", "arme_portee", "munitions", "cartouches", "equipement",
	"positions", "_records"}

const (
	ri27bPositions = 11
	ri27bRecords   = 12
)

// ri27bLu est UNE lecture : le slot du record, le crochet, l empreinte de la valeur.
type ri27bLu struct {
	slot    uint32
	crochet uint8
	valeur  uint64
	// bit : le premier bit du record, pour le seul pseudo-crochet `_records`.
	bit int
}

// ri27bPaquet designe un paquet delta.
type ri27bPaquet struct{ chunk, index int }

// ri27bCapteur recoit les appels des crochets des neuf lecteurs, sous le slot courant.
type ri27bCapteur struct {
	graine maphash.Seed
	slot   uint32
	tampon []ri27bLu
}

func (c *ri27bCapteur) noter(crochet int, valeur string) {
	c.tampon = append(c.tampon, ri27bLu{slot: c.slot, crochet: uint8(crochet), //nolint:gosec // moins de 13 crochets
		valeur: maphash.String(c.graine, valeur)})
}

// vider rend les lectures recues depuis le dernier vidage.
func (c *ri27bCapteur) vider() []ri27bLu {
	t := c.tampon
	c.tampon = nil
	return t
}

// brancher pose sur `obs` les crochets des neuf lecteurs ancres.
func (c *ri27bCapteur) brancher(obs *Observation) {
	obs.AbilityEnergyHook = func(m uint32, ch [AbilityEnergyCharges]int) { c.noter(0, fmt.Sprintf("%x:%v", m, ch)) }
	obs.CamoStateHook = func(st CamoState) { c.noter(1, fmt.Sprintf("%+v", st)) }
	obs.AbilityNonPredictedHook = func(st AbilityNonPredictedState) { c.noter(2, fmt.Sprintf("%+v", st)) }
	obs.SpartanAbilityHook = func(tag, sub, ref uint64, a bool) {
		c.noter(3, fmt.Sprintf("%x:%x:%x:%v", tag, sub, ref, a))
	}
	obs.AbilitySetHook = func(n uint64, rang, w int) { c.noter(4, fmt.Sprintf("%x:%d:%d", n, rang, w)) }
	obs.GrenadeSetHook = func(m uint32, sel int) { c.noter(5, fmt.Sprintf("%x:%d", m, sel)) }
	obs.GrenadeCountsHook = func(n uint64, v []uint64) { c.noter(6, fmt.Sprintf("%d:%v", n, v)) }
	obs.HeldWeaponHook = func(h, l uint32) { c.noter(7, fmt.Sprintf("%x:%x", h, l)) }
	obs.WeaponAmmoHook = func(a bool, m uint32, b bool, f uint32) { c.noter(8, fmt.Sprintf("%v:%d:%v:%d", a, m, b, f)) }
	obs.WeaponRoundsHook = func(n uint32) { c.noter(9, strconv.FormatUint(uint64(n), 10)) }
	obs.UnitEquipmentHook = func(u UnitEquipmentRead) { c.noter(10, fmt.Sprintf("%+v", u)) }
}

// ri27bAncrage marche les records ancres comme les lecteurs de production, et range leurs lectures
// par paquet.
func ri27bAncrage(t *testing.T, fc *FilmContext, graine maphash.Seed) map[ri27bPaquet][]ri27bLu {
	t.Helper()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage i0 : %v", err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		t.Fatalf("archetype bipede : %v", err)
	}
	obs := NouvelleObservation()
	capt := &ri27bCapteur{graine: graine}
	capt.brancher(obs)
	g := grammaireRecord{lay: lay, arch: arch, prof: fc.ProfilDeBalayage(), obs: obs}
	out := map[ri27bPaquet][]ri27bLu{}
	fc.parcourirLesAncresBipedes(func(r deltaBipedRecord) {
		capt.slot = r.Slot
		walkRecordComponents(r.Payload, r.I0, r.Total, r.Mask, g, func(int) bool { return true })
		capt.noter(ri27bPositions, strconv.Itoa(r.I0))
		capt.noter(ri27bRecords, "")
		capt.tampon[len(capt.tampon)-1].bit = r.I0
		k := ri27bPaquet{r.Chunk, r.Packet.Index}
		out[k] = append(out[k], capt.vider()...)
	})
	return out
}

// ri27bStats : ancrage, marche, communs, divergents, ancrage seul, marche seule, puis l ancrage seul
// reparti : la marche a lu un record du meme slot dans le paquet, ou n en a lu aucun.
type ri27bStats [8]int

// ri27bCanal est le canal des trames de la mesure : il relit les records de chaque trame et la
// compare a l ancrage de son paquet.
type ri27bCanal struct {
	reg    *Registry
	m      *MarcheDistribuee
	propre *ri27bCapteur // les crochets de la marche elle-meme
	relu   *ri27bCapteur
	obs    *Observation // l observation de la relecture
	ancre  map[ri27bPaquet][]ri27bLu
	stats  map[[2]string]*ri27bStats
}

func nouveauCanalRI27b(reg *Registry, ancre map[ri27bPaquet][]ri27bLu, graine maphash.Seed) *ri27bCanal {
	c := &ri27bCanal{reg: reg, ancre: ancre, stats: map[[2]string]*ri27bStats{},
		propre: &ri27bCapteur{graine: graine}, relu: &ri27bCapteur{graine: graine}, obs: NouvelleObservation()}
	c.relu.brancher(c.obs)
	return c
}

func (c *ri27bCanal) Interets() []Interet { return nil }
func (c *ri27bCanal) Clore(BilanDeMarche) {}
func (c *ri27bCanal) Brancher(obs *Observation, m *MarcheDistribuee) {
	c.m = m
	c.propre.brancher(obs)
}

// Trame relit les records de la trame, compare ses lectures bipedes a l ancrage du paquet, et la
// relecture de tous ses records aux crochets de la marche.
func (c *ri27bCanal) Trame(p *lecture.Paquet) {
	classe := ri27bClasse(p)
	propres := c.propre.vider()
	var marche, nouveaux, tous []ri27bLu
	recs, lus := c.m.recordsDeLaTrame()
	recuperee := false
	if !lus && listeAnnoncee(&p.VueA) {
		// La recuperation du canal des morts : la vue B lue depuis le debut que le localisateur
		// unique retrouve, sans crochet. Ses lectures n entrent pas dans la fidelite.
		var libre bool
		if recs, lus, libre = c.m.recupererLaListe(); lus {
			recuperee, classe = true, "recuperee_signature"
			if libre {
				classe = "recuperee_largeur_libre"
			}
		}
	}
	if lus {
		cfg := c.m.cadreDeLaMarche()
		cfg.Obs = c.obs
		for i := range recs {
			r := &recs[i]
			l := c.relire(p.Payload, r, cfg)
			if !recuperee {
				tous = append(tous, l...)
			}
			if r.TypeIndex != BipedTypeIndex {
				continue
			}
			switch r.Type {
			case recDelta:
				if len(r.Trace.Comps) > 0 && r.Trace.Comps[0].Index == 0 && r.Trace.Comps[0].Ported {
					l = append(l, ri27bLu{slot: r.Slot, crochet: ri27bPositions,
						valeur: maphash.String(c.relu.graine, strconv.Itoa(r.Trace.Comps[0].StartBit))})
				}
				l = append(l, ri27bLu{slot: r.Slot, crochet: ri27bRecords, valeur: maphash.String(c.relu.graine, ""),
					bit: r.HeaderBit})
				marche = append(marche, l...)
			case recNew:
				nouveaux = append(nouveaux, l...)
			}
		}
	}
	k := ri27bPaquet{p.Chunk, p.Index}
	c.comparer(classe, c.ancre[k], marche)
	c.situerLesAncresSeules(classe, c.ancre[k], marche, ri27bFinDeLecture(recs, lus))
	delete(c.ancre, k)
	for _, l := range nouveaux {
		c.stat(int(l.crochet), "nouveau")[1]++
	}
	c.fidelite(ri27bClasseDeFidelite(p), propres, tous)
}

// ri27bFinDeLecture rend le bit qui suit le dernier record que la marche a lu dans la trame, -1 sans
// lecture.
func ri27bFinDeLecture(recs []FrameRecord, lus bool) int {
	fin := -1
	if lus {
		for i := range recs {
			fin = max(fin, recs[i].FinBit)
		}
	}
	return fin
}

// situerLesAncresSeules range les records ancres dont la marche n a lu aucun record du meme slot dans
// la trame : en deca de la fin de sa lecture (elle est passee par la sans les voir) ou au-dela (elle
// n y est pas allee). Rows `_records` de classe `<classe>/en_deca` et `<classe>/au_dela`, colonne
// ancrage seul.
func (c *ri27bCanal) situerLesAncresSeules(classe string, a, w []ri27bLu, fin int) {
	vus := map[uint32]bool{}
	for _, l := range w {
		if l.crochet == ri27bRecords {
			vus[l.slot] = true
		}
	}
	for _, l := range a {
		if l.crochet != ri27bRecords || vus[l.slot] {
			continue
		}
		cote := "/au_dela"
		if l.bit < fin {
			cote = "/en_deca"
		}
		c.stat(ri27bRecords, classe+cote)[4]++
	}
}

// relire relit les composants lus d un record, sous le cadre de la marche, et rend les lectures des
// crochets sous son slot.
func (c *ri27bCanal) relire(pay []byte, r *FrameRecord, cfg FrameConfig) []ri27bLu {
	arch, ok := c.reg.Archetype(int(r.TypeIndex))
	if !ok {
		return nil
	}
	c.relu.slot = r.Slot
	for _, comp := range r.Trace.Comps {
		if !comp.Ported {
			break
		}
		if comp.Prov == lecture.LargeurCalibree || comp.Prov == lecture.LargeurBouchon {
			continue
		}
		br := LecteurSur(pay)
		br.poserCadre(cfg)
		br.SetBitPos(comp.StartBit)
		consumeByNameCapturing(br, comp.Name, r.TypeIndex, arch.Level(comp.Index))
	}
	return c.relu.vider()
}

// stat rend le compteur du crochet `crochet` dans la classe `classe`.
func (c *ri27bCanal) stat(crochet int, classe string) *ri27bStats {
	k := [2]string{ri27bCrochets[crochet], classe}
	if c.stats[k] == nil {
		c.stats[k] = &ri27bStats{}
	}
	return c.stats[k]
}

// ri27bGroupe range des lectures par (slot, crochet) : le multiensemble des empreintes.
func ri27bGroupe(lus []ri27bLu) map[[2]uint32]map[uint64]int {
	g := map[[2]uint32]map[uint64]int{}
	for _, l := range lus {
		k := [2]uint32{l.slot, uint32(l.crochet)}
		if g[k] == nil {
			g[k] = map[uint64]int{}
		}
		g[k][l.valeur]++
	}
	return g
}

// comparer compare, record par record et crochet par crochet, l ancrage `a` et la marche `w` d un
// paquet.
func (c *ri27bCanal) comparer(classe string, a, w []ri27bLu) {
	ga, gw := ri27bGroupe(a), ri27bGroupe(w)
	cles := map[[2]uint32]bool{}
	for k := range ga {
		cles[k] = true
	}
	for k := range gw {
		cles[k] = true
	}
	for k := range cles {
		na, nw, com := 0, 0, 0
		for v, n := range ga[k] {
			na += n
			com += min(n, gw[k][v])
		}
		for _, n := range gw[k] {
			nw += n
		}
		div := min(na-com, nw-com)
		s := c.stat(int(k[1]), classe)
		s[0] += na
		s[1] += nw
		s[2] += com
		s[3] += div
		s[4] += na - com - div
		s[5] += nw - com - div
		if _, lu := gw[[2]uint32{k[0], ri27bRecords}]; lu {
			s[6] += na - com - div
		} else {
			s[7] += na - com - div
		}
	}
}

// fidelite compare, par crochet, les crochets de la marche et la relecture de la trame.
func (c *ri27bCanal) fidelite(classe string, propres, relus []ri27bLu) {
	ga, gw := map[[2]uint64]int{}, map[[2]uint64]int{}
	for _, l := range propres {
		ga[[2]uint64{uint64(l.crochet), l.valeur}]++
	}
	for _, l := range relus {
		gw[[2]uint64{uint64(l.crochet), l.valeur}]++
	}
	for k, n := range ga {
		s := c.stat(int(k[0]), classe)
		s[0] += n
		s[2] += min(n, gw[k])
	}
	for k, n := range gw {
		c.stat(int(k[0]), classe)[1] += n
	}
}

// ri27bClasse range une trame delta par ce que la marche en a fait.
func ri27bClasse(p *lecture.Paquet) string {
	if p.Debut == lecture.DebutNonLocalise {
		return "non_localisee"
	}
	switch p.Fermeture.Verdict {
	case lecture.VerdictFerme:
		return "fermee"
	case lecture.VerdictRefuse:
		return "refusee"
	case lecture.VerdictQueueOpaque:
		return fmt.Sprintf("queue_%d", p.Fermeture.Queue.Cause)
	}
	return "sans_verdict"
}

// ri27bCarte lit la carte d un film dans ses faits d equivalence.
func ri27bCarte(t *testing.T, court string) string {
	t.Helper()
	brut, err := os.ReadFile(filepath.Join("..", "..", "replay", "testdata", "equivalence", court+".facts.json"))
	if err != nil {
		t.Fatalf("faits %s : %v", court, err)
	}
	var f struct {
		MapNames []string `json:"mapNames"`
	}
	if err := json.Unmarshal(brut, &f); err != nil || len(f.MapNames) == 0 {
		t.Fatalf("carte de %s introuvable (%v)", court, err)
	}
	return f.MapNames[0]
}

// ri27bContexte ouvre le contexte de film comme la cuisson : entree de catalogue et largeurs de la
// carte ([ti40dContexte]), generation stricte du profil de depart de killsource, puis le decoupage
// MPP que la grammaire resout pour le film. La calibration de killsource ne pose rien d autre sur le
// corpus : la largeur du mot de poignee y reste l invariant (« non discriminee » sur les 19
// temoins).
func ri27bContexte(t *testing.T, dir, carte string) *FilmContext {
	t.Helper()
	fc := ti40dContexte(t, dir, carte)
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.GenerationStricte = true
	fc.PoserProfilDeBalayage(bal)
	if res := fc.ResolutionMPP(); res.Decide() {
		fc.PoserMPP(res.Widths)
	}
	return fc
}

// ri27bFilm mesure un film et rend ses lignes.
func ri27bFilm(t *testing.T, racine, court string, graine maphash.Seed) []string {
	t.Helper()
	fc := ri27bContexte(t, filepath.Join(racine, court), ri27bCarte(t, court))
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("%s : registre : %v", court, err)
	}
	canal := nouveauCanalRI27b(reg, ri27bAncrage(t, fc, graine), graine)
	if err := Distribuer(fc, canal); err != nil {
		t.Fatalf("%s : distribution : %v", court, err)
	}
	for _, lus := range canal.ancre { // paquets ancres que la marche n a pas rendus
		canal.comparer("hors_marche", lus, nil)
	}
	var cles [][2]string
	for k := range canal.stats {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i][0] != cles[j][0] {
			return cles[i][0] < cles[j][0]
		}
		return cles[i][1] < cles[j][1]
	})
	var lignes []string
	for _, k := range cles {
		s := canal.stats[k]
		lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", court, k[0], k[1],
			s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7]))
	}
	return lignes
}

func TestRI27bCanauxMarche(t *testing.T) {
	films, racine, sortie := os.Getenv("RI27B_FILMS"), os.Getenv("RI27B_RACINE"), os.Getenv("RI27B_OUT")
	if films == "" || racine == "" || sortie == "" {
		t.Skip("instrument : RI27B_FILMS, RI27B_RACINE et RI27B_OUT requis")
	}
	graine := maphash.MakeSeed()
	lignes := []string{"film\tcrochet\tclasse\tancrage\tmarche\tcommuns\tdivergents\tancrage_seul\tmarche_seule\t" +
		"ancrage_seul_slot_lu\tancrage_seul_slot_non_lu"}
	for _, court := range strings.Split(films, ",") {
		l := ri27bFilm(t, racine, court, graine)
		lignes = append(lignes, l...)
		t.Logf("%s : %d lignes", court, len(l))
		if err := os.WriteFile(sortie, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
			t.Fatalf("sortie : %v", err)
		}
		runtime.GC()
	}
}

// ri27bClasseDeFidelite range une trame pour la fidelite : partie de la tete du paquet (aucun
// localisateur), ou d un debut de liste que le localisateur a cherche.
func ri27bClasseDeFidelite(p *lecture.Paquet) string {
	if p.Debut == lecture.DebutEnTete {
		return "_fidelite_tete"
	}
	return "_fidelite_liste"
}
