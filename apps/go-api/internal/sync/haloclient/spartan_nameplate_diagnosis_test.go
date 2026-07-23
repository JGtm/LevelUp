// Package haloclient — spartan_nameplate_diagnosis_test.go : tests du diagnostic
// structuré d'apparence (Lot E, plan .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md).
//
// Cadenasse le mapping verdict/detail par branche du resolver ET le cas
// canonique upstream_missing (emblème 3806589-SpartanEmblem, thought_log
// 2026-07-08). Les TestResolveNameplateURL_* existants (spartan_nameplate_resolver_test.go)
// restent la preuve du byte-identique : ce fichier n'ajoute que de NOUVEAUX tests.
package haloclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// emblemCMS3806589 reproduit le JSON GameCMS de l'emblème équipé par JGtm le
// 2026-07-03 : `Inventory/Spartan/Emblems/3806589-SpartanEmblem.json`, item
// « Women's History Month », configuration équipée -1766636888.
//
// Fait documenté au thought_log [2026-07-08] (« Bannière JGtm figée ») :
// emblème nouvelle génération `<id>-SpartanEmblem`, absent de mapping.json
// (243 entrées legacy `104-001-…`), AUCUNE cfg POSITIVE dans
// AvailableConfigurations, aucun PNG nameplate servi par le CDN (probes 404).
// C'est la signature exacte du verdict upstream_missing DÉFINITIF.
//
// Le payload d'octets exact n'a pas été conservé au journal ; cette fixture
// reproduit fidèlement l'invariant PROUVÉ (configuration unique, négative →
// firstPositiveEmblemCfg rend (0, true) = CMS lisible sans cfg positive).
const emblemCMS3806589 = `{
  "CommonData": {
    "Id": "3806589-SpartanEmblem",
    "Title": {"value": "Women's History Month"}
  },
  "AvailableConfigurations": [
    {"ConfigurationId": -1766636888}
  ]
}`

// redirectDefaultClient redirige http.DefaultClient vers srv le temps du test,
// restauré au retour. resolvePositiveEmblemCfg / refreshEmblemMapping tapent
// http.DefaultClient contre l'host GameCMS CONSTANT (non injectable) : c'est le
// seul moyen de piloter leur réponse en test. Réutilise redirectTransport
// (halo_client_extra_test.go).
func redirectDefaultClient(t *testing.T, srv *httptest.Server) func() {
	t.Helper()
	host := strings.TrimPrefix(srv.URL, "http://")
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = &redirectTransport{host: host}
	return func() { http.DefaultClient.Transport = old }
}

// ─── firstPositiveEmblemCfg (seam de parsing, cas 3806589) ───────────────────

func TestFirstPositiveEmblemCfg_UpstreamMissingCase3806589(t *testing.T) {
	// JSON CMS RÉEL du cas 3806589 : lisible mais AUCUNE cfg positive →
	// (0, true) = absence upstream DÉFINITIVE (upstream_missing, pas transient).
	cfg, parsed := firstPositiveEmblemCfg([]byte(emblemCMS3806589))
	if !parsed {
		t.Fatal("payload CMS réel : parsed attendu true (JSON valide)")
	}
	if cfg != 0 {
		t.Fatalf("cfg = %d, attendu 0 (aucune cfg positive → upstream_missing définitif)", cfg)
	}
}

func TestFirstPositiveEmblemCfg_TransientOnUnparseable(t *testing.T) {
	// Corps HTTP illisible (CMS KO) → parse impossible → (0, false) = indéterminé
	// (transient).
	cfg, parsed := firstPositiveEmblemCfg([]byte("<html>502 Bad Gateway</html>"))
	if parsed {
		t.Fatal("corps non-JSON : parsed attendu false (indéterminé → transient)")
	}
	if cfg != 0 {
		t.Fatalf("cfg = %d, attendu 0", cfg)
	}
}

func TestFirstPositiveEmblemCfg_PositiveFound(t *testing.T) {
	cfg, parsed := firstPositiveEmblemCfg(
		[]byte(`{"AvailableConfigurations":[{"ConfigurationId":-5},{"ConfigurationId":651339664}]}`))
	if !parsed || cfg != 651339664 {
		t.Fatalf("cfg=%d parsed=%v, attendu 651339664/true", cfg, parsed)
	}
}

// ─── DiagnoseNameplate — verdicts déterministes (sans réseau) ────────────────

