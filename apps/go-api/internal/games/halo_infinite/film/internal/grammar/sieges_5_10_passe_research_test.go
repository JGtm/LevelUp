//go:build research

package grammar

// sieges_5_10_passe_research_test.go — LA PASSE PARTAGEE DU LOT 5.10 : `i10 object-parent-state`
// et `i14 object-dissolver` lus sur la trame, avec la carte installee et l oracle de contenu.
//
// # CE QUE L ECRIVAIN DIT, ET IL LE DIT AVANT LA MESURE
//
// `FUN_140c1e4d0` (relu en lecture seule le 2026-09-21, image base 140000000) ecrit ses champs
// dans le composant a `lVar2 = *(param_3 + 0x10)` :
//
//	branche ATTACHEE (porte R(1) = 1)
//	  +0x274  FUN_1406d3140(categorie 1)   identifiant a largeur variable  -> `Quant16`
//	  +0x278  R(16) -> `Word16`   +0x27a  R(16) derriere une porte, sentinelle 0xffff -> `Opt16`
//	  +0x27c  R(1) -> `FlagA`     +0x27d  R(1) -> `FlagB`
//	  +0x280  FUN_140c1e924(..., 0x10)     le triplet 3 x R(16) -> `Mtx`, puis FUN_140c1e79c
//	branche LIBRE (porte R(1) = 0)
//	  +0x274 = 0xffffffff · +0x278 = +0x27a = 0xffff · +0x27c = 0   LES SENTINELLES
//	  si param_4 < 2 : +0x2a4 FUN_1408f0ac4 (identifiant) puis +0x2ac R(11) SIGNE
//	queue COMMUNE aux deux branches
//	  +0x3a0  R(1) de signe ; si 1 : R(6), sinon la SENTINELLE 0xffff -> `TailSign` / `Tail6`
//	  +0x3a4  R(1) -> `TailBit`   ·   +0x3a2 / +0x3a3  R(3) -> `Tail3`
//
// DEUX CHAMPS SEULEMENT PEUVENT PORTER UN EMBARQUEMENT, et ce sont ceux que les SENTINELLES
// designent : `Quant16` (+0x274, efface a 0xffffffff quand l objet est LIBRE — c est un HANDLE)
// et `Tail6` (+0x3a0, un entier de SIX bits a sentinelle, la meme largeur que le champ siege de
// l evenement d embarquement, `vehicleSeatBits = 6`). Le reste est une pose relative ou des
// drapeaux.
//
// LA PASSE LIT LES DEUX CHEMINS. Le chemin DELTA ne porte que ce qui CHANGE : une lecture d `i10`
// y est une TRANSITION (on monte, on descend). Le chemin IMAGE-CLE porte l ETAT : qui est a bord
// au moment de l image-cle. Les deux se completent, et aucune conclusion sur la couverture ne
// tient sans les deux.
//
// LES LARGEURS D AXE DE LA CARTE SONT INSTALLEES (piege 1 de la passation 5.7 : sans
// `PoserLargeursObjetDuMondeDepuisDecoupage` la position lit cinq bits de trop et tout ce qui
// suit est du bruit) et l oracle de contenu garde la conclusion.
//
// ENV : `SIEGE510_FILM` (repertoire du film), `SIEGE510_CARTE`, `SIEGE510_BORNES`.

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// s510Parent est UNE lecture d `i10` rattachee a son record.
type s510Parent struct {
	ts       uint64
	slot     uint32
	ti       uint32
	imageCle bool
	st       ObjectParentState
}

// s510Diss est UNE lecture d `i14` NON NEUTRE rattachee a son record.
type s510Diss struct {
	ts       uint64
	slot     uint32
	ti       uint32
	imageCle bool
	val      ObjectDissolver
}

// s510Del est UN record de SUPPRESSION (`recDel`, type 2) : le film retire une entite.
type s510Del struct {
	ts   uint64
	slot uint32
	gen  uint32
	ti   uint32
}

// s510Rec porte tout ce que la passe releve.
type s510Rec struct {
	parents []s510Parent
	diss    []s510Diss
	dels    []s510Del
	// arch memorise le dernier archetype VU pour un slot. Il ne s efface JAMAIS : un record de
	// suppression ne porte pas d archetype, et le monde, lui, delie le slot au meme instant.
	arch map[uint32]uint32
	// t0 est l horodatage du premier paquet : toutes les durees publiees en partent.
	t0 uint64
	// w est le monde de fin de passe : il resout un slot en archetype.
	w *World
	// fc est le contexte du film : il sert aux recensements d images-cles.
	fc *FilmContext
	// imagesCles porte les instants des images-cles marchees.
	imagesCles []uint64
	// compteurs de marche et etalon de contenu.
	paquets, pleins, pleinsTotal, vues, ti35, ti40, desync int
	kfRecords, kfDesync                                    int
	i14Lectures, i14NonNeutres                             int
	fautifs                                                map[int]int
	etalon                                                 map[int]int
}

