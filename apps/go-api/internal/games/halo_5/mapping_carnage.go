package halo_5

// mapping_carnage.go — projection PURE du carnage report Halo 5 (roster complet)
// vers []domain.MatchParticipantRow (shared.match_participants).
//
// Vit dans le package halo_5 (et pas dans ingest) : ingest ne peut pas importer
// halo_5 (cycle), et le mapper réutilise les parseurs de durée privés + produit
// directement le type domain consommé par ingest.CollectMatchBatch.AddParticipants.
//
// ⚠ Accuracy / DamageTaken NE SONT PAS fournis par l'API h5 (carnage = K/D/A bruts
// + dégâts INFLIGÉS) → laissés nil, jamais fabriqués.
//
// KDA — EXCEPTION DOCUMENTÉE : Halo 5 est le SEUL titre où le KDA est CALCULÉ À
// L'INGESTION (l'API h5 ne le fournit pas), puis STOCKÉ dans match_participants.kda
// pour alimenter les BDD — JAMAIS recalculé en lecture. La forme native h5 est le
// FDA NET (k + a/3) − d (par match), distinct du quotient KDA d'Infinite, et peut
// être NÉGATIF. Infinite garde son KDA d'API (jamais calculé).

import "levelup/go-api/internal/domain"

// h5GameModeSegment mappe le GameMode numérique (liste de matchs) vers le segment
// d'URL string du carnage (/h5/{mode}/matches/{id}). Fallback "arena".
func h5GameModeSegment(gameMode int) string {
	switch gameMode {
	case 2:
		return "campaign"
	case 3:
		return "custom"
	case 4:
		return "warzone"
	default: // 1 = arena (le plus courant)
		return "arena"
	}
}

// mapCarnageParticipants projette PlayerStats → []domain.MatchParticipantRow.
// resolveXUID(gamertag) → xuid Xbox résolu. RESOLVE-OR-SKIP : un joueur dont l'xuid
// ne résout pas ("") est SAUTÉ — la PK match_participants (match_id, xuid) ferait
// collisionner ≥2 joueurs non résolus sur xuid="" (fusion/violation). Le kill-feed
// (killer_victim_pairs, sans PK) garde lui le gamertag. Best-effort : joueur rare
// absent du roster d'un match (le viewer, seedé, résout toujours).
// Outcome dérivé du rang d'ÉQUIPE (jeu d'équipe) ou individuel (FFA).
func mapCarnageParticipants(matchID string, carnage *H5CarnageResponse, resolveXUID func(gamertag string) string) []domain.MatchParticipantRow {
	if carnage == nil || len(carnage.PlayerStats) == 0 {
		return nil
	}
	winTeam := winningTeamID(carnage)
	out := make([]domain.MatchParticipantRow, 0, len(carnage.PlayerStats))
	for i := range carnage.PlayerStats {
		p := &carnage.PlayerStats[i]
		xuid := resolveXUID(p.Player.Gamertag)
		if xuid == "" {
			continue // resolve-or-skip (cf. godoc : PK xuid="" collisionnerait)
		}
		out = append(out, domain.MatchParticipantRow{
			MatchID:           matchID,
			XUID:              xuid,
			Gamertag:          strPtrH5(p.Player.Gamertag),
			TeamID:            intPtrH5(p.TeamId),
			Rank:              intPtrH5(p.Rank),
			Score:             intPtrH5(p.PlayerScore),
			PersonalScore:     intPtrH5(p.PlayerScore),
			Kills:             intPtrH5(p.TotalKills),
			Deaths:            intPtrH5(p.TotalDeaths),
			Assists:           intPtrH5(p.TotalAssists),
			HeadshotKills:     intPtrH5(p.TotalHeadshots),
			ShotsFired:        intPtrH5(p.TotalShotsFired),
			ShotsHit:          intPtrH5(p.TotalShotsLanded),
			DamageDealt:       floatPtrH5(p.TotalWeaponDamage),
			MeleeKills:        intPtrH5(p.TotalMeleeKills),
			GrenadeKills:      intPtrH5(p.TotalGrenadeKills),
			PowerWeaponKills:  intPtrH5(p.TotalPowerWeaponKills),
			TimePlayedSeconds: parseISO8601DurationSeconds(p.TotalTimePlayed),
			AvgLifeSeconds:    iso8601DurationSecondsFloat(p.AvgLifeTimeOfPlayer),
			Outcome:           participantOutcome(carnage.IsTeamGame, p.TeamId, p.Rank, winTeam, p.DNF),
			// KDA : FDA NET h5 calculé À L'INGESTION (cf. godoc) → stocké, lu tel quel.
			KDA: h5NetFDA(p.TotalKills, p.TotalAssists, p.TotalDeaths),
			// Accuracy / DamageTaken : non fournis par l'API h5 → nil (jamais fabriqués).
		})
	}
	return out
}

// winningTeamID retourne le TeamId de l'équipe au Rank 1, ou -1 si indéterminé.
func winningTeamID(carnage *H5CarnageResponse) int {
	for i := range carnage.TeamStats {
		if carnage.TeamStats[i].Rank == 1 {
			return carnage.TeamStats[i].TeamId
		}
	}
	return -1
}

// participantOutcome dérive le code d'issue domain d'un joueur. DNF prioritaire ;
// jeu d'équipe → appartenance à l'équipe gagnante ; FFA → Rank individuel (1 =
// vainqueur). nil si indéterminé (équipe gagnante introuvable).
func participantOutcome(isTeamGame bool, teamID, rank, winTeam int, dnf bool) *int {
	var code int
	switch {
	case dnf:
		code = domain.OutcomeDNF
	case isTeamGame:
		if winTeam < 0 {
			return nil
		}
		if teamID == winTeam {
			code = domain.OutcomeWin
		} else {
			code = domain.OutcomeLoss
		}
	default: // FFA
		if rank == 1 {
			code = domain.OutcomeWin
		} else {
			code = domain.OutcomeLoss
		}
	}
	return &code
}

// iso8601DurationSecondsFloat parse "PT16.48S" → 16.48 (secondes flottantes). nil si KO.
func iso8601DurationSecondsFloat(s string) *float64 {
	ms, ok := parseISO8601DurationMs(s)
	if !ok {
		return nil
	}
	f := float64(ms) / 1000.0
	return &f
}

// h5NetFDA calcule le FDA NET Halo 5 d'un match : (kills + assists/3) − deaths.
// Forme native h5 (l'API ne renvoie pas de KDA) ; peut être négatif. Calculé À
// L'INGESTION et stocké dans match_participants.kda — jamais recalculé en lecture.
func h5NetFDA(kills, assists, deaths int) *float64 {
	v := float64(kills) + float64(assists)/3.0 - float64(deaths)
	return &v
}

func strPtrH5(s string) *string     { return &s }
func intPtrH5(n int) *int           { return &n }
func floatPtrH5(f float64) *float64 { return &f }
