package killcollector

// registry_flags.go — LES MARQUEURS DE FILM DU REGISTRE, POSES PAR L ETAPE 1.57.
//
// # CE QU ILS DISENT, ET CE QU ILS N ONT JAMAIS DIT
//
//	MBitFilmAbsent (1<<22)   le film Theater de ce match est DEFINITIVEMENT perdu
//	                         (404/410, ou manifeste a 0 chunk). Marqueur TERMINAL.
//	MBitWeaponKills (1<<21)  une passe de film a produit le detail par arme du match.
//
// Ni l un ni l autre n a jamais ete une donnee de la table `weapon_kills` : ce sont des
// faits sur le FILM. Ils etaient poses par l etape 1.55 (`MarkWeaponKillsDone`), supprimee
// le 2026-09-01 avec le producteur de corrélation. Leur poseur demenage ICI, dans l etape
// 1.57, qui telecharge le film du MEME match, au MEME moment, et qui sait donc exactement
// ce que les deux bits affirment. Semantique strictement identique, seul le producteur
// change : `snapshot_readiness` et `data_health_check` ne sont pas touches.
//
// # POURQUOI LE DEMENAGEMENT ETAIT OBLIGATOIRE, ET PAS LA SUPPRESSION
//
// Trois lecteurs VIVANTS dependent de MBitFilmAbsent, tous hors de la chaine supprimee :
// le rattrapage de l etape 1.57 (`conditionBacklog`), celui de l etape 1.58
// (`replayartifacts`) et `snapshot/snapshot_readiness.go` (motif `weapons_absent`). Le
// supprimer les casserait a la compilation ; supprimer son seul poseur figerait le
// marqueur, et les deux rattrapages redemanderaient A VIE les ~29 % de films
// irrecuperables (581 des 999 candidats du 2026-08-29).
//
// # BIT-HONNETE
//
// La garde d origine est conservee mot pour mot : MBitWeaponKills n est pose QUE si la
// passe a reellement publie au moins une ligne. Un bit pose sans donnee est un « bit
// menteur » — `scheduler/data_health_check.go` les compte, et c est exactement l anomalie
// qu il surveille.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/sync/matchflags"
)

// marquerRegistre pose UN bit sur `match_registry.backfill_completed`.
//
// UPDATE row-by-row, serialise par le lease RW (ADR 0013) : `match_registry` n est pas
// append-only et ses UPDATE bitmask sont le patron etabli du depot (cf. l en-tete de
// `no_art_patterns_test.go`, qui les exclut explicitement du scan ART).
func marquerRegistre(ctx context.Context, db *sql.DB, matchID string, bit int) error {
	_, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?
	`, bit, matchID)
	if err != nil {
		return fmt.Errorf("marquerRegistre(%s, bit=%d): %w", matchID, bit, err)
	}
	return nil
}

// marquerFilmParOutcome traduit le resultat d UN match en marqueur de registre, et
// n ecrit rien quand il n y a rien a affirmer.
//
// Le tableau complet des correspondances :
//
//	OutcomeNoFilm                      MBitFilmAbsent   le film est perdu, definitivement
//	OutcomeWritten avec morts > 0      MBitWeaponKills  le detail par arme existe
//	OutcomeWritten avec morts == 0     rien             bit-honnete : pas de ligne, pas de bit
//	OutcomeNoKillFeed                  rien             le film EXISTE, il est juste muet
//	OutcomeTimeout / NotSupported      rien             etat transitoire ou hors titre
//
// ⚠ `OutcomeNoKillFeed` NE POSE PAS MBitFilmAbsent, et c est delibere : le film est bien
// la, il ne porte simplement pas de chunk HIGHLIGHT. Le marquer « absent » le retirerait
// pour toujours des deux rattrapages alors qu une revision de decodeur pourrait en tirer
// quelque chose.
func marquerFilmParOutcome(outcome KillSourceOutcome, morts int) (bit int, aMarquer bool) {
	switch {
	case outcome == OutcomeNoFilm:
		return matchflags.MBitFilmAbsent, true
	case outcome == OutcomeWritten && morts > 0:
		return matchflags.MBitWeaponKills, true
	default:
		return 0, false
	}
}

// marquerFilm ecrit le marqueur d UN match, sous un lease RW court.
//
// BEST-EFFORT SIGNALE : un echec d ecriture ne remet pas en cause la passe (les morts sont
// deja en base) mais il se journalise — un marqueur manquant se traduit par un match
// redemande au cycle suivant, jamais par une perte de donnee.
func (c *KillSourceCollector) marquerFilm(ctx context.Context, matchID string, outcome KillSourceOutcome, morts int) {
	bit, aMarquer := marquerFilmParOutcome(outcome, morts)
	if !aMarquer || c.acquireShared == nil {
		return
	}
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		slog.WarnContext(ctx, "killsource: marqueur de film non pose — lease shared indisponible",
			"match_id", matchID, "bit", bit, "err", err)
		return
	}
	defer release()
	if err := marquerRegistre(ctx, db, matchID, bit); err != nil {
		slog.WarnContext(ctx, "killsource: marqueur de film non pose",
			"match_id", matchID, "bit", bit, "err", err)
	}
}
