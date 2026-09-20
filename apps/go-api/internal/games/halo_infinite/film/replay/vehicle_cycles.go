package replay

// vehicle_cycles.go — LE CYCLE DE REAPPARITION DES VEHICULES (schema 63), PAR EMPLACEMENT DE
// NAISSANCE.
//
// # LE FILM N ECRIT AUCUN MINUTEUR DE REAPPARITION DE VEHICULE, ET C EST UN NEGATIF MESURE
//
// L univers des 294 noms de composant a ete balaye (note 3.7 § 4) : rien. La reapparition ne peut
// donc que se DEDUIRE, et la forme de la deduction est celle que `PadCycle` tient deja pour les
// socles d arme — mediane, deciles, ecarts MESURES, manques COMPTES, cle ABSENTE quand le cycle
// n est pas etabli (cf. document_ground_weapons.go). Le juge est le MEME : `gwPadsCycleFromGaps`,
// avec ses deux regles (au moins deux ecarts, ecart-type sous 20 % de la mediane). Une seconde
// regle de stabilite ferait dire deux choses au mot « etabli ».
//
// # POURQUOI CE CALQUE N EXISTAIT PAS AVANT LE 2026-09-19, ET CE QUI L A DEBLOQUE
//
// Le cycle exige, par vie, la fin DATEE de la vie PRECEDENTE au meme emplacement. Or `TEnd`
// etait quasi toujours absent : 1 fin datee sur 109 vies (`a349fea8`), 1 sur 42 (`a521164d`),
// 3 sur 97 (`4f77afc1`) — sur 31 emplacements agglomeres pour `a349fea8`, le compte des ecarts
// mesurables etait ZERO et les 56 occasions etaient toutes des MANQUES. Publier alors aurait
// donne une cle jamais etablie.
//
// LA CAUSE N ETAIT PAS L APPARIEMENT MAIS LA GRAMMAIRE (lot 5.1.4, puis 5.1.7-b) : la marche de
// `ti=40` perdait 90 a 100 % des dead-states que le film ANNONCE, parce que l etat par defaut de
// l archetype n etait pas lu — le second mot de taille se lisait 79 bits trop tot et la boucle de
// composants ne demarrait jamais. Les cinq feuilles posees, `4f77afc1` passe de 97 a 149 vies
// publiees et de 3 a 11 fins datees, `a349fea8` a 14 — et `finDatee == mortsAppariees` sur les
// deux. C est cette lecture-la qui rend le cycle mesurable ; ce fichier n en est que la
// consequence arithmetique.
//
// # L EMPLACEMENT, ET POURQUOI IL EST AGGLOMERE
//
// Un vehicule ne renait pas au millimetre pres de sa naissance precedente. Les emplacements sont
// donc des AMAS de naissances a `vehicleCycleClusterM` metres, et c est la maille avec laquelle
// la mesure du 2026-09-17 a compte 31 emplacements sur `a349fea8`. Un amas est nomme par le
// BARYCENTRE de ses naissances : c est la seule position qui ne privilegie aucune des vies.
//
// L ECART SE MESURE DE LA MORT A LA NAISSANCE SUIVANTE, comme pour les socles d arme et pour la
// meme raison mesuree (item 2.4 : 24 socles etablis sur 57 contre 4 pour l horloge d apparition,
// aux memes regles) : l horloge du jeu repart quand l objet DISPARAIT, pas quand il apparait.
//
// # CE QUE LE CALQUE REFUSE DE FAIRE
//
// Il ne compte comme ecart que les couples dont la vie PRECEDENTE porte une fin DATEE ET
// DESTRUCTRICE (`VehicleEndDestroyed`). Une fin `film_end` n est pas une mort et une fin
// `unknown` n est pas datee : les compter reviendrait a mesurer le recensement des images-cles —
// exactement l erreur que `V2_SPAWNS_COOLDOWNS` avait nommee, et qui bornait la fin a +/- 20 s.
// Ces occasions se COMPTENT (`Missing`), elles ne se devinent pas.

import (
	"sort"
)

// vehicleCycleClusterM est le rayon d agglomeration des naissances en UN emplacement, en metres.
//
// C EST LA MAILLE DE LA MESURE, PAS UN REGLAGE : les 31 emplacements de `a349fea8` (dont 15
// credibles, 2 a 8 vies chacun) ont ete comptes a cette maille le 2026-09-17, et tout le
// raisonnement du lot 5.1.5 s y refere. La changer changerait le denominateur sans changer le
// film.
const vehicleCycleClusterM = 2.0

