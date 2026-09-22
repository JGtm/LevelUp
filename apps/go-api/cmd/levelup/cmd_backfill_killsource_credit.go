package main

// cmd_backfill_killsource_credit.go — LA PASSE CREDIT de `levelup backfill-killsource`.
//
// Fichier dedie extrait de `cmd_backfill_killsource.go` le 2026-09-22 (lot 5.24.4), quand le
// cablage de l arret et du suivi a pousse celui-ci au-dela des 500 lignes. Aucun changement de
// comportement : le bloc est deplace tel quel. La coupure suit la frontiere la plus nette de la
// commande — LA-BAS la passe des FILMS (decodage, ouvriers, porte de la base), ICI la passe
// CREDIT (une transformation SQL -> SQL, en serie, lineaire depuis le lot 5.12).

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/sync/killcollector"
)

// passeDuCredit : la transformation SQL -> SQL, sur TOUS les matchs du registre.
//
// Elle passe sur tous les matchs et pas seulement sur ceux sans film : c est le producteur
// lui-meme qui applique la preseance (il refuse un match qu une passe de film couvre deja), et
// centraliser cette regle a UN endroit vaut mieux que de la recopier dans la selection.
func passeDuCredit(ctx context.Context, db *sql.DB, o killsourceOptions, suivi *suiviDeLaPasse) error {
	ids, err := matchsDuRegistre(ctx, db, o.limit)
	if err != nil {
		return err
	}
	fmt.Printf("credit-seul : %d matchs a examiner\n", len(ids))
	if o.dryRun || len(ids) == 0 {
		return nil
	}
	credit := killcollector.NewCreditCollector(db, writerDeja(db))
	if suivi != nil {
		// LE MEME FICHIER D ETAT POUR LES DEUX PASSES : la commande en est une, son etat aussi.
		suivi.PhaseCredit(len(ids))
		credit = credit.AvecProgression(func(examines, _ int, reste time.Duration) {
			suivi.ProgressionCredit(examines, reste)
		})
	}
	debut := time.Now()
	sum := credit.CollectMatches(ctx, ids)
	fmt.Printf("credit : %d ecrits + %d enrichis par un film (%d morts), "+
		"%d sans evenement, %d erreurs — %s\n",
		sum.Written, sum.Enriched, sum.Deaths, sum.NoEvents, sum.Errors,
		time.Since(debut).Round(time.Second))
	return nil
}
