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
//	PUBLIÉ    les POWER-UPS de socle, depuis le 2026-08-19 — la voie `ti=37` (`powerup_pads.go`).
//	          LE NÉGATIF DE CORPUS QUI FIGURAIT ICI (« 5 apparitions sur 8 films, toutes lâchées
//	          à une mort, zéro grappe ») ÉTAIT UN ARTEFACT DE LA CHAÎNE QUI LE MESURAIT : il ne
//	          comptait que les poses que `confirmPlacements` retient, c'est-à-dire celles qui ont
//	          une vie DELTA — un socle n'en a aucune. Mesuré sans cet oracle, le socle central de
//	          Catalyst rend 9 et 7 créations au MÊME centimètre sur les deux films KOTH, et zéro
//	          sur les deux films CTF de la même carte (le sous-mode arme le socle).

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
	// Scanned dit que le film a été balayé jusqu'au bout (cf. WorldObjectScan.Scanned).
	Scanned bool `json:"scanned"`
	// Slots est la largeur de la bande de slots de l'archétype ; Anchors le nombre d'en-têtes
	// de création reconnus (bruit compris) ; Accepted ceux que l'oracle de position a validés.
	Slots    int `json:"slots"`
	Anchors  int `json:"anchors"`
	Accepted int `json:"accepted"`
	// Kept / Rejected : les créations acceptées dont l'identité SE RÉSOUT dans le catalogue
	// d'armes, et les autres. L'invariant `Kept + Rejected == Accepted` est testé, et les deux
	// termes sont COMPTÉS chacun sur son chemin — `Rejected` ne se déduit pas d'`Accepted`,
	// sans quoi la somme se vérifierait elle-même (correctif de revue du 2026-08-17).
	//
	// CE FILTRE EST LE GARDE-FOU DU CALQUE : l'acceptation seule ne discrimine pas (une bande
	// FANTÔME de même cardinalité rendait 398 créations acceptées contre 366 pour la vraie
	// bande sur un film), l'identité si — 13 fantômes croisées contre 1 785 réelles.
	Kept     int `json:"kept"`
	Rejected int `json:"rejected"`
	// Objectives est le SOUS-ENSEMBLE de `Rejected` que le manifeste du titre NOMME : les
	// objets d'objectif (le drapeau de CTF), écartés parce qu'on sait ce qu'ils sont.
	//
	// POURQUOI IL SE PUBLIE. Sans lui, un drapeau reconnu et un octet de bruit sortent par la
	// même porte et se comptent dans le même nombre : « 110 écartées » ne dirait pas que 110
	// d'entre elles sont le drapeau, et le jour où la table d'identité du titre serait vide,
	// rien ne le signalerait. Un film d'un autre mode le laisse à zéro, et c'est la mesure.
	Objectives int `json:"objectives"`
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
	// LES QUATRE CHAMPS SUIVANTS SONT LA VOIE `ti=37` — les socles de POWER-UP (schéma 17, cf.
	// powerup_pads.go). Tous les compteurs ci-DESSUS restent ceux de la voie `ti=42` : les
	// mélanger aurait rendu illisible le seul chiffre qui compte quand un calque se tait
	// (« combien de créations cette voie-là a-t-elle lues ? »).
	//
	// LES OCCUPATIONS, ELLES, SONT COMMUNES (`Occupancies` et ses trois issues, `Cycles`) : un
	// socle est un socle, quelle que soit sa nature, et l'invariant `Dated + Unknown + Never ==
	// Occupancies` vaut sur les deux voies réunies. `len(weaponPads) == Pads + PowerupPads`.
	//
	// PowerupScanned dit que le film a été balayé pour `ti=37` jusqu'au bout. Sans lui, un film
	// dont l'archétype est absent des images-clés et un film sans aucun power-up de socle
	// rendraient tous deux zéro, et se liraient pareil.
	PowerupScanned bool `json:"powerupScanned"`
	// PowerupAccepted est le nombre de records de création `ti=37` dont le corps s'est déroulé
	// (masque valide, position décodable) ; PowerupKept ceux dont l'identité `eqip` se résout en
	// famille `powerup_*` du manifeste.
	//
	// L'ÉCART ENTRE LES DEUX EST LE GARDE-FOU : l'en-tête NEW n'est pas sélectif (un quart des
	// positions de bit tirées au hasard passent le test de bande sur un film BTB), et c'est
	// l'identité qui discrimine. `PowerupKept == 0` avec `PowerupAccepted > 0` ne veut PAS dire
	// « pas de power-up » : cela peut aussi vouloir dire que la table de familles du titre est
	// vide, ou que les largeurs du bloc MPP n'ont pas été réinstallées.
	PowerupAccepted int `json:"powerupAccepted"`
	PowerupKept     int `json:"powerupKept"`
	// PowerupPads est le nombre de socles de power-up publiés — le sous-ensemble de
	// `weaponPads` que cette voie porte.
	PowerupPads int `json:"powerupPads"`
}

