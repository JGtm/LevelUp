//go:build research

package grammar

// m3_naissance_cloture_research_test.go — LOT M3.2, SUITE DE `m3_naissance_grammaire_research_test.go` :
// la reparation H-DS32 de l etat par defaut du bipede (la derniere feuille lit son R(32) quand sa
// porte vaut 1), mesuree record par record contre l oracle, puis la FERMETURE du record NEW de
// naissance : ce qui suit sa fin (un record propre ? un temoin decale ?), la chaine de records
// qui suit, et l oracle `n2` des images-cles.
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_SUIV=1 \
//	  go test -tags=research -count=1 -v -run '^TestM3Naissance(Suivant|Chaine)$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"testing"
)

// TestM3Registre publie les composants du bipede tels que le registre DU FILM les nomme.
func TestM3Registre(t *testing.T) {
	if os.Getenv("M3_REG") == "" {
		t.Skip("M3_REG absent")
	}
	tc := t516Cadre(t)
	arch, _ := tc.reg.Archetype(BipedTypeIndex)
	for i, n := range arch.Components {
		t.Logf("   i%-2d niveau %d %s", i, arch.Level(i), n)
	}
}

// m3EtatParDefautOpt32 est `consumeBipedDefaultState` dont la DERNIERE feuille
// (`uVar10 >= 12` : FUN_14080d69c) lit son R(32) quand sa porte vaut 1 — HYPOTHESE H-DS32.
func m3EtatParDefautOpt32(br *Lecteur) {
	uVar10 := uint32(13)
	if br.ReadBit() {
		uVar10 = uint32(br.ReadBits(8))
	}
	if br.ReadBit() {
		br.ReadBits(32)
	}
	if int32(uVar10) > 10 {
		consumeGate0R(br, 5)
	}
	consumeMultiplayerPropertiesBlock(br)
	if br.ReadBit() {
		br.ReadBits(6)
	}
	br.ReadBit()
	consumeOpt32(br)
	br.ReadBits(19)
	if int32(uVar10) > 5 {
		br.ReadBit()
	}
	if int32(uVar10) >= 12 {
		consumeOpt32(br)
	}
}

// m3TraverserDS32 est `TraverseEntity` (bipede) sous H-DS32.
func m3TraverserDS32(br *Lecteur, reg *Registry) EntityTrace {
	t := EntityTrace{DesyncAt: -1}
	t.TypeIndex = uint32(br.ReadBits(6))
	m3EtatParDefautOpt32(br)
	consumeBipedDefaultStateTail(br)
	t.DefaultBits = br.BitPos()
	t.Gate = br.ReadBit()
	t.Mask = consumeMask(br)
	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt = 0
		return t
	}
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// TestM3NaissanceDS32 : sous H-DS32, i43 tombe-t-il sur l oracle, record par record ?
func TestM3NaissanceDS32(t *testing.T) {
	in := os.Getenv("M3_EXPORT_IN")
	if in == "" {
		t.Skip("M3_EXPORT_IN absent")
	}
	tc := t516Cadre(t)
	lignes := m3LireExport(t, in)
	var n, justes, sansOracle int
	ecarts := map[int]int{}
	masques := map[string]int{}
	dumps := 0
	for _, l := range lignes {
		if l.kind != "NEW" {
			continue
		}
		n++
		br := LecteurSur(l.pay)
		br.poserCadre(tc.cfg)
		br.SetBitPos(3 + tc.cfg.IDLowBits + 2)
		tr := m3TraverserDS32(br, tc.reg)
		masques[fmt.Sprint(m3Indices(tr.Mask))]++
		i43 := -1
		for _, cr := range tr.Comps {
			if cr.Index == 43 {
				i43 = cr.StartBit
				break
			}
		}
		if l.oracle <= 0 {
			sansOracle++
			continue
		}
		ecarts[l.oracle-i43]++
		if i43 == l.oracle {
			justes++
		} else if dumps < 8 {
			dumps++
			t.Logf("   ECART slot %d oracle %d i43 %d desync %d masque %v", l.slot, l.oracle, i43, tr.DesyncAt,
				m3Indices(tr.Mask))
			t.Logf("        %s", m3Largeurs(tr, 0))
		}
	}
	t.Logf("== H-DS32 : %d records NEW ; i43 SUR l oracle %d ; sans oracle %d ; ecarts %s", n, justes,
		sansOracle, m3Tri(ecarts))
	for m, c := range masques {
		t.Logf("   masque %s : %d", m, c)
	}
}

