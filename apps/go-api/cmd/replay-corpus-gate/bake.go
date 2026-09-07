package main

// bake.go — CUIRE UN TEMOIN, SOUS LES MEMES PROTECTIONS QUE `cmd/replay-build`.
//
// Un film a la fois, verrou d'exclusion PARTAGE (filmproc.AcquireSolo sur lockRoot — pas
// workRoot : cf. roots.go), plafond memoire dur, priorite basse. C'est LITTERALEMENT le
// patron de `cmd/replay-build` (armerProtections) et de l'enfant de `backfill-replay` —
// troisieme appelant de `replaybuild.Builder`, meme construction (NewBuilder + BuildMatch),
// zero copie de la logique d'assemblage elle-meme.

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/replaybuild"
)

// outilNom : le nom que ce binaire porte dans le journal des protections (verrou, sentinelle,
// priorite) — visible par l'operateur suivant si un refus le concerne.
const outilNom = "replay-corpus-gate"

// resultatCuisson porte l'artefact produit et le temps qu'il a coute.
type resultatCuisson struct {
	Sortie replaybuild.Outcome
	Duree  time.Duration
}

// bakeTemoin construit l'artefact d'UN temoin dans `workRoot`, jamais dans le parc. `facts`
// porte l'identite complete du match et ses cartes candidates (lues du fichier de faits, cf.
// facts.go) : `replaybuild.Builder.BuildMatch` en a besoin, le short8 seul ne suffit pas.
//
// UN DEPASSEMENT MEMOIRE ARRETE TOUT LE PROCESSUS (`os.Exit`), MEME PATRON QUE
// `cmd/replay-build` : la sentinelle protege la MACHINE, pas seulement ce temoin — aucune
// mesure sur les 119 films du parc n'a jamais tranche (pic max 0,56 Gio sur un plafond de
// 3 Gio), et un depassement reel merite d'arreter le gate plutot que de continuer sur des
// bases fausses.
func bakeTemoin(workRoot, lockRoot, titleSlug string, facts replaybuild.FactsFile, filmDir string) (resultatCuisson, error) {
	lock, err := filmproc.AcquireSolo(lockRoot, outilNom, facts.MatchID)
	if err != nil {
		return resultatCuisson{}, fmt.Errorf("verrou de decodage : %w", err)
	}
	filmproc.LowerOwnPriority(outilNom)
	g := filmproc.Arm(outilNom, filmproc.DefaultLimitGiB, func(peak uint64) {
		slog.Error("replay-corpus-gate: plafond memoire depasse — cuisson abandonnee",
			"temoin", facts.MatchID, "pic_octets", peak, "pic_gio", float64(peak)/(1<<30))
		lock.Release()
		os.Exit(filmproc.CodeMemory)
	})

	builder, err := replaybuild.NewBuilder(workRoot, titleSlug)
	if err != nil {
		lock.Release()
		return resultatCuisson{}, fmt.Errorf("preparation du builder : %w", err)
	}

	debut := time.Now()
	out, err := builder.BuildMatch(facts.MatchID, facts.MapNames, filmDir, facts.MatchFacts)
	duree := time.Since(debut)

	g.Disarm()
	pic := g.Peak()
	slog.Info("replay-corpus-gate: pic memoire de la cuisson",
		"temoin", facts.MatchID, "octets", pic, "gio", float64(pic)/(1<<30))
	lock.Release()

	if err != nil {
		return resultatCuisson{}, fmt.Errorf("construction de l'artefact : %w", err)
	}
	return resultatCuisson{Sortie: out, Duree: duree}, nil
}
