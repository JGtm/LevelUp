// Package archlint — no_retention_window_inline_test.go : ratchet « UNE SEULE DEFINITION DE
// LA FENETRE DE RETENTION DES ARTEFACTS DE REJEU » (lot 7.10, 2026-09-07).
//
// # CE QU'IL EMPECHE
//
// Deux endroits repondent a « ce match est-il dans la fenetre de retention ? » : la FILE DE
// CUISSON, qui decide ce qui sera cuit, et la LECTURE TACTIQUE, qui annonce a l'utilisateur
// ce qui va l'etre (`matchs_en_attente` contre `matchs_non_cuisables`). Les deux DOIVENT
// dire la meme chose.
//
// Une seconde formulation ecrite a la main aurait fini par diverger — un `>` contre un
// `>=`, un fuseau, un `AddDate` sur le mois courant, ou UNE CONDITION OUBLIEE — et la page
// aurait promis une cuisson qui n'arrive jamais, ou declare definitivement perdu un match
// que la file reprend le lendemain. C'est la pire forme d'erreur d'un tel message : il
// envoie attendre pour rien, ou renoncer a tort. Le cas s'est produit : la ventilation
// comptait « en attente » des matchs dont le FILM EST PERDU, parce qu'elle ne reprenait que
// la fenetre et pas le marqueur terminal.
//
// La regle vit dans `analysis.SQLEligibleALaCuisson`, `analysis.SQLDansFenetreRetention` et
// `analysis.BorneRetention`.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// formulationsDeLaFenetre : les formes qui refont la borne a la main.
//
// `AddDate(0, -` capture le calcul de la borne quel que soit le nom de la variable de mois ;
// ` >= ?` capture la comparaison SQL de l'horodatage a cette borne ;
// `backfill_completed, 0) & ?` capture le test du marqueur de film perdu, TROISIEME condition
// de l'eligibilite (ajout 2026-09-07). Les trois ne sont cherchees que dans les fichiers qui
// NOMMENT la retention (cf. motsRetention), sans quoi elles remonteraient des dizaines de
// fenetres sans rapport.
var formulationsDeLaFenetre = []string{
	"AddDate(0, -",
	" >= ?",
	"backfill_completed, 0) & ?",
}

// paquetProprietaireDeLaFenetre : le seul endroit ou ces formes sont legitimes.
const paquetProprietaireDeLaFenetre = "internal/analysis/sql_fragments.go"

// motsRetention BORNE le balayage aux fichiers qui parlent de retention.
//
// SANS CETTE BORNE, LE RATCHET EST FAUX : `AddDate(0, -` est un calcul de date banal dans un
// depot qui fenetre des saisons, des historiques et des snapshots. Seul un fichier qui NOMME
// la retention peut en reecrire la regle.
//
// LES DEUX ORTHOGRAPHES SONT CHERCHEES : le depot ecrit ses commentaires en francais
// accentue (« rétention ») et ses identifiants sans accent (`RetentionMonths`). Ne chercher
// que la forme nue laissait passer un fichier entierement redige avec l'accent — le ratchet
// aurait alors garde le vide sans le dire.
var motsRetention = []string{"etention", "étention"}

// nommeLaRetention dit si un fichier parle de retention, sous l'une ou l'autre orthographe.
func nommeLaRetention(contenu string) bool {
	for _, mot := range motsRetention {
		if strings.Contains(contenu, mot) {
			return true
		}
	}
	return false
}

// TestUneSeuleDefinitionDeLaFenetreDeRetention — le ratchet.
//
// SELF-CHECK POSITIF : le proprietaire doit porter les DEUX formes, sinon le garde ne
// verifie plus rien (le fragment a-t-il ete renomme, ou la borne inlinee chez l'appelant ?).
func TestUneSeuleDefinitionDeLaFenetreDeRetention(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	vuChezLeProprietaire := map[string]bool{}
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		proprietaire := rel == paquetProprietaireDeLaFenetre
		if !proprietaire && !nommeLaRetention(string(data)) {
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, motif := range formulationsDeLaFenetre {
				if !strings.Contains(line, motif) {
					continue
				}
				if proprietaire {
					vuChezLeProprietaire[motif] = true
					continue
				}
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	for _, motif := range formulationsDeLaFenetre {
		if !vuChezLeProprietaire[motif] {
			t.Fatalf("la forme %q n'apparait plus dans %s : le fragment canonique a-t-il ete "+
				"renomme ? Le garde ne verifie plus rien", motif, paquetProprietaireDeLaFenetre)
		}
	}
	if len(violations) > 0 {
		t.Errorf("fenetre de retention recopiee hors de %s (%d) — appeler "+
			"analysis.SQLDansFenetreRetention / analysis.BorneRetention : deux definitions font "+
			"que la page annonce une cuisson que la file ne fera pas :\n  %s",
			paquetProprietaireDeLaFenetre, len(violations), strings.Join(violations, "\n  "))
	}
}
