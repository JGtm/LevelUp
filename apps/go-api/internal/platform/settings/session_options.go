package settings

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// SessionOptions est la règle unique de découpage en sessions d'un titre : les options
// d'analysis.DefaultSessionOptions, surchargées par le réglage résolu du titre (global puis
// overlay, cf. ResolveForTitle) — écart (session_gap_minutes), coupure sur changement
// classé/non classé (session_split_on_ranked_change) et réaction aux changements de
// coéquipiers (session_team_change_mode). Les champs que le réglage ne porte pas (heure de
// coupure, algorithme) gardent leur valeur par défaut.
//
// Tous les chemins qui écrivent session_id (synchro, recalcul complet, backfill
// d'enrichissement, import OpenSpartan) et la lecture à la volée de GET /pages/sessions
// partent d'ici : deux chemins qui calculeraient leurs options chacun de leur côté
// découperaient les mêmes matchs différemment. Garde-rail :
// archlint/session_options_single_source_test.go.
//
// Store nil ou réglage illisible : les défauts s'appliquent, journalisés en WARN avant.
// Les amis (FriendsXUIDs) ne sont pas posés ici : chaque appelant les résout pour le joueur.
func SessionOptions(ctx context.Context, store *Store, overlayPath string) domain.SessionComputeOptions {
	opts := analysis.DefaultSessionOptions()
	if store == nil {
		slog.WarnContext(ctx, "sessions: réglage indisponible (aucun store), règle par défaut appliquée",
			"team_change_mode", opts.TeamChangeMode, "gap_minutes", opts.GapMinutes)
		return opts
	}
	cfg, err := store.ResolveForTitle(overlayPath)
	if err != nil || cfg == nil {
		slog.WarnContext(ctx, "sessions: lecture du réglage échouée, règle par défaut appliquée",
			"overlay", overlayPath, "err", err)
		return opts
	}
	return applySessionSettings(ctx, opts, cfg)
}

// applySessionSettings pose le réglage sur les options par défaut. Écart <= 0 : l'écart par
// défaut reste. Mode de coéquipiers vide ou inconnu (le PATCH n'accepte que ignore, group et
// friends) : le mode par défaut reste, journalisé s'il est inconnu.
func applySessionSettings(ctx context.Context, opts domain.SessionComputeOptions, cfg *AppSettings) domain.SessionComputeOptions {
	if cfg.SessionGapMinutes > 0 {
		opts.GapMinutes = cfg.SessionGapMinutes
	}
	opts.SplitOnRankedChange = cfg.SessionSplitOnRankedChange
	switch mode := domain.TeamChangeMode(cfg.SessionTeamChangeMode); mode {
	case domain.TeamChangeModeIgnore, domain.TeamChangeModeGroup, domain.TeamChangeModeFriends:
		opts.TeamChangeMode = mode
	case "":
	default:
		slog.WarnContext(ctx, "sessions: mode de coéquipiers inconnu dans le réglage, mode par défaut appliqué",
			"session_team_change_mode", cfg.SessionTeamChangeMode, "default", opts.TeamChangeMode)
	}
	return opts
}

// SessionOptionsLoader rend la règle de SessionOptions sous forme de chargeur, relu à chaque
// appel : un réglage modifié s'applique au calcul suivant sans redémarrage.
func SessionOptionsLoader(store *Store, overlayPath string) func(context.Context) domain.SessionComputeOptions {
	return func(ctx context.Context) domain.SessionComputeOptions {
		return SessionOptions(ctx, store, overlayPath)
	}
}

// SessionOptionsForConfig rend la règle d'un titre depuis la configuration : le même
// fichier de réglages que le serveur (cfg.AppSettingsPath) et l'overlay du titre
// (cfg.TitleSettingsPath, celui de la fixture en démo), relus à chaque appel. Pour les sites
// (CLI, moteurs construits à la demande, import) qui n'ont pas de Store en main ; un
// SyncEngine la reçoit par SyncEngine.WithSessionOptionsFromConfig. cfg nil : défauts
// journalisés.
func SessionOptionsForConfig(ctx context.Context, cfg *config.AppConfig, titleSlug string) domain.SessionComputeOptions {
	if cfg == nil {
		return SessionOptions(ctx, nil, "")
	}
	return SessionOptions(ctx, NewStore(cfg.AppSettingsPath), cfg.TitleSettingsPath(titleSlug))
}
