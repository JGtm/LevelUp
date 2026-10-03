package killsource_test

// rev_test.go — LE GATE DE [killsource.Rev], SUR LE MECANISME CENTRAL (`film/revision`).
//
// # L HERITIER DU GATE DE `facts.Rev`
//
// Ce fichier est celui de `film/internal/facts/rev_test.go`, deplace au lot J3.3 (2026-09-26,
// DU-2 (c)) quand la revision unique de l arbre des faits a ete remplacee par UNE REVISION PAR
// CONSOMMATEUR : celle-ci pour la sortie du kill-feed, `objectives.Rev` pour les objectifs. Le
// golden est celui de `facts.Rev`, avec son historique, sous le nom de la couche.
//
// # CE QU IL TIENT, ET LES DEUX GESTES QU IL EXIGE
//
// Il hache le perimetre de `killsource` — la fermeture de ses imports de production (lot J3.2),
// figee par `testdata/killsource_perimetre.golden` : SANS `objectives/` ni `fallback/`, que ce
// paquet n importe pas — et les VALEURS de `source.Rev`, `profile.Rev` et `grammar.Rev`. Toucher
// la couche — ou une couche du dessous — le fait rougir ; le remettre au vert demande de rouvrir
// la ligne de la revision, donc de DECIDER si les lignes en base doivent etre redecodees.
//
// « La sortie des faits peut avoir change : backlog killsource sur signal utilisateur (D6),
// jamais automatique. » La regle vit ici ET dans `docs/SYNC_GUIDE` (FR + EN).

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updateKillsourceRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE (revue R1, P1-1).
var updateKillsourceRev = flag.Bool("update-killsource-rev", false,
	"reecrire testdata/killsource_rev.golden (revision ET empreinte) — CE golden seulement")

const (
	// cheminGoldenKillsourceRev : le golden, relatif au paquet.
	cheminGoldenKillsourceRev = "testdata/killsource_rev.golden"
	// prefixeRevisionKillsource : le prefixe de la serie, repris sans renumerotation (V15 (16)) —
	// les lignes deja en base portent ces valeurs-la dans `decoder_rev`.
	prefixeRevisionKillsource = "killsource"
	// commandeRegenerationKillsourceRev : la commande complete, telle qu on la tape.
	commandeRegenerationKillsourceRev = "LEVELUP_UPDATE_KILLSOURCE_REV=1 go test " +
		"./internal/games/halo_infinite/film/internal/facts/killsource/ -run TestKillsourceRevSuitLaSortie " +
		"-update-killsource-rev"
)

// fichiersDeChroniqueKillsource : les fichiers qui portent les ENTREES lues par la chronique —
// la constante et la suite vivante. L archive (`rev_chronique_archive.go`) precede le plancher.
var fichiersDeChroniqueKillsource = []string{"rev.go", "rev_chronique.go"}

// fichiersHorsKillsource : les fichiers qui DECRIVENT la revision ne sont pas de la couche.
var fichiersHorsKillsource = map[string]bool{
	"rev.go": true, "rev_chronique.go": true, "rev_chronique_archive.go": true,
}

// porteKillsourceRev / messagesKillsourceRev : ce que la couche declare au mecanisme central.
func porteKillsourceRev() revision.Porte { return revision.Porte{Nom: "killsource-rev"} }

func messagesKillsourceRev() revision.Messages {
	return revision.Messages{
		Couche:    prefixeRevisionKillsource,
		Constante: "killsource.Rev",
		Forme:     "killsource-AAAA-MM-JJ[.N]",
		Porte:     porteKillsourceRev(),
		Commande:  commandeRegenerationKillsourceRev,
		Question: "LA SORTIE DU KILL-FEED PEUT AVOIR CHANGE : BACKLOG KILLSOURCE SUR SIGNAL " +
			"UTILISATEUR (D6), JAMAIS AUTOMATIQUE. Chaque ligne de `match_kill_events` porte " +
			"cette revision dans `decoder_rev` ; une montee rend candidates au redecodage toutes " +
			"les lignes qui en portent une anterieure (`conditionBacklog`, " +
			"`sync/killcollector/postsync.go`). Le redecodage du parc reste un geste de " +
			"PRODUCTION, pris par le pilote.",
	}
}

