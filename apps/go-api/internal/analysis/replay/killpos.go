package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// killpos.go — OÙ CHAQUE MORT A EU LIEU.
//
// # CE QUE CE FICHIER PRODUIT, ET CE QU'IL NE PRODUIT PAS
//
// Il produit les COORDONNÉES du tueur et de la victime à l'instant d'une mort. Il ne produit ni
// le tueur, ni la victime, ni l'instant : ces trois faits sont DÉJÀ résolus ailleurs, par le
// chantier arme-par-kill, et persistés dans `match_kill_events` (mesuré le 2026-08-08 :
// 99,55 % des 74 909 morts des 951 films en cache portent leur arme, 98,87 % leur tueur).
//
// LES COUPLES ARRIVENT DONC EN ENTRÉE, ILS NE SONT PAS REDÉCODÉS ICI. C'est la même règle que
// pour `Options.Objectives`, et elle est doctrinale : « deux décodeurs du même fait
// divergeraient ». Ce paquet n'ouvre aucune base ; l'appelant lit les couples et les fournit.
//
// # POURQUOI CE FICHIER EXISTE
//
// La table `kill_positions` existe depuis le chantier Halo 5, sa migration dit « Halo 5 natif,
// Infinite plus tard », et elle est VIDE pour Infinite : 0 ligne. Halo 5 reçoit les positions de
// son API ; Infinite ne les publie nulle part — elles ne se lisent que dans le film. C'est ce
// « plus tard » que ce fichier solde.
//
// # LA RÈGLE DE PRUDENCE, IDENTIQUE AU RESTE DU CHANTIER
//
// Une position absente reste ABSENTE (nil), jamais approchée. Une mort dont NI le tueur NI la
// victime n'est localisable n'est pas écrite du tout. Les deux comptes sont rendus à l'appelant
// pour qu'il puisse dire ce qu'il a perdu — un producteur qui tait ses trous laisse croire à
// l'exhaustivité.

// killPosToleranceUS : écart maximal entre l'instant de la mort et l'échantillon de position
// retenu. MÊME valeur que le rattachement des tirs (`shotPosToleranceUS`) et pour la même
// raison : les positions arrivent à ~60 Hz, deux quanta de réplication bornent le bruit
// d'horloge sans autoriser une extrapolation.
const killPosToleranceUS = shotPosToleranceUS

// KillRef est une mort DÉJÀ appariée à son tueur, telle que `match_kill_events` la porte.
type KillRef struct {
	// KillerXUID / VictimXUID identifient les deux joueurs. Le xuid, jamais un index : un
	// index n'a de sens qu'à l'intérieur d'un film.
	KillerXUID, VictimXUID uint64
	// TimeMS est l'instant sur l'horloge du MATCH, celle du fil des morts.
	TimeMS int64
}

// MatchKillsInput porte les couples (tueur, victime, instant) d'un match ET le témoin de
// LECTURE qui distingue « source non lue » d'une liste vide — exactement le rôle que
// `KillsInput` tient pour la jointure d'équipement. Entrée de DONNÉES comme `Deaths` ou
// `Objectives` : ce paquet ne décode ni film ni base, l'appelant résout et fournit.
//
// # L'HORLOGE EST CELLE DU MATCH, PAR CONSTRUCTION ET NON PAR CALAGE
//
// C'est le point qui décidait de tout ce lot, et il se lit sur pièces plutôt qu'il ne s'estime :
// `killsource.Kill.TimeMS` et `Death.TimeMS` sont LE MÊME CHAMP DU MÊME ENREGISTREMENT du chunk
// highlight (`analysis.HighlightEvent.TimeMS`), lu par `ScanDeaths` d'un côté
// (deaths_source.go) et par `killsource.buildFeed` de l'autre. Or « l'horloge du match » de ce
// paquet EST celle du fil des morts, par définition — c'est elle que `deathOffsetMS` sert à
// rejoindre depuis les horodatages de paquet (`matchMS = TimestampUS/1000 − deathOffsetMS`,
// cf. bomb_carries.go). Les couples et les périodes de portage sont donc DÉJÀ sur le même axe.
//
// COROLLAIRE, ET C'EST UNE INTERDICTION : le pont `FilmToMatchOffsetMS` ne s'applique PAS ici.
// Il recale ce que le MANIFESTE date (l'anneau d'armement, cf. bomb_arms.go) ; l'appliquer à un
// couple décalerait chaque kill de sa valeur mesurée — 33 à 114 ms sur les cinq films d'Assaut,
// c'est-à-dire l'ordre de grandeur de la tolérance de fermeture elle-même.
//
// Contrôle algébrique indépendant, qui ne suppose rien de la lecture ci-dessus : la jointure
// d'équipement pose ces mêmes instants sur la grille de frames par `TimeMS − originMs`, et
// `originMs` est publié à moins de 100 ms de `firstPosUS/1000 − deathOffsetMS` (origin.go) —
// soit exactement le recalage qu'appliquerait un instant DÉJÀ sur l'horloge du match.
//
// LA SEULE HYPOTHÈSE QUI RESTE, ET ELLE PRÉEXISTE : les deux lecteurs LOCALISENT ce chunk
// différemment — `ScanDeaths` prend le DERNIER du manifeste, `killsource.loadKillFeed` celui
// qui rend le plus de kills. Si ces deux-là désignaient des chunks différents, ce n'est pas
// seulement l'horloge qui serait fausse : le pont gamertag -> xuid, en production depuis le lot
// F.1, le serait aussi. L'hypothèse est donc DÉJÀ portée par la chaîne, ce lot n'en ajoute pas.
type MatchKillsInput struct {
	// Read : la source a été lue. Faux = `bomb_carriers_killed` reste absent chez TOUS les
	// joueurs — jamais un zéro qui se lirait comme une mesure.
	Read bool
	// Kills sont les couples RÉSOLUS EN XUID DES DEUX CÔTÉS. Un couple dont une identité
	// manque n'entre pas : il ne rencontrerait aucune période, et l'inscrire gonflerait le
	// dénominateur d'une mort qu'on n'a pas su qualifier.
	Kills []KillRef
	// Dropped compte justement ces couples écartés — un producteur qui tait ses trous laisse
	// croire à l'exhaustivité. Il est JOURNALISÉ (cf. logBombStats, à côté du dénominateur
	// `kills` qu'il complète) et NON publié au document : `BombStatsCoverage` est un schéma
	// d'API, et un compteur de diagnostic ne justifie pas d'en élargir le contrat. Sa
	// ventilation par cause vit chez le producteur (`replaybuild.killResolution`).
	Dropped int
}

