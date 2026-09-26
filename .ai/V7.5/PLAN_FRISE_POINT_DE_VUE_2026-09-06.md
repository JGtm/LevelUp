# PLAN — LA FRISE DU REJEU : TRAIT DE LECTURE UNIQUE, POINT DE VUE SÉLECTIONNABLE, MÉDAILLES

> Ouvert le 2026-09-06. Branche cible : `feat/v75-frise-pov`, worktree DÉDIÉ
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-frise-pov`, depuis `feat/v75`.
> Le worktree principal est partagé — ne pas y exécuter ce plan.
> Préfixe `feat/` volontaire : une branche `wt/*` ne déclenche pas la CI.
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report d'une étape
> exécutable, statut obligatoire par item, vérification sur pièces avant de cocher, zéro fix
> hors périmètre). Skills à invoquer en cours de route : `frontend-patterns` (tout composant),
> `color-tokens` (toute couleur), `foundations-usage` (aucune page neuve ici, mais la frise
> change de forme), `delivery-checklist` (avant le commit final de chaque lot).
>
> Chantier 100 % `apps/web`. Aucun changement Go, aucune migration, aucune requête neuve,
> aucune cuisson d'artefact.

## 1. CE QU'ON LIVRE, ET COMMENT ON SAURA QUE C'EST FAIT

Quatre lots, livrables l'un après l'autre, chacun visible à l'écran :

| Lot | Ce que l'utilisateur voit | Effort |
|---|---|---|
| L1 | Un trait vertical unique traverse toutes les pistes et suit la lecture | rapide |
| L2 | Rien (refactor du point de vue, comportement identique) | moyen |
| L3 | La piste « Toi » devient un menu de joueurs, sections par équipe | moyen |
| L4 | Médailles sur la piste du joueur, présence ombrée hors partie | moyen |

**Dépendances entre lots** : L1 est autonome. L2 est autonome et invisible. **L3 exige L2**
(le menu n'est qu'un `setViewpoint`). L4 se scinde : la partie MÉDAILLES est autonome et peut
se faire dès L1 (elle vaut sur la piste « Toi » actuelle), la partie PRÉSENCE exige L3 pour
avoir un joueur sélectionné dont ombrer l'absence. Ordre retenu quand même : L1 → L2 → L3 → L4,
parce qu'un seul exécuteur avance en séquence et que les médailles touchent le même composant
que le menu.

**Critère de succès global** : sur le match témoin, je choisis un coéquipier dans le menu, les
pistes Dominance et Score inversent leurs couleurs, la carte et les fiches suivent, le curseur
n'a pas bougé d'une image, le son de fin reste celui de MON résultat, et un joueur arrivé en
cours de partie a sa piste ombrée jusqu'à son entrée.

## 2. DÉCISIONS TRANCHÉES PAR L'UTILISATEUR — ne pas rouvrir en cours d'exécution

Toutes datées du 2026-09-06, en conversation.

1. **La sélection change le point de vue, JAMAIS le temps.** Sélectionner un joueur absent à
   l'instant courant ne déplace pas le curseur, ne met pas en pause, ne bascule pas sur un
   remplaçant. Le curseur appartient à l'utilisateur.
2. **Les repères de présence sont cliquables** : c'est l'unique chemin vers l'instant
   d'arrivée ou de départ, et il est explicite. Précédent dans le même composant : les
   vignettes de la piste Médias sont déjà des boutons au milieu de pistes en
   `pointer-events-none`.
2 bis. **Le repère EST le glyphe du fil** (demande utilisateur du 2026-09-06) : la porte
   franchie par une flèche de `PresenceGlyph`
   ([ReplayPresenceLine.tsx](../apps/web/src/features/match-replay/ui/ReplayPresenceLine.tsx)),
   posée **à la frontière entre la zone ombrée et la zone jouée**, du côté ombré. Pour un
   arrivant, c'est donc le bord DROIT de l'ombre de tête ; pour un partant, le bord GAUCHE de
   l'ombre de queue. Un même glyphe, deux sens, déjà encodés par le composant (montant de porte
   à gauche pour entrer, à droite pour sortir). Le glyphe est aussi le BOUTON de la décision 2 :
   un seul objet, pas un repère plus une cible.
3. **Le son et la voix de fin de partie restent ancrés sur le joueur de la page.** Inspecter un
   adversaire ne doit pas jouer « Défaite » sur un match gagné. L'écran de victoire, lui, suit
   le point de vue.
4. **Amis distingués par la FORME, jamais par la couleur.** Un ami prend le losange, comme sur
   la carte (`replayMarkers.ts:203`, décision D5 : « la FORME dit l'identité, la couleur dit le
   camp »). Sur la frise la couleur est déjà prise deux fois (kill = `team-ally`, mort =
   `team-enemy`) et les deux seuls tokens d'équipe reprennent les couleurs d'outline choisies
   par l'utilisateur en jeu. Si la forme ne suffit pas à l'usage : la hauteur, jamais la
   couleur.
5. **Piste « Alliés » → « Coéquipiers »** : elle montre les coéquipiers du joueur sélectionné.
   Les amis restent une affaire d'identité (le losange), plus une piste. C'est un CHANGEMENT DE
   COMPORTEMENT PAR DÉFAUT assumé : aujourd'hui la piste montre les joueurs marqués `friend`,
   y compris ceux de l'équipe ADVERSE (`playerMarks.ts:9`).
6. **Colonne des libellés : 76 px → 100 px**, gamertags tronqués si besoin, nom entier dans la
   liste ouverte et dans `title`.
7. **Les bots sont sélectionnables.** Un bot sans ligne de scoreboard n'a pas d'équipe et va
   dans la section « Sans équipe » de `groupByTeam` ; sa sélection change la piste des marques
   et ne fait rien basculer d'autre. **Aucun camp deviné.**
7 bis. **Un joueur SANS ligne de scoreboard est listé mais DÉSACTIVÉ** dans le menu (décision
   superviseur du 2026-09-07, prise en lançant L3, réversible). Vérifié sur pièces : sans ligne
   de scoreboard il n'a ni camp ni xuid de base, `collectKillEvents` jette les kills d'un acteur
   absent du scoreboard (sa piste serait VIDE), et `resolveViewpoint` retomberait en silence sur
   le joueur de la page. La décision 7 disait « sélectionnable, change la piste des marques » :
   cette piste n'aurait rien à montrer. Une option qui ne fait rien doit le dire (`disabled` +
   infobulle « Aucune donnée de match pour ce joueur ») plutôt que faire semblant.
8. **Défaut = le joueur de la page**, toujours, à chaque montage. La sélection n'est pas
   persistée.
9. **Médailles : pas de glyphe supplémentaire quand la médaille est attachée à un kill.** Elle
   décore la marque de kill existante et s'écrit dans l'infobulle. Seules les médailles
   ORPHELINES (objectif) prennent une marque à elles.
10. **Le changement d'équipe en cours de match n'est pas traité** : le camp est une valeur par
    match au scoreboard, et le matchmaking Halo Infinite ne fait pas basculer d'équipe.

Conséquence concrète de la décision 5, à connaître : un ami dans l'équipe ADVERSE, dont les
kills étaient sur la piste « Alliés », **disparaît de la frise**. Il reste losange sur la carte
et nommé dans le fil. Accepté par la décision, écrit ici pour qu'aucune relecture n'y voie une
régression.

### Décisions ouvertes par la revue du 2026-09-06 — TRANCHÉES par l'utilisateur le même jour

Les quatre recommandations ci-dessous sont ACCEPTÉES telles quelles (« ok 11-14 »). Elles ont
même valeur que les décisions 1-10. S'y ajoute une **décision 15**, la plus contraignante :

16. **Les marques de kill sont CENTRÉES sur l'instant qu'elles désignent** (décision
    utilisateur du 2026-09-06, sur la découverte L1) : une marque de 3 px ancrée par son bord
    gauche est ~1,5 px à droite de l'instant ; la pastille du curseur, elle, est centrée. Marques,
    trait et pastille partagent désormais la même ancre : le CENTRE. Intégré au lot L1 (item
    ajouté), pas un lot à part — c'est une ligne, et elle conditionne la lecture du trait.
15. **RIEN NE CHANGE HORS DE LA PAGE DE REJEU.** Les surfaces de la page match (cinq charts
    par `resolveXuidMeta`, onglet Chronologie avec `MatchPadControlSection` et
    `MatchEquipmentUsageSection`) sont bit à bit identiques avant et après le chantier. Ce n'est
    pas une intention, c'est un GATE : l'étape L2a photographie l'existant AVANT toute
    modification, et L2b n'est close que si la photographie est intacte (§4, L2a/L2b).

11. **Les sons d'objectif EN COURS de match** (ticks de zone allié / adverse, voix d'objectif :
    `objectiveSound`, `zoneSound`) lisent le camp par `matchSides.allyTeamFromScoreboard`. Router
    `matchSides` par le point de vue les fait suivre. La décision 3 n'ancre que la FIN de partie.
    Recommandation : **ils suivent le point de vue** — ce sont des rendus de la perspective
    courante, comme les couleurs ; seul le VERDICT est ancré. _Statut : TRANCHÉ 2026-09-06, recommandation retenue._
12. **L'export vidéo** peint son propre panneau de victoire (`exportOverlayPanels.ts:113`,
    `useReplayExport.ts:221`) via `readVictory`. Recommandation : **l'export rend ce que l'écran
    montre** (point de vue courant, WYSIWYG), y compris son panneau de victoire — la personne
    qui exporte a choisi ce qu'elle regarde. _Statut : TRANCHÉ 2026-09-06, recommandation retenue._
13. **Le disque cerclé de la carte** (`shapeOfMark`, D5 : « joueur de la page ») suit-il le point
    de vue ? Si `playerMarks` est routé, oui mécaniquement. Recommandation : **oui**, et l'en-tête
    de `replayMarkers.ts` est mis à jour dans le même commit (« le disque cerclé = le point de
    vue, qui vaut le joueur de la page par défaut ») — sinon doc inversée. _Statut : TRANCHÉ 2026-09-06, recommandation retenue._
14. **Forme de la décoration de médaille** sur une marque de kill. Le plan disait « anneau ou
    marque pleine contre creuse » : c'est un choix laissé à l'exécuteur, donc une dérive
    possible. Recommandation : **anneau** (contour de 1 px de l'encre courante autour de la
    marque, la marque garde sa couleur) — le vocabulaire `ring` existe déjà sur la carte pour
    « celui qu'on regarde », et la marque pleine/creuse se lirait mal à 3 px de large.
    _Statut : TRANCHÉ 2026-09-06, recommandation retenue._

## 3. CONSTATS SUR PIÈCES — vérifiés le 2026-09-06, à re-vérifier avant de coder (règle 4)

Ce qui suit a été lu dans le code, pas déduit. Les numéros de ligne bougeront.

### 3.1 Le trait de lecture

- `--played` est écrit **sur le parent du champ** — `(el.parentElement ?? el)` —
  [useReplayPlayback.ts:215](../apps/web/src/features/match-replay/hooks/useReplayPlayback.ts#L215).
  Ce parent est le `div.relative.mt-[3px]` de la deuxième colonne de la grille : les pistes,
  qui vivent dans les rangées du dessus, **ne le voient pas**. Les propriétés perso héritent :
  le poser un cran plus haut suffit, le champ continue de le lire.
- **Deux géométries différentes, et c'est le piège.** `--played` est un pourcentage BRUT de la
  largeur ; les marques, elles, sont posées en géométrie de curseur — `calc(8px + (100% - 16px)
  * r)` ([replayTimelineTracksLogic.ts:32](../apps/web/src/features/match-replay/model/replayTimelineTracksLogic.ts#L32),
  `THUMB_PX = 16`). Un trait à `left: var(--played)` dériverait jusqu'à 8 px des marques qu'il
  désigne. La bulle de temps porte déjà ce décalage aujourd'hui — invisible faute de repère,
  le trait le rendrait visible.
- `writeCursor` est **le seul endroit** qui déplace le curseur, et son en-tête dit pourquoi :
  boucle, sauts, rembobinage, glissé et pose initiale l'appellent tous. Une variable ajoutée là
  est écrite par tous les chemins, sans exception.

### 3.2 Le point de vue

- **Dans tout `match-replay`, `resolveXuidMeta` n'est appelé QU'UNE FOIS** :
  [replayModel.ts:119](../apps/web/src/features/match-replay/model/replayModel.ts#L119),
  `resolveXuidMeta(scoreboard, meXUIDOf(scoreboard))`. Les cinq autres appels sont des charts
  de `match-view`, hors périmètre : ils gardent `meXUID`.
- `resolveXuidMeta` porte un `r.is_me ||` dans le calcul de `ally`
  ([xuidMeta.ts:35](../apps/web/src/features/match-view/xuidMeta.ts#L35)). **Ce court-circuit
  devient FAUX** dès que le point de vue est ailleurs : la ligne du joueur de la page resterait
  « alliée » vue depuis un adversaire. Il doit disparaître au profit d'une comparaison de camp
  unique — **pour le rejeu**. Nuance de revue : côté `match-view`, `meXUID` vient de
  `MatchViewPage.tsx:227` (`meRow?.xuid ?? null`, donc la ligne `is_me` elle-même) ; le
  court-circuit n'y sert que le cas `meXUID = null`, où il rend la ligne « moi » alliée quand
  même. Le retirer sans précaution changerait ce cas-là pour cinq charts hors périmètre. Donc :
  **ne pas changer la sémantique de la fonction partagée** : un paramètre de point de vue
  optionnel qui, quand il est fourni, désactive le repli `is_me`. (L'alternative « helper propre
  au rejeu à côté » est morte — §3.7 : le garde `xuidMeta.guard` interdit la cascade de camp
  hors de `xuidMeta.ts`.) Les cinq appels de `match-view` restent bit à bit identiques.
- **Liste EXHAUSTIVE des lecteurs de `is_me`** (grep du 2026-09-06 sur `features/match-replay/`
  + `lib/replay/`, commentaires exclus) — cinq sites de code, et rien d'autre :
  `lib/replay/playerMarks.ts:37`, `model/matchSides.ts:40` (dans `allyTeamFromScoreboard` — le nom `mySideID` employé jusqu'à la revue L2a n'existe pas, correction du 2026-09-06),
  `model/victoryLogic.ts:129` (`readVictory`), `MatchPadControlSection.tsx:103`,
  `MatchEquipmentUsageSection.tsx:108`. Noter que `playerMarks.ts` vit dans `lib/replay/`,
  **pas** dans `features/` : le garde-rail de L2 doit couvrir les deux dossiers.
- **CORRECTION DE REVUE (2026-09-06) — deux de ces cinq sites NE SONT PAS SUR LA PAGE DE REJEU.**
  `MatchPadControlSection` et `MatchEquipmentUsageSection` vivent dans le dossier
  `match-replay/` mais sont montés par `match-view/MatchViewTabChronology.tsx:142-154` : l'onglet
  Chronologie de la page MATCH, où aucun menu de point de vue n'existe. Ils restent sur `is_me`,
  et le garde-rail les met en **allowlist nommée et datée** avec cette raison. Le point de vue
  du rejeu ne route donc que TROIS lecteurs : `playerMarks`, `matchSides`, `readVictory`.
- **`matchSides` est un ÉVENTAIL, pas un site.** Il exporte `allyTeamFromScoreboard` (le seul
  qui dépende du point de vue) et `teamOfXuidFromScoreboard` (le camp propre de chaque xuid —
  indépendant du point de vue, il n'a PAS été routé, constat L2b), consommés par
  `useReplayBombBlast`, `useReplayFlagCarries`, `useZoneStates`, `sound/objectiveSound` (ticks
  de zone allié / adverse, `zoneSound.ts:163`) **et `sound/useReplaySound.ts`** — cinquième
  consommateur, absent de la liste initiale, trouvé et routé en L2b. Le router par le point de vue fait suivre les quatre d'un coup
  — y compris **les sons d'objectif en cours de match**, que la décision 3 ne nomme pas (elle
  ne parle que de la FIN de partie). Voir décision 11.
- **`readVictory` a QUATRE consommateurs, pas deux** : `ReplayVictoryOverlay.tsx:110`,
  `endMatchSound.ts:131`, et l'EXPORT vidéo — `export/exportOverlayPanels.ts:113` et
  `export/useReplayExport.ts:221`, qui peignent leur propre panneau de victoire. Voir
  décision 12.
- **CONFLIT À TRAITER EXPLICITEMENT — l'écran de victoire et le son partagent leur lecteur.**
  `endMatchSoundSpec` n'a pas de `is_me` à lui : il appelle `readVictory(scoreboard, outcomeCode)`
  ([endMatchSound.ts:127](../apps/web/src/features/match-replay/sound/endMatchSound.ts#L127)),
  et son en-tête revendique de ne jamais refaire un second décodage. Or la décision 3 veut que
  l'écran SUIVE le point de vue et que le son NON. Ils ne peuvent donc plus partager un
  `readVictory` qui devine son sujet : celui-ci doit **recevoir le joueur concerné en
  paramètre**, l'écran passant le point de vue et le son passant le joueur de la page. Explicite
  aux deux appels, deviné à aucun.
- **`KillEvent.ally` est cuit dans le modèle** depuis `xuidMeta`
  (`match-view/_momentum.ts`, `collectKillEvents` : `const meta = xuidMeta.get(e.actor_xuid)`,
  puis `ally: meta.ally`). Changer le point de vue invalide donc `identity` → `kills` → `feed`.
  `buildReplayModel` est mémoïsé en UN bloc sur `(doc, matchView, settings)`
  ([useReplayModel.ts:26](../apps/web/src/features/match-replay/model/useReplayModel.ts#L26)) :
  ajouter le point de vue aux dépendances refait TOUTE la jointure à chaque bascule
  (`buildPlayers` sur 99 traces, alignement statistique du fil, présence). **C'est mesurable —
  E2.3 le mesure avant de décider s'il faut découper la mémo. Pas de refactor préventif.**
- Même fonction, ligne suivante : `if (!meta) continue` — un kill dont l'acteur est absent du
  scoreboard est JETÉ. Comportement actuel, hors périmètre, à ne pas « corriger ».

### 3.3 Le menu

- `groupByTeam` existe déjà et rend exactement les sections voulues, **y compris un groupe
  `side: null`** pour qui n'a pas de ligne de scoreboard
  ([rosterLogic.ts](../apps/web/src/lib/replay/rosterLogic.ts), « un joueur sans ligne de
  scoreboard n'a pas d'équipe et se retrouve dans un groupe à part »).
- **PIÈGE DES BOTS, à traiter explicitement.** Un bot est clé `bot:<nom>` côté FILM
  (`botKey`, schéma 36) et `bid(N.0)` côté BASE ; la jointure se fait par nom nu. Or les
  marques de la frise sont appariées sur les xuid du FIL, qui viennent de la base
  (`reduceFeed` compare `k.xuid` aux clés de `marks`). **Un menu qui rendrait la clé film pour
  un bot donnerait une piste vide en silence.** La valeur sélectionnée doit être
  `p.board?.xuid` quand elle existe, et retomber sur `p.xuid` sinon.
- Le libellé vient de `playerName(p)` (base d'abord, film ensuite) — jamais de la clé.

### 3.4 Présence et médailles

- [presenceFeed.ts](../apps/web/src/features/match-replay/model/presenceFeed.ts) fait déjà tout
  le travail : API `first_joined_time` / `last_leave_time` (source précise et AFFIRMATIVE — un
  drapeau à `false` fait taire le joueur), repli film sur les bornes de vie avec ses marges
  (`PRESENCE_ARRIVE_MS = 10 s`, `PRESENCE_DEPART_MS = 20 s`), recalage déjà fait sur l'axe du
  rejeu, lignes déjà fusionnées au fil par `mergeFeedWithPresence`. **Rien à collecter, rien à
  ajouter côté Go.**
- `PresenceEvent` porte `source: 'api' | 'film'`. Il faut le PROPAGER jusqu'à l'ombrage : bord
  franc pour l'API, bord dégradé et réserve dans l'infobulle pour le film — la marge y est de
  10 à 20 s et le film ne distingue pas un départ d'une élimination définitive.
- **La « petite marge raisonnable » demandée existe déjà, mesurée — n'en inventer aucune.**
  Sur le chemin API il n'y en a pas besoin : un drapeau `joined_in_progress` à `false` n'émet
  RIEN, donc aucun risque de repère d'arrivée à 0:02 pour quelqu'un qui était là au coup
  d'envoi. Sur le repli film, `PRESENCE_ARRIVE_MS = 10 s` et `PRESENCE_DEPART_MS = 20 s`, plus
  de 2× la réapparition médiane mesurée (8,0 s). Aucun nouveau seuil, aucun nombre magique.
- **Le glyphe de présence est aujourd'hui PRIVÉ** : `PresenceGlyph` est une fonction locale non
  exportée de `ReplayPresenceLine.tsx`. Il faut l'EXTRAIRE dans son propre module et le faire
  consommer par les deux lecteurs — jamais le recopier (règle R6 ; précédent d'extraction :
  `useTeamCascades.ts`, `useSlotIdentity.ts`). Il est vectoriel, à l'encre courante, teinté par
  l'équipe de l'acteur quand elle est connue — et un bot sans camp joint garde l'encre du
  repli, jamais un camp deviné : cette règle vaut telle quelle sur la frise.
- Les médailles sont collectées **sans filtre d'identité** (`collectMedalEvents(...)` ne prend
  pas `xuidMeta`) : elles existent donc pour TOUS les joueurs, bots compris. Elles viennent du
  chunk highlight du film via le sync (`internal/sync/collect.go`), pas d'une API par joueur.
- Les médailles sont **déjà rattachées aux kills** à ±`MEDAL_ATTACH_MS`
  ([killFeedLogic.ts:507](../apps/web/src/features/match-replay/model/killFeedLogic.ts#L507)) :
  la majorité tombe sur une marque de kill déjà dessinée. C'est ce qui rend la décision 9
  applicable sans nouveau calcul.
- Sur les matchs antérieurs au backfill `backfill-medailles-feed` (tâche de RELEASE, au carnet
  Notion), le nom et l'image peuvent manquer. Dégradation : pas de décoration, jamais d'icône
  cassée.

### 3.5 Contraintes de forme

- Couleurs : `tokenCssVar` uniquement, aucun hex, aucune classe Tailwind de couleur dans
  `features/`. Fond de piste et ombrage = couleurs STRUCTURELLES (`bg-muted`, `border-border`),
  exception explicitement tolérée par le skill `color-tokens`, à commenter.
- i18n : toute string neuve entre dans `i18nContract.ts` + les deux tables FR et EN de
  `i18n.ts`. La parité est tenue par le typage `Record<ReplayLocale, ReplayText>` — un champ
  manquant ne compile pas.
- Seuils du dépôt (`wc -l` du 2026-09-06, corrigé en revue) : `ReplayTimelineTracks.tsx` fait
  **421** lignes et `replayTimelineTracksLogic.ts` **445**, `useReplayTimeline.ts` 253. Les deux
  premiers sont à moins de 80 lignes du plafond de 500 : L3 (menu) et L4 (ombrage, glyphes,
  médailles) les font franchir À COUP SÛR. L'extraction n'est donc pas une éventualité mais une
  étape PLANIFIÉE : `ui/ReplayPlayerTrack.tsx` (le menu + la piste du joueur + ses glyphes) et
  `model/presenceTrackLogic.ts` (intervalles d'ombrage, purs). `max-lines` est un ratchet : on
  extrait, on ne relève pas le seuil.
- `ui/rosterHeight.guard.test.ts` ne concerne PAS la frise (il refuse un `max-h-[%]` sur le
  bloc des fiches) : aucune interaction avec la hauteur de la piste. Constat de revue, item
  retiré de L4.

### 3.6 Ce que la grille `plan-review` demande et qui ne s'applique PAS ici — dit une fois

Le silence sur ces points est délibéré, pas un oubli.

- **Couches Go** (`analysis` / `domain` / `service` / `port` / handlers), **adapters
  canoniques**, **logging `slog`**, **repos DuckDB** : aucun. Le chantier ne touche pas
  `apps/go-api`. Toute donnée employée est déjà servie par la Match View et l'artefact de rejeu.
- **Nouvelle capability : aucune.** La page de rejeu est déjà gardée en amont (capability
  title-level `replay`), la piste Médias par `useCapability('media')`. Médailles et présence
  n'en demandent pas : une médaille sans nom ne se décore pas, un joueur sans drapeau de
  participation ne s'ombre pas. C'est la dégradation, et elle est locale — pas de 503, pas de
  branche par titre. **Aucune comparaison de slug nulle part** (ratchet
  `no_slug_comparison_test.go`).
- **Routes, query keys, `generate-types`** : aucun. Pas de route neuve, pas de requête neuve,
  pas de changement d'`openapi.yaml`.
- **Migration, cuisson d'artefact, backfill** : aucun.

### 3.7 Ce qui est testé AUJOURD'HUI hors page — inventaire du 2026-09-06 (décision 15)

La crainte de l'utilisateur est fondée : les surfaces hors page sont couvertes de façon inégale,
et **rien ne fixe les SORTIES de `resolveXuidMeta`**.

| Surface hors page | Test existant | Ce qu'il couvre |
|---|---|---|
| `xuidMeta.ts` | `xuidMeta.guard.test.ts` SEULEMENT | un grep : la cascade `team_side === allyTeam` n'existe qu'ici. **Aucun test de valeurs.** L'allowlist du garde réserve déjà le nom `xuidMeta.test.ts` — il n'existe pas encore. |
| `MatchScoreCurveChart`, `MatchScoreEventsChart`, `MatchTugOfWarChart` | `.test.tsx` présents | rendu avec fixtures `is_me` — bons témoins indirects |
| `MatchCadenceChart`, `MatchKDCumulChart` | **AUCUN** | rien |
| `MatchViewTabChronology` | `.t0.test.tsx` seulement | le recalage t0, pas les sections |
| `MatchPadControlSection`, `MatchEquipmentUsageSection` | `.test.tsx` présents | leur rendu, `is_me` compris |
| `matchSides.ts` (éventail : bombe, drapeaux, zones, sons) | **AUCUN test direct** | couvert seulement par ses consommateurs |
| `victoryLogic.ts` | `victoryLogic.test.ts`, 14 cas | dont « sans ligne `is_me` : null » et « `is_me` sans camp : null » — ce sont les ancres de l'ancien comportement de `readVictory` |

Pas de snapshot dans le dépôt (assertions explicites partout) : la « photographie » se fait
donc par des **tests de caractérisation** — des tests qui décrivent la sortie ACTUELLE, écrits
et passés au vert AVANT le changement, puis relancés INCHANGÉS après.

**Conséquence sur le mécanisme du point de vue** : le garde `xuidMeta.guard` interdit la
cascade de camp partout sauf dans `xuidMeta.ts`. L'option « helper propre au rejeu à côté »
(§3.2) est donc MORTE — elle recréerait la cascade ailleurs et ferait rougir le garde. Reste la
seule voie : **un troisième paramètre optionnel** de `resolveXuidMeta`, qui n'existe que pour le
rejeu et dont l'absence rend la fonction strictement identique à aujourd'hui.

## 4. ÉTAPES

Règle d'ordre : l'étape N+1 ne commence pas avant que N soit close. « Close » = tous ses items
statués (`[x]` fait / `[~]` couvert ailleurs avec référence / `[!]` non traité avec
justification écrite) ET son gate passé, sortie réelle collée au journal.

Gate commun à toute étape qui touche `apps/web` (à lancer HORS sandbox — vitest y échoue) :

```bash
make check-types
make test-web
```

En cas d'échec de `check-types` sans cause visible : purger `node_modules/.tmp`.

---

### E0 — Cadrage (aucune écriture)

- [x] Créer le worktree dédié et la branche : `git worktree add ../LevelUp-wt-frise-pov -b feat/v75-frise-pov feat/v75`
      — fait par le superviseur le 2026-09-06 depuis a059caefc ; `apps/web/node_modules` du
      worktree est une JONCTION vers celui du principal (à retirer en PowerShell AVANT tout
      `git worktree remove`, jamais par le remove lui-même).
- [x] Rouvrir les six fichiers cités en §3 et confirmer que les constats tiennent (numéros de
      ligne, signatures). Noter tout écart ICI avant de coder.
      **Écarts relevés par l'exécuteur L1 (2026-09-06), tous mineurs** : `THUMB_PX` ligne 30 et
      `trackLeft` 33-36 (plan : 32) ; la bulle portait un repli `var(--played, 0%)` (plan :
      `var(--played)`) — le décalage de 8 px est réel ; `writeCursor` a 5 appelants, tous dans
      le hook ; `useReplayPlayback.ts` fait 337 lignes. Point de méthode : `max-lines` du dépôt
      compte les lignes de CODE (`skipComments`, `skipBlankLines`) — la contrainte brute ≤ 500
      a été tenue quand même.
- [x] Choisir le match témoin et l'écrire dans ce plan : il DOIT avoir (a) au moins un joueur
      arrivé ou parti en cours de partie, (b) un mode à objectif pour que la piste Score
      existe, (c) au moins un bot si possible. Sans (a), le lot 4 n'est pas vérifiable.
      **FAIT par le superviseur le 2026-09-06** (requête ci-dessous via `cmd/diag_q`, serveur
      arrêté, 106 artefacts croisés avec `match_participants` × `match_registry`) — deux témoins,
      tous deux avec JGtm (joueur de la page), un bot, un artefact cuit, un mode CTF :
      - **PRINCIPAL `4ecdf3e7-ec9d-4936-927e-973f5b316a59`** — CTF Arena Neutral Flag, Quick
        Play, 10 participants dont 2 arrivés et 2 partis en cours, 1 bot. Le cas riche : deux
        remplacements.
      - **SECONDAIRE `bf5ced1b-4efc-47d9-b0ec-050116df71f4`** — CTF Arena, Quick Play, 9
        participants dont 1 arrivé et 1 parti, 1 bot. Le cas simple : un seul remplacement.
      Fait utile pour L3 et L4 : sur ces matchs le BOT EST le remplaçant (un partant, un bot
      qui comble, parfois un humain qui reprend le siège) — sélectionner le bot dans le menu
      exerce donc à la fois le piège des clés (§3.3) et l'ombrage de présence, sur le même témoin.
      URL de rejeu : `/t/halo_infinite/players/JGtm/matches/<match_id>/replay`.
      Requête de départ (MCP duckdb, ATTACH `READ_ONLY` — la shared est tenue RW par le
      serveur, jamais l'ouvrir en écriture) :
      ```sql
      ATTACH 'data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS sh (READ_ONLY);
      SELECT match_id, COUNT(*) FILTER (WHERE joined_in_progress) AS arrives,
             COUNT(*) FILTER (WHERE left_in_progress) AS partis
      FROM sh.match_participants GROUP BY 1 HAVING arrives > 0 OR partis > 0
      ORDER BY arrives + partis DESC LIMIT 20;
      ```
      puis ne retenir qu'un match dont l'artefact de rejeu EXISTE — le fichier porte l'ID
      COURT du film, les 8 premiers caractères du `match_id` avant le premier tiret
      (`data/cache/replays/halo_infinite/000d5950.json`, cf. `title/registry.go:745`) — sans
      cuire quoi que ce soit.
- [~] Relever l'état de départ : capture d'écran de la frise dépliée, à conserver pour la
      comparaison finale. **Couvert autrement** : le principal (base a059caefc, sans le lot)
      sert l'AVANT sur `:5173` et le worktree l'APRÈS sur `:5174`, même API `:8000` — la
      comparaison se fait en direct, deux onglets côte à côte, sans capture à archiver.

**Gate E0** : le match témoin est nommé, les trois propriétés vérifiées à l'écran.
_Statut 2026-09-06 : témoins nommés sur données (§ ci-dessus) ; vérification à l'écran des trois
propriétés faite par l'utilisateur en même temps que le gate L1._

---

### L1 — Le trait de lecture unique

- [ ] Dans `writeCursor`, écrire **une variable sans unité** `--played-r` (0..1, bornée comme
      l'est déjà `--played`) en plus de l'existante, ou remplacer `--played` par elle et
      dériver le pourcentage en CSS. Une seule écriture, au même endroit — l'en-tête du hook
      interdit tout second chemin.
- [ ] La poser sur le conteneur RACINE de `ReplayTimelineTracks` (le `div.relative` externe),
      pas sur le parent du champ. Vérifier que le dégradé du curseur est inchangé au pixel.
- [ ] Réaligner la bulle de temps sur la géométrie de curseur (elle porte le décalage
      aujourd'hui) : `left: clamp(1.4rem, calc(8px + (100% - 16px) * var(--played-r)), calc(100% - 1.4rem))`.
- [ ] Ajouter le trait : `pointer-events-none`, positionné en absolu sur la colonne des PISTES
      (pas sur toute la grille — la colonne des libellés fait 100 px + `gap-x-3`), de la
      première piste jusqu'au-dessus du curseur, jamais à travers la pastille.
- [ ] Le trait n'existe que quand `tracksExpanded` est vrai.
- [ ] Couleur : token sémantique ou couleur structurelle commentée — pas de hex.
- [ ] Test : garde-rail sur la géométrie partagée — un test qui échoue si le trait et
      `trackLeft` cessent d'employer la même formule (c'est LE défaut qu'on prévient).
      Mécanisme (précisé en revue, un « test de géométrie » sans mécanisme est un vœu) : la
      formule vit UNE fois, dans `replayTimelineTracksLogic.ts`, exportée sous deux formes —
      `trackLeft(ratio)` (valeur) et `trackLeftVar('--played-r')` (expression CSS sur une
      variable) ; le garde-rail lit `ReplayTimelineTracks.tsx` et refuse toute occurrence
      littérale de `THUMB_PX`, `16px`, `8px` ou `100% -` hors appel de ces deux helpers. Patron :
      les `*.guard.test.ts` de `ui/` et `layers/`.
- [ ] **Centrer les marques de kill** (décision 16, ajoutée le 2026-09-06 après la découverte
      L1) : la marque de `MarkTrack` se pose par son centre sur `trackLeft(ratio)`, pas par son
      bord gauche. Le trait garde son ancre (le point) : marques, trait et pastille coïncident
      alors exactement. Mettre à jour l'en-tête de `ReplayPlayhead.tsx` (§ « il partage aussi
      leur ancre ») qui décrit l'ancien bord d'attaque. Test : la marque porte la translation.
      La piste Médias ne bouge pas (photo déjà centrée par `marginLeft: -15`, un clip est un
      intervalle dont le bord gauche EST l'instant).
- [ ] Vérification à l'écran : mettre en lecture, confirmer que le trait passe exactement sur
      une marque de kill à l'instant où le fil affiche ce kill.

**Gate L1** : `make check-types` + `make test-web` verts ; capture du trait posé sur une marque,
et deuxième capture au ralenti confirmant zéro dérive aux deux extrémités (0 % et 100 %).

_Statut 2026-09-06 : tous les items de code `[x]` (exécuteur Opus, relus sur pièces par le
superviseur ; gates rejoués : vitest ciblé 79 fichiers / 1162 tests, `tsc -b --force` exit 0 ;
suite complète 614 fichiers / 6415 tests et lint 0 erreur côté exécuteur). Commit sur
`feat/v75-frise-pov`. La vérification à l'écran par l'utilisateur (AVANT `:5173` / APRÈS `:5174`,
témoin 4ecdf3e7) reste OUVERTE : remarque produit reçue et traitée (centrage des marques,
décision 16), verdict sur les 8 points attendu. L2a lancé en parallèle : il ne touche aucun
fichier de production, il ne dépend pas de ce verdict._

---

### L2a — Photographie de l'existant hors page (AVANT toute modification — décision 15)

Un commit à part, `test(frise-pov): caractérisation avant point de vue`, qui ne touche AUCUN
fichier de production. Il décrit ce que le code fait AUJOURD'HUI ; s'il ne passe pas au vert sur
le code actuel, c'est le test qui est faux, pas le code.

- [ ] `features/match-view/xuidMeta.test.ts` (le nom que l'allowlist du garde réserve déjà) :
      sur UNE fixture de scoreboard à deux camps + un bot + une ligne sans `team_side`, fixer
      la `Map` COMPLÈTE rendue par `resolveXuidMeta(sb, meXUID)` — égalité profonde, pas un
      champ — dans les cinq cas : (a) `meXUID` = la ligne `is_me` ; (b) `meXUID = null` avec
      une ligne `is_me` présente (**le cas du court-circuit**, c'est lui que l'on protège) ;
      (c) `meXUID` = une autre ligne que `is_me` ; (d) aucune ligne `is_me` ; (e) `meXUID`
      absent du scoreboard. Plus `meXUIDOf` sur (a) et (d).
- [ ] `features/match-replay/model/matchSides.test.ts` (n'existe pas) : fixer `mySideID`,
      `allyTeamFromScoreboard`, `teamOfXuidFromScoreboard` sur la même fixture, cas (a) et (d).
- [ ] `victoryLogic.test.ts` : relire les 14 cas ; ils SONT la caractérisation de `readVictory`
      à deux arguments. Ne rien y ajouter sauf trou avéré — et alors l'écrire ici.
- [ ] Relever le compte de tests passés, à l'unité, de la commande ci-dessous, et l'écrire
      dans ce plan (c'est le « avant » chiffré) :
      ```bash
      cd apps/web && npx vitest run src/features/match-view src/features/match-replay/MatchPadControlSection.test.tsx src/features/match-replay/MatchEquipmentUsageSection.test.tsx src/features/match-replay/model/victoryLogic.test.ts
      ```
- [ ] Captures d'écran du match témoin, page MATCH, onglet Chronologie (les cinq charts et les
      deux sections) — le « avant » visuel, à conserver avec la capture E0 du rejeu.

**Gate L2a** : le commit ne contient que des fichiers `*.test.ts(x)` ; `make test-web` vert ;
compte de tests et captures écrits dans ce plan.

_Statut 2026-09-06 : CLOS. `xuidMeta.test.ts` (10 cas, Map entière par égalité profonde) et
`matchSides.test.ts` (13 cas) écrits par Opus, verts du premier coup sur le code actuel, relus
sur pièces par le superviseur ; `victoryLogic.test.ts` non touché (les cas 1-2 fixent déjà
l'objet complet sur victoire et défaite). **COMPTE « AVANT » DE RÉFÉRENCE pour le gate L2b :
`Test Files 46 passed (46)` / `Tests 467 passed (467)`** (commande du point 4, rejouée par le
superviseur : identique). Typecheck après purge exit 0, lint 27 warnings = baseline. Les
captures de l'onglet Chronologie sont remplacées par la comparaison en direct `:5173` (principal
= AVANT) / `:5174` (worktree = APRÈS) au gate L2b. Commit de tests seuls sur la branche._

---

### L2b — Le point de vue explicite (invisible pour l'utilisateur)

Aucun changement de comportement attendu — ni sur la page de rejeu, ni AILLEURS (décision 15).
Un écart visible à ce lot est un bug du lot.

- [ ] Ajouter un TROISIÈME paramètre optionnel de point de vue à `resolveXuidMeta` (pas de
      helper à côté — §3.7). **Fourni** : `ally` se calcule par comparaison de `team_side` avec
      le camp du point de vue, sans le repli `r.is_me ||`. **Absent** : le code est celui
      d'aujourd'hui, à la ligne près — c'est ce que la caractérisation L2a vérifie. Les cinq
      appels de `match-view` ne sont pas touchés (le diff les exclut, gate L2b n°1).
- [ ] Introduire le point de vue dans la feature rejeu, **défaut `meXUIDOf(scoreboard)`**, et
      le faire descendre jusqu'à `buildReplayModel`. Un seul foyer — pas de second `is_me` lu
      en aval.
- [ ] Router les TROIS lecteurs de `is_me` de la page de rejeu par ce foyer : `playerMarks` (la
      marque `me` suit le point de vue — décision 13), `matchSides` (ses deux exports
      `allyTeamFromScoreboard` — le seul lecteur de `is_me` du fichier, ligne 40 — et
      `teamOfXuidFromScoreboard`, donc bombe, drapeaux, zones et sons d'objectif — décision 11 ;
      il n'existe PAS de `mySideID`, correction L2a), `readVictory` (qui reçoit désormais son
      sujet en paramètre).
      **Fait de L2a à connaître** : à deux arguments, `resolveXuidMeta` vu depuis un adversaire
      rend DEUX camps alliés à la fois (la ligne `is_me` reste alliée par le court-circuit). La
      sortie à trois arguments doit rendre UN SEUL camp allié — test à trois arguments à ajouter
      dans `xuidMeta.test.ts` (ajout seulement).
      **PAS `MatchPadControlSection` ni `MatchEquipmentUsageSection`** : montés sur la page
      match, ils restent sur `is_me` (§3.2, correction de revue).
- [ ] `readVictory(scoreboard, outcomeCode, subject)` : l'écran de victoire passe le point de
      vue, `endMatchSound` passe le joueur de la page, l'export passe ce que la décision 12
      retient. Quatre appels, quatre sujets explicites, zéro deviné.
- [ ] **`endMatchSound` NE PASSE PAS par le point de vue** (décision 3). Le laisser lire le
      joueur de la page, et l'écrire en commentaire à l'endroit exact — sinon la prochaine
      relecture « corrigera » l'incohérence apparente.
- [ ] **Garde-rail** (règle R6 : ≥3 copies → helper + test qui interdit l'ancien littéral) : un
      test grep refusant `is_me` dans `features/match-replay/` hors du module de point de vue,
      avec une allowlist explicite et datée pour `endMatchSound`. Le garde-rail porte sur
      `features/match-replay/` SEULEMENT : `match-view/xuidMeta.ts` et ses charts gardent leur
      `is_me`, ils sont hors périmètre.
- [ ] **E2.3 — MESURER avant d'optimiser.** Instrumenter une bascule de point de vue sur le
      match témoin et relever le temps de reconstruction du modèle. Coller le chiffre ici.
      - Sous 50 ms : ne rien découper, écrire la mesure en commentaire au-dessus de la mémo.
      - Au-delà : découper la mémo — la partie chère (alignement du fil, `buildPlayers`,
        présence) ne dépend PAS du point de vue ; seule la décoration `ally` en dépend.
- [ ] Tests unitaires : `resolveXuidMeta` vu depuis un adversaire (la ligne `is_me` doit
      ressortir NON alliée), vu depuis un joueur sans camp, vu depuis un xuid absent.

**Gate L2b — quatre verrous, tous mécaniques, aucun subjectif :**

1. **Le diff hors page est VIDE, par construction.** Commande, sortie attendue : rien.
   ```bash
   git diff --name-only feat/v75...HEAD -- apps/web/src/features/match-view/ | grep -v -E '^apps/web/src/features/match-view/xuidMeta\.(ts|test\.ts)$'
   git diff --name-only feat/v75...HEAD -- apps/web/src/features/match-replay/MatchPadControlSection.tsx apps/web/src/features/match-replay/MatchEquipmentUsageSection.tsx
   ```
   Seuls `xuidMeta.ts` et son test de caractérisation ont le droit de bouger côté match-view ;
   et le test de caractérisation ne bouge QUE pour ajouter des cas à trois arguments — le
   `git diff` de ce fichier ne doit contenir aucune ligne `-` (rien de supprimé, rien de
   modifié : uniquement des ajouts). Vérifier : `git diff feat/v75...HEAD -- .../xuidMeta.test.ts | grep '^-[^-]'` rend vide.
2. **La photographie L2a est intacte** : `matchSides.test.ts` et `victoryLogic.test.ts` sans
   aucune ligne supprimée ni modifiée (même vérification `grep '^-[^-]'`), et verts.
3. **Le compte de tests de la commande L2a est ≥ au « avant » écrit dans ce plan**, tous verts,
   et le garde `xuidMeta.guard` reste vert (la cascade n'a pas été recopiée).
4. **`make check-types` + `make test-web` verts** (cache `node_modules/.tmp` purgé avant) ; le
   nouveau garde-rail `is_me` échoue si on en réintroduit un ; **à l'écran, la page de rejeu
   est identique à la capture E0, et l'onglet Chronologie de la page match identique aux
   captures L2a** — mêmes couleurs, mêmes pistes, mêmes sections, même son de fin.

La mesure E2.3 est écrite dans ce plan.

_Statut 2026-09-07 : verrous 1 à 3 PASSÉS, rejoués par le superviseur — diff match-view limité
à `xuidMeta.ts` + son test, sections de l'onglet Chronologie intactes ; photographies L2a sans
une ligne supprimée ni modifiée (`grep '^-[^-]'` vide) ; commande scopée 46 fichiers / 493 tests
(+26 cas à trois arguments), garde `xuidMeta.guard` vert, nouveau garde
`noIsMeOutsideViewpoint` vert (allowlist à 5 entrées : `playerMarks`, `matchSides`,
`victoryLogic` — replis sans sujet fixés par L2a — `MatchPadControlSection`,
`MatchEquipmentUsageSection` — page match ; le foyer `replayViewpoint.ts` ne lit PAS `is_me`, il
délègue à `meXUIDOf`). Suite complète 619 fichiers / 6492 tests, `tsc -b --force` 0, lint =
baseline. **E2.3 : médiane 0,16 ms** (min 0,14 / max 0,25) sur 20 bascules, artefact réel
4ecdf3e7 (2714 images, 9 joueurs, 38 vies, 90 kills) — 300× sous le seuil, rien découpé,
mesure écrite au-dessus de la mémo ; le banc `replayModel.bench.test.ts` se saute sans
l'artefact (CI). Commit sur la branche. **Verrou 4 (visuel, utilisateur) OUVERT** : il se
vérifie sur l'état L3 avec le joueur par défaut sélectionné — rejeu et onglet Chronologie
identiques à `:5173`, à l'exception du menu lui-même._

---

### L3 — Le menu de joueurs

- [ ] Colonne des libellés : `grid-cols-[76px_1fr]` → `[100px_1fr]`. Vérifier que les pistes,
      qui sont en pourcentage de leur colonne, suivent sans retouche.
- [ ] Remplacer le `TrackLabel` de la première piste par un `<select>` **natif**, sections par
      équipe en `<optgroup>` alimentées par `groupByTeam`, groupe « Sans équipe » inclus.
      Natif et pas un menu maison : largeur fermée contrainte à 100 px pendant que la liste
      ouverte montre les noms entiers, clavier et lecteur d'écran gratuits.
- [ ] **La valeur d'une option est `p.board?.xuid ?? p.xuid`** (piège des bots, §3.3). Test
      dédié : un bot sélectionné doit peupler sa piste.
- [ ] Troncature CSS du libellé fermé, nom entier dans `title`.
- [ ] La sélection appelle le foyer de L2. **Elle ne touche ni au curseur, ni à `playing`, ni à
      la vitesse** (décision 1). Test : bascule pendant la lecture, le curseur ne bouge pas.
- [ ] Piste « Alliés » → « Coéquipiers » : elle liste désormais les coéquipiers du joueur
      sélectionné, plus les joueurs marqués `friend`. Mettre à jour `buildEventTracks` et son
      en-tête, qui documente l'ancienne règle.
- [ ] Les amis prennent le **losange** (décision 4) : marque tournée à 45°, couleur inchangée.
      Réutiliser le vocabulaire de `MarkerShape` plutôt que d'en inventer un second.
- [ ] i18n : libellé accessible du menu, nom du groupe « Sans équipe », nouvelle piste
      « Coéquipiers » / « Teammates ». FR + EN.
- [ ] Défaut au montage = joueur de la page, non persisté (décision 8). Test.
- [ ] **Score final de l'écran de victoire vu depuis l'autre camp** (découverte L2b, invisible
      tant que le point de vue ne bouge pas) : `victoryLogic.finalScoreFromHeader` rend
      `{ ally: score_mine, enemy: score_theirs }` ancré sur le joueur de la page, et
      `ReplayVictoryOverlay` (`FinalScoreLine`) lui donne la PRIORITÉ sur `readScoreBanner`, qui
      lui suit le point de vue. Vu depuis un adversaire, l'issue serait permutée mais le score
      dans l'ordre du joueur de la page. Permuter `ally`/`enemy` quand le sujet est de l'autre
      camp (même règle que `readVictory` à trois arguments). Test : sujet adverse → score
      inversé, sujet allié → identique.
- [ ] **Le focus ne reste pas sur le menu après un choix.** Vérifié en revue :
      `useReplayShortcuts.ts:81` coupe TOUS les raccourcis quand l'élément actif est un `SELECT`.
      Sans précaution, choisir un joueur laisse le focus sur le menu, et la barre Espace qui
      suit — le geste réflexe pour mettre en pause — OUVRE la liste au lieu d'arrêter la
      lecture. Règle : `blur()` du select dans son `onChange`, une fois la valeur posée. Le menu
      garde ses flèches tant qu'il est ouvert (elles changent de joueur, c'est attendu) ; il
      n'est PAS exempté par `data-replay-timeline`, qui vise nommément le curseur. Test : après
      un changement de joueur, Espace bascule la lecture.

**Gate L3** : `make check-types` + `make test-web` verts ; à l'écran, sélection d'un adversaire
→ Dominance et Score inversent leurs couleurs, la carte et les fiches suivent, le curseur n'a
pas bougé ; sélection d'un bot → sa piste est peuplée ; le son de fin reste celui du joueur de
la page (vérifié en atteignant la fin après bascule).

_Statut 2026-09-07 : tous les items de code `[x]` (exécuteur Opus, relus sur pièces :
`viewpointOptions.ts` pur avec la règle de valeur `board?.xuid ?? xuid` et la règle
« désactivé = pas de ligne », `ReplayViewpointSelect.tsx` natif avec `blur()`,
`buildEventTracks` sur un `TrackAudience { viewpoint, identity, marks }`, `finalScoreFromHeader`
avec sujet réutilisant `subjectCampIndex`). Nuance de l'exécuteur, juste : « pas de camp » et
« pas de ligne » ne sont pas la même condition — un joueur avec ligne mais sans `team_side` est
dans « Sans équipe » et reste ACTIF. La marque amie est un carré 6×6 tourné (une barre de 3 px
tournée donnerait une oblique, pas un losange). `trackYou` supprimé (plus de lecteur). Gates
rejoués par le superviseur : vitest match-replay + lib/replay + match-view 219 fichiers /
3026 tests, `tsc -b --force` 0, match-view intact ; suite complète 620 / 6522 et lint 0 erreur
côté exécuteur. Commit sur la branche. **Verrou visuel utilisateur OUVERT** (points a-f du
rapport, sur `:5174`)._

---

### L4 — Médailles et présence

- [ ] **Ombrage de présence** sur la piste du joueur sélectionné : intervalles hors partie
      grisés, dérivés des `PresenceEvent` déjà présents dans le fil. Aucun nouveau calcul de
      temps — réutiliser l'axe du rejeu.
- [ ] Même ombrage sur la piste Coéquipiers, agrégé (l'effectif qui passe de 4 à 3 explique une
      dominance qui s'effondre — c'est le gain caché de ce lot).
- [ ] Propager `source: 'api' | 'film'` : bord franc pour l'API, bord dégradé + réserve dans
      l'infobulle pour le film (marge 10-20 s, et le film ne distingue pas un départ d'une
      élimination définitive). **Ne pas dessiner une frontière au pixel sur une déduction.**
- [ ] **Extraire `PresenceGlyph`** de `ReplayPresenceLine.tsx` vers son propre module, et
      rebrancher la ligne du fil dessus. Aucune copie du SVG. Vérifier qu'il reste lisible aux
      dimensions de la piste (14×12 aujourd'hui dans le fil ; la piste fait ~18 px de haut —
      passer une taille en props, pas un second dessin).
- [ ] Poser le glyphe **à la frontière ombre/jeu, du côté OMBRÉ** (décision 2 bis) : bord droit
      de l'ombre de tête pour un arrivant, bord gauche de l'ombre de queue pour un partant.
      Côté ombré et pas à cheval : c'est ce qui garantit qu'il ne recouvre jamais une marque de
      kill, qui ne vit que dans la zone jouée.
- [ ] Le glyphe **EST le bouton** (décisions 2 et 2 bis) : un seul objet cliquable au milieu
      d'une piste en `pointer-events-none`, sur le patron des vignettes de la piste Médias. Le
      clic pose le curseur à cet instant ; libellé accessible explicite, et infobulle reprenant
      le vocabulaire du fil (l'API affirme « a rejoint / a quitté », le repli film reste au fait
      « entre en partie / ne reviendra plus »).
- [ ] Teinte du glyphe : équipe de l'acteur quand elle est connue, encre de repli sinon —
      jamais un camp deviné. Via `tokenCssVar`, aucun hex.
- [ ] **Aucun glyphe sur la piste Coéquipiers** : son ombrage est agrégé, quatre portes
      empilées y seraient illisibles. Les glyphes ne vivent que sur la piste du joueur
      sélectionné. Test.
- [ ] Cas des deux bornes sur le même joueur (arrivé ET parti) : deux ombres, deux glyphes,
      chacun sur sa frontière. Test.
- [ ] Médailles attachées à un kill : **décorer la marque existante** selon la décision 14
      (anneau), médaille nommée dans l'infobulle. Pas de second glyphe (décision 9).
- [ ] Médailles orphelines : marque propre, sur la piste du joueur sélectionné.
- [ ] Dégradation : médaille sans nom ni image (matchs d'avant le backfill) → aucune
      décoration, aucune icône cassée. Test.
- [ ] **Définition de l'ombrage « agrégé » de la piste Coéquipiers** (le plan disait
      « agrégé » sans le définir — critère subjectif, corrigé en revue) : opacité de l'ombre =
      part de l'effectif ABSENT à cet instant, par paliers (1 absent sur 3 = 1/3, etc.), calculée
      dans `presenceTrackLogic.ts` à partir des mêmes `PresenceEvent`. Test sur 4 coéquipiers
      dont un arrivant et un partant : trois paliers distincts, aux bons instants.
- [ ] **Dégradation sans horloge établie** : `presenceEntries` rend `[]` quand `!playWindow ||
      !clock` (même porte que les lignes du fil). L'ombrage est alors ABSENT, pas faux, et
      c'est voulu — l'écrire en commentaire à l'endroit où la piste lit la présence, sinon la
      prochaine relecture y verra un bug.
- [ ] Hauteur de la piste : 14 → ~18 px, pour loger anneau et glyphe. Aucun garde-rail
      existant ne la couvre (vérifié en revue : `rosterHeight.guard` vise les fiches).
- [ ] i18n des nouvelles infobulles, FR + EN.

**Gate L4** : `make check-types` + `make test-web` verts ; sur le match témoin, un joueur arrivé
en cours de partie a sa piste ombrée jusqu'à son entrée avec le glyphe de porte posé sur la
frontière, le clic sur ce glyphe y emmène le curseur, et une marque de kill médaillée se
distingue d'une marque nue. Capture d'écran des deux cas (arrivant, partant) au journal.

_Statut 2026-09-07 : tous les items de code `[x]` (exécuteur Opus ; i18n `[~]` : aucune string
neuve, le vocabulaire du fil couvre la frise, `presenceWording` centralisé dans `model/`).
Relu sur pièces : `presenceTrackLogic.ts` (ombres par joueur, paliers fondus à même effectif),
`ReplayPresenceShade.tsx` (porte = bouton, `-translate-x-full` pour l'arrivant, dégradé film
seulement, encre par camp / `currentColor`), `ReplayMarkTrack.tsx` (anneau, anneau creux 8 px,
losange 6 px). Incident : disque C: plein en cours de lot, repris sans perte. Deux corrections
superviseur : un 28e warning lint (`react-refresh`) éliminé en sortant `presenceWording` du
composant ; une écriture interdite dans le `thought_log` du worktree, restaurée. Gates rejoués
par le superviseur : vitest match-replay + lib/replay + match-view 3077 tests, `tsc -b --force`
0, lint 27 = baseline ; suite complète 622 / 6573 côté exécuteur. Commit sur la branche.
**Verrou visuel utilisateur OUVERT** (arrivant, partant, paliers Coéquipiers, anneau, porte
cliquable sans pause)._

---

### E5 — Clôture

- [x] Aucun item du plan sans statut (les items « vérification à l'écran » des gates L1, L3, L4
      restent `[ ]` : ils appartiennent à l'utilisateur, pas à l'exécution).
- [x] `.ai/thought_log.md` : entrée datée (règle CLAUDE.md, obligatoire avant commit) — tenue
      par le superviseur dans le checkout principal, mise à jour à chaque lot.
- [x] Tout report éventuel inscrit à `.ai/V7.5/REGISTRE_REPORTS.md` avec sa condition de
      reprise : 4 entrées (2026-09-07) — `useReplaySound` à 7 paramètres ; deux règles d'encre
      pour un camp inconnu (fil / frise) ; banc `replayModel.bench.test.ts` hors dépôt ;
      `header.outcome_label` en FR codé en dur côté Go (pré-existant, révélé par le menu).
- [x] Skill `delivery-checklist` passé (2026-09-07) : aucun TODO/FIXME dans le diff de branche ;
      lint 0 erreur / 27 warnings = baseline ; tsc -b --force 0 ; aucune string sans FR+EN ;
      aucun hex ; `routeTree.gen.ts` intact ; match-view limité à `xuidMeta.ts` + test ;
      photographies L2a intactes. Remarque : `ReplayTimelineTracks.tsx` (521) et
      `replayTimelineTracksLogic.ts` (566) dépassent 500 lignes BRUTES — sous 500 lignes de CODE,
      la métrique que `max-lines` du dépôt applique (`skipComments`, `skipBlankLines`) ; les
      commentaires de la feature sont longs par doctrine. Pas une exemption, un constat.
- [ ] Skill `adversarial-review` sur le diff complet des quatre lots — L2 y passe en priorité :
      c'est le lot qui touche à une notion lue partout, et son échec serait silencieux.
      **Ronde 1 (2026-09-07)** : deux relecteurs à contexte frais, aveugles l'un à l'autre
      (L3+L5 anti-patterns/front ; L6 ce que les tests ne couvrent pas). Recevables : 4 + 8, dont
      2 en commun. Triage superviseur, vérifié sur pièces :
      - **P0 le mot du verdict** (`ReplayVictoryOverlay.tsx:133`, `exportOverlayPanels.ts:133`) :
        camp, logo et score suivent le sujet, le titre reste `header.outcome_label` — « Victoire »
        au-dessus de l'équipe perdante. Trouvé par les DEUX relecteurs. Remède : libellé
        canonique `useOutcomeLabel` quand la lecture est permutée, `header.outcome_label` sinon.
      - **P0 la présence des bots** (`presenceFeed.ts`) : clé FILM `bot:<nom>` contre xuid de BASE
        partout ailleurs — sur le témoin, le bot remplaçant n'aurait ni ombre ni porte. Remède :
        `p.board?.xuid ?? p.xuid`.
      - **P0 le repli `is_me` tué par défaut** (`xuidMeta.ts`, 3e argument inconditionnel) :
        en MÊLÉE GÉNÉRALE (`team_side` nul, vérifié côté Go : pointeur posé seulement par le
        constructeur d'équipes) plus personne n'est allié, le pion du joueur de la page prend
        l'encre adverse. Requalifié P1 → P0. Remède : à trois arguments, le SUJET est allié par
        définition (généralisation exacte du repli `is_me`).
      - **P1 relais non typés** : retirer `viewpoint` des cinq appels de `ReplayCanvas` laisse
        2 673 tests verts (aucun test ne monte le canvas). Remède : paramètre REQUIS au typage
        sur les porteurs de la feature (l'oubli ne compile plus), arité caractérisée inchangée.
      - **P1 branchements sans test** : `model.score`, `useReplayViewpoint`, `useReplayTimeline`
        (ombres du sujet, coéquipiers excluant le sujet), `seekToFrame` sans pause. Remède : tests
        `renderHook` + cas modèle.
      - **P1 doc inversée** ×4 (`replayTimelineGrid`, `victoryLogic`, `ReplayVictoryOverlay`,
        `PresenceGlyph`).
      - Jetés : la sentinelle du garde géométrie « ne rougit pas » sur un seul composant
        (couverte par deux tests de rendu nommés) ; `skipIf` du banc (justifié).
      Gardes-rails prouvés rouges par inversion : géométrie (littéral recopié), `is_me` (lecture
      ajoutée, exemption périmée), `xuidMeta.guard`, `testDoc.guard`. Caractérisations : zéro
      ligne supprimée depuis leur naissance. Conditions qui tiennent : 23 + 27.
      **P0+P1 ronde 1 = 11 distincts.** Ronde de correction lancée (un exécuteur, F1-F6) ;
      ronde 2 = relecture des seules corrections par un contexte frais.
      **Corrections F1-F6 livrées (2026-09-07)** : F1 sujet allié par définition à 3 arguments
      (fixture FFA : 3 args vu du joueur de la page = 2 args, table entière ; modèle sans camps
      = 2 args) ; F2 `victoryIsFlipped` pure, `useOutcomeMapping(reading.outcome)?.label` quand
      permuté (distingue « pas de mapping » de « libellé » — sans ça une lecture permutée sans
      mappings écrirait « loss » en plein cadre ; dégradation = pas de panneau, testée),
      `header.outcome_label` sinon ; export via `useViewedOutcome` dans `useExportSeam` ;
      `finalScoreFromHeader` réécrit sur `victoryIsFlipped` (une seule définition de « l'autre
      camp ») ; F3 `p.board?.xuid ?? p.xuid` pour `presence.xuid` ET la clé de ligne ; F4 relais
      REQUIS au typage (mutation : retirer `viewpoint` des cinq appels = 5 erreurs `tsc`) ;
      F5 tests `renderHook` (`useReplayViewpoint` 8 cas, `useReplayTimeline` monté — mutation :
      3 cas rougissent —, `useReplayPlayback.seek.test.tsx`, `model.score`) ; F6 doc ×4.
      **Arbitrage superviseur** : le gate « photographies sans ligne supprimée » a rendu DEUX
      lignes — le cas `nomad-9` à TROIS arguments de `xuidMeta.test.ts`, né en L2b (pas en L2a),
      qui décrivait précisément le défaut F1. Accepté : la photographie L2a (deux arguments) est
      intacte, le cas L2b contredisait le remède. Retouche finale : `ReplayExportOptions.viewpoint`
      requis (découverte de l'exécuteur, même classe que F4). Gates exécuteur : 223 fichiers /
      3125 tests ciblés, suite complète 624 / 6621, tsc 0, lint 27 = baseline. Gates superviseur
      rejoués : 3125 tests, `tsc -b --force` 0, lint 27 ; formule F1, clé F3, `victoryIsFlipped`
      lus sur pièces. **Commit `047987d13`.** Ronde 2 lancée : un relecteur frais sur
      `60b462bdb...047987d13` seulement.
      **Ronde 2 (2026-09-07)** : F1-F6 « tiennent » avec preuves par mutation (F1 : 4 tests
      rouges ; F3 : 2 ; F4 : 7 erreurs `tsc`, une par porteur ; F5 : 4 mutations, 4
      rougissements) ; 31 conditions tiennent ; suite complète 624 / 6621. Recevables :
      **2 P1** (doc inversée : `useReplayCapture.ts:289` nomme `useOutcomeLabel` au lieu de
      `useOutcomeMapping` — suivre ce commentaire peindrait « loss » en clé brute ;
      `exportOverlayPanels.ts:103` « camp du joueur de la page » → sujet) et **3 P2** (calcul de
      `viewedLabel` sans test — le remplacer par `label` laisse tout vert ; prop d'ENTRÉE
      `ReplayCanvas.viewpoint` encore optionnelle alors que les relais internes sont requis ;
      commentaires affirmant que `header.outcome_label` et le mapping ont « la même source » —
      FAUX : le backend sert une map Go codée en dur en FR, `service/match_history_service.go`).
      **P0+P1 : 11 → 2, décroissance stricte.** Pas de ronde 3 : les deux P1 sont des
      commentaires, corrigés avec les deux P2 mécaniques (prop requise, test du calcul) dans une
      retouche finale sans nouvelle relecture. Le troisième P2 est PRÉ-EXISTANT et hors
      périmètre — inscrit au `REGISTRE_REPORTS` (issue par clé canonique côté Go, à décider) ;
      conséquence visible depuis L3 : en locale EN, « Victoire » par défaut et « Loss » permuté.
- [ ] Commits poussés, **CI surveillée jusqu'au vert au niveau JOB**. Tout rouge se répare,
      même préexistant.

_**EN ATTENTE (décision utilisateur du 2026-09-07)** : les verdicts visuels (verrous L1/L3/L4 et
revue) et le merge/push attendent que `feat/v75` soit LIBÉRÉE par l'audit en cours et ses
correctifs. La branche `feat/v75-frise-pov` reste à 6e0d808b9, 7 commits, non poussée. Protocole
de reprise : (1) `git merge feat/v75` dans `feat/v75-frise-pov` (feat/v75 aura bougé) et rejouer
les gates (suite complète, tsc --force, lint, verrou match-view, photographies) ; (2) relancer
API `:8000` + vite `:5174` si arrêtés ; (3) verdicts visuels utilisateur ; (4) go merge + push +
CI. Décision produit connexe actée : l'issue sera servie par sa clé canonique côté Go — lot à
part, au registre._

## 5. CE QUI EST HORS PÉRIMÈTRE — noter, ne pas traiter

Consigner ici toute découverte faite en chemin, sans y toucher :

- Le `if (!meta) continue` de `collectKillEvents` (kill d'un acteur hors scoreboard, jeté).
- Le backfill `backfill-medailles-feed` — tâche de RELEASE, jamais lancée par l'agent.
- Toute cuisson d'artefact en lot — INTERDITE sans accord explicite (bombe RAM, verrou
  `filmproc.AcquireSolo`).
- Les cinq appels `resolveXuidMeta` de `match-view` : ils reçoivent un paramètre de plus, leur
  comportement ne change pas. Ne pas en profiter pour les refondre.

### Découvertes (à remplir en cours d'exécution)

- **L1, 2026-09-06 — les marques sont ancrées par leur bord GAUCHE.** `MarkTrack`
  (`ReplayTimelineTracks.tsx`, `left: trackLeft(ratio)`, largeur 3 px, aucune translation) alors
  que la pastille du curseur est CENTRÉE sur ce même point : une marque est donc visuellement
  ~1,5 px à droite de l'instant qu'elle désigne. Comportement d'origine, documenté ; le trait de
  lecture a été aligné sur la même ancre (il longe le bord gauche de la marque) plutôt que
  d'arbitrer. À statuer hors lot : recentrer les marques (`-translate-x-1/2`) — une ligne, mais
  qui bouge toutes les marques du parc.
- **L1, 2026-09-06 — `apps/web/vite.config.ts:28`** : avertissement Vite à chaque run,
  `__dirname` non supporté par le futur `configLoader: 'native'`. Hors périmètre.

## 6. REPRISE DE SESSION

Avancement = les cases de la §4 de ce fichier, plus la dernière entrée de
`.ai/thought_log.md`. `git log --oneline -10` sur `feat/v75-frise-pov` dit où en sont les lots.
Reprendre à la première case non cochée de la première étape non close — jamais plus loin.
