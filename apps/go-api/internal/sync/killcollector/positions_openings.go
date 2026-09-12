package killcollector

// positions_openings.go — LA PASSE D ENTAMES : `shared.kill_openings`, la table sœur des
// positions (D5 du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md).
//
// POURQUOI UN FICHIER SEPARE DE positions.go : les deux passes sortent de LA MEME lecture du
// film (cf. composerPassePositions) mais elles n ont ni la meme couverture, ni les memes
// compteurs, ni le meme regime d echec — et positions.go depassait le plafond de 500 lignes du
// depot en les portant toutes les deux. La frontiere suit celle de la donnee : ce qui est
// PROPRE a l entame vit ici, la lecture du film et la composition restent la-bas.
//
// LE time_ms D UNE ENTAME EST CELUI DU COUP FATAL, ses coordonnees sont celles d un
// temps-pour-tuer plus tot. C est le fait le plus facile a se tromper de tout le lot, et il est
// garanti en amont : `replay.BuildKillOpenings` rend deja l instant du kill.

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// Compteurs de sante de la passe d ENTAMES (D5). Famille distincte de celle des positions, et
// ce n est pas un detail de nommage : la couverture des deux mesures ne peut PAS etre la meme
// (une mort survenue dans la premiere seconde et demie d un match n a pas d entame lisible),
// donc les additionner sous un meme compteur rendrait les deux illisibles.
const (
	metricOpeningsMatches    = "killsource_openings_matchs_couverts"
	metricOpeningsRows       = "killsource_openings_lignes_ecrites"
	metricOpeningsKillsNoPos = "killsource_openings_morts_sans_position"
	// metricOpeningsOutOfLife compte les COTES (pas les morts) dont la position d entame a ete
	// ECARTEE parce que le joueur avait reapparu entre l entame et le coup fatal : le point
	// d apparition n est pas une entame. Sans ce compteur, la seule trace d un film ou toutes
	// les entames tombent serait une couverture basse, sans cause lisible.
	metricOpeningsOutOfLife   = "killsource_openings_cotes_hors_vie"
	metricOpeningsWriteErrors = "killsource_openings_erreurs_ecriture"
)

// persistOpenings : la QUATRIEME ecriture de la passe — `shared.kill_openings` (D5).
//
// BEST-EFFORT AU CARRE, ET DANS LES DEUX SENS. Les positions sont deja un enrichissement
// troisieme ; l entame est un PROXY par-dessus. Son echec ne fait echouer NI la passe des morts
// (deja ecrite), NI celle des positions — il se journalise et se compte, il n interrompt rien.
// RECIPROQUEMENT, depuis la revue du 2026-09-06 (constat C7), un echec d ecriture des POSITIONS
// ne l annule plus : `ecrireLesDeuxPasses` l appelle dans tous les cas. Zero ligne n est pas une
// erreur non plus : c est l etat d un film dont aucune mort n a d echantillon un temps-pour-tuer
// plus tot, ou dont tous les joueurs avaient reapparu entre-temps.
func (c *KillSourceCollector) persistOpenings(ctx context.Context, matchID string, pass passePositions) {
	if len(pass.openRows) > 0 {
		if err := c.writeOpenings(ctx, matchID, pass.openRows); err != nil {
			observability.AddInt(metricOpeningsWriteErrors, 1)
			// LES COMPTEURS DE LECTURE SORTENT MEME ICI (residu 4.0d, 2026-09-06). Le film A
			// ETE LU : les morts non localisables et les cotes ecartes par le filtre de vie
			// sont des faits acquis, que l ecriture reussisse ou non. Les taire sur echec
			// faisait mentir la doc de publishOpeningsPass (« il compte MEME quand rien n est
			// ecrit ») et rendait un incident d ecriture indiscernable d une passe qui n avait
			// rien trouve. Ce qui reste conditionne au SUCCES, c est la couverture
			// (`matchs_couverts`, `lignes_ecrites`) : elle decrit ce qui est EN BASE.
			publishOpeningsReadCounters(pass.openRep)
			slog.ErrorContext(ctx, "killsource: entames — ecriture echouee",
				"match_id", matchID, "err", err,
				"kills", pass.openRep.Kills, "sans_position", pass.openRep.Dropped,
				"cotes_hors_vie", pass.openRep.OpeningOutOfLife)
			return
		}
	}
	publishOpeningsPass(ctx, matchID, pass.openRep, len(pass.openRows))
}

// publishOpeningsReadCounters : les deux pertes de LECTURE de la passe d entames.
//
// Elles decrivent ce que le decodeur a vu, jamais ce qui a ete ecrit : les morts sans aucune
// position (`Dropped`) et les COTES ecartes parce que le joueur avait reapparu entre l entame
// et le coup fatal (`OpeningOutOfLife`). D ou l extraction : les deux chemins de sortie de
// persistOpenings — succes et echec d ecriture — les publient, et chacun exactement une fois.
func publishOpeningsReadCounters(rep replay.KillPosReport) {
	if rep.Dropped > 0 {
		observability.AddInt(metricOpeningsKillsNoPos, int64(rep.Dropped))
	}
	if rep.OpeningOutOfLife > 0 {
		observability.AddInt(metricOpeningsOutOfLife, int64(rep.OpeningOutOfLife))
	}
}

