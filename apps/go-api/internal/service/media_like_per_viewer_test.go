//go:build cgo

// media_like_per_viewer_test.go — garde-rail de l'ISOLATION DES LIKES ENTRE
// VIEWERS (lot « likes par viewer », 2026-08-04).
//
// LA CLASSE DE RÉGRESSION QUE CE FICHIER VERROUILLE. L'état du cœur a longtemps
// vécu dans media_files.liked : UNE colonne booléenne PAR MÉDIA, partagée par
// tous les joueurs. Conséquence en production : dès qu'un coéquipier likait un
// clip, le cœur s'allumait chez TOUT LE MONDE — et un unlike l'éteignait chez
// tout le monde. Aucun test ne l'attrapait parce que tous les tests de like
// n'avaient qu'un seul acteur : avec un seul viewer, un état global et un état
// par viewer sont indiscernables.
//
// D'où le protocole ci-dessous : DEUX viewers sur LA MÊME base, sur LE MÊME
// média. C'est le seul montage qui distingue les deux sémantiques.
//
// Ce que le test fixe, en plus de l'isolation : l'asymétrie du contrat servi au
// front — `liked` est PERSONNEL (le viewer courant), tandis que `like_count` /
// `total_likers` restent GLOBAUX (tous likers confondus). Un compteur qui
// deviendrait personnel afficherait « 1 » à celui qui a liké et « 0 » aux
// autres : la ligne « ♥ Alice et Bob » ne voudrait plus rien dire.
package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

const perViewerClipPath = "JGtm/clip.mp4"

// mediaServiceForViewer construit le service de galerie tel que le wiring le
// produit POUR UN VIEWER DONNÉ (cf. wire.viewerSlugFor + MediaRepo.WithViewer) :
// même base, même propriétaire de galerie, seul le viewer change.
func mediaServiceForViewer(db *duckdb.DB, viewerSlug string) *MediaService {
	pdb := &duckdb.PlayerDB{SharedSocial: db, Gamertag: "JGtm"}
	repo := duckdb.NewMediaRepo(pdb).WithViewer(viewerSlug)
	return NewMediaService(repo, "",
		WithMediaWriterAcquirer(func() (*dblease.LeasedWriter, error) {
			return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
		}))
}

// firstMediaItem retourne l'unique média de la galerie servie à ce viewer.
func firstMediaItem(t *testing.T, svc *MediaService, filters domain.MediaPageRequest) *domain.MediaItem {
	t.Helper()
	page, err := svc.GetMediaPage(context.Background(), filters)
	if err != nil {
		t.Fatalf("GetMediaPage: %v", err)
	}
	if len(page.Items.Items) == 0 {
		return nil
	}
	return &page.Items.Items[0]
}

// TestMediaLike_IsolatedBetweenViewers : le like de Chocoboflor n'allume que SON
// cœur. JGtm voit le même média non liké — mais voit bien le compteur social à 1.
func TestMediaLike_IsolatedBetweenViewers(t *testing.T) {
	db, _ := socialDBForLikeTest(t)
	ctx := context.Background()

	liker := mediaServiceForViewer(db, "Chocoboflor")
	other := mediaServiceForViewer(db, "JGtm")

	// État initial : personne n'a liké — les deux viewers voient un cœur vide.
	for name, svc := range map[string]*MediaService{"Chocoboflor": liker, "JGtm": other} {
		if item := firstMediaItem(t, svc, domain.MediaPageRequest{}); item == nil || item.Liked {
			t.Fatalf("état initial pour %s : attendu un média non liké, got %+v", name, item)
		}
	}

	// Chocoboflor like.
	resp, err := liker.SetMediaLike(ctx, domain.MediaLikeRequest{
		FilePath:      perViewerClipPath,
		LikerSlug:     "Chocoboflor",
		LikerGamertag: "Chocoboflor",
		Liked:         true,
	})
	if err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}
	if !resp.Liked || resp.LikeCount != 1 || resp.TotalLikers != 1 {
		t.Fatalf("réponse du like = %+v, want Liked=true LikeCount=1 TotalLikers=1", resp)
	}

	// LE cas discriminant : deux lectures de la MÊME base, deux viewers.
	likerItem := firstMediaItem(t, liker, domain.MediaPageRequest{})
	otherItem := firstMediaItem(t, other, domain.MediaPageRequest{})
	if likerItem == nil || otherItem == nil {
		t.Fatal("la galerie doit servir le média aux deux viewers")
	}
	if !likerItem.Liked {
		t.Error("le liker doit voir SON cœur allumé (liked=true)")
	}
	if otherItem.Liked {
		t.Error("RÉGRESSION : le like d'un autre joueur allume le cœur de JGtm — " +
			"le like est redevenu global (media_files.liked ?)")
	}

	// Le compteur, lui, reste GLOBAL : les deux viewers voient « 1 like ».
	if likerItem.LikeCount != 1 || likerItem.TotalLikers != 1 {
		t.Errorf("compteur côté liker = %d/%d, want 1/1", likerItem.LikeCount, likerItem.TotalLikers)
	}
	if otherItem.LikeCount != 1 || otherItem.TotalLikers != 1 {
		t.Errorf("compteur côté non-liker = %d/%d, want 1/1 (le compteur est social, pas personnel)",
			otherItem.LikeCount, otherItem.TotalLikers)
	}
}

