// Package external implémente le relais OPT-IN des notifications coach vers un
// canal externe (webhook Discord). Il branche le port
// notifications.ExternalForwarder au point d'émission in-app et délègue la
// construction d'embed + l'envoi HTTP au package internal/notify (client Discord
// unique du projet — aucune duplication du transport ni de la sanitation du
// secret).
//
// Décision produit (2026-07-17) : OFF par défaut. Aucune émission externe tant
// que l'utilisateur n'a pas explicitement activé discord_notifications_enabled ET
// discord_notify_coach. Ce n'est PAS un kill-switch au sens de CLAUDE.md n°11
// (feature livrée OFF « pour plus tard ») : émettre vers un service tiers est une
// décision vie privée de l'utilisateur — l'opt-in est volontaire et permanent.
package external

import (
	"context"
	"errors"
	"time"

	"levelup/go-api/internal/notify"
)

// DefaultDiscordTimeout borne la durée d'un POST webhook coach (timeout court,
// best-effort : un canal externe lent ne doit jamais retenir une goroutine).
const DefaultDiscordTimeout = 5 * time.Second

// ErrExternalSendFailed signale un échec d'envoi externe. Sentinel STATIQUE :
// ne contient jamais l'URL du webhook (secret) ni de détail réseau — le package
// notify a déjà logué (secret expurgé) le détail transport.
var ErrExternalSendFailed = errors.New("external: webhook send failed")

// ExternalNotification est la vue title-agnostic d'une notification relayée. Elle
// ne porte AUCUN libellé FR/EN utilisateur (ceux-ci vivent côté front, clés i18n) :
// seulement la catégorie, la severity, l'identité joueur et les params métier.
type ExternalNotification struct {
	Category  string
	Severity  string
	Player    string // gamertag
	XUID      string
	TitleSlug string
	Params    map[string]any
	AppURL    string // lien app optionnel ("" = aucun)
	Lang      string
}

// ExternalNotifier est l'abstraction d'un canal de sortie externe. Best-effort :
// Notify borne sa propre durée et ne panique jamais ; il retourne une erreur pour
// permettre au dispatcher de compter/loguer l'échec.
type ExternalNotifier interface {
	Notify(ctx context.Context, webhookURL string, n ExternalNotification) error
}

// DiscordWebhookNotifier envoie un embed Discord (POST JSON) vers l'URL webhook.
// Réutilise notify.BuildCoachEmbed + notify.SendWebhookCtx (client + sanitation
// du secret partagés). Timeout court configurable (défaut DefaultDiscordTimeout).
type DiscordWebhookNotifier struct {
	// Timeout borne la durée du POST. <= 0 → DefaultDiscordTimeout.
	Timeout time.Duration
}

// Notify construit l'embed coach et l'envoie au webhook en respectant Timeout.
// webhookURL vide → ErrExternalSendFailed (aucun POST). Aucun secret n'est loggué
// ici : le détail transport est logué (expurgé) par notify.SendWebhookCtx.
func (d DiscordWebhookNotifier) Notify(ctx context.Context, webhookURL string, n ExternalNotification) error {
	if webhookURL == "" {
		return ErrExternalSendFailed
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = DefaultDiscordTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	embed := notify.BuildCoachEmbed(notify.CoachEmbedInput{
		Category: n.Category,
		Severity: n.Severity,
		Player:   n.Player,
		Params:   n.Params,
		AppURL:   n.AppURL,
		Lang:     n.Lang,
	}, notify.LabelsForSlug(n.TitleSlug))

	if notify.SendWebhookCtx(cctx, webhookURL, notify.WebhookPayload{Embeds: []notify.Embed{embed}}) {
		return nil
	}
	return ErrExternalSendFailed
}

// Compile-time check.
var _ ExternalNotifier = DiscordWebhookNotifier{}
