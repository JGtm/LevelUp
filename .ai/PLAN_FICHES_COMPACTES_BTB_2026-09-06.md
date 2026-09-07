# PLAN — Fiches compactes du rejeu (Grande équipe)

> Créé le 2026-09-06, **révisé le même jour après revue à quatre angles** (architecture,
> tests, performance, produit) — 4 verdicts « GO SOUS CONDITIONS », 40 constats, tous
> vérifiés sur pièces par le superviseur avant intégration. Maquette validée (variante A2) :
> https://claude.ai/code/artifact/2a4798f4-09eb-4f24-9ef0-cc8c9e14b909
> Exécution sous le contrat du skill `plan-execution` (ordre strict, aucun report d'étape
> exécutable, statut par item, vérification sur pièces, zéro fix hors périmètre).

## Objectif

Afficher **24 fiches joueur** dans la colonne latérale du rejeu (12v12 Grande équipe) là où
elle en montre 8 à la fois, **sans toucher au 4v4** ni à la mise en page de la page.

Moyen : quand le match est une Grande équipe (catégorie de mode `BTB`, cf. décision D1), la
colonne d'équipe passe à **deux colonnes de tuiles compactes de 115 × 62 px** au lieu d'une
colonne de tuiles de 235 px. Pour tout autre type de match, rien ne change — la fiche
actuelle reste identique, nœud DOM pour nœud DOM.

**Critère de succès** : sur le témoin BTB, à une fenêtre d'au moins 1 440 × 900, les 24
fiches sont visibles simultanément sans défilement ; le gamertag occupe la ligne entière ; le
témoin 4v4 rend un `innerHTML` strictement identique à la fixture prise avant le lot.

**Branche** : `feat/fiches-compactes`, worktree dédié `LevelUp-wt-fiches-compactes` créé
depuis `feat/v75`. Le worktree principal est partagé — interdiction d'y travailler, jamais de
`git add -A`, fichiers stagés nommément.

**Périmètre** : `apps/web/src/features/match-replay/` uniquement. **Aucun Go, aucune donnée,
aucun schéma, aucune cuisson d'artefact, aucun fichier de `lib/replay/`** (partagé avec la
Match View — voir I4).

**Témoins (vérifiés sur disque)** :
- BTB : `data/cache/replays/halo_infinite/4f77afc1.json` — schéma 34, 25 joueurs,
  11 111 images. Réserve : calques 38+ absents (`abilityCharges`, `translocations`), donc
  borne basse pour la perf. Aucune cuisson à demander.
- 4v4 : `data/cache/replays/halo_infinite/000d5950.json`.

---

## Décisions produit — tranchées avant exécution

**D1 — La densité se lit sur le TYPE DE MATCH, pas sur un compte de sièges : compacte si
`header.mode_category === 'BTB'`** (décision utilisateur du 2026-09-06 : « les matchs de type
4v4 on touche pas »). L'en-tête de la vue match porte déjà `mode_category`
(`domain.MatchViewHeader`, `json:"mode_category,omitempty"`, renseigné par
`applyMatchHeaderModeCategory` via la `ModeTaxonomy` du titre — `games/halo_infinite/
mode_category.go` : préfixes `BTB` et `BTB Heavies` → `ModeCategoryBTB`). Le web en dépend
déjà par une table `Record<string, ...>` (`features/session-detail/SessionParamPills.tsx`) :
même convention ici. Title-agnostic par construction : un titre sans taxonomie (Halo 5) laisse
la catégorie vide → densité normale, sans branche sur le slug. Toute heuristique sur le nombre
de sièges, de joueurs ou de lignes de tableau est **exclue** : un 4v4 reste un 4v4 quel que
soit le nombre de relais.

Conséquence : `ReplayTeams` reçoit la catégorie par son `header` (le type `PresenceHeader`
de `model/presenceFeed.ts` s'élargit d'un champ `mode_category?: string | null` ; la page
`replay.tsx` passe déjà `matchView?.header`). Sans en-tête (vue match indisponible) : normale.

**D2 — Pas de groupe « sans équipe ».** Humain ou bot, tout joueur a une équipe (décision
utilisateur du 2026-09-06). Le cas « bot absent du tableau → sans camp » relevé en revue est
un défaut de donnée, pas un cas produit : il n'est pas traité, ni en normale ni en compacte —
le rendu actuel (`repeat(groups.length, 1fr)`) est conservé tel quel et consigné en
découverte s'il est observé sur un artefact.

**D3 — FFA : chaque joueur est sa propre équipe.** `team_side` vaut `t{TeamID}` par joueur,
donc N groupes d'un siège, jamais `BTB` en catégorie → densité normale, affichage inchangé.
Hors périmètre.

**D4 — Fiche morte compacte « hors film »** : le repère `goneLabel` (« Hors film ») seul sur
la ligne 1 avec le triplet ; la ligne 2 reste vide et l'infobulle porte `goneValue`
(« ne revient plus », 107 px en mono 13 px, ne tient pas dans 85 px).

**D5 — Marques souples de la rangée d'inventaire** (`InventoryEmptyMark` « Mort » /
« Inventaire indisponible », `drawnUnknown` « dégainée ? », `grenadeSelUnknown` « sél. ? »,
`AbilityChargeMark` « ×N » / « plein ») : en compacte, elles passent dans l'infobulle de la
cellule concernée (arme, grenade, capacité), avec leurs deux âges quand elles en portent.
Aucune ne s'affiche en texte sur la tuile.

**D6 — Infobulle native (`title`)** : c'est la doctrine des fiches (52 `title=` dans
`ui/`). Deux limites assumées et écrites : absente au toucher ; le `title` d'une cellule masque
celui de la tuile (`fx.title`) — le gate 3 vérifie que `fx.title` reste atteignable sur une
zone sans cellule (la ligne de nom).

---

## Le gabarit tranché (A2) — cotes vérifiées, à ne pas rouvrir

Toutes les cotes sont des **champs d'un objet `CardGabarit`** (cf. I1) ; `GABARIT_NORMAL`
porte exactement les constantes d'aujourd'hui, `GABARIT_COMPACT` celles-ci.

Tuile compacte : largeur = cellule de grille (`auto-fill minmax(115px, 1fr)` → 115,5 px dans
un camp de 235 px), **hauteur 62 px**, marge 6 px, rayon 6 px, bordure 1 px, contenu 101 px.

| Ligne | Hauteur | Contenu (largeurs) |
|---|---|---|
| 1 | 14 px | Nom seul, `leading-[14px]`, 11,5 px gras majuscules `tracking-[.06em]`, ligne entière (101 px ≈ 12 caractères visibles, nom complet en `title`) |
| 2 | 12 px | Triplet F/M/A à droite : mono 9 px, cellules `min-width 10 px`, `gap 2 px`, fond FDA `padding 0 3px`, séparateurs `/` ≈ 5 px → **≈ 54 px** ; jauges à gauche en `flex-1` → **≈ 42 px** (bouclier 4 px sur santé 2 px, `gap 2 px`) |
| 3 | 16 px | Arme en main **48 px** · grenade sélectionnée **14 px** (= `GRENADE_ICON_PX`) · capacité **16 px** (= `HUD_ICON_PX`), `gap 5 px` → 88 px, 13 px d'air |

Interlignes 3 px. **Corps à hauteur fixe 31 px** (12 + 3 + 16). Somme : 1 + 6 + 14 + 3 + 31 +
6 + 1 = **62**. La ligne 1 impose `leading-[14px]` : sans elle, le preflight Tailwind v4
(`line-height: 1.5`) donne 17,25 px et la tuile 65.

Fiche morte compacte : encadré `padding 0 5px` (91 px utiles), deux lignes dans les 31 px —
ligne 1 (13 px) : `ÉLIMINÉ` en 7 px `tracking-[.15em]` (≈ 38 px) + triplet (≈ 54 px, sans
`px-1`) ; ligne 2 (14 px) : décompte mono 13 px à droite. Cas « hors film » : D4.

