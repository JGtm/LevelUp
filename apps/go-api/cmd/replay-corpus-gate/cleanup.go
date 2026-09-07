package main

// cleanup.go — LE NETTOYAGE COMPOSE, A UN SEUL POINT DE SORTIE (CORPUS-R1 C1/C2, 2026-09-07).
//
// # LES DEUX BUGS QUE CE FICHIER REMPLACE
//
// `main.go` composait autrefois son nettoyage par une variable `cleanup func()` reassignee en
// cours de route (`*previousCleanup = func(){ cleanupWt(); dejaLa() }`) sous un `defer
// cleanup()` DEJA ARME plus haut — Go fige la VALEUR de fonction au moment du `defer` : la
// reassignation posterieure n'etait JAMAIS vue, et le worktree detache de la base n'etait donc
// JAMAIS retire, meme en succes (C1). Le calcul du code de sortie appelait en outre `os.Exit`
// AU MILIEU de `executer` des qu'une perte etait trouvee — `os.Exit` saute TOUS les defers sans
// exception, y compris celui, correctement arme ou non, qui nettoie la racine de travail (C2) :
// c'est le chemin NOMINAL d'un gate qui detecte une perte, pas un cas rare.
//
// # LA REGLE QUI EN SORT
//
// Un SEUL point d'appel a `os.Exit`, dans `main()`, APRES le retour complet de `executer` —
// jamais avant, jamais a l'interieur. `executer` (et tout ce qu'il appelle) RETOURNE un code,
// il ne quitte jamais lui-meme le processus. Le nettoyage est un `nettoyeurCompose` : chaque
// etape qui cree une ressource jetable (racine de travail, worktree detache de la base)
// l'`Ajoute` des sa creation, dans n'importe quel ordre et depuis n'importe quelle fonction
// imbriquee — jamais une closure reassignee sous un defer deja arme.

import "sync"

// nettoyeurCompose accumule des actions de nettoyage et les executes TOUTES, dans l'ordre
// INVERSE de leur ajout (LIFO, comme des `defer` empiles), une seule fois (sync.Once) — un
// signal d'interruption pendant le gate (13 a 25 min) et le `defer` normal de fin de
// `executer()` ne peuvent donc jamais nettoyer deux fois la meme ressource.
type nettoyeurCompose struct {
	mu      sync.Mutex
	actions []func()
	once    sync.Once
}

// Ajouter enregistre une action de nettoyage supplementaire — appelable a tout moment avant
// Executer, y compris depuis une fonction appelee bien APRES qu'un `defer func() {
// n.Executer() }()` a deja ete arme sur ce meme `n` (c'est exactement le cas d'usage qui a
// echoue sous l'ancien design, cf. l'en-tete du fichier).
func (n *nettoyeurCompose) Ajouter(action func()) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.actions = append(n.actions, action)
}

// Executer joue toutes les actions enregistrees a cet instant, dans l'ordre inverse de leur
// ajout, une seule fois.
func (n *nettoyeurCompose) Executer() {
	n.once.Do(func() {
		n.mu.Lock()
		actions := append([]func(){}, n.actions...)
		n.mu.Unlock()
		for i := len(actions) - 1; i >= 0; i-- {
			actions[i]()
		}
	})
}
