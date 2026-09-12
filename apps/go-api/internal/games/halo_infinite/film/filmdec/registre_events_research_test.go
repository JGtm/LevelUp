package filmdec

// registre_events_research_test.go — LOT D1 : Y A-T-IL UNE TABLE DE NOMS D'EVENEMENTS DANS LE
// FILM, A COTE DU REGISTRE DE COMPOSANTS ?
//
// L'HYPOTHESE TESTEE (utilisateur) : « la grammaire de la trame est deja decrite quelque part
// dans le film ». Elle est plausible parce que le film porte DEJA le registre de replication en
// clair : 1 067 couples (archetype, composant), noms ASCII lisibles a `slot+8`. Si le format
// declare ses composants, il peut declarer ses evenements.
//
// TROIS MESURES, DANS CET ORDRE :
//   D1a — les noms du catalogue de l'exe, cherches EN CLAIR dans tous les chunks du film.
//         Verdict binaire, sans seuil : ils y sont ou ils n'y sont pas.
//   D1b — la TABLE PAR TYPE trouvee juste apres le registre : suite d'entiers u32 courts qui
//         s'arrete pile a la chaine d'identification du build. On la publie entiere, avec les
//         noms du catalogue en regard, sous chacune des cardinalites candidates.
//   D1c — le meme chunk_00 sur des films de BUILDS DIFFERENTS : c'est la seule facon de dire si
//         cette table suit le build (donc si elle decrit la grammaire de CE build) ou non.
//
// Garde CHUNK00_FILMS (liste `;`). Aucun code de production touche.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// registryBlocks est le nombre de blocs d'archetype que porte REELLEMENT le registre du build
// de reference : le bloc 49 est le dernier porteur et la section suivante commence exactement a
// `50 * archetypeBlockSize`. Le « 118 » du dossier n'est pas un nombre d'archetypes : c'est
// `len(chunk_00) / archetypeBlockSize`, donc la taille du fichier entier divisee par la taille
// d'un bloc — les blocs 50+ ne sont pas du registre.
const registryBlocks = 50

// TestD1NomsEnClair cherche les noms d'evenements du catalogue de l'exe dans TOUS les chunks du
// film, en ASCII et en UTF-16LE. Aucun seuil : c'est une presence ou une absence.
func TestD1NomsEnClair(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, chunk00 := readChunk00(t, dir)
		total := map[string][2]int{} // nom -> {occurrences chunk_00, occurrences autres chunks}
		for _, nom := range eventNomsCibles {
			total[nom] = [2]int{comptesNom(chunk00, nom), 0}
		}
		n := CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, nom := range eventNomsCibles {
				v := total[nom]
				v[1] += comptesNom(data, nom)
				total[nom] = v
			}
		}
		t.Logf("=== %s === %d chunks hors chunk_00", filepath.Base(dir), n)
		trouve := 0
		for _, nom := range eventNomsCibles {
			v := total[nom]
			if v[0]+v[1] > 0 {
				trouve++
			}
			t.Logf("  %-36s chunk_00 : %d   autres chunks : %d", nom, v[0], v[1])
		}
		t.Logf("  BILAN : %d des %d noms du catalogue presents quelque part dans le film",
			trouve, len(eventNomsCibles))
		temoinNomsComposants(t, chunk00)
	}
}

// comptesNom compte les occurrences d'un nom en ASCII et en UTF-16LE, plus deux orthographes
// voisines (tirets au lieu de soulignes, suffixe `-component` du registre de composants).
func comptesNom(data []byte, nom string) int {
	variantes := []string{
		nom,
		strings.ReplaceAll(nom, "_", "-"),
		strings.ReplaceAll(nom, "_", "-") + "-component",
		strings.ReplaceAll(nom, "_", " "),
	}
	n := 0
	for _, v := range variantes {
		n += bytes.Count(data, []byte(v))
		n += bytes.Count(data, utf16le(v))
	}
	return n
}

// utf16le encode une chaine ASCII en UTF-16 petit-boutiste.
func utf16le(s string) []byte {
	out := make([]byte, 0, 2*len(s))
	for i := 0; i < len(s); i++ {
		out = append(out, s[i], 0)
	}
	return out
}

// temoinNomsComposants est le CONTROLE POSITIF de D1a : si la recherche par sous-chaine
// fonctionne, elle doit retrouver des noms de composant, qui sont eux notoirement en clair dans
// chunk_00. Sans ce temoin, une absence ne se distingue pas d'un instrument casse.
func temoinNomsComposants(t *testing.T, chunk00 []byte) {
	t.Helper()
	temoins := []string{
		"object-position-dynamic-precision", "unit-desired-aiming-vector",
		"weapon-state-type-info", "biped-spartan-ability-energy",
	}
	var parts []string
	for _, nom := range temoins {
		parts = append(parts, fmt.Sprintf("%s=%d", nom, bytes.Count(chunk00, []byte(nom))))
	}
	t.Logf("  TEMOIN POSITIF (noms de composant, en clair par construction) : %s",
		strings.Join(parts, " "))
}

