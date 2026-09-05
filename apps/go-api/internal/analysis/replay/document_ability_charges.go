package replay

// document_ability_charges.go — LES CHARGES D'ÉQUIPEMENT RESTANTES, datées par le film et
// ATTRIBUÉES par le rang porté (schéma 38 enrichi, lot P5 du
// PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03).
//
// CE QUE C'EST. Le composant bipède i56 `biped-spartan-ability-energy` transmet, AU
// CHANGEMENT, un compteur de charges entières par emplacement armé — le quartet HAUT de sa
// valeur 7 bits (rapport R11 §2 : sur `1cd3848a` la série de JGtm vaut 4, 3, 2, 1, 0
// exactement aux cinq usages relevés au Theater ; 36 accroches de grappin sur 36 appariées
// à une baisse contre 2/36 pour un témoin décalé). `filmdec.ScanFilmAbilityCharges` en rend
// les lectures ARMÉES ; ce fichier leur donne une IDENTITÉ et les publie telles quelles.
//
// LES LECTURES SONT PUBLIÉES, JAMAIS UN COMPTE D'USAGES DÉRIVÉ — piège (b) de R11 : une
// baisse peut valoir plusieurs usages (7→3, 4→2, 2→0 observés), le film ne transmet pas
// toutes les valeurs intermédiaires. Le client lit « ce qu'il reste », pas « combien de
// fois ». Et RIEN n'est affirmé avant la première lecture : le film ne transmet rien au
// ramassage (masque à 0 = le moteur pose plein) — la première valeur est ce qui reste
// APRÈS le premier usage.
//
// L'IDENTITÉ NE VIENT PAS DE L'EMPLACEMENT — la spécialisation e0/e2 mesurée par R11 est
// une OBSERVATION, pas une règle. Elle vient du canal i48 : rang lu dans la MÊME VIE et
// ANTÉRIEUREMENT à l'instant (`abilityRankIndex.rankInLife`, le MÊME index que les
// impulsions — les deux contraintes sont mesurées, cf. document_ability_impulses.go), et le
// rang se nomme par la palette du match (`FamilyOf`) : jamais un rang en dur (le grappin
// vaut 4 en famille A et 20 en famille B).
//
// CE QUE LE CALQUE PUBLIE, ET CE QU'IL REFUSE. Seules les familles que le TITRE déclare
// MESURÉES sur ce canal (`LabelCatalog.AbilityChargeFamilies`) sont publiées — le grappin
// et le propulseur, et eux seuls : le RÉPULSEUR n'arme jamais i56 (négatif MESURÉ, R11
// §4-5 : 218 vies, 0 baisse, deux porteurs consommant leurs trois charges sans une seule
// lecture armée). Les autres sont écartées ET COMPTÉES (`otherFamily`) ; quand la chaîne
// d'attribution elle-même n'a pas pu tourner, les lectures tombent dans `noResolver` —
// jamais déguisées en mesure (la leçon H2 de la revue P3, appliquée d'emblée).

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// AbilityCharge est UNE lecture de charge publiée : qui, quand, quel équipement, et ce
// qu'il en reste.
type AbilityCharge struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée — donc une VIE, pas un joueur (le slot migre aux
	// réapparitions), comme sur tous les autres calques.
	Slot uint32 `json:"slot"`
	// Family est l'identité de l'équipement, dans le vocabulaire des familles du titre
	// (`grapple`, `thruster`) — la MÊME que celle des poses et des impulsions. Elle vient du
	// RANG i48 de la vie, nommé par la palette du match : jamais un rang en dur. Toujours
	// non vide : une lecture sans identité n'est pas publiée.
	Family string `json:"family"`
	// Charges est le compte de charges ENTIÈRES restantes — le quartet haut de la valeur
	// 7 bits d'i56, la lecture discrète du consommateur de l'exe (R11 §1.1). Zéro est une
	// MESURE (« plus aucune charge »), pas une absence : le type est un int nu, jamais
	// omitempty.
	Charges int `json:"charges"`
}

