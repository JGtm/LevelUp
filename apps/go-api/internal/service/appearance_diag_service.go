// Package service — appearance_diag_service.go : diagnostic apparence Spartan ID
// à la demande, par joueur suivi (volet 2 du plan
// .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md, Lot F).
//
// Pour un player_slug de db_profiles.json, produit le verdict des 4 composants du
// Spartan ID (bannière, emblème, backdrop, service tag) en combinant :
//   - la résolution LIVE via les primitives de diagnostic du Lot E (haloclient) ;
//   - les valeurs SERVIES (dernières connues) lues en DB (CareerServedReader).
//
// PIÈGE CRITIQUE (régression PR #63) : le xuid utilisé est TOUJOURS celui du
// PROFIL (db_profiles.json), JAMAIS ctxkeys.HaloXUID(ctx) qui désigne le compte
// CONNECTÉ. Le service ne prend le xuid que du PlayerSummary résolu.
//
// AUCUNE écriture DB : calcul à la demande, aucune persistance nouvelle. Toute
// défaillance live (tokens, réseau, capability) devient un VERDICT par composant,
// jamais une erreur 500 — seule l'absence de profil (404) ou un échec d'énumération
// des profils (500) remonte en erreur.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	haloclient "levelup/go-api/internal/sync/haloclient"
)

// ErrProfileNotFound est retourné quand le player_slug ne correspond à aucun
// profil de db_profiles.json (le handler le mappe en 404).
var ErrProfileNotFound = errors.New("profil apparence introuvable")

// appearanceDiagFetchBudget borne la durée TOTALE des fetchs live d'un diagnostic
// (critère du plan : verdict < 10 s). Chaque requête HTTP du client conserve par
// ailleurs son propre timeout de 20 s.
const appearanceDiagFetchBudget = 8 * time.Second

// CareerServedReader lit les valeurs SERVIES (dernières connues) + le dernier
// statut de fetch d'un joueur — la source de vérité de ce que l'app affiche.
// Implémenté par *duckdb.CareerLiveRepo (binding dans wire) ; mocké en test.
type CareerServedReader interface {
	LoadLastCareerRank(ctx context.Context, xuid string) (*domain.CareerRankRow, error)
	LoadLastFetchStatus(ctx context.Context, xuid string) (string, error)
}

// AppearanceFetcher abstrait le client Halo pour le diagnostic (testabilité) :
// entrées brutes + primitives de diagnostic par composant du Lot E. Construit par
// requête à partir des tokens du PROFIL.
type AppearanceFetcher interface {
	FetchAppearanceInputs(ctx context.Context, xuid string) (*haloclient.AppearanceInputs, error)
	DiagnoseNameplate(ctx context.Context, emblemPath string, cfg int64) haloclient.AppearanceDiagnosis
	DiagnoseCustomizationImage(ctx context.Context, inventoryPath string) haloclient.AppearanceDiagnosis
}

// AppearanceFetcherFactory construit un AppearanceFetcher pour un jeu de tokens
// (ownerXUID = xuid du profil, pour le rate budget partagé du compte).
type AppearanceFetcherFactory func(spartanToken, clearanceToken, ownerXUID string) AppearanceFetcher

// AppearanceDiagDeps regroupe les dépendances injectées (constructeur ≤ 1 param).
type AppearanceDiagDeps struct {
	// LoadPlayers énumère les profils db_profiles.json d'un titre (cfg.LoadPlayers).
	LoadPlayers func(titleSlug string) ([]domain.PlayerSummary, error)
	// ResolveServedReader ouvre le lecteur de valeurs servies d'un joueur.
	ResolveServedReader func(ctx context.Context, playerSlug, titleSlug string) (CareerServedReader, error)
	// FetchTokens résout les tokens Halo du PROFIL (jamais du compte connecté).
	// Échec (err ou tokens vides) → verdict auth_required, pas une erreur.
	FetchTokens func(ctx context.Context, xuid, gamertag string) (spartanToken, clearanceToken string, err error)
	// NewFetcher construit le fetcher live (prod : client Halo + rate budget).
	NewFetcher AppearanceFetcherFactory
	// HasSpartanCustomizer : true si le titre résout l'apparence via un pipeline
	// dédié (capability spartan_customizer) → la résolution nameplate/emblème live
	// n'est PAS la source (verdict not_supported), on n'affiche que le servi.
	HasSpartanCustomizer func(titleSlug string) bool
	// Clock : injectable pour les tests (défaut time.Now).
	Clock func() time.Time
}

// AppearanceDiagService produit le diagnostic apparence d'un joueur suivi, à la
// demande, SANS aucune écriture DB.
type AppearanceDiagService struct {
	deps AppearanceDiagDeps
}

