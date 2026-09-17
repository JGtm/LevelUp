package revision_test

// messages_test.go — CE QU UN MESSAGE DE GATE DOIT DIRE.
//
// Un gate d empreinte ne garde pas une egalite de chaines : il garde la QUESTION que son message
// pose. Ces verifications portent donc sur le CONTENU utile — la couche, la constante a monter,
// la question propre a la couche, la commande de regeneration — et pas sur une prose figee.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// messagesDeFaits : le parametrage tel que la couche `facts` le posera au lot 2.6.1.
func messagesDeFaits() revision.Messages {
	return revision.Messages{
		Couche:    "facts",
		Constante: "facts.Rev",
		Forme:     "facts-AAAA-MM-JJ[.N]",
		Porte:     revision.Porte{Nom: "facts-rev"},
		Commande:  "go test ./internal/games/halo_infinite/film/internal/facts/ -run FactsRev -update-facts-rev",
		Question: "Les lignes de kill deja en base redeviennent-elles candidates au redecodage " +
			"(backlog killsource) ? Le redecodage part sur SIGNAL UTILISATEUR (D6), jamais seul.",
	}
}

func TestMessagesPortentLaQuestionDeLaCouche(t *testing.T) {
	m := messagesDeFaits()
	msg := m.SourcesOntChange("facts-2026-09-17", "aaa", "bbb", 23)
	for _, attendu := range []string{
		"FACTS", "facts.Rev", "facts-AAAA-MM-JJ[.N]", "backlog killsource", "SIGNAL UTILISATEUR",
		m.Commande, "aaa", "bbb", "23 fichiers",
	} {
		if !strings.Contains(msg, attendu) {
			t.Errorf("le message « les sources ont change » ne dit pas %q :\n%s", attendu, msg)
		}
	}
}

func TestMessagesDistinguentLesTroisDerives(t *testing.T) {
	m := messagesDeFaits()
	sources := m.SourcesOntChange("facts-2026-09-17", "aaa", "bbb", 23)
	revSeule := m.RevisionSeuleAChange("facts-2026-09-18", "facts-2026-09-17", "aaa")
	perime := m.GoldenPerime("facts-2026-09-18", "bbb", 23)
	if sources == revSeule || sources == perime || revSeule == perime {
		t.Fatal("deux derives differentes rendent le meme message : le gate ne distingue plus " +
			"les gestes qu il exige (correctif P1-4 du 2026-09-12)")
	}
	if !strings.Contains(revSeule, "megarde") {
		t.Errorf("« la revision a change seule » ne propose pas la relecture qui s impose :\n%s", revSeule)
	}
	for _, msg := range []string{sources, revSeule, perime} {
		if !strings.Contains(msg, m.Commande) {
			t.Errorf("un message de gate sans commande de regeneration :\n%s", msg)
		}
	}
}

func TestMessageDeChroniqueCiteLaFaute(t *testing.T) {
	msg := messagesDeFaits().SansEntreeDeChronique("aucune entree de godoc (entrees lues : [])")
	if !strings.Contains(msg, "aucune entree de godoc") || !strings.Contains(msg, "F5") {
		t.Errorf("le message de chronique ne cite ni la faute ni le constat qui l a ouverte :\n%s", msg)
	}
}
