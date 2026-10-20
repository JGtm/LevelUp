package skill

// skill_v2_components.go — les 8 composantes de performance d'un match
// (lusr_component_history), écrites par le chemin LUSR v2.
//
// Les composantes décrivent CE QUE le joueur a fait dans le match (frags et morts contre
// l'attendu, issue, efficacité des dégâts, précision, exploit de médailles, rendement
// offensif, résistance défensive) : elles se calculent des statistiques du match et de
// moyennes glissantes du joueur, jamais de la note (mu, sigma). Prestige, le profil et
// l'escouade les lisent. Le calcul est celui de computeCompositeScoreWithBreakdown
// (skill_rating_composite.go) ; ce fichier ne porte que l'orchestration par match.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/persist"
)

// componentHistories : fenêtres glissantes d'un joueur dans un groupe de modes (au plus
// AccuracyHistorySize valeurs), dont les moyennes servent de référence aux composantes
// « relatives » (précision, efficacité des dégâts, exploit, rendement, résistance).
type componentHistories struct {
	accuracy, damageEff, medalExploit, offConv, defRes []float64
}

// componentTracker porte, pour UN cycle du shadow, le score d'exploit médailles du joueur
// (chargé une fois sous lecture) et ses fenêtres glissantes par groupe. Les fenêtres
// repartent à vide à chaque cycle : un rejeu complet (reset du filigrane) les reconstruit
// sur tout l'historique, dans l'ordre.
type componentTracker struct {
	medals  map[string]float64
	history map[string]*componentHistories
}

func newComponentTracker(medals map[string]float64) *componentTracker {
	return &componentTracker{medals: medals, history: make(map[string]*componentHistories)}
}

// rowsFor calcule les composantes du match `matchID` du joueur `xuid` dans le groupe `group`
// et fait avancer les fenêtres du groupe. Aucun résultat (nil, nil) quand le match n'a pas
// d'issue ou qu'aucune composante n'est calculable. sharedDB est le handle de la rafale.
func (c *componentTracker) rowsFor(ctx context.Context, sharedDB *sql.DB, xuid, matchID, group string) ([]persist.LUSRComponentInsert, error) {
	matches, err := queryLUSRMatchData(ctx, sharedDB, xuid, matchID)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("composantes: match %s introuvable parmi les matchs LUSR du joueur", matchID)
	}
	participants, err := loadLUSRParticipants(ctx, sharedDB, []string{matchID})
	if err != nil {
		return nil, err
	}
	h := c.history[group]
	if h == nil {
		h = &componentHistories{}
		c.history[group] = h
	}
	breakdown := matchComponents(matches[0], participants[matchID], c.medals[matchID], h)
	return componentRows(matchID, breakdown), nil
}

// matchComponents calcule le breakdown d'un match puis fait avancer les fenêtres. Vide pour
// un match sans issue (rien n'est calculé ni retenu). Pur : aucun accès base.
func matchComponents(m LusrMatchData, allParts []lusrParticipant, medalScore float64, h *componentHistories) map[string]float64 {
	if m.Outcome == nil {
		return nil
	}
	_, enemyKEs := splitParticipantKEs(m.TeamID, allParts)
	var enemyAvgKE *float64
	if len(enemyKEs) > 0 {
		sum := 0.0
		for _, ke := range enemyKEs {
			sum += ke
		}
		avg := sum / float64(len(enemyKEs))
		enemyAvgKE = &avg
	}
	offConv, defRes := ComputeCombatYield(m)
	_, breakdown := computeCompositeScoreWithBreakdown(&compositeMatchRow{
		Kills:               m.Kills,
		Deaths:              m.Deaths,
		Assists:             m.Assists,
		KillsExpected:       m.KillsExpected,
		DeathsExpected:      m.DeathsExpected,
		Outcome:             m.Outcome,
		DamageDealt:         m.DamageDealt,
		DamageTaken:         m.DamageTaken,
		Accuracy:            m.Accuracy,
		MedalExploitScore:   medalScore,
		OffensiveConversion: offConv,
		DefensiveResistance: defRes,
	}, rollingAvgPtr(h.accuracy), enemyAvgKE, rollingAvgPtr(h.damageEff),
		rollingAvgPtr(h.medalExploit), rollingAvgPtr(h.offConv), rollingAvgPtr(h.defRes))

	appendToHistory(&h.accuracy, m.Accuracy)
	if total := m.DamageDealt + m.DamageTaken; total > 0 {
		appendToHistory(&h.damageEff, ClampF(m.DamageDealt/total, 0, 1))
	}
	if medalScore > 0 {
		appendToHistory(&h.medalExploit, medalScore)
	}
	if offConv > 0 {
		appendToHistory(&h.offConv, offConv)
	}
	if defRes > 0 {
		appendToHistory(&h.defRes, defRes)
	}
	return breakdown
}

// componentRows met le breakdown en lignes lusr_component_history, triées par nom (ordre
// d'écriture déterministe), avec le poids de CompositeWeights.
func componentRows(matchID string, breakdown map[string]float64) []persist.LUSRComponentInsert {
	if len(breakdown) == 0 {
		return nil
	}
	names := make([]string, 0, len(breakdown))
	for name := range breakdown {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]persist.LUSRComponentInsert, 0, len(names))
	for _, name := range names {
		rows = append(rows, persist.LUSRComponentInsert{
			MatchID:       matchID,
			ComponentName: name,
			Value:         breakdown[name],
			Weight:        CompositeWeights[name],
		})
	}
	return rows
}

// writeMatchComponents calcule et écrit les composantes du match que la rafale vient de
// noter. Secondaire à la note : un échec est compté et journalisé, il ne retient ni la note
// ni le filigrane : les composantes sont secondaires à la note. Il se rattrape par un rejeu
// complet (RecomputeLUSRCanonicalForPlayer).
func (c shadowRunContext) writeMatchComponents(ctx context.Context, matchID, group string) {
	rows, err := c.components.rowsFor(ctx, c.sharedDB, c.xuid, matchID, group)
	if err == nil && len(rows) > 0 {
		err = persist.NewAppendOnlyLUSRPersister(c.playerDB).PersistComponents(ctx, rows)
	}
	if err != nil {
		componentWriteErrors.Add(1)
		slog.WarnContext(ctx, "LUSR v2: composantes du match non écrites — la note l'est, un rejeu complet les rétablit",
			"match_id", matchID, "group", group, "xuid", c.xuid, "err", err)
		return
	}
	if len(rows) > 0 {
		componentWritesTotal.Add(1)
	}
}
