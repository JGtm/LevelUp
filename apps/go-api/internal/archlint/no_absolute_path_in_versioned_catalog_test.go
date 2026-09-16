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
// MSYS (`/c/Users/`), foyer POSIX (`/home/`, `/Users/`). L'allowlist est DATÉE et nommée.
package archlint

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
}

// extensionsTexteCatalogue : les catalogues qui se lisent comme du texte. Les `.webp` et autres
// binaires du même dossier sont hors sujet — un octet quelconque y déclencherait un faux rouge.
var extensionsTexteCatalogue = map[string]bool{
	".json": true, ".csv": true, ".tsv": true, ".txt": true, ".toml": true, ".md": true,
}

// catalogesAvecCheminAbsoluTolere : la DETTE MESURÉE le 2026-09-17, chemins relatifs à la racine
// du dépôt. Deux fichiers, tous deux produits par un outil HORS de la frontière du lot 2.10 —
// les corriger demande de toucher leur fabricant, ce que la règle « zéro fix opportuniste »
// interdit ici. Consignés au §4 du plan décodeur (D1 (2.10)).
//
//   - data/titles/halo_infinite/reference/map_backgrounds/sgh_interlock.json : le champ de
//     diagnostic d'un module ABSENT recopie le message d'erreur de Windows, chemin compris
//     (`GetFileAttributesEx D:\<bibliotheque>\…`). Producteur : `cmd/mapfond-build`.
//   - data/titles/halo_infinite/reference/map_callouts.json : le champ `source` cite
//     `C:\Program Files (x86)\Steam\…` — le même défaut que `map_quant_bounds.json`, sur un
//     autre poste. Producteur : la chaîne des zones (`cmd/callouts-*`).
//
// AGRANDIR CETTE LISTE EXIGE LA MÊME DÉMONSTRATION : pourquoi le chemin ne peut pas être
// relatif, et la date. Un nouveau catalogue n'y a pas sa place — il naît propre.
var catalogesAvecCheminAbsoluTolere = map[string]bool{
	"data/titles/halo_infinite/reference/map_backgrounds/sgh_interlock.json": true,
	"data/titles/halo_infinite/reference/map_callouts.json":                  true,
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
			for _, re := range cheminsAbsolusDePoste {
				if m := re.Find(buf); m != nil {
					t.Errorf("%s porte un chemin absolu de poste (%q) — un fichier VERSIONNÉ ne "+
						"doit citer que la MÉTHODE et un chemin relatif ; sinon deux postes "+
						"régénèrent deux fichiers différents pour les mêmes données",
						rel, strings.TrimSpace(string(m)))
					return nil
				}
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
