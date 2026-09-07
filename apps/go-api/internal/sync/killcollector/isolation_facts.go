package killcollector

// isolation_facts.go — LA SECONDE PROJECTION DE LA PASSE DE POSITIONS : les faits d'isolement.
//
// # POURQUOI ICI, ET NON A LA CUISSON DU REJEU
//
// Decision utilisateur du 2026-09-07, ferme : « les donnees d'un match en base sont completes au
// sync ; seul le rejeu peut attendre la cuisson ». Une version precedente faisait dire au FILM,
// a la lecture d'une page, qui etait mort a l'instant d'une mort — elle le deduisait de la
// chronologie d'un artefact de rejeu. Deux defauts : le film ne sait pas dire qui est mort
// (`replay/owners.go` nomme aussi une vie par FERMETURE DE SLOT), et un fait de base se trouvait
// dependre du calendrier de cuisson.
//
// # AUCUN DECODAGE NOUVEAU
//
// `buildPositionRows` a DEJA tout lu : positions bipeds avec bornes de carte, fil des morts,
// index de joueur, pont slot->xuid et vies nommees. Ce fichier ne fait que projeter ce materiau
// une seconde fois. Rescanner le film aurait double le cout de la passe la plus chere du cycle
// pour des donnees deja en memoire.

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// Compteurs de sante des faits d'isolement (ADR 0009 : entiers, snake_case, aucun ratio).
const (
	metricIsolationMatches       = "killsource_isolement_matchs_couverts"
	metricIsolationLives         = "killsource_isolement_vies_ecrites"
	metricIsolationContexts      = "killsource_isolement_contextes_ecrits"
	metricIsolationNoTeams       = "killsource_isolement_sans_equipes"
	metricIsolationDeathsNoPlace = "killsource_isolement_morts_sans_lieu"
	metricIsolationWriteFail     = "killsource_isolement_erreurs_ecriture"
)

// projeterFaitsDIsolement ecrit `match_lives` et `match_death_context` a partir de ce que la
// passe de positions a deja lu.
//
// # ELLE N'EST JAMAIS BLOQUANTE
//
// Son echec ne doit couter ni le journal des morts ni les positions : ce sont deux ecritures
// deja faites, et beaucoup plus centrales au produit. Tout refus se journalise et se compte, et
// la passe continue. C'est la raison pour laquelle elle ne rend pas d'erreur a son appelant.
func (c *KillSourceCollector) projeterFaitsDIsolement(
	ctx context.Context, matchID string, mat materiauDIsolement, ids MatchIdentities,
	deaths []persist.KillEventInsert,
) {
	if len(ids.Equipes) == 0 {
		// SANS EQUIPES, LA QUESTION N'A PAS DE SENS : « isole » se mesure entre coequipiers, et
		// le film ne porte aucun camp. Un match dont `match_participants.team_id` est vide sort
		// de la lecture au lieu d'y entrer avec des camps devines.
		observability.AddInt(metricIsolationNoTeams, 1)
		slog.InfoContext(ctx, "killsource: isolement — aucune equipe en base, passe ignoree",
			"match_id", matchID)
		return
	}

	lives := toLifeRows(mat.report.ViesNommees())
	if len(lives) == 0 {
		slog.DebugContext(ctx, "killsource: isolement — aucune vie nommee, rien a projeter",
			"match_id", matchID)
		return
	}
	contexts := toDeathContextRows(mat, ids, deaths)
	if manquantes := len(deaths) - len(contexts); manquantes > 0 {
		observability.AddInt(metricIsolationDeathsNoPlace, int64(manquantes))
	}

	if err := c.writeIsolationFacts(ctx, matchID, persist.LivesBatch{
		MatchID: matchID, DecoderRev: IsolationDecoderRev, Lives: lives, Contexts: contexts,
	}); err != nil {
		observability.AddInt(metricIsolationWriteFail, 1)
		slog.ErrorContext(ctx, "killsource: isolement — ecriture echouee (le journal et les "+
			"positions restent ecrits)", "match_id", matchID, "err", err)
		return
	}
	observability.AddInt(metricIsolationMatches, 1)
	observability.AddInt(metricIsolationLives, int64(len(lives)))
	observability.AddInt(metricIsolationContexts, int64(len(contexts)))
	slog.InfoContext(ctx, "killsource: isolement — faits ecrits",
		"match_id", matchID, "vies", len(lives), "contextes", len(contexts),
		"morts_journal", len(deaths))
}