Ce qui quitte la tuile compacte et **passe en infobulle, avec sa valeur** : arme rangée
(nommée), munitions de la main (« 24 / 60 », « 87 % », « Munitions pleines », rien en D=2),
stock des trois grenades (réemploi de `grenadeBoxHint`, qui existe), score personnel (valeur
+ « à l'instant lu »), et les marques de D5.

**Strictement conservé** : cellules à largeur fixe (donnée absente = cellule vide, jamais un
décalage), estompage par âge cellule par cellule, fond FDA du triplet, badge de lancer de
grenade superposé à la cellule d'arme (48 px), échange d'arme animé (cf. I6), toutes les
couches d'effets, règle « lacune ≠ zéro », règle « document sans `sh`/`hp` = aucune barre ».

---

## Impacts — vérifiés sur pièces après revue

### I1. Deux gabarits, un seul jeu de composants (RISQUE PRINCIPAL, requalifié)

Le commit `10fe3228a` (2026-08-25) a supprimé le réglage `replay-compact-cards` ET la prop
`compact?: boolean` drillée dans `ReplayInventoryRow` avec sa branche `if (compact)`. La
première version de ce plan reproduisait cette forme (un discriminant `'compacte'` propagé
aux sous-composants). **Forme retenue** : `model/cardGabarit.ts` exporte un type
`CardGabarit` (`weaponCells: 1 | 2`, `weaponCellW`, `showAmmo`, `showGrenadeStock`,
`showScore`, `countCellW`, `scoreCellW`, `gaugeShieldPx`, `gaugeHealthPx`, `bodyPx`,
`iconGrenadePx`, `iconAbilityPx`, `watermarkPx`, `boltCount`, `namePx`) et deux constantes.
`model/cardDensity.ts` choisit la constante à partir des groupes. Les composants reçoivent
**des nombres et des booléens**, jamais le mot « compacte » ; le 4v4 rend le même DOM parce
que `GABARIT_NORMAL` porte les constantes actuelles (40 / 32 / 56 / 30 / 15 / 16 / 46 / 3).

Pas de prop de densité sur `ReplayTeams`, pas de réglage, pas de drapeau : la densité dérive
du nombre de sièges, les deux gabarits sont atteints par des matchs réels et testés.
Commentaire de tête de `ReplayTeams.tsx` corrigé dans le même commit (anti « doc inversée »).

**Garde-rail (règle n° 6, sept constantes de cotes dans quatre fichiers aujourd'hui)** :
`ui/cardGabarit.guard.test.ts` sur le patron de `settings/replayPreferences.guard.test.ts`,
qui interdit dans `ui/ReplayPlayerCard.tsx`, `ReplayWeaponsRow.tsx`, `ReplayInventoryRow.tsx`,
`ReplayCountersBadge.tsx`, `ReplayAbilityCell.tsx`, `ReplayVitality.tsx`,
`ReplayObjectiveMark.tsx` : toute déclaration `const \w+_(CELL_W|BOX_W|ICON_PX|PX) =`, tout
littéral `'compacte'` / `'normale'`, tout import depuis `../settings/`. Le littéral de densité
n'est admis que dans `model/cardDensity.ts` et ses tests.

### I2. Extraction — par responsabilité, pas par le plafond de lint

`max-lines: 500` compte hors commentaires et lignes vides : `ReplayTeams.tsx` ≈ 305-330
lignes comptées (488 brutes), aucun fichier n'approche le plafond. En revanche `PlayerCard`
fait 149 lignes brutes et `ReplayInventoryRow` 150 — la règle « fonction ≤ 80 L » est déjà
dépassée et non lintée (dette gelée, à ne pas accroître). **Extraction** : `PlayerCard`,
`ZoneFxOverlay`, `SHROUD_BOLTS`, `REPAIR_CROSSES` vers `ui/ReplayPlayerCard.tsx` ; les
lectures pures de `PlayerCard` (`playerCountersAt`, `playerStateAt`, `equippedWeapons`,
`activeEquipmentAt`, `positionAt`, `zonePresenceAt`, `objectiveMarkAt`, `lastTeleportAge`,
`playerCardFx`) vers `model/playerCardReadings.ts` — ce qui permet aussi la mesure « modèle
seul » de I4. `ReplayTeams.tsx` ne garde que la colonne, les groupes et la scène d'effets.

Contrainte d'extraction : sept tests existants atteignent la racine de la tuile par
`getByText('Alpha').parentElement.parentElement` et trois comptent ses enfants non
`aria-hidden` — la profondeur du DOM autour du nom ne bouge pas.

### I3. Tests existants — vérifié describe par describe : aucun ne casse, aucun ne se retouche

`ui/ReplayTeams.test.tsx` monte 1 siège par groupe via `renderTeams`, et cinq `describe`
montent leurs propres documents à 2-3 joueurs (vitalité, capteur adverse, D8, compteurs,
identité) : **maximum 2 sièges par groupe**, toujours sous le seuil. Aucun test n'asserte sur
le conteneur des sièges (`overflow-y-auto`, `gridTemplateColumns`), aucun snapshot n'existe
dans la feature — c'est précisément pourquoi la promesse « 4v4 identique » n'est protégée par
RIEN aujourd'hui, d'où 0.5.

**Règle** : `git diff feat/v75 -- apps/web/src/features/match-replay/ui/ReplayTeams.test.tsx`
doit rester **vide** à chaque gate. Toute retouche d'un test existant est une régression 4v4,
pas un test à adapter. Le travail de test est additif.

### I4. Performance — prémisse corrigée : les 24 fiches sont DÉJÀ rendues

`ReplayTeams.tsx` rend **tous** les sièges (`group.seats.map`, colonne `overflow-y-auto`) :
sur `4f77afc1`, 25 `PlayerCard` sont déjà montées et calculées à chaque publication, dont
~17 hors du clip de défilement. La colonne se re-rend à la cadence de **publication** (150 ms,
`hooks/useReplayClock.ts` `FRAME_PUBLISH_MS`), pas à 30 images/s. Ce lot ne change ni le
nombre de fiches calculées ni la cadence : **le coût JS est inchangé** — ce qui change est la
**surface visible peinte** (24 tuiles au lieu de 8) et donc la peinture et les animations CSS.

Ce qui reste vrai et préexistant : le modèle par fiche est linéaire sur les calques
(`inventoryAt` appelé deux fois par fiche + un balayage dans `drawnSwapAt`, `grenadeReadingAt`
sur `grenadeReads`, `zonePresenceAt` sur toutes les poses — 658 sur le témoin, le commentaire
« dizaines par film » est périmé), ≈ 6 600 itérations par fiche et ≈ 165 000 par publication
en BTB, soit 1-2 ms de modèle + 3-5 ms de réconciliation sur un budget de 150 ms.

**Ce que la première version disait de faux, retiré** : « 8 → 24 triple le travail » ; « la
compacte ne lit plus les munitions ni la boîte de grenades » (les infobulles consomment les
mêmes lectures, et la grenade sélectionnée dérive des compteurs complets) ; « `React.memo` »
(la prop `frame` change à chaque publication, memo ne sauterait jamais un rendu).

**Risque réel : la peinture.** `replay-death-flash` / `replay-respawn-flash` animent
`background-color` (non composé) ; `replay-flash-translocation` anime un `conic-gradient` +
`mask-composite` ; `replay-zone-bolt` cumule `filter: drop-shadow` et animation infinie ;
`backdrop-filter: blur()` par tuile en camouflage / écran. Sur le témoin, une mort toutes les
4,4 s ≈ une tuile au moins en éclat presque en permanence — aujourd'hui aux deux tiers hors du
clip, demain toutes visibles.

**Protocole de mesure (0.4 / 5.2), en deux volets** :
- *Volet JS, agent, rejouable* : `ui/ReplayTeams.perf.test.tsx`, gardé par
  `describe.skipIf(!process.env.REPLAY_PERF)` (jamais en CI) et par l'absence du témoin ;
  document construit par `testReplayDoc(JSON.parse(...))` (le garde `test/testDoc.guard.test.ts`
  interdit `normalizeReplayDocument` direct) ; tableau bâti depuis `doc.roster` ;
  `<Profiler id="ReplayTeams" onRender>` autour de `<ReplayTeams>` ; 20 rendus d'échauffement
  puis 100 `rerender` aux images `floor(5000 + k × 1.5)` (cadence de publication réelle) ;
  métrique `actualDuration` en p50 / p95 / total ; 5 répétitions, médiane des médianes ; plus
  la même boucle sur `playerCardReadings` seul (modèle sans React). Commande :
  `REPLAY_PERF=1 npx vitest run src/features/match-replay/ui/ReplayTeams.perf.test.tsx` hors
  sandbox. Seuil : BTB p50 APRÈS ≤ 1,10 × AVANT ; 4v4 APRÈS = AVANT ± 5 %.
- *Pas de volet navigateur chiffré* (retiré le 2026-09-07 sur décision utilisateur : « ce
  n'est qu'un changement UI »). Le risque de peinture reste une hypothèse de revue, pas un
  défaut mesuré ; il se juge à l'usage, au gate visuel du 5.3 — l'utilisateur lit le témoin
  BTB et dit si la lecture est fluide. Si elle ne l'est pas, ALORS on instrumente.

**Optimisations** : aucune spéculative. Dans le périmètre, un seul gain gratuit à appliquer si
le volet JS dégrade : calculer `inventoryAt` une fois dans la fiche et le passer à
`equippedWeapons` et à la rangée (supprime un des trois balayages d'inventaire par fiche).
L'index par document (échantillons groupés par slot, dichotomie dans `nearestReading`) est
**hors lot** : `nearestReading` vit dans `lib/replay/rosterLogic.ts`, partagé avec la Match
View et protégé par `rosterLogic.guard.test.ts`.

### I5. Grille et cas limites

- Colonnes d'équipe inchangées (`repeat(groups.length, 1fr)`) ; à l'intérieur d'un camp, les
  sièges en `grid-cols-[repeat(auto-fill,minmax(115px,1fr))]` en compacte — écrit en **classe
  Tailwind sans espace** (une valeur arbitraire avec espace ou `calc(` ne produit aucune
  règle, en silence — `rosterHeight.guard.test.ts` le documente) ; pas de `w-full` sur la
  tuile (un `div` remplit sa cellule). Le test de l'étape 4 asserte la chaîne exacte de la
  classe.
- La densité est celle du MATCH (D1) : un seul gabarit pour toute la colonne, quels que
  soient les effectifs.
- Sièges relayés : la fiche suit l'occupant, le compte ne bouge pas.
- Sièges surnuméraires en BTB (relais non appariés) : 7-8 rangées, la colonne défile — le
  comportement d'aujourd'hui, en plus dense.
- Catégorie absente (titre sans taxonomie, vue match indisponible) : normale.

### I6. Effets à cotes absolues — quatre, pas deux

- `SHROUD_BOLTS` : largeurs en px (42 / 34 / 28) sur des abscisses en % → chevauchement sur
  115 px. `boltCount` = 2 dans le gabarit compact.
- `REPAIR_CROSSES` : abscisses en % pur, glyphe 10 px → 18 / 53 / 85 px, **aucun
  chevauchement** ; restent à 3 dans les deux gabarits.
- `WATERMARK_PX = 46` (`ReplayObjectiveMark.tsx`), calibré sur un corps de 35 px, critère
  écrit « reste un fond » : `watermarkPx` = 34 dans le gabarit compact.
- `--replay-dx: 46px` (`globals.css`) = distance entre les DEUX cellules d'arme pour
  l'animation d'échange. À une cellule : `--replay-dx` posé à la largeur de la cellule (48)
  et seule la vignette entrante est animée (`replay-wswap-l`) — l'échange reste visible, il ne
  croise plus rien.
- Tout le reste est en `inset-0` et suit la tuile sans changement.

### I7. Hauteur disponible — formule réelle

À `xl`, la pile est `absolute inset-0` dans la colonne carte : plafond effectif des fiches =
`min(480, H − 192 − 12)` où H est la hauteur de la colonne carte (canvas borné 360..720 par
`useReplayView`). Besoin compact 12v12 : 6 × 62 + 5 × 4 + bandeau 23 + 6 = **421 px** → tient
si H ≥ 625, soit une fenêtre d'environ 900 px de haut ; à 1 280 × 720 (H ≈ 600) la colonne
défile de ~25 px. Sous `xl`, `max-h-[60vh]` s'applique. **Le plafond n'est pas touché** ; le
critère de succès nomme sa fenêtre minimale.

### I8. Export vidéo — non impacté (preuve corrigée)

L'export capture **la toile et elle seule** : `useReplayCapture.ts` (`canvasRef`,
`canvas.captureStream`), `replayCapture.ts` (`canvas.toBlob`). Les fiches sont du DOM hors
toile. `exportOverlayPanels.ts` décrit ce qui est repeint dans la toile, pas ce qui est capturé.
Les specs Playwright `e2e/replay-*-raster.spec.ts` rastérisent des primitives et sont hors
périmètre.

### I9. i18n

`weaponSecondaryHint` et `playerScoreLive` existent mais sont des libellés nus, sans place
pour une valeur. Nouvelles **fonctions de format** FR + EN (parité typée
`Record<ReplayLocale, ReplayText>`) : `weaponStowedFmt(name)`, `playerScoreLiveFmt(score)`,
`ammoHintFmt(mag, res | pct | full)`. `grenadeBoxHint` est réemployé tel quel pour le stock.
Les libellés de D5 existent déjà et sont réemployés dans les `title`.

### I10. Couleurs

Tokens sémantiques `info` / `success` / `destructive` / `warning` via `tokenCssVar` ;
variables de thème `var(--card)` / `var(--foreground)` et classes `text-foreground` /
`text-muted-foreground` / `bg-card` comme aujourd'hui (`card` et `foreground` ne sont pas des
`SemanticToken`). Aucun hex, aucune classe Tailwind couleur. Gate : `npm run lint:colors`.

---

## Étapes

Contrat : une étape à la fois, close ET vérifiée (gate passé) avant la suivante. Statut par
item : `[x]` fait · `[~]` couvert ailleurs (référence) · `[!]` non traité (justification).
Aucune case vide à la clôture. Chaque item nomme ses fichiers. Découvertes hors périmètre :
consignées en fin de document, jamais traitées. Un commit par étape, préfixé
`feat(fiches-compactes/etape-N)`, **après autorisation** de l'utilisateur.

Commandes canoniques (depuis `apps/web`, hors sandbox) :
- `T_EXIST` = `npx vitest run src/features/match-replay/ui/ReplayTeams.test.tsx`
- `T_DOM` = `npx vitest run src/features/match-replay/ui/ReplayTeams.dom4v4.test.tsx`
- `T_DIFF` = `git diff --stat feat/v75 -- src/features/match-replay/ui/ReplayTeams.test.tsx`
  (doit être vide)
- `T_ALL` = `Remove-Item -Recurse -Force node_modules\.tmp` puis `npx tsc -b --force`,
  `npm run lint`, `npm run lint:colors`, `npm run test:run` — codes de sortie lus, pas la
  sortie filtrée.

### Étape 0 — Préalables, fixation et mesure de départ

- [x] 0.1 Worktree `LevelUp-wt-fiches-compactes` / branche `feat/fiches-compactes` depuis
      `feat/v75` ; `git worktree list` et `git branch --show-current` le confirment.
      (2026-09-06 : confirmé ; `npm ci` du worktree neuf : 46 s, 507 paquets.)
- [x] 0.2 Témoins : lire `4f77afc1.json` et `000d5950.json` et noter dans ce plan
      `groups.length` et le nombre de poses ; vérifier par l'API locale (`/matches/{id}/view`)
      que l'en-tête du témoin BTB porte bien `mode_category: "BTB"` — sinon c'est un
      blocage à remonter (D1 n'a aucun repli).
      **Relevé (lecture seule, chemins du worktree principal)** : `4f77afc1` = schéma 34,
      25 entrées de roster, 253 traces, 11 111 images (100 ms), **658 poses**, 1 120
      `grenadeReads`, 153 objectifs ; `000d5950` = schéma 20, 8 entrées de roster, 99 traces,
      4 985 images, **295 poses**. `groups.length` n'est PAS lisible dans l'artefact : le camp
      vient du tableau (`team_side`), le film écrit `team: -1` sur toutes les traces — il vaut
      2 par construction du tableau (un tableau à deux camps) et se lit à la vue match.
      **`mode_category` du témoin BTB** : le serveur local (:8000) répond mais la vue match
      est gardée par l'ownership (403 `player_forbidden` sur les 9 profils, aucune session —
      aucun cookie fabriqué). Établi SUR PIÈCES côté Go : le témoin est documenté
      « BTB:CTF / Flood Gulch » (`.ai/ETAT_DE_L_ART_KILLWEAPON.md:1608, 1898`) ;
      `wire/registry_media.go:30` câble `InferCategory: halo_infinite.InferModeCategoryFromPairName` ;
      `InferModeCategoryFromPairName("BTB:CTF on Flood Gulch")` → `stripMapSuffix` → `BTB:CTF`
      → préfixe gauche `BTB` → `modePrefixToCategory["BTB"] = ModeCategoryBTB` (cas de test
      `{"BTB:Slayer", ModeCategoryBTB}`, `mode_category_test.go:183`) ; `applyMatchHeaderModeCategory`
      (`service/match_view_builders_header.go:278`) le pose dans `header.mode_category`. Le
      type généré `MatchViewHeader.mode_category?: string` existe déjà (`generated.ts:8324`).
      Reste à confirmer par l'utilisateur, en session, que la vue match du témoin rend bien
      `"BTB"` (lecture de `/players/{slug}/matches/4f77afc1-…`).
- [~] 0.3 Captures AVANT — **retirées le 2026-09-07** (décision utilisateur) : le 4v4 est
      protégé par la fixture DOM de 0.5, plus forte qu'une capture ; l'« avant » du BTB est la
      colonne qui défile aujourd'hui, sans valeur de comparaison. Le seul gate visuel est
      l'APRÈS (5.3).
- [x] 0.4 Mesure de perf AVANT, volet JS : écrire `ui/ReplayTeams.perf.test.tsx` selon I4,
      l'exécuter, consigner p50 / p95 BTB et 4v4 dans le journal d'exécution. Volet navigateur
      AVANT remis à l'utilisateur avec la liste exacte des relevés.
      Fait : test écrit (helper `test/scoreboardRow.ts` créé — 2e et dernière copie de la ligne
      de tableau, `ReplayTeams.test.tsx` garde la sienne), exécuté, chiffres au journal. Écart
      au protocole, dit : le témoin 4v4 n'a que 4 985 images, la base est 45 % de sa durée
      (image 2 243) au lieu de 5 000. Volet navigateur AVANT : retiré le 2026-09-07 (cf. I4).
- [x] 0.5 **Fixation DOM 4v4** : `ui/ReplayTeams.dom4v4.test.tsx` monte un document riche à
      2 camps × 4 sièges (`sbRow` `t0`/`t1`, loadouts 2 armes avec icônes, inventaire
      `d`/`am`/`g`/`gs`, `abilities`, `scoreTimeline` sur 2 joueurs, une pose d'écran, un
      épisode de camouflage, un mort à l'image lue, une lecture `empty: 'dead'`, un porteur
      d'objectif) et compare `container.innerHTML` à `ui/__fixtures__/replayTeams.4v4.html`
      enregistrée **avant toute modification de code**. Même test à 6 sièges (limite haute du
      normal) en `it.todo`, levé à l'étape 2. **Ce fichier est le premier commit de la
      branche, seul.**

**Gate 0** : `T_DOM` vert sur la fixture ; mesures AVANT (volet JS) chiffrées dans le
journal ; découvertes de 0.2 consignées. Aucun livrable utilisateur à ce gate (captures et
volet navigateur retirés le 2026-09-07). **Gate 0 passé le 2026-09-06.**

### Étape 1 — Gabarit, densité, extraction (la fiche ne change pas encore)

- [x] 1.1 `model/cardGabarit.ts` : type `CardGabarit`, `GABARIT_NORMAL` (constantes
      actuelles, migrées depuis `ReplayWeaponsRow`, `ReplayInventoryRow`,
      `ReplayCountersBadge`, `ReplayAbilityCell`, `ReplayObjectiveMark`, `ReplayTeams`),
      `GABARIT_COMPACT` (tableau ci-dessus). `EmptyWeaponCell({ width })` paramétrée.
      Fait (2026-09-06). Le type porte, en plus de la liste de I1, `ammoCellW` / `grenadesBoxW`
      (les deux constantes de `ReplayInventoryRow` que le garde-rail 1.5 interdit de laisser
      sur place) et `seatGrid: boolean` (le booléen que 1.4 consomme — pas de comparaison
      d'identité d'objet, pas de mot). Objets gelés. Les sous-composants reçoivent une prop
      `gabarit: CardGabarit` (`ReplayWeaponsRow`, `ReplayInventoryRow`, `ReplayCountersBadge`,
      `ReplayAbilityCell`) ou un nombre (`ReplayObjectiveMark.sizePx`, `VitalityBar.heightPx`
      déjà paramétrée, `WeaponChip.cellW`, `GrenadeChip.iconPx`, `AbilityUnknownMark.px`,
      `ZoneFxOverlay.boltCount`). `bodyPx` (35) et `namePx` (11,5) sont déclarés mais restent
      écrits en CLASSE (`h-[35px]`, `text-[11.5px]`) : une valeur arbitraire Tailwind doit être
      un littéral en clair — l'étape 2 les traduira par une table fermée de classes.
      Test `model/cardGabarit.test.ts` : le gabarit normal = les constantes d'aujourd'hui.
- [x] 1.2 `model/cardDensity.ts` : `cardDensity(header) -> CardGabarit` (compacte si
      `header?.mode_category === 'BTB'`, table `Record<string, CardGabarit>` sur la
      convention de `SessionParamPills`) + tests : `BTB` → compacte ; `Arena`, `Ranked`,
      `Fiesta`, `Firefight`, `''`, `undefined`, en-tête absent → normale ; un 12v12 dont
      la catégorie est vide reste en normale (aucun repli sur les effectifs). Élargissement
      de `PresenceHeader` (`model/presenceFeed.ts`) avec `mode_category?: string | null`.
      Fait. Lecture par `Object.hasOwn` (une catégorie « constructor » rendrait une fonction) ;
      `btb` / `BTB Heavies` → normale (la taxonomie du serveur a déjà classé, la fiche ne
      devine pas) ; le test « aucun repli sur les effectifs » asserte `cardDensity.length === 1`.
- [x] 1.3 Extraction : `PlayerCard`, `ZoneFxOverlay`, `SHROUD_BOLTS`, `REPAIR_CROSSES` →
      `ui/ReplayPlayerCard.tsx` ; lectures pures → `model/playerCardReadings.ts`.
      Déplacement pur, profondeur DOM autour du nom inchangée.
      Fait : `ReplayPlayerCard` (267 L brutes) rend, `playerCardReadings()` (137 L) lit ;
      `CardFxScene` vit avec les lectures. `ReplayTeams.tsx` passe de 488 à 219 lignes brutes.
      Preuve du déplacement pur : `T_DOM` vert sur la fixture prise avant le lot (md5 inchangé
      `a650f6b5…`). Le perf test importe désormais `playerCardReadings` (sa recopie a disparu).
- [x] 1.4 Grille : colonnes d'équipe inchangées ; sièges en
      `grid-cols-[repeat(auto-fill,minmax(115px,1fr))]` quand le gabarit est compact, colonne
      simple sinon. Fichier : `ui/ReplayTeams.tsx`.
      Fait : `SEATS_COLUMN_CLASS` = la chaîne d'aujourd'hui à l'octet ;
      `SEATS_GRID_CLASS` = `grid min-h-0 flex-1 auto-rows-max grid-cols-[repeat(auto-fill,minmax(115px,1fr))] gap-1 overflow-y-auto`
      (à asserter en 4.3), choisie sur `gabarit.seatGrid`.
- [x] 1.5 `ui/cardGabarit.guard.test.ts` (I1). Fait : sept composants nommés (et leur
      présence vérifiée — un garde qui ne trouve rien ne garde rien), trois interdits (cote
      locale `const X_(CELL_W|BOX_W|ICON_PX|PX) =`, mot de densité, import `../settings/`),
      contrôle du littéral de densité sur toute la feature hors `cardDensity.ts` (guillemets
      simples : deux commentaires citent une « explosion "normale" »), contre-épreuve sur le
      foyer (`GABARIT_NORMAL` / `GABARIT_COMPACT` / signature de `cardDensity` / `BTB:`).
- [x] 1.6 Commentaire de tête de `ReplayTeams.tsx` réécrit (I1), daté. Fait : l'en-tête dit
      la colonne (camps, scène, grille), D1 et les deux gabarits, et REQUALIFIE la note du
      2026-08-24 (« la fiche compacte est devenue la fiche ») — ce qui revient est un second
      jeu de cotes, ni l'option supprimée ni un réglage. La note inversée a été retirée.

**Gate 1** : `T_EXIST` vert, `T_DIFF` vide, `T_DOM` vert (DOM 4v4 identique après extraction),
garde-rail vert, `npx tsc -b --force` vert.

### Étape 2 — Le gabarit compact

Fichiers : `ui/ReplayPlayerCard.tsx`, `ui/ReplayVitality.tsx` (`VitalityBar` via
`gaugeShieldPx` / `gaugeHealthPx`, `EliminatedBox` compact), `ui/ReplayObjectiveMark.tsx`
(`watermarkPx`).

- [x] 2.1 Trois lignes aux cotes du tableau, `leading-[14px]` sur le nom, corps fixe 31 px.
      Fait (2026-09-07) : `ui/ReplayPlayerCard.tsx` porte une table FERMÉE `TILE_LAYOUT`
      keyed par `CardGabarit['bodyPx']` (35 = les classes d'aujourd'hui à l'octet, 31 = la
      tuile `rounded-md border px-1.5 py-1.5`, ligne 1 `relative flex leading-[14px]`, corps
      `mt-[3px] h-[31px]`, ligne 2 `flex h-[12px] items-center gap-[5px]`, ligne 3
      `mt-[3px] flex h-[16px] …`) ; `BODY_CLASS = { 35: 'h-[35px]', 31: 'h-[31px]' }` et
      `NAME_CLASS = { 11.5: 'text-[11.5px]' }` — littéraux en clair, jamais interpolés. Pour
      que ces tables soient closes À LA COMPILATION, `cardGabarit.ts` type `bodyPx: 35 | 31`,
      `namePx: 11.5`, `countCellW: 15 | 10` (unions de littéraux ; les valeurs des deux
      gabarits sont inchangées, `cardGabarit.test` intact).
- [x] 2.2 Ligne 1 : nom seul, pleine largeur, `title` = nom complet ; `fx.title` atteignable
      sur cette ligne (D6).
      Fait : le triplet quitte la ligne du nom (`countersOnNameLine: false` dans la table) ; le
      nom garde `title={name}` mais N'EST PAS en `flex-1` sur le corps de 31 — le reste de la
      ligne remonte à l'infobulle de la tuile (test (b) : `closest('[title]')` de la ligne du
      nom = la tuile, « écran occultant » sur Charlie).
- [x] 2.3 Ligne 2 : jauges `flex-1` + triplet ; règle « document sans `sh`/`hp` = aucune
      barre » conservée.
      Fait : `gauges: 'flex min-w-0 flex-1 flex-col gap-[2px]'`, jauges 4 / 2 px par
      `gaugeShieldPx` / `gaugeHealthPx` (déjà branchées à l'étape 1), `VitalityBar` rend
      toujours `null` sur une lecture nulle (test (g) : 0 jauge sans `sh`/`hp`, 100 % sur une
      vie sans mesure, 60 % sur une mesure).
- [x] 2.4 Ligne 3 : arme 48, grenade 14, capacité 16.
      Fait pour la LIGNE et ses cotes : conteneur `h-[16px] gap-x-[5px]`, cellule d'arme 48
      (`weaponCellW`), vignettes 14 / 16 (`iconGrenadePx` / `iconAbilityPx`) — tests (d) et (f).
      La COMPOSITION de la ligne (une seule cellule d'arme, pas de cellule de munitions ni de
      boîte de stock) est, par construction du plan, l'objet de 3.1 / 3.2 : au gate 2 la rangée
      émet encore les cellules du gabarit normal aux largeurs du compact.
- [x] 2.5 Encadré « Éliminé » compact sur deux lignes ; cas « hors film » selon D4 ; `title`
      « Réapparition dans » conservé ; aucun `progressbar`.
      Fait : `ui/ReplayVitality.tsx` — `EliminatedBox` reçoit `bodyPx` + `counters?` et
      dispatche par une table fermée `ELIMINATED_BY_BODY = { 35: EliminatedRow, 31:
      EliminatedStack }` ; `EliminatedRow` = le DOM d'aujourd'hui (fixture 4v4 inchangée) ;
      `EliminatedStack` = `px-[5px] py-[2px]`, ligne 1 `h-[13px]` (« ÉLIMINÉ » 7 px
      `tracking-[.15em]` + triplet), ligne 2 `h-[14px] justify-end` (décompte mono 13 px, `title`
      « Réapparition dans »). Hors film : ligne 2 vide, `goneValue` en `title` de l'ENCADRÉ
      (test (h)). Aucun `progressbar` (asserté).
- [x] 2.6 `boltCount` = 2 et `watermarkPx` = 34 en compacte ; croix inchangées (I6).
      Fait : câblage de l'étape 1 (`gabarit.boltCount`, `gabarit.watermarkPx`) exercé par les
      tests (i) 2 éclairs / 3 croix et (k) `<svg width="34">`, aucun `width="46"` ; les tests
      existants restent à 3 / 3 (`T_EXIST` 70/70).
- [x] 2.7 Levée du `it.todo` à 6 sièges de 0.5.
      Fait : `ReplayTeams.dom4v4.test.tsx` — `documentSixParCamp()` (le document riche + 4
      sièges vivants équipés), `tableau(4 | 6)`, fixture `__fixtures__/replayTeams.6v6.html`
      (65 427 octets, md5 `4dc466cc773622a09671d07bdb9efe8a`) prise AVANT le premier code de
      l'étape 2, 12 corps `h-[35px]`, aucune classe `auto-fill`.

**Gate 2** : `ui/ReplayPlayerCard.compact.test.tsx` (roster 12v12 et 7v7), liste fermée
d'`it` : (a) hauteur ET classes des rangées identiques vivant/mort ; (b) nom sur la ligne
entière, `title` complet ; (c) triplet présent en vie et en mort, `?` sans fond quand un
compteur manque, fond FDA à trois paliers ; (d) cellules rendues vides quand la donnée manque,
aucun `role=img` parasite ; (e) grenades non lues → rien ; (f) capacité hors table → glyphe +
rang dans `title` ; (g) aucune barre sans `sh`/`hp`, 100 % sans mesure ; (h) « hors film » ;
(i) 2 éclairs / 3 croix en compacte, tests existants toujours à 3 / 3 ; (j) fiche morte sans
cadre ni zone ; (k) filigrane présent à 34 px. Plus `T_EXIST`, `T_DIFF`, `T_DOM`.

### Étape 3 — Les sous-composants en gabarit compact

Fichiers : `ui/ReplayWeaponsRow.tsx`, `ui/ReplayInventoryRow.tsx`, `ui/ReplayCountersBadge.tsx`,
`ui/ReplayAbilityCell.tsx`, `i18n/i18n.ts`, `styles/globals.css` (`--replay-dx`).

- [x] 3.1 `ReplayWeaponsRow` : `weaponCells: 1` → une cellule (48), arme rangée nommée dans
      le `title` (`weaponStowedFmt`) ; loadout non lu → cellule vide + `title` « armes non
      lues » ; badge de lancer sur 48 px ; échange animé selon I6.
      Fait (2026-09-07) : `seule = gabarit.weaponCells === 1` → `cells = [weapons[0]]`, la
      vignette n'a PAS d'infobulle propre et c'est la RANGÉE (dont la boîte est alors la
      cellule) qui porte `handCellTitle` = `weaponStowedFmt(nom de weapons[1])` quand le
      sélecteur est lu (sinon le nom seul, sans affirmer « rangée »), puis `handHint` (3.2),
      puis l'âge. Loadout non lu : une seule `EmptyWeaponCell` de 48 sous
      `loadoutUnread · handHint`. Échange : `swap.dx = cellW` → la vignette pose
      `--replay-dx: 48px` (type `ChipStyle = CSSProperties & { '--replay-dx'?: string }`) et
      seule `replay-wswap-l` existe (k = 0) ; `globals.css` documente le défaut 46 px et le
      cas à une cellule. Badge de lancer : `absolute inset-0` dans le wrapper `relative` de la
      cellule → 48 px par construction ; NON exercé par un test (aucune fixture de lancer dans
      le test compact — dit au journal). Fiche normale : `dx: null`, `hint` inchangé, DOM 4v4
      identique.
- [x] 3.2 `ReplayInventoryRow` : `showAmmo: false`, `showGrenadeStock: false` → grenade
      sélectionnée + capacité ; munitions dans le `title` de l'arme (`ammoHintFmt`, trois cas
      + rien en D=2) ; stock via `grenadeBoxHint` ; marques de D5 dans les `title`, avec leurs
      âges ; distinction LU / DÉDUITE de la sélection dans le `title`.
      Fait : la cellule de munitions n'est rendue que si `showAmmo` ; la boîte de stock est
      remplacée par `GrenadeSelectedCell` (14 px, vignette du seul type qui partira, encre
      `warning`, `title` = provenance LUE / DÉDUITE · « sél. ? » si indéterminé (cellule VIDE)
      · `grenadeBoxHint` réemployé tel quel) ; `InventoryEmptyMark` n'est rendue que si
      `showInventoryMarks` (nouveau booléen du gabarit — normal `true`, compact `false` —
      ajouté à `cardGabarit.test`). Les munitions et les marques de la MAIN sont composées par
      la FICHE dans `model/handCellHint.ts` (pur : `isChargeWeapon` — `CHARGE_FX` y déménage,
      une seule définition —, `ammoOfHand` = les branches exactes de `AmmoCell`,
      `handCellHint` = munitions ou « dégainée ? », puis « Mort / Inventaire indisponible —
      inventoryEmptyHint » avec ses DEUX âges) et confiées à `ReplayWeaponsRow.handHint`.
      Coût : une lecture `inventoryAt` de plus par fiche COMPACTE (jamais en normal) — à
      mesurer au 5.2. `ReplayAbilityCell` : charges (`×N` + âge, ou « plein » + provenance)
      dans le `title` de la cellule quand `!showInventoryMarks`, via `chargeTitle` (une seule
      composition, partagée avec la marque en texte).
