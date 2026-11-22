package migrations

// medal_hard_target_test.go — la médaille custom « Cible difficile » (9000000002) : définition
// comme le Vengeur (custom, catégorie skill, rareté Normal) et textes par langue dans
// medal_translations, sur une base neuve comme sur une base dont la migration des textes a déjà
// retiré les colonnes `_fr` / `_en`.

import (
	"database/sql"
	"reflect"
	"testing"

	"levelup/go-api/internal/i18n"
)

func TestHardTarget_BaseNeuveEtRejeu(t *testing.T) {
	db := ouvrirMemoire(t)
	jouerMetadata(t, db)
	assertHardTarget(t, db)

	jouerMetadata(t, db)
	assertHardTarget(t, db)
}

// TestHardTarget_SurBaseDontLesTextesSontDejaMigres : le step seul, sur une base qui porte déjà
// la table sans colonnes de texte (production) : il ne dépend d'aucune colonne `_fr` / `_en`.
func TestHardTarget_SurBaseDontLesTextesSontDejaMigres(t *testing.T) {
	db := ouvrirMemoire(t)
	jouerMetadata(t, db)
	if _, err := db.Exec(`DELETE FROM medal_definitions WHERE medal_name_id = 9000000002;
		DELETE FROM medal_translations WHERE medal_name_id = 9000000002`); err != nil {
		t.Fatalf("retrait: %v", err)
	}
	for _, m := range medalHardTargetSteps() {
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("%s sur base migree: %v", m.Name, err)
		}
	}
	assertHardTarget(t, db)
}

func assertHardTarget(t *testing.T, db *sql.DB) {
	t.Helper()
	var n int
	var custom bool
	var difficulty, mtype string
	if err := db.QueryRow(`SELECT COUNT(*), BOOL_AND(is_custom), MIN(difficulty), MIN(medal_type)
		FROM medal_definitions WHERE medal_name_id = ?`, idMedailleCibleDifficile).Scan(&n, &custom, &difficulty, &mtype); err != nil {
		t.Fatalf("definition: %v", err)
	}
	if n != 1 || !custom || difficulty != "Normal" || mtype != "skill" {
		t.Fatalf("definition = (%d lignes, custom %v, %q, %q), attendu 1 / true / Normal / skill", n, custom, difficulty, mtype)
	}
	var vDiff, vType string
	if err := db.QueryRow(`SELECT difficulty, medal_type FROM medal_definitions WHERE medal_name_id = 9000000001`).
		Scan(&vDiff, &vType); err != nil {
		t.Fatalf("vengeur: %v", err)
	}
	if vDiff != difficulty || vType != mtype {
		t.Errorf("categorie et rarete (%q, %q) diffèrent de celles du Vengeur (%q, %q)", difficulty, mtype, vDiff, vType)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM medal_translations WHERE medal_name_id = ?`,
		idMedailleCibleDifficile).Scan(&n); err != nil {
		t.Fatalf("count medal_translations: %v", err)
	}
	if n != 2 {
		t.Errorf("medal_translations : %d lignes, attendu 2 (fr et en)", n)
	}
	names, descriptions := medalTextsByLanguage(t, db, idMedailleCibleDifficile)
	wantNames := i18n.FromColumns("Hard Target", "Cible difficile")
	wantDescs := i18n.FromColumns("Fully recharge your shield after it was broken.",
		"Rechargez complètement votre bouclier après qu'il a été brisé.")
	if !reflect.DeepEqual(names, wantNames) || !reflect.DeepEqual(descriptions, wantDescs) {
		t.Errorf("textes = %v / %v, attendu %v / %v", names, descriptions, wantNames, wantDescs)
	}
}
