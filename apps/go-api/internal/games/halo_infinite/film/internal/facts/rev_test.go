package facts_test

// rev_test.go — LE GATE DE [facts.Rev], SUR LE MECANISME CENTRAL (`film/revision`, lot 2.6.0).
//
// # L HERITIER DE `TestKillSourceDecoderRevSuitLeDecodeur`
//
// Ce fichier EST celui de `sync/killcollector`, deplace par `git mv` au lot 2.6.1 et remonte sur
// le mecanisme central : son empreinte a la main — 120 lignes de `sha256` + `filepath.WalkDir` —
// a disparu au profit de `revision.Calculer`, et le garde-rail
// `archlint/no_ad_hoc_source_fingerprint_test.go` perd du meme coup l entree qui le datait.
//
// # CE QU IL TIENT, ET LES DEUX GESTES QU IL EXIGE
//
// Il hache TOUT l arbre `film/facts/` (killsource, objectives, fallback) plus les VALEURS de
// `source.Rev` et de `grammar.GrammarRev`, et compare au golden, qui porte le couple
// (revision, empreinte) avec son historique. Toucher la couche — ou une couche du dessous — le
// fait rougir ; le remettre au vert demande de rouvrir la ligne de la revision, donc de DECIDER
// si les lignes en base doivent etre redecodees.
//
// # CE QUE SON MESSAGE DIT, ET C EST LA PARTIE QUI COMPTE
//
// « La sortie des faits peut avoir change : backlog killsource sur signal utilisateur (D6),
// jamais automatique. » Le backlog est un geste de PRODUCTION : un gate ne le declenche pas, il
// pose la question. La regle vit ici ET dans `docs/SYNC_GUIDE` (FR + EN).
//
// # POURQUOI `facts` IMPORTE `grammar` ICI, ET C EST LE SEUL SENS AUTORISE
//
// Pour sa VALEUR, pas pour son code : `facts` est AU-DESSUS de `grammar` dans le sens unique de
// l ADR 0034 D-1 (source -> profile -> grammar -> facts -> replay), donc l import descend. Tant
// que `grammar.Rev` n existe pas (volet grammaire du lot 2.6, apres 2.5.e), c est `GrammarRev`
// qui tient ce role.

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updateFactsRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE. Elle est NOMMEE
// (correctif de la revue R1, P1-1) : accrochee a un `-update` generique, elle laisserait
// `go test ./...facts/ -update` refiger l empreinte d une couche CASSEE en repondant `ok`.
var updateFactsRev = flag.Bool("update-facts-rev", false,
	"reecrire testdata/facts_rev.golden (revision ET empreinte) — CE golden seulement")

const (
	// cheminGoldenFactsRev : le golden, relatif au paquet.
	cheminGoldenFactsRev = "testdata/facts_rev.golden"
	// fichierPorteurDeRevisionFacts : le fichier qui PORTE la revision, exclu du hachage.
	fichierPorteurDeRevisionFacts = "rev.go"
	// prefixeRevisionFacts : le prefixe de la serie. C est `killsource` et PAS `facts` : la
	// serie est reprise sans renumerotation (V15 (16)), et les lignes deja en base portent ces
	// valeurs-la dans `decoder_rev`. Renommer la serie les rendrait toutes candidates au
	// backlog pour un changement de vocabulaire.
	prefixeRevisionFacts = "killsource"
	// commandeRegenerationFactsRev : la commande complete, telle qu on la tape.
	commandeRegenerationFactsRev = "LEVELUP_UPDATE_FACTS_REV=1 go test " +
		"./internal/games/halo_infinite/film/internal/facts/ -run TestFactsRevSuitLesFaits -update-facts-rev"
)

// porteFactsRev / messagesFactsRev : ce que la couche declare au mecanisme central.
func porteFactsRev() revision.Porte { return revision.Porte{Nom: "facts-rev"} }

func messagesFactsRev() revision.Messages {
	return revision.Messages{
		Couche:    "facts",
		Constante: "facts.Rev",
		Forme:     "killsource-AAAA-MM-JJ[.N]",
		Porte:     porteFactsRev(),
		Commande:  commandeRegenerationFactsRev,
		Question: "LA SORTIE DES FAITS PEUT AVOIR CHANGE : BACKLOG KILLSOURCE SUR SIGNAL " +
			"UTILISATEUR (D6), JAMAIS AUTOMATIQUE. Chaque ligne de `match_kill_events` porte " +
			"cette revision dans `decoder_rev` ; une montee rend candidates au redecodage toutes " +
			"les lignes qui en portent une anterieure (`conditionBacklog`, " +
			"`sync/killcollector/postsync.go`). Le redecodage du parc reste un geste de " +
			"PRODUCTION, pris par le pilote.",
	}
}

