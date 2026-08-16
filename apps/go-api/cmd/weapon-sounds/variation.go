package main

// variation.go — LA VARIATION PAR LECTURE : ce que le paquet RANGED autorise, et comment
// il remonte jusqu'a l'app.
//
// POURQUOI. Les `.wav` extraits sont PURS : un fichier, toujours le meme. Le jeu, lui,
// deplace a chaque coup le volume et la hauteur dans une fourchette declaree par le noeud
// (paquet RANGED). Sans cette fourchette, une rafale rejouee in-app est le meme echantillon
// repete a l'identique — le defaut 2 du chantier sons. La cuire dans les fichiers etait
// exclu (decision utilisateur 1) : c'est donc l'app qui l'applique, et il lui faut la
// fourchette.
//
// CE QUI EST EXPORTE, ET A QUELLE GRANULARITE :
//
//	couche      : la fourchette accumulee le long du chemin evenement -> ... -> Sound,
//	              enveloppee sur les variantes du point de choix
//	evenement   : la fourchette de sa COUCHE DOMINANTE (celle de plus fort gain de chemin)
//	mode de tir : la fourchette de l'evenement dont la couche dominante est la plus forte
//
// L'agregation par couche dominante suit le plan : une couche de renfort 20 dB en arriere
// ne doit pas dicter la variation du coup, sa propre variation s'entend a peine.
//
// LA PERSPECTIVE (1re / 3e personne) N'EST PAS MODELISEE ICI, et ce n'est pas un oubli :
// aucune structure du pipeline Go ne la porte (verifie a l'etape 1 — le seul endroit ou
// « 1p/3p » apparait est la liste de verbes candidats de `noms.go`, qui sert au hachage des
// noms). La distinction vit dans les EVENEMENTS, et c'est a cette granularite que la
// fourchette est exportee : le manifeste de l'app retient l'evenement 3e personne, le rejeu
// 2D etant une vue exterieure.

import (
	"fmt"
	"sort"
)

// etatChemin : ce qu'un chemin de la hierarchie accumule jusqu'a un `.wem`.
//
// Le gain se SOMME (mesure de l'etape 18 du chantier sons), et la fourchette de variation
// aussi : chaque noeud du chemin tire son propre ecart, les ecarts s'additionnent en dB
// comme en centiemes. La somme des bornes est l'enveloppe exacte du resultat possible.
type etatChemin struct {
	Gain float64
	Var  fourchetteSon
}

// avec rend l'etat obtenu en traversant un noeud qui porte `gain` et `variation`.
func (e etatChemin) avec(gain float64, v fourchetteSon) etatChemin {
	return etatChemin{Gain: e.Gain + gain, Var: sommeFourchettes(e.Var, v)}
}

// plusGain rend l'etat avec un gain additionnel, sans nouvelle variation (courbe de fondu).
func (e etatChemin) plusGain(gain float64) etatChemin {
	return etatChemin{Gain: e.Gain + gain, Var: e.Var}
}

func sommeFourchettes(a, b fourchetteSon) fourchetteSon {
	return fourchetteSon{
		VolumeDB: fourchette{Bas: a.VolumeDB.Bas + b.VolumeDB.Bas, Haut: a.VolumeDB.Haut + b.VolumeDB.Haut},
		PitchCts: fourchette{Bas: a.PitchCts.Bas + b.PitchCts.Bas, Haut: a.PitchCts.Haut + b.PitchCts.Haut},
		Lu:       a.Lu || b.Lu,
	}
}

// enveloppeFourchettes rend la fourchette qui contient les deux — l'agregation d'un point
// de choix, dont le moteur joue UNE variante parmi plusieurs.
func enveloppeFourchettes(a, b fourchetteSon) fourchetteSon {
	return fourchetteSon{
		VolumeDB: fourchette{Bas: min32(a.VolumeDB.Bas, b.VolumeDB.Bas), Haut: max32(a.VolumeDB.Haut, b.VolumeDB.Haut)},
		PitchCts: fourchette{Bas: min32(a.PitchCts.Bas, b.PitchCts.Bas), Haut: max32(a.PitchCts.Haut, b.PitchCts.Haut)},
		Lu:       a.Lu || b.Lu,
	}
}

// variationRendue : la fourchette telle qu'elle sort dans les rapports JSON, avec la couche
// dont elle provient. Citer la couche permet de contre-verifier une fourchette surprenante
// sans relancer l'extraction.
type variationRendue struct {
	VolumeDB fourchette `json:"volume_db"`
	PitchCts fourchette `json:"pitch_cents"`
	Couche   string     `json:"couche,omitempty"`
	GainDB   float32    `json:"gain_db_couche"`
}

// variationDeCouches rend la fourchette de la COUCHE DOMINANTE d'un evenement.
//
// Dominante = plus fort gain de chemin. Rend nil quand aucune couche ne declare d'ecart :
// le rapport reste alors muet plutot que de publier une fourchette nulle qui ressemblerait
// a une mesure.
func variationDeCouches(couches []brancheRendue) *variationRendue {
	meilleure := -1
	for i, c := range couches {
		if c.Variation == nil {
			continue
		}
		if meilleure < 0 || c.Variation.GainDB > couches[meilleure].Variation.GainDB {
			meilleure = i
		}
	}
	if meilleure < 0 {
		return nil
	}
	v := *couches[meilleure].Variation
	return &v
}

