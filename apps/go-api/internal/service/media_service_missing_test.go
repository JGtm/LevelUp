package service

// media_service_missing_test.go — autorisation, statuts et cohérence des actions sur les médias
// introuvables, avec de faux ports : un refus ne doit déclencher aucune écriture ni aucun retrait
// de fichier. L'écriture réelle (transaction, CHECKPOINT, fichiers) est dans
// media_service_missing_cgo_test.go.

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/port"
)

// fauxMissingRepo : un dépôt qui ne sert que les médias introuvables.
type fauxMissingRepo struct {
	port.MediaRepository
	player     string
	missing    map[string]domain.MediaFileRef
	present    map[string]domain.MediaFileRef
	loadable   map[string]domain.MediaDeletionTarget
	candidates []domain.MediaMergeCandidate
	limitSeen  int
	merges     int
}

func (r *fauxMissingRepo) CurrentPlayerSlug() string { return r.player }

func (r *fauxMissingRepo) ListMissingMedia(context.Context) ([]domain.MediaMissingItem, error) {
	return []domain.MediaMissingItem{{FilePath: "JGtm/clip.mp4"}}, nil
}

func (r *fauxMissingRepo) LoadMissingTarget(_ context.Context, p string) (domain.MediaFileRef, error) {
	if ref, ok := r.missing[p]; ok {
		return ref, nil
	}
	return domain.MediaFileRef{}, domain.ErrNotFound("media", p)
}

func (r *fauxMissingRepo) LoadMergeTarget(_ context.Context, p string) (domain.MediaFileRef, error) {
	if ref, ok := r.present[p]; ok {
		return ref, nil
	}
	return domain.MediaFileRef{}, domain.ErrNotFound("media", p)
}

func (r *fauxMissingRepo) ListMergeCandidates(_ context.Context, _ domain.MediaFileRef, limit int) ([]domain.MediaMergeCandidate, error) {
	r.limitSeen = limit
	return r.candidates, nil
}

func (r *fauxMissingRepo) LoadMediaForDeletion(_ context.Context, p string) (domain.MediaDeletionTarget, error) {
	if t, ok := r.loadable[p]; ok {
		return t, nil
	}
	return domain.MediaDeletionTarget{}, domain.ErrNotFound("media", p)
}

func ref(path, owner, kind string) domain.MediaFileRef {
	return domain.MediaFileRef{
		MediaDeletionTarget: domain.MediaDeletionTarget{ID: 1, OwnerSlug: owner, FilePath: path},
		Kind:                kind,
	}
}

