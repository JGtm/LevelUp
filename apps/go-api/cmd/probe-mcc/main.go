// Sonde live Halo: The Master Chief Collection (MCC) — exploration des endpoints.
//
// But : MCC n'est documenté par AUCUNE API publique/communautaire (cryptum couvre
// H5/HW2 ; les wrappers Grunt/SPNKr couvrent Infinite). Pourtant le site
// halowaypoint.com/halo-the-master-chief-collection/players/{gt} affiche bien un
// service record MCC → une API interne le sert. Cette sonde cherche QUEL host +
// QUEL préfixe + QUELLE convention de path répond, en réutilisant le SpartanToken
// v4 du pool (account-level, partagé inter-titres comme prouvé pour Halo 5).
//
// Conforme aux contraintes : REUTILISE les helpers auth testés du projet
// (RefreshHaloTokensViaStoreFirst), PAS de clé haloapi.com, PAS de réinvention de
// la résolution token. On teste deux conventions car MCC pourrait suivre soit le
// modèle Halo 5 (gamertag brut, /h5/, header Spartan, ?auth=st), soit le modèle
// Halo Infinite (xuid(NNN), /hi/, header Spartan + éventuel 343-clearance).
//
// Un CONTRÔLE Halo 5 (endpoint connu-bon) prouve que le token est valide : ainsi
// un 404/401 MCC est interprétable (host/path faux vs token mort).
//
// Usage : go run ./cmd/probe-mcc [ownerGamertag]
//
//	ownerGamertag : propriétaire des tokens ET joueur sondé (défaut JGtm).
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// User-Agent du client natif Halo (cpprestsdk). Certains services 343 gatent sur
// l'UA ; on reproduit celui qui marche pour la sonde Halo 5.
const userAgent = "cpprestsdk/2.4.0"

// Hosts candidats. spartanstats sert Halo 5 (donc accepte nos tokens v4) — premier
// suspect si MCC partage l'infra titres legacy. halostats sert Infinite. Les hosts
// dédiés sont des hypothèses : un échec DNS/transport est une info utile (host
// inexistant) et coûte peu.
const (
	hostSpartanStats = "https://spartanstats.svc.halowaypoint.com"
	hostHaloStats    = "https://halostats.svc.halowaypoint.com"
	hostMCCStats     = "https://mcc-stats.svc.halowaypoint.com" // hypothèse dédiée
	hostHMCC         = "https://hmcc.svc.halowaypoint.com"      // hypothèse dédiée
	hostSettings     = "https://settings.svc.halowaypoint.com"  // oban/clearance + settings (titre enregistré ?)
)

type probeTarget struct {
	label string
	url   string
	// clearance : si true, on enverra aussi un header 343-clearance vide-ok placeholder
	// (juste pour distinguer un refus clearance). Non utilisé au 1er passage.
}

