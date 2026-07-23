// Package service — appearance_diag_service_test.go : verdicts du diagnostic
// apparence par scénario (Lot F, F5). Mocks du fetcher live + du lecteur DB ;
// aucun réseau, aucune DB. Vérifie que chaque scénario produit le bon verdict et
// que les valeurs SERVIES (DB) + last_fetch_status transitent tels quels.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	haloclient "levelup/go-api/internal/sync/haloclient"
)

type mockServedReader struct {
	row         *domain.CareerRankRow
	fetchStatus string
	rankErr     error
	statusErr   error
}

func (m *mockServedReader) LoadLastCareerRank(context.Context, string) (*domain.CareerRankRow, error) {
	return m.row, m.rankErr
}

func (m *mockServedReader) LoadLastFetchStatus(context.Context, string) (string, error) {
	return m.fetchStatus, m.statusErr
}

type mockAppearanceFetcher struct {
	inputs    *haloclient.AppearanceInputs
	inputsErr error
	nameplate haloclient.AppearanceDiagnosis
	image     haloclient.AppearanceDiagnosis
}

func (m *mockAppearanceFetcher) FetchAppearanceInputs(context.Context, string) (*haloclient.AppearanceInputs, error) {
	return m.inputs, m.inputsErr
}

func (m *mockAppearanceFetcher) DiagnoseNameplate(context.Context, string, int64) haloclient.AppearanceDiagnosis {
	return m.nameplate
}

func (m *mockAppearanceFetcher) DiagnoseCustomizationImage(context.Context, string) haloclient.AppearanceDiagnosis {
	return m.image
}

var testFixedClock = func() time.Time { return time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC) }

var testServedRow = &domain.CareerRankRow{
	SpartanID:        "JG42",
	BannerImageURL:   "https://cdn/banner.png",
	EmblemImageURL:   "https://cdn/emblem.png",
	BackdropImageURL: "https://cdn/backdrop.png",
}

// baseDeps fournit un jeu de dépendances nominal (profil JGtm connu, lecteur DB
// peuplé, tokens présents, capability absente). Chaque test surcharge ce qu'il cible.
func baseDeps(t *testing.T, fetcher *mockAppearanceFetcher) AppearanceDiagDeps {
	t.Helper()
	return AppearanceDiagDeps{
		LoadPlayers: func(string) ([]domain.PlayerSummary, error) {
			return []domain.PlayerSummary{{PlayerSlug: "JGtm", Gamertag: "JGtm", XUID: "2535"}}, nil
		},
		ResolveServedReader: func(context.Context, string, string) (CareerServedReader, error) {
			return &mockServedReader{row: testServedRow, fetchStatus: "ok"}, nil
		},
		FetchTokens: func(context.Context, string, string) (string, string, error) {
			return "spartan-token", "clearance-token", nil
		},
		NewFetcher:           func(string, string, string) AppearanceFetcher { return fetcher },
		HasSpartanCustomizer: func(string) bool { return false },
		Clock:                testFixedClock,
	}
}

func findComponent(t *testing.T, comps []domain.AppearanceComponentDiagnosis, name string) domain.AppearanceComponentDiagnosis {
	t.Helper()
	for _, c := range comps {
		if c.Component == name {
			return c
		}
	}
	t.Fatalf("composant %q absent de la réponse", name)
	return domain.AppearanceComponentDiagnosis{}
}

