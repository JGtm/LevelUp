package replay

// ground_weapon_pads.go — LES SOCLES D'ARME du match : où une arme réapparaît, quand elle y est,
// et quand le socle redevient vide.
//
// CE QUE LE CALQUE PUBLIE, ET CE QU'IL REFUSE DE PUBLIER (plan
// `.ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md`, arbitrages des 2026-08-17) :
//
//	PUBLIÉ    les SOCLES du match — une position où des armes de MÊME famille réapparaissent
//	          (>= 2 fois, à moins d'un mètre) ; leurs apparitions ; l'intervalle de présence
//	          borné par le recensement des images-clés ; le cycle mesuré DEPUIS la disparition
//	          quand il est stable. Et les OCCUPATIONS qui se sont achevées (`padPickups`).
//
//	REFUSÉ    le RAMASSEUR. L'oracle des loadouts a été mesuré deux fois, par slot de vie puis
//	          par joueur (pont `SlotXUID` du constructeur) : 88,1 % puis 79,7 % là où il peut
//	          parler, contre >= 90 % exigé, et le seuil n'a pas été rebaissé. `PadPickup.XUID`
//	          existe, vaut `null` partout, et le champ dit pourquoi.
//
//	REFUSÉ    les « ramassages » d'armes LÂCHÉES à une mort. La mesure les a séparés des socles :
//	          l'accord de l'oracle y passe SOUS son propre témoin (32,1 % contre 65,0 %), ce qui
//	          est la signature d'un critère qui ne mesure rien — une arme lâchée à une mort
//	          disparaît le plus souvent toute seule (despawn). Elles ne sont pas publiées.
//
//	REFUSÉ    un CATALOGUE de carte. Le critère tranché du plan (>= 80 % de recouvrement, même
//	          famille, < 1 m, dans les deux sens) n'est atteint par aucune des trois paires de
//	          films mesurées (meilleure : 70,0 %). Ce que la mesure dit est plus fin : les DIX
//	          socles de Catalyst sont aux dix mêmes coordonnées AU CENTIMÈTRE dans deux films,
//	          mais trois portent une arme différente. **Le socle appartient à la carte, l'arme
//	          qui y apparaît appartient au match** — d'où une publication PAR MATCH.
//
//	REFUSÉ    les POWER-UPS de socle : négatif de corpus. 5 apparitions de power-up sur 8 films,
//	          toutes `powerup_overshield`, toutes lâchées à une mort, toutes avec une vie delta.
//	          Zéro grappe, zéro socle sur les cinq cartes natives d'arène du jeu de films.

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// WeaponPad est UN socle d'arme du match : une position où une arme de la même famille
// réapparaît, avec ses apparitions, ses intervalles de présence et son cycle quand il est établi.
//
// UNE DONNÉE DE MATCH, PAS DE CARTE : la position est celle du match mesuré (elle se retrouve au
// centimètre d'un film à l'autre), la FAMILLE ne l'est pas — deux matchs de la même carte y font
// apparaître deux armes différentes.
type WeaponPad struct {
	// X / Y : la position du socle, en coordonnées monde (mêmes axes que Point.X/Y) — le
	// CENTROÏDE des apparitions agglomérées. Z : l'altitude, gratuite (le même record la
	// porte) et publiée pour la cohérence d'étage.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// Weapon est la FAMILLE d'arme (high-32 du weapon-id), en hexadécimal — même écriture que
	// `Loadout.W`, donc même clé dans `weaponLabels`. Le tag brut reste À CÔTÉ du libellé,
	// jamais à sa place : une famille absente de la table garde son hexadécimal à l'écran et
	// n'emprunte pas le nom d'une arme voisine.
	Weapon string `json:"weapon"`
	// Spawns porte les instants d'APPARITION d'une arme sur ce socle, en frames (même axe que
	// Point.T), triés.
	Spawns []int `json:"spawns"`
	// Presence porte, pour chaque apparition, ce que le film permet d'affirmer de sa présence.
	Presence []PadPresence `json:"presence"`
	// Cycle est le délai de réapparition du socle, mesuré DEPUIS la disparition précédente.
	// ABSENT (`null`) quand il n'est pas ÉTABLI — jamais un chiffre instable publié comme s'il
	// était stable. C'est la décision 3 du plan, et elle coûte : 24 socles sur 57 seulement
	// portent un cycle.
	Cycle *PadCycle `json:"cycle,omitempty"`
}

