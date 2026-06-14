package assets

import (
	"context"
	"errors"
	"testing"
)

func TestDiscoveryMapFetcher_Supports(t *testing.T) {
	f := NewDiscoveryMapFetcher(nil)
	if !f.Supports(KindMapImage) {
		t.Error("doit supporter KindMapImage")
	}
	if f.Supports(KindMedalImage) {
		t.Error("ne doit PAS supporter KindMedalImage")
	}
}

func TestDiscoveryMapFetcher_Fetch_ResolvesURL(t *testing.T) {
	var gotTitle, gotMap, gotVer string
	f := NewDiscoveryMapFetcher(func(_ context.Context, titleID, mapID, versionID string) (string, error) {
		gotTitle, gotMap, gotVer = titleID, mapID, versionID
		return "https://blobs-infiniteugc.svc.halowaypoint.com/.../images/thumbnail.jpg", nil
	})
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "map-uuid", Variant: "ver-uuid"}
	p, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	u, ok := p.(URLPayload)
	if !ok || u.URL == "" {
		t.Fatalf("payload = %#v, want URLPayload non vide", p)
	}
	if u.ContentType != MimeImageJPEG {
		t.Errorf("ContentType = %q, want %q", u.ContentType, MimeImageJPEG)
	}
	if gotTitle != "hi" || gotMap != "map-uuid" || gotVer != "ver-uuid" {
		t.Errorf("closure args = (%q,%q,%q), want (hi,map-uuid,ver-uuid)", gotTitle, gotMap, gotVer)
	}
}

func TestDiscoveryMapFetcher_Fetch_NoVersion_NotFound(t *testing.T) {
	called := false
	f := NewDiscoveryMapFetcher(func(_ context.Context, _, _, _ string) (string, error) {
		called = true
		return "x", nil
	})
	// Variant (version) vide → DiscoveryUGC 404 garanti → on n'appelle même pas.
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "map-uuid"}
	if _, err := f.Fetch(context.Background(), ref); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if called {
		t.Error("la closure ne doit pas être appelée sans version_id")
	}
}

func TestDiscoveryMapFetcher_Fetch_NilFetcher_NotFound(t *testing.T) {
	f := NewDiscoveryMapFetcher(nil)
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "map-uuid", Variant: "ver"}
	if _, err := f.Fetch(context.Background(), ref); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound (fetchURL nil)", err)
	}
}

func TestDiscoveryMapFetcher_Fetch_UpstreamError(t *testing.T) {
	f := NewDiscoveryMapFetcher(func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("boom")
	})
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "map-uuid", Variant: "ver"}
	if _, err := f.Fetch(context.Background(), ref); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("err = %v, want ErrUpstreamUnavailable", err)
	}
}
