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
// facade d'`objectiveevents` (`NamedEvents`, `SlotIdentity`) re-balaient les enregistrements a
// chaque appel. Les enchainer coûterait trois balayages complets la ou un seul suffit — sur
// une machine qui paie deja le decodage des positions, ce n'est pas un detail (0,6 a 2,4 s et
// jusqu'a 21 Mo par film, mesure du corpus de 22). Depuis le lot 1 de PLAN_CUISSON_PERF, le
// FILM lui-meme n'est de toute facon plus relu : il arrive charge (`filmload.go`).
//
// # POURQUOI C'EST ICI ET PAS DANS `analysis/replay`
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

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/port"
)

// filmStats est ce que le second decodage rend au constructeur.
type filmStats struct {
	score      *replay.ScoreInput
	objectives []objectiveevents.IdentifiedEvent
	// flag porte les lectures du DRAPEAU VIVANT que seul cet etage peut faire : les
	// enregistrements d'entite (les memes que la courbe de score) et les bursts de capture. Les
	// SOCLES s'y ajoutent chez l'appelant (ils viennent du catalogue de carte, pas du film).
	flag replay.FlagInput
	// vip porte la COURONNE VIP : les memes enregistrements d'entite, plus la garde de mode
	// (`Scanned`) posee par l'appelant selon `game_variant_name` — `comp 22 A` vaut `flag_grabs`
	// en CTF, donc la couronne n'est lue que sur un film reconnu VIP.
	vip replay.VipInput
	// skull porte le PORTEUR DU CRANE d'Oddball : les memes enregistrements d'entite, plus la
	// garde de mode posee selon `game_variant_name` — `comp 0 A` est le score de mode de tout
	// mode, donc le porteur n'est lu que sur un film reconnu Oddball.
	skull replay.SkullInput
	// bomb porte L'ARMEMENT DE LA BOMBE d'Assaut : l'horloge du manifeste (le balayage de
	// l'anneau ti=12 se date sur `start_ms` par chunk), plus la garde de mode posee selon
	// `game_variant_name` — TOUTE la famille bomb, One Bomb comprise depuis le 2026-09-04
	// (cf. replaybuild/zones.go, isBombVariant).
	bomb replay.BombInput
}

// readFilmStats decode les enregistrements d'entite et assemble les entrees des deux calques.
//
// Rend un filmStats VIDE (score nil) quand le film n'est pas lisible par cette porte : le
// document sort alors sans courbe de score ET sans couverture de score, ce qui dit « rien n'a
// ete lu » plutot que « rien n'existait ».
//
// LES DEUX REFUS SONT DISTINCTS, et ils l'etaient deja : un film ILLISIBLE (chunks absents du
// cache) et un film SANS MANIFESTE. Le second garde son sens apres le lot 1 — le film se charge
// tres bien sans manifeste, mais aucun de ses chunks n'a alors de type ni de `start_ms`, donc
// rien n'est datable ici (cf. [chunksDuManifeste]).
func readFilmStats(ctx context.Context, matchID string, film *filmsource.Film,
	facts port.MatchFacts, deaths filmDeaths,
) filmStats {
	if film == nil || len(chunksDuManifeste(film)) == 0 {
		return filmStats{} // film illisible ou manifeste absent — deja journalise par filmload.go
	}
	recs, truncated := objectiveevents.StatRecordsCtx(ctx, film, matchID)
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
	// UN SEUL PONT D'IDENTITE POUR LES DEUX CALQUES QUI EN VIVENT (actions d'objectif et
	// drapeau vivant) : la meme table slot -> xuid, resolue AU PLUS UNE FOIS par cuisson.
	pont := &pontParManche{recs: recs, deaths: deathInstantsOf(deaths.list), lines: lines}
	return filmStats{
		score: &replay.ScoreInput{
			Records:    recs,
			Lines:      lines,
			TeamByXUID: teamByXUID(facts),
			TeamScores: facts.TeamScores,
			Truncated:  truncated,
		},
		objectives: identifiedEvents(ctx, matchID, deaths, recs, facts, pont),
		flag:       flagInput(recs, film, pont),
		vip:        vipInput(recs, isVipVariant(facts.GameVariantName)),
		skull:      skullInput(recs, isSkullVariant(facts.GameVariantName)),
		bomb:       bombInput(film, isBombVariant(facts.GameVariantName)),
	}
}

