// Sonde live Halo 5 (Phase 1, etape 0 du HANDOFF_HALO5_EXPERIMENTAL).
//
// But : confirmer que 343 sert ENCORE les endpoints internes Halo 5 en 2026, et
// que le SpartanToken v4 du pool Infinite est accepte par ces services (hypothese
// du handoff §1). Capture les status + shapes JSON reels pour ajuster la matrice
// de capabilities (§2) avant d'investir dans l'adapter/le client h5.
//
// Conforme aux contraintes : REUTILISE les helpers auth testes du projet
// (RefreshHaloTokensViaStoreFirst -> SpartanToken v4 du store), PAS de cle
// haloapi.com, PAS de reinvention de la resolution token. Les hosts + le shape de
// requete (header X-343-Authorization-Spartan, User-Agent cpprestsdk, query
// ?auth=st, gamertag brut en {player}, PAS de 343-clearance) sont calques sur
// cryptum-halodotapi (src/classes/Request + modules/api/authorities|endpoints/H5).
//
// Usage : go run ./cmd/probe-h5 [ownerGamertag]
//
//	ownerGamertag : proprietaire des tokens ET joueur sonde (defaut JGtm).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/auth"
)

// Hosts internes Halo 5 (cryptum src/modules/api/authorities/index.js).
const (
	hostSpartanStats = "https://spartanstats.svc.halowaypoint.com"
	hostHaloPlayer   = "https://haloplayer.svc.halowaypoint.com"
	hostUGC          = "https://ugc.svc.halowaypoint.com"
	// hostHalo5API : authority REQ d'après Halo5Reqs (MichaelJLiu). À confirmer par
	// la sonde — peut être un alias de spartanstats ou un service distinct/mort.
	hostHalo5API = "https://halo5api.svc.halowaypoint.com"
)

// User-Agent du client Halo 5 (cryptum : cpprestsdk). Certains services 343
// gatent sur l'UA ; on reproduit celui que la sonde de reference utilise.
const halo5UserAgent = "cpprestsdk/2.4.0"

type probeTarget struct {
	label string
	url   string
}

