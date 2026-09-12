package filmdec

// equipment_palette_section3_research_test.go — LOT 5, ADDENDUM B : LA PALETTE rang -> objet
// EST-ELLE DANS LE FILM ?
//
// ## L'HYPOTHÈSE, ET POURQUOI ELLE MÉRITE UNE MESURE
//
// Le canal i48 (`biped-desired-ability-set-component`) ne transmet qu'un RANG de palette — un
// indice, pas une identité. La table rang -> objet est définie par la VARIANTE DE MODE, donc
// propre au match. Or `chunk_00` porte une section 3 d'environ 538 ko « propres au match », que
// personne n'a jamais ouverte (carte de `chunk_00`, 2026-08-30, D2.4). C'est la candidate
// naturelle pour porter cette table.
//
// Si elle la porte, le nommage de l'équipement devient trivial ET par match. Si elle ne la
// porte pas, c'est un négatif consigné et la section 3 reste au chantier trame.
//
// ## CE QUE CET INSTRUMENT FAIT, ET CE QU'IL NE FAIT PAS
//
// Il ne se propose PAS d'explorer la section 3 : c'est un chantier à part, et l'ouvrir ici
// serait exactement le débordement de périmètre que la règle 7 interdit. Il pose UNE question
// fermée : les identifiants d'objet d'équipement que nous connaissons sont-ils écrits là,
// quelque part, en clair ?
//
// La question est fermée parce que nous connaissons les réponses possibles : les 21 GlobalID de
// tag `eqip` du manifeste du titre, établis par les fichiers du jeu et confirmés à 100 % de
// couverture sur les ramassages non-arme des deux films de référence.
//
// ## LES TROIS CONTRÔLES, ÉCRITS AVANT LA MESURE
//
//	B-POS  CONTRÔLE POSITIF. La section 3 est bit-packée, donc rien ne garantit qu'un mot y
//	       soit aligné sur un octet. Mais elle porte des chaînes UTF-16LE lisibles (les
//	       gamertags, mesurés à 0x13CCA6 et 0x13F7AF sur 000d5950). L'instrument les recense :
//	       s'il ne les trouve pas, il lit les mauvais octets et AUCUN de ses négatifs ne vaut.
//	B-NEG  CONTRÔLE NÉGATIF. Des valeurs de 32 bits TIRÉES AU HASARD servent de plancher. Sur
//	       ~537 ko et 2^32 valeurs, l'espérance est de 0,000125 occurrence par valeur : tout
//	       identifiant réel trouvé une seule fois est déjà hors du hasard, et le plancher
//	       mesuré le dit au lieu de le supposer.
//	B-REF  RÉFÉRENCE CROISÉE. Les familles d'ARME du catalogue sont cherchées de la même façon.
//	       Le film sait manifestement identifier une arme ; si ses identifiants ne sont pas
//	       dans la section 3 non plus, l'absence des identifiants d'équipement ne dit RIEN de
//	       particulier sur l'équipement — elle dit que la section 3 n'est pas un catalogue.
//	       C'est le contrôle qui empêche de sur-interpréter un blanc.
//
// VERDICT B1 : au moins un identifiant d'équipement du manifeste est trouvé dans la section 3,
// au-dessus du plancher B-NEG. Sinon, négatif consigné.
//
// Garde `BIPED_PICKUP_FILM`. Lecture seule, un film par process, aucune cuisson.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"unicode"
)

// eps3Debut est le début de la section 3, mesuré identique sur les trois films de la carte de
// `chunk_00` : les deux premières sections sont octet pour octet identiques entre films d'un
// même build, et le premier écart tombe exactement ici.
const eps3Debut = 0x0CB65C

// eps3EqipConnus — les 21 GlobalID de tag `eqip` du manifeste du titre. Ils sont recopiés ici
// et NON chargés depuis le TOML : ce paquet est `filmdec`, la couche title-agnostic, qui n'a
// pas à connaître le manifeste d'un titre. Un instrument de recherche peut porter la liste en
// dur, une production ne le pourrait pas.
var eps3EqipConnus = []uint32{
	0x0f5716ff, 0x273fe0eb, 0x2974c233, 0x32d97758, 0x430dda48, 0x4396db42, 0x4744d742,
	0x528fce46, 0x686b40c9, 0x72199cba, 0x72b63d69, 0x730dc70f, 0x7ca85adc, 0x8c77ffe7,
	0x8e2dc574, 0xaada07f3, 0xb781197a, 0xbcabbe43, 0xcaaadcb0, 0xe7be9f5c, 0xeef5d48d,
}

// eps3Chaines — le vocabulaire cherché en clair. `eqip` et `sofd`/`sofa` sont les fourCC de
// groupe de tag ; les autres sont les mots qu'une table de palette porterait si elle nommait
// ses entrées.
var eps3Chaines = []string{"eqip", "sofd", "sofa", "equipment", "grenade", "ability", "eqhs", "gggl"}

