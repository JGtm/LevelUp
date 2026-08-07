// cmd/refresh-career-ranks — One-shot CLI pour peupler career_rank_translations
// avec les libellés multilingues fournis par l'endpoint Waypoint GameCMS
//
//	GET /hi/Progression/file/RewardTracks/CareerRanks/careerRank1.json
//
// Le JSON Waypoint encapsule chaque champ texte dans un objet
// `{ "value": "...", "translations": { "fr-FR": "...", "de-DE": "...", ... } }`
// — un seul appel suffit donc à récupérer toutes les langues exposées par 343i.
//
// Usage :
//
//	refresh-career-ranks [--player JGtm] [--title-id halo_infinite]
//
// Acquisition du token Spartan :
//  1. Variable SPARTAN_TOKEN si présente (court-circuit, utile pour debug)
//  2. Sinon : reproduit le flow runtime — lit msal_token_cache /
//     oauth_refresh_token depuis sync_meta du player DB, puis Exchange via
//     auth.MSALProvider.
//
// Variables d'environnement :
//
//	SPARTAN_TOKEN     (optionnel) court-circuite l'auth, header brut
//	CLEARANCE_TOKEN   (optionnel) header 343-clearance accompagnant SPARTAN_TOKEN
//	LEVELUP_REPO_ROOT racine du repo (auto-détectée sinon)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	authpkg "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

const careerRankRewardTrackPath = "RewardTracks/CareerRanks/careerRank1.json"

// translatableString correspond au champ Waypoint `{value, translations}`.
type translatableString struct {
	Value        string            `json:"value"`
	Translations map[string]string `json:"translations"`
}

// rewardTrackPayload est la projection minimale du JSON career rank reward track.
type rewardTrackPayload struct {
	Ranks []rankEntry `json:"Ranks"`
}

type rankEntry struct {
	Rank         int                `json:"Rank"`
	RankTitle    translatableString `json:"RankTitle"`
	RankSubTitle translatableString `json:"RankSubTitle"`
	RankTier     translatableString `json:"RankTier"`
}

func main() {
	fs := flag.NewFlagSet("refresh-career-ranks", flag.ExitOnError)
	titleID := fs.String("title-id", titlePkg.DefaultSlug, "Title ID (ex: halo_infinite)")
	player := fs.String("player", "", "Slug joueur pour résoudre les tokens via sync_meta (ignoré si SPARTAN_TOKEN env est défini)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if err := run(*titleID, *player); err != nil {
		fmt.Fprintf(os.Stderr, "erreur : %v\n", err)
		os.Exit(1)
	}
}

