package main

// backfill_child_guard_test.go — LE GARDE-RAIL du blindage memoire.
//
// Une factorisation sans garde-rail re-diverge. Ce test tient deux invariants que le
// sinistre du 2026-08-20 a payes cher :
//
//  1. LE PARENT NE DECODE PAS. `BuildMatch` ne doit vivre QUE dans le fichier de l'enfant.
//     Le jour ou quelqu'un remet un `builder.BuildMatch(...)` dans une boucle du parent
//     « pour aller plus vite », c'est la bombe RAM qui revient — et ce test tombe.
//  2. CE PAQUET NE LANCE PLUS AUCUN PROCESSUS LUI-MEME. La regle s'est DURCIE le 2026-09-03
//     (PLAN_CUISSON_PERF item 5.4) : le motif parent/enfant ne vit plus dans un fichier de ce
//     paquet, il vit dans `internal/filmproc` — un seul lanceur pour la passe de backfill, le
//     post-sync, le harnais d'equivalence et l'attribution de zones. Auparavant le garde
//     TOLERAIT `exec.CommandContext` dans `backfill_child.go` (et l'EXIGEAIT meme, pour que
//     l'allowlist ne pourrisse pas) ; il l'interdit desormais PARTOUT dans le paquet. Une
//     copie locale du lanceur, c'est un second protocole de codes de sortie qui derive.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// motifGarde : un motif surveille, et le SEUL fichier qui a le droit de le porter.
//
// `doitExister` distingue deux natures de garde. Le motif qui DOIT exister ancre le
// blindage : s'il disparait, c'est que le decodage a demenage et que le garde-rail ne garde
// plus rien — on veut le savoir. Le motif qui PEUT ne pas exister n'est la que pour interdire
// une porte : `proprietaire` vide veut dire « aucun fichier de ce paquet n'a le droit ».
type motifGarde struct {
	motif        string
	proprietaire string
	doitExister  bool
}

var motifsGardes = []motifGarde{
	{motif: "BuildMatch(", proprietaire: "cmd_backfill_replay_child.go", doitExister: true},
	// AUCUN PROPRIETAIRE : le lancement de processus a quitte ce paquet pour `internal/filmproc`.
	{motif: "exec.CommandContext", proprietaire: ""},
	{motif: "exec.Command(", proprietaire: ""},
	{motif: "os.StartProcess", proprietaire: ""},
}

// sourcesDuPaquet : les .go du paquet, tests EXCLUS (un test cite forcement les motifs).
func sourcesDuPaquet(t *testing.T) map[string]string {
	t.Helper()
	noms, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	out := map[string]string{}
	for _, n := range noms {
		if strings.HasSuffix(n, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(n)
		if err != nil {
			t.Fatalf("lecture de %s: %v", n, err)
		}
		out[filepath.Base(n)] = string(raw)
	}
	if len(out) == 0 {
		t.Fatal("aucune source lue — le garde-rail ne garde rien")
	}
	return out
}

func TestGardeRail_MotifsCentralises(t *testing.T) {
	sources := sourcesDuPaquet(t)
	for _, g := range motifsGardes {
		var porteurs []string
		for nom, src := range sources {
			if strings.Contains(src, g.motif) {
				porteurs = append(porteurs, nom)
			}
		}
		if g.doitExister && len(porteurs) == 0 {
			t.Errorf("motif %q introuvable : le garde-rail garde un motif mort — "+
				"le decodage a demenage, corriger le proprietaire", g.motif)
		}
		for _, p := range porteurs {
			if p == g.proprietaire {
				continue
			}
			if g.proprietaire == "" {
				t.Errorf("%q apparait dans %s ; AUCUN fichier de ce paquet n a le droit de "+
					"lancer un processus.\nLe lanceur parent/enfant est `internal/filmproc` "+
					"(un processus par film, plafond memoire dur, priorite CPU basse) — une "+
					"copie locale, c est un second protocole de codes de sortie qui derive.", g.motif, p)
				continue
			}
			t.Errorf("%q apparait dans %s ; ce motif n appartient qu a %s.\n"+
				"Le parent d une passe de cuisson NE DECODE PAS.", g.motif, p, g.proprietaire)
		}
	}
}

// TestGardeRail_LeParentNeChargePasLesFaitsDuCorpus : la map de faits indexee par match,
// vivante pendant toute la passe, etait la seule structure du processus qui croissait avec le
// CORPUS et non avec le film. Chaque enfant lit desormais LES SIENS.
func TestGardeRail_LeParentNeChargePasLesFaitsDuCorpus(t *testing.T) {
	sources := sourcesDuPaquet(t)
	const disparu = "chargerFaitsReplay"
	for nom, src := range sources {
		if strings.Contains(src, disparu) {
			t.Errorf("%s porte encore %q : le chargement des faits de TOUT le lot est "+
				"precisement la retention inter-matchs que ce lot a supprimee", nom, disparu)
		}
	}
}
