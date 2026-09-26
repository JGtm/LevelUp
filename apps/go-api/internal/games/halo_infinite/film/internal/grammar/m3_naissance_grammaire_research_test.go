//go:build research

package grammar

// m3_naissance_grammaire_research_test.go — LOT M3.2 (b), INSTRUMENT DE MESURE : la grammaire
// du corps d un record NEW de bipede (ti=35), composants i1..i42, au bit pres.
//
// ORACLE (mesure seulement, JAMAIS une lecture de production) : la position de la porte du
// premier emplacement `weapon-state-type-info` localisee par catalogue (familles nommees par
// le document du match), comme la sonde P3 (§4, §5). L instrument essaie des VARIANTES de la
// traversee et compte, pour chacune, les records ou i43 tombe EXACTEMENT sur l oracle.
//
//	MOUV511_FILM=<depot>/data/cache/film_chunks/<id> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	M3_DOC=<depot>/data/cache/replays/halo_infinite/<id>.json \
//	  go test -tags=research -count=1 -v -run '^TestM3NaissanceGrammaire$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// m3Catalogue rend les familles (moities hautes) que le document nomme.
func m3Catalogue(t *testing.T, chemin string) map[uint32]bool {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("document : %v", err)
	}
	var d struct {
		WeaponLabels map[string]json.RawMessage `json:"weaponLabels"`
	}
	if err := json.Unmarshal(blob, &d); err != nil {
		t.Fatalf("document : %v", err)
	}
	out := map[uint32]bool{}
	for k := range d.WeaponLabels {
		s := strings.TrimPrefix(strings.ToLower(k), "0x")
		if len(s) > 8 {
			s = s[:8]
		}
		if v, err := strconv.ParseUint(s, 16, 32); err == nil && len(k) == 10 {
			out[uint32(v)] = true
		}
	}
	return out
}

// m3OracleArmes rend la porte du 1er emplacement localisee par catalogue dans [hdr+1000,
// hdr+1700) (score = familles du catalogue lues en enchainant 3 emplacements), -1 sinon.
func m3OracleArmes(pay []byte, hdr int, cfg FrameConfig, cat map[uint32]bool) (int, []uint32) {
	best, score := -1, 0
	var fams []uint32
	for o := hdr + 1000; o < hdr+1700 && o+33 <= len(pay)*8; o++ {
		br := LecteurSur(pay)
		br.poserCadre(cfg)
		br.SetBitPos(o)
		dedans, ok := 0, true
		var lus []uint32
		for i := 0; i < 3 && ok; i++ {
			debut := br.BitPos()
			br2 := LecteurSur(pay)
			br2.SetBitPos(debut)
			if br2.ReadBit() {
				hi := uint32(br2.ReadBits(32))
				switch {
				case cat[hi]:
					dedans++
					lus = append(lus, hi)
				case i == 0:
					ok = false
				default:
					lus = append(lus, hi)
				}
			} else if i == 0 {
				ok = false
			}
			consumeWeaponStateTypeInfoVariant(br)
			if br.BitPos() > len(pay)*8 {
				ok = false
			}
		}
		if ok && dedans > score {
			best, score, fams = o, dedans, lus
		}
	}
	return best, fams
}

// m3Variante est une variante de la traversee du record NEW.
type m3Variante struct {
	nom   string
	force uint64 // bits forces dans le masque de presence (masque par defaut de l archetype)
}

// m3Traverser rejoue `TraverseEntity` (bipede) avec une variante.
func m3Traverser(br *Lecteur, reg *Registry, v m3Variante) EntityTrace {
	t := EntityTrace{DesyncAt: -1}
	t.TypeIndex = uint32(br.ReadBits(6))
	consumeBipedDefaultState(br)
	consumeBipedDefaultStateTail(br)
	t.DefaultBits = br.BitPos()
	t.Gate = br.ReadBit()
	t.Mask = consumeMask(br) | v.force
	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt = 0
		return t
	}
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// m3Paquet retrouve le payload d une creation.
func m3Paquet(fc *FilmContext, c BipedCreation) []byte {
	data, pks, ok := fc.ChunkAt(c.Chunk)
	if !ok {
		return nil
	}
	for _, pk := range pks {
		if pk.Index == c.PacketIndex {
			return pk.Payload(data)
		}
	}
	return nil
}

