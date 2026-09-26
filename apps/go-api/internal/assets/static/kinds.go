// Package static — composition d'URLs et de chemins FS pour les assets
// statiques servis sous /static/, title-scopés.
//
// Ce package est volontairement pur :
//   - aucune connaissance d'un titre particulier (slug en paramètre, jamais en const) ;
//   - aucune connaissance de naming convention (encodage map names, format CSR rank, etc.) ;
//   - zéro I/O, zéro dépendance externe.
//
// La connaissance title-spécifique (encodage, fallbacks UUID) vit dans la
// couche 3 — `internal/games/{title}/adapter_asset_urls.go` — qui implémente
// `games.TitleAssetURLAdapter` et délègue ici la composition path/URL.
package static

// Kind identifie un type d'asset statique servi sous /static/.
//
// Chaque Kind correspond à un sous-dossier sous /static/ (cf. Folder).
// L'arborescence title-scopée est : /static/{folder}/{titleSlug}/{id}{ext}.
type Kind string

const (
	KindMap          Kind = "map"
	KindMedal        Kind = "medal"
	KindCSRRank      Kind = "csr-rank"
	KindWeapon       Kind = "weapon"
	KindCommendation Kind = "commendation"
	KindVehicle      Kind = "vehicle"
)

// Valid retourne vrai si k est un Kind connu de ce package.
func (k Kind) Valid() bool {
	switch k {
	case KindMap, KindMedal, KindCSRRank, KindWeapon, KindCommendation, KindVehicle:
		return true
	}
	return false
}