// Balanced vérifie les deux invariants du calque : toute création acceptée est retenue ou
// écartée, et toute occupation de socle a l'une des trois issues. Une somme fausse signale une
// fuite — un chemin de rejet non compté.
//
// IL PEUT ÉCHOUER, ET C'EST LE POINT (correctif de revue du 2026-08-17). Les deux sommes étaient
// des tautologies : `Rejected` était posé par différence, et les trois statuts d'occupation
// étaient incrémentés dans le même `switch` que le compteur d'occupations. Aucune des deux ne
// pouvait tomber, donc aucune ne contrôlait rien. `Rejected` se compte désormais sur le chemin
// de rejet, et la première somme tombe dès qu'une création acceptée n'atteint pas l'assemblage.
func (c GroundWeaponCoverage) Balanced() bool {
	return c.Kept+c.Rejected == c.Accepted &&
		c.Dated+c.Unknown+c.Never == c.Occupancies &&
		c.PowerupKept <= c.PowerupAccepted
}

// padChainInputs porte ce qu'une voie de la chaîne des socles consomme en plus de sa lecture de
// film et de sa règle (règle des 5 paramètres — ces trois-là voyagent toujours ensemble).
type padChainInputs struct {
	lives     map[uint32][]equipLife
	positions []filmdec.BipedPosition
	clock     replayClock
}

// padChainCounts porte les comptes d'UNE voie. La couverture les VENTILE ensuite : les armes
// champ par champ, les power-ups en trois nombres (cf. GroundWeaponCoverage).
//
// POURQUOI ILS NE S'ÉCRIVENT PAS DIRECTEMENT DANS LA COUVERTURE : les deux voies emploient le
// MÊME code, et un code qui écrirait dans des champs nommés « arme » depuis la voie des
// power-ups mélangerait deux dénominateurs — c'est exactement ce que la couverture existe pour
// empêcher.
type padChainCounts struct {
	accepted, kept, rejected, objectives int
	dropped, spawned, atRest             int
	clusters, pads                       int
}

// buildWeaponPads assemble les socles du match — LES DEUX NATURES — et les occupations achevées.
//
// LES DEUX VOIES ABOUTISSENT AU MÊME TABLEAU, et l'ordre est celui-ci : les ARMES d'abord, les
// POWER-UPS ensuite. Il est arbitraire mais il doit être STABLE, parce que `padPickups[].pad`
// est un index dans ce tableau — et c'est pourquoi les occupations de la seconde voie sont
// décalées du nombre de socles de la première.
//
// `positions` doit être TRIÉ par instant (c'est le cas de `sorted` dans BuildFromPositions) : la
// datation d'une disparition est une recherche dichotomique sur cette suite.
// LE QUATRIÈME RETOUR est la liste des OBJETS INDIVIDUELS de la voie des armes, telle que
// `padObjects` l'a bornée et datée. Les socles n'en publient que les grappes récurrentes ; le
// calque des armes au sol (`document_ground_weapon_items.go`, schéma 27) publie les vies
// individuelles — les deux consommateurs partagent LA MÊME chaîne au lieu d'en dérouler deux.
func buildWeaponPads(
	scans PadScans, positions []filmdec.BipedPosition, clock replayClock, cat padCatalogs,
) ([]WeaponPad, []PadPickup, *GroundWeaponCoverage, []gwPickupObject) {
	cov := &GroundWeaponCoverage{
		Scanned: scans.Weapons.Scanned, Slots: scans.Weapons.Stats.Slots,
		Anchors: scans.Weapons.Stats.Anchors, Accepted: scans.Weapons.Stats.Accepted,
		PowerupScanned: scans.Powerups.Scanned, PowerupAccepted: scans.Powerups.Stats.Accepted,
	}
	if clock.step == 0 {
		return nil, nil, cov, nil
	}
	in := padChainInputs{lives: equipmentLives(positions), positions: positions, clock: clock}
	pads, picks, wc, wObjs := buildPadChain(scans.Weapons, weaponPadRule(cat.ObjectiveObjects), in, cov)
	cov.Kept, cov.Rejected, cov.Objectives = wc.kept, wc.rejected, wc.objectives
	cov.Dropped, cov.Spawned, cov.AtRest = wc.dropped, wc.spawned, wc.atRest
	cov.Clusters, cov.Pads = wc.clusters, wc.pads
	pu, puPicks, pc, _ := buildPadChain(
		scans.Powerups, powerupPadRule(cat.EquipmentFamilies), in, cov)
	cov.PowerupKept, cov.PowerupPads = pc.kept, pc.pads
	for i := range puPicks {
		puPicks[i].Pad += len(pads)
	}
	out := append(pads, pu...)
	if len(out) == 0 {
		return nil, nil, cov, wObjs
	}
	return out, append(picks, puPicks...), cov, wObjs
}