// enteteChunk00 localise la section d'identification qui suit le registre : la chaine de version
// du build, dans un champ de 32 octets, precedee de la table d'entiers courts.
type enteteChunk00 struct {
	versionOff int      // offset de la chaine de version (debut du champ de 32 o)
	version    string   // ex. "6.10026.18411.0"
	build      string   // ex. "HI_1_13_0"
	saveur     string   // ex. "release"
	tableFin   int      // = versionOff : la table d'entiers s'arrete la
	valeurs    []uint32 // les entiers u32 lus a rebours puis remis dans l'ordre
	debutZeros int      // offset du premier octet non nul de la table
}

// lireEntete trouve la section d'identification et la table qui la precede.
func lireEntete(data []byte) (enteteChunk00, bool) {
	idx := bytes.Index(data, []byte("HI_"))
	if idx < 0 || idx < 0x20 {
		return enteteChunk00{}, false
	}
	e := enteteChunk00{versionOff: idx - 0x20}
	e.version = chaineChamp(data, e.versionOff)
	e.build = chaineChamp(data, idx)
	e.saveur = chaineChamp(data, idx+0x20)
	e.tableFin = e.versionOff
	off := e.tableFin
	for off >= 4 {
		v := leU32(data, off-4)
		if v > 0xffff {
			break
		}
		off -= 4
		e.valeurs = append(e.valeurs, v)
		if v == 0 && off >= 8 && leU32(data, off-4) == 0 && leU32(data, off-8) == 0 {
			break
		}
	}
	for i, j := 0, len(e.valeurs)-1; i < j; i, j = i+1, j-1 {
		e.valeurs[i], e.valeurs[j] = e.valeurs[j], e.valeurs[i]
	}
	e.debutZeros = off
	return e, true
}

// chaineChamp lit une chaine ASCII terminee par NUL a l'offset donne.
func chaineChamp(data []byte, off int) string {
	if off < 0 || off >= len(data) {
		return ""
	}
	end := min(off+0x20, len(data))
	raw := data[off:end]
	if z := bytes.IndexByte(raw, 0); z >= 0 {
		raw = raw[:z]
	}
	return string(raw)
}

// TestD1TableParType publie la table d'entiers qui precede l'identification du build, sous
// chacune des cardinalites candidates, avec les noms du catalogue en regard.
func TestD1TableParType(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		e, ok := lireEntete(data)
		if !ok {
			t.Logf("=== %s === aucune chaine de build : section d'identification absente",
				filepath.Base(dir))
			continue
		}
		t.Logf("=== %s ===", filepath.Base(dir))
		t.Logf("  fin du registre : bloc %d, offset 0x%06x", registryBlocks,
			registryBlocks*archetypeBlockSize)
		t.Logf("  identification @0x%06x : version=%q build=%q saveur=%q",
			e.versionOff, e.version, e.build, e.saveur)
		t.Logf("  suite d'apres @0x%06x : %08x %08x", e.versionOff+0x60,
			leU32(data, e.versionOff+0x60), leU32(data, e.versionOff+0x64))
		t.Logf("  table d'entiers : %d valeurs non nulles, de 0x%06x a 0x%06x (fin exclue)",
			len(e.valeurs), e.debutZeros, e.tableFin)
		t.Logf("  ecart entre la fin du registre et le debut de la table : %d octets",
			e.debutZeros-registryBlocks*archetypeBlockSize)
		histogramme(t, e.valeurs)
		suiteBrute(t, e.valeurs)
		for _, card := range []int{len(e.valeurs), 125, 128} {
			publierTable(t, e.valeurs, card)
		}
		ecrireTSV(t, e)
	}
}

// ecrireTSV depose la table par type dans le fichier designe par D1_TSV, quand la garde est
// posee. L'artefact sert de piece a confronter au tableau equivalent cote exe : c'est la seule
// facon de fixer l'index de chaque type nomme.
//
// LA PREMIERE VALEUR EST JETEE, ET C'EST DELIBERE : le walk arriere s'arrete sur le premier u32
// nul, qui appartient au bourrage du bloc de registre precedent et non a la table. Les valeurs
// suivantes sont les 123 entrees non nulles, prises comme les types 0..122.
func ecrireTSV(t *testing.T, e enteteChunk00) {
	t.Helper()
	chemin := os.Getenv("D1_TSV")
	if chemin == "" || len(e.valeurs) < 2 {
		return
	}
	var b strings.Builder
	b.WriteString("# table par type de chunk_00 — " + e.build + " " + e.version + "\n")
	b.WriteString("# valeurs u32 a 0x0CB208, une par type ; nom = catalogue statique de l'exe,\n")
	b.WriteString("# valable SEULEMENT si les deux espaces d'index coincident (non etabli).\n")
	b.WriteString("type\tvaleur\tnom_catalogue_exe\n")
	for i, v := range e.valeurs[1:] {
		b.WriteString(fmt.Sprintf("%d\t%d\t%s\n", i, v, nomDuType(i)))
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", chemin, err)
	}
	t.Logf("  table ecrite dans %s (%d types)", chemin, len(e.valeurs)-1)
}

