package replay

import (
	"log/slog"
	"sort"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// abilities.go — LA CAPACITÉ D'ARMURE PORTÉE, portée à la grille du rejeu.
//
// UNE SEULE GRANDEUR, DEUX SOURCES — et c'est tout le sujet de ce fichier. Le film dit deux
// fois quelle capacité un joueur porte, par deux chemins sans rapport l'un avec l'autre :
//
//	i48  `biped-desired-ability-set-component`, dans les paquets DELTA. Il transmet le RANG
//	     COMPLET dans la palette du match (R(6) après une porte). Rare — 0,03 à 0,09 % des
//	     records, à peu près une fois par vie — mais lu à 100 % : 748 lectures sur 8 films,
//	     zéro illisible. Cf. filmdec/ability_rank.go.
//	kf   le champ de 3 bits ancré dans les IMAGES-CLÉS (inventory_decode.go, règle R1).
//	     Dense (une lecture par joueur et par image-clé) mais BORGNE : son motif d'ancrage
//	     se termine par `010`, qui sont les bits de POIDS FORT du rang. Il ne voit donc que
//	     la fenêtre 16..23 de la palette, et rien d'autre.
//
// POURQUOI LES DEUX, ET POURQUOI DANS LE MÊME CHAMP. Jusqu'au 2026-08-14, le canal
// d'image-clé publiait `rang − 16` sous le nom d'« index de capacité » : une grandeur
// DIFFÉRENTE du rang, sous un nom qui ne le disait pas. C'est la cause, enfin nommée, de
// trois anomalies — les valeurs ne sortaient jamais de 3-7, 21 films sur 40 ne rendaient
// aucune lecture (aucun joueur n'y portait un rang 16-23), et les films « où les huit
// joueurs portent le même équipement » étaient un artefact. La correction n'est pas de
// choisir un canal : c'est de les ramener à LA MÊME grandeur — le rang — et de dire, sur
// chaque lecture, d'où elle vient. Deux canaux qui rendent des grandeurs différentes sous le
// même nom, c'est le défaut qui a coûté ce chantier ; deux canaux qui rendent la même
// grandeur en le disant, c'est une couverture.
//
// CE QUI N'EST PAS ICI, et qui est la demande d'écran non satisfaite : l'ÉTAT ACTIF. Savoir
// qu'un joueur PORTE un camouflage ne dit pas qu'il l'a DÉCLENCHÉ. La source d'état (`i57`)
// est lue sur 0,82 % des records et s'associe aux épisodes d'`i54` à 72,2 % contre 34 % de
// témoin — une erreur sur quatre. Tant qu'elle n'est pas fiable, aucun effet plein-fiche ne
// se code : le document publie l'IDENTITÉ, pas l'ÉTAT.

// Sources de lecture publiées sur AbilityRead.Src. Elles ne sont pas décoratives : la
// fenêtre de visibilité des deux canaux n'est pas la même, et un lecteur qui voudrait juger
// une couverture doit pouvoir les séparer.
const (
	// AbilitySrcI48 : paquet delta, rang complet sur toute la palette.
	AbilitySrcI48 = "i48"
	// AbilitySrcKeyframe : image-clé, fenêtre 16..23 UNIQUEMENT.
	AbilitySrcKeyframe = "kf"
)

// AbilityRead est UNE lecture de la capacité d'armure portée par un slot.
type AbilityRead struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée — donc une VIE, pas un joueur.
	Slot uint32 `json:"slot"`
	// R est le RANG dans la palette du match. Ce n'est PAS un index de capacité universel :
	// le même rang désigne des capacités différentes d'une palette à l'autre, et c'est
	// AbilityLabels — construite pour la palette de CE film — qui le nomme, ou pas.
	R int `json:"r"`
	// Src dit par quel canal la lecture est arrivée (AbilitySrcI48 / AbilitySrcKeyframe).
	// Publié parce que les deux canaux ne voient pas la même chose : une lecture `kf` est
	// forcément dans 16..23, une lecture `i48` peut valoir n'importe quel rang.
	Src string `json:"src"`
}

