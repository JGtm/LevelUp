package migration

// steps_metadata_citation_fix.go — corrige les image_path doublement encodés
// dans citation_mappings (cf. seed.go § ImagePath littéral).
//
// Pourquoi : 16 entrées du seed avaient des `%XX` URL-encodés directement dans
// `image_path`. `analysis.BuildCitationSnippets` applique `url.PathEscape` sur
// chaque segment → `%` devient `%25`. Le navigateur décode une fois, le handler
// statique cherche `H5G_citation_I%27m_just_perfect.png` mais le fichier sur
// disque s'appelle `H5G_citation_I'm_just_perfect.png` (apostrophe littérale).
// Résultat : 404 et image vide pour Zéro défaut, Annexion forcée, Œil de lynx,
// Tueur d'Élites, Épée à énergie, etc.
//
// Cette migration met à jour les BDD metadata existantes ; le seed.go a été
// corrigé en parallèle, donc les nouveaux setups partent juste avec les bons
// chemins. Idempotente (UPDATE par PK citation_name_norm).

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "fix_citation_image_paths_double_encoded",
		TargetDB:    TargetMetadata,
		Description: "Remplace les %XX URL-encodés par leurs caractères littéraux dans citation_mappings.image_path (16 entrées : Zéro défaut, Œil de lynx, etc.) — voir seed.go pour la liste canonique",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_À_la_charge.png'                  WHERE citation_name_norm = 'charge';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Annexion_forcée.png'              WHERE citation_name_norm = 'forced_annexation';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Défenseur_du_drapeau.png'         WHERE citation_name_norm = 'flag_defender';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Je_te_tiens_!.png'                WHERE citation_name_norm = 'got_you';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Écrasement.png'                   WHERE citation_name_norm = 'splatter';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Grenade_à_fragmentation.png'      WHERE citation_name_norm = 'frag_grenade';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Grenade_à_plasma.png'             WHERE citation_name_norm = 'plasma_grenade';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Combat_rapproché.png'             WHERE citation_name_norm = 'close_combat';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Tir_à_la_tête.png'                WHERE citation_name_norm = 'headshot';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Œil_de_lynx.png'                  WHERE citation_name_norm = 'eagle_eye';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Flag_''em_down.png'               WHERE citation_name_norm = 'flag_em_down';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_I''m_just_perfect.png'            WHERE citation_name_norm = 'im_just_perfect';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Destructeur_d''apparitions.png'   WHERE citation_name_norm = 'wraith_destroyer';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Tueur_d''Élites.png'              WHERE citation_name_norm = 'elite_slayer';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Tueur_de_répliques_de_Marines.png' WHERE citation_name_norm = 'marine_slayer';
				UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Épée_à_énergie.png'               WHERE citation_name_norm = 'energy_sword_mastery';
			`)
		},
	})
}
