package replay

// golden_assembly_counts_test.go — deux petits compteurs de tirs pour `golden_assembly_test.go`.
//
// SORTIS le 2026-10-22 (lot detonations, correction CI) : la montee de SchemaVersion 101->103 a
// fait depasser le plafond figé de `golden_assembly_test.go` (hors de l exception ecrite en tete
// de `film_file_size_test.go`, qui ne couvre QUE `document_chronicle.go` et `structure_test.go`).
// « Un plafond ne monte pas » — DEPLACEMENT PUR, aucune ligne de logique changee.

// countShotsWithAim compte les tirs publies avec un angle de visee (`H != 0`).
func countShotsWithAim(doc ReplayDocument) int {
	n := 0
	for _, s := range doc.Shots {
		if s.H != 0 {
			n++
		}
	}
	return n
}

// countShotsWithWeapon compte les tirs publies avec une arme nommee.
func countShotsWithWeapon(doc ReplayDocument) int {
	n := 0
	for _, s := range doc.Shots {
		if s.Weapon != "" {
			n++
		}
	}
	return n
}
