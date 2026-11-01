// Package sync — xbox_client.go : exploitation des succès Xbox (achievements v2). Le protocole
// (adresse, en-têtes, une page par appel, DTO bruts) est dans `internal/haloapi/xboxlive` ; ce fichier
// porte la boucle de pagination et ses journaux, le client HTTP (délai, garde réseau), le jeton et la
// conversion des DTO en PlayerAchievementRaw.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/haloapi/xboxlive"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/netguard"
)

// XboxAchievementsClient abstrait les appels à l'API Xbox Achievements.
// L'interface permet l'injection de mock dans les tests.
type XboxAchievementsClient interface {
	GetPlayerAchievements(ctx context.Context, xuid, lang string) ([]PlayerAchievementRaw, error)
}

// PlayerAchievementRaw contient les données brutes d'un achievement retourné par l'API Xbox.
type PlayerAchievementRaw struct {
	ID             string
	Name           string
	Description    string
	LockedDesc     string
	Gamerscore     int
	ImageURL       string
	IsSecret       bool
	RarityCategory string
	RarityPercent  float64
	Unlocked       bool
	UnlockedAt     time.Time
	// Progression (base game achievements)
	CurrentProgress int
	TargetProgress  int
	// XboxTitleID est le premier TitleAssociation.ID renvoyé par l'API (numérique → string).
	XboxTitleID string
	// ServiceConfigID identifie le jeu de manière unique (SCID Xbox — plus fiable que TitleID).
	ServiceConfigID string
}

// xboxHTTPClient implémente XboxAchievementsClient via l'API Xbox Live.
type xboxHTTPClient struct {
	api         *xboxlive.Client
	auth        xboxlive.Auth
	xboxTitleID string // ex: "1144039928" pour Halo Infinite
}

// NewXboxHTTPClient crée un xboxHTTPClient pour un titre Xbox donné.
// xboxTitleID est l'identifiant numérique Xbox du titre (ex: "1144039928" pour Halo Infinite) —
// utiliser titlePkg.XboxTitleIDFor(slug) pour le résoudre depuis un slug LevelUp.
func NewXboxHTTPClient(xstsResult *auth.XSTSResult, xboxTitleID string) XboxAchievementsClient {
	return &xboxHTTPClient{
		api:         nouvelleXboxAPI(&http.Client{Timeout: 30 * time.Second}),
		auth:        xboxlive.Auth{Autorisation: xstsResult.AuthHeader()},
		xboxTitleID: xboxTitleID,
	}
}

// nouvelleXboxAPI rend le client de la brique dont la sortie passe par `next`, gardée par netguard
// (mode démo : aucune sortie tierce, cf. internal/platform/netguard).
func nouvelleXboxAPI(next *http.Client) *xboxlive.Client {
	return xboxlive.New(netguard.Doer{Surface: "xbox_achievements.get", Next: next})
}

// GetPlayerAchievements récupère tous les achievements Halo Infinite pour un joueur.
// lang doit être un tag BCP-47 (ex: "en-US", "fr-FR").
// La pagination est gérée automatiquement via pagingInfo.continuationToken.
func (c *xboxHTTPClient) GetPlayerAchievements(ctx context.Context, xuid, lang string) ([]PlayerAchievementRaw, error) {
	var all []PlayerAchievementRaw
	skipItems := 0

	slog.DebugContext(ctx, "xbox_client: début récupération achievements", "xuid", xuid, "lang", lang)
	for {
		batch, continuationToken, err := c.fetchPage(ctx, xuid, lang, skipItems)
		if err != nil {
			slog.WarnContext(ctx, "xbox_client: erreur fetchPage",
				"xuid", xuid, "lang", lang, "skip", skipItems, "err", err)
			return nil, err
		}
		all = append(all, batch...)
		slog.DebugContext(ctx, "xbox_client: page achievements récupérée",
			"xuid", xuid, "lang", lang,
			"batch", len(batch), "total", len(all),
			"continuation", continuationToken != "",
		)
		if continuationToken == "" {
			break
		}
		skipItems += len(batch)
	}

	return all, nil
}

// fetchPage récupère une page d'achievements (avec skipItems pour la pagination) et la convertit.
func (c *xboxHTTPClient) fetchPage(ctx context.Context, xuid, lang string, skipItems int) ([]PlayerAchievementRaw, string, error) {
	page := xboxlive.Page{TitleID: c.xboxTitleID, SkipItems: skipItems}
	rep, err := c.api.GetSuccesPage(ctx, c.auth, xuid, lang, page)
	if err != nil {
		return nil, "", erreurXbox(err)
	}
	items := make([]PlayerAchievementRaw, 0, len(rep.Achievements))
	for _, a := range rep.Achievements {
		items = append(items, parseAchievementItem(a))
	}
	return items, rep.PagingInfo.ContinuationToken, nil
}

// erreurXbox compose l'erreur rendue depuis celle de la brique : sortie refusée (mode démo) telle
// quelle, échec d'émission `xbox_client: HTTP do`, le reste `xbox_client` ; `errors.As` voit les types.
func erreurXbox(err error) error {
	var transport *xboxlive.ErreurTransport
	switch {
	case errors.Is(err, haloapi.ErrDefinitif):
		return err
	case errors.As(err, &transport):
		return fmt.Errorf("xbox_client: HTTP do: %w", err)
	default:
		return fmt.Errorf("xbox_client: %w", err)
	}
}

// parseAchievementItem convertit un item API en PlayerAchievementRaw.
func parseAchievementItem(a xboxlive.Succes) PlayerAchievementRaw {
	raw := PlayerAchievementRaw{
		ID:              a.ID,
		Name:            a.Name,
		Description:     a.Description,
		LockedDesc:      a.LockedDescription,
		IsSecret:        a.IsSecret,
		RarityCategory:  a.Rarity.CurrentCategory,
		RarityPercent:   a.Rarity.CurrentPercentage,
		Unlocked:        a.ProgressState == "Achieved",
		ServiceConfigID: a.ServiceConfigID,
	}

	// Gamerscore depuis Rewards
	for _, r := range a.Rewards {
		if r.Type == "Gamerscore" {
			if v, err := strconv.Atoi(r.Value); err == nil {
				raw.Gamerscore = v
			}
			break
		}
	}

	// Image principale
	for _, m := range a.MediaAssets {
		if m.Type == "Icon" {
			raw.ImageURL = m.URL
			break
		}
	}

	// Xbox title ID depuis le premier TitleAssociation
	if len(a.TitleAssociations) > 0 {
		raw.XboxTitleID = strconv.Itoa(a.TitleAssociations[0].ID)
	}

	// Date de déverrouillage
	if a.Progression.TimeUnlocked != "" && a.Progression.TimeUnlocked != "0001-01-01T00:00:00Z" {
		if t, err := time.Parse(time.RFC3339, a.Progression.TimeUnlocked); err == nil {
			raw.UnlockedAt = t
		}
	}

	// Progression (premier requirement disponible)
	if len(a.Progression.Requirements) > 0 {
		req := a.Progression.Requirements[0]
		if v, err := strconv.Atoi(req.Current); err == nil {
			raw.CurrentProgress = v
		}
		if v, err := strconv.Atoi(req.Target); err == nil {
			raw.TargetProgress = v
		}
	}

	return raw
}
