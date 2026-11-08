package handlers_test

// media_missing_test.go — les routes des médias introuvables : traduction des chemins (URL servable
// vers clé de stockage et retour), identité du demandeur, codes d'erreur, et la passe lancée en
// arrière-plan après la confirmation d'une disparition massive. Le service est une doublure ; la
// frontière du visiteur et du co-membre (middleware d'ownership + service réel) est testée dans
// internal/api/media_missing_visitors_test.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

const missingURLPrefix = "/api/v1/players/" + testPlayerSlug + "/media/files/"

// missingMock : le service des médias introuvables, qui note la dernière requête de chaque route.
type missingMock struct {
	mockMediaService
	err error

	listReq    domain.MediaMissingListRequest
	candReq    domain.MediaMergeCandidatesRequest
	mergeReq   domain.MediaMergeRequest
	forgetReq  domain.MediaForgetRequest
	confirmReq domain.MediaConfirmBulkRequest
	revealReq  domain.MediaRevealRequest
	confirmed  bool
}

func (m *missingMock) ListMissingMedia(_ context.Context, req domain.MediaMissingListRequest) (*domain.MediaMissingResponse, error) {
	m.listReq = req
	thumb := "JGtm/thumbs/clip.webp"
	return &domain.MediaMissingResponse{
		Items:              []domain.MediaMissingItem{{FilePath: "JGtm/clip.mp4", Basename: "clip.mp4", Kind: "video", ThumbnailPath: &thumb}},
		RootUnavailable:    true,
		BulkMissingPending: 7,
	}, m.err
}

func (m *missingMock) ListMergeCandidates(_ context.Context, req domain.MediaMergeCandidatesRequest) (*domain.MediaMergeCandidatesResponse, error) {
	m.candReq = req
	return &domain.MediaMergeCandidatesResponse{
		FilePath:   req.FilePath,
		Candidates: []domain.MediaMergeCandidate{{FilePath: "JGtm/new.mp4", Basename: "new.mp4", Kind: "video"}},
	}, m.err
}

func (m *missingMock) MergeMissingMedia(_ context.Context, req domain.MediaMergeRequest) (*domain.MediaMergeResponse, error) {
	m.mergeReq = req
	if m.err != nil {
		return nil, m.err
	}
	return &domain.MediaMergeResponse{FilePath: req.FilePath, TargetFilePath: req.TargetFilePath, Merged: true}, nil
}

func (m *missingMock) ForgetMissingMedia(_ context.Context, req domain.MediaForgetRequest) (*domain.MediaForgetResponse, error) {
	m.forgetReq = req
	if m.err != nil {
		return nil, m.err
	}
	return &domain.MediaForgetResponse{FilePath: req.FilePath, Forgotten: true}, nil
}

func (m *missingMock) ConfirmBulkMissing(_ context.Context, req domain.MediaConfirmBulkRequest) (*domain.MediaConfirmBulkResponse, error) {
	m.confirmReq = req
	if m.err != nil {
		return nil, m.err
	}
	return &domain.MediaConfirmBulkResponse{Confirmed: m.confirmed, BulkMissingPending: 7}, nil
}

func (m *missingMock) RevealMedia(_ context.Context, req domain.MediaRevealRequest) (*domain.MediaRevealResponse, error) {
	m.revealReq = req
	if m.err != nil {
		return nil, m.err
	}
	return &domain.MediaRevealResponse{Revealed: true}, nil
}

