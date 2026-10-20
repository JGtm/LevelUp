package migration

// steps_shared_registry_integrity_columns.go — retrait de match_registry.participants_loaded,
// match_registry.player_count et de l'index idx_registry_player_count ; réparation du bit
// « participants » (matchflags.BackfillParticipants) de backfill_completed.
//
// POURQUOI. Les deux colonnes n'ont ni lecteur ni écrivain dans le code (verdict V-1/V-2 du plan
// `.ai/PLAN_BACKLOG_2026-10-15.md`) ; player_count est en plus INDEXÉE, donc toute réécriture de
// la colonne serait un vecteur du bug DuckDB ART #23645. Le bit participants, lui, est LU par la
// sélection du backfill (garde « participants » de sync/backfill.go) et manque sur des matchs qui
// ont leur roster : sans lui, ces matchs sont re-sélectionnés à chaque backfill.
//
// REMÈDE SANS UPDATE : UNE reconstruction par base (swapTableTx, comme
// shared_recompute_mode_category_v1). La table de construction reçoit le DDL exact de la table,
// moins les colonnes retirées (ALTER sur la table vide), puis les lignes, backfill_completed
// augmenté du bit là où le match a des participants quand le titre le demande. Échange en
// transaction ; index reposés après le RENAME sauf ceux qui couvrent une colonne retirée ; vues
// qui lisent la table rejouées (DuckDB fige l'étoile de v_match_full, mv_player_matches citait
// player_count).
//
// DEUX ENTRÉES, UNE RECONSTRUCTION PAR BASE. Le registre de migrations n'a qu'un moyen de borner
// un step à certains titres : `Migration.OnlyTitles`, par step. D'où :
//   - shared_registry_participants_bit_v1 (title-owned, OnlyTitles = les titres dont le moteur
//     écrit ce bit) : reconstruction AVEC le bit ;
//   - shared_registry_drop_integrity_columns_v1 (ci-dessous, tous les titres), placé APRÈS dans
//     canonicalOrder : reconstruction SANS le bit, no-op là où la première a déjà retiré les
//     colonnes.
//
// GARDES, toutes avant le COMMIT : cardinalité ; colonnes (nom, type, défaut, nullité, ordre) et
// contraintes identiques hors colonnes retirées ; index identiques hors ceux retirés ; table
// identique à la projection attendue calculée AVANT l'échange (empreinte) ; chaque vue lisant la
// table identique hors colonnes retirées et hors backfill_completed ; plus aucun match à roster
// sans le bit quand il est demandé. Idempotente : no-op sans écriture sur une base déjà conforme.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"levelup/go-api/internal/sync/matchflags"
)

const (
	integriteSuffix           = "__integrite"
	backfillCompletedColonne  = "backfill_completed"
	colonneParticipantsLoaded = "participants_loaded"
	colonnePlayerCount        = "player_count"
)

// colonnesIntegriteRetirees : les colonnes de match_registry que la reconstruction retire.
var colonnesIntegriteRetirees = []string{colonneParticipantsLoaded, colonnePlayerCount}

// vuesCanoniquesDuRegistre : les vues dont le DDL vit dans le code ; jouées tel quel après
// l'échange. Toute autre vue qui lit match_registry est rejouée depuis son DDL stocké.
var vuesCanoniquesDuRegistre = map[string]string{
	"mv_player_matches": mvPlayerMatchesViewSQL,
	"v_match_full":      VMatchFullViewSQL,
}

func init() {
	Register(Migration{
		Name:     "shared_registry_drop_integrity_columns_v1",
		TargetDB: TargetShared,
		Description: "Retire match_registry.participants_loaded, player_count et idx_registry_player_count " +
			"(swap sans UPDATE, vues rejouées)",
		ApplySchema: func(db *sql.DB) error { return applyRegistryIntegrity(db, false) },
	})
}

// ApplyRegistryParticipantsBitRebuild : la même reconstruction, qui pose en plus le bit
// participants sur chaque match qui a des lignes dans match_participants. Réservée au step
// title-owned des titres dont le moteur de synchronisation écrit ce bit.
func ApplyRegistryParticipantsBitRebuild(db *sql.DB) error { return applyRegistryIntegrity(db, true) }

