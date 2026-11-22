package killcollector

// postsync_bornes.go — les bornes par cycle de la source des frags : garde-fou et valeur hebergee de la
// capacite FilmCatchUp du profil de deploiement, budget de temps de la passe.

import "time"

// GardeFouFilmsParCycle : le nombre maximal de films qu'une passe de film du post-sync traite par
// cycle SOUS LA CAPACITE FilmCatchUp du profil de deploiement (desktop ; ailleurs, [FragsParCycleHeberge]
// et `replayartifacts.RejeuxParCycleHeberge`). C'EST UN GARDE-FOU, PAS LA LIMITE : la limite d'une passe
// est son TEMPS ([PostSyncBudget], `replayartifacts.BudgetParCycle`), parce qu'un match synchronise l'est en totalite, film traite
// compris, et que le cout d'un film varie d'un facteur dix. La borne ne mord que si une passe
// enchainait des centaines de films plus vite que le budget — elle empeche alors une liste de
// travail sans fin, pas un debit.
//
// UNE SEULE VALEUR POUR LES DEUX ETAPES FILM (cette source des frags et les artefacts de rejeu,
// `replayartifacts.maxPerCycle`) : deux copies auraient diverge. Elle borne aussi la LISTE de travail
// lue, donc la lecture de `perCycle` cartes avant le premier telechargement : la relever n'est pas
// gratuit, c'est une resolution de carte par match retenu.
const GardeFouFilmsParCycle = 200

// FragsParCycleHeberge : le nombre de films dont la source des frags est lue par cycle HORS de la capacite
// FilmCatchUp (serveur hebergé, 2 vCPU et 2 Go) : la borne d'avant le rattrapage de l'arriere de films, ou la
// limite d'une passe est ce nombre autant que son temps. Le solde est repris au cycle suivant, ou par
// `levelup backfill-killsource --online`.
const FragsParCycleHeberge = 8

// DefaultPostSyncPerCycle borne le nombre de films decodes par cycle quand l'appelant n'en donne pas :
// [GardeFouFilmsParCycle]. Le moteur de synchronisation, lui, part de [FragsParCycleHeberge] et ne monte au
// garde-fou que sous la capacite FilmCatchUp ([PostSyncHook.WithFilmCatchUp]). Le solde eventuel est repris
// au cycle suivant (l'etape est idempotente, `decoder_rev` fait foi) ou par `levelup backfill-killsource --online`.
//
// C EST UNE BORNE PAR CYCLE ET PAR TITRE, PAS PAR JOUEUR. Elle ne l est que parce qu une seule
// passe tourne par (processus, titre) — cf. `postsync_exclusivite.go`. Avant cette exclusivite
// (OPS-3), chaque joueur du cycle lancait sa propre passe sur le meme arriere, et la borne
// valait N fois la borne pour N joueurs synchronises ensemble.
const DefaultPostSyncPerCycle = GardeFouFilmsParCycle

// PostSyncBudget : LA LIMITE DE DUREE DU CYCLE, et la seule qui borne vraiment la passe.
//
// Le cout d un film varie d un facteur dix : le nombre de films ne dit rien de la duree. Sans
// budget, un lot de gros films rallongerait un cycle de sync de plusieurs dizaines de minutes, et
// le reste du post-sync (PSA, agregats, medias) attendrait derriere. Le budget arrete la passe ENTRE
// deux matchs : ce qui reste est repris au cycle suivant, l etape etant idempotente (`decoder_rev`
// fait foi). La borne en nombre ([GardeFouFilmsParCycle]) n est qu un garde-fou derriere lui.
const PostSyncBudget = 5 * time.Minute

// WithFilmCatchUp regle la borne de films par cycle selon la capacite FilmCatchUp du profil : vraie,
// [GardeFouFilmsParCycle] (le temps borne la passe) ; fausse, [FragsParCycleHeberge]. Le moteur de
// synchronisation l'appelle depuis le hook des rejeux, qui porte la capacite (`replayartifacts.NewHook`).
func (h *PostSyncHook) WithFilmCatchUp(on bool) *PostSyncHook {
	h.perCycle = FragsParCycleHeberge
	if on {
		h.perCycle = GardeFouFilmsParCycle
	}
	return h
}

// PerCycle rend la borne de films par cycle en vigueur.
func (h *PostSyncHook) PerCycle() int { return h.perCycle }
