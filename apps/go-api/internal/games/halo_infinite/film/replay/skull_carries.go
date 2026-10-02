package replay

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// skull_carries.go — LA REGLE : de quoi est faite une periode de portage du CRANE d'Oddball.
//
// # Le principe, en une phrase
//
// On ne decode PAS le crane : on lit QUI le PORTE. Le porteur est le joueur dont les TICS DE
// SCORE DE MODE montent — en Oddball, `comp 0 A` (le score de mode par joueur) compte les tics de
// possession (`skull_scoring_ticks`). Un TRAIN de tics d'un meme joueur EST une periode de
// portage ; le crane est a la position de ce joueur, et le client le pose sur sa piste deja
// publiee, comme la couronne VIP et le drapeau porte.
//
// # Ce qu'il a fallu pour l'etablir, et pourquoi ce canal
//
// Le portage a resiste a CINQ campagnes (proximite, traversee, score PERSONNEL) : negatifs, biais
// des longs portages. Le canal des TICS de score de MODE, lui, tient : gate oracle porteur
// PRINCIPAL correct 7/7 films, gate terrain manche 1 de d9781168 prises 9/9 et porteurs
// d'intervalle 8/9 (seuil 8/9). L'emplacement (`comp 0 A`) est identifie par l'oracle films
// confondus, PAS ajuste au film terrain. Detail : `ODDBALL_PORTEUR_PROTOCOLE.md` +
// `TERRAIN_*.log`.
//
// # PAR MANCHE, et c'est structurel
//
// Les tics sont lus MANCHE PAR MANCHE (`SeriesByRound`), et le porteur d'un train est nomme par
// l'identite de SA manche ([objectives.RoundIdentity.AtRound]) : le slot d'entite est
// reattribue d'une manche a l'autre. Une bascule de manche NE FERME PAS un portage par une fausse
// prise — les trains sont bornes par les seuls TROUS DE TICS, et chaque manche est un parcours
// distinct, donc le dernier train d'une manche et le premier de la suivante ne se melangent pas.
//
// # Ce qui n'est PAS decide ici, et c'est delibere
//
// Le MODE. `comp 0 A` est le score de mode de N'IMPORTE quel mode. La garde est chez l'APPELANT
// (`replaybuild`, qui connait `game_variant_name`), comme la couronne VIP et la colline de KOTH :
// ce paquet consomme un `SkullInput` et ne devine aucun mode. Un film non-Oddball ne fournit pas
// de `SkullInput.Scanned`, et le calque reste vide.

// skullTickGapMS : au-dela de ce trou entre deux tics d'un meme joueur, la periode de portage se
// ferme. Les tics tombent a ~1 Hz pendant le portage ; trois secondes separent nettement deux
// periodes distinctes sans couper une periode continue (meme valeur que l'instrument du gate).
const skullTickGapMS = 3000

// # LA DEMI-FENETRE DE TIC, AUX DEUX BORNES (lot 6.7-B1, item 2, 2026-09-11)
//
// LE DEFAUT, ET SA MESURE. Un train est borne par son PREMIER et son DERNIER tic : un train de
// n tics couvre donc (n-1) largeurs de tic, alors qu'il temoigne de n SECONDES de possession.
// La seconde d'amorce (entre la prise et le premier tic) et celle de chute (entre le dernier
// tic et le lacher) manquaient. Mesure du 2026-09-11, quatre films Oddball cuits hors ligne
// contre l'oracle API `time_as_skull_carrier_seconds` : 1 113,0 s publiees pour 1 249,0 s,
// soit 0,891 — et le manque est PROPORTIONNEL au nombre de periodes, pas a leur duree
// (100 periodes, 136,0 s manquantes, 1,36 s par periode).
//
// POURQUOI CE CALQUE ET PAS LES AUTRES (audit du 2026-09-10 §3.4). Le biais est propre aux
// calques a TRAIN DE TICS. Le drapeau lit des EVENEMENTS de statborg et SUR-mesure (le lacher
// volontaire ne ferme rien) ; la bombe et le crane PORTE lisent des transitions du canal des
// armes tenues et sont bornes exactement des deux cotes. Poser une demi-fenetre sur eux
// aggraverait leur biais au lieu de le corriger.
//
// LA LARGEUR N'EST PAS UNE CONSTANTE ECRITE ICI : elle se MESURE sur le film, par la mediane
// des ecarts entre tics consecutifs D'UN MEME TRAIN (cf. [skullTickWidthFrames]). Un film qui
// ne donne aucun ecart a mesurer ne recoit aucune fenetre.

