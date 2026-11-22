package mapfondsidecar

import (
	"path/filepath"
	"strings"
)

// suffixeRelief sépare la clé de l'extension dans le nom des deux fichiers du relief d'un fond
// (`<cle>.relief.webp`, `<cle>.relief.json.gz`) : le relief se pose sur le fond avec le même calage et
// ne s'en distingue que par ce suffixe.
const suffixeRelief = ".relief"

// extensionDonneesRelief est l'extension du fichier des données du relief : du JSON compressé en
// gzip. Elle ne finit PAS en `.json` : un lecteur qui globe `*.json` dans le dossier des fonds (index
// des fonds du décodeur, tests des fonds publiés) prendrait des données de relief pour un sidecar.
const extensionDonneesRelief = ".json.gz"

// NomImageRelief rend le nom du fichier image du relief d'une clé de fond pour l'extension `ext`
// (point compris : `.png`, `.webp`).
func NomImageRelief(cle, ext string) string {
	return cle + suffixeRelief + ext
}

// NomDonneesRelief rend le nom du fichier des données du relief d'une clé de fond
// (`<cle>.relief.json.gz`, JSON compressé en gzip).
func NomDonneesRelief(cle string) string {
	return cle + suffixeRelief + extensionDonneesRelief
}

// CleDeImageRelief rend la clé de fond dont `nomFichier` (sans dossier) est l'image du relief, et dit
// si c'en est une : `<cle>.relief.png` ou `<cle>.relief.webp`. Les données (`.relief.json.gz`) n'en
// sont pas une. Seul point qui sait lire le nom d'une image de relief : l'outil de conversion en WebP s'en
// sert pour ne pas prendre un relief pour un fond.
func CleDeImageRelief(nomFichier string) (cle string, ok bool) {
	ext := strings.ToLower(filepath.Ext(nomFichier))
	if ext != ".png" && ext != ".webp" {
		return "", false
	}
	cle, ok = strings.CutSuffix(strings.TrimSuffix(nomFichier, filepath.Ext(nomFichier)), suffixeRelief)
	return cle, ok && cle != ""
}

// EstDonneesRelief dit si `nomFichier` (sans dossier) est le fichier des données du relief d'une clé :
// `<cle>.relief.json.gz`, clé non vide. Un sidecar (`<cle>.json`) n'en est pas un.
func EstDonneesRelief(nomFichier string) bool {
	cle, ok := strings.CutSuffix(nomFichier, suffixeRelief+extensionDonneesRelief)
	return ok && cle != ""
}
