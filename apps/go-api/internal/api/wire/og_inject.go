package wire

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/ogmeta"
	"levelup/go-api/internal/port"
)

// ogInjectTimeout borne le cout de l'enrichissement OG (appel DB GetHomePage)
// pour une requete crawler. Depasse → repli sur la carte generique.
const ogInjectTimeout = 3 * time.Second

// serveIndexWithOG sert index.html en injectant les balises Open Graph adaptees
// a la requete, pour produire des apercus de liens sur les reseaux sociaux.
//
//   - Humains et routes non-joueur : carte generique de marque (cout nul, juste
//     l'origine corrigee pour que og:url/og:image pointent le bon domaine).
//   - Crawlers sur /players/{slug}/... d'une instance demo : carte enrichie
//     (gamertag + KPIs) via les services existants.
//
// L'enrichissement est volontairement limite au mode demo : les pages joueur
// reelles sont ownership-gated (un non-proprietaire voit "Indisponible"), donc
// exposer leurs KPIs a un crawler anonyme serait a la fois une fuite et un
// apercu trompeur. Filet de securite : toute erreur → carte generique, jamais
// d'echec de page.
func (reg *ServiceRegistry) serveIndexWithOG(w http.ResponseWriter, req *http.Request, indexPath string) {
	// no-cache : l'index ne doit pas masquer un nouveau build apres redeploiement.
	w.Header().Set("Cache-Control", "no-cache")

	indexHTML, err := os.ReadFile(indexPath)
	if err != nil {
		slog.ErrorContext(req.Context(), "og_inject: lecture index.html echouee", "err", err, "path", indexPath)
		http.ServeFile(w, req, indexPath)
		return
	}

	origin := requestOrigin(req)
	loc := ogmeta.ParseLocale(req.Header.Get("Accept-Language"))
	meta := ogmeta.DefaultMeta(origin, req.URL.Path, loc)

	if reg != nil && reg.cfg != nil && reg.cfg.DemoMode && ogmeta.IsCrawler(req.Header.Get("User-Agent")) {
		if slug, ok := playerSlugFromPath(req.URL.Path); ok {
			if enriched, ok := reg.playerOGMeta(req.Context(), origin, req.URL.Path, slug, loc); ok {
				meta = enriched
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(ogmeta.Render(indexHTML, meta))
}

// requestOrigin reconstitue "scheme://host" en tenant compte du reverse proxy
// (X-Forwarded-Proto / X-Forwarded-Host ; prod : LEVELUP_TRUST_PROXY_HEADERS=true).
func requestOrigin(req *http.Request) string {
	scheme := "http"
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = firstToken(proto)
	} else if req.TLS != nil {
		scheme = "https"
	}
	host := req.Host
	if fwd := req.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = firstToken(fwd)
	}
	return scheme + "://" + host
}

// firstToken renvoie le premier element d'une liste "a, b, c" (en-tetes proxy).
func firstToken(s string) string {
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// playerSlugFromPath extrait le slug de "/players/{slug}[/...]". false si le
// chemin n'est pas une route joueur.
func playerSlugFromPath(p string) (string, bool) {
	segs := strings.Split(strings.TrimPrefix(p, "/"), "/")
	if len(segs) < 2 || segs[0] != "players" || segs[1] == "" {
		return "", false
	}
	return segs[1], true
}

// playerOGMeta construit une carte enrichie via les services existants
// (HomeCtx → GetHomePage). Best-effort : (Meta{}, false) sur toute erreur.
func (reg *ServiceRegistry) playerOGMeta(ctx context.Context, origin, path, slug string, loc ogmeta.Locale) (ogmeta.Meta, bool) {
	ctx, cancel := context.WithTimeout(ctx, ogInjectTimeout)
	defer cancel()

	svc, xuid, gamertag, err := reg.HomeCtx(ctx, slug)
	if err != nil {
		slog.DebugContext(ctx, "og_inject: resolve player failed", "slug", slug, "err", err)
		return ogmeta.Meta{}, false
	}
	return ogMetaFromHome(ctx, svc, xuid, gamertag, origin, path, loc)
}

// ogMetaFromHome pose l'invariant d'identité de PAGE puis construit la carte
// enrichie depuis un HomeService déjà résolu. Extrait de playerOGMeta pour
// isoler (et tester) le forçage du xuid.
//
// HomeCtx est la variante NON authentifiée : le contexte d'un crawler anonyme ne
// porte aucun xuid. GetHomePage résout l'identité Spartan (SpartanIdentity) via
// GetSpartanIdentity, scopée sur ctxkeys.HaloXUID ; sans forçage cette résolution
// cible "" (identité vide). On pose donc le xuid du joueur de la PAGE — même
// invariant que HomeCtxWithAuth (cf. forcePageIdentityXUID, registry_auth.go),
// et title-agnostic (aucune dépendance au slug).
func ogMetaFromHome(ctx context.Context, svc port.HomeService, xuid, gamertag, origin, path string, loc ogmeta.Locale) (ogmeta.Meta, bool) {
	ctx = forcePageIdentityXUID(ctx, xuid)
	page, err := svc.GetHomePage(ctx, gamertag, string(loc))
	if err != nil {
		slog.DebugContext(ctx, "og_inject: GetHomePage failed", "gamertag", gamertag, "err", err)
		return ogmeta.Meta{}, false
	}

	kpis := page.Hero.KPIs
	meta := ogmeta.PlayerMeta(origin, path, page.Hero.PlayerName, "", ogmeta.KPIInput{
		KDR:          kpis.GlobalRatio,
		WinRate:      kpis.WinRate,
		TotalMatches: kpis.TotalMatches,
	}, loc)
	slog.DebugContext(ctx, "og_inject: enriched player card", "gamertag", gamertag)
	return meta, true
}
