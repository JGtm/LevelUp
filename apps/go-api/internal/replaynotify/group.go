// Package replaynotify — GROUPEMENT ANTI-SPAM des artefacts de rejeu qui deviennent
// disponibles (lot B v7.5, point 5 de l'encadré Notion « REPLAY 2D »).
//
// LE PROBLÈME. Un cycle de sync peut ranger plusieurs artefacts en quelques secondes, et
// un ouvrier distant en livre un par job. Une notification par artefact transformerait le
// canal Discord en journal de build — exactement ce que le besoin produit demande
// d'éviter (« attention au spam, faudrait peut-être grouper »).
//
// LA RÈGLE. Le PREMIER artefact d'un titre ARME une fenêtre (10 min par défaut) ; tous
// ceux qui arrivent pendant la fenêtre s'y accumulent ; à l'échéance, UN SEUL lot sort et
// la fenêtre est DÉSARMÉE. L'artefact suivant en réarme une neuve.
//
// POURQUOI PAS UNE FENÊTRE GLISSANTE. Une fenêtre repoussée à chaque arrivée ne se
// fermerait jamais sous un flux continu (backfill, rattrapage d'un ouvrier au réveil) : le
// message ne partirait qu'une fois le flux tari, c'est-à-dire au pire moment.
//
// GROUPEMENT PAR TITRE. Chaque titre a SA fenêtre, sa langue et ses libellés (le message
// est rendu par internal/notify avec `LabelsForSlug`). Mélanger deux titres dans un même
// message imposerait de choisir des libellés pour l'autre — c'est ce que la règle
// title-agnostic interdit.
//
// HORLOGE INJECTÉE, AUCUN MINUTEUR ICI. `Add` et `Due` reçoivent l'instant en paramètre :
// ce paquet n'appelle jamais time.Now, ne démarre aucune goroutine et n'a aucun ticker.
// C'est la boucle appelante (internal/api/wire) qui bat la mesure. Conséquence : les tests
// sont déterministes et instantanés, sans le moindre sleep (patron `ops.ShouldNotifyDisk`).
//
// PERTE DU GROUPE EN COURS AU REDÉMARRAGE : ACCEPTÉE, ET DÉLIBÉRÉMENT NON PERSISTÉE.
// Un redémarrage du serveur pendant une fenêtre armée perd les événements en attente :
// aucun message ne partira pour ces artefacts. Ce choix diverge de `ops.DiskWatchState`,
// qui persiste le sien — et la différence est le SENS du risque. L'état disque est
// persisté pour ne PAS re-notifier (une rafale d'alertes identiques a été observée en prod
// après des redémarrages en boucle) ; ici le risque symétrique n'existe pas : au pire on
// notifie MOINS. Ce qu'on perd est un message, jamais une donnée — les artefacts sont sur
// le disque et l'application continue de les servir. Persister imposerait un magasin de
// fichiers de plus, sa réhydratation et son cas « fichier corrompu », pour un gain nul.
package replaynotify

import (
	"sort"
	"sync"
	"time"
)

const (
	// DefaultWindow : durée d'une fenêtre de groupement. Le 1er artefact l'arme, le
	// message part à l'échéance. 10 min = le compromis du besoin produit : assez long
	// pour absorber un cycle de sync entier et une rafale d'ouvrier, assez court pour
	// que « ton rejeu est prêt » reste une information et pas un rappel.
	DefaultWindow = 10 * time.Minute

	// MaxListed : nombre maximum de matchs ÉNUMÉRÉS dans un message. Au-delà, le
	// message dit « et N autres ». Discord plafonne une description à 4096 caractères
	// et la valeur d'un champ à 1024 : une liste non bornée ferait rejeter l'envoi
	// entier, donc perdre l'information au lieu de la tronquer.
	MaxListed = 20

	// MaxPending : nombre maximum de matchs MÉMORISÉS par titre et par fenêtre.
	// Au-delà, seul le compteur monte. Ce plafond n'est pas cosmétique : sans lui, une
	// instance sans webhook configuré (le lot est alors construit puis jeté) ferait
	// croître la mémoire au rythme des artefacts, sans fin.
	MaxPending = 200
)