func TestM3NaissanceGrammaire(t *testing.T) {
	doc := os.Getenv("M3_DOC")
	if doc == "" {
		t.Skip("M3_DOC absent : instrument de recherche")
	}
	tc := t516Cadre(t)
	cat := m3Catalogue(t, doc)
	cres, st, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	t.Logf("== creations : %d (ancres %d) ; idLow %d ; catalogue %d familles", len(cres), st.Anchors,
		tc.cfg.IDLowBits, len(cat))
	vars := []m3Variante{{"PROD", 0}, {"I0", 1}}
	justes := map[string]int{}
	ecarts := map[string]map[int]int{}
	dumps := 0
	for _, c := range cres {
		pay := m3Paquet(tc.fc, c)
		if pay == nil {
			continue
		}
		hdr := c.BitPos
		g, fams := m3OracleArmes(pay, hdr, tc.cfg, cat)
		corps := hdr + 3 + tc.cfg.IDLowBits + 2
		ligne := fmt.Sprintf("slot %d g%d t=%d oracle %+d %08X", c.Slot, c.Generation, c.TimestampUS/100000,
			g-hdr, fams)
		for _, v := range vars {
			br := LecteurSur(pay)
			br.poserCadre(tc.cfg)
			br.SetBitPos(corps)
			tr := m3Traverser(br, tc.reg, v)
			i43 := -1
			for _, cr := range tr.Comps {
				if cr.Index == 43 {
					i43 = cr.StartBit
					break
				}
			}
			if ecarts[v.nom] == nil {
				ecarts[v.nom] = map[int]int{}
			}
			if i43 >= 0 && g >= 0 {
				ecarts[v.nom][g-i43]++
				if g == i43 {
					justes[v.nom]++
				}
			}
			ligne += fmt.Sprintf(" | %s i43 %+d desync %d fin %+d", v.nom, i43-hdr, tr.DesyncAt, tr.EndBit-hdr)
			if dumps < 6 && v.nom == "I0" {
				t.Logf("   DUMP %s slot %d : etat par defaut fin %+d masque %v", v.nom, c.Slot,
					tr.DefaultBits-hdr, m3Indices(tr.Mask))
				t.Logf("        %s", m3Largeurs(tr, hdr))
			}
		}
		dumps++
		t.Logf("   %s", ligne)
	}
	for _, v := range vars {
		t.Logf("== variante %-5s : i43 sur l oracle %d / %d ; ecarts %v", v.nom, justes[v.nom], len(cres),
			m3Tri(ecarts[v.nom]))
	}
}

// m3Indices rend les index poses d un masque.
func m3Indices(m uint64) []int {
	var out []int
	for i := 0; i < 64; i++ {
		if m&(uint64(1)<<uint(i)) != 0 {
			out = append(out, i)
		}
	}
	return out
}

// m3Largeurs formate les composants d une traversee avec leur debut et leur largeur.
func m3Largeurs(tr EntityTrace, hdr int) string {
	var sb strings.Builder
	for i, cr := range tr.Comps {
		fin := tr.EndBit
		if i+1 < len(tr.Comps) {
			fin = tr.Comps[i+1].StartBit
		}
		fmt.Fprintf(&sb, "i%d@%+d(%d) ", cr.Index, cr.StartBit-hdr, fin-cr.StartBit)
	}
	return sb.String()
}

// m3Tri formate une distribution d ecarts.
func m3Tri(m map[int]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var sb strings.Builder
	for _, k := range cles {
		fmt.Fprintf(&sb, " %+d:%d", k, m[k])
	}
	return sb.String()
}