func TestDiagnoseNameplate_MappingHitOK(t *testing.T) {
	// Cache mapping seedé → ok/mapping_hit, URL exacte du mapping.
	seedEmblemMappingCacheForTest(map[string]map[string]emblemMappingEntry{
		"104-001-olympus-campa-2ddbe23b": {
			"-809699482": {
				NameplateCmsPath: "images/nameplates/104-001-olympus-campa-2ddbe23b_n809699482.png",
			},
		},
	})
	defer resetEmblemMappingCacheForTest()

	diag := DiagnoseNameplate(context.Background(),
		"Inventory/Spartan/Emblems/104-001-olympus-campa-2ddbe23b.json",
		-809699482, "spartan", "clearance")
	if diag.Verdict != VerdictOK || diag.Detail != DetailMappingHit {
		t.Fatalf("verdict/detail = %q/%q, attendu ok/mapping_hit", diag.Verdict, diag.Detail)
	}
	if diag.ServedFrom != ServedFromLive {
		t.Fatalf("served_from = %q, attendu live", diag.ServedFrom)
	}
	want := "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/nameplates/104-001-olympus-campa-2ddbe23b_n809699482.png"
	if diag.ResolvedURL != want {
		t.Fatalf("url = %q, attendu %q", diag.ResolvedURL, want)
	}
}

func TestDiagnoseNameplate_PositiveCfgDirectOK(t *testing.T) {
	// Cache mapping VIDE mais frais → getEmblemMappingEntry ne déclenche AUCUN
	// refresh réseau (pas stale) et rate le lookup → fallback. cfg > 0 →
	// ok/mapping_miss (résolu via cfg positive, mapping absent).
	seedEmblemMappingCacheForTest(map[string]map[string]emblemMappingEntry{})
	defer resetEmblemMappingCacheForTest()

	diag := DiagnoseNameplate(context.Background(),
		"Inventory/Spartan/Emblems/104-001-test-emblem-abcdef.json",
		372285867, "s", "c")
	if diag.Verdict != VerdictOK || diag.Detail != DetailMappingMiss {
		t.Fatalf("verdict/detail = %q/%q, attendu ok/mapping_miss", diag.Verdict, diag.Detail)
	}
	want := "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/nameplates/104-001-test-emblem-abcdef_372285867.png"
	if diag.ResolvedURL != want {
		t.Fatalf("url = %q, attendu %q", diag.ResolvedURL, want)
	}
}

func TestDiagnoseNameplate_Guards(t *testing.T) {
	// Cache vide+frais pour éviter tout refresh réseau (les chemins guard
	// n'atteignent pas le fallback, mais on protège quand même).
	seedEmblemMappingCacheForTest(map[string]map[string]emblemMappingEntry{})
	defer resetEmblemMappingCacheForTest()

	empty := DiagnoseNameplate(context.Background(), "   ", 1, "s", "c")
	if empty.ResolvedURL != "" || empty.Detail != DetailNoEmblemPath {
		t.Fatalf("empty path : url=%q detail=%q, attendu \"\"/no_emblem_path", empty.ResolvedURL, empty.Detail)
	}
	nonEmblem := DiagnoseNameplate(context.Background(),
		"Inventory/Spartan/BackdropImages/x.json", 1, "s", "c")
	if nonEmblem.ResolvedURL != "" || nonEmblem.Detail != DetailNonEmblemPath {
		t.Fatalf("non-emblem : url=%q detail=%q, attendu \"\"/non_emblem_path", nonEmblem.ResolvedURL, nonEmblem.Detail)
	}
}

// ─── DiagnoseNameplate — full path via redirection http.DefaultClient ────────