- [x] 3.3 `ReplayCountersBadge` : `showScore: false`, `countCellW: 10`, score dans le `title`
      (`playerScoreLiveFmt`) ; la doctrine « cellule de score toujours rendue » ne vaut que
      pour le gabarit normal — dit dans le commentaire.
      Fait : cellule de score conditionnée à `showScore` ; `title` du triplet =
      `fdaTooltip · playerScoreLiveFmt(live.score)` sur un joueur publié seulement ;
      typographie par table fermée `TRIPLET_TYPO` keyed par `countCellW` (15 : classes
      d'aujourd'hui à l'octet, gap et `px-1` compris ; 10 : `text-[9px] leading-none`,
      `gap-[2px]`, `px-[3px]` → ≈ 54 px). Commentaire d'en-tête réécrit (la doctrine de la
      cellule vaut « là où la cellule existe »).
- [x] 3.4 Entrées i18n FR + EN (I9).
      Fait : `i18nContract.ts` — type `AmmoHint` (count / charge / full) + `weaponStowedFmt`,
      `playerScoreLiveFmt`, `ammoHintFmt` ; `i18n.ts` FR (« Arme rangée : X », « Score
      personnel à l'instant lu : N », « Munitions : m / r » / « Charge restante : p % » /
      « Munitions pleines ») et EN (« Stowed weapon: X », « Personal score at the moment being
      played: N », « Ammo: m / r » / « Charge left: p % » / « Ammo full ») — parité tenue par
      `Record<ReplayLocale, ReplayText>` (`tsc` exit 0), cas EN asserté dans le test compact.