// TestMediaLike_LikedOnlyFilterIsPerViewer : le filtre « favoris » de la barre
// d'outils suit le même viewer que le cœur. Sans cela, JGtm ouvrirait ses
// favoris et y trouverait les clips likés par quelqu'un d'autre.
func TestMediaLike_LikedOnlyFilterIsPerViewer(t *testing.T) {
	db, _ := socialDBForLikeTest(t)
	ctx := context.Background()

	liker := mediaServiceForViewer(db, "Chocoboflor")
	other := mediaServiceForViewer(db, "JGtm")

	if _, err := liker.SetMediaLike(ctx, domain.MediaLikeRequest{
		FilePath:      perViewerClipPath,
		LikerSlug:     "Chocoboflor",
		LikerGamertag: "Chocoboflor",
		Liked:         true,
	}); err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}

	likedOnly := domain.MediaPageRequest{LikedOnly: true}
	if item := firstMediaItem(t, liker, likedOnly); item == nil {
		t.Error("les favoris du liker doivent contenir le média qu'il vient de liker")
	}
	if item := firstMediaItem(t, other, likedOnly); item != nil {
		t.Errorf("RÉGRESSION : les favoris de JGtm contiennent %q, liké par un autre joueur",
			item.FilePath)
	}
}

// TestMediaLike_UnlikeOnlyAffectsItsOwner : l'unlike est lui aussi personnel.
// Avec la colonne globale, le unlike d'un joueur éteignait le cœur des autres —
// le pendant exact du bug d'allumage, et tout aussi invisible à un seul acteur.
func TestMediaLike_UnlikeOnlyAffectsItsOwner(t *testing.T) {
	db, _ := socialDBForLikeTest(t)
	ctx := context.Background()

	first := mediaServiceForViewer(db, "Chocoboflor")
	second := mediaServiceForViewer(db, "JGtm")

	for _, who := range []struct {
		svc  *MediaService
		slug string
	}{{first, "Chocoboflor"}, {second, "JGtm"}} {
		if _, err := who.svc.SetMediaLike(ctx, domain.MediaLikeRequest{
			FilePath:      perViewerClipPath,
			LikerSlug:     who.slug,
			LikerGamertag: who.slug,
			Liked:         true,
		}); err != nil {
			t.Fatalf("SetMediaLike(%s): %v", who.slug, err)
		}
	}

	// Chocoboflor retire son like ; JGtm garde le sien.
	if _, err := first.SetMediaLike(ctx, domain.MediaLikeRequest{
		FilePath:  perViewerClipPath,
		LikerSlug: "Chocoboflor",
		Liked:     false,
	}); err != nil {
		t.Fatalf("unlike: %v", err)
	}

	firstItem := firstMediaItem(t, first, domain.MediaPageRequest{})
	secondItem := firstMediaItem(t, second, domain.MediaPageRequest{})
	if firstItem == nil || secondItem == nil {
		t.Fatal("la galerie doit servir le média aux deux viewers")
	}
	if firstItem.Liked {
		t.Error("après son unlike, Chocoboflor doit voir un cœur vide")
	}
	if !secondItem.Liked {
		t.Error("RÉGRESSION : l'unlike d'un joueur a éteint le cœur de JGtm")
	}
	// Un seul liker restant : le compteur social suit.
	if secondItem.TotalLikers != 1 {
		t.Errorf("total_likers après un unlike sur deux likes = %d, want 1", secondItem.TotalLikers)
	}
}
