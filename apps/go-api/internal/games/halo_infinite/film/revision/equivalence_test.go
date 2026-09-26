package revision_test

// equivalence_test.go — LA NON-REGRESSION DES REVISIONS DE COUCHE : chaque empreinte egale son
// golden, mesuree ICI, par un second oracle.
//
// # CE QUE CE FICHIER PROUVAIT, ET CE QU IL PROUVE MAINTENANT
//
// Au lot 2.6.0 il prouvait que le mecanisme central rendait EXACTEMENT les empreintes deja
// figees, cadre par cadre : sans cette egalite, faire heriter les deux gates existants aurait
// oblige a regenerer leurs goldens — c est-a-dire a faire monter `grammar.Rev` et `facts.Rev`
// pour un changement d OUTILLAGE, et pour `facts` a rouvrir un backlog de redecodage de toutes
// les lignes de kill en base (V15 (16)). Les deux heritages ont eu lieu (volet facts + source le
// 2026-09-16, volet grammaire le 2026-09-17) et AUCUN n a renumerote pour l outillage : le cadre
// herite est supprime, la preuve est consommee.
//
// CE QU IL PROUVE DEPUIS LE LOT J3.2 (2026-09-26) : que chaque couche fige bien ce qu elle
// annonce, sous la MEME REGLE de perimetre — la fermeture de ses imports, arretee aux couches
// revisees — redeclaree ici : la liste d arret, les exclusions, et les valeurs amont que la
// fermeture doit rencontrer.
//
// # POURQUOI UN SECOND ORACLE, ALORS QUE CHAQUE COUCHE A DEJA SON GATE
//
// Le gate d une couche declare son perimetre ET le mesure : il ne peut pas rougir sur une erreur
// de perimetre, il regenererait simplement un golden coherent avec lui-meme. Ce fichier DECLARE
// le perimetre une seconde fois, a un autre endroit, a partir de la racine du module. Une
// divergence entre les deux declarations est exactement ce qu aucun gate de couche ne peut voir :
// une racine d arret oubliee, une exclusion en trop, une valeur amont de travers.
//
// Il ne lit rien du lecteur de chronique du paquet (`derniereLigneDeDonnees` est reecrit ici) :
// un oracle qui partage son lecteur avec ce qu il verifie ne verifie que lui-meme.
//
// # CE TEST NE MODIFIE RIEN
//
// Il LIT les goldens et les arborescences, par `runtime.Caller`. Une couche dont le perimetre
// change legitimement se corrige en DEUX endroits, et c est voulu : la seconde declaration est le
// prix de l oracle.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// coucheMesuree : le perimetre d une couche, redeclare ici.
type coucheMesuree struct {
	// nom : la couche, pour les messages et pour la fermeture.
	nom string
	// revisionDuCode : la valeur de la constante, telle que le code la porte.
	revisionDuCode string
	// horsCouche : les fichiers exclus, chemins relatifs a la racine de la couche.
	horsCouche []string
	// valeurs : les VALEURS des couches que la fermeture doit rencontrer — ni plus, ni moins.
	// Depuis le lot J3.2 l ordre n est plus declare : c est la liste d arret qui ordonne.
	valeurs map[string]string
	// golden : le fichier qui fige le couple (revision, empreinte).
	golden string
}

// couchesDeLOracle : LA LISTE D ARRET, REDECLAREE ICI et non prise a `revision.CouchesRevisees` —
// une racine oubliee ou deplacee d un cote se verrait comme une divergence de l autre.
func couchesDeLOracle() []revision.Couche {
	const film = "internal/games/halo_infinite/film/internal/"
	return []revision.Couche{
		{Nom: "source", Racine: film + "source"},
		{Nom: "profile", Racine: film + "profile"},
		{Nom: "grammar", Racine: film + "grammar"},
		{Nom: "killsource", Racine: film + "facts/killsource"},
		{Nom: "objectives", Racine: film + "facts/objectives"},
	}
}

