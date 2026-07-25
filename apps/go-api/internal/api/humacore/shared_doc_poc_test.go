// Package humacore — POC spike H0.5 (plan .ai/V7/PLAN_V72_HUMA_OPENAPI.md).
//
// TestSharedDocMergedFromSubrouters PROUVE l'hypothèse de dérisquage du chantier
// V72-01 : plusieurs adaptateurs Huma (un par sous-routeur chi HÉTÉROGÈNE) peuvent
// pointer vers LE MÊME document/registre *huma.OpenAPI et produire un document
// fusionné CORRECT, SANS changer le comportement HTTP.
//
// Deux archétypes réalistes de sous-routeur, calqués sur server_apiv1.go :
//   - GROUPE A : sous-routeur monté sous un préfixe à path param
//     (`/api/v1/players/{player_slug}`) + middleware chi témoin → NewSubrouterAPI ;
//   - GROUPE B : groupe middleware-only (RequireAuth-like) à chemin PLAT
//     (`/matches/{match_id}`) → NewAPIWithConfig.
//
// Vérifications :
//
//	(a) les handlers répondent (httptest) : path param PARENT extrait, middleware
//	    du sous-groupe exécuté (témoin + gate 401) ;
//	(b) le document PARTAGÉ contient les paths COMPLETS (préfixe parent inclus pour
//	    A, chemin plat pour B) et les schémas des DTOs des DEUX groupes ;
//	(c) registre de schémas partagé : le type commun CommonMeta est enregistré UNE
//	    fois et référencé ($ref) par les DTOs des deux groupes — pas d'écrasement ;
//	    operationIDs uniques inter-groupes (AddOperation ne panique pas).
package humacore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// DTOs partagés et par-groupe.
// ---------------------------------------------------------------------------

// CommonMeta est référencé par les DTOs des DEUX groupes → doit apparaître UNE
// seule fois dans le registre partagé.
type CommonMeta struct {
	TitleSlug string `json:"title_slug"`
	Count     int    `json:"count"`
}

type AlphaResponse struct {
	Slug          string     `json:"slug"`
	MiddlewareRan bool       `json:"middleware_ran"`
	Meta          CommonMeta `json:"meta"`
}

type BravoResponse struct {
	MatchID string     `json:"match_id"`
	AuthOK  bool       `json:"auth_ok"`
	Meta    CommonMeta `json:"meta"`
}

type SubmitBody struct {
	Note string `json:"note"`
}

type alphaInput struct {
	PlayerSlug string `path:"player_slug"`
}
type alphaOutput struct{ Body AlphaResponse }

type submitInput struct {
	PlayerSlug string `path:"player_slug"`
	Body       SubmitBody
}
type submitOutput struct{ Body AlphaResponse }

type bravoInput struct {
	MatchID string `path:"match_id"`
}
type bravoOutput struct{ Body BravoResponse }

type pocCtxKey string

const (
	witnessKey pocCtxKey = "poc_witness"
	authKey    pocCtxKey = "poc_auth"
)

// ---------------------------------------------------------------------------
// Construction du routeur fusionné (2 sous-routeurs, 1 document partagé).
// ---------------------------------------------------------------------------

const playersPrefix = "/api/v1/players/{player_slug}"

