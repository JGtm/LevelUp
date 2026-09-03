package main

// cmd_backfill_replay_passe.go — LA PASSE DU PARENT : un processus enfant par film.
//
// Le parent ne decode RIEN. Il lance, attend, traduit un code de sortie en categorie,
// compte, et PASSE AU SUIVANT — quoi qu'il soit arrive au precedent. C'est la seule
// propriete qui compte ici : la sante de la passe ne depend plus de la sante d'un film.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"levelup/go-api/internal/filmproc"
)

// replayBackfillReport : le rapport par categories.
//
// LES ECHECS SONT VENTILES, PAS ADDITIONNES. « Carte hors catalogue » est un echec VOULU
// (cartes Forge sans bornes) ; une mort memoire est un film-bombe a instruire ; une mort
// subite est un incident machine. Les fondre dans un seul compteur « erreurs » rendrait le
// recap inutile le jour ou il sert vraiment.
type replayBackfillReport struct {
	construits    int
	dejaAJour     int
	horsCatalogue int // echec VOULU (cartes Forge sans bornes), compte A PART
	horsRegistre  int // film en cache sans ligne match_registry
	sansArtefact  int // ecarte par --only-existing : aucun artefact sur disque
	erreurs       int // decodage en echec, film present
	preparation   int // l'enfant n'a pas pu commencer (builder, catalogue, ...)
	mortsMemoire  int // plafond memoire depasse — film-bombe
	mortsSubites  int // code hors protocole : crash, OOM systeme, tue par l'OS

	// Ventilation propre au mode --repair-impoverished (0 en passe ordinaire).
	dejaComplets      int // a jour AVEC compteurs de joueur — rien a reparer
	vacuitesLegitimes int // a jour, sans compteurs, mais base SANS joueur — re-cuire ne donnerait rien
	horsSchemaCourant int // artefact d'une version anterieure — domaine de --only-existing ordinaire
}

// executerPasseReplay lance un enfant par film, sequentiellement.
func executerPasseReplay(
	ctx context.Context, repoRoot string, o replayBackfillOptions, cacheRoot string,
	aFaire []replayCandidat, r *replayBackfillReport,
) error {
	// LE LANCEUR EST CELUI DU DEPOT (`internal/filmproc`, PLAN_CUISSON_PERF item 5.4) : meme
	// protocole de codes de sortie, meme interception du marqueur de pic memoire, meme relais
	// de journal — et EN PLUS la priorite CPU basse, qui manquait a la copie locale. Cette
	// passe tourne sur la machine de travail de l'utilisateur pendant qu'il s'en sert.
	runner, err := filmproc.NewRunner(repoRoot, os.Stdout)
	if err != nil {
		return err
	}
	fmt.Printf("mode parent/enfant : 1 processus par film, plafond memoire %s (racine %s)\n",
		libellePlafond(o.memLimitGiB), repoRoot)

	debut := time.Now()
	var picMax uint64
	for i, c := range aFaire {
		res := runner.Run(ctx, argsEnfantReplay(o, cacheRoot, c))
		if res.Peak > picMax {
			picMax = res.Peak
		}
		traiterResultatEnfant(r, res, c.matchID, i+1, len(aFaire))
	}
	fmt.Printf("passe terminee en %s (pic memoire max observe : %s)\n",
		time.Since(debut).Round(time.Second), libelleOctets(picMax))
	afficherRapportReplay(*r)
	return nil
}

// argsEnfantReplay : la ligne de commande de l'enfant pour CE film.
//
// L'enfant ne recoit AUCUN drapeau de planification (`--force`, `--limit`, `--only-existing`,
// `--dry-run`) : ces arbitrages sont deja rendus par le parent, et les repasser ouvrirait la
// porte a un enfant qui saute le film qu'on vient justement de decider de cuire.
func argsEnfantReplay(o replayBackfillOptions, cacheRoot string, c replayCandidat) []string {
	args := []string{
		"backfill-replay",
		"--one", c.matchID,
		"--cache", cacheRoot,
		"--title", o.titleSlug,
		"--mem-limit-gib", strconv.Itoa(o.memLimitGiB),
	}
	for _, n := range c.mapNames {
		args = append(args, "--map-name", n)
	}
	return args
}

// traiterResultatEnfant compte l'issue, la journalise et l'affiche.
func traiterResultatEnfant(r *replayBackfillReport, res filmproc.Result, matchID string, rang, total int) {
	switch res.Issue {
	case filmproc.IssueOK:
		r.construits++
	case filmproc.IssueSkipped:
		r.horsCatalogue++
	case filmproc.IssueFailed:
		r.erreurs++
	case filmproc.IssuePreparation:
		r.preparation++
	case filmproc.IssueMemory:
		r.mortsMemoire++
	default:
		r.mortsSubites++
	}
	if res.Issue != filmproc.IssueOK && res.Issue != filmproc.IssueSkipped {
		slog.Error("backfill-replay: enfant en echec — LA PASSE CONTINUE",
			"match_id", matchID, "issue", libelleIssue(res.Issue),
			"code", res.Code, "err", res.Err,
			"duree_s", res.Dur.Seconds(), "pic_octets", res.Peak)
	}
	fmt.Printf("  [%d/%d] %s : %s — code %d, %s%s\n", rang, total, matchID,
		libelleIssue(res.Issue), res.Code, res.Dur.Round(time.Second), suffixePic(res.Peak))
}

