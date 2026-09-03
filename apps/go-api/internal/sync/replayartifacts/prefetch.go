package replayartifacts

// prefetch.go — LE PONT DISQUE DU CYCLE, ET SON PRECHARGEMENT DE PROFONDEUR 1.
//
// # CE QUE CE FICHIER GAGNE (PLAN_CUISSON_PERF item 5.6, 2026-09-03)
//
// La cuisson d'un lot alternait deux travaux qui n'utilisent PAS la meme ressource : le pont
// disque (reseau — telecharger 24 Mo de morceaux depuis le CDN Azure) puis la cuisson (CPU —
// un enfant borne qui decode). Enchaines, ils s'attendent l'un l'autre : la machine ne fait
// rien pendant le telechargement, et le lien ne fait rien pendant le decodage. Le film du match
// SUIVANT se telecharge donc pendant que le courant cuit.
//
// # POURQUOI LA PROFONDEUR EST 1, ET PAS 2, NI « TOUT LE LOT »
//
// Un film pese ~24 Mo de morceaux DECOMPRESSES, tenus en memoire du serveur entre la reponse du
// CDN et l'ecriture du cache (`filmcache.Write`). C'est le processus qui tient les bases en
// ecriture : y accumuler des films est exactement la faute que le lot BUILDALL a supprimee
// ailleurs. UN film d'avance borne cette retention a ~24 Mo, quel que soit le lot ; deux la
// doubleraient pour un gain nul (le prochain decodage n'a besoin que du prochain film).
//
// # CE QUE LE PRECHARGEMENT NE FAIT PAS
//
// Il ne DECODE rien — le decodage reste serialise dans l'enfant borne, un film a la fois. Il ne
// s'accumule pas : il y a AU PLUS un telechargement en vol, et le cycle ne se termine jamais
// sans l'avoir attendu ou annule (cf. [pontDisque.fermer]). Il ne deborde pas le budget : passe
// la borne du cycle, on ne precharge plus rien.

import (
	"context"
	"log/slog"
	"time"
)

// resultatFilm : ce que le pont disque rend d'un match — le film a-t-il ete persiste, et
// est-il disponible.
type resultatFilm struct {
	sauve bool
	dispo bool
}

// prefetchFilm : UN telechargement en vol, et le seul.
type prefetchFilm struct {
	matchID string
	annuler context.CancelFunc
	fini    chan resultatFilm
}

// pontDisque porte le pont disque d'un cycle et son unique prechargement.
//
// IL N'EST PAS SUR POUR UN USAGE CONCURRENT, ET C'EST VOULU : il est manipule par la SEULE
// boucle de cuisson. La goroutine de prechargement, elle, ne touche que son canal.
type pontDisque struct {
	d     Deps
	enVol *prefetchFilm
}

// film rend le film d'un match : celui que le prechargement a deja rapporte, ou un
// telechargement fait ici meme.
//
// LE PRECHARGEMENT EST CONSOMME OU ABANDONNE, JAMAIS IGNORE : un telechargement en vol qui ne
// correspond pas au match demande (cas qui n'arrive que si la boucle change d'ordre) est annulé
// et attendu avant d'en lancer un autre — deux telechargements simultanes doubleraient la
// memoire que ce fichier borne.
func (p *pontDisque) film(ctx context.Context, matchID string) resultatFilm {
	if en := p.enVol; en != nil {
		p.enVol = nil
		if en.matchID == matchID {
			r := <-en.fini
			en.annuler()
			return r
		}
		en.annuler()
		<-en.fini
	}
	saved, ok := persistFilmToCache(ctx, p.d, matchID)
	return resultatFilm{sauve: saved, dispo: ok}
}

// precharger lance le telechargement du film suivant, SI le budget du cycle en laisse le temps.
//
// `restant` est ce qui reste du budget de cycle. A zero ou moins, on ne precharge rien : le
// cycle s'arretera au prochain tour de boucle, et un telechargement lance maintenant serait de
// la bande passante depensee pour un film que personne ne cuira ce cycle-ci.
func (p *pontDisque) precharger(ctx context.Context, matchID string, restant time.Duration) {
	if p.enVol != nil || matchID == "" || restant <= 0 {
		return
	}
	pctx, annuler := context.WithCancel(ctx)
	en := &prefetchFilm{matchID: matchID, annuler: annuler, fini: make(chan resultatFilm, 1)}
	go func() {
		saved, ok := persistFilmToCache(pctx, p.d, matchID)
		en.fini <- resultatFilm{sauve: saved, dispo: ok}
	}()
	p.enVol = en
	slog.DebugContext(ctx, "post-sync: rejeu 2D — prechargement du film suivant",
		"gamertag", p.d.Gamertag, "match_id", matchID, "budget_restant", restant)
}

// fermer coupe le prechargement en vol et ATTEND sa mort.
//
// L'ATTENTE N'EST PAS FACULTATIVE : sans elle, une goroutine du cycle N ecrirait dans le cache
// pendant le cycle N+1, et le pic memoire du serveur ne serait plus borne par ce que le cycle
// courant a decide. Un cycle qui se termine ne laisse rien derriere lui.
func (p *pontDisque) fermer() {
	if p == nil || p.enVol == nil {
		return
	}
	p.enVol.annuler()
	<-p.enVol.fini
	p.enVol = nil
}