func run(titleID, playerSlug string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	if cfg.RepoRoot == "" {
		return fmt.Errorf("LEVELUP_REPO_ROOT non défini")
	}

	ctx := context.Background()

	tokens, err := resolveTokens(ctx, cfg, playerSlug)
	if err != nil {
		return fmt.Errorf("résolution tokens: %w", err)
	}
	tokenProvider := func(_ context.Context) (*domain.HaloTokens, error) {
		return tokens, nil
	}

	fetcher := assets.NewGameCMSFetcher(http.DefaultClient, tokenProvider, "")
	payload, err := fetcher.Fetch(ctx, assets.Ref{
		Kind:    assets.KindRewardTrackDefinition,
		ID:      careerRankRewardTrackPath,
		TitleID: titleID,
	})
	if err != nil {
		return fmt.Errorf("fetch GameCMS career ranks: %w", err)
	}
	jsonPayload, ok := payload.(assets.JSONPayload)
	if !ok {
		return fmt.Errorf("payload inattendu (type %T)", payload)
	}

	var track rewardTrackPayload
	if err := json.Unmarshal(jsonPayload.RawJSON, &track); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	if len(track.Ranks) == 0 {
		return fmt.Errorf("aucun rang dans la payload")
	}

	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titleID)
	metaDB, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("ouverture metadata.duckdb: %w", err)
	}
	defer metaDB.Close()

	// La table peut ne pas encore exister si la migration n'a pas tourné — créer
	// idempotent pour permettre un run avant le premier boot du serveur.
	if _, err := metaDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS career_rank_translations (
			rank_id  INTEGER NOT NULL,
			lang     VARCHAR NOT NULL,
			title    VARCHAR,
			subtitle VARCHAR,
			tier     VARCHAR,
			fetched_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (rank_id, lang)
		);
	`); err != nil {
		return fmt.Errorf("ensure table: %w", err)
	}

	inserted, err := upsertTranslations(ctx, metaDB, track.Ranks)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}

	fmt.Printf("✅ %d rangs × %d langues = %d lignes upsertées dans career_rank_translations\n",
		len(track.Ranks), uniqueLangCount(track.Ranks), inserted)
	return nil
}

// resolveTokens récupère les tokens Halo soit via env SPARTAN_TOKEN, soit en
// rejouant la chaîne d'auth runtime depuis le player DB.
func resolveTokens(ctx context.Context, cfg *config.AppConfig, playerSlug string) (*domain.HaloTokens, error) {
	if envToken := os.Getenv("SPARTAN_TOKEN"); envToken != "" {
		return &domain.HaloTokens{
			SpartanToken:   envToken,
			ClearanceToken: os.Getenv("CLEARANCE_TOKEN"),
		}, nil
	}
	if playerSlug == "" {
		return nil, fmt.Errorf("SPARTAN_TOKEN absent ET --player non fourni — précisez l'un des deux")
	}

	pdb, err := config.ResolvePlayer(ctx, cfg, playerSlug, titlePkg.DefaultSlug)
	if err != nil {
		return nil, fmt.Errorf("résoudre player %q: %w", playerSlug, err)
	}

	provider := authpkg.NewSISUProvider()

	// ADR 0023 — pipeline canonique via MultiUserTokenStore puis legacy DuckDB.
	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	legacy := authpkg.LegacyAuthInputs{Source: "duckdb"}
	legacy.MSALCache, _ = duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	legacy.OAuthRT, _ = duckdb.ReadOAuthRefreshToken(ctx, pdb.Player)

	result, rerr := authpkg.RefreshHaloTokensViaStoreFirst(ctx, store, provider, pdb.XUID, pdb.Gamertag, legacy)
	if rerr != nil {
		return nil, rerr
	}
	if tokens := authpkg.HaloTokensFromExchange(result); tokens != nil {
		fmt.Fprintf(os.Stderr, "auth: tokens obtenus pour xuid=%s\n", pdb.XUID)
		return tokens, nil
	}
	return nil, fmt.Errorf("aucun token disponible pour player %q (ni MSAL cache ni OAuth refresh)", playerSlug)
}

// upsertTranslations parcourt chaque rang et insère une ligne par langue dans
// career_rank_translations. Le contenu `value` est stocké sous lang "en"
// (référence canonique anglaise de Waypoint).
func upsertTranslations(ctx context.Context, db *duckdb.DB, ranks []rankEntry) (int, error) {
	count := 0
	for _, r := range ranks {
		langs := collectLangs(r)
		for _, lang := range langs {
			title := pick(r.RankTitle, lang)
			subtitle := pick(r.RankSubTitle, lang)
			tier := pick(r.RankTier, lang)
			if _, err := db.Exec(ctx, `
				INSERT OR REPLACE INTO career_rank_translations
					(rank_id, lang, title, subtitle, tier, fetched_at)
				VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			`, r.Rank, lang, title, subtitle, tier); err != nil {
				return count, fmt.Errorf("rank %d lang %s: %w", r.Rank, lang, err)
			}
			count++
		}
	}
	return count, nil
}

// collectLangs retourne l'union triée des locales présentes dans les trois
// champs traduisibles ("en" pour la valeur canonique + chaque clé translations).
func collectLangs(r rankEntry) []string {
	set := map[string]struct{}{"en": {}}
	for _, ts := range []translatableString{r.RankTitle, r.RankSubTitle, r.RankTier} {
		for k := range ts.Translations {
			set[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// pick retourne la traduction pour `lang`, ou la valeur canonique si lang == "en"
// ou absent (fallback EN au cas où Waypoint omet la clé).
func pick(ts translatableString, lang string) string {
	if lang == "en" {
		return ts.Value
	}
	if v, ok := ts.Translations[lang]; ok {
		return v
	}
	return ts.Value
}

func uniqueLangCount(ranks []rankEntry) int {
	if len(ranks) == 0 {
		return 0
	}
	return len(collectLangs(ranks[0]))
}
