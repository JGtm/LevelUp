package domain

// PlayerSyncStatus est l'état de la synchronisation d'UN joueur, tel que l'accueil d'un joueur sans
// match le dit : ce qui tourne, ce qui vient de finir ou d'échouer, et si un sync manuel est
// accepté. Il vient du serveur (registre des jobs, porte de déduplication entre sources, délai du
// sync manuel) : le client ne le devine ni d'un identifiant gardé en mémoire ni du total de matchs.
type PlayerSyncStatus struct {
	PlayerSlug string `json:"player_slug"`
	// Syncing : une synchronisation de ce joueur est en vol, quelle qu'en soit la source (job de
	// l'API, synchronisation automatique, suivi en direct).
	Syncing bool `json:"syncing" doc:"Une synchronisation de ce joueur est en vol, quelle qu'en soit la source."`
	// Job : le job de sync actif du joueur, sinon le plus récent encore conservé (terminé, échoué,
	// interrompu) ; absent quand aucun n'est connu. Son player_slug est celui du joueur demandé.
	Job *AsyncJobStatus `json:"job,omitempty" doc:"Job de sync actif du joueur, sinon le plus récent encore conservé."`
	// CooldownRetryAfterSec : secondes avant qu'un sync manuel de ce joueur soit de nouveau accepté ;
	// 0 = accepté.
	CooldownRetryAfterSec int `json:"cooldown_retry_after_s" doc:"Secondes avant qu'un sync manuel soit de nouveau accepté ; 0 = accepté."`
}
