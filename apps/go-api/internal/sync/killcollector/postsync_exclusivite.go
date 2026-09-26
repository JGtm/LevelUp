package killcollector

// postsync_exclusivite.go — UNE SEULE PASSE POST-SYNC PAR (PROCESSUS, TITRE) A LA FOIS.
//
// # LE DEFAUT QU ELLE FERME (OPS-3, audit du decodeur de film, 2026-09-25)
//
// Le cycle v2 lance le post-sync de CHAQUE joueur dans sa propre goroutine
// (`sync/v2.RunPostSync`, `PostSyncParallelism = 0`), et chaque moteur porte son propre
// [PostSyncHook]. Or l arriere de l etape 1.57 est GLOBAL (`backlogAJour` ne filtre aucun
// joueur) : N joueurs synchronises ensemble lisaient le MEME arriere, en tiraient les MEMES
// films et les decodaient N fois en parallele. La borne [DefaultPostSyncPerCycle] valait
// N x 8 films par cycle, et la doctrine « le post-sync du serveur garde la boucle en serie »
// (en-tete de collector.go) n etait vraie qu a l interieur d UN appel.
//
// # LA REGLE
//
// Un appel prend la passe de son titre par `TryLock`, APRES ses gardes (dependances,
// capability `film.kill_source`) et AVANT son segment de lecture de l arriere ; il la rend par
// `defer`. Le perdant se retire : il rend 0, incremente
// [CompteurPostSyncPasseDejaEnCours] et journalise en DEBUG. CE N EST PAS UN DEFAUT : la
// passe en cours traite le meme arriere global, trie du plus recent au plus vieux, ou les
// matchs que le perdant vient d inserer sont en tete.
//
// # CE QU ELLE N EST PAS
//
//   - pas un `singleflight` par match : les perdants attendraient en file indienne, iteraient
//     N passes a la suite et compteraient N fois `ecrits` ;
//   - pas une etape deplacee au niveau du cycle : l orchestrateur v2 ignore les etapes
//     (ADR 0027 inchange) ;
//   - pas un verrou inter-processus : l exclusivite entre processus sur une meme base releve
//     du modele mono-writer (ADR 0013, ADR 0016), pas de cette etape.
//
// La cle est le titre et non le hook : un hook vit par moteur, donc par joueur. Un processus
// multi-titre garde une passe par titre (deux arrieres distincts, dans deux bases distinctes).

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/observability"
)

var (
	passesEnCoursMu sync.Mutex
	passesEnCours   = map[string]*sync.Mutex{}
)

// verrouDePasse rend (et cree au besoin) le verrou de passe d un titre. Meme forme que
// `platform/dblease.leaseMutex` : le registre ne retrecit jamais, il compte un verrou par titre
// actif du processus.
func verrouDePasse(titre string) *sync.Mutex {
	passesEnCoursMu.Lock()
	defer passesEnCoursMu.Unlock()
	if mu, ok := passesEnCours[titre]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	passesEnCours[titre] = mu
	return mu
}

// signalerPasseDejaEnCours compte et journalise le retrait d un appel perdant. DEBUG, pas WARN :
// c est l etat normal d un cycle qui synchronise plusieurs joueurs du meme titre.
func signalerPasseDejaEnCours(ctx context.Context, d PostSyncDeps) {
	observability.AddInt(CompteurPostSyncPasseDejaEnCours, 1)
	slog.DebugContext(ctx, "post-sync: killsource — passe deja en cours pour ce titre, appel retire",
		"title", d.TitleSlug, "gamertag", d.Gamertag)
}
