// Package watcher — player_watcher_title.go : le TITRE COURANT d'un joueur.
//
// Deux questions se ressemblent et n'ont pas la même réponse :
//
//   - « ce joueur joue-t-il AU TITRE QUE CE WATCHER SUIT ? » → `inGame`
//     (player_watcher.go). C'est la question du SYNC : elle pilote la FSM et le
//     MatchPoller, et elle doit rester fausse quand le joueur est ailleurs, sous
//     peine de synchroniser dans le mauvais titre.
//   - « ce joueur est-il en jeu, tout court ? » → `currentTitleSlug` ci-dessous.
//     C'est la question de l'UI de présence.
//
// Un joueur configuré halo_5 qui lance Halo Infinite répond NON à la première et
// OUI à la seconde. C'est pour cela que le handler de présence du daemon
// enregistre le titre AVANT de constater qu'il ne s'agit pas du sien (et de
// sortir en inactif) : capter après aurait affiché « hors jeu » un joueur en
// pleine partie.
//
// Fichier séparé de player_watcher.go, déjà au-delà du seuil de 500 lignes
// (dette gelée) : on ne l'accroît pas.
package watcher

// SetCurrentTitle mémorise le titre TRACKÉ sur lequel le joueur est vu. Appelé
// par le handler de présence du daemon dès qu'un titre du registre est reconnu
// (MatchPresence non nil). Deux chaînes vides = le joueur n'est sur aucun titre
// suivi (extinction, titre hors registre, ou payload sans titre).
func (pw *PlayerWatcher) SetCurrentTitle(slug, name string) {
	pw.mu.Lock()
	pw.currentTitleSlug = slug
	pw.currentTitleName = name
	pw.mu.Unlock()
}

// CurrentTitle retourne le titre tracké courant (slug, nom). Les deux valeurs
// sont vides si le joueur n'est sur aucun titre du registre.
func (pw *PlayerWatcher) CurrentTitle() (string, string) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.currentTitleSlug, pw.currentTitleName
}
