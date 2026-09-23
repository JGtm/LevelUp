package replayartifacts

// cuisson_non_finalise.go — LE REPORT D'UN FILM PAS ENCORE FINALISÉ, ET LE SIGNAL QUAND IL NE
// L'EST TOUJOURS PAS (lot L3 de PLAN_RETOURS_REJEU_2026-09-23, 2026-09-23).
//
// Extrait de cuisson.go avec la reprise du lot après sa revue adverse (constat L3-R1) : le
// report a désormais une MÉMOIRE (l'attente de chaque film), et cette mémoire ne tenait plus dans
// le fichier de la boucle de cuisson sous le seuil de 500 lignes.
//
// # DEUX ÉTATS QUI SE RESSEMBLENT ET NE SE SOIGNENT PAS PAREIL
//
//	FRAIS          le serveur Halo finalise un film environ une minute après la fin du match
//	               (DÉDUIT, rapport `ctf_ab526724` §2.3), et 23 matchs sur 96 sont détectés avant.
//	               Le report est NOMINAL : INFO, compteur [CompteurFilmsNonFinalises], et le match
//	               revient au cycle suivant par le rattrapage.
//	HORS DÉLAI     le même film, toujours sans morceau des temps forts [DelaiDeFinalisation] après
//	               son PREMIER refus. Ce n'est plus un film frais : c'est un build ou un mode qui
//	               ne publierait plus ses temps forts (ou les numéroterait autrement). WARN, et
//	               compteur DISTINCT [CompteurFilmsNonFinalisesHorsDelai], une fois par film.
//
// # POURQUOI LE SIGNAL SUFFIT, ET POURQUOI AUCUNE QUARANTAINE
//
// La règle « un film n'est archivé que finalisé » a un coût si elle se trompe : rien n'est
// archivé, et un film EXPIRE côté serveur. Mais il vit des SEMAINES (« pas d'expiration en 2
// semaines vérifiée empiriquement », `.ai/archive/V7/MATCH_DURATION_RESEARCH.md`), et
// `levelup archive-films` rattrape TOUT match sans film au cache, hors de la borne du rattrapage
// ([BacklogHorizon]). La perte n'est donc réelle que si la dérive reste INVISIBLE des semaines :
// c'est ce que le compteur hors délai et son WARN empêchent, au bout d'un quart d'heure.
// Archiver les morceaux d'un film non finalisé « en quarantaine » demanderait de les télécharger
// malgré le refus du client (`haloclient.fetchFilmChunks`) et une disposition de cache que
// personne ne lirait — pour protéger une donnée qui ne court aucun risque à cette échelle.
//
// # LA MÉMOIRE EST CELLE DU PROCESSUS, ET C'EST ASSEZ
//
// L'âge se mesure depuis le PREMIER refus vu par ce processus, pas depuis la fin du match : la
// fin du match n'atteint pas le pont disque (`buildWork` ne la porte pas). Un match est détecté
// APRÈS sa fin, donc cet âge est un MINORANT de l'âge réel — le WARN ne peut pas partir trop tôt.
// Un redémarrage remet l'attente à zéro : le WARN part alors un quart d'heure plus tard, jamais
// pas du tout.

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
)

// CompteurFilmsNonFinalises : films dont l'archivage et la cuisson sont REPORTÉS au cycle suivant
// parce que le serveur ne les a pas encore FINALISÉS — leur manifeste ne porte pas le morceau des
// temps forts (cf. `filmcache/finalise.go`). Compté PAR TITRE, à CHAQUE refus, comme le reste du
// paquet.
//
// CE N'EST PAS UN ÉCHEC : un compteur qui monte au rythme des matchs frais est NOMINAL. Ce qui ne
// l'est pas se lit dans [CompteurFilmsNonFinalisesHorsDelai].
const CompteurFilmsNonFinalises = "postsync_replay_films_non_finalises_total"

// CompteurFilmsNonFinalisesHorsDelai : films TOUJOURS non finalisés [DelaiDeFinalisation] après
// leur premier refus. Compté UNE fois par film et par attente. Il doit rester à ZÉRO : un seul
// incrément dit qu'un film que le serveur aurait dû finaliser depuis longtemps ne l'est pas —
// à instruire (manifeste de l'API, build, mode) avant que le film n'expire, puis
// `levelup archive-films` une fois la cause corrigée.
const CompteurFilmsNonFinalisesHorsDelai = "postsync_replay_films_non_finalises_hors_delai_total"