// racineDeLaCoucheFacts : l arbre des faits, resolu par `runtime.Caller` — pas un chemin relatif
// au repertoire courant : le jour ou la couche demenage, ce test doit suivre le paquet et non
// hacher un dossier vide.
func racineDeLaCoucheFacts(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(ici)
}

// empreinteDeLaCoucheFacts rend l empreinte et le nombre de fichiers haches. L ORDRE des valeurs
// amont fait partie du contrat : les inverser changerait l empreinte sans qu une source bouge.
func empreinteDeLaCoucheFacts(t *testing.T) (string, int) {
	t.Helper()
	res, err := revision.Calculer(revision.CadreRacine,
		[]string{racineDeLaCoucheFacts(t)},
		func(rel string) bool { return rel == fichierPorteurDeRevisionFacts },
		source.Rev, grammar.GrammarRev)
	if err != nil {
		t.Fatalf("empreinte de la couche facts : %v", err)
	}
	return res.Empreinte, res.Fichiers
}

// TestFactsRevSuitLesFaits — LE GATE.
func TestFactsRevSuitLesFaits(t *testing.T) {
	empreinte, fichiers := empreinteDeLaCoucheFacts(t)
	msg := messagesFactsRev()
	if *updateFactsRev {
		regenererGoldenFactsRev(t, empreinte)
		return
	}
	c, err := revision.LireChronique(prefixeRevisionFacts, fichierPorteurDeRevisionFacts,
		cheminGoldenFactsRev)
	if err != nil {
		t.Fatalf("chronique de `facts` : %v", err)
	}
	courante := c.Courante()
	switch {
	case courante.Revision == facts.Rev && courante.Empreinte != empreinte:
		t.Fatal(msg.SourcesOntChange(courante.Revision, courante.Empreinte, empreinte, fichiers))
	case courante.Revision != facts.Rev && courante.Empreinte == empreinte:
		t.Fatal(msg.RevisionSeuleAChange(facts.Rev, courante.Revision, empreinte))
	case courante.Revision != facts.Rev:
		t.Fatal(msg.GoldenPerime(facts.Rev, empreinte, fichiers))
	}
}

// regenererGoldenFactsRev : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenFactsRev(t *testing.T, empreinte string) {
	t.Helper()
	porte := porteFactsRev()
	ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable()))
	if !ouverte {
		t.Skip(raison)
	}
	ecrit, err := porte.Reecrire(cheminGoldenFactsRev, facts.Rev, empreinte)
	if err != nil {
		t.Fatalf("regeneration du golden : %v", err)
	}
	// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf`
	// serait INVISIBLE avec la commande documentee, et se lirait `ok`.
	t.Fatal(ecrit)
}

// TestChroniqueDesFaitsCouvreLaRevisionCourante : la revision courante a son ENTREE dans le godoc
// de `rev.go` ET la derniere ligne du golden.
//
// Le defaut qu il ferme est mesure cote grammaire (constat F5 de la revue de jalon M1) : la
// chronique s y etait arretee a `.12` pendant que la constante valait `.14`, et rien ne
// rougissait — les deux changements de comportement intermediaires n avaient aucune entree.
func TestChroniqueDesFaitsCouvreLaRevisionCourante(t *testing.T) {
	c, err := revision.LireChronique(prefixeRevisionFacts, fichierPorteurDeRevisionFacts,
		cheminGoldenFactsRev)
	if err != nil {
		t.Fatalf("chronique de `facts` : %v", err)
	}
	if err := c.VerifierCouverture(facts.Rev); err != nil {
		t.Fatal(messagesFactsRev().SansEntreeDeChronique(err.Error()))
	}
	// PLANCHER EXPLICITE a la valeur d heritage : les rangs anterieurs ont ete ecrits en prose
	// libre, du temps de `KillSourceDecoderRev`, et la chronique lisible par machine commence
	// ici. Normaliser le passe demanderait de reecrire des goldens deja figes — de la
	// comptabilite au prix d un backlog.
	if err := c.VerifierRangs(facts.Rev); err != nil {
		t.Fatal(err)
	}
}
