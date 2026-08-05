package objectiveevents

import "sort"

// named.go — les EVENEMENTS DE JOUEUR NOMMES, lus par le COMPOSANT et non par la valeur.
//
// # Ce que ce fichier remplace
//
// [LabelPersonalScore] (awards.go) etiquette un increment de score par sa VALEUR, croisee
// au quota `personal_score_awards` du joueur. C'est exact mais souvent ambigu : six
// recompenses valent 25 points, donc un +25 rend « l'un de : flag_returned, flag_stolen,
// runner_stopped » — un label ambigu est inexploitable.
//
// L'identifiant unique existe et le film le porte : c'est **l'emplacement de statistique**.
// L'archetype 6 (statborg) declare 28 `statborg-current-round-value-stat-component` ; le
// nom ne les distingue pas, **c'est l'INDEX qui est l'identite de la statistique**. Chaque
// recompense a son emplacement, donc **une emission d'un emplacement EST un evenement**,
// date a la ms et attribuable au joueur par son slot d'entite. On ne lit plus la valeur du
// score, on lit QUEL emplacement a bouge.
//
// # Le sens d'un emplacement DEPEND DU MODE — et ce n'est pas une precaution de style
//
// `comp 21 A` vaut `zone_secures` en zones et `flag_captures` en CTF. Un balayage tous
// modes confondus rend des noms contradictoires avec des chiffres d'apparence solide : il a
// ete fait, il etait faux, et c'est lui qui a appris la regle. D'ou le parametre de mode,
// obligatoire, et l'absence de table « tous modes ».
//
// # D'ou vient la table, et ce qui la limite
//
// Balayage `cmd/tmp_statnames` : la valeur finale de chaque emplacement, pour un slot dont
// l'identite est connue, est confrontee au compte de chaque recompense de
// `personal_score_awards`, puis les candidates sont intersectees sur les films.
// **Controle** : la table est ajustee separement sur les films de rang pair et de rang
// impair (aucun film partage) et ne garde que les cles sur lesquelles les deux moities sont
// d'accord — le controle a rejete 8 des 19 cles CTF que le balayage donnait pour resolues.
// Table figee : `.ai/refs/TABLE_STATS_STATBORG.tsv` ; detail :
// `.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §17.
//
// ATTENTION a ce que cette table figee EST, et a ce qu'elle n'est pas (arbitrage du
// 2026-08-05) : elle recense TOUS les emplacements decodes du statborg, tous consommateurs
// confondus — le nommage ci-dessous, mais aussi le pont d'identite (`slotidentity.go`, qui
// lit `2 B = deaths`, absent d'ici a bon droit) et la courbe de score (`score.go`). Sa
// colonne `lecteur` dit qui lit quoi. On n'attend donc PAS une egalite ligne a ligne avec
// `namedStatSlots` : la concordance porte sur les seules lignes de lecteur `named.go`, et
// elle est verifiee dans les deux sens par TestTableStatborgConcordeAvecNamedStatSlots.
//
// Limite assumee, ecrite ici pour qu'on ne l'oublie pas : l'ancre d'identite du balayage
// est `comp 2 A = nombre de frags`, donc le nommage de `comp 2 A` est CIRCULAIRE pour
// lui-meme. Il est etabli par ailleurs, nominativement (§17.2) ; les autres cles, elles, ne
// doivent rien a cette circularite.

// statSlotKey designe une valeur lisible d'un enregistrement d'entite : l'index du
// composant et le cote (A ou B) de sa paire.
type statSlotKey struct {
	Comp int
	Side string
}

// Cotes d'une paire de valeurs de composant.
const (
	sideA = "A"
	sideB = "B"
)

