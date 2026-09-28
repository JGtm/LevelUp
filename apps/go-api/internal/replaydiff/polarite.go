package replaydiff

// polarite.go — CHAQUE MESURE DE COUVERTURE A UNE POLARITE DECLAREE.
//
// `classerNombres` nomme « perte » toute mesure qui BAISSE : c'est juste pour ce qui mesure une
// richesse (vies nommees, actions rattachees, portages, durees), faux pour les compteurs que la
// couverture publie pour dire ce que la cuisson N'A PAS su faire, et faux encore pour un
// denominateur, qui ne dit ni l'un ni l'autre.
//
// HISTORIQUE DU DEFAUT. Jusqu'au 2026-09-28 une liste FERMEE de derniers segments (`noSlot`,
// `unpublished`...) designait les compteurs d'echec, et toute autre mesure gardait la lecture
// « plus = mieux ». Le G-corpus J11 (`.ai/V7.5/film_re/G_CORPUS_J11_2026-09-28.md` §4) en a
// chiffre le cout : environ 110 des 390 lignes « perte » etaient des compteurs d'echec qui
// BAISSAIENT (`holes*`, `*Unlocated`, `no_owner`, `desync`, `uncovered`...), et 70 compteurs
// d'echec qui MONTAIENT sortaient en « gain », donc absents du rapport. Une liste fermee se
// periment au premier compteur neuf, dans le sens qui cache la perte.
//
// LA REGLE (lot R4). La classification est TOTALE : chaque feuille numerique de `coverage.*` et de
// `bombStats.coverage.*` a une polarite declaree dans `polarite_table.go`, et un ratchet
// (`polarite_ratchet_test.go`) derive l'inventaire des feuilles de la forme du document et rougit
// sur toute feuille non classee. Une feuille de couverture INCONNUE a l'execution (artefact d'un
// schema que la table ne connait pas) se lit en CHANGEMENT : ni gain ni perte presumes, mais
// visible et bloquante au gate de corpus — jamais un gain silencieux.

import (
	"strings"
	"sync"
)

// Polarite est le sens de lecture d'une mesure de couverture.
type Polarite int

// Les classes de polarite (critere dans l'en-tete de `polarite_table.go`).
const (
	// PolariteInconnue : feuille de couverture absente de la table (ratchet rouge a la forme
	// courante ; lue en changement a l'execution).
	PolariteInconnue Polarite = iota
	// PolariteEchec : plus = pire. Sens generique inverse.
	PolariteEchec
	// PolariteSucces : plus = mieux. Sens generique.
	PolariteSucces
	// PolariteNeutre : denominateur, population, ventilation : un changement.
	PolariteNeutre
	// PolariteTelemetrie : reglage ou forme de l'outil : un changement, que le verdict du gate
	// de corpus affiche sans le compter.
	PolariteTelemetrie
)

// jokerCle designe, dans la table, toute cle d'une map de la couverture.
const jokerCle = "*"

// indexPolarites : la table depliee, cle complete (ou motif a joker) -> polarite.
type indexPolarites struct {
	exactes map[string]Polarite
	motifs  []motifPolarite
	// doublons : cles declarees deux fois — le ratchet exige qu'il n'y en ait aucune.
	doublons []string
}

// motifPolarite : une entree a joker, decoupee autour de `*`.
type motifPolarite struct {
	cle, avant, apres string
	pol               Polarite
}

var (
	indexUnique sync.Once
	indexTable  indexPolarites
)

// polarites rend l'index de la table courante et des feuilles heritees, construit une fois.
func polarites() *indexPolarites {
	indexUnique.Do(func() {
		indexTable = construireIndex(append(append([]blocPolarites{}, tablePolarites...),
			polaritesHeritees...))
	})
	return &indexTable
}

// construireIndex deplie des lignes de table en index. Une cle declaree deux fois garde sa
// premiere polarite et entre dans `doublons`.
func construireIndex(lignes []blocPolarites) indexPolarites {
	idx := indexPolarites{exactes: map[string]Polarite{}}
	for _, l := range lignes {
		for _, bloc := range l.Blocs {
			for _, c := range []struct {
				liste string
				pol   Polarite
			}{{l.Echecs, PolariteEchec}, {l.Succes, PolariteSucces}, {l.Neutres, PolariteNeutre},
				{l.Telemetrie, PolariteTelemetrie}} {
				for _, f := range strings.Fields(c.liste) {
					idx.ajouter(bloc+f, c.pol)
				}
			}
		}
	}
	return idx
}

func (idx *indexPolarites) ajouter(cle string, pol Polarite) {
	if _, deja := idx.exactes[cle]; deja {
		idx.doublons = append(idx.doublons, cle)
		return
	}
	idx.exactes[cle] = pol
	if i := strings.Index(cle, jokerCle); i >= 0 {
		idx.motifs = append(idx.motifs, motifPolarite{cle: cle, avant: cle[:i],
			apres: cle[i+len(jokerCle):], pol: pol})
	}
}

// lire rend la polarite d'un chemin : entree exacte d'abord, sinon le motif a joker le plus
// precis (le plus long litteral) qui l'accepte.
func (idx *indexPolarites) lire(chemin string) Polarite {
	if p, ok := idx.exactes[chemin]; ok {
		return p
	}
	meilleur, longueur := PolariteInconnue, -1
	for _, m := range idx.motifs {
		if len(chemin) <= len(m.avant)+len(m.apres) ||
			!strings.HasPrefix(chemin, m.avant) || !strings.HasSuffix(chemin, m.apres) {
			continue
		}
		if n := len(m.avant) + len(m.apres); n > longueur {
			meilleur, longueur = m.pol, n
		}
	}
	return meilleur
}

