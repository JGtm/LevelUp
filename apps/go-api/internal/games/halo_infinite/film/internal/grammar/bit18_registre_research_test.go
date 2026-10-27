//go:build research

package grammar

// bit18_registre_research_test.go — LE BIT DE TROP DES BUILDS <= HI_1_8_0 : QUELLE DONNEE DECLAREE
// PAR LE FILM SEPARE LES DEUX FAMILLES (branche feat/ri-ecart-108, rapport
// `.ai/archive/V7.5/RAPPORT_BIT_BUILDS_18_2026-10-09.md`). Aucun fichier de production n est touche.
//
// Deux tests :
//
//	TestBit18Parc      : pour chaque film du cache, l en-tete de `chunk_00` (majeure, format), la
//	                     section 2 (build, table par type, bit de controle) et l empreinte du
//	                     registre. Sortie `parc.tsv`.
//	TestBit18Registre  : pour une liste de films, la comparaison OCTET A OCTET de tout ce qui
//	                     precede le corps de `chunk_00` (en-tete, registre complet — noms,
//	                     niveaux, bourrage des noms —, table par type, section 2) contre un film
//	                     de reference. Sortie `diff_<film>.txt`.
//
//	BIT18_RACINE=<film_chunks> BIT18_OUT=<dossier> [BIT18_FILMS=<id,...>] [BIT18_REF=<id>] \
//	  go test -tags=research -count=1 -run '^TestBit18' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func bit18Env(t *testing.T) (racine, out string) {
	t.Helper()
	racine, out = os.Getenv("BIT18_RACINE"), os.Getenv("BIT18_OUT")
	if racine == "" || out == "" {
		t.Skip("BIT18_RACINE et BIT18_OUT requis")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	return racine, out
}

func bit18Chunk0(racine, id string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(racine, id, "chunk_00.bin"))
	if err != nil {
		return nil, err
	}
	return source.Inflate(b), nil
}

