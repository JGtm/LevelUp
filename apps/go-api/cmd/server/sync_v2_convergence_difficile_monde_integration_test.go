//go:build integration

// sync_v2_convergence_difficile_monde_integration_test.go : faux service Halo des cas difficiles (une liste par joueur, des stats par match, des pannes par match).
//
// Voir le commentaire de sync_v2_convergence_difficile_env_integration_test.go pour le banc dans son ensemble.
package main

import (
	"io"
	"maps"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing/iotest"

	"github.com/JGtm/halo-api/haloapitest"

	"levelup/go-api/internal/observability/logging"
)

const (
	difficileGamertagB = "ConvergenceJoueurB"
	difficileXUIDB     = "2533274800000003"
	// difficileFirstIDB : numéro du premier match du joueur B, pour que ses identifiants ne rencontrent
	// jamais ceux du joueur A (convergenceMatchID formate le numéro sur 4 chiffres hexadécimaux).
	difficileFirstIDB = 3000
)

type difficileJoueur struct{ gamertag, xuid string }

var (
	difficileA = difficileJoueur{convergenceGamertag, convergenceXUID}
	difficileB = difficileJoueur{difficileGamertagB, difficileXUIDB}
)

// ---------------------------------------------------------------------------------------------
// Le faux monde Halo : une liste par joueur, des stats par match, des pannes par match.
// ---------------------------------------------------------------------------------------------

// difficileMonde est le faux service Halo des cas difficiles : un fakeHalo par joueur sert la liste de
// l'historique de CE joueur (la requête porte son xuid), les stats d'un match sont servies avec pour
// participants tous les joueurs suivis dont la liste contient le match. [difficileMonde.transport] en fait
// le transport HTTP du banc.
type difficileMonde struct {
	mu          sync.Mutex
	listes      map[string]*fakeHalo // xuid -> liste
	noms        map[string]string    // xuid -> gamertag
	statsCalls  map[string]int       // match -> demandes de stats depuis la dernière remise à zéro
	statsStatus map[string]int       // match -> statut durable des stats (404, 500...)
	statsTotal  int
	onStats     func(id string, nth int) // crochet appelé avant la réponse, nth = rang de la demande
	unexpected  []string
	postStats   int // demandes de stats du post-sync depuis la dernière remise à zéro
}

func newDifficileMonde() *difficileMonde {
	return &difficileMonde{
		listes: map[string]*fakeHalo{}, noms: map[string]string{},
		statsCalls: map[string]int{}, statsStatus: map[string]int{},
	}
}

// ajouter crée la liste du joueur j : total matchs dont le premier a le numéro first.
func (m *difficileMonde) ajouter(j difficileJoueur, first, total int) *fakeHalo {
	f := &fakeHalo{statsStatus: map[string]int{}, statsCalls: map[string]int{}, next: first}
	f.addOnTop(total)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listes[j.xuid] = f
	m.noms[j.xuid] = j.gamertag
	return f
}

// partagerEnHaut crée n matchs communs plus récents que tout, en haut des listes de a et de b ; rend
// leurs identifiants, le plus récent d'abord.
func (m *difficileMonde) partagerEnHaut(a, b difficileJoueur, n int) []string {
	fa, fb := m.liste(a), m.liste(b)
	ids := fa.addOnTop(n)
	fb.mu.Lock()
	fb.ids = append(append([]string{}, ids...), fb.ids...)
	fb.mu.Unlock()
	return ids
}

// partagerAu remplace, dans la liste de b, les matchs aux positions to.. par ceux de a aux positions from.. :
// des matchs communs déjà profonds dans les deux historiques.
func (m *difficileMonde) partagerAu(a, b difficileJoueur, from, to, n int) {
	fa, fb := m.liste(a), m.liste(b)
	for i := 0; i < n; i++ {
		id := fa.idAt(from + i)
		fb.mu.Lock()
		fb.ids[to+i] = id
		fb.mu.Unlock()
	}
}

func (m *difficileMonde) liste(j difficileJoueur) *fakeHalo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listes[j.xuid]
}

func (m *difficileMonde) setStatus(id string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statsStatus[id] = status
}

func (m *difficileMonde) setOnStats(h func(id string, nth int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onStats = h
}

// resetCounters remet à zéro les lectures de pages de chaque liste et les demandes de stats.
func (m *difficileMonde) resetCounters() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range m.listes {
		f.resetCounters()
	}
	m.statsCalls = map[string]int{}
	m.postStats = 0
	m.statsTotal = 0
}

// reads rend, par xuid, les décalages des pages d'historique demandées depuis la dernière remise à zéro.
func (m *difficileMonde) reads() map[string][]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string][]int{}
	for x, f := range m.listes {
		out[x] = f.historyReads()
	}
	return out
}

func (m *difficileMonde) statsSnapshot() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.statsCalls)
}

func (m *difficileMonde) unexpectedRequests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.unexpected...)
}

