package migration

// steps_shared_weapon_hit_distance.go — table `match_weapon_hit_distance` : LA DISTANCE
// TIREUR<->VICTIME DES TOUCHES, PAR ARME. Table SOEUR de `match_weapon_shots` (meme grain
// `match x joueur x arme`), dediee a la vue (b) « precision par arme selon la distance ».
//
// ─── POURQUOI UNE TABLE NEUVE PLUTOT QU UNE COLONNE DE weapon_accuracy ─────────────────────
//
// La precision par arme (numerateur/denominateur) est ecrite dans la table EXISTANTE
// `weapon_accuracy` (shots_fired / shots_landed), partagee avec Halo 5. La DISTANCE n y a
// PAS sa place : ajouter une colonne a une table DEJA PEUPLEE (H5) forcerait le rebuild ART
// d ADR 0026. La distance vit donc dans cette table soeur, NET-NEUVE — la vue (b) est une
// jointure `weapon_accuracy x match_weapon_hit_distance` a la lecture.
//
// ─── CE QUE STOCKE CETTE TABLE ────────────────────────────────────────────────────────────
//
//	dist_bucket_json  STOCKE   l histogramme des distances des touches de cette arme, en
//	                           JSON (bornes = celles de l instrument d attribution, sondeDistEdges).
//	                           Le grain est deja `match x joueur x arme` : le JSON tient donc
//	                           un petit tableau de comptes par tranche, pas un point par touche.
//	dist_n            STOCKE   nombre de touches dont LES DEUX positions (tireur, victime) se
//	                           sont resolues au ts du degat. C est l effectif de l histogramme
//	                           ci-dessus. dist_n < WeaponHitsMinShots => l histogramme n est
//	                           PAS publiable (voir la porte plus bas).
//	distance brute    HORS PORTEE  aucune ligne par touche : la distance individuelle n a pas
//	                           d usage produit, seul l histogramme par arme est affiche.
//
// ─── L IDENTIFIANT D ARME : MEME ESPACE QUE match_weapon_shots ─────────────────────────────
//
//	weapon_id  UBIGINT  identifiant filmshell 64 bits, celui de metadata.weapon_labels.weapon_id
//	                    et du fire-event (action_weapon_fire 0xD2). CE N EST PAS le tag `jpt!`
//	                    32 bits du dead-state. Insere en CHAINE DECIMALE (piege ubigintArg : un
//	                    id dont le bit de poids fort vaut 1 serait NEGATIF en int64 signe).
//
// ─── LA PORTE DE PUBLICATION (Nmin) — MESUREE, PAS DEVINEE ─────────────────────────────────
//
// Une forme par arme (taux d accuracy, histogramme de distance) n est publiable que si son
// effectif atteint WeaponHitsMinShots : denominateur `shots_fired` pour le taux, `dist_n` pour
// l histogramme. Sous ce seuil, une poignee d observations donne une forme qui n est que du bruit.
//
// WeaponHitsMinShots = 8, FIXE PAR MESURE le 2026-09-01 (instrument lot1_nmin_effectif_research_test)
// sur les trois films de reference, fenetre W = 250 ms, 12 chunks temoin (fenetre TRONQUEE : un
// match d arene complet est plusieurs fois plus long, les effectifs reels sont superieurs).
// Distribution par cle (joueur, arme) du nombre de TIRS decodes :
//
//	film      cles(joueur,arme)  median tirs/cle  cles >=5  cles >=8  cles >=10
//	000d5950         33                5             17       11         10
//	01e1f945         14               60             12       11         11
//	00502e52         34                5             18       11         10
//
// LA JUSTIFICATION DU 8 : le sous-ensemble « >= 8 tirs » est REMARQUABLEMENT STABLE d un film a
// l autre — 11 cles sur les trois, quand le total va de 14 a 34. Le seuil isole donc les armes
// avec lesquelles le joueur s est REELLEMENT battu et jette la longue traine des ramassages a
// 1-4 tirs (33 -> 11, 14 -> 11, 34 -> 11). C est le meme ordre de grandeur que le garde-fou
// `n < 5` deja code dans l instrument d attribution (attribM2/attribM3), releve a 8 : la DISTANCE
// est plus exigeante encore (chaque touche compte dans `dist_n` seulement si SES DEUX positions
// se resolvent, ~2/3 des touches — reserve #3 du plan), mais sur un match complet (non tronque a
// 12 chunks) l effectif d une arme reellement utilisee depasse largement 8.
//
// ─── APPEND-ONLY, MEME UNITE DE GENERATION QUE match_weapon_shots ─────────────────────────
//
// PK technique `id`, `written_at`, INSERT purs (ADR 0026/0030 : aucun UPDATE/DELETE/ON CONFLICT).
// La vue `_latest` retient LA DERNIERE PASSE PAR MATCH (`decode_pass`), jamais la derniere ligne
// par cle : l unite de production est le FILM ENTIER. Un `_latest` ligne par ligne laisserait
// survivre les armes d une passe precedente que la nouvelle ne decode plus. Le persister
// (INSERT-only) arrive au Lot 3 ; ce step ne pose que le schema.

