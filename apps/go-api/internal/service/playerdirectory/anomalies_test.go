package playerdirectory

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestComputeAnomalies : la fonction pure, cas par cas, sans aucune source.
func TestComputeAnomalies(t *testing.T) {
	profile := domain.ProfileRef{TitleSlug: testTitle, Key: "Spartan", SyncEnabled: true}

	cases := []struct {
		name string
		rec  domain.IdentityRecord
		want []domain.IdentityAnomaly
	}{
		{
			name: "identite complete",
			rec: domain.IdentityRecord{
				XUID:     "111",
				Profiles: []domain.ProfileRef{profile},
				Account:  &domain.AccountRef{Username: "spartan"},
				Token:    &domain.TokenRef{HasRefreshToken: true},
				Watched:  []string{testTitle},
			},
			want: nil,
		},
		{
			name: "compte sans profil",
			rec:  domain.IdentityRecord{XUID: "999", Account: &domain.AccountRef{Username: "inconnu"}},
			want: []domain.IdentityAnomaly{
				{Code: domain.AnomalyAccountWithoutProfile, Severity: domain.AnomalySeverityWarning, Detail: "inconnu"},
			},
		},
		{
			name: "token orphelin",
			rec:  domain.IdentityRecord{XUID: "777", Token: &domain.TokenRef{}},
			want: []domain.IdentityAnomaly{
				{Code: domain.AnomalyTokenOrphan, Severity: domain.AnomalySeverityWarning, Detail: "777"},
			},
		},
		{
			name: "token avec compte : pas orphelin, mais compte sans profil",
			rec: domain.IdentityRecord{
				XUID:    "999",
				Account: &domain.AccountRef{Username: "inconnu"},
				Token:   &domain.TokenRef{},
			},
			want: []domain.IdentityAnomaly{
				{Code: domain.AnomalyAccountWithoutProfile, Severity: domain.AnomalySeverityWarning, Detail: "inconnu"},
			},
		},
		{
			name: "dossier orphelin",
			rec: domain.IdentityRecord{
				XUID:       "111",
				Profiles:   []domain.ProfileRef{profile},
				Account:    &domain.AccountRef{Username: "spartan"},
				Token:      &domain.TokenRef{},
				OrphanDirs: []domain.OrphanDirRef{{TitleSlug: testOtherTitle, Name: "Spartan"}},
			},
			want: []domain.IdentityAnomaly{
				{Code: domain.AnomalyPlayerDirOrphan, Severity: domain.AnomalySeverityWarning, Detail: testOtherTitle + "/Spartan"},
			},
		},
		{
			name: "suivi live sans profil sur ce titre",
			rec: domain.IdentityRecord{
				XUID:     "111",
				Profiles: []domain.ProfileRef{profile},
				Account:  &domain.AccountRef{Username: "spartan"},
				Token:    &domain.TokenRef{},
				Watched:  []string{testTitle, testOtherTitle},
			},
			want: []domain.IdentityAnomaly{
				{Code: domain.AnomalyWatchedWithoutProfile, Severity: domain.AnomalySeverityWarning, Detail: testOtherTitle},
			},
		},
		{
			name: "profil d'ami : deux info, aucun warning",
			rec:  domain.IdentityRecord{XUID: "222", Profiles: []domain.ProfileRef{profile}},
			want: []domain.IdentityAnomaly{
				{Code: domain.AnomalyProfileWithoutAccount, Severity: domain.AnomalySeverityInfo},
				{Code: domain.AnomalyProfileWithoutToken, Severity: domain.AnomalySeverityInfo},
			},
		},
		{
			name: "profil en pause : suivi non, mais ce n'est pas une anomalie",
			rec: domain.IdentityRecord{
				XUID:     "222",
				Profiles: []domain.ProfileRef{{TitleSlug: testTitle, Key: "Pause"}},
				Account:  &domain.AccountRef{Username: "pause"},
				Token:    &domain.TokenRef{},
			},
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeAnomalies(tc.rec)
			if len(got) != len(tc.want) {
				t.Fatalf("anomalies = %+v, want %+v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("anomalie %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestComputeAnomalies_WarningsAvantInfos : l'ordre est stable et met les
// incohérences devant, ce sur quoi s'appuie le tri des identités.
func TestComputeAnomalies_WarningsAvantInfos(t *testing.T) {
	rec := domain.IdentityRecord{
		XUID:       "111",
		Profiles:   []domain.ProfileRef{{TitleSlug: testTitle, Key: "Spartan"}},
		OrphanDirs: []domain.OrphanDirRef{{TitleSlug: testTitle, Name: "Autre"}},
	}
	got := computeAnomalies(rec)
	if len(got) != 3 {
		t.Fatalf("anomalies = %+v, want 3", got)
	}
	if got[0].Severity != domain.AnomalySeverityWarning {
		t.Fatalf("première anomalie = %+v, want warning", got[0])
	}
	if got[1].Severity != domain.AnomalySeverityInfo || got[2].Severity != domain.AnomalySeverityInfo {
		t.Fatalf("anomalies = %+v, want les info en queue", got)
	}
}
