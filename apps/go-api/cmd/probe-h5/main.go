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
	"levelup/go-api/internal/platform/auth"
)

// Hosts internes Halo 5 (cryptum src/modules/api/authorities/index.js).
const (
	hostSpartanStats = "https://spartanstats.svc.halowaypoint.com"
	hostHaloPlayer   = "https://haloplayer.svc.halowaypoint.com"
	hostUGC          = "https://ugc.svc.halowaypoint.com"
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
	provider := auth.NewMSALProvider()

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
		{"SpartanStats.MATCHES", fmt.Sprintf("%s/h5/players/%s/matches?start=0&count=5&auth=st", hostSpartanStats, gt)},
		{"SpartanStats.COMMENDATIONS", fmt.Sprintf("%s/h5/players/%s/commendations?auth=st", hostSpartanStats, gt)},
		{"SpartanStats.CREDITS", fmt.Sprintf("%s/h5/players/%s/credits?auth=st", hostSpartanStats, gt)},
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
