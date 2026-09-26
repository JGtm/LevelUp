// Package watcher — player_watcher.go : goroutine de surveillance par joueur.
//
// Un PlayerWatcher orchestre :
//   - La FSM (state_machine.go) pour l'état du joueur
//   - Le MatchPoller (match_poller.go) pour détecter les nouveaux matchs
//   - Les callbacks de présence (RTA Xbox / Steam) pour piloter la FSM
//
// Cycle de vie :
//  1. Présence détectée (RTA/Steam) → FSM Idle→Watching + démarrage MatchPoller
//  2. Nouveau match détecté → FSM Watching→Syncing + envoi match_ids au sync
//  3. Sync terminé → FSM Syncing→Cooling (cooldown)
//  4. Cooldown expiré → FSM Cooling→Watching (si encore en jeu) ou →Idle
//  5. Présence perdue → FSM →Idle + arrêt MatchPoller
package watcher

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/presence"
)

const (
	// defaultCooldown après un sync réussi.
	defaultCooldown = 90 * time.Second

	// defaultPostExitGrace : durée pendant laquelle le MatchPoller continue
	// de tourner après détection Inactive (extinction Halo, ou bascule sur un
	// titre non tracké comme le Dashboard Xbox).
	//
	// Pourquoi : l'API Halo expose un match terminé en ~30-60s après la fin
	// de la partie. Si on stop le MatchPoller dès qu'on détecte Inactive, on
	// rate le dernier match. 90s = ~1.5 cycle MatchPoller (60s) + marge, ce
	// qui garantit la capture du dernier match. Si Active revient pendant
	// la grâce, on cancel le timer et on reste en Watching.
	defaultPostExitGrace = 90 * time.Second
)

// SyncTrigger déclenche un sync pour les match_ids détectés.
type SyncTrigger interface {
	TriggerSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error
}

// PlayerWatcher surveille un joueur et orchestre présence + match polling + sync.
type PlayerWatcher struct {
	gamertag string
	xuid     string
	// titleSlug : titre configuré du joueur (PlayerSummary.TitleSlug). Posé sur
	// le ctx du poller (startPoller) → le fetch match-history est routé par titre
	// (host PMT-1 via ctxkeys) ET le MatchRequest porte le titre jusqu'au
	// CoordinatorRequest. Vide ⇒ halo_infinite (byte-identique). Champ intrinsèque
	// (pas lu du ctx entrant) → robuste au broadcast de présence inter-joueurs.
	titleSlug string

	fsm           *FSM
	fetcher       MatchFetcher
	syncTrigger   SyncTrigger
	liveRefresh   LiveRefreshTrigger // nil si non configuré
	cooldown      time.Duration
	postExitGrace time.Duration

	pollerCancel      context.CancelFunc
	pollerMu          sync.Mutex
	warnNoFetcherOnce sync.Once // log Warn-once si fetcher==nil au démarrage du poller

	// postExitTimer : si non-nil, un timer de grâce post-extinction tourne.
	// Cancel sur OnPresenceActive (le user est revenu en jeu).
	postExitTimer *time.Timer

	// inGame track si la présence dit "en jeu" (RTA ou Steam)
	inGame bool
	// subscribeError conserve la dernière erreur d'abonnement RTA (nil si abonné avec succès)
	subscribeError error
	// lastSeen : dernière info `lastSeen` reçue (titre + timestamp) — utile
	// pour afficher "vu il y a 2h sur Halo Infinite" dans la WatcherCard
	// Settings. Mis à jour à chaque event qui contient un bloc lastSeen.
	lastSeen *presence.LastSeenInfo
	// lastPresenceState : dernier état Xbox brut ("Online"/"Away"/"Offline")
	// reçu d'un event présence. Exposé via WatcherStatus pour différencier UI
	// "Hors-ligne" vs "Absent" vs "En ligne" — plus précis que le state FSM
	// (qui reste "Idle" dans les 3 cas).
	lastPresenceState string
	// currentTitleSlug / currentTitleName : titre TRACKÉ sur lequel le joueur est
	// vu, quel que soit le titre suivi par CE watcher — à ne pas confondre avec
	// `inGame` ci-dessus. Accès et sémantique : player_watcher_title.go.
	currentTitleSlug string
	currentTitleName string
	// lastEventAt : instant du dernier event de présence reçu (REST poll ou
	// RTA), peu importe son contenu. Mis à jour à CHAQUE event dispatché par le
	// handler — donc à chaque poll REST réussi (cf. rest_poller.tickOnce), pas
	// seulement sur changement d'état. C'est un témoin de vivacité du daemon :
	// si lastEventAt se fige alors que le daemon tourne, les polls échouent en
	// boucle (backoff auth/réseau) → watcher "mort" malgré daemon_running=true.
	// Zéro si aucun event reçu depuis le boot.
	lastEventAt time.Time
	mu          sync.Mutex
}

