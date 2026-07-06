package sync

import "strings"

// skill_merge.go — pont entre les DTOs skill (haloclient) et les ParticipantRow
// de sync. Reste dans sync (utilise ParticipantRow) alors que la récupération +
// le parsing skill vivent dans le sous-package haloclient (extraction K3e).
// MatchSkillData est ré-exporté depuis haloclient (cf. haloclient_reexport.go).

// MergeSkillIntoParticipants reporte les données skill (MMR, expected/stddev) sur
// les ParticipantRow correspondants (par XUID). No-op si skill est vide.
func MergeSkillIntoParticipants(
	participants []ParticipantRow,
	skill map[string]*MatchSkillData,
) []ParticipantRow {
	if len(skill) == 0 {
		return participants
	}
	for i := range participants {
		sd, ok := skill[participants[i].XUID]
		if !ok || sd == nil {
			continue
		}
		if sd.TeamMMR != nil {
			participants[i].TeamMMR = sd.TeamMMR
		}
		if sd.EnemyMMR != nil {
			participants[i].EnemyMMR = sd.EnemyMMR
		}
		if sd.KillsExpected != nil {
			participants[i].KillsExpected = sd.KillsExpected
		}
		if sd.KillsStdDev != nil {
			participants[i].KillsStddev = sd.KillsStdDev
		}
		if sd.DeathsExpected != nil {
			participants[i].DeathsExpected = sd.DeathsExpected
		}
		if sd.DeathsStdDev != nil {
			participants[i].DeathsStddev = sd.DeathsStdDev
		}
	}
	return participants
}

// ParticipantXUIDs extrait la liste des XUIDs (humains uniquement) d'une slice
// de ParticipantRow — utilisée pour passer les bons XUIDs à GetMatchSkill.
func ParticipantXUIDs(rows []ParticipantRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.XUID == "" || strings.HasPrefix(r.XUID, "bid(") {
			continue
		}
		out = append(out, r.XUID)
	}
	return out
}