// s510Passe ouvre le film sous sa carte, marche les paquets delta ET les images-cles, et rend
// les lectures d `i10` / `i14`. Le booleen rendu est l ORACLE : faux = trame non cadree.
func s510Passe(t *testing.T) (*s510Rec, bool) {
	t.Helper()
	dir := os.Getenv("SIEGE510_FILM")
	if dir == "" {
		t.Skip("SIEGE510_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := s510Contexte(t, film)
	reg, errR := fc.Registry()
	if errR != nil {
		t.Fatalf("registre : %v", errR)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	rec := &s510Rec{fautifs: map[int]int{}, etalon: map[int]int{}, arch: map[uint32]uint32{}}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		s510LierArchetypes(rec, data, pks)
		for _, pk := range pks {
			s510ImageCle(pk, data, reg, fc, rec)
			s510Paquet(pk, data, w, cfg, rec)
		}
	}
	rec.w, rec.fc = w, fc
	sort.SliceStable(rec.parents, func(i, j int) bool { return rec.parents[i].ts < rec.parents[j].ts })
	sort.SliceStable(rec.diss, func(i, j int) bool { return rec.diss[i].ts < rec.diss[j].ts })
	sort.SliceStable(rec.dels, func(i, j int) bool { return rec.dels[i].ts < rec.dels[j].ts })
	return rec, s510Oracle(t, rec)
}

// s510LierArchetypes seme le memo d archetypes avec ce que les IMAGES-CLES declarent. Sans lui,
// un slot supprime sans avoir jamais ete decode dans un paquet localise serait rendu « archetype
// inconnu » — et la suppression d un vehicule passerait pour celle d une entite anonyme.
func s510LierArchetypes(rec *s510Rec, data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			rec.arch[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // valeurs de registre
		}
	}
}

// s510Contexte pose la carte : meme geste que 5.3.4, 5.7 et 5.9.
func s510Contexte(t *testing.T, film *source.Film) *FilmContext {
	t.Helper()
	os.Setenv("MOUV534_CARTE", os.Getenv("SIEGE510_CARTE"))
	os.Setenv("MOUV534_BORNES", os.Getenv("SIEGE510_BORNES"))
	return m534Contexte(t, film)
}

// s510Paquet traite UN paquet delta : localisation stricte puis marche a trois vues.
func s510Paquet(pk FilmPacket, data []byte, w *World, cfg FrameConfig, rec *s510Rec) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := 2
	if _, present := PacketHeadEventType(pay); present {
		rec.pleinsTotal++
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			return
		}
		rec.pleins++
	}
	rec.paquets++
	if rec.t0 == 0 {
		rec.t0 = pk.TimestampUS
	}
	recs, n := DecodeFrameViews(pay, w, cfg, 3, debut)
	rec.vues += n
	for _, r := range recs {
		s510Record(r, pk.TimestampUS, false, rec)
	}
}

// s510ImageCle marche la TABLE d une image-cle : l etat, par opposition aux transitions du
// chemin delta. Les records desynchronises sont comptes, jamais lus.
func s510ImageCle(pk FilmPacket, data []byte, reg *Registry, fc *FilmContext, rec *s510Rec) {
	if pk.Type != PacketTypeKeyframe {
		return
	}
	if rec.t0 == 0 {
		rec.t0 = pk.TimestampUS
	}
	rec.imagesCles = append(rec.imagesCles, pk.TimestampUS)
	recs, _ := WalkKeyframeRecords(pk.Payload(data), reg, fc.ContexteDeLecture())
	for _, r := range recs {
		rec.kfRecords++
		if r.DesyncAt >= 0 {
			rec.kfDesync++
			continue
		}
		s510Record(FrameRecord{
			Slot: uint32(r.Slot), TypeIndex: uint32(r.TI), //nolint:gosec // valeurs de registre
			DesyncAt: -1, Trace: EntityTrace{Mask: r.Mask, Comps: r.Comps},
		}, pk.TimestampUS, true, rec)
	}
}

