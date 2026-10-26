# Handoff — page « Tendances » (2026-10-05)

Statut : **implémentée le 2026-10-05 / 06** sur `claude/tendances-mockup-review-c21607` (qui part de `65aff8ebd`,
la v14 commitée sur `claude/tendances-multi-horizons-37316d`), sous le plan `.ai/PLAN_TENDANCES_2026-10-05.md`
(toutes les étapes closes, journal et découvertes). La maquette v15 et ce handoff sont sur la même branche.
Les sections qui suivent décrivent l'étude ; ce qui a été tranché à l'implémentation est dans « Implémentation ».

## Implémentation (état au 2026-10-06)

- **API** : `POST /players/{slug}/pages/trends`, corps `TrendsQueryRequest{view, game_type, selected_gamertags,
  exact_composition, locale}` ; une réponse porte la matrice, les séries à tous les pas sur 365 j, le calendrier,
  les blocs par horizon, les types joués, les capacités et, en vue Escouade, les membres (xuid, gamertag ; joueur
  principal d'abord). Changer d'horizon ou de pas ne relit rien : le web découpe.
  Go : `internal/analysis/trends/` (calcul pur, `BuildSolo` / `BuildSquad`), `internal/domain/trends.go`,
  `internal/service/trends_service*.go`, `internal/service/teammates/teammates_service_trends.go` (vue Escouade :
  population de la page Escouade, ADR 0033), `internal/api/handlers/trends.go`, `internal/api/wire/registry_trends.go`.
  Web : `apps/web/src/features/tendances/` (route `.../ascension/tendances`), manifeste `lib/i18n/manifests/tendances.toml`.
- **Tranché à l'implémentation** : Solo = matchs sans ami (comme les pages Solo de l'app) ; dégradation par
  capacités (`team_mmr`, `lusr`, `ranked`, `match.skill.snapshot`, `match.objective.stats`,
  `film.usage_summary`) ; seuil de 10 matchs des deux côtés gardé (case sans comparaison, infobulle) ; jours,
  semaines et mois en heure locale (`cfg.UserTimezone`) ; type de partie = chaîne de performance ; heures = temps
  joué du joueur ; matchs sans MMR exclus de l'écart de MMR ; CSR affiché dès qu'il y a des matchs classés
  (variante = groupe de file, « Classé »).
- **Écarts assumés avec la maquette** (contrainte « briques existantes seulement ») : matrice en
  `Heatmap2DChart` par groupe (pas de groupes repliables, pas de clic vers le graphique, pas de case grise :
  case neutre + infobulle) ; courbes à points de taille fixe (ni nuage à taille variable ni tendance lissée) ;
  calendrier en heatmap sans case pâle sous 3 matchs ; coefficient r dans l'infobulle des haltères ; bascules au
  gabarit de `FilterOmnibar` (composant local à la feature) ; aucune barre de dégradé en légende.
- **Non tranché, laissé comme dans la maquette** : écart chiffré absent des cases d'horizon (couleur + infobulle) ;
  graphique « MMR des adversaires et de l'équipe » conservé.
- **Sélection d'escouade** : la même que la page Escouade (clés `localStorage` par joueur, lues et écrites par
  `features/squad/squadSelectionStorage.ts` et `exactComposition.ts`) ; « Enregistrer » actif grâce aux membres
  de la réponse.
- **Vérifications** : référence de la maquette rejouée par test (`TestReferenceMaquette*`, 1e-9) ; données réelles
  de JGtm comparées aux formules de la maquette à date égale (écarts tous expliqués, journal du plan 8.2) ; revue
  adversariale en deux rondes (ronde 1 : 10 constats recevables, tous corrigés ; ronde 2 : 2 constats, corrigés,
  relus par le superviseur) ; gates complets Go et web verts (journal du plan 8.1).
