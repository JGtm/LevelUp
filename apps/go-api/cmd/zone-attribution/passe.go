package main

// passe.go — LA MESURE SOUS L'EXECUTEUR BORNE : un film, un processus, un plafond.
//
// # POURQUOI CETTE COMMANDE A CHANGE DE FORME (sinistre du 2026-08-26)
//
// La mesure de Total Control a sature la machine DE TRAVAIL de l'utilisateur, deux fois. Une
// premiere fois avec sept films BTB dans UN processus — la bombe RAM deja documentee. Une
// seconde fois, apres correction, avec un film par processus mais SANS plafond ni priorite
// basse : un seul film BTB a travers `replay.BuildFromFilm` suffit a prendre le poste en otage.
//
// « Un film = un processus » est NECESSAIRE ET PAS SUFFISANT. Il faut les trois ensemble, et
// c'est ce que `internal/filmproc` fournit : un processus par film, un PLAFOND DUR par
// processus, une PRIORITE CPU BASSE.
//
// # LE PARENT PLANIFIE, L'ENFANT MESURE
//
// Le parent enumere le corpus (base en lecture seule, catalogues) et ne decode RIEN. Pour chaque
// film il re-execute son propre binaire avec `-child -match <film>`, attend sa mort, et compte
// son issue. L'enfant mesure UN film sous la sentinelle, imprime son rapport, rend son pic et
// sort avec un CODE. Un film-bombe n'emporte donc que lui-meme, et la passe continue.
//
// # COMPILER UNE FOIS, EXECUTER LE BINAIRE
//
// Ne PAS lancer cette commande par `go run` dans une boucle : chaque invocation recompilerait, et
// surtout l'executable vit alors dans un dossier temporaire que le parent re-executerait. La
// forme juste est `go build -o <bin> ./cmd/zone-attribution` puis l'appel du binaire.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/filmproc"
)

// childFlag : le drapeau qui fait d'un processus l'ENFANT d'une passe.
const childFlag = "-child"

// runChild mesure UN film sous la sentinelle memoire et rend le CODE de sortie du protocole.
//
// LA SENTINELLE EST ARMEE ICI ET NULLE PART AILLEURS. Elle mene a un arret du processus : elle
// n'a droit de cite que dans un processus qui ne tient AUCUNE ecriture. C'est le cas — cette
// commande n'ouvre la base qu'en lecture (`OpenReadForQuery`) et n'ecrit aucun artefact.
func runChild(ctx context.Context, r *runner, eligible []eligible, tune runTuning) int {
	if len(eligible) != 1 {
		fmt.Fprintf(os.Stderr, "enfant : %d film eligible, attendu exactement 1 "+
			"(le parent passe -match)\n", len(eligible))
		return filmproc.CodePreparation
	}
	g := filmproc.Arm("zone-attribution", filmproc.MeasureLimitGiB, func(peak uint64) {
		// LE PIC PART AVANT LA MORT : `os.Exit` ne joue pas les differes, et sans cette ligne
		// le parent ne saurait pas a quelle hauteur l'enfant a ete coupe.
		filmproc.EmitPeak(peak)
		fmt.Fprintf(os.Stderr, "enfant : plafond memoire depasse (%d octets) — film abandonne\n", peak)
		os.Exit(filmproc.CodeMemory)
	})
	defer func() {
		g.Disarm()
		filmproc.EmitPeak(g.Peak())
	}()

	r.results = append(r.results, r.measure(ctx, eligible[0]))
	printResults(r.results, tune)
	return filmproc.CodeOK
}

// runParent lance un enfant par film et rend le recap de la passe.
func runParent(ctx context.Context, repoRoot string, eligible []eligible, tune runTuning) error {
	runner, err := filmproc.NewRunner(repoRoot, os.Stdout)
	if err != nil {
		return err
	}
	fmt.Printf("mode parent/enfant : 1 processus par film, plafond %d Gio, priorite basse\n",
		filmproc.MeasureLimitGiB)
	recap := map[filmproc.Issue]int{}
	for i, m := range eligible {
		fmt.Printf("\n########## FILM %d/%d — %s (%s) ##########\n",
			i+1, len(eligible), m.short, m.mapName)
		res := runner.Run(ctx, childArgs(m.short, tune))
		recap[res.Issue]++
		fmt.Printf("  issue=%s code=%d duree=%s%s\n",
			res.Issue, res.Code, res.Dur.Round(1e6), peakSuffix(res.Peak))
		if res.Err != nil {
			fmt.Printf("  erreur de lancement : %v\n", res.Err)
		}
	}
	printPasseRecap(len(eligible), recap)
	return nil
}

// childArgs : la ligne de commande de l'enfant pour CE film.
//
// L'ENFANT NE RECOIT AUCUN DRAPEAU DE PLANIFICATION (`-census`, `-select-only`) : ils
// appartiennent au parent, et les transmettre porterait a un enfant l'ordre de ne rien mesurer.
func childArgs(short string, tune runTuning) []string {
	args := []string{childFlag, "-match", short, "-role", tune.role}
	if tune.cacheDir != "" {
		args = append(args, "-cache", tune.cacheDir)
	}
	if tune.dump {
		args = append(args, "-dump")
	}
	return args
}

// peakSuffix : le pic memoire de l'enfant, quand il a eu le temps de le rendre.
func peakSuffix(peak uint64) string {
	if peak == 0 {
		return " pic=non rendu"
	}
	return fmt.Sprintf(" pic=%.2f Gio", float64(peak)/(1<<30))
}

// printPasseRecap imprime le recap de la passe, issue par issue.
//
// LES MORTS SUBITES SONT NOMMEES A PART, et c'est tout l'objet du patron : un enfant tue par
// l'OS ne doit jamais se confondre avec un film mesure. Un recap qui les noierait dans les
// echecs laisserait croire que la mesure a couvert le corpus.
func printPasseRecap(total int, recap map[filmproc.Issue]int) {
	fmt.Printf("\nPASSE — %d film(s)\n", total)
	for _, i := range []filmproc.Issue{
		filmproc.IssueOK, filmproc.IssueSkipped, filmproc.IssueFailed,
		filmproc.IssuePreparation, filmproc.IssueMemory, filmproc.IssueSuddenDeath,
	} {
		if n := recap[i]; n > 0 {
			fmt.Printf("  %-12s %d\n", i.String(), n)
		}
	}
	if n := recap[filmproc.IssueMemory] + recap[filmproc.IssueSuddenDeath]; n > 0 {
		fmt.Printf("  ATTENTION : %d film(s) n'ont PAS ete mesures — le corpus de la mesure est "+
			"incomplet, et un taux calcule dessus ne porte pas sur ce qu'il annonce\n", n)
	}
}

// hasChildFlag dit si la ligne de commande fait de ce processus un enfant. Lu AVANT `flag.Parse`
// n'est pas necessaire : le drapeau est declare comme les autres. Cette fonction sert au
// garde-rail de compilation, qui verifie que la boucle passe bien par l'executeur.
func hasChildFlag(args []string) bool {
	for _, a := range args {
		if a == childFlag || strings.HasPrefix(a, childFlag+"=") {
			return true
		}
	}
	return false
}
