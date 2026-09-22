package grammar_test

// rev_test.go — LE GATE DE [grammar.Rev], SUR LE MECANISME CENTRAL (`film/revision`, lot 2.6.0).
//
// # CE QU IL TIENT, ET LES DEUX GESTES QU IL EXIGE
//
// Il hache les sources non-test de la couche `grammar` — l arbre entier, `positions/`,
// `weaponscan/` et `weaponv3/` compris — AVEC les valeurs de `profile.Rev` et de `source.Rev`, et
// compare au golden, qui porte le couple (revision, empreinte). Toucher la couche le fait rougir ;
// le remettre au vert demande de rouvrir la ligne de la revision — donc de DECIDER si la lecture
// des octets a change.
//
// # CE QUE LE LOT 2.6.1 A CHANGE ICI, ET POURQUOI
//
// Ce gate hachait CINQ RACINES EN OCTETS : `source/`, `profile/`, `grammar/`,
// `facts/killsource/` et `facts/objectives/`. C etait le seul moyen de fermer le faux negatif de
// l amont tant qu une seule revision existait — mais il rendait le meme diagnostic pour une borne
// de carte, un ordre de composants et un appariement de kill-feed, et il faisait monter la
// grammaire pour un octet de `facts`, c est-a-dire pour une couche du DESSOUS d elle.
//
// Depuis ce lot les quatre couches ont chacune leur revision (decision V15 (11)) : `facts.Rev`
// hache `facts/`, `source.Rev` hache `source/`, `profile.Rev` hache `profile/`. Le sens unique
// est tenu par les VALEURS AMONT (V15 (12)) et non par les octets d autrui : `grammar` hache
// `profile.Rev` et `source.Rev`, `facts` hache `grammar.Rev`. Rien n est relache — une montee
// d une couche du dessous remonte mecaniquement jusqu au backlog killsource — et le diagnostic
// designe desormais la couche qui a bouge.
//
// # POURQUOI IL N Y A PLUS UNE LIGNE D EMPREINTE ICI
//
// CLAUDE.md regle 6 : le calcul, la chronique, la porte et les messages sont CENTRAUX depuis le
// lot 2.6.0, et le garde-rail `archlint/no_ad_hoc_source_fingerprint_test.go` interdit la copie —
// son allowlist est VIDE depuis que ce fichier a herite. Les 120 lignes de `sha256` +
// `filepath.WalkDir` qui vivaient ici n existent plus.
//
// # CE QU IL NE FAIT PAS, ET C EST ASSUME
//
// Il ne distingue pas un changement de grammaire d une reformulation de commentaire : le hachage
// porte sur les OCTETS. Un garde-rail qui voudrait ne mordre que sur le « significatif » devrait
// comprendre le decodeur — il rendrait des faux negatifs, c est-a-dire le defaut qu on ferme. Un
// faux positif coute une ligne a mettre a jour.

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updateGrammarRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE. Elle est NOMMEE
// (correctif de la revue R1, P1-1) : accrochee au `-update` du fuzz, elle laissait
// `go test ./...grammar/ -update` (sans `-run`) refiger l empreinte d une grammaire CASSEE en
// repondant `ok`.
var updateGrammarRev = flag.Bool("update-grammar-rev", false,
	"reecrire testdata/grammar_rev.golden (revision ET empreinte) — CE golden seulement")

const (
	// cheminGoldenGrammarRev : le golden, relatif au paquet.
	cheminGoldenGrammarRev = "testdata/grammar_rev.golden"
	// prefixeRevisionGrammar : le prefixe de la serie, pour la chronique.
	prefixeRevisionGrammar = "grammar"
	// plancherDesRangsGrammar : LE RANG A PARTIR DUQUEL LA SUITE EST CONTINUE.
	//
	// Avant le `.12`, la serie porte des trous mesures (2026-09-17 : `.6` -> `.8`, `.8` -> `.12`)
	// — des rangs reserves par des lots paralleles dont la fusion n a pas eu lieu. Renumeroter le
	// passe demanderait de reecrire des revisions deja figees dans des goldens, c est-a-dire
	// d ouvrir des backlogs pour de la comptabilite. Le plancher est donc EXPLICITE.
	plancherDesRangsGrammar = "grammar-2026-09-15.12"
	// commandeRegenerationGrammarRev : la commande complete, telle qu on la tape.
	commandeRegenerationGrammarRev = "LEVELUP_UPDATE_GRAMMAR_REV=1 go test " +
		"./internal/games/halo_infinite/film/internal/grammar/ -run TestGrammarRevSuitLaGrammaire " +
		"-update-grammar-rev"
)

// fichiersHorsGrammaire : les TROIS fichiers qui PORTENT la revision et sa chronique ne sont pas
// de la grammaire — ils la DECRIVENT.
//
// L EXCLUSION REND LA SECONDE BRANCHE DU GATE ATTEIGNABLE (revue R1, P2-3). Tant que `rev.go`
// etait hache, faire monter la revision SEULE changeait aussi l empreinte : le cas « la revision
// a change sans que la grammaire bouge » ne pouvait jamais se produire, et son message etait du
// code mort.
var fichiersHorsGrammaire = map[string]bool{
	"rev.go": true, "rev_chronique.go": true, "rev_chronique_archive.go": true,
	"rev_chronique_archive_2.go": true,
}

// fichiersDeChroniqueGrammar : les fichiers qui portent les ENTREES, dans l ordre
// chronologique — l archive d abord, la suite vivante ensuite.
var fichiersDeChroniqueGrammar = []string{"rev_chronique_archive.go", "rev_chronique_archive_2.go",
	"rev_chronique_archive_3.go", "rev_chronique_archive_4.go", "rev_chronique.go"}

