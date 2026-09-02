package filmdec

// chunk00_carte_research_test.go — LOT D2 : LA CARTE EXHAUSTIVE DE chunk_00.
//
// CE QUE LE DEPOT SAIT DEJA. `registry.go` lit chunk_00 comme une suite de blocs
// d'archetype de 16 640 octets (64 slots de 260) et n'en retient que les slots a nom non
// vide : 118 blocs, 49 porteurs, 1 067 couples (archetype, composant). Personne n'a jamais
// regarde le RESTE : ce que valent les octets hors des noms, ce qu'il y a APRES le dernier
// bloc, et si deux films rendent vraiment le meme buffer.
//
// CE QUE CET INSTRUMENT MESURE, SANS SEUIL A DECLARER (recensement, pas verdict) :
//   1. taille inflatee, empreinte, nombre de blocs, RESTE en fin de buffer ;
//   2. occupation slot par slot : combien de slots nommes par bloc, ou s'arrete le nom ;
//   3. les octets NON NULS hors des zones lues par `parseRegistry` (kind, flags, nom) —
//      c'est la seule facon de dire « il n'y a rien d'autre » ou « il y a autre chose » ;
//   4. toutes les chaines ASCII imprimables de 4 caracteres ou plus, avec leur position, et
//      la part d'entre elles qui ne commence PAS a `slot+8` (donc hors registre connu) ;
//   5. le champ `kind` (u32 a slot+0), que `parseRegistry` jette ;
//   6. la comparaison octet a octet des buffers inflates de plusieurs films.
//
// Garde CHUNK00_FILMS (liste separee par `;` de repertoires de film). Aucun code de
// production touche : le test vit dans le paquet et se sert des symboles internes.

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// chunk00Films rend les repertoires de film de la garde d'environnement.
func chunk00Films(t *testing.T, envName string) []string {
	t.Helper()
	v := os.Getenv(envName)
	if v == "" {
		t.Skipf("%s absent : instrument saute", envName)
	}
	var out []string
	for _, p := range strings.Split(v, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// readChunk00 lit et decompresse chunk_00.bin d'un repertoire de film.
func readChunk00(t *testing.T, dir string) (raw, data []byte) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
	if err != nil {
		t.Fatalf("lecture chunk_00 de %s : %v", dir, err)
	}
	return b, filmsource.Inflate(b)
}

// slotSpan decrit la zone REELLEMENT lue par parseRegistry dans un slot nomme :
// [off, off+8) = kind|flags, [off+8, off+8+len(nom)) = le nom, puis le NUL terminateur.
type slotSpan struct {
	off, nameLen int
}

// parsedSpans rejoue le decoupage de parseRegistry et rend les slots nommes.
func parsedSpans(data []byte) []slotSpan {
	var out []slotSpan
	for b := 0; b < len(data)/archetypeBlockSize; b++ {
		base := b * archetypeBlockSize
		for s := 0; s < archetypeBlockSlots; s++ {
			off := base + s*registrySlotSize
			name := slotName(data, off)
			if name == "" {
				break
			}
			out = append(out, slotSpan{off: off, nameLen: len(name)})
		}
	}
	return out
}

// couvert construit le masque des octets que parseRegistry lit vraiment (kind, flags, nom,
// NUL de fin de nom).
func couvert(data []byte, spans []slotSpan) []bool {
	m := make([]bool, len(data))
	for _, s := range spans {
		for i := s.off; i < s.off+8+s.nameLen+1 && i < len(m); i++ {
			m[i] = true
		}
	}
	return m
}

// runsNonNuls recense les plages d'octets non nuls hors du masque `lu`.
type run struct {
	off, n int
	sample string
}

func runsNonNuls(data []byte, lu []bool) []run {
	var out []run
	i := 0
	for i < len(data) {
		if data[i] == 0 || lu[i] {
			i++
			continue
		}
		j := i
		for j < len(data) && data[j] != 0 && !lu[j] {
			j++
		}
		out = append(out, run{off: i, n: j - i, sample: fmt.Sprintf("% x", data[i:min(j, i+16)])})
		i = j
	}
	return out
}

// chainesASCII recense toutes les suites d'au moins 4 caracteres imprimables.
func chainesASCII(data []byte, minLen int) []run {
	var out []run
	i := 0
	for i < len(data) {
		if data[i] < 0x20 || data[i] > 0x7e {
			i++
			continue
		}
		j := i
		for j < len(data) && data[j] >= 0x20 && data[j] <= 0x7e {
			j++
		}
		if j-i >= minLen {
			out = append(out, run{off: i, n: j - i, sample: string(data[i:j])})
		}
		i = j
	}
	return out
}

// TestChunk00Carte publie la carte exhaustive de chunk_00 pour chaque film de la garde.
func TestChunk00Carte(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	buffers := map[string][]byte{}
	for _, dir := range dirs {
		raw, data := readChunk00(t, dir)
		buffers[dir] = data
		carteUnFilm(t, dir, raw, data)
	}
	comparerFilms(t, dirs, buffers)
}

// carteUnFilm publie taille, blocs, reste, occupation et zones non nulles d'un chunk_00.
func carteUnFilm(t *testing.T, dir string, raw, data []byte) {
	t.Helper()
	nBlocks := len(data) / archetypeBlockSize
	reste := len(data) % archetypeBlockSize
	t.Logf("=== %s ===", filepath.Base(dir))
	t.Logf("  compresse %d o -> inflate %d o ; sha256 %x", len(raw), len(data), sha256.Sum256(data))
	t.Logf("  blocs de %d o : %d ; RESTE en fin de buffer : %d o", archetypeBlockSize, nBlocks, reste)

	spans := parsedSpans(data)
	lu := couvert(data, spans)
	nLus := 0
	for _, b := range lu {
		if b {
			nLus++
		}
	}
	t.Logf("  slots nommes : %d ; octets lus par parseRegistry : %d (%.3f %% du buffer)",
		len(spans), nLus, 100*float64(nLus)/float64(len(data)))

	occupationBlocs(t, data, nBlocks)
	zonesNonNulles(t, data, lu)
	if reste > 0 {
		queue := data[nBlocks*archetypeBlockSize:]
		nz := 0
		for _, b := range queue {
			if b != 0 {
				nz++
			}
		}
		t.Logf("  QUEUE (%d o hors blocs) : %d octets non nuls ; premiers 64 : % x",
			len(queue), nz, queue[:min(64, len(queue))])
	}
}

// occupationBlocs publie, bloc par bloc, le nombre de slots nommes.
func occupationBlocs(t *testing.T, data []byte, nBlocks int) {
	t.Helper()
	var porteurs []string
	total := 0
	for b := 0; b < nBlocks; b++ {
		n := 0
		for s := 0; s < archetypeBlockSlots; s++ {
			if slotName(data, b*archetypeBlockSize+s*registrySlotSize) == "" {
				break
			}
			n++
		}
		total += n
		if n > 0 {
			porteurs = append(porteurs, fmt.Sprintf("%d:%d", b, n))
		}
	}
	t.Logf("  occupation (bloc:slots) — %d porteurs, %d slots : %s",
		len(porteurs), total, strings.Join(porteurs, " "))
}

// zonesNonNulles publie les plages d'octets non nuls que parseRegistry ne lit pas.
func zonesNonNulles(t *testing.T, data []byte, lu []bool) {
	t.Helper()
	rs := runsNonNuls(data, lu)
	tot := 0
	for _, r := range rs {
		tot += r.n
	}
	t.Logf("  HORS ZONES LUES : %d plages non nulles, %d octets au total", len(rs), tot)
	sort.Slice(rs, func(i, j int) bool { return rs[i].n > rs[j].n })
	for i, r := range rs {
		if i == 20 {
			t.Logf("    ... (%d autres plages)", len(rs)-20)
			break
		}
		bloc, dansBloc := r.off/archetypeBlockSize, r.off%archetypeBlockSize
		t.Logf("    @0x%06x (bloc %d, slot %d, +%d) %d o : %s",
			r.off, bloc, dansBloc/registrySlotSize, dansBloc%registrySlotSize, r.n, r.sample)
	}
}

// TestChunk00Chaines recense toutes les chaines ASCII et isole celles qui ne sont pas des
// noms de composant a `slot+8` — c'est la ou se cacherait une SECONDE table de noms.
func TestChunk00Chaines(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		chaines := chainesASCII(data, 4)
		nomsSlots := map[int]bool{}
		for _, s := range parsedSpans(data) {
			nomsSlots[s.off+8] = true
		}
		var hors []run
		for _, c := range chaines {
			if !nomsSlots[c.off] {
				hors = append(hors, c)
			}
		}
		t.Logf("=== %s === %d chaines >=4 car. ; %d a slot+8 (noms de composant) ; %d HORS",
			filepath.Base(dir), len(chaines), len(chaines)-len(hors), len(hors))
		for i, c := range hors {
			if i == 60 {
				t.Logf("  ... (%d autres)", len(hors)-60)
				break
			}
			bloc, dansBloc := c.off/archetypeBlockSize, c.off%archetypeBlockSize
			t.Logf("  @0x%06x (bloc %d, slot %d, +%d) %q",
				c.off, bloc, dansBloc/registrySlotSize, dansBloc%registrySlotSize, c.sample)
		}
	}
}

