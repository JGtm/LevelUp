// Package archlint — no_absolute_path_in_versioned_catalog_test.go : UN CATALOGUE VERSIONNÉ NE
// PORTE PAS LE CHEMIN D'INSTALLATION DU POSTE QUI L'A PRODUIT (découverte D2 (3.1.2),
// 2026-09-16 ; traité au lot 2.10.2, 2026-09-17).
//
// # Le défaut que ce ratchet ferme
//
// `data/titles/halo_infinite/reference/map_quant_bounds.json` est un fichier VERSIONNÉ, dérivé
// des `.module` du jeu installé par `cmd/mapquant-build`. Son champ `source` citait le chemin
// ABSOLU du poste de fabrication :
//
//	"… lus dans D:\<bibliotheque>\<jeux>\common\Halo Infinite\deploy\ds\levels\multi"
//
// Conséquence : deux postes qui régénèrent LE MÊME catalogue produisent deux fichiers
// différents alors qu'aucune borne n'a bougé. Tout gate « commis = régénéré » à l'octet — le
// modèle de `TestCatalogueCommisEgaleCatalogueRegenere` — rougit alors pour une trace de
// fabrication, jamais pour une donnée. Le lot 3.1.2 l'avait contourné en excluant `source` de
// l'empreinte (`TestEmpreinteDesBornesIgnoreLaTraceDeFabrication`) ; 2.10.2 en a retiré la
// cause, et ce ratchet empêche la cause de revenir par un autre catalogue.
//
// Accessoirement : un chemin absolu publie le nom d'utilisateur et l'arborescence de disque de
// qui a lancé l'outil.
//
// # Ce que le ratchet vérifie
//
// Tout fichier TEXTE sous `data/titles/*/reference/` (hors `generated/`, non versionné) ne
// contient aucune forme de chemin absolu de poste : lettre de lecteur (`D:\`, `C:/`), chemin
// MSYS (`/c/Users/`), foyer POSIX (`/home/`, `/Users/`), partage réseau (`\\hote\partage\`),
// racines Unix (`/opt/`, `/mnt/`, `/tmp/`, `/var/`, `/Volumes/`). L'allowlist est DATÉE et nommée, et
// `TestAllowlistDesCheminsAbsolusNEstPasPerimee` la fait rougir dès qu'une de ses entrées n'a
// plus d'objet — catalogue disparu, ou redevenu propre.
package archlint

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// cheminsAbsolusDePoste : les formes d'un chemin absolu propre à une machine. La lettre de
// lecteur exige de n'être précédée d'aucun caractère de mot — sans quoi `https://` passerait
// pour un `s:/`.
var cheminsAbsolusDePoste = []*regexp.Regexp{
	regexp.MustCompile(`(^|[^A-Za-z0-9])[A-Za-z]:[\\/]`),
	regexp.MustCompile(`/[a-z]/Users/`),
	regexp.MustCompile(`/home/`),
	regexp.MustCompile(`/Users/`),
	regexp.MustCompile(`\\{1,2}Users\\{1,2}`), // dossier utilisateur Windows, brut ou échappé dans un JSON
	// Partage réseau `\\hote\partage\`, brut ou échappé dans un JSON : deux à quatre barres inverses
	// au début, hors d'un mot (« a\\b\\c » est un chemin relatif échappé).
	regexp.MustCompile(`(^|[^\\\w])\\{2,4}[^\\/\s"]+\\{1,2}[^\\/\s"]+\\{1,2}`),
	// Racines Unix hors dossier personnel, au début d'un mot de chemin : « any/var/ » n'en est pas une.
	regexp.MustCompile(`(^|[^\w/.-])/(?:opt|mnt|tmp|var|Volumes)/`),
}

// extensionsTexteCatalogue : les catalogues qui se lisent comme du texte. Les `.webp` et autres
// binaires du même dossier sont hors sujet — un octet quelconque y déclencherait un faux rouge.
var extensionsTexteCatalogue = map[string]bool{
	".json": true, ".csv": true, ".tsv": true, ".txt": true, ".toml": true, ".md": true,
}

