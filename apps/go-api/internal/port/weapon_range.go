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
// les joueurs, tous les matchs). Sans MatchIDs, ce serait un scan complet — rejeté par
// Validate(). MatchIDs présent, il faut EN PLUS soit un désignant de joueur, soit
// AllPlayers (cf. ce champ).
type WeaponRangeFilters struct {
	// MatchIDs borne la lecture aux matchs du scope (déjà filtré par période côté service).
	MatchIDs []string

	// Gamertag désigne le joueur dont on lit les frags et les morts (résolu en xuid via
	// xuid_aliases, comme tous les lecteurs de ce paquet).
	Gamertag string

	// XUIDs désigne le ou les joueurs par xuid (alternative à Gamertag).
	XUIDs []string

	// AllPlayers lit les frags de TOUS LES JOUEURS des matchs du scope, sans désignant.
	//
	// POURQUOI LA GARDE ANTI-SCAN TIENT QUAND MÊME. Ce qu'elle interdit, c'est un balayage
	// NON BORNÉ de la table partagée ; elle ne protège pas un joueur en particulier. Quand
	// `MatchIDs` borne la lecture, le coût est celui de ces matchs-là — le même que la
	// lecture d'un match entier que la Match view fait déjà (KillDistanceRepo.LoadMatch,
	// aucun filtre xuid). Sans `MatchIDs`, `Validate()` refuse comme avant.
	//
	// À QUOI ÇA SERT : la portée d'un joueur ne se lit que RELATIVEMENT à son lobby (la
	// médiane du lobby d'un BTB vaut le double de celle d'une arène). Le référentiel exige
	// donc les frags des huit ou seize joueurs, camp adverse compris.
	AllPlayers bool
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
	// AllPlayers remplace le désignant de joueur : le scan reste borné par MatchIDs,
	// qui vient d'être exigé ci-dessus.
	if !f.AllPlayers && f.Gamertag == "" && len(f.XUIDs) == 0 {
		return ErrWeaponRangeFiltersTooBroad
	}
	return nil
}

// WeaponLabel est le nom d'affichage d'une clé d'arme, dans les deux ordres de langue.
//
// Deux champs et non un, pour la même raison que `MatchKillDistanceWeapon` : la réponse sert
// les deux locales et la résolution est UNE seule requête. Vides tous les deux quand la
// metadata du titre ne connaît pas la clé — l'appelant retombe alors sur la clé elle-même,
// jamais sur un libellé inventé côté Go.
type WeaponLabel struct {
	Label   string `json:"label,omitempty"`
	LabelEN string `json:"label_en,omitempty"`
}

// WeaponLabelResolver traduit des clés de registre en noms d'affichage.
//
// POURQUOI C'EST UN CONTRAT SÉPARÉ, PORTÉ PAR LE MÊME REPO. La traduction est une lecture de
// METADATA (`weapon_name_labels` / `weapon_labels`), pas une lecture de frags : elle n'a ni
// le même scope, ni la même base, ni le même régime d'échec. Mais elle vit chez le même
// implémenteur, parce que la résolution canonique du dépôt est déjà écrite là
// (`resolveWeaponKeyLabelsAny`, partagée avec KillDistanceRepo) et qu'aucune couche au-dessus
// de `platform/` n'a le droit de toucher DuckDB.
type WeaponLabelResolver interface {
	// ResolveWeaponLabels rend le libellé de chaque clé connue. BEST-EFFORT : une clé
	// absente de la metadata est simplement absente de la map (jamais une entrée vide
	// fabriquée), et une metadata non migrée rend une map vide sans erreur.
	ResolveWeaponLabels(ctx context.Context, weaponKeys []string) (map[string]WeaponLabel, error)
}

// WeaponDimensions décrit la place d'une arme dans le REGISTRE du titre : sa classe (axe de
// manipulation — épaule, poing, lourde...), son rôle (fonction de combat — précision,
// automatique, sniper...) et sa famille.
//
// POURQUOI CE TYPE EXISTE ALORS QUE `WeaponKillRow` PORTE DÉJÀ LES TROIS CHAMPS. Les frags
// MESURÉS (`analysis.MeasuredKill`) ne passent pas par `WeaponKillRow` : ils portent une clé
// de registre et rien d'autre. Publier la portée par RÔLE (D1 du plan
// .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md) demande donc de traduire des clés en
// dimensions, sans requête neuve — la résolution canonique existe déjà chez l'implémenteur.
//
// Les trois champs sont des CLÉS, jamais des libellés : le nom affichable passe par
// `WeaponLabelResolver`, et aucun libellé FR/EN ne s'écrit côté Go.
type WeaponDimensions struct {
	Class  string
	Role   string
	Family string
}

// WeaponRangeRepository expose les deux lectures de frags mesurés, et la traduction des clés
// d'arme qu'elles rendent.
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

	// ResolveWeaponDimensions traduit un lot de clés de registre en dimensions
	// (classe / rôle / famille).
	//
	// BEST-EFFORT, MÊME RÉGIME QUE ResolveWeaponLabels : une clé absente du registre est
	// absente de la map (jamais une entrée vide fabriquée), et une metadata non migrée
	// rend une map vide SANS erreur. L'appelant décide alors quoi faire de l'inconnue —
	// pour le profil d'armes, l'écarter et la compter (D8), jamais la ranger sous un seau
	// fourre-tout qui mélangerait une épée et un fusil de précision.
	//
	// AUCUNE REQUÊTE NEUVE : la résolution canonique clé -> dimensions existe déjà chez
	// l'implémenteur, partagée avec les lecteurs de kills par arme.
	ResolveWeaponDimensions(ctx context.Context, titleSlug string, keys []string) (map[string]WeaponDimensions, error)

	// WeaponLabelResolver : les deux lectures ci-dessus rendent des CLÉS de registre
	// (`analysis.MeasuredKill.WeaponKey`), jamais des noms. Le service n'a pas d'autre
	// chemin vers la metadata du titre, et il ne doit surtout pas en inventer un : un
	// libellé FR/EN écrit en Go serait un adaptateur de titre déguisé.
	WeaponLabelResolver
}
