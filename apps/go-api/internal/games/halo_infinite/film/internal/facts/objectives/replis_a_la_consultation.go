package objectives

// replis_a_la_consultation.go — LES DEUX REPLIS QUI SE DECLENCHENT A LA CONSULTATION, COMPTES PAR
// EVENEMENT DISTINCT (lot J8.7-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision du
// superviseur du 2026-09-28).
//
// # LE PROBLEME
//
// `repli_emission_hors_domaine_jetee` (les filtres de [rawSeriesByRound] et [rawSeriesByKey]) et
// `repli_instant_sur_la_premiere_manche` ([RoundIdentity.roundOfTime]) ne vivent pas dans une
// construction qui rend un resultat unique : ils se declenchent dans des LECTURES que chaque calque
// de `replay` et la cuisson (`replaybuild`) refont sur les memes enregistrements. Compter par
// consultation ferait dependre le compte du nombre de calques qui lisent la meme emission — un compte
// qui mesure l architecture, pas le film.
//
// # LA REGLE
//
// Un enregistreur PAR DOCUMENT, partage par POINTEUR : toutes les lectures d un document (les copies
// d un [RoundIdentity], et les series que chaque calque redemande) y notent la CLE de l evenement, et
// le compte est la taille de l ensemble. N lectures du meme evenement le comptent une fois ; le compte
// ne depend ni de l ordre ni du nombre des lectures.
//
//	emission jetee          (composant, cote, slot, manche, instant) — la cle de la serie et l instant
//	instant avant manches   l instant
//
// POURQUOI PAR DOCUMENT ET PAS PAR RESOLVEUR. Un document resout son identite par manche jusqu a
// quatre fois (le pont de la cuisson, puis la couronne, les compteurs de joueurs multi-manche, et le
// drapeau ou le crane sans pont fourni) : un enregistreur par resolveur compterait deux fois l instant
// que deux resolutions lisent. La cuisson cree donc l enregistreur et le passe aux options du rejeu
// (`replay.ReplisHorsBalayage.Consultations`), que la table de versement lit en fin d assemblage.
//
// Nil ne compte rien : c est la valeur des outils hors production (`cmd/`) et des tests qui ne
// regardent pas les replis. Le garde-rail `archlint/film_consultations_comptees_test.go` interdit le
// nil litteral dans les lectures de `replay` et de `replaybuild`.

import (
	"sync"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ReplisALaConsultation retient les evenements DISTINCTS des deux replis qui se declenchent a la
// lecture. La valeur zero est prete a l emploi ; elle se partage par pointeur.
type ReplisALaConsultation struct {
	mu        sync.Mutex
	emissions map[cleDEmission]struct{}
	instants  map[int]struct{}
}

// cleDEmission : la serie (composant, cote, slot, manche) et l instant de l emission jetee.
type cleDEmission struct {
	key    statSlotKey
	slot   int
	round  int
	timeMS int
}

// noterEmissionJetee note une emission que le filtre de domaine jette (`repli_emission_hors_domaine_jetee`).
func (r *ReplisALaConsultation) noterEmissionJetee(key statSlotKey, rec types.StatRecord) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.emissions == nil {
		r.emissions = map[cleDEmission]struct{}{}
	}
	r.emissions[cleDEmission{key: key, slot: rec.Slot, round: rec.Round, timeMS: rec.TimeMS}] = struct{}{}
}

// noterInstantAvantLesManches note un instant anterieur a toute manche connue, range dans la
// premiere (`repli_instant_sur_la_premiere_manche`).
func (r *ReplisALaConsultation) noterInstantAvantLesManches(timeMS int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.instants == nil {
		r.instants = map[int]struct{}{}
	}
	r.instants[timeMS] = struct{}{}
}

// ComptesDesReplis rend le nombre d evenements DISTINCTS notes. Nil : zero.
func (r *ReplisALaConsultation) ComptesDesReplis() ComptesDesReplis {
	if r == nil {
		return ComptesDesReplis{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return ComptesDesReplis{
		EmissionsHorsDomaineJetees:  len(r.emissions),
		InstantsSurLaPremiereManche: len(r.instants),
	}
}