func main() {
	ownerGT := "JGtm"
	if len(os.Args) > 1 {
		ownerGT = os.Args[1]
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatalf("config.Load: %v", err)
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		fatalf("LoadPlayers: %v", err)
	}
	var ownerXUID string
	for i := range players {
		if players[i].Gamertag == ownerGT && ownerXUID == "" {
			ownerXUID = players[i].XUID
		}
	}
	if ownerXUID == "" {
		fatalf("owner %q introuvable dans db_profiles", ownerGT)
	}

	storeDir := titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
	store := auth.NewMultiUserTokenStore(storeDir)
	provider := auth.NewSISUProvider()

	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, ownerXUID, ownerGT)
	if err != nil || res == nil || res.Tokens == nil {
		fatalf("refresh tokens %s: err=%v res=%v", ownerGT, err, res)
	}
	spartan := res.Tokens.SpartanToken
	preamble := "?"
	if len(spartan) >= 3 {
		preamble = spartan[:3] // "v4=" attendu
	}
	fmt.Printf("owner=%s xuid=%s spartan_len=%d preamble=%q\n\n", ownerGT, ownerXUID, len(spartan), preamble)

	gt := url.PathEscape(ownerGT)
	q := url.QueryEscape(ownerGT)
	xuidTok := fmt.Sprintf("xuid(%s)", ownerXUID)
	xuidPath := url.PathEscape(xuidTok)

	targets := []probeTarget{
		// ── CONTRÔLE : endpoint Halo 5 connu-bon → prouve que le token est valide.
		{label: "CONTROL.h5.matches", url: fmt.Sprintf("%s/h5/players/%s/matches?start=0&count=3&auth=st", hostSpartanStats, gt)},
	}

	// ── Convention A : modèle Halo 5 (gamertag brut, ?auth=st) sur spartanstats.
	for _, prefix := range []string{"hmcc", "mcc"} {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("A.spartanstats./%s/players/{gt}/matches", prefix),
				url: fmt.Sprintf("%s/%s/players/%s/matches?start=0&count=5&auth=st", hostSpartanStats, prefix, gt)},
			probeTarget{label: fmt.Sprintf("A.spartanstats./%s/servicerecords/global", prefix),
				url: fmt.Sprintf("%s/%s/servicerecords/global?players=%s&auth=st", hostSpartanStats, prefix, q)},
		)
	}

	// ── Convention B : modèle Halo Infinite (xuid(NNN)) sur halostats.
	for _, prefix := range []string{"hmcc", "mcc"} {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("B.halostats./%s/players/xuid/matches", prefix),
				url: fmt.Sprintf("%s/%s/players/%s/matches?count=5", hostHaloStats, prefix, xuidPath)},
			probeTarget{label: fmt.Sprintf("B.halostats./%s/players/xuid/matchmade/servicerecord", prefix),
				url: fmt.Sprintf("%s/%s/players/%s/matchmade/servicerecord", hostHaloStats, prefix, xuidPath)},
			// Variante : même convention mais sur spartanstats (host H5).
			probeTarget{label: fmt.Sprintf("B.spartanstats./%s/players/xuid/matches", prefix),
				url: fmt.Sprintf("%s/%s/players/%s/matches?count=5", hostSpartanStats, prefix, xuidPath)},
		)
	}

	// ── Hosts dédiés (hypothèses) : résolution DNS = info en soi.
	for _, h := range []string{hostMCCStats, hostHMCC} {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("C.dedicated %s gt/matches", h),
				url: fmt.Sprintf("%s/hmcc/players/%s/matches?start=0&count=5&auth=st", h, gt)},
		)
	}

	// ── Round D : DÉCOUVERTE TITRE via le host settings/oban. Si MCC est un titre
	// 343 enregistré, son slug interne apparaît ici. hi/h5 servent de contrôles : on
	// teste plusieurs slugs candidats pour MCC (hmcc/mcc/mccpc/mcc-pc/hmccpc).
	for _, slug := range []string{"hi", "h5", "hmcc", "mcc", "mccpc", "mcc-pc", "hmccpc"} {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("D.oban clearance titles/%s", slug),
				url: fmt.Sprintf("%s/oban/flight-configurations/titles/%s/audiences/RETAIL/active", hostSettings, slug)},
		)
	}
	// settings/<slug> public (pas d'auth requise pour hipc d'après la doc communautaire).
	for _, slug := range []string{"hipc", "h5", "hmcc", "mcc", "mccpc"} {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("D.settings/%s", slug),
				url: fmt.Sprintf("%s/settings/%s", hostSettings, slug)},
		)
	}

	// ── Round E : le slug hmcc est confirmé (oban=200) → on cherche SON host stats.
	// On teste /hmcc/players/{gt}/matches + service-record sur tous les hosts plausibles
	// (liste cryptum + noms dédiés). Un 200/401/403 = host pertinent ; 404/DNS = non.
	otherHosts := map[string]string{
		"stats.svc":     "https://stats.svc.halowaypoint.com",
		"hacs.svc":      "https://hacs.svc.halowaypoint.com",
		"telemetry.svc": "https://telemetry.svc.halowaypoint.com",
		"presence.svc":  "https://presence.svc.halowaypoint.com",
		"mccstats.svc":  "https://mccstats.svc.halowaypoint.com",
		"hmccstats.svc": "https://hmccstats.svc.halowaypoint.com",
		"halomcc.svc":   "https://halomcc.svc.halowaypoint.com",
	}
	for name, h := range otherHosts {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("E.%s /hmcc/players/{gt}/matches", name),
				url: fmt.Sprintf("%s/hmcc/players/%s/matches?start=0&count=5&auth=st", h, gt)},
		)
	}

	// ── Round F : préfixe = segment d'URL du SITE "halo-the-master-chief-collection".
	// (a) sur les hosts stats (au cas où le service routerait sur le segment long) ;
	// (b) sur www.halowaypoint.com lui-même (route API/SSR éventuellement adressable).
	const seg = "halo-the-master-chief-collection"
	const hostWWW = "https://www.halowaypoint.com"
	for _, h := range []string{hostSpartanStats, hostHaloStats} {
		targets = append(targets,
			probeTarget{label: fmt.Sprintf("F.%s /%s/players/{gt}/matches", strings.TrimPrefix(h, "https://"), seg),
				url: fmt.Sprintf("%s/%s/players/%s/matches?start=0&count=5&auth=st", h, seg, gt)},
		)
	}
	targets = append(targets,
		probeTarget{label: "F.www players/{gt}/matches", url: fmt.Sprintf("%s/%s/players/%s/matches?count=5", hostWWW, seg, gt)},
		probeTarget{label: "F.www players/{gt}/service-record", url: fmt.Sprintf("%s/%s/players/%s/service-record", hostWWW, seg, gt)},
		probeTarget{label: "F.www players/{gt}", url: fmt.Sprintf("%s/%s/players/%s", hostWWW, seg, gt)},
		probeTarget{label: "F.www api players/{gt}/service-record", url: fmt.Sprintf("%s/api/%s/players/%s/service-record", hostWWW, seg, gt)},
	)

	// ── Round G : CONFIRMÉ par capture XHR navigateur (2026-06-26). Le vrai backend
	// MCC = mccapi.svc.halowaypoint.com, préfixe /hmcc/, convention users/gt(GT)/…,
	// auth = même header x-343-authorization-spartan v4 (notre pool). On valide ici
	// que notre token headless suffit (sans navigateur).
	const hostMCCAPI = "https://mccapi.svc.halowaypoint.com"
	// Sujet du round G : par défaut le propriétaire du token, surchargé par arg 2
	// (pour tester l'accès cross-joueur / "roster" sans ownership).
	subjectGT := ownerGT
	if len(os.Args) > 2 {
		subjectGT = os.Args[2]
	}
	fmt.Printf("\n[round G] subject=%s\n", subjectGT)
	gtWrap := fmt.Sprintf("gt(%s)", subjectGT)
	gtWrapEsc := url.PathEscape(gtWrap)
	u := func(p string) string { return hostMCCAPI + p }
	ug := func(p string) string { return fmt.Sprintf("%s/hmcc/users/%s%s", hostMCCAPI, gtWrapEsc, p) }
	// Surface COMPLÈTE extraite du client officiel (chunk 2303, 12 fonctions getHaloMcc*).
	targets = append(targets,
		// Catalogues (métadonnées, pas de joueur)
		probeTarget{label: "G.ranks", url: u("/hmcc/ranks?lang=en")},
		probeTarget{label: "G.maps", url: u("/hmcc/maps?lang=en")},
		probeTarget{label: "G.seasons", url: u("/hmcc/seasons?lang=en")},
		probeTarget{label: "G.season(S8)", url: u("/hmcc/seasons/S8?lang=en")},
		probeTarget{label: "G.playlists", url: u("/hmcc/playlists?lang=en")},
		// Carrière / stats joueur
		probeTarget{label: "G.service-record", url: ug("/service-record")},
		probeTarget{label: "G.campaign-summary", url: ug("/service-record/campaign-summary")},
		probeTarget{label: "G.campaign(h1)", url: ug("/service-record/h1/campaign")},
		probeTarget{label: "G.campaign(reach)", url: ug("/service-record/reach/campaign")},
		probeTarget{label: "G.skill-ranks", url: ug("/skill-ranks?platform=Xbox&hoppers=")},
		probeTarget{label: "G.achievements", url: ug("/achievements?lang=en")},
		// Personnalisation / inventaire
		probeTarget{label: "G.inventory", url: ug("/inventory")},
		// Matchs + filtres (categoryId, title)
		probeTarget{label: "G.matches", url: ug("/matches?page=1&pageSize=25")},
		probeTarget{label: "G.matches(title=Halo2)", url: ug("/matches?page=1&pageSize=25&title=Halo2")},
		probeTarget{label: "G.matches(categoryId=1)", url: ug("/matches?page=1&pageSize=25&categoryId=1")},
		// Sonde du plafond de pagination (maxPage observé = 4 @ pageSize 25).
		probeTarget{label: "G.matches(pageSize=100)", url: ug("/matches?page=1&pageSize=100")},
		probeTarget{label: "G.matches(pageSize=100,page=2)", url: ug("/matches?page=2&pageSize=100")},
	)

	client := &http.Client{Timeout: 20 * time.Second}
	for _, t := range targets {
		status, ctype, body, derr := probe(ctx, client, t.url, spartan)
		fmt.Printf("── %s\n   %s\n", t.label, t.url)
		if derr != "" {
			fmt.Printf("   ERREUR transport : %s\n\n", derr)
			continue
		}
		preview := strings.TrimSpace(body)
		if len(preview) > 1400 {
			preview = preview[:1400] + " …[tronqué]"
		}
		// Dump complet des 200 pour analyse hors-ligne.
		if status == http.StatusOK {
			dump := fmt.Sprintf("%s/mcc_%s.json", os.TempDir(), sanitize(t.label))
			_ = os.WriteFile(dump, []byte(body), 0o644)
			fmt.Printf("   [dump] %s\n", dump)
		}
		fmt.Printf("   HTTP %d · %s · %d bytes\n   %s\n\n", status, ctype, len(body), preview)
	}
}

// probe exécute un GET authentifié façon natif Halo : header Spartan v4 + UA
// cpprestsdk + Accept json. Retourne status, content-type, corps, message transport.
func probe(ctx context.Context, client *http.Client, rawURL, spartan string) (int, string, string, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return -1, "", "", err.Error()
	}
	req.Header.Set("X-343-Authorization-Spartan", spartan)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return -1, "", "", err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(b), ""
}

func sanitize(s string) string {
	r := strings.NewReplacer("/", "_", ".", "_", " ", "_", "{", "", "}", "")
	return strings.ToLower(r.Replace(s))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
