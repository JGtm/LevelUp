// Package analysis — weapon_distance_records.go : LE FRAG LE PLUS LOINTAIN PAR ARME.
//
// Section « Records de distance par arme » de la Synthèse
// (plan .ai/V7.5/PLAN_RECORDS_DISTANCE_2026-09-20.md).
//
// # POURQUOI CE N'EST PAS `WeaponRangeAggregate`
//
// L'agrégat de portée rend `Min`/`Max` par couple (arme, côté) mais PERD la clé du frag : un
// maximum sans son match et son instant n'est qu'un nombre, invérifiable. Un record est un
// OBJET différent : un frag unique, identifié, que le joueur peut aller revoir dans le rejeu.
// C'est la seule façon de publier un extrême sans le faire passer pour une statistique — la
// doctrine D6 interdit de TRACER min/max comme une portée ; ici on ne trace pas une portée, on
// NOMME un frag, avec son effectif à côté pour que le lecteur le pèse.
//
// # AUCUN SEUIL D'EFFECTIF
//
// Décision utilisateur du 2026-09-20 : un record sur trois frags est un record. La médiane
// accompagne le record comme repère d'habitude, calculée sur les mêmes frags, sans seuil non
// plus. Les deux nombres décrivent la même population : ce que le décodeur a su placer.
//
// PUR : aucune I/O, aucune traduction de sens (la clé d'arme entre déjà classifiée, la classe
// d'arme qui décide d'une exclusion vit chez l'appelant qui la lit du registre).
package analysis

import "sort"

// WeaponDistanceRecord : une arme, son frag mesuré le plus lointain, et le contexte pour le peser.
type WeaponDistanceRecord struct {
	WeaponKey string
	// Measured est le nombre de frags mesurés de cette arme — le dénominateur du record.
	Measured int
	// MedianM est la distance médiane de ces frags, en mètres — le repère d'habitude.
	MedianM float64
	// RecordM est la distance du frag le plus lointain, en mètres.
	RecordM float64
	// RecordMatchID, RecordKillerXUID, RecordTimeMS IDENTIFIENT ce frag : c'est par eux que
	// l'appelant retrouve le match, puis l'instant dans le rejeu.
	RecordMatchID    string
	RecordKillerXUID string
	RecordTimeMS     int64
}

// WeaponDistanceRecords rend, pour chaque arme du côté demandé, son frag le plus lointain.
//
// TRI PAR RECORD CROISSANT : la règle se lit du contact vers la longue portée, et l'ordre est
// décidé ICI, jamais rejoué côté web (deux tris du même fait divergeraient au premier
// changement). À record égal, la clé d'arme départage — l'ordre ne dépend jamais de l'ordre
// d'arrivée des lignes SQL.
//
// DÉPARTAGE DU RECORD LUI-MÊME : à distance strictement égale entre deux frags d'une même
// arme, le plus petit dans la clé (match_id, time_ms) l'emporte, pour la même raison.
//
// Une clé d'arme vide n'est pas une arme : la ligne est ignorée (le producteur écarte déjà
// les sources hors registre, ceci n'est qu'une défense).
//
// PUR : l'entrée n'est ni triée ni mutée (le tri porte sur une copie des distances).
func WeaponDistanceRecords(kills []MeasuredKill, side Side) []WeaponDistanceRecord {
	type acc struct {
		rec  MeasuredKill
		dist []float64
	}
	byWeapon := make(map[string]*acc)
	for _, k := range kills {
		if k.Side != side || k.WeaponKey == "" {
			continue
		}
		a, ok := byWeapon[k.WeaponKey]
		if !ok {
			a = &acc{rec: k}
			byWeapon[k.WeaponKey] = a
		}
		a.dist = append(a.dist, k.DistanceM)
		if beatsRecord(k, a.rec) {
			a.rec = k
		}
	}
	out := make([]WeaponDistanceRecord, 0, len(byWeapon))
	for key, a := range byWeapon {
		sort.Float64s(a.dist)
		out = append(out, WeaponDistanceRecord{
			WeaponKey:        key,
			Measured:         len(a.dist),
			MedianM:          percentileLinear(a.dist, 50),
			RecordM:          a.rec.DistanceM,
			RecordMatchID:    a.rec.MatchID,
			RecordKillerXUID: a.rec.KillerXUID,
			RecordTimeMS:     a.rec.TimeMS,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RecordM != out[j].RecordM {
			return out[i].RecordM < out[j].RecordM
		}
		return out[i].WeaponKey < out[j].WeaponKey
	})
	return out
}

// beatsRecord dit si `k` remplace `cur` comme record : plus loin, ou aussi loin et plus petit
// dans la clé du frag (match_id, time_ms).
func beatsRecord(k, cur MeasuredKill) bool {
	if k.DistanceM != cur.DistanceM {
		return k.DistanceM > cur.DistanceM
	}
	if k.MatchID != cur.MatchID {
		return k.MatchID < cur.MatchID
	}
	return k.TimeMS < cur.TimeMS
}
