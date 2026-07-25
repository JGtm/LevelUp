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
	"fmt"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/migration"
)

// ApplyWeaponRegistry expose applyWeaponRegistry pour le reseed. Idempotent
// (CREATE IF NOT EXISTS + INSERT OR IGNORE) → sert de CHEMIN DE RÉCONCILIATION au
// boot (cmd/server/main.go, metadata du titre par défaut ET des titres additionnels
// dont H5) : converge les stock_ids/filmshell ajoutés au seed APRÈS que la migration
// one-shot fut marquée "done" (sans ça, inertes sur une DB déjà migrée — cas
// h5_other_ugc). Hors surface ART (tables PK-only, aucun index secondaire, aucun
// UPDATE/DELETE per-row).
func ApplyWeaponRegistry(db *sql.DB) error {
	return applyWeaponRegistry(db)
}

// ReconcileWeaponRegistry ré-applique le seed du registre d'armes canonique
// (weapon_families/weapons/weapon_ids) de façon idempotente à CHAQUE boot, sur la
// metadata.duckdb du titre `titleSlug`. Cross-titre par conception (le seed contient
// les lignes de tous les titres, chaque metadata en reçoit une copie complète ;
// resolveWeaponMeta filtre ensuite par wi.title_slug).
//
// Motif : le runner de migrations saute tout step déjà « done » (registry.go). Les
// lignes ajoutées au seed APRÈS la 1re application de add_weapon_registry (buckets
// non-combat H5 : h5_unattributed « Spartan » 3168248199, h5_frag_grenade,
// véhicules/tourelles…) ne sont donc JAMAIS insérées sur une DB prod déjà migrée →
// resolveWeaponMeta ne matche pas → class/role vides → filtre non-combat inerte + nom
// FR frag grenade manquant. Rejouer le seed à chaque boot referme ce trou et
// auto-guérit tout ajout futur (aucun nouveau step à versionner).
//
// Sûreté : le seed est INSERT OR IGNORE + CREATE TABLE IF NOT EXISTS (référentiel
// STATIQUE, PK simple, zéro writer concurrent, hors périmètre du bug ART #23046 —
// décision 2026-06-23). Le rejouer n'insère QUE les lignes manquantes : aucune
// écriture destructive, aucun UPDATE. Retourne le nombre de lignes nouvellement
// insérées (delta pré/post seed) pour la télémétrie de boot.
func ReconcileWeaponRegistry(db *sql.DB, titleSlug string) (int, error) {
	if db == nil {
		return 0, nil
	}
	before := countWeaponRegistryRows(db) // tolérant : 0 si tables absentes (DB vierge)
	if err := applyWeaponRegistry(db); err != nil {
		return 0, fmt.Errorf("reconcile weapon_registry: %w", err)
	}
	reconciled := countWeaponRegistryRows(db) - before
	if reconciled < 0 {
		reconciled = 0
	}
	slog.InfoContext(migration.BootCtx(), "weapon_registry reconciled",
		"title", titleSlug, "rows_inserted", reconciled)
	return reconciled, nil
}

// countWeaponRegistryRows somme les lignes des 3 tables du registre. Tolérant : une
// table absente (DB vierge, pré-seed) compte pour 0 — le delta post-seed reste exact
// (full seed sur DB vierge, 0 sur DB déjà complète, N sur DB partielle prod).
func countWeaponRegistryRows(db *sql.DB) int {
	total := 0
	for _, tbl := range []string{"weapon_families", "weapons", "weapon_ids"} {
		var n int
		if err := db.QueryRowContext(migration.BootCtx(), "SELECT COUNT(*) FROM "+tbl).Scan(&n); err == nil {
			total += n
		}
	}
	return total
}

type weaponFamilyRow struct{ key, en, fr string }

