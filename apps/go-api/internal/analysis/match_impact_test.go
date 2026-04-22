package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

func TestComputeSingleMatchImpact_Tourist(t *testing.T) {
	input := analysis.MatchImpactInput{
		KillEvents: nil,
		MyXUID:     "xuid1",
		MyKills:    0,
	}
	badges := analysis.ComputeSingleMatchImpact(input)
	found := false
	for _, b := range badges {
		if b.BadgeKey == "tourist" {
			found = true
		}
	}
	if !found {
		t.Error("attendu badge 'tourist' quand MyKills=0")
	}
}

func TestComputeSingleMatchImpact_FirstBlood(t *testing.T) {
	input := analysis.MatchImpactInput{
		KillEvents: []analysis.ImpactEvent{
			{TimeMS: 5000, KillerXUID: "xuid1", VictimXUID: "xuid2"},
			{TimeMS: 10000, KillerXUID: "xuid3", VictimXUID: "xuid4"},
		},
		MyXUID:  "xuid1",
		MyKills: 1,
	}
	badges := analysis.ComputeSingleMatchImpact(input)
	found := false
	for _, b := range badges {
		if b.BadgeKey == "first_blood" {
			found = true
		}
	}
	if !found {
		t.Error("attendu badge 'first_blood' quand premier kill du match")
	}
}

func TestComputeSingleMatchImpact_NotFirstBlood(t *testing.T) {
	input := analysis.MatchImpactInput{
		KillEvents: []analysis.ImpactEvent{
			{TimeMS: 5000, KillerXUID: "xuid3", VictimXUID: "xuid2"},
			{TimeMS: 10000, KillerXUID: "xuid1", VictimXUID: "xuid4"},
		},
		MyXUID:  "xuid1",
		MyKills: 1,
	}
	badges := analysis.ComputeSingleMatchImpact(input)
	for _, b := range badges {
		if b.BadgeKey == "first_blood" {
			t.Error("badge 'first_blood' ne doit pas être attribué")
		}
	}
}

func TestComputeSingleMatchImpact_NoTouristWithKills(t *testing.T) {
	input := analysis.MatchImpactInput{
		KillEvents: []analysis.ImpactEvent{
			{TimeMS: 5000, KillerXUID: "xuid1", VictimXUID: "xuid2"},
		},
		MyXUID:  "xuid1",
		MyKills: 1,
	}
	badges := analysis.ComputeSingleMatchImpact(input)
	for _, b := range badges {
		if b.BadgeKey == "tourist" {
			t.Error("badge 'tourist' ne doit pas être attribué quand MyKills>0")
		}
	}
}