// skullTickWidthFrames rend la LARGEUR d'un tic de possession, en images de l'axe publie,
// mesuree sur les trains du film lui-meme.
//
// La mediane, et pas la moyenne : un train peut sauter un tic (replication perdue), ce qui
// produit un ecart double ; la moyenne s'en trouverait tiree vers le haut, la mediane non.
// Seuls les ecarts INTRA-TRAIN entrent (au-dela de [skullTickGapMS] ce n'est plus un ecart
// entre deux tics mais la separation de deux periodes). Zero quand aucun ecart n'est mesurable
// — un film dont tous les trains tiennent en un seul tic, ou un axe sans echelle.
func skullTickWidthFrames(recs []types.StatRecord, ctx matchClock, cons *objectives.ReplisALaConsultation) int {
	var ecarts []int
	for _, byRound := range objectives.SeriesByRound(recs, objectives.SkullTicksComponent(), false, cons) {
		for _, pts := range byRound {
			inst := skullTickInstants(pts)
			for i := 1; i < len(inst); i++ {
				if d := inst[i] - inst[i-1]; d > 0 && d <= skullTickGapMS {
					ecarts = append(ecarts, d)
				}
			}
		}
	}
	if len(ecarts) == 0 {
		return 0
	}
	sort.Ints(ecarts)
	return ctx.slackFrames(ecarts[len(ecarts)/2])
}

// skullHalfTickFrames rend la demi-fenetre a poser DE PART ET D'AUTRE d'un train, en images.
//
// POURQUOI (w-1)/2 ET NON w/2. L'intervalle publie est FERME sur la grille : `[t0, t1]` compte
// `t1 - t0 + 1` images, donc les bornes rendent DEJA une image de plus que l'ecart qu'elles
// enserrent. Poser w/2 de chaque cote ferait publier n largeurs de tic PLUS une image, et un
// porteur passerait au-dessus de son propre oracle (mesure : 2 joueurs sur 28, +0,1 s chacun).
// (w-1)/2 est la plus grande fenetre symetrique en images ENTIERES qui ne depasse jamais
// n largeurs de tic — le sens dans lequel on veut se tromper.
func skullHalfTickFrames(recs []types.StatRecord, ctx matchClock, cons *objectives.ReplisALaConsultation) int {
	w := skullTickWidthFrames(recs, ctx, cons)
	if w <= 1 {
		return 0
	}
	return (w - 1) / 2
}

