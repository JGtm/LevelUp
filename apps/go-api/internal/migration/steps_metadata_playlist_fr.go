// Package migration — steps_metadata_playlist_fr.go : seed des traductions FR
// de playlists pour corriger les entrées asset_translations où l'API Halo a
// renvoyé l'EN brut en lang fr-FR (corruption ingestion observée 2026-05-09).
//
// Stratégie : UPDATE ciblé par EN canonique connu, garde-fou WHERE name = EN
// pour ne JAMAIS écraser une traduction FR déjà correcte. Idempotent.
//
// Liste maintenue manuellement, basée sur la localisation Halo Infinite
// officielle (FR-FR Microsoft). Les playlists custom (SURVIVE THE UNDEAD,
// etc.) ne sont volontairement pas localisées.
package migration

import (
	"database/sql"
	"fmt"
	"strings"
)

// playlistFRMapping représente une correspondance canonique EN → FR pour les
// noms de playlist Halo Infinite qui ont une localisation officielle.
type playlistFRMapping struct {
	en string
	fr string
}

// playlistFRSeeds : ordre alphabétique sur le label EN. À mettre à jour quand
// une nouvelle saison ajoute des playlists ou que la corruption asset_translations
// est observée sur de nouveaux UUIDs.
var playlistFRSeeds = []playlistFRMapping{
	{"Big Team Battle", "Grand combat en équipe"},
	{"Big Team Battle: Refresh", "Grand combat en équipe : Renouveau"},
	{"Big Team Heavies", "Grand combat lourd"},
	{"Big Team Social", "Combat social en équipe"},
	{"Bot Bootcamp", "Camp d'entraînement bots"},
	{"Firefight", "Combat de feu"},
	{"Firefight: Heroic King of the Hill", "Combat de feu : Roi de la colline héroïque"},
	{"Firefight: King of the Hill", "Combat de feu : Roi de la colline"},
	{"Firefight: Legendary King of the Hill", "Combat de feu : Roi de la colline légendaire"},
	{"Fracture: Tenrai", "Fracture : Tenrai"},
	{"Fracture: Tenrai - Refresh", "Fracture : Tenrai – Renouveau"},
	{"Husky Raid", "Husky Raid"},
	{"Quick Play", "Partie rapide"},
	{"Quick Play: Refresh", "Partie rapide : Renouveau"},
	{"Ranked Arena", "Arène classée"},
	{"Ranked Doubles", "Duo classé"},
	{"Ranked Slayer", "Assassin classé"},
	{"Ranked Snipers", "Snipers classés"},
	{"Rumble Pit", "Combat libre"},
	{"Squad Battle", "Combat en escouade"},
	{"Super Fiesta", "Méga fiesta"},
	{"Super Husky Raid", "Super Husky Raid"},
	{"Tactical Slayer", "Assassin tactique"},
	{"Tactical Slayer (Snipers)", "Assassin tactique (Snipers)"},
	{"Team Doubles", "Duo en équipe"},
	{"Team Slayer", "Assassin en équipe"},
	{"Team Snipers", "Snipers en équipe"},
}

// applyPlaylistFRSeeds met à jour metadata.asset_translations pour les
// playlists où la lang fr-FR (et fr) contient l'EN raw, en remplaçant par la
// traduction canonique. Garde-fou strict : WHERE name = EN — n'écrase JAMAIS
// une traduction FR déjà correcte.
//
// Égalemement insère une ligne fr-FR si aucune n'existait pour ce asset_id
// (cas rare où l'ingestion a écrit en-US uniquement) — INSERT OR IGNORE pour
// rester idempotent.
func applyPlaylistFRSeeds(db *sql.DB) error {
	for _, seed := range playlistFRSeeds {
		// 1. UPDATE des lignes existantes où fr/fr-FR == EN raw.
		//    Le matching est par NAME (pas par asset_id) car asset_translations
		//    est indexée par UUID + lang ; on identifie les lignes corrompues
		//    par leur valeur courante == nom EN.
		if _, err := db.ExecContext(bootCtx(), `
			UPDATE asset_translations
			SET name = ?
			WHERE asset_type = 'playlist'
			  AND lang IN ('fr', 'fr-FR')
			  AND name = ?`,
			seed.fr, seed.en,
		); err != nil {
			return fmt.Errorf("applyPlaylistFRSeeds UPDATE %q: %w", seed.en, err)
		}

		// 2. INSERT OR IGNORE des lignes fr-FR manquantes : si on a une ligne
		//    en-US avec ce nom mais aucune ligne fr-FR pour le même asset_id.
		//    Couvre le cas où la sync n'a stocké que la lang en-US.
		if _, err := db.ExecContext(bootCtx(), `
			INSERT OR IGNORE INTO asset_translations (asset_id, asset_type, lang, name)
			SELECT en.asset_id, 'playlist', 'fr-FR', ?
			FROM asset_translations en
			WHERE en.asset_type = 'playlist'
			  AND en.lang = 'en-US'
			  AND en.name = ?
			  AND NOT EXISTS (
			    SELECT 1 FROM asset_translations fr
			    WHERE fr.asset_id = en.asset_id
			      AND fr.asset_type = 'playlist'
			      AND fr.lang IN ('fr', 'fr-FR')
			  )`,
			seed.fr, seed.en,
		); err != nil {
			// INSERT OR IGNORE peut échouer sur PK si fr existe déjà pour ce asset_id —
			// c'est OK, on log mais on continue.
			if !strings.Contains(err.Error(), "Constraint Error") {
				return fmt.Errorf("applyPlaylistFRSeeds INSERT %q: %w", seed.en, err)
			}
		}
	}
	return nil
}
