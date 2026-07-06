package service

import (
	"strconv"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/openspartan/mapper"
	"levelup/go-api/internal/sync"
)

// toSyncRegistry converts a mapper.MatchRegistryRow (PR 2 pivot type) into
// the sync.MatchRegistryRow consumed by the shared persist path
// (persist.BatchBuilder.SetMatch in openspartan_import_service.go). Required
// because the sync types predate the mapper and use slightly different
// nullability conventions.
func toSyncRegistry(m mapper.MatchRegistryRow) sync.MatchRegistryRow {
	durSec := m.DurationSeconds
	playableSec := m.PlayableDurationSeconds

	var team0, team1 *int
	if m.Team0Score != nil {
		v := int(*m.Team0Score)
		team0 = &v
	}
	if m.Team1Score != nil {
		v := int(*m.Team1Score)
		team1 = &v
	}

	modeCategory := "Other"
	if m.ModeCategory != nil && *m.ModeCategory != "" {
		modeCategory = *m.ModeCategory
	}

	return sync.MatchRegistryRow{
		MatchID:                 m.MatchID,
		StartTime:               m.StartTime,
		EndTime:                 m.EndTime,
		PlaylistID:              m.PlaylistID,
		PlaylistName:            m.PlaylistName,
		PlaylistVersionID:       m.PlaylistVersionID,
		MapID:                   m.MapID,
		MapName:                 m.MapName,
		MapVersionID:            m.MapVersionID,
		PairID:                  m.PairID,
		PairName:                m.PairName,
		PairVersionID:           m.PairVersionID,
		GameVariantID:           m.GameVariantID,
		GameVariantName:         m.GameVariantName,
		GameVariantVersionID:    m.GameVariantVersionID,
		ModeCategory:            modeCategory,
		IsRanked:                m.IsRanked,
		IsFirefight:             m.IsFirefight,
		DurationSeconds:         &durSec,
		PlayableDurationSeconds: &playableSec,
		RealStartTime:           m.RealStartTime,
		Team0Score:              team0,
		Team1Score:              team1,
		Team0PSScore:            m.Team0PsScore,
		Team1PSScore:            m.Team1PsScore,
		FirstSyncBy:             m.FirstSyncBy,
		SeasonID:                m.SeasonID, // Phase 9.5 — propage le SeasonID OpenSpartan vers le sync writer
	}
}

func toSyncParticipants(parts []mapper.MatchParticipantRow) []sync.ParticipantRow {
	out := make([]sync.ParticipantRow, len(parts))
	for i, p := range parts {
		out[i] = toSyncParticipant(p)
	}
	return out
}

func toSyncParticipant(p mapper.MatchParticipantRow) sync.ParticipantRow {
	teamID := p.TeamID
	outcome := p.Outcome
	rank := int(p.Rank)
	score := p.Score
	kills := int(p.Kills)
	deaths := int(p.Deaths)
	assists := int(p.Assists)
	kda := p.KDA
	accuracy := p.Accuracy
	shotsF := p.ShotsFired
	shotsH := p.ShotsHit
	damageD := p.DamageDealt
	damageT := p.DamageTaken
	psScore := p.PersonalScore
	tp := p.TimePlayedSeconds
	head := int(p.HeadshotKills)
	grenade := int(p.GrenadeKills)
	melee := int(p.MeleeKills)
	power := int(p.PowerWeaponKills)
	beginning := p.PresentAtBeginning
	completion := p.PresentAtCompletion
	joined := p.JoinedInProgress
	left := p.LeftInProgress

	row := sync.ParticipantRow{
		MatchID:             p.MatchID,
		XUID:                p.XUID,
		Gamertag:            p.Gamertag,
		TeamID:              &teamID,
		Outcome:             &outcome,
		Rank:                &rank,
		Score:               &score,
		Kills:               &kills,
		Deaths:              &deaths,
		Assists:             &assists,
		KDA:                 &kda,
		Accuracy:            &accuracy,
		ShotsFired:          &shotsF,
		ShotsHit:            &shotsH,
		DamageDealt:         &damageD,
		DamageTaken:         &damageT,
		PersonalScore:       &psScore,
		TimePlayedSeconds:   &tp,
		AvgLifeSeconds:      p.AvgLifeSeconds,
		KillsExpected:       p.KillsExpected,
		DeathsExpected:      p.DeathsExpected,
		KillsStddev:         p.KillsStdDev,
		DeathsStddev:        p.DeathsStdDev,
		TeamMMR:             p.TeamMMR,
		EnemyMMR:            p.EnemyMMR,
		HeadshotKills:       &head,
		GrenadeKills:        &grenade,
		MeleeKills:          &melee,
		PowerWeaponKills:    &power,
		PresentAtBeginning:  &beginning,
		PresentAtCompletion: &completion,
		JoinedInProgress:    &joined,
		LeftInProgress:      &left,
	}
	if p.MaxKillingSpree != nil {
		v := int(*p.MaxKillingSpree)
		row.MaxKillingSpree = &v
	}
	return row
}

func toSyncMedals(medals []mapper.MedalEarnedRow) []sync.MedalRow {
	out := make([]sync.MedalRow, len(medals))
	for i, m := range medals {
		out[i] = sync.MedalRow{
			MatchID:     m.MatchID,
			XUID:        m.XUID,
			MedalNameID: m.MedalNameID,
			Count:       int(m.Count),
		}
	}
	return out
}

// toAnalysisEvent adapts a mapper.HighlightEventRow into the
// analysis.HighlightEvent shape that sync.InsertHighlightEvents expects.
// Missing fields default to zero — preserved further via RawJSON storage at
// the SQL layer.
func toAnalysisEvent(h mapper.HighlightEventRow) analysis.HighlightEvent {
	e := analysis.HighlightEvent{EventType: h.EventType}
	if h.XUID != nil {
		if u, err := strconv.ParseUint(*h.XUID, 10, 64); err == nil {
			e.XUID = u
		}
	}
	if h.TimeMs != nil {
		e.TimeMS = *h.TimeMs
	}
	if h.TypeHint != nil {
		e.TypeHint = *h.TypeHint
	}
	return e
}
