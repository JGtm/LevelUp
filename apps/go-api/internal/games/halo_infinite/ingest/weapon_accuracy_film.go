package ingest

// weapon_accuracy_film.go — MAPPER de la precision par arme d Infinite DEPUIS LE FILM. Miroir
// Infinite du mapper natif Halo 5 (internal/games/halo_5/ingest/weapon_accuracy.go) : la ou H5
// derive shots_fired/shots_landed des events natifs `weapon_drop`, Infinite n a AUCUN compteur
// natif — le numerateur (touches) se reconstruit du film par l appariement tir<->degat
// (filmdec.PairWeaponHits, methode PAR LE TIR, NOTE_ATTRIBUTION_ARME_TIR_2026-08-31).
//
// ENTREE : les stats du film par (FilmIndex, WeaponID) + LE PONT FilmIndex->xuid (le film ne porte
// aucun xuid cote replication : l identite se resout par l indice, cf. killcollector).
//
// SORTIE, DEUX SURFACES :
//   - []persist.WeaponAccuracyInsert : shots_fired = ShotsPaired (tirs appariables), shots_landed =
//     Hits (tirs apparies a >=1 degat), drops = 0 (non pertinent Infinite — pas de ramassage/lacher).
//     Ecrit dans la table PARTAGEE weapon_accuracy (la meme que H5).
//   - persist.WeaponHitDistanceBatch : par arme, l histogramme des distances des touches +
//     l effectif dist_n. Table SOEUR match_weapon_hit_distance (grain identique).
//
// LE PONT FilmIndex->xuid : resolveXUID rend "" pour un indice non rattache (bot, ou indice absent
// du roster). Une cle sans xuid n a PAS de ligne — mieux vaut une arme absente de la table qu une
// arme attribuee au mauvais joueur (meme doctrine que la ventilation des tirs). Deux FilmIndex
// resolvant au meme xuid sont AGREGES (collision d indice : on somme plutot que de perdre).

import (
	"encoding/json"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/persist"
)

// filmAccuracyKey : le grain de sortie (xuid, arme). Le FilmIndex a deja ete resolu en xuid.
type filmAccuracyKey struct {
	xuid   string
	weapon uint64
}

// filmAccuracyAgg : les compteurs cumules d une cle (xuid, arme).
type filmAccuracyAgg struct {
	shotsPaired int
	hits        int
	buckets     []int
}

// MapWeaponAccuracyFilm agrege les stats film par (xuid, weapon_id) et rend les deux surfaces
// (precision + distance). decoderRev estampille la passe de distance (WeaponHitDistanceDecoderRev).
// Une cle sans xuid resolu, sans arme, ou sans aucun tir appariable est ecartee.
func MapWeaponAccuracyFilm(
	matchID string,
	stats []filmdec.WeaponHitStats,
	resolveXUID func(filmIndex int) string,
	decoderRev string,
) ([]persist.WeaponAccuracyInsert, persist.WeaponHitDistanceBatch) {
	acc := map[filmAccuracyKey]*filmAccuracyAgg{}
	order := make([]filmAccuracyKey, 0)

	for i := range stats {
		s := stats[i]
		xuid := resolveXUID(s.FilmIndex)
		if xuid == "" || s.WeaponID == 0 || s.ShotsPaired == 0 {
			continue
		}
		k := filmAccuracyKey{xuid: xuid, weapon: s.WeaponID}
		a := acc[k]
		if a == nil {
			a = &filmAccuracyAgg{buckets: make([]int, filmdec.WeaponHitBucketCount())}
			acc[k] = a
			order = append(order, k)
		}
		a.shotsPaired += s.ShotsPaired
		a.hits += s.Hits
		for bi, c := range s.DistBuckets {
			if bi < len(a.buckets) {
				a.buckets[bi] += c
			}
		}
	}

	dist := persist.WeaponHitDistanceBatch{MatchID: matchID, DecoderRev: decoderRev}
	if len(order) == 0 {
		return nil, dist
	}
	return assembleFilmSurfaces(matchID, dist, order, acc)
}

// assembleFilmSurfaces materialise les deux surfaces dans un ordre STABLE (premiere apparition
// d une cle). Une passe doit etre reproductible : deux executions sur le meme film ecrivent les
// memes lignes dans le meme ordre, sinon un diff de controle ne veut rien dire.
func assembleFilmSurfaces(
	matchID string, dist persist.WeaponHitDistanceBatch,
	order []filmAccuracyKey, acc map[filmAccuracyKey]*filmAccuracyAgg,
) ([]persist.WeaponAccuracyInsert, persist.WeaponHitDistanceBatch) {
	accRows := make([]persist.WeaponAccuracyInsert, 0, len(order))
	for _, k := range order {
		a := acc[k]
		accRows = append(accRows, persist.WeaponAccuracyInsert{
			MatchID:     matchID,
			XUID:        k.xuid,
			WeaponID:    k.weapon,
			ShotsFired:  a.shotsPaired,
			ShotsLanded: a.hits,
			Drops:       0,
		})
		distN := 0
		for _, c := range a.buckets {
			distN += c
		}
		if distN == 0 {
			continue // aucune touche a distance resolue : pas de ligne distance (histogramme vide)
		}
		dist.Rows = append(dist.Rows, persist.WeaponHitDistanceRow{
			XUID:           k.xuid,
			WeaponID:       k.weapon,
			DistBucketJSON: marshalBuckets(a.buckets),
			DistN:          distN,
		})
	}
	return accRows, dist
}

// marshalBuckets serialise l histogramme de distance en JSON (tableau de comptes par tranche).
// Une erreur de serialisation d un []int est impossible en pratique ; on rend "[]" par prudence
// plutot que de propager une erreur qui ne peut pas survenir.
func marshalBuckets(buckets []int) string {
	b, err := json.Marshal(buckets)
	if err != nil {
		return "[]"
	}
	return string(b)
}