// planIntegrite : ce que la reconstruction doit faire sur une base donnée.
type planIntegrite struct {
	retirees    []string // colonnes à retirer présentes dans la table
	poserBit    bool     // backfill_completed reçoit le bit participants
	sansBit     int64    // matchs à roster sans le bit (si poserBit)
	indexGardes []string // DDL des index reposés après l'échange
	indexOtes   []string // DDL des index retirés (couvrent une colonne retirée)
}

func applyRegistryIntegrity(db *sql.DB, bitDemande bool) error {
	ctx := bootCtx()
	if err := recoverOrphanTable(ctx, db, registryTable, integriteSuffix); err != nil {
		return err
	}
	if ok, err := tableExists(db, registryTable); err != nil || !ok {
		return err
	}
	p, err := planifierIntegrite(ctx, db, bitDemande)
	if err != nil {
		return err
	}
	if len(p.retirees) == 0 && p.sansBit == 0 {
		return nil
	}
	n, err := reconstruireRegistre(ctx, db, p)
	if err != nil {
		return fmt.Errorf("reconstruction de %s (colonnes d'intégrité): %w", registryTable, err)
	}
	slog.WarnContext(ctx, "migration: match_registry reconstruite (swap, sans UPDATE)",
		"colonnes_retirees", p.retirees, "index_retires", p.indexOtes,
		"bit_participants_pose", p.sansBit, "lignes", n.Rebuilt)
	return nil
}

func planifierIntegrite(ctx context.Context, db *sql.DB, bitDemande bool) (planIntegrite, error) {
	var p planIntegrite
	for _, col := range colonnesIntegriteRetirees {
		has, err := columnExists(db, registryTable, col)
		if err != nil {
			return p, fmt.Errorf("%s: colonne %s: %w", registryTable, col, err)
		}
		if has {
			p.retirees = append(p.retirees, col)
		}
	}
	if bitDemande {
		bit, err := bitApplicable(db)
		if err != nil {
			return p, err
		}
		p.poserBit = bit
	}
	if p.poserBit {
		n, err := matchsSansBit(ctx, db)
		if err != nil {
			return p, err
		}
		p.sansBit = n
	}
	index, err := ddlDesIndex(ctx, db, registryTable)
	if err != nil {
		return p, err
	}
	p.indexGardes, p.indexOtes = partagerIndex(index, p.retirees)
	return p, nil
}

// bitApplicable : le bit ne se pose que si la table le porte et que le roster existe.
func bitApplicable(db *sql.DB) (bool, error) {
	has, err := columnExists(db, registryTable, backfillCompletedColonne)
	if err != nil || !has {
		return false, err
	}
	return tableExists(db, "match_participants")
}

