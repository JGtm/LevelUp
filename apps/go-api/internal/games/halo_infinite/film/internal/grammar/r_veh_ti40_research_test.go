//go:build research && campagne_overlay

package grammar

// r_veh_ti40_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L4 (a) (2026-10-02) : quelle
// largeur `ti=40` est fausse en image-cle (D-52) ? Mesure seulement, aucun fichier de production.
//
// EXIGE LA SURCOUCHE DE RECHERCHE (tag `campagne_overlay`) : les composants `ti=40` non portes
// sont lus par le crochet [b2vCrochet] de `campagne_bis2_lecteurs_research_test.go` (grammaires
// de T6 §4), la porte `+0x818` est posee PAR CHASSIS (table CAMPAGNE_PHYSIQUE, chassis inconnu ->
// levee), comme la variante « porte par chassis » de BIS_2 §4.2.
//
// Deux mesures :
//
//  1. [TestRVehTi40Diagnostic] : par record d image-cle `ti=40` borne, l issue, l exces
//     (fin lue - frontiere), les portes de l etat par defaut (bVar14, cVar3), les deux mots de
//     taille n1 et n2, et la largeur de chaque composant lu.
//  2. [TestRVehTi40Balayage] : pour chaque composant lu d un record non ferme (et pour la fin de
//     l etat par defaut), la reprise de la boucle a la fin de ce composant DECALEE de d bits
//     (|d| <= CAMPAGNE_DECALAGE, 32 par defaut) ; on compte les records que chaque (composant, d)
//     ferme. Une largeur fixe fausse donne un d MODAL ; le hasard se repartit sur tous les d
//     (temoin : la colonne des d opposes et le compte par d voisin).
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestRVehTi40' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rvPre : ce que la marche lit avant la boucle de composants (FUN_142e2bfd0, FUN_1410a5a74).
type rvPre struct {
	n1, n2       uint64
	b14, c3      bool
	liste        uint64
	dsDebut      int // premier bit de l etat par defaut
	dsFin        int // premier bit apres l etat par defaut (avant le controle et n2)
	boucle       int // premier bit de la boucle de composants
	quat         int // bits de la feuille 4 (quaternion) quand bVar14
	sansEtat     bool
	sansBoucle   bool
	controleFilm bool
	feuille      rvF4
}

// rvF4 : la feuille 4 de l etat par defaut (FUN_14076e494 niveau 0x10 puis FUN_140c1e79c).
type rvF4 struct {
	brute bool
	idx   int
	w     [3]uint
}

// rvFeuille4 rejoue [consumeVehicleMediaFrame] etape par etape ([lireE494Sur] sur les tables du
// profil) en relevant l index de plage et les largeurs lues.
func rvFeuille4(br *Lecteur) rvF4 {
	var f rvF4
	if rvPortee || fullPrecisionGate(br) {
		br.ReadBits(rawVec3Bits)
		f.brute, f.idx = true, -2
	} else {
		pos, _ := lireE524Sur(br, niveauPosition, br.tablesDuProfil())
		f.idx, f.w = pos.idx, pos.w
	}
	consume140c1e79c(br)
	return f
}

// rvPrefixe rejoue l en-tete, n1, l etat par defaut `ti=40` feuille par feuille (meme suite que
// [consumeDefaultStateTI40]) et n2, en relevant les portes.
func rvPrefixe(pay []byte, bit int, ctx ContexteDeLecture) rvPre {
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(bit + br.cadre().EnTeteBits)
	mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil
	var p rvPre
	p.n1 = br.ReadBits(mot)
	p.dsDebut = br.BitPos()
	p.controleFilm = br.p.Grammaire.ControleDeCorruption
	if int32(p.n1) > 0 { //nolint:gosec // compare signe comme le jeu
		consumeVersionPrefix(br)
		consumeMultiplayerPropertiesBlock(br)
		if p.b14 = br.ReadBit(); p.b14 {
			a := br.BitPos()
			p.feuille = rvFeuille4(br)
			p.quat = br.BitPos() - a
		}
		br.ReadBits(19)
		if rvV.etatSansListe {
			consumeOpt32(br)
		} else if p.c3 = br.ReadBit(); !p.c3 {
			consumeOpt32(br)
		} else {
			p.liste = br.ReadBits(2)
			for i := uint64(0); i < p.liste; i++ {
				consumeOpt32(br)
			}
		}
		p.dsFin = br.BitPos()
		if p.controleFilm {
			br.ReadBits(mot)
		}
	} else {
		p.sansEtat = true
		p.dsFin = br.BitPos()
	}
	p.n2 = br.ReadBits(mot)
	p.sansBoucle = int32(p.n2) <= 0 //nolint:gosec // idem
	p.boucle = br.BitPos()
	return p
}