// TestDiagnose_Nominal_OK : inputs présents + primitives ok → 4 composants ok,
// valeurs servies et last_fetch_status transmis.
func TestDiagnose_Nominal_OK(t *testing.T) {
	fetcher := &mockAppearanceFetcher{
		inputs:    &haloclient.AppearanceInputs{ServiceTag: "JG42", EmblemPath: "Inventory/Spartan/Emblems/x.json"},
		nameplate: haloclient.AppearanceDiagnosis{ServedFrom: haloclient.ServedFromLive, Verdict: haloclient.VerdictOK, Detail: haloclient.DetailMappingHit},
		image:     haloclient.AppearanceDiagnosis{ServedFrom: haloclient.ServedFromLive, Verdict: haloclient.VerdictOK, Detail: haloclient.DetailImageResolved},
	}
	svc := NewAppearanceDiagService(baseDeps(t, fetcher))
	resp, err := svc.Diagnose(context.Background(), "JGtm", "halo_infinite")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if resp.XUID != "2535" || resp.Gamertag != "JGtm" || resp.TitleSlug != "halo_infinite" {
		t.Fatalf("entête réponse inattendu : %+v", resp)
	}
	if resp.LastFetchStatus != "ok" {
		t.Fatalf("last_fetch_status = %q (attendu ok)", resp.LastFetchStatus)
	}
	if len(resp.Components) != 4 {
		t.Fatalf("nb composants = %d (attendu 4)", len(resp.Components))
	}
	banner := findComponent(t, resp.Components, domain.AppearanceComponentBanner)
	if banner.Verdict != "ok" || banner.ServedFrom != "live" || banner.ServedValue != "https://cdn/banner.png" {
		t.Fatalf("bannière inattendue : %+v", banner)
	}
	tag := findComponent(t, resp.Components, domain.AppearanceComponentServiceTag)
	if tag.Verdict != "ok" || tag.ServedValue != "JG42" {
		t.Fatalf("service tag inattendu : %+v", tag)
	}
}

// TestDiagnose_UpstreamMissing : le verdict nameplate remonte tel quel sur la
// bannière (cas emblème sans nameplate upstream — JGtm 3806589).
func TestDiagnose_UpstreamMissing(t *testing.T) {
	fetcher := &mockAppearanceFetcher{
		inputs:    &haloclient.AppearanceInputs{ServiceTag: "JG42", EmblemPath: "Inventory/Spartan/Emblems/3806589-SpartanEmblem.json"},
		nameplate: haloclient.AppearanceDiagnosis{ServedFrom: haloclient.ServedFromCarry, Verdict: haloclient.VerdictUpstreamMissing, Detail: haloclient.DetailNoPositiveCfg},
		image:     haloclient.AppearanceDiagnosis{ServedFrom: haloclient.ServedFromLive, Verdict: haloclient.VerdictOK, Detail: haloclient.DetailImageResolved},
	}
	svc := NewAppearanceDiagService(baseDeps(t, fetcher))
	resp, err := svc.Diagnose(context.Background(), "JGtm", "halo_infinite")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	banner := findComponent(t, resp.Components, domain.AppearanceComponentBanner)
	if banner.Verdict != "upstream_missing" || banner.ServedFrom != "carry" || banner.Detail != "no_positive_cfg" {
		t.Fatalf("bannière inattendue : %+v", banner)
	}
	// La bannière servie reste la dernière connue (jamais vide).
	if banner.ServedValue != "https://cdn/banner.png" {
		t.Fatalf("valeur servie bannière = %q (attendu la dernière connue)", banner.ServedValue)
	}
	// L'emblème, indépendant, reste ok.
	if emblem := findComponent(t, resp.Components, domain.AppearanceComponentEmblem); emblem.Verdict != "ok" {
		t.Fatalf("emblème inattendu : %+v", emblem)
	}
}

// TestDiagnose_Transient_FetchError : échec du fetch appearance → 4 composants
// transient (uniform), valeurs servies conservées.
func TestDiagnose_Transient_FetchError(t *testing.T) {
	fetcher := &mockAppearanceFetcher{inputsErr: errors.New("HTTP 500")}
	svc := NewAppearanceDiagService(baseDeps(t, fetcher))
	resp, err := svc.Diagnose(context.Background(), "JGtm", "halo_infinite")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	for _, c := range resp.Components {
		if c.Verdict != "transient" || c.ServedFrom != "carry" {
			t.Fatalf("composant non transient : %+v", c)
		}
	}
	if b := findComponent(t, resp.Components, domain.AppearanceComponentBackdrop); b.ServedValue != "https://cdn/backdrop.png" {
		t.Fatalf("valeur servie backdrop perdue : %+v", b)
	}
}

