package assets

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFSStore_Path_NoVariant(t *testing.T) {
	s := NewLocalFSStore("/cache")
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "42"}
	got := s.Path(ref)
	want := filepath.Join("/cache", "medal-image", "hi", "42.png")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocalFSStore_Path_WithVariant(t *testing.T) {
	s := NewLocalFSStore("/cache")
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "42", Variant: "thumb"}
	got := s.Path(ref)
	want := filepath.Join("/cache", "medal-image", "hi", "42.thumb.png")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocalFSStore_Path_AlreadyHasExtension(t *testing.T) {
	s := NewLocalFSStore("/cache")
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "42.png"}
	got := s.Path(ref)
	want := filepath.Join("/cache", "medal-image", "hi", "42.png")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocalFSStore_Path_JSONKind(t *testing.T) {
	s := NewLocalFSStore("/cache")
	ref := Ref{Kind: KindMedalMetadata, TitleID: "hi", ID: "metadata"}
	got := s.Path(ref)
	want := filepath.Join("/cache", "medal-meta", "hi", "metadata.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocalFSStore_Path_WithOverride(t *testing.T) {
	s := NewLocalFSStore("/cache").WithRootOverride(KindMapImage, "/static/maps")
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "LiveFire"}
	got := s.Path(ref)
	want := filepath.Join("/static/maps", "map-image", "hi", "LiveFire.png")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocalFSStore_LookupBinary_Miss(t *testing.T) {
	s := NewLocalFSStore(t.TempDir())
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "999"}
	result, err := s.LookupBinary(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if result != nil {
		t.Error("attendu nil pour fichier absent")
	}
}

func TestLocalFSStore_PersistAndLookup_PNG(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalFSStore(dir)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "42"}

	// PNG magic bytes
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x01}
	payload := BinaryPayload{ContentType: "image/png", Bytes: pngBytes, ETag: "abc123"}

	if err := s.PersistBinary(context.Background(), ref, payload); err != nil {
		t.Fatalf("PersistBinary: %v", err)
	}

	result, err := s.LookupBinary(context.Background(), ref)
	if err != nil {
		t.Fatalf("LookupBinary: %v", err)
	}
	if result == nil {
		t.Fatal("attendu un résultat non-nil")
	}
	if result.ContentType != "image/png" {
		t.Errorf("ContentType: got %q, want %q", result.ContentType, "image/png")
	}
	if len(result.Bytes) != len(pngBytes) {
		t.Errorf("Bytes len: got %d, want %d", len(result.Bytes), len(pngBytes))
	}
	if result.ETag == "" {
		t.Error("ETag ne devrait pas être vide")
	}
}

func TestLocalFSStore_PersistBinary_AtomicWrite(t *testing.T) {
	// Vérifie qu'aucun fichier .tmp ne reste après une écriture réussie.
	dir := t.TempDir()
	s := NewLocalFSStore(dir)
	ref := Ref{Kind: KindChallengeBadge, TitleID: "hi", ID: "weekly-heroic"}
	payload := BinaryPayload{ContentType: "image/png", Bytes: []byte{0x89, 0x50, 0x4e, 0x47}}

	if err := s.PersistBinary(context.Background(), ref, payload); err != nil {
		t.Fatalf("PersistBinary: %v", err)
	}

	expected := s.Path(ref)
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("fichier final absent: %v", err)
	}

	tmp := expected + ".tmp"
	if _, err := os.Stat(tmp); err == nil {
		t.Error("fichier .tmp ne devrait pas exister après écriture réussie")
	}
}

func TestLocalFSStore_PersistBinary_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalFSStore(filepath.Join(dir, "deeply", "nested"))
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	payload := BinaryPayload{ContentType: "image/png", Bytes: []byte{1, 2, 3}}

	if err := s.PersistBinary(context.Background(), ref, payload); err != nil {
		t.Errorf("PersistBinary avec répertoires imbriqués: %v", err)
	}
}

func TestDetectContentType_PNG(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if got := detectContentType(png); got != "image/png" {
		t.Errorf("got %q, want image/png", got)
	}
}

func TestDetectContentType_JPEG(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff}
	if got := detectContentType(jpeg); got != "image/jpeg" {
		t.Errorf("got %q, want image/jpeg", got)
	}
}

func TestDetectContentType_Unknown(t *testing.T) {
	other := []byte{0x00, 0x01, 0x02}
	if got := detectContentType(other); got != "application/octet-stream" {
		t.Errorf("got %q, want application/octet-stream", got)
	}
}

func TestContentHash_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := contentHash(data)
	h2 := contentHash(data)
	if h1 != h2 {
		t.Error("contentHash doit être déterministe")
	}
	if len(h1) != 16 { // 8 octets hex = 16 caractères
		t.Errorf("longueur inattendue: %d", len(h1))
	}
}

func TestContentHash_DifferentData(t *testing.T) {
	if contentHash([]byte("a")) == contentHash([]byte("b")) {
		t.Error("hashes distincts attendus pour données différentes")
	}
}

func TestExtensionForKind(t *testing.T) {
	cases := []struct {
		kind Kind
		ext  string
	}{
		{KindMedalImage, ".png"},
		{KindMapImage, ".png"},
		{KindChallengeBadge, ".png"},
		{KindBPTrackImage, ".png"},
		{KindBPBackground, ".png"},
		{KindMedalMetadata, ".json"},
		{KindChallengeDefinition, ".json"},
		{KindRewardTrackDefinition, ".json"},
	}
	for _, tc := range cases {
		got := extensionForKind(tc.kind)
		if got != tc.ext {
			t.Errorf("extensionForKind(%q) = %q, want %q", tc.kind, got, tc.ext)
		}
	}
}