// AbilityChargeCoverage dit ce que le calque a lu et ce qu'il a écarté — l'entonnoir
// complet, sans lequel « N lectures » ne se juge pas. La somme des six cases
// (published + beforeOrigin + unpublished + noIdentity + otherFamily + noResolver) vaut
// EXACTEMENT reads — l'invariant est testé.
type AbilityChargeCoverage struct {
	// Reads est le nombre de lectures ARMÉES décodées du film (un emplacement armé d'un
	// record = une lecture ; un masque à 000 n'en produit aucune).
	Reads int `json:"reads"`
	// Published est le nombre de lectures publiées dans le document.
	Published int `json:"published"`
	// BeforeOrigin compte les lectures antérieures à la première frame — écartées.
	BeforeOrigin int `json:"beforeOrigin"`
	// Unpublished compte les lectures dont le slot n'a pas de trajectoire publiée — écartées
	// (le client n'aurait aucune fiche où poser le compte).
	Unpublished int `json:"unpublished"`
	// NoIdentity compte les lectures SANS rang i48 lisible dans leur vie avant l'instant :
	// la charge est mesurée, l'équipement ne l'est pas. Écartées — jamais devinées.
	NoIdentity int `json:"noIdentity"`
	// OtherFamily compte les lectures dont la famille résolue n'est PAS déclarée mesurée sur
	// ce canal par le titre. Écartées parce que le canal n'est prouvé que pour les familles
	// déclarées (R11) — les compter à part est ce qui distingue « le film n'en porte pas »
	// de « on refuse de l'affirmer ».
	//
	// UNE FAMILLE A DONC ÉTÉ RÉSOLUE (ou le rang n'en porte aucune). Une lecture que la
	// chaîne d'attribution n'a même pas pu examiner ne tombe PAS ici : cf. NoResolver.
	OtherFamily int `json:"otherFamily"`
	// NoResolver compte les lectures que le calque n'a même pas pu SOUMETTRE à
	// l'attribution, faute d'une des trois pièces qu'elle exige : palette du match non
	// classée, titre sans famille déclarée mesurée sur ce canal, ou aucune vie découpée.
	//
	// POURQUOI UN COMPTEUR À PART — la même raison, mot pour mot, que
	// `AbilityImpulseCoverage.NoResolver` (constat H2 de la revue P3) : sans lui, un film à
	// palette non classée verserait ses lectures dans `otherFamily` (« un AUTRE
	// équipement ») et un film sans vies dans `noIdentity` (« le rang n'a pas été lu ») —
	// deux indisponibilités déguisées en mesures.
	NoResolver int `json:"noResolver"`
	// ComponentAbsent dit que l'archétype bipède du film ne déclare pas i56 : le film ne
	// transmet pas ce canal du tout. Sans ce témoin, un zéro se lirait comme « personne n'a
	// usé de charge ».
	ComponentAbsent bool `json:"componentAbsent,omitempty"`
}

// abilityChargeInputs rassemble ce dont la jointure a besoin. Une structure plutôt que six
// paramètres : le seuil du dépôt est à cinq, et ces six-là forment UNE question (« que
// portait ce joueur quand cette valeur a été transmise ? »).
type abilityChargeInputs struct {
	// reads : les lectures armées brutes du film (filmdec).
	reads []filmdec.AbilityCharge
	// stats : les dénominateurs du balayage — c'est d'eux que vient `ComponentAbsent`.
	stats filmdec.AbilityChargeStats
	// ranks : les identités de capacité transmises par i48, le SEUL canal d'identité.
	ranks []filmdec.AbilityRank
	// lives : le découpage des vies, tel que le pont l'a déjà fait sur les positions BRUTES.
	lives []lifeSpan
	// palette : la palette du match, qui nomme le rang. Nil = film non classé -> aucune
	// identité, donc aucune lecture publiée (mieux vaut muet que faux).
	palette *AbilityPalette
	// measured : les familles que le TITRE déclare mesurées sur CE canal (les charges — pas
	// celles du canal d'impulsion : deux mesures, deux listes).
	measured []string
}