// PadPresence est UNE occupation du socle : de l'apparition de l'arme à sa disparition, telle
// que le recensement des images-clés la BORNE.
//
// TROIS INSTANTS, ET LEUR SENS N'EST PAS LE MÊME. `T0` est mesuré (le record de création le
// porte). `TLow` est le dernier instant où l'arme est PROUVÉE présente ; `THigh` le premier où
// son absence est prouvée. Entre les deux, le film ne dit rien : les images-clés sont espacées
// de ~20 s (médiane mesurée 20,00 s), et ce calque ne prétend pas être plus fin que sa source.
// Un rendu honnête montre le socle vide dès `TLow` et incertain jusqu'à `THigh`.
type PadPresence struct {
	// T0 : l'apparition, en frames.
	T0 int `json:"t0"`
	// TLow : dernier instant où l'arme est prouvée présente, en frames.
	TLow int `json:"tLow"`
	// THigh : premier instant où son absence est prouvée, en frames. Vaut la fin du rejeu quand
	// l'arme est encore recensée à la DERNIÈRE image-clé du film — elle n'a alors pas été prise,
	// ou elle l'a été après, ce que rien ne dit.
	THigh int `json:"tHigh"`
}

// PadCycle est le délai de réapparition d'un socle, en secondes, mesuré du moment où il se vide
// à la réapparition suivante.
//
// POURQUOI DEPUIS LA DISPARITION ET NON DEPUIS L'APPARITION : l'horloge d'un socle repart quand
// on prend l'arme, pas quand elle apparaît. La mesure le tranche — 24 socles au cycle établi
// contre 4 pour l'horloge d'apparition, aux MÊMES règles de stabilité, et un pic mesuré à 30,5 s
// (55 écarts sur 142 dans la tranche 30-35 s, tenant dans 0,34 s) que l'horloge d'apparition ne
// montre pas. Les armes de puissance en sortent d'elles-mêmes : S7 Sniper 114 à 134 s, Energy
// Sword 194,5 s, Needler 100,9 s.
type PadCycle struct {
	// MedianS / P10S / P90S : la médiane et les déciles des écarts mesurés, en secondes.
	MedianS float32 `json:"medianS"`
	P10S    float32 `json:"p10S"`
	P90S    float32 `json:"p90S"`
	// Gaps est le nombre d'écarts mesurés — le dénominateur. Un cycle n'est ÉTABLI qu'à partir
	// de deux : un écart unique n'a pas d'écart-type, et rien ne dit qu'il se répète.
	Gaps int `json:"gaps"`
}

// PadPickup est une occupation de socle qui S'EST ACHEVÉE : le socle s'est vidé quelque part
// dans [TLow, THigh].
//
// C'EST UN INTERVALLE, PAS UN INSTANT, et c'est délibéré. Le film ne porte aucun événement de
// ramassage (mesuré) et le record de suppression d'entité n'est pas isolable : la seule preuve
// de disparition est le recensement des images-clés, espacé de ~20 s. Publier un instant
// donnerait une précision que la source n'a pas.
type PadPickup struct {
	// Pad est l'index du socle dans `weaponPads`.
	Pad int `json:"pad"`
	// TLow / THigh : les bornes de la disparition, en frames (même axe que Point.T).
	TLow  int `json:"tLow"`
	THigh int `json:"tHigh"`
	// XUID est le joueur qui a pris l'arme — TOUJOURS `null` aujourd'hui, et le champ existe
	// pour dire que la question a été posée et tranchée, pas oubliée.
	//
	// POURQUOI IL EST VIDE. L'oracle indépendant du ramassage est le loadout d'image-clé du
	// ramasseur présumé : porte-t-il l'arme du socle à l'image-clé suivante ? Mesuré deux fois,
	// sur la seule population que la mesure qualifie (les socles) : 111/126 = 88,1 % en suivant
	// le SLOT de vie, 102/128 = 79,7 % en suivant le JOUEUR à travers ses réapparitions (pont
	// `SlotXUID` du constructeur). Le seuil du plan était >= 90 % et il n'a pas été rebaissé.
	//
	// CE QUI LE LÈVERAIT n'est pas un meilleur pont — celui-ci nomme 94 % des ramasseurs, avec
	// zéro collision de slot sur huit films — mais un oracle plus RAPPROCHÉ que 20 s :
	// l'inventaire lu dans le flux delta, ou toute source qui observe le porteur avant qu'il
	// puisse mourir. Un joueur qui meurt dans les 20 s lâche ce qu'il vient de prendre, et
	// l'image-clé suivante montre son loadout de réapparition. Condition de reprise au
	// registre `.ai/V7.5/REGISTRE_REPORTS.md`.
	//
	// POINTEUR, ET SANS `omitempty` : le champ doit se VOIR à `null`. Un client qui ne le
	// trouve pas pourrait croire qu'il a affaire à un artefact plus ancien.
	XUID *string `json:"xuid"`
}

