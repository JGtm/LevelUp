# LUSR v2 — Prédire si on était supposés gagner un match

> Idée à implémenter, pas encore codée. Doc créé 2026-05-28.

## L'idée en une phrase

Pour chaque match passé, on peut calculer rétroactivement : "quelle était la
chance que mon équipe gagne, AVANT que le match commence ?" — en se basant
uniquement sur les ratings LUSR v2 des joueurs à ce moment-là.

## Pourquoi c'est gratuit

Quand le modèle met à jour un rating après un match, il utilise déjà cette
probabilité en interne : c'est elle qui décide combien le rating doit bouger.
Si on était favoris à 80% et qu'on a gagné, peu de mouvement (résultat attendu).
Si on était à 20% et qu'on a gagné, gros boost (l'upset). On n'a donc pas
besoin de nouvelle math — juste de l'exposer comme une valeur de sortie.

## Ce qu'on en ferait côté UX

Quelques exemples concrets :

| Situation | Affichage |
|---|---|
| Victoire à 70% de chances | "Match attendu. Pas de surprise." |
| Défaite à 70% de chances de gagner | "Upset. Coup dur sur un match gagnable." |
| Victoire à 25% de chances | "Belle perf. Vous avez retourné un match donné perdant." |
| Match à 50/50 | "Match équilibré. Tout pouvait basculer." |

Sur la durée, ça permet :
- **Filtrer les défaites** entre "matchs gagnables" et "matchs vraiment au-dessus de nous"
- **Identifier les bonnes performances** (victoires sur matchs donnés perdants)
- **Repérer les tilts** (défaites multiples sur matchs gagnables = peut-être pause à prendre)
- **Mesurer le "match equity"** sur une session : "vous avez joué 10 matchs, score attendu = 5.2 victoires, score réel = 7 → +1.8 sur-performance"

## Les données qu'on a déjà

Tout est déjà là, rien à backfill :

- Le rating de chaque joueur juste avant chaque match est dans
  `player_skill_state_v2` (table append-only — on a un snapshot par match)
- Les paramètres du modèle (échelle, bruit, marge de match nul) sont dans
  `lusr_hyperparams_v2` ou par défaut dans le code Go

Il manque juste le code qui combine ces deux pour sortir un nombre.

## Comment ça s'implémenterait

1. **Fonction de calcul pur** dans `internal/analysis/skill_v2/predict.go` :
   prend les ratings des 2 équipes, retourne (P_team_A_gagne, P_match_nul, P_team_B_gagne).
   Pas d'accès DB, testable unitairement.

2. **Loader sync** : pour un match donné, requête les ratings "juste avant"
   ce match dans `player_skill_state_v2`. Fallback rating initial (`μ_0=25,
   σ_0=8.33`) si le joueur n'avait jamais joué dans ce mode.

3. **Service** : oriente la fonction depuis la perspective du joueur tracké
   (sa team = team A) → retourne "votre probabilité de gagner ce match".

4. **Affichage** : soit ajouter une colonne `expected_win_prob` dans la table
   `match_skill_rank` au moment de l'écriture canonique (Stratégie C), soit
   un endpoint HTTP calculé à la volée. La 1ère option évite tout recalcul
   côté serveur quand on affiche l'historique.

Effort estimé : **demi-journée**. Tests inclus.

## Limites à signaler à l'utilisateur

Le modèle a deux angles morts qui ne sont pas encore corrigés dans le code
actuel :

1. **Les escouades** : si vous jouez à 4 stack régulier, le modèle vous voit
   comme 4 joueurs indépendants au lieu d'une équipe coordonnée. La proba
   peut donc sous-estimer de 5 à 10 points votre chance de gagner.

2. **Les coéquipiers et adversaires non trackés** : pour ces joueurs, on
   utilise le rating par défaut "joueur moyen" (25.0). Si on tombe sur des
   adversaires en réalité bien meilleurs que la moyenne (ou bien pires),
   la proba sera fausse dans cette direction.

Ces deux limites s'estompent au fur et à mesure que la base de joueurs
trackés grandit et que la Phase 2 (gestion des escouades) sera implémentée.

## Connexions avec le reste du plan LUSR v2

Cette fonctionnalité **ne dépend de rien d'autre** dans la roadmap restante.
Elle peut être livrée dans n'importe quel ordre par rapport aux autres
phases. C'est un quick win — beaucoup de valeur produit pour peu d'effort.

Voir `.ai/LUSR_V2_HANDOFF.md` pour la roadmap complète.
