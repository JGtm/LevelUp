// Package gameproc : savoir si le jeu tourne sur ce poste, par l'énumération des processus du
// système et rien d'autre (ni lecture de mémoire, ni fenêtre, ni accroche : un nom d'exécutable
// présent dans la liste des processus, comme le ferait le gestionnaire des tâches).
//
// Trois pièces :
//
//   - [Lister] : l'énumération (ToolHelp32 sous Windows ; ailleurs, jamais de jeu détecté) ;
//   - [Mode] : l'état « jeu en cours » et la porte du travail lourd ([Mode.HeavyWorkAllowed],
//     [Mode.WaitHeavyAllowed]) que consultent la cuisson des rejeux et les tâches de fond avant
//     chaque lancement : une tâche lancée n'est jamais tuée, elle ne démarre pas ;
//   - [Watcher] : le sondage périodique qui nourrit le mode, journalise les passages
//     `game_mode: enter` / `game_mode: leave` et demande une synchronisation à la fermeture du jeu
//     (anti-rebond [DefaultDebounce]).
//
// Les noms d'exécutables viennent des descripteurs de titres (`process_names` du title.toml) : ce
// paquet ne connaît aucun titre.
package gameproc