- **Livraison** : trois commits sur la branche (`3267bd96e` API Go, `829f8a0b0` onglet web, `78116af1e`
  documents), puis fusion de `feat/v75` (`05cc77397`, seul conflit : le journal, résolu par union) et avance
  rapide de `feat/v75` sur ce commit, poussée le 2026-10-06 (accord de l'utilisateur).
- **Reste** : CI de `feat/v75` sur cette fusion (gate d'autorité) ; gate visuel de l'utilisateur dans l'app
  (témoins : matrice par groupe, barre « Horizon » collante, filtre « Type de partie » après bascule de vue, vue
  Escouade avec « Enregistrer »).

## Fichiers

| Fichier | Rôle |
|---|---|
| `.ai/MAQUETTE_TENDANCES_2026-10-04.html` | Maquette v15 sur données réelles (1 158 matchs de JGtm, « aujourd'hui » = 27/09/2026). ECharts 6.1 depuis jsdelivr. Notes d'étude repliées en bas de page : historique v4 → v15. |
| `.ai/TENDANCES_extraction_roles_2026-10-05.sql` | Extraction Prendre / Défendre / Tenir par match (joueur + équipe + effectif), à lancer sur une COPIE de `shared_matches_v2.duckdb`. |
| `Downloads/Tendances_LevelUp_2026-10-04.zip` | Dossier d'origine : étude du 30/09 (§6.7 périmé, la maquette fait foi), maquette v3, `extraction_matchs.sql`. |
| `.ai/thought_log.md` | Entrées du 04 et du 05/10 (v4 à v15), une par version. |

## Décisions de l'utilisateur (font foi)

- Emplacement : 6e onglet « Tendances » d'Ascension, sélecteur Solo / Escouade, seul filtre = type de partie.
- **Rôle de la page : voir les tendances de ses statistiques**, pas l'analyse poussée. Celle-ci reste dans
  « Séries temporelles » (FDA par match, CSR / LUSR par match, XP de carrière, coordination, précision par arme,
  cartes) : ces blocs n'entrent pas dans Tendances.
- **Horizons : 7 / 30 / 90 / 365 j**, chacun comparé à la période d'avant de même durée, seuil de 10 matchs.
- **Un seul sélecteur d'horizon** pour tous les blocs sous la matrice (Évolution, calendrier, « Victoires et
  défaites », médailles, matchs par type de partie). Aucun bloc n'a son propre choix d'horizon ; seul le pas de
  temps reste propre à « Évolution ».
- **Jamais de « soirée »** : ni horizon, ni groupe de lignes, ni enchaînements (dit deux fois).
- **Pas de valeurs attendues du matchmaking** (écart au FDA / frags / morts attendus) : calculées par le jeu
  sur le niveau estimé, elles reviennent vers 0 quand le MMR rattrape le joueur. Elles restent dans la vue match et dans Sessions.
- Le matchmaking ramène vers 50 % : la progression se lit dans le **MMR adverse**. Le taux de victoire et
  le score de performance (rang parmi les 50 derniers matchs de la même chaîne, `sync/performance.go`)
  reviennent vers 50 par construction.
- **CSR pour le classé, LUSR pour le non classé** (correction du 05/10 : la consigne « un graphique LUSR va
  toujours avec un graphique MMR » visait le CSR). Un graphique LUSR va avec un graphique CSR.
- **Le contenu s'adapte aux types de partie du joueur** : lignes de la matrice, graphiques et options du filtre
  n'apparaissent que pour ce qu'il joue (pas de CSR sans classé, pas de LUSR pour un joueur de classé seul,
  options « Classé » et « BTB » du filtre si elles ont des matchs).
- Graphiques **affichés d'office** : ni clic pour déplier, ni liste déroulante. Les choix se font par boutons visibles.
- Forme préférée pour « stat face au MMR adverse » : **un graphique à deux axes sur l'axe des temps** (stat
  à gauche en bleu, MMR adverse à droite en orange). Validée le 02/10, redemandée le 05/10. Ne pas lui substituer
  une autre forme sans la montrer d'abord.
- Retirés : « Taux de victoire selon la statistique, par horizon » ; « Par match : assistances et meilleure
  série » ; « Parts : tête, arme lourde, équipement » (leurs lignes restent dans la matrice) ; barres de
  dégradé en légende du calendrier et de la matrice.
- **Une maquette se juge sur son contenu et sa forme.** Les cases grises faute de matchs dans la copie, l'écart
  avec les conventions de l'app et les états vides sont des sujets d'implémentation (section suivante).

## À régler à l'implémentation (pas dans la maquette)

- **Sens de « Solo »** : dans l'app, les pages Solo ne prennent que les matchs sans amis (`match_context = solo`,
  `domain/filters.go`) ; l'onglet Solo de la maquette prend tous les matchs (JGtm, 365 j : 553 seul, 589 avec un
  coéquipier suivi). Coller à ce qui existe.
- **Titres sans MMR ni objectifs (Halo 5)** : une page plus pauvre est acceptée, gérée par les capacités, jamais
  par le slug. Sans MMR par match (capacité `team_mmr` du titre, `no_team_mmr = true` côté Halo 5) : le groupe
  « Niveau » et l'axe de droite disparaissent, et les statistiques de « Face au MMR adverse » se tracent seules,
  sur un axe. Sans `match.objective.stats` : pas de groupe « Objectifs ». Sans `match.skill.snapshot` : pas de CSR.
- **Seuil à 7 j** : avec 10 matchs exigés sur la semaine et sur la semaine d'avant, la colonne 7 j reste sans
  comparaison pour un joueur occasionnel (JGtm, 22 matchs par semaine en moyenne : 33 semaines comparables sur 51).
  Choisir le seuil et dessiner l'état « pas assez de matchs ».
