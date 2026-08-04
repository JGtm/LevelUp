package weapons

// labels.go — table weapon_labels (weapon_id -> nom EN/FR) : DDL partagé par tous
// les titres + seed des IDs Halo Infinite. Le step add_weapon_labels
// (halo_infinite/migrations) y délègue, comme h5_add_weapon_labels pour le DDL
// seul (les IDs d'armes Halo 5 diffèrent : sa table reste VIDE, peuplée par le
// fetcher halo5api.svc/weapons).
//
// Origine : internal/migration/steps_metadata.go → halo_infinite/migrations
// (Phase 1.5 b6) → ici (2026-08-04, cf. registry.go). Seed statique pur
// (INSERT OR IGNORE, aucune lecture DB, aucune API).

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// labelEnergySwordFR — label FR récurrent (épée à énergie + variantes).
const labelEnergySwordFR = "Épée à énergie"

// EnsureLabelsTable crée weapon_labels (idempotent), SANS aucune donnée. Source
// unique du DDL : un titre dont les IDs d'armes diffèrent (Halo 5) crée sa table
// via cette fonction plutôt que de recopier le CREATE TABLE.
func EnsureLabelsTable(db *sql.DB) error {
	_, err := db.ExecContext(migration.BootCtx(), `
		CREATE TABLE IF NOT EXISTS weapon_labels (
			weapon_id UBIGINT PRIMARY KEY,
			name_en   VARCHAR NOT NULL,
			name_fr   VARCHAR NOT NULL
		)
	`)
	return err
}

// ApplyLabels crée weapon_labels et la peuple avec les IDs Halo Infinite connus.
// Idempotent (CREATE TABLE IF NOT EXISTS + INSERT OR IGNORE) : sert à la fois de
// step de migration (add_weapon_labels) et de reseed CLI (cmd/seed-weapon-labels),
// même quand schema_migrations marque la migration comme done.
func ApplyLabels(db *sql.DB) error {
	if err := EnsureLabelsTable(db); err != nil {
		return err
	}

	// Sentinels + confirmed weapons (portage de WEAPON_INT_TO_NAME + WEAPON_NAME_FR)
	//
	// Contrainte driver : database/sql ne supporte pas uint64 avec bit63=1.
	// Contournement : injecter weapon_id comme littéral décimal (valeur constante),
	// les noms restent paramétrisés.
	type label struct {
		id uint64
		en string
		fr string
	}
	labels := []label{
		{0, "Grenade", "Grenade"},
		{1, "Melee", "Corps à corps"},
		{2, "Vehicle", "Véhicule"},
		{0x6acdc44d42c9679f, "Bandit Evo", "Bandit EVO"},
		{0x2b1824d542c9679f, "BR75", "BR75"},
		{0x230447b142c9679f, "Cindershot", "Crémateur"},
		{0xb619d84a42c9679f, "CQS48 Bulldog", "CQS48 Bulldog"},
		{0x84bd29ed42c9679f, "Disruptor", "Disrupteur"},
		{0x9d6aaed242c9679f, "Fuel Rod SPNKr", nameM41SPNKr},
		{0x841ac5e542c9679f, "Gravity Hammer", "Marteau antigravité"},
		{0x2ac9c2ff42c9679f, "Heatwave", "Calcineur"},
		{0x71ab0a2c42c9679f, nameM41SPNKr, nameM41SPNKr},
		{0x2fb21c8742c9679f, "M392 Bandit", "Bandit EVO"},
		{0x48c19d2d42c9679f, "MA40 AR", "MA40 AR"},
		{0xf5c335dfe7232c0f, "MA5K Avenger", "MA5K Avenger"},
		{0x80977ba542c9679f, "Mangler", "Déchiqueteur"},
		{0x767db96d42c9679f, "MLRS-2 Hydra", "Hydra"},
		{0xf408190f42c9679f, "Mk51 Sidekick", "MK50 Sidekick"},
		{0xd791556542c9679f, "Mutilator", "Mutilateur"},
		{0xb7262ca1c8fb11d0, "Mythic Sandwich", "Mythic Sandwich"},
		{0xb533957e42c9679f, "Needler", "Needler"},
		{0xc354294642c9679f, "Plasma Pistol", "Pistolet à plasma"},
		{0x30484ea642c9679f, "Pulse Carbine", "Carabine à impulsion"},
		{0xc30d87c742c9679f, "Ravager", "Ravageur"},
		{0x0a1992bc42c9679f, "S7 Sniper", "S7 Sniper"},
		{0x880fe0bc42c9679f, "Sandwich", "Sandwich"},
		{0xa0955e9e42c9679f, "Sentinel Beam", "Rayon de Sentinelle"},
		{0x9387a8b942c9679f, "Shock Rifle", "Fusil électrique"},
		{0x1a22fee642c9679f, "Shock Rifle (Ranked)", "Fusil électrique"},
		{0x0d20c46942c9679f, "Skewer", "Empaleur"},
		{0xdaf193c742c9679f, "Stalker Rifle", "Fusil traqueur"},
		{0x3e07021742c9679f, "Vestige Carbine", "Carabine Vestige"},
		{0xfd98554c42c9679f, "VK78 Commando", "VK78 Commando"},
		{0x4ff3937e42c9679f, "Energy Sword", labelEnergySwordFR},
		{0x4ff3937e8978aa7a, "Duelist Energy Sword", labelEnergySwordFR},
		{0x4ff3937e1ec48c7a, "Elite Bloodblade", labelEnergySwordFR},
		{0x0c55765f7a9376a0, "Infected Energy Sword", labelEnergySwordFR},
		{0x841ac5e5a730e49f, "Diminisher of Hope", "Marteau antigravité"},
		{0x841ac5e5d8d07ca1, "Rushdown Hammer", "Marteau antigravité"},
		{0xb6dbead842c9679f, "Frag Grenade", "Grenade frag"},
		{0xc1e1bab042c9679f, "Plasma Grenade", "Grenade plasma"},
		{0x3ad55da442c9679f, "Dynamo Grenade", "Grenade dynamo"},
	}

	for _, l := range labels {
		// Contournement driver : database/sql ne supporte pas uint64 avec bit63=1.
		// weapon_id est une constante interne (pas user input) → littéral décimal sûr.
		q := fmt.Sprintf( //nolint:gosec
			"INSERT OR IGNORE INTO weapon_labels (weapon_id, name_en, name_fr) VALUES (%d, ?, ?)",
			l.id,
		)
		if _, err := db.ExecContext(migration.BootCtx(), q, l.en, l.fr); err != nil {
			return err
		}
	}
	return nil
}
