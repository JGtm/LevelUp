package replay

// build_objective_objects.go — LE CÂBLAGE du calque des objets d'objectif LIBRES.
//
// # POURQUOI CE CALQUE N'EST PAS SOUS LA GARDE DE MODE DU DRAPEAU
//
// `attachFlagCarries` s'arrête hors CTF, et c'est un correctif de production JUSTIFIÉ : le pont
// d'identité qu'il construit ensuite (`SlotIdentityByDeaths`) déroule le compteur de morts de
// chaque slot du statborg, ce qui coûtait 19 à 22 Go sur un film dont la grammaire n'est pas
// celle du CTF. La garde protège CE pont-là.
//
// CE CALQUE-CI N'EN A PAS BESOIN, ET C'EST VÉRIFIABLE LIGNE À LIGNE : il ne lit ni le statborg,
// ni les morts, ni l'identité des joueurs. Il ne consomme que le balayage `ti=42` de la CHAÎNE
// DES SOCLES — déjà payé pour les armes au sol sur TOUS les films, quel que soit le mode — et la
// table d'identité du manifeste. Le placer sous la garde du drapeau l'aurait éteint sur Oddball,
// c'est-à-dire exactement là où il sert.
//
// AUCUNE LECTURE DE FILM N'EST AJOUTÉE PAR CE CALQUE. C'est la même propriété que le calque des
// vies libres du drapeau : `opt.Pads.Weapons` porte déjà les créations et les pistes delta de la
// bande de slots `ti=42`.
//
// HORS LIGNE — appelé par l'assemblage, comme les autres calques.

import (
	"log/slog"
	"sort"
)

// attachObjectiveObjects pose les vies LIBRES des objets d'objectif publiables sur le document,
// avec leur couverture et leur journal.
func attachObjectiveObjects(doc *ReplayDocument, opt Options, clock replayClock) {
	lives, cov := buildObjectiveObjects(
		opt.Pads.Weapons, opt.Labels.ObjectiveObjects, opt.Labels.ObjectiveFamilies, clock)
	doc.ObjectiveObjects = lives
	if doc.Coverage != nil {
		doc.Coverage.ObjectiveObjects = cov
	}
	logObjectiveObjectsCoverage(cov)
}

// objectiveObjectPublished — les familles dont les vies libres sont PUBLIÉES.
//
// LA LISTE EST PLUS COURTE QUE CELLE DES OBJETS RECONNUS, ET C'EST UNE MESURE, PAS UN REPORT.
// Le manifeste déclare `flag` et `ball` ; les deux sont écartés des socles d'armes, les deux ont
// des vies libres lisibles. Seul `ball` est publié parce que le CONTRÔLE 3 du lot du drapeau a
// ÉCHOUÉ sur ses vies à lui : 149/197 = 75,6 % naissent à un socle ou aux pieds du porteur qui
// vient de finir, pour un seuil de 90 % écrit avant la mesure (le témoin, lui, tient à 12,8 %).
// Un quart des vies du drapeau reste inexpliqué ; les publier reviendrait à dessiner un drapeau
// là où l'on n'est pas sûr qu'il soit.
//
// LE JOUR OÙ CE NÉGATIF SERA LEVÉ, une ligne suffira ici et AUCUNE CLÉ DU DOCUMENT NE BOUGERA :
// c'est pourquoi la forme publiée porte `family` plutôt que de s'appeler « crâne ».
// familleCrane — l'identifiant de famille du crane, tel que le MANIFESTE le publie. Constante
// parce qu'il est repris par la table ci-dessous et par tous les cas de test du paquet : un
// litteral disperse est le premier a diverger le jour ou le manifeste renommerait la famille.
const familleCrane = "ball"

var objectiveObjectPublished = map[string]bool{familleCrane: true}

