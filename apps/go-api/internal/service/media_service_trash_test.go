package service

// media_service_trash_test.go — DeleteMedia avec la Corbeille : ce qui part à la Corbeille, ce qui
// est effacé définitivement, et surtout ce qu'un échec ne fait JAMAIS (effacer l'original).
// Faux ports uniquement : aucun fichier réel n'est envoyé à la Corbeille.

import (
	"context"
	"errors"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
)

type fauxRemover struct {
	appels [][]string
	err    error
}

func (f *fauxRemover) RemoveMediaFiles(_ context.Context, paths []string) (int, error) {
	f.appels = append(f.appels, slices.Clone(paths))
	return len(paths), f.err
}

func (f *fauxRemover) tous() []string {
	var out []string
	for _, a := range f.appels {
		out = append(out, a...)
	}
	return out
}

// fauxTrasher : issue par chemin ; un chemin absent de la table part à la Corbeille.
type fauxTrasher struct {
	appels  []string
	horsBac map[string]bool  // support sans Corbeille : (false, nil)
	echecs  map[string]error // échec de l'envoi
}

func (f *fauxTrasher) TrashMediaFile(_ context.Context, p string) (bool, error) {
	f.appels = append(f.appels, p)
	if err := f.echecs[p]; err != nil {
		return false, err
	}
	return !f.horsBac[p], nil
}

// videoConvertie : original déposé, version de l'app et miniature.
func videoConvertie() domain.MediaDeletionTarget {
	return domain.MediaDeletionTarget{
		FilePath:      "JGtm/hls/clip/master.m3u8",
		HLSPath:       "JGtm/hls/clip/master.m3u8",
		SourcePath:    "JGtm/clip.mkv",
		ThumbnailPath: "JGtm/thumbs/clip.webp",
	}
}

func serviceAvec(remover *fauxRemover, trasher *fauxTrasher) *MediaService {
	s := &MediaService{fileRemover: remover}
	if trasher != nil {
		s.fileTrasher = trasher
	}
	return s
}

// retirer enchaîne les deux temps de DeleteMedia autour du masquage en base : l'envoi à la Corbeille,
// puis le retrait de ce qui reste. Sur erreur de la Corbeille, rien d'autre n'est retiré.
func retirer(s *MediaService, target domain.MediaDeletionTarget) (int, bool, error) {
	d, err := s.trashFirst(context.Background(), target)
	if err != nil {
		return d.trashedCount, d.trashedCount > 0, err
	}
	return s.removeRemaining(context.Background(), target, d)
}

func TestRemoveMediaFiles_CorbeilleReussie(t *testing.T) {
	rem, tr := &fauxRemover{}, &fauxTrasher{}
	removed, trashed, err := retirer(serviceAvec(rem, tr), videoConvertie())
	if err != nil || !trashed {
		t.Fatalf("removed=%d trashed=%v err=%v ; attendu trashed=true sans erreur", removed, trashed, err)
	}
	if !slices.Equal(tr.appels, []string{"JGtm/clip.mkv"}) {
		t.Errorf("originaux envoyés = %v", tr.appels)
	}
	// Miniature et version convertie sont effacées définitivement ; l'original ne l'est PAS.
	got := rem.tous()
	slices.Sort(got)
	want := []string{"JGtm/hls/clip/master.m3u8", "JGtm/thumbs/clip.webp"}
	if !slices.Equal(got, want) {
		t.Errorf("suppressions définitives = %v, attendu %v", got, want)
	}
	if removed != 3 {
		t.Errorf("fichiers retirés = %d, attendu 3 (original à la Corbeille + 2)", removed)
	}
}

func TestRemoveMediaFiles_SupportSansCorbeille(t *testing.T) {
	rem := &fauxRemover{}
	tr := &fauxTrasher{horsBac: map[string]bool{"JGtm/clip.mkv": true}}
	_, trashed, err := retirer(serviceAvec(rem, tr), videoConvertie())
	if err != nil || trashed {
		t.Fatalf("trashed=%v err=%v ; attendu trashed=false sans erreur", trashed, err)
	}
	if !slices.Contains(rem.tous(), "JGtm/clip.mkv") {
		t.Errorf("l'original sans Corbeille doit être effacé définitivement : %v", rem.tous())
	}
}

// L'échec de l'envoi : erreur remontée, AUCUNE suppression définitive, pas même de la miniature.
func TestRemoveMediaFiles_EchecNeSupprimeRien(t *testing.T) {
	rem := &fauxRemover{}
	tr := &fauxTrasher{echecs: map[string]error{"JGtm/clip.mkv": errors.New("fichier verrouillé")}}
	_, trashed, err := retirer(serviceAvec(rem, tr), videoConvertie())
	if err == nil || trashed {
		t.Fatalf("trashed=%v err=%v ; attendu une erreur", trashed, err)
	}
	if len(rem.appels) != 0 {
		t.Errorf("suppression définitive après un échec de la Corbeille : %v", rem.appels)
	}
}

// Deux originaux : le premier part à la Corbeille, le second échoue ; le troisième chemin n'est pas
// touché et rien n'est effacé définitivement.
func TestRemoveMediaFiles_EchecApresUnSucces(t *testing.T) {
	rem := &fauxRemover{}
	tr := &fauxTrasher{echecs: map[string]error{"JGtm/copie.mp4": errors.New("refusé")}}
	cible := domain.MediaDeletionTarget{
		FilePath: "JGtm/copie.mp4", SourcePath: "JGtm/clip.mp4", ThumbnailPath: "JGtm/thumbs/clip.webp",
	}
	_, trashed, err := retirer(serviceAvec(rem, tr), cible)
	if err == nil {
		t.Fatal("attendu une erreur")
	}
	if !trashed {
		t.Error("le premier original est dans la Corbeille : trashed doit le dire")
	}
	if len(rem.appels) != 0 {
		t.Errorf("suppression définitive après un échec : %v", rem.appels)
	}
}

// Sans trasher (instance hébergée) : comportement d'avant, tous les chemins stockés, dans l'ordre.
func TestRemoveMediaFiles_SansTrasherInchange(t *testing.T) {
	rem := &fauxRemover{}
	cible := videoConvertie()
	removed, trashed, err := retirer(serviceAvec(rem, nil), cible)
	if err != nil || trashed {
		t.Fatalf("trashed=%v err=%v", trashed, err)
	}
	if len(rem.appels) != 1 || !slices.Equal(rem.appels[0], cible.StoredPaths()) {
		t.Errorf("appels = %v, attendu un seul appel avec %v", rem.appels, cible.StoredPaths())
	}
	if removed != len(cible.StoredPaths()) {
		t.Errorf("removed = %d", removed)
	}
}

// Une erreur de la suppression définitive est remontée (la Corbeille n'y change rien).
func TestRemoveMediaFiles_ErreurDeSuppressionDefinitive(t *testing.T) {
	rem := &fauxRemover{err: errors.New("disque en lecture seule")}
	_, _, err := retirer(serviceAvec(rem, &fauxTrasher{}), videoConvertie())
	if err == nil {
		t.Fatal("attendu l'erreur de la suppression définitive")
	}
}