// catalogesAvecCheminAbsoluTolere : la DETTE MESURÉE le 2026-09-17, chemins relatifs à la racine
// du dépôt. Un fichier, produit par un outil HORS de la frontière du lot 2.10 — le corriger
// demande de toucher son fabricant, ce que la règle « zéro fix opportuniste » interdit ici.
// Consigné au §4 du plan décodeur (D1 (2.10)).
//
//   - data/titles/halo_infinite/reference/map_callouts.json : le champ `source` cite
//     `C:\Program Files (x86)\Steam\…` — le même défaut que `map_quant_bounds.json`, sur un
//     autre poste. Producteur : la chaîne des zones (`cmd/callouts-*`).
//
// Les fonds de carte (`map_backgrounds/*.json`) n'ont PAS leur place ici : `cmd/mapfond-build`
// réduit chaque dégradation au nom de son module avant d'écrire le fichier
// (`TestFondsDeCartePublies_SansCheminAbsolu`).
//
// AGRANDIR CETTE LISTE EXIGE LA MÊME DÉMONSTRATION : pourquoi le chemin ne peut pas être
// relatif, et la date. Un nouveau catalogue n'y a pas sa place — il naît propre.
var catalogesAvecCheminAbsoluTolere = map[string]bool{
	"data/titles/halo_infinite/reference/map_callouts.json": true,
}

// TestCatalogueVersionneSansCheminAbsolu — aucun catalogue versionné ne cite le disque de son
// fabricant.
//
// Mutation qui doit le faire rougir : remettre un chemin absolu dans le champ `source` de
// `map_quant_bounds.json` (jouée le 2026-09-17).
func TestCatalogueVersionneSansCheminAbsolu(t *testing.T) {
	racine := racineDuDepot(t)
	vus := 0
	for _, dir := range dossiersReference(t, racine) {
		err := filepath.WalkDir(dir, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "generated" { // sortie du runtime, hors de git
					return fs.SkipDir
				}
				return nil
			}
			if !extensionsTexteCatalogue[strings.ToLower(filepath.Ext(chemin))] {
				return nil
			}
			rel, _ := filepath.Rel(racine, chemin)
			rel = filepath.ToSlash(rel)
			vus++
			if catalogesAvecCheminAbsoluTolere[rel] {
				return nil
			}
			buf, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if rerr != nil {
				return rerr
			}
			if m := premierCheminAbsolu(buf); m != nil {
				t.Errorf("%s porte un chemin absolu de poste (%q) — un fichier VERSIONNÉ ne "+
					"doit citer que la MÉTHODE et un chemin relatif ; sinon deux postes "+
					"régénèrent deux fichiers différents pour les mêmes données",
					rel, strings.TrimSpace(string(m)))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", dir, err)
		}
	}
	if vus == 0 {
		t.Fatal("aucun catalogue texte balayé — le balayage s'est cassé")
	}
	t.Logf("%d catalogue(s) texte balayé(s), %d toléré(s)", vus, len(catalogesAvecCheminAbsoluTolere))
}

// TestFondsDeCartePublies_SansCheminAbsolu — aucun fond de carte publié
// (`data/titles/*/reference/map_backgrounds/*.json`) ne cite le disque du poste de cuisson :
// une dégradation recopie le message d'une brique, qui porte le chemin du module absent. Le
// test ne passe PAS par l'allowlist : un fond ne s'y ajoute jamais.
//
// Mutation qui doit le faire rougir : remettre un chemin `C:\…` dans une dégradation d'un fond.
func TestFondsDeCartePublies_SansCheminAbsolu(t *testing.T) {
	racine := racineDuDepot(t)
	vus := 0
	for _, dir := range dossiersReference(t, racine) {
		fonds, err := filepath.Glob(filepath.Join(dir, "map_backgrounds", "*.json"))
		if err != nil {
			t.Fatalf("balayage de %s : %v", dir, err)
		}
		for _, chemin := range fonds {
			buf, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if rerr != nil {
				t.Fatalf("lecture de %s : %v", chemin, rerr)
			}
			vus++
			if m := premierCheminAbsolu(buf); m != nil {
				rel, _ := filepath.Rel(racine, chemin)
				t.Errorf("%s porte un chemin absolu de poste (%q) — une dégradation ne doit citer "+
					"que le nom du module (cmd/mapfond-build, sansCheminAbsolu)",
					filepath.ToSlash(rel), strings.TrimSpace(string(m)))
			}
		}
	}
	if vus == 0 {
		t.Fatal("aucun fond de carte publié balayé — le balayage s'est cassé")
	}
	t.Logf("%d fond(s) de carte balayé(s)", vus)
}

