package archlint

// film_file_size_test.go — LE RATCHET DE TAILLE DES FICHIERS DU DECODEUR ET DE SES VOISINS
// (lot 2.7, 2026-09-16).
//
// # POURQUOI CE RATCHET EXISTE
//
// CLAUDE.md regle 5 fixe le seuil a 500 lignes par fichier. Il n etait garde par rien, et la
// mesure de la revue de jalon M1 (2026-09-16, constats C1 / C2) a dit ce que ca coute : entre
// `783ae680d` et `34fa53da5`, SIX fichiers deja au-dela du seuil ont GROSSI sans que personne
// ne le voie — `document_chronicle.go` +171, `killcollector/collector.go` +138,
// `replay/document.go` +30, `replay/build.go` +16, `replay/zone_states_hill.go` +13,
// `filmdec/traverse.go` +3. Aucun de ces lots n avait tort ; simplement, rien ne sonnait.
//
// # LA REGLE, EN DEUX LIGNES
//
//	fichier hors table   plafond 500 lignes
//	fichier de la table  plafond = sa taille du jour de sa mise en table, JAMAIS accru
//
// Un fichier qui descend sous son plafond n est pas une erreur — c est le sens de la marche.
// Quand il repasse sous 500, son entree se RETIRE de la table (une entree perimee finit par
// autoriser n importe quoi) ; le test le dit lui-meme.
//
// # CE QUE LE RATCHET NE FAIT PAS, ET C EST VOULU
//
// Il ne demande a personne de scinder les fichiers de la table. Il interdit de les AGRANDIR.
// La scission se decide lot par lot, avec la preuve d equivalence qui va avec (le lot 2.7 l a
// faite pour `filmdec` et `killcollector` ; le volet publication viendra apres la fusion des
// lots 1.9.9 / 1.9.11 / 1.9.14, qui tiennent ces fichiers).
//
// Il ne distingue pas non plus un fichier de test d un fichier de production. Une table de
// fixtures de 1 200 lignes est exactement le genre de fichier qui grossit sans qu on s en
// apercoive, et la regle 5 ne l exempte pas.

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// seuilLignesParFichier : le seuil du depot (CLAUDE.md regle 5).
const seuilLignesParFichier = 500

// racinesSurveilleesTaille : les arborescences que ce ratchet couvre, relatives a `apps/go-api`.
// Ce sont celles du chantier du decodeur de film — le perimetre du lot 2.7.
var racinesSurveilleesTaille = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/sync/killcollector",
	"internal/analysis/objectiveevents",
}

// plancherFichiersBalayesTaille : LE PLANCHER CONTRE UN BALAYAGE MUET. 1 298 fichiers `.go`
// mesures le 2026-09-16 dans les quatre racines ci-dessus ; un balayage qui en rend nettement
// moins n a pas trouve l arborescence (racine renommee, chemin relatif casse) et ne garde plus
// rien. Il doit ECHOUER bruyamment, pas rendre vert sur du vide.
const plancherFichiersBalayesTaille = 1200

