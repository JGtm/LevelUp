package replaybuild

// matchfacts.go — LE DEUXIEME DECODAGE DU FILM, ET CE QU'IL ALIMENTE.
//
// # CE QUE CE FICHIER FAIT
//
// Il lit UNE fois les enregistrements d'entite du film (le « statborg ») et en tire les deux
// calques qui en dependent :
//
//	la COURBE DE SCORE     les deux camps et les compteurs vivants de chaque joueur ;
//	les ACTIONS D'OBJECTIF nommees (capture, retour, prise de zone) et attribuees a un xuid.
//
// UN SEUL DECODAGE POUR LES DEUX, et c'est la raison d'etre du fichier : les fonctions de
// facade d'`objectives` (`NamedEvents`, `SlotIdentity`) re-balaient les enregistrements a
// chaque appel. Les enchainer coûterait trois balayages complets la ou un seul suffit — sur
// une machine qui paie deja le decodage des positions, ce n'est pas un detail (0,6 a 2,4 s et
// jusqu'a 21 Mo par film, mesure du corpus de 22). Depuis le lot 1 de PLAN_CUISSON_PERF, le
// FILM lui-meme n'est de toute facon plus relu : il arrive charge (`filmload.go`).
//
// # POURQUOI C'EST ICI ET PAS DANS `games/halo_infinite/film/replay`
//
// Meme frontiere que pour les morts sans revendication : `analysis/` est title-agnostic et pur,
// ce paquet est la couche d'ASSEMBLAGE. C'est lui qui sait ou vit le cache film du titre, et
// c'est lui qui recoit de l'appelant les faits de base (`port.MatchFacts`) que ni l'un ni
// l'autre ne va chercher en base.
//
// # TOUTE DEGRADATION EST JOURNALISEE, JAMAIS AVALEE
//
// Film sans manifeste, faits absents, mode sans famille d'objectif : chacun de ces cas rend un
// artefact PARFAITEMENT VALIDE, seulement plus pauvre. Le taire laisserait croire que le film
// ne portait rien.

