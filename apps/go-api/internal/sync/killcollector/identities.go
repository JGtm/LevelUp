package killcollector

// identities.go — L IDENTITE D UN JOUEUR DANS UN FILM, ET LA SEULE REGLE QUI LA RESOUT.
//
// Ce fichier existe pour une raison mesuree : la regle de resolution a ete FAUSSE pendant une
// passe entiere de backfill (16 908 morts ecrites, 10 avec un xuid de victime), et une regle
// qui se trompe en silence doit vivre a UN seul endroit, sous son propre en-tete.

import (
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// MatchIdentities : ce que la passe demande a la base sur les participants d un match.
//
// LES CHAMPS NE SE DEDUISENT PAS LES UNS DES AUTRES, et c est le piege qu ils evitent :
// `ShotsFired` est ABSENTE quand la colonne est NULL (« pas de reference »), alors que le joueur,
// lui, EXISTE. Deriver la liste des xuids des cles de la reference perdrait silencieusement ces
// joueurs-la — leurs tirs seraient decodes et jamais attribues.
type MatchIdentities struct {
	// ParNom : `gamertag -> xuid`. Les noms AMBIGUS (deux participants homonymes) en sont
	// ABSENTS : ecrire les morts d un joueur sous le xuid d un autre serait pire que rien.
	ParNom map[string]string
	// XUIDs : tous les participants porteurs d un xuid, dans un ordre stable.
	XUIDs []string
	// ShotsFired : la reference de l API, par xuid. Une entree absente veut dire « aucune
	// reference » — la porte de publication REFUSE alors, elle ne suppose pas.
	ShotsFired map[string]int
	// Equipes : `xuid -> numero d equipe` (`match_participants.team_id`).
	//
	// ELLE VIENT DE LA BASE, et cette passe-ci n a pas d autre source A SA DISPOSITION : elle
	// travaille sur la sortie de `killsource`, qui ne lit pas la trame d etat. Sans elle, la
	// lecture d isolement compterait un adversaire proche comme un accompagnement. Une entree
	// ABSENTE veut dire « equipe non renseignee » — le joueur n entre alors dans aucun camp,
	// jamais dans un camp par defaut.
	//
	// ⚠ LA JUSTIFICATION D ORIGINE — « le film ne porte AUCUN camp (`Track.Team` vaut -1
	// partout) » — EST FAUSSE DEPUIS LE LOT 1.7 (2026-09-14) : le designateur d equipe est ecrit
	// dans l etat par defaut de ti=9 et `grammar.ScanPlayerTeams` le lit ; `Track.Team` et
	// `roster[].team` de l artefact en viennent (ADR 0034, D-9). Ce qui reste vrai est la portee
	// de CETTE passe, pas une propriete du film.
	Equipes map[string]int
	// Participants : le TABLEAU DE L API — les participants du match (bots COMPRIS, sous
	// `bid(N.0)`) et leurs bornes de participation. C est la meme projection que la cuisson
	// (`replaybuild.participantsDuTableau`) : sans elle, le registre d identite ne peut pas
	// departager un siege d index PARTAGE entre un bot et un humain arrive en cours (lot 5.1,
	// revue de vague 4, constat P2 — cf. identity_registry_scoreboard.go pour la regle).
	Participants []replay.Participant
	// replis : le compteur des replis de la passe du film (lot J8.7, replis_de_la_passe.go), recopie
	// du contexte par `IdentitiesForMatch` pour les fonctions pures qui recoivent ces identites. nil :
	// rien n est compte (identites construites a la main, tests).
	replis *decfilm.Compteur
	// Variante et CarteID : `match_registry.game_variant_name` et `map_id`. Feuille : la ligne
	// de match de chaque participant (frags, morts, assistances). Ce sont les trois faits de
	// base que la lecture des PORTEURS au sync demande (plan Emprise vies, lot V2) : la garde de
	// mode (variante), les socles de drapeau (carte, meme cle que la cuisson) et le pont par
	// manche (le triplet de la feuille). Memes colonnes que `ReplayFactsRepo`, cote cuisson.
	Variante string
	CarteID  string
	Feuille  []decfilm.PlayerLine
}

// Resoudre : LE nom que le film donne devient un xuid et un gamertag. UNE SEULE COPIE DE CETTE
// REGLE EXISTE, et c est deliberé — elle a deja coute une passe entiere de backfill.
//
// LE FILM DONNE UN GAMERTAG, tel que le kill-feed le porte (un event sans gamertag n entre pas dans
// le fil, cf. `killsource.buildFeed`). Il se resout contre le roster du match, et un nom inconnu
// reste sans xuid.
func (m MatchIdentities) Resoudre(nom string) (xuid, gamertag string) {
	xuid, gamertag = m.resoudre(nom)
	if xuid == "" {
		// Repli `repli_xuid_vide_pour_nom_inconnu` : le nom ne se resout dans aucune table (lot J8.7).
		m.replis.Declenche(decfilm.NomXuidVidePourNomInconnu)
	}
	return xuid, gamertag
}

// resoudre est la regle de [MatchIdentities.Resoudre], sans le compte.
func (m MatchIdentities) resoudre(nom string) (xuid, gamertag string) {
	return m.ParNom[nom], nom
}