// NewPlayerWatcher crée un watcher pour un joueur.
func NewPlayerWatcher(gamertag, xuid string, fetcher MatchFetcher, syncTrigger SyncTrigger) *PlayerWatcher {
	pw := &PlayerWatcher{
		gamertag:      gamertag,
		xuid:          xuid,
		fetcher:       fetcher,
		syncTrigger:   syncTrigger,
		cooldown:      defaultCooldown,
		postExitGrace: defaultPostExitGrace,
	}
	pw.fsm = NewFSM(gamertag, pw.onTransition)
	return pw
}

// WithPostExitGrace override la grâce post-extinction (pour tests).
func (pw *PlayerWatcher) WithPostExitGrace(d time.Duration) *PlayerWatcher {
	pw.postExitGrace = d
	return pw
}

// FSM retourne la FSM du watcher (pour lecture d'état).
func (pw *PlayerWatcher) FSM() *FSM {
	return pw.fsm
}

// SetSubscribeError enregistre (ou efface) la dernière erreur d'abonnement RTA.
func (pw *PlayerWatcher) SetSubscribeError(err error) {
	pw.mu.Lock()
	pw.subscribeError = err
	pw.mu.Unlock()
}

// SetTitleSlug fixe le titre du joueur surveillé (Phase 1.9). Source =
// PlayerSummary.TitleSlug (titre configuré du joueur). Le slug est ensuite posé
// sur le ctx du poller (startPoller) pour router le fetch par titre (host PMT-1)
// et alimenter MatchRequest.TitleSlug → CoordinatorRequest. Vide ⇒ halo_infinite.
func (pw *PlayerWatcher) SetTitleSlug(slug string) {
	pw.mu.Lock()
	pw.titleSlug = slug
	pw.mu.Unlock()
}

// SubscribeError retourne la dernière erreur d'abonnement RTA, ou nil si abonné.
func (pw *PlayerWatcher) SubscribeError() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.subscribeError
}

// RecordLastSeen mémorise la dernière info `lastSeen` reçue de Xbox.
// Appelé par le handler watcher pour chaque event qui contient un bloc
// lastSeen (typiquement les snapshots Offline). Copie superficielle pour
// éviter qu'un mutateur tiers ne modifie l'état stocké.
//
// Loggue en DEBUG (filtré par défaut, activable via LEVELUP_LOGS_FILE_LEVEL=
// debug) pour faciliter le debug des transitions présence sans polluer la
// production.
func (pw *PlayerWatcher) RecordLastSeen(ctx context.Context, info *presence.LastSeenInfo) {
	if info == nil {
		return
	}
	cp := *info
	pw.mu.Lock()
	pw.lastSeen = &cp
	pw.mu.Unlock()

	slog.DebugContext(ctx, "player_watcher: last_seen mis à jour",
		"gamertag", pw.gamertag,
		"title_name", info.TitleName,
		"title_id", info.TitleID,
		"timestamp", info.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
	)
}

