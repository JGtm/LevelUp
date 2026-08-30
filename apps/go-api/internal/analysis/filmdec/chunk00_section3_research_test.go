package filmdec

// chunk00_section3_research_test.go — LOT D2 : CE QU'EST LA TROISIEME SECTION DE chunk_00.
//
// LA CARTE (TestChunk00Carte) etablit trois sections dans un chunk_00 de 1 973 120 octets :
//   1. le REGISTRE de composants, blocs 0..49, `0x000000..0x0CB200` — connu du depot ;
//   2. l'EN-TETE : table par type + identification du build, `0x0CB200..~0x0CB460` — lot D1 ;
//   3. une TROISIEME SECTION, de `0x0CB65C` jusqu'au dernier octet non nul, jamais regardee.
//
// Elle pese environ un demi-megaoctet, elle DIFFERE integralement d'un film a l'autre (le premier
// ecart entre deux films tombe exactement a son debut) et elle represente a elle seule la
// quasi-totalite du poids compresse du fichier. Ce que cet instrument en dit :
//   — ses bornes exactes et son poids ;
//   — son entropie par octet, comparee a celle du registre (temoin de donnee structuree) et a la
//     valeur d'un flux incompressible (8,00 bits/octet) ;
//   — si elle se lit comme une suite de paquets a en-tete de 16 octets, comme les autres chunks ;
//   — si elle contient un flux zlib imbrique ;
//   — sa periodicite, mesuree sur des pas candidats.
//
// AUCUNE de ces mesures ne pretend la decoder. Le lot D avait pour mandat de dire ce que chunk_00
// CONTIENT ; il le dit, et il nomme ce qui reste ouvert.
//
// Garde CHUNK00_FILMS. Aucun code de production touche.

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"testing"
)

// section3Debut est l'offset ou commence la troisieme section : le premier octet qui differe
// entre deux films du meme build (mesure : `TestChunk00Carte`, 00162144 vs 000d5950 vs 00502e52).
const section3Debut = 0x0cb65c

// TestChunk00Section3 caracterise la troisieme section sans pretendre la decoder.
func TestChunk00Section3(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		fin := dernierNonNul(data)
		if fin <= section3Debut {
			t.Logf("=== %s === pas de troisieme section", filepath.Base(dir))
			continue
		}
		sec := data[section3Debut : fin+1]
		t.Logf("=== %s ===", filepath.Base(dir))
		t.Logf("  section 3 : 0x%06x .. 0x%06x = %d octets (%.1f %% du buffer inflate)",
			section3Debut, fin, len(sec), 100*float64(len(sec))/float64(len(data)))
		t.Logf("  entropie : section 3 = %.3f bits/octet ; registre (blocs 0..49) = %.3f ;"+
			" plafond incompressible = 8,000",
			entropieOctet(sec), entropieOctet(data[:registryBlocks*archetypeBlockSize]))
		t.Logf("  octets nuls : %.2f %% de la section (le registre en compte %.2f %%)",
			100*partNuls(sec), 100*partNuls(data[:registryBlocks*archetypeBlockSize]))
		paquetsDansSection(t, sec)
		zlibImbrique(t, sec)
		periodicite(t, sec)
		hexFenetre(t, data, section3Debut, 128)
	}
}

// TestChunk00ChainesUTF16 complete le recensement des chaines : `TestChunk00Chaines` ne voit que
// l'ASCII, or les formats Microsoft ecrivent volontiers en UTF-16. Une section de 537 ko propre a
// un film pourrait porter des noms (joueurs, carte, variante) sous cette forme.
func TestChunk00ChainesUTF16(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		chaines := utiles(chainesUTF16(data, 3))
		t.Logf("=== %s === %d chaines UTF-16LE d'au moins 3 caracteres, hors suites d'espaces",
			filepath.Base(dir), len(chaines))
		for i, c := range chaines {
			if i == 40 {
				t.Logf("  ... (%d autres)", len(chaines)-40)
				break
			}
			t.Logf("  @0x%06x (%d car.) %q%s", c.off, c.n, c.sample, ailleurs(dir, c))
		}
	}
}

// ailleurs cherche une chaine d'au moins 5 caracteres dans les AUTRES chunks du film. Une chaine
// que le fil des evenements porte aussi est un participant du match, pas un artefact de decoupe.
func ailleurs(dir string, c run) string {
	if c.n < 5 {
		return ""
	}
	n := CountFilmChunks(dir)
	total, chunks := 0, 0
	for i := 1; i <= n; i++ {
		data, err := ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		k := bytes.Count(data, []byte(c.sample))
		if k > 0 {
			total += k
			chunks++
		}
	}
	return fmt.Sprintf("  [aussi %d fois dans %d autres chunks]", total, chunks)
}

