package replay

// bomb_stats.go — LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT, reconstruites de sources
// DÉJÀ décodées.
//
// Noyau PUR : aucune I/O, aucune base, aucun réseau, aucun film ouvert. L'appelant fournit ce
// que `BuildFromFilm` a déjà lu — aucun second décodage n'est payé ici. C'est la même règle
// que pour `Options.Objectives` et `BuildKillPositions`, et elle est doctrinale : deux
// décodeurs du même fait divergeraient.
//
// # D'OÙ VIENT CHAQUE CHIFFRE, ET CE QUE ÇA VAUT — mesure contre déduction
//
//	bomb_detonations              MESURE. Compteur du moteur : statborg `comp 0` canal A,
//	                              nommé `StatBombDetonations` (objectiveevents/named.go), puis
//	                              identifié PAR MANCHE par l'appelant hors ligne. Gate A4
//	                              (2026-09-01) : la somme des slots joueurs vaut exactement les
//	                              explosions du film, 4/4 films sur moitiés disjointes.
//	bomb_grabs                    MESURE de transitions, DÉDUCTION du sens. Le canal des armes
//	                              tenues porte la bombe comme une arme ; une transition VERS la
//	                              famille `0x3fee4fcf` ouvre une période de portage
//	                              (held_object_carry.go). Que cette transition SOIT un
//	                              ramassage est la déduction — B1 (2026-09-01) : famille unique
//	                              candidate hors catalogue d'armes des 9 films d'Assaut, prise
//	                              et lâchée sur chacun ; l'atlas HUD la confirme séparément.
//	time_as_bomb_carrier_seconds  MESURE, avec une réserve écrite. Somme des périodes FERMÉES
//	                              du même canal (`HeldObjectCarry.CarryMSByXUID`). Les périodes
//	                              fermées par la MORT du porteur y sont : le canal n'émet aucun
//	                              lâcher à la mort, c'est le fil des morts qui date la
//	                              fermeture. Une période restée OUVERTE à la fin du film n'y
//	                              est PAS — sa fin n'est pas connue, et la deviner gonflerait
//	                              le chiffre. Le compte des périodes ouvertes est publié.
//	bomb_arms                     JOINTURE de deux canaux, et SEULE voie possible. L'anneau
//	                              `ti=12` date l'armement sans nommer personne, et le moteur
//	                              lui-même n'expose que l'ÉQUIPE qui arme (Lua
//	                              `primitive_carriable_arming_base`, mesure du 2026-09-04) :
//	                              l'acteur se ferme par le canal des armes tenues. DEUX règles
//	                              ordonnées : le LÂCHER (un geste observé) puis, à défaut, le
//	                              PORTEUR ACTIF (une présence constatée) — et l'événement dit
//	                              laquelle l'a nommé (`BombEvent.ActorSource`). Les règles,
//	                              leur fenêtre et le RECALAGE D'HORLOGE qu'elles exigent vivent
//	                              dans bomb_arms.go — le seul endroit de ce noyau où deux
//	                              horloges se rencontrent.
//	bomb_carriers_killed          DÉDUCTION, pas un compteur du moteur. Le fil des kills déjà
//	                              apparié (`KillRef`, tel que `match_kill_events` le porte) est
//	                              croisé aux périodes : un kill compte si sa VICTIME portait la
//	                              bombe à l'instant du kill. Patron validé 10/10 sur
//	                              `skull_carriers_killed` en Oddball (témoin `43716616`,
//	                              bombe_b2_chronologie_test.go) : les 10 événements th=10 non
//	                              couverts par un heartbeat de possession créditaient tous le
//	                              TUEUR d'un porteur, vérifié contre le fil des kills.
//
// # LA RÉSERVE DE `bomb_carriers_killed` : ni camp, ni tir ami
//
// `KillRef` ne porte AUCUNE information d'équipe. Un tir ami sur un porteur de son propre camp
// compterait donc ici, là où le compteur officiel de l'API (pour les modes qui en publient un)
// ne compte que les porteurs ADVERSES. Le suicide, lui, est écarté — tueur et victime
// confondus ne créditent personne — comme un kill dont le tueur n'est pas nommé (xuid 0).
// Cette réserve est écrite plutôt que corrigée : la corriger demanderait le camp de chaque
// joueur, qui n'est pas une entrée de ce noyau.
//
// # DEUX HORLOGES, ET UN SEUL PONT ENTRE ELLES
//
// Les explosions et les armements sont datés sur l'horloge du FILM (celle du manifeste, celle
// des `ObjectiveAction`) ; les périodes de portage et les kills sont datés sur l'horloge du
// MATCH (celle du fil des morts — cf. l'en-tête de bomb_carries.go, qui explique pourquoi le
// portage est calé sur `deathOffsetMS` et non sur l'origine du chunk 1).
//
// QUATRE des cinq statistiques ne consultent qu'UNE horloge et ne demandent donc aucun
// recalage. `bomb_arms` est la seule à joindre les deux : c'est bomb_arms.go qui porte le
// pont, avec sa dérivation et son test de non-régression. `FilmToMatchOffsetMS` n'est lu QUE
// là ; nulle part ailleurs dans ce noyau une horloge n'est convertie.
//
// # ABSENT N'EST PAS ZÉRO
//
// Chaque statistique est un POINTEUR. `nil` dit « source non mesurée » ; `0` dit « mesuré à
// zéro ». Une source non lue laisse son champ à `nil` sur TOUS les joueurs — jamais un zéro
// qui laisserait croire à une mesure. C'est la règle de `KillsInput.Read` et de
// `EquipmentCoverage.KillsRead`, appliquée par champ.

