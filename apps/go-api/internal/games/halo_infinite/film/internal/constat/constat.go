// Package constat porte les DIAGNOSTICS du decodeur de film : ce que les couches `grammar` et
// `facts` constatent, rendu a l orchestrateur qui les journalise (ADR 0034 D-4, lot J12.3 du
// 2026-09-30).
//
// # LA REGLE
//
// `grammar` et `facts` ne journalisent pas : ils n ont pas le contexte de l appelant, et une ligne
// emise sous `context.Background()` perd tout ce que le contexte porte (requete, cycle de
// synchronisation). Ce qu ils journalisaient devient un [Diagnostic] : une VALEUR, notee dans un
// [Diagnostics] que l appelant a ouvert, et que l orchestrateur (`film/replay`, `replaybuild`,
// `sync/killcollector`) journalise avec SON `ctx` (`replay.JournaliserDiagnostics`, seul endroit
// du decodeur ou un diagnostic devient une ligne de journal). Garde-rail :
// `archlint/film_no_slog_in_layers_test.go`.
//
// # CE QU UN DIAGNOSTIC NE FAIT PAS
//
// Il n entre dans AUCUNE structure serialisee : ni le document de rejeu, ni les faits persistes,
// ni le resultat killsource tel qu il se compare. Les comptes qui DOIVENT se publier voyagent deja
// comme donnees (couverture, rapport de replis) ; un diagnostic n en est que le recit.
//
// # POURQUOI UN PAQUET A PART
//
// Une FEUILLE sans import du depot, importable par toutes les couches : `grammar` le nomme, et
// `facts/objectives` aussi — or des tests internes de `grammar` importent `objectives`, donc le
// type ne pouvait vivre ni dans `grammar` (cycle de test) ni dans `film/types` (types de donnees
// SANS methode, et l accumulateur en porte).
//
// # POURQUOI LES NIVEAUX VALENT CEUX DE `log/slog`
//
// L orchestrateur convertit par `slog.Level(d.Niveau)` : les quatre constantes reprennent les
// valeurs de `slog.LevelDebug` .. `slog.LevelError`, epinglees par
// `replay.TestNiveauxDesConstatsEgalentSlog`.
package constat

import "sync"

// Niveau est la gravite d un diagnostic ; ses valeurs sont celles de `slog.Level`.
type Niveau int

// Les quatre niveaux, aux valeurs de `slog.LevelDebug`, `LevelInfo`, `LevelWarn`, `LevelError`.
const (
	NiveauDebug Niveau = -4
	NiveauInfo  Niveau = 0
	NiveauWarn  Niveau = 4
	NiveauError Niveau = 8
)

// Code nomme un diagnostic de facon STABLE : c est lui que l on greppe et que l on teste, jamais
// le message. Chaque couche declare les siens (`grammar.Diag*`, `killsource.Diag*`, ...).
type Code string

// Diagnostic est UN constat d une couche de decodage, a journaliser par l orchestrateur.
//
// `Attrs` suit la convention de `log/slog` : des paires cle, valeur.
type Diagnostic struct {
	Code    Code
	Niveau  Niveau
	Message string
	Attrs   []any
}

// Diagnostics accumule les diagnostics d UN decodage. Sur par concurrence ; un recepteur nil ne
// note rien (instruments et tests qui n en ont pas l usage — jamais la production, dont chaque
// orchestrateur ouvre le sien).
type Diagnostics struct {
	mu sync.Mutex
	ds []Diagnostic
}

// Signaler note un diagnostic.
func (j *Diagnostics) Signaler(d Diagnostic) {
	if j == nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ds = append(j.ds, d)
}

// Relever rend les diagnostics notes, dans l ordre, et VIDE l accumulateur : un diagnostic se
// journalise une fois, quel que soit le nombre de releves.
func (j *Diagnostics) Relever() []Diagnostic {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	out := j.ds
	j.ds = nil
	return out
}

// Verser deplace les diagnostics de `j` dans `dst` (les deux peuvent etre nil).
func (j *Diagnostics) Verser(dst *Diagnostics) {
	for _, d := range j.Relever() {
		dst.Signaler(d)
	}
}
