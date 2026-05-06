// Package duckdb — ubigint_scanner.go : helper sql.Scanner pour lire une
// colonne DuckDB UBIGINT en int64 par reinterpret bit-à-bit.
//
// Pourquoi : le driver `database/sql` standard ne supporte pas nativement
// uint64 avec bit63=1 — un Scan direct vers `*int64` plante silencieusement
// (le `if err == nil` swallow l'erreur de conversion). Pour les colonnes
// UBIGINT susceptibles de dépasser INT64_MAX (notamment les hash filmshell
// `weapon_id` Halo Infinite, où bit63 est à 1 pour les armes hashées),
// scanner en uint64 puis reinterpréter en int64 préserve les bits.
//
// Le type int64 est conservé côté domaine pour rester compatible avec
// JSON, les map keys, et les tests existants. La conversion est purement
// représentationnelle (pas de perte d'info).
//
// Usage :
//
//	var id UBigint
//	rows.Scan(&id, &label)
//	weaponID := id.Int64()  // ou int64(id)
//	labels[id.Int64()] = label
package duckdb

import "fmt"

// UBigint est un sql.Scanner pour les colonnes DuckDB UBIGINT (uint64)
// stockées côté Go en int64 (bit-preserving reinterpret).
//
// Implémente database/sql/driver.Valuer si jamais on veut écrire en sortie,
// mais le cas d'usage ici est uniquement lecture.
type UBigint int64

// Scan implémente sql.Scanner. Accepte uint64 (DuckDB UBIGINT natif),
// int64 (DuckDB BIGINT, par sécurité) et nil (NULL → 0).
func (u *UBigint) Scan(src any) error {
	if src == nil {
		*u = 0
		return nil
	}
	switch v := src.(type) {
	case uint64:
		*u = UBigint(v) //nolint:gosec // bit-preserving reinterpret
	case int64:
		*u = UBigint(v)
	case []byte:
		// fallback : certains drivers convertissent les grands entiers en bytes.
		return fmt.Errorf("UBigint.Scan: unsupported []byte source (raw=%q)", v)
	default:
		return fmt.Errorf("UBigint.Scan: unsupported type %T", src)
	}
	return nil
}

// Int64 retourne la valeur scannée en int64 (bit-preserving reinterpret).
func (u UBigint) Int64() int64 { return int64(u) }

// NullableUBigint est la variante nullable (équivalent sql.NullInt64).
//
// Usage :
//
//	var id NullableUBigint
//	rows.Scan(&id, ...)
//	if id.Valid { use(id.Value.Int64()) }
type NullableUBigint struct {
	Value UBigint
	Valid bool
}

// Scan implémente sql.Scanner pour la variante nullable.
func (n *NullableUBigint) Scan(src any) error {
	if src == nil {
		n.Valid = false
		n.Value = 0
		return nil
	}
	if err := n.Value.Scan(src); err != nil {
		n.Valid = false
		return err
	}
	n.Valid = true
	return nil
}