// rvTailleVehicule : le memset de vtable[0x88] (`FUN_14058c2ec`, `memset(param_5, 0, 0x8d8)`,
// HI_1_13_0) — la valeur que n2 doit lire quand l etat par defaut a la bonne largeur.
const rvTailleVehicule = 0x8d8

// rvOuEstN2 rend les decalages d (|d| <= 96) tels que le mot de 32 bits lu a `dsFin + d` vaille
// la taille de l etat vehicule : la position VRAIE de n2, donc la largeur vraie de l etat par
// defaut (largeur lue + d).
func rvOuEstN2(pay []byte, p rvPre) []int {
	var out []int
	for d := -96; d <= 96; d++ {
		q := p.dsFin + d
		if q < 0 || q+32 > len(pay)*8 {
			continue
		}
		if sourceBits32(pay, q) == rvTailleVehicule {
			out = append(out, d)
		}
	}
	return out
}

// sourceBits32 lit 32 bits a la position q (meme convention que le lecteur).
func sourceBits32(pay []byte, q int) uint64 {
	br := LecteurSur(pay)
	br.SetBitPos(q)
	return br.ReadBits(32)
}

// rvReprendre reprend la boucle de composants a `pos`, a partir de l index `from`.
func rvReprendre(pay []byte, pos, from int, arch Archetype, ctx ContexteDeLecture) EntityTrace {
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(pos)
	t := EntityTrace{DesyncAt: -1, TypeIndex: 40, Mask: ^uint64(0)}
	traverseComponentLoopFrom(br, arch, &t, from)
	t.EndBit = br.BitPos()
	return t
}

// rvReprendreApresEtat reprend apres l etat par defaut decale : controle, n2, puis la boucle.
func rvReprendreApresEtat(pay []byte, pos int, p rvPre, arch Archetype, ctx ContexteDeLecture) EntityTrace {
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(pos)
	mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil
	if p.controleFilm && !p.sansEtat {
		br.ReadBits(mot)
	}
	if int32(br.ReadBits(mot)) <= 0 { //nolint:gosec // idem
		return EntityTrace{DesyncAt: -1, EndBit: br.BitPos()}
	}
	t := EntityTrace{DesyncAt: -1, TypeIndex: 40, Mask: ^uint64(0)}
	traverseComponentLoopFrom(br, arch, &t, 0)
	t.EndBit = br.BitPos()
	return t
}

// rvClasse range un chassis par type de physique.
func rvClasse(physique map[uint32]int, mpp uint32, vu bool) (string, int, bool) {
	if !vu {
		return "sans-mpp", -1, false
	}
	ty, ok := physique[mpp]
	if !ok {
		return "inconnu", -1, false
	}
	return "type" + strconv.Itoa(ty), ty, true
}

// rvEnv : films, sortie, table de physique, decalage maximal.
func rvEnv(t *testing.T) (racine, sortie string, films []string, physique map[uint32]int, dmax int) {
	racine, sortie, films = b2Env(t)
	physique = b2vPhysique()
	dmax = 32
	rvV = rvVariante{nom: "env", crochet: true, porte: "chassis", i0: os.Getenv("CAMPAGNE_I0_ECRIVAIN") == "1"}
	rvV.portee, _ = strconv.Atoi(os.Getenv("CAMPAGNE_PORTEE"))
	if os.Getenv("CAMPAGNE_MPP83") == "1" {
		rvV.mpp = profile.MPPWidths{Lead: 8, Index: 3}
	}
	rvPortee = rvV.portee >= 1
	if v, err := strconv.Atoi(os.Getenv("CAMPAGNE_DECALAGE")); err == nil && v > 0 {
		dmax = v
	}
	return racine, sortie, films, physique, dmax
}

// rvRecord : un record d image-cle `ti=40` borne et sa marche par chassis.
type rvRecord struct {
	chunk  int
	b      keyframeBorne
	pay    []byte
	mpp    uint32
	vu     bool
	classe string
	tr     EntityTrace
	pre    rvPre
	issue  string
}