// SkullInput est CE QUE L'APPELANT FOURNIT du crane. Entree de DONNEES, comme `Flag` et `Vip`.
//
// LA GARDE DE MODE EST ICI, chez l'appelant : `comp 0 A` est le score de mode de tout mode, donc
// seul un appelant qui SAIT que le match est Oddball (par `game_variant_name`) doit poser
// `Scanned`. `Scanned` faux = ni calque ni couverture.
type SkullInput struct {
	// Scanned dit que l'appelant a RECONNU un film Oddball et fournit de quoi lire.
	Scanned bool
	// Records sont les enregistrements d'entite du film — les MEMES que la courbe de score, le
	// drapeau et la couronne : ils portent les tics de score de mode (`comp 0 A`), les prises
	// (`comp 21 B`) et les progressions du compteur de morts qui identifient les slots. Aucun fait
	// de match n'entre : le porteur se nomme par les instants de mort, et le calque est donc
	// publiable hors ligne.
	Records []types.StatRecord
	// Identity est le pont slot statborg -> xuid que l'APPELANT a deja resolu, PAR MANCHE.
	// Valeur zero : ce paquet le resout lui-meme par les seuls instants de mort (cf.
	// [skullIdentityOf]), et l'artefact reste publiable hors ligne.
	//
	// POURQUOI L'APPELANT PEUT FAIRE MIEUX, ET DE COMBIEN. Le pont par morts exige TROIS
	// progressions coincidentes du compteur de morts (`deathInstantMin`) : un joueur qui meurt
	// moins de trois fois dans la manche lui echappe PAR CONSTRUCTION — et en Oddball c'est
	// souvent le porteur, que son equipe protege. Son train de tics part alors en `NoBridge` et
	// aucun intervalle n'est publie. Mesure du 2026-09-10 sur les quatre films Oddball du parc
	// (cuisson hors ligne, oracle API `time_as_skull_carrier_seconds`) : `43716616` perdait
	// 62,3 s sur son plus gros porteur, `c88ec007` 25,8 s, `d9781168` un train. Le pont COMPLETE
	// (morts + triplet de la feuille + elimination par manche) que la couche d'assemblage resout
	// deja pour les actions d'objectif et le drapeau ferme ce trou, sans qu'aucun fait de match
	// n'entre ici : ce qui descend est une TABLE slot -> xuid.
	Identity objectives.RoundIdentity
}

// SkullCarryScan porte ce que le film rend du porteur. Les lectures voyagent ensemble, et
// `Scanned` dit qu'elles ont abouti : une liste vide sans lui serait indistinguable d'un film
// non-Oddball.
type SkullCarryScan struct {
	Scanned bool
	// Records : les enregistrements d'entite (tics de score de mode + prises).
	Records []types.StatRecord
	// Identity est le pont slot statborg -> xuid PAR MANCHE (par les instants de mort).
	Identity objectives.RoundIdentity
	// Consultations : l enregistreur des replis a la consultation du document (lot J8.7-bis), que
	// les series du crane lues ici recoivent. Nil : rien n est note.
	Consultations *objectives.ReplisALaConsultation
}

// L'AXE DE TEMPS est le `matchClock` partagé (match_clock.go) : la conversion match -> frames
// était la même que celle du drapeau et de la couronne, elle n'est plus écrite qu'une fois.
// Le trou de fermeture d'un train se traduit en frames par `slackFrames(skullTickGapMS)` : un
// portage dont le dernier tic est a moins d'un trou de la fin de l'axe n'a ete ferme par aucun
// fait — il court jusqu'au bout (le film s'arrete pendant le portage). Au-dela, une fermeture
// est un vrai fait (une chute suivie d'une reprise, ou une fin de manche).

// skullRawCarry est une periode de portage reconstruite : un train de tics d'un meme slot dans une
// manche, en horloge du MATCH.
type skullRawCarry struct {
	xuid       string
	round      int
	t0MS, t1MS int
}

