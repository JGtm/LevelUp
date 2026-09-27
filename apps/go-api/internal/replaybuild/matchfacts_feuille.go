package replaybuild

// matchfacts_feuille.go — LES PROJECTIONS DE LA FEUILLE DE MATCH que la cuisson donne a l assemblage :
// le tableau des participants (registre d identite) et la table des camps (controle des equipes,
// zones, identification des slots d equipe).
//
// SORTIES DE `matchfacts.go` AU LOT J8.7 (2026-09-27) PAR DEPLACEMENT PUR : leurs deux replis s y
// comptent desormais (`repli_participant_sans_xuid_retire`, `repli_camp_inconnu_retire_de_la_table`)
// et le fichier passait 500 lignes.

import (
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

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
//
// `fb` recoit `repli_participant_sans_xuid_retire` pour chaque ligne sans xuid (lot J8.7).
func participantsDuTableau(facts port.MatchFacts, fb *decfilm.Compteur) []replay.Participant {
	out := make([]replay.Participant, 0, len(facts.Players))
	for _, p := range facts.Players {
		if p.XUID == "" {
			fb.Declenche(decfilm.NomParticipantSansXuidRetire)
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
	out, _ := tableDesCamps(facts)
	return out
}

// tableDesCamps est [teamByXUID], plus le nombre de joueurs RETIRES faute de camp connu — le compte
// de `repli_camp_inconnu_retire_de_la_table` (lot J8.7). La table sert trois fois par cuisson ; le
// compte se prend UNE fois, au controle des equipes (`buildReplayOptions`).
func tableDesCamps(facts port.MatchFacts) (map[string]int, int) {
	out := make(map[string]int, len(facts.Players))
	retires := 0
	for _, p := range facts.Players {
		if p.TeamID < 0 {
			retires++
			continue
		}
		out[p.XUID] = p.TeamID
	}
	if len(out) == 0 {
		return nil, retires
	}
	return out, retires
}
