package migrations

// steps_metadata_reclass_sword_hammer.go — L'ÉPÉE ET LE MARTEAU DEVIENNENT DES ARMES
// LOURDES SUR LES BASES DÉJÀ SEMÉES.
//
// # Pourquoi une migration, alors que le registre est du code
//
// `weapons.ReconcileRegistry` rejoue le seed à CHAQUE boot — mais en `INSERT OR IGNORE`,
// et c'est écrit noir sur blanc dans son en-tête : « Le rejouer n'insère QUE les lignes
// manquantes : aucune écriture destructive, AUCUN UPDATE » (décision du 2026-06-23).
// Conséquence mesurée le 2026-09-01 : ajouter une CLÉ au registre Go se propage tout seul
// (les 14 véhicules et tourelles sont arrivés ainsi), mais CHANGER LA CLASSE d'une clé
// déjà semée ne se propage JAMAIS. Sans cette étape, le reclassement resterait vrai dans
// le binaire et faux dans toutes les `metadata.duckdb` existantes — le pire des deux.
//
// # Ce qu'il corrige, et le chiffre qui le fonde
//
// `hinf_energy_sword` et `hinf_gravity_hammer` étaient `class = 'melee'`. Le lecteur de la
// répartition des frags écarte cette classe : son TOTAL vient du compteur API, autoritatif
// (décision D4 du plan `.ai/V7.5/PLAN_SOURCE_UNIQUE_ARME_2026-09-01.md`).
//
// OR CE COMPTEUR NE LES COMPTE PAS. Mesure du 2026-09-01 sur 200 matchs :
// `match_participants.melee_kills` vaut 1 717, quand l'épée et le marteau pèsent 2 514 à
// eux deux. Sur le corpus entier : marteau 6 727, épée 3 014 — 9 741 frags qui tombaient
// dans « Non attribué » sans qu'aucune surface ne les serve. Le registre conflatait l'ARME
// de corps à corps et la MÉCANIQUE de corps à corps ; le jeu, lui, ne les confond pas.
// Décision de l'utilisateur : « bien sûr que les épées et marteaux ça ne compte pas dans
// les stats de mêlée ».
//
// Aucun double comptage possible : l'écart est MESURÉ, pas supposé.
//
// # Sûreté
//
// UPDATE de deux lignes désignées par leur clé primaire sur un référentiel STATIQUE, sans
// writer concurrent — le périmètre exact que la décision du 2026-06-23 place hors du bug
// ART #23046. Idempotent : rejoué, il réécrit les mêmes valeurs.

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// reclasserEpeeEtMarteau passe les deux clés en `heavy` / `power`.
//
// Le `WHERE` nomme les clés et le titre : ce sont des DONNÉES de la table (chaque ligne du
// registre porte son `title_slug`), pas une branche de code sur un slug. Sur une metadata
// d'un autre titre l'UPDATE ne trouve rien et ne fait rien.
func reclasserEpeeEtMarteau(db *sql.DB) error {
	if has, err := migration.TableExists(db, "weapons"); err != nil || !has {
		// Base sans registre d'armes (metadata vierge) : le seed posera directement les
		// bonnes valeurs, il n'y a rien à corriger.
		return err
	}
	_, err := db.ExecContext(migration.BootCtx(), `
		UPDATE weapons
		SET class = 'heavy', role = 'power'
		WHERE title_slug = 'halo_infinite'
		  AND weapon_key IN ('hinf_energy_sword', 'hinf_gravity_hammer')`)
	if err != nil {
		return fmt.Errorf("reclassement epee/marteau: %w", err)
	}
	return nil
}

// stepReclassSwordHammer — l'étape, référencée depuis Steps().
func stepReclassSwordHammer() migration.Migration {
	return migration.Migration{
		Name:        "metadata_reclass_sword_hammer_heavy_v1",
		TargetDB:    migration.TargetMetadata,
		Description: "Épée à énergie et marteau à gravité : classe melee -> heavy, rôle power (le compteur API melee_kills ne les compte pas)",
		ApplySchema: reclasserEpeeEtMarteau,
	}
}
