package mapvar

// label_hunt_test.go — INSTRUMENT sous garde : la chasse aux labels non résolus.
//
// La table `labelNames` s'est construite par recherche exhaustive d'identifiants snake_case
// dont le murmur3 retombe sur un hash observé (cf. objectives.go). Cet instrument rejoue
// cette recherche, BORNÉE à des radicaux de mode donnés, sur un dossier de `.mvar` — et
// publie la liste testée, pour que le journal puisse dire ce qui a été essayé, pas
// seulement ce qui a été trouvé.
//
// Il ne tourne que si MAPVAR_LABEL_HUNT_DIR pointe un dossier de `.mvar` (jamais en CI).
// Radicaux et suffixes se passent par MAPVAR_LABEL_HUNT_RADICALS / _SUFFIXES (séparés par
// des virgules) ; à défaut, la liste du lot C-ter volet 2 (colline de KOTH) est jouée.
//
// GARDE-FOU (le même qu'objectives.go) : une correspondance n'est PAS une résolution. Un
// hash trouvé ici entre dans labelNames seulement s'il est sémantiquement cohérent ET
// confirmé par la géométrie ou un témoin d'exécution.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// huntDefaultRadicals — les radicaux essayés pour la colline de KOTH (lot C-ter volet 2).
var huntDefaultRadicals = []string{
	"koth", "koth_hill", "koth_zone", "koth_hill_zone", "koth_zone_include", "koth_hills",
	"hill", "hill_zone", "hills", "hill_include", "hill_zone_include",
	"king_of_the_hill", "king_of_the_hill_hill", "king_of_the_hill_zone", "kingofthehill", "king",
	"crown", "crown_include", "crown_zone",
	"zone", "zones", "capture_zone", "capture_hill", "control_zone", "control_point",
	"territory", "territories", "territory_zone",
	"total_control", "total_control_zone", "land_grab", "land_grab_zone", "landgrab",
	"koth_neutral", "koth_moving", "moving_hill", "rotating_hill", "sequence_hill", "hill_sequence",
	"koth_sequence", "koth_order", "hill_order", "hill_marker", "koth_marker",
	"crown_hill", "the_hill", "objective_hill", "objective_zone", "mode_zone",
}

// huntDefaultSuffixes — les suffixes composés à chaque radical (le radical nu est toujours
// essayé aussi).
var huntDefaultSuffixes = []string{
	"_include", "_exclude", "_zone", "_spawn", "_hill", "_area", "_volume", "_marker",
	"_navpoint", "_socket", "_objective", "_capture", "_point", "_1", "_2", "_3", "_a", "_b", "_c",
	"_01", "_02", "_03", "_include_zone", "_zone_include", "_neutral", "_neutral_include",
	"_active", "_inactive", "_default", "_multi", "_multi_exclude", "_neutral_exclude",
}

// TestHuntLabels — voir l'en-tête. Sortie : pour chaque candidat trouvé, le hash, le nom,
// et le nombre d'objets par fichier ; puis la taille de la liste testée.
func TestHuntLabels(t *testing.T) {
	dir := os.Getenv("MAPVAR_LABEL_HUNT_DIR")
	if dir == "" {
		t.Skip("MAPVAR_LABEL_HUNT_DIR non posé : instrument sous garde")
	}
	radicals := huntDefaultRadicals
	if s := os.Getenv("MAPVAR_LABEL_HUNT_RADICALS"); s != "" {
		radicals = strings.Split(s, ",")
	}
	suffixes := huntDefaultSuffixes
	if s := os.Getenv("MAPVAR_LABEL_HUNT_SUFFIXES"); s != "" {
		suffixes = strings.Split(s, ",")
	}
	observed := huntObservedHashes(t, dir) // hash -> fichier -> nb objets
	if len(observed) == 0 {
		t.Fatalf("aucun label non résolu dans %s", dir)
	}
	candidates := huntCandidates(radicals, suffixes)
	found := 0
	for _, c := range candidates {
		h := LabelHash(c)
		perFile, ok := observed[h]
		if !ok {
			continue
		}
		found++
		files := make([]string, 0, len(perFile))
		for f, n := range perFile {
			files = append(files, f+"="+itoa(n))
		}
		sort.Strings(files)
		t.Logf("TROUVE %d = %q : %s", h, c, strings.Join(files, " "))
	}
	t.Logf("candidats testés : %d (radicaux %d x suffixes %d + radicaux nus) ; hashs non résolus observés : %d ; trouvés : %d",
		len(candidates), len(radicals), len(suffixes), len(observed), found)
	t.Logf("liste testée : %s", strings.Join(candidates, " "))
}

