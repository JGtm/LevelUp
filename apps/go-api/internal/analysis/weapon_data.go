package analysis

// weapon_data.go — Données statiques armes Halo Infinite (IDs filmshell, timings, fusions).
//
// Port de src/analysis/_weapon_data.py.
// Toutes les constantes sont des uint64 (big-endian) pour les 8 bytes filmshell.

import "encoding/binary"

// ══════════════════════════════════════════════════════════════════════════════
//  Sentinel IDs (réservés, ne collident pas avec les uint64 film)
// ══════════════════════════════════════════════════════════════════════════════

const (
	MeleeWeaponID   uint64 = 1
	GrenadeWeaponID uint64 = 0
	VehicleWeaponID uint64 = 2
)

// SentinelIDs regroupe les IDs non-arme pour filtrage.
var SentinelIDs = map[uint64]bool{
	MeleeWeaponID: true, GrenadeWeaponID: true, VehicleWeaponID: true,
}

// Noms canoniques d'armes Halo Infinite réutilisés comme têtes de famille
// dans WeaponFamilyMap et WeaponTimings — déclarés ici pour éviter les doublons.
const (
	WeaponNameEnergySword   = "Energy Sword"
	WeaponNameGravityHammer = "Gravity Hammer"
)

// ══════════════════════════════════════════════════════════════════════════════
//  Weapon ID Map (hex filmshell → nom anglais)
// ══════════════════════════════════════════════════════════════════════════════

// hexToUint64 convertit un hex littéral 8-bytes en uint64 big-endian.
func hexToUint64(b [8]byte) uint64 { return binary.BigEndian.Uint64(b[:]) }

// WeaponEntry associe bytes filmshell + nom.
type WeaponEntry struct {
	Bytes [8]byte
	ID    uint64
	Name  string
}

// weaponEntries — liste ordonnée, initialisée dans init().
var weaponEntries []WeaponEntry

// WeaponIDToName — map[uint64]string pour lookup rapide.
var WeaponIDToName map[uint64]string

// WeaponNameToID — reverse map[string]uint64.
var WeaponNameToID map[string]uint64

// WeaponIDs — set pour validation rapide.
var WeaponIDs map[uint64]bool

// WeaponBytesMap — [8]byte → bool pour scan Formula A.
var WeaponBytesMap map[[8]byte]bool

// WeaponBytesToInt — [8]byte → uint64.
var WeaponBytesToInt map[[8]byte]uint64

