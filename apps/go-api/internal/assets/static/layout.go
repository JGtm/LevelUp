package static

// MountPoint est le préfixe HTTP sous lequel les assets statiques sont servis.
const MountPoint = "/static"

// Folder retourne le sous-dossier sous /static/ pour un Kind donné.
//
// Centralise la convention des sous-dossiers (medals → "medals",
// weapons → "weapons-assets", etc.) pour qu'aucun caller n'ait à connaître le mapping.
//
// Retourne "" si k n'est pas un Kind valide.
func Folder(k Kind) string {
	switch k {
	case KindMap:
		return "maps"
	case KindMedal:
		return "medals"
	case KindCSRRank:
		return "ranks"
	case KindWeapon:
		return "weapons-assets"
	case KindCommendation:
		return "commendations"
	case KindVehicle:
		return "vehicles-assets"
	}
	return ""
}
