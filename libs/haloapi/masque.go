package haloapi

// Masque rend ce qui peut figurer dans un journal ou une chaîne de diagnostic à la place d'un secret
// (jeton, secret client) : "" si `s` est vide, "***" sinon. Ni la longueur ni un préfixe de `s` ne
// transparaissent. Les `String` et `LogValue` des types qui portent un secret s'en servent ; le
// garde-rail `archlint/haloapi_masque_unique_test.go` interdit d'en redéclarer une copie sous
// `libs/haloapi/`.
func Masque(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}
