package native

import (
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/notifications"
)

// Kind classe une notification de la cloche pour le relais natif : un type court, jamais un libellé. La
// coquille ne s'en sert pas pour choisir un texte (il arrive composé), seulement pour le journal.
type Kind string

// Les types relayés. Une catégorie absente de [KindOf] n'est pas relayée : la cloche en garde la
// trace, Windows n'est pas dérangé pour un conseil du coach.
const (
	KindSync      Kind = "sync"
	KindSyncError Kind = "sync_error"
	KindChallenge Kind = "challenge"
	KindStreak    Kind = "streak"
	KindRecord    Kind = "record"
	KindRank      Kind = "rank"
)

// kindByCategory : les catégories de la cloche relayées, par type. Fin de synchronisation, erreur de
// synchronisation, défis et objectifs, séries, records, rang (carrière et compétitif).
var kindByCategory = map[notifications.Category]Kind{
	notifications.CategoryMatchSynced:        KindSync,
	notifications.CategorySyncError:          KindSyncError,
	notifications.CategoryChallengeAdded:     KindChallenge,
	notifications.CategoryChallengeCompleted: KindChallenge,
	notifications.CategoryObjectiveCompleted: KindChallenge,
	notifications.CategoryStreakMilestone:    KindStreak,
	notifications.CategoryPersonalRecord:     KindRecord,
	notifications.CategoryCareerRank:         KindRank,
	notifications.CategorySkillTier:          KindRank,
}

// KindOf rend le type d'une catégorie, faux si elle n'est pas relayée.
func KindOf(c notifications.Category) (Kind, bool) {
	k, ok := kindByCategory[c]
	return k, ok
}

// Kinds rend les types relayés (tests, documentation), sans doublon.
func Kinds() []Kind {
	return []Kind{KindSync, KindSyncError, KindChallenge, KindStreak, KindRecord, KindRank}
}

// Compose rend le titre et le corps de la notification Windows d'un type, dans la langue loc. Le texte
// vient du catalogue des messages (desktop.toml), jamais d'un littéral : une langue ajoutée à
// config/locales.toml doit y être complète.
func Compose(loc i18n.Locale, k Kind) (title, body string) {
	switch k {
	case KindSync:
		return i18n.Msg(loc, "desktop.notify.sync_title", nil), i18n.Msg(loc, "desktop.notify.sync_body", nil)
	case KindSyncError:
		return i18n.Msg(loc, "desktop.notify.sync_error_title", nil), i18n.Msg(loc, "desktop.notify.sync_error_body", nil)
	case KindChallenge:
		return i18n.Msg(loc, "desktop.notify.challenge_title", nil), i18n.Msg(loc, "desktop.notify.challenge_body", nil)
	case KindStreak:
		return i18n.Msg(loc, "desktop.notify.streak_title", nil), i18n.Msg(loc, "desktop.notify.streak_body", nil)
	case KindRecord:
		return i18n.Msg(loc, "desktop.notify.record_title", nil), i18n.Msg(loc, "desktop.notify.record_body", nil)
	case KindRank:
		return i18n.Msg(loc, "desktop.notify.rank_title", nil), i18n.Msg(loc, "desktop.notify.rank_body", nil)
	}
	return "", ""
}
