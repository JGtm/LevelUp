//go:build research

package grammar

// c2_decor_ti40_research_test.go — SONDE C2 (campagne retours rejeu, 2026-09-23) : un vehicule
// de DECOR (jamais simule, jamais pilotable) se distingue-t-il DES SA NAISSANCE, dans la
// grammaire, d un vehicule gare mais simule ?
//
// # CE QUE L INSTRUMENT RELEVE, PAR VIE `(slot, gen)` DE `ti=40`
//
//	KF  pour chaque record `ti=40` de chaque IMAGE-CLE (etat complet, lecture de production
//	    `WalkKeyframeFullState`) : les trois champs d en-tete qui suivent `[eid][archetype]`
//	    (R(32) -> entree+0x0c, R(4) -> +0x08, R(8) -> +0x09), puis l etat par defaut de
//	    `FUN_1410a5a74` SEGMENT PAR SEGMENT (version, les onze sous-champs du bloc MPP, porte
//	    bVar14 + feuille 4, R(19), porte cVar3 + liste), puis les BITS BRUTS de chaque composant
//	    porte (i0 a i29, arret au premier composant non porte), l index de desynchronisation et
//	    l emprise du record.
//	DL  pour chaque record `ti=40` des paquets DELTA, par la MARCHE de production
//	    (`marchRecordsOf`, meme cadre que `ScanMarchFacts`) : type de record, masque, desync.
//
// Les segments de l etat par defaut sont relus avec les FONCTIONS DE PRODUCTION (aucune largeur
// recopiee) ; le controle de coherence est que le curseur apres `n2` tombe EXACTEMENT sur le
// premier composant de la marche de production (colonne `coh`).
//
// LECTURE SEULE, UN FILM PAR INVOCATION, sortie TSV dans C2_OUT (hors depot) :
//
//	C2_FILM=<data>/cache/film_chunks/f0220a96 C2_CARTE=starboard C2_OUT=<scratch>/f0220a96.tsv \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestC2DecorTi40$' -v -timeout 60m

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestC2DecorTi40(t *testing.T) {
	dir, carte, out := os.Getenv("C2_FILM"), os.Getenv("C2_CARTE"), os.Getenv("C2_OUT")
	if dir == "" || carte == "" || out == "" {
		t.Skip("instrument de mesure : C2_FILM, C2_CARTE et C2_OUT requis")
	}
	fc := bv14Contexte(t, dir, carte)
	if restore, err := InstallFilmFormatMPP(fc); err == nil {
		defer restore()
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatalf("sortie : %v", err)
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriter(f)
	defer func() { _ = w.Flush() }()
	nkf, incoh := c2Imagescles(t, fc, w)
	ndl := c2Deltas(t, fc, w)
	t.Logf("%s : MPP %s | records ti=40 d image-cle = %d (incoherents %d) | records ti=40 delta = %d",
		filepath.Base(dir), fc.ProfilDeBalayage().MPP, nkf, incoh, ndl)
}

// c2Imagescles releve chaque record `ti=40` de chaque image-cle.
func c2Imagescles(t *testing.T, fc *FilmContext, w *bufio.Writer) (n, incoh int) {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	ctx := fc.ContexteDeLecture()
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, b := range keyframeBornesToutes(pay) {
				if b.TI != VehicleTypeIndex {
					continue
				}
				n++
				champs, coh := c2LireRecord(pay, b, reg, ctx)
				if !coh {
					incoh++
				}
				gen := kfReadBits(pay, b.Bit, 32) >> 30
				for _, c := range champs {
					fmt.Fprintf(w, "KF\t%d\t%d\t%d\t%s\t%s\n", b.Slot, gen, pk.TimestampUS, c[0], c[1])
				}
			}
		}
	}
	return n, incoh
}

// c2Brut rend les bits [a, b) du payload : valeur hexa pour <= 64 bits, sinon longueur + FNV.
func c2Brut(pay []byte, a, b int) string {
	n := b - a
	if n <= 0 {
		return "0:-"
	}
	if n <= 64 {
		return strconv.Itoa(n) + ":" + strconv.FormatUint(kfReadBits(pay, a, n), 16)
	}
	h := fnv.New64a()
	var sb strings.Builder
	for i := a; i < b; i++ {
		sb.WriteByte(byte('0' + kfReadBits(pay, i, 1)))
	}
	_, _ = h.Write([]byte(sb.String()))
	return strconv.Itoa(n) + ":#" + strconv.FormatUint(h.Sum64()&0xffffffff, 16)
}

