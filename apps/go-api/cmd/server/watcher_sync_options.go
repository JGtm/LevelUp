// cmd/server — watcher_sync_options.go : options de synchronisation du daemon
// watcher (chemin LIVE, in-process).
//
// Pourquoi un fichier dédié plutôt qu'un literal inline dans startWatcherDaemon :
// le literal indépendant a déjà coûté DEUX incidents identiques (cf.
// watcherSyncOptions). Isoler la construction la rend testable — le garde-rail
// TestWatcherSyncOptions_InheritsExtractionFlags casse dès qu'un flag
// d'extraction diverge de domain.DefaultSyncOptions().
package main

import "levelup/go-api/internal/domain"

const (
	// watcherSyncMaxMatches : le watcher est un chemin LIVE incrémental (il suit
	// une session en cours), pas un rattrapage d'historique — fenêtre courte,
	// volontairement plus petite que DefaultSyncOptions().MaxMatches (200).
	watcherSyncMaxMatches = 25
	// watcherSyncRequestsPerSecond : le watcher partage le quota Microsoft
	// par-token avec le scheduler auto-sync — débit réduit vs
	// DefaultSyncOptions().RequestsPerSecond (10) pour éviter les 429 croisés.
	watcherSyncRequestsPerSecond = 5
)

// watcherSyncOptions retourne les SyncOptions du daemon watcher, DÉRIVÉES de
// domain.DefaultSyncOptions(). Seuls les deux réglages de DÉBIT ci-dessus sont
// surchargés ; tout flag d'EXTRACTION (WithParticipants, WithMedals,
// WithHighlightEvents, WithObjectiveStats, et tous ceux à venir) est hérité par
// construction.
//
// Cette dérivation est la correction structurelle de deux incidents identiques,
// tous deux causés par un literal indépendant qui « oubliait » un flag :
//
//  1. WithHighlightEvents (constaté 2026-06-04) : le watcher insérait
//     registry+participants SANS events → highlight_events, puis
//     killer_victim_pairs et weapon_kills (frags par arme) restaient vides.
//  2. WithObjectiveStats (contre-revue V7.2, 2026-07-25) : reproduction exacte —
//     aucun match ramené par le watcher n'avait ses stats objectifs
//     (match_objective_stats vide pour CTF / Zones / Oddball).
//
// Dans les deux cas AUCUN rattrapage n'est possible : le delta du scheduler
// (qui, lui, part de DefaultSyncOptions) s'arrête au premier match déjà
// « connu » — le bit manquant n'est jamais reposé, il faut un backfill forcé.
// D'où la règle : le watcher NE construit JAMAIS ses options ex nihilo.
func watcherSyncOptions() domain.SyncOptions {
	opts := domain.DefaultSyncOptions()
	opts.MaxMatches = watcherSyncMaxMatches
	opts.RequestsPerSecond = watcherSyncRequestsPerSecond
	return opts
}