// rvRecords marche tous les records `ti=40` bornes d un film, porte par chassis.
func rvRecords(f *cmFilm, physique map[uint32]int, rappel func(r *rvRecord)) {
	ctx := rvContexte(f)
	marche := f.fc.MarcheDImageCle()
	for _, num := range f.fc.ChunkNumbers() {
		data, pks, ok := f.fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, b := range seulesBornees(keyframeBornesDe(marche.Records(pay))) {
				if b.TI != 40 {
					continue
				}
				bis2Intercepteur = nil
				_, mpp, vu := b2vMarcherImageCle(pay, b, f.reg, ctx)
				classe, ty, connu := rvClasse(physique, mpp, vu)
				if rvV.crochet {
					bis2Intercepteur = b2vCrochet(true, false)
				}
				switch rvV.porte {
				case "posee":
					b2vPorte = true
				case "levee":
					b2vPorte = false
				default:
					b2vPorte = connu && ty == 6
				}
				r := &rvRecord{chunk: num, b: b, pay: pay, mpp: mpp, vu: vu, classe: classe}
				r.tr = rvMarcher(pay, b.Bit, f.reg, ctx)
				r.pre = rvPrefixe(pay, b.Bit, ctx)
				switch {
				case r.tr.DesyncAt >= 0:
					r.issue = "desync:" + nomComposantBloquant(f.reg, b.TI, r.tr.DesyncAt)
				case r.tr.EndBit == b.Want:
					r.issue = "fermee"
				case r.tr.EndBit < b.Want:
					r.issue = "sous"
				default:
					r.issue = "sur"
				}
				rappel(r)
				bis2Intercepteur = nil
			}
		}
	}
}

// rvOuvrir ouvre un film sous la sentinelle et installe son decoupage MPP.
func rvOuvrir(t *testing.T, racine, id, nom string) (*cmFilm, func()) {
	garde := filmproc.Arm(nom, 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	f, ok := cmOuvrir(t, racine, id, cmUtiles(t))
	if !ok {
		garde.Disarm()
		return nil, nil
	}
	restore, errMPP := InstallFilmFormatMPP(f.fc)
	// Le decoupage MPP 8/3 ne vient QUE de l environnement : la variante courante d un autre
	// film ne doit jamais fuir dans celui-ci.
	if os.Getenv("CAMPAGNE_MPP83") == "1" {
		f.fc.PoserMPP(profile.MPPWidths{Lead: 8, Index: 3})
	}
	debut := time.Now()
	return f, func() {
		if errMPP == nil {
			restore()
		}
		t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
}

// TestRVehTi40Diagnostic : une ligne par record `ti=40` borne, plus le registre `ti=40` du film.
func TestRVehTi40Diagnostic(t *testing.T) {
	racine, sortie, films, physique, _ := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte = nil, false }()
	recs := []string{"film\tbuild\tchunk\tslot\tvoisin\tchassis\tclasse\tissue\texces\tlongueur\tn1\tn2\t" +
		"b14\tquat\tf4_idx\tf4_largeurs\tc3\tliste\tetat_bits\tcomposants_lus\tlargeurs"}
	regs := []string{"film\tbuild\ti\tcomposant\tniveau"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-diag")
		if f == nil {
			continue
		}
		if arch, ok := f.reg.Archetype(40); ok {
			for i, c := range arch.Components {
				regs = append(regs, fmt.Sprintf("%s\t%s\t%02d\t%s\t%d", id, f.build, i, c, arch.Level(i)))
			}
		}
		rvRecords(f, physique, func(r *rvRecord) {
			var ls []string
			for j, c := range r.tr.Comps {
				fin := r.tr.EndBit
				if j+1 < len(r.tr.Comps) {
					fin = r.tr.Comps[j+1].StartBit
				}
				ls = append(ls, fmt.Sprintf("i%d=%d", c.Index, fin-c.StartBit))
			}
			recs = append(recs, fmt.Sprintf("%s\t%s\t%04d\t%05d\t%t\t%08x\t%s\t%s\t%d\t%d\t%d\t%d\t%t\t%d\t%d\t%v\t%v\t%t\t%d\t%d\t%d\t%s",
				f.id, f.build, r.chunk, r.b.Slot, r.b.Voisin, r.mpp, r.classe, r.issue, r.tr.EndBit-r.b.Want,
				r.b.Want-r.b.Bit, r.pre.n1, r.pre.n2, r.pre.b14, r.pre.quat, r.pre.feuille.idx, r.pre.feuille.w, rvOuEstN2(r.pay, r.pre),
				r.pre.c3, r.pre.liste,
				r.pre.dsFin-r.pre.dsDebut, len(r.tr.Comps), strings.Join(ls, ",")))
		})
		fin()
	}
	b2Ecrire(t, sortie, "r_veh_ti40_records.tsv", recs)
	b2Ecrire(t, sortie, "r_veh_ti40_registre.tsv", regs)
}

// rvCle : un (film, classe, composant, decalage).
type rvCle struct {
	film, build, classe, comp string
	d                         int
}

// TestRVehTi40Balayage : decalage d une frontiere de composant, records fermes par (composant, d).
func TestRVehTi40Balayage(t *testing.T) {
	racine, sortie, films, physique, dmax := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte = nil, false }()
	lignes := []string{"film\tbuild\tclasse\tcomposant\td\trecords_fermes\trecords_candidats"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-balayage")
		if f == nil {
			continue
		}
		arch, ok := f.reg.Archetype(40)
		if !ok {
			fin()
			continue
		}
		ctx := rvContexte(f)
		compte := map[rvCle]int{}
		candidats := map[[2]string]int{}
		rvRecords(f, physique, func(r *rvRecord) {
			if r.issue != "sous" && r.issue != "sur" {
				return
			}
			candidats[[2]string{r.classe, ""}]++
			vus := map[[2]string]bool{}
			ferme := func(comp string, d int, tr EntityTrace) {
				if tr.DesyncAt < 0 && tr.EndBit == r.b.Want && !vus[[2]string{comp, strconv.Itoa(d)}] {
					vus[[2]string{comp, strconv.Itoa(d)}] = true
					compte[rvCle{f.id, f.build, r.classe, comp, d}]++
				}
			}
			for d := -dmax; d <= dmax; d++ {
				if d == 0 {
					continue
				}
				if !r.pre.sansBoucle {
					ferme("etat-par-defaut", d, rvReprendreApresEtat(r.pay, r.pre.dsFin+d, r.pre, arch, ctx))
				}
				for j, c := range r.tr.Comps {
					fin := r.tr.EndBit
					if j+1 < len(r.tr.Comps) {
						fin = r.tr.Comps[j+1].StartBit
					}
					if fin+d < 0 {
						continue
					}
					nom := fmt.Sprintf("i%02d %s", c.Index, strings.TrimSuffix(c.Name, "-component"))
					ferme(nom, d, rvReprendre(r.pay, fin+d, c.Index+1, arch, ctx))
				}
			}
		})
		for k, n := range compte {
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%d", k.film, k.build, k.classe, k.comp, k.d, n,
				candidats[[2]string{k.classe, ""}]))
		}
		fin()
	}
	sort.Strings(lignes[1:])
	b2Ecrire(t, sortie, "r_veh_ti40_balayage.tsv", lignes)
}