import (
	"context"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

// filmStats est ce que le second decodage rend au constructeur.
type filmStats struct {
	score      *replay.ScoreInput
	objectives []decfilm.IdentifiedEvent
	// objectivesUnnamed est le nombre d evenements d objectif que le film NOMMAIT et que le
	// pont par manche n a pas su attribuer. Il voyage jusqu au document parce qu il est le
	// DENOMINATEUR manquant : sans lui, `coverage.objectives.available` compte les rescapes
	// et un calque partiel se lit ~100 % (cf. decfilm.IdentifyNamedEventsByRound).
	objectivesUnnamed int
	// objectivesRefused est le nombre d evenements que la GARDE D EFFECTIF refuse de publier :
	// le match compte plus de joueurs que le statborg n a de slots d entite
	// (decfilm.RosterFitsStatborg). Il voyage pour la meme raison que le precedent —
	// un calque muet doit dire ce que son silence coute.
	objectivesRefused int
	// flag porte les lectures du DRAPEAU VIVANT que seul cet etage peut faire : les
	// enregistrements d'entite (les memes que la courbe de score) et les bursts de capture. Les
	// SOCLES s'y ajoutent chez l'appelant (ils viennent du catalogue de carte, pas du film).
	flag replay.FlagInput
	// vip porte la COURONNE VIP : les memes enregistrements d'entite, plus la garde de mode
	// (`Scanned`) posee selon `game_variant_name` par `replay.GardesDeLaVariante` — `comp 22 A`
	// vaut `flag_grabs` en CTF, donc la couronne n'est lue que sur un film reconnu VIP.
	vip replay.VipInput
	// skull porte le PORTEUR DU CRANE d'Oddball : les memes enregistrements d'entite, plus la
	// garde de mode posee selon `game_variant_name` (`replay.GardesDeLaVariante`) — `comp 0 A` est
	// le score de mode de tout mode, donc le porteur n'est lu que sur un film reconnu Oddball.
	skull replay.SkullInput
	// bomb porte L'ARMEMENT DE LA BOMBE d'Assaut : l'horloge du manifeste (le balayage de
	// l'anneau ti=12 se date sur `start_ms` par chunk), plus la garde de mode posee selon
	// `game_variant_name` — TOUTE la famille bomb, One Bomb comprise depuis le 2026-09-04
	// (cf. `replay.GardesDeLaVariante`, qui porte la garde depuis le 2026-09-28).
	bomb replay.BombInput
	// statborgIdentity est le pont slot d entite -> xuid PAR MANCHE, deja resolu pour les deux
	// calques d objectif. Il voyage jusqu au document parce que le REGISTRE d identite le
	// publie avec sa provenance (`identity.statborgSlots`) — il ne le recalcule pas.
	statborgIdentity decfilm.RoundIdentity
}

// statborgDuFilm LIT la section statborg d un film : les enregistrements d entite, les instants
// de rafale de capture, le temoin de troncature et l horloge des chunks du manifeste.
//
// C EST LA SEULE MOITIE QUI A BESOIN DU FILM (lot 4.1.2) : l assemblage des entrees de calque, lui,
// est PUR ([assemblerFilmStats]) et sert les deux chemins — le film et les faits persistes.
//
// Rend une section VIDE (aucun enregistrement) quand le film n est pas lisible par cette porte :
// le document sort alors sans courbe de score ET sans couverture de score, ce qui dit « rien n a
// ete lu » plutot que « rien n existait ».
//
// LES DEUX REFUS SONT DISTINCTS, et ils l etaient deja : un film ILLISIBLE (chunks absents du
// cache) et un film SANS MANIFESTE. Le second garde son sens apres le lot 1 — le film se charge
// tres bien sans manifeste, mais aucun de ses chunks n a alors de type ni de `start_ms`, donc
// rien n est datable ici (cf. [chunksDuManifeste]).
func statborgDuFilm(ctx context.Context, matchID string, film *decfilm.Film) replay.FilmStatborg {
	if film == nil || len(chunksDuManifeste(film)) == 0 {
		return replay.FilmStatborg{} // illisible ou sans manifeste — deja journalise par filmload.go
	}
	recs, truncated := decfilm.StatRecordsCtx(ctx, film, matchID)
	return replay.FilmStatborg{
		Records: recs, BurstMS: decfilm.CaptureBurstTimes(film), Truncated: truncated,
		ChunkStartMS: horlogeDesChunks(film),
	}
}

// assemblerFilmStats assemble les entrees des calques A PARTIR DE LA SECTION STATBORG, sans
// toucher au film. Cf. `filmfacts_stats.go`.
func assemblerFilmStats(ctx context.Context, matchID string, sb replay.FilmStatborg,
	facts port.MatchFacts, deaths filmDeaths,
) filmStats {
	if len(sb.Records) == 0 && len(sb.ChunkStartMS) == 0 {
		return filmStats{} // section vide : rien n a ete lu (cf. statborgDuFilm)
	}
	recs, truncated := sb.Records, sb.Truncated
	if len(recs) == 0 {
		slog.InfoContext(ctx, "replaybuild: aucun enregistrement d'entite dans le film — courbe de score vide",
			"match_id", matchID)
	}
	lines := playerLines(facts)
	if facts.Empty() {
		slog.WarnContext(ctx, "replaybuild: aucun fait de match fourni — pas de compteurs de joueur "+
			"ni d'actions d'objectif, et l'identite des camps retombe sur les frags",
			"match_id", matchID, "enregistrements", len(recs))
	}
	// UN SEUL PONT D'IDENTITE POUR LES CALQUES QUI EN VIVENT (actions d'objectif, drapeau vivant,
	// porteur du crane) : la meme table slot -> xuid, resolue AU PLUS UNE FOIS par cuisson. Le pont
	// et les entrees des calques de porteur vivent dans `replay` depuis le 2026-09-28 : le
	// collecteur de sync les lit par les memes fonctions (`replay.PortagesAuSync`).
	pont := replay.NouveauPontParManche(recs, deathInstantsOf(deaths.list), lines)
	gardes := replay.GardesDeLaVariante(facts.GameVariantName)
	objectifs, nonNommes, refuses := identifiedEvents(ctx, matchID, deaths, recs, facts, pont)
	return filmStats{
		score: &replay.ScoreInput{
			Records:    recs,
			Lines:      lines,
			TeamByXUID: teamByXUID(facts),
			TeamScores: facts.TeamScores,
			Truncated:  truncated,
		},
		objectives:        objectifs,
		objectivesUnnamed: nonNommes,
		objectivesRefused: refuses,
		flag:              replay.EntreeDuDrapeau(recs, sb.BurstMS, pont),
		vip:               replay.EntreeDeLaCouronne(recs, gardes.VIP),
		skull:             replay.EntreeDuCrane(recs, gardes.Crane, pont),
		bomb:              replay.EntreeDeLaBombe(sb.ChunkStartMS, gardes.Bombe),
		statborgIdentity:  pont.Identite(),
	}
}

// chunksDuManifeste rend les chunks du film que le MANIFESTE decrit.
//
// ZERO N'EST PAS UN TYPE DE CHUNK : c'est ce que `decfilm.LoadDir` synthetise pour un
// `chunk_NN.bin` present au cache mais ABSENT du manifeste. Mesure du 2026-09-02 sur les
// 1 380 manifestes du cache : trois valeurs seulement — 1 pour l'en-tete, 2 pour les chunks de
// jeu, 3 pour le pied — et jamais 0. Un chunk hors manifeste n'a donc pas de debut connu, et
// l'inscrire a zero dans l'horloge dirait au balayage de l'anneau « ce chunk commence a 0 » au
// lieu de « je ne sais pas » (`filmdec/navpoint_radial_scan.go`, `hasStart`). Un film du cache
// A ETE dans ce cas : `7b0d89c4` portait le 2026-09-02 les fichiers 31 et 32 sans les avoir au
// manifeste — un film archive avant sa finalisation, restaure complet le 2026-09-16. Depuis le
// lot L3 (2026-09-23), la cuisson REFUSE un tel film (`refuserMorceauxHorsManifeste`) : ce filtre
// reste la garde de datation des films charges hors de cette porte.
func chunksDuManifeste(film *decfilm.Film) []decfilm.ChunkMeta {
	if film == nil {
		return nil
	}
	meta := film.Meta()
	out := make([]decfilm.ChunkMeta, 0, len(meta))
	for _, m := range meta {
		if m.ChunkType == 0 {
			continue
		}
		out = append(out, m)
	}
	return out
}

// identifiedEvents nomme les actions d'objectif du film et les attribue a un xuid PAR MANCHE.
//
// LE PONT EST PAR MANCHE, PAR LES INSTANTS DE MORT ([decfilm.ResolveRoundIdentity]),
// comme la couronne VIP, le drapeau et le porteur du crane — et PLUS par les TOTAUX du match. Le
// slot d'entite statborg est REATTRIBUE d'une manche a l'autre : un pont par totaux collait les
// actions d'apres-bascule au mauvais joueur (le compteur de morts repart de zero a chaque manche,
// si bien qu'il ne voyait que la premiere).
//
// IL EST COMPLETE PAR LE TRIPLET SUR LES FILMS MONO-MANCHE (correctif du 2026-09-06). Le pont par
// morts exige TROIS instants coincidents : un joueur qui meurt moins de trois fois lui echappe par
// construction, et ce sont les MEILLEURS joueurs — ceux qui portent le drapeau. `d173b1a8c`
// (2026-08-28) a bascule ce calque du pont par TRIPLET vers le pont par MORTS en annoncant une
// « neutralite mono-manche prouvee par construction » : elle etait vraie contre le pont PLAT PAR
// MORTS, mais ce calque-ci n'etait pas sur ce pont-la — il etait sur le triplet. Mesure sur
// `c0a82e88` (une manche) : 17 actions avant, 12 apres, les deux SEULES actions de famille `flag`
// perdues avec le slot de leur auteur (7 frags, 2 morts). `CompletedByLines` rend la parite, sans
// toucher a la correction multi-manche. Voir son en-tete pour les trois gardes et le controle
// croise qui etablit les attributions rendues.
//
// LES LIGNES DE MATCH RESTENT FACULTATIVES : sans elles, `CompletedByLines` rend l'identite
// inchangee et le calque reste publiable hors ligne, exactement comme le drapeau.
//
// TROIS REFUS EXPLICITES, ET AUCUN N'EST AVALE : sans famille d'objectif (mode sans table nommee,
// variante inconnue) ou sans aucun emplacement nomme, aucun nom n'est possible ; sans fil des
// morts lisible, aucun slot ne peut etre apparie par manche. Chacun rend nil, journalise.
//
// LE FIL DES MORTS N'EST PLUS RELU ICI (lot 1, 2026-09-02) : il arrive DEJA LU, par `deaths`.
// C'est la meme lecture que `killRefs` consomme (kills.go), la ou les deux ouvraient et
// reparsaient chacune le chunk highlight. Le second decodage du statborg, lui, n'a jamais ete
// refait — `recs` est reutilise.
// LE SECOND RETOUR EST LE NOMBRE D'ACTIONS QUE LE PONT N'A PAS NOMMEES, et il n'est pas une
// commodite de journal : il devient `coverage.objectives.noSlot` dans l'artefact servi (cf.
// replay/objectives.go). Sans lui la perte n'existait que dans un `slog` non durable, qui ne
// voyage ni dans le document ni dans le contrat — et `noSlot` valait 0 sur les 111 artefacts du
// parc, sans une seule exception. Le fil des morts ILLISIBLE rend tout le calque perdu, et c'est
// cette perte-la qu'il faut publier, pas zero. LES DEUX COMPTES SONT RESTREINTS AUX FAMILLES
// D'OBJECTIF (D.2, 2026-09-13) : ils font le denominateur de `coverage.objectives`, ou `kills` et
// `assists` n'ont rien a faire (raison mesuree : replay/objectives.go).
func identifiedEvents(ctx context.Context, matchID string, deaths filmDeaths,
	recs []decfilm.StatRecord, facts port.MatchFacts,
	pont *replay.PontParManche) ([]decfilm.IdentifiedEvent, int, int) {
	named := decfilm.NamedEventsFrom(recs, decfilm.ObjectiveTypeOf(facts.GameVariantName))
	if len(named) == 0 {
		return nil, 0, 0
	}
	// GARDE D'EFFECTIF, symetrique a `IsFlagFilm` pour le PORTAGE (lot 6.7-B1, item 5) : au-dela
	// de huit joueurs le statborg n'a plus de slot pour dire de qui il parle, et les comptes
	// publies ne correspondent a personne — `4f77afc1` annoncait 65 prises de drapeau pour 4 a
	// l'oracle. Le calque se tait ENTIEREMENT, et le refus se compte et se journalise.
	if sieges := siegesAuCoupDEnvoi(facts); !decfilm.RosterFitsStatborg(sieges) {
		slog.WarnContext(ctx, "replaybuild: actions d'objectif REFUSEES — effectif hors du format du statborg",
			"match_id", matchID, "nommees", len(named), "sieges", sieges,
			"lignes", len(facts.Players), "slots", decfilm.StatPlayerSlots)
		return nil, 0, decfilm.CountObjectiveFamily(named)
	}
	if deaths.err != nil {
		slog.WarnContext(ctx, "replaybuild: fil des morts illisible — actions d'objectif non identifiees",
			"err", deaths.err, "match_id", matchID, "nommees", len(named))
		return nil, decfilm.CountObjectiveFamily(named), 0
	}
	out, _ := decfilm.IdentifyNamedEventsByRound(named, pont.Identite())
	nonNommes := decfilm.CountObjectiveFamily(named) - decfilm.CountObjectiveFamily(out)
	slog.InfoContext(ctx, "replaybuild: actions d'objectif identifiees par manche",
		"match_id", matchID, "nommees", len(named), "identifiees", len(out),
		"nonNommees", nonNommes, "lignes", len(facts.Players))
	return out, nonNommes, 0
}

// siegesAuCoupDEnvoi compte les SIEGES qu'un match ouvre, et non les lignes de sa feuille.
//
// Un siege est occupe par au plus une personne a la fois : une ligne de BOT (il remplit une
// place liberee) et une ligne de joueur ARRIVE EN COURS (il en prend une) n'en ouvrent aucun.
// Trois lignes peuvent ainsi se partager un seul siege — c'est le cas de cinq films d'arene du
// parc, qui portent 9 ou 10 lignes pour huit sieges (cf. `decfilm.RosterFitsStatborg`).
func siegesAuCoupDEnvoi(facts port.MatchFacts) int {
	n := 0
	for _, p := range facts.Players {
		if p.JoinedInProgress || analysis.IsBot(p.XUID) {
			continue
		}
		n++
	}
	return n
}

// deathInstantsOf traduit le fil des morts du film dans la forme qu'attend le pont d'identite.
func deathInstantsOf(deaths []replay.Death) []decfilm.DeathInstant {
	out := make([]decfilm.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		out = append(out, decfilm.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS)})
	}
	return out
}