import (
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// BombEventDetonated est la valeur `event_type` d'une explosion de bombe datée, sous
// l'objectif `objectiveevents.ObjectiveTypeBomb`.
const BombEventDetonated = "bomb_detonated"

// bombCarrierKillToleranceMS borne l'écart accepté entre l'instant d'un kill et la FERMETURE
// de la période de portage de sa victime.
//
// Les deux instants viennent du même fil des morts, donc ils coïncident en principe ; la
// tolérance n'absorbe que le bruit d'horloge entre deux lectures de ce fil, exactement comme
// `deathMatchWindowMS` le fait entre la fin d'une vie et une mort (médiane mesurée 34 ms,
// maximum 36). Elle est DÉRIVÉE de cette constante plutôt que réécrite : une troisième copie
// du même seuil re-divergerait. Elle ne s'applique QU'À la borne de fin — un kill antérieur à
// la prise n'est pas un kill de porteur, et élargir la borne de début créditerait un tueur
// pour une bombe que la victime n'avait pas encore.
const bombCarrierKillToleranceMS = deathMatchWindowMS

// BombStatsInput est CE QUE L'APPELANT FOURNIT, déjà décodé. Chaque source porte son propre
// témoin de LECTURE : il distingue « non mesuré » d'un résultat vide, et c'est lui qui décide
// si le champ correspondant sort à `nil` ou à zéro.
type BombStatsInput struct {
	// DetonationsRead : le statborg a été décodé ET le mode est bien de la famille bomb.
	// Faux = `bomb_detonations` reste absent partout.
	DetonationsRead bool
	// Objectives est le calque des actions d'objectif NOMMÉES ET IDENTIFIÉES PAR MANCHE, tel
	// que `Options.Objectives` le porte. Seules les entrées `StatBombDetonations` sont lues :
	// les autres statistiques du même flux (frags, assistances) ne concernent pas ce noyau.
	Objectives []objectiveevents.IdentifiedEvent
	// CarryRead : le canal des armes tenues a été balayé et filtré sur la famille bombe
	// (`BombInput.CarryScanned`). Faux = `bomb_grabs` et `time_as_bomb_carrier_seconds`
	// restent absents partout.
	CarryRead bool
	// Carry est la chronologie reconstruite du portage (`BuildHeldObjectCarry`), horloge du
	// MATCH.
	Carry HeldObjectCarry
	// KillsRead : le fil des kills appariés a été lu. Faux = `bomb_carriers_killed` reste
	// absent partout. La statistique demande les DEUX témoins : sans portage, il n'y a aucune
	// victime à qualifier, et le champ reste absent même si les kills sont lus.
	KillsRead bool
	// Kills sont les morts DÉJÀ appariées à leur tueur, horloge du MATCH.
	Kills []KillRef
	// ArmingsRead : l'anneau d'armement a été balayé ET publié (`BombInput.Scanned` vraie,
	// confrontation locale tenue — cf. bomb_armings.go). Faux = ni faits `bomb_armed`, ni
	// `bomb_arms`. Comme pour les kills, `bomb_arms` demande les DEUX témoins : sans portage,
	// aucun armement ne peut être nommé, et le champ reste absent.
	ArmingsRead bool
	// Armings sont les armements DATÉS publiés par le calque, horloge du FILM.
	Armings []BombArming
	// FilmToMatchOffsetMS est le RECALAGE : millisecondes à AJOUTER à un instant de l'horloge
	// du FILM pour obtenir l'horloge du MATCH. L'appelant le calcule comme
	// `premierPaquetDuFilmUS/1000 − deathOffsetMS` (dérivation complète dans bomb_arms.go).
	// Lu UNIQUEMENT par la jointure de `bomb_arms` ; 16-81 ms mesurés sur les films témoins.
	FilmToMatchOffsetMS int
}

// BombPlayerStats porte les statistiques d'UN joueur. Un champ à `nil` n'a pas été mesuré.
type BombPlayerStats struct {
	// XUID du joueur, en décimal — la même clef que `ObjectiveAction.XUID` et `Track.XUID`.
	XUID string
	// Detonations : explosions de bombe créditées au joueur.
	Detonations *int
	// Arms : armements attribués au joueur par la jointure de bomb_arms.go.
	Arms *int
	// Grabs : ramassages de la bombe.
	Grabs *int
	// TimeAsCarrierSeconds : temps cumulé bombe en main, périodes fermées seulement.
	TimeAsCarrierSeconds *float64
	// CarriersKilled : porteurs de bombe tués (cf. la réserve de l'en-tête).
	CarriersKilled *int
}

// BombEvent est un fait daté de la bombe, sur l'horloge du FILM — la même que celle des
// `ObjectiveAction`, superposable sans recalage.
type BombEvent struct {
	// Type est la valeur `event_type` (cf. BombEventDetonated).
	Type string
	// TimeMS est l'instant sur l'horloge du film.
	TimeMS int
	// XUID de l'acteur, en décimal. Vide = fait daté SANS acteur résolu : il se publie quand
	// même, jamais avec un acteur deviné.
	XUID string
	// ActorSource dit QUELLE RÈGLE a nommé l'acteur — `BombActorSourceDrop` (la bombe a quitté
	// les mains dans la fenêtre : un geste OBSERVÉ) ou `BombActorSourceActiveCarry` (le repli :
	// personne d'autre ne la tenait, une présence CONSTATÉE). Vide quand XUID l'est aussi.
	//
	// Ce champ n'est pas décoratif : les deux règles n'ont pas la même force de preuve, et un
	// lecteur qui les confondrait surestimerait ce que la mesure établit. Il n'a de sens que
	// pour `BombEventArmed` — une explosion est nommée par le statborg, pas par une jointure.
	ActorSource string
}

// BombStatsCoverage dit ce qui a été lu et ce qui a été écarté : publier des chiffres sans
// dire sur quel dénominateur ils portent laisserait croire à l'exhaustivité.
type BombStatsCoverage struct {
	// Les quatre témoins de lecture, recopiés de l'entrée.
	DetonationsRead, CarryRead, KillsRead, ArmingsRead bool
	// Detonations est le nombre d'explosions identifiées (le numérateur publié).
	Detonations int
	// Armings est le nombre d'armements DATÉS fournis — le dénominateur du contrôle de
	// cohérence de la jointure. ArmingsAttributed est le nombre d'armements auxquels un
	// poseur a été nommé, VENTILÉ PAR RÈGLE : ArmingsByDrop pour la source primaire (le
	// lâcher, un geste observé), ArmingsByActiveCarry pour le repli (le porteur actif, une
	// présence constatée). Publier les deux séparément est ce qui permet de lire la
	// couverture SANS confondre deux forces de preuve.
	//
	// Les TROIS raisons de publier un armement SANS acteur se comptent à part :
	// ArmingsNoCarrier (aucune période candidate : ni lâcher dans la fenêtre, ni porteur actif
	// à l'instant armé), ArmingsNoBridge (une période retenue, mais son slot n'est pas ponté)
	// et ArmingsAmbiguous (DEUX porteurs couvraient l'instant : le repli ne tranche jamais).
	//
	// INVARIANT PUBLIÉ, vrai par construction : la somme des `bomb_arms` de tous les joueurs
	// vaut ArmingsAttributed ; ArmingsAttributed == ArmingsByDrop + ArmingsByActiveCarry ; et
	// ArmingsAttributed + ArmingsNoCarrier + ArmingsNoBridge + ArmingsAmbiguous == Armings.
	// Un armement n'est jamais attribué deux fois, ni à un joueur deviné.
	Armings, ArmingsAttributed, ArmingsByDrop, ArmingsByActiveCarry int
	ArmingsNoCarrier, ArmingsNoBridge, ArmingsAmbiguous             int
	// Periods / PeriodsNoBridge / PeriodsOpen / PeriodsByDeath ventilent le portage : combien
	// de périodes, combien sans identité pontée (écartées), combien restées ouvertes à la fin
	// du film (comptées en ramassages mais PAS en temps), combien fermées par la mort.
	Periods, PeriodsNoBridge, PeriodsOpen, PeriodsByDeath int
	// Kills est le nombre de morts fournies — le dénominateur de CarriersKilled.
	Kills int
	// KillsOnCarrier est le nombre de morts retenues comme kill de porteur.
	KillsOnCarrier int
	// Players est le nombre de joueurs publiés.
	Players int
}

// BombMatchStats est le résultat par match : les joueurs et ce qui a été vu.
type BombMatchStats struct {
	// Players, triés par xuid — un ordre total, donc une sortie reproductible.
	Players []BombPlayerStats
	// Coverage dit sur quoi ces chiffres portent.
	Coverage BombStatsCoverage
}

// BuildBombStats calcule les cinq statistiques d'Assaut et les faits datés.
//
// PUR. Un joueur n'apparaît que s'il porte au moins un fait mesuré ; pour lui, chaque champ
// dont la SOURCE a été lue est renseigné (zéro compris — c'est un zéro mesuré), et chaque
// champ dont la source ne l'a pas été reste `nil`. Un joueur qu'aucune source ne nomme n'est
// pas publié : ce noyau ne connaît pas le roster, et inventer une ligne à zéro pour lui serait
// affirmer une mesure qui n'a pas eu lieu.
func BuildBombStats(in BombStatsInput) (BombMatchStats, []BombEvent) {
	out := BombMatchStats{Coverage: BombStatsCoverage{
		DetonationsRead: in.DetonationsRead, CarryRead: in.CarryRead,
		KillsRead: in.KillsRead, ArmingsRead: in.ArmingsRead,
	}}
	tallies := bombTallies{}
	detonations, events := bombDetonationsByXUID(in)
	tallies.detonations = detonations
	out.Coverage.Detonations = len(events)
	arms, armEvents := bombArmsByXUID(in, &out.Coverage)
	tallies.arms = arms
	events = sortedBombEvents(append(events, armEvents...))
	tallies.grabs, tallies.seconds = bombCarryStatsByXUID(in, &out.Coverage)
	tallies.killed = bombCarriersKilledByXUID(in, &out.Coverage)
	out.Players = bombPlayerRows(in, tallies)
	out.Coverage.Players = len(out.Players)
	return out, events
}

// sortedBombEvents range les faits datés par instant croissant sur l'horloge du FILM, puis par
// type et par acteur — un ordre TOTAL, donc une sortie reproductible quel que soit l'ordre
// d'assemblage des sources.
func sortedBombEvents(events []BombEvent) []BombEvent {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].TimeMS != events[j].TimeMS {
			return events[i].TimeMS < events[j].TimeMS
		}
		if events[i].Type != events[j].Type {
			return events[i].Type < events[j].Type
		}
		return events[i].XUID < events[j].XUID
	})
	return events
}

