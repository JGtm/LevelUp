package killcollector

// phantoms_repair.go — LES COUPLES `killer_victim_pairs` SELON L APPARIEMENT COURANT, apres la purge
// des temps forts fantomes (`migration.PurgeHighlightPhantoms`) ou un changement de l appariement.
//
// Les couples sont recalcules depuis `highlight_events` par `analysis.ComputeKillerVictimPairs` :
// retirer les couples d un fantome ne suffit pas, il faut re-apparier. Ce fichier ne fait que
// CALCULER, en lecture seule ([StalePairs]) : l ecriture est un echange de table
// (`migration.ReplaceKillerVictimPairs`, jamais de DELETE) que la commande joue ensuite.
//
// LA SELECTION EST LA DIFFERENCE, PAS LA LISTE DES MATCHS PURGES : un match est recalcule quand ses
// couples stockes different du calcul. Une reparation interrompue entre la purge et le recalcul
// se reprend donc d elle-meme — les matchs purges gardent des couples qui ne sont plus ceux du
// calcul — et un second passage ne trouve plus rien.
//
// LES NOMS : l annuaire de lecture (`v_gamertag_lookup`, ADR 0035), puis le nom que les couples
// existants du match portent ; a defaut, aucun nom (NULL). Jamais le xuid a la place d un nom.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/persist"
)

// StalePairsReport : les matchs dont les couples stockes different de l appariement courant, et
// leurs couples recalcules.
type StalePairsReport struct {
	// Examined : matchs qui portent au moins un kill ou une mort dans `highlight_events`.
	Examined int
	// MatchIDs : ceux dont les couples stockes different du calcul, dans l ordre des identifiants.
	MatchIDs []string
	// Pairs : les couples recalcules de chacun de `MatchIDs` (liste vide : aucun couple).
	Pairs map[string][]persist.KVPairCompletion
}

// StalePairs confronte les couples stockes de CHAQUE match a l appariement courant. Lecture seule.
func StalePairs(ctx context.Context, db *sql.DB) (StalePairsReport, error) {
	rep := StalePairsReport{Pairs: map[string][]persist.KVPairCompletion{}}
	ids, err := matchsAvecKillsOuMorts(ctx, db)
	if err != nil {
		return rep, err
	}
	annuaire, err := (&chargeurDAnnuaire{}).annuaireDe(ctx, db, "reparation des couples")
	if err != nil {
		return rep, err
	}
	for _, id := range ids {
		rep.Examined++
		pairs, err := couplesRecalcules(ctx, db, id, annuaire)
		if err != nil {
			return rep, err
		}
		stockes, err := couplesStockes(ctx, db, id)
		if err != nil {
			return rep, err
		}
		if memesCouples(stockes, pairs) {
			continue
		}
		rep.MatchIDs = append(rep.MatchIDs, id)
		rep.Pairs[id] = pairs
	}
	return rep, nil
}

// couplesRecalcules : les couples d un match selon l appariement courant. Un gamertag que ni
// l annuaire ni les couples existants ne donnent reste vide (l appariement, lui, y met le xuid).
func couplesRecalcules(ctx context.Context, db *sql.DB, matchID string, annuaire *annuaireDesNoms) (
	[]persist.KVPairCompletion, error,
) {
	raw, err := evenementsARapparier(ctx, db, matchID, annuaire)
	if err != nil {
		return nil, err
	}
	pairs := analysis.ComputeKillerVictimPairs(raw, analysis.KillDeathPairToleranceMS)
	rows := make([]persist.KVPairCompletion, 0, len(pairs))
	for _, p := range pairs {
		rows = append(rows, persist.KVPairCompletion{
			KillerXUID: p.KillerXUID, KillerGamertag: nomSaufXUID(p.KillerGT, p.KillerXUID),
			VictimXUID: p.VictimXUID, VictimGamertag: nomSaufXUID(p.VictimGT, p.VictimXUID), TimeMS: p.TimeMS,
		})
	}
	return rows, nil
}

// nomSaufXUID : le nom, ou vide quand il n est que le xuid.
func nomSaufXUID(nom, xuid string) string {
	if nom == xuid {
		return ""
	}
	return nom
}

// evenementsARapparier : kills et morts du match (doublons exacts ecartes), nommes.
func evenementsARapparier(ctx context.Context, db *sql.DB, matchID string, annuaire *annuaireDesNoms) (
	[]analysis.RawEvent, error,
) {
	noms, err := nomsDesCouplesExistants(ctx, db, matchID)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT xuid, LOWER(event_type), COALESCE(time_ms, 0) FROM highlight_events
		WHERE match_id = ? AND LOWER(event_type) IN (?, ?) AND xuid IS NOT NULL AND xuid <> ''
		ORDER BY 3, 1, 2`, matchID, highlightevent.EventTypeKill, highlightevent.EventTypeDeath)
	if err != nil {
		return nil, fmt.Errorf("couples: evenements %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	var out []analysis.RawEvent
	for rows.Next() {
		var ev analysis.RawEvent
		if err := rows.Scan(&ev.XUID, &ev.EventType, &ev.TimeMS); err != nil {
			return nil, fmt.Errorf("couples: scan %s: %w", matchID, err)
		}
		ev.Gamertag = annuaire.nom(ev.XUID)
		if ev.Gamertag == "" {
			ev.Gamertag = noms[ev.XUID]
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("couples: lignes %s: %w", matchID, err)
	}
	return out, nil
}

// nomsDesCouplesExistants : xuid -> nom porte par les couples stockes du match, hors noms qui ne
// sont que le xuid.
func nomsDesCouplesExistants(ctx context.Context, db *sql.DB, matchID string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT killer_xuid, MAX(killer_gamertag) FROM killer_victim_pairs
		WHERE match_id = ? AND killer_gamertag IS NOT NULL AND killer_gamertag <> killer_xuid GROUP BY 1
		UNION
		SELECT victim_xuid, MAX(victim_gamertag) FROM killer_victim_pairs
		WHERE match_id = ? AND victim_gamertag IS NOT NULL AND victim_gamertag <> victim_xuid GROUP BY 1
		ORDER BY 1, 2`,
		matchID, matchID)
	if err != nil {
		return nil, fmt.Errorf("couples: noms %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	noms := map[string]string{}
	for rows.Next() {
		var x, g string
		if err := rows.Scan(&x, &g); err != nil {
			return nil, fmt.Errorf("couples: noms scan %s: %w", matchID, err)
		}
		if _, ok := noms[x]; !ok {
			noms[x] = g
		}
	}
	return noms, rows.Err()
}