// libelleIssue : le mot que lit l'operateur.
//
// LES MOTS SONT CEUX DE LA PASSE, PAS CEUX DU LANCEUR. `filmproc.Issue.String()` rend des
// libelles de journal generiques (« ecarte », « echec ») ; ici on nomme ce que la categorie veut
// dire POUR UNE CUISSON D'ARTEFACT — « carte hors catalogue » est un echec VOULU, et l'operateur
// qui lit le recap doit le distinguer d'une erreur de decodage sans consulter un tableau.
func libelleIssue(i filmproc.Issue) string {
	switch i {
	case filmproc.IssueOK:
		return "construit"
	case filmproc.IssueSkipped:
		return "carte hors catalogue (echec voulu)"
	case filmproc.IssueFailed:
		return "ERREUR de decodage"
	case filmproc.IssuePreparation:
		return "ECHEC de preparation"
	case filmproc.IssueMemory:
		return "MORT MEMOIRE (plafond depasse)"
	default:
		return "MORT SUBITE (crash / tue par l'OS)"
	}
}

// libellePlafond : le plafond memoire, ou son absence.
func libellePlafond(gib int) string {
	if gib <= 0 {
		return "DESARME"
	}
	return strconv.Itoa(gib) + " GiB"
}

// libelleOctets : une taille lisible. 0 = jamais mesure.
func libelleOctets(n uint64) string {
	if n == 0 {
		return "inconnu"
	}
	if n >= octetsParGiB {
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(octetsParGiB))
	}
	return fmt.Sprintf("%.0f MiB", float64(n)/(1024*1024))
}

// suffixePic : le pic memoire de l'enfant, quand il a eu le temps de le rendre.
func suffixePic(n uint64) string {
	if n == 0 {
		return " (pic inconnu)"
	}
	return " (pic " + libelleOctets(n) + ")"
}

// afficherPlanReplay : le plan de passe, avec sa queue de films chers en evidence.
func afficherPlanReplay(candidats []replayCandidat) {
	total, gros := 0, 0
	for _, c := range candidats {
		total += c.chunks
		if c.chunks > 50 {
			gros++
		}
	}
	fmt.Printf("  chunks a decoder : %d au total, %d film(s) au-dela de 50 chunks (passes en dernier)\n", total, gros)
	for i, c := range candidats {
		if i >= 5 && i < len(candidats)-3 {
			continue
		}
		if i == 5 {
			fmt.Println("  ...")
		}
		fmt.Printf("  %-40s %3d chunks  %v\n", c.matchID, c.chunks, c.mapNames)
	}
}

// afficherRapportReplay : le rapport final par categories.
func afficherRapportReplay(r replayBackfillReport) {
	fmt.Println("rapport de passe :")
	fmt.Printf("  construits           %d\n", r.construits)
	fmt.Printf("  deja a jour          %d\n", r.dejaAJour)
	fmt.Printf("  carte hors catalogue %d (echec voulu : cartes sans bornes, Forge en tete)\n", r.horsCatalogue)
	fmt.Printf("  hors registre        %d (film en cache sans match en base)\n", r.horsRegistre)
	fmt.Printf("  sans artefact        %d (ecartes par --only-existing)\n", r.sansArtefact)
	fmt.Printf("  erreurs de decodage  %d\n", r.erreurs)
	fmt.Printf("  echecs de preparation %d (l enfant n a pas pu demarrer)\n", r.preparation)
	fmt.Printf("  morts memoire        %d (plafond depasse — film-bombe, passe poursuivie)\n", r.mortsMemoire)
	fmt.Printf("  morts subites        %d (crash / tue par l OS — passe poursuivie)\n", r.mortsSubites)
	// Lignes propres au mode reparation : muettes en passe ordinaire (elles y valent 0).
	if r.dejaComplets+r.vacuitesLegitimes+r.horsSchemaCourant > 0 {
		fmt.Printf("  deja complets        %d (artefact a jour avec compteurs de joueur — rien a reparer)\n", r.dejaComplets)
		fmt.Printf("  vacuites legitimes   %d (a jour, sans compteurs, mais aucun joueur en base — re-cuire ne donnerait rien)\n", r.vacuitesLegitimes)
		fmt.Printf("  hors schema courant  %d (version anterieure — passe --only-existing ordinaire d apres bump)\n", r.horsSchemaCourant)
	}
}
