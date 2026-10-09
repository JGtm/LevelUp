//go:build research

package grammar

// ecart108_research_test.go — INSTRUCTION DES ECARTS DE FIN MULTIPLES DE -108 BITS (hypothese R-VEH-4,
// plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md` §7, branche feat/ri-ecart-108). Aucun
// fichier de production n est touche.
//
// CE QUE LE JEU LIT (Ghidra, lecture seule, HaloInfinite.exe) :
//   - la table d image-cle est un tableau de 0x1fff entrees de 200 octets indexe par SLOT
//     (`FUN_1408be074` : `FUN_1411b2400(param_1, 0x1fff)` ; acces `(eid & 0x3fffffff) * 200 + base`
//     avec `*entree == eid`, `FUN_1405d5dbc`, `FUN_1406cb5f0`) ; l ecrivain (`FUN_142e2d08c`) ecrit
//     TOUTES les entrees, le lecteur (`FUN_142e2bfd0`) les relit toutes, aucun compte dans le flux ;
//   - en-tete d entree : R(32) id, R(32) archetype, R(32) mot +0xc, R(4), R(8) = 108 bits ; corps
//     (n1, etat par defaut, controle, n2, composants) seulement si archetype != 0xffffffff ;
//   - une entree liberee (`FUN_1408f1948`) porte id = archetype = mot +0xc = 0xffffffff.
//
// L instrument lit, pour chaque record d image-cle borne, les entrees que la marche ne voit pas
// entre la fin de sa traversee et l ancre suivante, et simule deux regles :
//
//	R108  : sauter les entrees d archetype 0xffffffff (108 bits chacune) avant de comparer a l ancre ;
//	Rslot : lire exactement (slot suivant - slot - 1) entrees comme le jeu (en-tete, corps si archetype
//	        != 0xffffffff), puis comparer a l ancre.
//
// Sortie (RI27C_OUT) : `ecart108.tsv` — lignes
//
//	K   film fmt ts slot slotSuivant k ecartSlots r108 rslot blocs      records bipedes a ecart -108k
//	G   film fmt cle compte                                             agregats bipedes
//	X   film fmt ti cle compte                                          agregats tous archetypes
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> RI27C_CARTES=<id=Carte;...> \
//	  go test -tags=research -count=1 -run '^TestEcart108$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// e108EnTete est la largeur de l en-tete d entree que lit `FUN_142e2bfd0` (32+32+32+4+8).
const e108EnTete = 108

// e108Entree est l en-tete d une entree de table, tel que l ecrit `FUN_142e2d08c`.
type e108Entree struct {
	id, arch, mot uint32
	f4, f8        uint8
}

func e108Lire(pay []byte, q int) (e e108Entree, ok bool) {
	if q < 0 || q+e108EnTete > len(pay)*8 {
		return e, false
	}
	e.id = uint32(source.BitsBourres(pay, q, 32))      //nolint:gosec // 32 bits
	e.arch = uint32(source.BitsBourres(pay, q+32, 32)) //nolint:gosec // 32 bits
	e.mot = uint32(source.BitsBourres(pay, q+64, 32))  //nolint:gosec // 32 bits
	e.f4 = uint8(source.BitsBourres(pay, q+96, 4))     //nolint:gosec // 4 bits
	e.f8 = uint8(source.BitsBourres(pay, q+100, 8))    //nolint:gosec // 8 bits
	return e, true
}

// classe rend une etiquette compacte de l entree : forme de l id, de l archetype et du mot +0xc.
func (e e108Entree) classe(slotAttendu int) string {
	var id string
	switch {
	case e.id == kfSent:
		id = "idFF"
	case e.id>>30 == 0:
		id = "gen0"
	default:
		id = "genOK"
	}
	if e.id != kfSent {
		if int(e.id&0x3FFFFFFF) == slotAttendu {
			id += "/slot="
		} else {
			id += "/slot!="
		}
	}
	var ar string
	switch {
	case e.arch == keyframeArchetypeNone:
		ar = "archFF"
	case e.arch < kfArchMax:
		ar = fmt.Sprintf("arch%d", e.arch)
	default:
		ar = "archHors"
	}
	mot := "motAutre"
	if e.mot == kfSent {
		mot = "motFF"
	}
	return fmt.Sprintf("%s,%s,%s,f4=%x,f8=%02x", id, ar, mot, e.f4, e.f8)
}