// statSlot decrit ce qu'un emplacement porte pour un mode donne.
type statSlot struct {
	// Stat est le nom canonique de la statistique (cf. les constantes Stat*).
	Stat string
	// Redundant marque un emplacement qui REPRODUIT un autre du meme mode. Le statborg
	// duplique certaines statistiques : `comp 12 A` reproduit `comp 2 A` (frags) et
	// `comp 12 B` reproduit `comp 3 A` (assistances). Un emplacement redondant n'emet
	// pas d'evenement — sinon chaque frag serait compte deux fois — mais il reste dans
	// la table parce qu'il sert de CONTROLE CROISE (cf. CrossCheckNamedEvents).
	Redundant bool
}

// namedStatSlots : par famille d'objectif, ce que porte chaque emplacement. Seules les
// cles confirmees par le controle sur moities disjointes y figurent.
//
// Ce qui est TOMBE au controle et n'est donc pas ici : `5 B`, `6 A`, `8 A`, `8 B`, `9 A`,
// `22 B`, `23 B`, `25 A` en CTF et `5 B` en zones — les cles a faible observation, nommees
// par coincidence de comptes et non par une correspondance reelle.
var namedStatSlots = map[string]map[statSlotKey]statSlot{
	ObjectiveTypeFlag: {
		{0, sideA}:  {Stat: StatFlagCaptures, Redundant: true},
		{21, sideA}: {Stat: StatFlagCaptures},
		{20, sideB}: {Stat: StatFlagCaptureAssists},
		{21, sideB}: {Stat: StatFlagCarriersKilled},
		{22, sideA}: {Stat: StatFlagGrabs},
		{23, sideA}: {Stat: StatFlagReturns},
		{24, sideA}: {Stat: StatFlagSteals},
		{2, sideA}:  {Stat: StatKills},
		{12, sideA}: {Stat: StatKills, Redundant: true},
		{3, sideA}:  {Stat: StatAssists},
		{12, sideB}: {Stat: StatAssists, Redundant: true},
	},
	ObjectiveTypeZone: {
		{20, sideB}: {Stat: StatZoneCaptures},
		{21, sideA}: {Stat: StatZoneSecures},
		{2, sideA}:  {Stat: StatKills},
		{12, sideA}: {Stat: StatKills, Redundant: true},
		{3, sideA}:  {Stat: StatAssists},
		{12, sideB}: {Stat: StatAssists, Redundant: true},
	},
}

// Noms canoniques des STATISTIQUES, tels que `match_objective_stats` les nomme et que le
// binaire les declare (`CtfStats_FlagGrabs`, `StrongholdsStats_StrongholdCaptures`...).
//
// # Pourquoi des noms de STATS et non de RECOMPENSES — et ce que ca a coute de l'ignorer
//
// Une premiere version portait les noms de `personal_score_awards` (`flag_taken`,
// `killed_player`...). C'etait une confusion de nature : le statborg replique des
// COMPTEURS DE STATISTIQUE, pas les recompenses de score que le serveur en derive. Pour la
// plupart les deux coincident numeriquement, ce qui masquait l'erreur.
//
// **`comp 22 A` l'a demasquee.** Nomme `flag_taken`, il comptait systematiquement plus que
// la recompense — jamais moins. L'oracle a 8 joueurs (`match_objective_stats_latest`) donne
// la reponse : ses valeurs sont EXACTEMENT `flag_grabs`, slot par slot (16 pour Madina97294
// la ou la recompense `flag_taken` dit 4). Le binaire disait la meme chose depuis le debut
// avec `CtfStats_FlagGrabs`. Ramasser le drapeau au sol se COMPTE a chaque fois, mais ne se
// RECOMPENSE pas a chaque fois.
//
// Trois sources concordantes, un seul nom juste : `flag_grabs`.
const (
	StatFlagCaptures       = "flag_captures"
	StatFlagCaptureAssists = "flag_capture_assists"
	StatFlagGrabs          = "flag_grabs"
	StatFlagReturns        = "flag_returns"
	StatFlagSteals         = "flag_steals"
	StatFlagCarriersKilled = "flag_carriers_killed"
	StatZoneCaptures       = "zone_captures"
	StatZoneSecures        = "zone_secures"
	StatKills              = "kills"
	StatAssists            = "assists"
)