// TestM3NaissanceBits publie les bits bruts des records NEW de bipede, alignes sur l en-tete,
// par lignes de 64, pour comparer les records a l oeil (champs fixes / decalages).
func TestM3NaissanceBits(t *testing.T) {
	if os.Getenv("M3_BITS") == "" {
		t.Skip("M3_BITS absent")
	}
	tc := t516Cadre(t)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	n := 0
	for _, c := range cres {
		pay := m3Paquet(tc.fc, c)
		if pay == nil {
			continue
		}
		n++
		if n > 12 {
			break
		}
		t.Logf("== slot %d g%d t=%d bit %d paquet %d chunk %d (%d bits)", c.Slot, c.Generation,
			c.TimestampUS/100000, c.BitPos, c.PacketIndex, c.Chunk, len(pay)*8)
		for o := 0; o < 1800; o += 64 {
			var sb strings.Builder
			for k := 0; k < 64; k++ {
				p := c.BitPos + o + k
				if p >= len(pay)*8 {
					break
				}
				sb.WriteByte('0' + byte(PeekBits(pay, p, 1)))
			}
			t.Logf("   +%04d %s", o, sb.String())
		}
	}
}

// TestM3NaissanceExport ecrit, pour analyse HORS LIGNE (sans rouvrir le film), une ligne par
// record NEW de bipede : slot, generation, instant, oracle, puis les bits bruts [hdr, hdr+2600).
// Et une ligne par record d image-cle ti=35 (ancre du balayeur) : les bits [ancre, ancre+2600).
func TestM3NaissanceExport(t *testing.T) {
	sortie := os.Getenv("M3_EXPORT")
	doc := os.Getenv("M3_DOC")
	if sortie == "" || doc == "" {
		t.Skip("M3_EXPORT / M3_DOC absents")
	}
	tc := t516Cadre(t)
	cat := m3Catalogue(t, doc)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	var sb strings.Builder
	bits := func(pay []byte, de, n int) string {
		var b strings.Builder
		for k := 0; k < n && de+k < len(pay)*8; k++ {
			b.WriteByte('0' + byte(PeekBits(pay, de+k, 1)))
		}
		return b.String()
	}
	for _, c := range cres {
		pay := m3Paquet(tc.fc, c)
		if pay == nil {
			continue
		}
		g, fams := m3OracleArmes(pay, c.BitPos, tc.cfg, cat)
		fmt.Fprintf(&sb, "NEW\t%d\t%d\t%d\t%d\t%d\t%d\t%08X\t%s\n", c.Slot, c.Generation, c.TimestampUS,
			c.Chunk, c.PacketIndex, g-c.BitPos, fams, bits(pay, c.BitPos, 2600))
	}
	nkf := 0
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
				nkf++
				fmt.Fprintf(&sb, "KF\t%d\t%d\t%d\t%d\t%d\t%d\t-\t%s\n", r.Slot, r.Gen, pk.TimestampUS, ch,
					pk.Index, r.Bit, bits(pay, r.Bit, 2600))
			}
		}
	}
	if err := os.WriteFile(sortie, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	t.Logf("== export : %d NEW, %d KF -> %s", len(cres), nkf, sortie)
}

// TestM3ImageCleBipede rejoue l ETAT COMPLET des records d image-cle ti=35 (marche de
// production `WalkKeyframeFullState`) et publie ou tombe i43 et ce qu il lit : la famille lue
// est-elle au catalogue ? C est le temoin « les deserialiseurs i1..i42 sont-ils justes hors du
// record NEW ».
func TestM3ImageCleBipede(t *testing.T) {
	doc := os.Getenv("M3_DOC")
	if doc == "" || os.Getenv("M3_KF") == "" {
		t.Skip("M3_DOC / M3_KF absents")
	}
	tc := t516Cadre(t)
	cat := m3Catalogue(t, doc)
	ctx := tc.fc.ContexteDeLecture()
	var n, auCat, desync int
	dumps := 0
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
				n++
				tr := WalkKeyframeFullState(pay, r.Bit, tc.reg, ctx)
				if tr.DesyncAt >= 0 && tr.DesyncAt < 43 {
					desync++
				}
				fam := uint32(0)
				for _, cr := range tr.Comps {
					if cr.Index == 43 {
						if PeekBits(pay, cr.StartBit, 1) == 1 {
							fam = uint32(PeekBits(pay, cr.StartBit+1, 32))
						}
					}
				}
				if cat[fam] {
					auCat++
				}
				if dumps < 5 {
					dumps++
					t.Logf("   KF slot %d ch%d t=%d desync %d fam %08X au catalogue %v", r.Slot, ch,
						pk.TimestampUS/100000, tr.DesyncAt, fam, cat[fam])
					t.Logf("        %s", m3Largeurs(tr, r.Bit))
				}
			}
		}
	}
	t.Logf("== image-cle ti=35 : %d records, i43 lit une famille du catalogue %d, desync avant i43 %d",
		n, auCat, desync)
}