// e108Sauter applique R108 : depuis `pos`, saute les entrees d archetype 0xffffffff tant que la
// position n atteint pas `want` ; rend le nombre d entrees sautees et si `want` est atteint.
func e108Sauter(pay []byte, pos, want int) (int, bool) {
	k := 0
	for pos < want {
		e, ok := e108Lire(pay, pos)
		if !ok || e.arch != keyframeArchetypeNone {
			return k, false
		}
		pos += e108EnTete
		k++
	}
	return k, pos == want
}

// e108Suivre applique Rslot : lit `n` entrees depuis `pos` comme `FUN_142e2bfd0`.
func e108Suivre(pay []byte, pos, n int, reg *Registry, ctx ContexteDeLecture) (int, bool) {
	for range n {
		e, ok := e108Lire(pay, pos)
		if !ok {
			return pos, false
		}
		if e.arch == keyframeArchetypeNone {
			pos += e108EnTete
			continue
		}
		if e.arch >= kfArchMax {
			return pos, false
		}
		tr := WalkKeyframeFullState(pay, pos, reg, ctx)
		if tr.DesyncAt >= 0 {
			return pos, false
		}
		pos = tr.EndBit
	}
	return pos, true
}

// e108Verdict est ce que les deux regles disent d une fin de traversee `fin` contre l ancre `want`.
type e108Verdict struct {
	ecart, k int
	r108     bool
	rslot    bool
}

func e108Juger(pay []byte, fin, want, ecartSlots int, reg *Registry, ctx ContexteDeLecture) e108Verdict {
	v := e108Verdict{ecart: fin - want}
	if fin == want {
		v.r108, v.rslot = true, ecartSlots == 0
		if !v.rslot {
			pos, ok := e108Suivre(pay, fin, ecartSlots, reg, ctx)
			v.rslot = ok && pos == want
		}
		return v
	}
	if fin < want {
		v.k, v.r108 = e108Sauter(pay, fin, want)
	}
	if ecartSlots >= 0 && fin <= want {
		pos, ok := e108Suivre(pay, fin, ecartSlots, reg, ctx)
		v.rslot = ok && pos == want
	}
	return v
}

func e108ClasseEcart(ecart int) string {
	switch {
	case ecart == 0:
		return "zero"
	case ecart < 0 && (-ecart)%e108EnTete == 0:
		return "mult_-108"
	case ecart > 0:
		return "depasse"
	}
	return "autre_sous"
}

// e108Canal mesure les records bipedes (admission T1 et T2 comprise).
type e108Canal struct {
	*ri27d0Canal
	reg    *Registry
	fm     string
	g      map[string]int
	lignes []string
}

func (e *e108Canal) compter(cle string) { e.g[cle]++ }

func (e *e108Canal) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	ctx := e.fc.ContexteDeLecture()
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != keyframeBipedTI || r.Desync == lecture.CorpsNonParcouru {
			continue
		}
		e.compter("records")
		g := e.lire(p, r, ctx)
		t1, t2 := e.temoinT(&g, int(r.Desync))
		a := r.Preuve == lecture.PreuveFerme
		b := !a && g.atteint[archIndexOf(e.arch, invDeltaGrenadeCountsName)] && g.compte == 4 && len(g.gren) == 4
		if i+1 >= len(p.Records) {
			e.compter("sans_frontiere")
			e.admission(a, false, b, t1, t2)
			continue
		}
		suiv := &p.Records[i+1]
		want, fin := int(suiv.Debut), int(r.Debut+r.Bits)
		ecartSlots := int(suiv.Vie.Slot) - int(r.Vie.Slot) - 1
		var v e108Verdict
		classe := "desync"
		if r.Desync < 0 {
			v = e108Juger(p.Payload, fin, want, ecartSlots, e.reg, ctx)
			classe = e108ClasseEcart(v.ecart)
		}
		e.compter("ecart:" + classe)
		simA := r.Desync < 0 && v.r108
		e.compter(fmt.Sprintf("ecart:%s|r108=%v|rslot=%v", classe, v.r108, v.rslot))
		if classe == "mult_-108" {
			e.noterK(p, r, suiv, v, ecartSlots, fin)
		}
		e.admission(a, simA, b, t1, t2)
		e.temoin(p, r, want, ecartSlots, ctx)
	}
}

