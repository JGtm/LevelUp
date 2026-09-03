// Command backfill_t0_film REPARE LE T0 DE L'HISTORIQUE avec le coup d'envoi MESURE dans le
// film, sans recuire un seul artefact.
//
// # LE DEFAUT QU'ELLE FERME
//
// `match_registry.real_start_time` porte le debut de gameplay, ESTIME des `first_joined_time`
// de l'API (`cmd/backfill_t0` / `analysis/timeline.ComputeT0`). Sur 10 a 15 % des matchs ces
// horodatages collent au `start_time` et le T0 tombe a ~0 alors que le decompte dure une
// vingtaine de secondes ; sur une autre part du corpus il est decale d'une heure entiere
// (dette de fuseau historique). Le film, lui, porte le coup d'envoi directement : la grille
// se leve d'un coup et tout le monde part dans la meme seconde.
//
// # CE QU'ELLE FAIT, ET CE QU'ELLE NE FAIT PAS
//
// Elle balaye les artefacts de rejeu DEJA SUR DISQUE, applique le detecteur de production
// (`replay.DetectT0Film`) a leurs pistes publiees, et reporte le resultat dans le registre.
// Elle NE DECODE AUCUN FILM et ne declenche aucune cuisson : les artefacts portent deja les
// positions, et c'est tout ce dont le detecteur a besoin (decision D4 du plan T0-film du
// 2026-09-02). La recuisson des artefacts pour y PUBLIER le champ `t0FilmMs` est un autre
// geste, qui ne se lance pas sans accord explicite.
//
// ELLE NE DEGRADE JAMAIS. Un refus du detecteur n'ecrit RIEN : le T0 de l'API deja en base
// reste en place. Aucune ligne ne repasse a NULL, aucune qualite n'est effacee.
//
// # USAGE
//
//	go run ./cmd/backfill_t0_film                    # SIMULATION (lecture seule) — le defaut
//	go run ./cmd/backfill_t0_film --commit           # ecrit en base
//	go run ./cmd/backfill_t0_film --title halo_5     # autre titre
//
// La racine du depot se resout par `title.FindRepoRoot()` : la variable d'environnement
// `LEVELUP_REPO_ROOT` si elle est posee, sinon la remontee depuis le repertoire courant. Les
// chemins (artefacts, base partagee) sortent ensuite du `PathResolver` — jamais assembles a la
// main (ADR 0008).
//
// ⚠ `--commit` EXIGE LA BASE POUR LUI SEUL : arreter le serveur local avant, un seul process
// writer par DB (ADR 0013/0016). La SIMULATION, elle, ouvre en lecture seule et cohabite.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain/title"
)