// Vec3 est une position monde. Rendue par pointeur chez l'appelant : une coordonnée absente
// doit rester distinguable d'un zéro, qui est une position valide.
type Vec3 struct{ X, Y, Z float64 }

// KillPosition est le résultat : une mort et les positions qu'on a su lui donner.
type KillPosition struct {
	KillRef
	// Killer / Victim sont nil quand la position n'est pas localisable à cet instant.
	Killer, Victim *Vec3
}

// KillPosReport dit ce que la production a placé et ce qu'elle a laissé de côté.
type KillPosReport struct {
	// Kills est le nombre de morts fournies en entrée — le dénominateur.
	Kills int
	// Both / KillerOnly / VictimOnly ventilent ce qui a été écrit.
	Both, KillerOnly, VictimOnly int
	// Dropped compte les morts dont AUCUN des deux n'est localisable : rien n'est écrit.
	Dropped int
	// NoBridge compte les morts dont un xuid n'a aucun slot au pont — un sous-cas de Dropped,
	// isolé parce qu'il désigne le chantier du PONT et non celui des positions.
	NoBridge int
}

// BuildKillPositions rend les positions monde des deux joueurs de chaque mort.
//
// PUR : aucune I/O, aucune base. `offsetUS` est le décalage entre l'horloge du fil des morts et
// celle du film, résolu par mesure (cf. bestDeathOffset) — le passer en paramètre évite que ce
// producteur ne refasse un calage que le pont a déjà fait.
func BuildKillPositions(pos []filmdec.BipedPosition, slotXUID map[uint32]uint64,
	kills []KillRef, offsetUS int64) ([]KillPosition, KillPosReport) {
	rep := KillPosReport{Kills: len(kills)}
	if len(pos) == 0 || len(slotXUID) == 0 || len(kills) == 0 {
		rep.Dropped = len(kills)
		return nil, rep
	}
	tracks := indexBySlot(pos)
	byXUID := slotsByXUID(slotXUID)
	out := make([]KillPosition, 0, len(kills))
	for _, k := range kills {
		tUS := uint64(k.TimeMS*1000 + offsetUS)
		kp := KillPosition{KillRef: k}
		kp.Killer = positionOf(tracks, byXUID[k.KillerXUID], tUS)
		kp.Victim = positionOf(tracks, byXUID[k.VictimXUID], tUS)
		if len(byXUID[k.KillerXUID]) == 0 || len(byXUID[k.VictimXUID]) == 0 {
			rep.NoBridge++
		}
		switch {
		case kp.Killer != nil && kp.Victim != nil:
			rep.Both++
		case kp.Killer != nil:
			rep.KillerOnly++
		case kp.Victim != nil:
			rep.VictimOnly++
		default:
			rep.Dropped++
			continue // aucune des deux positions : on n'écrit rien plutôt qu'une ligne vide
		}
		out = append(out, kp)
	}
	return out, rep
}

// slotsByXUID inverse le pont : un joueur possède plusieurs slots au cours d'un match (un par
// vie), et c'est précisément pour cela qu'il faut les chercher tous.
func slotsByXUID(slotXUID map[uint32]uint64) map[uint64][]uint32 {
	out := map[uint64][]uint32{}
	for slot, x := range slotXUID {
		out[x] = append(out[x], slot)
	}
	for x := range out {
		sort.Slice(out[x], func(i, j int) bool { return out[x][i] < out[x][j] })
	}
	return out
}

// positionOf rend la position du joueur à l'instant tUS, ou nil.
//
// L'AMBIGUÏTÉ EST TRAITÉE COMME AILLEURS : si DEUX slots du même joueur portent un échantillon
// dans la tolérance, on ne tranche pas — deux corps pour un joueur signifie que le découpage des
// vies est faux à cet instant, et poser la mort sur l'un des deux serait un coup de dé.
func positionOf(tracks map[uint32]slotTrack, slots []uint32, tUS uint64) *Vec3 {
	var found filmdec.BipedPosition
	n := 0
	for _, s := range slots {
		p, d := tracks[s].at(tUS)
		if d > killPosToleranceUS || !p.HasWorld {
			continue
		}
		found, n = p, n+1
	}
	if n != 1 {
		return nil
	}
	return &Vec3{X: float64(found.X), Y: float64(found.Y), Z: float64(found.Z)}
}
