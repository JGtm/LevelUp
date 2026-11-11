//go:build research

package divergence

// faitsdiff_research_test.go : DIFFERENTIEL DE DEUX FICHIERS DE FAITS (`<id>.filmfacts.bin`), canal par
// canal (plan marche complete, A2.P, expliquer les ecarts du gate de non-regression). Chaque canal de
// `replay.FilmFacts` est descendu recursivement ; chaque tranche est reduite a un multi-ensemble de
// lignes texte (pointeurs suivis), et le rapport donne, par chemin, les comptes des deux cotes et
// quelques elements propres a chaque cote.
//
//	FAITS_A=<avant.filmfacts.bin> FAITS_CARTE_A=<catalogue de l avant.json> \
//	FAITS_B=<apres.filmfacts.bin> FAITS_CARTE_B=<catalogue de l apres.json> \
//	FAITS_SORTIE=<rapport.txt> [FAITS_EXEMPLES=8] go test -tags=research ./research/divergence/ -run TestFaitsDiff

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func lireFaits(t *testing.T, chemin, carte string) *replay.FilmFactsFile {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	ent, err := replay.DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(carte)
	if err != nil {
		t.Fatal(err)
	}
	var derniere error
	for _, e := range entreesDuNiveau(cat, ent.MapModule) {
		// Les faits cuits avant le patch de la cle de A2.P ne hachaient pas la table par region : les deux formes
		// sont essayees, sur chaque entree du module (les entrees d un module ne portent pas toutes les memes bornes).
		for _, avecRegions := range []bool{true, false} {
			x := e
			if !avecRegions {
				x.Regions = nil
			}
			f, err := replay.DecodeFilmFactsFile(blob, x)
			if err == nil {
				return f
			}
			derniere = err
		}
	}
	t.Fatalf("%s : %v", chemin, derniere)
	return nil
}

// texte rend une valeur en texte, pointeurs suivis.
func texte(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		return texte(v.Elem())
	case reflect.Struct:
		var b strings.Builder
		b.WriteByte('{')
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(v.Type().Field(i).Name + ":" + texte(v.Field(i)))
		}
		b.WriteByte('}')
		return b.String()
	case reflect.Slice, reflect.Array:
		parts := make([]string, v.Len())
		for i := range parts {
			parts[i] = texte(v.Index(i))
		}
		return "[" + strings.Join(parts, " ") + "]"
	case reflect.Map:
		keys := v.MapKeys()
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = texte(k) + ":" + texte(v.MapIndex(k))
		}
		sort.Strings(parts)
		return "map[" + strings.Join(parts, " ") + "]"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func soustraire(a, b []string) []string {
	mb := make(map[string]int, len(b))
	for _, s := range b {
		mb[s]++
	}
	var out []string
	for _, s := range a {
		if mb[s] > 0 {
			mb[s]--
			continue
		}
		out = append(out, s)
	}
	return out
}

type rapporteur struct {
	w  strings.Builder
	ex int
}

func (r *rapporteur) descendre(chemin string, a, b reflect.Value) {
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if a.Type().Field(i).IsExported() {
				r.descendre(chemin+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
			}
		}
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				fmt.Fprintf(&r.w, "## %s : nil %v -> %v\n", chemin, a.IsNil(), b.IsNil())
			}
			return
		}
		r.descendre(chemin, a.Elem(), b.Elem())
	case reflect.Slice:
		la, lb := make([]string, a.Len()), make([]string, b.Len())
		for i := range la {
			la[i] = texte(a.Index(i))
		}
		for i := range lb {
			lb[i] = texte(b.Index(i))
		}
		seulA, seulB := soustraire(la, lb), soustraire(lb, la)
		if len(seulA) == 0 && len(seulB) == 0 {
			return
		}
		fmt.Fprintf(&r.w, "## %s : avant %d, apres %d ; seulement avant %d, seulement apres %d\n", chemin, len(la), len(lb), len(seulA), len(seulB))
		sort.Strings(seulA)
		sort.Strings(seulB)
		for k := 0; k < len(seulA) && k < r.ex; k++ {
			fmt.Fprintf(&r.w, "   - %.700s\n", seulA[k])
		}
		for k := 0; k < len(seulB) && k < r.ex; k++ {
			fmt.Fprintf(&r.w, "   + %.700s\n", seulB[k])
		}
	default:
		sa, sb := texte(a), texte(b)
		if sa != sb {
			fmt.Fprintf(&r.w, "## %s : %.300s -> %.300s\n", chemin, sa, sb)
		}
	}
}