// utiles ecarte les chaines qui ne portent aucune lettre ni chiffre : dans une zone de flottants,
// l'octet 0x20 suivi d'un octet nul est frequent et fabrique des suites d'espaces qui ne sont pas
// du texte.
func utiles(in []run) []run {
	var out []run
	for _, r := range in {
		for i := 0; i < len(r.sample); i++ {
			c := r.sample[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// chainesUTF16 recense les suites d'au moins minLen caracteres ASCII encodes en UTF-16LE
// (octet imprimable suivi d'un octet nul).
func chainesUTF16(data []byte, minLen int) []run {
	var out []run
	i := 0
	for i+1 < len(data) {
		if data[i] < 0x20 || data[i] > 0x7e || data[i+1] != 0 {
			i++
			continue
		}
		j := i
		var sb []byte
		for j+1 < len(data) && data[j] >= 0x20 && data[j] <= 0x7e && data[j+1] == 0 {
			sb = append(sb, data[j])
			j += 2
		}
		if len(sb) >= minLen {
			out = append(out, run{off: i, n: len(sb), sample: string(sb)})
		}
		i = j + 1
	}
	return out
}

// TestChunk00DebutDense localise ou commence VRAIMENT la donnee dense de la section 3 : le
// premier offset suivi de 64 octets tous non nuls. Entre l'en-tete et ce point, la section ne
// porte que quelques champs isoles.
func TestChunk00DebutDense(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		dense := premierBlocDense(data, section3Debut, 64)
		t.Logf("=== %s === premier bloc de 64 octets tous non nuls apres 0x%06x : 0x%06x"+
			" (soit %d octets plus loin)", filepath.Base(dir), section3Debut, dense,
			dense-section3Debut)
		if dense > 0 {
			hexFenetre(t, data, dense-32, 128)
		}
	}
}

// premierBlocDense rend l'offset du premier bloc de n octets consecutifs non nuls a partir de
// debut, ou -1.
func premierBlocDense(data []byte, debut, n int) int {
	suite := 0
	for i := debut; i < len(data); i++ {
		if data[i] == 0 {
			suite = 0
			continue
		}
		suite++
		if suite == n {
			return i - n + 1
		}
	}
	return -1
}

// dernierNonNul rend l'offset du dernier octet non nul, ou -1.
func dernierNonNul(data []byte) int {
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] != 0 {
			return i
		}
	}
	return -1
}

// entropieOctet rend l'entropie de Shannon par octet.
func entropieOctet(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	var h [256]int
	for _, v := range b {
		h[v]++
	}
	e := 0.0
	n := float64(len(b))
	for _, c := range h {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		e -= p * math.Log2(p)
	}
	return e
}

// partNuls rend la part d'octets nuls.
func partNuls(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	n := 0
	for _, v := range b {
		if v == 0 {
			n++
		}
	}
	return float64(n) / float64(len(b))
}

// paquetsDansSection teste la lecture en paquets a en-tete de 16 octets, comme les autres chunks.
func paquetsDansSection(t *testing.T, sec []byte) {
	t.Helper()
	ps := WalkPackets(sec)
	couvert := 0
	types := map[uint16]int{}
	for _, p := range ps {
		couvert += packetHeaderSize + p.Size
		types[p.Type]++
	}
	t.Logf("  lecture en paquets (en-tete 16 o) : %d paquets, %d octets couverts sur %d (%.1f %%)"+
		" ; types rencontres : %v",
		len(ps), couvert, len(sec), 100*float64(couvert)/float64(len(sec)), types)
	if len(ps) == 0 || couvert < len(sec)/2 {
		t.Log("  -> la section N'EST PAS un flux de paquets : le decoupage n'en couvre pas la moitie")
	}
}

// zlibImbrique cherche une en-tete zlib plausible dans la section.
func zlibImbrique(t *testing.T, sec []byte) {
	t.Helper()
	n := 0
	premier := -1
	for i := 0; i+1 < len(sec); i++ {
		if sec[i] != 0x78 {
			continue
		}
		if c := sec[i+1]; c == 0x01 || c == 0x5e || c == 0x9c || c == 0xda {
			if (uint16(sec[i])<<8|uint16(c))%31 == 0 {
				n++
				if premier < 0 {
					premier = i
				}
			}
		}
	}
	t.Logf("  en-tetes zlib plausibles : %d (premiere a +0x%06x) — au hasard on en attend environ"+
		" %d sur %d octets, donc ce compte ne vaut que s'il est tres au-dessus",
		n, premier, len(sec)/4096, len(sec))
}

// periodicite mesure, pour des pas candidats, la part d'octets egaux a distance de ce pas. Une
// structure a enregistrement de taille fixe fait ressortir son pas.
func periodicite(t *testing.T, sec []byte) {
	t.Helper()
	pas := []int{2, 4, 8, 12, 16, 20, 24, 32, 48, 64, 128, 256, 260, 512, 1024}
	var meilleur int
	meilleurTaux := 0.0
	var lignes []string
	for _, p := range pas {
		if p >= len(sec) {
			continue
		}
		eg := 0
		for i := 0; i+p < len(sec); i++ {
			if sec[i] == sec[i+p] {
				eg++
			}
		}
		taux := float64(eg) / float64(len(sec)-p)
		lignes = append(lignes, fmt.Sprintf("%d:%.3f%%", p, 100*taux))
		if taux > meilleurTaux {
			meilleurTaux, meilleur = taux, p
		}
	}
	t.Logf("  periodicite (part d'octets egaux a distance du pas) : %v", lignes)
	t.Logf("  -> meilleur pas %d a %.3f %% ; le hasard sur des octets uniformes donne 0,391 %%",
		meilleur, 100*meilleurTaux)
}