// VehicleCycle est LE CYCLE DE REAPPARITION d un emplacement de naissance de vehicule : le delai,
// en secondes, entre la destruction d un vehicule et la naissance du suivant AU MEME ENDROIT.
//
// C EST UNE DONNEE DE MATCH, PAS DE CARTE, et elle n est publiee que la ou la recurrence est
// MESUREE — meme doctrine que `WeaponPad` / `PadCycle`. Un emplacement dont le cycle n est pas
// etabli n apparait pas dans la liste : il n y a rien a en dire.
type VehicleCycle struct {
	// X / Y sont le BARYCENTRE des naissances de l amas, en coordonnees monde (memes axes que
	// `Point.X/Y` et `VehicleSpawn.X/Y`).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// Family est la famille DOMINANTE des vies nees ici (`warthog`, `ghost`, ...), quand la table
	// des chassis en nomme au moins une. ABSENTE sinon : un emplacement dont tous les chassis
	// sont inconnus garde son cycle et ne prend pas le nom d un voisin — la meme regle que le
	// chassis affiche en hexadecimal a cote de sa famille inconnue.
	Family string `json:"family,omitempty"`
	// MedianS / P10S / P90S : la mediane et les deciles des ecarts mesures, en secondes.
	MedianS float32 `json:"medianS"`
	P10S    float32 `json:"p10S"`
	P90S    float32 `json:"p90S"`
	// Gaps est le nombre d ecarts MESURES. Un cycle n est ETABLI qu a partir de deux : un ecart
	// unique n a pas d ecart-type, et rien ne dit qu il se repete.
	Gaps int `json:"gaps"`
	// Missing est le nombre de naissances dont la vie PRECEDENTE au meme emplacement n a pas de
	// fin datee et destructrice : autant d ecarts que l emplacement offrait et que la mesure n a
	// pas pu prendre.
	//
	// C EST L AUTRE MOITIE DU DENOMINATEUR, et sans `omitempty` pour la meme raison que
	// `PadCycle.Missing` : un zero DIT quelque chose — l emplacement a rendu tout ce qu il
	// offrait.
	Missing int `json:"missing"`
}

// vehicleCycleCluster est un amas de naissances en cours de construction.
type vehicleCycleCluster struct {
	sx, sy   float64
	n        int
	familles map[string]int
	// lives porte, par vie et dans l ordre de naissance, la frame de naissance et la frame de
	// mort DATEE (-1 quand la vie n en a pas).
	lives []vehicleCycleLife
}

// vehicleCycleLife est ce qu une vie apporte a son emplacement : quand elle nait, et quand le
// film ECRIT qu elle meurt.
type vehicleCycleLife struct {
	t0   int
	tEnd int // -1 : pas de fin datee et destructrice
}

