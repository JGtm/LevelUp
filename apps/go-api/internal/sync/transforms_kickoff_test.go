package sync

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/kickoff"
)

// payloadCoupDEnvoi : un match de 600 s à début fixe, avec deux joueurs présents au début (arrivés
// à +28 s et +28,2 s), un bot présent au début arrivé tard (exclu de l'estimation), un retardataire
// et un joueur sans ParticipationInfo (présence inconnue).
func payloadCoupDEnvoi(start time.Time) map[string]any {
	pi := func(arrivee time.Duration, present bool, depart *time.Duration, joue string) map[string]any {
		m := map[string]any{
			"PresentAtBeginning": present,
			"FirstJoinedTime":    start.Add(arrivee).Format(time.RFC3339Nano),
			"TimePlayed":         joue,
		}
		if depart != nil {
			m["LastLeaveTime"] = start.Add(*depart).Format(time.RFC3339Nano)
		}
		return m
	}
	parti := 300 * time.Second
	return map[string]any{
		"MatchId": "m-coup-d-envoi",
		"MatchInfo": map[string]any{
			"StartTime": start.Format(time.RFC3339),
			"Duration":  "PT10M",
		},
		"Players": []any{
			map[string]any{"PlayerId": "xuid(2603225103667623)", "ParticipationInfo": pi(28*time.Second, true, nil, "PT10M")},
			map[string]any{"PlayerId": "xuid(2547408409520575)", "ParticipationInfo": pi(28200*time.Millisecond, true, &parti, "PT5M")},
			map[string]any{"PlayerId": "bid(1.0)", "ParticipationInfo": pi(90*time.Second, true, nil, "PT10M")},
			map[string]any{"PlayerId": "xuid(2664271391936720)", "ParticipationInfo": pi(200*time.Second, false, nil, "PT6M40S")},
			map[string]any{"PlayerId": "xuid(2535000000000001)"},
		},
	}
}

// Le coup d'envoi se résout sur les lignes extraites (bots et retardataires exclus de l'estimation),
// se range avec sa source, et le temps de jeu de chaque joueur se calcule dessus ; la présence
// inconnue garde le temps de jeu de l'API.
func TestApplyKickoffToRows_EstimationEtTempsDeJeu(t *testing.T) {
	start := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	j := payloadCoupDEnvoi(start)
	reg, err := ExtractRegistry(j, "")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	if reg.RealStartTime != nil || reg.RealStartSource != "" {
		t.Fatalf("ExtractRegistry ne decide plus le coup d'envoi : (%v, %q)", reg.RealStartTime, reg.RealStartSource)
	}
	parts := ExtractParticipants(j)
	k := kickoff.ApplyToRows(reg, parts)
	if k.Source != timeline.KickoffSourceAPI || k.Estimate != timeline.T0QualityOK {
		t.Fatalf("coup d'envoi = %+v, attendu source api, estimation ok", k)
	}
	if reg.RealStartTime == nil || !reg.RealStartTime.Equal(start.Add(28*time.Second)) || reg.RealStartSource != "api" {
		t.Fatalf("registre = (%v, %q)", reg.RealStartTime, reg.RealStartSource)
	}
	attendu := map[string]*int{
		"2603225103667623": intPtr(572), // 600 - 28
		"2547408409520575": intPtr(271), // 300 - 28,2, tronqué
		"bid(1.0)":         intPtr(510), // 600 - 90
		"2664271391936720": intPtr(400), // 600 - 200
		"2535000000000001": nil,         // présence inconnue, aucune valeur de l'API
	}
	for _, p := range parts {
		want := attendu[p.XUID]
		switch {
		case want == nil && p.TimePlayedSeconds != nil:
			t.Errorf("%s : temps de jeu %d, attendu absent", p.XUID, *p.TimePlayedSeconds)
		case want != nil && (p.TimePlayedSeconds == nil || *p.TimePlayedSeconds != *want):
			t.Errorf("%s : temps de jeu %v, attendu %d", p.XUID, p.TimePlayedSeconds, *want)
		}
	}
}