// writeIsolationFacts : l'ecriture, sous son PROPRE lease court — meme raison que writePositions
// (le lease RW de shared est la ressource la plus disputee du process, ADR 0013).
func (c *KillSourceCollector) writeIsolationFacts(ctx context.Context, matchID string, batch persist.LivesBatch) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", matchID, err)
	}
	defer release()
	return persist.NewLivesPersister(db).PersistPass(ctx, batch)
}

// IsolationDecoderRev — la version du producteur des faits d'isolement, ecrite sur CHAQUE ligne
// des deux tables.
//
// ELLE EST DISTINCTE DE [KillSourceDecoderRev] parce que les deux passes evoluent separement :
// un changement de la regle de visibilite ou de l'ordre des etats doit faire redecoder les faits
// d'isolement SANS forcer un redecodage du journal des morts, qui n'a pas bouge. Meme espace de
// valeurs, meme colonne `decoder_rev`, unites de fraicheur differentes.
const IsolationDecoderRev = "isolement-2026-09-07"

// materiauDIsolement : ce que la passe de positions a lu et que la projection reutilise.
type materiauDIsolement struct {
	report    replay.OwnerReport
	positions []filmdec.BipedPosition
	slotXUID  map[uint32]uint64
}

// toLifeRows traduit les vies pures en lignes ecrivables.
func toLifeRows(vies []replay.VieNommee) []persist.LifeInsert {
	out := make([]persist.LifeInsert, 0, len(vies))
	for _, v := range vies {
		out = append(out, persist.LifeInsert{
			XUID:     strconv.FormatUint(v.XUID, 10),
			StartMS:  v.DebutMS,
			EndMS:    v.FinMS,
			EndCause: v.Cause,
			NamedBy:  v.NomPar,
		})
	}
	return out
}

// toDeathContextRows calcule le contexte de chaque mort du JOURNAL et le traduit en lignes.
//
// LE JOURNAL EST LA SOURCE DES MORTS, pas le film. C'est la meme liste que celle qui part dans
// `match_kill_events`, donc les deux tables se joignent sur (match_id, victim_xuid, time_ms)
// sans rapprocher deux horloges.
func toDeathContextRows(mat materiauDIsolement, ids MatchIdentities,
	deaths []persist.KillEventInsert,
) []persist.DeathContextInsert {
	ctxs := replay.ContextesDesMorts(replay.EntreeContexteMorts{
		Positions:  mat.positions,
		SlotXUID:   mat.slotXUID,
		DecalageMS: mat.report.DeathOffsetMS,
		Journal:    journalDesMorts(deaths),
		Equipes:    equipesNumeriques(ids.Equipes),
		DepartMS:   departsNumeriques(ids.DepartMS),
	})
	out := make([]persist.DeathContextInsert, 0, len(ctxs))
	for _, c := range ctxs {
		out = append(out, persist.DeathContextInsert{
			VictimXUID:          strconv.FormatUint(c.VictimeXUID, 10),
			TimeMS:              c.TempsMS,
			NearestTeammateM:    c.PlusProcheM,
			TeammatesVisible:    c.Visibles,
			TeammatesWaiting:    c.EnAttente,
			TeammatesOutOfSight: c.HorsDeVue,
			TeammatesLeft:       c.Partis,
			TeammatesTotal:      c.Total,
		})
	}
	return out
}

// journalDesMorts ne garde que les morts dont la VICTIME est resolue. Une victime sans xuid
// (bot, nom non resolu) ne peut ni etre situee dans une equipe ni etre jointe au journal.
func journalDesMorts(deaths []persist.KillEventInsert) []replay.MortDuJournal {
	out := make([]replay.MortDuJournal, 0, len(deaths))
	for i := range deaths {
		v, ok := parseXUID(deaths[i].VictimXUID)
		if !ok {
			continue
		}
		out = append(out, replay.MortDuJournal{VictimeXUID: v, TempsMS: int64(deaths[i].TimeMS)})
	}
	return out
}

// equipesNumeriques / departsNumeriques traduisent les tables texte de MatchIdentities. Un xuid
// non decimal est ECARTE : il ne peut pas correspondre a un joueur du film.
func equipesNumeriques(par map[string]int) map[uint64]int {
	out := make(map[uint64]int, len(par))
	for s, t := range par {
		if v, ok := parseXUID(s); ok {
			out[v] = t
		}
	}
	return out
}

func departsNumeriques(par map[string]int64) map[uint64]int64 {
	out := make(map[uint64]int64, len(par))
	for s, t := range par {
		if v, ok := parseXUID(s); ok {
			out[v] = t
		}
	}
	return out
}