// s510Record ventile UN record : etalon de contenu sur la bande bipede, puis les captures.
func s510Record(r FrameRecord, ts uint64, imageCle bool, rec *s510Rec) {
	if r.TypeIndex != 0 {
		rec.arch[r.Slot] = r.TypeIndex
	}
	if r.Type == recDel {
		rec.dels = append(rec.dels, s510Del{
			ts: ts, slot: r.Slot, gen: r.ID >> 30, ti: rec.arch[r.Slot],
		})
		return
	}
	switch r.TypeIndex {
	case BipedTypeIndex:
		if imageCle {
			break
		}
		rec.ti35++
		if r.DesyncAt >= 0 {
			rec.desync++
			rec.fautifs[r.DesyncAt]++
		}
		for _, b := range []int{0, 1, 10, 21, 25} {
			if r.Trace.Mask&(1<<uint(b)) != 0 {
				rec.etalon[b]++
			}
		}
	case VehicleTypeIndex:
		rec.ti40++
	}
	if r.DesyncAt != -1 {
		return
	}
	for _, comp := range r.Trace.Comps {
		s510Comp(comp, r, ts, imageCle, rec)
	}
}

// s510Comp range UNE capture : la parente i10, ou la dissolution i14 quand elle n est pas neutre.
func s510Comp(comp CompResult, r FrameRecord, ts uint64, imageCle bool, rec *s510Rec) {
	if st, ok := comp.ParentOf(); ok {
		rec.parents = append(rec.parents, s510Parent{
			ts: ts, slot: r.Slot, ti: r.TypeIndex, imageCle: imageCle, st: st,
		})
		return
	}
	d, ok := comp.DissolverOf()
	if !ok {
		return
	}
	rec.i14Lectures++
	if d.Etat == objectDissolverEtatNeutre {
		return
	}
	rec.i14NonNeutres++
	rec.diss = append(rec.diss, s510Diss{
		ts: ts, slot: r.Slot, ti: r.TypeIndex, imageCle: imageCle, val: d,
	})
}

// s510Oracle publie l etalon de contenu et REFUSE de conclure sous 50 % d `i21`.
func s510Oracle(t *testing.T, rec *s510Rec) bool {
	t.Helper()
	t.Logf("MARCHE (3 vues) : %d paquets delta, dont %d a liste pleine sur %d (%.1f %% localises) · %d vues · "+
		"%d records ti=35 (dont %d desynchronises, %.2f %%) · %d records ti=40",
		rec.paquets, rec.pleins, rec.pleinsTotal, m533bPart(rec.pleins, rec.pleinsTotal),
		rec.vues, rec.ti35, rec.desync,
		m533bPart(rec.desync, rec.ti35), rec.ti40)
	t.Logf("  COMPOSANT FAUTIF : %s", m533cTable(rec.fautifs))
	t.Logf("IMAGES-CLES : %d marchees · %d records, dont %d desynchronises (%.2f %%)",
		len(rec.imagesCles), rec.kfRecords, rec.kfDesync, m533bPart(rec.kfDesync, rec.kfRecords))
	t.Logf("ETALON sur ti=35 (delta) : i0 %.1f %% · i1 %.1f %% · i21 %.1f %% · i25 %.1f %% · i10 %.1f %%",
		m533bPart(rec.etalon[0], rec.ti35), m533bPart(rec.etalon[1], rec.ti35),
		m533bPart(rec.etalon[21], rec.ti35), m533bPart(rec.etalon[25], rec.ti35),
		m533bPart(rec.etalon[10], rec.ti35))
	t.Logf("CAPTURES : %d lectures i10 · %d lectures i14 dont %d NON NEUTRES",
		len(rec.parents), rec.i14Lectures, rec.i14NonNeutres)
	if m533bPart(rec.etalon[21], rec.ti35) < 50 {
		t.Logf("ETALON REFUSE : `i21` sous 50 %% — la trame n est pas cadree. Aucune conclusion.")
		return false
	}
	t.Logf("ETALON TENU : les tableaux qui suivent portent sur une trame cadree.")
	return true
}

// ms rend un instant en millisecondes depuis le premier paquet marche.
func (r *s510Rec) ms(ts uint64) int64 {
	if ts < r.t0 {
		return 0
	}
	return int64((ts - r.t0) / 1000)
}

// archetype rend l archetype du slot, ou 0xffffffff quand le monde ne le lie pas.
func (r *s510Rec) archetype(slot uint32) uint32 {
	if ti, ok := r.w.ArchetypeForSlot(slot); ok {
		return ti
	}
	return 0xffffffff
}
