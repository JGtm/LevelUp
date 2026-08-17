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

// LES QUATRE TYPES PUBLIÉS DU CALQUE (`WeaponPad`, `PadPresence`, `PadCycle`, `PadPickup`)
// vivent dans `document_ground_weapons.go` : c'est la FORME de l'artefact, avec la chronique du
// schéma 11. Ce fichier-ci porte la RÈGLE qui la remplit.

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
	pads, assign := gwPadsClusterAssign(atRest)
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
	picks := make([]PadPickup, 0, len(members))
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
	// LE SILENCE QU'IL FAUT ROMPRE : des créations acceptées dont AUCUNE ne résout d'arme n'est
	// pas « un film sans socle », c'est une lecture qui a échoué en bloc — largeurs du bloc MPP
	// non réinstallées, ou grammaire de l'état par défaut qui a bougé. Sans ce warn, un film BTB
	// entier sortait avec zéro socle sans que rien ne le signale.
	if c.Kept == 0 && c.Accepted > 0 {
		slog.Warn("rejeu : identite ti=42 non resolue sur AUCUNE creation — largeurs MPP ?",
			"acceptees", c.Accepted, "retenues", c.Kept, "ancres", c.Anchors)
	}
}