func init() {
	type raw struct {
		hex  [8]byte
		name string
	}
	entries := []raw{
		// Confirmées (liste maître 2026-03-11)
		{h("6acdc44d42c9679f"), "Bandit Evo"},
		{h("2b1824d542c9679f"), "BR75"},
		{h("230447b142c9679f"), "Cindershot"},
		{h("b619d84a42c9679f"), "CQS48 Bulldog"},
		{h("84bd29ed42c9679f"), "Disruptor"},
		{h("9d6aaed242c9679f"), "Fuel Rod SPNKr"},
		{h("841ac5e542c9679f"), WeaponNameGravityHammer},
		{h("2ac9c2ff42c9679f"), "Heatwave"},
		{h("71ab0a2c42c9679f"), "M41 SPNKr"},
		{h("2fb21c8742c9679f"), "M392 Bandit"},
		{h("48c19d2d42c9679f"), "MA40 AR"},
		{h("f5c335dfe7232c0f"), "MA5K Avenger"},
		{h("80977ba542c9679f"), "Mangler"},
		{h("767db96d42c9679f"), "MLRS-2 Hydra"},
		{h("f408190f42c9679f"), "Mk51 Sidekick"},
		{h("d791556542c9679f"), "Mutilator"},
		{h("b7262ca1c8fb11d0"), "Mythic Sandwich"},
		{h("b533957e42c9679f"), "Needler"},
		{h("c354294642c9679f"), "Plasma Pistol"},
		{h("30484ea642c9679f"), "Pulse Carbine"},
		{h("c30d87c742c9679f"), "Ravager"},
		{h("0a1992bc42c9679f"), "S7 Sniper"},
		{h("880fe0bc42c9679f"), "Sandwich"},
		{h("a0955e9e42c9679f"), "Sentinel Beam"},
		{h("9387a8b942c9679f"), "Shock Rifle"},
		{h("1a22fee642c9679f"), "Shock Rifle (Ranked)"},
		{h("0d20c46942c9679f"), "Skewer"},
		{h("daf193c742c9679f"), "Stalker Rifle"},
		{h("3e07021742c9679f"), "Vestige Carbine"},
		{h("fd98554c42c9679f"), "VK78 Commando"},
		// Energy Sword family
		{h("4ff3937e42c9679f"), WeaponNameEnergySword},
		{h("4ff3937e8978aa7a"), "Duelist Energy Sword"},
		{h("4ff3937e1ec48c7a"), "Elite Bloodblade"},
		{h("0c55765f7a9376a0"), "Infected Energy Sword"},
		// Gravity Hammer family
		{h("841ac5e5a730e49f"), "Diminisher of Hope"},
		{h("841ac5e5d8d07ca1"), "Rushdown Hammer"},
		// Grenades (non vérifiées en filmshell)
		{h("b6dbead842c9679f"), "Frag Grenade"},
		{h("c1e1bab042c9679f"), "Plasma Grenade"},
		{h("3ad55da442c9679f"), "Dynamo Grenade"},
	}

	weaponEntries = make([]WeaponEntry, len(entries))
	WeaponIDToName = make(map[uint64]string, len(entries))
	WeaponNameToID = make(map[string]uint64, len(entries))
	WeaponIDs = make(map[uint64]bool, len(entries))
	WeaponBytesMap = make(map[[8]byte]bool, len(entries))
	WeaponBytesToInt = make(map[[8]byte]uint64, len(entries))

	for i, e := range entries {
		id := hexToUint64(e.hex)
		weaponEntries[i] = WeaponEntry{Bytes: e.hex, ID: id, Name: e.name}
		WeaponIDToName[id] = e.name
		WeaponNameToID[e.name] = id
		WeaponIDs[id] = true
		WeaponBytesMap[e.hex] = true
		WeaponBytesToInt[e.hex] = id
	}
}

// h parse un hex string 16 chars en [8]byte.
func h(s string) [8]byte {
	var b [8]byte
	for i := 0; i < 8; i++ {
		b[i] = hexByte(s[i*2], s[i*2+1])
	}
	return b
}

func hexByte(hi, lo byte) byte {
	return (hexNibble(hi) << 4) | hexNibble(lo)
}

func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// ══════════════════════════════════════════════════════════════════════════════
//  Weapon Timing — (swap_ms, travel_max_ms) par arme
// ══════════════════════════════════════════════════════════════════════════════

// WeaponTiming représente les délais physiques d'une arme.
type WeaponTiming struct {
	SwapMS    int
	TravelMax int
}

// DefaultTiming est utilisé quand l'arme n'est pas dans la table.
var DefaultTiming = WeaponTiming{SwapMS: 650, TravelMax: 2000}

// WeaponTimingByName — par nom anglais.
var WeaponTimingByName = map[string]WeaponTiming{
	"Mk51 Sidekick":         {400, 300},
	"Plasma Pistol":         {450, 300},
	"MA40 AR":               {650, 500},
	"BR75":                  {650, 500},
	"Bandit Evo":            {650, 500},
	"M392 Bandit":           {650, 500},
	"CQS48 Bulldog":         {650, 500},
	"VK78 Commando":         {650, 500},
	"Vestige Carbine":       {650, 500},
	"Shock Rifle":           {650, 500},
	"Shock Rifle (Ranked)":  {650, 500},
	"Mangler":               {650, 500},
	"Pulse Carbine":         {650, 500},
	"Disruptor":             {650, 5000},
	"Heatwave":              {650, 2000},
	"Needler":               {650, 2000},
	"MLRS-2 Hydra":          {650, 2000},
	"Ravager":               {650, 5000},
	"S7 Sniper":             {900, 300},
	"Stalker Rifle":         {900, 300},
	"Mutilator":             {900, 300},
	"Skewer":                {900, 3000},
	"Sentinel Beam":         {900, 500},
	"Cindershot":            {900, 5000},
	"M41 SPNKr":             {1100, 2000},
	"Fuel Rod SPNKr":        {1100, 2000},
	WeaponNameGravityHammer: {1100, 1400},
	"Diminisher of Hope":    {1100, 1400},
	"Rushdown Hammer":       {1100, 1400},
	WeaponNameEnergySword:   {1100, 1400},
	"Duelist Energy Sword":  {1100, 1400},
	"Elite Bloodblade":      {1100, 1400},
	"Infected Energy Sword": {1100, 1400},
	"Frag Grenade":          {950, 1650},
	"Plasma Grenade":        {950, 1350},
	"Dynamo Grenade":        {950, 1500},
}