func buildMergedRouter() (*chi.Mux, huma.Config) {
	cfg := NewSharedConfig()
	r := chi.NewRouter()

	// GROUPE A — sous-routeur à préfixe path-param + middleware témoin.
	r.Route(playersPrefix, func(sr chi.Router) {
		sr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := context.WithValue(req.Context(), witnessKey, true)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		apiA := NewSubrouterAPI(sr, cfg, playersPrefix)
		huma.Get(apiA, "/pages/probe", alphaHandler)
		huma.Post(apiA, "/pages/submit", submitHandler)
	})

	// GROUPE B — groupe middleware-only RequireAuth-like, chemin PLAT.
	r.Group(func(gr chi.Router) {
		gr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Header.Get("X-Auth") == "" {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"code":"unauthorized"}` + "\n"))
					return
				}
				ctx := context.WithValue(req.Context(), authKey, true)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		apiB := NewAPIWithConfig(gr, cfg)
		huma.Get(apiB, "/matches/{match_id}", bravoHandler)
	})

	return r, cfg
}

func alphaHandler(ctx context.Context, in *alphaInput) (*alphaOutput, error) {
	out := &alphaOutput{}
	out.Body.Slug = in.PlayerSlug
	out.Body.MiddlewareRan, _ = ctx.Value(witnessKey).(bool)
	out.Body.Meta = CommonMeta{TitleSlug: "halo_infinite", Count: 1}
	return out, nil
}

func submitHandler(ctx context.Context, in *submitInput) (*submitOutput, error) {
	out := &submitOutput{}
	out.Body.Slug = in.PlayerSlug
	out.Body.Meta = CommonMeta{TitleSlug: "halo_infinite", Count: len(in.Body.Note)}
	return out, nil
}

func bravoHandler(ctx context.Context, in *bravoInput) (*bravoOutput, error) {
	out := &bravoOutput{}
	out.Body.MatchID = in.MatchID
	out.Body.AuthOK, _ = ctx.Value(authKey).(bool)
	out.Body.Meta = CommonMeta{TitleSlug: "halo_5", Count: 2}
	return out, nil
}

// ---------------------------------------------------------------------------
// (a) Comportement HTTP : params parents + middlewares.
// ---------------------------------------------------------------------------

func TestSharedDoc_HTTPBehaviorPreserved(t *testing.T) {
	r, _ := buildMergedRouter()

	// Groupe A : path param parent {player_slug} + middleware témoin.
	var a struct {
		Slug          string     `json:"slug"`
		MiddlewareRan bool       `json:"middleware_ran"`
		Meta          CommonMeta `json:"meta"`
	}
	doJSON(t, r, http.MethodGet, "/api/v1/players/Madina97294/pages/probe", nil, http.StatusOK, &a)
	if a.Slug != "Madina97294" {
		t.Errorf("A: slug = %q, want Madina97294 (path param parent non lu)", a.Slug)
	}
	if !a.MiddlewareRan {
		t.Error("A: middleware du sous-groupe NON hérité par la route Huma")
	}
	if a.Meta.TitleSlug != "halo_infinite" {
		t.Errorf("A: meta.title_slug = %q, want halo_infinite", a.Meta.TitleSlug)
	}

	// Groupe A POST avec body typé.
	var ap struct {
		Slug string     `json:"slug"`
		Meta CommonMeta `json:"meta"`
	}
	doJSON(t, r, http.MethodPost, "/api/v1/players/Madina97294/pages/submit",
		strings.NewReader(`{"note":"hi"}`), http.StatusOK, &ap)
	if ap.Slug != "Madina97294" || ap.Meta.Count != 2 {
		t.Errorf("A POST: slug=%q count=%d, want Madina97294/2", ap.Slug, ap.Meta.Count)
	}

	// Groupe B : gate RequireAuth-like — 401 sans en-tête.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/matches/xyz", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("B sans auth: code = %d, want 401 (middleware non exécuté)", rec.Code)
	}

	// Groupe B : 200 avec en-tête → param plat + flag auth.
	var b struct {
		MatchID string `json:"match_id"`
		AuthOK  bool   `json:"auth_ok"`
	}
	req := httptest.NewRequest(http.MethodGet, "/matches/xyz", nil)
	req.Header.Set("X-Auth", "tok")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("B avec auth: code = %d: %s", rec2.Code, rec2.Body.String())
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &b); err != nil {
		t.Fatalf("B: JSON invalide: %v", err)
	}
	if b.MatchID != "xyz" || !b.AuthOK {
		t.Errorf("B: match_id=%q auth_ok=%t, want xyz/true", b.MatchID, b.AuthOK)
	}
}

// ---------------------------------------------------------------------------
// (b) Document fusionné : paths COMPLETS (préfixe parent inclus).
// ---------------------------------------------------------------------------

func TestSharedDoc_MergedPathsHaveParentPrefix(t *testing.T) {
	_, cfg := buildMergedRouter()
	paths := cfg.OpenAPI.Paths

	// Chemins ABSOLUS attendus (préfixe parent inclus pour A).
	if pi := paths["/api/v1/players/{player_slug}/pages/probe"]; pi == nil || pi.Get == nil {
		t.Errorf("path A GET absent ou sans opération : %+v", pi)
	}
	if pi := paths["/api/v1/players/{player_slug}/pages/submit"]; pi == nil || pi.Post == nil {
		t.Errorf("path A POST absent ou sans opération : %+v", pi)
	}
	if pi := paths["/matches/{match_id}"]; pi == nil || pi.Get == nil {
		t.Errorf("path B GET absent ou sans opération : %+v", pi)
	}

	// Le chemin LOCAL nu (sans préfixe parent) NE DOIT PAS exister : preuve que le
	// préfixe a bien été appliqué au document (et pas seulement au routeur).
	if _, ok := paths["/pages/probe"]; ok {
		t.Error("path LOCAL /pages/probe présent dans le document — préfixe parent NON appliqué au doc")
	}
	if _, ok := paths["/pages/submit"]; ok {
		t.Error("path LOCAL /pages/submit présent dans le document — préfixe parent NON appliqué au doc")
	}

	// operationIDs uniques inter-groupes (sinon AddOperation aurait paniqué).
	ids := collectOperationIDs(paths)
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("operationID dupliqué dans le document fusionné : %q", id)
		}
		seen[id] = true
	}
	if len(ids) < 3 {
		t.Errorf("operations fusionnées = %d, want >= 3 (2 groupes)", len(ids))
	}
}

// ---------------------------------------------------------------------------
// (c) Registre de schémas PARTAGÉ : type commun enregistré une fois, référencé
// par les deux groupes ; pas d'écrasement homonyme.
// ---------------------------------------------------------------------------

func TestSharedDoc_SchemaRegistryMergedNoClobber(t *testing.T) {
	_, cfg := buildMergedRouter()
	schemas := cfg.OpenAPI.Components.Schemas.Map()

	for _, name := range []string{"AlphaResponse", "BravoResponse", "CommonMeta", "SubmitBody"} {
		if schemas[name] == nil {
			t.Errorf("schéma %q absent du registre partagé (DTO d'un des deux groupes)", name)
		}
	}

	// CommonMeta référencé par les DTOs des DEUX groupes → même $ref, une seule
	// définition (preuve du registre partagé + dé-duplication, pas d'écrasement).
	assertMetaRef(t, schemas["AlphaResponse"], "AlphaResponse")
	assertMetaRef(t, schemas["BravoResponse"], "BravoResponse")

	// La définition partagée est cohérente (ses propres champs préservés).
	cm := schemas["CommonMeta"]
	if cm == nil || cm.Properties["title_slug"] == nil || cm.Properties["count"] == nil {
		t.Errorf("CommonMeta partagé incohérent (champs perdus) : %+v", cm)
	}
}

func assertMetaRef(t *testing.T, s *huma.Schema, owner string) {
	t.Helper()
	if s == nil {
		t.Errorf("%s: schéma nil", owner)
		return
	}
	meta := s.Properties["meta"]
	if meta == nil {
		t.Errorf("%s: propriété meta absente", owner)
		return
	}
	if !strings.HasSuffix(meta.Ref, "/CommonMeta") {
		t.Errorf("%s.meta.$ref = %q, want .../CommonMeta (registre non partagé ?)", owner, meta.Ref)
	}
}

// ---------------------------------------------------------------------------
// Helpers.
// ---------------------------------------------------------------------------

func doJSON(t *testing.T, r http.Handler, method, path string, body io.Reader, wantStatus int, out any) {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s: code = %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: JSON invalide: %v — corps: %s", method, path, err, rec.Body.String())
		}
	}
}

func collectOperationIDs(paths map[string]*huma.PathItem) []string {
	var ids []string
	for _, pi := range paths {
		for _, op := range []*huma.Operation{pi.Get, pi.Post, pi.Put, pi.Patch, pi.Delete} {
			if op != nil && op.OperationID != "" {
				ids = append(ids, op.OperationID)
			}
		}
	}
	return ids
}
