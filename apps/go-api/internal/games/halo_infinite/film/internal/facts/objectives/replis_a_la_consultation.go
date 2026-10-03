package objectives

// replis_a_la_consultation.go — LE REPLI QUI SE DECLENCHE A LA CONSULTATION, COMPTE PAR
// EVENEMENT DISTINCT (lot J8.7-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision du
// superviseur du 2026-09-28).
//
// # LE PROBLEME
//
// `repli_emission_hors_domaine_jetee` (les filtres de [rawSeriesByRound] et [rawSeriesByKey]) ne vit pas
// dans une construction qui rend un resultat unique : il se declenche dans des LECTURES que chaque
// calque de `replay` et la cuisson (`replaybuild`) refont sur les memes enregistrements. Compter par
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
//
// POURQUOI PAR DOCUMENT ET PAS PAR RESOLVEUR. Un document resout son identite par manche jusqu a
// quatre fois (le pont de la cuisson, puis la couronne, les compteurs de joueurs multi-manche, et le
// drapeau ou le crane sans pont fourni) : un enregistreur par resolveur compterait deux fois l emission
// que deux resolutions lisent. La cuisson cree donc l enregistreur et le passe aux options du rejeu
// (`replay.ReplisHorsBalayage.Consultations`), que la table de versement lit en fin d assemblage.
//
// Nil ne compte rien : c est la valeur des outils hors production (`cmd/`) et des tests qui ne
// regardent pas les replis. Le garde-rail `archlint/film_consultations_comptees_test.go` interdit le
// nil litteral dans les lectures de `replay` et de `replaybuild`.

import (
	"sync"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ReplisALaConsultation retient les evenements DISTINCTS du repli qui se declenche a la lecture. La valeur zero est prete a l emploi ; elle se partage par pointeur.
type ReplisALaConsultation struct {
	mu        sync.Mutex
	emissions map[cleDEmission]struct{}
	// diag : les DIAGNOSTICS des lectures faites sous cet enregistreur (bornes de deroulage, serie
	// non chronologique — lot J12.3, ADR 0034 D-4). L assemblage qui le porte les releve.
	diag constat.Diagnostics
}

// cleDEmission : la serie (composant, cote, slot, manche) et l instant de l emission jetee.
type cleDEmission struct {
	key    statSlotKey
	slot   int
	round  int
	timeMS int
}

// Diagnostics rend les diagnostics des lectures faites sous cet enregistreur (lot J12.3). nil
// sur un enregistreur nil : les outils hors production n en recueillent pas.
func (r *ReplisALaConsultation) Diagnostics() *constat.Diagnostics {
	if r == nil {
		return nil
	}
	return &r.diag
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

// ComptesDesReplis rend le nombre d evenements DISTINCTS notes. Nil : zero.
func (r *ReplisALaConsultation) ComptesDesReplis() ComptesDesReplis {
	if r == nil {
		return ComptesDesReplis{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return ComptesDesReplis{
		EmissionsHorsDomaineJetees: len(r.emissions),
	}
}