// plafondsParFichier — LA TABLE DATEE DES FICHIERS DEJA AU-DELA DU SEUIL, avec leur taille du
// 2026-09-16 (lot 2.7, apres la scission de `filmdec` et de `killcollector`). Chemin relatif a
// `apps/go-api`, en slash.
//
// CHAQUE VALEUR NE PEUT QUE DESCENDRE. Faire monter une entree, c est autoriser exactement la
// derive que la mesure C1 / C2 a constatee ; la reponse a « mon lot ajoute vingt lignes ici »
// est de sortir vingt lignes ailleurs dans le fichier, ou de le scinder.
//
// UNE SEULE EXCEPTION, ECRITE : `replay/document_chronicle.go`. C est une CHRONIQUE — une entree
// par version de schema, ajoutee en meme temps que la montee de `SchemaVersion`, et le depot a
// deja tranche qu elle ne se scinde pas (item 2.7.1 du PLAN_DECODEUR_FILM : « c est une
// chronique : exemption ecrite en tete, pas de scission »). Son plafond monte donc du volume de
// l entree ajoutee, DANS LE COMMIT QUI MONTE `SchemaVersion`, et jamais autrement. Toute autre
// montee de cette ligne est le meme abus que pour les autres.
var plafondsParFichier = map[string]int{
	// --- publication du rejeu : le volet 2.7p les scindera, apres la fusion des lots
	// 1.9.9 / 1.9.11 / 1.9.14 qui les tiennent aujourd hui.
	"internal/games/halo_infinite/film/replay/document_chronicle.go": 1476,
	"internal/games/halo_infinite/film/replay/document.go":           627,
	"internal/games/halo_infinite/film/replay/lives.go":              536,
	"internal/games/halo_infinite/film/replay/build.go":              532,
	"internal/games/halo_infinite/film/replay/zone_states_hill.go":   526,
	// --- production, hors perimetre du lot 2.7 (aucune preuve d equivalence ne couvrait
	// leur scission : elle se decidera au lot qui les rouvrira).
	"internal/analysis/objectiveevents/statborg.go":                   652,
	"internal/replaybuild/replaybuild.go":                             577,
	"internal/games/halo_infinite/film/filmdec/equipment_creation.go": 508,
	// --- tests et instruments de mesure : tables de fixtures et balayages de recherche.
	"internal/games/halo_infinite/film/replay/golden_assembly_test.go":                   1180,
	"internal/games/halo_infinite/film/filmdec/i59_anchor_test.go":                       1152,
	"internal/games/halo_infinite/film/replay/structure_test.go":                         1135,
	"internal/games/halo_infinite/film/replay/t0_mouvement_research_test.go":             872,
	"internal/games/halo_infinite/film/replay/inventory_position_i22_test.go":            833,
	"internal/games/halo_infinite/film/replay/ground_link_research_test.go":              814,
	"internal/games/halo_infinite/film/replay/ctf_retour_zone_research_test.go":          812,
	"internal/games/halo_infinite/film/replay/assaut_manches_research_test.go":           655,
	"internal/games/halo_infinite/film/replay/mapvar/noms_lieux_hunt_test.go":            627,
	"internal/games/halo_infinite/film/killicon/killicon_test.go":                        610,
	"internal/games/halo_infinite/film/filmdec/components_hooks_test.go":                 600,
	"internal/games/halo_infinite/film/replay/closures_test.go":                          598,
	"internal/games/halo_infinite/film/replay/vehicules_v1a_test.go":                     587,
	"internal/games/halo_infinite/film/filmdec/vehicules_v13_deadstate_test.go":          583,
	"internal/games/halo_infinite/film/replay/minifilm_test.go":                          581,
	"internal/games/halo_infinite/film/filmdec/ground_weapon_lifecycle_research_test.go": 574,
	"internal/games/halo_infinite/film/replay/equipment_uses_join_test.go":               570,
	"internal/analysis/objectiveevents/assaut_pied_ancre_test.go":                        558,
	"internal/sync/killcollector/positions_test.go":                                      557,
	"internal/games/halo_infinite/film/filmdec/golden_minibobine_test.go":                553,
	"internal/games/halo_infinite/film/filmdec/vehicules_v2_items_test.go":               533,
	"internal/games/halo_infinite/film/replay/attachement_phase0_bord_test.go":           529,
	"internal/sync/killcollector/collector_test.go":                                      527,
	"internal/games/halo_infinite/film/replay/visee_chronologie_research_test.go":        502,
	"internal/games/halo_infinite/film/filmdec/imagecle_oracle_n2_research_test.go":      502,
}