// NewAppearanceDiagService construit le service (Clock défaut = time.Now).
func NewAppearanceDiagService(deps AppearanceDiagDeps) *AppearanceDiagService {
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	return &AppearanceDiagService{deps: deps}
}

// servedContext porte les valeurs servies (dernière ligne DB) + le dernier statut
// de fetch persisté. row peut être nil (aucune donnée connue pour ce joueur).
type servedContext struct {
	row         *domain.CareerRankRow
	fetchStatus string
}

// Diagnose calcule le diagnostic des 4 composants du Spartan ID d'un joueur suivi.
// Erreur SEULEMENT si le profil est introuvable (→ 404) ou si l'énumération des
// profils échoue (→ 500) ; toute défaillance live devient un verdict par composant.
func (s *AppearanceDiagService) Diagnose(ctx context.Context, playerSlug, titleSlug string) (domain.AppearanceDiagnosisResponse, error) {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	startWall := time.Now()
	slog.InfoContext(ctx, "appearance_diag: start", "player", playerSlug, "titleSlug", titleSlug)

	profile, err := s.resolveProfile(titleSlug, playerSlug)
	if err != nil {
		return domain.AppearanceDiagnosisResponse{}, err
	}

	served := s.loadServed(ctx, playerSlug, titleSlug, profile.XUID)

	var components []domain.AppearanceComponentDiagnosis
	if s.deps.HasSpartanCustomizer != nil && s.deps.HasSpartanCustomizer(titleSlug) {
		// Titre à pipeline appearance dédié (capability spartan_customizer, ex. H5) :
		// la résolution nameplate/emblème live n'est pas la source de vérité →
		// not_supported, valeurs servies affichées, aucun fetch HINF cross-titre.
		components = uniformComponents(served.row, haloclient.VerdictNotSupported)
		slog.InfoContext(ctx, "appearance_diag: not_supported (spartan_customizer)",
			"player", playerSlug, "titleSlug", titleSlug)
	} else {
		components = s.liveComponents(ctx, titleSlug, profile, served)
	}

	resp := domain.AppearanceDiagnosisResponse{
		PlayerSlug:      profile.PlayerSlug,
		Gamertag:        profile.Gamertag,
		XUID:            profile.XUID,
		TitleSlug:       titleSlug,
		GeneratedAt:     s.deps.Clock().UTC().Format(time.RFC3339),
		LastFetchStatus: served.fetchStatus,
		Components:      components,
	}
	slog.InfoContext(ctx, "appearance_diag: done",
		"player", playerSlug, "titleSlug", titleSlug,
		"duration_ms", time.Since(startWall).Milliseconds())
	return resp, nil
}

// resolveProfile trouve le PlayerSummary du slug dans les profils du titre. Le
// xuid en sort (PROFIL, jamais compte connecté — cf. entête, régression PR #63).
func (s *AppearanceDiagService) resolveProfile(titleSlug, playerSlug string) (*domain.PlayerSummary, error) {
	players, err := s.deps.LoadPlayers(titleSlug)
	if err != nil {
		return nil, fmt.Errorf("appearance_diag: chargement des profils (%s): %w", titleSlug, err)
	}
	for i := range players {
		if players[i].PlayerSlug == playerSlug {
			return &players[i], nil
		}
	}
	return nil, fmt.Errorf("%w: slug=%q", ErrProfileNotFound, playerSlug)
}

// loadServed lit best-effort les valeurs servies + le dernier fetch status. Toute
// erreur est loggée puis dégradée (row nil / status "") : l'affichage des valeurs
// connues ne doit jamais faire échouer le diagnostic.
func (s *AppearanceDiagService) loadServed(ctx context.Context, playerSlug, titleSlug, xuid string) servedContext {
	reader, err := s.deps.ResolveServedReader(ctx, playerSlug, titleSlug)
	if err != nil || reader == nil {
		slog.WarnContext(ctx, "appearance_diag: lecteur valeurs servies indisponible",
			"player", playerSlug, "titleSlug", titleSlug, "err", err)
		return servedContext{}
	}
	var out servedContext
	if row, rerr := reader.LoadLastCareerRank(ctx, xuid); rerr != nil {
		slog.WarnContext(ctx, "appearance_diag: lecture valeurs servies échouée",
			"player", playerSlug, "err", rerr)
	} else {
		out.row = row
	}
	if status, serr := reader.LoadLastFetchStatus(ctx, xuid); serr != nil {
		slog.WarnContext(ctx, "appearance_diag: lecture last_fetch_status échouée",
			"player", playerSlug, "err", serr)
	} else {
		out.fetchStatus = status
	}
	return out
}