// newMissingRouter monte le handler média, authentification appliquée, sur le service doublé.
func newMissingRouter(mock port.MediaService, rescan handlers.MediaRescan) *chi.Mux {
	factory := func(_ context.Context, slug string) (port.MediaService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	h := handlers.NewMediaHandler(factory, nil, "").WithAuthEnforced(true).
		WithRequester(sessionPlayerAsOwnedProfile).WithUserLookup(mediaAccounts{"Chocoboflor": domain.RoleAdmin}).
		WithAuthorsContext(func(_ context.Context, slug string) (string, string, error) {
			return "halo_5", "Gamertag-" + slug, nil
		}, nil).
		WithMediaRescan(rescan)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	return r
}

func callMissing(r http.Handler, method, route string, body any, sess *domain.SessionData) *httptest.ResponseRecorder {
	var rd *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, "/players/"+testPlayerSlug+route, rd)
	req.Header.Set("Content-Type", "application/json")
	if sess != nil {
		req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMissingList_ConvertsPathsAndPassesRequester(t *testing.T) {
	mock := &missingMock{}
	r := newMissingRouter(mock, nil)

	w := callMissing(r, http.MethodGet, "/media/missing", nil, sessionFor("JGtm", "user"))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	var resp domain.MediaMissingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].FilePath != missingURLPrefix+"JGtm/clip.mp4" {
		t.Errorf("fiche = %+v, attendu l'URL servable", resp.Items)
	}
	if resp.Items[0].ThumbnailPath == nil || *resp.Items[0].ThumbnailPath != missingURLPrefix+"JGtm/thumbs/clip.webp" {
		t.Errorf("miniature = %v, attendu l'URL servable", resp.Items[0].ThumbnailPath)
	}
	if !resp.RootUnavailable || resp.BulkMissingPending != 7 {
		t.Errorf("état du dossier = %+v", resp)
	}
	if mock.listReq.RequesterSlug != "JGtm" || !mock.listReq.AuthEnforced || mock.listReq.RequesterIsAdmin {
		t.Errorf("demandeur = %+v, attendu JGtm, auth appliquée, non admin", mock.listReq.MediaRequester)
	}
}

func TestMissingCandidates_StoredPathInURLPathOut(t *testing.T) {
	mock := &missingMock{}
	r := newMissingRouter(mock, nil)
	const raw = missingURLPrefix + "JGtm/clip.mp4"

	w := callMissing(r, http.MethodGet, "/media/missing/candidates?file_path="+raw, nil, sessionFor("JGtm", "user"))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	if mock.candReq.FilePath != "JGtm/clip.mp4" {
		t.Errorf("chemin reçu par le service = %q, attendu la clé de stockage", mock.candReq.FilePath)
	}
	var resp domain.MediaMergeCandidatesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.FilePath != raw || resp.Candidates[0].FilePath != missingURLPrefix+"JGtm/new.mp4" {
		t.Errorf("réponse = %+v, attendu l'écho de la requête et des URL servables", resp)
	}
	if w := callMissing(r, http.MethodGet, "/media/missing/candidates", nil, sessionFor("JGtm", "user")); w.Code != http.StatusBadRequest {
		t.Errorf("sans file_path : statut %d, attendu 400", w.Code)
	}
}

func TestMissingMerge_ConvertsBothPathsAndEchoesThem(t *testing.T) {
	mock := &missingMock{}
	r := newMissingRouter(mock, nil)
	body := map[string]string{
		"file_path":        missingURLPrefix + "JGtm/clip.mp4",
		"target_file_path": missingURLPrefix + "JGtm/new.mp4",
	}

	w := callMissing(r, http.MethodPost, "/media/missing/merge", body, sessionFor("JGtm", "user"))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	if mock.mergeReq.FilePath != "JGtm/clip.mp4" || mock.mergeReq.TargetFilePath != "JGtm/new.mp4" {
		t.Errorf("requête du service = %+v, attendu les clés de stockage", mock.mergeReq)
	}
	if mock.mergeReq.RequesterSlug != "JGtm" || !mock.mergeReq.AuthEnforced {
		t.Errorf("demandeur = %+v", mock.mergeReq.MediaRequester)
	}
	var resp domain.MediaMergeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.FilePath != body["file_path"] || resp.TargetFilePath != body["target_file_path"] || !resp.Merged {
		t.Errorf("réponse = %+v, attendu l'écho des chemins reçus", resp)
	}
}

// L'identité ne vient jamais du corps : un RequesterSlug glissé dans le JSON est ignoré.
func TestMissingMerge_RequesterNeverReadFromBody(t *testing.T) {
	mock := &missingMock{}
	r := newMissingRouter(mock, nil)
	raw := []byte(`{"file_path":"JGtm/clip.mp4","target_file_path":"JGtm/new.mp4","RequesterSlug":"Chocoboflor","RequesterIsAdmin":true,"AuthEnforced":false}`)
	req := httptest.NewRequest(http.MethodPost, "/players/"+testPlayerSlug+"/media/missing/merge", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.InjectSession(req.Context(), sessionFor("Mallory", "user")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	got := mock.mergeReq.MediaRequester
	if got.RequesterSlug != "Mallory" || got.RequesterIsAdmin || !got.AuthEnforced {
		t.Errorf("demandeur = %+v, attendu Mallory, non admin, auth appliquée", got)
	}
}

func TestMissingForgetAndReveal_ForwardStoredPath(t *testing.T) {
	mock := &missingMock{}
	r := newMissingRouter(mock, nil)
	body := map[string]string{"file_path": missingURLPrefix + "JGtm/clip.mp4"}

	if w := callMissing(r, http.MethodPost, "/media/missing/forget", body, sessionFor("JGtm", "user")); w.Code != http.StatusOK {
		t.Fatalf("forget : statut %d : %s", w.Code, w.Body.String())
	}
	if mock.forgetReq.FilePath != "JGtm/clip.mp4" {
		t.Errorf("forget : chemin = %q", mock.forgetReq.FilePath)
	}
	if w := callMissing(r, http.MethodPost, "/media/reveal", body, sessionFor("JGtm", "user")); w.Code != http.StatusOK {
		t.Fatalf("reveal : statut %d : %s", w.Code, w.Body.String())
	}
	if mock.revealReq.FilePath != "JGtm/clip.mp4" {
		t.Errorf("reveal : chemin = %q", mock.revealReq.FilePath)
	}
}

func TestMissingWrites_ErrorsAreTranslated(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		sess *domain.SessionData
		want int
		code string
	}{
		{"refus du service", domain.ErrForbidden("no"), sessionFor("Mallory", "user"), http.StatusForbidden, "media_missing_forbidden"},
		{"fiche inconnue", domain.ErrNotFound("media", "x"), sessionFor("JGtm", "user"), http.StatusNotFound, "not_found"},
		{"requête incohérente", domain.ErrBadRequest("kinds"), sessionFor("JGtm", "user"), http.StatusBadRequest, "bad_request"},
		{"aucune session", nil, nil, http.StatusUnauthorized, "_requires_session"},
	} {
		for route, op := range map[string]string{
			"/media/missing/merge":        "merge",
			"/media/missing/forget":       "forget",
			"/media/missing/confirm-bulk": "confirm",
			"/media/reveal":               "reveal",
		} {
			mock := &missingMock{err: tc.err}
			w := callMissing(newMissingRouter(mock, nil), http.MethodPost, route,
				map[string]string{"file_path": "JGtm/a.mp4", "target_file_path": "JGtm/b.mp4"}, tc.sess)
			if w.Code != tc.want {
				t.Errorf("%s %s : statut %d, attendu %d (%s)", tc.name, route, w.Code, tc.want, w.Body.String())
				continue
			}
			want := tc.code
			if tc.want == http.StatusUnauthorized {
				want = op + tc.code
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(want)) {
				t.Errorf("%s %s : corps sans le code %q : %s", tc.name, route, want, w.Body.String())
			}
		}
	}
}

func TestMissingMerge_MalformedBodyIs400(t *testing.T) {
	r := newMissingRouter(&missingMock{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/players/"+testPlayerSlug+"/media/missing/merge", bytes.NewReader([]byte("{pas du json")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.InjectSession(req.Context(), sessionFor("JGtm", "user")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("statut %d, attendu 400 : %s", w.Code, w.Body.String())
	}
}

// Une confirmation effective lance la passe du joueur, détachée de la requête, sous le titre de la
// requête ; une confirmation sans objet n'en lance aucune.
func TestMissingConfirmBulk_LaunchesRescanOnlyWhenConfirmed(t *testing.T) {
	type call struct {
		gamertag string
		title    string
	}
	calls := make(chan call, 4)
	rescan := func(ctx context.Context, gamertag string) {
		calls <- call{gamertag, ctxkeys.TitleSlug(ctx)}
	}

	idle := &missingMock{confirmed: false}
	if w := callMissing(newMissingRouter(idle, rescan), http.MethodPost, "/media/missing/confirm-bulk", nil, sessionFor("JGtm", "user")); w.Code != http.StatusOK {
		t.Fatalf("sans objet : statut %d : %s", w.Code, w.Body.String())
	}
	select {
	case c := <-calls:
		t.Fatalf("passe lancée pour une confirmation sans objet : %+v", c)
	case <-time.After(150 * time.Millisecond):
	}

	live := &missingMock{confirmed: true}
	if w := callMissing(newMissingRouter(live, rescan), http.MethodPost, "/media/missing/confirm-bulk", nil, sessionFor("JGtm", "user")); w.Code != http.StatusOK {
		t.Fatalf("confirmée : statut %d : %s", w.Code, w.Body.String())
	}
	select {
	case c := <-calls:
		if c.gamertag != "Gamertag-"+testPlayerSlug || c.title != "halo_5" {
			t.Errorf("passe = %+v, attendu le joueur et le titre de la requête", c)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("aucune passe lancée après une confirmation effective")
	}
}

// Deux confirmations rapprochées ne lancent pas deux passes sur la même base.
func TestMissingConfirmBulk_NoSecondRescanWhileOneRuns(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	rescan := func(context.Context, string) {
		started <- struct{}{}
		<-release
	}
	r := newMissingRouter(&missingMock{confirmed: true}, rescan)
	for range 2 {
		if w := callMissing(r, http.MethodPost, "/media/missing/confirm-bulk", nil, sessionFor("JGtm", "user")); w.Code != http.StatusOK {
			t.Fatalf("statut %d : %s", w.Code, w.Body.String())
		}
	}
	<-started
	select {
	case <-started:
		t.Error("une seconde passe a été lancée pendant la première")
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
}
