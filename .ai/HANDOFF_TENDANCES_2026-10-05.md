# Handoff — page « Tendances » (2026-10-05)

Statut : **étude et maquette, rien d'implémenté**. Branche `claude/tendances-multi-horizons-37316d`,
fichiers non commités. Prochaine étape : plan d'implémentation soumis au skill `plan-review`,
pas avant l'accord de l'utilisateur.

## Fichiers

| Fichier | Rôle |
|---|---|
| `.ai/MAQUETTE_TENDANCES_2026-10-04.html` | Maquette v14 sur données réelles (1 158 matchs de JGtm, « aujourd'hui » = 27/09/2026). ECharts 6.1 depuis jsdelivr. Notes d'étude repliées en bas de page : historique v4 → v14. |
| `.ai/TENDANCES_extraction_roles_2026-10-05.sql` | Extraction Prendre / Défendre / Tenir par match (joueur + équipe + effectif), à lancer sur une COPIE de `shared_matches_v2.duckdb`. |
| `Downloads/Tendances_LevelUp_2026-10-04.zip` | Dossier d'origine : étude du 30/09 (§6.7 périmé, la maquette fait foi), maquette v3, `extraction_matchs.sql`. |
| `.ai/thought_log.md` | Entrées du 04 et du 05/10 (v4 à v14), une par version. |

## Décisions de l'utilisateur (font foi)

- Emplacement : 6e onglet « Tendances » d'Ascension, sélecteur Solo / Escouade, seul filtre = type de partie.
- **Horizons : 7 / 30 / 90 / 365 j**, chacun comparé à la période d'avant de même durée, seuil de 10 matchs.
- **Jamais de « soirée »** : ni horizon, ni groupe de lignes, ni enchaînements (dit deux fois).
- **Pas de valeurs attendues du matchmaking** (écart au FDA / frags / morts attendus) : calculées par le jeu
  sur le niveau estimé, elles reviennent vers 0 quand le MMR rattrape le joueur. Elles restent dans la vue match et dans Sessions.
- Le matchmaking ramène vers 50 % : la progression se lit dans le **MMR adverse**. Le taux de victoire et
  le score de performance (rang parmi les 50 derniers matchs de la même chaîne, `sync/performance.go`)
  reviennent vers 50 par construction.
- **LUSR ne couvre pas le classé** : un graphique LUSR va toujours avec un graphique MMR.
- Graphiques **affichés d'office** : ni clic pour déplier, ni liste déroulante. Les choix se font par boutons visibles.
- Forme préférée pour « stat face au MMR adverse » : **un graphique à deux axes sur l'axe des temps** (stat
  à gauche en bleu, MMR adverse à droite en orange). Validée le 02/10, redemandée le 05/10. Ne pas lui substituer
  une autre forme sans la montrer d'abord.
- Retirés : « Taux de victoire selon la statistique, par horizon » ; « Par match : assistances et meilleure
  série » ; « Parts : tête, arme lourde, équipement » (leurs lignes restent dans la matrice).

## Contenu de la page (v14, onglet Solo)

1. **Matrice** « Indicateurs par mois et par horizon » : 12 mois + 365 / 90 / 30 / 7 j, groupes repliables :
   Niveau (MMR adverse et équipe, LUSR ×3), Résultats (taux de victoire, score de performance), Combat,
   Façon de jouer, Objectifs (Prendre / Défendre / Tenir en part de l'équipe), Activité (matchs, heures,
   jours joués, taux d'abandon). Clic sur une ligne = défilement jusqu'à son graphique.
2. **Évolution** : boutons d'horizon et de pas (7 j : match / jour ; 30 j : match / jour / semaine ;
   90 j : jour / semaine / mois ; 365 j : semaine / mois). Moyenne de la période d'avant en pointillé.
   - Face au MMR adverse (8 graphiques à deux axes) : FDA, frags, morts, taux de victoire (masqué au pas
     « match »), score de performance, précision, durée de vie, rendement + résistance (réunis).
   - Niveau, combat et style (4) : MMR adversaires / équipe, LUSR par type de partie, dégâts infligés /
     subis, objectifs Prendre / Défendre / Tenir (part de l'équipe, parité en pointillé).
3. Calendrier des résultats ; « Moyenne par match en défaite et en victoire » (avec rendement et
   résistance) ; médailles par match ; matchs par type de partie.

Onglet Escouade : matrice (11 lignes) + 6 graphiques (avec / sans l'escouade, FDA par membre, part des
frags par membre en barres empilées, part des frags de l'équipe, écart de MMR, matchs joués ensemble).

## Briques Go / web à réutiliser à l'implémentation

- Rôles d'objectif : `analysis/narrative/objective_roles.go` (source unique) et `analysis/sessionusage/objectives.go`.
- Rendement et résistance : `analysis/combat_yield.go` ; FDA agrégé selon l'ADR 0006.
- Découpage temporel : `temporal.BucketByGranularity` + `ResolveAdaptive`, sur des bornes en jours glissants
  (`temporal.Period` ne connaît pas 90 j).
- Contraintes ADR 0036 : un seul chargement borné par requête (730 j pour avoir la période d'avant de 365 j),
  lectures `_latest`, cache invalidé au sync.
- Gating par capacité, jamais par slug : MMR (`team_mmr`), LUSR, objectifs, film (équipement).
- Web : jetons `objective-role-*`, matrice en tableau TanStack plutôt qu'une carte ECharts (couleurs `oklch` non
  interpolées par ECharts).

## Points ouverts

- Pas maquetté faute de données dans la copie : coordination (riposte, appui, morts en isolement), score
  escouade (`squad_score.go`), XP de carrière par horizon.
- Jours et semaines découpés en UTC dans la maquette : à faire en heure locale dans l'app.
- Escouade : pas de section « Face au MMR adverse » (non demandée).
- Données de JGtm : 7 j = une seule journée jouée, donc « 7 j par jour » est vide ; 30 j = 3 jours joués.
- Contrôle de la maquette : script jsdom avec ECharts local (`apps/web/node_modules` du checkout principal),
  qui ignore les erreurs canvas de jsdom. Une modification doit garder zéro erreur et aucun id en double sur les
  20 combinaisons horizon × pas.
