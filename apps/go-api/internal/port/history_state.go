package port

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
)

// HistoryCompletionReader dit, en lecture seule, si la fin de l'historique d'un joueur a déjà été
// vue (clé `history_complete_at` de sa base, règle de `sync/historystate`). Il sert à
// l'ordonnanceur quand la passe d'un joueur a échoué avant de pouvoir le dire : sans fin rangée,
// le cycle suivant doit venir vite.
//
// Contrat : jamais d'écriture ni de bail ; une base joueur absente vaut « incomplet » (aucune passe
// n'a encore parcouru l'historique) ; une lecture en échec rend l'erreur et l'appelant la journalise.
type HistoryCompletionReader interface {
	HistoryComplete(ctx context.Context, titleSlug, gamertag string) (bool, error)
}

// HistoryProgress est ce qu'une lecture de l'état de l'historique d'un joueur rend : ce que la base
// partagée du titre a rangé de ses matchs, et si la fin de l'historique a été vue.
type HistoryProgress struct {
	// MatchCount : matchs du joueur rangés dans la base partagée du titre.
	MatchCount int
	// OldestAt : début du plus ancien match rangé ; zéro si aucun.
	OldestAt time.Time
	// Complete : la fin de l'historique a été vue (clé de fin rangée).
	Complete bool
}

// HistoryProgressReader lit l'état de l'historique d'un joueur sur un titre, sans écrire : nombre de
// matchs rangés et date du plus ancien (base partagée, par son fournisseur), clé de fin d'historique
// (base joueur, par OpenReadForQuery).
type HistoryProgressReader interface {
	ReadHistoryProgress(ctx context.Context, titleSlug, gamertag, xuid string) (HistoryProgress, error)
}

// SyncPassStatus est l'issue d'une passe de synchronisation d'un joueur.
type SyncPassStatus string

const (
	// SyncPassOK : la passe est allée au bout.
	SyncPassOK SyncPassStatus = "ok"
	// SyncPassPartial : la passe a rangé ce qu'elle a pu, des erreurs partielles restent.
	SyncPassPartial SyncPassStatus = "partial"
	// SyncPassFailed : la passe a échoué.
	SyncPassFailed SyncPassStatus = "failed"
	// SyncPassSkipped : la passe n'a pas eu lieu (jeton absent, autre synchronisation en vol).
	SyncPassSkipped SyncPassStatus = "skipped"
)

// SyncPass est la dernière passe d'un joueur sur un titre, telle que l'ordonnanceur l'a vue depuis son
// démarrage.
type SyncPass struct {
	At              time.Time
	MatchesInserted int
	Status          SyncPassStatus
}

// SyncActivity est ce que l'ordonnanceur sait de son calendrier et de ses passes. Lecture seule.
type SyncActivity interface {
	// AutoSyncEnabled dit si la synchronisation automatique est activée dans les réglages.
	AutoSyncEnabled() bool
	// NextCycleAt rend l'instant du prochain cycle périodique ; zéro tant que la boucle n'a pas démarré.
	NextCycleAt() time.Time
	// LastPass rend la dernière passe du joueur sur le titre depuis le démarrage ; faux si aucune.
	LastPass(titleSlug, gamertag string) (SyncPass, bool)
}

// HistorySyncService compose l'état de l'historique d'un joueur : base, ordonnanceur et libellés.
type HistorySyncService interface {
	// HistorySyncStatus rend l'état de l'historique du joueur sur chacun de ses titres suivis, libellés
	// dans la langue de la requête. Un joueur sans titre suivi rend une liste vide.
	HistorySyncStatus(ctx context.Context, playerSlug string) (*domain.HistorySyncStatus, error)
}