// buildAbilityReads projette les deux canaux sur la grille de frames du rejeu.
//
// Les lectures ANTÉRIEURES à l'origine du rejeu sont écartées : elles n'ont pas de place sur
// l'axe, et leur en inventer une les poserait sur la première image comme si elles y avaient
// été mesurées.
//
// AUCUNE FUSION, AUCUN ARBITRAGE : les deux canaux disent la même grandeur, on publie les
// deux lectures telles quelles. Départager deux mesures concordantes n'apporterait rien ;
// les départager quand elles divergent supposerait de savoir laquelle a tort, ce qu'on ne
// sait pas — et le client, lui, prend simplement la plus récente.
func buildAbilityReads(
	ranks []filmdec.AbilityRank, inv []KeyframeInventory, origin, step uint64,
) []AbilityRead {
	out := make([]AbilityRead, 0, len(ranks)+len(inv))
	for _, r := range ranks {
		if r.TimestampUS < origin {
			continue
		}
		out = append(out, AbilityRead{
			T: int((r.TimestampUS - origin) / step), Slot: r.Slot, R: r.Rank, Src: AbilitySrcI48,
		})
	}
	for _, r := range inv {
		if r.TimestampUS < origin || r.AbilityRank < 0 {
			continue
		}
		out = append(out, AbilityRead{
			T: int((r.TimestampUS - origin) / step), Slot: r.Slot, R: r.AbilityRank,
			Src: AbilitySrcKeyframe,
		})
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T != out[j].T {
			return out[i].T < out[j].T
		}
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Src < out[j].Src
	})
	return out
}

// keepAbilitiesOfPublishedTracks écarte les lectures dont le slot n'a pas de trajectoire
// publiée : le client n'aurait aucune fiche où les poser.
func keepAbilitiesOfPublishedTracks(reads []AbilityRead, tracks []Track) []AbilityRead {
	return keepOfPublishedTracks(reads, tracks,
		func(a AbilityRead, published map[uint32]bool) bool { return published[a.Slot] })
}

// abilityRankDomainMax : rang maximal PLAUSIBLE dans une palette du titre — au-delà, une
// lecture n'est plus une « capacité inconnue » mais du BRUIT DE BALAYAGE bit-à-bit
// (RAPPORT_E0_2026-09-10 §3, même signature que l'unique rang 44 de `084a804d` déjà consigné
// ci-dessous en §CLASSEMENT DE PALETTE). Les blocs `sofd` du jeu comptent au plus ~27
// entrées (mesure indépendante par inversion, aucun marqueur de groupe de tags dans le
// registre) : un rang plus grand ne peut correspondre à AUCUNE capacité, connue ou future,
// de ce titre — le motif d'en-tête de record a coïncidé avec autre chose et R(6) a lu une
// valeur arbitraire.
//
// MESURÉ SUR LES 64 FILMS DU PARC (2026-09-10) : EXACTEMENT deux lectures dépassent ce
// plafond sur les 4 952 lectures i48+kf du corpus — rang 32 (`9ffce8ef`, slot 530, image
// 6140, 8 min 47 s après la SEULE lecture précédente du slot) et rang 34 (`a03a5e65`, slot
// 574, image 2746, aucune autre lecture du slot). Zéro collatéral.
//
// UN FILTRE PAR FENÊTRE DE VIE A ÉTÉ ENVISAGÉ PUIS ÉCARTÉ, mesure à l'appui : sur le même
// parc, 14 lectures i48 LÉGITIMES (rang appartenant à la palette, cohérent avec le reste de
// la vie du slot) tombent elles aussi hors de toute fenêtre de vie, à des écarts de 64 à 465
// images — et DEUX d'entre elles (46c3f91d, écart 64 ; 0a44c6cc, écart 465) ENCADRENT des
// deux côtés les 176 images de la lecture bruitée de `a03a5e65`. Aucune tolérance de fenêtre,
// quelle qu'elle soit, ne peut donc séparer le bruit des lectures légitimes ; le domaine du
// rang, lui, les sépare EXACTEMENT.
const abilityRankDomainMax = 27

