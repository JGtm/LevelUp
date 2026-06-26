package halo_5

// events_dto.go — projection de /h5/matches/{id}/events (timeline NATIVE Halo 5).
//
// Shape CONFIRMÉE par la sonde live (cf. HANDOFF_HALO5_EXPERIMENTAL §0-quater).
// GameEvents est un tableau HÉTÉROGÈNE discriminé par EventName ; un seul struct
// capte tous les champs possibles (les absents → valeur zéro). Le mapper
// (events.go) lit les champs pertinents selon EventName.
//
// Identité = GAMERTAG brut (Xuid toujours null en Halo 5).

type h5EventPlayer struct {
	Gamertag string `json:"Gamertag"`
}

type h5WorldLocation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// h5GameEvent — un événement de la timeline. Champs groupés par EventName :
//   - Death : Is*, Killer/Victim, KillerWeaponStockId, Killer/VictimWorldLocation.
//   - Medal : MedalId, Player. Impulse : ImpulseId, Player.
//   - WeaponPickup/WeaponDrop : WeaponStockId, Player. PlayerSpawn : Player.
//   - RoundStart/RoundEnd : RoundIndex.
type h5GameEvent struct {
	EventName      string `json:"EventName"`
	TimeSinceStart string `json:"TimeSinceStart"` // ISO8601 PT..S
	RoundIndex     *int   `json:"RoundIndex"`

	// Death (kill)
	IsHeadshot          bool             `json:"IsHeadshot"`
	IsMelee             bool             `json:"IsMelee"`
	IsGroundPound       bool             `json:"IsGroundPound"`
	IsShoulderBash      bool             `json:"IsShoulderBash"`
	IsWeapon            bool             `json:"IsWeapon"`
	Killer              *h5EventPlayer   `json:"Killer"`
	Victim              *h5EventPlayer   `json:"Victim"`
	KillerWeaponStockId int64            `json:"KillerWeaponStockId"`
	KillerWorldLocation *h5WorldLocation `json:"KillerWorldLocation"`
	VictimWorldLocation *h5WorldLocation `json:"VictimWorldLocation"`
	// Assistants : joueurs ayant assisté le kill (natif Halo 5). Alimente le signal
	// d'engagement « support » (event_type=assist dans highlight_events).
	Assistants []h5EventPlayer `json:"Assistants"`

	// Medal / Impulse / WeaponPickup / WeaponDrop / PlayerSpawn
	Player        *h5EventPlayer `json:"Player"`
	MedalId       int64          `json:"MedalId"`
	ImpulseId     int64          `json:"ImpulseId"`
	WeaponStockId int64          `json:"WeaponStockId"`

	// WeaponDrop : tirs comptabilisés pour l'arme lâchée → précision PAR ARME
	// (Halo 5 natif). Somme par (joueur, arme) sur le match = TotalShotsFired/
	// TotalShotsLanded du carnage (validé EXACT 8/8 joueurs) — là où le carnage
	// WeaponStats[] est servi vide. 0 sur les autres EventName.
	ShotsFired  int `json:"ShotsFired"`
	ShotsLanded int `json:"ShotsLanded"`
}

// h5MatchEventsResponse — racine de /h5/matches/{id}/events.
type h5MatchEventsResponse struct {
	GameEvents []h5GameEvent `json:"GameEvents"`
}
