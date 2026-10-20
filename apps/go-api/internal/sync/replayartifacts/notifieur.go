package replayartifacts

// notifieur.go — CE QUI SUIT LES DÉRIVATIONS : prévenir qu'un match a ses dérivations écrites,
// pour que ce qui les lit (les citations du film de chaque joueur suivi) soit recalculé tout de
// suite plutôt qu'au prochain cycle de synchronisation de chacun.
//
// UN SEUL POINT DE BRANCHEMENT, AU BOOT DU SERVEUR ([SetNotifieurDerivations]) : les trois
// appelants de [Deriver] (cuisson locale, dépôt d'ouvrier, rattrapage des dérivés) passent par
// [marquerDerivations], qui notifie. Un CLI ne le branche jamais : ses dérivations ne
// déclenchent rien hors de son processus.

import (
	"context"
	"sync/atomic"
)

// NotifieurDerivations reçoit les matchs dont les dérivations viennent d'être écrites. Il doit
// rendre la main AUSSITÔT (il est appelé sur le chemin d'un dépôt HTTP) : poser le travail dans
// une file, jamais le faire.
type NotifieurDerivations func(ctx context.Context, titleSlug string, matchIDs []string)

var notifieur atomic.Pointer[NotifieurDerivations]

// SetNotifieurDerivations branche (ou débranche, nil) le notifieur du processus.
func SetNotifieurDerivations(n NotifieurDerivations) {
	if n == nil {
		notifieur.Store(nil)
		return
	}
	notifieur.Store(&n)
}

// notifierDerivations prévient le notifieur branché, s'il y en a un et s'il y a des matchs.
func notifierDerivations(ctx context.Context, titleSlug string, matchIDs []string) {
	if len(matchIDs) == 0 {
		return
	}
	if n := notifieur.Load(); n != nil {
		(*n)(ctx, titleSlug, matchIDs)
	}
}
