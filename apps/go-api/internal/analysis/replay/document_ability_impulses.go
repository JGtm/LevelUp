package replay

// document_ability_impulses.go — LES IMPULSIONS DE CAPACITÉ, datées par le film et
// ATTRIBUÉES par le rang porté (schéma 38, lot P3 du
// PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03).
//
// CE QUE C'EST. Le corps de tag externe `R(2) == 1` des composants bipède
// `biped-spartan-ability` (i57) et `-non-predicted-state` (i59) date une IMPULSION — le
// même composant dont le tag 3 porte déjà le grappin, désérialisé depuis le 2026-08-16.
// `filmdec.ScanFilmAbilityImpulses` en rend les lectures ; ce fichier les replie en
// épisodes et leur donne une IDENTITÉ.
//
// L'IDENTITÉ NE VIENT PAS DU COMPOSANT — elle vient du canal i48, rang lu DANS LA MÊME VIE
// et ANTÉRIEUREMENT à l'instant. Les deux contraintes sont nécessaires, et mesurées : le
// slot MIGRE aux réapparitions (sans le découpage par vie, une impulsion hérite du rang de
// la vie précédente du même slot — signature vue sur `11de8353`, R8 §8.4), et un joueur
// peut ramasser un second équipement dans la même vie (sans l'antériorité, 4 des
// 8 lectures de `00ba2e1c` passent du rang du propulseur à celui du grappin). Le `sub` du
// composant a été essayé comme discriminant puis RÉFUTÉ par le corpus (R8 §8.5).
//
// CE QUE LE CALQUE PUBLIE, ET CE QU'IL REFUSE. Seules les familles que le TITRE déclare
// MESURÉES sur ce canal (`LabelCatalog.AbilityImpulseFamilies`) sont publiées. Aujourd'hui
// c'est le PROPULSEUR, et lui seul : 0,361 impulsion par vie de propulseur contre 0,011 par
// vie de répulseur — plus porté que lui — et 0,000 sur 132 vies de grappin, qui a son propre
// tag (R8 §8.8, quatre films). LE RÉPULSEUR N'EST PAS DANS CE CANAL, et ce négatif est
// MESURÉ, pas supposé (rapport R9 : ses trois portes restantes sont fermées). Publier ses
// rares lectures ferait dessiner un geste que le film n'enregistre pas ; elles sont donc
// ÉCARTÉES ET COMPTÉES (`otherFamily`) — le calque dit ce qu'il refuse.
//
// VÉRITÉ TERRAIN (utilisateur, Theater, film `1cd3848a`, 2026-09-03) : 5 usages de
// propulseur relevés à 1:51, 1:54, 2:03, 2:05, 2:14 ; cette chaîne en rend 5, à 1:52, 1:55,
// 2:03, 2:05, 2:15 — précision 5/5, rappel 5/5, écart ≤ 1 s. C'est le seul chiffre de rappel
// que le canal possède, et il vient d'un relevé, pas d'un modèle.

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// abilityImpulseEpisodeGapUS : deux lectures du même slot séparées de moins d'une seconde
// portent LE MÊME geste. Le seuil vient des instruments de mesure (R8/R9,
// `r8MobEpisodeGapUS`), et il a deux raisons d'être : i57 et i59 sont CO-TRANSMIS (un même
// geste apparaît souvent dans les deux, et les replier évite de le compter deux fois), et le
// film retransmet un état sur quelques trames. Les usages successifs d'une même vie sont à
// plusieurs secondes (la salve témoin de `1cd3848a` : 1:52, 1:55, 2:03, 2:05, 2:15).
const abilityImpulseEpisodeGapUS = 1_000_000

// AbilityImpulse est UNE impulsion de capacité publiée : qui, quand, et de quel équipement.
type AbilityImpulse struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée — donc une VIE, pas un joueur (le slot migre aux
	// réapparitions), comme sur tous les autres calques.
	Slot uint32 `json:"slot"`
	// Family est l'identité de l'équipement, dans le vocabulaire des familles du titre
	// (`thruster`) — la MÊME que celle des poses d'équipement. Elle vient du RANG i48 de la
	// vie, nommé par la palette du match : jamais un rang en dur (5 en famille A, 21 en
	// famille B). Toujours non vide : une impulsion sans identité n'est pas publiée.
	Family string `json:"family"`
}