// rvVariante : une maniere de lire les records `ti=40` d image-cle.
type rvVariante struct {
	nom     string
	crochet bool   // les 16 composants non portes lus par le crochet (T6 §4)
	portee  int    // 0 : production ; 1 : portee sur l etat par defaut ; 2 : sur tout le record
	i0      bool   // GrammaireEcrivainI0 (chemin absolu d i0 = grammaire de l ecrivain)
	porte   string // `+0x818` : chassis (inconnu -> levee), posee, levee
	// etatSansListe : etat par defaut SANS la porte cVar3 ni la liste (feuilles 6-7b), un seul
	// opt32 apres le R(19) — hypothese des builds anciens, ou n1 vaut 172 (0xac) et non 176.
	etatSansListe bool
	// mpp : decoupage MPP pose pour la variante (vide = celui du film, [InstallFilmFormatMPP]).
	mpp profile.MPPWidths
	// decale : temoin negatif, la boucle de composants commence `decale` bits apres n2.
	decale int
}

// rvV : la variante courante (une mesure a la fois, aucun parallelisme).
var rvV rvVariante

// rvPortee : la variante « portee de l etat par defaut ». FUN_142e2bfd0 pose DAT_144e61ea0 = 1
// (@142e2c46f) avant l appel de vtable[0x60] (@142e2c47b) et le remet a 0 (@142e2c530) ; sous
// cette portee, FUN_14076f91c rend vrai et FUN_14076e494 lit R(96) (FUN_1411b259c). Faux = la
// lecture de production (quantifiee, [consumeVehicleMediaFrame]).
var rvPortee bool

// rvMarcher : [WalkKeyframeFullState] pour `ti=40`, l etat par defaut lu par [rvPrefixe] (donc
// sous la variante [rvPortee]) ; sans la variante, la marche de production.
func rvMarcher(pay []byte, bit int, reg *Registry, ctx ContexteDeLecture) EntityTrace {
	if !rvPortee {
		return WalkKeyframeFullState(pay, bit, reg, ctx)
	}
	p := rvPrefixe(pay, bit, ctx)
	t := EntityTrace{DesyncAt: -1, TypeIndex: 40}
	arch, ok := reg.Archetype(40)
	if !ok || p.sansBoucle {
		t.EndBit = p.boucle
		return t
	}
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(p.boucle + rvV.decale)
	t.Mask = ^uint64(0)
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// rvContexte : le contexte du film, avec les variantes de lecture des positions demandees par
// l environnement (CAMPAGNE_PORTEE=2 : portee sur TOUT le record, = PorteeBaseline ;
// CAMPAGNE_I0_ECRIVAIN=1 : GrammaireEcrivainI0).
func rvContexte(f *cmFilm) ContexteDeLecture {
	ctx := f.fc.ContexteDeLecture()
	if rvV.portee == 2 {
		ctx.Profil.Grammaire.PorteeBaseline = true
	}
	if rvV.i0 {
		ctx.Profil.Grammaire.GrammaireEcrivainI0 = true
	}
	return ctx
}
