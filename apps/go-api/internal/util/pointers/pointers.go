// Package pointers — helpers de conversion valeur→pointeur.
//
// Centralise le 1-liner `func xPtr(v X) *X { return &v }` qui avait été recopié
// sous plusieurs noms (strPtr, strPtrH5…) dans server/, api/handlers/,
// games/halo_5/ (H5, 2026-07-04 — leçon CLAUDE.md règle 6). Le garde-rail
// archlint/no_local_ptr_helper_test.go interdit toute nouvelle copie locale.
//
// ATTENTION : Ptr renvoie TOUJOURS un pointeur non-nil (même sur la valeur zéro).
// Pour la sémantique « nil si vide », ce n'est PAS le bon helper — voir les
// variantes locales strPtrNonEmpty (sync) / strPtrOrNil (openspartan) qui restent
// distinctes et volontairement séparées.
package pointers

// Ptr retourne un pointeur vers v. Générique : utilisable pour tout type
// (string, int, bool, structs…). Toujours non-nil.
func Ptr[T any](v T) *T { return &v }
