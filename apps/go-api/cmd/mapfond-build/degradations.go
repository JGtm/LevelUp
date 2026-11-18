package main

import (
	"regexp"
	"sort"
	"strings"
)

// Un chemin absolu du poste de cuisson (lecteur Windows, partage réseau, racine Unix) ne doit jamais
// entrer dans un fond de carte publié : il révèle le poste et change d'un poste à l'autre.
//
// Un dossier peut porter des espaces et des points (« Halo Infinite », « Jean D. ») mais jamais « : » ni
// séparateur ; il se termine par un séparateur, ce qui distingue le dossier du nom de fichier suivi d'un
// texte (« src.module vers »). La lettre de lecteur ne suit aucun caractère de mot (« https:// » n'en est
// pas un). Ce qui suit le dernier séparateur (le nom du module) est conservé. Un chemin Unix n'est reconnu
// qu'au début du texte ou après un blanc ou un guillemet, pour ne pas toucher « any/globals ».
const (
	dossierWindows = `(?:[^\\/:*?"<>|\s]+(?: [^\\/:*?"<>|\s]+)*[\\/])`
	dossierUnix    = `(?:[^\s/:"'()]+(?: [^\s/:"'()]+)*/)`
)

var (
	cheminWindows = regexp.MustCompile(`(^|[^A-Za-z0-9])[A-Za-z]:[\\/]` + dossierWindows + `*`)
	cheminPartage = regexp.MustCompile(`\\\\` + dossierWindows + `+`)
	cheminUnix    = regexp.MustCompile(`(^|[\s"'(])/` + dossierUnix + `+`)

	separateursRacine = regexp.MustCompile(`[\\/]+`)
)

// sansCheminAbsolu rend une copie des dégradations où chaque chemin absolu est réduit au nom de son
// dernier élément (le module) ; une dégradation sans chemin est rendue telle quelle. Rend nil pour une
// liste vide, comme le champ absent d'un fond sans dégradation.
//
// Les racines que l'outil possède (installation du jeu, dépôt) sont retirées d'abord, avec le chemin qui
// les prolonge, quelle que soit leur forme (séparateurs, casse) : un dossier personnel que le motif
// générique ne reconnaîtrait pas n'est publié par aucun de ces chemins. Une racine vide ou non absolue est
// ignorée.
func sansCheminAbsolu(degradations []string, racines ...string) []string {
	if len(degradations) == 0 {
		return nil
	}
	motifs := motifsRacines(racines)
	out := make([]string, len(degradations))
	for i, d := range degradations {
		for _, m := range motifs {
			d = m.ReplaceAllString(d, "$1")
		}
		d = cheminWindows.ReplaceAllString(d, "$1")
		d = cheminPartage.ReplaceAllString(d, "")
		out[i] = cheminUnix.ReplaceAllString(d, "$1")
	}
	return out
}

// motifsRacines compile un motif par racine absolue, la plus longue d'abord (une racine contenue dans une
// autre ne doit pas en couper la queue). Le motif tolère les deux séparateurs et ignore la casse ; il
// consomme la suite de dossiers qui prolonge la racine et laisse le nom du dernier élément. Une racine suivie
// d'un caractère de mot (« Halo » dans « Halo2 ») n'est pas la racine.
func motifsRacines(racines []string) []*regexp.Regexp {
	absolues := make([]string, 0, len(racines))
	for _, r := range racines {
		r = strings.TrimRight(strings.TrimSpace(r), `\/`)
		if r != "" && estRacineAbsolue(r) {
			absolues = append(absolues, r)
		}
	}
	sort.SliceStable(absolues, func(i, j int) bool { return len(absolues[i]) > len(absolues[j]) })
	motifs := make([]*regexp.Regexp, 0, len(absolues))
	for _, r := range absolues {
		parts := separateursRacine.Split(r, -1)
		for i, p := range parts {
			parts[i] = regexp.QuoteMeta(p)
		}
		corps := strings.Join(parts, `[\\/]+`)
		motifs = append(motifs, regexp.MustCompile(
			`(?i)`+corps+`(?:[\\/]+`+dossierWindows+`*|([^\w\\/]|$))`))
	}
	return motifs
}

// estRacineAbsolue reconnaît une racine de forme Windows (lecteur, partage) ou Unix, indépendamment du
// système qui exécute l'outil.
func estRacineAbsolue(r string) bool {
	if strings.HasPrefix(r, "/") || strings.HasPrefix(r, `\\`) {
		return true
	}
	return len(r) >= 2 && r[1] == ':' && (r[0]|0x20) >= 'a' && (r[0]|0x20) <= 'z'
}