// buildVehicleCycles agglomere les naissances publiees et rend les cycles ETABLIS, tries.
//
// IL NE LIT QUE LE DOCUMENT : les vies, leurs naissances et leurs fins sont deja publiees et
// deja fusionnees (les relais). Recalculer depuis le balayage publierait un cycle sur des vies
// que le document ne montre pas.
func buildVehicleCycles(tracks []VehicleTrack, stepUS uint64, cov *VehicleCoverage) []VehicleCycle {
	if len(tracks) == 0 || stepUS == 0 {
		return nil
	}
	amas := clusterVehicleSpawns(tracks)
	if cov != nil {
		cov.CycleLocations = len(amas)
	}
	secPerFrame := float64(stepUS) / 1e6
	out := make([]VehicleCycle, 0, len(amas))
	for _, a := range amas {
		gaps, manques := vehicleCycleGaps(a.lives, secPerFrame)
		if cov != nil {
			cov.CycleGaps += len(gaps)
			cov.CycleMissing += manques
		}
		c := gwPadsCycleFromGaps(gaps)
		if !c.Established {
			continue
		}
		out = append(out, VehicleCycle{
			X: round2(float32(a.sx / float64(a.n))), Y: round2(float32(a.sy / float64(a.n))),
			Family:  dominantVehicleFamily(a.familles),
			MedianS: round2(float32(c.MedianS)), P10S: round2(float32(c.P10S)),
			P90S: round2(float32(c.P90S)), Gaps: c.Gaps, Missing: manques,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].Y < out[j].Y
	})
	if cov != nil {
		cov.Cycles = len(out)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// clusterVehicleSpawns agglomere les naissances SITUEES a `vehicleCycleClusterM` metres.
//
// L AGGLOMERATION EST GLOUTONNE ET ORDONNEE PAR LE TEMPS : une naissance rejoint le premier amas
// dont le barycentre COURANT est assez proche, sinon elle en ouvre un. C est la meme discipline
// que les grappes de socle d arme, et elle est stable parce que l ordre d entree est celui du
// film — pas celui d une carte Go.
//
// UNE VIE SANS NAISSANCE SITUEE N ENTRE DANS AUCUN AMAS : elle n a pas d emplacement, et lui en
// inventer un ferait naitre des ecarts entre des endroits differents.
func clusterVehicleSpawns(tracks []VehicleTrack) []*vehicleCycleCluster {
	ordre := make([]int, 0, len(tracks))
	for i := range tracks {
		if tracks[i].Spawn != nil {
			ordre = append(ordre, i)
		}
	}
	sort.SliceStable(ordre, func(i, j int) bool { return tracks[ordre[i]].T0 < tracks[ordre[j]].T0 })
	var out []*vehicleCycleCluster
	for _, i := range ordre {
		tr := tracks[i]
		a := nearestVehicleCluster(out, tr.Spawn.X, tr.Spawn.Y)
		if a == nil {
			a = &vehicleCycleCluster{familles: map[string]int{}}
			out = append(out, a)
		}
		a.sx += float64(tr.Spawn.X)
		a.sy += float64(tr.Spawn.Y)
		a.n++
		if tr.Family != "" {
			a.familles[tr.Family]++
		}
		a.lives = append(a.lives, vehicleCycleLife{t0: tr.T0, tEnd: vehicleDatedEnd(tr)})
	}
	return out
}

// nearestVehicleCluster rend l amas dont le barycentre courant est le plus proche du point, s il
// tient dans le rayon. Nil : le point ouvre un amas.
func nearestVehicleCluster(amas []*vehicleCycleCluster, x, y float32) *vehicleCycleCluster {
	var best *vehicleCycleCluster
	bestD := float64(vehicleCycleClusterM) * float64(vehicleCycleClusterM)
	for _, a := range amas {
		if a.n == 0 {
			continue
		}
		d := sqDist(x, y, float32(a.sx/float64(a.n)), float32(a.sy/float64(a.n)))
		if d <= bestD {
			best, bestD = a, d
		}
	}
	return best
}

// vehicleDatedEnd rend la frame de fin d une vie quand le film l ECRIT comme une destruction,
// -1 sinon. C est le seul instant de fin qui vaille un ecart (cf. l en-tete).
func vehicleDatedEnd(tr VehicleTrack) int {
	if tr.End != VehicleEndDestroyed || tr.TEnd == nil {
		return -1
	}
	return *tr.TEnd
}

// vehicleCycleGaps rend les ecarts mesurables d un emplacement, en secondes, et le compte des
// occasions PERDUES.
//
// UNE OCCASION EST UN COUPLE (vie precedente, vie suivante) au meme emplacement : elle rend un
// ecart si la precedente porte une fin datee, et un MANQUE sinon. Un ecart negatif ou nul est
// lui aussi un manque : une naissance anterieure a la mort qui la precede dans l ordre du temps
// est une contradiction du recensement, pas un cycle de zero seconde.
func vehicleCycleGaps(lives []vehicleCycleLife, secPerFrame float64) ([]float64, int) {
	if len(lives) < 2 {
		return nil, 0
	}
	sort.SliceStable(lives, func(i, j int) bool { return lives[i].t0 < lives[j].t0 })
	gaps := make([]float64, 0, len(lives)-1)
	var manques int
	for i := 1; i < len(lives); i++ {
		fin := lives[i-1].tEnd
		if fin < 0 || lives[i].t0 <= fin {
			manques++
			continue
		}
		gaps = append(gaps, float64(lives[i].t0-fin)*secPerFrame)
	}
	return gaps, manques
}

// dominantVehicleFamily rend la famille la plus representee de l amas — a egalite, la premiere
// dans l ordre alphabetique, pour que deux cuissons du meme film rendent le meme nom.
func dominantVehicleFamily(m map[string]int) string {
	best, bestN := "", 0
	for f, n := range m {
		if n > bestN || (n == bestN && best != "" && f < best) {
			best, bestN = f, n
		}
	}
	return best
}