// transport rend le http.RoundTripper du faux monde : l'enregistreur partagé des tests de la brique, qui
// sert [difficileMonde.serve].
func (m *difficileMonde) transport() http.RoundTripper {
	return &haloapitest.Enregistreur{Repondre: func(_ int, req *http.Request) *http.Response { return m.serve(req) }}
}

// serve répond en mémoire aux requêtes du client Halo de production.
func (m *difficileMonde) serve(req *http.Request) *http.Response {
	// Comme un vrai transport : une requête au contexte annulé échoue (l'arrêt du cycle coupe la récupération).
	if err := req.Context().Err(); err != nil {
		return reponseContexteAnnule(req, err)
	}
	path := req.URL.Path
	switch {
	case strings.Contains(path, "/players/") && strings.HasSuffix(path, "/matches"):
		m.mu.Lock()
		f := m.listes[between(path, "xuid(", ")")]
		m.mu.Unlock()
		if f == nil {
			return replyJSON(req, map[string]any{"Results": []any{}})
		}
		return f.history(req)
	case strings.Contains(path, "/matches/") && strings.HasSuffix(path, "/stats"):
		return m.stats(req, between(path, "/matches/", "/stats"))
	case strings.Contains(path, "/matches/") && strings.HasSuffix(path, "/skill"):
		return (&fakeHalo{}).skill(req, "")
	case strings.Contains(path, "/films/matches/"):
		return reply(req, http.StatusNotFound, "film absent")
	}
	m.mu.Lock()
	m.unexpected = append(m.unexpected, req.Method+" "+req.URL.Host+req.URL.RequestURI())
	m.mu.Unlock()
	return reply(req, http.StatusNotFound, "non simulé")
}

// reponseContexteAnnule est l'issue d'une requête dont le contexte est annulé : l'enregistreur partagé
// ne peut pas rendre d'erreur de transport, la réponse est donc un 200 dont la lecture du corps échoue
// avec l'erreur du contexte. Le client de la brique classe les deux issues pareil (erreur de transport,
// retentée dans la limite des essais ; l'attente finit sur le contexte).
func reponseContexteAnnule(req *http.Request, err error) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(iotest.ErrReader(err)),
		Request:    req,
	}
}

func (m *difficileMonde) stats(req *http.Request, id string) *http.Response {
	// Les demandes du post-sync (convergence des événements et des scores personnels : le faux monde ne
	// sert ni film ni score, ces passes redemandent les stats des plus récents à chaque cycle) ne sont
	// pas celles de la phase de récupération : comptées à part.
	post := strings.HasPrefix(logging.CurrentEvent(req.Context()), "sync.postSync")
	m.mu.Lock()
	var hook func(string, int)
	var nth int
	if post {
		m.postStats++
	} else {
		m.statsCalls[id]++
		m.statsTotal++
		nth, hook = m.statsTotal, m.onStats
	}
	status := m.statsStatus[id]
	owners, pos := m.ownersOf(id)
	m.mu.Unlock()
	if hook != nil {
		hook(id, nth)
	}
	if status != 0 {
		return reply(req, status, "erreur simulée")
	}
	return replyJSON(req, difficileMatchStats(id, pos, owners, m.noms))
}

// ownersOf rend les xuids suivis dont la liste contient le match (triés) et la position du match dans la
// première liste qui le porte. Appelé sous m.mu.
func (m *difficileMonde) ownersOf(id string) (owners []string, pos int) {
	xuids := make([]string, 0, len(m.listes))
	for x := range m.listes {
		xuids = append(xuids, x)
	}
	sort.Strings(xuids)
	pos = -1
	for _, x := range xuids {
		if f := m.listes[x]; f.has(id) {
			owners = append(owners, x)
			if pos < 0 {
				pos = f.positionOf(id)
			}
		}
	}
	return owners, max(pos, 0)
}

// has dit si la liste contient le match (positionOf rend 0 pour un absent : indiscernable de la tête).
func (f *fakeHalo) has(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.ids {
		if x == id {
			return true
		}
	}
	return false
}

// difficileMatchStats est le corps de stats d'un match : les joueurs suivis qui le portent (premiers),
// puis l'adversaire. Sans propriétaire connu, le joueur A (cas d'un match hors de toute liste).
func difficileMatchStats(id string, pos int, owners []string, noms map[string]string) map[string]any {
	body := convergenceMatchStats(id, pos)
	players := body["Players"].([]any)
	tracked, adversary := players[0].(map[string]any), players[1]
	if len(owners) == 0 {
		owners = []string{convergenceXUID}
	}
	out := make([]any, 0, len(owners)+1)
	for i, x := range owners {
		p := maps.Clone(tracked)
		p["PlayerId"] = "xuid(" + x + ")"
		if n := noms[x]; n != "" {
			p["PlayerName"] = n
		}
		p["Rank"] = float64(i + 1)
		out = append(out, p)
	}
	body["Players"] = append(out, adversary)
	return body
}

// postSyncStats rend le nombre de demandes de stats du post-sync depuis la dernière remise à zéro.
func (m *difficileMonde) postSyncStats() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.postStats
}