// NamedEvent est une action de joueur nommee, datee a la milliseconde sur l'horloge du
// film — la meme que celle des ObjectiveEvent et des positions du rejeu, superposable sans
// recalage.
type NamedEvent struct {
	// TimeMS est l'instant de l'emission qui porte cette action.
	TimeMS int
	// Slot identifie l'entite de joueur (10..24 pairs ; cf. IsTeamSlot).
	Slot int
	// Stat est le nom canonique de la statistique repliquee (cf. les constantes Stat*).
	Stat string
	// Comp et Side disent quel emplacement de statistique a porte l'evenement : c'est la
	// provenance, et elle se verifie.
	Comp int
	Side string
}

// NamedEvents rend les evenements de joueur nommes d'un film, tries par instant puis par
// slot. objectiveType est la famille d'objectif du match ([ObjectiveTypeOf] la donne a
// partir du `game_variant_name`) : sans elle, aucun nommage n'est possible.
//
// Un mode sans table (KOTH, Oddball) rend nil — pas d'erreur, pas de nom invente. Les
// emplacements de `hill` et `ball` n'ont pas encore ete nommes : le balayage est le meme,
// c'est le corpus qui manque.
func NamedEvents(src FilmSource, objectiveType string) []NamedEvent {
	return namedEventsFrom(StatRecords(src), objectiveType)
}

// namedEventsFrom est le coeur pur : il travaille sur des enregistrements deja decodes, ce
// qui le rend testable sans film.
func namedEventsFrom(recs []StatRecord, objectiveType string) []NamedEvent {
	table, ok := namedStatSlots[objectiveType]
	if !ok {
		return nil
	}
	var out []NamedEvent
	for key, slot := range table {
		if slot.Redundant {
			continue
		}
		for entity, pts := range seriesBySlot(recs, key) {
			for _, t := range incrementTimes(pts) {
				out = append(out, NamedEvent{
					TimeMS: t, Slot: entity, Stat: slot.Stat, Comp: key.Comp, Side: key.Side,
				})
			}
		}
	}
	sortNamedEvents(out)
	return out
}

// sortNamedEvents ordonne par instant, puis par slot, puis par nom : un ordre total, donc
// une sortie reproductible malgre le parcours de map.
func sortNamedEvents(evs []NamedEvent) {
	sort.SliceStable(evs, func(i, j int) bool {
		if evs[i].TimeMS != evs[j].TimeMS {
			return evs[i].TimeMS < evs[j].TimeMS
		}
		if evs[i].Slot != evs[j].Slot {
			return evs[i].Slot < evs[j].Slot
		}
		return evs[i].Stat < evs[j].Stat
	})
}

// seriesBySlot rend, par slot de JOUEUR, la suite chronologique des valeurs d'un
// emplacement, debarrassee des ancrages parasites.
//
// Le filtre est le meme que celui du score de mode et pour la meme raison : un compteur de
// recompense ne recule jamais, donc la plus longue sous-suite NON DECROISSANTE est la vraie
// suite. Non decroissante et non strictement croissante : un composant porte deux valeurs
// et il est reemis des que l'UNE des deux bouge, donc la meme valeur revient legitimement.
func seriesBySlot(recs []StatRecord, key statSlotKey) map[int][]ScorePoint {
	raw := map[int][]ScorePoint{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[key.Comp]
		if !ok {
			continue
		}
		val := v.A
		if key.Side == sideB {
			val = v.B
		}
		// Une emission NEGATIVE est un ancrage parasite : un compteur de recompense est
		// positif. Elle est jetee ICI, avant le choix de la sous-suite, et pas apres —
		// sinon elle fausse ce choix. Mesure : sur la suite (1, -115, 1), la plus longue
		// sous-suite non decroissante retenue devenait (-115, 1), ce qui datait
		// l'evenement de la DERNIERE emission au lieu de la premiere.
		if val < 0 {
			continue
		}
		raw[r.Slot] = append(raw[r.Slot], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: val})
	}
	out := make(map[int][]ScorePoint, len(raw))
	for slot, pts := range raw {
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
		out[slot] = longestRun(pts, false)
	}
	return out
}