// Sans arrivée exploitable : coup d'envoi inconnu, et le temps de jeu de l'API reste.
func TestApplyKickoffToRows_InconnuGardeLAPI(t *testing.T) {
	start := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	j := map[string]any{
		"MatchId":   "m-sans-arrivee",
		"MatchInfo": map[string]any{"StartTime": start.Format(time.RFC3339), "Duration": "PT10M"},
		"Players": []any{map[string]any{
			"PlayerId": "xuid(2603225103667623)", "ParticipationInfo": map[string]any{"TimePlayed": "PT9M"},
		}},
	}
	reg, err := ExtractRegistry(j, "")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	parts := ExtractParticipants(j)
	if k := kickoff.ApplyToRows(reg, parts); k.Known() || k.Estimate != timeline.T0QualityNoData {
		t.Fatalf("coup d'envoi = %+v, attendu inconnu (no_data)", k)
	}
	if reg.RealStartTime != nil || reg.RealStartSource != "" {
		t.Fatalf("registre = (%v, %q), attendu inconnu", reg.RealStartTime, reg.RealStartSource)
	}
	if parts[0].TimePlayedSeconds == nil || *parts[0].TimePlayedSeconds != 540 {
		t.Fatalf("temps de jeu = %v, attendu 540 (API)", parts[0].TimePlayedSeconds)
	}
}

// TEMOIN AVANT / APRES : le calcul d'avant (estimation sur le payload, fenêtre recalculée à part,
// temps de jeu réécrit seulement en qualité ok) est rejoué ici tel qu'il était, et le chemin unique
// doit rendre les mêmes valeurs sur le même payload.
func TestApplyKickoffToRows_TemoinDuCalculDAvant(t *testing.T) {
	start := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	j := payloadCoupDEnvoi(start)
	reg, _ := ExtractRegistry(j, "")
	parts := ExtractParticipants(j)
	kickoff.ApplyToRows(reg, parts)

	avantT0, avantQ := ancienT0(j, start)
	if !avantQ.Computed() || !reg.RealStartTime.Equal(start.Add(time.Duration(avantT0)*time.Millisecond)) {
		t.Fatalf("coup d'envoi : avant %d ms (%s), apres %v", avantT0, avantQ, reg.RealStartTime)
	}
	tl := domain.NewMatchTimelineAt(start, 600_000, avantT0)
	for _, p := range parts {
		if p.FirstJoinedTime == nil {
			continue
		}
		secs, q := timeline.ComputeTimePlayedFor(timeline.TimePlayedInput{
			FirstJoinedTime: p.FirstJoinedTime.UTC(), LastLeaveTime: p.LastLeaveTime}, tl)
		if q == timeline.TimePlayedOK && (p.TimePlayedSeconds == nil || int64(*p.TimePlayedSeconds) != secs) {
			t.Errorf("%s : avant %d s, apres %v", p.XUID, secs, p.TimePlayedSeconds)
		}
	}
}

// ancienT0 rejoue l'estimation d'avant, directement sur le payload.
func ancienT0(j map[string]any, start time.Time) (int64, timeline.T0Quality) {
	var in []timeline.ParticipationT0Input
	for _, p := range j["Players"].([]any) {
		player := p.(map[string]any)
		pinfo, ok := player["ParticipationInfo"].(map[string]any)
		if !ok {
			continue
		}
		fjt, err := parseISO(asString(pinfo["FirstJoinedTime"]))
		if err != nil {
			continue
		}
		present, _ := pinfo["PresentAtBeginning"].(bool)
		in = append(in, timeline.ParticipationT0Input{
			FirstJoinedTime: fjt, PresentAtBeginning: present,
			IsBot: !timeline.HumanXUID(extractXUID(asString(player[jsonKeyPlayerID]))),
		})
	}
	return timeline.ComputeT0(in, start)
}

func intPtr(v int) *int { return &v }
