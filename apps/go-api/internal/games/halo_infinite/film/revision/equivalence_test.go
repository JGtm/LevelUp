package revision_test

// equivalence_test.go — LA PREUVE QUE LE MECANISME CENTRAL REND LES EMPREINTES DEJA FIGEES.
//
// # CE QUE CETTE PREUVE ACHETE
//
// Le lot 2.6.1 fera heriter les deux gates existants de ce paquet. S il s averait alors que
// `revision.Empreinte` ne rend pas la valeur figee dans les goldens, la seule sortie serait de
// regenerer ces goldens — c est-a-dire de faire monter `grammar.Rev` et `facts.Rev` pour un
// changement d OUTILLAGE, sans qu un octet de film soit lu autrement. Pour `facts`, une montee
// rouvre un backlog de redecodage de toutes les lignes de kill en base (V15 (16) : M2 est un
// jalon a ZERO difference de contenu, il n ouvre aucun backlog). La preuve est donc faite
// MAINTENANT, avant que quoi que ce soit ne soit re-pointe.
//
// # CE TEST NE MODIFIE RIEN
//
// Il LIT les deux goldens existants et les deux arborescences hachees, par `runtime.Caller`
// comme les tests qu il double. Il ne touche ni `filmdec/`, ni `killsource/`, ni
// `killcollector/` : au lot 2.6.0 ces paquets sont mutes par d autres executeurs, et une
// preuve qui doit modifier ce qu elle prouve ne prouve rien.
//
// # LES DEUX CADRES, ET POURQUOI
//
// Les deux mecanismes d aujourd hui n encadrent PAS les octets de la meme facon (cf. l arbitrage
// de `revision.CadreRacine`). L equivalence se prouve donc cadre par cadre :
//
//	killsource   revision.Empreinte          == killsource_decoder_rev.golden
//	grammaire    revision.EmpreinteHeritee   == grammar_rev.golden
//
// Autrement dit : le contrat COURANT est deja exactement celui du gate `killsource` — c est le
// gate de la grammaire qui porte le cadre a retirer, et il le porte seul.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// fichierPorteurDeRevision : le fichier qui PORTE `GrammarRev`, exclu du hachage par le gate de
// la grammaire — la constante decrit la couche, elle n en fait pas partie.
const fichierPorteurDeRevision = "grammar_rev.go"

// TestEmpreinteEgaleLeGoldenDeKillsource : le contrat COURANT rend l empreinte figee par
// `sync/killcollector/testdata/killsource_decoder_rev.golden`.
func TestEmpreinteEgaleLeGoldenDeKillsource(t *testing.T) {
	api := racineAPI(t)
	racine := filepath.Join(api, "internal", "games", "halo_infinite", "film", "killsource")
	golden := filepath.Join(api, "internal", "sync", "killcollector", "testdata",
		"killsource_decoder_rev.golden")

	empreinte, err := revision.Empreinte([]string{racine}, nil)
	if err != nil {
		t.Fatalf("empreinte de %s : %v", racine, err)
	}
	rev, figee := derniereLigneDeDonnees(t, golden)
	t.Logf("killsource : revision %s, figee %s, calculee %s", rev, figee, empreinte)
	if empreinte != figee {
		t.Fatalf("LE MECANISME CENTRAL NE REND PAS L EMPREINTE FIGEE DE `killsource`.\n"+
			"  golden    : %s\n  revision  : %s\n  figee     : %s\n  calculee  : %s\n"+
			"Le lot 2.6.1 ne pourra pas faire heriter ce gate sans regenerer son golden, donc "+
			"sans monter `facts.Rev` — ce qui rouvrirait un backlog de redecodage pour un "+
			"changement d outillage. Chercher la cause (ordre des fichiers, encadrement, "+
			"normalisation des fins de ligne) AVANT de toucher au golden.",
			golden, rev, figee, empreinte)
	}
}

// TestEmpreinteHeriteeEgaleLeGoldenDeGrammaire : le cadre herite rend l empreinte figee par
// `filmdec/testdata/grammar_rev.golden`, sur les TROIS racines et avec la meme exclusion.
func TestEmpreinteHeriteeEgaleLeGoldenDeGrammaire(t *testing.T) {
	api := racineAPI(t)
	film := filepath.Join(api, "internal", "games", "halo_infinite", "film")
	racines := []string{
		filepath.Join(film, "filmdec"),
		filepath.Join(film, "killsource"),
		filepath.Join(api, "internal", "analysis", "objectiveevents"),
	}
	golden := filepath.Join(film, "filmdec", "testdata", "grammar_rev.golden")

	// L exclusion du gate de la grammaire porte sur le NOM du fichier, a n importe quelle
	// profondeur ; `path.Base` du chemin relatif la reproduit exactement.
	exclure := func(rel string) bool {
		return rel == fichierPorteurDeRevision || strings.HasSuffix(rel, "/"+fichierPorteurDeRevision)
	}
	res, err := revision.Calculer(revision.CadreHeriteGrammaire, racines, exclure)
	if err != nil {
		t.Fatalf("empreinte heritee : %v", err)
	}
	rev, figee := derniereLigneDeDonnees(t, golden)
	t.Logf("grammaire : revision %s, figee %s, calculee %s (%d fichiers)",
		rev, figee, res.Empreinte, res.Fichiers)
	if res.Empreinte != figee {
		t.Fatalf("LE MECANISME CENTRAL NE REND PAS L EMPREINTE FIGEE DE LA GRAMMAIRE.\n"+
			"  golden    : %s\n  revision  : %s\n  figee     : %s\n  calculee  : %s (%d fichiers)\n"+
			"Meme consequence que pour `killsource` : sans cette egalite, 2.6.1 renumerote "+
			"`grammar.Rev` pour un changement d outillage. Chercher la cause AVANT de toucher "+
			"au golden.", golden, rev, figee, res.Empreinte, res.Fichiers)
	}
}

// derniereLigneDeDonnees rend le couple (revision, empreinte) de la DERNIERE ligne de donnees
// d un golden — celle qui vaut pour le code d aujourd hui.
//
// Volontairement re-ecrit ici plutot que pris a `revision.LireChronique` : cette preuve doit
// pouvoir rougir meme si le lecteur de chronique est faux. Un oracle qui partage son lecteur
// avec ce qu il verifie ne verifie que lui-meme.
func derniereLigneDeDonnees(t *testing.T, chemin string) (rev, empreinte string) {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin deduit de la racine du module
	if err != nil {
		t.Fatalf("golden %s illisible : %v — il est VERSIONNE, son absence est une erreur",
			chemin, err)
	}
	for _, ligne := range strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		champs := strings.Split(ligne, "\t")
		if len(champs) != 2 || champs[0] == "" || champs[1] == "" {
			t.Fatalf("golden %s : ligne malformee %q — attendu `revision<TAB>empreinte`",
				chemin, ligne)
		}
		rev, empreinte = champs[0], champs[1]
	}
	if rev == "" {
		t.Fatalf("golden %s : aucune ligne de donnees", chemin)
	}
	return rev, empreinte
}
