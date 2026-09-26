| 2026-08-16 23:58 | Lot B (phase 1bis + 4 + 5, exécuteur Opus, 85 outils, ~22 min) | gates REJOUÉS : `tsc -b --force` 0, `vitest src/features/match-replay src/features/match-view/MatchHeader src/routes` 532/532 ; PNG `replay-black.png` OUVERT (silhouette flèches + triangle, ovale gris transparent, 488×405) ; route relue (`useSettings` → `buildPlayerMarks`, `colorOfTeam` en `useCallback`, grille `xl:grid-cols-[minmax(240px,300px)_minmax(0,1fr)_minmax(0,28rem)]`, enveloppe `order-2 grid grid-cols-2 xl:contents` = écart assumé sur 5.1, correct : sans parent commun fil+fiches ne peuvent pas être côte à côte sous `xl`) ; 4.4 `[~]` accepté (3 cas ajoutés à `MatchHeader.test.tsx`, pas de doublon de montage) | CLOSE côté code — GATE VISUEL USER OUVERT |
| 2026-08-16 23:58 | Serveur de gate visuel | vite du worktree lancé détaché sur `http://localhost:5174` (log `apps/web/vite-wt-habillage.log`), proxy vers l'API locale :8000 déjà en marche ; le vite du principal (:5173) n'est PAS lancé | PRÊT |
# Plan — Habillage du rejeu 2D : marqueurs, noms, amis, logo, mise en page 3 colonnes

> Ecrit le 2026-08-16. Demande utilisateur (mot pour mot) : « Diminuer la taille des points,
> ajouter le pseudo en dessous des points (avec une bordure sur les lettres pour faciliter la
> lecture sur les fonds clairs et foncés), supprimer le bâton qui sort de ces points [...].
> Pour les amis il faut trouver un moyen de les distinguer, une forme différente peut être,
> ou un symbole à côté de leur nom [...] dans les fiches on distingue les amis [...].
> Remplacer le logo du replay 2D par des logos blanc ou noir de ce logo (en fonction du mode
> sombre ou clair de l'app) `Downloads/Replay-logo.png`. Le killfeed sera déplacé à gauche
> du replay, sur la même rangée, alignement sur la hauteur [...] le scroll descendant ira
> chercher les événements anciens. À droite de la map s'afficheront les deux colonnes des
> fiches des joueurs de chaque équipe et chaque équipe devra retrouver son nom (Équipe Cobra
> / Équipe Eagle sur le scoreboard, sans réinventer la roue). » Autorisation explicite de
> PROPOSER (« je veux un truc propre et joli »).
>
> Branche `feat/v75` (mode branche unique), contrat `plan-execution`. Zéro Go : tout est
> dans `apps/web`. Garde prod du rejeu (`handlers/replay_local_gate.go`) intacte.

## 0. Ce qui est établi — vérifié sur pièces le 2026-08-16

| Sujet | Où c'est aujourd'hui | Constat |
|---|---|---|
| Marqueur joueur | `features/match-replay/replayMarkers.ts` `drawMarker` : `CORE_RADIUS 4.6`, `HALO_RADIUS 9`, anneaux d'étage `RING_GAP 3.6`, liseré `OUTLINE_PAD 1.15` | tailles du POC ; aucune étiquette de nom sur la carte |
| « Le bâton » | `drawAimCone` : cône dégradé (`AIM_LENGTH 46`) **+ un AXE en deux traits** (`strokeAxis` : liseré `AIM_UNDERLINE_WIDTH 3.6` + trait `AIM_AXIS_WIDTH 1.8`) | l'axe est le bâton ; le cône seul dit déjà la direction du regard. Bouton `Visée` (`showAim`) existant |
| Couleur des marqueurs | `ReplayCanvas.tsx:188-191` `getSeriesColors(doc.tracks.length, TRACK_TOKENS)` indexé par **trace** ; `drawTracksLayer` lit `style.colors[i]` (i = index de trace = de VIE) | **une trace = une vie** (99 traces / 8 joueurs sur 000d5950, cf. `rosterLogic.ts` en-tête) : un joueur change de couleur à chaque réapparition, contrairement à la doctrine écrite dans `rosterLogic.ts` (« couleur attribuée AU JOUEUR »). `colorIndex` de `ReplayPlayer` n'est lu nulle part |
| Fil (kill feed) | `ReplayKillFeed.tsx` : carte `max-h-64`, liste `overflow-y-auto`, plus récent EN TÊTE, recollage en tête si le lecteur y était (`AT_TOP_PX`) | le sens « descendre = plus ancien » est DÉJÀ celui du fil ; seule la hauteur est figée. Noms colorés par équipe via `teamColorResolver` |
| Fiches | `ReplayTeams.tsx` : grille `repeat(groups.length, 1fr)`, en-tête `<span>{group.side ?? t.teamUnknown}</span>` coloré `compare-a/b` | **l'en-tête affiche `t0` / `t1` brut** ; aucune marque ami/moi ; colonnes non bornées en hauteur |
| Nom d'équipe | `MatchScoreboard.tsx:522-534` et `MatchObjectivesSection.tsx:59-83` (`teamVisual`) : cascade `team_name` backend → `resolveTeamName(team_side)` → `t.teamLabelFmt` / `labelHasTeamWord` → `teamNumberedFmt` → `teamUnknown` | **DEUX copies** ; le rejeu serait la troisième → centraliser + garde-rail (règle 6 CLAUDE.md). Couleur : `features/match-view/teamColor.ts` `teamColorResolver` déjà centralisé (utilisé par le fil du rejeu) |
| Amis | `settings.friend_gamertags` (`useSettings()`, `features/settings/queries.ts`) ; appariement par gamertag normalisé dans `features/match-view/colors.ts` (`normalizeGamertag`, privé) ; scoreboard `is_me` = joueur de la page | aucune notion d'ami dans `match-replay/` |
| Logo | `MatchHeader.replayLink.tsx` : SVG « projecteur » inline `currentColor` ; page rejeu : `<h1>` texte seul | précédent PNG noir/blanc par thème : `public/icons/{halowaypoint,nemesis,victim}-{black,white}.png`, suffixe calculé dans `lib/match-nav/waypointUrl.ts:35` ET `MatchNemesisCards.tsx:27` (**deux copies** du `theme === 'light' ? 'black' : 'white'`) → le rejeu serait la troisième |
| Mise en page | `routes/.../replay.tsx` : `space-y-4` — carte, PUIS fil, PUIS fiches, empilés | le POC (artefact `eb7b8af2`, spec de rendu) a la rangée `carte | fil | équipe 1 | équipe 2` avec les colonnes latérales `position:absolute; inset:0` pour que la CARTE impose la hauteur (« aucune colonne ne peut étirer la ligne ») |
| POC — nom sur la carte | `strokeText`/`fillText` 8,7 px gras, contour noir 2,8 px `lineJoin round`, couleur d'équipe, à DROITE du marqueur | à porter SOUS le marqueur (demande) ; le contour est la technique du POC (« lisible sur un fond qui va du blanc au noir ») |
| Fond de carte | PNG cuits : aire jouable gris clair, pourtour sombre | confirme le besoin d'un contour de lettres qui tienne sur les deux |
| Tests | `ReplayTeams.test.tsx`, `ReplayKillFeed.test.tsx`, `test/recordingContext.ts` (contexte canvas enregistreur, sans pixel) | les tests de géométrie émise sont possibles pour l'étiquette et la forme du marqueur |
| Seuils | `ReplayCanvas.tsx` 733 L (dette gelée), `ReplayTeams.tsx` 450 L, `replayMarkers.ts` 400 L | ne pas accroître : extraire (`ReplayTeamHeader.tsx`, `replayLabels.ts`), logique pure dans `rosterLogic.ts` |

## 1. Décisions produit — TRANCHÉES (propositions soumises au user AVANT exécution)

Chaque décision est numérotée ; l'exécuteur ne rouvre aucune d'elles. Si le user en amende
une, la ligne est réécrite ici avant le lot.

- **D1 (AMENDÉE par le user le 16/08) — Couleur des marqueurs = la COULEUR D'ÉQUIPE CHOISIE PAR LE USER dans les paramètres d'accessibilité** (`team-ally` / `team-enemy`, tokens surchargeables — `theme-provider.tsx` pose `--ac-team-ally/enemy`, `useColorPaletteVersion` observe le style de `:root` donc la carte se re-résout au changement), par JOUEUR (jamais par vie). Allié = même `team_side` que le joueur de la page (`xuidMeta.ally`), sinon adversaire ; vie sans xuid ou sans équipe → encre neutre du thème (`--muted-foreground`). Le nom sous le point distingue les coéquipiers. Corrige au passage la couleur qui changeait à chaque réapparition. **COHÉRENCE DE PAGE (décision d'exécution, à confirmer par le user au point d'étape)** : les en-têtes de fiches (D8) et les noms du fil emploient les MÊMES tokens `team-ally/enemy` sur la page de rejeu — un point bleu et un nom rouge pour la même équipe seraient une page cassée ; le fil de la Match View (`MatchKillFeed`) garde sa cascade d'identité, on ne le touche pas (résolveur passé en prop au fil du rejeu). Les 8 tokens de série restent aux zones nommées (`zoneColors`) et à rien d'autre.
- **D2 — Taille.** Noyau 4,6 → 3,2 px d'écran (−30 %), halo 9 → 6,5, accroissement par étage
  0,9 → 0,7 et 2 → 1,5, écart d'anneaux 3,6 → 2,8, liseré 1,15 → 1,0, croix de mort 5 → 4.
  Anneau d'apparition inchangé (c'est un événement, il doit se voir).