// LastSeen retourne la dernière info `lastSeen` connue, ou nil si jamais
// reçue. Copie défensive pour ne pas exposer l'état interne.
func (pw *PlayerWatcher) LastSeen() *presence.LastSeenInfo {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.lastSeen == nil {
		return nil
	}
	cp := *pw.lastSeen
	return &cp
}

// RecordPresenceState mémorise le dernier state Xbox brut reçu via event
// (Online / Away / Offline). Une chaîne vide est ignorée pour ne pas
// effacer un état précédent valide quand Xbox renvoie un payload incomplet.
func (pw *PlayerWatcher) RecordPresenceState(state string) {
	if state == "" {
		return
	}
	pw.mu.Lock()
	pw.lastPresenceState = state
	pw.mu.Unlock()
}

// LastPresenceState retourne le dernier state Xbox brut connu, ou ""
// si aucun event n'a encore été reçu (juste après boot).
func (pw *PlayerWatcher) LastPresenceState() string {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.lastPresenceState
}

// RecordEvent mémorise l'instant du dernier event de présence reçu. Appelé
// par le handler du daemon pour CHAQUE event (avant tout filtrage de titre),
// donc à chaque poll REST réussi. Aucun log : opération à très haute fréquence.
func (pw *PlayerWatcher) RecordEvent(ts time.Time) {
	pw.mu.Lock()
	pw.lastEventAt = ts
	pw.mu.Unlock()
}

// LastEventAt retourne l'instant du dernier event reçu, ou le zéro time.Time
// si aucun event depuis le boot.
func (pw *PlayerWatcher) LastEventAt() time.Time {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.lastEventAt
}

// WithLiveRefresh configure le refresher live BP/Challenges.
// Retourne le watcher pour permettre le chaînage.
func (pw *PlayerWatcher) WithLiveRefresh(r LiveRefreshTrigger) *PlayerWatcher {
	pw.liveRefresh = r
	return pw
}

// OnPresenceActive est appelé quand le joueur est détecté en jeu (RTA ou Steam).
//
// Si un timer de grâce post-extinction est en cours (extinction récente non
// encore confirmée), il est cancel : le joueur est revenu en jeu avant la fin
// de la grâce, on reste en Watching sans stop le poller.
func (pw *PlayerWatcher) OnPresenceActive(ctx context.Context) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.inGame = true

	// Annule la grâce post-extinction si en cours.
	if pw.postExitTimer != nil {
		pw.postExitTimer.Stop()
		pw.postExitTimer = nil
		slog.InfoContext(ctx, "player_watcher: gracieux post-extinction annulé (jeu repris)",
			"gamertag", pw.gamertag)
	}

	state := pw.fsm.State()
	if state == StateIdle {
		if err := pw.fsm.GoWatching(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition →Watching",
				"gamertag", pw.gamertag, "err", err)
			return
		}
		pw.startPoller(ctx)
	}

	if pw.liveRefresh != nil {
		pw.liveRefresh.OnPresenceActive(ctx)
	}
}

// OnPresenceInactive est appelé quand le joueur quitte le jeu.
//
// Au lieu de stop le MatchPoller immédiatement, on lance un timer de grâce
// (postExitGrace, 90s par défaut). Le MatchPoller continue à tourner pendant
// ce délai pour capter un éventuel dernier match (latence Halo API ~30-60s
// pour exposer un match terminé). Si OnPresenceActive est appelé avant la
// fin du timer, le timer est cancel et on reste en Watching.
//
// Pas de grâce si :
//   - postExitGrace == 0 (config explicite, ou tests qui veulent stop immédiat)
//   - state n'est pas Watching/Cooling (rien à stopper)
func (pw *PlayerWatcher) OnPresenceInactive(ctx context.Context) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.inGame = false

	state := pw.fsm.State()
	needsStop := state == StateWatching || state == StateCooling

	if needsStop && pw.postExitGrace > 0 {
		// Démarre le timer si pas déjà en cours.
		if pw.postExitTimer == nil {
			slog.InfoContext(ctx, "player_watcher: extinction détectée — grâce post-extinction démarrée",
				"gamertag", pw.gamertag,
				"grace", pw.postExitGrace,
			)
			pw.postExitTimer = time.AfterFunc(pw.postExitGrace, func() {
				pw.finalizeExit(context.Background())
			})
		}
		if pw.liveRefresh != nil {
			pw.liveRefresh.OnPresenceInactive(ctx)
		}
		return
	}

	if needsStop {
		// Mode pas-de-grâce : stop immédiat (legacy comportement).
		pw.stopPoller()
		if err := pw.fsm.GoIdle(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition →Idle",
				"gamertag", pw.gamertag, "err", err)
		}
	}

	if pw.liveRefresh != nil {
		pw.liveRefresh.OnPresenceInactive(ctx)
	}
}