// admission compte la regle U-1 amendee avant (A ferme) et apres R108 (A ou ferme par R108).
func (e *e108Canal) admission(a, simA, b, t1, t2 bool) {
	if a {
		e.compter("fermes_avant")
	}
	if a || simA {
		e.compter("fermes_r108")
	}
	if (a || b) && t1 && t2 {
		e.compter("admis_avant")
	}
	bApres := b && !simA
	if (a || simA || bApres) && t1 && t2 {
		e.compter("admis_r108")
	}
	if simA && !a {
		e.compter(fmt.Sprintf("gagne_r108|B=%v|T1=%v|T2=%v", b, t1, t2))
	}
}

// temoin rejoue la marche decalee d un bit (temoin nomme) et la juge sous les deux regles.
func (e *e108Canal) temoin(p *lecture.Paquet, r *lecture.Record, want, ecartSlots int, ctx ContexteDeLecture) {
	ctx.Obs = nil
	tr := walkKeyframeFullState(p.Payload, int(r.Debut), e.reg, ctx, keyframeFullStateTemoin{EnTeteBits: 109})
	if tr.DesyncAt >= 0 {
		e.compter("temoin:desync")
		return
	}
	v := e108Juger(p.Payload, tr.EndBit, want, ecartSlots, e.reg, ctx)
	e.compter(fmt.Sprintf("temoin:ferme=%v|r108=%v|rslot=%v", v.ecart == 0, v.r108, v.rslot))
}

func (e *e108Canal) noterK(p *lecture.Paquet, r, suiv *lecture.Record, v e108Verdict, ecartSlots, fin int) {
	k := -v.ecart / e108EnTete
	rel := "k<ecartSlots"
	switch {
	case k == ecartSlots:
		rel = "k=ecartSlots"
	case k > ecartSlots:
		rel = "k>ecartSlots"
	}
	e.compter("k108:" + rel)
	blocs := make([]string, 0, k)
	for j := range k {
		ent, _ := e108Lire(p.Payload, fin+j*e108EnTete)
		c := ent.classe(int(r.Vie.Slot) + 1 + j)
		blocs = append(blocs, c)
		e.compter("bloc:" + c)
	}
	e.compter(fmt.Sprintf("k108:k=%d", min(k, 9)))
	e.lignes = append(e.lignes, fmt.Sprintf("K\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%v\t%v\t%s", e.court, e.fm, p.TS,
		r.Vie.Slot, suiv.Vie.Slot, k, ecartSlots, v.r108, v.rslot, strings.Join(blocs, ";")))
}

// e108TousArchetypes mesure, archetype par archetype, la fermeture avant et sous les regles, le
// temoin decale d un bit croise avec la lecture vraie, et la preuve d ancre (`PreuveDImageCle.prouve`)
// avant et apres le saut des entrees liberees (la population de [KeyframeClosure] : records bornes).
func e108TousArchetypes(t *testing.T, fc *FilmContext, reg *Registry, court, fm string) []string {
	t.Helper()
	x := map[string]int{}
	pv := fc.PreuveDImageCle()
	for p, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		ctx := fc.ContexteDeLecture()
		ctx.Obs = nil
		for i := 0; i+1 < len(p.Records); i++ {
			e108UnRecord(x, p, i, reg, ctx)
			e108Preuve(x, pv, p, i)
		}
	}
	return ri27d0Trier(court+"\t"+fm, "X", x)
}