// GroundWeaponCoverage dit ce que le calque a lu, ce qu'il a retenu, et ce qu'il a écarté.
//
// SANS ELLE, ZÉRO SOCLE NE SE LIT PAS. Un film sans arme au sol lisible, un film dont toutes les
// apparitions sont des armes lâchées à une mort (Super Fiesta : 82,3 % de lâchers, zéro socle) et
// un film qu'on n'a pas su balayer rendent tous trois zéro socle — seuls ces compteurs les
// distinguent.
type GroundWeaponCoverage struct {
	// Scanned dit que le film a été balayé jusqu'au bout (cf. GroundWeaponScan.Scanned).
	Scanned bool `json:"scanned"`
	// Slots est la largeur de la bande de slots de l'archétype ; Anchors le nombre d'en-têtes
	// de création reconnus (bruit compris) ; Accepted ceux que l'oracle de position a validés.
	Slots    int `json:"slots"`
	Anchors  int `json:"anchors"`
	Accepted int `json:"accepted"`
	// Kept / Rejected : les créations acceptées dont l'identité SE RÉSOUT dans le catalogue
	// d'armes, et les autres. L'invariant `Kept + Rejected == Accepted` est testé.
	//
	// CE FILTRE EST LE GARDE-FOU DU CALQUE : l'acceptation seule ne discrimine pas (une bande
	// FANTÔME de même cardinalité rendait 398 créations acceptées contre 366 pour la vraie
	// bande sur un film), l'identité si — 13 fantômes croisées contre 1 785 réelles.
	Kept     int `json:"kept"`
	Rejected int `json:"rejected"`
	// Dropped / Spawned : les apparitions retenues, classées par la règle du lâcher (une vie de
	// bipède s'achève à moins de 2 frames et 1,5 m). AtRest en est le sous-ensemble sans vie
	// delta — les candidats « apparus au repos », d'où sortent les socles.
	Dropped int `json:"dropped"`
	Spawned int `json:"spawned"`
	AtRest  int `json:"atRest"`
	// Clusters est le nombre de grappes formées ; Pads celles qui récurrent (>= 2 apparitions)
	// et sont donc publiées comme socles.
	Clusters int `json:"clusters"`
	Pads     int `json:"pads"`
	// Occupancies est le nombre d'apparitions PORTÉES par un socle publié — le dénominateur des
	// trois suivants. Dated : un joueur est passé à moins de 1,5 m dans l'intervalle de
	// disparition. Unknown : personne n'est passé, la disparition reste bornée sans plus.
	// Never : l'arme est encore recensée à la dernière image-clé du film.
	//
	// LA DISTINCTION EST PUBLIÉE MÊME SI L'ARTEFACT NE PUBLIE PAS LA DATATION : c'est elle qui
	// dit sur quelle proportion des occupations le cycle a pu se mesurer.
	Occupancies int `json:"occupancies"`
	Dated       int `json:"dated"`
	Unknown     int `json:"unknown"`
	Never       int `json:"never"`
	// Cycles est le nombre de socles dont le cycle est ÉTABLI, donc publié. Les autres portent
	// `cycle: null`.
	Cycles int `json:"cycles"`
}

