package prestige

import (
	"reflect"
	"testing"
)

// Scénarios produit (PLAN_COACH_V3_GENERATION § Identité d'escouade) :
// xuids symboliques A/B/C/D + R (random).
const (
	xA = "xuidA"
	xB = "xuidB"
	xC = "xuidC"
	xD = "xuidD"
	xR = "xuidRandom"
)

func TestMatchCountsForSquad_NoOverlapRule(t *testing.T) {
	trio := toXUIDSet([]string{xA, xB, xC})
	duo := toXUIDSet([]string{xA, xB})

	tests := []struct {
		name         string
		roster       map[string]struct{}
		otherKnown   map[string]struct{}
		participants []string
		want         bool
	}{
		{
			name:         "trio complet → compte",
			roster:       trio,
			participants: []string{xA, xB, xC},
			want:         true,
		},
		{
			name:         "trio complet + random → compte (random ignoré)",
			roster:       trio,
			participants: []string{xA, xB, xC, xR},
			want:         true,
		},
		{
			name:         "trio incomplet (C absent) → ne compte pas",
			roster:       trio,
			participants: []string{xA, xB, xR},
			want:         false,
		},
		{
			name:         "duo, match trio (C coéquipier connu présent) → ne compte PAS (no-overlap)",
			roster:       duo,
			otherKnown:   toXUIDSet([]string{xC}), // C ∈ autre escouade définie
			participants: []string{xA, xB, xC},
			want:         false,
		},
		{
			name:         "duo, match duo + random → compte (random pas un coéquipier connu)",
			roster:       duo,
			otherKnown:   toXUIDSet([]string{xC}),
			participants: []string{xA, xB, xR},
			want:         true,
		},
		{
			name:         "duo, match duo strict → compte",
			roster:       duo,
			otherKnown:   toXUIDSet([]string{xC}),
			participants: []string{xA, xB},
			want:         true,
		},
		{
			name:         "duo, autre coéquipier connu D présent → ne compte pas",
			roster:       duo,
			otherKnown:   toXUIDSet([]string{xC, xD}),
			participants: []string{xA, xB, xD},
			want:         false,
		},
		{
			name:         "roster vide → ne compte jamais",
			roster:       toXUIDSet(nil),
			participants: []string{xA, xB, xC},
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchCountsForSquad(tc.roster, tc.otherKnown, tc.participants)
			if got != tc.want {
				t.Errorf("MatchCountsForSquad() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOtherKnownTeammates(t *testing.T) {
	// Joueur avec un trio {A,B,C} et un duo {A,B}. Pour le duo courant,
	// l'autre coéquipier connu est C (et pas A/B qui sont dans le roster courant).
	got := OtherKnownTeammates(
		[]string{xA, xB},
		[][]string{{xA, xB, xC}, {xA, xB}},
	)
	want := toXUIDSet([]string{xC})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OtherKnownTeammates() = %v, want %v", got, want)
	}
}

func TestOtherKnownTeammates_ExcludesCurrentRoster(t *testing.T) {
	// Même membre présent dans plusieurs escouades ne doit pas se disqualifier
	// lui-même : A est dans le roster courant ET dans une autre escouade.
	got := OtherKnownTeammates(
		[]string{xA, xB, xC},
		[][]string{{xA, xD}, {xA, xB, xC}},
	)
	want := toXUIDSet([]string{xD})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OtherKnownTeammates() = %v, want %v", got, want)
	}
}

func TestFilterSquadMatches_TrioVsDuoSession(t *testing.T) {
	// Scénario « session » : sur 4 matchs, le trio joue 2 fois (dont 1 avec
	// random), puis C part et A+B finissent en duo 2 fois.
	matches := []SquadMatchParticipants{
		{MatchID: "m1", Xuids: []string{xA, xB, xC}},     // trio
		{MatchID: "m2", Xuids: []string{xA, xB, xC, xR}}, // trio + random
		{MatchID: "m3", Xuids: []string{xA, xB, xR}},     // duo + random (C parti)
		{MatchID: "m4", Xuids: []string{xA, xB}},         // duo strict
	}

	// Vue TRIO {A,B,C} : autre coéquipier connu = aucun (le duo {A,B} ⊂ trio).
	trioOther := OtherKnownTeammates([]string{xA, xB, xC}, [][]string{{xA, xB, xC}, {xA, xB}})
	gotTrio := FilterSquadMatches([]string{xA, xB, xC}, trioOther, matches)
	if !reflect.DeepEqual(gotTrio, []string{"m1", "m2"}) {
		t.Errorf("trio matches = %v, want [m1 m2]", gotTrio)
	}

	// Vue DUO {A,B} : autre coéquipier connu = C. Les matchs trio (m1,m2) sont
	// disqualifiés ; seuls les vrais matchs duo (m3,m4) comptent.
	duoOther := OtherKnownTeammates([]string{xA, xB}, [][]string{{xA, xB, xC}, {xA, xB}})
	gotDuo := FilterSquadMatches([]string{xA, xB}, duoOther, matches)
	if !reflect.DeepEqual(gotDuo, []string{"m3", "m4"}) {
		t.Errorf("duo matches = %v, want [m3 m4]", gotDuo)
	}
}
