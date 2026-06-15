package migrations

// mode_playlist_fr.go — steps add_mode_name_tr + seed_playlist_fr_translations
// (named-func), déplacés depuis internal/migration (Phase 1.5 b7, voie B). Ils
// PARTAGENT les consts mode* → déplacés ensemble (sinon const inaccessibles).
// Seeds statiques idempotents (INSERT OR IGNORE / UPDATE gardé), zéro API.
// Les noms restent dans migration.canonicalOrder.

import (
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/migration"
)

// Mode canoniques (EN) — utilisés comme clés dans mode_name_tr et comme labels FR
// identiques (Halo n'a pas de traduction officielle pour ces modes).
const (
	modeAttrition  = "Attrition"
	modeExtraction = "Extraction"
	modeOddball    = "Oddball"
)

// Labels mode partagés entre mode_name_tr et playlist_fr.
const (
	modeTeamSlayer    = "Team Slayer"
	modeTeamSnipers   = "Team Snipers"
	modeTeamSlayerFR  = "Assassin en équipe"
	modeTeamSnipersFR = "Snipers en équipe"
)

// applyModeNameTr crée et peuple mode_name_tr avec les traductions connues.
func applyModeNameTr(db *sql.DB) error {
	if _, err := db.ExecContext(migration.BootCtx(), `
		CREATE TABLE IF NOT EXISTS mode_name_tr (
			mode_en VARCHAR NOT NULL,
			lang    VARCHAR NOT NULL,
			name    VARCHAR NOT NULL,
			PRIMARY KEY (mode_en, lang)
		)
	`); err != nil {
		return err
	}

	type modeRow struct{ modeEN, lang, name string }
	rows := []modeRow{
		{"Assault", "en", "Assault"},
		{modeAttrition, "en", modeAttrition},
		{"CTF", "en", "CTF"},
		{"CTF 3 Captures", "en", "CTF (3 Captures)"},
		{"Escalation Slayer", "en", "Escalation Slayer"},
		{modeExtraction, "en", modeExtraction},
		{"FFA Slayer", "en", "FFA Slayer"},
		{"Fiesta CTF", "en", "Fiesta CTF"},
		{"Fiesta Slayer", "en", "Fiesta Slayer"},
		{"Fiesta Total Control", "en", "Fiesta Total Control"},
		{"Heroic KOTH", "en", "King of the Hill (Heroic)"},
		{"Heroic King of the Hill", "en", "King of the Hill (Heroic)"},
		{"King of the Hill", "en", "King of the Hill"},
		{"Land Grab", "en", "Land Grab"},
		{"Legendary King of the Hill", "en", "King of the Hill (Legendary)"},
		{"Neutral Bomb", "en", "Neutral Bomb"},
		{"Neutral Bomb Squad", "en", "Neutral Bomb Squad"},
		{"Neutral Flag CTF", "en", "Neutral Flag CTF"},
		{modeOddball, "en", modeOddball},
		{"One Bomb", "en", "One Bomb"},
		{"One Flag CTF", "en", "One Flag CTF"},
		{"Sentry Defense", "en", "Sentry Defense"},
		{"Shotty Snipe Slayer FFA", "en", "Shotty Snipers FFA"},
		{"Shotty Snipes Slayer", "en", "Shotty Snipers"},
		{"Slayer", "en", "Slayer"},
		{"Stockpile", "en", "Stockpile"},
		{"Strongholds", "en", "Strongholds"},
		{modeTeamSlayer, "en", modeTeamSlayer},
		{modeTeamSnipers, "en", modeTeamSnipers},
		{"Total Control", "en", "Total Control"},
		{"VIP", "en", "VIP"},
		// FR
		{"Assault", "fr", "Assaut"},
		{modeAttrition, "fr", modeAttrition},
		{"CTF", "fr", "Capture du drapeau"},
		{"CTF 3 Captures", "fr", "CDD 3 captures"},
		{"Escalation Slayer", "fr", "Escalade"},
		{modeExtraction, "fr", modeExtraction},
		{"FFA Slayer", "fr", "Chacun pour soi"},
		{"Fiesta CTF", "fr", "Fiesta CDD"},
		{"Fiesta Slayer", "fr", "Fiesta"},
		{"Fiesta Total Control", "fr", "Fiesta Contrôle total"},
		{"Heroic KOTH", "fr", "Roi de la colline héroïque"},
		{"Heroic King of the Hill", "fr", "Roi de la colline héroïque"},
		{"King of the Hill", "fr", "Roi de la colline"},
		{"Land Grab", "fr", "Bases"},
		{"Legendary King of the Hill", "fr", "Roi de la colline légendaire"},
		{"Neutral Bomb", "fr", "Bombe neutre"},
		{"Neutral Bomb Squad", "fr", "Escouade bombe neutre"},
		{"Neutral Flag CTF", "fr", "Drapeau neutre"},
		{modeOddball, "fr", modeOddball},
		{"One Bomb", "fr", "Bombe neutre"},
		{"One Flag CTF", "fr", "Drapeau neutre"},
		{"Sentry Defense", "fr", "Défense sentinelle"},
		{"Shotty Snipe Slayer FFA", "fr", "Fusils snipers à grenaille FFA"},
		{"Shotty Snipes Slayer", "fr", "Fusils snipers à grenaille"},
		{"Slayer", "fr", "Assassin"},
		{"Stockpile", "fr", "Stockage"},
		{"Strongholds", "fr", "Bases"},
		{modeTeamSlayer, "fr", modeTeamSlayerFR},
		{modeTeamSnipers, "fr", modeTeamSnipersFR},
		{"Total Control", "fr", "Contrôle total"},
		{"VIP", "fr", "VIP"},
	}

	for _, r := range rows {
		if _, err := db.ExecContext(migration.BootCtx(),
			"INSERT OR IGNORE INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)",
			r.modeEN, r.lang, r.name,
		); err != nil {
			return err
		}
	}
	return nil
}