func main() {
	slug := flag.String("title", title.DefaultSlug, "slug du titre")
	commit := flag.Bool("commit", false,
		"ecrit en base (defaut : simulation, lecture seule). Arreter le serveur local avant : "+
			"un seul process writer par DB.")
	flag.Parse()

	if err := run(context.Background(), *slug, *commit); err != nil {
		slog.Error("reparation du T0 par le film", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, slug string, commit bool) error {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return err
	}
	paths := title.NewPathResolver(repoRoot)
	dir := paths.ReplayArtifactsDir(slug)
	verdicts, err := scannerArtefacts(dir)
	if err != nil {
		return err
	}
	db, release, err := ouvrirBase(paths.SharedDBPath(slug), commit)
	if err != nil {
		return fmt.Errorf("ouverture de la base partagee (serveur local encore actif ?): %w", err)
	}
	defer release()
	registre, err := chargerRegistre(ctx, db)
	if err != nil {
		return err
	}
	b := confronter(verdicts, registre)
	imprimerBilan(b, slug, dir)
	if !commit {
		fmt.Println("\n[SIMULATION] aucune ecriture. Relancer avec --commit pour persister.")
		return nil
	}
	n, err := ecrireReparations(ctx, db, b.reparations)
	if err != nil {
		return err
	}
	fmt.Printf("\n[COMMIT] %d ligne(s) de match_registry reparee(s).\n", n)
	return nil
}

// reparation : UNE ligne de registre a corriger, avec l'ancien et le nouveau T0 — les DEUX,
// parce qu'une reparation d'historique qui ne publie pas ce qu'elle remplace n'est pas
// verifiable.
type reparation struct {
	fichier string
	matchID string
	// ancienMs est le T0 actuellement en base (`real_start_time` − start canonique), en ms.
	// Invalide quand `real_start_time` est NULL.
	ancienMs  sql.NullInt64
	ancienneQ string
	// nouveauMs est le T0 mesure dans le film, sur le meme axe.
	nouveauMs int64
	// nouveau est la valeur ecrite : start canonique + nouveauMs, en UTC.
	nouveau time.Time
}

// bilan : le compte rendu complet d'une passe. Chaque artefact du corpus tombe dans
// EXACTEMENT une categorie — c'est ce qui rend le total verifiable.
type bilan struct {
	reparations []reparation
	// refus : nombre d'artefacts par raison (les quatre refus du detecteur et les trois
	// causes d'artefact inexploitable).
	refus map[string]int
	// sansLigne : artefacts dont le `matchId` n'existe pas au registre.
	sansLigne []string
	// inchanges : deja `film_movement` a la MEME valeur — rien a reecrire.
	inchanges int
	total     int
}

// confronter croise les verdicts du corpus avec le registre.
func confronter(verdicts []verdictArtefact, registre map[string]ligneRegistre) bilan {
	b := bilan{refus: map[string]int{}, total: len(verdicts)}
	for _, v := range verdicts {
		if !v.detecte {
			b.refus[v.raison]++
			continue
		}
		ligne, ok := registre[v.matchID]
		if !ok {
			b.sansLigne = append(b.sansLigne, v.fichier)
			continue
		}
		r := reparation{
			fichier: v.fichier, matchID: v.matchID, ancienneQ: ligne.qualite,
			nouveauMs: v.t0FilmMs,
			nouveau:   ligne.startUTC.Add(time.Duration(v.t0FilmMs) * time.Millisecond).UTC(),
		}
		if ligne.realStart.Valid {
			r.ancienMs = sql.NullInt64{
				Valid: true,
				Int64: ligne.realStart.Time.UTC().Sub(ligne.startUTC).Milliseconds(),
			}
		}
		if dejaAJour(ligne, r) {
			b.inchanges++
			continue
		}
		b.reparations = append(b.reparations, r)
	}
	return b
}

// dejaAJour dit si la ligne porte DEJA cette mesure. Le geste serait sans effet : le repeter
// ferait une ecriture par passe sur tout le corpus deja repare.
func dejaAJour(ligne ligneRegistre, r reparation) bool {
	return ligne.qualite == string(timeline.T0QualityFilmMovement) &&
		r.ancienMs.Valid && r.ancienMs.Int64 == r.nouveauMs
}

// imprimerBilan publie le compte rendu. Sortie CLI volontaire (fmt), pas du journal.
func imprimerBilan(b bilan, slug, dir string) {
	fmt.Printf("=== REPARATION DU T0 PAR LE FILM — %s ===\n\n", slug)
	fmt.Printf("  corpus : %d artefact(s) balaye(s) dans %s\n", b.total, dir)
	imprimerReparations(b.reparations)
	imprimerRefus(b.refus)
	if n := len(b.sansLigne); n > 0 {
		fmt.Printf("\n  SANS LIGNE AU REGISTRE (%d) — artefact d'un match absent de match_registry\n", n)
		for _, f := range b.sansLigne {
			fmt.Printf("    %s\n", f)
		}
	}
	fmt.Printf("\n  INCHANGES (deja film_movement a la meme valeur) : %d\n", b.inchanges)
	fmt.Printf("  TOTAL : %d reparation(s) + %d refus + %d sans ligne + %d inchange(s) = %d\n",
		len(b.reparations), totalRefus(b.refus), len(b.sansLigne), b.inchanges, b.total)
}

// imprimerReparations publie l'ancien ET le nouveau T0 de chaque match a reparer.
func imprimerReparations(reps []reparation) {
	fmt.Printf("\n  REPARATIONS (%d)\n", len(reps))
	if len(reps) == 0 {
		return
	}
	fmt.Printf("    %-14s %12s  %-16s %12s %12s\n",
		"artefact", "T0 actuel", "qualite", "T0 film", "ecart")
	for _, r := range reps {
		ancien, ecart := "(NULL)", ""
		if r.ancienMs.Valid {
			ancien = fmt.Sprintf("%d ms", r.ancienMs.Int64)
			ecart = fmt.Sprintf("%+d ms", r.nouveauMs-r.ancienMs.Int64)
		}
		qualite := r.ancienneQ
		if qualite == "" {
			qualite = "(vide)"
		}
		fmt.Printf("    %-14s %12s  %-16s %9d ms %12s\n",
			r.fichier, ancien, qualite, r.nouveauMs, ecart)
	}
	imprimerDispersion(reps)
}

// imprimerDispersion publie min / mediane / max des T0 ecrits. C'est le TEMOIN DE PLAUSIBILITE
// du lot : la mesure fondatrice place le coup d'envoi dans la plage 15-45 s, et une passe qui
// sortirait de cette plage doit se voir sans rouvrir un artefact.
func imprimerDispersion(reps []reparation) {
	vals := make([]int64, 0, len(reps))
	for _, r := range reps {
		vals = append(vals, r.nouveauMs)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	fmt.Printf("    T0 film min / mediane / max : %d / %d / %d ms\n",
		vals[0], vals[len(vals)/2], vals[len(vals)-1])
}

// imprimerRefus publie les refus par raison, tries par effectif decroissant.
func imprimerRefus(refus map[string]int) {
	fmt.Printf("\n  REFUS — aucune ecriture, le T0 de l'API reste en place (%d)\n", totalRefus(refus))
	raisons := make([]string, 0, len(refus))
	for r := range refus {
		raisons = append(raisons, r)
	}
	sort.Slice(raisons, func(i, j int) bool {
		if refus[raisons[i]] != refus[raisons[j]] {
			return refus[raisons[i]] > refus[raisons[j]]
		}
		return raisons[i] < raisons[j]
	})
	for _, r := range raisons {
		fmt.Printf("    %-22s %4d\n", r, refus[r])
	}
}

func totalRefus(refus map[string]int) int {
	n := 0
	for _, c := range refus {
		n += c
	}
	return n
}
