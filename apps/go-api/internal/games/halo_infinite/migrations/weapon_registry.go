package migrations

// weapon_registry.go — step add_weapon_registry (named-func, TargetMetadata).
//
// Registre d'armes canonique = passage PRINCIPAL de la résolution d'arme (cf.
// .ai/PLAN_WEAPON_TAXONOMY.md). 3 tables référentielles dans metadata.duckdb :
//   - weapons          : 1 ligne par arme par titre (class/role/family/faction/damage_type + extra JSON).
//   - weapon_ids       : N ids par arme (filmshell/stock_id/module…) → un id résout vers UN weapon_key.
//   - weapon_families  : référentiel des familles cross-titre (clé → libellés FR/EN).
//
// Choix de schéma (décision 2026-06-23) : PK simple + INSERT OR IGNORE, comme
// weapon_labels.go / career_ranks / mode_name_tr. C'est un référentiel STATIQUE
// seedé au boot (zéro writer concurrent, zéro UPDATE per-match) → hors périmètre
// du bug ART #23046, donc pas d'append-only `_latest`. L'extensibilité (TTK & co
// « un jour ») passe par la colonne `extra` JSON, pas par une nouvelle génération.
//
// Seed = table §6 du plan, VÉRIFIÉE halopedia.org + wiki.halo.fr. Les filmshell
// ids Infinite proviennent de weapon_labels.go (suffixe 42c9679f = vraie arme) ;
// les stock_ids H5 du catalogue officiel weapon_labels metadata H5 (peuplé par
// cmd/h5-metadata-fetch), figés ici (Halo 5 gelé). Plusieurs ids/arme = variantes/
// skins (Halo 2 BR, SPNKr, Retro Beam, Flagnum…) qui résolvent vers l'arme canonique.

import (
	"database/sql"
	"strconv"

	"levelup/go-api/internal/migration"
)

// ApplyWeaponRegistry expose applyWeaponRegistry pour un éventuel CLI de reseed.
// Idempotent (CREATE IF NOT EXISTS + INSERT OR IGNORE).
func ApplyWeaponRegistry(db *sql.DB) error {
	return applyWeaponRegistry(db)
}

type weaponFamilyRow struct{ key, en, fr string }

// weaponRow — class = manipulation (poing/épaule/lourde/mêlée/grenade) ;
// role = fonction de combat (automatic/precision/sniper/shotgun/sidearm/power/
// special/melee/grenade) ; family = identité précise cross-titre.
type weaponRow struct {
	key, title, name, nameFR, class, role, family, faction, damage, manufacturer string
}

// weaponNumericID — id numérique d'une arme (filmshell Infinite OU stock_id H5).
// uint64 littéral (bit63 possible côté filmshell) formaté en décimal string à
// l'insertion (id_value = VARCHAR).
type weaponNumericID struct {
	key string
	id  uint64
}

const (
	titleHINF = "halo_infinite"
	titleH5   = "halo_5"
)

// applyWeaponRegistry crée les 3 tables et seede familles + armes + ids Infinite.
func applyWeaponRegistry(db *sql.DB) error {
	if err := migration.ExecScript(db, `
		CREATE TABLE IF NOT EXISTS weapon_families (
			family_key VARCHAR PRIMARY KEY,
			name_en    VARCHAR NOT NULL,
			name_fr    VARCHAR NOT NULL
		);
		CREATE TABLE IF NOT EXISTS weapons (
			weapon_key   VARCHAR NOT NULL,
			title_slug   VARCHAR NOT NULL,
			name         VARCHAR NOT NULL,
			name_fr      VARCHAR,
			class        VARCHAR,
			role         VARCHAR,
			family_key   VARCHAR,
			faction      VARCHAR,
			damage_type  VARCHAR,
			manufacturer VARCHAR,
			extra        JSON,
			PRIMARY KEY (title_slug, weapon_key)
		);
		CREATE TABLE IF NOT EXISTS weapon_ids (
			title_slug VARCHAR NOT NULL,
			id_kind    VARCHAR NOT NULL,
			id_value   VARCHAR NOT NULL,
			weapon_key VARCHAR NOT NULL,
			PRIMARY KEY (title_slug, id_kind, id_value)
		);
	`); err != nil {
		return err
	}
	if err := seedWeaponFamilies(db); err != nil {
		return err
	}
	if err := seedWeapons(db); err != nil {
		return err
	}
	if err := seedWeaponFilmshellIDs(db); err != nil {
		return err
	}
	return seedWeaponStockIDs(db)
}