// couchesMesurees rend les perimetres des couches, resolus depuis la racine du module.
//
// LE SENS UNIQUE SE LIT DANS LA COLONNE DES VALEURS : `source` et `profile` n en ont aucune (la
// fermeture de leurs imports ne rencontre aucune couche), `grammar` rencontre `profile` et
// `source`, `killsource` rencontre `source`, `profile` et `grammar`, `objectives` rencontre `source` seule (lot J3.3 : une revision par consommateur de faits). Chaque couche hache SES jetons,
// ceux des paquets qu elle importe hors couche, et les VALEURS des couches qu elle importe —
// jamais leurs octets (ADR 0034 D-1, decision V15 (12), lot J3.2).
func couchesMesurees(t *testing.T) []coucheMesuree {
	t.Helper()
	film := filepath.Join(racineAPI(t), "internal", "games", "halo_infinite", "film", "internal")
	return []coucheMesuree{
		{
			nom: "source", revisionDuCode: source.Rev,
			horsCouche: []string{"rev.go"},
			golden:     filepath.Join(film, "source", "testdata", "source_rev.golden"),
		},
		{
			nom: "profile", revisionDuCode: profile.Rev,
			horsCouche: []string{"rev.go"},
			golden:     filepath.Join(film, "profile", "testdata", "profile_rev.golden"),
		},
		{
			nom: "grammar", revisionDuCode: grammar.Rev,
			// LA MEME LISTE QUE `grammar/rev_test.go`, ET ELLE COUVRE TOUTE LA CHRONIQUE
			// (D3 du lot 5.20, corrigee au lot 5.21) : `_3`, `_4` et `_5` etaient hachees
			// alors que l exclusion existe pour les tenir hors de l empreinte.
			horsCouche: []string{"rev.go", "rev_chronique.go", "rev_chronique_archive.go",
				"rev_chronique_archive_2.go", "rev_chronique_archive_3.go",
				"rev_chronique_archive_4.go", "rev_chronique_archive_5.go", "rev_chronique_archive_6.go"},
			valeurs: map[string]string{"profile": profile.Rev, "source": source.Rev},
			golden:  filepath.Join(film, "grammar", "testdata", "grammar_rev.golden"),
		},
		{
			nom: "killsource", revisionDuCode: killsource.Rev,
			horsCouche: []string{"rev.go", "rev_chronique.go", "rev_chronique_archive.go"},
			valeurs:    map[string]string{"source": source.Rev, "profile": profile.Rev, "grammar": grammar.Rev},
			golden:     filepath.Join(film, "facts", "killsource", "testdata", "killsource_rev.golden"),
		},
		{
			nom: "objectives", revisionDuCode: objectives.Rev,
			horsCouche: []string{"rev.go"},
			valeurs:    map[string]string{"source": source.Rev},
			golden:     filepath.Join(film, "facts", "objectives", "testdata", "objectives_rev.golden"),
		},
	}
}

// empreinteMesuree rend l empreinte d une couche sous le perimetre redeclare, avec `valeurs` en
// guise de valeurs amont.
func empreinteMesuree(t *testing.T, c coucheMesuree, valeurs map[string]string) revision.Resultat {
	t.Helper()
	m, err := revision.ModuleDe(racineAPI(t))
	if err != nil {
		t.Fatalf("module : %v", err)
	}
	hors := map[string]bool{}
	for _, f := range c.horsCouche {
		hors[f] = true
	}
	res, _, err := m.CalculerCouche(c.nom, couchesDeLOracle(), func(rel string) bool { return hors[rel] }, valeurs)
	if err != nil {
		t.Fatalf("empreinte de la couche %s : %v", c.nom, err)
	}
	return res
}

