// Package ogmeta construit les balises Open Graph / Twitter Card injectees dans
// index.html pour produire des apercus de liens (link unfurling) sur les reseaux
// sociaux (Facebook, WhatsApp, Reddit, Discord, Slack, X, Telegram...).
//
// Ce package est PUR : aucune dependance HTTP ni DB, tout est testable en
// isolation. Le wiring HTTP (detection du crawler, resolution du joueur, appel
// des services) vit dans internal/api/og_inject.go.
package ogmeta

import (
	"fmt"
	"html"
	"strings"
)

// Locale de rendu des textes OG.
type Locale string

const (
	LocaleFR Locale = "fr"
	LocaleEN Locale = "en"
)

// SiteName : nom de marque affiche par les unfurlers (og:site_name).
const SiteName = "LevelUp"

// DefaultImagePath : asset image auto-heberge (1200x630), servi a la racine du
// front. Une image auto-hebergee est fiable, contrairement aux URLs de banniere
// Waypoint/CDN par joueur (qui peuvent expirer / exiger une auth).
const DefaultImagePath = "/og-default.png"

// Marqueurs delimitant le bloc OG remplacable dans index.html. Le serveur
// remplace tout ce qui se trouve entre ces deux commentaires.
const (
	markerStart = "<!-- og:start -->"
	markerEnd   = "<!-- og:end -->"
)

// Meta contient les valeurs resolues d'une carte d'apercu.
type Meta struct {
	Title       string
	Description string
	URL         string // absolue (scheme + host + path)
	Image       string // absolue
	ImageWidth  int
	ImageHeight int
}

// KPIInput : sous-ensemble des KPIs joueur necessaires a la description.
// Les pointeurs nil signifient "donnee absente" et sont omis du texte.
type KPIInput struct {
	KDR          *float64 // ratio kills/deaths (domain.HeroKPIs.GlobalRatio)
	WinRate      float64  // unite 0..1 (ADR 0006), convertie en % a l'affichage
	TotalMatches int
}

// DefaultMeta construit la carte generique de marque pour une origine + un chemin.
func DefaultMeta(origin, path string, loc Locale) Meta {
	return withImage(Meta{
		Title:       defaultTitle(loc),
		Description: defaultDescription(loc),
		URL:         origin + path,
	}, origin)
}

// PlayerMeta construit une carte enrichie pour un joueur. titleLabel est le nom
// affichable du titre (ex. "Halo Infinite") ; vide = titre sans mention du jeu.
func PlayerMeta(origin, path, gamertag, titleLabel string, kpis KPIInput, loc Locale) Meta {
	return withImage(Meta{
		Title:       playerTitle(gamertag, titleLabel, loc),
		Description: playerDescription(kpis, loc),
		URL:         origin + path,
	}, origin)
}

// withImage complete les champs image (constants) a partir de l'origine.
func withImage(m Meta, origin string) Meta {
	m.Image = origin + DefaultImagePath
	m.ImageWidth = 1200
	m.ImageHeight = 630
	return m
}

func defaultTitle(loc Locale) string {
	if loc == LocaleEN {
		return SiteName + " — Halo stats"
	}
	return SiteName + " — Stats Halo"
}

func defaultDescription(loc Locale) string {
	if loc == LocaleEN {
		return "Your Halo stats dashboard: KDR, win rate, sessions, medals and more."
	}
	return "Ton dashboard de statistiques Halo : KDR, taux de victoire, sessions, medailles et plus."
}

func playerTitle(gamertag, titleLabel string, loc Locale) string {
	gt := strings.TrimSpace(gamertag)
	if gt == "" {
		return defaultTitle(loc)
	}
	label := strings.TrimSpace(titleLabel)
	if label == "" {
		return fmt.Sprintf("%s — %s", gt, SiteName)
	}
	if loc == LocaleEN {
		return fmt.Sprintf("%s — %s stats · %s", gt, label, SiteName)
	}
	return fmt.Sprintf("%s — Stats %s · %s", gt, label, SiteName)
}

// playerDescription assemble les KPIs disponibles en une phrase compacte.
// Repli sur la description generique si aucun KPI exploitable.
func playerDescription(k KPIInput, loc Locale) string {
	var parts []string
	if k.KDR != nil {
		parts = append(parts, fmt.Sprintf("KDR %.2f", *k.KDR))
	}
	wr := k.WinRate * 100
	if loc == LocaleEN {
		parts = append(parts, fmt.Sprintf("%.0f%% win rate", wr))
		if k.TotalMatches > 0 {
			parts = append(parts, fmt.Sprintf("%d matches", k.TotalMatches))
		}
	} else {
		parts = append(parts, fmt.Sprintf("%.0f %% de victoires", wr))
		if k.TotalMatches > 0 {
			parts = append(parts, fmt.Sprintf("%d matchs", k.TotalMatches))
		}
	}
	if len(parts) == 0 {
		return defaultDescription(loc)
	}
	return strings.Join(parts, " · ")
}