// buildPadChain déroule UNE voie : identité, classement, grappes, socles, occupations.
//
// `cov` ne reçoit ici QUE les compteurs d'occupation, qui sont communs aux deux voies ; tout le
// reste sort par `padChainCounts`, que l'appelant ventile.
func buildPadChain(
	scan WorldObjectScan, rule padRule, in padChainInputs, cov *GroundWeaponCoverage,
) ([]WeaponPad, []PadPickup, padChainCounts, []gwPickupObject) {
	var n padChainCounts
	if !scan.Scanned {
		return nil, nil, n, nil
	}
	n.accepted = scan.Stats.Accepted
	// `rejected` est COMPTÉ sur le chemin de rejet, jamais déduit d'`Accepted` : c'est ce qui
	// fait de l'invariant `Kept + Rejected == Accepted` un contrôle et non une tautologie.
	objs, rejected := padObjects(scan, rule, in.lives, in.positions)
	n.kept, n.rejected, n.objectives = len(objs), rejected.total, rejected.objectives
	atRest, src := gwAtRestOf(objs, &n)
	pads, assign := gwPadsClusterAssign(atRest)
	n.clusters = len(pads)
	members := map[int][]int{}
	for j := range atRest {
		members[assign[j]] = append(members[assign[j]], src[j])
	}
	// LE FILTRE « UNE GRAPPE DE >= 2 APPARITIONS EST UN SOCLE » EST CELUI DE `gwPadsKeep`, jamais
	// une seconde écriture du seuil ici (correctif de revue du 2026-08-17). `padSrc` ramène chaque
	// socle retenu à son index d'avant filtre — celui sur lequel ses membres sont indexés.
	keep, padSrc := gwPadsKeep(pads)
	out := make([]WeaponPad, 0, len(keep))
	var picks []PadPickup
	for p := range keep {
		pad, padPicks := gwBuildPad(keep[p], objs, members[padSrc[p]], in.clock, cov)
		for i := range padPicks {
			padPicks[i].Pad = len(out)
		}
		out, picks = append(out, pad), append(picks, padPicks...)
	}
	n.pads = len(out)
	return out, picks, n, objs
}