func seedWeaponFamilies(db *sql.DB) error {
	const q = `INSERT OR IGNORE INTO weapon_families (family_key, name_en, name_fr) VALUES (?, ?, ?)`
	for _, f := range weaponRegistryFamilies {
		if _, err := db.ExecContext(migration.BootCtx(), q, f.key, f.en, f.fr); err != nil {
			return err
		}
	}
	return nil
}

func seedWeapons(db *sql.DB) error {
	const q = `INSERT OR IGNORE INTO weapons
		(weapon_key, title_slug, name, name_fr, class, role, family_key, faction, damage_type, manufacturer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, w := range weaponRegistryWeapons {
		if _, err := db.ExecContext(migration.BootCtx(), q,
			w.key, w.title, w.name, w.nameFR, w.class, w.role, w.family, w.faction, w.damage, w.manufacturer); err != nil {
			return err
		}
	}
	return nil
}

func seedWeaponFilmshellIDs(db *sql.DB) error {
	const q = `INSERT OR IGNORE INTO weapon_ids (title_slug, id_kind, id_value, weapon_key) VALUES (?, 'filmshell', ?, ?)`
	for _, f := range weaponRegistryInfiniteFilmshell {
		idValue := strconv.FormatUint(f.id, 10)
		if _, err := db.ExecContext(migration.BootCtx(), q, titleHINF, idValue, f.key); err != nil {
			return err
		}
	}
	return nil
}

func seedWeaponStockIDs(db *sql.DB) error {
	const q = `INSERT OR IGNORE INTO weapon_ids (title_slug, id_kind, id_value, weapon_key) VALUES (?, 'stock_id', ?, ?)`
	for _, s := range weaponRegistryH5Stock {
		idValue := strconv.FormatUint(s.id, 10)
		if _, err := db.ExecContext(migration.BootCtx(), q, titleH5, idValue, s.key); err != nil {
			return err
		}
	}
	return nil
}

// weaponRegistryFamilies — référentiel des familles cross-titre (union HINF + H5, §6.3).
var weaponRegistryFamilies = []weaponFamilyRow{
	{"battle_rifle", "Battle Rifle", "Fusil de combat"},
	{"dmr", "DMR", "DMR"},
	{"stalker_rifle", "Stalker Rifle", "Fusil traqueur"},
	{"assault_rifle", "Assault Rifle", "Fusil d'assaut"},
	{"smg", "SMG", "Mitraillette"},
	{"commando", "Commando", "Commando"},
	{"sniper_rifle", "Sniper Rifle", "Fusil de précision"},
	{"shotgun", "Shotgun", "Fusil à pompe"},
	{"hydra", "Hydra", "Hydra"},
	{"rocket_launcher", "Rocket Launcher", "Lance-roquettes"},
	{"magnum", "Magnum", "Magnum"},
	{"plasma_pistol", "Plasma Pistol", "Pistolet à plasma"},
	{"needler", "Needler", "Needler"},
	{"sentinel_beam", "Sentinel Beam", "Laser de Sentinelle"},
	{"energy_sword", "Energy Sword", "Épée à énergie"},
	{"gravity_hammer", "Gravity Hammer", "Marteau antigravité"},
	{"skewer", "Skewer", "Empaleur"},
	{"cindershot", "Cindershot", "Crémator"},
	{"heatwave", "Heatwave", "Calcineur"},
	{"ravager", "Ravager", "Ravageur"},
	{"shock_rifle", "Shock Rifle", "Fusil électrique"},
	{"disruptor", "Disruptor", "Disrupteur"},
	{"mangler", "Mangler", "Déchiqueteur"},
	{"pulse_carbine", "Pulse Carbine", "Carabine à impulsion"},
	{"carbine", "Carbine", "Carabine"},
	{"frag_grenade", "Frag Grenade", "Grenade à fragmentation"},
	{"plasma_grenade", "Plasma Grenade", "Grenade à plasma"},
	{"dynamo_grenade", "Dynamo Grenade", "Grenade Dynamo"},
	{"grenade_launcher", "Grenade Launcher", "Lance-grenades"},
	{"railgun", "Railgun", "Railgun"},
	{"saw", "SAW", "SAW"},
	{"spartan_laser", "Spartan Laser", "Laser Spartan"},
	{"plasma_rifle", "Plasma Rifle", "Fusil à plasma"},
	{"fuel_rod", "Fuel Rod Cannon", "Canon à combustible"},
	{"storm_rifle", "Storm Rifle", "Fusil Storm"},
	{"beam_rifle", "Beam Rifle", "Fusil à rayon"},
	{"plasma_caster", "Plasma Caster", "Canon plasma"},
	{"light_rifle", "Light Rifle", "Fusil léger"},
	{"binary_rifle", "Binary Rifle", "Fusil binaire"},
	{"boltshot", "Boltshot", "Pistolet à particules"},
	{"incineration_cannon", "Incineration Cannon", "Canon incendiaire"},
	{"suppressor", "Suppressor", "Éradicateur"},
	{"scattershot", "Scattershot", "Répercuteur"},
}

// weaponRegistryWeapons — 59 armes (29 Infinite §6.1 + 30 Halo 5 §6.2), vérifiées
// halopedia.org + wiki.halo.fr. faction = ORIGINE de conception (pas le porteur).
var weaponRegistryWeapons = []weaponRow{
	// Colonnes : key, title, name, name_fr, class, role, family, faction, damage, manufacturer.
	// ── Halo Infinite (§6.1) ──
	{"hinf_br75", titleHINF, "BR75", "Fusil de combat", "shoulder", "precision", "battle_rifle", "human", "ballistic", "Misriah Armory"},
	{"hinf_bandit", titleHINF, "M392 Bandit", "Bandit", "shoulder", "precision", "dmr", "human", "ballistic", "Sevine Arms"},
	{"hinf_ma40_ar", titleHINF, "MA40 AR", "Fusil d'assaut", "shoulder", "automatic", "assault_rifle", "human", "ballistic", "Misriah Armory"},
	{"hinf_ma5k_avenger", titleHINF, "MA5K Avenger", "Avenger", "shoulder", "automatic", "smg", "human", "ballistic", "Misriah Armory"},
	{"hinf_vk78_commando", titleHINF, "VK78 Commando", "Commando", "shoulder", "automatic", "commando", "human", "ballistic", "Vakara GesmbH"},
	{"hinf_s7_sniper", titleHINF, "S7 Sniper", "Fusil de précision S7", "heavy", "sniper", "sniper_rifle", "human", "ballistic", "Misriah Armory"},
	{"hinf_cqs48_bulldog", titleHINF, "CQS48 Bulldog", "Bulldog", "shoulder", "shotgun", "shotgun", "human", "ballistic", "Misriah Armory"},
	{"hinf_hydra", titleHINF, "MLRS-2 Hydra", "Hydra", "heavy", "power", "hydra", "human", "explosive", "Chalybs Defense Solutions"},
	{"hinf_m41_spnkr", titleHINF, "M41 SPNKr", "Lance-roquettes", "heavy", "power", "rocket_launcher", "human", "explosive", "Misriah Armory"},
	{"hinf_fuel_rod_spnkr", titleHINF, "Fuel Rod SPNKr", "Lance-roquettes Fuel Rod", "heavy", "power", "rocket_launcher", "banished", "explosive", "Banished (SPNKr modifié)"},
	{"hinf_sidekick", titleHINF, "Mk50 Sidekick", "Sidekick", "sidearm", "sidearm", "magnum", "human", "ballistic", "Emerson Tactical Systems"},
	{"hinf_plasma_pistol", titleHINF, "Plasma Pistol", "Pistolet à plasma", "sidearm", "sidearm", "plasma_pistol", "covenant", "plasma", "Iruiru Armory"},
	{"hinf_needler", titleHINF, "Needler", "Needler", "shoulder", "special", "needler", "covenant", "spike", "Lodam Armory"},
	{"hinf_sentinel_beam", titleHINF, "Sentinel Beam", "Laser de Sentinelle", "heavy", "special", "sentinel_beam", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"hinf_energy_sword", titleHINF, "Energy Sword", "Épée à énergie", "melee", "melee", "energy_sword", "covenant", "plasma", "Merchants of Qikost"},
	{"hinf_gravity_hammer", titleHINF, "Gravity Hammer", "Marteau antigravité", "melee", "melee", "gravity_hammer", "covenant", "gravitic", "Sacred Promissory"},
	{"hinf_skewer", titleHINF, "Skewer", "Empaleur", "heavy", "power", "skewer", "banished", "spike", "Flaktura Workshop"},
	{"hinf_cindershot", titleHINF, "Cindershot", "Crémator", "heavy", "power", "cindershot", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"hinf_heatwave", titleHINF, "Heatwave", "Calcineur", "heavy", "shotgun", "heatwave", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"hinf_ravager", titleHINF, "Ravager", "Ravageur", "shoulder", "automatic", "ravager", "banished", "plasma", "Veporokk Workshop"},
	{"hinf_shock_rifle", titleHINF, "Shock Rifle", "Fusil électrique", "heavy", "sniper", "shock_rifle", "banished", "shock", "Sicatt Workshop"},
	{"hinf_disruptor", titleHINF, "Disruptor", "Disrupteur", "sidearm", "sidearm", "disruptor", "banished", "shock", "Sicatt Workshop"},
	{"hinf_mangler", titleHINF, "Mangler", "Déchiqueteur", "sidearm", "sidearm", "mangler", "banished", "spike", "Ukala Workshop"},
	{"hinf_pulse_carbine", titleHINF, "Pulse Carbine", "Carabine à impulsion", "shoulder", "automatic", "pulse_carbine", "covenant", "plasma", "Lodam Armory"},
	{"hinf_stalker_rifle", titleHINF, "Stalker Rifle", "Fusil traqueur", "shoulder", "precision", "stalker_rifle", "covenant", "plasma", "Merchants of Qikost"},
	{"hinf_vestige_carbine", titleHINF, "Vestige Carbine", "Vestige Carbine", "shoulder", "precision", "carbine", "covenant", "plasma", "Sangheili"},
	{"hinf_frag_grenade", titleHINF, "Frag Grenade", "Grenade à fragmentation", "grenade", "grenade", "frag_grenade", "human", "explosive", "Misriah Armory"},
	{"hinf_plasma_grenade", titleHINF, "Plasma Grenade", "Grenade à plasma", "grenade", "grenade", "plasma_grenade", "covenant", "plasma", ""},
	{"hinf_dynamo_grenade", titleHINF, "Dynamo Grenade", "Grenade Dynamo", "grenade", "grenade", "dynamo_grenade", "banished", "shock", ""},
	// ── Halo 5: Guardians (§6.2) ──
	{"h5_assault_rifle", titleH5, "Assault Rifle (MA5D)", "Fusil d'assaut", "shoulder", "automatic", "assault_rifle", "human", "ballistic", "Misriah Armory"},
	{"h5_battle_rifle", titleH5, "Battle Rifle (BR55HB)", "Fusil de combat", "shoulder", "precision", "battle_rifle", "human", "ballistic", "Misriah Armory"},
	{"h5_dmr", titleH5, "DMR (M395B)", "DMR", "shoulder", "precision", "dmr", "human", "ballistic", "Misriah Armory"},
	{"h5_magnum", titleH5, "Magnum (M6H2)", "Magnum", "sidearm", "sidearm", "magnum", "human", "ballistic", "Misriah Armory"},
	{"h5_smg", titleH5, "SMG (M20)", "Mitraillette", "shoulder", "automatic", "smg", "human", "ballistic", "Misriah Armory"},
	{"h5_shotgun", titleH5, "Shotgun (M45D)", "Fusil à pompe", "shoulder", "shotgun", "shotgun", "human", "ballistic", "Misriah Armory"},
	{"h5_sniper_rifle", titleH5, "Sniper Rifle (SRS99-S5)", "Fusil de précision", "heavy", "sniper", "sniper_rifle", "human", "ballistic", "Misriah Armory"},
	{"h5_rocket_launcher", titleH5, "Rocket Launcher (M41D)", "Lance-roquettes", "heavy", "power", "rocket_launcher", "human", "explosive", "Misriah Armory"},
	{"h5_grenade_launcher", titleH5, "Grenade Launcher (M319)", "Lance-grenades", "heavy", "power", "grenade_launcher", "human", "explosive", "Misriah Armory"},
	{"h5_railgun", titleH5, "Railgun (ARC-920)", "Railgun", "heavy", "power", "railgun", "human", "explosive", "Acheron Security"},
	{"h5_saw", titleH5, "SAW (M739)", "SAW", "shoulder", "automatic", "saw", "human", "ballistic", "Misriah Armory"},
	{"h5_hydra", titleH5, "Hydra (MLRS-1)", "Hydra", "heavy", "power", "hydra", "human", "explosive", "Chalybs Defense Solutions"},
	{"h5_spartan_laser", titleH5, "Spartan Laser (M6/E)", "Laser Spartan", "heavy", "power", "spartan_laser", "human", "hardlight", "Misriah Armory"},
	{"h5_carbine", titleH5, "Carbine (Mosa)", "Carabine", "shoulder", "precision", "carbine", "covenant", "spike", "Iruiru Armory"},
	{"h5_energy_sword", titleH5, "Energy Sword", "Épée à énergie", "melee", "melee", "energy_sword", "covenant", "plasma", "Merchants of Qikost"},
	{"h5_plasma_pistol", titleH5, "Plasma Pistol", "Pistolet à plasma", "sidearm", "sidearm", "plasma_pistol", "covenant", "plasma", ""},
	{"h5_plasma_rifle", titleH5, "Brute Plasma Rifle", "Fusil à plasma brute", "shoulder", "automatic", "plasma_rifle", "covenant", "plasma", "Sacred Promissory"},
	{"h5_fuel_rod", titleH5, "Fuel Rod Cannon", "Canon à combustible", "heavy", "power", "fuel_rod", "covenant", "explosive", "Forge of Raansek"},
	{"h5_storm_rifle", titleH5, "Storm Rifle", "Fusil Storm", "shoulder", "automatic", "storm_rifle", "covenant", "plasma", "Lodam Armory"},
	{"h5_beam_rifle", titleH5, "Beam Rifle", "Fusil de sniper covenant", "heavy", "sniper", "beam_rifle", "covenant", "particle_beam", "Merchants of Qikost"},
	{"h5_plasma_caster", titleH5, "Plasma Caster", "Canon plasma", "heavy", "power", "plasma_caster", "covenant", "plasma", "Merchants of Qikost"},
	{"h5_needler", titleH5, "Needler", "Needler", "shoulder", "special", "needler", "covenant", "spike", "Lodam Armory"},
	{"h5_gravity_hammer", titleH5, "Gravity Hammer", "Marteau antigravité", "melee", "melee", "gravity_hammer", "covenant", "gravitic", "Sepulo'tez Workshop"},
	{"h5_light_rifle", titleH5, "Light Rifle (Z-250)", "Fusil léger", "shoulder", "precision", "light_rifle", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"h5_binary_rifle", titleH5, "Binary Rifle (Z-750)", "Fusil binaire", "heavy", "sniper", "binary_rifle", "forerunner", "particle_beam", "Ferrarius Assembler Vats"},
	{"h5_boltshot", titleH5, "Boltshot (Z-110)", "Pistolet à particules", "sidearm", "sidearm", "boltshot", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"h5_incineration_cannon", titleH5, "Incineration Cannon (Z-390)", "Canon incendiaire", "heavy", "power", "incineration_cannon", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"h5_sentinel_beam", titleH5, "Sentinel Beam", "Laser de Sentinelle", "heavy", "special", "sentinel_beam", "forerunner", "particle_beam", "Ferrarius Assembler Vats"},
	{"h5_suppressor", titleH5, "Suppressor (Z-130)", "Éradicateur", "shoulder", "automatic", "suppressor", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
	{"h5_scattershot", titleH5, "Scattershot (Z-180)", "Répercuteur", "shoulder", "shotgun", "scattershot", "forerunner", "hardlight", "Ferrarius Assembler Vats"},
}

// weaponRegistryInfiniteFilmshell — ids filmshell Infinite (source weapon_labels.go,
// suffixe 42c9679f = vraie arme). Plusieurs ids/arme = variantes (Ranked, skins).
var weaponRegistryInfiniteFilmshell = []weaponNumericID{
	{"hinf_br75", 0x2b1824d542c9679f},
	{"hinf_bandit", 0x2fb21c8742c9679f},
	{"hinf_bandit", 0x6acdc44d42c9679f}, // "Bandit Evo"
	{"hinf_ma40_ar", 0x48c19d2d42c9679f},
	{"hinf_ma5k_avenger", 0xf5c335dfe7232c0f},
	{"hinf_vk78_commando", 0xfd98554c42c9679f},
	{"hinf_s7_sniper", 0x0a1992bc42c9679f},
	{"hinf_cqs48_bulldog", 0xb619d84a42c9679f},
	{"hinf_hydra", 0x767db96d42c9679f},
	{"hinf_m41_spnkr", 0x71ab0a2c42c9679f},
	{"hinf_fuel_rod_spnkr", 0x9d6aaed242c9679f},
	{"hinf_sidekick", 0xf408190f42c9679f},
	{"hinf_plasma_pistol", 0xc354294642c9679f},
	{"hinf_needler", 0xb533957e42c9679f},
	{"hinf_sentinel_beam", 0xa0955e9e42c9679f},
	{"hinf_energy_sword", 0x4ff3937e42c9679f},
	{"hinf_energy_sword", 0x4ff3937e8978aa7a}, // Duelist
	{"hinf_energy_sword", 0x4ff3937e1ec48c7a}, // Elite Bloodblade
	{"hinf_energy_sword", 0x0c55765f7a9376a0}, // Infected
	{"hinf_gravity_hammer", 0x841ac5e542c9679f},
	{"hinf_gravity_hammer", 0x841ac5e5a730e49f}, // Diminisher of Hope
	{"hinf_gravity_hammer", 0x841ac5e5d8d07ca1}, // Rushdown
	{"hinf_skewer", 0x0d20c46942c9679f},
	{"hinf_cindershot", 0x230447b142c9679f},
	{"hinf_heatwave", 0x2ac9c2ff42c9679f},
	{"hinf_ravager", 0xc30d87c742c9679f},
	{"hinf_shock_rifle", 0x9387a8b942c9679f},
	{"hinf_shock_rifle", 0x1a22fee642c9679f}, // Ranked
	{"hinf_disruptor", 0x84bd29ed42c9679f},
	{"hinf_mangler", 0x80977ba542c9679f},
	{"hinf_pulse_carbine", 0x30484ea642c9679f},
	{"hinf_stalker_rifle", 0xdaf193c742c9679f},
	{"hinf_vestige_carbine", 0x3e07021742c9679f},
	{"hinf_frag_grenade", 0xb6dbead842c9679f},
	{"hinf_plasma_grenade", 0xc1e1bab042c9679f},
	{"hinf_dynamo_grenade", 0x3ad55da442c9679f},
}

// weaponRegistryH5Stock — stock_ids H5 (source : catalogue officiel weapon_labels
// metadata H5, www.haloapi.com via cmd/h5-metadata-fetch, figé 2026-06-23).
// Plusieurs ids/arme = variantes/skins (Halo 2 BR, Retro Beam Rifle, SPNKr,
// Flagnum, Halo One Pistol) qui résolvent vers l'arme canonique.
var weaponRegistryH5Stock = []weaponNumericID{
	{"h5_assault_rifle", 313138863},
	{"h5_battle_rifle", 424645655},
	{"h5_battle_rifle", 4222743534}, // Halo 2 Battle Rifle
	{"h5_dmr", 523953283},
	{"h5_magnum", 4096745987},
	{"h5_magnum", 2758094302}, // Halo One Pistol
	{"h5_magnum", 2244200496}, // Flagnum
	{"h5_smg", 723388907},
	{"h5_shotgun", 3484334713},
	{"h5_sniper_rifle", 669296699},
	{"h5_rocket_launcher", 723523180},
	{"h5_rocket_launcher", 2902827823},  // SPNKr Rocket Launcher
	{"h5_grenade_launcher", 1390323522}, // Reach Grenade Launcher
	{"h5_railgun", 3682788176},
	{"h5_saw", 2278207101},
	{"h5_hydra", 1579758889}, // Hydra Launcher
	{"h5_spartan_laser", 3885603197},
	{"h5_carbine", 4108759423},
	{"h5_energy_sword", 2650887244},
	{"h5_plasma_pistol", 524558978},
	{"h5_plasma_rifle", 2015271382}, // Brute Plasma Rifle
	{"h5_fuel_rod", 2670072722},     // Fuel Rod Cannon
	{"h5_storm_rifle", 2133511419},
	{"h5_beam_rifle", 2862629816},
	{"h5_beam_rifle", 907086443}, // Retro Beam Rifle
	{"h5_plasma_caster", 4054937266},
	{"h5_needler", 2050745863},
	{"h5_gravity_hammer", 2899979324},
	{"h5_light_rifle", 2511447508}, // "LightRifle"
	{"h5_binary_rifle", 2140505068},
	{"h5_boltshot", 4153405209},
	{"h5_incineration_cannon", 4086418184},
	{"h5_sentinel_beam", 3143603656},
	{"h5_suppressor", 2681172411},
	{"h5_scattershot", 3808094875},
}
