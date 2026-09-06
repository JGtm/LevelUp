package port

// weapon_range.go — LE PORT DE LA PORTÉE MESURÉE PAR ARME (plan
// .ai/PLAN_DUELS_PORTEE_2026-09-06.md, lot 3).
//
// # CE QUE LE REPO REND, ET CE QU'IL NE FAIT PAS
//
// Il rend des FRAGS MESURÉS — un par mort, avec son arme, sa distance et son dénivelé brut —
// des DEUX CÔTÉS : ceux que le joueur a infligés (« où je frague ») et ceux qu'il a subis
// (« où je meurs, et à quelle arme »). Il n'agrège rien : les percentiles, le seuil de
// publication et la ventilation du dénivelé vivent dans `analysis.WeaponRangeAggregate`, qui
// est pur et testable seul. C'est la frontière habituelle du dépôt.
//
// # LA PÉRIODE EST DÉJÀ RÉSOLUE — AUCUN FILTRE TEMPOREL ICI (D10)
//
// Le scope arrive par `MatchIDs`, que le service a déjà filtré par période (même chemin que
// `loadWeaponAccuracy`). Ajouter un filtre de dates dans ce repo créerait une SECONDE
// définition de la période, et deux définitions divergent : celle de la Synthèse fait foi.

import (
	"context"
	"errors"

	"levelup/go-api/internal/analysis"
)

// WeaponRangeFilters paramètre la lecture des frags mesurés.
//
// Garde-fou identique à WeaponAccuracyFilters : la requête balaie une table PARTAGÉE (tous
// les joueurs, tous les matchs). Sans MatchIDs ni (Gamertag/XUIDs), ce serait un scan
// complet — rejeté par Validate().
type WeaponRangeFilters struct {
	// MatchIDs borne la lecture aux matchs du scope (déjà filtré par période côté service).
	MatchIDs []string

	// Gamertag désigne le joueur dont on lit les frags et les morts (résolu en xuid via
	// xuid_aliases, comme tous les lecteurs de ce paquet).
	Gamertag string

	// XUIDs désigne le ou les joueurs par xuid (alternative à Gamertag).
	XUIDs []string
}

// ErrWeaponRangeFiltersTooBroad est retournée par Validate() quand les filtres laisseraient
// passer un scan complet.
var ErrWeaponRangeFiltersTooBroad = errors.New(
	"port: WeaponRangeFilters too broad (provide MatchIDs and (Gamertag or XUIDs))")

// Validate vérifie que les filtres bornent la lecture (mêmes règles que
// WeaponAccuracyFilters) : MatchIDs requis, et au moins un désignant de joueur.
func (f WeaponRangeFilters) Validate() error {
	if len(f.MatchIDs) == 0 {
		return ErrWeaponRangeFiltersTooBroad
	}
	if f.Gamertag == "" && len(f.XUIDs) == 0 {
		return ErrWeaponRangeFiltersTooBroad
	}
	return nil
}

// WeaponRangeRepository expose les deux lectures de frags mesurés.
//
// Capability gating : les deux méthodes retournent games.ErrCapabilityNotSupported quand
// leur table de positions est ABSENTE (titre sans décodeur de film, base non migrée) —
// dégradation gracieuse côté service, même contrat que WeaponAccuracyRepository. Une table
// PRÉSENTE MAIS VIDE rend zéro ligne sans erreur : c'est l'état nominal d'un scope dont
// aucun match n'a encore été décodé.
type WeaponRangeRepository interface {
	// LoadWeaponRange rend les frags mesurés À L'INSTANT DU COUP FATAL, des deux côtés
	// (Side porté par chaque MeasuredKill). L'appelant doit avoir validé les filtres ; le
	// repo re-valide en défense.
	LoadWeaponRange(ctx context.Context, slug string, filters WeaponRangeFilters) ([]analysis.MeasuredKill, error)

	// LoadWeaponOpening rend les MÊMES frags mesurés UN TEMPS-POUR-TUER PLUS TÔT (proxy
	// d'entame, `replay.OpeningLeadMS`, validé le 2026-09-06 : écart médian 1,24 m contre
	// un gate de 2 m). Chaque ligne porte la clé du frag (MatchID, KillerXUID, TimeMS) :
	// c'est par elle que l'appelant apparie l'entame à son coup fatal, jamais par un écart
	// entre deux médianes.
	//
	// La couverture est PARTIELLE PAR CONSTRUCTION tant que le backfill n'a pas tourné
	// (décision utilisateur, jamais lancé d'office) : la section publie « N frags
	// mesurés », jamais un zéro.
	LoadWeaponOpening(ctx context.Context, slug string, filters WeaponRangeFilters) ([]analysis.MeasuredKill, error)
}
