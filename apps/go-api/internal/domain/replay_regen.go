package domain

// replay_regen.go — l'état de la régénération des rejeux après une mise à jour (application de bureau,
// ADR 0045) : GET /desktop/replay-regen et les trois commandes (démarrer, mettre en pause, reprendre) rendent
// cette forme.

// ReplayRegenStatus : combien de rejeux sont en retard sur la version de l'application, et où en est la file qui
// les régénère.
type ReplayRegenStatus struct {
	// Checked : le décompte des rejeux en retard a été fait au moins une fois depuis le démarrage. Tant qu'il vaut
	// faux, Stale ne dit rien (le décompte lit chaque rejeu rangé et se fait en arrière-plan).
	Checked bool `json:"checked" doc:"Le décompte des rejeux en retard est fait. Faux : Stale n'est pas encore connu."`
	// Stale : rejeux rangés plus anciens que la version de l'application et régénérables (film encore dans l'archive).
	// Pendant la file : ce qu'il en reste.
	Stale int `json:"stale" doc:"Rejeux rangés en retard sur la version de l'application, dont le film est encore dans l'archive. Pendant la régénération : ce qu'il en reste."`
	// Raw : rejeux encore en JSON brut (avant la compression des rejeux), comptés par les noms de fichiers au même
	// décompte que Stale. Quand il ne reste que des bruts (Stale = 0), la passe lancée par l'utilisateur ne fait que
	// les compresser.
	Raw int `json:"raw" doc:"Rejeux encore en JSON brut, à compresser. La passe de l'utilisateur les compresse, même sans rejeu à refaire."`
	// Running : la file de régénération est active (en pause comprise).
	Running bool `json:"running" doc:"La file de régénération est active (mise en pause comprise)."`
	// Done : rejeux traités par la file courante ou la dernière (régénérés, déjà à jour, écartés ou en échec).
	Done int `json:"done" doc:"Rejeux traités par la file courante ou la dernière."`
	// Total : rejeux que la file courante ou la dernière devait traiter ; 0 tant que la file prépare sa liste.
	Total int `json:"total" doc:"Rejeux que la file courante ou la dernière devait traiter. 0 tant qu'elle prépare sa liste."`
	// Paused : la file est en pause à la demande de l'utilisateur.
	Paused bool `json:"paused" doc:"La file est en pause à la demande de l'utilisateur."`
	// WaitingForGame : la file attend la fin de la partie (mode jeu) avant de lancer le film suivant.
	WaitingForGame bool `json:"waiting_for_game" doc:"La file attend la fin de la partie avant de lancer le rejeu suivant."`
	// Failed : rejeux de la file courante ou de la dernière que la régénération n'a pas pu refaire.
	Failed int `json:"failed" doc:"Rejeux de la file courante ou de la dernière qui n'ont pas pu être régénérés."`
	// Compressing : la file compresse, à sa fin, les rejeux encore en JSON brut qu'elle n'a pas pu refaire (film absent
	// de l'archive).
	Compressing bool `json:"compressing" doc:"La file compresse les rejeux bruts qu'elle n'a pas pu refaire (fin de passe)."`
	// Compressed : rejeux bruts compressés par la file courante ou la dernière.
	Compressed int `json:"compressed" doc:"Rejeux bruts compressés par la file courante ou la dernière."`
	// CompressFailed : rejeux bruts que la compression n'a pas pu convertir (illisibles, écriture ou vérification en
	// échec, doublon de contenu différent, suppression du brut à relancer) ; chacun reste lisible tel quel.
	CompressFailed int `json:"compress_failed" doc:"Rejeux bruts que la compression n'a pas pu convertir ; ils restent lisibles."`
}