// TestChunk00Kind publie le champ `kind` (u32 a slot+0) que parseRegistry jette : combien de
// valeurs distinctes, et si un meme nom porte toujours le meme kind.
func TestChunk00Kind(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		parNom := map[string]map[uint32]int{}
		parKind := map[uint32]int{}
		for _, s := range parsedSpans(data) {
			nom := slotName(data, s.off)
			k := leU32(data, s.off)
			parKind[k]++
			if parNom[nom] == nil {
				parNom[nom] = map[uint32]int{}
			}
			parNom[nom][k]++
		}
		multi := 0
		for _, m := range parNom {
			if len(m) > 1 {
				multi++
			}
		}
		t.Logf("=== %s === %d noms distincts ; %d valeurs de kind distinctes ; %d noms portant PLUSIEURS kinds",
			filepath.Base(dir), len(parNom), len(parKind), multi)
		var ks []uint32
		for k := range parKind {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(i, j int) bool { return parKind[ks[i]] > parKind[ks[j]] })
		for i, k := range ks {
			if i == 12 {
				t.Logf("  ... (%d autres valeurs)", len(ks)-12)
				break
			}
			t.Logf("  kind=0x%08x : %d slots", k, parKind[k])
		}
	}
}

// TestChunk00Sections cherche les FRONTIERES de chunk_00 : la fin du registre, le debut de
// la zone qui porte la chaine de build, la fin des octets non nuls. Puis il vide en
// hexadecimal les fenetres demandees par CHUNK00_HEX (liste `offset:longueur` en decimal).
func TestChunk00Sections(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		t.Logf("=== %s === %d octets", filepath.Base(dir), len(data))
		dernierNonNul := -1
		for i := len(data) - 1; i >= 0; i-- {
			if data[i] != 0 {
				dernierNonNul = i
				break
			}
		}
		t.Logf("  dernier octet non nul @0x%06x ; %d octets nuls en queue",
			dernierNonNul, len(data)-1-dernierNonNul)
		zonesNulles(t, data, 256)
		for _, spec := range strings.Split(os.Getenv("CHUNK00_HEX"), ",") {
			off, n, ok := parseFenetre(spec)
			if !ok {
				continue
			}
			hexFenetre(t, data, off, n)
		}
	}
}

