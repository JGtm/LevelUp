package assets

import (
	"context"
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// ChainFetcher
// ---------------------------------------------------------------------------

func TestChainFetcher_Supports_AnyFetcher(t *testing.T) {
	a := &stubFetcher{supported: false}
	b := &stubFetcher{supported: true}
	c := NewChainFetcher(a, b)
	if !c.Supports(KindMedalImage) {
		t.Error("Supports devrait retourner true si au moins un fetcher supporte")
	}
}

func TestChainFetcher_Supports_NoneSupports(t *testing.T) {
	a := &stubFetcher{supported: false}
	b := &stubFetcher{supported: false}
	c := NewChainFetcher(a, b)
	if c.Supports(KindMedalImage) {
		t.Error("Supports devrait retourner false si aucun fetcher ne supporte")
	}
}

func TestChainFetcher_Supports_Empty(t *testing.T) {
	c := NewChainFetcher()
	if c.Supports(KindMedalImage) {
		t.Error("Supports devrait retourner false sur chaîne vide")
	}
}

func TestChainFetcher_Fetch_FirstSucceeds(t *testing.T) {
	payload := URLPayload{URL: "https://cdn.example.com/medal.png"}
	a := &stubFetcher{supported: true, payload: payload}
	b := &stubFetcher{supported: true, payload: URLPayload{URL: "https://fallback.example.com"}}
	c := NewChainFetcher(a, b)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	res, err := c.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	u, ok := res.(URLPayload)
	if !ok || u.URL != payload.URL {
		t.Errorf("attendu %q, got %v", payload.URL, res)
	}
	if b.calls != 0 {
		t.Error("le deuxième fetcher ne devrait pas être appelé")
	}
}

func TestChainFetcher_Fetch_FirstNotFound_FallsToSecond(t *testing.T) {
	fallbackPayload := URLPayload{URL: "https://fallback.example.com/medal.png"}
	a := &stubFetcher{supported: true, err: ErrNotFound}
	b := &stubFetcher{supported: true, payload: fallbackPayload}
	c := NewChainFetcher(a, b)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	res, err := c.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	u, ok := res.(URLPayload)
	if !ok || u.URL != fallbackPayload.URL {
		t.Errorf("attendu fallback URL, got %v", res)
	}
}

func TestChainFetcher_Fetch_NonNotFoundError_StopsChain(t *testing.T) {
	// Une erreur non-404 arrête la chaîne immédiatement.
	a := &stubFetcher{supported: true, err: ErrUpstreamUnavailable}
	b := &stubFetcher{supported: true, payload: URLPayload{URL: "https://fallback.example.com"}}
	c := NewChainFetcher(a, b)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	_, err := c.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("attendu ErrUpstreamUnavailable, got %v", err)
	}
	if b.calls != 0 {
		t.Error("le deuxième fetcher ne devrait pas être appelé sur erreur non-404")
	}
}

func TestChainFetcher_Fetch_AllNotFound_ReturnsLastErr(t *testing.T) {
	a := &stubFetcher{supported: true, err: ErrNotFound}
	b := &stubFetcher{supported: true, err: ErrNotFound}
	c := NewChainFetcher(a, b)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	_, err := c.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("attendu ErrNotFound, got %v", err)
	}
}

func TestChainFetcher_Fetch_SkipsUnsupportedFetchers(t *testing.T) {
	a := &stubFetcher{supported: false}
	b := &stubFetcher{supported: true, payload: URLPayload{URL: "https://cdn.example.com"}}
	c := NewChainFetcher(a, b)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	res, err := c.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if a.calls != 0 {
		t.Error("le fetcher non-supportant ne devrait pas être appelé")
	}
	if _, ok := res.(URLPayload); !ok {
		t.Error("attendu URLPayload")
	}
}

func TestChainFetcher_Fetch_EmptyChain_ReturnsUnsupported(t *testing.T) {
	c := NewChainFetcher()
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	_, err := c.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Errorf("attendu ErrUnsupportedKind, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// SpritesheetFallbackFetcher
// ---------------------------------------------------------------------------

func TestSpritesheetFallback_Supports_OnlyMedalImage(t *testing.T) {
	primary := &stubFetcher{supported: true}
	f := NewSpritesheetFallbackFetcher(primary, "")
	if !f.Supports(KindMedalImage) {
		t.Error("devrait supporter KindMedalImage")
	}
	if f.Supports(KindMapImage) {
		t.Error("ne devrait pas supporter KindMapImage")
	}
}

func TestSpritesheetFallback_Fetch_PrimarySucceeds(t *testing.T) {
	payload := BinaryPayload{ContentType: "image/png", Bytes: []byte{0x89, 0x50}}
	primary := &stubFetcher{supported: true, payload: payload}
	f := NewSpritesheetFallbackFetcher(primary, "https://gamecms.example.com")

	ref := Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}
	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if _, ok := res.(BinaryPayload); !ok {
		t.Error("attendu BinaryPayload du primary")
	}
}

func TestSpritesheetFallback_Fetch_PrimaryNotFound_ReturnsSpritesheet(t *testing.T) {
	primary := &stubFetcher{supported: true, err: ErrNotFound}
	f := NewSpritesheetFallbackFetcher(primary, "https://gamecms.example.com")

	ref := Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}
	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	u, ok := res.(URLPayload)
	if !ok {
		t.Fatal("attendu URLPayload spritesheet")
	}
	wantURL := "https://gamecms.example.com/hi/Progression/file/medals/sprites/halo_infinite.png"
	if u.URL != wantURL {
		t.Errorf("URL spritesheet: got %q, want %q", u.URL, wantURL)
	}
}

func TestSpritesheetFallback_Fetch_PrimaryUpstreamError_Propagates(t *testing.T) {
	primary := &stubFetcher{supported: true, err: ErrUpstreamUnavailable}
	f := NewSpritesheetFallbackFetcher(primary, "")

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("attendu ErrUpstreamUnavailable, got %v", err)
	}
}

func TestSpritesheetFallback_Fetch_WrongKind(t *testing.T) {
	primary := &stubFetcher{supported: true}
	f := NewSpritesheetFallbackFetcher(primary, "")

	ref := Ref{Kind: KindChallengeBadge, TitleID: "hi", ID: "badge"}
	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Errorf("attendu ErrUnsupportedKind, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// isLockError
// ---------------------------------------------------------------------------

func TestIsLockError_LockInMessage(t *testing.T) {
	if !isLockError(errors.New("database is locked")) {
		t.Error("devrait détecter 'database is locked'")
	}
	if !isLockError(errors.New("could not set lock on file")) {
		t.Error("devrait détecter 'could not set lock'")
	}
	if !isLockError(errors.New("LOCK acquisition failed")) {
		t.Error("devrait détecter 'lock' (insensible à la casse)")
	}
}

func TestIsLockError_NilError(t *testing.T) {
	if isLockError(nil) {
		t.Error("nil ne devrait pas être une lock error")
	}
}

func TestIsLockError_OtherError(t *testing.T) {
	if isLockError(errors.New("disk full")) {
		t.Error("'disk full' ne devrait pas être une lock error")
	}
}