// DelaiDeFinalisation : au-delà, un film non finalisé n'est plus un film frais.
//
// SEUIL NOMMÉ, POSÉ LE 2026-09-23 (reprise du lot L3). Base : le serveur finalise environ une
// minute après la fin du match (DÉDUIT de `ab526724` : manifeste validé à 21:35:50, morceaux 34 à
// 36 servis à 21:36:01 — rapport `ctf_ab526724` §1.2 et §2.3) ; quinze minutes laissent une marge
// de quinze fois. Ce n'est pas un repli : il ne décide aucun fait, il choisit le niveau d'un
// journal et un compteur. CRITÈRE DE RÉVISION : un film compté hors délai puis finalisé plus tard
// (il sort du report et s'archive) prouve que le seuil est trop court — le monter, avec la mesure.
const DelaiDeFinalisation = 15 * time.Minute

// oubliDesAttentes : une attente plus vieille que cette borne est oubliée au prochain refus d'un
// autre film. Elle borne la mémoire du processus ; un film encore refusé après repart d'une
// attente neuve — il sera re-signalé [DelaiDeFinalisation] plus tard, jamais tu.
const oubliDesAttentes = 48 * time.Hour

// attentesDeFinalisation : le premier refus de chaque film encore non finalisé, par titre et match.
//
// SÛRE POUR UN USAGE CONCURRENT, ET IL LE FAUT : le préchargement du pont disque
// (`prefetch.go`) appelle [persistFilmToCache] dans sa propre goroutine.
type attentesDeFinalisation struct {
	mu         sync.Mutex
	premier    map[string]time.Time
	signale    map[string]bool
	maintenant func() time.Time
}

func nouvellesAttentes(maintenant func() time.Time) *attentesDeFinalisation {
	return &attentesDeFinalisation{premier: map[string]time.Time{}, signale: map[string]bool{},
		maintenant: maintenant}
}

// attentes : LA mémoire du processus. Une variable de paquet, parce que la mémoire doit survivre
// aux cycles (chaque cycle reconstruit ses `Deps`) ; les tests la remplacent pour tenir l'horloge.
var attentes = nouvellesAttentes(time.Now)

// noter enregistre un refus. Rend l'âge de l'attente, et `franchi` vrai la PREMIÈRE fois que cet
// âge dépasse [DelaiDeFinalisation] — le moment de compter le film hors délai.
func (a *attentesDeFinalisation) noter(cle string) (age time.Duration, horsDelai, franchi bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.maintenant()
	for k, t0 := range a.premier {
		if k != cle && now.Sub(t0) > oubliDesAttentes {
			delete(a.premier, k)
			delete(a.signale, k)
		}
	}
	t0, vu := a.premier[cle]
	if !vu || now.Sub(t0) > oubliDesAttentes {
		t0 = now
		a.premier[cle] = now
		delete(a.signale, cle)
	}
	age = now.Sub(t0)
	if age <= DelaiDeFinalisation {
		return age, false, false
	}
	franchi = !a.signale[cle]
	a.signale[cle] = true
	return age, true, franchi
}

// oublier clôt l'attente d'un film : il est archivé.
func (a *attentesDeFinalisation) oublier(cle string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.premier, cle)
	delete(a.signale, cle)
}

// cleDAttente : un film par titre et par match.
func cleDAttente(ctx context.Context, matchID string) string {
	return ctxkeys.TitleSlug(ctx) + "/" + matchID
}

// reporterNonFinalise compte et journalise le report d'un film non finalisé. Rend vrai quand
// `err` en est un — l'appelant s'arrête alors là : le film n'est ni archivé ni disponible.
func reporterNonFinalise(ctx context.Context, d Deps, matchID string, err error) bool {
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) {
		return false
	}
	titre := ctxkeys.TitleSlug(ctx)
	observability.AddIntT(titre, CompteurFilmsNonFinalises, 1)
	age, horsDelai, franchi := attentes.noter(cleDAttente(ctx, matchID))
	if !horsDelai {
		slog.InfoContext(ctx, "post-sync: rejeu 2D — film pas encore finalisé côté serveur (sans temps "+
			"forts), archivage et cuisson reportés au cycle suivant",
			"gamertag", d.Gamertag, "match_id", matchID, "attente", age, "err", err)
		return true
	}
	if franchi {
		observability.AddIntT(titre, CompteurFilmsNonFinalisesHorsDelai, 1)
	}
	slog.WarnContext(ctx, "post-sync: rejeu 2D — film TOUJOURS non finalisé au-delà du délai de "+
		"finalisation : le serveur ne publie pas son morceau des temps forts (build ou mode qui "+
		"aurait changé ?) — instruire avant son expiration, puis `levelup archive-films`",
		"gamertag", d.Gamertag, "match_id", matchID, "attente", age,
		"delai", DelaiDeFinalisation, "err", err)
	return true
}