// TestDiagnose_AuthRequired : tokens absents → 4 composants auth_required, le
// fetcher n'est JAMAIS construit.
func TestDiagnose_AuthRequired(t *testing.T) {
	deps := baseDeps(t, nil)
	deps.FetchTokens = func(context.Context, string, string) (string, string, error) {
		return "", "", errors.New("AADSTS: refresh token mort")
	}
	deps.NewFetcher = func(string, string, string) AppearanceFetcher {
		t.Fatalf("NewFetcher ne doit pas être appelé quand les tokens sont absents")
		return nil
	}
	svc := NewAppearanceDiagService(deps)
	resp, err := svc.Diagnose(context.Background(), "JGtm", "halo_infinite")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	for _, c := range resp.Components {
		if c.Verdict != "auth_required" {
			t.Fatalf("composant non auth_required : %+v", c)
		}
	}
}

// TestDiagnose_NotSupported_Capability : titre à pipeline appearance dédié
// (spartan_customizer) → 4 composants not_supported, aucun fetch de tokens.
func TestDiagnose_NotSupported_Capability(t *testing.T) {
	deps := baseDeps(t, nil)
	deps.HasSpartanCustomizer = func(string) bool { return true }
	deps.FetchTokens = func(context.Context, string, string) (string, string, error) {
		t.Fatalf("FetchTokens ne doit pas être appelé pour un titre spartan_customizer")
		return "", "", nil
	}
	deps.NewFetcher = func(string, string, string) AppearanceFetcher {
		t.Fatalf("NewFetcher ne doit pas être appelé pour un titre spartan_customizer")
		return nil
	}
	svc := NewAppearanceDiagService(deps)
	resp, err := svc.Diagnose(context.Background(), "JGtm", "halo_5")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	for _, c := range resp.Components {
		if c.Verdict != "not_supported" {
			t.Fatalf("composant non not_supported : %+v", c)
		}
	}
	// Valeurs servies affichées malgré not_supported.
	if e := findComponent(t, resp.Components, domain.AppearanceComponentEmblem); e.ServedValue != "https://cdn/emblem.png" {
		t.Fatalf("valeur servie emblème perdue : %+v", e)
	}
}

// TestDiagnose_ProfileNotFound : slug inconnu → ErrProfileNotFound (→ 404 côté handler).
func TestDiagnose_ProfileNotFound(t *testing.T) {
	deps := baseDeps(t, nil)
	svc := NewAppearanceDiagService(deps)
	_, err := svc.Diagnose(context.Background(), "Inconnu", "halo_infinite")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("erreur = %v (attendu ErrProfileNotFound)", err)
	}
}

// TestDiagnose_ServedReaderError_Degraded : lecteur DB en erreur → pas d'échec,
// valeurs servies vides, le diagnostic live continue.
func TestDiagnose_ServedReaderError_Degraded(t *testing.T) {
	fetcher := &mockAppearanceFetcher{
		inputs:    &haloclient.AppearanceInputs{ServiceTag: "JG42"},
		nameplate: haloclient.AppearanceDiagnosis{ServedFrom: haloclient.ServedFromLive, Verdict: haloclient.VerdictOK},
		image:     haloclient.AppearanceDiagnosis{ServedFrom: haloclient.ServedFromLive, Verdict: haloclient.VerdictOK},
	}
	deps := baseDeps(t, fetcher)
	deps.ResolveServedReader = func(context.Context, string, string) (CareerServedReader, error) {
		return nil, errors.New("player DB inaccessible")
	}
	svc := NewAppearanceDiagService(deps)
	resp, err := svc.Diagnose(context.Background(), "JGtm", "halo_infinite")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if resp.LastFetchStatus != "" {
		t.Fatalf("last_fetch_status = %q (attendu vide)", resp.LastFetchStatus)
	}
	if b := findComponent(t, resp.Components, domain.AppearanceComponentBanner); b.ServedValue != "" {
		t.Fatalf("valeur servie devrait être vide : %+v", b)
	}
}
