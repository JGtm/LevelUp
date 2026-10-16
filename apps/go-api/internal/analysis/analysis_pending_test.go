package analysis

import (
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