func apiCode(err error) string {
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// serviceOver monte un service dont le writer échoue s'il est demandé : un refus ne l'atteint pas.
func serviceOver(repo port.MediaRepository, rem *fauxRemover) *MediaService {
	return NewMediaService(repo, "",
		WithMediaTxOpener(failingTxOpener{err: errors.New("le writer ne doit pas être demandé")}),
		WithMediaFileRemover(rem))
}

func enforced(slug string) domain.MediaRequester {
	return domain.MediaRequester{RequesterSlug: slug, AuthEnforced: true}
}

func TestMergeMissingMedia_RefusalsWriteAndRemoveNothing(t *testing.T) {
	admin := domain.MediaRequester{RequesterIsAdmin: true, AuthEnforced: true}
	for _, tc := range []struct {
		name     string
		who      domain.MediaRequester
		path     string
		target   string
		missing  map[string]domain.MediaFileRef
		present  map[string]domain.MediaFileRef
		wantCode string
	}{
		{
			name: "cible identique", who: enforced("JGtm"), path: "JGtm/a.mp4", target: "JGtm/a.mp4",
			wantCode: "bad_request",
		},
		{
			name: "chemin vide", who: enforced("JGtm"), path: "JGtm/a.mp4", target: "",
			wantCode: "bad_request",
		},
		{
			name: "la fiche n'est pas introuvable", who: enforced("JGtm"), path: "JGtm/a.mp4", target: "JGtm/b.mp4",
			present:  map[string]domain.MediaFileRef{"JGtm/b.mp4": ref("JGtm/b.mp4", "JGtm", "video")},
			wantCode: "not_found",
		},
		{
			name: "la cible n'est pas présente", who: enforced("JGtm"), path: "JGtm/a.mp4", target: "JGtm/b.mp4",
			missing:  map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")},
			wantCode: "not_found",
		},
		{
			name: "demandeur étranger à la fiche introuvable", who: enforced("Mallory"), path: "JGtm/a.mp4", target: "JGtm/b.mp4",
			missing:  map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")},
			present:  map[string]domain.MediaFileRef{"JGtm/b.mp4": ref("JGtm/b.mp4", "JGtm", "video")},
			wantCode: "forbidden",
		},
		{
			name: "demandeur étranger à la cible", who: enforced("JGtm"), path: "JGtm/a.mp4", target: "Autre/b.mp4",
			missing:  map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")},
			present:  map[string]domain.MediaFileRef{"Autre/b.mp4": ref("Autre/b.mp4", "Autre", "video")},
			wantCode: "forbidden",
		},
		{
			name: "propriétaires différents (admin)", who: admin, path: "JGtm/a.mp4", target: "Autre/b.mp4",
			missing:  map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")},
			present:  map[string]domain.MediaFileRef{"Autre/b.mp4": ref("Autre/b.mp4", "Autre", "video")},
			wantCode: "bad_request",
		},
		{
			name: "natures différentes", who: enforced("JGtm"), path: "JGtm/a.mp4", target: "JGtm/b.png",
			missing:  map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")},
			present:  map[string]domain.MediaFileRef{"JGtm/b.png": ref("JGtm/b.png", "JGtm", "image")},
			wantCode: "bad_request",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fauxMissingRepo{player: "JGtm", missing: tc.missing, present: tc.present}
			rem := &fauxRemover{}
			_, err := serviceOver(repo, rem).MergeMissingMedia(context.Background(), domain.MediaMergeRequest{
				MediaRequester: tc.who, FilePath: tc.path, TargetFilePath: tc.target,
			})
			if got := apiCode(err); got != tc.wantCode {
				t.Fatalf("err = %v (code %q), attendu %q", err, got, tc.wantCode)
			}
			if repo.merges != 0 || len(rem.appels) != 0 {
				t.Errorf("refus mais écriture (%d fusion) ou retrait de fichiers (%v)", repo.merges, rem.appels)
			}
		})
	}
}

func TestMergeMissingMedia_RequiresWriterAndRemover(t *testing.T) {
	repo := &fauxMissingRepo{player: "JGtm"}
	req := domain.MediaMergeRequest{MediaRequester: enforced("JGtm"), FilePath: "JGtm/a.mp4", TargetFilePath: "JGtm/b.mp4"}
	for name, svc := range map[string]*MediaService{
		"sans retrait disque": NewMediaService(repo, "", WithMediaTxOpener(failingTxOpener{})),
		"sans writer":         NewMediaService(repo, "", WithMediaFileRemover(&fauxRemover{})),
	} {
		if _, err := svc.MergeMissingMedia(context.Background(), req); apiCode(err) != "internal_error" {
			t.Errorf("%s : err = %v, attendu internal_error avant toute lecture", name, err)
		}
	}
}

