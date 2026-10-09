package grammar

// porte_des_essais_guard_test.go — LES GARDE-RAILS DE LA PORTE DES ESSAIS ([crochetsDeCanal]).
//
// # CE QU ILS EMPECHENT DE REVENIR
//
// Une porte de publication qui oublie un crochet ne casse AUCUN test : les essais publient plus,
// et « plus » ressemble a « mieux ». Mesures : les etats de mouvement, absents de la porte avant le
// lot 5.7.4, publiaient 14 a 152 fois les lectures retenues ; les onze crochets des lecteurs de
// composants bipedes, absents avant le lot 2.7.b de la representation intermediaire, 5 a 40 fois
// dans les trames dont le localisateur cherche la liste (`ri27b_canaux_marche_research_test.go`).
// Il faut donc TROIS gardes : sur la composition (la porte eteint-elle chaque crochet, et le
// rend-elle ?), sur les CANAUX (tout crochet qu un canal de la marche pose est-il dans la porte ?)
// et sur les APPELANTS (le localisateur la declare-t-il ?).

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// nomsDesCrochetsDeCanal : les champs de l observation que [crochetsDeCanal] porte, dans son ordre.
var nomsDesCrochetsDeCanal = []string{
	"EtatMouvementHook", "AbilityEnergyHook", "CamoStateHook", "AbilityNonPredictedHook",
	"SpartanAbilityHook", "AbilitySetHook", "GrenadeSetHook", "GrenadeCountsHook", "HeldWeaponHook",
	"WeaponAmmoHook", "WeaponRoundsHook", "UnitEquipmentHook",
}

// crochetsDeTrame : les crochets qu un canal pose et que la porte n eteint pas, chacun avec sa
// raison. Un crochet de TRAME n est publie qu une fois par trame, par la marche retenue : aucun
// essai de record ne l atteint, et l essai de fermeture du localisateur pose sa propre observation
// ([lectureDEssai]).
var crochetsDeTrame = map[string]string{
	"VueControleHook": "verdict de la vue C, publie une fois par trame par la marche retenue",
}

// porteLocalisateurs : les trois fonctions du LOCALISATEUR de paquet (`localisateur.go`), seuls
// chemins speculatifs du depot qui ne passent pas par une inference, donc seuls a devoir appeler
// la porte eux-memes. [LocaliserBoucleDeRecords] rend en plus le verdict du repli
// `repli_localisation_largeur_libre` : le motif accepte un rendu multiple.
var porteLocalisateurs = []string{"marchLocateStrict", "marchLocateFallback", "LocaliserBoucleDeRecords"}

// TestLaListeDesCrochetsDeCanalSuitLaPorte : la liste du garde-rail a autant d entrees que la
// porte a de champs — un crochet ajoute a l une sans l autre rougit ici.
func TestLaListeDesCrochetsDeCanalSuitLaPorte(t *testing.T) {
	if n := reflect.TypeFor[crochetsDeCanal]().NumField(); n != len(nomsDesCrochetsDeCanal) {
		t.Fatalf("la porte porte %d crochets, le garde-rail en liste %d", n, len(nomsDesCrochetsDeCanal))
	}
}

// TestLaPorteEteintEtRendChaqueCrochetDeCanal : pour chacune des trois neutralisations et chaque
// crochet de canal, le crochet est eteint pendant l essai et RENDU ensuite, le meme.
func TestLaPorteEteintEtRendChaqueCrochetDeCanal(t *testing.T) {
	neutralisations := map[string]func(*Observation) func(){
		"neutraliserLesCrochetsDeCanal": (*Observation).neutraliserLesCrochetsDeCanal,
		"neutraliserCaptures":           (*Observation).neutraliserCaptures,
		"neutraliserCapturePosition":    (*Observation).neutraliserCapturePosition,
	}
	for nomPorte, neutraliser := range neutralisations {
		for _, nom := range nomsDesCrochetsDeCanal {
			t.Run(nomPorte+"/"+nom, func(t *testing.T) {
				obs := NouvelleObservation()
				champ := reflect.ValueOf(obs).Elem().FieldByName(nom)
				if !champ.IsValid() || champ.Kind() != reflect.Func {
					t.Fatalf("l observation n a pas de crochet %s", nom)
				}
				tire := 0
				champ.Set(reflect.MakeFunc(champ.Type(), func([]reflect.Value) []reflect.Value {
					tire++
					return nil
				}))
				rendre := neutraliser(obs)
				if !champ.IsNil() {
					t.Fatalf("%s est encore branche pendant l essai : une lecture speculative serait "+
						"publiee comme une lecture retenue", nom)
				}
				rendre()
				if champ.IsNil() {
					t.Fatalf("%s n a pas ete RENDU : les records retenus qui suivent ne publieraient plus", nom)
				}
				args := make([]reflect.Value, champ.Type().NumIn())
				for i := range args {
					args[i] = reflect.Zero(champ.Type().In(i))
				}
				champ.Call(args)
				if tire != 1 {
					t.Fatalf("%s rendu n est pas celui d avant (%d appels)", nom, tire)
				}
			})
		}
	}
	// Sans observateur, la marche tourne : rien ne doit paniquer.
	var nul *Observation
	nul.neutraliserLesCrochetsDeCanal()()
	nul.neutraliserCaptures()()
	nul.neutraliserCapturePosition()()
}