// m3Ligne est un record exporte.
type m3Ligne struct {
	kind   string
	slot   int
	oracle int
	fams   string
	pay    []byte
	nbits  int
}

// m3LireExport relit l export (bits en texte) et reconstruit les octets.
func m3LireExport(t *testing.T, chemin string) []m3Ligne {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("export : %v", err)
	}
	var out []m3Ligne
	for _, l := range strings.Split(strings.TrimSpace(string(blob)), "\n") {
		c := strings.Split(l, "\t")
		if len(c) < 9 {
			continue
		}
		s := c[8]
		pay := make([]byte, (len(s)+7)/8)
		for i := 0; i < len(s); i++ {
			if s[i] == '1' {
				pay[i/8] |= 1 << uint(7-i%8)
			}
		}
		slot, _ := strconv.Atoi(c[1])
		or, _ := strconv.Atoi(c[6])
		out = append(out, m3Ligne{kind: c[0], slot: slot, oracle: or, fams: c[7], pay: pay, nbits: len(s)})
	}
	return out
}

// m3AtterritSur dit si la boucle de composants lancee a `b` sur les SEULS composants `comps`
// (dans l ordre) fait commencer i43 exactement a `cible`.
func m3AtterritSur(pay []byte, arch Archetype, comps []int, b, cible int) bool {
	var m uint64
	for _, i := range comps {
		m |= uint64(1) << uint(i)
	}
	m |= uint64(1) << 43
	br := LecteurSur(pay)
	br.SetBitPos(b)
	t := EntityTrace{DesyncAt: -1, TypeIndex: BipedTypeIndex, Mask: m}
	traverseComponentLoopFrom(br, arch, &t, comps[0])
	for _, cr := range t.Comps {
		if cr.Index == 43 {
			return cr.StartBit == cible
		}
	}
	return false
}

// TestM3NaissanceRetro : pour chaque composant j du masque (avant i43), l ensemble des debuts b
// d ou la boucle de production, lancee sur j..i42, atterrit EXACTEMENT sur l oracle d i43.
func TestM3NaissanceRetro(t *testing.T) {
	in := os.Getenv("M3_EXPORT_IN")
	if in == "" {
		t.Skip("M3_EXPORT_IN absent")
	}
	tc := t516Cadre(t)
	arch, _ := tc.reg.Archetype(BipedTypeIndex)
	lignes := m3LireExport(t, in)
	vus := 0
	for _, l := range lignes {
		if l.kind != "NEW" || l.oracle <= 0 {
			continue
		}
		vus++
		if vus > 10 {
			break
		}
		br := LecteurSur(l.pay)
		br.SetBitPos(3 + tc.cfg.IDLowBits + 2)
		tr := m3Traverser(br, tc.reg, m3Variante{"PROD", 0})
		var comps []int
		for _, i := range m3Indices(tr.Mask) {
			if i < 43 {
				comps = append(comps, i)
			}
		}
		fwd := map[int]int{}
		for _, cr := range tr.Comps {
			fwd[cr.Index] = cr.StartBit
		}
		t.Logf("== slot %d oracle %d %s ; masque %v", l.slot, l.oracle, l.fams, comps)
		for j := len(comps) - 1; j >= 0; j-- {
			var cands []int
			for b := 200; b < l.oracle; b++ {
				if m3AtterritSur(l.pay, arch, comps[j:], b, l.oracle) {
					cands = append(cands, b)
				}
			}
			s := fmt.Sprint(cands)
			if len(cands) > 24 {
				s = fmt.Sprintf("%v ... (%d)", cands[:24], len(cands))
			}
			t.Logf("   i%-2d prod %4d : %s", comps[j], fwd[comps[j]], s)
		}
	}
}