// playlistFRMapping représente une correspondance canonique EN → FR pour les
// noms de playlist Halo Infinite qui ont une localisation officielle.
type playlistFRMapping struct {
	en string
	fr string
}

// playlistFRSeeds : ordre alphabétique sur le label EN.
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
	{modeTeamSlayer, modeTeamSlayerFR},
	{modeTeamSnipers, modeTeamSnipersFR},
}

// ReconcileMetadataSeeds ré-applique les seeds de traduction idempotents (modes +
// playlists FR) sur la metadata.duckdb. À appeler au boot, juste après
// RunForDB(db, TargetMetadata). Déplacé depuis internal/migration (Phase 1.5 b7) :
// dépend de applyModeNameTr/applyPlaylistFRSeeds, désormais title-owned.
//
// Pourquoi (fix 2026-05-30) : les seeds sont portés par des migrations one-shot ;
// quand de nouvelles traductions sont ajoutées APRÈS qu'une base a marqué la
// migration "done", elle ne les reçoit jamais. Rejouer à chaque boot (strictement
// idempotent) fait converger toute traduction ajoutée. Non destructif.
func ReconcileMetadataSeeds(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := applyModeNameTr(db); err != nil {
		return fmt.Errorf("reconcile mode_name_tr: %w", err)
	}
	if err := applyPlaylistFRSeeds(db); err != nil {
		return fmt.Errorf("reconcile playlist FR: %w", err)
	}
	return nil
}

// applyPlaylistFRSeeds met à jour metadata.asset_translations pour les playlists
// où la lang fr-FR (et fr) contient l'EN raw, en remplaçant par la traduction
// canonique. Garde-fou strict : WHERE name = EN — n'écrase JAMAIS une traduction
// FR déjà correcte. Insère aussi une ligne fr-FR si aucune n'existait (idempotent).
func applyPlaylistFRSeeds(db *sql.DB) error {
	for _, seed := range playlistFRSeeds {
		if _, err := db.ExecContext(migration.BootCtx(), `
			UPDATE asset_translations
			SET name = ?
			WHERE asset_type = 'playlist'
			  AND lang IN ('fr', 'fr-FR')
			  AND name = ?`,
			seed.fr, seed.en,
		); err != nil {
			return fmt.Errorf("applyPlaylistFRSeeds UPDATE %q: %w", seed.en, err)
		}

		if _, err := db.ExecContext(migration.BootCtx(), `
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
			if !strings.Contains(err.Error(), "Constraint Error") {
				return fmt.Errorf("applyPlaylistFRSeeds INSERT %q: %w", seed.en, err)
			}
		}
	}
	return nil
}