// incrementTimes rend un instant par UNITE gagnee par le compteur : c'est la conversion
// d'un compteur en evenements.
//
// La premiere valeur observee est comptee depuis zero — un compteur de recompense part de
// zero au coup d'envoi. Si le film ne montre le slot qu'apres coup, les unites deja
// acquises sont datees de cette premiere emission, ce qui MAJORE leur instant.
//
// # Le garde-fou, et il a ete paye
//
// `prev` ne redescend JAMAIS, sinon la meme unite se compte deux fois apres un creux. Sans
// cela, une seule emission aberrante a -115 faisait remonter le compteur de 0 a 1 en **116**
// evenements (mesure sur `1bc77d2e`, slot 24, comp 0 A). Les emissions negatives elles-memes
// sont ecartees plus tot, par [seriesBySlot].
func incrementTimes(pts []ScorePoint) []int {
	var out []int
	prev := int64(0)
	for _, p := range pts {
		for ; prev < p.Value; prev++ {
			out = append(out, p.TimeMS)
		}
	}
	return out
}

// KnownStats rend les statistiques qu'une famille d'objectif sait nommer. C'est
// l'inventaire de ce que la lecture par composant couvre pour ce mode — et donc, en
// creux, ce qu'elle ne couvre pas.
func KnownStats(objectiveType string) map[string]bool {
	out := map[string]bool{}
	for _, slot := range namedStatSlots[objectiveType] {
		out[slot.Stat] = true
	}
	return out
}

// CountsBySlot compte les evenements nommes par slot puis par recompense. C'est la forme
// qui se confronte directement a `personal_score_awards` (l'oracle de l'API) : meme
// granularite, memes noms.
func CountsBySlot(evs []NamedEvent) map[int]map[string]int {
	out := map[int]map[string]int{}
	for _, e := range evs {
		if out[e.Slot] == nil {
			out[e.Slot] = map[string]int{}
		}
		out[e.Slot][e.Stat]++
	}
	return out
}

// CrossCheckNamedEvents confronte chaque emplacement REDONDANT a l'emplacement canonique
// qui porte le meme nom, et rend les slots ou les deux comptes different.
//
// Ce n'est pas une verification de forme : le statborg duplique certaines statistiques,
// donc la redondance est une source de controle GRATUITE et interne au film — si `comp 12 A`
// et `comp 2 A` divergent sur un slot, l'un des deux decodages a derape sur ce slot, et on
// le sait sans oracle externe.
func CrossCheckNamedEvents(src FilmSource, objectiveType string) map[int]map[string][2]int {
	return crossCheckFrom(StatRecords(src), objectiveType)
}

// crossCheckFrom est le coeur pur de [CrossCheckNamedEvents].
func crossCheckFrom(recs []StatRecord, objectiveType string) map[int]map[string][2]int {
	table, ok := namedStatSlots[objectiveType]
	if !ok {
		return nil
	}
	canonical := map[string]statSlotKey{}
	for key, slot := range table {
		if !slot.Redundant {
			canonical[slot.Stat] = key
		}
	}
	out := map[int]map[string][2]int{}
	for key, slot := range table {
		ref, hasRef := canonical[slot.Stat]
		if !slot.Redundant || !hasRef {
			continue
		}
		dupCounts, refCounts := countsOf(recs, key), countsOf(recs, ref)
		for entity, got := range dupCounts {
			want := refCounts[entity]
			if got == want {
				continue
			}
			if out[entity] == nil {
				out[entity] = map[string][2]int{}
			}
			out[entity][slot.Stat] = [2]int{want, got}
		}
	}
	return out
}

// countsOf rend, par slot, le nombre d'unites gagnees par un emplacement.
func countsOf(recs []StatRecord, key statSlotKey) map[int]int {
	out := map[int]int{}
	for slot, pts := range seriesBySlot(recs, key) {
		out[slot] = len(incrementTimes(pts))
	}
	return out
}