// TestM3ImageCleDS32 rejoue l etat complet des records d image-cle ti=35 sous les DEUX lectures
// de l etat par defaut (production / H-DS32) et publie : la porte lue, n1, et si i43 lit une
// famille du catalogue.
func TestM3ImageCleDS32(t *testing.T) {
	doc := os.Getenv("M3_DOC")
	if doc == "" || os.Getenv("M3_KF") == "" {
		t.Skip("M3_DOC / M3_KF absents")
	}
	tc := t516Cadre(t)
	cat := m3Catalogue(t, doc)
	ctx := tc.fc.ContexteDeLecture()
	arch, _ := tc.reg.Archetype(BipedTypeIndex)
	type cpt struct{ n, cat, desync int }
	res := map[string]*cpt{"PROD": {}, "DS32": {}}
	portes, n1s := map[uint64]int{}, map[uint64]int{}
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI != BipedTypeIndex {
					continue
				}
				for _, v := range []string{"PROD", "DS32"} {
					br := LecteurSur(pay)
					br.PoserContexte(ctx)
					br.SetBitPos(r.Bit + br.cadre().EnTeteBits)
					n1 := br.ReadBits(32)
					if v == "DS32" {
						n1s[n1]++
					}
					debut := br.BitPos()
					if v == "PROD" {
						consumeBipedDefaultState(br)
					} else {
						m3EtatParDefautOpt32(br)
						// la porte de la derniere feuille : relue a la position ou la lecture
						// de production s arrete
						b2 := LecteurSur(pay)
						b2.PoserContexte(ctx)
						b2.SetBitPos(debut)
						consumeBipedDefaultState(b2)
						portes[PeekBits(pay, b2.BitPos()-1, 1)]++
					}
					consumeBipedDefaultStateTail(br)
					br.ReadBits(32) // n2
					tr := EntityTrace{DesyncAt: -1, TypeIndex: BipedTypeIndex, Mask: ^uint64(0)}
					traverseComponentLoop(br, arch, &tr)
					c := res[v]
					c.n++
					if tr.DesyncAt >= 0 && tr.DesyncAt < 43 {
						c.desync++
					}
					for _, cr := range tr.Comps {
						if cr.Index == 43 && PeekBits(pay, cr.StartBit, 1) == 1 &&
							cat[uint32(PeekBits(pay, cr.StartBit+1, 32))] {
							c.cat++
						}
					}
				}
			}
		}
	}
	for _, v := range []string{"PROD", "DS32"} {
		t.Logf("== image-cle ti=35 %s : %d records ; i43 au catalogue %d ; desync avant i43 %d", v,
			res[v].n, res[v].cat, res[v].desync)
	}
	t.Logf("   porte de la derniere feuille (bit precedant la fin de la lecture de production) : %v ; n1 %v",
		portes, n1s)
}

// m3MarcherVueB rejoue la vue B depuis `debut` avec une traversee de record NEW choisie : c est
// `t519Marcher`, mais le record NEW de bipede y est lu sous H-DS32 quand `ds32`.
func m3Fermeture(pay []byte, w *World, cfg FrameConfig, debut int) (fin int, hitEnd, ferme bool, recs int) {
	mar := t519Marcher(pay, w, cfg, debut)
	reste := len(pay)*8 - mar.m.FinVueC
	return mar.m.FinVueB, mar.m.HitEndB, mar.m.HitEndB && reste >= 0 && reste <= m5116GateOctet &&
		c514ResteNul(pay, mar.m.FinVueC), len(mar.recs)
}