// buildSkullCarries rend les periodes de portage du crane en FRAMES et la couverture du calque.
//
// Rend (nil, nil) quand rien n'a ete balaye (film non-Oddball), et (nil, couverture) quand le film
// est Oddball mais qu'aucune periode ne sort — la couverture dit alors POURQUOI.
//
// `presence` est l'index des vies bipedes publiees (cf. [carrierPresence]) et decide, par
// [carrierPresence.gate], de ce qui sort : un portage dont les pistes publiees prouvent que le
// porteur etait AILLEURS est un FANTOME et part en `CarrierAbsent` ; un portage qui deborde d'une
// vie NOMMEE du porteur est ROGNE a elle. Partout ailleurs — porteur jamais nomme, ou vie ANONYME
// couvrant l'intervalle — le gate S'ABSTIENT : on ne rejette pas l'inconnu. `presence` zero
// (tests) laisse donc passer tous les trains.
func buildSkullCarries(scan SkullCarryScan, ctx matchClock, presence carrierPresence) ([]SkullCarry, *SkullCarriesCoverage) {
	if !scan.Scanned {
		return nil, nil
	}
	cov := &SkullCarriesCoverage{SkullFilm: true, Grabs: skullGrabCount(scan.Records, scan.Consultations)}
	raws := skullCarryIntervals(scan.Records, scan.Identity, scan.Consultations)
	cov.Trains = len(raws)
	openThreshold := ctx.frames - 1 - ctx.slackFrames(skullTickGapMS)
	// La demi-fenetre se mesure UNE FOIS par film (cf. l'en-tete de [skullTickWidthFrames]).
	demi := skullHalfTickFrames(scan.Records, ctx, scan.Consultations)
	out := make([]SkullCarry, 0, len(raws))
	for _, r := range raws {
		if r.xuid == "" {
			cov.NoBridge++
			continue
		}
		f0 := ctx.frameOfMatchMS(int64(r.t0MS))
		if f0 < 0 || f0 >= ctx.frames {
			cov.OutOfWindow++
			continue
		}
		f1 := max(clampFrame(ctx.frameOfMatchMS(int64(r.t1MS)), ctx.frames), f0)
		// LA DEMI-FENETRE DE TIC, AUX DEUX BORNES — posee APRES le rejet hors fenetre (un
		// train qui commence avant l'axe reste hors fenetre : ce n'est pas la demi-fenetre qui
		// doit l'y ramener) et AVANT le gate de presence (une seconde d'amorce hors de toute
		// vie du porteur ne doit pas etre publiee).
		if f0 -= demi; f0 < 0 {
			f0 = 0
		}
		f1 = clampFrame(f1+demi, ctx.frames)
		// Gate de PRESENCE : le porteur doit etre sur la carte pendant le portage.
		var ok bool
		if f0, f1, ok = presence.gate(r.xuid, f0, f1); !ok {
			cov.CarrierAbsent++
			continue
		}
		closed := f1 < openThreshold
		out = append(out, SkullCarry{XUID: r.xuid, T0: f0, T1: f1, Closed: closed})
		if closed {
			cov.Closed++
		} else {
			cov.Open++
		}
	}
	cov.Carries = len(out)
	return out, cov
}

// skullCarryIntervals reconstruit les periodes de portage : les trains de tics de score de mode,
// PAR MANCHE et par slot, chaque train nomme par l'identite de sa manche. Ordre TOTAL (instant,
// puis manche, puis xuid) : sans lui le parcours de map rendrait une sortie differente a chaque
// execution.
func skullCarryIntervals(recs []types.StatRecord, identity objectives.RoundIdentity,
	cons *objectives.ReplisALaConsultation) []skullRawCarry {
	bySlot := objectives.SeriesByRound(recs, objectives.SkullTicksComponent(), false, cons)
	var out []skullRawCarry
	for slot, byRound := range bySlot {
		for round, pts := range byRound {
			inst := skullTickInstants(pts)
			if len(inst) == 0 {
				continue
			}
			xuid := identity.AtRound(round, slot)
			start, last := inst[0], inst[0]
			for _, t := range inst[1:] {
				if t-last > skullTickGapMS {
					out = append(out, skullRawCarry{xuid: xuid, round: round, t0MS: start, t1MS: last})
					start = t
				}
				last = t
			}
			out = append(out, skullRawCarry{xuid: xuid, round: round, t0MS: start, t1MS: last})
		}
	}
	trierPortagesDeCrane(out)
	return out
}

// skullTickInstants rend un instant par UNITE gagnee par le compteur de tics (deroulage par
// valeur : la meme valeur reemise ne rajoute rien, si bien que chaque tic est date a sa PREMIERE
// emission).
func skullTickInstants(pts []types.ScorePoint) []int {
	var out []int
	prev := int64(0)
	for _, p := range pts {
		for ; prev < p.Value; prev++ {
			out = append(out, p.TimeMS)
		}
	}
	return out
}