// AbilityImpulseCoverage dit ce que le calque a lu et ce qu'il a écarté — l'entonnoir
// complet, sans lequel « N impulsions » ne se juge pas.
type AbilityImpulseCoverage struct {
	// Reads est le nombre de lectures `tag == 1` décodées du film (i57 et i59 confondus).
	Reads int `json:"reads"`
	// Episodes est le nombre de GESTES qu'elles forment après repliement (cf.
	// abilityImpulseEpisodeGapUS). C'est le dénominateur de tout ce qui suit.
	Episodes int `json:"episodes"`
	// Published est le nombre d'impulsions publiées dans le document.
	Published int `json:"published"`
	// BeforeOrigin compte les épisodes antérieurs à la première frame — écartés.
	BeforeOrigin int `json:"beforeOrigin"`
	// Unpublished compte les épisodes dont le slot n'a pas de trajectoire publiée — écartés
	// (le client n'aurait aucune piste où poser le geste).
	Unpublished int `json:"unpublished"`
	// NoIdentity compte les épisodes SANS rang i48 lisible dans leur vie avant l'instant :
	// le geste est mesuré, l'équipement ne l'est pas. Écartés — jamais devinés.
	NoIdentity int `json:"noIdentity"`
	// OtherFamily compte les épisodes dont la famille résolue n'est PAS déclarée mesurée sur
	// ce canal par le titre. Ils sont écartés parce que le canal n'est prouvé que pour les
	// familles déclarées (R8/R9) — les compter à part est ce qui distingue « le film n'en
	// porte pas » de « on refuse de l'affirmer ».
	//
	// UNE FAMILLE A DONC ÉTÉ RÉSOLUE (ou le rang n'en porte aucune). Un épisode que la chaîne
	// d'attribution n'a même pas pu examiner ne tombe PAS ici : cf. NoResolver.
	OtherFamily int `json:"otherFamily"`
	// NoResolver compte les épisodes que le calque n'a même pas pu SOUMETTRE à l'attribution,
	// faute d'une des trois pièces qu'elle exige : la palette du match n'a pas été classée
	// (signature ambiguë ou moins de dix lectures i48 — cf. classifyAbilityPalette), le titre
	// ne déclare AUCUNE famille mesurée sur ce canal, ou aucune vie n'a été découpée (le pont
	// n'a rien rendu).
	//
	// POURQUOI UN COMPTEUR À PART, ET PAS UN REPLI SUR LES DEUX AUTRES. Sans lui, un film à
	// palette non classée versait tous ses gestes dans `otherFamily` — c'est-à-dire annonçait
	// « ces gestes viennent d'un AUTRE équipement » alors qu'AUCUNE famille n'avait jamais été
	// résolue ; et un film sans vies les versait dans `noIdentity`, qui affirme « le rang n'a
	// pas été lu dans la vie » là où il n'y avait pas de vie du tout. Les deux se lisaient
	// comme des mesures alors que ce sont des indisponibilités. La doctrine du chantier est de
	// publier l'incertitude, jamais de la déguiser en mesure.
	NoResolver int `json:"noResolver"`
	// ComponentAbsent dit que l'archétype bipède du film ne déclare NI i57 NI i59 : le film ne
	// transmet pas ce canal du tout. Sans ce témoin, un zéro se lirait comme « personne ne
	// s'est servi de son propulseur ».
	ComponentAbsent bool `json:"componentAbsent,omitempty"`
}

// abilityImpulseInputs rassemble ce dont la jointure a besoin. Une structure plutôt que sept
// paramètres : le seuil du dépôt est à cinq, et ces cinq-là forment UNE question (« que
// portait ce joueur quand il a fait ce geste ? »).
type abilityImpulseInputs struct {
	// reads : les lectures brutes du film (filmdec).
	reads []filmdec.AbilityImpulse
	// stats : les dénominateurs du balayage — c'est d'eux que vient `ComponentAbsent`.
	stats filmdec.AbilityImpulseStats
	// ranks : les identités de capacité transmises par i48, le SEUL canal d'identité.
	ranks []filmdec.AbilityRank
	// lives : le découpage des vies, tel que le pont l'a déjà fait sur les positions BRUTES.
	lives []lifeSpan
	// palette : la palette du match, qui nomme le rang. Nil = film non classé -> aucune
	// identité, donc aucune impulsion publiée (mieux vaut muet que faux).
	palette *AbilityPalette
	// measured : les familles que le TITRE déclare mesurées sur ce canal.
	measured []string
}