// TestM3NaissanceCloture : le record NEW de naissance (lu sous H-DS32) se ferme-t-il sur le record
// suivant de la vue B ? Trois temoins : la fin du record egale le debut localise par la signature
// du slot 123 ; la vue B relancee depuis la fin du record se ferme au terminateur et le paquet au
// bit ; la vue B relancee depuis l EN-TETE du record NEW (sous la grammaire corrigee) aussi.
func TestM3NaissanceCloture(t *testing.T) {
	if os.Getenv("M3_CLOT") == "" {
		t.Skip("M3_CLOT absent")
	}
	tc := t516Cadre(t)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	type cle struct{ ch, pk int }
	parPaquet := map[cle][]BipedCreation{}
	for _, c := range cres {
		parPaquet[cle{c.Chunk, c.PacketIndex}] = append(parPaquet[cle{c.Chunk, c.PacketIndex}], c)
	}
	var n, propres, finSig, liste, fermeApres, fermeDepuisNew, fermeSig, desync, dumpsClot int
	ecartsSig := map[int]int{}
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		w.PoserChunkCourant(ch)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			nes := parPaquet[cle{ch, pk.Index}]
			if len(nes) == 0 {
				continue
			}
			pay := pk.Payload(data)
			_, aListe := PacketHeadEventType(pay)
			sig := -1
			if aListe {
				liste++
				sig = marchLocate(pay, w, tc.cfg)
			}
			for _, c := range nes {
				n++
				br := LecteurSur(pay)
				br.poserCadre(tc.cfg)
				br.SetBitPos(c.BitPos + 3 + tc.cfg.IDLowBits + 2)
				tr := m3TraverserDS32(br, tc.reg)
				if tr.DesyncAt != -1 {
					desync++
					continue
				}
				propres++
				if sig == tr.EndBit {
					finSig++
				}
				if sig >= 0 && dumpsClot < 12 {
					dumpsClot++
					t.Logf("   slot %d hdr %d fin %d sig %d (ecart %d) masque %v : %s", c.Slot, c.BitPos,
						tr.EndBit, sig, sig-tr.EndBit, m3Indices(tr.Mask), m3Largeurs(tr, c.BitPos))
				}
				if sig >= 0 {
					ecartsSig[sig-tr.EndBit]++
				}
				snap := w.Snapshot()
				if _, _, f, _ := m3Fermeture(pay, w, tc.cfg, tr.EndBit); f {
					fermeApres++
				}
				w.Restore(snap)
				if sig >= 0 {
					if _, _, f, _ := m3Fermeture(pay, w, tc.cfg, sig); f {
						fermeSig++
					}
					w.Restore(snap)
				}
				_ = fermeDepuisNew
			}
		}
	}
	t.Logf("== cloture : %d naissances (%d en paquet a liste) ; traversee propre %d (desync %d) ; fin du "+
		"NEW = debut signature 123 : %d ; vue B relancee a la fin du NEW fermee au bit : %d ; vue B depuis la "+
		"signature fermee : %d", n, liste, propres, desync, finSig, fermeApres, fermeSig)
	t.Logf("   ecarts signature - fin du NEW : %s", m3Tri(ecartsSig))
}

// m3SuivantPropre classe ce qui suit `p` : un record DELTA propre sur un slot lie, un NEW
// d archetype valide dont le corps se traverse, un DEL, le terminateur, ou rien de lisible.
func m3SuivantPropre(pay []byte, p int, w *World, cfg FrameConfig) string {
	if p+3 > len(pay)*8 {
		return "hors"
	}
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.SetBitPos(p)
	typ := readRecordType(br)
	switch typ {
	case recEnd:
		return "fin"
	case recDelta:
		if _, _, ok := TryDeltaAt(pay, p, w, cfg); ok {
			return "delta"
		}
		return "delta-sale"
	case recNew:
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		ti := uint32(PeekBits(pay, br.BitPos(), 6))
		if lie, ok := w.ArchetypeForSlot(id & 0x3fffffff); ok && lie == ti {
			return "new-lie"
		}
		if ta, _, ok := w.anticipee.ArchetypeApres(id, w.chunkCourant); ok && ta == ti {
			return "new-anticipe"
		}
		return "new-inconnu"
	case recDel:
		return "del"
	}
	return "?"
}

