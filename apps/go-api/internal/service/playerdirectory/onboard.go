// Package playerdirectory — onboard.go : le SEUL chemin de création d'un profil
// joueur (ADR 0035 D4).
//
// POURQUOI UN CHEMIN UNIQUE. Le 2026-07-23, trois des quatre registres d'une
// identité ont été écrits sans le quatrième : le SSO a créé le compte, persisté
// les credentials et ajouté le joueur au watcher, mais aucun profil n'a jamais
// été déclaré — seul le wizard de mise en place en crée. Deux heures plus tard un
// sync écrivait 25 matchs pour un joueur que plus rien en aval ne savait résoudre.
// La cause n'est pas un oubli ponctuel : c'est qu'il y avait DEUX endroits où l'on
// décidait qu'un joueur existe. Il n'y en a plus qu'un.
//
// L'ORDRE N'EST PAS NÉGOCIABLE : profil d'abord, watcher ensuite. Depuis l'étape 2
// du plan, `watcher.Daemon.AddPlayer` porte une porte qui LIT `db_profiles.json`
// (ADR 0035 D3) : notifier le watcher avant d'avoir écrit le profil se solderait
// par un refus systématique. Le test `TestOnboard_ProfilAvantWatcher` tient ce lien.
//
// UN ÉCHEC DE NOTIFICATION N'EST PAS UN ÉCHEC DE MISE EN PLACE : le profil est la
// vérité, et `initPlayers` reprend la liste des profils suivis à chaque démarrage
// du daemon. L'échec est journalisé en ERROR et rendu dans WatcherNotified=false —
// jamais avalé, jamais propagé en erreur.
package playerdirectory

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// ProfileCreator écrit le profil de suivi dans `db_profiles.json` et crée le
// dossier du joueur. Implémenté par *service.ProfileService, injecté au câblage :
// l'annuaire n'écrit jamais lui-même un registre, il les compose.
type ProfileCreator interface {
	CreatePlayer(req domain.CreatePlayerProfileRequest) (playerKey string, warnings []string, err error)
}

// WatcherNotifier est la prise en charge live d'un joueur. Implémenté par
// watcher.DaemonController (le daemon du process serveur) ; absent en CLI.
type WatcherNotifier interface {
	IsRunning() bool
	AddPlayer(ctx context.Context, p domain.PlayerSummary) error
}

// ErrOnboardNoCreator : l'annuaire a été construit sans créateur de profil (Deps
// .Creator nil). Refuser franchement vaut mieux que rendre un succès sans profil —
// c'est exactement l'état qui a produit l'incident du 2026-07-23.
var ErrOnboardNoCreator = errors.New("player_directory: createur de profil absent")

// ErrOnboardInvalidGamertag : un profil est indexe par son gamertag, il ne peut
// pas etre vide.
var ErrOnboardInvalidGamertag = errors.New("player_directory: gamertag vide")

// Onboard crée le profil de suivi d'un joueur, puis l'ajoute au suivi live si le
// daemon tourne. Voir l'en-tête du fichier pour l'ordre et la politique d'erreur.
func (d *Directory) Onboard(ctx context.Context, req domain.OnboardRequest) (domain.OnboardResult, error) {
	gamertag := strings.TrimSpace(req.Gamertag)
	if gamertag == "" {
		return domain.OnboardResult{}, ErrOnboardInvalidGamertag
	}
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if d.creator == nil {
		slog.ErrorContext(ctx, "directory: création de profil impossible — aucun créateur câblé",
			"gamertag", gamertag, "title_slug", titleSlug)
		return domain.OnboardResult{}, ErrOnboardNoCreator
	}

	playerKey, warnings, err := d.creator.CreatePlayer(domain.CreatePlayerProfileRequest{
		Gamertag:          gamertag,
		XUID:              req.XUID,
		TitleSlug:         titleSlug,
		InitialMaxMatches: req.InitialMaxMatches,
	})
	if err != nil {
		slog.ErrorContext(ctx, "directory: création de profil échouée", "err", err,
			"gamertag", gamertag, "xuid", req.XUID, "title_slug", titleSlug, "actor", req.ActorUsername)
		return domain.OnboardResult{}, err
	}
	slog.InfoContext(ctx, "directory: profil créé", "xuid", req.XUID, "gamertag", gamertag,
		"player_key", playerKey, "title_slug", titleSlug, "actor", req.ActorUsername)

	res := domain.OnboardResult{PlayerKey: playerKey, Warnings: warnings}
	if d.fs != nil {
		res.DBPath = d.fs.PlayerDBPath(titleSlug, playerKey)
		res.DBCreated = d.fs.PlayerDBExists(titleSlug, playerKey)
	}
	res.WatcherNotified = d.notifyWatcher(ctx, domain.PlayerSummary{
		PlayerSlug:        playerKey,
		Gamertag:          gamertag,
		XUID:              req.XUID,
		WaypointPlayer:    gamertag,
		TitleSlug:         titleSlug,
		SyncEnabled:       true,
		InitialMaxMatches: req.InitialMaxMatches,
	})
	return res, nil
}

// notifyWatcher ajoute le joueur au suivi live. Rend false — sans erreur — dans
// tous les cas où le suivi live n'est pas la bonne réponse MAINTENANT : pas de
// watcher dans ce process, daemon arrêté, ou joueur sans identité Xbox (le
// watcher interroge la présence PAR XUID : sans xuid il n'y a rien à suivre, et
// un profil « manuel » est un cas normal, pas une panne).
func (d *Directory) notifyWatcher(ctx context.Context, p domain.PlayerSummary) bool {
	switch {
	case d.watcher == nil:
		slog.DebugContext(ctx, "directory: pas de watcher dans ce process — suivi live non demandé",
			"gamertag", p.Gamertag, "title_slug", p.TitleSlug)
		return false
	case !d.watcher.IsRunning():
		slog.InfoContext(ctx, "directory: watcher arrêté — le joueur sera pris au prochain démarrage",
			"gamertag", p.Gamertag, "title_slug", p.TitleSlug)
		return false
	case p.XUID == "":
		slog.InfoContext(ctx, "directory: profil sans identité Xbox — pas de suivi live",
			"gamertag", p.Gamertag, "title_slug", p.TitleSlug)
		return false
	}

	if err := d.watcher.AddPlayer(ctx, p); err != nil {
		slog.ErrorContext(ctx, "directory: watcher non notifié — le profil fait foi, le daemon reprendra le joueur au prochain démarrage",
			"err", err, "gamertag", p.Gamertag, "xuid", p.XUID, "title_slug", p.TitleSlug)
		return false
	}
	slog.InfoContext(ctx, "directory: joueur ajouté au suivi live", "gamertag", p.Gamertag,
		"xuid", p.XUID, "title_slug", p.TitleSlug)
	return true
}