// liveComponents résout les tokens du PROFIL puis diagnostique chaque composant
// via les primitives du Lot E. Tokens absents → auth_required ; fetch KO / vide →
// transient. Aucune erreur remontée (dégradation par verdict).
func (s *AppearanceDiagService) liveComponents(
	ctx context.Context, titleSlug string, profile *domain.PlayerSummary, served servedContext,
) []domain.AppearanceComponentDiagnosis {
	spartanToken, clearanceToken, terr := s.deps.FetchTokens(ctx, profile.XUID, profile.Gamertag)
	if terr != nil || spartanToken == "" {
		slog.WarnContext(ctx, "appearance_diag: tokens du profil indisponibles -> auth_required",
			"player", profile.PlayerSlug, "xuid", profile.XUID, "err", terr)
		return uniformComponents(served.row, haloclient.VerdictAuthRequired)
	}

	fetchCtx, cancel := context.WithTimeout(ctxkeys.WithTitleSlug(ctx, titleSlug), appearanceDiagFetchBudget)
	defer cancel()

	fetcher := s.deps.NewFetcher(spartanToken, clearanceToken, profile.XUID)
	inputs, ferr := fetcher.FetchAppearanceInputs(fetchCtx, profile.XUID)
	if ferr != nil {
		slog.WarnContext(ctx, "appearance_diag: fetch appearance échoué -> transient",
			"player", profile.PlayerSlug, "err", ferr)
		return uniformComponents(served.row, haloclient.VerdictTransient)
	}
	if inputs == nil {
		// 401/403 propriétaire ou réponse vide : indéterminé, se répare au refresh.
		slog.InfoContext(ctx, "appearance_diag: appearance vide (gated/empty) -> transient",
			"player", profile.PlayerSlug)
		return uniformComponents(served.row, haloclient.VerdictTransient)
	}

	banner := fetcher.DiagnoseNameplate(fetchCtx, inputs.EmblemPath, inputs.EmblemConfigID)
	emblem := fetcher.DiagnoseCustomizationImage(fetchCtx, inputs.EmblemPath)
	backdrop := fetcher.DiagnoseCustomizationImage(fetchCtx, inputs.BackdropImagePath)
	tag := haloclient.DiagnoseServiceTag(inputs.ServiceTag)

	return []domain.AppearanceComponentDiagnosis{
		componentFromDiagnosis(domain.AppearanceComponentBanner, served.row, banner),
		componentFromDiagnosis(domain.AppearanceComponentEmblem, served.row, emblem),
		componentFromDiagnosis(domain.AppearanceComponentBackdrop, served.row, backdrop),
		componentFromDiagnosis(domain.AppearanceComponentServiceTag, served.row, tag),
	}
}

// componentFromDiagnosis assemble un composant DTO : la valeur SERVIE (DB) + le
// verdict/detail/served_from issus de la primitive de diagnostic.
func componentFromDiagnosis(
	component string, row *domain.CareerRankRow, diag haloclient.AppearanceDiagnosis,
) domain.AppearanceComponentDiagnosis {
	return domain.AppearanceComponentDiagnosis{
		Component:   component,
		ServedValue: servedValueFor(component, row),
		ServedFrom:  string(diag.ServedFrom),
		Verdict:     string(diag.Verdict),
		Detail:      string(diag.Detail),
	}
}

// uniformComponents produit les 4 composants avec un MÊME verdict (chemins
// dégradés : not_supported / auth_required / transient global). ServedFrom=carry,
// pas de détail technique — seul le verdict porte le sens.
func uniformComponents(row *domain.CareerRankRow, verdict haloclient.Verdict) []domain.AppearanceComponentDiagnosis {
	components := []string{
		domain.AppearanceComponentBanner,
		domain.AppearanceComponentEmblem,
		domain.AppearanceComponentBackdrop,
		domain.AppearanceComponentServiceTag,
	}
	out := make([]domain.AppearanceComponentDiagnosis, 0, len(components))
	for _, component := range components {
		out = append(out, domain.AppearanceComponentDiagnosis{
			Component:   component,
			ServedValue: servedValueFor(component, row),
			ServedFrom:  string(haloclient.ServedFromCarry),
			Verdict:     string(verdict),
			Detail:      "",
		})
	}
	return out
}

// servedValueFor renvoie la valeur actuellement servie pour un composant depuis
// la dernière ligne DB (URL pour banner/emblem/backdrop, texte pour service_tag).
func servedValueFor(component string, row *domain.CareerRankRow) string {
	if row == nil {
		return ""
	}
	switch component {
	case domain.AppearanceComponentBanner:
		return row.BannerImageURL
	case domain.AppearanceComponentEmblem:
		return row.EmblemImageURL
	case domain.AppearanceComponentBackdrop:
		return row.BackdropImageURL
	case domain.AppearanceComponentServiceTag:
		return row.SpartanID
	default:
		return ""
	}
}