// porteGrammarRev / messagesGrammarRev : ce que la couche declare au mecanisme central.
func porteGrammarRev() revision.Porte { return revision.Porte{Nom: "grammar-rev"} }

func messagesGrammarRev() revision.Messages {
	return revision.Messages{
		Couche:    prefixeRevisionGrammar,
		Constante: "grammar.Rev",
		Forme:     "grammar-AAAA-MM-JJ[.N]",
		Porte:     porteGrammarRev(),
		Commande:  commandeRegenerationGrammarRev,
		Question: "LA GRAMMAIRE DE LECTURE A CHANGE : une largeur, un cadre, un ordre de " +
			"composants, un lecteur neuf. Se demander AUSSI : la SORTIE des faits peut-elle " +
			"changer ? `facts.Rev` hache la VALEUR de cette revision, donc elle montera " +
			"mecaniquement — et les lignes de kill deja en base deviennent candidates au backlog " +
			"de redecodage (D6, signal utilisateur). Le CONTENU CUIT change-t-il ? alors " +
			"`replay.SchemaVersion` monte a son tour, et `backfill-replay` re-cuit.",
	}
}

// racineDeLaCoucheGrammaire : le paquet lui-meme, resolu par `runtime.Caller` — pas un chemin
// relatif au repertoire courant : le jour ou la couche demenage, ce test doit suivre le paquet et
// non hacher un dossier vide.
func racineDeLaCoucheGrammaire(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(ici)
}

// empreinteDeLaCoucheGrammaire rend l empreinte et le nombre de fichiers haches.
//
// LES VALEURS AMONT SONT `profile.Rev` PUIS `source.Rev`, hachees EN TETE dans cet ordre
// (V15 (12)) : l ordre fait partie du contrat, deux ordres differents rendent deux empreintes.
func empreinteDeLaCoucheGrammaire(t *testing.T) (string, int) {
	t.Helper()
	res, err := revision.Calculer(
		[]string{racineDeLaCoucheGrammaire(t)},
		func(rel string) bool { return fichiersHorsGrammaire[rel] },
		profile.Rev, source.Rev)
	if err != nil {
		t.Fatalf("empreinte de la couche grammaire : %v", err)
	}
	return res.Empreinte, res.Fichiers
}

// TestGrammarRevSuitLaGrammaire — LE GATE.
func TestGrammarRevSuitLaGrammaire(t *testing.T) {
	empreinte, fichiers := empreinteDeLaCoucheGrammaire(t)
	msg := messagesGrammarRev()
	if *updateGrammarRev {
		regenererGoldenGrammarRev(t, empreinte)
		return
	}
	c, err := revision.LireChronique(prefixeRevisionGrammar, fichiersDeChroniqueGrammar,
		cheminGoldenGrammarRev)
	if err != nil {
		t.Fatalf("chronique de `grammar` : %v", err)
	}
	courante := c.Courante()
	switch {
	case courante.Revision == grammar.Rev && courante.Empreinte != empreinte:
		t.Fatal(msg.SourcesOntChange(courante.Revision, courante.Empreinte, empreinte, fichiers))
	case courante.Revision != grammar.Rev && courante.Empreinte == empreinte:
		t.Fatal(msg.RevisionSeuleAChange(grammar.Rev, courante.Revision, empreinte))
	case courante.Revision != grammar.Rev:
		t.Fatal(msg.GoldenPerime(grammar.Rev, empreinte, fichiers))
	}
}

// regenererGoldenGrammarRev : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenGrammarRev(t *testing.T, empreinte string) {
	t.Helper()
	porte := porteGrammarRev()
	ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable()))
	if !ouverte {
		t.Skip(raison)
	}
	ecrit, err := porte.Reecrire(cheminGoldenGrammarRev, grammar.Rev, empreinte)
	if err != nil {
		t.Fatalf("regeneration du golden : %v", err)
	}
	// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf`
	// serait INVISIBLE avec la commande documentee, et se lirait `ok`.
	t.Fatal(ecrit)
}

// TestChroniqueCouvreLaRevisionCourante : [grammar.Rev] a-t-elle son entree, des DEUX cotes, et
// les rangs se suivent-ils sans trou depuis le plancher ?
//
// # LE DEFAUT QUE CE TEST FERME (revue de jalon M1, ronde 2, constat F5)
//
// Trois lots de corrections partis de la meme base `.11` ont empile trois blocs annoncant chacun
// « `.11` -> `.12` », suivis de lignes « FUSION ... au rang suivant » qui racontaient une
// renumerotation que l integration n a jamais faite. Resultat : la chronique s arretait a `.12`
// pendant que la constante valait `.14`, et les changements de COMPORTEMENT portes par `.13` et
// `.14` n avaient AUCUNE entree. Rien ne rougissait — le ratchet d empreinte ne tient que le
// couple (revision, empreinte), jamais ce que la revision RACONTE.
func TestChroniqueCouvreLaRevisionCourante(t *testing.T) {
	c, err := revision.LireChronique(prefixeRevisionGrammar, fichiersDeChroniqueGrammar,
		cheminGoldenGrammarRev)
	if err != nil {
		t.Fatalf("chronique de `grammar` : %v", err)
	}
	if err := c.VerifierCouverture(grammar.Rev); err != nil {
		t.Fatal(messagesGrammarRev().SansEntreeDeChronique(err.Error()))
	}
	if err := c.VerifierRangs(plancherDesRangsGrammar); err != nil {
		t.Fatal(err)
	}
}