**Gate 3** : `ui/ReplayPlayerCard.compact.test.tsx` étendu — chaque report est présent dans un
`title` avec sa valeur (arme rangée, munitions 3 cas, stock, score, marques D5 avec âges), et
ABSENT en D=2 ; `fx.title` atteignable ; estompage par cellule (`opacity` sur la cellule, pas
sur la rangée), âge négatif « dans X s ». `npm run lint`, `npm run lint:colors`, `T_EXIST`,
`T_DIFF`, `T_DOM`.

### Étape 4 — Cas limites

Fichiers : `ui/ReplayTeams.tsx`, tests.

Les cinq cas de COLONNE vivent dans un nouveau fichier `ui/ReplayTeams.density.test.tsx`
(un cas = un `it`, document NU : une vie par siège, avec ou sans vitalité, horloge établie par
`originMs: 0` / `frameIntervalMs: 100`) ; le sixième, un cas de TUILE, est le `it` (l) de
`ui/ReplayPlayerCard.compact.test.tsx`. Exécuté le 2026-09-07.

- [x] 4.1 Match `BTB` à effectifs inégaux (12v9 après départs) : un seul gabarit sur toute
      la colonne, les deux camps en compacte.
      Fait : 21 corps de 31, 0 de 35 ; deux conteneurs distincts de 12 et 9 enfants, tous deux
      à la chaîne `SEATS_GRID_CLASS` ; 21 tuiles `rounded-md border`.