// skullGrabCount rend le nombre total de PRISES du crane (`comp 21 B`), toutes manches — le
// denominateur de couverture, independant des trains de tics.
func skullGrabCount(recs []types.StatRecord, cons *objectives.ReplisALaConsultation) int {
	total := 0
	for _, byRound := range objectives.SeriesByRound(recs, objectives.SkullGrabsComponent(), false, cons) {
		for _, pts := range byRound {
			if n := len(pts); n > 0 {
				total += int(pts[n-1].Value)
			}
		}
	}
	return total
}

// attachSkullCarries pose les periodes de portage du crane sur le document, avec leur couverture.
//
// LE PONT D'IDENTITE (slot statborg -> xuid) SE FAIT ICI, comme pour la couronne et le drapeau,
// par les seuls INSTANTS DE MORT et PAR MANCHE — aucune base. `reg.DeathOffsetMS()` cale l'horloge
// des enregistrements (meme horloge que le fil des morts) sur l'axe des frames.
func attachSkullCarries(ctx context.Context, doc *ReplayDocument, opt Options, reg IdentityRegistry, clock replayClock,
	deduced map[int]bool) {
	in := opt.Skull
	if !in.Scanned {
		return
	}
	scan := SkullCarryScan{
		Scanned:       true,
		Records:       in.Records,
		Identity:      skullIdentityOf(in, opt),
		Consultations: opt.consultations(),
	}
	presence := carrierPresenceOf(doc.Tracks, deduced)
	carries, cov := buildSkullCarries(scan, matchClock{
		origin: clock.origin, step: clock.step, frames: clock.frames,
		deathOffsetMS: reg.DeathOffsetMS(),
	}, presence)
	// LA PORTE DE PRESENCE EST PARTAGEE AVEC LA BOMBE : c est ICI qu elle prend le nom du crane.
	clock.fb.DeclencheN(fallback.NomCranePorteurSansVieNommee, presence.porteursSansVieNommee())
	doc.SkullCarries = carries
	if doc.Coverage != nil {
		doc.Coverage.SkullCarries = cov
	}
	logSkullCarriesCoverage(ctx, cov)
}

// skullIdentityOf rend le pont d'identite du calque : celui de l'appelant s'il en a fourni un,
// sinon celui que ce paquet resout par les seuls INSTANTS DE MORT.
//
// MEME REGLE, MEME ECRITURE QUE [flagIdentityOf], et pour la meme raison : la preference va a
// l'appelant, qui ne peut que COMPLETER. Le pont que la couche d'assemblage fournit part des
// MEMES enregistrements et du MEME fil des morts (`replay.ScanDeaths`, celui-la meme que
// `BuildFromFilm` pose dans `opt.Deaths`), PLUS les completions par le triplet de la feuille et
// par l'elimination : c'est un SUR-ENSEMBLE — aucun slot nomme ici ne peut y perdre son nom ni
// y changer de joueur.
//
// [objectives.RoundIdentity.Resolved] et non un compte de noms : un appelant hors ligne qui
// ne fournit RIEN doit tomber sur la resolution locale, alors qu'un pont fourni qui ne nomme
// personne est une reponse, pas un silence.
func skullIdentityOf(in SkullInput, opt Options) objectives.RoundIdentity {
	if in.Identity.Resolved() {
		return in.Identity
	}
	return objectives.ResolveRoundIdentity(in.Records, deathInstantsOf(opt.Deaths), opt.consultations())
}

// logSkullCarriesCoverage journalise ce que le calque publie — et ce qu'il ecarte.
func logSkullCarriesCoverage(ctx context.Context, cov *SkullCarriesCoverage) {
	if cov == nil {
		return
	}
	slog.InfoContext(ctx, "rejeu : portage du crane d'Oddball",
		"prises", cov.Grabs, "trains", cov.Trains, "portages", cov.Carries,
		"fermes", cov.Closed, "ouverts", cov.Open,
		"sansPont", cov.NoBridge, "horsFenetre", cov.OutOfWindow,
		"porteurAbsent", cov.CarrierAbsent)
}