// finalizeExit est appelé par le timer postExitGrace quand la grâce expire
// sans qu'OnPresenceActive ait été rappelé. Stop le MatchPoller + transition
// vers Idle.
func (pw *PlayerWatcher) finalizeExit(ctx context.Context) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Re-check : si OnPresenceActive a été appelé entre temps, postExitTimer
	// est nil et inGame est true → on ne fait rien.
	if pw.postExitTimer == nil || pw.inGame {
		return
	}
	pw.postExitTimer = nil

	state := pw.fsm.State()
	if state != StateWatching && state != StateCooling {
		return
	}

	slog.InfoContext(ctx, "player_watcher: grâce post-extinction expirée — arrêt MatchPoller",
		"gamertag", pw.gamertag)
	pw.stopPoller()
	if err := pw.fsm.GoIdle(); err != nil {
		slog.WarnContext(ctx, "player_watcher: erreur transition →Idle (post-grace)",
			"gamertag", pw.gamertag, "err", err)
	}
}

// OnNewMatches est le callback appelé par le MatchPoller quand de nouveaux matchs
// sont trouvés. Retourne true si les matchs sont ACCEPTÉS (état Watching → sync
// lancé), false sinon (état busy / transition KO) — le poller s'en sert pour ne
// PAS marquer ces matchs connus et les re-signaler au prochain poll (W4).
func (pw *PlayerWatcher) OnNewMatches(ctx context.Context, matchIDs []string) bool {
	if len(matchIDs) == 0 {
		return false
	}

	state := pw.fsm.State()
	if state != StateWatching {
		slog.DebugContext(ctx, "player_watcher: matchs ignorés (état != Watching)",
			"gamertag", pw.gamertag,
			"state", state.String(),
			"count", len(matchIDs),
		)
		return false
	}

	if err := pw.fsm.GoSyncing(); err != nil {
		slog.WarnContext(ctx, "player_watcher: erreur transition →Syncing",
			"gamertag", pw.gamertag, "err", err)
		return false
	}

	// Lancer le sync en goroutine pour ne pas bloquer le poller
	go pw.runSync(ctx, matchIDs)
	return true
}

// OnSyncComplete est appelé quand le sync est terminé.
func (pw *PlayerWatcher) OnSyncComplete(ctx context.Context) {
	if err := pw.fsm.GoCooling(pw.cooldown); err != nil {
		slog.WarnContext(ctx, "player_watcher: erreur transition →Cooling",
			"gamertag", pw.gamertag, "err", err)
		return
	}

	// Lancer le cooldown timer en goroutine
	go pw.waitCooldown(ctx)
}

// runSync exécute le sync et gère la transition post-sync.
func (pw *PlayerWatcher) runSync(ctx context.Context, matchIDs []string) {
	slog.InfoContext(ctx, "player_watcher: démarrage sync",
		"gamertag", pw.gamertag,
		"match_count", len(matchIDs),
	)

	err := pw.syncTrigger.TriggerSync(ctx, pw.gamertag, pw.xuid, matchIDs)
	if err != nil {
		slog.ErrorContext(ctx, "player_watcher: sync échoué",
			"gamertag", pw.gamertag,
			"err", err,
		)
	} else {
		slog.InfoContext(ctx, "player_watcher: sync terminé",
			"gamertag", pw.gamertag,
		)
	}

	pw.OnSyncComplete(ctx)
}