// Balanced vérifie les deux invariants du calque : toute création acceptée est retenue ou
// écartée, et toute occupation de socle a l'une des trois issues. Une somme fausse signale une
// fuite — un chemin de rejet non compté.
func (c GroundWeaponCoverage) Balanced() bool {
	return c.Kept+c.Rejected == c.Accepted &&
		c.Dated+c.Unknown+c.Never == c.Occupancies
}

// buildWeaponPads assemble les socles du match et les occupations achevées.
//
// `positions` doit être TRIÉ par instant (c'est le cas de `sorted` dans BuildFromPositions) : la
// datation d'une disparition est une recherche dichotomique sur cette suite.
func buildWeaponPads(
	scan GroundWeaponScan, positions []filmdec.BipedPosition, clock replayClock,
) ([]WeaponPad, []PadPickup, *GroundWeaponCoverage) {
	cov := &GroundWeaponCoverage{
		Scanned: scan.Scanned, Slots: scan.Stats.Slots,
		Anchors: scan.Stats.Anchors, Accepted: scan.Stats.Accepted,
	}
	if !scan.Scanned || clock.step == 0 {
		return nil, nil, cov
	}
	objs := groundWeaponObjects(scan, equipmentLives(positions), positions)
	cov.Kept, cov.Rejected = len(objs), scan.Stats.Accepted-len(objs)
	atRest, src := gwAtRestOf(objs, cov)
	pads, assign := gwPadsClusterAssign(atRest, gwPadRadiusM)
	cov.Clusters = len(pads)
	members := map[int][]int{}
	for j := range atRest {
		members[assign[j]] = append(members[assign[j]], src[j])
	}
	out := make([]WeaponPad, 0, len(pads))
	var picks []PadPickup
	for p := range pads {
		if len(pads[p].TS) < gwPadMinHits {
			continue
		}
		pad, padPicks := gwBuildPad(pads[p], objs, members[p], clock, cov)
		for i := range padPicks {
			padPicks[i].Pad = len(out)
		}
		out, picks = append(out, pad), append(picks, padPicks...)
	}
	cov.Pads = len(out)
	if len(out) == 0 {
		return nil, nil, cov
	}
	return out, picks, cov
}

// gwAtRestOf sélectionne les apparitions « apparues au repos » — `spawned` SANS vie delta — et
// rend, pour chacune, l'index de l'objet dont elle vient.
//
// POURQUOI CE JEU ET PAS `spawned` TEL QUEL : le témoin du plan (« le nombre de grappes est
// petit et stable, 6 à 12 sur une arène ») tient sur `at_rest` — 6 à 10 socles sur quatre
// cartes — et PAS sur `spawned` littéral, qui va de 1 à 21. La différence est mesurée et
// nommée : 280 apparitions sur 1 790 sont `spawned` tout en ayant bougé, presque toutes des
// `MA40 AR` / `Mk51 Sidekick` — l'arme de départ qu'un joueur lâche en ramassant autre chose.
func gwAtRestOf(objs []gwPickupObject, cov *GroundWeaponCoverage) ([]gwPadApparition, []int) {
	app, src := make([]gwPadApparition, 0, len(objs)), make([]int, 0, len(objs))
	for i, o := range objs {
		if o.Appar.Class == gwClassDropped {
			cov.Dropped++
			continue
		}
		cov.Spawned++
		if o.Appar.HasDelta {
			continue
		}
		cov.AtRest++
		app, src = append(app, o.Appar), append(src, i)
	}
	return app, src
}

