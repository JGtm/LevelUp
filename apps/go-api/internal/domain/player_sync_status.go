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
	// HistoryComplete : la fin de l'historique du joueur chez ce titre a déjà été atteinte par une passe de
	// synchronisation sans erreur. Faux tant que le rattrapage de l'historique n'est pas terminé (ou
	// que l'état n'a pu être lu) : des matchs plus anciens peuvent encore manquer.
	HistoryComplete bool `json:"history_complete" doc:"Vrai quand une synchronisation a atteint la fin de l'historique de ce joueur dans le titre courant ; faux tant que le rattrapage des matchs anciens n'est pas terminé."`
}