// bombTallies regroupe les quatre comptes intermédiaires — un seul paramètre au lieu de
// quatre, pour tenir le plafond de paramètres.
type bombTallies struct {
	detonations map[string]int
	arms        map[string]int
	grabs       map[string]int
	seconds     map[string]float64
	killed      map[string]int
}

// xuids rend l'union des joueurs nommés par au moins une source mesurée.
func (t bombTallies) xuids() map[string]bool {
	out := map[string]bool{}
	for _, m := range []map[string]int{t.detonations, t.arms, t.grabs, t.killed} {
		for x := range m {
			out[x] = true
		}
	}
	for x := range t.seconds {
		out[x] = true
	}
	return out
}

// bombPlayerRows assemble une ligne par joueur, en appliquant la règle « absent n'est pas
// zéro » champ par champ selon les témoins de lecture.
func bombPlayerRows(in BombStatsInput, t bombTallies) []BombPlayerStats {
	xuids := t.xuids()
	rows := make([]BombPlayerStats, 0, len(xuids))
	for x := range xuids {
		row := BombPlayerStats{XUID: x}
		if in.DetonationsRead {
			row.Detonations = measuredInt(t.detonations[x])
		}
		// `bomb_arms` demande les DEUX canaux : l'anneau date l'armement, le portage le
		// nomme. Sans l'un des deux, le champ reste absent — jamais un zéro qui laisserait
		// croire que le joueur n'a rien armé.
		if in.ArmingsRead && in.CarryRead {
			row.Arms = measuredInt(t.arms[x])
		}
		if in.CarryRead {
			row.Grabs = measuredInt(t.grabs[x])
			row.TimeAsCarrierSeconds = measuredSeconds(t.seconds[x])
		}
		if in.CarryRead && in.KillsRead {
			row.CarriersKilled = measuredInt(t.killed[x])
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].XUID < rows[j].XUID })
	return rows
}