// gwBuildPad rend UN socle publié et les occupations achevées qui lui appartiennent.
func gwBuildPad(
	pad gwPadCluster, objs []gwPickupObject, members []int, clock replayClock,
	cov *GroundWeaponCoverage,
) (WeaponPad, []PadPickup) {
	out := WeaponPad{X: pad.X, Y: pad.Y, Z: pad.Z, Weapon: gwPadWeaponID(objs, members)}
	var picks []PadPickup
	for _, i := range gwMembersByTime(objs, members) {
		o := objs[i]
		low, high := gwFrameOf(o.Bounds.LowUS, clock), gwFrameOf(o.Bounds.HighUS, clock)
		out.Spawns = append(out.Spawns, gwFrameOf(o.Appar.TUS, clock))
		out.Presence = append(out.Presence, PadPresence{
			T0: gwFrameOf(o.Appar.TUS, clock), TLow: low, THigh: high,
		})
		cov.Occupancies++
		switch o.Status {
		case gwPickupStatusNever:
			cov.Never++
			continue // le socle ne s'est jamais vidé : aucune occupation achevée
		case gwPickupStatusDated:
			cov.Dated++
		default:
			cov.Unknown++
		}
		picks = append(picks, PadPickup{TLow: low, THigh: high})
	}
	gaps, _ := gwPickupPadGaps(objs, members)
	if c := gwPadsCycleFromGaps(gaps); c.Established {
		out.Cycle = &PadCycle{
			MedianS: round2(float32(c.MedianS)), P10S: round2(float32(c.P10S)),
			P90S: round2(float32(c.P90S)), Gaps: c.Gaps,
		}
		cov.Cycles++
	}
	return out, picks
}

// gwMembersByTime rend les membres d'un socle dans l'ordre de leur apparition.
func gwMembersByTime(objs []gwPickupObject, members []int) []int {
	ms := append([]int(nil), members...)
	sort.Slice(ms, func(i, j int) bool { return objs[ms[i]].Appar.TUS < objs[ms[j]].Appar.TUS })
	return ms
}

// gwPadWeaponID rend l'identifiant de famille publié pour le socle : celui de sa PREMIÈRE
// apparition.
//
// UN SEUL IDENTIFIANT POUR UNE GRAPPE QUI EN PORTE PEUT-ÊTRE DEUX : la grappe est keyée sur le
// NOM CANONIQUE de l'arme (alias repliés), et un même canon apparaît sous deux identifiants
// distincts. Prendre celui de la première apparition rend le choix déterministe — l'ordre des
// membres est total — et les deux identifiants nomment la même arme dans `weaponLabels`.
func gwPadWeaponID(objs []gwPickupObject, members []int) string {
	ms := gwMembersByTime(objs, members)
	if len(ms) == 0 {
		return ""
	}
	return formatWeaponFamily(objs[ms[0]].FamilyID)
}

// gwFrameOf projette un instant sur l'axe de frames du document, en le RAMENANT dans l'axe.
//
// LE CLAMP EST VOULU, ET IL DIFFÈRE DES POSES : une pose hors de l'axe est un événement qu'on
// n'a rien à dessiner, donc on l'écarte. Un socle est un LIEU : l'écarter parce que sa dernière
// apparition tombe après le dernier paquet de position effacerait un socle que la mesure a bel
// et bien vu. On garde le socle et on ramène l'instant à la borne de l'axe.
func gwFrameOf(atUS uint64, clock replayClock) int {
	return clampFrame(frameOf(atUS, clock.origin, clock.step), clock.frames)
}

// logGroundWeaponCoverage publie au journal ce que le calque a rendu — les mêmes dénominateurs
// que l'artefact, pour qu'un build se juge sans ouvrir le JSON.
func logGroundWeaponCoverage(c *GroundWeaponCoverage) {
	if c == nil {
		return
	}
	slog.Info("rejeu : socles d'arme au sol",
		"balaye", c.Scanned, "ancres", c.Anchors, "acceptees", c.Accepted,
		"retenues", c.Kept, "ecartees", c.Rejected,
		"lachees", c.Dropped, "apparues", c.Spawned, "auRepos", c.AtRest,
		"grappes", c.Clusters, "socles", c.Pads, "occupations", c.Occupancies,
		"datees", c.Dated, "sansPassage", c.Unknown, "jamaisVidees", c.Never,
		"cyclesEtablis", c.Cycles)
}
