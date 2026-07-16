package ogmeta

import (
	"strings"
	"testing"
)

func ptrF(v float64) *float64 { return &v }

func TestIsCrawler(t *testing.T) {
	cases := map[string]bool{
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)": true,
		"Twitterbot/1.0": true,
		"WhatsApp/2.23":  true,
		"Discordbot/2.0 (+https://discordapp.com)": true,
		"Mozilla/5.0 (compatible; redditbot/1.0)":  true,
		"Slackbot-LinkExpanding 1.0":               true,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120 Safari/537.36": false,
		"": false,
	}
	for ua, want := range cases {
		if got := IsCrawler(ua); got != want {
			t.Errorf("IsCrawler(%q) = %v, want %v", ua, got, want)
		}
	}
}

func TestDefaultMeta(t *testing.T) {
	m := DefaultMeta("https://demo.lvelup.info", "/players/demo-player/home", LocaleFR)
	if m.URL != "https://demo.lvelup.info/players/demo-player/home" {
		t.Errorf("URL = %q", m.URL)
	}
	if m.Image != "https://demo.lvelup.info/og-default.png" {
		t.Errorf("Image = %q", m.Image)
	}
	if m.ImageWidth != 1200 || m.ImageHeight != 630 {
		t.Errorf("dimensions = %dx%d", m.ImageWidth, m.ImageHeight)
	}
	if !strings.Contains(m.Description, "victoire") {
		t.Errorf("FR description attendue, got %q", m.Description)
	}
	en := DefaultMeta("https://lvelup.info", "/", LocaleEN)
	if !strings.Contains(en.Description, "win rate") {
		t.Errorf("EN description attendue, got %q", en.Description)
	}
}

func TestPlayerMeta_FR(t *testing.T) {
	m := PlayerMeta("https://demo.lvelup.info", "/players/demo-player/home",
		"JGtm", "Halo Infinite",
		KPIInput{KDR: ptrF(1.42), WinRate: 0.58, TotalMatches: 320}, LocaleFR)

	if !strings.Contains(m.Title, "JGtm") {
		t.Errorf("Title doit contenir le gamertag, got %q", m.Title)
	}
	if !strings.Contains(m.Title, "Halo Infinite") {
		t.Errorf("Title doit contenir le titre, got %q", m.Title)
	}
	for _, want := range []string{"KDR 1.42", "58 % de victoires", "320 matchs"} {
		if !strings.Contains(m.Description, want) {
			t.Errorf("Description %q doit contenir %q", m.Description, want)
		}
	}
}

func TestPlayerMeta_EN(t *testing.T) {
	m := PlayerMeta("https://lvelup.info", "/players/foo/home",
		"Foo", "", KPIInput{KDR: ptrF(0.9), WinRate: 0.5, TotalMatches: 10}, LocaleEN)
	if m.Title != "Foo — LevelUp" {
		t.Errorf("Title = %q (titre vide → sans mention du jeu)", m.Title)
	}
	for _, want := range []string{"KDR 0.90", "50% win rate", "10 matches"} {
		if !strings.Contains(m.Description, want) {
			t.Errorf("Description %q doit contenir %q", m.Description, want)
		}
	}
}

func TestPlayerMeta_MissingKPIs(t *testing.T) {
	// KDR nil + 0 matchs → pas de "KDR", pas de "matchs", mais win rate present.
	m := PlayerMeta("https://lvelup.info", "/players/foo/home",
		"Foo", "", KPIInput{KDR: nil, WinRate: 0.0, TotalMatches: 0}, LocaleFR)
	if strings.Contains(m.Description, "KDR") {
		t.Errorf("KDR nil ne doit pas apparaitre, got %q", m.Description)
	}
	if strings.Contains(m.Description, "matchs") {
		t.Errorf("0 matchs ne doit pas apparaitre, got %q", m.Description)
	}
	if !strings.Contains(m.Description, "0 % de victoires") {
		t.Errorf("win rate attendu, got %q", m.Description)
	}

	// Gamertag vide → repli sur le titre generique.
	empty := PlayerMeta("https://lvelup.info", "/", "", "", KPIInput{}, LocaleFR)
	if empty.Title != defaultTitle(LocaleFR) {
		t.Errorf("gamertag vide → titre generique, got %q", empty.Title)
	}
}