func main() {
	ownerGT := "JGtm"
	if len(os.Args) > 1 {
		ownerGT = os.Args[1]
	}
	// Racine portant config/titles/<slug> (constants.toml) pour l'EndpointResolver
	// title-agnostic de la Phase 3 (CMS assets). Défaut = worktree.
	configRoot := "c:/Users/Guillaume/Downloads/Scripts/levelup-multititre"
	if len(os.Args) > 2 {
		configRoot = os.Args[2]
	}
	slug := halo5.TitleSlug
	if len(os.Args) > 3 {
		slug = os.Args[3]
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
		if players[i].Gamertag == ownerGT {
			ownerXUID = players[i].XUID
		}
	}
	if ownerXUID == "" {
		fatalf("owner %q introuvable dans db_profiles", ownerGT)
	}

	storeDir := titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
	store := auth.NewMultiUserTokenStore(storeDir)
	provider := auth.NewSISUProvider()

	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, ownerXUID, ownerGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatalf("refresh tokens %s: err=%v res=%v", ownerGT, err, res)
	}
	spartan := res.Tokens.SpartanToken
	preamble := "?"
	if len(spartan) >= 3 {
		preamble = spartan[:3] // "v4=" attendu
	}
	fmt.Printf("owner=%s spartan_len=%d spartan_preamble=%q\n", ownerGT, len(spartan), preamble)
	fmt.Printf("player_sonde=%s (gamertag brut)\n\n", ownerGT)

	gt := url.PathEscape(ownerGT)
	q := url.QueryEscape(ownerGT)
	targets := []probeTarget{
		{"SpartanStats.SERVICE_RECORDS[arena]", fmt.Sprintf("%s/h5/servicerecords/arena?players=%s&auth=st", hostSpartanStats, q)},
		{"SpartanStats.SERVICE_RECORDS[warzone]", fmt.Sprintf("%s/h5/servicerecords/warzone?players=%s&auth=st", hostSpartanStats, q)},
		{"SpartanStats.MATCHES", fmt.Sprintf("%s/h5/players/%s/matches?start=0&count=5&include-times=true&auth=st", hostSpartanStats, gt)}, // include-times=true → MatchCompletedDate horodaté précis (fidelity 2)
		{"SpartanStats.COMMENDATIONS", fmt.Sprintf("%s/h5/players/%s/commendations?auth=st", hostSpartanStats, gt)},
		{"SpartanStats.CREDITS", fmt.Sprintf("%s/h5/players/%s/credits?auth=st", hostSpartanStats, gt)},
		// REQ packs/cards (inventaire joueur) + catalogue — sonde « sonde d'abord » REQ.
		// Plusieurs hosts/paths candidats (Halo5Reqs vs cryptum) : on confirme lequel
		// répond 200 avant d'investir dans la feature REQ-as-progression.
		{"SpartanStats.REQ_PACKS", fmt.Sprintf("%s/h5/players/%s/packs?auth=st", hostSpartanStats, gt)},
		{"SpartanStats.REQ_CARDS", fmt.Sprintf("%s/h5/players/%s/cards?auth=st", hostSpartanStats, gt)},
		{"Halo5API.REQ_PACKS", fmt.Sprintf("%s/h5/players/%s/packs?auth=st", hostHalo5API, gt)},
		{"Halo5API.REQ_CARDS", fmt.Sprintf("%s/h5/players/%s/cards?auth=st", hostHalo5API, gt)},
		{"Halo5API.REQ_CATALOG", fmt.Sprintf("%s/en-us/reqs?auth=st", hostHalo5API)},
		{"HaloPlayer.SPARTAN", fmt.Sprintf("%s/h5/profiles/%s/spartan?auth=st", hostHaloPlayer, gt)},
		{"HaloPlayer.APPEARANCE", fmt.Sprintf("%s/h5/profiles/%s/appearance?auth=st", hostHaloPlayer, gt)},
	}

	client := &http.Client{Timeout: 20 * time.Second}
	for _, t := range targets {
		status, ctype, body, derr := probe(ctx, client, t.url, spartan)
		fmt.Printf("── %s\n   %s\n", t.label, t.url)
		if derr != "" {
			fmt.Printf("   ERREUR transport : %s\n\n", derr)
			continue
		}
		preview := strings.TrimSpace(body)
		if len(preview) > 2200 {
			preview = preview[:2200] + " …[tronque]"
		}
		fmt.Printf("   HTTP %d · %s · %d bytes\n   %s\n\n", status, ctype, len(body), preview)
	}

	// ── Phase 2 : carnage detail + EVENTS (per-match) — les 2 endpoints les plus
	// riches, JAMAIS sondes. On lit la liste de matchs, on prend le 1er, et on tape
	// sa carnage detail (scoreboard etendu) + sa timeline d'events (kill-feed
	// structure : death/medal/weapon horodates). C'est ce qui valide que Halo 5
	// expose nativement ce qu'Infinite n'a qu'au prix du decodage film.
	fmt.Println("══════ Phase 2 : carnage detail + match events ══════")
	probeMatchDetails(ctx, client, spartan, gt)

	// ── Phase 3 : surfaces CMS ASSETS (rangs SR, médailles, designations CSR, maps).
	// Host résolu TITLE-AGNOSTIC via l'EndpointResolver (constants.toml), pas hardcodé.
	// But : confirmer hosts/paths/shapes avant d'écrire les fetchers (Track A du plan).
	fmt.Printf("\n══════ Phase 3 : CMS assets (slug=%s, host via EndpointResolver) ══════\n", slug)
	probeAssetCMS(ctx, client, spartan, slug, configRoot)
}