// chunksDuManifeste rend les chunks du film que le MANIFESTE decrit.
//
// ZERO N'EST PAS UN TYPE DE CHUNK : c'est ce que `filmsource.LoadDir` synthetise pour un
// `chunk_NN.bin` present au cache mais ABSENT du manifeste. Mesure du 2026-09-02 sur les
// 1 380 manifestes du cache : trois valeurs seulement — 1 pour l'en-tete, 2 pour les chunks de
// jeu, 3 pour le pied — et jamais 0. Un chunk hors manifeste n'a donc pas de debut connu, et
// l'inscrire a zero dans l'horloge dirait au balayage de l'anneau « ce chunk commence a 0 » au
// lieu de « je ne sais pas » (`filmdec/navpoint_radial_scan.go`, `hasStart`). Un film du cache
// est dans ce cas : `7b0d89c4` porte les fichiers 31 et 32 sans les avoir au manifeste.
func chunksDuManifeste(film *filmsource.Film) []filmsource.ChunkMeta {
	if film == nil {
		return nil
	}
	meta := film.Meta()
	out := make([]filmsource.ChunkMeta, 0, len(meta))
	for _, m := range meta {
		if m.ChunkType == 0 {
			continue
		}
		out = append(out, m)
	}
	return out
}

// bombInput assemble ce que LA BOMBE lit hors film, sous UNE SEULE garde de mode — la
// FAMILLE, One Bomb comprise depuis le 2026-09-04 (la garde de nom est levee, cf.
// replaybuild/zones.go) :
//
//	l'ARMEMENT (schema 33)  l'horloge du manifeste (start_ms par chunk, le balayage de
//	                        l'anneau la demande pour dater sur la meme base que les
//	                        explosions du statborg) ;
//	le PORTAGE (schema 34)  aucune donnee de plus (le canal des armes tenues est deja
//	                        balaye par BuildFromFilm) : la garde seule.
//
// Hors de la famille bomb, il rend un input VIDE : ni balayage, ni calque, ni couverture.
func bombInput(film *filmsource.Film, bomb bool) replay.BombInput {
	if !bomb {
		return replay.BombInput{}
	}
	in := replay.BombInput{CarryScanned: true}
	chunks := chunksDuManifeste(film)
	clock := make(map[int]int, len(chunks))
	for _, c := range chunks {
		clock[c.Index] = c.StartMS
	}
	in.Scanned = true
	in.ChunkStartMS = clock
	return in
}

// skullInput assemble ce que le PORTEUR DU CRANE lit dans le film — les memes enregistrements
// d'entite que la courbe de score, garde par le mode. Hors Oddball, il rend un input VIDE (ni
// records ni Scanned) : le calque ne sera ni construit ni publie.
func skullInput(recs []objectiveevents.StatRecord, isSkull bool) replay.SkullInput {
	if !isSkull {
		return replay.SkullInput{}
	}
	return replay.SkullInput{Scanned: true, Records: recs}
}

// vipInput assemble ce que la COURONNE VIP lit dans le film — les memes enregistrements d'entite
// que la courbe de score et le drapeau, gardes par le mode. Hors VIP, elle rend un input VIDE
// (ni records ni Scanned) : le calque ne sera ni construit ni publie.
func vipInput(recs []objectiveevents.StatRecord, isVip bool) replay.VipInput {
	if !isVip {
		return replay.VipInput{}
	}
	return replay.VipInput{Scanned: true, Records: recs}
}

// flagInput assemble ce que le calque du DRAPEAU VIVANT lit dans le film.
//
// DEUX GRAMMAIRES, DEUX PARCOURS, ET LE SECOND EST INEVITABLE. Les enregistrements d'entite sont
// deja la (ils portent les evenements nommes du drapeau et les progressions du compteur de
// morts) ; les BURSTS DE CAPTURE, eux, sont des evenements de score et se lisent ailleurs dans
// le film. Sans eux le discriminant de mode ne tient pas : la table d'emplacements du drapeau,
// appliquee a un film Oddball, rend 1 470 « prises » et 994 « vols ». Le cout est un parcours de
// plus des paquets deja decoupes — depuis le lot 1, ce n'est plus une relecture du film.
//
// LE PONT D'IDENTITE DESCEND JUSQU'ICI DEPUIS LE 2026-09-06 (schema 42), ET C'EST TOUT LE LOT.
// Le calque le resolvait lui-meme par les seuls INSTANTS DE MORT, qui exigent TROIS instants
// coincidents : un joueur qui meurt moins de trois fois — le meilleur, celui qui porte le
// drapeau — lui echappait par construction, sa prise etait comptee `noBridge` et AUCUN portage
// n'etait publie pour elle. `c0a82e88` : 3 prises, 3 `noBridge`, 0 portage. Le pont COMPLETE
// (par morts + triplet, cf. [pontParManche]) est le meme que celui des actions d'objectif, et
// il vit ICI parce que c'est ici que les lignes de match arrivent — `analysis/replay` continue
// de n'en voir aucune.
//
// IL N'EST DEMANDE QUE SUR UN FILM DE CTF, et la garde est la MEME que celle du calque
// (`replay.attachFlagCarries`) : le verdict de mode vient des trois signaux du FILM, jamais du
// nom de variante. Hors CTF, le pont n'est pas resolu du tout — c'est la protection posee le
// 2026-08-18, quand le deroulage du compteur de morts sur un film d'une autre grammaire montait
// a 19-22 Go.
//
// AUCUN FAIT DE MATCH N'ENTRE DANS LE CALQUE : ce qui descend est une TABLE slot -> xuid. Sans
// lignes de match, `CompletedByLines` rend le pont par morts inchange et l'artefact reste
// exactement celui d'avant — la propriete « publiable hors ligne » est conservee.
func flagInput(recs []objectiveevents.StatRecord, film *filmsource.Film,
	pont *pontParManche) replay.FlagInput {
	return withFlagIdentity(replay.FlagInput{
		Scanned: true,
		Records: recs,
		Bursts:  objectiveevents.CaptureBurstTimes(film),
	}, pont)
}