// c2LireRecord rend les champs nommes d UN record `ti=40` d image-cle, et la coherence entre la
// relecture segmentee de l etat par defaut et la marche de production.
func c2LireRecord(pay []byte, b keyframeBorne, reg *Registry, ctx ContexteDeLecture) ([][2]string, bool) {
	var out [][2]string
	add := func(k, v string) { out = append(out, [2]string{k, v}) }
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	hdr := br.cadre().EnTeteBits
	add("hdr.w0c", c2Brut(pay, b.Bit+64, b.Bit+96))
	add("hdr.r4", c2Brut(pay, b.Bit+96, b.Bit+100))
	add("hdr.r8", c2Brut(pay, b.Bit+100, b.Bit+hdr))
	if b.Want >= 0 {
		add("len", strconv.Itoa(b.Want-b.Bit))
	}
	br.SetBitPos(b.Bit + hdr)
	mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil
	seg := func(nom string, f func()) {
		a := br.BitPos()
		f()
		add(nom, c2Brut(pay, a, br.BitPos()))
	}
	var n1 int32
	seg("n1", func() { n1 = int32(br.ReadBits(mot)) }) //nolint:gosec // comparaison signee
	if n1 > 0 {
		c2EtatParDefaut(br, seg)
		if br.p.Grammaire.ControleDeCorruption {
			seg("ds.ctrl", func() { br.ReadBits(mot) })
		}
	}
	seg("n2", func() { br.ReadBits(mot) })
	apres := br.BitPos()
	tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
	coh := len(tr.Comps) == 0 || tr.Comps[0].StartBit == apres
	add("desync", strconv.Itoa(tr.DesyncAt))
	for k, c := range tr.Comps {
		if !c.Ported {
			break
		}
		fin := tr.EndBit
		if k+1 < len(tr.Comps) {
			fin = tr.Comps[k+1].StartBit
		}
		add(fmt.Sprintf("i%02d", c.Index), c2Brut(pay, c.StartBit, fin))
	}
	return out, coh
}

// c2EtatParDefaut relit `consumeDefaultStateTI40` (FUN_1410a5a74) segment par segment, avec les
// fonctions de production, le bloc MPP (FUN_14080cfe8) eclate en ses sous-champs.
func c2EtatParDefaut(br *Lecteur, seg func(string, func())) {
	seg("ds.V", func() { consumeVersionPrefix(br) })
	seg("mpp.lead", func() { br.ReadBits(uint(br.mppWidths().Lead)) })
	seg("mpp.w32", func() { br.ReadBits(32) })
	seg("mpp.variant", func() {
		if !br.ReadBit() {
			br.ReadBits(32)
		}
	})
	seg("mpp.r18", func() {
		if br.ReadBit() {
			br.ReadBits(18)
		}
	})
	seg("mpp.d524", func() { consumeMppD524(br) })
	seg("mpp.r2", func() { br.ReadBits(2) })
	seg("mpp.idx", func() { br.ReadBits(uint(br.mppWidths().Index)) })
	seg("mpp.liste", func() {
		if c := br.ReadBits(3); c <= 4 {
			for i := uint64(0); i < c; i++ {
				br.ReadBits(5)
				consumeOpt32(br)
			}
		}
	})
	seg("mpp.d4d0", func() { consumeMppD4d0(br) })
	seg("mpp.queue", func() {
		if br.ReadBit() {
			br.ReadBits(32)
			consumeOpt32(br)
			br.ReadBits(14)
		}
	})
	seg("ds.bv14", func() {
		if br.ReadBit() {
			consumeVehicleMediaFrame(br)
		}
	})
	seg("ds.r19", func() { br.ReadBits(19) })
	seg("ds.cvar3", func() {
		if !br.ReadBit() {
			consumeOpt32(br)
			return
		}
		n := br.ReadBits(2)
		for i := uint64(0); i < n; i++ {
			consumeOpt32(br)
		}
	})
}

// c2Deltas deroule la MARCHE de production et releve chaque record `ti=40`.
func c2Deltas(t *testing.T, fc *FilmContext, w *bufio.Writer) int {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	kfs, deltas := marchPacketsOf(fc)
	cfg, _, _, _ := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	tl := newMarchTimeline(reg, kfs)
	n := 0
	for _, d := range deltas {
		wd := tl.advanceTo(d.timestampUS)
		start, _, ok := marchStartOf(d.payload, wd, cfg)
		if !ok {
			continue
		}
		for _, r := range marchRecordsOf(d.payload, wd, cfg, start) {
			if r.TypeIndex != uint32(VehicleTypeIndex) {
				continue
			}
			n++
			fmt.Fprintf(w, "DL\t%d\t%d\t%d\ttype=%d\tmask=%s\tdesync=%d\n", r.Slot, r.ID>>30,
				d.timestampUS, r.Type, c2Masque(r.Trace.Mask), r.DesyncAt)
		}
	}
	return n
}

// c2Masque rend la liste des index d un masque de composants.
func c2Masque(m uint64) string {
	if m == ^uint64(0) {
		return "plein"
	}
	var s []string
	for i := 0; i < 64; i++ {
		if m&(1<<uint(i)) != 0 {
			s = append(s, strconv.Itoa(i))
		}
	}
	return strings.Join(s, ",")
}
