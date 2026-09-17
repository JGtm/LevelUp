package revision_test

// equivalence_test.go — LA NON-REGRESSION DES QUATRE REVISIONS : chaque empreinte egale son
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
// CE QU IL PROUVE DEPUIS LE VOLET GRAMMAIRE : que les QUATRE couches figent bien ce qu elles
// annoncent, chacune sous son propre perimetre — les racines, les exclusions et l ORDRE de ses
// valeurs amont.
//
// # POURQUOI UN SECOND ORACLE, ALORS QUE CHAQUE COUCHE A DEJA SON GATE
//
// Le gate d une couche declare son perimetre ET le mesure : il ne peut pas rougir sur une erreur
// de perimetre, il regenererait simplement un golden coherent avec lui-meme. Ce fichier DECLARE
// le perimetre une seconde fois, a un autre endroit, a partir de la racine du module. Une
// divergence entre les deux declarations est exactement ce qu aucun gate de couche ne peut voir :
// une racine oubliee, une exclusion en trop, deux valeurs amont interverties.
//
// Il ne lit rien du lecteur de chronique du paquet (`derniereLigneDeDonnees` est reecrit ici) :
// un oracle qui partage son lecteur avec ce qu il verifie ne verifie que lui-meme.
//
// # CE TEST NE MODIFIE RIEN
//
// Il LIT les quatre goldens et les quatre arborescences, par `runtime.Caller`. Une couche dont
// le perimetre change legitimement se corrige en DEUX endroits, et c est voulu : la seconde
// declaration est le prix de l oracle.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// coucheMesuree : le perimetre d une couche, redeclare ici.
type coucheMesuree struct {
	// nom : la couche, pour les messages.
	nom string
	// revisionDuCode : la valeur de la constante, telle que le code la porte.
	revisionDuCode string
	// racines : les dossiers haches, dans l ordre (l ordre fait partie du contrat).
	racines []string
	// horsCouche : les fichiers exclus, chemins relatifs a leur racine.
	horsCouche []string
	// amonts : les VALEURS amont, dans l ordre ou elles sont hachees.
	amonts []string
	// golden : le fichier qui fige le couple (revision, empreinte).
	golden string
}

// couchesMesurees rend les quatre perimetres, resolus depuis la racine du module.
//
// LE SENS UNIQUE SE LIT DANS LA COLONNE DES AMONTS : `source` n en a aucun, `profile` hache la
// valeur de `source`, `grammar` celles de `profile` puis de `source`, `facts` celles de `source`
// puis de `grammar`. Chaque couche hache SES octets et les VALEURS de celles dont elle depend —
// jamais leurs octets (ADR 0034 D-1, decision V15 (12)).
func couchesMesurees(t *testing.T) []coucheMesuree {
	t.Helper()
	film := filepath.Join(racineAPI(t), "internal", "games", "halo_infinite", "film", "internal")
	return []coucheMesuree{
		{
			nom: "source", revisionDuCode: source.Rev,
			racines:    []string{filepath.Join(film, "source")},
			horsCouche: []string{"rev.go"},
			golden:     filepath.Join(film, "source", "testdata", "source_rev.golden"),
		},
		{
			nom: "profile", revisionDuCode: profile.Rev,
			racines:    []string{filepath.Join(film, "profile")},
			horsCouche: []string{"rev.go"},
			amonts:     []string{source.Rev},
			golden:     filepath.Join(film, "profile", "testdata", "profile_rev.golden"),
		},
		{
			nom: "grammar", revisionDuCode: grammar.Rev,
			racines:    []string{filepath.Join(film, "grammar")},
			horsCouche: []string{"rev.go", "rev_chronique.go", "rev_chronique_archive.go"},
			amonts:     []string{profile.Rev, source.Rev},
			golden:     filepath.Join(film, "grammar", "testdata", "grammar_rev.golden"),
		},
		{
			nom: "facts", revisionDuCode: facts.Rev,
			racines:    []string{filepath.Join(film, "facts")},
			horsCouche: []string{"rev.go"},
			amonts:     []string{source.Rev, grammar.Rev},
			golden:     filepath.Join(film, "facts", "testdata", "facts_rev.golden"),
		},
	}
}