// buildAbilityImpulses replie les lectures en épisodes, leur donne une identité par le rang
// i48 de leur vie, et ne publie que les familles déclarées mesurées par le titre.
func buildAbilityImpulses(
	in abilityImpulseInputs, tracks []Track, origin, step uint64,
) ([]AbilityImpulse, AbilityImpulseCoverage) {
	cov := AbilityImpulseCoverage{Reads: len(in.reads), ComponentAbsent: in.stats.Absent}
	if len(in.reads) == 0 || step == 0 {
		return nil, cov
	}
	b := &abilityImpulseBuilder{
		byLife: newAbilityRankIndex(in.ranks, in.lives), palette: in.palette,
		measured: make(map[string]bool, len(in.measured)),
		origin:   origin, step: step, cov: &cov,
	}
	for _, f := range in.measured {
		b.measured[f] = true
	}
	// LES TROIS PIÈCES DE L'ATTRIBUTION, vérifiées UNE fois : sans l'une d'elles, la chaîne
	// d'identité ne peut pas tourner du tout, et ses refus n'auraient aucun sens (cf.
	// AbilityImpulseCoverage.NoResolver).
	b.resolvable = in.palette != nil && len(b.measured) > 0 && len(in.lives) > 0
	published := publishedSlots(tracks)
	var out []AbilityImpulse
	for _, ep := range foldAbilityImpulses(in.reads) {
		cov.Episodes++
		switch {
		case ep.tsUS < origin:
			cov.BeforeOrigin++
		case !published[ep.slot]:
			cov.Unpublished++
		case !b.resolvable:
			cov.NoResolver++
		default:
			if im, ok := b.resolve(ep); ok {
				out = append(out, im)
			}
		}
	}
	cov.Published = len(out)
	if len(out) == 0 {
		return nil, cov
	}
	return out, cov
}

// abilityImpulseBuilder porte ce que la résolution d'identité consulte à chaque épisode.
// Une structure plutôt que huit paramètres : le seuil du dépôt est à cinq, et ces champs-là
// sont le contexte d'une seule question, pas des réglages indépendants.
type abilityImpulseBuilder struct {
	byLife   *abilityRankIndex
	palette  *AbilityPalette
	measured map[string]bool
	// resolvable dit que les TROIS pièces de l'attribution sont là (palette classée, familles
	// mesurées déclarées, vies découpées). Faux, aucun épisode n'est soumis à `resolve` : ses
	// refus décriraient une mesure qui n'a pas eu lieu.
	resolvable   bool
	origin, step uint64
	cov          *AbilityImpulseCoverage
}

// resolve donne son identité à UN épisode et rend l'impulsion à publier. ok=false quand elle
// est écartée — et les deux refus sont comptés à part : « aucun rang lu dans la vie » et
// « rang lu mais famille non mesurée » ne disent pas la même chose du film.
func (b *abilityImpulseBuilder) resolve(ep abilityImpulseEpisode) (AbilityImpulse, bool) {
	rank, ok := b.byLife.rankInLife(ep.slot, ep.tsUS)
	if !ok {
		b.cov.NoIdentity++
		return AbilityImpulse{}, false
	}
	fam := b.palette.FamilyOf(rank)
	if fam == "" || !b.measured[fam] {
		b.cov.OtherFamily++
		return AbilityImpulse{}, false
	}
	return AbilityImpulse{
		T: int((ep.tsUS - b.origin) / b.step), Slot: ep.slot, Family: fam,
	}, true
}

// abilityImpulseEpisode est UN geste : le slot et l'instant de sa PREMIÈRE lecture.
type abilityImpulseEpisode struct {
	slot uint32
	tsUS uint64
}