// buildAbilityCharges donne une identité aux lectures armées par le rang i48 de leur vie,
// et ne publie que les familles déclarées mesurées par le titre. AUCUN repliement : chaque
// lecture est une VALEUR transmise au changement, la publier telle quelle est le contrat
// (piège (b) de R11 — jamais un compte d'usages dérivé).
func buildAbilityCharges(
	in abilityChargeInputs, tracks []Track, origin, step uint64,
) ([]AbilityCharge, AbilityChargeCoverage) {
	cov := AbilityChargeCoverage{Reads: len(in.reads), ComponentAbsent: in.stats.Absent}
	if len(in.reads) == 0 || step == 0 {
		return nil, cov
	}
	b := &abilityChargeBuilder{
		byLife: newAbilityRankIndex(in.ranks, in.lives), palette: in.palette,
		measured: make(map[string]bool, len(in.measured)),
		origin:   origin, step: step, cov: &cov,
	}
	for _, f := range in.measured {
		b.measured[f] = true
	}
	// LES TROIS PIÈCES DE L'ATTRIBUTION, vérifiées UNE fois : sans l'une d'elles, la chaîne
	// d'identité ne peut pas tourner du tout, et ses refus n'auraient aucun sens (cf.
	// AbilityChargeCoverage.NoResolver).
	b.resolvable = in.palette != nil && len(b.measured) > 0 && len(in.lives) > 0
	// LE SEUL FILTRE « pas de piste publiée, pas de calque » de ce canal, ICI parce que
	// c'est ici qu'on peut le COMPTER (`Unpublished`) — la place que les impulsions ont déjà
	// arbitrée (constat H2 de la revue P3 : jamais une seconde passe après construction).
	published := publishedSlots(tracks)
	var out []AbilityCharge
	for _, r := range in.reads {
		switch {
		case r.TimestampUS < origin:
			cov.BeforeOrigin++
		case !published[r.Slot]:
			cov.Unpublished++
		case !b.resolvable:
			cov.NoResolver++
		default:
			if c, ok := b.resolve(r); ok {
				out = append(out, c)
			}
		}
	}
	cov.Published = len(out)
	if len(out) == 0 {
		return nil, cov
	}
	return out, cov
}

// abilityChargeBuilder porte ce que la résolution d'identité consulte à chaque lecture —
// le patron d'abilityImpulseBuilder, sur l'autre canal.
type abilityChargeBuilder struct {
	byLife   *abilityRankIndex
	palette  *AbilityPalette
	measured map[string]bool
	// resolvable dit que les TROIS pièces de l'attribution sont là (palette classée,
	// familles mesurées déclarées, vies découpées). Faux, aucune lecture n'est soumise à
	// `resolve` : ses refus décriraient une mesure qui n'a pas eu lieu.
	resolvable   bool
	origin, step uint64
	cov          *AbilityChargeCoverage
}

// resolve donne son identité à UNE lecture et rend la charge à publier. ok=false quand elle
// est écartée — et les deux refus sont comptés à part : « aucun rang lu dans la vie » et
// « rang lu mais famille non mesurée » ne disent pas la même chose du film.
func (b *abilityChargeBuilder) resolve(r filmdec.AbilityCharge) (AbilityCharge, bool) {
	rank, ok := b.byLife.rankInLife(r.Slot, r.TimestampUS)
	if !ok {
		b.cov.NoIdentity++
		return AbilityCharge{}, false
	}
	fam := b.palette.FamilyOf(rank)
	if fam == "" || !b.measured[fam] {
		b.cov.OtherFamily++
		return AbilityCharge{}, false
	}
	return AbilityCharge{
		T: int((r.TimestampUS - b.origin) / b.step), Slot: r.Slot,
		Family: fam, Charges: r.Charges,
	}, true
}

// logAbilityChargeCoverage sort la couverture du calque — le patron de
// logAbilityImpulseCoverage : un journal qui ne dirait que les publiées laisserait croire
// que le canal n'a rien refusé.
func logAbilityChargeCoverage(cov AbilityChargeCoverage) {
	slog.Info("rejeu : charges d equipement",
		"lectures", cov.Reads, "publiees", cov.Published,
		"sansIdentite", cov.NoIdentity, "familleNonMesuree", cov.OtherFamily,
		"attributionIndisponible", cov.NoResolver,
		"avantOrigine", cov.BeforeOrigin, "sansPiste", cov.Unpublished,
		"composantAbsent", cov.ComponentAbsent)
}
