package historystate

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"

	"levelup/go-api/internal/sync/syncmeta"
)

// StreakKey est la clé de `sync_meta` qui porte, par joueur, le compte des passes de rattrapage
// consécutives en échec sur un même point de la liste : sans lui, un seul échec durable garderait le
// joueur en rattrapage pour toujours. Neutre : l'appelant nomme le point ([Streak.Subject]), ce paquet
// ne sait pas ce qu'est une page ou un match.
//
// Valeur : un objet JSON `{"subject":"...","attempts":N,"skipped":K,"last_error":"..."}`. Une valeur
// vide (compte effacé) ou illisible vaut « aucun échec en cours ». Comme le marque-page, l'effacement
// range une valeur vide au lieu de supprimer la ligne (règle anti-ART).
const StreakKey = "history_failure_streak"

// Streak est le compte d'échecs consécutifs d'un point du rattrapage.
type Streak struct {
	// Subject identifie le point en échec, choisi par l'appelant (ex. l'identifiant du dernier match
	// traité du marque-page : tant que le marque-page ne bouge pas, c'est le même point).
	Subject string
	// Attempts est le nombre de passes consécutives qui ont échoué sur la première unité non abandonnée
	// après le point.
	Attempts int
	// Skipped est le nombre d'unités déjà abandonnées juste après le point : la passe suivante les saute.
	Skipped int
	// LastError est le texte de la dernière erreur, pour le diagnostic seulement.
	LastError string
}

// streakWire est la forme rangée dans `sync_meta`.
type streakWire struct {
	Subject   string `json:"subject"`
	Attempts  int    `json:"attempts"`
	Skipped   int    `json:"skipped,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

// LoadStreak lit le compte d'échecs du joueur pour `subject`. Un compte rangé pour un autre sujet,
// absent, effacé ou illisible rend un compte vide (zéro échec) : la lecture en échec est journalisée.
func LoadStreak(ctx context.Context, db *sql.DB, subject string) Streak {
	empty := Streak{Subject: subject}
	if db == nil {
		return empty
	}
	raw, found, err := syncmeta.Get(ctx, db, StreakKey)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture du compte d'échecs du rattrapage échouée — compte remis à zéro", "err", err)
		return empty
	}
	if !found || strings.TrimSpace(raw) == "" {
		return empty
	}
	var w streakWire
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		slog.WarnContext(ctx, "sync: compte d'échecs du rattrapage illisible — compte remis à zéro", "err", err)
		return empty
	}
	if w.Subject != subject || w.Attempts < 0 || w.Skipped < 0 {
		return empty
	}
	return Streak(w)
}

// SaveStreak range le compte d'échecs du joueur. L'appelant tient le bail d'écriture du joueur (voir le
// doc du paquet). Rend vrai quand il est rangé, faux (journalisé) sinon.
func SaveStreak(ctx context.Context, db *sql.DB, s Streak) bool {
	if db == nil {
		slog.WarnContext(ctx, "sync: compte d'échecs du rattrapage non rangé — base joueur indisponible")
		return false
	}
	raw, err := json.Marshal(streakWire(s))
	if err != nil {
		slog.WarnContext(ctx, "sync: compte d'échecs du rattrapage non sérialisable", "err", err)
		return false
	}
	if err := syncmeta.Set(ctx, db, StreakKey, string(raw)); err != nil {
		slog.WarnContext(ctx, "sync: écriture du compte d'échecs du rattrapage échouée", "err", err)
		return false
	}
	return true
}

// ClearStreak efface le compte d'échecs du joueur (valeur vide). L'appelant tient le bail d'écriture du
// joueur. Sans compte rangé, rien n'est écrit. Rend vrai quand il n'y a plus de compte.
func ClearStreak(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return false
	}
	raw, found, err := syncmeta.Get(ctx, db, StreakKey)
	if err != nil {
		slog.WarnContext(ctx, "sync: lecture du compte d'échecs du rattrapage échouée avant effacement", "err", err)
		return false
	}
	if !found || strings.TrimSpace(raw) == "" {
		return true
	}
	if err := syncmeta.Set(ctx, db, StreakKey, ""); err != nil {
		slog.WarnContext(ctx, "sync: effacement du compte d'échecs du rattrapage échoué", "err", err)
		return false
	}
	return true
}
