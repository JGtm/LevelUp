package killsource

// golden_test.go — LA MEMOIRE DES CHIFFRES DU CHANTIER, FIGEE.
//
// POURQUOI DES GOLDEN ET PAS SEULEMENT DES ASSERTIONS. Les chiffres de ce chantier vivaient dans
// un journal de 9 000 lignes ou personne ne les retrouvait. Une assertion dit << 93 attendu >> ;
// un golden montre CE QUI A CHANGE, tag par tag, ligne par ligne, dans un diff qu on peut lire.
//
// LES FIXTURES NE SONT PAS VERSIONNEES, ET C EST UN CHOIX ASSUME : les quatre films de reference
// pesent 21 a 35 Mo chacun (107 Mo au total). Ce sont les SORTIES qui sont versionnees, sous
// `testdata/`, et la recette de regeneration est ecrite dans chaque fichier. Consequence a
// connaitre : sans fixture, [TestGoldenFilms] passe en `t.Skip`.
//
// CE SKIP A COUTE UNE DERIVE NON VUE, et il ne protege plus seul. Le 2026-08-02 on a etabli que
// le commit `47c9e72ac` avait fait bouger la sortie du decodeur sur les quatre films en annonçant
// << killsource (golden) vert >> : le test avait skippe. DEUX filets tournent donc TOUJOURS, sans
// fixture ni variable d environnement, et ce sont eux qui interdisent la derive silencieuse :
//
//	[TestGoldenPhrasesJustes]  les golden n ont pas degenere en nombres nus.
//	[TestGoldenMiniBobine]     LE DECODEUR LUI-MEME, sur des PAQUETS REELS VERSIONNES
//	                           (`testdata/minibobine_000d5950`, 3,8 Mo) — voir minibobine_test.go,
//	                           qui dit aussi PAR OU il detecte, temoin a l appui.
//
// REGENERATION :
//
//	KILLSOURCE_FIXTURES=<dir> go test ./internal/games/halo_infinite/film/killsource/ \
//	    -run Golden -update
//
// CE QUE LES GOLDEN FIGENT, ET POURQUOI CHACUN :
//
//	COUVERTURE AVEC SON DENOMINATEUR NOMME  un nombre nu ne prouve rien : trois denominateurs
//	                                        coexistent hors ligne et ne donnent pas le meme taux.
//	LES 30 ANCRES THEATER, TAGS COMPRIS     la ressource la plus rare du chantier. C est le filet
//	                                        le plus important : un compte peut rester juste
//	                                        pendant qu une etiquette devient fausse.
//	LA LIGNE `fccc61cd 01:25`               LA SEULE des 371 qui distingue une configuration
//	                                        juste d une configuration fausse (RE_LOG 7ter.71).
//	LES 8 MORTS AUTO-INFLIGEES              tag ET credit divergent : la doctrine des deux verites
//	                                        se verifie la, ou elles ne designent pas le meme
//	                                        responsable.
//	LE CONTROLE NEGATIF                     le scan n invente pas de morts. Sans lui, un decodeur
//	                                        qui accepterait n importe quoi passerait les autres.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

var updateGolden = flag.Bool("update", false, "reecrire les fichiers golden de testdata/")

// apiDeaths : LE QUATRIEME DENOMINATEUR, ET IL N EST PAS CALCULABLE HORS LIGNE.
//
// C est le nombre de morts que l API de match declare pour les quatre films de reference, bots
// compris — la SEULE reference complete. Il est ecrit ici comme une CONSTANTE DOCUMENTEE, pas
// derive : rien dans le film ne le rend. Le noter permet de publier le taux honnete
// (380/380 = 100.0 % depuis RE_LOG 7ter.79 ; il valait 376/380 = 98.9 % tant que les morts
// INFLIGEES PAR un bot n etaient decodees par personne) sans faire croire qu il sort du decodage.
// SEUL LE NUMERATEUR A BOUGE : cette constante-ci n a jamais change.
const apiDeaths = 380

