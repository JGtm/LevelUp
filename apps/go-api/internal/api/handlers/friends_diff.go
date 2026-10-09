// friends_diff.go — diff d'une liste d'amis et notifications `friend_added`.
//
// Extrait de settings.go le 2026-09-15 (amis par joueur) : la liste d'amis n'est
// plus un réglage global d'instance modifié par PATCH /settings, mais la
// propriété d'un profil joueur écrite par PUT /players/{slug}/friends. Le diff
// et l'émission de notifications sont inchangés, seul leur appelant a changé.
package handlers

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
)

// newFriendsAdded retourne les gamertags présents dans next mais pas dans prev
// (set diff case-insensitive + trim, cohérent avec friendGamertagsChanged).
// Retourne les gamertags **dans la casse next** (préserve la saisie utilisateur).
func newFriendsAdded(prev, next []string) []string {
	prevSet := make(map[string]struct{}, len(prev))
	for _, gt := range prev {
		prevSet[normalizeGamertag(gt)] = struct{}{}
	}
	var added []string
	for _, gt := range next {
		if _, ok := prevSet[normalizeGamertag(gt)]; !ok {
			added = append(added, gt)
		}
	}
	return added
}

// friendGamertagsChanged compare deux listes (set-equality, case-insensitive).
func friendGamertagsChanged(prev, next []string) bool {
	if len(prev) != len(next) {
		return true
	}
	set := make(map[string]struct{}, len(prev))
	for _, gt := range prev {
		set[normalizeGamertag(gt)] = struct{}{}
	}
	for _, gt := range next {
		if _, ok := set[normalizeGamertag(gt)]; !ok {
			return true
		}
	}
	return false
}

func normalizeGamertag(gt string) string {
	return strings.ToLower(strings.TrimSpace(gt))
}

// emitFriendsAdded émet 1 notification friend_added (in-app) par nouveau
// gamertag, et déclenche le webhook Discord si activé. §6.A + §6.B.
// Best-effort sur les 2 canaux : warn log + continue, jamais bloquant.
//
// `slug` est le joueur dont la liste a changé (destinataire des notifications) ;
// `appSettingsPath` sert au chargement de la config Discord.
func emitFriendsAdded(ctx context.Context, notifierFor NotificationsEmitterFactory,
	appSettingsPath, slug, source string, added []string,
) {
	if notifierFor == nil {
		return
	}
	em, err := notifierFor(ctx, slug)
	if err != nil || em == nil {
		slog.WarnContext(ctx, "notifications: friend_added emitter factory failed",
			"player_slug", slug, "err", err)
		return
	}
	// Charger NotifyConfig une fois pour la batch (évite N reads disk).
	notifyCfg := notify.LoadNotifyConfig(appSettingsPath)
	// PMT-11 : libellés Discord du titre courant (outcomes + footer). Failsafe Halo.
	notifyCfg.Labels = notify.LabelsForSlug(ctxkeys.TitleSlug(ctx))
	for _, gt := range added {
		if err := em.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryFriendAdded,
			Severity: notifications.SeverityInfo,
			TitleKey: "notif.friend_added.title",
			BodyKey:  "notif.friend_added.body",
			Params:   map[string]any{"gamertag": gt},
			Source:   source,
		}); err != nil {
			slog.WarnContext(ctx, "notifications: friend_added emit", "gamertag", gt, "err", err)
		}
		// §6.B Discord : webhook failsafe (no-op si webhook vide / NotifyFriends off).
		go notify.NotifyFriendAdded(notifyCfg, gt)
	}
}