// zonesNulles publie les plages de zeros d'au moins minLen octets : ce sont les coutures
// entre sections.
func zonesNulles(t *testing.T, data []byte, minLen int) {
	t.Helper()
	var out []run
	i := 0
	for i < len(data) {
		if data[i] != 0 {
			i++
			continue
		}
		j := i
		for j < len(data) && data[j] == 0 {
			j++
		}
		if j-i >= minLen {
			out = append(out, run{off: i, n: j - i})
		}
		i = j
	}
	t.Logf("  plages de zeros >= %d o : %d", minLen, len(out))
	for i, r := range out {
		if i == 30 {
			t.Logf("    ... (%d autres)", len(out)-30)
			break
		}
		t.Logf("    @0x%06x .. 0x%06x (%d o)", r.off, r.off+r.n, r.n)
	}
}

// parseFenetre lit une specification `offset:longueur` en decimal.
func parseFenetre(spec string) (off, n int, ok bool) {
	parts := strings.Split(strings.TrimSpace(spec), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[0], "%v", &off); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%v", &n); err != nil {
		return 0, 0, false
	}
	return off, n, n > 0
}

// hexFenetre vide une fenetre en hexadecimal avec sa colonne ASCII.
func hexFenetre(t *testing.T, data []byte, off, n int) {
	t.Helper()
	if off < 0 || off >= len(data) {
		return
	}
	end := min(off+n, len(data))
	t.Logf("  --- fenetre 0x%06x .. 0x%06x ---", off, end)
	for p := off; p < end; p += 32 {
		q := min(p+32, end)
		var asc strings.Builder
		for _, b := range data[p:q] {
			if b >= 0x20 && b <= 0x7e {
				asc.WriteByte(b)
			} else {
				asc.WriteByte('.')
			}
		}
		t.Logf("  %06x  % x  |%s|", p, data[p:q], asc.String())
	}
}

// leU32 lit un u32 petit-boutiste, 0 si hors borne.
func leU32(data []byte, off int) uint32 {
	if off+4 > len(data) {
		return 0
	}
	return uint32(data[off]) | uint32(data[off+1])<<8 | uint32(data[off+2])<<16 | uint32(data[off+3])<<24
}

// comparerFilms compare les buffers inflates deux a deux, octet a octet.
func comparerFilms(t *testing.T, dirs []string, buffers map[string][]byte) {
	t.Helper()
	if len(dirs) < 2 {
		return
	}
	ref, refData := dirs[0], buffers[dirs[0]]
	for _, d := range dirs[1:] {
		other := buffers[d]
		n := min(len(refData), len(other))
		diffs, premier := 0, -1
		zones := map[int]int{}
		for i := 0; i < n; i++ {
			if refData[i] != other[i] {
				diffs++
				if premier < 0 {
					premier = i
				}
				zones[i/archetypeBlockSize]++
			}
		}
		t.Logf("DIFF %s vs %s : tailles %d / %d ; %d octets differents sur %d comparables",
			filepath.Base(ref), filepath.Base(d), len(refData), len(other), diffs, n)
		if premier >= 0 {
			t.Logf("  premier ecart @0x%06x (bloc %d, slot %d, +%d) : %02x vs %02x",
				premier, premier/archetypeBlockSize,
				(premier%archetypeBlockSize)/registrySlotSize,
				(premier%archetypeBlockSize)%registrySlotSize, refData[premier], other[premier])
			var bl []int
			for b := range zones {
				bl = append(bl, b)
			}
			sort.Ints(bl)
			var parts []string
			for _, b := range bl {
				parts = append(parts, fmt.Sprintf("%d:%d", b, zones[b]))
			}
			t.Logf("  repartition des ecarts par bloc : %s", strings.Join(parts, " "))
		}
	}
}