// Event : un artefact de rejeu qui vient de devenir disponible.
//
// Le groupeur ne connaît que l'identité — ni chemin, ni taille. Tout le reste (libellé de
// carte, lien vers la page de rejeu) est résolu au moment du flush par l'appelant, qui
// seul sait lire la base et construire une URL.
type Event struct {
	TitleSlug string
	MatchID   string
}

// Batch : ce qu'un flush rend pour un titre.
//
// Total compte les artefacts DISTINCTS de la fenêtre (y compris ceux qu'on n'énumère pas) ;
// MatchIDs en porte au plus MaxListed, dans l'ordre d'arrivée ; Omitted = Total - len(MatchIDs).
type Batch struct {
	TitleSlug string
	MatchIDs  []string
	Total     int
	Omitted   int
}

// Grouper accumule les artefacts par titre et rend les lots à échéance.
// Sûr en concurrence : la publication vient du chemin d'écriture (quel que soit le
// goroutine), le flush vient de la boucle.
type Grouper struct {
	mu      sync.Mutex
	window  time.Duration
	byTitle map[string]*fenetre
}

// fenetre : l'état d'un titre pendant une fenêtre armée.
type fenetre struct {
	armeeA   time.Time
	matchIDs []string            // ordre d'arrivée, plafonné à MaxPending
	vus      map[string]struct{} // déduplication
	deborde  int                 // artefacts distincts non mémorisés (plafond atteint)
}

// New construit un groupeur. window <= 0 → DefaultWindow.
func New(window time.Duration) *Grouper {
	if window <= 0 {
		window = DefaultWindow
	}
	return &Grouper{window: window, byTitle: map[string]*fenetre{}}
}

// Add enregistre un artefact disponible. Le premier d'un titre ARME la fenêtre de ce titre.
//
// DÉDUPLICATION par (titre, match) : une re-cuisson du même match dans la fenêtre ne
// compte qu'une fois, et n'AVANCE PAS l'échéance — l'instant d'armement reste celui du
// tout premier artefact, sinon un match re-cuit en boucle repousserait le message.
//
// Un événement sans titre ni match est ignoré : il n'y aurait rien à énumérer. L'appelant
// valide et JOURNALISE ce cas avant d'appeler (le groupeur reste pur, il ne loggue pas).
func (g *Grouper) Add(now time.Time, ev Event) {
	if ev.TitleSlug == "" || ev.MatchID == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	f, ok := g.byTitle[ev.TitleSlug]
	if !ok {
		f = &fenetre{armeeA: now, vus: map[string]struct{}{}}
		g.byTitle[ev.TitleSlug] = f
	}
	if _, dejaVu := f.vus[ev.MatchID]; dejaVu {
		return
	}
	if len(f.vus) >= MaxPending {
		// Plafond atteint : on ne mémorise plus l'identité (donc plus de déduplication
		// au-delà), mais on continue de COMPTER — le message dira combien.
		f.deborde++
		return
	}
	f.vus[ev.MatchID] = struct{}{}
	f.matchIDs = append(f.matchIDs, ev.MatchID)
}

// Due rend les lots dont la fenêtre est ÉCHUE à `now`, et les retire du groupeur (la
// fenêtre est désarmée : le prochain artefact du titre en réarmera une neuve).
//
// Lots triés par titre : deux titres échus au même tick doivent sortir dans un ordre
// stable (parcours de map non déterministe en Go).
func (g *Grouper) Due(now time.Time) []Batch {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []Batch
	for slug, f := range g.byTitle {
		if now.Sub(f.armeeA) < g.window {
			continue
		}
		total := len(f.matchIDs) + f.deborde
		listes := f.matchIDs
		if len(listes) > MaxListed {
			listes = listes[:MaxListed]
		}
		out = append(out, Batch{
			TitleSlug: slug,
			MatchIDs:  append([]string(nil), listes...),
			Total:     total,
			Omitted:   total - len(listes),
		})
		delete(g.byTitle, slug)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TitleSlug < out[j].TitleSlug })
	return out
}

// Pending rend l'état courant (titres armés, artefacts mémorisés) — sert la jauge et le
// log de la boucle appelante.
func (g *Grouper) Pending() (titles, artifacts int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, f := range g.byTitle {
		artifacts += len(f.matchIDs) + f.deborde
	}
	return len(g.byTitle), artifacts
}
