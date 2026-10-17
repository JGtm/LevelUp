package analysis

import (
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

var instantRef = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func probeSynchroIlYA(age time.Duration, mod func(*domain.AnalysisProbe)) domain.AnalysisProbe {
	t := instantRef.Add(-age)
	p := domain.AnalysisProbe{FirstSyncAt: &t, KillSourceDone: true, HasCitationRows: true}
	if mod != nil {
		mod(&p)
	}
	return p
}

func TestIsAnalysisPending(t *testing.T) {
	tests := []struct {
		name string
		in   AnalysisPendingInput
		want bool
	}{
		{
			name: "tout est fait : rien en cours",
			in:   AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, nil), ReplayExpected: true, ReplayKnown: true, HasReplay: true},
			want: false,
		},
		{
			name: "c1 : source du kill non traitee",
			in: AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.KillSourceDone = false }),
				ReplayExpected: true, ReplayKnown: true, HasReplay: true},
			want: true,
		},
		{
			name: "c2 : citations non calculees",
			in: AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.HasCitationRows = false }),
				ReplayExpected: true, ReplayKnown: true, HasReplay: true},
			want: true,
		},
		{
			name: "c3 : rejeu absent, film non declare absent",
			in:   AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, nil), ReplayExpected: true, ReplayKnown: true, HasReplay: false},
			want: true,
		},
		{
			name: "c3 : rejeu absent mais film declare absent -> rien ne viendra",
			in: AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.FilmAbsent = true }),
				ReplayExpected: true, ReplayKnown: true, HasReplay: false},
			want: false,
		},
		{
			name: "c3 : titre sans artefact de rejeu -> jamais une attente",
			in:   AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, nil), ReplayExpected: false, ReplayKnown: true, HasReplay: false},
			want: false,
		},
		{
			name: "c3 : presence des rejeux non lue -> condition ignoree",
			in:   AnalysisPendingInput{Probe: probeSynchroIlYA(time.Hour, nil), ReplayExpected: true, ReplayKnown: false, HasReplay: false},
			want: false,
		},
		{
			name: "borne : 23 h 59 min 59 s -> encore en cours",
			in:   AnalysisPendingInput{Probe: probeSynchroIlYA(AnalysisPendingWindow-time.Second, func(p *domain.AnalysisProbe) { p.KillSourceDone = false })},
			want: true,
		},
		{
			name: "borne : pile 24 h -> faux",
			in:   AnalysisPendingInput{Probe: probeSynchroIlYA(AnalysisPendingWindow, func(p *domain.AnalysisProbe) { p.KillSourceDone = false })},
			want: false,
		},
		{
			name: "borne : au-dela de 24 h -> faux quoi qu'il manque",
			in: AnalysisPendingInput{Probe: probeSynchroIlYA(30*time.Hour, func(p *domain.AnalysisProbe) {
				p.KillSourceDone, p.HasCitationRows = false, false
			}), ReplayExpected: true, ReplayKnown: true},
			want: false,
		},
		{
			name: "premiere synchro inconnue -> faux (pas d'indicateur sans borne d'age)",
			in:   AnalysisPendingInput{Probe: domain.AnalysisProbe{}},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Now = instantRef
			if got := IsAnalysisPending(tc.in); got != tc.want {
				t.Fatalf("IsAnalysisPending = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAnalysisPendingParts(t *testing.T) {
	const (
		ks    = domain.AnalysisPartKillSource
		cit   = domain.AnalysisPartCitations
		rep   = domain.AnalysisPartReplay
		heure = time.Hour
	)
	sansKill := func(p *domain.AnalysisProbe) { p.KillSourceDone = false }
	sansCit := func(p *domain.AnalysisProbe) { p.HasCitationRows = false }
	rejeu := func(mod func(*domain.AnalysisProbe), hasReplay bool) AnalysisPendingInput {
		return AnalysisPendingInput{Probe: probeSynchroIlYA(heure, mod), ReplayExpected: true, ReplayKnown: true, HasReplay: hasReplay}
	}
	tests := []struct {
		name string
		in   AnalysisPendingInput
		want []string
	}{
		{"tout est fait", rejeu(nil, true), nil},
		{"kill_source seule", rejeu(sansKill, true), []string{ks}},
		{"citations seules", rejeu(sansCit, true), []string{cit}},
		{"replay seul", rejeu(nil, false), []string{rep}},
		{"kill_source + citations", rejeu(func(p *domain.AnalysisProbe) { sansKill(p); sansCit(p) }, true), []string{ks, cit}},
		{"kill_source + replay", rejeu(sansKill, false), []string{ks, rep}},
		{"citations + replay", rejeu(sansCit, false), []string{cit, rep}},
		{"les trois, ordre stable", rejeu(func(p *domain.AnalysisProbe) { sansKill(p); sansCit(p) }, false), []string{ks, cit, rep}},
		{
			"film absent sans rejeu : pas de replay, les citations restent",
			rejeu(func(p *domain.AnalysisProbe) { p.FilmAbsent = true; p.HasCitationRows = false }, false),
			[]string{cit},
		},
		{"film absent sans rejeu, le reste fait : rien", rejeu(func(p *domain.AnalysisProbe) { p.FilmAbsent = true }, false), nil},
		{
			"sans capability rejeu : jamais replay",
			AnalysisPendingInput{Probe: probeSynchroIlYA(heure, sansCit), ReplayExpected: false, ReplayKnown: true},
			[]string{cit},
		},
		{
			"presence des rejeux non lue : jamais replay",
			AnalysisPendingInput{Probe: probeSynchroIlYA(heure, sansKill), ReplayExpected: true, ReplayKnown: false},
			[]string{ks},
		},
		{
			"hors fenetre : rien, quoi qu'il manque",
			AnalysisPendingInput{Probe: probeSynchroIlYA(AnalysisPendingWindow, func(p *domain.AnalysisProbe) {
				sansKill(p)
				sansCit(p)
			}), ReplayExpected: true, ReplayKnown: true},
			nil,
		},
		{"premiere synchro inconnue : rien", AnalysisPendingInput{Probe: domain.AnalysisProbe{}}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Now = instantRef
			got := AnalysisPendingParts(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("AnalysisPendingParts = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAnalysisPending_InvarianteBooleenEtCauses balaie toutes les combinaisons de faits et de
// ages : `IsAnalysisPending` doit valoir `len(AnalysisPendingParts) > 0`, et les causes
// rendues restent dans l'ordre stable, sans doublon.
func TestAnalysisPending_InvarianteBooleenEtCauses(t *testing.T) {
	ordre := map[string]int{domain.AnalysisPartKillSource: 0, domain.AnalysisPartCitations: 1, domain.AnalysisPartReplay: 2}
	ages := []time.Duration{time.Hour, AnalysisPendingWindow - time.Second, AnalysisPendingWindow, 30 * time.Hour}
	for _, age := range ages {
		for mask := 0; mask < 1<<7; mask++ {
			bit := func(i uint) bool { return mask&(1<<i) != 0 }
			in := AnalysisPendingInput{
				Probe: probeSynchroIlYA(age, func(p *domain.AnalysisProbe) {
					p.KillSourceDone, p.HasCitationRows, p.FilmAbsent = bit(0), bit(1), bit(2)
				}),
				Now: instantRef, ReplayExpected: bit(3), ReplayKnown: bit(4), HasReplay: bit(5),
			}
			if bit(6) {
				in.Probe.FirstSyncAt = nil
			}
			parts := AnalysisPendingParts(in)
			if got := IsAnalysisPending(in); got != (len(parts) > 0) {
				t.Fatalf("age=%v mask=%07b : IsAnalysisPending=%v mais parts=%v", age, mask, got, parts)
			}
			for i := 1; i < len(parts); i++ {
				if ordre[parts[i-1]] >= ordre[parts[i]] {
					t.Fatalf("age=%v mask=%07b : ordre instable %v", age, mask, parts)
				}
			}
		}
	}
}

func TestAnalysisStateOf(t *testing.T) {
	cas := []struct {
		nom, match, decodant, want string
	}{
		{"aucun décodage", "m1", "", domain.AnalysisStateQueued},
		{"décodage de ce match", "m1", "m1", domain.AnalysisStateInProgress},
		{"casse différente", "ABC-1", "abc-1", domain.AnalysisStateInProgress},
		{"décodage d'un autre match", "m1", "m2", domain.AnalysisStateQueued},
		{"match inconnu et rien ne décode", "", "", domain.AnalysisStateQueued},
		{"match inconnu, un autre décode", "", "m2", domain.AnalysisStateQueued},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := AnalysisStateOf(c.match, c.decodant); got != c.want {
				t.Fatalf("AnalysisStateOf(%q, %q) = %q, want %q", c.match, c.decodant, got, c.want)
			}
		})
	}
}