// TestTailleDesFichiersDuFilmNeCroitPas : aucun fichier des racines surveillees ne depasse son
// plafond — 500 lignes par defaut, sa taille figee s il est dans la table.
func TestTailleDesFichiersDuFilmNeCroitPas(t *testing.T) {
	tailles := balayerTaillesFilm(t)
	if len(tailles) < plancherFichiersBalayesTaille {
		t.Fatalf("balayage muet : %d fichiers .go vus dans %v, plancher %d. "+
			"L arborescence a bouge ou le chemin relatif est casse — ce ratchet ne garde "+
			"plus rien et doit echouer bruyamment.",
			len(tailles), racinesSurveilleesTaille, plancherFichiersBalayesTaille)
	}
	chemins := make([]string, 0, len(tailles))
	for rel := range tailles {
		chemins = append(chemins, rel)
	}
	sort.Strings(chemins)
	for _, rel := range chemins {
		n := tailles[rel]
		plafond, fige := plafondsParFichier[rel]
		if !fige {
			plafond = seuilLignesParFichier
		}
		if n <= plafond {
			continue
		}
		if fige {
			t.Errorf("%s : %d lignes, plafond fige a %d (2026-09-16, lot 2.7).\n"+
				"UN PLAFOND NE MONTE PAS. Sortir autant de lignes ailleurs dans le fichier, "+
				"ou le scinder par deplacement pur avec sa preuve d equivalence.\n"+
				"Seule exception ecrite : document_chronicle.go, dont l entree monte du volume "+
				"de son entree de chronique, dans le commit qui monte SchemaVersion.", rel, n, plafond)
			continue
		}
		t.Errorf("%s : %d lignes, seuil %d (CLAUDE.md regle 5).\n"+
			"Extraire un *_helpers.go / *_types.go, ou un sous-paquet. Mettre le fichier "+
			"dans plafondsParFichier N EST PAS une reponse : cette table est datee et fermee "+
			"au 2026-09-16, elle recense la dette constatee ce jour-la, pas la dette a venir.",
			rel, n, seuilLignesParFichier)
	}
}

// TestPlafondsDeTailleNeSontPasPerimes : une entree de la table qui designe un fichier disparu,
// ou qui est repassee sous le seuil, se RETIRE. Une allowlist perimee finit par autoriser
// n importe quoi (meme regle que les autres ratchets de ce paquet).
func TestPlafondsDeTailleNeSontPasPerimes(t *testing.T) {
	tailles := balayerTaillesFilm(t)
	entrees := make([]string, 0, len(plafondsParFichier))
	for rel := range plafondsParFichier {
		entrees = append(entrees, rel)
	}
	sort.Strings(entrees)
	for _, rel := range entrees {
		n, vu := tailles[rel]
		if !vu {
			t.Errorf("%s est au plafond mais n existe plus (renomme, deplace ou supprime) : "+
				"retirer l entree, ou la reecrire au nouveau chemin.", rel)
			continue
		}
		if n <= seuilLignesParFichier {
			t.Errorf("%s : %d lignes, soit sous le seuil de %d — retirer son entree de "+
				"plafondsParFichier. Le fichier est rentre dans la regle, la table n a plus "+
				"a le connaitre.", rel, n, seuilLignesParFichier)
		}
	}
}

// balayerTaillesFilm rend, par chemin relatif a `apps/go-api` (en slash), le nombre de lignes de
// chaque `.go` des racines surveillees. Le compte est celui de `wc -l` : le nombre de fins de
// ligne, la derniere ligne comprise si le fichier finit par un saut.
func balayerTaillesFilm(t *testing.T) map[string]int {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	out := map[string]int{}
	for _, racine := range racinesSurveilleesTaille {
		base := filepath.Join(goAPIRoot, filepath.FromSlash(racine))
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				nom := d.Name()
				if chemin != base && (strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") {
				return nil
			}
			blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du paquet
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(goAPIRoot, chemin)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(rel)] = bytes.Count(blob, []byte("\n"))
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", base, err)
		}
	}
	if len(out) == 0 {
		t.Fatal(fmt.Sprint("aucun fichier .go balaye : ", racinesSurveilleesTaille))
	}
	return out
}
