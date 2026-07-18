package canonical

// events.go — modèle CANONIQUE de timeline d'événements de match (multi-titre).
//
// MatchEvent est la timeline BRUTE et COMPLÈTE d'un match (chaque kill, médaille,
// ramassage d'arme, spawn, round), horodatée. À distinguer de HighlightEvent
// (match.go) qui est une liste CURÉE de « temps forts » (highlights narratifs) :
// MatchEvent = source brute, HighlightEvent = sous-ensemble éditorial.
//
// Origine du modèle : Halo 5 sert nativement cette timeline (`/h5/matches/{id}/
// events`, JSON propre — tueur·victime·arme·type·position·instant). Infinite la
// reconstitue depuis `highlight_events` (+ appariement temporel killer/victim) au
// référentiel T0 ; l'arme-par-kill (RE film en cours) et les positions monde y
// sont absentes → dégradation par capability. Cf. `.ai/PLAN_CANONICAL_MATCH_EVENTS.md`.
//
// MAPPING Infinite `highlight_events.event_type` → MatchEventType (Phase 2) :
//   - kill + death (2 rows, 1 xuid chacune) → 1 MatchEventKill (appariement
//     temporel killer↔death ; cf. ComputeKillerVictimPairs, tolérance 5 ms).
//   - assist → MatchEventAssist ; medal → MatchEventMedal ; mode → MatchEventImpulse.
// L'appariement peut ÉCHOUER (double-kill simultané) ou ne pas exister (kill
// d'environnement/suicide) → Killer ou Victim nil (cf. doc de MatchEvent).
//
// Identités : *PlayerIdentity (et NON *string xuid comme HighlightEvent) — Halo 5
// est gamertag-keyé (Xuid null), Infinite xuid-keyé ; PlayerIdentity porte les deux.
// ⚠️ Côté Infinite, SEUL XUID est garanti rempli (le gamertag est best-effort via
// xuid_aliases, souvent absent pour bots/xuid orphelins). L'affichage DOIT passer
// par le chokepoint canonique (GamertagLookupViewSQL / displayPlayerName front),
// JAMAIS `gamertag || xuid` (cf. règle projet d'affichage gamertag). Un xuid non
// résolu → CapabilityGap d'identité partielle au niveau Timeline si pertinent.

// MatchEventType discrimine la nature d'un événement de timeline.
type MatchEventType string

const (
	MatchEventKill         MatchEventType = "kill"   // 1 event = tueur+victime (Halo 5 Death ; Infinite : appariement kill↔death)
	MatchEventAssist       MatchEventType = "assist" // assistance à un kill (Infinite highlight_events ; Halo 5 : absent du timeline)
	MatchEventMedal        MatchEventType = "medal"
	MatchEventImpulse      MatchEventType = "impulse" // déclencheur de mode (objectif, power-up…) — Halo 5 ImpulseId ; Infinite event_type "mode"
	MatchEventWeaponPickup MatchEventType = "weapon_pickup"
	MatchEventWeaponDrop   MatchEventType = "weapon_drop"
	MatchEventSpawn        MatchEventType = "spawn"
	MatchEventRoundStart   MatchEventType = "round_start"
	MatchEventRoundEnd     MatchEventType = "round_end"
)

// IsKnownMatchEventType indique si t est un type d'événement reconnu.
func IsKnownMatchEventType(t MatchEventType) bool {
	switch t {
	case MatchEventKill, MatchEventAssist, MatchEventMedal, MatchEventImpulse, MatchEventWeaponPickup,
		MatchEventWeaponDrop, MatchEventSpawn, MatchEventRoundStart, MatchEventRoundEnd:
		return true
	}
	return false
}

// AllMatchEventTypes retourne tous les types d'événements canoniques.
func AllMatchEventTypes() []MatchEventType {
	return []MatchEventType{
		MatchEventKill, MatchEventAssist, MatchEventMedal, MatchEventImpulse, MatchEventWeaponPickup,
		MatchEventWeaponDrop, MatchEventSpawn, MatchEventRoundStart, MatchEventRoundEnd,
	}
}

// KillKind précise la MÉCANIQUE d'un kill (orthogonale au modificateur Headshot :
// un headshot est un kill d'arme ET un headshot ⇒ Kind=weapon + Headshot=true).
// Assassination est une mécanique cross-titre (Halo 5 `IsAssassination` par kill ;
// Infinite expose aussi les assassinats) prioritaire sur weapon/melee (un assassinat
// est déclenché depuis un corps-à-corps mais reste sa propre catégorie).
type KillKind string

const (
	KillKindWeapon        KillKind = "weapon"
	KillKindMelee         KillKind = "melee"
	KillKindGroundPound   KillKind = "groundpound"
	KillKindShoulderBash  KillKind = "shoulderbash"
	KillKindAssassination KillKind = "assassination"
)

// IsKnownKillKind indique si k est une mécanique de kill reconnue.
func IsKnownKillKind(k KillKind) bool {
	switch k {
	case KillKindWeapon, KillKindMelee, KillKindGroundPound, KillKindShoulderBash, KillKindAssassination:
		return true
	}
	return false
}

