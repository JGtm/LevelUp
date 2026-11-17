package config

// env_reads_sync.go — réglages de sync et de notification lus dans l'environnement, hors
// AppConfig pour la même raison que env_reads.go : leurs appelants (moteur de sync, évaluation de
// la préparation des instantanés, câblage des notifications) tournent sans AppConfig sous la
// main. Chaque variable se lit ICI ; le garde-rail `env_centralization_test.go` interdit de la
// relire ailleurs. Une valeur absente ou invalide rend la valeur par défaut de l'appelant, qui en
// reste propriétaire.

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Noms des variables lues par ce fichier.
const (
	envPublicBaseURL        = "LEVELUP_PUBLIC_BASE_URL"
	envFilmRetryWindowHours = "LEVELUP_FILM_RETRY_WINDOW_HOURS"
	envFreshFilmRetry       = "LEVELUP_FRESH_FILM_RETRY"
	envSnapshotGraceHours   = "LEVELUP_SNAPSHOT_GRACE_HOURS"
)

// PublicBaseURLFromEnv rend la base publique de l'application demandée par
// LEVELUP_PUBLIC_BASE_URL, sans espaces de bord ni « / » final. Vide = aucune base configurée (cas
// nominal en local) : les messages de notification sortent alors sans lien profond. Le relais
// externe et la notification groupée des rejeux lisent la même valeur ici.
func PublicBaseURLFromEnv() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv(envPublicBaseURL)), "/")
}

// FilmRetryWindowFromEnv rend la fenêtre de reprise d'un film absent demandée par
// LEVELUP_FILM_RETRY_WINDOW_HOURS (entier d'heures strictement positif) ; def sinon.
func FilmRetryWindowFromEnv(def time.Duration) time.Duration {
	return hoursFromEnv(envFilmRetryWindowHours, def)
}

// SnapshotGraceWindowFromEnv rend l'ancienneté de forçage de la préparation d'un instantané
// demandée par LEVELUP_SNAPSHOT_GRACE_HOURS (entier d'heures strictement positif) ; def sinon.
func SnapshotGraceWindowFromEnv(def time.Duration) time.Duration {
	return hoursFromEnv(envSnapshotGraceHours, def)
}

// FreshFilmRetryDelaysFromEnv rend les intervalles de reprise du fetch du film d'un match frais
// demandés par LEVELUP_FRESH_FILM_RETRY : "0" rend nil (plus d'attente) ; une liste de secondes
// strictement positives séparées par des virgules rend ces intervalles (les éléments invalides sont
// ignorés) ; une variable absente, vide ou sans aucun élément valide rend def.
func FreshFilmRetryDelaysFromEnv(def []time.Duration) []time.Duration {
	v := os.Getenv(envFreshFilmRetry)
	if v == "0" {
		return nil
	}
	if v != "" {
		var out []time.Duration
		for _, p := range strings.Split(v, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
				out = append(out, time.Duration(n)*time.Second)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return def
}

// hoursFromEnv lit une durée en heures strictement positives ; def si la variable est absente ou
// invalide.
func hoursFromEnv(key string, def time.Duration) time.Duration {
	if h := positiveIntFromEnv(key); h > 0 {
		return time.Duration(h) * time.Hour
	}
	return def
}
