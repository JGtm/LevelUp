// Package scheduler — auto_sync_passes.go : la dernière passe de chaque joueur, par titre.
//
// Le pipeline V2 ne range pas ses joueurs dans playerOutcomes (le diag et la notification Discord les
// comptent par un autre chemin : y ajouter les joueurs V2 les compterait deux fois). La dernière passe
// que l'écran « état de l'historique » affiche vit donc dans sa propre table, alimentée par les deux
// chemins (issue du cycle V2, passe de syncPlayer). Mémoire seule : un redémarrage la vide, l'écran
// dit alors qu'aucune passe n'a eu lieu depuis le démarrage.
package scheduler

import (
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	syncv2 "levelup/go-api/internal/sync/v2"
)

var _ port.SyncActivity = (*AutoSyncScheduler)(nil)

// passKey identifie un couple (titre, joueur) : un même gamertag peut être suivi sur deux titres.
func passKey(titleSlug, gamertag string) string { return titleSlug + "\x00" + gamertag }

// recordPass range la dernière passe d'un joueur sur un titre.
func (s *AutoSyncScheduler) recordPass(titleSlug, gamertag string, pass port.SyncPass) {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()
	if s.lastPasses == nil {
		s.lastPasses = make(map[string]port.SyncPass)
	}
	s.lastPasses[passKey(titleSlug, gamertag)] = pass
}

// LastPass rend la dernière passe du joueur sur le titre depuis le démarrage (port.SyncActivity).
func (s *AutoSyncScheduler) LastPass(titleSlug, gamertag string) (port.SyncPass, bool) {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	pass, ok := s.lastPasses[passKey(titleSlug, gamertag)]
	return pass, ok
}

// AutoSyncEnabled dit si la synchronisation automatique est activée dans les réglages (port.SyncActivity).
// Une lecture des réglages en échec vaut « désactivée » : rien n'est promis.
func (s *AutoSyncScheduler) AutoSyncEnabled() bool {
	cfg, err := s.settings.Load()
	if err != nil {
		slog.Warn("auto_sync: lecture settings échouée — synchronisation automatique réputée désactivée", "err", err)
		return false
	}
	return cfg.SpnkrAutoSyncEnabled
}

// statusFailed est le statut « échec » d'une passe : issue du pipeline V2 (syncv2.PlayerOutcome.Status) et PlayerOutcomeDetail.Outcome.
const statusFailed = "failed"

// passStatusOfV2 traduit le statut d'une issue du pipeline V2 ; tout statut inconnu vaut « différée ».
func passStatusOfV2(status string) port.SyncPassStatus {
	switch status {
	case "ok":
		return port.SyncPassOK
	case "partial":
		return port.SyncPassPartial
	case statusFailed:
		return port.SyncPassFailed
	default:
		return port.SyncPassSkipped
	}
}

// recordV2Passes range la passe de chaque joueur d'un cycle V2 qui a une issue.
func (s *AutoSyncScheduler) recordV2Passes(players []domain.PlayerSummary, per map[string]syncv2.PlayerOutcome) {
	now := s.clock()
	for _, p := range players {
		o, ok := per[p.PlayerSlug]
		if !ok {
			continue
		}
		s.recordPass(resolveTitleSlug(p), p.Gamertag, port.SyncPass{At: now, MatchesInserted: o.MatchesInserted, Status: passStatusOfV2(o.Status)})
	}
}

// recordFailedPasses range une passe en échec pour chacun des joueurs : le cycle V2 a échoué en bloc.
func (s *AutoSyncScheduler) recordFailedPasses(players []domain.PlayerSummary) {
	now := s.clock()
	for _, p := range players {
		s.recordPass(resolveTitleSlug(p), p.Gamertag, port.SyncPass{At: now, Status: port.SyncPassFailed})
	}
}

// passStatusOfSync traduit l'issue d'un syncPlayer : réussie avec des erreurs partielles = partielle.
func passStatusOfSync(outcome syncOutcome, detail PlayerOutcomeDetail) port.SyncPassStatus {
	switch outcome {
	case outcomeOK:
		if detail.ErrorCount > 0 {
			return port.SyncPassPartial
		}
		return port.SyncPassOK
	case outcomeFailed:
		return port.SyncPassFailed
	default:
		return port.SyncPassSkipped
	}
}
