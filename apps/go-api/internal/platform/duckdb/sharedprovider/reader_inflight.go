package sharedprovider

// reader_inflight.go — QUI tient la base, et depuis quand.
//
// Un lecteur en vol bloque toute prise d'écriture de la base (le drain l'attend, au plus
// drainTimeout, puis abandonne). Le compteur seul ne dit pas QUI : le provider retient donc, pour
// chaque lecteur en vol, l'instant de son Get et les adresses des appels qui l'ont demandé
// (résolues en noms de fonctions seulement au moment d'écrire un journal). Deux usages :
//   - au relâchement, un lecteur tenu au-delà de slowReaderThreshold est journalisé en WARN avec
//     son appelant et sa durée, et compté (shared_provider_reader_slow_total) ;
//   - à l'expiration d'un drain, le WARN « drain timeout » nomme les lecteurs encore en vol.
//
// Coût sur le chemin chaud : une capture de pile bornée (runtime.Callers) et une entrée de map
// sous p.mu, déjà tenu par Get.

import (
	"log/slog"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// defaultSlowReaderThreshold : au-delà, une lecture est journalisée au relâchement. Même frontière
// que defaultRWHoldWatchdog : en dessous les lectures sont saines, au-dessus elles retardent déjà
// les écritures, et à drainTimeout (5 s) elles les font échouer. <= 0 désactive le journal.
const defaultSlowReaderThreshold = 2 * time.Second

// profondeurPileLecteur : appels retenus au-dessus de Get. Les lectures passent souvent par un ou
// deux adaptateurs (SharedAccess.Read, sharedReadDepot…) : il faut remonter jusqu'au demandeur réel.
const profondeurPileLecteur = 6

// lecteurEnVol : un Get non encore relâché.
type lecteurEnVol struct {
	depuis time.Time
	pile   [profondeurPileLecteur]uintptr
	n      int
}

// nouveauLecteur capture l'appelant de Get (skip : runtime.Callers, nouveauLecteur, Get).
func nouveauLecteur() lecteurEnVol {
	l := lecteurEnVol{depuis: time.Now()}
	l.n = runtime.Callers(3, l.pile[:])
	return l
}

// appelant rend la chaîne d'appels du lecteur, du plus proche au plus lointain, sans le paquet.
func (l lecteurEnVol) appelant() string {
	if l.n == 0 {
		return "inconnu"
	}
	frames := runtime.CallersFrames(l.pile[:l.n])
	var noms []string
	for {
		f, encore := frames.Next()
		if nom := f.Function; nom != "" && !strings.HasPrefix(nom, "runtime.") {
			noms = append(noms, nom[strings.LastIndex(nom, "/")+1:])
		}
		if !encore {
			break
		}
	}
	return strings.Join(noms, " < ")
}

// trackReaderLocked comptabilise un lecteur qui vient d'obtenir le handle et rend son numéro.
// Doit être appelé avec p.mu tenu, AVANT de le relâcher — sinon un swap concurrent drainerait sans
// nous attendre.
func (p *providerImpl) trackReaderLocked(l lecteurEnVol) uint64 {
	p.readers++
	readersInUse.Add(1)
	if p.enVol == nil {
		p.enVol = make(map[uint64]lecteurEnVol)
	}
	p.prochainLecteur++
	p.enVol[p.prochainLecteur] = l
	return p.prochainLecteur
}

// releaseReader comptabilise la fin du lecteur id, réveille le drain en attente si c'était le
// dernier, et journalise une lecture tenue au-delà du seuil. Appelé une seule fois par lecteur
// (sync.OnceFunc).
func (p *providerImpl) releaseReader(id uint64) {
	p.mu.Lock()
	l, connu := p.enVol[id]
	delete(p.enVol, id)
	p.readers--
	if p.readers <= 0 && p.drainCh != nil {
		close(p.drainCh)
		p.drainCh = nil
	}
	seuil := p.slowReaderThreshold
	p.mu.Unlock()
	readersInUse.Add(-1)
	if !connu || seuil <= 0 {
		return
	}
	if tenu := time.Since(l.depuis); tenu >= seuil {
		readerSlowTotal.Add(1)
		slog.Warn("sharedprovider: lecture tenue longtemps — toute écriture de la base attend sa fin",
			"path", p.path, "appelant", l.appelant(),
			"held_ms", tenu.Milliseconds(), "threshold_ms", seuil.Milliseconds())
	}
}

// decrireLecteursEnVol rend les lecteurs encore en vol, « appelant (N ms) », les plus anciens
// d'abord, pour le journal d'un drain expiré.
func (p *providerImpl) decrireLecteursEnVol() []string {
	p.mu.Lock()
	lecteurs := make([]lecteurEnVol, 0, len(p.enVol))
	for _, l := range p.enVol {
		lecteurs = append(lecteurs, l)
	}
	p.mu.Unlock()
	maintenant := time.Now()
	sort.Slice(lecteurs, func(i, j int) bool { return lecteurs[i].depuis.Before(lecteurs[j].depuis) })
	out := make([]string, 0, len(lecteurs))
	for _, l := range lecteurs {
		out = append(out, l.appelant()+" ("+strconv.FormatInt(maintenant.Sub(l.depuis).Milliseconds(), 10)+" ms)")
	}
	return out
}
