package migration

// killer_victim_pairs_swap.go — REMPLACER LES COUPLES DE PLUSIEURS MATCHS SANS DELETE.
//
// L entretien `levelup repair-highlight-phantoms` recalcule les couples `killer_victim_pairs` des
// matchs dont les couples stockes different de l appariement courant. Il ne les reecrit pas par
// DELETE + INSERT : une base nee avant la migration `shared_killer_victim_pairs_drop_legacy_indexes_v1`
// porte encore des index ART sur la table, et un DELETE y est le declencheur du bug DuckDB ART
// #23645 (ADR 0019). La table est reconstruite a cote — les couples des autres matchs tels quels,
// ceux des matchs recalcules depuis une table de travail sans index — puis echangee dans UNE
// transaction ([swapTableTx] : garde de cardinalite, DDL et index identiques, rollback integral).

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// KillerVictimPair : un couple par-kill de `killer_victim_pairs`.
type KillerVictimPair struct {
	KillerXUID, KillerGamertag string
	VictimXUID, VictimGamertag string
	TimeMS                     int64
}

const (
	kvpTable  = "killer_victim_pairs"
	kvpSuffix = "__couples"
	// kvpTravail / kvpMatchs : tables de travail sans index, recreees a chaque appel.
	kvpTravail = "killer_victim_pairs__recalcul"
	kvpMatchs  = "killer_victim_pairs__recalcul_matchs"
)

// ReplaceKillerVictimPairs remplace les couples de chaque match de `parMatch` par ceux donnes (une
// liste vide retire les couples du match) et garde ceux des autres matchs. Rend le nombre de
// lignes de la table apres l echange. Aucun match : rien n est ecrit.
func ReplaceKillerVictimPairs(ctx context.Context, db *sql.DB, parMatch map[string][]KillerVictimPair) (int64, error) {
	if len(parMatch) == 0 {
		return 0, nil
	}
	if err := remplirLesTablesDeTravail(ctx, db, parMatch); err != nil {
		return 0, err
	}
	defer func() {
		_, _ = db.ExecContext(context.WithoutCancel(ctx), // tables de travail : leur reste est refait au prochain appel
			fmt.Sprintf(`DROP TABLE IF EXISTS %s; DROP TABLE IF EXISTS %s`, kvpTravail, kvpMatchs))
	}()
	ddl, err := ddlDeTable(ctx, db, kvpTable)
	if err != nil {
		return 0, err
	}
	creer, err := ddlDeConstructionSuffixe(ddl, kvpTable, kvpSuffix)
	if err != nil {
		return 0, err
	}
	index, err := ddlDesIndex(ctx, db, kvpTable)
	if err != nil {
		return 0, err
	}
	garder := fmt.Sprintf(`WHERE match_id NOT IN (SELECT match_id FROM %s)`, kvpMatchs)
	n, err := swapTableTx(ctx, db, tableSwap{
		Table:  kvpTable,
		Suffix: kvpSuffix,
		Expected: fmt.Sprintf(`SELECT (SELECT COUNT(*) FROM %s %s) + (SELECT COUNT(*) FROM %s)`,
			kvpTable, garder, kvpTravail),
		Build: []string{creer,
			fmt.Sprintf(`INSERT INTO %s%s SELECT * FROM %s %s`, kvpTable, kvpSuffix, kvpTable, garder),
			fmt.Sprintf(`INSERT INTO %s%s (match_id, killer_xuid, killer_gamertag, victim_xuid,
				victim_gamertag, time_ms) SELECT match_id, killer_xuid, killer_gamertag, victim_xuid,
				victim_gamertag, time_ms FROM %s`, kvpTable, kvpSuffix, kvpTravail),
		},
		PostRename: index,
		Verify: func(ctx context.Context, tx *sql.Tx) error {
			return verifierSchemaIdentique(ctx, tx, kvpTable, ddl, index)
		},
	})
	if err != nil {
		return 0, err
	}
	return n.Rebuilt, nil
}

// remplirLesTablesDeTravail : les couples recalcules et la liste des matchs, dans deux tables sans
// index, en une transaction.
func remplirLesTablesDeTravail(ctx context.Context, db *sql.DB, parMatch map[string][]KillerVictimPair) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("couples: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit
	if err := execAll(ctx, tx, kvpTable, []string{
		fmt.Sprintf(`DROP TABLE IF EXISTS %s`, kvpTravail),
		fmt.Sprintf(`DROP TABLE IF EXISTS %s`, kvpMatchs),
		fmt.Sprintf(`CREATE TABLE %s (match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL, victim_gamertag VARCHAR,
			time_ms INTEGER)`, kvpTravail),
		fmt.Sprintf(`CREATE TABLE %s (match_id VARCHAR NOT NULL)`, kvpMatchs),
	}); err != nil {
		return err
	}
	ids := make([]string, 0, len(parMatch))
	for id := range parMatch {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES (?)`, kvpMatchs), id); err != nil {
			return fmt.Errorf("couples: match %s: %w", id, err)
		}
		for _, p := range parMatch[id] {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES (?, ?, ?, ?, ?, ?)`, kvpTravail),
				id, p.KillerXUID, nulSiVide(p.KillerGamertag), p.VictimXUID, nulSiVide(p.VictimGamertag),
				p.TimeMS); err != nil {
				return fmt.Errorf("couples: couple de %s: %w", id, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couples: commit des tables de travail: %w", err)
	}
	return nil
}

// nulSiVide : un gamertag inconnu s ecrit NULL, jamais une chaine vide.
func nulSiVide(s string) any {
	if s == "" {
		return nil
	}
	return s
}
