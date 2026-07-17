package external

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/observability"
)

// Compteurs expvar (convention "<categorie>_<souscle>" snake_case).
const (
	counterSent   = "notifications_external_coach_sent"
	counterFailed = "notifications_external_coach_failed"
)

// Config paramètre un Dispatcher. AppSettingsPath est relu à CHAQUE Forward :
// le webhook et le flag d'activation restent réactifs aux PATCH /settings sans
// reconstruire le Service (mis en cache par xuid pour la vie du process).
type Config struct {
	AppSettingsPath string                   // chemin app_settings.json (source du webhook + flags)
	Player          string                   // gamertag injecté dans l'embed
	XUID            string                   // xuid du joueur (traçage/log)
	TitleSlug       string                   // titre du joueur (footer title-aware + lang)
	AppBaseURL      string                   // base URL publique optionnelle (lien app dans l'embed)
	Forwarded       []notifications.Category // catégories relayées ; nil → DefaultForwardedCategories()
	Notifier        ExternalNotifier         // canal de sortie ; nil → DiscordWebhookNotifier{}
}

// Dispatcher implémente notifications.ExternalForwarder. Il filtre par catégorie
// et par activation (relue à chaud), puis relaie en asynchrone (goroutine +
// recover) — best-effort strict : jamais bloquant, jamais paniquant pour le flux
// d'émission in-app.
type Dispatcher struct {
	appSettingsPath string
	player          string
	xuid            string
	titleSlug       string
	appBaseURL      string
	forwarded       map[notifications.Category]struct{}
	notifier        ExternalNotifier
}

// NewDispatcher construit un Dispatcher à partir d'une Config (défauts appliqués).
func NewDispatcher(cfg Config) *Dispatcher {
	forwardedList := cfg.Forwarded
	if forwardedList == nil {
		forwardedList = DefaultForwardedCategories()
	}
	set := make(map[notifications.Category]struct{}, len(forwardedList))
	for _, c := range forwardedList {
		set[c] = struct{}{}
	}
	notifier := cfg.Notifier
	if notifier == nil {
		notifier = DiscordWebhookNotifier{}
	}
	return &Dispatcher{
		appSettingsPath: cfg.AppSettingsPath,
		player:          cfg.Player,
		xuid:            cfg.XUID,
		titleSlug:       cfg.TitleSlug,
		appBaseURL:      cfg.AppBaseURL,
		forwarded:       set,
		notifier:        notifier,
	}
}

// Forward relaie la notification si (a) sa catégorie est forwardée ET (b) le relais
// coach est actif (discord_notifications_enabled + discord_notify_coach + webhook
// présent). Inactif → retour silencieux (zéro log de bruit). Actif → envoi async.
func (d *Dispatcher) Forward(ctx context.Context, n *notifications.Notification) {
	if d == nil || n == nil {
		return
	}
	if _, ok := d.forwarded[n.Category]; !ok {
		return // catégorie non relayée (sync, média, version…) — silencieux
	}
	ncfg := notify.LoadNotifyConfig(d.appSettingsPath)
	if ncfg.WebhookURL == "" || !ncfg.NotifyCoach {
		return // relais inactif (opt-in non activé) — silencieux, zéro bruit
	}

	extN := ExternalNotification{
		Category:  string(n.Category),
		Severity:  string(n.Severity),
		Player:    d.player,
		XUID:      d.xuid,
		TitleSlug: d.titleSlug,
		Params:    decodeParams(n.Params),
		AppURL:    d.appLink(n.TargetRoute),
		Lang:      ncfg.Lang,
	}
	webhookURL := ncfg.WebhookURL

	// Détache le cycle de vie de la requête/sync appelante (ctx potentiellement
	// annulé au retour du handler) : le relais externe survit à l'appelant, borné
	// par le timeout interne du notifier.
	detached := context.WithoutCancel(ctx)
	go d.deliver(detached, webhookURL, extN)
}

// deliver exécute l'envoi externe sous recover (best-effort strict) + compteurs.
func (d *Dispatcher) deliver(ctx context.Context, webhookURL string, n ExternalNotification) {
	defer func() {
		if r := recover(); r != nil {
			observability.IncCounter(counterFailed)
			slog.ErrorContext(ctx, "notifications.external: panic relais coach récupéré",
				"category", n.Category, "recover", fmt.Sprintf("%v", r))
		}
	}()
	if err := d.notifier.Notify(ctx, webhookURL, n); err != nil {
		observability.IncCounter(counterFailed)
		slog.WarnContext(ctx, "notifications.external: relais coach échoué",
			"category", n.Category, "err", err)
		return
	}
	observability.IncCounter(counterSent)
	slog.InfoContext(ctx, "notifications.external: relais coach envoyé",
		"category", n.Category, "player", n.Player)
}

// appLink construit un lien app absolu si une base URL publique est configurée et
// que la notification porte une route cible. "" sinon (pas de champ lien).
func (d *Dispatcher) appLink(targetRoute string) string {
	if d.appBaseURL == "" || targetRoute == "" {
		return ""
	}
	return strings.TrimRight(d.appBaseURL, "/") + "/" + strings.TrimLeft(targetRoute, "/")
}

// decodeParams déserialise les params JSON de la notification en map (best-effort :
// payload vide/illisible → map nil, jamais d'erreur propagée).
func decodeParams(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// LogBootState émet UNE ligne INFO au démarrage indiquant l'état du relais coach
// externe (comme les autres sous-systèmes). Ne loggue jamais le webhook (secret).
func LogBootState(ctx context.Context, appSettingsPath string) {
	ncfg := notify.LoadNotifyConfig(appSettingsPath)
	active := ncfg.WebhookURL != "" && ncfg.NotifyCoach
	slog.InfoContext(ctx, "notifications.external: relais coach Discord",
		"actif", active,
		"categories", len(DefaultForwardedCategories()))
}

// Compile-time check : Dispatcher satisfait le port.
var _ notifications.ExternalForwarder = (*Dispatcher)(nil)