import "database/sql"

// WeaponHitDistanceDecoderRev est la version du numerateur film qui alimente cette table.
// Distincte du decodeur de tirs (match_weapon_shots) : distance et touches sortent de
// l appariement tir<->degat (methode PAR LE TIR, NOTE_ATTRIBUTION_ARME_TIR_2026-08-31),
// pas du simple comptage des fire-events. Sert a savoir QUELS matchs redecoder apres un
// changement du numerateur, au lieu de tout redecoder.
const WeaponHitDistanceDecoderRev = "whd-v1"

// WeaponHitsMinShots est l effectif minimal (`dist_n`) pour qu un histogramme de distances
// par arme soit publiable. Fixe a 8 par mesure (cf. doc-header). Une cle sous ce seuil n est
// JAMAIS affichee : trop peu de touches pour une forme de distribution sensee.
const WeaponHitsMinShots = 8

func init() {
	Register(Migration{
		Name:        "shared_match_weapon_hit_distance_v1",
		TargetDB:    TargetShared,
		Description: "Table append-only match_weapon_hit_distance (match x joueur x arme, distances tireur<->victime des touches) + vue _latest par passe de decodage",
		ApplySchema: applyMatchWeaponHitDistance,
	})
}

// applyMatchWeaponHitDistance cree la table, son index et sa vue. Idempotente. La table etant
// NET-NEUVE, le piege « CREATE TABLE IF NOT EXISTS n ajoute jamais une PK a une table
// existante » ne s applique pas.
func applyMatchWeaponHitDistance(db *sql.DB) error {
	if err := execScript(db, ddlMatchWeaponHitDistance); err != nil {
		return err
	}
	return execScript(db, ddlMatchWeaponHitDistanceLatest)
}

// ddlMatchWeaponHitDistance : la table + son unique index. UN SEUL INDEX, meme argument que
// match_weapon_shots : DuckDB est colonnaire, le seul acces ponctuel reel est « les touches de
// CE match » ; chaque index en plus coute a l INSERT et elargit la surface ART.
const ddlMatchWeaponHitDistance = `
	CREATE SEQUENCE IF NOT EXISTS match_weapon_hit_distance_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_weapon_hit_distance (
		-- identite technique (append-only : PK non naturelle, ADR 0026)
		id                BIGINT    PRIMARY KEY DEFAULT nextval('match_weapon_hit_distance_id_seq'),
		match_id          VARCHAR   NOT NULL,
		-- decode_pass : identifiant d UNE passe de decodage d UN film. Unite de generation.
		decode_pass       VARCHAR   NOT NULL,
		-- decoder_rev : version du numerateur film (WeaponHitDistanceDecoderRev). Sert a savoir
		-- QUELS matchs redecoder apres un changement, au lieu de tout redecoder.
		decoder_rev       VARCHAR   NOT NULL,
		written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		-- ── LE JOUEUR ───────────────────────────────────────────────────────────────────
		-- xuid : identite du tireur. La distance n est calculee QUE pour un joueur resolu
		-- (elle exige deux positions monde) : pas de ligne « bot » ni d indice non rattache.
		xuid              VARCHAR   NOT NULL,

		-- ── L ARME ──────────────────────────────────────────────────────────────────────
		-- weapon_id : identifiant filmshell 64 bits, joignable a metadata.weapon_labels ET a
		-- match_weapon_shots.weapon_id. ⚠ CE N EST PAS le tag jpt! 32 bits de match_kill_events.
		weapon_id         UBIGINT   NOT NULL,

		-- ── LA MESURE : L HISTOGRAMME DE DISTANCE ───────────────────────────────────────
		-- dist_bucket_json : histogramme des distances (m) des touches de cette arme, en JSON
		-- (comptes par tranche, bornes = sondeDistEdges de l instrument d attribution).
		dist_bucket_json  VARCHAR   NOT NULL,
		-- dist_n : nombre de touches dont LES DEUX positions se sont resolues = effectif de
		-- l histogramme. dist_n < WeaponHitsMinShots => non publiable (le lecteur tranche).
		dist_n            INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_weapon_hit_distance_match
		ON match_weapon_hit_distance(match_id, written_at);
`

// ddlMatchWeaponHitDistanceLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026 — une lecture
// brute sert les lignes d une passe de decodage precedente). Retient LA DERNIERE PASSE PAR
// MATCH, jamais la derniere ligne par cle : l unite de production est le film entier.
const ddlMatchWeaponHitDistanceLatest = `
	CREATE OR REPLACE VIEW match_weapon_hit_distance_latest AS
	SELECT d.*
	FROM match_weapon_hit_distance AS d
	QUALIFY d.decode_pass = FIRST_VALUE(d.decode_pass) OVER (
		PARTITION BY d.match_id ORDER BY d.written_at DESC, d.id DESC
	);
`
