package grammar

// keyframe_scan_counters.go — LES COMPTEURS DE FERMETURE D'IMAGE-CLE DES BALAYAGES DE
// PRODUCTION (lot 1.4.3, ADR 0009).
//
// # POURQUOI CE FICHIER EXISTE
//
// Le releve du 2026-09-13 (`NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`, section C.1) disait un
// manque en toutes lettres : « IL N'EXISTE AUCUN COMPTEUR DE PRODUCTION "LA TABLE D'IMAGE-CLE
// A DERAILLE" ». `KillSourceHealth` compte des candidats d'attribution de mort, pas des
// records ; les seuls compteurs de cette nature etaient les `Key*` des deux balayages, et ils
// ne sortaient jamais du processus. Une table d'image-cle qui deraille en production etait donc
// INVISIBLE — c'est exactement ce que le lot 1.4 corrige, et une correction qu'on ne peut pas
// observer n'est pas tenue.
//
// # LA DEFINITION FORTE, PAS LA FAIBLE
//
// `KeyChained` dit seulement que la position d'arrivee porte un motif d'en-tete valide : c'est
// la definition FAIBLE de la sante, et la production y plafonnait a 1,9 % (mesure du
// 2026-09-13) alors qu'elle fermait 0 record sur 390. Ce qui se publie ici est la definition
// FORTE : `closed` = la marche atterrit EXACTEMENT sur le premier bit du record suivant ;
// `total` = les records BORNES de l'archetype, ceux qui ont une frontiere a atteindre.
//
// # LE PAQUET NE DEPEND PAS DE `internal/observability`
//
// Meme patron que `KillSourceHealth.ExpvarPairs` : `filmdec` NOMME ses compteurs, l'appelant
// les cable. C'est ce qui garde le decodeur sans dependance interne et ces fonctions testables
// sans expvar. Le seul cableur de production est `replay.decodeFilmBombReads`
// (`bomb_armings.go`), le seul appelant de `ScanNavpointRadial`.

import "fmt"

// keyframeClosureExpvarPairs nomme les deux compteurs de fermeture d'un archetype.
//
// NOMMAGE (ADR 0009) : `<categorie>_<sous_cle>` en snake_case. Le plan ecrit
// `grammar.keyframe.<ti>.{closed,total}` ; la forme physique du depot est
// `filmdec_keyframe_ti<N>_{closed,total}`, parce que c'est la convention que tout
// `/debug/vars` de ce service suit deja (`killsource_*`, `replay_artifact_*`).
//
// AUCUN RATIO N'EST PUBLIE, pour la meme raison qu'au `KillSourceHealth` : un ratio se calcule
// a la lecture, sinon une agregation multi-films moyennerait des pourcentages.
func keyframeClosureExpvarPairs(ti, closed, bounded int) []ExpvarPair {
	return []ExpvarPair{
		{Name: fmt.Sprintf("filmdec_keyframe_ti%d_closed", ti), Value: int64(closed)},
		{Name: fmt.Sprintf("filmdec_keyframe_ti%d_total", ti), Value: int64(bounded)},
	}
}

// KeyframeExpvarPairs rend les compteurs de fermeture d'image-cle de l'anneau ti=12, prets pour
// `observability.AddInt`. C'est LE balayage de production de la table d'image-cle.
func (s *NavpointRadialScan) KeyframeExpvarPairs() []ExpvarPair {
	if s == nil {
		return nil
	}
	return keyframeClosureExpvarPairs(navpointRadialArchIndex, s.KeyClosed, s.KeyBounded)
}

// IL N'Y A PAS D'EQUIVALENT POUR `ScanObjectives` (ti=11), ET C'EST VOULU. Ce balayage n'a
// aucun appelant de production — son propre en-tete le declare instrument de mesure — donc une
// methode de publication y serait du code que rien n'appelle. Ses comptes de fermeture vivent
// dans `ObjectiveScan` (`KeyClosed`, `KeyBounded`), lisibles par qui le mesure ; le jour ou ce
// balayage entre en production, la paire se nomme ici, en une ligne.