// matchsSansBit : matchs qui ont des participants et pas le bit.
func matchsSansBit(ctx context.Context, q lecteurSQL) (int64, error) {
	var n int64
	err := q.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM match_registry
		WHERE match_id IN (SELECT match_id FROM match_participants)
		AND (COALESCE(backfill_completed, 0) & %d) = 0`, matchflags.BackfillParticipants)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("%s: matchs sans bit participants: %w", registryTable, err)
	}
	return n, nil
}

// projectionIntegrite : la table attendue après l'échange, en SELECT sur la table actuelle.
func projectionIntegrite(p planIntegrite) string {
	sel := "*"
	if len(p.retirees) > 0 {
		sel += " EXCLUDE (" + strings.Join(p.retirees, ", ") + ")"
	}
	if p.poserBit {
		sel += fmt.Sprintf(` REPLACE (CASE WHEN match_id IN (SELECT match_id FROM match_participants)
			THEN COALESCE(backfill_completed, 0) | %d ELSE backfill_completed END AS backfill_completed)`,
			matchflags.BackfillParticipants)
	}
	return "SELECT " + sel + " FROM " + registryTable
}

// avantIntegrite : ce que les gardes comparent après l'échange.
type avantIntegrite struct {
	colonnes    []string
	contraintes []string
	index       []string
	attendue    EmpreinteVue
	vues        map[string]EmpreinteVue
}

// buildIntegriteSupplementaire : couture d'injection des tests, statements joués après le
// remplissage de la table de construction. Vide en production ; un test y glisse une altération
// pour prouver que les gardes annulent l'échange.
var buildIntegriteSupplementaire = func() []string { return nil }

func reconstruireRegistre(ctx context.Context, db *sql.DB, p planIntegrite) (swapCounts, error) {
	ddl, err := ddlDeTable(ctx, db, registryTable)
	if err != nil {
		return swapCounts{}, err
	}
	creer, err := ddlDeConstructionSuffixe(ddl, registryTable, integriteSuffix)
	if err != nil {
		return swapCounts{}, err
	}
	vues, err := vuesDuRegistre(ctx, db)
	if err != nil {
		return swapCounts{}, err
	}
	avant, err := mesurerAvantIntegrite(ctx, db, p, vues)
	if err != nil {
		return swapCounts{}, err
	}
	scratch := registryTable + integriteSuffix
	build := []string{creer}
	for _, col := range p.retirees {
		build = append(build, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, scratch, col))
	}
	build = append(build, fmt.Sprintf(`INSERT INTO %s BY NAME %s`, scratch, projectionIntegrite(p)))
	build = append(build, buildIntegriteSupplementaire()...)
	return swapTableTx(ctx, db, tableSwap{
		Table:      registryTable,
		Suffix:     integriteSuffix,
		Expected:   `SELECT COUNT(*) FROM match_registry`,
		Build:      build,
		PostRename: append(append([]string(nil), p.indexGardes...), rejouerVues(vues)...),
		Verify: func(ctx context.Context, tx *sql.Tx) error {
			return verifierIntegrite(ctx, tx, p, avant)
		},
	})
}

// partagerIndex sépare les index gardés de ceux qui couvrent une colonne retirée.
func partagerIndex(index, retirees []string) (gardes, otes []string) {
	for _, ddl := range index {
		if couvreUneColonne(ddl, retirees) {
			otes = append(otes, ddl)
			continue
		}
		gardes = append(gardes, ddl)
	}
	return gardes, otes
}

func couvreUneColonne(ddl string, cols []string) bool {
	bas := strings.ToLower(ddl)
	for _, col := range cols {
		if strings.Contains(bas, col) {
			return true
		}
	}
	return false
}

// vueDuRegistre : une vue qui lit match_registry et son DDL stocké.
type vueDuRegistre struct{ nom, sql string }

func vuesDuRegistre(ctx context.Context, q lecteurSQL) ([]vueDuRegistre, error) {
	rows, err := q.QueryContext(ctx, `SELECT view_name, sql FROM duckdb_views()
		WHERE NOT internal AND schema_name = 'main' AND sql ILIKE '%match_registry%' ORDER BY view_name`)
	if err != nil {
		return nil, fmt.Errorf("vues de %s: %w", registryTable, err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []vueDuRegistre
	for rows.Next() {
		var v vueDuRegistre
		if err := rows.Scan(&v.nom, &v.sql); err != nil {
			return nil, fmt.Errorf("vues de %s: %w", registryTable, err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// rejouerVues : le DDL qui recrée chaque vue après l'échange — canonique si le code le porte,
// sinon le DDL stocké (une vue stockée qui cite une colonne retirée fait échouer l'échange).
func rejouerVues(vues []vueDuRegistre) []string {
	out := make([]string, 0, len(vues))
	for _, v := range vues {
		if ddl, ok := vuesCanoniquesDuRegistre[v.nom]; ok {
			out = append(out, ddl)
			continue
		}
		out = append(out, strings.Replace(v.sql, "CREATE VIEW ", "CREATE OR REPLACE VIEW ", 1))
	}
	return out
}

// horsColonnesDesVues : colonnes ignorées par l'empreinte des vues (celles qui changent).
func (p planIntegrite) horsColonnesDesVues() []string {
	return append(append([]string(nil), colonnesIntegriteRetirees...), backfillCompletedColonne)
}

func mesurerAvantIntegrite(ctx context.Context, db *sql.DB, p planIntegrite,
	vues []vueDuRegistre) (avantIntegrite, error) {
	a := avantIntegrite{index: p.indexGardes}
	var err error
	if a.colonnes, err = colonnesDuRegistre(ctx, db, p.retirees); err != nil {
		return a, err
	}
	if a.contraintes, err = contraintesDuRegistre(ctx, db, p.retirees); err != nil {
		return a, err
	}
	if a.attendue, err = empreinteDeSelection(ctx, db, registryTable, projectionIntegrite(p)); err != nil {
		return a, err
	}
	a.vues = make(map[string]EmpreinteVue, len(vues))
	for _, v := range vues {
		e, err := empreinteHorsColonnes(ctx, db, v.nom, p.horsColonnesDesVues())
		if err != nil {
			return a, err
		}
		a.vues[v.nom] = e
	}
	return a, nil
}

func verifierIntegrite(ctx context.Context, tx *sql.Tx, p planIntegrite, avant avantIntegrite) error {
	if err := verifierStructure(ctx, tx, avant); err != nil {
		return err
	}
	apres, err := empreinteDeSelection(ctx, tx, registryTable, `SELECT * FROM match_registry`)
	if err != nil {
		return err
	}
	if apres != avant.attendue {
		return fmt.Errorf("%s ne rend pas la projection attendue : attendu %+v, obtenu %+v",
			registryTable, avant.attendue, apres)
	}
	for nom, want := range avant.vues {
		got, err := empreinteHorsColonnes(ctx, tx, nom, p.horsColonnesDesVues())
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("vue %s changée hors colonnes d'intégrité : avant %+v, après %+v", nom, want, got)
		}
	}
	if p.poserBit {
		if n, err := matchsSansBit(ctx, tx); err != nil || n != 0 {
			return fmt.Errorf("%d match(s) à roster encore sans bit participants (err %v)", n, err)
		}
	}
	return nil
}

// verifierStructure : colonnes, contraintes et index identiques à l'état d'avant, colonnes
// retirées exceptées.
func verifierStructure(ctx context.Context, tx *sql.Tx, avant avantIntegrite) error {
	cols, err := colonnesDuRegistre(ctx, tx, nil)
	if err != nil {
		return err
	}
	if strings.Join(cols, "\n") != strings.Join(avant.colonnes, "\n") {
		return fmt.Errorf("colonnes de %s changées :\navant %v\naprès %v", registryTable, avant.colonnes, cols)
	}
	contraintes, err := contraintesDuRegistre(ctx, tx, nil)
	if err != nil {
		return err
	}
	if strings.Join(contraintes, "\n") != strings.Join(avant.contraintes, "\n") {
		return fmt.Errorf("contraintes de %s changées : avant %v, après %v", registryTable, avant.contraintes, contraintes)
	}
	index, err := ddlDesIndex(ctx, tx, registryTable)
	if err != nil {
		return err
	}
	if strings.Join(index, "\n") != strings.Join(avant.index, "\n") {
		return fmt.Errorf("index de %s changés : attendu %v, obtenu %v", registryTable, avant.index, index)
	}
	return nil
}

// colonnesDuRegistre : `nom type défaut nullité` de chaque colonne, dans l'ordre de la table,
// sans les colonnes `hors`.
func colonnesDuRegistre(ctx context.Context, q lecteurSQL, hors []string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT column_name, data_type, COALESCE(column_default, ''), is_nullable
		FROM duckdb_columns() WHERE schema_name = 'main' AND table_name = ? ORDER BY column_index`, registryTable)
	if err != nil {
		return nil, fmt.Errorf("colonnes de %s: %w", registryTable, err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []string
	for rows.Next() {
		var nom, typ, defaut string
		var nullable bool
		if err := rows.Scan(&nom, &typ, &defaut, &nullable); err != nil {
			return nil, fmt.Errorf("colonnes de %s: %w", registryTable, err)
		}
		if slices.Contains(hors, nom) {
			continue
		}
		out = append(out, fmt.Sprintf("%s %s default=%s nullable=%t", nom, typ, defaut, nullable))
	}
	return out, rows.Err()
}

// contraintesDuRegistre : type et colonnes de chaque contrainte, triés, sans celles qui portent
// sur une colonne `hors`.
func contraintesDuRegistre(ctx context.Context, q lecteurSQL, hors []string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT constraint_type, array_to_string(constraint_column_names, ',')
		FROM duckdb_constraints() WHERE schema_name = 'main' AND table_name = ? ORDER BY 1, 2`, registryTable)
	if err != nil {
		return nil, fmt.Errorf("contraintes de %s: %w", registryTable, err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []string
	for rows.Next() {
		var typ, cols string
		if err := rows.Scan(&typ, &cols); err != nil {
			return nil, fmt.Errorf("contraintes de %s: %w", registryTable, err)
		}
		if couvreUneColonne(cols, hors) {
			continue
		}
		out = append(out, typ+" ("+cols+")")
	}
	return out, rows.Err()
}