// probeAssetCMS sonde les surfaces CMS d'assets d'un titre. L'host (gamecms/ugc) et
// le game_prefix sont résolus par l'EndpointResolver depuis constants.toml (zéro
// host hardcodé — title-agnostic). Les PATHS restent h5-spécifiques (c'est l'objet
// de la sonde : confirmer lesquels répondent 200). SR-manifest = confirmé den.dev ;
// le reste = hypothèses à valider.
func probeAssetCMS(ctx context.Context, client *http.Client, spartan, slug, configRoot string) {
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(configRoot, []string{slug, "halo_infinite"}, nil); len(errs) != 0 {
		fmt.Printf("   load mappings (%s) échoué: %v\n", configRoot, errs)
		return
	}
	resolver := games.NewMappingsEndpointResolver(reg, "halo_infinite")
	gamecms, ok := resolver.HostFor(slug, games.EndpointGameCMS)
	if !ok {
		fmt.Printf("   host gamecms non résolu pour %q (constants.toml [endpoints].gamecms ?)\n", slug)
		return
	}
	ugc, _ := resolver.HostFor(slug, games.EndpointDiscoveryUGC)
	prefix, _ := resolver.GamePrefixFor(slug)
	fmt.Printf("   gamecms=%s  ugc=%s  prefix=%s\n\n", gamecms, ugc, prefix)

	// Noms de contenu CONFIRMÉS par cryptum (Alexis-Bize/cryptum-halodotapi
	// ContentHacs) — pas des devinettes : doivent répondre 200. Cible = les noms de
	// maps/modes/playlists que les GUIDs du match (MapId/HopperId/GameBaseVariantId)
	// résolvent. Le param Count borne la taille (les contenus catalogue sont paginés).
	targets := []probeTarget{
		{"SR_MANIFEST", fmt.Sprintf("%s/contents/SpartanRankManifest?auth=st", gamecms)},
		{"HOPPER", fmt.Sprintf("%s/contents/Hopper?StartAt=0&Count=3&auth=st", gamecms)},
		{"GAME_BASE_VARIANT", fmt.Sprintf("%s/contents/GameBaseVariant?StartAt=0&Count=3&auth=st", gamecms)},
		{"GAME_VARIANT_DEFINITION", fmt.Sprintf("%s/contents/GameVariantDefinition?StartAt=0&Count=3&auth=st", gamecms)},
		{"EMBLEM", fmt.Sprintf("%s/contents/Emblem?StartAt=0&Count=2&auth=st", gamecms)},
		{"REQ", fmt.Sprintf("%s/contents/REQ?StartAt=0&Count=2&auth=st", gamecms)},
	}
	if ugc != "" {
		targets = append(targets, probeTarget{"UGC_MAP_VARIANTS", fmt.Sprintf("%s/%s/players/JGtm/mapvariants?auth=st", ugc, prefix)})
	}

	for _, t := range targets {
		status, ctype, body, derr := probe(ctx, client, t.url, spartan)
		fmt.Printf("── %s\n   %s\n", t.label, t.url)
		if derr != "" {
			fmt.Printf("   ERREUR transport : %s\n\n", derr)
			continue
		}
		// Dump complet des manifests 200 pour analyse hors-ligne (parsing du shape).
		if status == http.StatusOK {
			dump := fmt.Sprintf("%s/h5_asset_%s.json", os.TempDir(), strings.ToLower(t.label))
			_ = os.WriteFile(dump, []byte(body), 0o644)
			fmt.Printf("   [dump] %s\n", dump)
		}
		preview := strings.TrimSpace(body)
		if len(preview) > 700 {
			preview = preview[:700] + " …[tronque]"
		}
		fmt.Printf("   HTTP %d · %s · %d bytes\n   %s\n\n", status, ctype, len(body), preview)
	}
}

// matchesResp — projection minimale de /h5/players/{gt}/matches pour extraire un
// match exploitable (Id + Links) afin de taper sa carnage detail + ses events.
type matchesResp struct {
	Results []struct {
		Id struct {
			MatchId  string `json:"MatchId"`
			GameMode int    `json:"GameMode"`
		} `json:"Id"`
		Links map[string]struct {
			Path        string `json:"Path"`
			AuthorityId string `json:"AuthorityId"`
		} `json:"Links"`
	} `json:"Results"`
}

// gameModeSeg mappe le GameMode numerique (liste de matchs) vers le segment d'URL
// string attendu par la carnage detail. Fallback "arena" (le plus courant).
func gameModeSeg(n int) string {
	switch n {
	case 1:
		return "arena"
	case 2:
		return "campaign"
	case 3:
		return "custom"
	case 4:
		return "warzone"
	default:
		return "arena"
	}
}

// matchPick = un match exploitable (id + segment de mode resolu).
type matchPick struct {
	id      string
	modeSeg string
}

// probeMatchDetails liste les matchs, choisit un match ARENA + un match d'un autre
// mode (warzone si dispo), et sonde pour chacun : carnage detail + plusieurs
// VARIANTES d'endpoint events (avec/sans segment de mode), + le film manifest UGC.
// Objectif : trancher si 343 sert UNE quelconque timeline d'events, ou si le
// per-kill est uniquement derriere le film (comme Infinite).
func probeMatchDetails(ctx context.Context, client *http.Client, spartan, gt string) {
	listURL := fmt.Sprintf("%s/h5/players/%s/matches?start=0&count=25&auth=st", hostSpartanStats, gt)
	status, _, body, derr := probe(ctx, client, listURL, spartan)
	if derr != "" || status != http.StatusOK {
		fmt.Printf("   impossible de lister les matchs (HTTP %d, %s)\n", status, derr)
		return
	}
	var mr matchesResp
	if err := json.Unmarshal([]byte(body), &mr); err != nil {
		fmt.Printf("   parse matches: %v\n", err)
		return
	}

	var arena, other *matchPick
	for _, r := range mr.Results {
		if r.Id.MatchId == "" {
			continue
		}
		modeSeg := gameModeSeg(r.Id.GameMode)
		if l, ok := r.Links["StatsMatchDetails"]; ok && l.Path != "" {
			parts := strings.Split(strings.Trim(l.Path, "/"), "/")
			if len(parts) >= 2 && parts[0] == "h5" {
				modeSeg = parts[1]
			}
		}
		if modeSeg == "arena" && arena == nil {
			arena = &matchPick{r.Id.MatchId, modeSeg}
		} else if modeSeg != "arena" && other == nil {
			other = &matchPick{r.Id.MatchId, modeSeg}
		}
	}

	if arena != nil {
		fmt.Printf("   match ARENA : %s\n", arena.id)
		probeOneMatch(ctx, client, spartan, *arena, true)
	}
	if other != nil {
		fmt.Printf("   match %s : %s\n", strings.ToUpper(other.modeSeg), other.id)
		probeOneMatch(ctx, client, spartan, *other, false)
	} else {
		fmt.Println("   (aucun match non-arena dans les 25 derniers — pas de test warzone)")
	}
}

