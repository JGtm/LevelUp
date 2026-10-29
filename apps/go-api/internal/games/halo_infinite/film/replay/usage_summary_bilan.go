package replay

// usage_summary_bilan.go — LE PERIMETRE DU BILAN D'EQUIPEMENT ET LE CANAL DE SON COTE « UTILISE »,
// COPIE JUMELLE DE `internal/domain/equipmentusage` (plan D2, item 2.8 ; regle 6 de CLAUDE.md).
//
// Le decodeur de film est une brique qui n'importe rien de LevelUp. L'agrégat de session
// (`analysis/sessionusage`), la page Escouade (`analysis/squademprise`) et les séries temporelles
// (`analysis/trends`) lisent la MEME connaissance dans `domain/equipmentusage`, qui reste dans
// LevelUp. Deux copies, donc un garde-rail qui refuse qu'elles divergent :
// `domain/equipmentusage/jumeau_film_test.go` compare les deux listes du bilan (ordre compris) et
// le prédicat de pièce engendrée sur toutes les familles connues des deux côtés.
//
// CE QUI EST COPIÉ, ET RIEN DE PLUS : la liste ordonnée des familles du bilan, la table des
// familles à pièce engendrée et les deux fonctions qui les servent. Les clés de famille (mur,
// capteur, bonus, grappin, propulseur) sont des constantes du paquet
// (`usage_summary_families.go`) ; leurs valeurs sont celles du manifeste
// (`config/titles/halo_infinite/mappings/replay_labels.toml`), et le garde-rail
// `usage_summary_families_guard_test.go` les recolle à la liste fermée du valideur.

// equipmentOutcomeFamilies — LES FAMILLES QUI PORTENT UNE LIGNE D'ISSUE, dans l'ordre où le
// bilan les cite. Une LISTE et non une map : l'agrégat de session publie ces familles dans cet
// ordre.
//
// C'est le PÉRIMÈTRE DU BILAN, pas une table de reconnaissance (la famille est publiée par le
// document depuis le schéma 51) : une famille du manifeste absente d'ici reste hors bilan — le
// répulseur, le grappin et le propulseur portent une famille, ils n'ont aucune ligne « pris /
// utilisé / gardé / lâché ».
var equipmentOutcomeFamilies = []string{
	usageFamilyWall,
	usageFamilySensor,
	familleDuTranslocateur,
	"shroud_screen",
	"threat_seeker",
	usageFamilyRepairField,
	usageFamilyPowerupCamo,
	usageFamilyPowerupOvershield,
}

// usageFamiliesWithSpawnedPiece — LES FAMILLES QUI ENGENDRENT UNE PIÈCE DISTINCTE, seules dont le
// canal des POSES voit le déploiement. Transcription d'une donnée écrite : les familles dont le
// manifeste porte au moins un objet `kind = "deployed"` (aujourd'hui les deux panneaux du mur).
// Recollée au manifeste par `TestUsageFamiliesWithSpawnedPieceMatchManifest`.
//
// Une pose `origin: deployed` sur un objet PORTÉ mesure un LÂCHER volontaire, pas un
// déploiement (rapport E0 du 2026-09-10, question 5) : c'est pourquoi la table est fermée.
var usageFamiliesWithSpawnedPiece = map[string]bool{
	usageFamilyWall: true,
}

// EquipmentOutcomeFamilies rend les familles qui portent une ligne d'issue, dans l'ordre du bilan
// (copie défensive : l'appelant peut la modifier).
func EquipmentOutcomeFamilies() []string {
	out := make([]string, len(equipmentOutcomeFamilies))
	copy(out, equipmentOutcomeFamilies)
	return out
}

// UsageFamilySpawnsPiece dit si le canal des POSES voit le déploiement de cette famille — donc si
// son côté « utilisé » se lit sur `deployed` (mur) ou sur les CONSOMMATIONS (tout le reste).
func UsageFamilySpawnsPiece(family string) bool {
	return usageFamiliesWithSpawnedPiece[family]
}