// huntObservedHashes relève les hashs non résolus de chaque `.mvar` du dossier.
func huntObservedHashes(t *testing.T, dir string) map[int32]map[string]int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	out := map[int32]map[string]int{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".mvar") {
			continue
		}
		buf, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		v, err := Parse(buf)
		if err != nil {
			t.Logf("parse %s : %v (ignoré)", e.Name(), err)
			continue
		}
		for h, n := range v.UnresolvedLabels() {
			if out[h] == nil {
				out[h] = map[string]int{}
			}
			out[h][e.Name()] += n
		}
	}
	return out
}

// huntCandidates compose radicaux et suffixes, dédupliqués, dans un ordre stable.
func huntCandidates(radicals, suffixes []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, r := range radicals {
		add(r)
		for _, s := range suffixes {
			add(r + s)
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// TestHuntLabelsBrute — même instrument, en FORCE BRUTE sur le radical : toutes les chaînes
// sur l'alphabet [a-z_] jusqu'à MAPVAR_LABEL_HUNT_BRUTE_LEN caractères, filtrées d'abord par
// « radical + _include » (le filtre de mode que tout objet de mode porte), puis confirmées
// par un second suffixe de rôle. Une correspondance DOUBLE (deux hashs distincts observés,
// deux suffixes du même radical) n'est pas une coïncidence : 29 hashs cibles sur 2^32,
// deux fois de suite, c'est 1e-16 par radical.
//
// Coût : 27^N hashs. N = 6 : ~6 s ; N = 7 : ~3 min ; N = 8 : > 1 h — ne pas dépasser 7 sans
// raison écrite.
func TestHuntLabelsBrute(t *testing.T) {
	dir := os.Getenv("MAPVAR_LABEL_HUNT_DIR")
	maxLen := os.Getenv("MAPVAR_LABEL_HUNT_BRUTE_LEN")
	if dir == "" || maxLen == "" {
		t.Skip("MAPVAR_LABEL_HUNT_DIR / MAPVAR_LABEL_HUNT_BRUTE_LEN non posés : instrument sous garde")
	}
	n := 0
	for _, ch := range maxLen {
		n = n*10 + int(ch-'0')
	}
	if n < 1 || n > 8 {
		t.Fatalf("longueur %d hors [1, 8]", n)
	}
	observed := huntObservedHashes(t, dir)
	roleSuffixes := []string{"_zone", "_hill", "_area", "_volume", "_capture", "_point", "_region",
		"_territory", "_spot", "_hill_zone", "_zone_hill", "_control", "_capture_zone", "_objective",
		"_goal", "_target", "_pad", "_platform", "_ring", "_exclude", "_spawn", "_navpoint", "_marker",
		"_socket", "_flag", "_bomb", "_ball", "_skull"}
	const alphabet = "abcdefghijklmnopqrstuvwxyz_"
	buf := make([]byte, 0, n+16)
	tested := 0
	var walk func(depth int)
	walk = func(depth int) {
		if depth > 0 {
			tested++
			inc := LabelHash(string(append(append(buf[:0:0], buf...), "_include"...)))
			if perFile, ok := observed[inc]; ok {
				radical := string(buf)
				t.Logf("filtre : %q + _include = %d (%v)", radical, inc, perFile)
				for _, s := range roleSuffixes {
					h := LabelHash(radical + s)
					if pf, ok2 := observed[h]; ok2 {
						t.Logf("CONFIRME %d = %q (%v)", h, radical+s, pf)
					}
				}
			}
		}
		if depth == n {
			return
		}
		for i := 0; i < len(alphabet); i++ {
			buf = append(buf, alphabet[i])
			walk(depth + 1)
			buf = buf[:len(buf)-1]
		}
	}
	walk(0)
	t.Logf("radicaux testés : %d (alphabet %d, longueur <= %d)", tested, len(alphabet), n)
}