// goldenDir : les sorties figees.
const goldenDir = "testdata"

func goldenPath(name string) string { return filepath.Join(goldenDir, name+".golden") }

// TestGoldenFilms : regenere la sortie de chaque film et la compare a sa version figee.
func TestGoldenFilms(t *testing.T) {
	dir := os.Getenv("KILLSOURCE_FIXTURES")
	if dir == "" {
		t.Skip("KILLSOURCE_FIXTURES non defini : la comparaison sur film reel est ignoree " +
			"(TestGoldenPhrasesJustes, lui, tourne quand meme)")
	}
	cumul := newCumul()
	for _, ref := range references {
		t.Run(ref.film, func(t *testing.T) {
			res, neg := decoderFixture(t, filepath.Join(dir, ref.film))
			cumul.ajouter(res, neg)
			comparerGolden(t, ref.film, rendreFilm(ref.film, res, neg))
		})
	}
	if t.Failed() {
		t.Log("un film a diverge : le cumul n est pas regenere pour eviter de figer un etat casse")
		return
	}
	comparerGolden(t, "cumul", cumul.rendre())
}

// decoderFixture : le decodage, plus le controle negatif qui exige de relire les paquets.
func decoderFixture(t *testing.T, dir string) (*Result, negControle) {
	t.Helper()
	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Skipf("film absent : %v", err)
	}
	res, err := Decode(t.Context(), filepath.Base(dir), src, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return res, controleNegatif(t, src, len(res.Roster.IndexToName))
}

// negControle : LE CONTROLE NEGATIF DU SCAN, mesure sur les paquets SANS evenement.
//
// LES MORTS SONT 93/93 DANS LES PAQUETS A EVENTS (RE_LOG 7ter.33). Les paquets sans evenement
// sont donc une population ou la cible est ABSENTE par construction : tout candidat qui y tombe
// est un faux positif, et leur taux mesure directement le pouvoir des quatre tests du scan.
// Mesure d origine : 1 candidat sur 208 913 712 bits (7ter.61).
type negControle struct {
	paquets   int
	bits      int
	candidats int
}

func controleNegatif(t *testing.T, src *filmsource.Film, nPlay int) negControle {
	t.Helper()
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("controle negatif : %v", err)
	}
	var n negControle
	for i := range f.t0 {
		p := &f.t0[i]
		if hasEvents(p) {
			continue
		}
		n.paquets++
		n.bits += len(p.payload) * 8
		n.candidats += len(scanPayload(p.payload, nPlay))
	}
	return n
}

// comparerGolden : compare (ou reecrit) une sortie figee.
func comparerGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)
	if *updateGolden {
		if err := os.MkdirAll(goldenDir, 0o750); err != nil {
			t.Fatalf("creation de %s : %v", goldenDir, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", path, err)
		}
		t.Logf("golden reecrit : %s", path)
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // chemin construit a partir d'un nom de film fige
	if err != nil {
		t.Fatalf("golden absent : %v — regenerer avec -update", err)
	}
	if string(want) == got {
		return
	}
	t.Errorf("la sortie a change par rapport a %s.\n%s", path, premierEcart(string(want), got))
}

// premierEcart : la premiere ligne qui differe, avec son numero. Un diff complet de 400 lignes
// noie l information ; ce qu on veut savoir, c est QUELLE ligne a bouge.
func premierEcart(want, got string) string {
	lw, lg := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(lw) || i < len(lg); i++ {
		a, b := ligne(lw, i), ligne(lg, i)
		if a == b {
			continue
		}
		return fmt.Sprintf("premier ecart ligne %d :\n  fige   : %s\n  obtenu : %s\n"+
			"(%d ligne(s) figees, %d obtenues)", i+1, a, b, len(lw), len(lg))
	}
	return "les fichiers different sans qu aucune ligne ne differe (fin de fichier ?)"
}

func ligne(ls []string, i int) string {
	if i >= len(ls) {
		return "<absente>"
	}
	return ls[i]
}
