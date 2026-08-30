// Package service — match_view_builders_assists.go : le bloc « assistances » de l'onglet
// Combat (ASSISTANT -> TUEUR ASSISTÉ).
//
// Fichier dédié plutôt qu'un ajout à match_view_builders_combat.go (432 lignes) : le sujet
// est distinct de l'agrégation killer→victim et le fichier voisin est proche du seuil.
//
// CE QUE CE BUILDER FAIT : il pose le gamertag du TUEUR (résolu depuis le scoreboard) sur
// des paires déjà agrégées en base, et il décide de l'ÉMISSION DU BLOC. Il ne compte rien —
// tout le décompte est fait par Q21d, y compris les éliminations volées.
package service

import "levelup/go-api/internal/domain"

// buildAssistPairs assemble le bloc `combat_tab.assist_pairs`.
//
// LA SEULE DÉCISION DE CE BUILDER, ET ELLE EST STRUCTURANTE : émettre ou non le bloc.
//
//	scope.MatchDeaths == 0  ->  nil. Le match n'a AUCUNE ligne de film : ni le titre ni la
//	                            passe n'ont produit quoi que ce soit. Publier un bloc vide
//	                            forcerait l'écran à choisir un message là où il n'y a rien
//	                            à dire — et sur un titre sans décodeur, ce message
//	                            apparaîtrait sur TOUS les matchs.
//	sinon                   ->  le bloc, avec sa portée. `MeasuredDeaths == 0` y est un
//	                            ÉTAT PUBLIÉ (« non mesuré »), pas une absence.
//
// LES DEUX NOMS SE RÉSOLVENT AU MÊME ENDROIT — le scoreboard, par xuid.
//
// Le tueur l'a toujours fait (comme buildKillerVictimPairs). L'assistant, lui, arrivait
// nommé PAR LE FILM (Q21d rend `assist_gamertag`) : deux sources de noms dans un seul
// graphe, donc un même joueur affichable sous deux orthographes — le film écrit le
// gamertag capté à l'enregistrement, le scoreboard sert celui de l'API (alias compris).
// C'est exactement la raison pour laquelle Q32d, côté escouade, ne rend AUCUN gamertag.
//
// Repli : le nom du film quand le xuid est absent du scoreboard (un assistant peut avoir
// quitté avant la fin et manquer au tableau des scores) — mieux vaut le nom d'hier que
// pas de nom. Pour le TUEUR le repli reste la chaîne VIDE : c'est le contrat livré, et le
// front a déjà son masque « Joueur #### ». On ne fabrique aucun nom, et on ne recopie
// jamais un xuid dans un champ de nom.
func buildAssistPairs(
	raw []domain.MatchAssistPairRaw,
	scope domain.MatchAssistScopeRaw,
	scoreboard []domain.ScoreboardRaw,
) *domain.MatchAssistPairs {
	if scope.MatchDeaths == 0 {
		return nil
	}
	gtByXUID := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		if s.Gamertag != "" {
			gtByXUID[s.XUID] = s.Gamertag
		}
	}
	pairs := make([]domain.MatchAssistPair, 0, len(raw))
	for _, r := range raw {
		assistGT := gtByXUID[r.AssistXUID]
		if assistGT == "" {
			assistGT = r.AssistGamertag
		}
		pairs = append(pairs, domain.MatchAssistPair{
			AssistXUID:     r.AssistXUID,
			AssistGamertag: assistGT,
			KillerXUID:     r.KillerXUID,
			KillerGamertag: gtByXUID[r.KillerXUID],
			AssistCount:    r.AssistCount,
			StolenCount:    r.StolenCount,
			AvgAssistPct:   r.AvgAssistPct,
		})
	}
	return &domain.MatchAssistPairs{
		MeasuredDeaths: scope.MeasuredDeaths,
		Pairs:          pairs,
	}
}