// rejectAbilityScanNoise écarte les lectures dont le rang dépasse abilityRankDomainMax et
// rend le nombre écarté — jamais un refus muet (cf. AbilityCoverage.ScanNoise). Appelée
// AVANT keepAbilitiesOfPublishedTracks : le bruit n'est pas une question de trajectoire
// publiée, c'est une question de valeur — inutile de la faire survivre à un filtre qui ne la
// concerne pas.
func rejectAbilityScanNoise(reads []AbilityRead) ([]AbilityRead, int) {
	if len(reads) == 0 {
		return nil, 0
	}
	out := reads[:0]
	noise := 0
	for _, r := range reads {
		if r.R > abilityRankDomainMax {
			noise++
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, noise
	}
	return out, noise
}

// AbilityCoverage est la couverture du calque IDENTITÉ DE CAPACITÉ PORTÉE (cf.
// buildAbilityReads / rejectAbilityScanNoise) : combien de lectures i48/image-clé le film a
// transmises, combien le décodeur a écartées comme BRUIT DE BALAYAGE (rang hors domaine
// plausible, RAPPORT_E0_2026-09-10 §3), combien faute de trajectoire publiée, et combien
// restent. `Reads == ScanNoise + Unpublished + Published`, exactement.
//
// TÉLÉMÉTRIE PURE, SANS MONTÉE DE SCHEMAVERSION (lot 5.6, 2026-09-10) : même règle
// qu'Inventory (cf. TestStructureIsOptionalInDocument) — champ OPTIONNEL, aucun rendu n'en
// dépend, seule la mesure du parc en bénéficie.
type AbilityCoverage struct {
	// Reads est le nombre de lectures i48/image-clé DISPONIBLES avant tout filtre.
	Reads int `json:"reads"`
	// ScanNoise compte les lectures ÉCARTÉES pour rang hors domaine plausible.
	ScanNoise int `json:"scanNoise"`
	// Unpublished est le nombre de lectures retirées faute de trajectoire publiée pour leur
	// slot (keepAbilitiesOfPublishedTracks) — comptée à part, même filtre que les autres
	// calques.
	Unpublished int `json:"unpublished"`
	// Published est le nombre de lectures effectivement publiées dans doc.Abilities.
	Published int `json:"published"`
}

// buildAbilityCoverage assemble la couverture depuis les trois étapes du filtrage : brut
// (buildAbilityReads), nettoyé du bruit (rejectAbilityScanNoise), publié
// (keepAbilitiesOfPublishedTracks).
func buildAbilityCoverage(raw, clean, published []AbilityRead, noise int) AbilityCoverage {
	return AbilityCoverage{
		Reads:       len(raw),
		ScanNoise:   noise,
		Unpublished: countUnpublished(len(clean), len(published)),
		Published:   len(published),
	}
}

// logAbilityCoverage journalise la couverture du calque — un rejet compté mais jamais
// journalisé serait à moitié muet.
func logAbilityCoverage(cov AbilityCoverage) {
	slog.Info("rejeu : identite de capacite portee",
		"lectures", cov.Reads, "bruitDeBalayage", cov.ScanNoise,
		"sansTrajectoirePubliee", cov.Unpublished, "publiees", cov.Published)
}

// AbilityPalette est une palette de capacités du titre : les rangs qui la SIGNENT, et les
// noms qu'elle donne à ceux d'entre eux qui sont établis. Elle vient du catalogue du titre
// (`config/titles/{slug}/mappings/replay_labels.toml`), jamais du code.
type AbilityPalette struct {
	// ID nomme la palette dans les journaux. Il ne sort jamais à l'écran.
	ID string
	// Markers sont les rangs dont l'observation signe cette palette.
	Markers []int
	// Ranks nomme les rangs établis. Partielle par nature.
	Ranks map[int]Label
	// Families donne l'IDENTITÉ STABLE d'un rang, dans le vocabulaire des familles
	// d'équipement du titre (`thruster`, `grapple`, `wall`...). C'est ce qui permet à un
	// calque de désigner une capacité sans jamais écrire son RANG dans du Go : le propulseur
	// vaut 5 en famille A et 21 en famille B, et un littéral en dur rendrait le calque muet
	// sur l'autre famille.
	//
	// PLUS PARTIELLE QUE `Ranks` : un rang nommé peut n'avoir aucune famille (les power-ups,
	// dont l'objet du manifeste n'est pas l'emplacement de capacité). Absente = aucune
	// jointure, jamais une jointure devinée.
	Families map[int]string
}

// FamilyOf rend la famille d'équipement d'un rang dans cette palette, ou "" quand elle n'est
// pas établie. Nil-safe : une palette non classée ne nomme aucune famille, exactement comme
// elle ne nomme aucun rang.
func (p *AbilityPalette) FamilyOf(rank int) string {
	if p == nil {
		return ""
	}
	return p.Families[rank]
}

// LE CLASSEMENT DE PALETTE — la règle, et les chiffres qui la fondent.
//
// POURQUOI UNE RÈGLE STATISTIQUE ET NON UNE LECTURE. On a d'abord cherché une DÉSIGNATION
// dans le film, et elle n'y est pas : le registre du chunk_00 est bit-à-bit identique d'un
// film à l'autre pour les noms et flags de composants, sa longueur ne suit pas les familles
// (1 973 120 octets aussi bien sur `00162144`, famille A, que sur les trois films de la
// famille B), et aucun marqueur de groupe de tags (`sofd`, `eqip`, `vcdd`, `uwfa`, `glpa`)
// n'apparaît dans aucun chunk. La palette se déduit donc de ce que le film MONTRE.
//
// CE QUE LA MESURE DONNE (7 films, i48, 748 lectures, zéro illisible) :
//
//	00ba2e1c   1:31 2:25 4:34 5:20 6:36 10:21 23:35                       -> 202/202 famille A
//	06dfe6d9   1:19 2:25 4:35 5:22 6:38 8:2 9:2 10:34 11:3 12:4 23:35     -> 219/219 famille A
//	00162144   2:14 4:9 9:2 10:10                                          ->   35/35  famille A
//	084a804d   1:13 4:48 5:11 6:18 8:10 9:8 10:2 19:4 23:15 44:1           ->  125/130 famille A
//	000d5950   19:18 20:22 21:26 22:16                                     ->   82/82  famille B
//	00502e52   19:22 20:17 21:8 22:18                                      ->   65/65  famille B
//	07aa428d   19:11 20:10 21:13 22:8                                      ->   42/42  famille B
//
// SIX FILMS SUR SEPT SONT PURS À 100 %, le septième à 96,2 %. La règle est donc une règle de
// MAJORITÉ, et le seuil n'est pas un réglage sensible : n'importe quelle valeur entre 50 % et
// 96 % donne le MÊME classement sur ce corpus. On prend 90 %.
//
// LES QUATRE LECTURES `19` ET L'UNIQUE `44` DE `084a804d` sont le bruit attendu d'un balayage
// bit à bit : le motif d'en-tête de record peut coïncider avec autre chose, et 44 est même
// hors de toute palette connue (un `sofd` compte ~27 entrées). Les compter contre la pureté
// plutôt que les ignorer est délibéré — c'est ce qui fait que la règle REFUSERAIT un film
// réellement mélangé.
//
// SOUS LE PLANCHER, ON N'ASSOUPLIT PAS — ON EXIGE ENTIER (RAPPORT_E0_2026-09-10 §2, lot 5.6).
// Le plancher de 10 lectures ferme à tort SEPT films du parc dont la rareté n'est PAS un
// défaut de qualité : le canal image-clé ne voit QUE la fenêtre 16..23 (les quatre rangs de
// la famille B), donc AUCUN film de famille A ne récolte jamais de lecture `kf` — mesuré sur
// 39 films de famille A, 0 lecture `kf`, quand le plus pauvre des films de famille B en
// transmet 153. Le plancher ne mesure donc pas la palette de ces sept films, il mesure une
// DURÉE croisée à un canal borgne. Les sept ont pourtant n dans [1, 7] et 100 % DE PURETÉ SUR
// UNE SEULE FAMILLE — la même propriété que la règle ci-dessus juge suffisante à 90 % dès que
// n ≥ 10. La généralisation naturelle de « une lecture parasite ne doit pas disqualifier un
// film pur » est donc : EN DESSOUS DU PLANCHER, ON N'EN TOLÈRE AUCUNE — l'unanimité (100 %)
// remplace la pureté à 90 %.
//
// VÉRIFIÉ EXHAUSTIVEMENT SUR LES 64 FILMS DU PARC : 7 films non classés -> famille A (les
// sept ci-dessus), 39 famille A -> famille A (inchangé), 17 famille B -> famille B
// (inchangé), 1 non classé -> non classé (`4f77afc1`, le SEUL film qui mélange réellement les
// deux palettes — 58 lectures de famille A et 129 de famille B — et qui reste hors périmètre
// du Grand combat). ZÉRO reclassement fautif, zéro perte.
const (
	// abilityPalettePurity : part minimale des lectures portant les marqueurs d'UNE palette,
	// appliquée quand n >= abilityPaletteMinReads.
	abilityPalettePurity = 0.90
	// abilityPaletteMinReads : le seuil qui SÉPARE les deux régimes de classifyAbilityPalette
	// — pureté à 90 % au-dessus, unanimité (100 %) en dessous. Le chiffre est DÉRIVÉ du seuil
	// et non choisi à part — c'est le plus petit n tel qu'UNE lecture parasite ne suffise pas
	// à disqualifier un film pur : (n−1)/n ≥ 0,90, donc n ≥ 10. Au-dessus, le corpus est loin
	// au-dessus : le film le plus pauvre en transmet 35.
	abilityPaletteMinReads = 10
)

// classifyAbilityPalette choisit la palette du film d'après ce qu'il MONTRE, ou rend nil.
//
// UNE SIGNATURE AMBIGUË NE NOMME RIEN, et c'est le point de toute l'étape : un film dont les
// lectures se partagent entre deux palettes sort AVEC SES RANGS et sans un seul nom. Nommer
// au jugé mettrait « grappin » sur un propulseur.
//
// DEUX RÉGIMES, UN SEUL SEUIL DE BASCULE (RAPPORT_E0_2026-09-10 §2) : à partir de
// abilityPaletteMinReads lectures, la pureté doit atteindre abilityPalettePurity ; en deçà,
// elle doit être ENTIÈRE (1.0) — un film trop rare pour absorber même une seule lecture
// parasite ne doit en tolérer AUCUNE. Une seule table de comptage sert les deux régimes :
// seul le seuil de comparaison change.
func classifyAbilityPalette(reads []AbilityRead, palettes []AbilityPalette) *AbilityPalette {
	if len(reads) == 0 || len(palettes) == 0 {
		return nil
	}
	counts := make([]int, len(palettes))
	for _, r := range reads {
		for i, p := range palettes {
			if containsRank(p.Markers, r.R) {
				counts[i]++
				break
			}
		}
	}
	threshold := abilityPalettePurity
	if len(reads) < abilityPaletteMinReads {
		threshold = 1.0
	}
	for i := range palettes {
		if float64(counts[i])/float64(len(reads)) >= threshold {
			return &palettes[i]
		}
	}
	return nil
}

// paletteIDOrNone rend l'identifiant de la palette retenue, ou une mention explicite quand
// AUCUNE ne l'a ete — un journal muet se lirait comme « pas de capacite lue ».
func paletteIDOrNone(p *AbilityPalette) string {
	if p == nil {
		return "non classee"
	}
	return p.ID
}

func containsRank(marks []int, r int) bool {
	for _, m := range marks {
		if m == r {
			return true
		}
	}
	return false
}

// abilityLabelsUsed nomme les rangs de capacité que le document emploie RÉELLEMENT.
//
// Un rang hors table n'entre pas : il gardera son numéro à l'écran, marqué comme non
// interprétable. La table est partielle ET propre à une palette — le dire vaut mieux que
// combler.
func abilityLabelsUsed(reads []AbilityRead, palette *AbilityPalette) map[string]Label {
	if palette == nil {
		return nil
	}
	out := map[string]Label{}
	for _, r := range reads {
		name, ok := palette.Ranks[r.R]
		if !ok {
			continue
		}
		// LA FAMILLE DU MANIFESTE VOYAGE AVEC LE NOM (schéma 51, lot 4.3) : sans elle, tout
		// lecteur du document cuit reconstruisait la famille depuis la racine du libellé.
		// `Families` est PLUS PARTIELLE que `Ranks` — un rang nommé sans famille garde une
		// `Family` vide, et c'est une réponse.
		name.Family = palette.FamilyOf(r.R)
		out[strconv.Itoa(r.R)] = name
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