- [x] 4.2 Sièges relayés en BTB : le compte ne bouge pas, la fiche suit l'occupant.
      Fait : `Nord12` (`left_in_progress`, 20:00:10Z) relayé par `Zulu` (`joined_in_progress`,
      même seconde, camp `t0`) à l'image 100 ; à l'image 50 : 24 corps, `Nord12` présent, `Zulu`
      absent ; `rerender` à 150 : 24 corps, `Zulu` présent, `Nord12` absent — la vie de `Nord12`
      reste ouverte jusqu'à 300, donc sans appariement la colonne compterait 25. Grille
      compacte aux deux images.
- [x] 4.3 Chaîne exacte de la classe de grille assertée (technique `rosterHeight.guard`).
      Fait : au SOURCE (commentaires ôtés — l'en-tête de la constante cite lui-même `calc(`) :
      `'grid min-h-0 flex-1 auto-rows-max grid-cols-[repeat(auto-fill,minmax(115px,1fr))] gap-1 overflow-y-auto'`
      et `'flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto'` présents à l'octet, aucun
      `calc(`, aucun espace dans la valeur arbitraire, un seul `grid-cols-[` ; au DOM : `BTB` →
      grille sur les deux camps, sans catégorie → colonne simple sur les deux.
- [x] 4.4 Titre sans décodage film (document sans `sh`/`hp`, sans inventaire) en compacte :
      cellules vides, aucune barre, aucun zéro.
      Fait : 24 corps de 31 ; 0 `Bouclier` / `Santé`, 0 `role=img`, 0 `svg` ; tableau à
      compteurs NON NULS (2/1/3) pour qu'un « 0 » soit un zéro inventé → aucun `0`, aucun
      `×` / `%` / `m / r`, aucune infobulle de grenade / inventaire / munitions ; 24 cellules
      d'arme vides sous « armes non lues sur cette vie » ; les largeurs fixes du DOM sont
      EXACTEMENT 24 × [48, 14, 16] px, toutes à `textContent` vide (règle « cellule fixe,
      jamais un décalage ») ; 24 triplets « 2/1/3 », aucun « ? ».
- [x] 4.5 En-tête sans catégorie sur un roster de 24 : densité normale, DOM de la colonne
      identique à celui d'aujourd'hui (aucun repli sur les effectifs — D1).
      Fait : cinq en-têtes qui ne disent pas `BTB` (`undefined`, `null`, `start_time` seul,
      `Arena`, `''`) rendent le MÊME `innerHTML` à l'octet ; 24 corps de 35, 0 de 31, colonne
      simple sur les deux camps, 24 tuiles `rounded-lg border px-2.5 py-2`, ni `auto-fill` ni
      `rounded-md` ; contre-épreuve : le même document et le même tableau sous `BTB` diffèrent
      et portent `auto-fill`.
- [x] 4.6 (ajouté par le superviseur, 2026-09-07) Les couches d'effets de la tuile
      (`replay-card-fx`, `ZoneFxOverlay` et ses enfants capteur / translocation, filigrane)
      gardaient `rounded-lg` (8 px) sur une tuile compacte en `rounded-md` (6 px).
      Fait : `TILE_LAYOUT.layerRadius` (`'rounded-lg'` pour 35 — la chaîne d'aujourd'hui à
      l'octet —, `'rounded-md'` pour 31), littéraux en clair ; `ZoneFxOverlay` et
      `ReplayObjectiveMark` reçoivent `radiusClass` ; commentaires de `ReplayPlayerCard` et de
      `playerCardFx.ts` (« `inset-0 rounded-lg` ») réécrits. Test (l) : aucun `rounded-lg` dans
      la colonne BTB, couche sous le contenu (Charlie), incrustation (Charlie, Delta) et
      filigrane (India) en `rounded-md` ; au source, plus aucun `className="…rounded-lg"` dans
      les deux fichiers, capteur et translocation composés sur `${radiusClass}`. Fixture 4v4
      md5 `a650f6b57ba38463ddeabf4ef7694817` inchangé, 6v6 `4dc466cc…` inchangé.

**Gate 4** : les six cas sont des `it` verts ; `T_EXIST`, `T_DIFF`, `T_DOM`. **Passé le
2026-09-07** (journal).

### Étape 5 — Gates de livraison, perf, revue

- [x] 5.1 `T_ALL` (codes de sortie vérifiés).
      Fait (2026-09-07, avant-plan) : `rm node_modules/.tmp` + `npx tsc -b --force` exit 0 ;
      `npm run lint` exit 0 (27 avertissements préexistants hors périmètre, inchangés depuis le
      gate 3 ; `eslint --max-warnings 0` sur les six fichiers touchés à l'étape 4-5 : exit 0) ;
      `npm run lint:colors` 0 violation ; `npm run test:run` : 618 fichiers verts + 1 sauté (le
      perf test, `REPLAY_PERF` absent), **6 440 tests verts, 16 sautés, exit 0**, 88 s. `tsc`
      rejoué après la retouche du perf test (5.2) : exit 0.
- [x] 5.2 Perf APRÈS, volet JS, même protocole que 0.4 ; comparaison aux seuils ; si
      dégradation, `inventoryAt` calculé une fois (I4) puis nouvelle mesure.
      Fait (2026-09-07) : mesuré et consigné au journal, trois passes. **Recadrage utilisateur du
      même jour : le chiffre est une information, pas un gate — aucune optimisation, la
      boucle « si dépassement » est retirée.** Écart au protocole, dit : le perf test montait
      `<ReplayTeams>` SANS en-tête, donc en gabarit NORMAL sur le témoin BTB aussi (il date de
      l'étape 0, avant `mode_category`) — la tuile compacte et la lecture `inventoryAt` de
      `handCellHint` n'y passaient jamais. Ajout ADDITIF d'une troisième ligne (le témoin BTB
      sous `mode_category: 'BTB'`) ; les deux lignes de 0.4 restent mesurées telles quelles.
- [ ] 5.3 Gate visuel (utilisateur, son gate habituel) : ouvrir le rejeu du témoin BTB
      `4f77afc1` sur le dev local — 24 fiches visibles, gamertags lisibles, lecture fluide à
      1× — et le rejeu du témoin 4v4 `000d5950` — rien n'a changé. Le compact qui apparaît sur
      le BTB prouve au passage que la catégorie `BTB` arrive bien à la page.
- [~] 5.4 Revue adversariale du diff — **retirée le 2026-09-07** (décision utilisateur : lot
      d'interface, hors des lots à risque que CLAUDE.md réserve à cette revue). Couvert par la
      revue du plan à quatre angles et par la relecture de chaque diff par le superviseur, gates
      rejoués à chaque étape.
- [ ] 5.5 Entrée `.ai/thought_log.md` (date, titre, statut, décision, résultats, suite).
- [ ] 5.6 `delivery-checklist` déroulée ; DEMANDER l'autorisation de commit final et de
      merge dans `feat/v75` ; après poussée : `gh run list --branch feat/fiches-compactes
      --limit 1` puis `gh run watch <id> --exit-status` — tout rouge se répare, même
      préexistant.

**Gate 5** : tous les gates verts, perf JS dans les seuils, gate visuel validé par
l'utilisateur, CI de branche verte.

---

## Reprise de session

1. `git worktree list` → `LevelUp-wt-fiches-compactes` ; `git branch --show-current` →
   `feat/fiches-compactes`.
2. `git log --oneline -10` : un commit par étape `feat(fiches-compactes/etape-N)`.
3. Les cases de ce fichier font foi ; le dernier gate passé est écrit dans le journal
   d'exécution ci-dessous. Reprendre à la première case vide de la première étape non close.

## Journal d'exécution

_(une ligne par gate : date, gate, commandes, résultat chiffré)_

- **2026-09-06 — base AVANT toute modification** : `T_EXIST` = 70 tests verts, exit 0, 17,3 s.
- **2026-09-06 — 0.4, mesure JS AVANT** (`REPLAY_PERF=1 REPLAY_PERF_DIR=<principal>/data/cache/replays/halo_infinite
  npx vitest run src/features/match-replay/ui/ReplayTeams.perf.test.tsx`, exit 0, 28,5 s ;
  jsdom, Node 24, 5 répétitions, médiane des médianes, `actualDuration` du `<Profiler>` sur
  100 rerender après 20 d'échauffement, cadence 1,5 image) :

  | Témoin | Fiches | Base | Colonne p50 | Colonne p95 | Colonne total/100 | Modèle seul p50 | Modèle p95 |
  |---|---|---|---|---|---|---|---|
  | BTB `4f77afc1` (658 poses) | 25 | 5 000 | **2,979 ms** | **5,314 ms** | 378 ms | **0,168 ms** | 0,200 ms |
  | 4v4 `000d5950` (295 poses) | 8 | 2 243 (45 %) | **1,658 ms** | **11,212 ms** | 393 ms | **0,088 ms** | 0,130 ms |

  p50 de la colonne par répétition : BTB 3,11 / 2,89 / 2,99 / 2,98 / 2,70 ; 4v4 1,90 / 1,56 /
  1,66 / 1,60 / 1,74. Le p95 du 4v4 (11,2 ms pour un p50 de 1,7) et son total supérieur au BTB
  sont des pointes de jsdom/GC, pas du modèle (modèle p95 0,13 ms). Le modèle seul mesure
  0,17 ms pour 25 fiches : l'estimation « 1-2 ms » de I4 est dix fois trop haute en jsdom.
  Seuils du 5.2 : BTB colonne p50 APRÈS ≤ 3,28 ms ; 4v4 colonne p50 APRÈS dans 1,58-1,74 ms.
  Le modèle seul est mesuré à l'étape 0 par une RECOPIE des lectures de `PlayerCard` dans le
  perf test ; l'étape 1.3 la remplace par l'import de `model/playerCardReadings.ts`.
- **2026-09-06 — gate 0** : `T_DOM` exit 0 (1 test vert + 1 todo), fixture
  `ui/__fixtures__/replayTeams.4v4.html` écrite par `toMatchFileSnapshot` AVANT toute
  modification de source (première passe 18,4 s, seconde passe 3,1 s identique) ; couverture
  vérifiée dans le HTML : « Éliminé » + « Réapparition dans » (Hotel), « Mort » (Echo),
  « sél. ? » (Charlie), « dégainée ? » (Charlie), « 75% » (Foxtrot), « Munitions pleines »
  (Bravo, ajouté par une reprise de la fixture — toujours avant tout changement de source),
  « 350 » (score Alpha), « Grappin » (icône) et « capacité non identifiée (rang 9) » (Golf),
  3 `replay-zone-bolt` + nuage (Charlie), verre `backdrop-filter` (Delta), filigrane
  `<svg width="46">` (Foxtrot), bandeaux « Équipe Eagle » / « Équipe Cobra » avec
  `var(--ac-team-ally)` ; 0 hex. Lint des trois nouveaux fichiers : eslint exit 0 ;
  `Remove node_modules/.tmp` + `npx tsc -b --force` exit 0. Suite complète passée
  incidemment (commande `-u` mal ciblée) : 6 394 verts, 16 sautés, 1 snapshot écrit (le mien).
- **2026-09-07 — gate 1** (exécuté en avant-plan, codes de sortie lus) : `T_EXIST` 70/70,
  exit 0 ; `T_DIFF` vide (0 ligne) ; `T_DOM` 1 vert + 1 todo, exit 0, fixture md5 inchangé
  `a650f6b57ba38463ddeabf4ef7694817` — le DOM 4v4 est identique après l'extraction ;
  garde-rail `cardGabarit.guard` + `cardGabarit.test` + `cardDensity.test` : 3 fichiers,
  20/20, exit 0 ; `rm node_modules/.tmp` + `npx tsc -b --force` exit 0 ; eslint sur les 14
  fichiers touchés exit 0 ; `npm run lint:colors` 0 violation. Contrôle du perf test sur
  l'extraction (`REPLAY_PERF=1`, exit 0) : BTB colonne p50 2,831 / p95 4,401 ms, modèle p50
  0,166 ms ; 4v4 colonne p50 1,591 / p95 10,955 ms, modèle p50 0,091 ms — dans le bruit de
  la mesure AVANT (l'extraction ne coûte rien ; ce n'est PAS la mesure APRÈS du 5.2).
  Aucun `git commit` : l'arbre est laissé modifié pour l'autorisation de l'utilisateur
  (7 fichiers modifiés, 11 nouveaux dont la fixture).
- **2026-09-07 — gate 2** (avant-plan, codes de sortie lus) : fixture 6v6 prise AVANT tout code
  de l'étape 2 (`T_DOM` 2 verts, snapshot écrit, 4v4 md5 inchangé `a650f6b5…`) ; puis
  `ui/ReplayPlayerCard.compact.test.tsx` (a)–(k) : 11/11, exit 0 (un premier rouge sur (a) était
  une erreur du TEST — la trace de Kilo posée hors roster en 7v7 fabriquait un 15e siège — corrigée
  dans le test, pas dans le code) ; `T_EXIST` 70/70 ; `T_DIFF` 0 ligne ; `T_DOM` 2/2 (4v4 md5
  `a650f6b57ba38463ddeabf4ef7694817` inchangé après la tuile compacte, 6v6
  `4dc466cc773622a09671d07bdb9efe8a`) ; garde-rail `cardGabarit.guard` 6/6, `cardGabarit.test`
  3/3, `cardDensity.test` 11/11 (6 fichiers, 103 tests, exit 0) ; `rm node_modules/.tmp` +
  `npx tsc -b --force` exit 0 (après une annotation `string[]` dans le test) ; eslint sur les 5
  fichiers touchés exit 0 ; `npm run lint:colors` 0 violation. Aucun `git commit`.
- **2026-09-07 — gate 3** (avant-plan, codes de sortie lus) : `ui/ReplayPlayerCard.compact.test.tsx`
  étendu de 8 `it` (une seule cellule de 48 + arme rangée nommée + loadout non lu + ligne 3 =
  48 / 14 / 16 ; munitions 3 cas + RIEN en D=2 ; stock + LUE / DÉDUITE / « sél. ? » sur 14 px ;
  score dans le `title`, aucune cellule de score ; marques D5 avec âges — « Mort » 0,8 s /
  1,7 s, « dégainée ? », charges « ×2 · 1,2 s » et « plein » ; `fx.title` atteignable malgré
  les infobulles de cellule ; estompage par cellule + âge négatif « dans 0.8 s » ; EN) :
  19/19, exit 0 (un premier rouge sur la ligne 3 était une erreur du TEST — il comptait les
  largeurs des éclairs de l'incrustation, hors corps — corrigée dans le test) ; `T_EXIST`
  70/70 ; `T_DIFF` 0 ligne ; `T_DOM` 2/2, md5 4v4 `a650f6b57ba38463ddeabf4ef7694817` inchangé,
  6v6 `4dc466cc773622a09671d07bdb9efe8a` inchangé ; `cardGabarit.guard` 6/6, `cardGabarit.test`
  3/3 (+ `showInventoryMarks`), `cardDensity.test` 11/11 — 6 fichiers, **111 tests, exit 0** ;
  `rm node_modules/.tmp` + `npx tsc -b --force` exit 0 (après un typage `ReplayInventory[]`
  dans le test) ; `npm run lint` exit 0 (27 avertissements PRÉEXISTANTS hors périmètre —
  `react-hooks/incompatible-library` sur TanStack Table etc. ; `eslint --max-warnings 0` sur
  les 13 fichiers touchés : exit 0) ; `npm run lint:colors` 0 violation. Aucun `git commit`,
  arbre laissé modifié : 13 fichiers modifiés, 3 nouveaux (`model/handCellHint.ts`,
  `ui/ReplayPlayerCard.compact.test.tsx`, `ui/__fixtures__/replayTeams.6v6.html`). NON
  exercé par un test : le badge de lancer sur la cellule de 48 (aucune fixture de lancer dans
  le test compact ; la géométrie est `absolute inset-0` dans le wrapper de la cellule) et le
  rendu réel de l'animation d'échange à une cellule (`--replay-dx` posé, asserté nulle part —
  le gate visuel 5.3 les verra). L'étape 4 n'est PAS commencée.
- **2026-09-07 — gate 4** (avant-plan, codes de sortie lus) : `ui/ReplayTeams.density.test.tsx`
  5/5 (deux premiers rouges = erreurs du TEST, corrigées dans le test : le commentaire de
  `ReplayTeams.tsx` cite `calc(` — commentaires ôtés avant le contrôle ; la tuile compacte rend
  ses TROIS cellules fixes vides, 48/14/16, pas la seule cellule d'arme — asserté tel quel ; un
  troisième sur le `title` du triplet, qui porte la FDA quand les trois compteurs sont lus) ;
  `ui/ReplayPlayerCard.compact.test.tsx` 20/20 (19 + le (l) du 4.6) ; `T_EXIST` 70/70 ; `T_DIFF`
  0 ligne ; `T_DOM` 2/2, md5 4v4 `a650f6b57ba38463ddeabf4ef7694817` et 6v6
  `4dc466cc773622a09671d07bdb9efe8a` inchangés après le rayon des couches ; `cardGabarit.guard`
  6/6, `cardGabarit.test` 3/3, `cardDensity.test` 11/11 — **7 fichiers, 117 tests, exit 0**.
  Fichiers touchés : `ui/ReplayPlayerCard.tsx` (`layerRadius`), `ui/ReplayObjectiveMark.tsx`
  (`radiusClass`), `model/playerCardFx.ts` (commentaire), les deux tests. Aucun `git commit`.
- **2026-09-07 — 5.1** : voir l'item (tsc 0 / lint 0 / lint:colors 0 / test:run 6 440 verts, exit 0).
- **2026-09-07 — 5.2, mesure JS APRÈS** (`REPLAY_PERF=1 REPLAY_PERF_DIR=<principal>/data/cache/replays/halo_infinite
  npx vitest run src/features/match-replay/ui/ReplayTeams.perf.test.tsx`, hors sandbox, exit 0,
  3 passes de ~6 s ; même protocole que 0.4 — jsdom, 5 répétitions, médiane des médianes,
  100 rerender après 20 d'échauffement, cadence 1,5 image — plus la ligne compacte ajoutée) :

  | Témoin | Gabarit | Colonne p50 (3 passes) | Colonne p95 | Modèle seul p50 | Modèle p95 |
  |---|---|---|---|---|---|
  | BTB `4f77afc1` (25 fiches, base 5 000) | normal (sans en-tête, = 0.4) | **1,357 / 1,356 / 1,351 ms** | 1,943 / 1,893 / 1,855 | 0,087 / 0,086 / 0,084 | 0,097 |
  | BTB `4f77afc1` | **compacte (`mode_category: 'BTB'`)** | **1,308 / 1,299 / 1,305 ms** | 1,849 / 1,508 / 1,541 | 0,091 / 0,086 / 0,088 | 0,098 |
  | 4v4 `000d5950` (8 fiches, base 2 243) | normal (= 0.4) | **0,781 / 0,725 / 0,747 ms** | 1,336 / 1,028 / 1,259 | 0,050 / 0,049 / 0,049 | 0,065 |

  Face aux seuils : BTB colonne p50 ≤ 3,28 ms → 1,31 ms (compacte) et 1,36 ms (normale),
  tenu ; 4v4 colonne p50 dans 1,58-1,74 ms → **0,73-0,78 ms, HORS fenêtre par le bas**. Cette
  sortie n'est pas un effet du lot : le MODÈLE SEUL, dont le code n'a pas changé depuis
  l'extraction (0,168 ms AVANT, 0,166 au contrôle du gate 1), mesure lui aussi 0,087 ms, soit
  −48 %, et le 4v4 normal — qui rend, nœud pour nœud, la fixture prise avant le lot — baisse
  dans la même proportion (−55 %). L'état de la machine entre les deux sessions (charge, pas
  d'horloge) explique la translation ; la comparaison ABSOLUE à l'AVANT n'est donc pas
  interprétable, et c'est la comparaison INTRA-passe qui informe : sur le même témoin, dans la
  même passe, **la tuile compacte coûte 3,5 % de MOINS que la normale** (1,30 contre 1,35 ms,
  stable sur les trois passes ; p95 1,5-1,8 contre 1,9) — la lecture `inventoryAt` de plus par
  fiche compacte (3.2) est absorbée par les cellules en moins (munitions, boîte de stock, score,
  seconde arme). Aucune optimisation appliquée (recadrage utilisateur : le chiffre est une
  information, pas un gate). Le perf test reste sauté sans `REPLAY_PERF` (1 fichier sauté dans
  `test:run`). `tsc` et `eslint --max-warnings 0` sur le perf test modifié : exit 0.
  **Les items 5.3 à 5.6 ne sont pas commencés** (superviseur et utilisateur).

## Ce que ce plan NE fait pas

- Aucune modification d'un match qui n'est pas `BTB` (DOM identique à la fixture 0.5).
- Aucune heuristique d'effectif : ni sièges, ni joueurs, ni lignes de tableau (D1).
- Aucune modification du plafond de hauteur ni de `replay.tsx`.
- Aucune modification du fil des éliminations, du bandeau d'équipe, de la carte, de
  `lib/replay/`.
- Aucun traitement du FFA (D3) ni d'un éventuel groupe sans camp (D2).
- Aucune cuisson d'artefact, aucun réglage utilisateur, aucun drapeau.

## Découvertes hors périmètre

- `model/equipmentZones.ts` : commentaire de charge « dizaines de poses par film, huit
  fiches » périmé — 658 à 892 poses et 25 fiches sur les artefacts BTB.
- FFA : N groupes d'un siège → `repeat(N, 1fr)`, colonnes de ~50 px (D3, inchangé).
- Bot absent du tableau → sans camp → troisième colonne de 153 px où la fiche normale
  déborde : défaut de donnée (tout joueur a une équipe, D2), non traité par ce lot.
- `PlayerCard` (149 L) et `ReplayInventoryRow` (150 L) dépassent la règle « fonction ≤ 80 L »,
  non lintée (pas de `max-lines-per-function`) — l'extraction 1.3 réduit la première.
- `ReplayTeams.tsx:161-162` (« hors rangée, rien ne défile ») contredit `replay.tsx` (borné à
  60 vh sous `xl`).
- La « fiche actuelle 235 × 74 » de la maquette mesure ≈ 77 px sur le DOM (interligne 1,5 non
  bridé sur le nom).
- (0.2) La vue match locale est gardée par l'ownership (ADR 0029) : sans session, 403
  `player_forbidden` sur les 9 profils — un agent ne peut pas lire `mode_category` par l'API
  sans cookie ; établi sur pièces Go à la place (cf. 0.2). Non traité.
- (0.2) `groups.length` n'est pas lisible dans un artefact de rejeu : le film écrit
  `team: -1` sur toutes les traces (253/253 et 99/99 sur les témoins), le camp vient
  uniquement du tableau. Le perf test doit donc INVENTER des camps (alternance t0/t1 depuis le
  roster) — sa mesure de colonne n'exerce pas les vrais effectifs par camp (13/12 au lieu du
  réel). Non traité.
- (0.4) L'estimation « 1-2 ms de modèle par publication en BTB » de I4 est dix fois trop
  haute en jsdom : 0,17 ms mesuré pour 25 fiches. Le coût JS est dans la réconciliation
  (≈ 3 ms), pas dans les lectures. Le risque réel reste la peinture (volet navigateur).
- (0.4) Le témoin 4v4 `000d5950` n'a que 4 985 images : le protocole « base 5 000 » ne
  s'applique pas tel quel (base 45 %, dite dans le résultat). Un témoin 4v4 plus long
  permettrait la même base sur les deux.
- (1.1) `layers/replayDraw.ts:256` déclare un `GRENADE_ICON_PX = 18` homonyme de l'ancienne
  constante de la rangée d'inventaire : c'est la vignette du CANVAS (18 px), pas celle de la
  fiche (14 px) — pas une copie, mais un nom qui prête à confusion. Hors périmètre, non traité.
- (1.1) `bodyPx` / `namePx` ne peuvent pas s'appliquer en `style` sans changer le DOM 4v4 ni
  en classe interpolée (Tailwind) : l'étape 2 devra passer par une table fermée
  `{ 35: 'h-[35px]', 31: 'h-[31px]' }` — noté pour l'exécuteur de l'étape 2, pas une
  découverte à traiter.
- (0.5) `npx vitest run -u <fichier>` lance TOUTE la suite (le `-u` avale le filtre) : à
  n'utiliser qu'avec `--update` explicite ou en supprimant la fixture avant la passe. Suite
  complète passée à cette occasion : 6 394 verts, 16 sautés, 1 seul snapshot écrit.
- (2.7 / gate 2) Une TRACE dont le joueur n'est ni au roster ni au tableau fabrique un siège
  de plus (`buildPlayers`) — observé en 7v7 quand la vie de Kilo restait posée hors effectif :
  le siège apparaît sans camp. Même famille que le cas « bot absent du tableau » de D2 ;
  artefact de fixture ici, non traité.
- (3.1) `--replay-dx` vaut 46 px par défaut dans `globals.css` (valeur du POC) alors que la
  distance réelle entre les deux cellules d'arme de la fiche normale est 40 + 5 = 45 px. Un
  pixel d'écart sur la course de l'animation, préexistant, hors périmètre (le lot ne touche pas
  le gabarit normal) — non traité.
- (3.2) `handCellHint` refait une lecture `inventoryAt` par fiche compacte (la rangée
  d'inventaire fait déjà la sienne, et `equippedWeapons` une troisième) : c'est le « gain
  gratuit » de I4 (calculer `inventoryAt` une fois dans la fiche) qui devient plus tentant —
  réservé au 5.2 si le volet JS dégrade, conformément au plan.
- (3.1) Le test compact n'exerce pas le badge de lancer (`GrenadeThrowBadge`) : aucune fixture
  de lancer (`grenadeThrows`) n'a été bâtie — la couverture existante du badge vit dans les
  tests du gabarit normal. À couvrir si le gate visuel 5.3 montre un défaut sur 48 px.
- (4.6) Le test (l) n'exerce pas au DOM l'anneau du capteur adverse ni le fourreau de
  translocation en compacte (le document du test compact ne pose ni capteur ni translocation) :
  leur rayon est asserté au SOURCE (composition sur `${radiusClass}`). Le gate visuel 5.3 les
  verra s'ils surviennent sur le témoin.
- (5.2) L'en-tête de `ui/ReplayTeams.perf.test.tsx` dit encore « la peinture et les animations
  CSS … sont le volet navigateur, remis à l'utilisateur (DevTools) » : ce volet a été RETIRÉ
  le 2026-09-07 (I4, décision utilisateur). Doc périmée d'une phrase, non traitée (hors item).
- (5.2) Les chiffres AVANT de 0.4 (BTB 2,98 ms, 4v4 1,66 ms, modèle 0,17 ms) ont été pris sur
  une machine dans un autre état : la même mesure, code du modèle inchangé, rend aujourd'hui
  moitié moins sur toutes les lignes. Une fenêtre absolue (« 4v4 dans 1,58-1,74 ms ») ne tient
  pas d'une session à l'autre ; un seuil RELATIF intra-passe (compacte / normale sur le même
  témoin) serait le seul robuste. Non traité — noté pour la revue 5.4.
- (4.3) Le commentaire de tête des constantes de grille de `ReplayTeams.tsx` cite `calc(` en
  prose : un garde qui grep le source brut (comme `rosterHeight.guard`) le prendrait pour une
  faute. Le test 4.3 ôte les commentaires avant de chercher ; non traité côté source.