// TestM3NaissanceSuivant : que trouve-t-on a la fin du record NEW corrige, et a un temoin decale ?
func TestM3NaissanceSuivant(t *testing.T) {
	if os.Getenv("M3_SUIV") == "" {
		t.Skip("M3_SUIV absent")
	}
	tc := t516Cadre(t)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	type cle struct{ ch, pk int }
	parPaquet := map[cle][]BipedCreation{}
	for _, c := range cres {
		parPaquet[cle{c.Chunk, c.PacketIndex}] = append(parPaquet[cle{c.Chunk, c.PacketIndex}], c)
	}
	fins, temoins := map[string]int{}, map[string]int{}
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		w.PoserChunkCourant(ch)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			for _, c := range parPaquet[cle{ch, pk.Index}] {
				pay := pk.Payload(data)
				br := LecteurSur(pay)
				br.poserCadre(tc.cfg)
				br.SetBitPos(c.BitPos + 3 + tc.cfg.IDLowBits + 2)
				tr := m3TraverserDS32(br, tc.reg)
				fins[m3SuivantPropre(pay, tr.EndBit, w, tc.cfg)]++
				for _, d := range []int{-7, -3, -1, 1, 3, 7} {
					temoins[m3SuivantPropre(pay, tr.EndBit+d, w, tc.cfg)]++
				}
			}
		}
	}
	t.Logf("== suivant a la fin du NEW corrige : %v ; temoin (fin +-1,3,7) : %v", fins, temoins)
}

// m3Chaine suit les records NEW propres depuis `p` (au plus `max`) et rend la classe du premier
// record qui n est pas un NEW propre, et le nombre de NEW traverses.
func m3Chaine(pay []byte, p int, w *World, cfg FrameConfig, max int) (string, int) {
	for k := 0; k <= max; k++ {
		c := m3SuivantPropre(pay, p, w, cfg)
		if c != "new-lie" && c != "new-anticipe" && c != "new-inconnu" {
			return c, k
		}
		br := LecteurSur(pay)
		br.poserCadre(cfg)
		br.SetBitPos(p)
		readRecordType(br)
		readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		// un NEW de bipede se traverse sous H-DS32, les autres par la production
		if PeekBits(pay, br.BitPos(), 6) == BipedTypeIndex {
			tr := m3TraverserDS32(br, w.Reg)
			p = tr.EndBit
		} else {
			tr := TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			p = tr.EndBit
		}
	}
	return "budget", max
}

// TestM3NaissanceChaine : la chaine de records NEW qui suit la naissance atteint-elle un delta
// propre ? Et au temoin decale ?
func TestM3NaissanceChaine(t *testing.T) {
	if os.Getenv("M3_SUIV") == "" {
		t.Skip("M3_SUIV absent")
	}
	tc := t516Cadre(t)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	type cle struct{ ch, pk int }
	parPaquet := map[cle][]BipedCreation{}
	for _, c := range cres {
		parPaquet[cle{c.Chunk, c.PacketIndex}] = append(parPaquet[cle{c.Chunk, c.PacketIndex}], c)
	}
	fins, temoins, hops := map[string]int{}, map[string]int{}, map[int]int{}
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		w.PoserChunkCourant(ch)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			for _, c := range parPaquet[cle{ch, pk.Index}] {
				pay := pk.Payload(data)
				br := LecteurSur(pay)
				br.poserCadre(tc.cfg)
				br.SetBitPos(c.BitPos + 3 + tc.cfg.IDLowBits + 2)
				tr := m3TraverserDS32(br, tc.reg)
				cl, k := m3Chaine(pay, tr.EndBit, w, tc.cfg, 4)
				fins[cl]++
				hops[k]++
				for _, d := range []int{-7, -3, -1, 1, 3, 7} {
					cl2, _ := m3Chaine(pay, tr.EndBit+d, w, tc.cfg, 4)
					temoins[cl2]++
				}
			}
		}
	}
	t.Logf("== chaine apres la naissance : %v (NEW traverses %v) ; temoin decale : %v", fins, hops, temoins)
}