// TestAllowlistDesCheminsAbsolusNEstPasPerimee : une entrée de `catalogesAvecCheminAbsoluTolere`
// qui désigne un fichier disparu, ou un catalogue qui ne porte PLUS de chemin absolu, se RETIRE
// — dans le commit même qui la résout. Une allowlist périmée finit par autoriser autre chose que
// ce qu'elle a été posée pour tolérer (même règle que les autres ratchets de ce paquet :
// `TestPlafondsDeTailleNeSontPasPerimes`, `TestAllowlistsDesCouchesNeSontPasPerimees`).
//
// Mutation qui doit le faire rougir : ajouter à la table une entrée sans objet — un catalogue
// propre, ou un chemin qui n'existe pas (jouée le 2026-09-16).
func TestAllowlistDesCheminsAbsolusNEstPasPerimee(t *testing.T) {
	racine := racineDuDepot(t)
	entrees := make([]string, 0, len(catalogesAvecCheminAbsoluTolere))
	for rel := range catalogesAvecCheminAbsoluTolere {
		entrees = append(entrees, rel)
	}
	sort.Strings(entrees)
	for _, rel := range entrees {
		buf, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(rel))) //nolint:gosec // chemin de test, lecture seule
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s est toléré mais n'existe plus (renommé, déplacé ou supprimé) : retirer "+
				"l'entrée de catalogesAvecCheminAbsoluTolere, ou la réécrire au nouveau chemin.",
				rel)
			continue
		}
		if err != nil {
			t.Fatalf("lecture de %s : %v", rel, err)
		}
		if premierCheminAbsolu(buf) == nil {
			t.Errorf("%s ne porte plus de chemin absolu de poste — retirer l'entrée de "+
				"catalogesAvecCheminAbsoluTolere. Le catalogue est rentré dans la règle, la "+
				"table n'a plus à le connaître.", rel)
		}
	}
	t.Logf("%d entrée(s) tolérée(s) vérifiée(s)", len(entrees))
}

// TestFormesDeCheminAbsoluReconnues fige ce que les motifs voient et ne voient pas : chaque forme
// de chemin de poste citée est reconnue, brute ou échappée dans un JSON ; un chemin relatif, une
// adresse web et un mot qui contient « var » ou « tmp » restent libres.
func TestFormesDeCheminAbsoluReconnues(t *testing.T) {
	vus := []string{
		`lu dans D:\Jeux\Halo\deploy`,
		`lu dans C:/Jeux/Halo`,
		`"source": "C:\\Jeux\\Halo Infinite\\x.module"`,
		`lu dans \\hote\partage\Halo\x.module`,
		`"source": "\\\\hote\\partage\\Halo\\x.module"`,
		`/opt/halo/deploy`,
		`lu dans /opt/halo/deploy`,
		`"/mnt/d/Jeux/Halo"`,
		`(/tmp/work/x.module)`,
		`lu dans /var/lib/halo`,
		`lu dans /Volumes/Jeux/Halo`,
		`/Users/Jean Dupont/Halo`,
		`C:\Users\Jean D.\Halo`,
	}
	for _, s := range vus {
		if premierCheminAbsolu([]byte(s)) == nil {
			t.Errorf("chemin de poste non reconnu : %q", s)
		}
	}
	libres := []string{
		`any/levels/multi/globals`,
		`https://example.org/var/lib/x`,
		`module any/var/tmp/opt introuvable`,
		`un chemin relatif a\\b\\c\\d echappe`,
		`partie \\ seule`,
		`ratio 1/opt/ ou x/mnt/y`,
	}
	for _, s := range libres {
		if m := premierCheminAbsolu([]byte(s)); m != nil {
			t.Errorf("faux positif sur %q : %q", s, m)
		}
	}
}

// premierCheminAbsolu rend la première forme de chemin absolu de poste trouvée dans le contenu,
// ou nil. UNE seule lecture des motifs pour les deux tests du fichier : le ratchet et son test
// de péremption doivent juger sur EXACTEMENT le même critère, sans quoi une entrée pourrait être
// « périmée » pour l'un et nécessaire pour l'autre.
func premierCheminAbsolu(contenu []byte) []byte {
	for _, re := range cheminsAbsolusDePoste {
		if m := re.Find(contenu); m != nil {
			return m
		}
	}
	return nil
}

// racineDuDepot rend la racine du dépôt, déduite de l'emplacement de ce fichier.
func racineDuDepot(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	return filepath.Dir(filepath.Dir(goAPIRoot))               // .../<racine>
}

// dossiersReference rend les `data/titles/*/reference` existants.
func dossiersReference(t *testing.T, racine string) []string {
	t.Helper()
	titres, err := os.ReadDir(filepath.Join(racine, "data", "titles"))
	if err != nil {
		t.Fatalf("lecture de data/titles : %v", err)
	}
	var dirs []string
	for _, e := range titres {
		if !e.IsDir() {
			continue
		}
		d := filepath.Join(racine, "data", "titles", e.Name(), "reference")
		if st, serr := os.Stat(d); serr == nil && st.IsDir() {
			dirs = append(dirs, d)
		}
	}
	if len(dirs) == 0 {
		t.Fatal("aucun dossier data/titles/*/reference — le balayage s'est cassé")
	}
	return dirs
}