// writeOpenings : l ecriture des entames, sous SON PROPRE lease court — meme raison que
// writePositions, et un lease SEPARE a dessein : les deux passes ne doivent pas se tenir
// mutuellement, un echec de l une ne fait pas retomber l autre.
func (c *KillSourceCollector) writeOpenings(ctx context.Context, matchID string, rows []persist.KillOpeningInsert) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", matchID, err)
	}
	defer release()
	return persist.NewKillOpeningPersister(db).PersistPass(ctx, matchID, rows)
}

// toKillOpeningRows traduit les positions d ENTAME en lignes ecrivables. PROJECTION DEDIEE, et
// pas `toKillPositionRows` appliquee a la sortie de l entame — les deux types de row visent deux
// tables et deux mesures, un type partage laisserait ecrire l une dans l autre.
//
// LE time_ms EST CELUI DU KILL, ET C EST LE POINT CRITIQUE. La table est clee sur l instant DU
// COUP FATAL — c est par lui que `match_kill_events_latest` se joint (`kp.time_ms = e.time_ms`).
// Ecrire l instant DECALE rendrait la jointure du lecteur VIDE, en silence : zero entame, aucune
// erreur, rien a voir dans les journaux.
//
// AUCUNE ARITHMETIQUE ICI, ET C EST LA BASCULE DU 2026-09-06 (item 3.10 bis du plan) :
// `replay.BuildKillOpenings` rend DEJA l instant du kill. La version precedente appelait
// `BuildKillPositions` sur des couples decales et readditionnait `replay.OpeningLeadMS` — les
// deux gestes etaient indissociables, et ils ont bascule ENSEMBLE. Readditionner ici par-dessus
// la nouvelle fonction decalerait toutes les lignes de 1,5 s.
func toKillOpeningRows(matchID string, openings []replay.KillPosition) []persist.KillOpeningInsert {
	out := make([]persist.KillOpeningInsert, 0, len(openings))
	for i := range openings {
		p := &openings[i]
		row := persist.KillOpeningInsert{
			MatchID:    matchID,
			KillerXUID: strconv.FormatUint(p.KillerXUID, 10),
			TimeMS:     int(p.TimeMS),
		}
		if p.Killer != nil {
			row.KillerX, row.KillerY, row.KillerZ = &p.Killer.X, &p.Killer.Y, &p.Killer.Z
		}
		if p.Victim != nil {
			row.VictimX, row.VictimY, row.VictimZ = &p.Victim.X, &p.Victim.Y, &p.Victim.Z
		}
		out = append(out, row)
	}
	return out
}

// publishOpeningsPass : les compteurs de sante de la passe d entames (ADR 0009) et sa trace.
//
// ZERO LIGNE NE COMPTE PAS UN MATCH COUVERT. La couverture de l entame est structurellement
// partielle (cf. persistOpenings), donc compter un match « couvert » sans aucune ligne rendrait
// le compteur muet sur la seule question qu il sert a poser : sur combien de matchs ce proxy
// est-il reellement mesure.
//
// LES DEUX PERTES SE COMPTENT AVANT CE PARTAGE, parce qu elles decrivent la LECTURE et pas
// l ecriture (`publishOpeningsReadCounters`) : les morts sans aucune position (`Dropped`) et
// les COTES ecartes par le filtre de vie (`OpeningOutOfLife` — le joueur avait reapparu entre
// l entame et le coup fatal). Un film dont toutes les entames tombent doit se lire comme tel,
// pas comme une couverture basse sans cause. Depuis le residu 4.0d (2026-09-06), le chemin
// d ECHEC D ECRITURE les publie lui aussi, et cette fonction n est alors pas appelee.
func publishOpeningsPass(ctx context.Context, matchID string, rep replay.KillPosReport, rowsWritten int) {
	publishOpeningsReadCounters(rep)
	if rowsWritten == 0 {
		slog.InfoContext(ctx, "killsource: entames — aucune position d entame sur ce film",
			"match_id", matchID, "kills", rep.Kills, "sans_position", rep.Dropped,
			"cotes_hors_vie", rep.OpeningOutOfLife)
		return
	}
	observability.AddInt(metricOpeningsMatches, 1)
	observability.AddInt(metricOpeningsRows, int64(rowsWritten))
	slog.InfoContext(ctx, "killsource: entames decodees",
		"match_id", matchID, "kills", rep.Kills, "deux_cotes", rep.Both,
		"tueur_seul", rep.KillerOnly, "victime_seule", rep.VictimOnly,
		"sans_position", rep.Dropped, "cotes_hors_vie", rep.OpeningOutOfLife,
		"sans_pont_identite", rep.NoBridge, "lignes", rowsWritten)
}
