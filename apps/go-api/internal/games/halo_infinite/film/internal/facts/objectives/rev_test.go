package objectives_test

// rev_test.go — LE GATE DE [objectives.Rev], SUR LE MECANISME CENTRAL (`film/revision`).
//
// NE AU LOT J3.3 (2026-09-26, DU-2 (c)) avec la revision qu il garde : il hache le perimetre du
// paquet — la fermeture de ses imports de production, figee par
// `testdata/objectives_perimetre.golden` — et la VALEUR de `source.Rev`, et compare au golden.
// Toucher la couche le fait rougir ; le remettre au vert demande de DECIDER si la sortie des
// objectifs change. Une montee n ouvre AUCUN backlog killsource : elle perime les calques
// d objectifs et les faits persistes.

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updateObjectivesRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE (revue R1, P1-1).
var updateObjectivesRev = flag.Bool("update-objectives-rev", false,
	"reecrire testdata/objectives_rev.golden (revision ET empreinte) — CE golden seulement")

const (
	// cheminGoldenObjectivesRev : le golden, relatif au paquet.
	cheminGoldenObjectivesRev = "testdata/objectives_rev.golden"
	// fichierPorteurDeRevisionObjectives : le fichier qui PORTE la revision et sa chronique,
	// exclu du hachage.
	fichierPorteurDeRevisionObjectives = "rev.go"
	// prefixeRevisionObjectives : le prefixe de la serie.
	prefixeRevisionObjectives = "objectives"
	// commandeRegenerationObjectivesRev : la commande complete, telle qu on la tape.
	commandeRegenerationObjectivesRev = "LEVELUP_UPDATE_OBJECTIVES_REV=1 go test " +
		"./internal/games/halo_infinite/film/internal/facts/objectives/ -run TestObjectivesRevSuitLaSortie " +
		"-update-objectives-rev"
)

// porteObjectivesRev / messagesObjectivesRev : ce que la couche declare au mecanisme central.
func porteObjectivesRev() revision.Porte { return revision.Porte{Nom: "objectives-rev"} }

func messagesObjectivesRev() revision.Messages {
	return revision.Messages{
		Couche:    prefixeRevisionObjectives,
		Constante: "objectives.Rev",
		Forme:     "objectives-AAAA-MM-JJ[.N]",
		Porte:     porteObjectivesRev(),
		Commande:  commandeRegenerationObjectivesRev,
		Question: "LA SORTIE DES OBJECTIFS PEUT AVOIR CHANGE (statborg, actions d objectif, " +
			"manches, drapeau, couronne, crane, bombe). Une montee perime les calques qu elle date " +
			"(`couchesDesCalques`, `film/replay/layers.go`) et les faits persistes : le verdict de " +
			"recuisson dit `redecoder`. Elle n ouvre AUCUN backlog killsource.",
	}
}

// racineDeLaCoucheObjectives : le paquet, resolu par `runtime.Caller`.
func racineDeLaCoucheObjectives(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(ici)
}

// empreinteDeLaCoucheObjectives rend l empreinte et le nombre de fichiers haches.
func empreinteDeLaCoucheObjectives(t *testing.T) (string, int) {
	t.Helper()
	res, err := revision.EmpreinteDeCouche(racineDeLaCoucheObjectives(t), "objectives",
		func(rel string) bool { return rel == fichierPorteurDeRevisionObjectives },
		map[string]string{"source": source.Rev})
	if err != nil {
		t.Fatalf("empreinte de la couche objectives : %v", err)
	}
	return res.Empreinte, res.Fichiers
}

// TestObjectivesRevSuitLaSortie — LE GATE.
func TestObjectivesRevSuitLaSortie(t *testing.T) {
	empreinte, fichiers := empreinteDeLaCoucheObjectives(t)
	msg := messagesObjectivesRev()
	if *updateObjectivesRev {
		regenererGoldenObjectivesRev(t, empreinte)
		return
	}
	c, err := revision.LireChronique(prefixeRevisionObjectives,
		[]string{fichierPorteurDeRevisionObjectives}, cheminGoldenObjectivesRev)
	if err != nil {
		t.Fatalf("chronique de `objectives` : %v", err)
	}
	courante := c.Courante()
	switch {
	case courante.Revision == objectives.Rev && courante.Empreinte != empreinte:
		t.Fatal(msg.SourcesOntChange(courante.Revision, courante.Empreinte, empreinte, fichiers))
	case courante.Revision != objectives.Rev && courante.Empreinte == empreinte:
		t.Fatal(msg.RevisionSeuleAChange(objectives.Rev, courante.Revision, empreinte))
	case courante.Revision != objectives.Rev:
		t.Fatal(msg.GoldenPerime(objectives.Rev, empreinte, fichiers))
	}
}

// regenererGoldenObjectivesRev : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenObjectivesRev(t *testing.T, empreinte string) {
	t.Helper()
	porte := porteObjectivesRev()
	ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable()))
	if !ouverte {
		t.Skip(raison)
	}
	ecrit, err := porte.Reecrire(cheminGoldenObjectivesRev, objectives.Rev, empreinte)
	if err != nil {
		t.Fatalf("regeneration du golden : %v", err)
	}
	// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf`
	// serait INVISIBLE avec la commande documentee, et se lirait `ok`.
	t.Fatal(ecrit)
}

// TestChroniqueDObjectivesCouvreLaRevisionCourante : la revision courante a son ENTREE dans le
// godoc de `rev.go` ET la derniere ligne du golden, et les rangs se suivent sans trou.
func TestChroniqueDObjectivesCouvreLaRevisionCourante(t *testing.T) {
	c, err := revision.LireChronique(prefixeRevisionObjectives,
		[]string{fichierPorteurDeRevisionObjectives}, cheminGoldenObjectivesRev)
	if err != nil {
		t.Fatalf("chronique de `objectives` : %v", err)
	}
	if err := c.VerifierCouverture(objectives.Rev); err != nil {
		t.Fatal(messagesObjectivesRev().SansEntreeDeChronique(err.Error()))
	}
	// Plancher VIDE : la chronique nait avec ce lot, elle tient la regle depuis son premier rang.
	if err := c.VerifierRangs(""); err != nil {
		t.Fatal(err)
	}
}