func TestForgetMissingMedia_RefusalsRemoveNothing(t *testing.T) {
	repo := &fauxMissingRepo{player: "JGtm",
		missing: map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")}}
	for name, tc := range map[string]struct {
		who  domain.MediaRequester
		path string
		code string
	}{
		"étranger":                    {enforced("Mallory"), "JGtm/a.mp4", "forbidden"},
		"fiche non introuvable (404)": {enforced("JGtm"), "JGtm/present.mp4", "not_found"},
		"chemin vide":                 {enforced("JGtm"), "", "bad_request"},
	} {
		rem := &fauxRemover{}
		_, err := serviceOver(repo, rem).ForgetMissingMedia(context.Background(),
			domain.MediaForgetRequest{MediaRequester: tc.who, FilePath: tc.path})
		if apiCode(err) != tc.code {
			t.Errorf("%s : err = %v, attendu %q", name, err, tc.code)
		}
		if len(rem.appels) != 0 {
			t.Errorf("%s : fichiers retirés malgré le refus : %v", name, rem.appels)
		}
	}
}

func TestListMissingMedia_AuthorizationAndRootState(t *testing.T) {
	const player = "ListeJoueur"
	repo := &fauxMissingRepo{player: player}
	svc := serviceOver(repo, &fauxRemover{})
	ops.RecordRootState("halo_infinite", player, ops.RootState{Unavailable: true, BulkPending: 4})

	resp, err := svc.ListMissingMedia(context.Background(),
		domain.MediaMissingListRequest{MediaRequester: enforced(player)})
	if err != nil {
		t.Fatalf("propriétaire : %v", err)
	}
	if !resp.RootUnavailable || resp.BulkMissingPending != 4 || len(resp.Items) != 1 {
		t.Errorf("réponse = %+v, attendu l'état enregistré et une fiche", resp)
	}
	for name, who := range map[string]domain.MediaRequester{
		"admin":                      {RequesterIsAdmin: true, AuthEnforced: true},
		"mode sans authentification": {},
	} {
		if _, err := svc.ListMissingMedia(context.Background(), domain.MediaMissingListRequest{MediaRequester: who}); err != nil {
			t.Errorf("%s : %v", name, err)
		}
	}
	// Un co-membre de groupe lit le profil mais ne voit pas ses fichiers introuvables.
	if _, err := svc.ListMissingMedia(context.Background(),
		domain.MediaMissingListRequest{MediaRequester: enforced("Mallory")}); apiCode(err) != "forbidden" {
		t.Errorf("visiteur : err = %v, attendu forbidden", err)
	}
}

func TestListMergeCandidates_AuthorizationStatusAndLimit(t *testing.T) {
	repo := &fauxMissingRepo{player: "JGtm",
		missing:    map[string]domain.MediaFileRef{"JGtm/a.mp4": ref("JGtm/a.mp4", "JGtm", "video")},
		candidates: []domain.MediaMergeCandidate{{FilePath: "JGtm/b.mp4"}}}
	svc := serviceOver(repo, &fauxRemover{})
	ctx := context.Background()

	resp, err := svc.ListMergeCandidates(ctx, domain.MediaMergeCandidatesRequest{MediaRequester: enforced("JGtm"), FilePath: "JGtm/a.mp4"})
	if err != nil || len(resp.Candidates) != 1 || repo.limitSeen != 50 {
		t.Fatalf("resp=%+v err=%v limite=%d ; attendu un candidat, limite 50", resp, err, repo.limitSeen)
	}
	if _, err := svc.ListMergeCandidates(ctx, domain.MediaMergeCandidatesRequest{MediaRequester: enforced("Mallory"), FilePath: "JGtm/a.mp4"}); apiCode(err) != "forbidden" {
		t.Errorf("visiteur : err = %v, attendu forbidden", err)
	}
	if _, err := svc.ListMergeCandidates(ctx, domain.MediaMergeCandidatesRequest{MediaRequester: enforced("JGtm"), FilePath: "JGtm/inconnu.mp4"}); apiCode(err) != "not_found" {
		t.Errorf("fiche non introuvable : err = %v, attendu not_found", err)
	}
}

func TestConfirmBulkMissing_OnlyWhenPendingAndOnlyForTheOwner(t *testing.T) {
	const player = "ConfirmJoueur"
	repo := &fauxMissingRepo{player: player}
	svc := serviceOver(repo, &fauxRemover{})
	ctx := context.Background()

	resp, err := svc.ConfirmBulkMissing(ctx, domain.MediaConfirmBulkRequest{MediaRequester: enforced(player)})
	if err != nil || resp.Confirmed {
		t.Fatalf("sans disparition en attente : resp=%+v err=%v, attendu Confirmed=false", resp, err)
	}
	if ops.ConsumeBulkConfirmation("halo_infinite", player) {
		t.Error("une confirmation sans objet a été enregistrée")
	}

	ops.RecordRootState("halo_infinite", player, ops.RootState{BulkPending: 3})
	if _, err := svc.ConfirmBulkMissing(ctx, domain.MediaConfirmBulkRequest{MediaRequester: enforced("Mallory")}); apiCode(err) != "forbidden" {
		t.Fatalf("visiteur : err = %v, attendu forbidden", err)
	}
	if ops.ConsumeBulkConfirmation("halo_infinite", player) {
		t.Error("un visiteur a confirmé la disparition d'un autre joueur")
	}

	resp, err = svc.ConfirmBulkMissing(ctx, domain.MediaConfirmBulkRequest{MediaRequester: enforced(player)})
	if err != nil || !resp.Confirmed || resp.BulkMissingPending != 3 {
		t.Fatalf("resp=%+v err=%v, attendu Confirmed avec 3 en attente", resp, err)
	}
	if !ops.ConsumeBulkConfirmation("halo_infinite", player) {
		t.Error("la confirmation n'a pas été enregistrée pour la passe suivante")
	}
}

type fauxRevealer struct{ opened []string }

func (f *fauxRevealer) Reveal(_ context.Context, abs string) error {
	f.opened = append(f.opened, abs)
	return nil
}

func TestRevealMedia_SourceThenFilePath_FromTheRowNotTheClient(t *testing.T) {
	base := t.TempDir()
	repo := &fauxMissingRepo{player: "JGtm", loadable: map[string]domain.MediaDeletionTarget{
		"JGtm/hls/clip/master.m3u8": {OwnerSlug: "JGtm", FilePath: "JGtm/hls/clip/master.m3u8", SourcePath: "JGtm/clip.mkv"},
		"JGtm/plain.mp4":            {OwnerSlug: "JGtm", FilePath: "JGtm/plain.mp4"},
	}}
	rev := &fauxRevealer{}
	svc := NewMediaService(repo, "", WithMediaFileRevealer(rev, func() string { return base }))
	who := enforced("JGtm")

	for _, p := range []string{"JGtm/hls/clip/master.m3u8", "JGtm/plain.mp4"} {
		if resp, err := svc.RevealMedia(context.Background(), domain.MediaRevealRequest{MediaRequester: who, FilePath: p}); err != nil || !resp.Revealed {
			t.Fatalf("%s : resp=%+v err=%v", p, resp, err)
		}
	}
	want := []string{filepath.Join(base, "JGtm", "clip.mkv"), filepath.Join(base, "JGtm", "plain.mp4")}
	if len(rev.opened) != 2 || rev.opened[0] != want[0] || rev.opened[1] != want[1] {
		t.Errorf("ouvert = %v, attendu %v (source d'abord, sinon file_path)", rev.opened, want)
	}
}

func TestRevealMedia_RefusalsOpenNothing(t *testing.T) {
	repo := &fauxMissingRepo{player: "JGtm", loadable: map[string]domain.MediaDeletionTarget{
		"JGtm/clip.mp4": {OwnerSlug: "JGtm", FilePath: "JGtm/clip.mp4"},
	}}
	rev := &fauxRevealer{}
	svc := NewMediaService(repo, "", WithMediaFileRevealer(rev, func() string { return t.TempDir() }))
	ctx := context.Background()

	if _, err := svc.RevealMedia(ctx, domain.MediaRevealRequest{MediaRequester: enforced("Mallory"), FilePath: "JGtm/clip.mp4"}); apiCode(err) != "forbidden" {
		t.Errorf("visiteur : err = %v, attendu forbidden", err)
	}
	if _, err := svc.RevealMedia(ctx, domain.MediaRevealRequest{MediaRequester: enforced("JGtm"), FilePath: "JGtm/inconnu.mp4"}); apiCode(err) != "not_found" {
		t.Errorf("fiche inconnue : err = %v, attendu not_found", err)
	}
	if len(rev.opened) != 0 {
		t.Errorf("explorateur ouvert malgré le refus : %v", rev.opened)
	}
}

// Sans la capacité du déploiement, la route répond comme une route inexistante, pour tout le monde.
func TestRevealMedia_WithoutCapabilityIsNotFound(t *testing.T) {
	svc := NewMediaService(&fauxMissingRepo{player: "JGtm"}, "")
	_, err := svc.RevealMedia(context.Background(),
		domain.MediaRevealRequest{MediaRequester: domain.MediaRequester{RequesterIsAdmin: true}, FilePath: "JGtm/clip.mp4"})
	if apiCode(err) != "not_found" {
		t.Errorf("err = %v, attendu not_found", err)
	}
}

// failingTxOpener : un ouvreur de transactions dont l'ouverture échoue (ou rend l'erreur donnée) ; il prouve
// qu'un refus ne demande pas le writer.
type failingTxOpener struct{ err error }

func (o failingTxOpener) BeginMediaTx(context.Context, string) (port.MediaTx, error) {
	if o.err != nil {
		return nil, o.err
	}
	return nil, errors.New("transaction indisponible")
}