// withFlagIdentity pose le pont COMPLETE sur l'entree du calque — et SEULEMENT sur un film que
// les trois signaux reconnaissent comme du CTF. Coeur PUR, sans film : c'est la regle, seule.
func withFlagIdentity(in replay.FlagInput, pont *pontParManche) replay.FlagInput {
	signals := objectiveevents.FlagFilmSignalsFrom(in.Bursts,
		objectiveevents.NamedEventsFrom(in.Records, objectiveevents.ObjectiveTypeFlag))
	if signals.IsFlagFilm() {
		in.Identity = pont.identite()
	}
	return in
}

// identifiedEvents nomme les actions d'objectif du film et les attribue a un xuid PAR MANCHE.
//
// LE PONT EST PAR MANCHE, PAR LES INSTANTS DE MORT ([objectiveevents.ResolveRoundIdentity]),
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
func identifiedEvents(ctx context.Context, matchID string, deaths filmDeaths,
	recs []objectiveevents.StatRecord, facts port.MatchFacts,
	pont *pontParManche) []objectiveevents.IdentifiedEvent {
	named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeOf(facts.GameVariantName))
	if len(named) == 0 {
		return nil
	}
	if deaths.err != nil {
		slog.WarnContext(ctx, "replaybuild: fil des morts illisible — actions d'objectif non identifiees",
			"err", deaths.err, "match_id", matchID, "nommees", len(named))
		return nil
	}
	out := objectiveevents.IdentifyNamedEventsByRound(named, pont.identite())
	slog.InfoContext(ctx, "replaybuild: actions d'objectif identifiees par manche",
		"match_id", matchID, "nommees", len(named), "identifiees", len(out), "lignes", len(facts.Players))
	return out
}

// pontParManche est LE pont slot d'entite -> xuid de la cuisson : resolu par manche via les
// instants de mort, COMPLETE par le triplet quand le film est mono-manche et que les lignes de
// match sont la. Coeur PUR, sans I/O — testable sans film.
//
// POURQUOI IL EST MEMORISE, ET POURQUOI IL EST PARESSEUX. Deux calques le consomment (les
// ACTIONS d'objectif et le DRAPEAU VIVANT) et le resolvaient chacun de leur cote sur les MEMES
// enregistrements et le MEME fil des morts — deux deroulages complets du compteur de morts par
// cuisson de CTF. Il est memorise pour n'en payer qu'un ; il est PARESSEUX pour n'en payer AUCUN
// sur les films qu'aucun des deux calques ne lit (le deroulage sur un film d'une autre grammaire
// est ce qui montait a 19-22 Go avant la garde du 2026-08-18).
//
// `lines` vide = pont par morts seul : `CompletedByLines` rend l'identite inchangee, et les deux
// calques restent publiables hors ligne, sans base.
type pontParManche struct {
	recs   []objectiveevents.StatRecord
	deaths []objectiveevents.DeathInstant
	lines  []objectiveevents.PlayerLine
	resolu bool
	id     objectiveevents.RoundIdentity
}

// identite rend le pont, en le resolvant au premier appel.
func (p *pontParManche) identite() objectiveevents.RoundIdentity {
	if !p.resolu {
		p.id = objectiveevents.ResolveRoundIdentity(p.recs, p.deaths).CompletedByLines(p.recs, p.lines)
		p.resolu = true
	}
	return p.id
}

// deathInstantsOf traduit le fil des morts du film dans la forme qu'attend le pont d'identite.
func deathInstantsOf(deaths []replay.Death) []objectiveevents.DeathInstant {
	out := make([]objectiveevents.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		out = append(out, objectiveevents.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS)})
	}
	return out
}

// playerLines traduit les faits de match en lignes d'appariement.
func playerLines(facts port.MatchFacts) []objectiveevents.PlayerLine {
	if len(facts.Players) == 0 {
		return nil
	}
	out := make([]objectiveevents.PlayerLine, 0, len(facts.Players))
	for _, p := range facts.Players {
		out = append(out, objectiveevents.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
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