- **D3 — Le bâton disparaît, le cône reste.** L'axe (`strokeAxis`, `AIM_AXIS_*`,
  `AIM_UNDERLINE_*`) est SUPPRIMÉ. Le cône dégradé reste la seule marque du regard,
  raccourci (46 → 30 px) et plus discret (alpha 0,44 → 0,30), toujours sous le bouton
  `Visée` (défaut allumé). Le texte de l'infobulle du bouton dit ce que c'est.
- **D4 — Nom sous le point.** `displayPlayerName(playerName(player), xuid)` centré sous le
  marqueur, 8,5 px d'écran semi-gras (`600`), fond de contour 2,6 px `lineJoin round`, encre
  de contour `--replay-label-stroke` (variable de thème, sombre dans les DEUX thèmes — choix
  du POC, validé à l'usage sur des fonds qui vont du blanc au noir), lettres à la couleur du
  marqueur. Pas d'étiquette pour une vie sans xuid, ni pendant la croix de mort. Bouton
  `Noms` (défaut allumé) au même rang que `Zones` : un BTB à 24 joueurs doit pouvoir
  l'éteindre.
- **D5 — Amis et « moi », UNE grammaire sur les trois panneaux.**
  Sur la CARTE : ami = marqueur en LOSANGE (même rayon circonscrit) ; moi (`is_me` du
  scoreboard = joueur de la page) = disque + anneau externe 1,5 px à l'encre `--foreground`.
  Dans les FICHES et le FIL : un GLYPHE avant le nom (`PlayerMark`) — `friend` : deux
  silhouettes, `me` : disque plein — 10 px, `currentColor`, infobulle FR/EN. Le nom garde sa
  couleur (équipe / mort) : on ajoute une marque, on ne change pas une couleur qui dit déjà
  autre chose. Amis = `settings.friend_gamertags` du COMPTE CONNECTÉ, gamertag normalisé
  comme dans `match-view/colors.ts` ; le joueur de la page n'est jamais marqué ami.
- **D6 — Logo : paire PNG `replay-black.png` / `replay-white.png`** dans `public/icons/`,
  dérivée du PNG source (silhouette = les deux flèches + le triangle ; l'ovale gris devient
  transparent), choisie par le thème local via UN helper `themedIconSrc(name, theme)` qui
  absorbe les deux copies existantes. Employé par le bouton `ReplayLink` (remplace le SVG
  projecteur) ET dans le titre de la page de rejeu.
- **D7 — Rangée `fil | carte | fiches`**, hauteur imposée par la carte (technique du POC :
  colonnes latérales `absolute inset-0`, contenu `min-h-0` qui défile). Fil ≈ 240-300 px, la
  carte prend le reste, fiches = 2 colonnes d'équipe côte à côte (≈ 440 px). Sous le point
  de rupture `xl` (1280 px) : carte pleine largeur, puis fil et fiches côte à côte, chacun
  borné à `60vh`. La hauteur du canvas (`CANVAS_HEIGHT` 480) ne change PAS dans ce lot.
- **D8 — En-tête d'équipe des fiches** = libellé résolu (`Équipe Eagle` / `Team Cobra`,
  `Équipe 12` hors référentiel, `Équipe inconnue` sans côté) + couleur d'équipe D1 (`team-ally`/`team-enemy`, liseré
  gauche 3 px + fond `color-mix 14 %`, texte en `foreground` — même grammaire que l'en-tête
  du scoreboard). Les tokens `compare-a/b` de `teamColor(index)` sont SUPPRIMÉS (code mort).

## 1bis. Amendement du 2026-08-16 (soir) — le STYLE DE LA PLANCHE validé (item A1)

Verdict user sur la planche de validation tenue par l'autre session (`PLAN_RETOURS_PLANCHE_2026-08-16.md`,
item A1, routé vers CE plan ; paramètres dans `NOTE_STYLE_MARQUEURS_PLANCHE_2026-08-16.md`, worktree
principal) : « Parfait. Je veux exactement ce style pour les points, la croix de mort et la
traînée. L'icône de visée je veux celui-là mais un peu plus prononcé, juste un peu. Je préfère
ce rendu à l'actuel. » Ce verdict AMENDE D2 et D3 (valeurs), pas D1 (la couleur reste celle des
réglages d'accessibilité — décision user de la même journée, la note dit elle-même que la
couleur des traces « reste à la règle actuelle »). Valeurs en pixels d'ÉCRAN (× k) :

- **D2' — le point** : disque plein rayon **3,4** (était 3,2 après D2) ; **plus de halo diffus**
  (`HALO_*` supprimés) ; **marqueur d'étage = anneau rayon 6,5, trait 1, encre `--foreground`
  (style.ink), alpha 0,9** — un anneau par étage au-dessus du sol (règle actuelle du COMPTE
  conservée, `RING_GAP` inchangé), c'est le STYLE de l'anneau qui change (encre du thème, pas
  la couleur de trace ; 1 px, pas 1,5). Le liseré sombre (`OUTLINE_*`) et les formes D5
  (losange ami, anneau moi) sont conservés — la planche ne les contredit pas.
- **D2' — la traînée (7 s)** : trait **1,6** (était 1,9), `lineCap round`, **alpha qui monte vers
  le présent : 0,08 (queue) → 0,63 (tête), linéaire** sur la fenêtre — donc la polyligne se
  dessine par SEGMENTS (un `stroke` par segment avec son alpha), plus un seul `stroke` à 0,55.
