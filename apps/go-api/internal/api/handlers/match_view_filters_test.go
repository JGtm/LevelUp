// Package handlers — tests purs pour parseNeighborsFilterSpec (Phase 2b).
package handlers

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

// buildReq construit un *http.Request avec query params bien encodés.
// Les valeurs avec espaces / caractères réservés sont URL-escapées via
// net/url. La signature accepte une chaîne brute "key=value&..." que la
// fonction réencode proprement.
func buildReq(rawQS string) *http.Request {
	u := &url.URL{Path: "/players/p/matches/m/neighbors"}
	if rawQS != "" {
		// Les pairs k=v sont déjà séparés par & ; on les redécode pour
		// gérer espaces / quotes proprement.
		vals, _ := url.ParseQuery(rawQS)
		u.RawQuery = vals.Encode()
	}
	return &http.Request{Method: http.MethodGet, URL: u}
}

func mustParse(t *testing.T, qs string) (out struct {
	playlist   *string
	mode       *string
	session    *string
	outcome    *string
	withPlayer *string
	from       *time.Time
	to         *time.Time
	isEmpty    bool
}) {
	t.Helper()
	req := buildReq(qs)
	spec := parseNeighborsFilterSpec(req)
	if spec == nil {
		out.isEmpty = true
		return
	}
	out.playlist = spec.PlaylistName
	out.mode = spec.ModeCategory
	out.session = spec.SessionID
	out.outcome = spec.Outcome
	out.withPlayer = spec.WithPlayerXuid
	out.from = spec.DateFrom
	out.to = spec.DateTo
	return
}

// parseRawValue : helper qui construit un request avec une **valeur brute**
// non encodée (utilisée pour tester le rejet de caractères interdits — le
// charset hostile est passé tel quel à `url.Values.Set` qui l'accepte mais
// le handler doit le rejeter via la regex whitelist).
func parseRawValue(t *testing.T, key, value string) (out struct {
	playlist *string
	isEmpty  bool
}) {
	t.Helper()
	u := &url.URL{Path: "/p/m/n", RawQuery: key + "=" + url.QueryEscape(value)}
	req := &http.Request{Method: http.MethodGet, URL: u}
	spec := parseNeighborsFilterSpec(req)
	if spec == nil {
		out.isEmpty = true
		return
	}
	out.playlist = spec.PlaylistName
	return
}

func ptrEq[T comparable](got *T, want T) bool {
	return got != nil && *got == want
}

func TestParseNeighborsFilterSpec_Empty(t *testing.T) {
	r := mustParse(t, "")
	if !r.isEmpty {
		t.Errorf("query vide : spec doit être nil")
	}
}

func TestParseNeighborsFilterSpec_AllValid(t *testing.T) {
	qs := "playlist=Ranked+Arena&mode=BTB&session=session-123&outcome=win&from=2026-04-01T00:00:00Z&to=2026-05-01T00:00:00Z"
	r := mustParse(t, qs)
	if r.isEmpty {
		t.Fatalf("spec non vide attendu")
	}
	if !ptrEq(r.playlist, "Ranked Arena") {
		t.Errorf("playlist = %v, want Ranked Arena", r.playlist)
	}
	if !ptrEq(r.mode, "BTB") {
		t.Errorf("mode = %v, want BTB", r.mode)
	}
	if !ptrEq(r.session, "session-123") {
		t.Errorf("session = %v, want session-123", r.session)
	}
	if !ptrEq(r.outcome, "win") {
		t.Errorf("outcome = %v, want win", r.outcome)
	}
	if r.from == nil || r.from.Year() != 2026 || r.from.Month() != time.April {
		t.Errorf("from mal parsé : %v", r.from)
	}
	if r.to == nil || r.to.Year() != 2026 || r.to.Month() != time.May {
		t.Errorf("to mal parsé : %v", r.to)
	}
}