// RenderTags serialise les balises OG + Twitter Card. Toutes les valeurs
// dynamiques sont HTML-escaped (protection contre un gamertag contenant " ou <).
func (m Meta) RenderTags() string {
	esc := html.EscapeString
	lines := []string{
		`    <meta property="og:type" content="website" />`,
		fmt.Sprintf(`    <meta property="og:site_name" content="%s" />`, esc(SiteName)),
		fmt.Sprintf(`    <meta property="og:title" content="%s" />`, esc(m.Title)),
		fmt.Sprintf(`    <meta property="og:description" content="%s" />`, esc(m.Description)),
		fmt.Sprintf(`    <meta property="og:url" content="%s" />`, esc(m.URL)),
		fmt.Sprintf(`    <meta property="og:image" content="%s" />`, esc(m.Image)),
		fmt.Sprintf(`    <meta property="og:image:width" content="%d" />`, m.ImageWidth),
		fmt.Sprintf(`    <meta property="og:image:height" content="%d" />`, m.ImageHeight),
		`    <meta name="twitter:card" content="summary_large_image" />`,
		fmt.Sprintf(`    <meta name="twitter:title" content="%s" />`, esc(m.Title)),
		fmt.Sprintf(`    <meta name="twitter:description" content="%s" />`, esc(m.Description)),
		fmt.Sprintf(`    <meta name="twitter:image" content="%s" />`, esc(m.Image)),
	}
	return strings.Join(lines, "\n")
}

// Render remplace le bloc <!-- og:start -->...<!-- og:end --> du gabarit HTML
// par les balises de m. Si les marqueurs sont absents ou mal ordonnes, renvoie
// le gabarit inchange (filet de securite : jamais de HTML casse).
func Render(htmlTemplate []byte, m Meta) []byte {
	s := string(htmlTemplate)
	i := strings.Index(s, markerStart)
	if i < 0 {
		return htmlTemplate
	}
	j := strings.Index(s, markerEnd)
	if j < 0 || j < i {
		return htmlTemplate
	}
	j += len(markerEnd)

	var b strings.Builder
	b.Grow(len(s) + len(markerStart) + len(markerEnd) + 512)
	b.WriteString(s[:i])
	b.WriteString(markerStart)
	b.WriteByte('\n')
	b.WriteString(m.RenderTags())
	b.WriteByte('\n')
	b.WriteString("    ")
	b.WriteString(markerEnd)
	b.WriteString(s[j:])
	return []byte(b.String())
}

// crawlerUAs : sous-chaines (minuscules) identifiant les robots d'apercu social.
// Liste volontairement large : un faux positif ne fait que servir une carte
// enrichie a un client qui l'ignore, sans cout pour l'utilisateur humain.
var crawlerUAs = []string{
	"facebookexternalhit", // Facebook + WhatsApp
	"facebookcatalog",
	"meta-externalagent",
	"twitterbot",
	"linkedinbot",
	"slackbot",
	"slack-imgproxy",
	"discordbot",
	"telegrambot",
	"redditbot",
	"pinterest",
	"whatsapp",
	"skypeuripreview",
	"vkshare",
	"embedly",
	"quora link preview",
	"outbrain",
	"flipboard",
	"bitlybot",
	"nuzzel",
	"mastodon",
	"iframely",
	"google-inspectiontool",
	"applebot",
}

// IsCrawler retourne true si l'User-Agent correspond a un robot d'apercu social.
func IsCrawler(userAgent string) bool {
	if userAgent == "" {
		return false
	}
	ua := strings.ToLower(userAgent)
	for _, sub := range crawlerUAs {
		if strings.Contains(ua, sub) {
			return true
		}
	}
	return false
}

// LocaleFromParams resout la locale d'un apercu de lien. Un parametre de requete
// ?lang= explicite (fr/en) PRIME sur l'Accept-Language : le crawler social envoie
// SA propre langue (souvent en/absente), pas celle du partageur — un lien portant
// ?lang=fr permet donc a l'auteur du partage de figer la langue de la carte. Vide
// ou inconnu -> repli sur l'Accept-Language (ParseLocale, defaut FR).
func LocaleFromParams(queryLang, acceptLanguage string) Locale {
	switch strings.ToLower(strings.TrimSpace(queryLang)) {
	case "en":
		return LocaleEN
	case "fr":
		return LocaleFR
	}
	return ParseLocale(acceptLanguage)
}

// ParseLocale deduit la locale depuis un en-tete Accept-Language. Defaut FR
// (audience primaire) ; EN seulement si la langue preferee est l'anglais.
func ParseLocale(acceptLanguage string) Locale {
	al := strings.ToLower(strings.TrimSpace(acceptLanguage))
	if al == "" {
		return LocaleFR
	}
	first := al
	if i := strings.IndexAny(first, ",;"); i >= 0 {
		first = first[:i]
	}
	if strings.HasPrefix(strings.TrimSpace(first), "en") {
		return LocaleEN
	}
	return LocaleFR
}