// waitCooldown attend la fin du cooldown puis transite vers Watching ou Idle.
func (pw *PlayerWatcher) waitCooldown(ctx context.Context) {
	remaining := pw.fsm.CooldownRemaining()
	if remaining <= 0 {
		remaining = pw.cooldown
	}

	slog.DebugContext(ctx, "player_watcher: cooldown démarré",
		"gamertag", pw.gamertag,
		"duration", remaining,
	)

	select {
	case <-ctx.Done():
		return
	case <-time.After(remaining):
	}

	pw.mu.Lock()
	inGame := pw.inGame
	pw.mu.Unlock()

	if inGame {
		if err := pw.fsm.GoWatching(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition Cooling→Watching",
				"gamertag", pw.gamertag, "err", err)
		}
	} else {
		if err := pw.fsm.GoIdle(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition Cooling→Idle",
				"gamertag", pw.gamertag, "err", err)
		}
		pw.stopPoller()
	}
}

// startPoller démarre le MatchPoller dans une goroutine.
//
// Garde-fou : si pw.fetcher est nil (config dégradée — pool de tokens absent,
// tests, etc.), on skip la création du poller pour ne pas paniquer dans
// MatchPoller.poll(). Log Warn-once par PlayerWatcher pour observabilité
// sans spammer (transitions Idle↔Watching répétées).
func (pw *PlayerWatcher) startPoller(ctx context.Context) {
	if pw.fetcher == nil {
		pw.warnNoFetcherOnce.Do(func() {
			slog.WarnContext(ctx, "player_watcher: pas de MatchFetcher configuré — poller désactivé",
				"gamertag", pw.gamertag,
				"xuid", pw.xuid,
			)
		})
		return
	}

	pw.pollerMu.Lock()
	defer pw.pollerMu.Unlock()

	// Arrêter un éventuel poller existant
	if pw.pollerCancel != nil {
		pw.pollerCancel()
	}

	// Phase 1.9 : poser le titre du joueur sur le ctx du poller → le fetch
	// match-history est routé par titre (host PMT-1 via ctxkeys) et le ctx remonte
	// jusqu'à TriggerSync → MatchRequest.TitleSlug → CoordinatorRequest. Le slug
	// vient du CHAMP pw.titleSlug (lu sous le lock détenu par OnPresenceActive, seul
	// appelant prod), pas du ctx entrant → robuste au broadcast de présence (le ctx
	// d'un autre joueur ne contamine pas ce poller). Vide ⇒ ctx inchangé ⇒ halo_infinite.
	pollerBaseCtx := ctx
	if pw.titleSlug != "" {
		pollerBaseCtx = ctxkeys.WithTitleSlug(ctx, pw.titleSlug)
	}
	pollerCtx, cancel := context.WithCancel(pollerBaseCtx)
	pw.pollerCancel = cancel

	poller := NewMatchPoller(pw.xuid, pw.gamertag, pw.fetcher, func(matchIDs []string) bool {
		return pw.OnNewMatches(pollerCtx, matchIDs)
	})

	go poller.Run(pollerCtx)

	slog.InfoContext(ctx, "player_watcher: match poller démarré",
		"gamertag", pw.gamertag,
	)
}

// stopPoller arrête le MatchPoller.
func (pw *PlayerWatcher) stopPoller() {
	pw.pollerMu.Lock()
	defer pw.pollerMu.Unlock()

	if pw.pollerCancel != nil {
		pw.pollerCancel()
		pw.pollerCancel = nil
		slog.Info("player_watcher: match poller arrêté",
			"gamertag", pw.gamertag,
		)
	}
}

// onTransition callback de la FSM.
func (pw *PlayerWatcher) onTransition(from, to State) {
	slog.Info("player_watcher: FSM transition",
		"gamertag", pw.gamertag,
		"from", from.String(),
		"to", to.String(),
	)
}