// TestPaletteEquipementDansSection3 — ADDENDUM B. Chercher la table rang -> objet dans la
// section 3 de `chunk_00`, par les identifiants que nous connaissons déjà.
func TestPaletteEquipementDansSection3(t *testing.T) {
	dir := os.Getenv(egaFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", egaFilmEnv)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	data, err := eps3Inflate(raw)
	if err != nil {
		t.Fatalf("chunk_00 non décompressable : %v", err)
	}
	if len(data) <= eps3Debut {
		t.Fatalf("chunk_00 trop court (%d octets) : pas de section 3", len(data))
	}
	// Fin de la section 3 : le dernier octet non nul du fichier (la section 4 est du zéro).
	fin := len(data)
	for fin > eps3Debut && data[fin-1] == 0 {
		fin--
	}
	sec := data[eps3Debut:fin]
	t.Logf("== ADDENDUM B — LA PALETTE EST-ELLE DANS LA SECTION 3 ? · %s ==", dir)
	t.Logf("chunk_00 inflaté : %d octets · section 3 : [0x%06X, 0x%06X) = %d octets",
		len(data), eps3Debut, fin, len(sec))

	// B-POS — le contrôle positif : sait-on lire ce que la section 3 porte de lisible ?
	chaines := eps3ChainesUTF16(sec, 7)
	t.Logf("B-POS · chaînes UTF-16LE de 7 caractères ou plus trouvées : %d", len(chaines))
	for i, c := range chaines {
		if i >= 8 {
			t.Logf("   … et %d autre(s)", len(chaines)-8)
			break
		}
		t.Logf("   0x%06X  %q", eps3Debut+c.off, c.s)
	}
	if len(chaines) == 0 {
		t.Fatal("CONTRÔLE POSITIF EN ÉCHEC : aucune chaîne lisible dans la section 3 — l'instrument ne lit pas les bons octets, aucun négatif ne vaut")
	}

	// B-NEG — le plancher du hasard, mesuré sur 2 000 valeurs tirées au sort.
	rng := rand.New(rand.NewSource(20260901))
	planchers := 0
	const nTemoin = 2000
	for i := 0; i < nTemoin; i++ {
		planchers += len(eps3Occurrences(sec, rng.Uint32()))
	}
	t.Logf("B-NEG · plancher du hasard : %d occurrence(s) pour %d valeurs tirées au sort (%.5f par valeur)",
		planchers, nTemoin, float64(planchers)/nTemoin)

	// La mesure : les 21 identifiants d'équipement du manifeste.
	trouves, totalOcc := 0, 0
	for _, id := range eps3EqipConnus {
		occ := eps3Occurrences(sec, id)
		if len(occ) == 0 {
			continue
		}
		trouves++
		totalOcc += len(occ)
		t.Logf("   TROUVÉ %08x : %d occurrence(s) %s", id, len(occ), eps3Offsets(occ))
	}
	t.Logf("ÉQUIPEMENT : %d/%d identifiants du manifeste trouvés · %d occurrence(s) au total",
		trouves, len(eps3EqipConnus), totalOcc)

	// B-REF — la référence croisée : les familles d'arme y sont-elles, elles ?
	familles := eps3FamillesArme(t, dir)
	trouvesArme, occArme := 0, 0
	for _, id := range familles {
		occ := eps3Occurrences(sec, id)
		if len(occ) == 0 {
			continue
		}
		trouvesArme++
		occArme += len(occ)
		t.Logf("   TROUVÉ (arme) %08x : %d occurrence(s) %s", id, len(occ), eps3Offsets(occ))
	}
	t.Logf("B-REF · ARMES : %d/%d familles observées dans CE film trouvées · %d occurrence(s)",
		trouvesArme, len(familles), occArme)

	// Le vocabulaire en clair.
	for _, s := range eps3Chaines {
		a := eps3CompteASCII(sec, s)
		u := eps3CompteUTF16(sec, s)
		if a == 0 && u == 0 {
			continue
		}
		t.Logf("   CHAÎNE %q : ASCII x%d · UTF-16LE x%d", s, a, u)
	}
	t.Logf("VERDICT B1 (au moins un identifiant d'équipement du manifeste dans la section 3) : %v",
		trouves > 0)
	t.Logf("LECTURE : si B1 est faux ET B-REF est nul, la section 3 n'est pas un catalogue d'objets — l'absence ne vise pas l'équipement en particulier.")
}

// eps3Inflate décompresse un chunk_00 zlib, ou rend l'entrée telle quelle si elle est déjà
// inflatée. Même logique que ParseRegistryChunk, dupliquée ici parce que celle-ci n'expose pas
// les octets.
func eps3Inflate(raw []byte) ([]byte, error) {
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw, nil
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	dec, err := io.ReadAll(zr)
	if err != nil && len(dec) == 0 {
		return nil, err
	}
	return dec, nil
}

// eps3Occurrences rend les décalages où la valeur v apparaît comme u32, en little-endian ET en
// big-endian, alignée sur l'octet. Les deux boutismes sont cherchés parce que rien ne dit dans
// quel sens une table du film écrirait un GlobalID.
func eps3Occurrences(sec []byte, v uint32) []int {
	var le, be [4]byte
	binary.LittleEndian.PutUint32(le[:], v)
	binary.BigEndian.PutUint32(be[:], v)
	var out []int
	for i := 0; i+4 <= len(sec); i++ {
		if sec[i] == le[0] && sec[i+1] == le[1] && sec[i+2] == le[2] && sec[i+3] == le[3] {
			out = append(out, i)
			continue
		}
		if sec[i] == be[0] && sec[i+1] == be[1] && sec[i+2] == be[2] && sec[i+3] == be[3] {
			out = append(out, i)
		}
	}
	return out
}

// eps3Offsets rend au plus cinq décalages, en absolu dans le fichier inflaté.
func eps3Offsets(occ []int) string {
	out := "["
	for i, o := range occ {
		if i >= 5 {
			out += " …"
			break
		}
		if i > 0 {
			out += " "
		}
		out += eps3Hex(eps3Debut + o)
	}
	return out + "]"
}

// eps3Hex formate un décalage en hexadécimal sur six chiffres.
func eps3Hex(v int) string {
	const d = "0123456789ABCDEF"
	b := []byte("0x000000")
	for i := 7; i >= 2 && v > 0; i-- {
		b[i] = d[v&0xf]
		v >>= 4
	}
	return string(b)
}

// eps3Chaine est une chaîne lisible trouvée dans la section, avec son décalage relatif.
type eps3Chaine struct {
	off int
	s   string
}

// eps3ChainesUTF16 recense les suites de caractères ASCII imprimables encodées en UTF-16LE
// (octet, puis zéro) d'au moins minLen caractères. C'est la forme sous laquelle les gamertags
// ont été trouvés dans cette section.
func eps3ChainesUTF16(sec []byte, minLen int) []eps3Chaine {
	var out []eps3Chaine
	i := 0
	for i+2 <= len(sec) {
		if !eps3Imprimable(sec[i]) || sec[i+1] != 0 {
			i++
			continue
		}
		start, buf := i, []byte{}
		for i+2 <= len(sec) && eps3Imprimable(sec[i]) && sec[i+1] == 0 {
			buf = append(buf, sec[i])
			i += 2
		}
		if len(buf) >= minLen {
			out = append(out, eps3Chaine{off: start, s: string(buf)})
		}
	}
	sort.Slice(out, func(a, b int) bool { return len(out[a].s) > len(out[b].s) })
	return out
}

// eps3Imprimable dit si un octet est un caractère ASCII imprimable hors espace.
func eps3Imprimable(c byte) bool { return c > 0x20 && c < 0x7f && unicode.IsPrint(rune(c)) }

// eps3CompteASCII compte les occurrences d'une sous-chaîne en ASCII.
func eps3CompteASCII(sec []byte, s string) int {
	n, b := 0, []byte(s)
	for i := 0; i+len(b) <= len(sec); i++ {
		if string(sec[i:i+len(b)]) == string(b) {
			n++
		}
	}
	return n
}

// eps3CompteUTF16 compte les occurrences d'une sous-chaîne encodée en UTF-16LE.
func eps3CompteUTF16(sec []byte, s string) int {
	b := make([]byte, 0, len(s)*2)
	for i := 0; i < len(s); i++ {
		b = append(b, s[i], 0)
	}
	n := 0
	for i := 0; i+len(b) <= len(sec); i++ {
		if string(sec[i:i+len(b)]) == string(b) {
			n++
		}
	}
	return n
}

// eps3FamillesArme rend les familles d'arme RÉELLEMENT observées dans ce film — celles que le
// canal des ramassages natifs porte sur ses classes 0 et 1. Prendre les familles vues plutôt
// qu'un catalogue entier rend le contrôle croisé comparable : on cherche dans la section 3 des
// identifiants dont on SAIT que ce match les a joués.
func eps3FamillesArme(t *testing.T, dir string) []uint32 {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	pickups, _, err := ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	seen := map[uint32]bool{}
	var out []uint32
	for _, p := range pickups {
		if !BipedPickupIsWeaponClass(p.Class) || seen[p.CatalogID] {
			continue
		}
		seen[p.CatalogID] = true
		out = append(out, p.CatalogID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
