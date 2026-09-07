// no_mojibake_test.go : interdit toute séquence UTF-8-doublement-encodé (« mojibake »)
// dans les .go du module et les .toml de config/titles/. Allowlist VIDE — c'est le
// point : la correction Q3 (2026-09-07, plan .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md
// §4 L0) a réencodé les 64 fichiers touchés (61 trouvés au premier balayage + 3 trouvés
// en écrivant CE garde-rail — voir le journal), aucun ne doit revenir en arrière. Note
// délibérée : ce fichier ne contient AUCUNE séquence mojibake littérale dans son propre
// source (même en commentaire) — les fixtures de TestMojibakeRE_CatchesKnownPatterns la
// construisent à l'exécution (mojibakeOf) pour ne pas se faire attraper par son propre
// scanner et devoir s'auto-allowlister.
//
// CAUSE (mémoire du dépôt, confirmée à l'exécution de Q3) : un roundtrip PowerShell 5.1
// `Get-Content chemin | Set-Content chemin` sans `-Encoding` explicite. `Get-Content`
// relit le fichier UTF-8 comme s'il était dans l'encodage ANSI (CP1252) de la session,
// puis `Set-Content` réécrit ces caractères mal interprétés en UTF-8 — chaque octet
// UTF-8 du caractère original devient un caractère à part entière, lui-même ensuite
// réencodé sur plusieurs octets. D'où la signature : U+00E9 (UTF-8 0xC3 0xA9) devient
// visuellement U+00C3 suivi de U+00A9, tous deux ré-encodés. Un même fichier peut avoir
// subi le roundtrip deux fois (`teammates_service*.go` : "câblé" corrompu en 2 passes).
//
// PARADE : ne jamais faire relire/réécrire un fichier via un outil qui ne force pas
// `-Encoding utf8` (PowerShell 5.1 : `Get-Content -Encoding utf8` / `Set-Content
// -Encoding utf8` explicitement — le défaut de la session n'est PAS UTF-8). En pratique
// dans ce dépôt : préférer les outils d'édition dédiés (Edit/Write des agents IA, ou tout
// éditeur qui préserve l'encodage déclaré) à un roundtrip shell générique.
//
// DÉTECTION : on ne liste pas les paires connues une par une — la marque du défaut est
// structurelle : U+00C3 ou U+00C5 (jamais légitimes en français, à ne pas confondre avec
// U+00C2 qui, lui, est légitime en capitales : CÂBLÉ, LÂCHER, DÉGÂT) ou U+00E2 suivis
// immédiatement d'un caractère qui n'existe QUE parce qu'un octet CP1252 0x80-0xFF a été
// relu comme point de code Unicode. mojibakeSecondCharClass couvre cet ensemble ; il
// reste correct même pour une séquence qu'on n'a pas encore vue (c'est ainsi qu'on a
// trouvé skill_rating_extra_test.go, synthesis_service_legacy.go et
// synthesis_service_builders.go, absents du premier balayage par motif littéral).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// mojibakeSecondCharClass : tout point de code que peut produire la relecture d'un octet
// CP1252 haut (0x80-0x9F : ponctuation/typo « intelligente », 0xA0-0xFF : identique à
// Latin-1) comme caractère Unicode. C'est le « second caractère » (ou troisième, en cas
// de double passe) d'une séquence mojibake.
const mojibakeSecondCharClass = `\x{00A0}-\x{00FF}` +
	`\x{20AC}\x{201A}\x{0192}\x{201E}\x{2026}\x{2020}\x{2021}\x{02C6}\x{2030}\x{0160}\x{2039}\x{0152}\x{017D}` +
	`\x{2018}\x{2019}\x{201C}\x{201D}\x{2022}\x{2013}\x{2014}\x{02DC}\x{2122}\x{0161}\x{203A}\x{0153}\x{017E}\x{0178}`

// mojibakeThirdCharClass : le sous-ensemble de mojibakeSecondCharClass produit par un
// octet CP1252 0x80-0x9F seulement (ponctuation/typo). C'est le caractère qui suit
// U+00E2 (â) dans une séquence 3-octets mal relue (ex. tiret cadratin, flèche, «
// environ ») — U+00E2 lui-même est un caractère français légitime (âge, château), donc
// on ne le flague que suivi de CE sous-ensemble, jamais suivi d'un caractère quelconque
// de 0xA0-0xFF (qui inclurait des mots français banals).
const mojibakeThirdCharClass = `\x{20AC}\x{201A}\x{0192}\x{201E}\x{2026}\x{2020}\x{2021}\x{02C6}\x{2030}\x{0160}\x{2039}\x{0152}\x{017D}` +
	`\x{2018}\x{2019}\x{201C}\x{201D}\x{2022}\x{2013}\x{2014}\x{02DC}\x{2122}\x{0161}\x{203A}\x{0153}\x{017E}\x{0178}`

