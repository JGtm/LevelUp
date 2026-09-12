package killcollector

// positions_identity_entree.go — LA COUTURE ENTRE LE COLLECTEUR ET LE REGISTRE D'IDENTITE.
//
// Extrait de positions.go (lot 5.1, revue de vague 4, constat P2) : le fichier frolait le
// plafond de 500 lignes du depot au moment d'ajouter `Bots`/`Participants` a cette fonction.
// Meme fichier LOGIQUE (la couture reste appelee par `buildPositionRows`), autre fichier
// PHYSIQUE — aucun changement de comportement.

import (
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// entreeDuRegistre assemble ce que le collecteur donne au registre d'identite. PURE — aucun film,
// aucune base : c'est la COUTURE par laquelle un test pince ce que la production transmet.
//
// # POURQUOI ELLE EXISTE PLUTOT QU'UN LITTERAL EN LIGNE
//
// Quatre champs comptent ici, et aucun n'est evident a la lecture de l'appelant.
//
// `RosterXUIDs` : sans lui, l'identite par ELIMINATION n'a aucun candidat, et un joueur qui ne
// meurt jamais reste anonyme dans `match_lives` — le defaut meme que le lot P2 ferme.
//
// `BipedCreations` : sans lui, le collecteur retomberait sur le pont par morts alors que la
// cuisson lit le lien DIRECT dans le film — deux producteurs, deux nommages, exactement ce que la
// decision D11 interdit. Son retrait ne casserait AUCUN test unitaire (ceux du registre
// construisent leur propre entree) : c'est le role de cette couture, et de
// `TestEntreeDuRegistrePorteLesCreationsDeBipede`.
//
// `Bots` et `Participants` (lot 5.1, revue de vague 4, constat P2) : SANS EUX,
// `resolveByScoreboard` (identity_registry_scoreboard.go) ne s'execute jamais — le collecteur
// perdait deux garanties que la cuisson tient deja : un index QUE BOT_METADATA DECLARE n'est
// plus jamais resolu vers un humain par le lien direct (verdict I0, `identityDesBotsDeclares`),
// et un siege d'index PARTAGE entre un bot et un humain arrive en cours est DEPARTAGE par la
// fenetre de participation du tableau au lieu d'attribuer TOUT le siege — vies d'avant
// l'arrivee comprises — au xuid de l'humain. C'est la MEME projection que `replaybuild`
// (`replayidentity.BotIdentities`, `participantsDuTableau`), pas une resolution locale.
//
// Les quatre sont la raison des bumps successifs d'[IsolationDecoderRev]. Ecrits en ligne, leur
// retrait laissait toute la suite verte — c'est exactement le defaut que
// `composerPassePositions` avait deja corrige pour le decalage d'entame (constat B1 de la revue
// du 2026-09-06).
func entreeDuRegistre(
	l lecturesDuFilm, ids MatchIdentities, bots []replay.BotIdentity, matchID string,
) replay.IdentityInput {
	return replay.IdentityInput{
		Positions: l.positions, BipedCreations: l.creations, Deaths: l.deaths,
		PlayerIndices: l.idx, RosterXUIDs: rosterUint64(ids.XUIDs),
		Bots: bots, Participants: ids.Participants, MatchID: matchID,
	}
}

// lecturesDuFilm groupe les QUATRE lectures que le collecteur fait du film avant de composer le
// registre. Une structure plutot que quatre parametres de plus : le depot borne a cinq, et un
// appelant qui ajoute une lecture ne doit pas reecrire la signature de la couture.
type lecturesDuFilm struct {
	positions []filmdec.BipedPosition
	creations []filmdec.BipedCreation
	deaths    []replay.Death
	idx       replay.PlayerIndexTable
}