// bombDetonationsByXUID compte les explosions par joueur et rend les faits datés
// correspondants, sur l'horloge du FILM. Aucun recalcul : le nommage vient de
// `NamedEventsFrom` (via `StatBombDetonations`) et l'identité du pont par manche.
func bombDetonationsByXUID(in BombStatsInput) (map[string]int, []BombEvent) {
	if !in.DetonationsRead {
		return nil, nil
	}
	counts := map[string]int{}
	events := make([]BombEvent, 0, len(in.Objectives))
	for _, e := range in.Objectives {
		if e.Stat != objectiveevents.StatBombDetonations || e.XUID == "" {
			continue
		}
		counts[e.XUID]++
		events = append(events, BombEvent{
			Type: BombEventDetonated, TimeMS: e.TimeMS, XUID: e.XUID,
		})
	}
	// L'ordre est posé une seule fois, sur l'ensemble des faits (sortedBombEvents) : trier
	// ici en ferait une seconde définition du même ordre.
	return counts, events
}

// bombCarryStatsByXUID rend les ramassages et le temps de portage, et remplit la ventilation
// des périodes.
//
// UN RAMASSAGE = UNE PÉRIODE OUVERTE, y compris une période restée ouverte à la fin du film :
// le joueur a bien ramassé la bombe, seul son temps est inconnu. Une période dont le slot
// n'est pas ponté ne crédite personne (elle est comptée sous `PeriodsNoBridge`) — attribuer un
// ramassage à un joueur arbitraire serait exactement l'erreur que le pont existe pour éviter.
func bombCarryStatsByXUID(
	in BombStatsInput, cov *BombStatsCoverage,
) (map[string]int, map[string]float64) {
	if !in.CarryRead {
		return nil, nil
	}
	grabs := map[string]int{}
	cov.Periods = len(in.Carry.Periods)
	for _, p := range in.Carry.Periods {
		if p.XUID == 0 {
			cov.PeriodsNoBridge++
			continue
		}
		grabs[strconv.FormatUint(p.XUID, 10)]++
		if p.Ouverte {
			cov.PeriodsOpen++
		}
		if p.FinParMort {
			cov.PeriodsByDeath++
		}
	}
	// Le temps vient de `CarryMSByXUID`, qui écarte déjà les périodes ouvertes et les slots
	// non pontés : le recalculer ici en ferait une deuxième définition du même fait.
	seconds := make(map[string]float64, len(in.Carry.CarryMSByXUID))
	for xuid, ms := range in.Carry.CarryMSByXUID {
		seconds[strconv.FormatUint(xuid, 10)] = float64(ms) / 1000
	}
	return grabs, seconds
}

