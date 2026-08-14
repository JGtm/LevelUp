package replay

// golden_guard_test.go — LE GARDE-FOU DU GOLDEN LUI-MEME.
//
// POURQUOI CE FICHIER EXISTE. Un golden peut se degrader sans jamais casser : il suffit qu un
// jour quelqu un simplifie le rendu et n y laisse que des nombres. Le fichier passerait encore,
// et il ne prouverait plus rien — un « 475 » nu ne dit pas de quoi il est le numerateur, et
// trois denominateurs coexistent dans ce chantier (evenements disponibles, vies, traces
// publiees) qui ne donnent pas le meme taux.
//
// CE TEST NE CONSTRUIT AUCUN DOCUMENT ET NE LIT AUCUNE ENTREE. Il lit le fichier fige et
// verifie qu il porte encore les PHRASES qui qualifient ses chiffres. C est le seul test de ce
// paquet qui protege la FORME de la preuve plutot que sa valeur — patron repris du paquet
// killsource, ou il a deja servi.

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// phrasesGolden : ce que la sortie figee doit DIRE, et pas seulement chiffrer.
var phrasesGolden = []string{
	"AUCUN OCTET DE FILM n est lu pour produire cette sortie",
	"l ASSEMBLAGE ; le DECODAGE est verrouille a part par la mini-bobine",
	"une trace est UNE VIE, pas un joueur (le slot migre a chaque reapparition)",
	"NOMMEE(S) par le pont (fil des morts, puis fermetures)",
	"une vie que le film ne clot",
	"par aucun evenement reste sans identite, et c est une LIMITE, pas une erreur",
	"le xuid IDENTIFIE, l index ORDONNE ; ils ne sont pas interchangeables",
	"rattaches sur DISPONIBLES, et la ventilation de tout ce qui est ecarte",
	"rejets : slot introuvable=",
	"somme de controle : rattaches + rejets = disponibles",
	"le lancer porte son auteur ; ce qui se mesure est OU il est pose",
	"par source : projectile=",
	"(aucun pont)",
	"le dernier point est la DERNIERE POSITION REPLIQUEE, jamais un impact",
	"le SEUL champ qui certifie une fin de vol",
	"lu aux images-cles, jamais interpole entre deux",
	"a PLUSIEURS candidats : la plus longue a ete retenue",
	"nombre de candidats est publie, pour que le departage reste visible",
	"TABLE PARTIELLE — 5 index OBSERVES sur le corpus, 2 nommes",
	"l inventaire, pas la main : le loadout ne dit pas QUELLE arme est degainee",
	"deux sources nommees : la LECTURE, puis la FERMETURE",
	"signalerait une TROISIEME SOURCE",
	"une fermeture n attribue QUE lorsqu un seul candidat reste possible, jamais par vote",
	"un controle qui ne rejette rien ne prouve rien",
	"un rapport publie sans son denominateur ne se juge pas",
	"le tag brut reste A COTE du libelle, jamais a sa place",
	"n emprunte pas le nom d une arme voisine",
}

func lireGoldenAssembly(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(goldenAssemblyPath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden illisible : %v — regenerer avec go test -run GoldenAssembly -update", err)
	}
	return string(b)
}

// TestGoldenAssemblyPhrasesJustes : la sortie figee dit-elle encore ce que ses chiffres signifient ?
func TestGoldenAssemblyPhrasesJustes(t *testing.T) {
	g := lireGoldenAssembly(t)
	for _, p := range phrasesGolden {
		if !strings.Contains(g, p) {
			t.Errorf("phrase absente du golden : %q\n"+
				"UN CHIFFRE SANS SA PHRASE NE PROUVE RIEN — le golden a perdu sa qualification", p)
		}
	}
}

// TestGoldenAssemblyPorteSesDenominateurs : AUCUN TAUX SANS SON DENOMINATEUR.
//
// C est la regle la plus chere de ce chantier : « 91,5 % » ne veut rien dire tant qu on n a pas
// dit 519. Le test exige donc que chaque pourcentage du fichier soit precede, sur la meme ligne,
// d une fraction « a / b ».
func TestGoldenAssemblyPorteSesDenominateurs(t *testing.T) {
	rePct := regexp.MustCompile(`\d+\.\d+ %`)
	reFrac := regexp.MustCompile(`\d+ rattache\(s\) / \d+ disponible\(s\)`)
	for i, l := range strings.Split(lireGoldenAssembly(t), "\n") {
		if !rePct.MatchString(l) || reFrac.MatchString(l) {
			continue
		}
		t.Errorf("ligne %d : un pourcentage sans sa fraction — %q", i+1, l)
	}
}

// TestGoldenAssemblyFigeLesChiffresDuChantier : le golden porte-t-il les valeurs publiees ?
//
// LE POINT N EST PAS DE RE-TESTER L ASSEMBLAGE — les tests nommes de golden_assembly_test.go le
// font sur le document. C est de verifier que le FICHIER FIGE, celui qu un humain relit dans un
// diff, contient bien ces chiffres-la : un rendu qui cesserait de les ecrire viderait le golden
// de sa substance sans qu aucun autre test ne s en apercoive.
func TestGoldenAssemblyFigeLesChiffresDuChantier(t *testing.T) {
	g := lireGoldenAssembly(t)
	for _, l := range []string{
		// 483/519 = 93.1 % depuis la ronde de correction du 2026-08-09 (93.3 % avec les
		// fermetures non corrigees, 91.5 % avant elles).
		fmt.Sprintf("%d rattache(s) / %d disponible(s) = 93.1 %%", wantShotsAttached, wantShotsAvailable),
		fmt.Sprintf("%d rattache(s) / %d disponible(s) = 100.0 %%", wantGrenades, wantGrenades),
		fmt.Sprintf("%d vie(s) nommee(s) / %d", wantLivesNamed, wantLivesTotal),
		fmt.Sprintf("%d trajectoire(s) ·", wantProjectiles),
		fmt.Sprintf("%d etat(s) publie(s) ·", wantInventory),
		fmt.Sprintf("%d chunk(s) de replication concordants · 0 desaccord(s) d identite · "+
			"0 collision(s) de slot", wantIndexReadings),
	} {
		if !strings.Contains(g, l) {
			t.Errorf("chiffre du chantier absent du golden : %q", l)
		}
	}
	if strings.Contains(g, "non publiable") {
		t.Error("le golden fige un verdict NON PUBLIABLE : un golden ne fige pas un echec")
	}
}