// TestChaqueRevisionEgaleSonGolden : les quatre couples (revision, empreinte) figes sont ceux que
// le perimetre redeclare ici rend.
func TestChaqueRevisionEgaleSonGolden(t *testing.T) {
	for _, c := range couchesMesurees(t) {
		t.Run(c.nom, func(t *testing.T) {
			hors := map[string]bool{}
			for _, f := range c.horsCouche {
				hors[f] = true
			}
			res, err := revision.Calculer(c.racines, func(rel string) bool { return hors[rel] }, c.amonts...)
			if err != nil {
				t.Fatalf("empreinte de la couche %s : %v", c.nom, err)
			}
			rev, figee := derniereLigneDeDonnees(t, c.golden)
			t.Logf("%s : revision %s, figee %s, calculee %s (%d fichiers, amonts %v)",
				c.nom, rev, figee, res.Empreinte, res.Fichiers, c.amonts)
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
					"  calculee  : %s (%d fichiers)\n  racines   : %v\n  hors      : %v\n"+
					"  amonts    : %v\n"+
					"DEUX LECTURES, ET LA SECONDE EST CELLE QUE CE FICHIER EXISTE POUR ATTRAPER :\n"+
					"  - le gate de la couche est rouge lui aussi : la couche a change, le decider "+
					"et regenerer la ;\n"+
					"  - le gate de la couche est VERT : les deux declarations de perimetre ont "+
					"DIVERGE (une racine, une exclusion, l ordre des amonts). Chercher laquelle "+
					"des deux a raison AVANT de toucher a un golden.",
					strings.ToUpper(c.nom), c.golden, rev, figee, res.Empreinte, res.Fichiers,
					c.racines, c.horsCouche, c.amonts)
			}
		})
	}
}

// TestUneMutationRougitSaCoucheEtCellesQuiEnDependent : LA PREUVE QUE LE CHAINAGE MORD.
//
// Le sens unique ne vaut que s il est mesure : une mutation de la couche N doit changer
// l empreinte de N et celles de toutes les couches qui la nomment en amont. Le test le joue
// SANS toucher au depot — il substitue la valeur amont, ce qui est exactement ce qu une montee
// de revision fait.
//
// Ce que ce test NE remplace PAS : la mutation d un OCTET de source, jouee a la main par
// l executeur du lot et collee a son compte rendu. Ici on prouve la PROPAGATION ; la morsure sur
// les octets est prouvee par `TestEmpreinteMordSurLesSources`.
func TestUneMutationRougitSaCoucheEtCellesQuiEnDependent(t *testing.T) {
	couches := couchesMesurees(t)
	reference := map[string]string{}
	for _, c := range couches {
		hors := map[string]bool{}
		for _, f := range c.horsCouche {
			hors[f] = true
		}
		res, err := revision.Calculer(c.racines, func(rel string) bool { return hors[rel] }, c.amonts...)
		if err != nil {
			t.Fatalf("empreinte de reference de %s : %v", c.nom, err)
		}
		reference[c.nom] = res.Empreinte
	}
	// Les couches qui doivent bouger quand la valeur d une couche du dessous bouge.
	dependants := map[string][]string{
		"source":  {"profile", "grammar", "facts"},
		"profile": {"grammar"},
		"grammar": {"facts"},
	}
	for mute, attendus := range dependants {
		for _, aval := range attendus {
			c := coucheParNom(t, couches, aval)
			hors := map[string]bool{}
			for _, f := range c.horsCouche {
				hors[f] = true
			}
			amonts := amontsAvecMutation(c, mute, couches)
			res, err := revision.Calculer(c.racines, func(rel string) bool { return hors[rel] }, amonts...)
			if err != nil {
				t.Fatalf("empreinte mutee de %s : %v", aval, err)
			}
			if res.Empreinte == reference[aval] {
				t.Errorf("LA MUTATION DE `%s.Rev` NE FAIT PAS BOUGER L EMPREINTE DE `%s`.\n"+
					"  amonts de reference : %v\n  amonts mutes        : %v\n"+
					"Le sens unique n est pas tenu : une montee de la couche du dessous passerait "+
					"inapercue, et c est le faux negatif que les valeurs amont (V15 (12)) existent "+
					"pour fermer.", mute, aval, c.amonts, amonts)
			}
		}
	}
}

// amontsAvecMutation rend les amonts de `c`, la valeur de la couche `mute` remplacee par une
// valeur differente — ce qu une montee de revision produit.
func amontsAvecMutation(c coucheMesuree, mute string, couches []coucheMesuree) []string {
	valeur := ""
	for _, autre := range couches {
		if autre.nom == mute {
			valeur = autre.revisionDuCode
		}
	}
	out := make([]string, len(c.amonts))
	for i, a := range c.amonts {
		out[i] = a
		if a == valeur {
			out[i] = a + "-MUTE"
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