// gwAtRestOf sélectionne les apparitions « apparues au repos » — `spawned` SANS vie delta — et
// rend, pour chacune, l'index de l'objet dont elle vient.
//
// POURQUOI CE JEU ET PAS `spawned` TEL QUEL : le témoin du plan (« le nombre de grappes est
// petit et stable, 6 à 12 sur une arène ») tient sur `at_rest` — 6 à 10 socles sur quatre
// cartes — et PAS sur `spawned` littéral, qui va de 1 à 21. La différence est mesurée et
// nommée : 280 apparitions sur 1 790 sont `spawned` tout en ayant bougé, presque toutes des
// `MA40 AR` / `Mk51 Sidekick` — l'arme de départ qu'un joueur lâche en ramassant autre chose.
//
// C'EST AUSSI CE QUI ÉCARTE LE POWER-UP LÂCHÉ À UNE MORT (2026-08-19), et par les DEUX critères
// à la fois : il naît là où son porteur meurt (donc `dropped`) et il tombe (donc vie delta). Il
// reste publié par `equipmentPlacements` avec son origine ; il n'est pas un socle.
func gwAtRestOf(objs []gwPickupObject, n *padChainCounts) ([]gwPadApparition, []int) {
	app, src := make([]gwPadApparition, 0, len(objs)), make([]int, 0, len(objs))
	for i, o := range objs {
		if o.Appar.Class == gwClassDropped {
			n.dropped++
			continue
		}
		n.spawned++
		if o.Appar.HasDelta {
			continue
		}
		n.atRest++
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
	ms := gwMembersByTime(objs, members)
	// LE DÉNOMINATEUR SE COMPTE À PART DES TERMES, et le `switch` qui suit n'a PAS de branche
	// attrape-tout : sans ces deux précautions, `Dated + Unknown + Never == Occupancies` était
	// vrai par construction et ne contrôlait rien. Un statut inattendu déséquilibre désormais la
	// couverture au lieu d'être silencieusement compté comme « sans passage ».
	cov.Occupancies += len(ms)
	for _, i := range ms {
		o := objs[i]
		low, high := gwFrameOf(o.Bounds.LowUS, clock), gwFrameOf(o.Bounds.HighUS, clock)
		out.Spawns = append(out.Spawns, gwFrameOf(o.Appar.TUS, clock))
		out.Presence = append(out.Presence, PadPresence{
			T0: gwFrameOf(o.Appar.TUS, clock), TLow: low, THigh: high,
		})
		switch o.Status {
		case gwPickupStatusNever:
			cov.Never++
			continue // le socle ne s'est jamais vidé : aucune occupation achevée
		case gwPickupStatusDated:
			cov.Dated++
		case gwPickupStatusUnknown:
			cov.Unknown++
		}
		picks = append(picks, PadPickup{TLow: low, THigh: high})
	}
	// `manques` VOYAGE AVEC LES ÉCARTS : ce sont les réapparitions dont la disparition
	// précédente n'est pas datée, c'est-à-dire les écarts que ce socle offrait et que la mesure
	// n'a pas pu prendre. Les jeter faisait lire « 2 écarts » comme « 2 sur 2 ».
	gaps, manques := gwPickupPadGaps(objs, members)
	if c := gwPadsCycleFromGaps(gaps); c.Established {
		out.Cycle = &PadCycle{
			MedianS: round2(float32(c.MedianS)), P10S: round2(float32(c.P10S)),
			P90S: round2(float32(c.P90S)), Gaps: c.Gaps, Missing: manques,
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
// UN POWER-UP PUBLIE SA FAMILLE, PAS SON IDENTIFIANT `eqip` (2026-08-19), et ce n'est pas une
// commodité : `weaponPads[].weapon` est la clé que le client joint à `weaponLabels`, une table
// d'ARMES où aucun identifiant d'équipement n'entre. La famille du manifeste
// (`powerup_overshield`) est en revanche le vocabulaire que la règle de taille du calque
// connaît déjà (`POWER_PAD_KEYS`). Publier l'hexadécimal aurait rendu un socle que rien ne
// nomme et que rien n'agrandit.
func gwPadWeaponID(objs []gwPickupObject, members []int) string {
	ms := gwMembersByTime(objs, members)
	if len(ms) == 0 {
		return ""
	}
	if o := objs[ms[0]]; o.Appar.Kind == gwPadKindPowerup {
		return o.Appar.Family
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
		"retenues", c.Kept, "ecartees", c.Rejected, "objetsDObjectif", c.Objectives,
		"lachees", c.Dropped, "apparues", c.Spawned, "auRepos", c.AtRest,
		"grappes", c.Clusters, "socles", c.Pads, "occupations", c.Occupancies,
		"datees", c.Dated, "sansPassage", c.Unknown, "jamaisVidees", c.Never,
		"cyclesEtablis", c.Cycles)
	// LA VOIE `ti=37` A SA PROPRE LIGNE, et il le faut : ses dénominateurs sont ceux d'un AUTRE
	// balayage. Les fondre dans la ligne ci-dessus aurait rendu illisible le seul rapport qui
	// dit si la lecture a réussi — acceptées contre retenues par l'identité.
	slog.Info("rejeu : socles de power-up",
		"balaye", c.PowerupScanned, "acceptees", c.PowerupAccepted,
		"retenues", c.PowerupKept, "socles", c.PowerupPads)
	// LE SILENCE QU'IL FAUT ROMPRE : des créations acceptées dont AUCUNE ne résout d'arme n'est
	// pas « un film sans socle », c'est une lecture qui a échoué en bloc — largeurs du bloc MPP
	// non réinstallées, ou grammaire de l'état par défaut qui a bougé. Sans ce warn, un film BTB
	// entier sortait avec zéro socle sans que rien ne le signale.
	if c.Kept == 0 && c.Accepted > 0 {
		slog.Warn("rejeu : identite ti=42 non resolue sur AUCUNE creation — largeurs MPP ?",
			"acceptees", c.Accepted, "retenues", c.Kept, "ancres", c.Anchors)
	}
}