// AllKillKinds retourne toutes les mécaniques de kill canoniques.
func AllKillKinds() []KillKind {
	return []KillKind{KillKindWeapon, KillKindMelee, KillKindGroundPound, KillKindShoulderBash, KillKindAssassination}
}

// Vec3 est une position dans le monde du match (unités monde du moteur, sans
// normalisation). Servie nativement par Halo 5 (KillerWorldLocation/Victim…) ;
// `not_exposed` pour Infinite (le parser film n'extrait pas de coordonnées).
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// MatchEvent est un événement horodaté de la timeline d'un match, discriminé par
// Type. Les champs non pertinents pour un Type donné sont nil (refs *T) ou la
// valeur zéro (enums string "" / bool false) — comme les autres enums canonical
// (Outcome, MatchType) qui n'utilisent pas de pointeur.
//
// Par Type :
//   - kill          : Killer, Victim, Kind, Headshot ; Weapon (degraded Infinite),
//     KillerLoc/VictimLoc (not_exposed Infinite). Killer nil = kill SANS tueur
//     attribué (environnement/chute/suicide) OU appariement Infinite échoué ;
//     Victim devrait toujours être présent pour un kill.
//   - assist        : Player (l'assistant). Halo 5 : absent du timeline (dérivable).
//   - medal         : Player, RefID (= medal id).
//   - impulse       : Player, RefID (= impulse id : objectif/power-up mode-spécifique).
//   - spawn         : Player.
//   - weapon_pickup : Player, Weapon.
//   - weapon_drop   : Player, Weapon, ShotsFired, ShotsLanded (tirs comptabilisés
//     pour l'arme lâchée — Halo 5 natif ; nil pour les autres types/titres).
//   - round_start / round_end     : Round (index).
type MatchEvent struct {
	Type MatchEventType `json:"type"`
	// TimeMs : ms depuis le début du match, AU RÉFÉRENTIEL T0 (countdown pré-match
	// retranché). Les events de countdown (TimeMs<0) sont skippés par les builders
	// (cf. internal/analysis/narrative/first_events.go). Halo 5 : converti depuis
	// TimeSinceStart (ISO8601). Infinite : timeline.CorrectEvents.
	TimeMs int `json:"time_ms"`

	Killer    *PlayerIdentity `json:"killer,omitempty"`     // kill
	Victim    *PlayerIdentity `json:"victim,omitempty"`     // kill
	Kind      KillKind        `json:"kind,omitempty"`       // kill : mécanique
	Headshot  bool            `json:"headshot,omitempty"`   // kill : modificateur orthogonal
	Weapon    *AssetReference `json:"weapon,omitempty"`     // kill / pickup / drop
	KillerLoc *Vec3           `json:"killer_loc,omitempty"` // kill (Halo 5 plein)
	VictimLoc *Vec3           `json:"victim_loc,omitempty"` // kill (Halo 5 plein)
	// Assists : assistants du kill (Halo 5 Death.Assistants[] — natif). Vide hors
	// kill / titres sans assists au timeline (Infinite : assists hors timeline).
	Assists []PlayerIdentity `json:"assists,omitempty"` // kill

	Player *PlayerIdentity `json:"player,omitempty"` // medal / impulse / spawn / pickup / drop
	// RefID : identifiant catalogue référencé, discriminé par Type (medal id pour
	// medal, impulse id pour impulse). Générique pour rester extensible.
	RefID *string `json:"ref_id,omitempty"`
	Round *int    `json:"round,omitempty"` // round_start / round_end : index de round

	// ShotsFired / ShotsLanded : tirs tirés / touchés pour l'arme lâchée, agrégés
	// par le moteur au drop/swap (weapon_drop). Halo 5 natif (WeaponDrop) ; somme
	// par (joueur, arme) = TotalShotsFired/Landed du carnage (validé exact). nil
	// hors weapon_drop / titres sans cette donnée.
	ShotsFired  *int `json:"shots_fired,omitempty"`  // weapon_drop
	ShotsLanded *int `json:"shots_landed,omitempty"` // weapon_drop
}

// MatchEventTimeline est la surface (séparée, chargée ON-DEMAND par
// LoadMatchEvents) portant la timeline d'événements d'un match. NE PAS la mettre
// dans MatchDetail (volume) — c'est un appel dédié.
type MatchEventTimeline struct {
	MatchID string       `json:"match_id"`
	Events  []MatchEvent `json:"events"`
	// Limitations : dégradations connues pour ce titre (ex. arme-par-kill degraded
	// côté Infinite, positions not_exposed). Vide = timeline complète.
	Limitations []CapabilityGap `json:"limitations,omitempty"`
}

// MatchEventOptions filtre le chargement de la timeline (réduit la charge utile :
// un kill-feed ne demande que kill+medal, pas les 269 weapon_pickup). Types vide
// = tous les types.
type MatchEventOptions struct {
	Types []MatchEventType
}

// Wants indique si le type t doit être inclus selon les options (Types vide = tout).
func (o MatchEventOptions) Wants(t MatchEventType) bool {
	if len(o.Types) == 0 {
		return true
	}
	for _, want := range o.Types {
		if want == t {
			return true
		}
	}
	return false
}