// suiteBrute publie la table entiere, dans l'ordre du fichier. C'est CE MOTIF qu'un lot Ghidra
// pourra aligner sur le tableau equivalent cote exe pour fixer l'index de chaque type nomme :
// la suite est distinctive, la mettre en regard suffit.
func suiteBrute(t *testing.T, vals []uint32) {
	t.Helper()
	parts := make([]string, 0, len(vals))
	for _, v := range vals {
		parts = append(parts, fmt.Sprint(v))
	}
	t.Logf("  SUITE BRUTE (ordre du fichier, %d valeurs) : %s",
		len(vals), strings.Join(parts, ","))
}

// histogramme publie la distribution des valeurs de la table.
func histogramme(t *testing.T, vals []uint32) {
	t.Helper()
	h := map[uint32]int{}
	for _, v := range vals {
		h[v]++
	}
	var ks []uint32
	for k := range h {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	var parts []string
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("%d:%d", k, h[k]))
	}
	t.Logf("  distribution des valeurs (valeur:effectif) : %s", strings.Join(parts, " "))
}

// publierTable affiche les entrees != 1 sous l'hypothese « la table compte `card` entrees et son
// index 0 est le type 0 » : c'est cette hypothese que les noms en regard permettent de juger.
func publierTable(t *testing.T, vals []uint32, card int) {
	t.Helper()
	if card <= 0 {
		return
	}
	decal := card - len(vals) // index du premier element mesure
	var parts []string
	for i, v := range vals {
		if v == 1 {
			continue
		}
		ti := i + decal
		parts = append(parts, fmt.Sprintf("[%d]=%d %s", ti, v, nomDuType(ti)))
	}
	t.Logf("  HYPOTHESE cardinal=%d (index 0 = type 0) — entrees != 1 : %s",
		card, strings.Join(parts, " · "))
}

// TestD1Builds groupe des films par BUILD : empreinte du registre, chaine de build, empreinte de
// la table par type. Si la table suit le build, elle decrit la grammaire de ce build — ce que le
// lot cherche a etablir. Garde D1_BUILD_DIR : repertoire de cache des films.
func TestD1Builds(t *testing.T) {
	racine := os.Getenv("D1_BUILD_DIR")
	if racine == "" {
		t.Skipf("D1_BUILD_DIR absent : instrument saute")
	}
	entries, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	groupes := map[string]*groupeBuild{}
	lus := 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := filepath.Join(racine, ent.Name())
		raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
		if err != nil {
			continue
		}
		lus++
		data := filmsource.Inflate(raw)
		cle, g := classerFilm(data)
		if groupes[cle] == nil {
			groupes[cle] = g
		}
		if len(groupes[cle].films) < 4 {
			groupes[cle].films = append(groupes[cle].films, ent.Name())
		} else {
			groupes[cle].films[3] = "..."
		}
		groupes[cle].blocs++
	}
	t.Logf("%d chunk_00 lus dans %s ; %d groupes distincts", lus, racine, len(groupes))
	var cles []string
	for k := range groupes {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return groupes[cles[i]].blocs > groupes[cles[j]].blocs })
	for _, k := range cles {
		g := groupes[k]
		t.Logf("  %-60s : %4d films (ex. %s)", k, g.blocs, strings.Join(g.films, " "))
		t.Logf("      empreinte registre 0x%016x ; table de %d valeurs", g.empr, len(g.valeurs))
	}
}

// groupeBuild rassemble les films qui partagent la meme identification de build ET la meme
// table par type.
type groupeBuild struct {
	films   []string
	valeurs []uint32
	empr    uint64
	blocs   int
}

// classerFilm rend la cle de groupement d'un chunk_00 : build + cardinal et somme de la table.
func classerFilm(data []byte) (string, *groupeBuild) {
	reg := parseRegistry(data)
	e, ok := lireEntete(data)
	somme := uint64(0)
	for _, v := range e.valeurs {
		somme = somme*1099511628211 ^ uint64(v)
	}
	cle := fmt.Sprintf("build=%q version=%q table=%d/%016x", e.build, e.version, len(e.valeurs), somme)
	if !ok {
		cle = "SANS SECTION D'IDENTIFICATION"
	}
	return cle, &groupeBuild{valeurs: e.valeurs, empr: RegistryFingerprint(reg)}
}
