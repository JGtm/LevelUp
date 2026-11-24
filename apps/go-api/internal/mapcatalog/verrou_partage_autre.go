//go:build !windows

package mapcatalog

// partageRefuse : le partage refusé est propre à Windows ; ailleurs, un verrou tenu rend « existe déjà ».
func partageRefuse(error) bool { return false }