func TestParseNeighborsFilterSpec_OutcomeWhitelist(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"win", "win", true},
		{"loss", "loss", true},
		{"draw", "draw", true},
		{"dnf", "dnf", true},
		{"WIN", "", false},     // case-sensitive
		{"victory", "", false}, // hors whitelist
		{"abandoned", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			r := mustParse(t, "outcome="+tc.raw)
			if tc.ok {
				if !ptrEq(r.outcome, tc.want) {
					t.Errorf("outcome=%q : want %q got %v", tc.raw, tc.want, r.outcome)
				}
			} else {
				if !r.isEmpty {
					t.Errorf("outcome=%q (invalid) : spec doit être vide, got %v", tc.raw, r.outcome)
				}
			}
		})
	}
}

func TestParseNeighborsFilterSpec_DateInvalid(t *testing.T) {
	// from invalide → ignoré silencieusement, spec doit être vide
	r := mustParse(t, "from=not-a-date")
	if !r.isEmpty {
		t.Errorf("from invalide : spec doit être vide, got from=%v", r.from)
	}
	// from invalide + playlist valide → seul playlist est conservé
	r = mustParse(t, "from=not-a-date&playlist=Ranked")
	if r.isEmpty || !ptrEq(r.playlist, "Ranked") || r.from != nil {
		t.Errorf("from invalide + playlist valide : seule playlist doit rester, got %+v", r)
	}
}

func TestParseNeighborsFilterSpec_PlaylistWhitelistRegex(t *testing.T) {
	cases := []struct {
		raw string
		ok  bool
	}{
		{"Ranked Arena", true},
		{"BTB:CTF", true},
		{"v1.2", true},
		{"a-b_c", true},
		// Charset hors whitelist
		{"DROP TABLE foo;", false}, // ; non autorisé
		{"<script>", false},        // < > non autorisés
		{"a'b", false},             // ' non autorisé
		{"line\nbreak", false},     // \n non autorisé
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			r := parseRawValue(t, "playlist", tc.raw)
			if tc.ok {
				if !ptrEq(r.playlist, tc.raw) {
					t.Errorf("playlist=%q : want %q got %v", tc.raw, tc.raw, r.playlist)
				}
			} else {
				if !r.isEmpty {
					t.Errorf("playlist=%q (invalid) : spec doit être vide, got %v", tc.raw, r.playlist)
				}
			}
		})
	}
}

func TestParseNeighborsFilterSpec_LengthLimit(t *testing.T) {
	// playlist > 64 chars → rejeté
	long := ""
	for i := 0; i < 65; i++ {
		long += "a"
	}
	r := mustParse(t, "playlist="+long)
	if !r.isEmpty {
		t.Errorf("playlist > 64 chars : doit être ignoré")
	}
}

func TestParseNeighborsFilterSpec_TrimsWhitespace(t *testing.T) {
	r := mustParse(t, "playlist=%20%20Ranked%20%20")
	// strings.TrimSpace appliqué : doit retourner "Ranked"
	if !ptrEq(r.playlist, "Ranked") {
		t.Errorf("trim espaces : want 'Ranked' got %v", r.playlist)
	}
}

func TestParseNeighborsFilterSpec_WithPlayer_Numeric(t *testing.T) {
	r := mustParse(t, "with_player=2533274791785593")
	if !ptrEq(r.withPlayer, "2533274791785593") {
		t.Errorf("with_player numérique : want 2533274791785593, got %v", r.withPlayer)
	}
}

func TestParseNeighborsFilterSpec_WithPlayer_NonNumeric_Rejected(t *testing.T) {
	cases := []string{
		"abc123",     // alphanumérique
		"<script>",   // injection
		"123-456",    // tiret
		"123 456",    // espace
		"1'; DROP--", // SQL injection
	}
	for _, raw := range cases {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			r := parseRawValue(t, "with_player", raw)
			if !r.isEmpty {
				t.Errorf("with_player=%q (invalid) : doit être ignoré, got playlist=%v", raw, r.playlist)
			}
		})
	}
}

func TestParseNeighborsFilterSpec_WithPlayer_LengthLimit(t *testing.T) {
	// XUID > 32 chars → rejeté
	long := ""
	for i := 0; i < 33; i++ {
		long += "1"
	}
	r := mustParse(t, "with_player="+long)
	if !r.isEmpty {
		t.Errorf("with_player > 32 chars : doit être ignoré, got %v", r.withPlayer)
	}
}