// WeaponTimingByID — par weapon_id uint64. Construit dans init().
var WeaponTimingByID map[uint64]WeaponTiming

func init() {
	WeaponTimingByID = make(map[uint64]WeaponTiming, len(WeaponTimingByName))
	for name, t := range WeaponTimingByName {
		if id, ok := WeaponNameToID[name]; ok {
			WeaponTimingByID[id] = t
		}
	}
}

// GetTiming retourne le timing pour un weapon_id donné.
func GetTiming(weaponID uint64) WeaponTiming {
	if t, ok := WeaponTimingByID[weaponID]; ok {
		return t
	}
	return DefaultTiming
}

// ══════════════════════════════════════════════════════════════════════════════
//  Fusion — variantes → canonique
// ══════════════════════════════════════════════════════════════════════════════

// WeaponFusionMap — nom variante → nom canonique.
var WeaponFusionMap = map[string]string{
	"M392 Bandit":           "Bandit Evo",
	"Fuel Rod SPNKr":        "M41 SPNKr",
	"Shock Rifle (Ranked)":  "Shock Rifle",
	"Vestige Carbine":       "Pulse Carbine",
	"Duelist Energy Sword":  WeaponNameEnergySword,
	"Elite Bloodblade":      WeaponNameEnergySword,
	"Infected Energy Sword": WeaponNameEnergySword,
	"Diminisher of Hope":    WeaponNameGravityHammer,
	"Rushdown Hammer":       WeaponNameGravityHammer,
}

// WeaponFusionMapID — weapon_id variante → weapon_id canonique.
var WeaponFusionMapID map[uint64]uint64

func init() {
	WeaponFusionMapID = make(map[uint64]uint64, len(WeaponFusionMap))
	for variant, canon := range WeaponFusionMap {
		vID, vOK := WeaponNameToID[variant]
		cID, cOK := WeaponNameToID[canon]
		if vOK && cOK {
			WeaponFusionMapID[vID] = cID
		}
	}
}

// ══════════════════════════════════════════════════════════════════════════════
//  Médailles melee / grenade
// ══════════════════════════════════════════════════════════════════════════════

var MeleeMedals = map[string]bool{
	"Pummel": true, "Assassination": true, "Back Smack": true,
	"Melee": true, "Quigley": true, "Ninja": true, "Pancake": true,
}

var GrenadeMedals = map[string]bool{
	"Sticky Fingers": true, "Grenadier": true, "Boom!": true,
	"Kong": true, "Stick": true, "Grenade Stick": true,
}

// ══════════════════════════════════════════════════════════════════════════════
//  Suffixes Formula A
// ══════════════════════════════════════════════════════════════════════════════

// CommonWeaponSuffix = derniers 4 bytes de la majorité des armes.
var CommonWeaponSuffix = [4]byte{0x42, 0xc9, 0x67, 0x9f}

// AllFormulaASuffixes — tous les suffixes 4B uniques des weapon IDs.
var AllFormulaASuffixes map[[4]byte]bool

func init() {
	AllFormulaASuffixes = make(map[[4]byte]bool)
	for b := range WeaponBytesMap {
		var suffix [4]byte
		copy(suffix[:], b[4:])
		AllFormulaASuffixes[suffix] = true
	}
}