// buildObjectiveObjects assemble les vies libres publiables. PUR : aucune lecture de film, aucune
// base — tout vient du balayage déjà fait et des tables du manifeste.
func buildObjectiveObjects(scan WorldObjectScan, labels map[uint32]Label,
	families map[uint32]string, clock replayClock,
) ([]ObjectiveObjectLife, *ObjectiveObjectsCoverage) {
	cov := &ObjectiveObjectsCoverage{Scanned: scan.Scanned}
	publiables := map[uint32]Label{}
	for id, fam := range families {
		if objectiveObjectPublished[fam] && labels[id] != (Label{}) {
			publiables[id] = labels[id]
		}
	}
	cov.Declared = len(publiables)
	if !scan.Scanned || len(publiables) == 0 || clock.step == 0 {
		return nil, cov
	}
	out := make([]ObjectiveObjectLife, 0, len(publiables))
	for _, l := range flagFreeLives(scan, publiables) {
		vie, ok := objectiveObjectLifeOf(l, families[l.ID], publiables[l.ID], clock)
		if !ok {
			cov.OutOfAxis++
			continue
		}
		cov.Lives, cov.Points = cov.Lives+1, cov.Points+len(vie.Pts)
		if len(vie.Pts) == 1 {
			cov.Motionless++
		}
		out = append(out, vie)
	}
	if len(out) == 0 {
		return nil, cov
	}
	// ORDRE TOTAL : l'instant de départ, puis la famille. `flagFreeLives` trie déjà par instant
	// et par clé de vie ; ce tri-ci le rend indépendant de cette garantie plutôt que de s'y
	// adosser — deux tris cohérents coûtent moins qu'un couplage silencieux.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].Family < out[j].Family
	})
	return out, cov
}

// objectiveObjectLifeOf convertit UNE vie libre en sa forme publiée. Faux quand la vie tombe
// hors de l'axe de frames — elle ne serait pas dessinable, et l'inventer serait pire.
func objectiveObjectLifeOf(l flagFreeLife, family string, label Label,
	clock replayClock,
) (ObjectiveObjectLife, bool) {
	if len(l.Pts) == 0 {
		return ObjectiveObjectLife{}, false
	}
	t0 := frameOf(l.T0US, clock.origin, clock.step)
	if t0 < 0 || t0 >= clock.frames {
		return ObjectiveObjectLife{}, false
	}
	vie := ObjectiveObjectLife{Family: family, En: label.En, Fr: label.Fr, T0: t0, T1: t0}
	for _, p := range l.Pts {
		t := frameOf(p.TUS, clock.origin, clock.step)
		if t < 0 || t >= clock.frames {
			continue // un point hors axe ne se dessine pas ; la vie, elle, reste publiable
		}
		// UN SEUL POINT PAR FRAME : deux échantillons de la même frame se dessineraient au même
		// endroit de l'axe, et le second seul vaut — c'est la position la plus récente connue.
		if n := len(vie.Pts); n > 0 && vie.Pts[n-1].T == t {
			vie.Pts[n-1] = ObjectiveObjectPoint{T: t, X: p.X, Y: p.Y}
			continue
		}
		vie.Pts = append(vie.Pts, ObjectiveObjectPoint{T: t, X: p.X, Y: p.Y})
	}
	if len(vie.Pts) == 0 {
		return ObjectiveObjectLife{}, false
	}
	vie.T0, vie.T1 = vie.Pts[0].T, vie.Pts[len(vie.Pts)-1].T
	return vie, true
}

// logObjectiveObjectsCoverage journalise ce que le calque a publié et ce qu'il a écarté. Un
// calque vide doit DIRE lequel de ses silences il sert.
func logObjectiveObjectsCoverage(cov *ObjectiveObjectsCoverage) {
	if cov == nil {
		return
	}
	slog.Info("rejeu : objets d objectif libres",
		"balaye", cov.Scanned, "declares", cov.Declared, "vies", cov.Lives,
		"points", cov.Points, "immobiles", cov.Motionless, "horsAxe", cov.OutOfAxis)
}