// e108UnRecord classe un record borne : ecart, regles R108 (libre), R108s (k = ecart de slots) et
// Rslot, puis le temoin decale d un bit, croise avec la lecture vraie sous R108s.
func e108UnRecord(x map[string]int, p *lecture.Paquet, i int, reg *Registry, ctx ContexteDeLecture) {
	r, s := &p.Records[i], &p.Records[i+1]
	pre := fmt.Sprintf("%d\t", r.TI)
	es := int(s.Vie.Slot) - int(r.Vie.Slot) - 1
	x[pre+"bornes"]++
	if r.Preuve == lecture.PreuveFerme {
		x[pre+"fermes_avant"]++
		if es != 0 {
			x[pre+"fermes_avant_ecartSlots>0"]++
		}
	}
	vrai, vraiSlot := false, false
	if r.Desync < 0 {
		v := e108Juger(p.Payload, int(r.Debut+r.Bits), int(s.Debut), es, reg, ctx)
		x[pre+"ecart:"+e108ClasseEcart(v.ecart)]++
		if v.r108 {
			x[pre+"fermes_r108"]++
		}
		vrai = v.r108 && (v.ecart == 0 || v.k == es)
		if vrai {
			x[pre+"fermes_r108s"]++
		}
		if v.rslot {
			x[pre+"fermes_rslot"]++
			vraiSlot = true
		}
		if v.ecart < 0 && v.r108 && v.k != es {
			x[pre+"r108_k!=ecartSlots"]++
		}
	}
	tr := walkKeyframeFullState(p.Payload, int(r.Debut), reg, ctx, keyframeFullStateTemoin{EnTeteBits: 109})
	if tr.DesyncAt >= 0 {
		return
	}
	v := e108Juger(p.Payload, tr.EndBit, int(s.Debut), es, reg, ctx)
	tem := v.r108 && (v.ecart == 0 || v.k == es)
	if v.ecart == 0 {
		x[pre+"temoin_avant"]++
		if r.Preuve != lecture.PreuveFerme {
			x[pre+"temoin_avant_hasard"]++
		}
	}
	if v.rslot {
		x[pre+"temoin_rslot"]++
		if !vraiSlot {
			x[pre+"temoin_rslot_hasard"]++
		}
	}
	if tem {
		x[pre+"temoin_r108s"]++
		if !vrai {
			x[pre+"temoin_r108s_hasard"]++
		}
	}
}

// e108Preuve rejoue la preuve d ancre du record `i` ([PreuveDImageCle.prouve]) et la meme preuve
// apres le saut des entrees liberees qui suivent la traversee (archetype 0xffffffff, 108 bits).
func e108Preuve(x map[string]int, pv *PreuveDImageCle, p *lecture.Paquet, i int) {
	r := &p.Records[i]
	pre := fmt.Sprintf("%d\t", r.TI)
	avant := pv.prouve(p.Payload, int(r.Debut), int(r.Vie.Slot))
	apres := avant
	if !avant {
		total := len(p.Payload) * 8
		cadre := pv.ctx.Profil.Cadre
		n1 := int32(source.BitsBourres(p.Payload, int(r.Debut)+cadre.EnTeteBits, cadre.MotDeTailleBits)) //nolint:gosec // mot signe
		tr := WalkKeyframeFullState(p.Payload, int(r.Debut), pv.reg, pv.ctx)
		pos := tr.EndBit
		for n1 > 0 && tr.DesyncAt < 0 && len(tr.Comps) > 0 {
			ent, ok := e108Lire(p.Payload, pos)
			if !ok || ent.arch != keyframeArchetypeNone {
				break
			}
			pos += e108EnTete
		}
		h, ok := readKeyframeHeader(p.Payload, pos, total)
		apres = n1 > 0 && tr.DesyncAt < 0 && len(tr.Comps) > 0 && ok && h.Slot > int(r.Vie.Slot)
	}
	if avant {
		x[pre+"prouve_avant"]++
	}
	if apres {
		x[pre+"prouve_r108"]++
	}
}

func TestEcart108(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	noms := weaponv3.KnownWeaponHigh32Copie()
	known := make(map[uint32]bool, len(noms))
	for f := range noms {
		known[f] = true
	}
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		arch, err := fc.bipedArchetype()
		if err != nil {
			t.Fatalf("%s : archetype : %v", court, err)
		}
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre : %v", court, err)
		}
		vf, _ := FilmFormatVersion(fc.film)
		fm := fmt.Sprintf("f%d", vf)
		c := &ri27d0Canal{fc: fc, arch: arch, court: court, known: known, noms: noms, roles: ri27d0Roles(arch),
			classes: map[string]int{}, ech: map[string]int{}, fams: map[string]int{}, x: nouvelleExt(reg, false)}
		e := &e108Canal{ri27d0Canal: c, reg: reg, fm: fm, g: map[string]int{}}
		if err := Distribuer(fc, e); err != nil {
			t.Fatalf("%s : distribution : %v", court, err)
		}
		lignes = append(lignes, ri27d0Trier(court+"\t"+fm, "G", e.g)...)
		lignes = append(lignes, e.lignes...)
		lignes = append(lignes, e108TousArchetypes(t, fc, reg, court, fm)...)
		t.Logf("%s %s : %d bipedes, fermes %d -> %d, admis %d -> %d", court, fm, e.g["records"],
			e.g["fermes_avant"], e.g["fermes_r108"], e.g["admis_avant"], e.g["admis_r108"])
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "ecart108.tsv"), lignes)
}