- **365 j contre l'année d'avant** : demande 730 jours d'historique synchronisé ; à défaut la colonne n'a pas de comparaison.
- Jours et semaines découpés en UTC dans la maquette : à faire en heure locale dans l'app.

## Contenu de la page (v15, onglet Solo)

1. **Matrice** « Indicateurs par mois et par horizon » : 12 mois + 365 / 90 / 30 / 7 j, groupes repliables :
   Niveau (MMR adverse et équipe, CSR, LUSR ×3), Résultats (taux de victoire, score de performance), Combat,
   Façon de jouer, Objectifs (Prendre / Défendre / Tenir en part de l'équipe), Activité (matchs, heures,
   jours joués, taux d'abandon). Clic sur une ligne = défilement jusqu'à son graphique, quand elle en a un.
2. **Barre « Horizon »** (7 / 30 / 90 / 365 j), visible pendant le défilement : elle pilote tout ce qui suit.
3. **Évolution** : boutons de pas (7 j : match / jour ; 30 j : match / jour / semaine ; 90 j : jour / semaine /
   mois ; 365 j : semaine / mois). Moyenne de la période d'avant en pointillé.
   - Face au MMR adverse (8 graphiques à deux axes) : FDA, frags, morts, taux de victoire (masqué au pas
     « match »), score de performance, précision, durée de vie, rendement + résistance (réunis).
   - Niveau, combat et style (5) : CSR par file classée, LUSR par type de partie (non classé), MMR adversaires /
     équipe, dégâts infligés / subis, objectifs Prendre / Défendre / Tenir (part de l'équipe, parité en pointillé).
4. Calendrier des résultats ; « Moyenne par match en défaite et en victoire » (avec rendement et résistance) ;
   médailles par match ; matchs par type de partie (un bâton par pas de temps). Tous sur l'horizon choisi.

Onglet Escouade : matrice (11 lignes), barre « Horizon » + 6 graphiques (avec / sans l'escouade, FDA par membre,
part des frags par membre en barres empilées, part des frags de l'équipe, écart de MMR, matchs joués ensemble).

## Briques Go / web à réutiliser à l'implémentation

- Rôles d'objectif : `analysis/narrative/objective_roles.go` (source unique) et `analysis/sessionusage/objectives.go`.
- Rendement et résistance : `analysis/combat_yield.go` ; FDA agrégé selon l'ADR 0006.
- CSR : vue `match_csrs_latest` (les matchs de placement n'ont pas de valeur) ; tracé existant dans
  `features/timeseries/TimeseriesSkillProgression.tsx` (une série par file et par type de classement).
- Découpage temporel : `temporal.BucketByGranularity` + `ResolveAdaptive`, sur des bornes en jours glissants
  (`temporal.Period` ne connaît pas 90 j).
- Contraintes ADR 0036 : un seul chargement borné par requête (730 j pour avoir la période d'avant de 365 j),
  lectures `_latest`, cache invalidé au sync.
- Gating par capacité, jamais par slug : MMR (`team_mmr`), CSR (`match.skill.snapshot`), LUSR, objectifs
  (`match.objective.stats`), film (équipement).
- Web : jetons `objective-role-*`, matrice en tableau TanStack plutôt qu'une carte ECharts (couleurs `oklch` non
  interpolées par ECharts).

## Points ouverts

- **Écart chiffré dans les cases d'horizon de la matrice** (proposition de la revue, non tranchée) : la case
  affiche la valeur, et seule sa couleur dit si elle monte ou descend ; l'écart est dans l'infobulle. Variante :
  écrire l'écart dans la case et laisser la case neutre quand il est plus petit que la marge d'erreur.
- Pas maquetté faute de données dans la copie : frags par arme, vies isolées, score escouade (`squad_score.go`).
- CSR : JGtm n'a qu'une valeur (667, Or 1, 06/11/2025 ; 7 de ses 8 matchs classés sont des placements). La carte
  « CSR par file classée » de la maquette est un emplacement en pointillé, pas une courbe.
- Escouade : pas de section « Face au MMR adverse » (non demandée).
- Données de JGtm : 7 j = une seule journée jouée, donc « 7 j par jour » est vide ; 30 j = 3 jours joués.
- Infobulle de « Moyenne par match en défaite et en victoire » : quatre phrases, à ramener à trois.
- Contrôle de la maquette : script jsdom avec ECharts local (`apps/web/node_modules` du checkout principal),
  qui ignore les erreurs canvas de jsdom. Une modification doit garder zéro erreur et aucun id en double sur les
  20 combinaisons horizon × pas (10 en solo, 10 en escouade).