func TestRenderTags_Escaping(t *testing.T) {
	m := PlayerMeta("https://lvelup.info", "/players/foo/home",
		`Ev"il<script>`, "", KPIInput{WinRate: 0.5}, LocaleFR)
	tags := m.RenderTags()
	if strings.Contains(tags, `Ev"il<script>`) {
		t.Errorf("valeur non echappee dans les balises:\n%s", tags)
	}
	if !strings.Contains(tags, "&#34;") && !strings.Contains(tags, "&quot;") {
		t.Errorf("guillemet non echappe:\n%s", tags)
	}
	if !strings.Contains(tags, "&lt;script&gt;") {
		t.Errorf("chevrons non echappes:\n%s", tags)
	}
	// Balises essentielles presentes.
	for _, want := range []string{`property="og:title"`, `property="og:image"`, `name="twitter:card"`} {
		if !strings.Contains(tags, want) {
			t.Errorf("balise manquante %q dans:\n%s", want, tags)
		}
	}
}

func TestRender_ReplacesBlock(t *testing.T) {
	tmpl := []byte("<head>\n    <!-- og:start -->\n    <meta property=\"og:title\" content=\"OLD\" />\n    <!-- og:end -->\n  </head>")
	m := DefaultMeta("https://lvelup.info", "/", LocaleFR)
	out := string(Render(tmpl, m))

	if strings.Contains(out, "OLD") {
		t.Errorf("ancien contenu non remplace:\n%s", out)
	}
	if !strings.Contains(out, `content="https://lvelup.info/og-default.png"`) {
		t.Errorf("nouvelle image absente:\n%s", out)
	}
	// Marqueurs preserves (idempotence : un 2e Render doit re-fonctionner).
	if !strings.Contains(out, markerStart) || !strings.Contains(out, markerEnd) {
		t.Errorf("marqueurs perdus:\n%s", out)
	}
	out2 := string(Render([]byte(out), PlayerMeta("https://x.io", "/p", "Bob", "", KPIInput{WinRate: 1}, LocaleFR)))
	if !strings.Contains(out2, "Bob") {
		t.Errorf("second Render non applique:\n%s", out2)
	}
}

func TestRender_NoMarkers_Unchanged(t *testing.T) {
	tmpl := []byte("<head><title>x</title></head>")
	out := Render(tmpl, DefaultMeta("https://lvelup.info", "/", LocaleFR))
	if string(out) != string(tmpl) {
		t.Errorf("gabarit sans marqueurs doit rester inchange, got %q", out)
	}
}

func TestParseLocale(t *testing.T) {
	cases := map[string]Locale{
		"":                        LocaleFR,
		"fr-FR,fr;q=0.9":          LocaleFR,
		"en-US,en;q=0.9,fr;q=0.8": LocaleEN,
		"en":                      LocaleEN,
		"de-DE":                   LocaleFR, // langue inconnue → defaut FR
		"fr":                      LocaleFR,
	}
	for al, want := range cases {
		if got := ParseLocale(al); got != want {
			t.Errorf("ParseLocale(%q) = %v, want %v", al, got, want)
		}
	}
}

func TestLocaleFromParams(t *testing.T) {
	cases := []struct {
		name       string
		queryLang  string
		acceptLang string
		want       Locale
	}{
		{"lang=en prime sur Accept-Language fr", "en", "fr-FR,fr;q=0.9", LocaleEN},
		{"lang=fr prime sur Accept-Language en", "fr", "en-US,en;q=0.9", LocaleFR},
		{"lang insensible a la casse + espaces", " EN ", "fr-FR", LocaleEN},
		{"lang vide → repli Accept-Language en", "", "en-US,en;q=0.9", LocaleEN},
		{"lang inconnu → repli Accept-Language fr", "de", "fr-FR", LocaleFR},
		{"tout vide → defaut FR", "", "", LocaleFR},
	}
	for _, c := range cases {
		if got := LocaleFromParams(c.queryLang, c.acceptLang); got != c.want {
			t.Errorf("%s: LocaleFromParams(%q, %q) = %v, want %v", c.name, c.queryLang, c.acceptLang, got, c.want)
		}
	}
}