func TestFaitsDiff(t *testing.T) {
	a, ca, b, cb, sortie := os.Getenv("FAITS_A"), os.Getenv("FAITS_CARTE_A"), os.Getenv("FAITS_B"),
		os.Getenv("FAITS_CARTE_B"), os.Getenv("FAITS_SORTIE")
	if a == "" || b == "" || ca == "" || cb == "" || sortie == "" {
		t.Skip("FAITS_A, FAITS_CARTE_A, FAITS_B, FAITS_CARTE_B et FAITS_SORTIE requis")
	}
	refuserSortieSousData(t, sortie)
	r := &rapporteur{ex: 8}
	if v, err := strconv.Atoi(os.Getenv("FAITS_EXEMPLES")); err == nil {
		r.ex = v
	}
	fa, fb := lireFaits(t, a, ca), lireFaits(t, b, cb)
	r.descendre("Facts", reflect.ValueOf(fa.Facts.FilmInputs), reflect.ValueOf(fb.Facts.FilmInputs))
	fmt.Fprintf(&r.w, "## replis du balayage : avant %d apres %d ; kills avant %v apres %v\n",
		len(fa.Fallbacks), len(fb.Fallbacks), fa.Kills != nil, fb.Kills != nil)
	if err := os.WriteFile(sortie, []byte(r.w.String()), 0o644); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
}

// TestModuleDesFaits : le module de carte que porte l en-tete d un fichier de faits (FAITS_A).
func TestModuleDesFaits(t *testing.T) {
	a := os.Getenv("FAITS_A")
	if a == "" {
		t.Skip("FAITS_A requis")
	}
	blob, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	ent, err := replay.DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("MODULE %s\n", ent.MapModule)
}

// TestFaitsFiltre : tous les elements (des deux cotes) des canaux dont la ligne contient FAITS_FILTRE ;
// FAITS_CANAL (sous-chaine du chemin du canal) restreint les canaux. Meme environnement que TestFaitsDiff.
func TestFaitsFiltre(t *testing.T) {
	a, ca, b, cb, sortie, filtre := os.Getenv("FAITS_A"), os.Getenv("FAITS_CARTE_A"), os.Getenv("FAITS_B"),
		os.Getenv("FAITS_CARTE_B"), os.Getenv("FAITS_SORTIE"), os.Getenv("FAITS_FILTRE")
	if a == "" || b == "" || filtre == "" {
		t.Skip("FAITS_A, FAITS_B, FAITS_FILTRE requis")
	}
	refuserSortieSousData(t, sortie)
	canal := os.Getenv("FAITS_CANAL")
	fa, fb := lireFaits(t, a, ca), lireFaits(t, b, cb)
	var w strings.Builder
	collecterFiltre(&w, "Facts", "A", reflect.ValueOf(fa.Facts.FilmInputs), canal, filtre)
	collecterFiltre(&w, "Facts", "B", reflect.ValueOf(fb.Facts.FilmInputs), canal, filtre)
	lignes := strings.Split(strings.TrimSpace(w.String()), "\n")
	sort.Slice(lignes, func(i, j int) bool { return cleTS(lignes[i]) < cleTS(lignes[j]) })
	if err := os.WriteFile(sortie, []byte(strings.Join(lignes, "\n")+"\n"), 0o644); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
}

// cleTS extrait le premier TimestampUS d une ligne pour le tri (0 sinon).
func cleTS(l string) uint64 {
	const m = "TimestampUS:"
	i := strings.Index(l, m)
	if i < 0 {
		return 0
	}
	j := i + len(m)
	k := j
	for k < len(l) && l[k] >= '0' && l[k] <= '9' {
		k++
	}
	n, _ := strconv.ParseUint(l[j:k], 10, 64)
	return n
}

// collecterFiltre descend dans les structures et ecrit chaque element de tranche dont la ligne contient filtre.
func collecterFiltre(w *strings.Builder, chemin, cote string, v reflect.Value, canal, filtre string) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			collecterFiltre(w, chemin, cote, v.Elem(), canal, filtre)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				collecterFiltre(w, chemin+"."+v.Type().Field(i).Name, cote, v.Field(i), canal, filtre)
			}
		}
	case reflect.Slice:
		if canal != "" && !strings.Contains(chemin, canal) {
			return
		}
		for k := 0; k < v.Len(); k++ {
			if s := texte(v.Index(k)); strings.Contains(s, filtre) {
				fmt.Fprintf(w, "%s\t%s\t%.600s\n", chemin, cote, s)
			}
		}
	}
}
