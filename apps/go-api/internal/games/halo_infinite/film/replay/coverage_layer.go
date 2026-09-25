package replay

// coverage_layer.go — LA COUVERTURE D UN CALQUE D EVENEMENTS (tirs, grenades, actions d objectif),
// sortie de coverage.go par DEPLACEMENT PUR au lot M4b (2026-09-24) : le fichier passait le seuil
// de 500 lignes (CLAUDE.md regle 5). La doctrine reste en tete de coverage.go.

import (
	"log/slog"
)

// rejectSampleThreshold : au-delà de cette proportion d'une catégorie de rejet, le calque
// émet un avertissement. Un log par événement noierait le journal (519 tirs sur un film) ;
// un seuil global le rend lisible tout en gardant l'alerte.
const rejectSampleThreshold = 0.10

// LayerCoverage est la couverture d'un calque : combien il a rattaché, sur combien
// existaient, et pourquoi il a écarté le reste.
type LayerCoverage struct {
	// Available est le nombre d'événements DISPONIBLES dans le film pour ce calque —
	// le dénominateur sans lequel un compte de rattachés ne se juge pas.
	Available int `json:"available"`
	// Attached est le nombre d'événements effectivement rattachés à un slot.
	Attached int `json:"attached"`
	// NoSlot : aucun slot de ce joueur n'est connu à cet instant (le pont ne le couvre pas).
	NoSlot int `json:"noSlot"`
	// Ambiguous : plusieurs slots de ce joueur couvrent l'instant (vies qui se recouvrent).
	Ambiguous int `json:"ambiguous"`
	// OutOfWindow : le slot est connu, mais aucune position n'est répliquée assez près de
	// l'instant pour poser l'événement sur la carte.
	OutOfWindow int `json:"outOfWindow"`
	// Unpublished : l'événement était rattaché, mais son slot n'a pas de trajectoire publiée
	// (track trop courte). Compté à part : ce n'est pas un échec de rattachement.
	Unpublished int `json:"unpublished"`
	// RefusedByRoster : événements que le calque REFUSE DE PUBLIER parce que l'effectif du
	// match dépasse ce que le format peut porter — huit slots d'entité de joueur au statborg
	// (cf. objectives.RosterFitsStatborg). Le calque se tait ENTIÈREMENT, et ce compteur
	// dit combien d'actions ce silence coûte : un calque muet dont personne ne sait pourquoi
	// il est muet est pire que le calque faux qu'il remplace.
	//
	// Peuplé par le seul calque des ACTIONS d'objectif ; zéro partout ailleurs.
	//
	// `omitempty` PARCE QUE LA FORME DU DOCUMENT NE DOIT PAS BOUGER POUR RIEN (même règle que
	// `Coverage.Abilities`, lot 5.6) : le compteur vaut zéro sur 65 des 68 artefacts du parc,
	// et l'écrire quand même y changerait chaque octet — donc obligerait à recuire le parc
	// entier pour un champ vide. `SchemaVersion` ne monte pas : un champ optionnel ne change
	// pas la forme (cf. `build_test.go`).
	RefusedByRoster int `json:"refusedByRoster,omitempty"`
	// ByUnit : parmi `Attached`, les tirs poses par leur REFERENCE 0 — le slot du bipede qui a
	// tire, ecrit dans le record (lot M4b.4, `shots.go`) ; UnitOtherIndex : parmi eux, ceux dont
	// l index de tireur designe un AUTRE joueur que celui du slot (l index n est pas le tireur :
	// build HI_1_4_1, ou pont slot -> joueur en defaut). Peuples par le seul calque des tirs.
	ByUnit         int `json:"byUnit,omitempty"`
	UnitOtherIndex int `json:"unitOtherIndex,omitempty"`
}

// Balanced vérifie l'invariant : tout ce qui existait est soit rattaché, soit rejeté sous
// une cause nommée. Une somme fausse signale une fuite — un chemin de rejet non compté.
func (c LayerCoverage) Balanced() bool {
	return c.Attached+c.NoSlot+c.Ambiguous+c.OutOfWindow+c.Unpublished+c.RefusedByRoster ==
		c.Available
}

// rejectReason nomme la cause d'un rejet, pour le comptage.
type rejectReason int

const (
	reasonAttached rejectReason = iota
	reasonNoSlot
	reasonAmbiguous
	reasonOutOfWindow
)

// count incrémente le compteur de la cause.
func (c *LayerCoverage) count(r rejectReason) {
	switch r {
	case reasonAttached:
		c.Attached++
	case reasonNoSlot:
		c.NoSlot++
	case reasonAmbiguous:
		c.Ambiguous++
	case reasonOutOfWindow:
		c.OutOfWindow++
	}
}

// warnIfLossy émet un avertissement ÉCHANTILLONNÉ — un par calque, pas un par événement —
// quand une catégorie de rejet dépasse le seuil. Le nom du calque est passé en clair pour
// que le journal désigne le chantier concerné.
//
// LES QUATRE CATÉGORIES DE REJET SONT SURVEILLÉES, `Unpublished` COMPRISE (correctif du
// 2026-09-06). Elle en était absente, et c'est précisément la seule qui BOUGE sur le parc : sur
// `3372e7eb`, 46 % des actions d'objectif disparaissaient sans une seule ligne de journal, pour
// un seuil de 10 %. Une catégorie de rejet sans alarme est un rejet avalé — l'anti-patron que
// l'en-tête de ce fichier existe pour interdire.
//
// `RefusedByRoster` N'EST PAS DANS CETTE LISTE, ET CE N'EST PAS UN OUBLI : elle vaut TOUT ou
// RIEN (le calque se tait entièrement), et son alarme est émise à la source, là où la décision
// se prend et où l'effectif est connu — `replaybuild.identifiedEvents`. L'ajouter ici ferait
// journaliser deux fois le même refus.
func (c LayerCoverage) warnIfLossy(layer string) {
	if c.Available == 0 {
		return
	}
	for _, cat := range []struct {
		name string
		n    int
	}{{"slotIntrouvable", c.NoSlot}, {"slotAmbigu", c.Ambiguous}, {"horsFenetre", c.OutOfWindow},
		{"sansTrajectoirePubliee", c.Unpublished}} {
		if float64(cat.n)/float64(c.Available) < rejectSampleThreshold {
			continue
		}
		slog.Warn("rejeu : rejets au-dessus du seuil",
			"calque", layer, "cause", cat.name, "rejetes", cat.n, "disponibles", c.Available,
			"rattaches", c.Attached)
	}
}