// weaponRow — class = manipulation (poing/épaule/lourde/mêlée/grenade) ;
// role = fonction de combat (automatic/precision/sniper/shotgun/sidearm/power/
// special/melee/grenade) ; family = identité précise cross-titre.
//
// V72-06 : plus de name_fr ici. Le registre ne porte QUE les dimensions ; le nom
// d'affichage (en/fr) est la source unique keyée par weapon_key dans
// config/titles/{slug}/mappings/weapon_names.toml (→ table weapon_name_labels).
// `name` reste l'identité EN canonique de l'arme (dimension), pas un libellé affiché.
type weaponRow struct {
	key, title, name, class, role, family, faction, damage, manufacturer string
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

// Valeurs de dimension réutilisées ≥4 fois dans le registre (goconst). class =
// manipulation, role = fonction, faction = origine de conception, damage = type de
// dégât. clsSidearm/clsGrenade/clsMelee servent à la fois de class ET de role.
const (
	clsShoulder = "shoulder"
	clsHeavy    = "heavy"
	clsSidearm  = "sidearm"
	clsGrenade  = "grenade"
	clsMelee    = "melee"

	roleAuto      = "automatic"
	rolePrecision = "precision"
	roleSniper    = "sniper"
	rolePower     = "power"
	roleSpecial   = "special"

	facHuman      = "human"
	facCovenant   = "covenant"
	facBanished   = "banished"
	facForerunner = "forerunner"

	dmgBallistic = "ballistic"
	dmgExplosive = "explosive"
	dmgPlasma    = "plasma"
	dmgHardlight = "hardlight"
	dmgSpike     = "spike"
)

// Fabricants d'armes réutilisés ≥4 fois (goconst).
const (
	mfrMisriah   = "Misriah Armory"
	mfrLodam     = "Lodam Armory"
	mfrFerrarius = "Ferrarius Assembler Vats"
	mfrQikost    = "Merchants of Qikost"
)

// Clés d'armes partagées entre le registre de dimensions (weaponRegistryWeapons) et
// les tables d'ids numériques (filmshell/stock_id) — la constante évite la dérive de
// frappe entre les deux tables. nameM41SPNKr : nom canonique partagé avec weapon_labels.go.
const (
	keyHinfBandit        = "hinf_bandit"
	keyHinfGravityHammer = "hinf_gravity_hammer"
	keyHinfShockRifle    = "hinf_shock_rifle"
	keyHinfEnergySword   = "hinf_energy_sword"
	keyH5Magnum          = "h5_magnum"
	keyH5OtherUGC        = "h5_other_ugc"
	nameM41SPNKr         = "M41 SPNKr"
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
	// V72-06 : la colonne name_fr « inventée » du registre a été retirée du schéma
	// (CREATE ci-dessus) et de tout le code (resolver, weaponregistry) — le nom
	// d'affichage vient désormais de weapon_name_labels, keyé par weapon_key. Sur une
	// metadata.duckdb prod DÉJÀ migrée, la colonne subsiste (inerte, jamais lue) : on ne
	// la DROP PAS ici car DuckDB refuse ALTER DROP COLUMN tant qu'un index (ici la PK ART
	// title_slug+weapon_key) existe → risque d'abort au boot. Purge = migration dédiée
	// (rebuild table-rename) hors périmètre V72-06.
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
		(weapon_key, title_slug, name, class, role, family_key, faction, damage_type, manufacturer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, w := range weaponRegistryWeapons {
		if _, err := db.ExecContext(migration.BootCtx(), q,
			w.key, w.title, w.name, w.class, w.role, w.family, w.faction, w.damage, w.manufacturer); err != nil {
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
	{"splinter_grenade", "Splinter Grenade", "Grenade Splinter"},
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
	// Long-tail H5 (frags v_weapon_kills réels) : armes de mêlée d'objectif / REQ.
	{"golf_club", "Golf Club", "Club de golf"},
	{"oddball", "Oddball", "Oddball"},
	// Hors-arsenal H5 (frags non-combat classés 2026-07-17) : familles neutres par
	// catégorie (véhicule/tourelle/environnement/non-attribué/autres). Réceptacle
	// pour le donut « Frags par type d'arme » ; exclues de l'insight coach côté web.
	{"vehicle", "Vehicle", "Véhicule"},
	{"turret", "Turret", "Tourelle"},
	{"environmental", "Environmental", "Environnement"},
	{"unattributed", "Unattributed", "Non attribué"},
	{"other", "Other", "Autres"},
}

// weaponRegistryWeapons — 84 entrées : 29 Infinite (§6.1) + 55 Halo 5 (§6.2 :
// 35 arsenal + 5 long-tail grenades/mêlée + 20 hors-arsenal non-combat classés
// 2026-07-17), vérifiées halopedia.org + wiki.halo.fr. faction = ORIGINE de
// conception (pas le porteur ; vide pour les buckets non-combat).
var weaponRegistryWeapons = []weaponRow{
	// Colonnes : key, title, name, class, role, family, faction, damage, manufacturer.
	// (name_fr retiré V72-06 — noms d'affichage dans weapon_names.toml, keyés weapon_key.)
	// ── Halo Infinite (§6.1) ──
	{"hinf_br75", titleHINF, "BR75", clsShoulder, rolePrecision, "battle_rifle", facHuman, dmgBallistic, mfrMisriah},
	{keyHinfBandit, titleHINF, "M392 Bandit", clsShoulder, rolePrecision, "dmr", facHuman, dmgBallistic, "Sevine Arms"},
	{"hinf_ma40_ar", titleHINF, "MA40 AR", clsShoulder, roleAuto, "assault_rifle", facHuman, dmgBallistic, mfrMisriah},
	{"hinf_ma5k_avenger", titleHINF, "MA5K Avenger", clsShoulder, roleAuto, "smg", facHuman, dmgBallistic, mfrMisriah},
	{"hinf_vk78_commando", titleHINF, "VK78 Commando", clsShoulder, roleAuto, "commando", facHuman, dmgBallistic, "Vakara GesmbH"},
	{"hinf_s7_sniper", titleHINF, "S7 Sniper", clsHeavy, roleSniper, "sniper_rifle", facHuman, dmgBallistic, mfrMisriah},
	{"hinf_cqs48_bulldog", titleHINF, "CQS48 Bulldog", clsShoulder, "shotgun", "shotgun", facHuman, dmgBallistic, mfrMisriah},
	{"hinf_hydra", titleHINF, "MLRS-2 Hydra", clsHeavy, rolePower, "hydra", facHuman, dmgExplosive, "Chalybs Defense Solutions"},
	{"hinf_m41_spnkr", titleHINF, nameM41SPNKr, clsHeavy, rolePower, "rocket_launcher", facHuman, dmgExplosive, mfrMisriah},
	{"hinf_fuel_rod_spnkr", titleHINF, "Fuel Rod SPNKr", clsHeavy, rolePower, "rocket_launcher", facBanished, dmgExplosive, "Banished (SPNKr modifié)"},
	{"hinf_sidekick", titleHINF, "Mk50 Sidekick", clsSidearm, clsSidearm, "magnum", facHuman, dmgBallistic, "Emerson Tactical Systems"},
	{"hinf_plasma_pistol", titleHINF, "Plasma Pistol", clsSidearm, clsSidearm, "plasma_pistol", facCovenant, dmgPlasma, "Iruiru Armory"},
	{"hinf_needler", titleHINF, "Needler", clsShoulder, roleSpecial, "needler", facCovenant, dmgSpike, mfrLodam},
	{"hinf_sentinel_beam", titleHINF, "Sentinel Beam", clsHeavy, roleSpecial, "sentinel_beam", facForerunner, dmgHardlight, mfrFerrarius},
	{keyHinfEnergySword, titleHINF, "Energy Sword", clsMelee, clsMelee, "energy_sword", facCovenant, dmgPlasma, mfrQikost},
	{keyHinfGravityHammer, titleHINF, "Gravity Hammer", clsMelee, clsMelee, "gravity_hammer", facCovenant, "gravitic", "Sacred Promissory"},
	{"hinf_skewer", titleHINF, "Skewer", clsHeavy, rolePower, "skewer", facBanished, dmgSpike, "Flaktura Workshop"},
	{"hinf_cindershot", titleHINF, "Cindershot", clsHeavy, rolePower, "cindershot", facForerunner, dmgHardlight, mfrFerrarius},
	{"hinf_heatwave", titleHINF, "Heatwave", clsHeavy, "shotgun", "heatwave", facForerunner, dmgHardlight, mfrFerrarius},
	{"hinf_ravager", titleHINF, "Ravager", clsShoulder, roleAuto, "ravager", facBanished, dmgPlasma, "Veporokk Workshop"},
	{keyHinfShockRifle, titleHINF, "Shock Rifle", clsHeavy, roleSniper, "shock_rifle", facBanished, "shock", "Sicatt Workshop"},
	{"hinf_disruptor", titleHINF, "Disruptor", clsSidearm, clsSidearm, "disruptor", facBanished, "shock", "Sicatt Workshop"},
	{"hinf_mangler", titleHINF, "Mangler", clsSidearm, clsSidearm, "mangler", facBanished, dmgSpike, "Ukala Workshop"},
	{"hinf_pulse_carbine", titleHINF, "Pulse Carbine", clsShoulder, roleAuto, "pulse_carbine", facCovenant, dmgPlasma, mfrLodam},
	{"hinf_stalker_rifle", titleHINF, "Stalker Rifle", clsShoulder, rolePrecision, "stalker_rifle", facCovenant, dmgPlasma, mfrQikost},
	{"hinf_vestige_carbine", titleHINF, "Vestige Carbine", clsShoulder, rolePrecision, "carbine", facCovenant, dmgPlasma, "Sangheili"},
	{"hinf_frag_grenade", titleHINF, "Frag Grenade", clsGrenade, clsGrenade, "frag_grenade", facHuman, dmgExplosive, mfrMisriah},
	{"hinf_plasma_grenade", titleHINF, "Plasma Grenade", clsGrenade, clsGrenade, "plasma_grenade", facCovenant, dmgPlasma, ""},
	{"hinf_dynamo_grenade", titleHINF, "Dynamo Grenade", clsGrenade, clsGrenade, "dynamo_grenade", facBanished, "shock", ""},
	// ── Halo 5: Guardians (§6.2) ──
	{"h5_assault_rifle", titleH5, "Assault Rifle (MA5D)", clsShoulder, roleAuto, "assault_rifle", facHuman, dmgBallistic, mfrMisriah},
	{"h5_battle_rifle", titleH5, "Battle Rifle (BR55HB)", clsShoulder, rolePrecision, "battle_rifle", facHuman, dmgBallistic, mfrMisriah},
	{"h5_dmr", titleH5, "DMR (M395B)", clsShoulder, rolePrecision, "dmr", facHuman, dmgBallistic, mfrMisriah},
	{keyH5Magnum, titleH5, "Magnum (M6H2)", clsSidearm, clsSidearm, "magnum", facHuman, dmgBallistic, mfrMisriah},
	{"h5_smg", titleH5, "SMG (M20)", clsShoulder, roleAuto, "smg", facHuman, dmgBallistic, mfrMisriah},
	{"h5_shotgun", titleH5, "Shotgun (M45D)", clsShoulder, "shotgun", "shotgun", facHuman, dmgBallistic, mfrMisriah},
	{"h5_sniper_rifle", titleH5, "Sniper Rifle (SRS99-S5)", clsHeavy, roleSniper, "sniper_rifle", facHuman, dmgBallistic, mfrMisriah},
	{"h5_rocket_launcher", titleH5, "Rocket Launcher (M41D)", clsHeavy, rolePower, "rocket_launcher", facHuman, dmgExplosive, mfrMisriah},
	{"h5_grenade_launcher", titleH5, "Grenade Launcher (M319)", clsHeavy, rolePower, "grenade_launcher", facHuman, dmgExplosive, mfrMisriah},
	{"h5_railgun", titleH5, "Railgun (ARC-920)", clsHeavy, rolePower, "railgun", facHuman, dmgExplosive, "Acheron Security"},
	{"h5_saw", titleH5, "SAW (M739)", clsShoulder, roleAuto, "saw", facHuman, dmgBallistic, mfrMisriah},
	{"h5_hydra", titleH5, "Hydra (MLRS-1)", clsHeavy, rolePower, "hydra", facHuman, dmgExplosive, "Chalybs Defense Solutions"},
	{"h5_spartan_laser", titleH5, "Spartan Laser (M6/E)", clsHeavy, rolePower, "spartan_laser", facHuman, dmgHardlight, mfrMisriah},
	{"h5_carbine", titleH5, "Carbine (Mosa)", clsShoulder, rolePrecision, "carbine", facCovenant, dmgSpike, "Iruiru Armory"},
	{"h5_energy_sword", titleH5, "Energy Sword", clsMelee, clsMelee, "energy_sword", facCovenant, dmgPlasma, mfrQikost},
	{"h5_plasma_pistol", titleH5, "Plasma Pistol", clsSidearm, clsSidearm, "plasma_pistol", facCovenant, dmgPlasma, ""},
	{"h5_plasma_rifle", titleH5, "Brute Plasma Rifle", clsShoulder, roleAuto, "plasma_rifle", facCovenant, dmgPlasma, "Sacred Promissory"},
	{"h5_fuel_rod", titleH5, "Fuel Rod Cannon", clsHeavy, rolePower, "fuel_rod", facCovenant, dmgExplosive, "Forge of Raansek"},
	{"h5_storm_rifle", titleH5, "Storm Rifle", clsShoulder, roleAuto, "storm_rifle", facCovenant, dmgPlasma, mfrLodam},
	{"h5_beam_rifle", titleH5, "Beam Rifle", clsHeavy, roleSniper, "beam_rifle", facCovenant, "particle_beam", mfrQikost},
	{"h5_plasma_caster", titleH5, "Plasma Caster", clsHeavy, rolePower, "plasma_caster", facCovenant, dmgPlasma, mfrQikost},
	{"h5_needler", titleH5, "Needler", clsShoulder, roleSpecial, "needler", facCovenant, dmgSpike, mfrLodam},
	{"h5_gravity_hammer", titleH5, "Gravity Hammer", clsMelee, clsMelee, "gravity_hammer", facCovenant, "gravitic", "Sepulo'tez Workshop"},
	{"h5_light_rifle", titleH5, "Light Rifle (Z-250)", clsShoulder, rolePrecision, "light_rifle", facForerunner, dmgHardlight, mfrFerrarius},
	{"h5_binary_rifle", titleH5, "Binary Rifle (Z-750)", clsHeavy, roleSniper, "binary_rifle", facForerunner, "particle_beam", mfrFerrarius},
	{"h5_boltshot", titleH5, "Boltshot (Z-110)", clsSidearm, clsSidearm, "boltshot", facForerunner, dmgHardlight, mfrFerrarius},
	{"h5_incineration_cannon", titleH5, "Incineration Cannon (Z-390)", clsHeavy, rolePower, "incineration_cannon", facForerunner, dmgHardlight, mfrFerrarius},
	{"h5_sentinel_beam", titleH5, "Sentinel Beam", clsHeavy, roleSpecial, "sentinel_beam", facForerunner, "particle_beam", mfrFerrarius},
	{"h5_suppressor", titleH5, "Suppressor (Z-130)", clsShoulder, roleAuto, "suppressor", facForerunner, dmgHardlight, mfrFerrarius},
	{"h5_scattershot", titleH5, "Scattershot (Z-180)", clsShoulder, "shotgun", "scattershot", facForerunner, dmgHardlight, mfrFerrarius},
	// ── Halo 5 long-tail (frags v_weapon_kills réels, mappage 2026-07-17) ──
	// Grenades : rôle `grenade` (parité HINF), class `grenade`. NON double-comptées
	// avec les sentinels match_participants (grenade_kills) car l'agrégation des
	// classes gun de la FragDistribution ignore IsGrenadeMelee (fragdist.Build) — les
	// grenades sont servies par le compteur API, jamais par le registre.
	{"h5_frag_grenade", titleH5, "Frag Grenade", clsGrenade, clsGrenade, "frag_grenade", facHuman, dmgExplosive, mfrMisriah},
	{"h5_plasma_grenade", titleH5, "Plasma Grenade", clsGrenade, clsGrenade, "plasma_grenade", facCovenant, dmgPlasma, ""},
	{"h5_splinter_grenade", titleH5, "Splinter Grenade", clsGrenade, clsGrenade, "splinter_grenade", facForerunner, dmgHardlight, mfrFerrarius},
	// Mêlée d'objectif / REQ : rôle `melee`, class `melee`.
	{"h5_golf_club", titleH5, "Golf Club", clsMelee, clsMelee, "golf_club", facHuman, "kinetic", ""},
	{"h5_oddball", titleH5, "Oddball", clsMelee, clsMelee, "oddball", facHuman, "kinetic", ""},
	// ── Halo 5 hors-arsenal (frags NON-COMBAT, décision produit 2026-07-17) ──
	// Rôles dédiés `vehicle`/`turret`/`environmental`/`unattributed`/`other` : ils
	// alimentent le donut « Frags par type d'arme » (JOIN weapons.role) mais sont
	// EXCLUS de l'insight coach (NON_COMBAT_WEAPON_ROLES, web) — sinon « Spartan »
	// (~8.8k) fausserait blind_spot_power. faction/damage vides (non pertinents pour
	// un bucket non-combat). class == role. name = proper noun / libellé neutre.
	{"h5_vehicle_ghost", titleH5, "Ghost", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_mongoose", titleH5, "Mongoose", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_warthog", titleH5, "Warthog", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_banshee", titleH5, "Banshee", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_mantis", titleH5, "Mantis", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_scorpion", titleH5, "Scorpion", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_wraith", titleH5, "Wraith", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_wasp", titleH5, "Wasp", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_vehicle_phaeton", titleH5, "Phaeton", "vehicle", "vehicle", "vehicle", "", "", ""},
	{"h5_turret_chaingun", titleH5, "Chaingun Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_splinter", titleH5, "Splinter Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_rocket_pod", titleH5, "Rocket Pod Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_gauss", titleH5, "Gauss Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_shade_plasma", titleH5, "Shade Plasma Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_scorpion_ai", titleH5, "Scorpion Anti-Infantry Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_plasma", titleH5, "Plasma Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_turret_hunter_arm", titleH5, "Hunter Arm Turret", "turret", "turret", "turret", "", "", ""},
	{"h5_environmental", titleH5, "Environmental Explosives", "environmental", "environmental", "environmental", "", "", ""},
	{"h5_unattributed", titleH5, "Unattributed", "unattributed", "unattributed", "unattributed", "", "", ""},
	{keyH5OtherUGC, titleH5, "Other", "other", "other", "other", "", "", ""},
}

// weaponRegistryInfiniteFilmshell — ids filmshell Infinite (source weapon_labels.go,
// suffixe 42c9679f = vraie arme). Plusieurs ids/arme = variantes (Ranked, skins).
var weaponRegistryInfiniteFilmshell = []weaponNumericID{
	{"hinf_br75", 0x2b1824d542c9679f},
	{keyHinfBandit, 0x2fb21c8742c9679f},
	{keyHinfBandit, 0x6acdc44d42c9679f}, // "Bandit Evo"
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
	{keyHinfEnergySword, 0x4ff3937e42c9679f},
	{keyHinfEnergySword, 0x4ff3937e8978aa7a}, // Duelist
	{keyHinfEnergySword, 0x4ff3937e1ec48c7a}, // Elite Bloodblade
	{keyHinfEnergySword, 0x0c55765f7a9376a0}, // Infected
	{keyHinfGravityHammer, 0x841ac5e542c9679f},
	{keyHinfGravityHammer, 0x841ac5e5a730e49f}, // Diminisher of Hope
	{keyHinfGravityHammer, 0x841ac5e5d8d07ca1}, // Rushdown
	{"hinf_skewer", 0x0d20c46942c9679f},
	{"hinf_cindershot", 0x230447b142c9679f},
	{"hinf_heatwave", 0x2ac9c2ff42c9679f},
	{"hinf_ravager", 0xc30d87c742c9679f},
	{keyHinfShockRifle, 0x9387a8b942c9679f},
	{keyHinfShockRifle, 0x1a22fee642c9679f}, // Ranked
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
	{keyH5Magnum, 4096745987},
	{keyH5Magnum, 2758094302}, // Halo One Pistol
	{keyH5Magnum, 2244200496}, // Flagnum
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
	// ── Long-tail v_weapon_kills (audit 2026-07-17, stock_ids réels) ──
	// Grenades (armes tenues, rôle `grenade`) — gros volume du long-tail.
	{"h5_frag_grenade", 4106030681},     // "FRAG GRENADE"    (~10.5k frags)
	{"h5_plasma_grenade", 2460880172},   // "PLASMA GRENADE"  (~5.3k frags)
	{"h5_splinter_grenade", 3190813201}, // "SPLINTER GRENADE" (~2.1k frags)
	// Mêlée d'objectif / REQ (armes tenues, rôle `melee`).
	{"h5_golf_club", 409331533}, // "Golf Club" (~0.36k frags)
	{"h5_oddball", 393532233},   // "Ball" = Oddball (~0.18k frags)
	//
	// ── Hors-arsenal classé (décision produit 2026-07-17) ──
	// Les 26 stock_ids suivants sont des frags NON-COMBAT. Ils ONT désormais un rôle
	// dédié pour apparaître dans le donut « Frags par type d'arme », MAIS ces rôles
	// (vehicle/turret/environmental/unattributed/other) sont exclus de l'insight
	// coach côté web (NON_COMBAT_WEAPON_ROLES). Ne PAS les mapper vers les sentinels
	// mêlée/grenade (disjoints) — sinon double-comptage.
	// Véhicules → h5_vehicle_* (role `vehicle`) :
	{"h5_vehicle_ghost", 3010146366},    // "Ghost"    (~1.2k)
	{"h5_vehicle_mongoose", 1063919886}, // "Mongoose" (~0.9k)
	{"h5_vehicle_warthog", 4028516791},  // "Warthog"  (~0.8k)
	{"h5_vehicle_banshee", 419783896},   // "Banshee"  (~0.4k)
	{"h5_vehicle_mantis", 3227919741},   // "Mantis"   (~0.25k)
	{"h5_vehicle_scorpion", 1730553442}, // "Scorpion" (~0.24k)
	{"h5_vehicle_wraith", 1206711506},   // "Wraith"   (~0.1k)
	{"h5_vehicle_wasp", 3207900961},     // "Wasp"     (~0.1k)
	{"h5_vehicle_phaeton", 3394982816},  // "Phaeton"  (~5)
	// Tourelles → h5_turret_* (role `turret`) :
	{"h5_turret_chaingun", 2988661926},    // "Chaingun Turret"    (~0.36k)
	{"h5_turret_splinter", 1749823285},    // "Splinter Turret"    (~0.24k)
	{"h5_turret_rocket_pod", 2907783784},  // "Rocket Pod Turret"  (~0.2k)
	{"h5_turret_gauss", 4233134183},       // "Gauss Turret"       (~68)
	{"h5_turret_shade_plasma", 698769165}, // "Shade Plasma Turret" (~25)
	{"h5_turret_scorpion_ai", 244872079},  // "Scorpion Anti-Infantry Turret" (~8)
	{"h5_turret_plasma", 2023669721},      // "Plasma Turret"      (~7)
	{"h5_turret_hunter_arm", 1351500565},  // "Hunter Arm Turret"  (~6)
	// Environnement → h5_environmental (role `environmental`) :
	{"h5_environmental", 47178948}, // "Environmental Explosives" (~0.6k, hasard de map)
	// Bucket d'attribution → h5_unattributed (role `unattributed`). IRRÉDUCTIBLE :
	// stock_id natif réel, disjoint des compteurs mêlée/assassinat (le ventiler =
	// double-comptage, investigation 2026-07-17).
	{"h5_unattributed", 3168248199}, // "Spartan" (~8.8k)
	// UGC / inconnus (absents de weapon_labels) → un SEUL bucket h5_other_ugc
	// (role `other`). Plusieurs weapon_ids → un weapon_key (autorisé).
	{keyH5OtherUGC, 2457457776}, // (~2.3k)
	{keyH5OtherUGC, 390856427},  // (~11)
	{keyH5OtherUGC, 3541732101}, // (~6)
	{keyH5OtherUGC, 642449794},  // (~4)
	{keyH5OtherUGC, 2497647768}, // (~4)
	{keyH5OtherUGC, 2631958027}, // (~2)
	{keyH5OtherUGC, 2957796559}, // (~2)
}
