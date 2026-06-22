package halo_5

// mapping_carnage_detail.go — projection PURE du carnage report Halo 5 (roster
// complet) + de l'entrée d'historique correspondante vers canonical.MatchDetail.
//
// Parallèle à mapping_carnage.go (qui produit []domain.MatchParticipantRow pour
// l'ingestion DuckDB). Ici on produit le type CANONIQUE consommé par le service
// Match View (Part B). Mappers stateless, zéro IO — testables sans réseau.
//
// ⚠ Mêmes non-fabrications que mapping_carnage : Accuracy / DamageTaken NE SONT PAS
// fournis par l'API h5 → laissés nil, jamais reconstruits. Le KDA suit la règle h5
// (FDA NET (k+a/3)−d, forme native, calculé ici — h5 est le seul titre où le KDA
// est calculé hors API).

import (
	"strings"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// mapCarnageToCanonicalDetail assemble un canonical.MatchDetail à partir de :
//   - matchID : l'identifiant du match ;
//   - header  : l'entrée d'historique du joueur (refs map/playlist/variant, durée,
//     dates, isRanked, matchType, teams). Peut être nil → le détail reste pauvre
//     en header (refs nil) mais le roster carnage est exploitable.
//   - carnage : le roster complet (participants + équipes + flag jeu d'équipe).
//
// Retourne nil si carnage est nil/vide (le caller dégrade en
// ErrCapabilityNotSupported). Les refs header réutilisent EXACTEMENT le mapper
// summary (mapOneMatchSummary) pour garantir la parité de projection avec
// l'historique canonique.
func mapCarnageToCanonicalDetail(matchID, viewerGamertag string, header *canonical.MatchSummary, carnage *H5CarnageResponse) *canonical.MatchDetail {
	if carnage == nil || len(carnage.PlayerStats) == 0 {
		return nil
	}
	detail := &canonical.MatchDetail{
		MatchID:      matchID,
		Participants: mapCarnageToCanonicalParticipants(carnage),
		Teams:        mapCarnageTeams(carnage),
		Skill:        nil, // CSR pré/post par match = phase ultérieure (cf. plan PART A)
		Limitations:  h5MatchDetailLimitations(),
		// Commendations NATIVES du viewer sur CE match (AXE B). Affichage natif tel
		// quel — pas de reconstruction par tier/composite (décision produit).
		Commendations: mapViewerCommendations(viewerGamertag, carnage),
	}
	if header != nil {
		detail.StartedAtUTC = header.StartedAtUTC
		detail.EndedAtUTC = endedAtFromSummary(header)
		detail.Map = header.Map
		detail.Playlist = header.Playlist
		detail.GameVariant = header.GameVariant
		detail.IsRanked = header.IsRanked
		detail.IsPvE = header.IsPvE
		detail.MatchType = header.MatchType
	} else {
		detail.MatchType = canonical.MatchTypeUnknownMT
	}
	return detail
}

// mapCarnageToCanonicalParticipants projette PlayerStats → []canonical.MatchParticipant.
// Identité GAMERTAG-keyée (h5Identity) : XUID vide, jamais fabriqué (Player.Xuid null).
// Contrairement au mapper d'ingestion (mapCarnageParticipants), on NE saute PAS les
// joueurs : la PK match_participants (xuid="") ne s'applique pas au payload canonique
// in-memory, et le scoreboard Match View doit montrer TOUT le roster.
// Outcome dérivé du rang d'ÉQUIPE (jeu d'équipe) ou du rang individuel (FFA), via la
// même logique data-grounded que mapping.go (deriveOutcome).
func mapCarnageToCanonicalParticipants(carnage *H5CarnageResponse) []canonical.MatchParticipant {
	if carnage == nil || len(carnage.PlayerStats) == 0 {
		return nil
	}
	out := make([]canonical.MatchParticipant, 0, len(carnage.PlayerStats))
	for i := range carnage.PlayerStats {
		p := &carnage.PlayerStats[i]
		out = append(out, canonical.MatchParticipant{
			Identity:    h5Identity(p.Player.Gamertag),
			TeamID:      intPtrH5(p.TeamId),
			RankInMatch: intPtrH5(p.Rank),
			Outcome:     carnageParticipantOutcome(carnage, p),
			Score:       intPtrH5(p.PlayerScore),
			// PersonalScore : même source que Score côté h5 (PlayerScore), comme
			// l'ingestion DuckDB (mapping_carnage : Score == PersonalScore).
			PersonalScore: intPtrH5(p.PlayerScore),
			Kills:         intPtrH5(p.TotalKills),
			Deaths:        intPtrH5(p.TotalDeaths),
			Assists:       intPtrH5(p.TotalAssists),
			HeadshotKills: intPtrH5(p.TotalHeadshots),
			// DamageDealt : TotalWeaponDamage est un float côté API ; le canonique
			// l'expose en *int (arrondi) — dégâts INFLIGÉS uniquement.
			DamageDealt:      damageDealtIntPtr(p.TotalWeaponDamage),
			ShotsFired:       intPtrH5(p.TotalShotsFired),
			ShotsHit:         intPtrH5(p.TotalShotsLanded),
			KDA:              h5NetFDA(p.TotalKills, p.TotalAssists, p.TotalDeaths),
			TimePlayed:       parseISO8601DurationSeconds(p.TotalTimePlayed),
			MeleeKills:       intPtrH5(p.TotalMeleeKills),
			GrenadeKills:     intPtrH5(p.TotalGrenadeKills),
			PowerWeaponKills: intPtrH5(p.TotalPowerWeaponKills),
			AvgLifeSeconds:   iso8601DurationSecondsFloat(p.AvgLifeTimeOfPlayer),
			// Accuracy / DamageTaken : NON fournis par l'API h5 → nil (jamais fabriqués).
		})
	}
	return out
}

// carnageParticipantOutcome dérive l'Outcome canonique d'un joueur du carnage en
// réutilisant deriveOutcome (mapping.go) : rang d'ÉQUIPE prioritaire en jeu
// d'équipe (un Rank individuel 1 dans une équipe perdante ne doit pas devenir un
// faux Win), rang individuel en FFA. DNF non distingué ici : le carnage porte le
// flag DNF mais deriveOutcome historique ne le prend pas (parité avec mapping.go) ;
// un quitter ressort en Loss/Tie (dégradation documentée, cohérente avec l'UI).
func carnageParticipantOutcome(carnage *H5CarnageResponse, p *H5CarnagePlayer) canonical.Outcome {
	if p.DNF {
		return canonical.OutcomeDNF
	}
	var teamRank *int
	if carnage.IsTeamGame {
		teamRank = carnageTeamRank(carnage, p.TeamId)
	}
	return deriveOutcome(p.Rank, teamRank, carnage.IsTeamGame)
}

// carnageTeamRank retourne le Rank de l'équipe teamID dans le carnage, ou nil si
// absente (→ deriveOutcome renvoie OutcomeTie, indéterminé documenté).
func carnageTeamRank(carnage *H5CarnageResponse, teamID int) *int {
	for i := range carnage.TeamStats {
		if carnage.TeamStats[i].TeamId == teamID {
			rank := carnage.TeamStats[i].Rank
			return &rank
		}
	}
	return nil
}

// mapCarnageTeams projette TeamStats → []canonical.TeamSnapshot. Les XUIDs des
// participants restent nil (Halo 5 n'a aucun xuid). nil si aucune équipe.
func mapCarnageTeams(carnage *H5CarnageResponse) []canonical.TeamSnapshot {
	if carnage == nil || len(carnage.TeamStats) == 0 {
		return nil
	}
	out := make([]canonical.TeamSnapshot, 0, len(carnage.TeamStats))
	for i := range carnage.TeamStats {
		score := carnage.TeamStats[i].Score
		out = append(out, canonical.TeamSnapshot{
			TeamID: carnage.TeamStats[i].TeamId,
			Score:  &score,
			// MMR + ParticipantsXUIDs : indisponibles en h5 (pas de MMR, xuid null).
		})
	}
	return out
}

// endedAtFromSummary retourne la fin du match (StartedAtUTC + durée) si la durée
// est connue. Halo 5 ne donne nativement que la fin (MatchCompletedDate) ; le
// summary l'a déjà soustraite pour obtenir StartedAtUTC, on reconstitue donc la
// fin. Nil si durée ou début indisponible (best-effort, cf. cible PART A).
func endedAtFromSummary(s *canonical.MatchSummary) *time.Time {
	if s == nil || s.DurationSeconds == nil || s.StartedAtUTC.IsZero() {
		return nil
	}
	end := s.StartedAtUTC.Add(time.Duration(*s.DurationSeconds) * time.Second)
	return &end
}

// damageDealtIntPtr convertit les dégâts infligés (float API) en *int (arrondi
// vers le bas, comme un compte de points de dégâts). Valeur négative aberrante →
// nil (donnée corrompue, ne pas polluer).
func damageDealtIntPtr(damage float64) *int {
	if damage < 0 {
		return nil
	}
	v := int(damage)
	return &v
}

// h5MatchDetailLimitations déclare les gaps de capability connus d'un MatchDetail
// h5 servi depuis la carnage live (vs un détail Infinite enrichi). Reporté dans
// MatchDetail.Limitations pour que le consommateur cadre l'affichage (scoreboard
// sans précision/dégâts subis, pas de snapshot skill par match).
func h5MatchDetailLimitations() []canonical.CapabilityGap {
	return []canonical.CapabilityGap{
		{
			CapabilityKey: "match.participant.accuracy",
			ReasonCode:    "accuracy_not_provided",
			Severity:      "info",
			Message:       "précision (accuracy) non fournie par l'API Halo 5 (seuls les tirs tirés/touchés bruts sont disponibles)",
		},
		{
			CapabilityKey: "match.participant.damage_taken",
			ReasonCode:    "damage_taken_not_provided",
			Severity:      "info",
			Message:       "dégâts subis non fournis par l'API Halo 5 (seuls les dégâts infligés sont disponibles)",
		},
		{
			CapabilityKey: "match.skill.snapshot",
			ReasonCode:    "per_match_csr_not_loaded",
			Severity:      "info",
			Message:       "snapshot de skill par match (CSR pré/post) non chargé pour Halo 5 (phase ultérieure)",
		},
	}
}

// h5HeaderMatchID compare un MatchID au champ historique (insensible à la casse,
// les IDs h5 étant des GUID). Utilisé pour retrouver l'entrée d'historique
// correspondant au matchID demandé.
func h5HeaderMatchID(resultMatchID, wanted string) bool {
	return strings.EqualFold(strings.TrimSpace(resultMatchID), strings.TrimSpace(wanted))
}

// mapViewerCommendations projette les commendations natives progressées par le
// VIEWER sur ce match (carnage PlayerStats[viewer].ProgressiveCommendationDeltas +
// MetaCommendationDeltas) vers []canonical.Commendation. Count = Progress −
// PreviousProgress (> 0 seulement). Affichage NATIF tel quel — Name/IconURL vides
// (les définitions natives sont une suite, cf. AXE B Phase 1 : la donnée brute, ID
// + count, suffit). viewerGamertag vide ou absent du roster → nil (dégradation
// gracieuse). Ordre du payload préservé (déterminisme).
func mapViewerCommendations(viewerGamertag string, carnage *H5CarnageResponse) []canonical.Commendation {
	if carnage == nil || strings.TrimSpace(viewerGamertag) == "" {
		return nil
	}
	var p *H5CarnagePlayer
	for i := range carnage.PlayerStats {
		if strings.EqualFold(strings.TrimSpace(carnage.PlayerStats[i].Player.Gamertag), strings.TrimSpace(viewerGamertag)) {
			p = &carnage.PlayerStats[i]
			break
		}
	}
	if p == nil {
		return nil // viewer absent du roster carnage
	}
	var out []canonical.Commendation
	out = appendCanonicalCommendations(out, p.ProgressiveCommendationDeltas)
	out = appendCanonicalCommendations(out, p.MetaCommendationDeltas)
	return out
}

// appendCanonicalCommendations convertit des deltas carnage en commendations
// canoniques (Count = Progress − PreviousProgress > 0, Id non vide).
func appendCanonicalCommendations(out []canonical.Commendation, deltas []H5CommendationDelta) []canonical.Commendation {
	for i := range deltas {
		d := deltas[i]
		if d.Id == "" {
			continue
		}
		count := d.Progress - d.PreviousProgress
		if count <= 0 {
			continue
		}
		out = append(out, canonical.Commendation{ID: d.Id, Count: count})
	}
	return out
}
