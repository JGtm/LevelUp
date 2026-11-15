package main

// decisions.go — LES FAUX ACCEPTÉS PAR L'UTILISATEUR, INSCRITS DANS LE MANIFESTE.
//
// Un FAUX du banc que l'utilisateur accepte (une lecture qui change exprès) ne doit pas laisser le
// gate rouge à chaque passage, ni ne vivre que dans la description d'une demande de fusion. Il
// s'inscrit dans `config/replay_corpus.toml`, section `[[decision]]` : la mesure du banc, les témoins
// concernés (aucun = tous), la version du décodeur qui l'introduit, la date et la raison. Le gate
// requalifie alors le constat « accepté » ([replayverite.StatutAccepte]) au lieu de FAUX. La mesure
// peut aussi nommer une feuille du document (`coverage.shots.aim.record`) : un FILET sur cette feuille
// sort des pertes bloquantes et s'imprime « accepté ».
//
// UNE DÉCISION QUI NE SERT PLUS SE RETIRE : quand la référence du gate porte déjà le changement, le
// FAUX disparaît et la décision ne requalifie plus rien. Le gate le dit, pour que la liste ne
// grossisse pas.

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/replayverite"
)

// Decision est UNE acceptation de l'utilisateur.
type Decision struct {
	// Mesure : la mesure du banc, telle que le rapport la nomme (« R-1 repli <nom> », un score...).
	Mesure string `toml:"mesure"`
	// Temoins : les témoins concernés ; vide = tous.
	Temoins []string `toml:"temoins"`
	// Decodeur : la version du décodeur qui introduit le changement.
	Decodeur string `toml:"decodeur"`
	// Date : la date de la décision de l'utilisateur (AAAA-MM-JJ).
	Date string `toml:"date"`
	// Raison : ce qui a été accepté, et pourquoi.
	Raison string `toml:"raison"`
}

// valider refuse une décision incomplète (sans mesure, décodeur, date ou raison, elle ne dit pas ce
// qu'elle accepte ni qui l'a décidé) et une décision qui nomme un témoin hors du manifeste.
func (d Decision) valider(i int, temoins map[string]bool) error {
	if d.Mesure == "" || d.Date == "" || d.Raison == "" || d.Decodeur == "" {
		return fmt.Errorf("decision #%d : mesure, decodeur, date et raison sont obligatoires", i)
	}
	for _, id := range d.Temoins {
		if !temoins[id] {
			return fmt.Errorf("decision #%d : temoin %q absent du manifeste", i, id)
		}
	}
	return nil
}

// concerne dit si la décision vaut pour le témoin `id`.
func (d Decision) concerne(id string) bool {
	return len(d.Temoins) == 0 || slices.Contains(d.Temoins, id)
}

// motif est le détail ajouté au constat accepté.
func (d Decision) motif() string {
	return fmt.Sprintf("accepte (decision du %s, decodeur %s) : %s", d.Date, d.Decodeur, d.Raison)
}

// decisionsAppliquees : pour chaque décision du manifeste, le nombre de constats qu'elle a
// requalifiés sur l'exécution. Partagé par les témoins (une carte : le contexte se copie).
type decisionsAppliquees map[int]int

// appliquerLesDecisions requalifie les FAUX acceptés du témoin `id` et compte ce que chaque
// décision a servi.
func appliquerLesDecisions(c *replayverite.Comparaison, id string, ds []Decision, vues decisionsAppliquees) {
	if c == nil {
		return
	}
	for i, d := range ds {
		if d.concerne(id) {
			vues[i] += c.Accepter(d.Mesure, d.motif())
		}
	}
}

// accepterLesFilets sort des filets bloquants de `l` ceux dont la feuille est la mesure d'une
// décision qui concerne le témoin ; ils restent au rapport, sous « accepté ».
//
// LE NOMBRE DE REPLIS SUIT SES REPLIS ACCEPTÉS : `coverage.fallbacks/n` monte d'un nom par repli
// nouveau de la liste. Une hausse égale au nombre de replis nouveaux que le banc a acceptés sur ce
// témoin n'est que leur arrivée ([hausseDesReplisAcceptes]) ; toute autre hausse reste bloquante.
func accepterLesFilets(l *ligneRapport, ds []Decision, vues decisionsAppliquees) {
	gardes := l.Filets[:0]
	for _, f := range l.Filets {
		accepte := f.Metrique == metriqueNombreDeReplis && hausseDesReplisAcceptes(l, f)
		for i, d := range ds {
			if d.concerne(l.Temoin.ID) && d.Mesure == f.Metrique {
				vues[i]++
				accepte = true
			}
		}
		if accepte {
			l.FiletsAcceptes = append(l.FiletsAcceptes, f)
			continue
		}
		gardes = append(gardes, f)
	}
	l.Filets = gardes
}

// metriqueNombreDeReplis : la feuille qui compte les noms de la liste des replis du document.
const metriqueNombreDeReplis = "coverage.fallbacks/n"

// hausseDesReplisAcceptes dit si la hausse du filet `f` (le nombre de replis) est exactement le
// nombre de replis passés de zéro à un compte non nul que le banc a acceptés sur ce témoin.
func hausseDesReplisAcceptes(l *ligneRapport, f replaydiff.Difference) bool {
	if l.Verite == nil {
		return false
	}
	avant, okA := nombreDEcart(f.Ancien)
	apres, okB := nombreDEcart(f.Nouveau)
	if !okA || !okB || apres <= avant {
		return false
	}
	nouveaux := 0
	for _, c := range l.Verite.Constats {
		if c.Sens == replayverite.StatutAccepte && strings.HasPrefix(c.Mesure, prefixeR1) && c.Avant == "0" {
			nouveaux++
		}
	}
	return nouveaux > 0 && apres-avant == float64(nouveaux)
}

// prefixeR1 : le début du nom d'une mesure R-1 du banc (« R-1 repli <nom> »).
const prefixeR1 = "R-1 repli "

// executionComplete dit si les décisions ont été mises à l'épreuve de TOUT le corpus : manifeste non
// filtré, chaque témoin cuit et comparé. Sinon, une décision qui n'a rien requalifié n'est pas
// « sans effet ».
func executionComplete(lignes []ligneRapport, manifest Manifest, filtre bool) bool {
	if filtre || len(lignes) != len(manifest.Temoins) {
		return false
	}
	for _, l := range lignes {
		if l.Absent || l.Erreur != nil {
			return false
		}
	}
	return true
}

// signalerLesDecisionsSansEffet journalise les décisions qui n'ont rien requalifié sur une exécution
// complète ([executionComplete]) : la référence porte déjà le changement, ou la mesure a changé de nom. Elles sont à
// retirer du manifeste.
func signalerLesDecisionsSansEffet(ctx context.Context, ds []Decision, vues decisionsAppliquees) {
	for i, d := range ds {
		if vues[i] == 0 {
			slog.WarnContext(ctx, "replay-corpus-gate: decision sans effet — a retirer du manifeste",
				"mesure", d.Mesure, "date", d.Date, "decodeur", d.Decodeur)
		}
	}
}