func TestDiagnoseNameplate_UpstreamMissing_RealCMS(t *testing.T) {
	// Cache mapping vide+frais → pas de refresh ; cfg<=0 → fallback vers
	// resolvePositiveEmblemCfg, redirigé vers un httptest qui SERT le JSON CMS
	// RÉEL du cas 3806589 → verdict upstream_missing/no_positive_cfg + log Info.
	seedEmblemMappingCacheForTest(map[string]map[string]emblemMappingEntry{})
	defer resetEmblemMappingCacheForTest()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/progression/file/Inventory/Spartan/Emblems/3806589-SpartanEmblem.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(emblemCMS3806589))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	defer redirectDefaultClient(t, srv)()

	diag := DiagnoseNameplate(context.Background(),
		"Inventory/Spartan/Emblems/3806589-SpartanEmblem.json",
		-1766636888, "spartan", "clearance")
	if diag.Verdict != VerdictUpstreamMissing || diag.Detail != DetailNoPositiveCfg {
		t.Fatalf("verdict/detail = %q/%q, attendu upstream_missing/no_positive_cfg", diag.Verdict, diag.Detail)
	}
	if diag.ServedFrom != ServedFromCarry {
		t.Fatalf("served_from = %q, attendu carry", diag.ServedFrom)
	}
	if diag.ResolvedURL != "" {
		t.Fatalf("url = %q, attendu vide", diag.ResolvedURL)
	}
}

func TestDiagnoseNameplate_Transient_HTTPError(t *testing.T) {
	// CMS répond 500 → resolvePositiveEmblemCfg indéterminé → transient/cms_http_error + log Warn.
	seedEmblemMappingCacheForTest(map[string]map[string]emblemMappingEntry{})
	defer resetEmblemMappingCacheForTest()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer redirectDefaultClient(t, srv)()

	diag := DiagnoseNameplate(context.Background(),
		"Inventory/Spartan/Emblems/3806589-SpartanEmblem.json",
		-1766636888, "spartan", "clearance")
	if diag.Verdict != VerdictTransient || diag.Detail != DetailCMSHTTPError {
		t.Fatalf("verdict/detail = %q/%q, attendu transient/cms_http_error", diag.Verdict, diag.Detail)
	}
	if diag.ServedFrom != ServedFromCarry {
		t.Fatalf("served_from = %q, attendu carry", diag.ServedFrom)
	}
}

// ─── DiagnoseCustomizationImage (emblème / backdrop) ─────────────────────────

func TestDiagnoseCustomizationImage_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/progression/file/Inventory/Spartan/Emblems/") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{"DisplayPath": map[string]any{
					"Media": map[string]any{"MediaUrl": map[string]any{"Path": "progression/Emblems/e.png"}}}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		gameCMSBaseURL: srv.URL,
		limiter:        fastLimiter(),
	}
	diag := c.DiagnoseCustomizationImage(context.Background(), "Inventory/Spartan/Emblems/e.json")
	if diag.Verdict != VerdictOK || diag.Detail != DetailImageResolved {
		t.Fatalf("verdict/detail = %q/%q, attendu ok/image_resolved", diag.Verdict, diag.Detail)
	}
	if diag.ServedFrom != ServedFromLive {
		t.Fatalf("served_from = %q, attendu live", diag.ServedFrom)
	}
	if diag.ResolvedURL != srv.URL+"/hi/images/file/progression/Emblems/e.png" {
		t.Fatalf("url = %q", diag.ResolvedURL)
	}
}

func TestDiagnoseCustomizationImage_TransientOnHTTPError(t *testing.T) {
	// 404 (ressource absente) : doGet rend l'erreur immédiatement (pas de retry
	// interne, contrairement à un 5xx) → resolve KO → transient, test rapide.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		gameCMSBaseURL: srv.URL,
		limiter:        fastLimiter(),
	}
	diag := c.DiagnoseCustomizationImage(context.Background(), "Inventory/Spartan/Emblems/e.json")
	if diag.Verdict != VerdictTransient || diag.Detail != DetailImageUnresolved {
		t.Fatalf("verdict/detail = %q/%q, attendu transient/image_unresolved", diag.Verdict, diag.Detail)
	}
	if diag.ServedFrom != ServedFromCarry {
		t.Fatalf("served_from = %q, attendu carry", diag.ServedFrom)
	}
}

// ─── DiagnoseServiceTag ──────────────────────────────────────────────────────

func TestDiagnoseServiceTag(t *testing.T) {
	present := DiagnoseServiceTag("JGTM")
	if present.Verdict != VerdictOK || present.Detail != DetailServiceTagPresent || present.ServedFrom != ServedFromLive {
		t.Fatalf("present: %+v, attendu ok/service_tag_present/live", present)
	}
	absent := DiagnoseServiceTag("   ")
	if absent.Verdict != VerdictTransient || absent.Detail != DetailNoServiceTag || absent.ServedFrom != ServedFromCarry {
		t.Fatalf("absent: %+v, attendu transient/no_service_tag/carry", absent)
	}
}