// foldAbilityImpulses replie les lectures en gestes : deux lectures du même slot à moins
// d'abilityImpulseEpisodeGapUS l'une de l'autre n'en font qu'un. La sortie est TRIÉE par
// instant puis par slot — l'ordre du document, déterministe.
func foldAbilityImpulses(reads []filmdec.AbilityImpulse) []abilityImpulseEpisode {
	ordered := make([]filmdec.AbilityImpulse, len(reads))
	copy(ordered, reads)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Slot != ordered[j].Slot {
			return ordered[i].Slot < ordered[j].Slot
		}
		return ordered[i].TimestampUS < ordered[j].TimestampUS
	})
	last := map[uint32]uint64{}
	out := make([]abilityImpulseEpisode, 0, len(ordered))
	for _, r := range ordered {
		if prev, seen := last[r.Slot]; seen && r.TimestampUS-prev <= abilityImpulseEpisodeGapUS {
			// LE REPLIEMENT SUIT LA DERNIÈRE LECTURE, pas la première du geste : une salve de
			// retransmissions espacées de moins d'une seconde chacune est UN geste, comme le
			// mesurent les instruments R8/R9 dont ce code reprend la règle.
			last[r.Slot] = r.TimestampUS
			continue
		}
		last[r.Slot] = r.TimestampUS
		out = append(out, abilityImpulseEpisode{slot: r.Slot, tsUS: r.TimestampUS})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].tsUS != out[j].tsUS {
			return out[i].tsUS < out[j].tsUS
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// abilityRankIndex répond à « quel rang ce slot portait-il à cet instant, dans CETTE vie ? ».
// Il indexe une fois ce que la question relirait pour chaque impulsion.
type abilityRankIndex struct {
	ranks map[uint32][]filmdec.AbilityRank
	lives map[uint32][]lifeSpan
}

func newAbilityRankIndex(ranks []filmdec.AbilityRank, lives []lifeSpan) *abilityRankIndex {
	idx := &abilityRankIndex{
		ranks: make(map[uint32][]filmdec.AbilityRank, len(ranks)),
		lives: make(map[uint32][]lifeSpan, len(lives)),
	}
	for _, r := range ranks {
		idx.ranks[r.Slot] = append(idx.ranks[r.Slot], r)
	}
	for _, l := range lives {
		idx.lives[l.slot] = append(idx.lives[l.slot], l)
	}
	return idx
}

// rankInLife rend le rang de capacité lu pour ce slot DANS LA MÊME VIE que `at` et
// ANTÉRIEUREMENT. ok=false quand la vie n'est pas retrouvée, ou qu'aucune émission i48 ne la
// précède : l'identité est alors inconnue, et c'est une information — pas un rang nul.
//
// LA TOLÉRANCE `lifeGapUS` SUR LES BORNES est celle de la mesure d'origine (`r8RankInLife`) :
// le découpage des vies vient des POSITIONS, qui ne couvrent pas exactement l'instant d'une
// impulsion en bord de vie. La resserrer écarterait des gestes réels ; l'élargir remettrait
// le rang de la vie précédente en jeu, ce que ce découpage existe précisément pour empêcher.
func (idx *abilityRankIndex) rankInLife(slot uint32, at uint64) (int, bool) {
	span, found := lifeSpan{}, false
	for _, l := range idx.lives[slot] {
		if int64(at)+lifeGapUS >= l.from && int64(at) <= l.to+lifeGapUS {
			span, found = l, true
			break
		}
	}
	if !found {
		return 0, false
	}
	best, bestT, got := 0, uint64(0), false
	for _, r := range idx.ranks[slot] {
		if int64(r.TimestampUS) < span.from || r.TimestampUS > at {
			continue
		}
		if !got || r.TimestampUS >= bestT {
			best, bestT, got = r.Rank, r.TimestampUS, true
		}
	}
	return best, got
}

// keepAbilityImpulsesOfPublishedTracks écarte les impulsions dont le slot n'a pas de
// trajectoire publiée. LE FILTRE EST DÉJÀ APPLIQUÉ à la construction (et compté) : cette
// fonction est le garde-fou commun des calques, appliqué APRÈS le nommage des vies — deux
// passes sur la même règle, comme pour les tirs et les grenades.
func keepAbilityImpulsesOfPublishedTracks(in []AbilityImpulse, tracks []Track) []AbilityImpulse {
	return keepOfPublishedTracks(in, tracks,
		func(a AbilityImpulse, published map[uint32]bool) bool { return published[a.Slot] })
}

// logAbilityImpulseCoverage sort la couverture du calque. Un journal qui ne dirait que les
// publiées laisserait croire que le canal n'a rien refusé — or il refuse deux choses de
// natures différentes, et c'est justement ce qu'il faut pouvoir lire.
func logAbilityImpulseCoverage(cov AbilityImpulseCoverage) {
	slog.Info("rejeu : impulsions de capacite",
		"lectures", cov.Reads, "episodes", cov.Episodes, "publiees", cov.Published,
		"sansIdentite", cov.NoIdentity, "familleNonMesuree", cov.OtherFamily,
		"attributionIndisponible", cov.NoResolver,
		"avantOrigine", cov.BeforeOrigin, "sansPiste", cov.Unpublished,
		"composantAbsent", cov.ComponentAbsent)
}