// bombCarriersKilledByXUID crédite le TUEUR de chaque victime qui portait la bombe à l'instant
// du kill. Les deux sources sont sur l'horloge du MATCH ; aucun recalage.
//
// Sont écartés : le kill sans tueur nommé (xuid 0) et le suicide (tueur = victime) — ni l'un ni
// l'autre ne désigne quelqu'un à créditer.
func bombCarriersKilledByXUID(in BombStatsInput, cov *BombStatsCoverage) map[string]int {
	if !in.CarryRead || !in.KillsRead {
		return nil
	}
	cov.Kills = len(in.Kills)
	byVictim := map[uint64][]HeldObjectPeriod{}
	for _, p := range in.Carry.Periods {
		if p.XUID != 0 {
			byVictim[p.XUID] = append(byVictim[p.XUID], p)
		}
	}
	out := map[string]int{}
	for _, k := range in.Kills {
		if k.KillerXUID == 0 || k.KillerXUID == k.VictimXUID {
			continue
		}
		if !bombCarriedAt(byVictim[k.VictimXUID], k.TimeMS) {
			continue
		}
		out[strconv.FormatUint(k.KillerXUID, 10)]++
		cov.KillsOnCarrier++
	}
	return out
}

// bombCarriedAt dit si l'une des périodes couvre l'instant t (horloge du match), borne de fin
// élargie de bombCarrierKillToleranceMS — cf. la constante pour le pourquoi de l'asymétrie.
func bombCarriedAt(periods []HeldObjectPeriod, t int64) bool {
	for _, p := range periods {
		if t >= int64(p.DebutMS) && t <= int64(p.FinMS)+bombCarrierKillToleranceMS {
			return true
		}
	}
	return false
}

// measuredInt / measuredSeconds rendent un pointeur sur une valeur MESURÉE — le zéro qu'ils portent est
// un zéro mesuré, à ne pas confondre avec le `nil` d'une source non lue.
func measuredInt(v int) *int { return &v }

func measuredSeconds(v float64) *float64 { return &v }