// TestChaqueRevisionEgaleSonGolden : les couples (revision, empreinte) figes sont ceux que le
// perimetre redeclare ici rend.
func TestChaqueRevisionEgaleSonGolden(t *testing.T) {
	for _, c := range couchesMesurees(t) {
		t.Run(c.nom, func(t *testing.T) {
			res := empreinteMesuree(t, c, c.valeurs)
			rev, figee := derniereLigneDeDonnees(t, c.golden)
			t.Logf("%s : revision %s, figee %s, calculee %s (%d fichiers, valeurs %v)",
				c.nom, rev, figee, res.Empreinte, res.Fichiers, c.valeurs)
			if rev != c.revisionDuCode {
				t.Errorf("LA DERNIERE LIGNE DU GOLDEN DE %s NE FIGE PAS LA REVISION DU CODE.\n"+
					"  golden   : %s\n  figee    : %s\n  code     : %s\n"+
					"Le gate de la couche le dit aussi ; s il est vert et celui-ci rouge, c est ce "+
					"fichier qui lit le mauvais golden.", strings.ToUpper(c.nom), c.golden, rev,
					c.revisionDuCode)
			}
			if res.Empreinte != figee {
				t.Fatalf("LE PERIMETRE REDECLARE NE REND PAS L EMPREINTE FIGEE DE %s.\n"+
					"  golden    : %s\n  revision  : %s\n  figee     : %s\n"+
					"  calculee  : %s (%d fichiers)\n  hors      : %v\n  valeurs   : %v\n"+
					"DEUX LECTURES, ET LA SECONDE EST CELLE QUE CE FICHIER EXISTE POUR ATTRAPER :\n"+
					"  - le gate de la couche est rouge lui aussi : la couche a change, le decider "+
					"et regenerer la ;\n"+
					"  - le gate de la couche est VERT : les deux declarations de perimetre ont "+
					"DIVERGE (une racine d arret, une exclusion, une valeur amont). Chercher "+
					"laquelle des deux a raison AVANT de toucher a un golden.",
					strings.ToUpper(c.nom), c.golden, rev, figee, res.Empreinte, res.Fichiers,
					c.horsCouche, c.valeurs)
			}
		})
	}
}

// TestUneMutationRougitSaCoucheEtCellesQuiEnDependent : LA PREUVE QUE LE CHAINAGE MORD.
//
// Une mutation de la couche N doit changer l empreinte de toutes les couches dont la fermeture la
// rencontre. Le test le joue SANS toucher au depot — il substitue la valeur amont, ce qui est
// exactement ce qu une montee de revision fait.
//
// Ce que ce test NE remplace PAS : la mutation d un jeton de source, jouee a la main par
// l executeur du lot et collee a son compte rendu. Ici on prouve la PROPAGATION ; la morsure sur
// les jetons est prouvee par `TestEmpreinteMordSurLesSources` et `jetons_test.go`.
func TestUneMutationRougitSaCoucheEtCellesQuiEnDependent(t *testing.T) {
	couches := couchesMesurees(t)
	reference := map[string]string{}
	for _, c := range couches {
		reference[c.nom] = empreinteMesuree(t, c, c.valeurs).Empreinte
	}
	// Les couches qui doivent bouger quand la valeur d une couche du dessous bouge : celles dont
	// la fermeture des imports la RENCONTRE (lot J3.2). `profile` n importe pas `source`, donc
	// n en depend plus.
	dependants := map[string][]string{
		"source":  {"grammar", "killsource", "objectives"},
		"profile": {"grammar", "killsource"},
		"grammar": {"killsource"},
	}
	for mute, attendus := range dependants {
		for _, aval := range attendus {
			c := coucheParNom(t, couches, aval)
			valeurs := valeursAvecMutation(c, mute)
			if empreinteMesuree(t, c, valeurs).Empreinte == reference[aval] {
				t.Errorf("LA MUTATION DE `%s.Rev` NE FAIT PAS BOUGER L EMPREINTE DE `%s`.\n"+
					"  valeurs de reference : %v\n  valeurs mutees        : %v\n"+
					"Le sens unique n est pas tenu : une montee de la couche du dessous passerait "+
					"inapercue, et c est le faux negatif que les valeurs amont (V15 (12)) existent "+
					"pour fermer.", mute, aval, c.valeurs, valeurs)
			}
		}
	}
}

// valeursAvecMutation rend les valeurs amont de `c`, celle de la couche `mute` remplacee par une
// valeur differente — ce qu une montee de revision produit.
func valeursAvecMutation(c coucheMesuree, mute string) map[string]string {
	out := make(map[string]string, len(c.valeurs))
	for nom, v := range c.valeurs {
		out[nom] = v
		if nom == mute {
			out[nom] = v + "-MUTE"
		}
	}
	return out
}

// coucheParNom rend une couche mesuree, ou echoue : un nom inconnu dans la table des dependances
// est une faute de ce fichier, pas un resultat.
func coucheParNom(t *testing.T, couches []coucheMesuree, nom string) coucheMesuree {
	t.Helper()
	for _, c := range couches {
		if c.nom == nom {
			return c
		}
	}
	t.Fatalf("couche %q absente de `couchesMesurees`", nom)
	return coucheMesuree{}
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
