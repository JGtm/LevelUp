package grammar

// vue_a_detonations.go — LES DEUX MESSAGES DE DETONATION/IMPACT DE PROJECTILE DE LA VUE A, LUS AVEC
// LEURS VALEURS (lot detonations du 2026-10-22).
//
// `projectile_detonate` (genre 5, [chargeDetonation]) et `projectile_impact_effect` (genre 6,
// [chargeImpact]) portent chacun, en plus de leur charge (vue_a_charges_armes.go) :
//
//   - une référence 0 en domaine 5 — LA MÊME TABLE que l'entité projectile de `ScanProjectiles`
//     (ti=41, cf. projectiles.go) — qui nomme le projectile dont c'est la détonation ;
//   - une position quantifiée au niveau 0xf (détonation, [niveauDetonation]) ou 0xc (impact,
//     [niveauImpact]), lue par `FUN_14076e524` sur les tables installées sur le lecteur
//     (`br.vueA.positions` : la région jouée, [tablesDeLaRegionJouee]).
//
// Les deux étaient lues pour avancer le curseur de bits puis jetées (`_, ok := lireE494Sur(...)`
// avant ce lot) : AUCUN bit de plus n'est consommé ici, le changement est l'inverse d'un oubli —
// la valeur déjà lue est désormais gardée. Ce fichier range la valeur sur le lecteur ; la lecture
// de la vue A la recueille ([lireLaVueA]). Le balayage d'un film (compte, appariement au
// projectile) vit dans `detonations.go`.

import (
	"github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

// Les genres des deux messages de detonation/impact (numerotation de `FUN_140e453b4`), nommes pour
// ne plus relire les litteraux 5/6 ailleurs que dans cette grammaire.
const (
	GenreProjectileDetonate     = 5
	GenreProjectileImpactEffect = 6
)

// dequantDetonationPosition dequantifie la position que `chargeDetonation`/`chargeImpact` viennent
// de lire (`pos`) et la range sur `br.detonationLue`.
//
// MEMES BORNES QUE `readTranslocVec` (transloc_events.go), POUR LA MEME RAISON : la table DEFAUT
// du build quand la porte est posee (`pos.idx < 0` — `FUN_14076e524` n a alors lu AUCUN index, ses
// largeurs viennent de `profile.LargeursAxeParDefautDuBuild`, qui n a de sens que sur les bornes
// +/-20000 du moteur), celles de la region jouee sinon (`pos.idx >= 0` ; `lireE524Sur` a deja refuse
// tout index different de `br.vueA.positions.region` — jamais une autre region). Une lecture EN
// PLEINE PRECISION (`pos.brute`, garde de `FUN_14076f91c`) ne porte aucun quantum a deplacer : rien
// n est range, [lecture.DetonationPositionLue] reste a 0.
func dequantDetonationPosition(br *Lecteur, pos positionQuantifiee) {
	if pos.brute {
		return
	}
	rng := profile.QuantRangeParDefautDuBuild()
	if pos.idx >= 0 {
		rng = vec3RangeDeBornes(br.vueA.positions.bornesCarte)
	}
	lay := profile.I0Layout{AxisW: pos.w}
	br.detonationLue.X = DequantBipedAxis(uint32(pos.q[0]), 0, lay, rng)
	br.detonationLue.Y = DequantBipedAxis(uint32(pos.q[1]), 1, lay, rng)
	br.detonationLue.Z = DequantBipedAxis(uint32(pos.q[2]), 2, lay, rng)
	br.detonationLue.Drapeaux |= lecture.DetonationPositionLue
}

// messageDeDetonation assemble le message de detonation/impact que la lecture de la vue A vient de
// lire : sa position (deja rangee sur le lecteur par [dequantDetonationPosition]), son genre, et sa
// reference 0 (domaine 5 — le projectile).
func messageDeDetonation(br *Lecteur, genre, debut int) lecture.MessageDeDetonation {
	m := br.detonationLue
	m.Debut = uint32(debut) //nolint:gosec // position dans un payload
	m.Genre = uint8(genre)  //nolint:gosec // genre R(7), 5 ou 6
	if r := br.refsDuMessage[0]; r.presente {
		m.Ref0, m.Gen0 = uint16(r.index), uint8(r.gen) //nolint:gosec // index du domaine 5, 8 bits ; generation 2 bits
		m.Drapeaux |= lecture.DetonationRef0
	}
	return m
}

// aLeDrapeauDetonation dit que le drapeau `d` est pose sur le message `m`.
func aLeDrapeauDetonation(m lecture.MessageDeDetonation, d uint8) bool { return m.Drapeaux&d != 0 }