- **D3' — le cône de visée** : rayon **52** (était 30 après D3), demi-ouverture **0,42 rad** (était
  0,30), dégradé radial couleur → transparent, alpha **0,55** (était 0,30) ; fraîcheur (`AIM_FADE`)
  conservée ; toujours PAS d'axe (D3 tient). « Un peu plus prononcé » que la planche = ces
  valeurs, à valider à l'œil.
- **D2' — la croix de mort (1,5 s)** : demi-taille **5**, trait **1,6**, `lineCap round`, alpha
  **0,9** ; couleur de trace du mort ; le grossissement `DEATH_GROWTH` disparaît (la planche
  montre une croix FIXE qui s'estompe, pas une croix qui grandit).
- **D2' — l'anneau d'apparition (0,8 s)** : cercle qui s'ouvre de **2 → 14**, trait **1,2**, alpha
  **0,8 → 0**, couleur de trace.

### Phase 1bis — Le style de la planche (rapide ; APRÈS la phase 3, AVANT la phase 4)

- [x] 1bis.1 `replayMarkers.ts` : constantes et dessins mis aux valeurs D2'/D3' ci-dessus ; halo
      supprimé (constantes ET code) ; anneaux d'étage à l'encre `style.ink` ; traînée par
      segments à alpha croissant ; croix fixe ; anneau d'apparition 2 → 14 / 1,2 px. En-tête
      du fichier : remplacer la mention « réduites de 30 % » par « style de la planche validée
      le 2026-08-16 (item A1), valeurs dans le plan d'habillage §1bis ».
- [x] 1bis.2 `replayLabels.ts` : l'étiquette se pose sous le BORD EXTERNE du marqueur
      (`markerEdge` = rayon de l'anneau d'étage le plus large, ou du disque + liseré si étage 0)
      — vérifier que le nom ne chevauche pas l'anneau à 6,5 px.
- [x] 1bis.3 `replayMarkers.test.ts` : adapter les attentes (plus de halo = un `arc` de moins
      au marqueur ; traînée = n−1 `stroke` ; croix : 2 `moveTo` + 2 `lineTo`, aucun changement
      de rayon dans le temps ; cône : ouverture 0,42 lue sur l'`arc`).

**Gate 1bis** : `cd apps/web && npx tsc -b && npx vitest run src/features/match-replay`.
**PASSÉ le 2026-08-16 (worktree `wt/habillage-rejeu`)** : `npx tsc -b` exit 0 ; `npx vitest run
src/features/match-replay` = 35 fichiers / 470 tests, 0 échec (+14 depuis la phase 3) ;
`npx eslint src/features/match-replay` = 0 erreur, 0 avertissement. Quatre précisions
d'exécution, toutes dans la lettre du §1bis mais qu'il ne chiffrait pas :
1. **L'anneau d'étage est posé à un rayon FIXE** (`RING_RADIUS 6.5`), les suivants à
   `6,5 + RING_GAP × (r−1)` — la formule d'avant (`noyau + RING_GAP × r`) aurait rendu 6,2 au
   premier anneau et l'aurait fait grossir avec l'étage, alors que la note de planche donne
   6,5 pour l'anneau lui-même. Le COMPTE (un anneau par étage) et `RING_GAP` sont inchangés,
   comme demandé ; `RING_ALPHA_DECAY` aussi (0,9 pour le premier anneau, puis −0,18).
2. **`CORE_PER_FLOOR` (0,7) est CONSERVÉ** : le §1bis ne le mentionne pas, donc il ne le
   change pas. Marge vérifiée à l'étage le plus haut (fl = 2, le seul possible : `FLOOR_BANDS`
   vaut 3) : bord du liseré 5,8 px contre anneau à 6,5 px — les deux ne se touchent pas.
3. **`markerEdge` prend le MAXIMUM de trois bords** (liseré, anneau d'étage, anneau « moi »)
   au lieu du seul liseré : sans cela, le nom d'un joueur à l'étage passait SOUS l'anneau à
   6,5 px, et celui du joueur de la page sous son anneau d'identité à 6,0 px. La fonction
   reçoit donc la forme du marqueur en plus de l'étage.
4. **La traînée n'émet plus qu'un `stroke` par segment** (n−1 pour n positions) : c'est la
   seule façon de faire monter l'opacité vers la tête. Coût mesuré nul à l'échelle des tests ;
   à surveiller au gate visuel sur un BTB (24 vies × ~70 positions).

## 2. Objectif et critère de succès

Sur `000d5950` (4v4, Cliffhanger) et sur un BTB : la page de rejeu tient sur une rangée
`fil | carte | fiches` à 1440 px de large ; chaque point porte son nom lisible sur zone
claire ET sombre ; aucun trait ne sort des points ; amis et moi se repèrent sans lire ; les
fiches titrent `Équipe Eagle` / `Équipe Cobra` ; le logo suit le thème. `make check-types`,
`vitest` du périmètre et `eslint` verts. Gate visuel = le USER (jamais l'agent).

## 3. Phases

Ordre strict ; une phase est CLOSE quand tous ses items sont statués `[x]`/`[~]`/`[!]` ET
que son gate a été exécuté avec sortie collée dans le CR. Statuts : `[x]` fait, `[~]`
couvert ailleurs (référence), `[!]` non traité (justification écrite).

### Phase 0 — Socle partagé : helpers centralisés + garde-rails (rapide)

- [x] 0.1 (`lib/ui/` n'existe pas : fichier posé en `apps/web/src/lib/themedIcon.ts`, à côté de `staticAssets.ts`) `apps/web/src/lib/ui/themedIcon.ts` : `themedIconSrc(name: string, theme: UiTheme): string`
      → `/icons/${name}-${theme === 'light' ? 'black' : 'white'}.png`. Migrer les 2 copies :
      `lib/match-nav/waypointUrl.ts:35` (`waypointLogoSrc` devient un appel) et
      `features/match-view/MatchNemesisCards.tsx:27,54,62`. Garde-rail
      `lib/ui/themedIcon.guard.test.ts` : grep interdisant `'black' : 'white'` et
      `-black.png`/`-white.png` littéraux dans `src/**` hors `themedIcon.ts`.
- [x] 0.2 `apps/web/src/lib/halo/teamLabel.ts` : `resolveTeamLabel(rows: MatchScoreboardRow[],
      teamSide: string | null, text: TeamLabelText): string` avec
      `TeamLabelText = { teamLabelFmt(name): string; teamNumberedFmt(n): string; teamUnknown: string }`
      (cascade EXACTE de `MatchScoreboard.tsx:522-534`). Migrer `MatchScoreboard.tsx` et
      `MatchObjectivesSection.tsx` (`teamVisual` garde la couleur via `teamColorResolver(rows)(teamID, isMyTeam)`
      — 3e copie de la cascade couleur, absorbée dans le même geste). Garde-rail
      `lib/halo/teamLabel.guard.test.ts` : `labelHasTeamWord(` interdit hors `lib/halo/`.
      Tests unitaires : backend `Équipe Cobra` non re-préfixé ; `t0` → `Équipe Eagle` (FR) /
      `Team Eagle` (EN) ; `t12` → `Équipe 12` ; `null` → `Équipe inconnue`.
- [x] 0.3 Exporter `normalizeGamertag` : le déplacer dans `lib/players/displayName.ts`
      (`normalizeGamertagKey`), `match-view/colors.ts` l'importe. Aucun changement de
      comportement (test de non-régression sur `buildMatchPlayerColors` déjà présent ou à
      ajouter : 1 cas ami allié).
- [x] 0.4 `features/match-replay/playerMarks.ts` (PUR) : `type PlayerMarkKind = 'me' | 'friend'` ;
      `buildPlayerMarks(scoreboard, friendGamertags: readonly string[]): ReadonlyMap<string, PlayerMarkKind>`
      (clé xuid ; `is_me` → `me` ; sinon gamertag normalisé ∈ amis → `friend`). Tests :
      moi jamais ami ; casse/espaces ignorés ; ami adverse marqué aussi (la marque dit
      l'identité, pas le camp).
- [x] 0.5 `features/match-replay/PlayerMark.tsx` : glyphe SVG inline 10 px `currentColor`,
      `role="img"` + `aria-label` = `t.markFriend` / `t.markMe`, `title` idem. Deux
      dessins : `friend` (deux silhouettes), `me` (disque plein). Test : rend l'aria-label
      attendu par variante ; rien pour `undefined`.
- [x] 0.6 i18n `features/match-replay/i18n.ts` (FR + EN, typage `Record<Locale, T>`) :
      `teamLabelFmt`, `teamNumberedFmt`, `markFriend` (« Ami » / « Friend »), `markMe`
      (« Moi » / « Me »), `layerNames` (« Noms » / « Names »), `layerNamesHint`,
      `layerAimHint` réécrit (« Direction du regard (cône) » / « Look direction (cone) »).
- [x] 0.7 `globals.css` : `--replay-label-stroke` défini dans les DEUX thèmes (sombre dans
      les deux — décision D4), à côté des `--replay-fx-*`. `canvasInk.ts` : `InkVar`
      accepte `'--replay-label-stroke'`. Le garde-rail existant `fxInk.guard.test.ts` ne
      bouge pas ; ajouter un cas dans un test d'encres : la variable est lue sans erreur. FAIT : `canvasInk.guard.test.ts` (variable présente pour :root + dark + light).

**Gate 0** : `cd apps/web && npx vitest run src/lib src/features/match-view src/features/match-replay/playerMarks src/features/match-replay/PlayerMark`
+ `make check-types`. Grep de contrôle : `grep -rn "'black' : 'white'" apps/web/src` → 1 seul
fichier ; `grep -rn "labelHasTeamWord(" apps/web/src` → `lib/halo/` seulement.
**PASSÉ le 2026-08-16** : tsc exit 0 ; vitest `src/lib src/features/match-view` + 3 fichiers replay = 1101/1102 (le seul rouge = timeout 5 s de `langSegmentInheritance.test.ts` sous charge, hors périmètre, VERT relancé seul) ; grep `'black' : 'white'` -> `lib/themedIcon.ts` seul ; grep `labelHasTeamWord(` -> `lib/halo/` seul.

### Phase 1 — La carte : marqueurs, bâton, noms, formes (moyen)

- [x] 1.1 Couleur PAR JOUEUR (D1) : dans `rosterLogic.ts`, `colorBySlot(players: ReplayPlayer[],
      colorOf: (ally: boolean) => string, ally: (xuid) => boolean, neutral: string): Map<number, string>`
      (slot de chaque vie → couleur de son propriétaire ; vie sans propriétaire → `neutral`).
      `ReplayCanvas` reçoit `scoreboard` et `xuidMeta` (la route les a déjà) et remplace
      `colors`/`colorBySlot` (lignes 188-191, 227-234) par cet appel ; `drawTracksLayer`
      reçoit `colorOfSlot: (slot) => string` au lieu de `colors[]` (`MarkerStyle.colors`
      supprimé). `killFx` continue de lire `colorBySlot`. Supprimer le champ `colorIndex`
      de `ReplayPlayer` (jamais lu). Tests `rosterLogic.test.ts` : 2 vies du même xuid →
      même couleur ; vie sans xuid → neutre.
- [x] 1.2 Tailles (D2) : constantes de `replayMarkers.ts` mises aux valeurs de D2 ; en-tête
      du fichier mis à jour (la mention « tailles du POC » devient « réduites de 30 % le
      2026-08-16 à la demande de l'utilisateur »).
- [x] 1.3 Bâton (D3) : supprimer `strokeAxis`, `AIM_AXIS_ALPHA/WIDTH`, `AIM_UNDERLINE_ALPHA/WIDTH`
      et leur appel ; `AIM_LENGTH 30`, `AIM_CONE_ALPHA 0.30`. Le cône reste sous `showAim`.
- [x] 1.4 Formes (D5) : `drawMarker(ctx, c, style, color, fl, shape: 'circle' | 'diamond' | 'ring')`
      — losange = 4 `lineTo` sur le rayon du noyau (+ liseré de même forme), `ring` =
      disque + `arc` externe 1,5 px `style.ink`. `MarkerStyle` reçoit `markOfSlot: (slot) => PlayerMarkKind | undefined`
      (dérivé de `buildPlayerMarks` + slot → xuid, calculé UNE fois dans `rosterLogic`).
- [x] 1.5 Étiquette (D4) : nouveau fichier `features/match-replay/replayLabels.ts`
      (`drawNameLabel(ctx, c, name, style, color, fl)`) : `font = "600 ${8.5*k}px ui-sans-serif, system-ui, sans-serif"`,
      `textAlign center`, `textBaseline top`, `y = c.y + core + OUTLINE_PAD*k + 2.5*k`,
      `lineJoin round`, `lineWidth 2.6*k`, `strokeStyle = style.labelStroke`, puis `fillText`
      à `color`. Appelée depuis `drawLivingTrack` quand `style.showNames && nameOfSlot(slot)`.
      `ReplayCanvas` : état `showNames` (défaut `true`) + bouton `Noms` (même patron que
      `Zones`), `labelStroke = readInk('--replay-label-stroke')` mémoïsé par thème,
      `nameOfSlot` = `displayPlayerName(playerName(p), p.xuid)` par slot (rosterLogic).
- [x] 1.6 Tests (`recordingContext`) dans `replayMarkers.test.ts` (nouveau) : (a) une vie
      vivante émet `strokeText` PUIS `fillText` du nom avec `lineJoin round` ; (b) aucun
      `lineTo` n'est émis par le calque de visée (le bâton n'existe plus) ; (c) `friend` →
      chemin à 4 `lineTo` (losange), `me` → un `arc` de plus que le cercle simple ;
      (d) `showNames=false` → aucun `fillText`.
- [x] 1.7 En-têtes de `replayMarkers.ts` et `ReplayCanvas.tsx` : commentaires « 8 tokens de
      série = un par entité » et « couleur par trace » réécrits selon D1.

**Gate 1** : `cd apps/web && npx vitest run src/features/match-replay` + `make check-types`.
Vérifier sur pièces que `TRACK_TOKENS` n'est plus lu que par `zoneColors`.
**PASSÉ le 2026-08-16** : `npx tsc -b` exit 0 ; `npx vitest run src/features/match-replay`
= 35 fichiers / 456 tests, 0 échec. `TRACK_TOKENS` a été RENOMMÉ `ZONE_TOKENS` (il ne sert
plus qu'à `zoneColors`, une seule lecture — le nom mentait autrement). Précisions
d'exécution : `colorBySlot` s'appuie sur un foyer générique `indexBySlot` (3e descente
« joueur -> vies » avec `markBySlot`/`nameBySlot`, règle CLAUDE.md n°6) ; `drawNameLabel`
reçoit le bord externe du marqueur (`markerEdge`) plutôt que `fl`, pour que `replayLabels.ts`
n'importe RIEN de `replayMarkers.ts` (cycle évité) ; le bouton `Noms` vit dans
`useReplaySettings` (clé `replay-show-names`, défaut allumé) + `LayersSection` du tiroir,
les boutons de calques ayant migré dans `ReplaySettingsDrawer` depuis l'écriture du plan.

### Phase 2 — Les fiches : nom d'équipe, marques, colonnes qui défilent (moyen)

- [x] 2.1 Extraire `ReplayTeamHeader.tsx` (`ReplayTeams.tsx` frôle 500 L) : libellé via
      `resolveTeamLabel(rowsOfGroup, group.side, t)` ; couleur via `tokenCssVar(ally ? 'team-ally' : 'team-enemy')` (D1 amendée ; `ally` = un joueur du groupe est allié selon `xuidMeta`)
      ; rendu D8 (liseré gauche 3 px + fond `color-mix(in srgb, <couleur> 14%, transparent)`,
      texte `text-foreground`, compteur de joueurs conservé). Supprimer `teamColor(index)`.
- [x] 2.2 `ReplayTeams` reçoit `marks: ReadonlyMap<string, PlayerMarkKind>` ; `PlayerCard`
      rend `<PlayerMark kind={marks.get(player.xuid)} />` AVANT le nom (gap 1). Hauteur de
      la ligne de nom inchangée (glyphe 10 px dans une ligne de 11,5 px).
- [x] 2.3 Colonnes bornées : racine `grid h-full min-h-0`, chaque colonne d'équipe
      `flex h-full min-h-0 flex-col`, l'en-tête `shrink-0`, la liste des fiches
      `min-h-0 flex-1 overflow-y-auto` (leçon POC : « `min-height: 0` autorise le
      rétrécissement, le défilement vit à l'INTÉRIEUR de la carte »). Hors rangée (repli
      étroit), la racine n'a pas de hauteur imposée → aucun défilement, comportement actuel.
- [x] 2.4 Tests `ReplayTeams.test.tsx` : en-tête `Équipe Eagle` pour `t0` (FR) et `Team Cobra`
      pour `t1` (EN) ; `Équipe 12` pour `t12` ; glyphe `Ami` sur la fiche d'un ami, `Moi`
      sur `is_me`, aucun glyphe sinon. Adapter les fixtures existantes qui attendaient
      `t0` brut si elles existent (vérifier sur pièces).

**Gate 2** : `cd apps/web && npx vitest run src/features/match-replay/ReplayTeams` + `make check-types`.
**PASSÉ le 2026-08-16** : `npx tsc -b` exit 0 ; `npx vitest run src/features/match-replay/ReplayTeams`
= 1 fichier / 35 tests, 0 échec (6 nouveaux : libellés `t0`/`t1`/`t12`, colonne sans camp,
teinte allié/adverse, marques Moi/Ami). Aucune fixture n'attendait `t0` brut (vérifié sur
pièces : tous les rendus existants passaient `scoreboard={[]}`, donc « Sans équipe »).
`PlayerMark` est posé FRÈRE du nom (pas parent) pour ne pas décaler la profondeur DOM sur
laquelle les tests de hauteur constante comptent les rangées d'une fiche.

### Phase 3 — Le fil : marques + hauteur de colonne (rapide)

- [x] 3.1 `ReplayKillFeed` reçoit `marks` ET `colorOf?: TeamColorResolver` (défaut = `teamColorResolver(scoreboard)`, la Match View ne bouge pas ; la route du rejeu passe le résolveur D1 `team-ally/enemy`) ; `PlayerMark` avant le nom du tueur, de la
      victime, du mort neutre et de l'assistant (les 4 endroits où un nom s'écrit —
      lignes ~215, ~293, ~337, ~368, ~388 ; vérifier sur pièces).
- [x] 3.2 Hauteur : racine `flex h-full min-h-0 flex-col`, liste `min-h-0 flex-1 overflow-y-auto`
      (retirer `max-h-64`, garder `min-h-[4.5rem]`). Sens de lecture INCHANGÉ (plus récent
      en tête, descendre = plus ancien) ; recollage en tête inchangé. Commentaire d'en-tête
      : ajouter « la hauteur est celle de la rangée, jamais la sienne ».
- [x] 3.3 Tests `ReplayKillFeed.test.tsx` : glyphe `Ami` sur une ligne dont le tueur est ami ;
      `Moi` sur la victime `is_me` ; classes de défilement présentes sur la liste.

**Gate 3** : `cd apps/web && npx vitest run src/features/match-replay/ReplayKillFeed` + `make check-types`.
**PASSÉ le 2026-08-16** : `npx tsc -b` exit 0 ; `npx vitest run src/features/match-replay/ReplayKillFeed`
= 1 fichier / 26 tests, 0 échec. Deux précisions d'exécution :
1. **L'ASSISTANT N'A PAS DE XUID** dans l'événement du film (`KillEvent.assistGamertag` seul,
   cf. `features/match-view/_momentum.ts`) — vérifié sur pièces. Sa marque passe donc par une
   réindexation `gamertag normalisé -> marque` via le scoreboard (seule table qui porte les
   deux clés). 5 noms marqués au total, pas 4 : le décoré d'une ligne de MÉDAILLE SEULE en
   est un aussi, le laisser nu aurait cassé la grammaire.
2. `min-h-0` ET `min-h-[4.5rem]` sur la même liste se contredisent (deux `min-height`) : seul
   `min-h-[4.5rem]` est posé — il écrase déjà le `min-height:auto` du flex, donc la liste se
   rétrécit et défile exactement comme `min-h-0` l'aurait permis, avec en plus le plancher
   voulu.
`MedalBadges` a été extrait dans `MedalBadges.tsx` : le fil passait à 507 L (seuil 500,
CLAUDE.md n°5) en recevant les marques — la dette n'a pas été accrue, elle a été rendue.

**Fin de mission phases 1-3, le 2026-08-16** : `npx vitest run src/features/match-replay
src/features/match-view src/lib` = 146 fichiers / 1568 tests, 0 échec ; `npx eslint
src/features/match-replay src/features/match-view src/lib` = 0 erreur, 6 avertissements TOUS
préexistants et hors périmètre (`match-view` : TanStack Table incompatible-library ×2,
exhaustive-deps ×3, react-refresh ×1) ; `npx tsc -b` exit 0.

> Phase 1bis (style de la planche, §1bis) s'exécute ICI, entre la phase 3 et la phase 4.

### Phase 4 — Le logo (rapide)

- [x] 4.1 Génération de `apps/web/public/icons/replay-black.png` et `replay-white.png` depuis
      `C:\Users\Guillaume\Downloads\Replay-logo.png` par un programme Go JETABLE dans le
      scratchpad (`image/png`, pas de Python, pas d'ImageMagick — absent du poste) :
      `alphaOut = alphaIn × clamp((R − max(G,B)) / 160, 0, 1)` (les rouges = encre, l'ovale
      gris/blanc = transparent), couleur pleine noire / blanche, recadrage à la boîte
      englobante + marge 4 %, taille native conservée. Contrôle : ouvrir les deux PNG
      (Read) — silhouette = deux flèches + triangle, aucun résidu gris. Le programme n'est
      PAS versionné (git garde les PNG, pas l'outil).
- [x] 4.2 `MatchHeader.replayLink.tsx` : `<img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-4 w-auto" />`
      à la place du SVG projecteur ; `theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)`.
- [x] 4.3 Route `replay.tsx` : le `<h1>` devient `flex items-center gap-2` avec le logo
      (`h-5 w-auto`) devant `t.title`.
- [~] 4.4 Test `MatchHeader.replayLink.test.tsx` (nouveau si absent — vérifier) : `src` se
      termine par `replay-white.png` en thème sombre, `replay-black.png` en clair ; rien
      quand `available=false`.

**Gate 4** : `cd apps/web && npx vitest run src/features/match-view/MatchHeader` + `make check-types`
+ contrôle visuel des deux PNG.
**PASSÉ le 2026-08-16** : `npx tsc -b` exit 0 ; `npx vitest run src/features/match-view/MatchHeader`
= 1 fichier / 32 tests, 0 échec (3 nouveaux : logo blanc en sombre, noir en clair, aucun logo
sans artefact). Contrôle visuel des PNG : source 453x369, boîte d'encre (0,0)-(451,368),
marge 18 px, **sortie 488x405** pour les deux ; silhouette = les deux flèches en boucle + le
triangle de lecture, **aucun résidu de l'ovale gris** (le seuil du plan passe sans retouche :
le rouge du logo donne R − max(G,B) ≈ 196, le gris 0). Trois précisions :
1. **`MatchHeader.replayLink.test.tsx` n'a PAS été créé** : `MatchHeader.test.tsx` teste déjà
   le lien de rejeu (`describe('MatchHeaderCard — lien vers le rejeu 2D')`, 4 cas) avec tous
   ses mocks de route et de requêtes. Les 3 cas du plan y ont été ajoutés — un second fichier
   aurait dupliqué 100 lignes de montage pour observer le même composant. Statut `[~]`.
2. **Les deux PNG portent exactement le MÊME canal alpha** (vérifié par un second programme
   jetable : 0 pixel de différence sur 197 640 ; 43 794 opaques, 9 513 semi-transparents pour
   l'anti-crénelage des diagonales) — les variantes ne diffèrent que par l'encre.
3. Le blanc étant invisible sur le fond de l'outil de lecture, il a été **composé sur un fond
   sombre** dans le scratchpad pour être vu : même silhouette, bords propres.

### Phase 5 — La rangée `fil | carte | fiches` (moyen ; dépend de 2.3 et 3.2)

- [x] 5.1 `routes/.../replay.tsx` : remplacer les trois blocs empilés par
      ```
      <div className="grid gap-3 xl:grid-cols-[minmax(240px,300px)_minmax(0,1fr)_minmax(0,28rem)] xl:items-stretch">
        <aside className="relative order-2 min-h-[12rem] xl:order-1 xl:min-h-0">
          <div className="flex max-h-[60vh] flex-col xl:absolute xl:inset-0 xl:max-h-none"><ReplayKillFeed .../></div>
        </aside>
        <section className="order-1 min-w-0 xl:order-2"><ReplayCanvas .../></section>
        <aside className="relative order-3 min-h-[12rem] xl:min-h-0">
          <div className="flex max-h-[60vh] flex-col xl:absolute xl:inset-0 xl:max-h-none"><ReplayTeams .../></div>
        </aside>
      </div>
      ```
      Justification écrite dans le commentaire de la route (technique du POC : la carte
      impose la hauteur ; `absolute inset-0` empêche les colonnes de l'étirer). Sous `xl`,
      un `grid grid-cols-2 gap-3` enveloppe fil + fiches (le fil à gauche).
- [x] 5.2 Câblage des nouvelles props : `useSettings()` dans la route → `friendGamertags`
      ; `marks = useMemo(buildPlayerMarks(scoreboard, friendGamertags))` ; passer
      `scoreboard`, `xuidMeta`, `marks` à `ReplayCanvas`, `marks` à `ReplayTeams` et
      `ReplayKillFeed`.
- [x] 5.3 Commentaires de la route réécrits (« LE FIL SOUS LA CARTE » et « les fiches vivent
      À CÔTÉ ») selon la nouvelle rangée.
- [x] 5.4 Vérifier sur pièces que le `ResizeObserver` de `ReplayCanvas` mesure bien la
      colonne centrale (`min-w-0` posé) et que la carte ne déborde pas horizontalement
      quand une colonne latérale contient un gamertag long (`min-w-0`/`truncate` déjà
      présents dans les fiches et le fil).

**Gate 5** : `make check-types` + `cd apps/web && npx vitest run src/features/match-replay src/routes`
+ `cd apps/web && npx eslint src/features/match-replay src/features/match-view src/lib src/routes`
+ **GATE VISUEL USER** (la main au user, jamais l'agent) : `000d5950` (4v4) et un BTB, à
1440 px et à 1920 px, thème clair ET sombre, avec un match où figure un ami de
`friend_gamertags`. Témoins à nommer par le user.
**PASSÉ (part agent) le 2026-08-16** : `npx tsc -b` exit 0 ; `npx vitest run
src/features/match-replay src/routes` = 38 fichiers / 500 tests, 0 échec ; `npx eslint
src/features/match-replay src/features/match-view src/lib src/routes` = 0 erreur, 6
avertissements — les MÊMES qu'à la clôture des phases 1-3, tous dans `match-view` et hors
périmètre. **Le gate VISUEL reste ouvert : il appartient au user.** Trois précisions
d'exécution :
1. **Une enveloppe `xl:contents` apparie le fil et les fiches sous 1280 px.** La grille du
   plan est plate (trois enfants), mais D7 demande aussi « fil et fiches CÔTE À CÔTE » sous
   `xl` — impossible sans un parent commun. L'enveloppe porte donc
   `order-2 grid grid-cols-2 gap-3 xl:contents` : sous `xl` elle est une grille à deux
   colonnes placée après la carte ; à partir de `xl` elle passe en `display: contents`,
   disparaît de la mise en page et rend ses deux `aside` directement à la rangée, qui
   retrouve exactement la structure à trois colonnes du plan (`xl:order-1/2/3`). L'`order-2`
   est indispensable : sans lui l'enveloppe (ordre 0 par défaut) passerait AVANT la carte
   (`order-1`) dans le repli étroit.
2. **Le résolveur de couleur D1 est un `useCallback`** dans la route, pas une fonction
   littérale : le fil le prend en dépendance de `useMemo`, une identité neuve à chaque rendu
   aurait reconstruit son index à chaque image.
3. **Sous `xl`, ce sont le `min-h-0` des racines (2.3 / 3.2) et le rétrécissement flex par
   défaut qui font défiler les panneaux dans leur borne de 60vh** — la borne est un
   `max-height`, elle n'impose donc aucune hauteur quand le contenu est court.

### Phase 6 — Clôture

- [x] 6.1 `cd apps/web && npx vitest run` (suite web complète) + `make check-types` + `npx eslint .`
      (depuis `apps/web`) — sorties collées dans le CR.
- [x] 6.2 Registre `.ai/V7.5/REGISTRE_REPORTS.md` : une ligne par item `[!]` s'il y en a ;
      ligne « hauteur du canvas adaptative à la colonne (480 px figés) » avec condition de
      reprise = retour du gate visuel user.
- [x] 6.3 `.ai/thought_log.md` : entrée datée (décisions D1-D8, mesures du gate, ce qui a
      surpris). `.ai/V7.5/README.md` : ce plan indexé sous `replay2d/`.
- [x] 6.4 Commit(s) sur `feat/v75` APRÈS accord du user (règle 16), messages
      `feat(v7.5-rejeu): ...` en français, un commit par phase 0-5 ou un seul si le user
      préfère. Pas de merge (mode branche unique).

## 4. Hors périmètre — NE PAS TRAITER, consigner ici

- Hauteur du canvas adaptative (`CANVAS_HEIGHT` 480 figé) — voir 6.2.
- Colonnes des fiches à 3+ équipes (FFA) : la grille suit `groups.length`, mais la largeur
  `28rem` est pensée pour deux colonnes ; à revoir si un gate FFA le demande.
- Rebrancher `killicon` pour l'icône d'arme des fiches (déjà au registre).
- Cascade couleur d'équipe dupliquée dans `MatchScoreboard.tsx:539-541` : absorbée en 0.2
  UNIQUEMENT si le geste tient en < 10 lignes ; sinon, ligne au registre.

## 5. Découvertes (à remplir pendant l'exécution)

Constats faits en cours de route, HORS périmètre du plan — aucun n'a été corrigé (règle 7 du
contrat d'exécution).

1. **Une carte SANS amplitude verticale met tous les joueurs à l'étage 1** (découvert en
   écrivant `replayMarkers.test.ts`, phase 1.6). `floorOf(z, minZ, maxZ)` passe par
   `altitudeRatio`, qui rend **0,5 quand `minZ == maxZ`** (`replayLogic.ts`, « 0,5 si
   plate ») ; `Math.floor(0.5 × 3) = 1`. Chaque marqueur porte donc un anneau d'étage
   permanent sur un document dont les bornes verticales sont absentes ou égales — un anneau
   qui affirme un étage là où il n'y a aucune mesure. Le repli « milieu » est juste pour une
   OPACITÉ de décor (l'usage d'origine d'`altitudeRatio`), il ne l'est pas pour un COMPTEUR
   d'anneaux. Correction probable : `floorOf` rend 0 quand l'amplitude est nulle. Non traité :
   hors périmètre, et le rendu réel (documents avec `minZ`/`maxZ`) n'est pas concerné.
2. **Les EFFETS DE MORT sur la carte changent de couleur avec D1.** `drawKillFxLayer` lit la
   même table `colorOfSlot` que les marqueurs : les traits tueur -> victime passent des 8
   teintes de série aux deux couleurs d'équipe. C'est cohérent (l'effet appartient au tueur),
   mais ce n'est écrit nulle part dans D1 — à confirmer au gate visuel : sur un kill entre
   deux adversaires, le trait était bicolore par joueur, il sera bicolore par CAMP.
3. **`ReplayCanvas.tsx` reste au-dessus du seuil** (779 L, dette gelée, 731 L avant le lot).
   Les items 1.1/1.5 ont été extraits dans `rosterLogic.ts`, `replayLabels.ts` et
   `useSlotIdentity.ts` pour limiter la croissance à +48 L (props documentées, deux mémos de
   couleur, quatre entrées de dépendances). La descendre sous 500 demanderait de sortir le
   corps de `draw()` — chantier propre, hors de ce plan.
4. **`ReplayTeams.tsx` est à 488 L** (450 avant) : la prochaine addition franchira le seuil.
   `PlayerCard` (≈ 130 L) est le candidat naturel à l'extraction.
5. **`replayMarkers.ts` est à 493 L** après la phase 1bis (445 avant) : le fichier est à sept
   lignes du seuil, pour l'essentiel du commentaire qui documente le style de la planche. Le
   prochain geste sur ce calque devra extraire — `drawProjectilesLayer` (≈ 40 L) n'a rien à
   voir avec le marqueur d'un joueur et sortirait sans rien casser. Non traité : hors
   périmètre du plan (règle 7).
6. **`PalmaresRelationsPage.test.tsx > rend les badges solid (duo gagnant)` tombe en timeout
   5 s dans la suite web COMPLÈTE, et passe seul** (14/14 en 3,9 s). Même signature que le
   rouge observé au gate 0 sur `langSegmentInheritance.test.ts` : un test au timeout par
   défaut de 5 s dans un fichier qui monte une page entière, sous la charge de 434 fichiers en
   parallèle. Ce n'est pas une régression de ce plan (aucun fichier de `palmares` n'a été
   touché) mais ce sont maintenant DEUX fichiers qui flanchent ainsi — la suite complète est
   en train de devenir non déterministe sur ce poste. Non traité : hors périmètre.

## 6. Reprise de session

Lire ce fichier (statuts), puis `git log --oneline -10` sur `feat/v75`, puis le dernier CR
dans `.ai/thought_log.md`. Reprendre à la première case non statuée de la première phase
non close. Ne jamais commencer la phase N+1 tant que le gate N n'a pas été exécuté.

## 7. Journal superviseur (annoté au fil de l'eau — verdicts DE REVUE, distincts des cases des exécuteurs)

| Quand | Quoi | Vérifié comment | Verdict |
|---|---|---|---|
| 2026-08-16 22:45 | Phase 0 (faite par le superviseur AVANT le passage en pilotage — faute de méthode consignée en mémoire : worktree principal partagé, « ok plan » ≠ go) | migrée dans `LevelUp-wt-habillage` ; `npm ci`, `tsc -b` 0, 24 tests phase 0 verts ; principal remis à l'état de l'autre agent (`git checkout --` + rm de mes seuls fichiers) | CLOSE |
| 2026-08-16 23:31 | Lot A (phases 1-3, exécuteur Opus, 161 outils, ~40 min) | gates REJOUÉS par le superviseur : `tsc -b --force` exit 0, `vitest src/features/match-replay` 466/466 ; lecture sur pièces de `replayMarkers.ts`, `replayLabels.ts`, `useSlotIdentity.ts`, `ReplayTeamHeader.tsx` ; écarts assumés acceptés (`TRACK_TOKENS`→`ZONE_TOKENS`, `MedalBadges` extrait, assistant marqué par gamertag) | CLOSE — à voir à l'œil : couleurs d'équipe sur les traits de mort (découverte 2) |
| 2026-08-16 23:40 | Amendement §1bis (style de la planche A1, note de l'autre session routée vers ce plan ; go user « ça fait partie de l'amélioration UI ») | valeurs recopiées de `NOTE_STYLE_MARQUEURS_PLANCHE_2026-08-16.md` (principal, lecture seule) ; D1/D5 explicitement conservées | ÉCRIT — piège rencontré : insertion perl a double-encodé le fichier, réparé (0 mojibake vérifié) |
| 2026-08-16 23:45 | Lot B lancé (phase 1bis + 4 + 5, exécuteur Opus, même worktree) | — | EN COURS |
| 2026-08-16 23:58 | Lot B (phase 1bis + 4 + 5, exécuteur Opus, 85 outils, ~22 min) | gates REJOUÉS : `tsc -b --force` 0, `vitest src/features/match-replay src/features/match-view/MatchHeader src/routes` 532/532 ; PNG `replay-black.png` OUVERT (silhouette flèches + triangle, ovale gris transparent, 488×405) ; route relue (`useSettings` → `buildPlayerMarks`, `colorOfTeam` en `useCallback`, grille `xl:grid-cols-[minmax(240px,300px)_minmax(0,1fr)_minmax(0,28rem)]`, enveloppe `order-2 grid grid-cols-2 xl:contents` = écart assumé sur 5.1, correct : sans parent commun fil+fiches ne peuvent pas être côte à côte sous `xl`) ; 4.4 `[~]` accepté (3 cas ajoutés à `MatchHeader.test.tsx`, pas de doublon de montage) | CLOSE côté code — GATE VISUEL USER OUVERT |
| 2026-08-16 23:58 | Serveur de gate visuel | vite du worktree lancé détaché sur `http://localhost:5174` (log `apps/web/vite-wt-habillage.log`), proxy vers l'API locale :8000 déjà en marche ; le vite du principal (:5173) n'est PAS lancé | PRÊT |
| (à venir) | Gate visuel USER : témoins à nommer (000d5950 + un BTB, 1440/1920, clair/sombre, un ami) | | |
| 2026-08-17 00:27 | Réalignement `wt/habillage-rejeu` sur `feat/v75` local (merge --no-ff, 55 commits : sons, carte de chaleur, callouts) | 2 conflits de contenu résolus EN GARDANT LES DEUX INTENTIONS : `useReplaySettings.ts` (le hook porte désormais `showNames`/`toggleNames` ET `showHeatmap`/`toggleHeatmap`/`heatmapMode`/`setHeatmapMode`, clés `replay-show-names` + `replay-show-heatmap`/`replay-heatmap-mode` toutes conservées) et `ReplayCanvas.tsx` (la seule collision était la déstructuration du hook ; calque des noms/`useSlotIdentity`/couleurs d'équipe D1 ET calque heatmap + légende + cuisson hors écran cohabitent, le tiroir reçoit `showNames`/`onToggleNames` ET `heatmap={...}`). Auto-merges RELUS : `ReplaySettingsDrawer.tsx` (LayersSection = visée + Noms + Zones, HeatmapSection distincte), `i18n.ts` (`layerNames`/`layerNamesHint` + `layerHeatmap*`/`heatmapMode*` FR et EN), `ReplaySettingsDrawer.test.tsx` et `useReplaySettings.test.tsx` (les cas des deux lots coexistent). Gates RÉELS après `npm ci` : `tsc -b --force` exit 0 ; `vitest src/features/match-replay src/features/match-view src/lib src/routes` 153 fichiers / 1662 tests verts ; `eslint` sur les mêmes chemins exit 0 (6 avertissements préexistants hors rejeu) ; `vitest run` complet 439 fichiers / 3965 verts + 14 skippés, 0 échec | CLOSE — aucune régression, aucune fonctionnalité perdue d'aucun côté |
| (à venir) | Arbitrage merge : `ReplaySettingsDrawer.tsx` touché aussi par le lot R1 de l'autre session (overlay) | | |
| 2026-08-17 00:35 | Réalignement (exécuteur, 32 outils, ~14 min) — merge `feat/v75` local `3058afbba` → `2c956ac9a` | gates REJOUÉS : `tsc -b --force` 0, `vitest src/features/match-replay` 527/527 ; parents et arbre propre vérifiés ; exécuteur : suite complète 3965/0, eslint 0 erreur, ratchets pre-push verts ; il a commis une fois avec `--no-verify` puis corrigé par `--amend` avec hooks (SHA final = avec hooks) | CLOSE |
| 2026-08-17 00:40 | Phase 6 docs : 6.1 `[x]` (gates ci-dessus, `make check-types` = `tsc -b` ; `npx eslint .` complet non rejoué — périmètre rejeu+match-view+lib+routes à 0 erreur), 6.2 `[x]` registre (5 lignes : canvas 480, floorOf, seuils de fichiers, timeouts, tiroir ×2 lots), 6.3 `[x]` thought_log (entrée du 17/08 en tête) + README déjà indexé | écrit dans le worktree, commit docs sur la branche | 6.4 EN ATTENTE : merge `--no-ff` dans `feat/v75` sur confirmation user |
| 2026-08-17 01:30 | Réalignement 2 : merge `--no-ff` de `feat/v75` local (`d635a96b6`, lot R1 « retours de la planche » : tiroir en OVERLAY, bascules Effets de tirs / Effets de mort, explosions 2,4 s, éclat de réapparition 1,2 s + « Réapparition dans X s », grenades en images, marque d'assistance en vignette du jeu, callouts à 9,5 px) | 5 conflits résolus EN GARDANT LES DEUX INTENTIONS : (1) `.ai/thought_log.md` ajout en tête — les 3 entrées conservées, ordre antéchronologique (assistance 01:05, R1 00:37, habillage 00:32), aucune ligne perdue ; (2) `useReplaySettings.ts` — le calque NOMS passe au `usePersistedFlag` de R1 (6e copie centralisée, doc mise à jour), les six interrupteurs sur leur propre clé ; (3) `ReplayCanvas.tsx` — déstructuration : noms + effets + vitesse ; (4) `ReplayKillFeed.tsx` — la ligne d'assistance porte À LA FOIS la vignette `AssistMark` (killfeed-62, R1) ET le glyphe `PlayerMark` de l'assistant (habillage), `PICTOGRAM_PX` retenu (nom généralisé de R1), `MEDAL_PX` non réintroduit (il vit dans `MedalBadges.tsx` extrait par l'habillage) ; (5) `ReplaySettingsDrawer.test.tsx` — `renderDrawer` rend les 7 espions des deux lots. Auto-merges RELUS : `ReplaySettingsDrawer.tsx` (le panneau overlay `absolute inset-y-0 right-0 z-20` porte visée + Noms + Zones + chaleur + les deux effets), `i18n.ts` (`layerNames*` + `effects`/`layerShotFx*`/`layerKillFx*` + `killFeedAssistMark`/`killFeedAssistShare` + `markMe`/`markFriend`, FR et EN), `ReplayTeams.tsx` (en-tête `ReplayTeamHeader` + marques ET `RESPAWN_FLASH_S` 1,2 s), `globals.css` (`--replay-label-stroke` ET `.replay-flash-respawn` 1,2 s), `ReplayKillFeed.test.tsx` / `useReplaySettings.test.tsx` (les cas des deux lots coexistent). Gates RÉELS (lock npm inchangé, pas de `npm ci`) : `tsc -b --force` exit 0 ; `vitest src/features/match-replay src/features/match-view src/lib src/routes` 155 fichiers / 1683 tests verts ; `eslint` sur les mêmes chemins exit 0 (6 avertissements préexistants, tous dans `match-view`) ; suite complète `npx vitest run` 441 fichiers / 3 986 tests verts + 14 skippés préexistants, 0 échec (213 s) | CLOSE — rien de perdu d'aucun côté ; découverte non traitée (registre) : `ReplayKillFeed.tsx` 518 L et `ReplayCanvas.tsx` 861 L, les deux au-delà du seuil de 500 L, dette gelée des deux lots |
| 2026-08-17 01:30 | 6.4 FAIT — merge `--no-ff` `wt/habillage-rejeu` (`17059b2ee`, réalignement 2 sur R1 revu : tsc 0, rejeu vert, 5 conflits gardant les deux intentions) dans `feat/v75` = **`8f33ffe19`**, poussé (`8fbf69bf4..8f33ffe19`, emporte les commits locaux de l'autre session) | `git merge-base --is-ancestor` vérifié avant (aucun conflit possible), hooks actifs | LOT CLOS — reste le GATE VISUEL USER sur `feat/v75` (décision user : après fusion) ; découvertes du réalignement 2 au registre à verser : `ReplayKillFeed.tsx` 518 L, `ReplayCanvas.tsx` 861 L (dette cumulée des deux lots), ligne d'assistance à DEUX pictogrammes (vignette + glyphe ami/moi) jamais vue à l'écran |