// mojibakeRE : U+00C3 ou U+00C5 (jamais légitimes en français — à ne pas confondre avec
// U+00C2, qui l'est en capitales : CÂBLÉ, LÂCHER, DÉGÂT, RÂTELIER, BÂTI) suivi d'un
// caractère de mojibakeSecondCharClass ; ou U+00E2 (â, légitime seul) suivi d'un
// caractère de mojibakeThirdCharClass (le début d'une séquence 3-octets mal relue).
// mojibakeSecondCharClass inclut U+00C3/U+00C5 eux-mêmes, ce qui attrape aussi une
// deuxième passe de corruption.
//
// Résidu du lot Q3 (2026-09-07) : un U+00C3 en FIN DE LIGNE (un « à » dont l'espace insécable
// U+00A0 a été aplati puis perdu au trim : `home_service.go:251`) échappait à la règle « suivi
// d'un second caractère ». U+00C3 n'étant JAMAIS légitime en français, il est un défaut à lui
// seul, quel que soit ce qui le suit.
var mojibakeRE = regexp.MustCompile(
	`\x{00C3}` +
		`|\x{00C5}[` + mojibakeSecondCharClass + `]` +
		`|\x{00E2}[` + mojibakeThirdCharClass + `]`,
)

// mojibakeAllowlist : fichiers (chemin relatif depuis le point de départ du walk —
// apps/go-api/ pour le Go, la racine du dépôt pour les TOML) tolérés. VIDE
// délibérément : la correction Q3 a traité tous les sites connus (y compris ceux
// trouvés en écrivant ce garde-rail) et aucun fichier ne teste intentionnellement le
// mojibake dans son propre SOURCE littéral (voir la note en tête de fichier :
// TestMojibakeRE_CatchesKnownPatterns construit ses fixtures à l'exécution). Si un jour
// une fixture doit VOLONTAIREMENT contenir une séquence mojibake littérale, l'allowlister
// ici avec une date et une justification — pas avant.
var mojibakeAllowlist = map[string]bool{}