// racineDeLaCoucheKillsource : le paquet, resolu par `runtime.Caller`.
func racineDeLaCoucheKillsource(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(ici)
}

// empreinteDeLaCoucheKillsource rend l empreinte et le nombre de fichiers haches.
func empreinteDeLaCoucheKillsource(t *testing.T) (string, int) {
	t.Helper()
	res, err := revision.EmpreinteDeCouche(racineDeLaCoucheKillsource(t), "killsource",
		func(rel string) bool { return fichiersHorsKillsource[rel] },
		map[string]string{"source": source.Rev, "profile": profile.Rev, "grammar": grammar.Rev})
	if err != nil {
		t.Fatalf("empreinte de la couche killsource : %v", err)
	}
	return res.Empreinte, res.Fichiers
}

// TestKillsourceRevSuitLaSortie — LE GATE.
func TestKillsourceRevSuitLaSortie(t *testing.T) {
	empreinte, fichiers := empreinteDeLaCoucheKillsource(t)
	msg := messagesKillsourceRev()
	if *updateKillsourceRev {
		regenererGoldenKillsourceRev(t, empreinte)
		return
	}
	c, err := revision.LireChronique(prefixeRevisionKillsource, fichiersDeChroniqueKillsource,
		cheminGoldenKillsourceRev)
	if err != nil {
		t.Fatalf("chronique de `killsource` : %v", err)
	}
	courante := c.Courante()
	switch {
	case courante.Revision == killsource.Rev && courante.Empreinte != empreinte:
		t.Fatal(msg.SourcesOntChange(courante.Revision, courante.Empreinte, empreinte, fichiers))
	case courante.Revision != killsource.Rev && courante.Empreinte == empreinte:
		t.Fatal(msg.RevisionSeuleAChange(killsource.Rev, courante.Revision, empreinte))
	case courante.Revision != killsource.Rev:
		t.Fatal(msg.GoldenPerime(killsource.Rev, empreinte, fichiers))
	}
}

// regenererGoldenKillsourceRev : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenKillsourceRev(t *testing.T, empreinte string) {
	t.Helper()
	porte := porteKillsourceRev()
	ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable()))
	if !ouverte {
		t.Skip(raison)
	}
	ecrit, err := porte.Reecrire(cheminGoldenKillsourceRev, killsource.Rev, empreinte)
	if err != nil {
		t.Fatalf("regeneration du golden : %v", err)
	}
	// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf`
	// serait INVISIBLE avec la commande documentee, et se lirait `ok`.
	t.Fatal(ecrit)
}

// TestChroniqueDeKillsourceCouvreLaRevisionCourante : la revision courante a son ENTREE dans la
// chronique ET la derniere ligne du golden (constat F5 de la revue de jalon M1).
func TestChroniqueDeKillsourceCouvreLaRevisionCourante(t *testing.T) {
	c, err := revision.LireChronique(prefixeRevisionKillsource, fichiersDeChroniqueKillsource,
		cheminGoldenKillsourceRev)
	if err != nil {
		t.Fatalf("chronique de `killsource` : %v", err)
	}
	if err := c.VerifierCouverture(killsource.Rev); err != nil {
		t.Fatal(messagesKillsourceRev().SansEntreeDeChronique(err.Error()))
	}
	// PLANCHER EXPLICITE a la valeur courante, herite du gate de `facts.Rev` : les rangs
	// anterieurs ont ete ecrits en prose libre, du temps de `KillSourceDecoderRev`.
	if err := c.VerifierRangs(killsource.Rev); err != nil {
		t.Fatal(err)
	}
}
