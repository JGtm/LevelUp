package main

import "regexp"

// Un chemin absolu du poste de cuisson (lecteur Windows, partage réseau, racine Unix) ne doit jamais
// entrer dans un fond de carte publié : il révèle le poste et change d'un poste à l'autre.
//
// Un dossier d'un chemin Windows peut porter des espaces (« Halo Infinite ») mais jamais « : » ni
// séparateur ; seul son premier mot peut porter un point, pour qu'un nom de fichier suivi d'un texte
// (« src.module vers ») ne soit pas pris pour un dossier. La lettre de lecteur ne suit aucun caractère de
// mot (« https:// » n'en est pas un). Ce qui suit le dernier séparateur (le nom du module) est conservé.
// Un chemin Unix n'est reconnu qu'au début du texte ou après un blanc ou un guillemet, pour ne pas toucher
// « any/globals ».
const dossierWindows = `(?:[^\\/:*?"<>|\s]+(?: [^\\/:*?"<>|\s.]+)*[\\/])`

var (
	cheminWindows = regexp.MustCompile(`(^|[^A-Za-z0-9])[A-Za-z]:[\\/]` + dossierWindows + `*`)
	cheminPartage = regexp.MustCompile(`\\\\` + dossierWindows + `+`)
	cheminUnix    = regexp.MustCompile(`(^|[\s"'(])/(?:[^\s/:"'()]+/)+`)
)

// sansCheminAbsolu rend une copie des dégradations où chaque chemin absolu est réduit au nom de son
// dernier élément (le module) ; une dégradation sans chemin est rendue telle quelle. Rend nil pour une
// liste vide, comme le champ absent d'un fond sans dégradation.
func sansCheminAbsolu(degradations []string) []string {
	if len(degradations) == 0 {
		return nil
	}
	out := make([]string, len(degradations))
	for i, d := range degradations {
		d = cheminWindows.ReplaceAllString(d, "$1")
		d = cheminPartage.ReplaceAllString(d, "")
		out[i] = cheminUnix.ReplaceAllString(d, "$1")
	}
	return out
}