// estCouverture dit si un chemin aplati appartient a la couverture (tete ou calque d'Assaut).
func estCouverture(chemin string) bool {
	return strings.HasPrefix(chemin, racineCouverture) || strings.HasPrefix(chemin, racineAssaut)
}

// PolariteDe rend la polarite d'une metrique d'ecart (`Difference.Metrique`, chemin aplati sans
// axe) et si elle est une mesure de couverture. Hors couverture : (PolariteInconnue, false).
func PolariteDe(metrique string) (Polarite, bool) {
	if !estCouverture(metrique) {
		return PolariteInconnue, false
	}
	return polarites().lire(metrique), true
}

// sensSelonPolarite applique la polarite d'une mesure de couverture (chemin aplati, sans axe)
// au sens generique mesure par `classer`. Hors couverture, le sens generique est rendu tel quel.
func sensSelonPolarite(chemin, sens string) string {
	pol, couverture := PolariteDe(chemin)
	if !couverture || sens == SensChangement {
		return sens
	}
	switch pol {
	case PolariteEchec:
		return inverserSens(sens)
	case PolariteSucces:
		return sens
	default: // neutre, telemetrie, inconnue
		return SensChangement
	}
}

// marqueurParXUID / marqueurParSlot : segments des cles ventilees par joueur
// (`.../par-xuid/<xuid>`, `tracks/vies-par-xuid/<xuid>`, `.../duree-totale/par-xuid/<xuid>`) ou,
// pour la MEME mesure quand le porteur n'a pas de nom, par slot (`.../par-slot/<slot>`, cf.
// `empreinte_durees.go`). Les deux ventilations d'une meme mesure forment UN groupe : un trajet
// qui passe de « slot 513 sans nom » a « joueur X » (lot E2, 2026-09-08 : 14 slots de `084a804d`)
// n'a pas disparu, il a trouve son porteur — la somme sur les deux ventilations le dit.
const (
	marqueurParXUID = "par-xuid/"
	marqueurParSlot = "par-slot/"
	marqueurGroupe  = "par-*/"
)

// groupeParXUID rend la cle de GROUPE d'une mesure ventilee par joueur ou par slot (tout ce qui
// precede le marqueur, le marqueur neutralise), et false si la mesure n'est pas ventilee.
func groupeParXUID(k string) (string, bool) {
	for _, m := range []string{marqueurParXUID, marqueurParSlot} {
		if i := strings.LastIndex(k, m); i >= 0 {
			return k[:i] + marqueurGroupe, true
		}
	}
	return "", false
}

// groupesConserves : les groupes par joueur dont la SOMME sur tous les joueurs est la meme des
// deux cotes. Une baisse chez un joueur y est une REATTRIBUTION — ce que le lien direct
// corps -> joueur (lot E2, 2026-09-08) fait a raison quand il rend a l'un ce que le pont par
// morts attribuait a l'autre (ramassages : 114, 402, 332... identiques ; trajets en vehicule :
// 74 et 28 995 frames conserves) — pas une perte. Une somme qui baisse reste une perte a
// instruire ; une somme qui monte, un gain.
func groupesConserves(a, b map[string]Mesure) map[string]bool {
	sa, sb := map[string]float64{}, map[string]float64{}
	presents := map[string]bool{}
	for k, m := range a {
		if g, ok := groupeParXUID(k); ok && m.EstNum {
			sa[g] += m.Num
			presents[g] = true
		}
	}
	for k, m := range b {
		if g, ok := groupeParXUID(k); ok && m.EstNum {
			sb[g] += m.Num
			presents[g] = true
		}
	}
	out := map[string]bool{}
	for g := range presents {
		// Somme conservee OU en hausse : la part attribuee peut monter (des ramassages qui
		// n'avaient pas d'auteur en trouvent un) pendant qu'un joueur en perd au profit d'un
		// autre — c'est encore une reattribution. Seule une somme qui BAISSE est une perte.
		if proches(sa[g], sb[g]) || sb[g] > sa[g] {
			out[g] = true
		}
	}
	return out
}

// estReattribution dit si la mesure `k` appartient a un groupe par joueur conserve.
func estReattribution(k string, conserves map[string]bool) bool {
	g, ok := groupeParXUID(k)
	return ok && conserves[g]
}

// inverserSens retourne le sens d'un ecart pour un compteur d'echec : une baisse (perte
// generique) devient un gain, une hausse une perte ; un compteur qui apparait (> 0) est une
// perte, un compteur qui disparait est un gain. Un changement textuel reste un changement.
func inverserSens(sens string) string {
	switch sens {
	case SensPerte:
		return SensGain
	case SensGain:
		return SensPerte
	case SensApparu:
		return SensPerte
	case SensDisparu:
		return SensGain
	}
	return sens
}

// String nomme la classe, pour les messages et les rapports.
func (p Polarite) String() string {
	switch p {
	case PolariteEchec:
		return "echec"
	case PolariteSucces:
		return "succes"
	case PolariteNeutre:
		return "neutre"
	case PolariteTelemetrie:
		return "telemetrie"
	}
	return "inconnue"
}
