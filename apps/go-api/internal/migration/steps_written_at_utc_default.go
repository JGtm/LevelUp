package migration

// steps_written_at_utc_default.go — REPARE le DEFAULT de `written_at` sur les bases
// EXISTANTES (lot de suivi S2, suite du constat R1).
//
// Le defaut : `written_at TIMESTAMP ... DEFAULT now()` (ou CURRENT_TIMESTAMP). Les deux
// expressions rendent un TIMESTAMPTZ que DuckDB coerce vers une colonne TIMESTAMP NAIVE
// par le fuseau de SESSION. Sur un poste a UTC+2, toute ligne ecrite SANS valeur
// explicite se date donc deux heures dans le FUTUR, alors que les ecrivains applicatifs
// posent `time.Now().UTC()`. Or `written_at` est la colonne de TRI des vues `<table>_latest`
// (ADR 0026) : melanger deux horloges sur cette colonne, c'est melanger deux PRESEANCES.
// Une ligne datee au fuseau local gagne l'arbitrage contre toute ligne UTC ecrite dans les
// deux heures qui suivent — l'enrichissement disparait de la lecture sans erreur, sans
// compteur, sans qu'un seul nom ni un seul compte ne bouge (mecanisme demontre par R1).
//
// Le DDL source est corrige a la racine (forme canonique TimestampDefaultUTC partout,
// tenue par arbitration_clocks_utc_guard_test.go), ce qui suffit aux bases NEUVES. Ce step traite
// les bases DEJA CREEES : `CREATE TABLE IF NOT EXISTS` ne retouche jamais une table
// existante, seul un ALTER le fait.
//
// Data-driven a dessein : il repare TOUTE colonne `written_at` naive dont le DEFAULT est
// sensible au fuseau, sans liste de tables a maintenir. Enregistre sur les cinq targets
// (une base peut porter n'importe quel sous-ensemble de ces tables) ; no-op complet quand
// il n'y a rien a reparer. Idempotent : apres l'ALTER, le DEFAULT normalise par DuckDB
// (`CAST(main.timezone('UTC', now()) AS TIMESTAMP)`) ne matche plus le predicat.

import (
	"database/sql"
)

// TimestampDefaultUTC — forme canonique du DEFAULT de TOUTE colonne d'horodatage naive
// du depot, et de toute ecriture explicite d'horloge sur une telle colonne. Tout DDL doit
// l'utiliser telle quelle (garde-rail : arbitration_clocks_utc_guard_test.go).
//
// Nommee `WrittenAtDefaultUTC` jusqu'au lot S5 : le lot S6 a etendu la campagne aux autres
// horodatages d'arbitrage (`computed_at` arbitre lusr_component_history_latest, `liked_at`
// arbitre la vue des likes media), la constante n'est donc plus propre a `written_at`.
const TimestampDefaultUTC = "CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)"

// writtenAtColumnFilter restreint la reparation a la seule colonne `written_at` — le
// perimetre historique de ce step. Le predicat complet (bases attachees, TIMESTAMP naif
// seul, idempotence) vit desormais dans steps_zz_arbitration_clocks_utc_default.go : les deux
// steps de la campagne partagent une implementation unique, ce step-ci n'en est que la
// restriction.
const writtenAtColumnFilter = `AND c.column_name = 'written_at'`

// EnsureWrittenAtDefaultsUTC bascule vers TimestampDefaultUTC le DEFAULT de toute colonne
// `written_at` naive encore datee sur le fuseau de session. Retourne le nombre de tables
// reparees. Exporte : rejoue aussi hors runner (outillage de reparation ponctuelle).
//
// Conserve tel quel apres le lot S6 alors qu'EnsureTimestampDefaultsUTC le subsume : son
// step est inscrit au ledger de toutes les bases de prod, et le retirer du registre
// desynchroniserait l'audit d'ordre (order_audit_test.go).
func EnsureWrittenAtDefaultsUTC(db *sql.DB) (int, error) {
	return repairDefectiveDefaults(db, writtenAtColumnFilter)
}

// writtenAtUTCStepName construit le nom du step par target (un nom = une ligne
// schema_migrations, et chaque base a son propre ledger).
func writtenAtUTCStepName(target TargetDB) string {
	return "written_at_default_utc_" + string(target)
}

func init() {
	for _, target := range []TargetDB{
		TargetShared, TargetPlayer, TargetSharedPvE, TargetSharedSocial, TargetMetadata,
	} {
		Register(Migration{
			Name:        writtenAtUTCStepName(target),
			TargetDB:    target,
			Description: "written_at : DEFAULT en UTC explicite (fin du melange de fuseaux sur la colonne de tri des vues _latest)",
			ApplySchema: func(db *sql.DB) error {
				_, err := EnsureWrittenAtDefaultsUTC(db)
				return err
			},
		})
	}
}