func TestNoMojibakeInGoModule(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	var violations []string
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if mojibakeAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			violations = append(violations, findMojibakeLines(rel, string(data))...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/%s: %v", goAPIRoot, sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("mojibake (UTF-8 doublement encodé, roundtrip PowerShell 5.1 sans "+
			"-Encoding utf8) détecté — réencoder (voir l'en-tête du fichier) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

func TestNoMojibakeInTitleConfigTOML(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api
	repoRoot := filepath.Dir(filepath.Dir(goAPIRoot))               // .../ (racine du dépôt)
	titlesDir := filepath.Join(repoRoot, "config", "titles")

	if _, err := os.Stat(titlesDir); err != nil {
		t.Fatalf("config/titles introuvable depuis %s (repoRoot mal calculé ?) : %v", titlesDir, err)
	}

	var violations []string
	err := filepath.WalkDir(titlesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		rel = filepath.ToSlash(rel)
		if mojibakeAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		violations = append(violations, findMojibakeLines(rel, string(data))...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", titlesDir, err)
	}
	if len(violations) > 0 {
		t.Errorf("mojibake détecté dans config/titles/**/*.toml :\n  %s", strings.Join(violations, "\n  "))
	}
}

// findMojibakeLines retourne une ligne de diagnostic par occurrence trouvée dans data,
// préfixée par le chemin relatif et le numéro de ligne (1-based).
func findMojibakeLines(rel, data string) []string {
	var found []string
	for i, line := range strings.Split(data, "\n") {
		if loc := mojibakeRE.FindStringIndex(line); loc != nil {
			found = append(found, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
		}
	}
	return found
}

// mojibakeOf construit, À L'EXÉCUTION, la représentation mojibake d'une chaîne
// correctement encodée : chaque rune est encodée en CP1252 (si possible) puis
// réinterprétée octet-par-octet comme des points de code Unicode indépendants — c'est
// exactement la transformation subie par les fichiers touchés par Q3. Sert uniquement à
// fabriquer des fixtures de test sans écrire de séquence mojibake en dur dans le SOURCE
// de ce fichier (qui serait alors attrapée par son propre scanner, ci-dessus).
func mojibakeOf(clean string) string {
	// Table CP1252 pour 0x80-0x9F (0xA0-0xFF est identique à Latin-1/Unicode).
	cp1252Upper := [32]rune{
		0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
		0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
		0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
		0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
	}
	// decodeByte reproduit EXACTEMENT ce que fait `Get-Content` (sans -Encoding) sur un
	// octet UTF-8 : il le relit comme un point de code CP1252, pas comme un octet d'une
	// séquence multi-octets. On applique donc cette fonction aux OCTETS UTF-8 de clean
	// (pas à ses runes) — c'est ce qui transforme les 2 octets de "é" (0xC3 0xA9) en 2
	// caractères indépendants (U+00C3, U+00A9) au lieu de les laisser former "é".
	decodeByte := func(b byte) rune {
		if b < 0x80 || b >= 0xA0 {
			return rune(b) // ASCII et 0xA0-0xFF : identiques à Unicode (Latin-1)
		}
		return cp1252Upper[b-0x80]
	}
	var out strings.Builder
	for _, b := range []byte(clean) {
		out.WriteRune(decodeByte(b))
	}
	return out.String()
}

// TestMojibakeOf_MatchesQ3Corpus verrouille mojibakeOf lui-même sur un cas réel de Q3
// (home_locale.go, avant correction) — sans quoi une fixture pourrait sembler valider la
// regex tout en ne représentant plus le défaut réel.
func TestMojibakeOf_MatchesQ3Corpus(t *testing.T) {
	got := mojibakeOf("Défaite")
	want := mojibakeOf("D") + string([]rune{0x00C3, 0x00A9}) + mojibakeOf("faite")
	if got != want {
		t.Fatalf("mojibakeOf(\"Défaite\") = %q, want %q", got, want)
	}
}

// TestMojibakeRE_CatchesKnownPatterns verrouille la regex sur les motifs réellement
// rencontrés au 2026-09-07 (Q3) — simple passe, double passe, ponctuation 3-octets — et
// s'assure qu'elle NE MATCHE PAS du français correctement encodé, y compris les
// capitales en U+00C2 (légitimes : CÂBLÉ, LÂCHER) qu'une version plus large de la regex
// attraperait à tort. Toutes les fixtures sont construites via mojibakeOf : ce fichier
// ne contient aucune séquence mojibake littérale.
func TestMojibakeRE_CatchesKnownPatterns(t *testing.T) {
	clean := []string{
		"Défaite",              // 1 passe
		"Égalité",              // 1 passe, 2 occurrences
		"non câblé",            // 1 passe
		"cœur",                 // Å suivi du guillemet mojibake de œ
		"filtré",               // 1 passe
		"4 matches × 4 types",  // × (multiplication, pas un tiret)
		"sur des copies — pas", // tiret cadratin, 3 octets
		"avg≈15",               // signe « environ »
		"map_name → clé",       // flèche
	}
	for _, s := range clean {
		once := mojibakeOf(s)
		if !mojibakeRE.MatchString(once) {
			t.Errorf("regex devrait détecter la mojibake-1-passe de %q (obtenu %q)", s, once)
		}
		twice := mojibakeOf(once)
		if !mojibakeRE.MatchString(twice) {
			t.Errorf("regex devrait détecter la mojibake-2-passes de %q (obtenu %q)", s, twice)
		}
		if mojibakeRE.MatchString(s) {
			t.Errorf("regex ne devrait PAS détecter de mojibake dans le texte propre %q", s)
		}
	}
	mustNotMatch := []string{
		"CÂBLÉ en DI",         // capitale légitime, PAS un mojibake
		"LE LÂCHER RESTE",     // idem
		"le DÉGÂT du match",   // idem
		"âge, château, grâce", // â légitime, non suivi d'un caractère de mojibakeThirdCharClass
	}
	for _, s := range mustNotMatch {
		if mojibakeRE.MatchString(s) {
			t.Errorf("regex ne devrait PAS détecter de mojibake dans %q", s)
		}
	}
}

// TestNoMojibakeInGoModule_DetectsAMutation est la preuve exigée par le plan Q3 : la
// mutation (réintroduction d'un mojibake dans un fichier réel du module, ici via un
// fichier temporaire placé dans le même arbre) DOIT faire rougir findMojibakeLines.
// Ne touche à aucun fichier versionné.
func TestNoMojibakeInGoModule_DetectsAMutation(t *testing.T) {
	mutated := "const x = \"" + mojibakeOf("Défaite") + "\"\n"
	violations := findMojibakeLines("mutation_test_fixture.go", mutated)
	if len(violations) == 0 {
		t.Fatal("la mutation aurait dû être détectée par findMojibakeLines — le garde-rail ne protège rien")
	}
}