// probeOneMatch sonde la carnage + variantes events (+ film manifest si withFilm)
// d'un match donne.
func probeOneMatch(ctx context.Context, client *http.Client, spartan string, m matchPick, withFilm bool) {
	targets := []probeTarget{
		{"CARNAGE_DETAIL", fmt.Sprintf("%s/h5/%s/matches/%s?auth=st", hostSpartanStats, m.modeSeg, m.id)},
		{"EVENTS[mode-seg]", fmt.Sprintf("%s/h5/%s/matches/%s/events?auth=st", hostSpartanStats, m.modeSeg, m.id)},
		{"EVENTS[no-mode]", fmt.Sprintf("%s/h5/matches/%s/events?auth=st", hostSpartanStats, m.id)},
	}
	if withFilm {
		targets = append(targets,
			probeTarget{"UGC_FILM_MANIFEST", fmt.Sprintf("%s/h5/films/%s?view=film-manifest&auth=st", hostUGC, m.id)},
		)
	}
	for _, t := range targets {
		status, ctype, body, derr := probe(ctx, client, t.url, spartan)
		fmt.Printf("── %s\n   %s\n", t.label, t.url)
		if derr != "" {
			fmt.Printf("   ERREUR transport : %s\n\n", derr)
			continue
		}
		// Pour la carnage : signaler explicitement si les tableaux per-kill fins
		// (EnemyKills/WeaponStats/Impulses) sont presents et non vides.
		extra := ""
		if t.label == "CARNAGE_DETAIL" {
			extra = fmt.Sprintf("   [fins] EnemyKills_nonempty=%t WeaponStats_nonempty=%t Impulses_nonempty=%t\n",
				strings.Contains(body, `"EnemyKills":[{`),
				strings.Contains(body, `"WeaponStats":[{`),
				strings.Contains(body, `"Impulses":[{`))
		}
		// Dump complet des events pour analyse hors-ligne (types + shape kill event).
		if t.label == "EVENTS[no-mode]" && status == http.StatusOK {
			_ = os.WriteFile(os.TempDir()+"/h5_events.json", []byte(body), 0o644)
			fmt.Printf("   [dump] %s/h5_events.json\n", os.TempDir())
		}
		// Dump complet du carnage pour le mapping participants (shape PlayerStats).
		if t.label == "CARNAGE_DETAIL" && status == http.StatusOK {
			_ = os.WriteFile(os.TempDir()+"/h5_carnage.json", []byte(body), 0o644)
			fmt.Printf("   [dump] %s/h5_carnage.json\n", os.TempDir())
		}
		preview := strings.TrimSpace(body)
		limit := 1500
		if t.label == "UGC_FILM_MANIFEST" {
			limit = 3000
		}
		if len(preview) > limit {
			preview = preview[:limit] + " …[tronque]"
		}
		fmt.Printf("   HTTP %d · %s · %d bytes\n%s   %s\n\n", status, ctype, len(body), extra, preview)
	}
}

// probe execute un GET authentifie facon Halo 5 (cryptum) : header Spartan v4 +
// UA cpprestsdk + Accept json. PAS de 343-clearance (Halo 5 n'en utilise pas ;
// le quirk ?auth=st est porte par l'URL). Retourne status, content-type, corps,
// et un message d'erreur transport eventuel.
func probe(ctx context.Context, client *http.Client, rawURL, spartan string) (int, string, string, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return -1, "", "", err.Error()
	}
	req.Header.Set("X-343-Authorization-Spartan", spartan)
	req.Header.Set("Accept", "application/json")
	// NB : on NE fixe PAS Accept-Encoding manuellement — sinon le Transport Go
	// désactive la décompression gzip transparente et resp.Body reste compressé
	// (corps illisible). Sans le header, Go demande+décompresse gzip tout seul.
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", halo5UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return -1, "", "", err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(b), ""
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