// TestBit18Parc publie, par film, les donnees declarees de `chunk_00` qui peuvent porter une
// grammaire : majeure, format, build, bit de controle, cardinal et contenu de la table par type,
// empreinte et nombre de blocs du registre.
func TestBit18Parc(t *testing.T) {
	racine, out := bit18Env(t)
	ents, err := os.ReadDir(racine)
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	sb.WriteString("film\tmajeure\tformat\tbuild\tcontrole\tgenres\tblocs\tempreinte\tnommes\ttable\tmoteur\n")
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		d, err := bit18Chunk0(racine, e.Name())
		if err != nil || len(d) < 8 {
			continue
		}
		maj, _ := FilmMajorVersionFromHeader(d)
		fmtv, _ := FilmFormatVersionFromHeader(d)
		id, ierr := ReadFilmIdentity(d)
		build, ctrl, genres, table, moteur := "-", "-", -1, "-", "-"
		blocs, empreinte, nommes := 0, uint64(0), 0
		if ierr == nil {
			build, ctrl, genres = id.Build, fmt.Sprint(id.ControleDeCorruption), len(id.TypeVersions)
			var tb strings.Builder
			for _, v := range id.TypeVersions {
				fmt.Fprintf(&tb, "%d,", v)
			}
			table = tb.String()
			moteur = fmt.Sprintf("lue=%v,presente=%v,type=%d", id.Variante.Lue, id.Variante.Presente, id.Variante.TypeDeMoteur)
			blocs, empreinte, nommes = id.RegistryBlocks, id.RegistryFingerprint, id.RegistryNamedSlots
		} else {
			reg := parseRegistry(d)
			blocs, empreinte, nommes = len(reg.Archetypes), RegistryFingerprint(reg), RegistryNamedSlots(reg)
		}
		fmt.Fprintf(&sb, "%s\t%#x\t%d\t%s\t%s\t%d\t%d\t%#x\t%d\t%s\t%s\n", e.Name(), maj, fmtv, build, ctrl,
			genres, blocs, empreinte, nommes, table, moteur)
	}
	if err := os.WriteFile(filepath.Join(out, "parc.tsv"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// bit18Zone nomme l octet `o` de `chunk_00` : en-tete, entree de registre (bloc, entree, champ)
// ou zone apres le registre.
func bit18Zone(o, finRegistre int) string {
	switch {
	case o < registryEntryBase:
		return fmt.Sprintf("entete+%d", o)
	case o < finRegistre:
		r := o - registryEntryBase
		b, s, c := r/archetypeBlockSize, (r%archetypeBlockSize)/registrySlotSize, r%registrySlotSize
		champ := "nom"
		if c >= registryEntryLevelOffset {
			champ = "niveau"
		}
		return fmt.Sprintf("bloc%d/i%d/%s+%d", b, s, champ, c)
	default:
		return fmt.Sprintf("apres_registre+%#x", o-finRegistre)
	}
}

// TestBit18Registre compare chaque film de BIT18_FILMS au film BIT18_REF, octet a octet, sur tout
// ce qui precede le corps de `chunk_00`. Les octets des chaines de version et de build, de
// l identifiant de build, de la changelist et de l horodatage sont attendus differents ; tout le
// reste est une donnee declaree candidate.
func TestBit18Registre(t *testing.T) {
	racine, out := bit18Env(t)
	ref := os.Getenv("BIT18_REF")
	films := strings.Split(os.Getenv("BIT18_FILMS"), ",")
	if ref == "" || len(films) == 0 {
		t.Skip("BIT18_REF et BIT18_FILMS requis")
	}
	dr, err := bit18Chunk0(racine, ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range films {
		d, err := bit18Chunk0(racine, f)
		if err != nil {
			t.Fatal(err)
		}
		bit18Comparer(t, out, ref, f, dr, d)
	}
}

func bit18Comparer(t *testing.T, out, refID, id string, dr, d []byte) {
	t.Helper()
	regR, reg := parseRegistry(dr), parseRegistry(d)
	finR := registryEntryBase + len(regR.Archetypes)*archetypeBlockSize
	fin := registryEntryBase + len(reg.Archetypes)*archetypeBlockSize
	var sb strings.Builder
	fmt.Fprintf(&sb, "ref=%s film=%s blocs ref=%d film=%d finRegistre ref=%#x film=%#x\n", refID, id,
		len(regR.Archetypes), len(reg.Archetypes), finR, fin)
	// 1. Le registre, sur la longueur commune.
	n := min(finR, fin)
	diffs := map[string]int{}
	var lignes []string
	for o := range n {
		if d[o] != dr[o] {
			z := bit18Zone(o, n)
			diffs[strings.SplitN(z, "+", 2)[0]]++
			if len(lignes) < 4000 {
				lignes = append(lignes, fmt.Sprintf("%#08x\t%s\tref=%02x\tfilm=%02x", o, z, dr[o], d[o]))
			}
		}
	}
	fmt.Fprintf(&sb, "octets differents dans le registre commun : %d\n", len(lignes))
	// 2. Niveaux et noms, entree par entree (lecture du jeu : nom @+0, niveau @+0x100).
	for b := range min(len(regR.Archetypes), len(reg.Archetypes)) {
		for s := range archetypeBlockSlots {
			oR := registryEntryBase + b*archetypeBlockSize + s*registrySlotSize
			nmR, nm := entryName(dr, oR), entryName(d, oR)
			lvR, lv := entryLevel(dr, oR), entryLevel(d, oR)
			if nmR != nm || lvR != lv {
				fmt.Fprintf(&sb, "ENTREE bloc%d i%d ref=%q/%d film=%q/%d\n", b, s, nmR, lvR, nm, lv)
			}
		}
	}
	// 3. Apres le registre : table par type et section 2, alignees sur la fin de chaque registre.
	idR, errR := ReadFilmIdentity(dr)
	idF, errF := ReadFilmIdentity(d)
	if errR == nil && errF == nil {
		fmt.Fprintf(&sb, "build ref=%s film=%s genres ref=%d film=%d controle ref=%v film=%v\n", idR.Build,
			idF.Build, len(idR.TypeVersions), len(idF.TypeVersions), idR.ControleDeCorruption, idF.ControleDeCorruption)
		for g := range max(len(idR.TypeVersions), len(idF.TypeVersions)) {
			var a, b int64 = -1, -1
			if g < len(idR.TypeVersions) {
				a = int64(idR.TypeVersions[g])
			}
			if g < len(idF.TypeVersions) {
				b = int64(idF.TypeVersions[g])
			}
			if a != b {
				fmt.Fprintf(&sb, "GENRE %d ref=%d film=%d\n", g, a, b)
			}
		}
		// La section apres la table : depuis la chaine de version jusqu au corps.
		corpsR := idR.BuildOffset + identBoolOff + identApresBoolBytes
		corps := idF.BuildOffset + identBoolOff + identApresBoolBytes
		baseR, base := idR.BuildOffset-identFieldBytes, idF.BuildOffset-identFieldBytes
		m := min(corpsR-baseR, corps-base)
		nd := 0
		for k := range m {
			if dr[baseR+k] != d[base+k] {
				nd++
				if nd <= 200 {
					fmt.Fprintf(&sb, "SECTION2 +%#x ref=%02x film=%02x\n", k, dr[baseR+k], d[base+k])
				}
			}
		}
		fmt.Fprintf(&sb, "section2 : %d octets differents sur %d\n", nd, m)
	}
	keys := make([]string, 0, len(diffs))
	for k := range diffs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&sb, "ZONE %s %d\n", k, diffs[k])
	}
	sb.WriteString(strings.Join(lignes, "\n"))
	sb.WriteString("\n")
	if err := os.WriteFile(filepath.Join(out, "diff_"+id+".txt"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// bit18Fenetre ecrit (BIT18_FENETRE non vide) la ligne W d un record bipede de la sonde 2.7.d1 : le
// debut et la largeur de chaque composant de i47 a i58, et les bits BRUTS du flux depuis le debut
// d i48 jusqu a 64 bits apres le debut d i54. Appelee par `dumpRecord`.
//
//	W  film ts slot debut bits  i47:debut:bits,...  bits_bruts
func bit18Fenetre(court string, p *lecture.Paquet, r *lecture.Record) string {
	if os.Getenv("BIT18_FENETRE") == "" {
		return ""
	}
	var cs []string
	d48, d54 := -1, -1
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if co.Index < 47 || co.Index > 58 {
			continue
		}
		cs = append(cs, fmt.Sprintf("i%d:%d:%d", co.Index, int(co.Debut)-int(r.Debut), co.Bits))
		switch co.Index {
		case 48:
			d48 = int(co.Debut)
		case 54:
			d54 = int(co.Debut)
		}
	}
	if d48 < 0 || d54 < 0 {
		return ""
	}
	var sb strings.Builder
	for b := d48; b < d54+64 && b < len(p.Payload)*8; b++ {
		sb.WriteByte(byte('0' + source.BitAt(p.Payload, b))) //nolint:gosec // un bit
	}
	return fmt.Sprintf("W\t%s\t%d\t%d\t%d\t%d\t%s\t%s", court, p.TS, r.Vie.Slot, r.Debut, r.Bits,
		strings.Join(cs, ","), sb.String())
}

// bit18InsererBit rend une copie de `pay` ou un bit `b` est insere AVANT le bit `q` : les bits
// [0, q) sont inchanges, le bit q vaut b, l ancien bit i >= q devient le bit i+1.
func bit18InsererBit(pay []byte, q int, b int) []byte {
	out := make([]byte, len(pay)+1)
	n := len(pay) * 8
	for i := range n + 1 {
		var v int
		switch {
		case i < q:
			v = source.BitAt(pay, i)
		case i == q:
			v = b
		default:
			v = source.BitAt(pay, i-1)
		}
		if v != 0 {
			out[i/8] |= 0x80 >> (i % 8)
		}
	}
	return out
}

// bit18DrapeauFinalI53 rend le bit du DRAPEAU FINAL de `FUN_1407efc5c` (1407efeba) dans le record
// tel que la marche l a lu : debut d i53, porte R(1)+R(8), 11 x (R(1)+R(12)), 7 drapeaux. Le drapeau
// sous `niveau > 1` (1407efeaf) n est pas lu au niveau 1. -1 si i53 n a pas ete traverse.
func bit18DrapeauFinalI53(p *lecture.Paquet, r *lecture.Record, niveau uint32) int {
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if co.Index != 53 || (co.Etat != lecture.EtatInterprete && co.Etat != lecture.EtatDelimite) {
			continue
		}
		q := int(co.Debut)
		if source.BitAt(p.Payload, q) == 1 {
			q += 8
		}
		q++
		for range 11 {
			g := source.BitAt(p.Payload, q)
			q++
			if g == 1 {
				q += 12
			}
		}
		q += 7
		if niveau > 1 {
			q++
		}
		return q
	}
	return -1
}

// bit18Variante (BIT18_VARIANTE non vide) rejoue la marche de PRODUCTION du record bipede i avec
// un bit nul insere a la place du drapeau final d i53 — la grammaire du jeu lit alors, a partir
// de ce point, le flux tel qu il serait s il portait ce drapeau. C est une MESURE de la position du
// bit, pas une regle : elle s applique a tous les films, quel que soit leur build.
//
//	V  film  ts  slot  ecart_orig  ferme_orig  ecart_ctrl  ecart_variante  ferme_variante  desync_variante
//
// ecart = fin de lecture - debut du record suivant (meme definition que l ecart de fin) ; ctrl =
// la meme marche sans insertion (controle de l outil).
func bit18Variante(c *ri27d0Canal, p *lecture.Paquet, i int, ctx ContexteDeLecture) {
	x, arch := c.x, c.arch
	if os.Getenv("BIT18_VARIANTE") == "" || i+1 >= len(p.Records) {
		return
	}
	r := &p.Records[i]
	suivant := int(p.Records[i+1].Debut)
	ctx.Obs = nil
	ctrl := WalkKeyframeFullState(p.Payload, int(r.Debut), x.reg, ctx)
	ecartCtrl := ctrl.EndBit - suivant
	q := bit18DrapeauFinalI53(p, r, arch.Level(53))
	ecartV, desV := ecartCtrl, ctrl.DesyncAt
	if q >= 0 && q < suivant {
		tv := WalkKeyframeFullState(bit18InsererBit(p.Payload, q, 0), int(r.Debut), x.reg, ctx)
		ecartV, desV = tv.EndBit-(suivant+1), tv.DesyncAt
	}
	ecartO := int(r.Debut) + int(r.Bits) - suivant
	x.lignes = append(x.lignes, fmt.Sprintf("V\t%s\t%d\t%d\t%d\t%v\t%d\t%d\t%v\t%d", c.court, p.TS, r.Vie.Slot,
		ecartO, r.Preuve == lecture.PreuveFerme, ecartCtrl, ecartV, desV < 0 && ecartV == 0, desV))
}