// variationDominante rend, parmi plusieurs fourchettes deja agregees, celle de la couche
// au plus fort gain. Sert a remonter d'un ensemble d'evenements a un mode de tir.
func variationDominante(vs []*variationRendue) *variationRendue {
	var out *variationRendue
	for _, v := range vs {
		if v == nil {
			continue
		}
		if out == nil || v.GainDB > out.GainDB {
			out = v
		}
	}
	if out == nil {
		return nil
	}
	copie := *out
	return &copie
}

// variationsDeArme indexe, par identifiant d'evenement, la fourchette calculee en passe 1.
// C'est le seul endroit ou la variation traverse la frontiere entre les deux passes : elle
// voyage par le JSON, comme tout le reste (les deux modules ne coexistent jamais en RAM).
func variationsDeArme(a armeLot) map[uint32]*variationRendue {
	out := make(map[uint32]*variationRendue, len(a.Evenements))
	for _, e := range a.Evenements {
		if e.Variation != nil {
			out[e.IDEvent] = e.Variation
		}
	}
	return out
}

// modesDeArme rend les modes de tir d'une arme, chacun avec ses evenements et la fourchette
// de sa couche dominante.
func modesDeArme(tags []uint32, eventsParTag map[uint32]map[uint32]bool,
	armeDeEvent map[uint32]int, arme int, varDeEvent map[uint32]*variationRendue) []modeDeTir {
	var out []modeDeTir
	for _, tag := range tags {
		var evs []string
		var vars []*variationRendue
		for e := range eventsParTag[tag] {
			if armeDeEvent[e] != arme {
				continue
			}
			evs = append(evs, fmt.Sprintf("%08x", e))
			vars = append(vars, varDeEvent[e])
		}
		if len(evs) == 0 {
			continue
		}
		sort.Strings(evs)
		out = append(out, modeDeTir{
			TagSon: fmt.Sprintf("%08x", tag), Events: evs, Variation: variationDominante(vars),
		})
	}
	return out
}

// statsSignesVariation compte les signes observes dans le paquet RANGED.
//
// POURQUOI CE COMPTEUR. Le format donne deux composantes par propriete sans dire laquelle
// est le minimum. Deux lectures restent possibles : des OFFSETS SIGNES autour du nominal
// (une composante negative, une positive) ou deux MAGNITUDES positives a retrancher et
// ajouter. Le chantier ne postule pas : ce compteur, imprime en fin d'extraction, tranche
// sur les donnees reelles. Une majorite de couples (negatif, positif) confirme les offsets
// signes ; deux composantes positives partout imposerait l'autre lecture.
var statsSignesVariation struct {
	Couples, Negatives, Positives, Nulles int
}

func noterSignesVariation(a, b float32) {
	statsSignesVariation.Couples++
	for _, v := range [2]float32{a, b} {
		switch {
		case v < 0:
			statsSignesVariation.Negatives++
		case v > 0:
			statsSignesVariation.Positives++
		default:
			statsSignesVariation.Nulles++
		}
	}
}

// afficherSignesVariation imprime le releve. Appele en fin des modes qui parcourent des
// banks — c'est la ligne que le pilote lira pour statuer sur l'interpretation.
func afficherSignesVariation() {
	s := statsSignesVariation
	if s.Couples == 0 {
		fmt.Println("variation RANGED : aucun couple lu")
		return
	}
	fmt.Printf("variation RANGED : %d couples lus | composantes negatives %d, positives %d, nulles %d\n",
		s.Couples, s.Negatives, s.Positives, s.Nulles)
}

// SCHEMA DU MANIFESTE DESTINE A L'APP
// (`static/weapons-assets/halo_infinite/sons/index.json`, nom `index.json` impose par la
// convention des trois manifestes d'assets existants).
//
// Il n'est PAS declare en structures Go : rien dans ce depot ne produit ce fichier — le
// rendu des `.wav` vit hors depot, et livrer un producteur sans contenu ferait du code
// mort. Le contrat est donc pose ici, et son type vivant est celui du lecteur web, ecrit
// et teste a l'etape 3 du plan.
//
//	{
//	  "source": "<provenance : chantier + date de generation>",
//	  "sons": [
//	    {
//	      "arme": "<cle d'arme, celle du manifeste d'icones jeu/index.json>",
//	      "fichier": "<nom du .wav, relatif a ce dossier>",
//	      "mode": "<tag de son de tir, absent si l'arme n'a qu'un mode>",
//	      "variation": {
//	        "volume_db":    {"bas": -2.0, "haut": 1.0},
//	        "pitch_cents":  {"bas": -50.0, "haut": 50.0}
//	      }
//	    }
//	  ]
//	}
//
// Unites : `volume_db` en decibels, `pitch_cents` en centiemes de demi-ton (l'app en tire
// un `playbackRate` par 2^(cents/1200)). Les bornes sont des OFFSETS autour de la valeur
// nominale du fichier, exactement ce que rend `variationRendue`. Champ `variation` absent
// ou fourchettes nulles : LECTURE PURE, sans erreur ni silence — c'est l'etat courant tant
// que l'extraction n'a pas tourne.