// motifBrancher repere le corps d une methode `Brancher` d un canal des trames ; motifCrochetPose,
// un crochet de l observation qu elle pose.
var (
	motifBrancher    = regexp.MustCompile(`(?s)\nfunc \([^)]*\) Brancher\(([a-z_]+) \*Observation[^)]*\) \{(.*?)\n\}`)
	motifCrochetPose = regexp.MustCompile(`\b([A-Z][A-Za-z]*Hook) =`)
)

// TestToutCrochetPoseParUnCanalEstDansLaPorte : chaque crochet qu une methode `Brancher` du
// paquet pose est un crochet de canal, ou un crochet de trame inscrit avec sa raison. C est un
// ratchet sur la SOURCE : le defaut qu il garde est une omission, qui ne se voit pas a
// l execution.
func TestToutCrochetPoseParUnCanalEstDansLaPorte(t *testing.T) {
	dansLaPorte := map[string]bool{}
	for _, n := range nomsDesCrochetsDeCanal {
		dansLaPorte[n] = true
	}
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	vus := 0
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f) //nolint:gosec // un fichier du paquet lui-meme
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range motifBrancher.FindAllStringSubmatch(string(data), -1) {
			vus++
			for _, c := range motifCrochetPose.FindAllStringSubmatch(m[2], -1) {
				if _, trame := crochetsDeTrame[c[1]]; !dansLaPorte[c[1]] && !trame {
					t.Errorf("%s : un canal pose %s, que la porte des essais n eteint pas — chaque essai "+
						"du localisateur ou de l inference le publierait (l inscrire dans crochetsDeCanal)",
						f, c[1])
				}
			}
		}
	}
	if vus == 0 {
		t.Fatal("aucune methode Brancher trouvee : le ratchet ne garde plus rien (motif a mettre a jour)")
	}
}

// TestLocalisateursDeclarentLaPorte : les trois fonctions du localisateur appellent la porte. Un
// ratchet sur la SOURCE, pour la meme raison.
func TestLocalisateursDeclarentLaPorte(t *testing.T) {
	const fichier = "localisateur.go"
	data, err := os.ReadFile(fichier) //nolint:gosec // un fichier du paquet lui-meme
	if err != nil {
		t.Fatalf("lecture de %s : %v", fichier, err)
	}
	src := string(data)
	for _, nom := range porteLocalisateurs {
		re := regexp.MustCompile(`(?s)\nfunc ` + regexp.QuoteMeta(nom) + `\([^)]*\) (?:int|\(int, bool\)) \{(.*?)\n\}`)
		m := re.FindStringSubmatch(src)
		if m == nil {
			t.Fatalf("%s : fonction %s introuvable — le ratchet ne garde plus rien (elle a ete renommee "+
				"ou deplacee : mettre a jour `porteLocalisateurs`)", fichier, nom)
		}
		if !strings.Contains(m[1], "neutraliserLesCrochetsDeCanal()()") {
			t.Errorf("%s : %s NE DECLARE PAS sa speculation : sans `defer "+
				"cfg.Obs.neutraliserLesCrochetsDeCanal()()`, chaque offset essaye PUBLIE ses lectures "+
				"de composant a une position de bit qui sera jetee.", fichier, nom)
		}
	}
}