// playerLines traduit les faits de match en lignes d'appariement.
func playerLines(facts port.MatchFacts) []decfilm.PlayerLine {
	if len(facts.Players) == 0 {
		return nil
	}
	out := make([]decfilm.PlayerLine, 0, len(facts.Players))
	for _, p := range facts.Players {
		out = append(out, decfilm.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
	}
	return out
}

// rosterXUIDs projette la feuille de match vers le roster d'appoint du rejeu.
//
// LA REGLE N'EST PLUS ICI (lot 1.0, revue R1, constat R1-1) : elle vit dans
// `replay.RosterXUIDsOf`, avec le champ qu'elle remplit, pour que le FIXTURE d'entrees puisse
// l'appeler lui aussi — il passait `nil`, et un joueur a zero mort manquait alors a la table
// d'index du golden sans que rien ne le dise. Cette fonction-ci n'est plus que l'adaptateur du
// type de la base vers celui de la regle.
func rosterXUIDs(facts port.MatchFacts) []uint64 {
	xuids := make([]string, 0, len(facts.Players))
	for _, p := range facts.Players {
		xuids = append(xuids, p.XUID)
	}
	return replay.RosterXUIDsOf(xuids)
}

// participantsDuTableau projette la feuille de match vers le TABLEAU que le registre d'identite
// consomme (cf. replay.Participant).
//
// TOUTES LES LIGNES ENTRENT, BOTS COMPRIS, et c'est tout l'objet : `rosterXUIDs` ignore
// justement les `bid(N.0)` parce que l'elimination raisonne sur des xuids. Le tableau, lui, sert
// a nommer les corps que la table d'index ne nomme PAS — et ce sont precisement les bots.
//
// L'INSTANT D'ARRIVEE NE VOYAGE QUE S'IL EST DECLARE. Une ligne sans `joined_in_progress` ou
// sans `first_joined_time` ne departage aucun siege : la porter avec un zero fabriquerait une
// arrivee au coup d'envoi, ce qui donnerait TOUTES les vies du siege a l'humain.
func participantsDuTableau(facts port.MatchFacts) []replay.Participant {
	out := make([]replay.Participant, 0, len(facts.Players))
	for _, p := range facts.Players {
		if p.XUID == "" {
			continue
		}
		out = append(out, replay.Participant{
			ID: p.XUID, JoinedInProgress: p.JoinedInProgress, JoinMatchMS: p.JoinMatchMS,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// teamByXUID rend le camp de chaque joueur. Un camp inconnu (-1) n'entre PAS dans la table :
// il ferait entrer un faux camp dans la somme des frags qui identifie les slots d'equipe.
func teamByXUID(facts port.MatchFacts) map[string]int {
	out := make(map[string]int, len(facts.Players))
	for _, p := range facts.Players {
		if p.TeamID < 0 {
			continue
		}
		out[p.XUID] = p.TeamID
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
