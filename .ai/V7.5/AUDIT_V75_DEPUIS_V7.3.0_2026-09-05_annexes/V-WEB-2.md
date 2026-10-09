# Vérification adverse V-WEB-2

Dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, branche `feat/v75`, HEAD `736ccf3c3`.
Lecture seule. Tous les chemins ci-dessous sont absolus ou relatifs à
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/apps/web/src/`.

---

## Constat 1 — Deux formateurs d'horloge sur le même écran (P0) : RÉFUTÉ

### Ce que j'ai vérifié

**Le mécanisme allégué est réel, ligne par ligne :**

- `features/match-replay/replayLogic.ts:310-314` — `formatClock` : `Math.max(Math.floor(ms / 1000), 0)`.
- `lib/formatters/duration.ts:44-49` — `formatClockMMSS` : `Math.max(0, Math.round(... / 1000))`.
- `features/match-replay/useReplayTimeline.ts:122-125` :
  `const clockOf = useCallback((replayMs) => formatClockMMSS(displayClockMs(replayMs, playWindow)), [playWindow])`.
- `features/match-replay/ReplayKillFeed.tsx:234` : `replayMs: displayClockMs(entry.replayMs, playWindow)`
  puis `:274` `{formatClock(ms)}` dans `FeedClock`.
- Les deux partent bien du **même** `displayClockMs` (`replayWindow.ts:138-141`), et de la même source
  (`feedEntries` → `reduceFeed` → `buildEventTracks`, `useReplayTimeline.ts:126-129`).
- La valeur peut être un ms non entier de seconde : `displayClockMs` = `ms - playWindow.startMs`, et
  `startMs = Math.max(0, startT0Ms - originMs)` (`replayWindow.ts:122`) est un ms arbitraire du film.
  L'écart `1:05` / `1:06` est donc **arithmétiquement possible**.

**Mais l'auditeur n'a pas suivi la chaîne jusqu'à l'écran.** Commande :

```
grep -rn "TrackMark\|clockOf" --include=*.ts --include=*.tsx features/match-replay/ | grep -v "\.test\."
```

`TrackMark.clock` (le champ produit par `useReplayTimeline.ts:123`) a **exactement un** consommateur
dans tout le dépôt : `ReplayTimelineTracks.tsx:416`, `title={m.clock}` — sur un `<span>` dont la
classe est posée deux lignes plus haut :

```
ReplayTimelineTracks.tsx:411:
  className={`pointer-events-none absolute rounded-[2px] ${tall ? '…' : '…'}`}
ReplayTimelineTracks.tsx:416:
  title={m.clock}
```

`pointer-events: none` retire l'élément du hit-test : le navigateur ne le désigne jamais comme cible
de survol, donc **l'infobulle native `title` ne s'affiche jamais**. Ce n'est pas un accident, c'est la
décision écrite en tête du fichier (`ReplayTimelineTracks.tsx:21-22`) :

> « LES PISTES NE CAPTENT PAS LE POINTEUR (`pointer-events-none`), sauf les vignettes de médias qui
> sont des boutons : la frise reste saisissable au pixel près, y compris SOUS une marque. »

### Ce que l'auditeur n'a pas vu

1. **La chaîne `Math.round` n'aboutit à aucun pixel lisible.** Le `1:06` de la démonstration est calculé,
   stocké dans un attribut `title`, et jamais rendu. Aucune « même élimination horodatée différemment à
   20 cm d'écart » n'est reproductible à l'écran.

2. **Toutes les horloges VISIBLES de la page passent par `formatClock` (floor), sans exception.**
   Commande : `grep -rn "formatClock\b\|formatClockMMSS" --include=*.ts --include=*.tsx . | grep -v "\.test\."`
   - fil : `ReplayKillFeed.tsx:274` — floor
   - bandeau : `ReplayScoreBanner.tsx:121` — floor
   - transport (le chrono de la frise lui-même, `clockRef`) : `useReplayClock.ts:67,76` — floor
   - infobulle de pose : `ReplayPlacementTip.tsx:108` — floor
   - dialogue d'export : `useReplayExport.ts:402,406` → `ReplayExportDialog.tsx:126,133,195` — floor
   Les deux formateurs ne se rencontrent donc sur **aucune** surface visible.

3. **Le seul autre `formatClockMMSS` du dossier porte sur un AUTRE fait.**
   `ReplayMediaLightbox.tsx:84,132,133` : `formatClockMMSS(item.durationMs ?? 0)` — la DURÉE d'un
   média, pas l'instant du match. Domaine disjoint.

4. **La « doc inversée » est une mélecture, pas une doc fausse.** Le commentaire incriminé vit dans
   `features/match-view/_scoreCurve.ts:33-35`, module de logique pure de `MatchScoreCurveChart` :
   > « L'instant en M:SS a UN foyer dans le dépôt (`lib/formatters`) : la carte et la frise du rejeu
   > l'appellent toutes deux, aucune ne le réécrit. »
   Dans la prose de ce dossier, « la carte » désigne la CARTE/le chart de la vue match, pas le bandeau
   du rejeu : `MatchScoreCurveChart.tsx:8` « c'est cette carte qui le pose », `:13` « la carte ne rend
   RIEN », `:61` « `hidden` efface cette carte », `:113` « la carte s'efface ». Et de fait
   `_scoreCurve.ts:35` fait `export { formatClockMMSS as formatClock } from '@/lib/formatters'` : la
   carte de courbe **appelle** `formatClockMMSS`, la frise du rejeu (`useReplayTimeline.ts:123`) aussi,
   et **aucune des deux ne le réécrit**. La phrase est exacte mot pour mot. L'auditeur y a lu
   `ReplayScoreBanner`, qui n'est ni « la carte » de cette phrase ni sur la même page que ce module.

### Conséquence réelle reformulée

Il reste un second formateur (`Math.round`) dont l'unique consommateur est un `title` rendu inatteignable
par `pointer-events-none` : c'est un attribut mort sur une marque de frise, pas une divergence
d'horodatage visible — donc ni P0, ni le risque décrit ; au plus un P2 « consommateur mort » hors du
périmètre duplication.

---

## Constat 2 — Index « vies par slot » réécrit 4 fois (P1) : TIENT

### Ce que j'ai vérifié

Recompte des bornes citées, puis `diff` réel bloc à bloc :

```
sed -n '51,56p' fireMark.ts  > /tmp/a1
sed -n '63,68p' grappleLayer.ts > /tmp/a2
sed -n '90,95p' shotFx.ts > /tmp/a3
sed -n '143,148p' thrusterDashFx.ts > /tmp/a4
diff a1 a2 → vide ; diff a1 a3 → vide ; diff a1 a4 → vide
md5sum → a3b6defce455b84f6d81c90f2285b5b4  (les 4)
```

Les quatre blocs de 6 lignes sont **strictement byte-identiques**, aux lignes exactement citées.
Il n'existe aucun `buildLivesBySlot` : `livesPosition.ts` (102 L, lu en entier) n'expose que
`posOfPlayerAt`, `buildLivesByXuid` (par **xuid**), `deathWindowFrames`, `buildPlayerPosAt`.

Le garde-rail voisin ne couvre pas le motif : `livesPosition.guard.test.ts:45` teste
`src.includes('livesByXuid')` — une chaîne que ces quatre fichiers ne contiennent pas. Son propre
en-tête (`:26-28`) le dit : « une réécriture qui renommerait sa carte passerait ».

### Ce qui confirme

- La 5ᵉ variante citée (`abilityChargeLogic.ts:133`) est bien différente ET **documentée**
  (`:131-132` : « La VIE qui couvre l'image — jamais `Map(slot → vie)`, qui écraserait les vies d'un
  même slot (P0 de la revue P4 ; patron de `buildFireMarks`) »). L'auditeur l'avait déjà classée à part
  comme « variante linéaire » : classement correct, et elle ne rachète pas les 4 copies littérales.
- Nuance à porter au débit du constat, sans le renverser : les *lookups* qui suivent (`fireMark.ts:60`,
  `grappleLayer.ts:72`, `shotFx.ts:101`, `thrusterDashFx.ts:152`) ne sont **pas** identiques
  (`s.t` / `l.t0` / `imp.t`, variables `l` vs `v`). L'auditeur les avait listés séparément et n'a jamais
  prétendu l'inverse — la seule affirmation de byte-identité porte sur les 6 lignes, et elle est vraie.

### Conséquence réelle reformulée

Le même index « vies par slot » existe en quatre exemplaires littéraux qu'aucun test ne surveille : une
correction de la règle de sélection de vie (chantier SIÈGES) doit être écrite quatre fois ou deux calques
se contrediront sur le même tir.

---

## Constat 3 — `allyTeamFromScoreboard` extraite « et pas dupliquée », dupliquée 3× (P1) : TIENT

### Ce que j'ai vérifié

Le commentaire de la canonique est bien celui cité (`objectiveSound.ts:139-145`) :

> « EXTRAITE DE `sideResolverFromScoreboard` LE 2026-08-27, **et pas dupliquée** … Deux lectures du
> tableau de score qui divergeraient feraient sonner l'ennemi pour l'allié sans que rien ne le signale. »

Diff des trois copies :

```
sed -n '79,82p' useReplayBombBlast.ts / '119,122p' useReplayFlagCarries.ts / '78,81p' useZoneStates.ts
md5sum → b642d391b71ff48c433e7adf7a11c163  (les 3)
```

Byte-identiques, aux lignes exactes du constat. Contenu :
`() => parseTeamSideID(scoreboard?.find((r) => r.is_me)?.team_side ?? null)` dans un `useMemo([scoreboard])`.

Bloc frère `teamOfXuid` : `diff` de `useReplayBombBlast.ts:70-77` contre
`useReplayFlagCarries.ts:167-174` → **sortie vide**.

Qui importe la canonique :
```
grep -rn "allyTeamFromScoreboard" --include=*.ts --include=*.tsx . | grep -v "\.test\."
```
→ `objectiveSound.ts` (définition) et `useReplaySound.ts:72,309`. **Aucun des trois hooks de calque.**

Aucun obstacle technique : les trois importent déjà `parseTeamSideID` depuis `@/lib/halo/teamNames`
(`useReplayBombBlast.ts:22`, `useZoneStates.ts:33`) et pourraient importer la canonique de la même façon.
Le seul frein est un frein de PLACEMENT (une fonction de camp logée dans un module `*Sound.ts`) —
ce qui renforce le constat au lieu de l'excuser.

Aucun garde-rail : `grep -rn "r.is_me"` rend 6 sites dans `match-replay/` (dont les 4 lectures du camp)
et 4 dans `match-view/`, sans aucun test grep qui les borne.

### Ce qui confirme

La factorisation est datée, motivée par écrit contre ce risque précis, et non propagée : c'est
littéralement l'anti-pattern n° 8 du CLAUDE.md (« créer le helper canonique sans migrer les copies ni
poser le garde-rail »).

### Conséquence réelle reformulée

Quatre lectures indépendantes de « quel camp est le mien » commandent l'encre de l'Assaut, du drapeau,
des zones et des sons sur le même écran, sans que rien ne détecte leur divergence — le risque exact que
le commentaire de la canonique dit couvrir.

---

## Constat 4 — Passage monde → canvas ré-emballé 8 fois (P1) : TIENT (chiffres à corriger, à la hausse)

### Ce que j'ai vérifié

Les 4 `project()` :

```
placementShapes.ts:44-46  export function project(p: XY, view: PlacementView): XY
replayMarkers.ts:524-526  function project(p: XY, view: CanvasView): XY
replayProjectiles.ts:56-58  function project(p: XY, view: CanvasView): XY
thrusterDashFx.ts:321-323 function project(p: XY, view: CanvasView): XY
```
md5 des trois derniers (signature + corps) : identiques (`59be3dbe…`). Le quatrième
(`placementShapes.ts`) ne diffère que par `export`, l'alias de type `PlacementView` et une ligne de doc :
le **corps** est byte-identique aux quatre, exactement comme l'affirme le constat
(« corps identique `return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)` »).

Les 4 closures `px` (`calloutsLayer.ts:178`, `flagCarriesLayer.ts:260`, `objectivesLayer.ts:158`,
`zoneStatesLayer.ts:268`) : md5 des 4 lignes → `f5abb9210faf3fc61d16b0f7f044b46f` pour les quatre.
**Strictement byte-identiques**, y compris le nom `px` et la signature `(p: XY)`.

Aucune canonique : `grep -rn "projectTo"` → aucun résultat ; `worldToCanvas` est exporté
(`replayLogic.ts:108`) et appelé partout. Aucun garde-rail.

### Ce que mon recompte change (dans le sens défavorable à l'audité)

```
grep -rn "worldToCanvas(" --include=*.ts --include=*.tsx features/match-replay | grep -v "\.test\." | wc -l
→ 36
```
36 − 1 définition (`replayLogic.ts:108`) = **35 sites d'appel** ; − 1 fixture de test
(`test/placementFixtures.ts:118`) = **34 en production**, contre « 32 » annoncés. L'auditeur a par
ailleurs manqué `replayLogic.ts:540` (dans `layerOffset` — interne au module canonique, donc légitime)
et compte 24 dépliages là où j'en dénombre 25 dans sa propre liste. L'erreur va donc dans le sens de la
sous-estimation : elle ne réfute rien.

### Conséquence réelle reformulée

Le cadrage est un objet que rien ne sait projeter : 8 emballages littéraux et ~25 dépliages des 5
arguments, donc un champ ajouté au cadrage se paie en 34 sites.

---

## Constat 5 — `alphaOf` + 4 constantes copiés dans 3 calques (P1) : TIENT (conséquence à nuancer)

### Ce que j'ai vérifié

```
sed -n '81,86p' vipCrownLayer.ts / '85,90p' skullCarrierLayer.ts / '114,119p' bombCarrierLayer.ts
md5sum → ca9ba49f9521b40914f1abb812bd6e1d  (les 3)
```
`alphaOf` **byte-identique** aux trois emplacements exacts du constat.

```
grep -rn "ALPHA_SOLID\|PULSE_MIN\|PULSE_MAX\|PULSE_PERIOD_FRAMES" --include=*.ts . | grep -v "\.test\."
```
- `ALPHA_SOLID = 0.95` : `vipCrownLayer.ts:59`, `skullCarrierLayer.ts:59`, `bombCarrierLayer.ts:60`,
  `flagCarriesLayer.ts:165` → **4 copies**.
- `PULSE_MIN = 0.42` / `PULSE_MAX = 0.78` / `PULSE_PERIOD_FRAMES = 26` : `vipCrownLayer.ts:61-63`,
  `skullCarrierLayer.ts:61-63`, `bombCarrierLayer.ts:68-70` → **3 copies chacune**, mêmes valeurs.

Aucun module commun, aucun garde-rail. R6 (≥ 3 copies) et « Copy-paste config » sont enfreints sur pièces.

### Ce que l'auditeur n'a pas vu (nuance, pas réfutation)

La conséquence énoncée — « un match d'Assaut avec VIP montrerait deux rythmes concurrents » — est
douteuse : la couronne VIP, le crâne (Oddball/Crâne) et la bombe (Assaut) appartiennent à **trois modes
distincts** ; les trois glyphes ne se rencontrent pas sur la même carte. Le scénario de désynchronisation
visible n'est donc pas démontré. La violation R6, elle, est indépendante de la co-occurrence : trois
copies littérales d'une fonction et de ses seuils, sans foyer ni garde-rail, restent trois copies.
Gravité maintenue à P1 (3 copies + 4 constantes + zéro garde-rail), conséquence à reformuler.

### Conséquence réelle reformulée

La règle de pulsation d'un glyphe porté et ses quatre seuils existent en trois exemplaires littéraux
qu'aucun test ne relie : régler l'un ne règle pas les deux autres, et rien ne le signale.

---

## Constat 6 — « L'intervalle couvre l'image » : 10 écritures, 2 orthographes (P1) : TIENT (gravité → P2)

### Ce que j'ai vérifié

```
grep -rn "frame >= [a-z]*\.t0 && frame <= [a-z]*\.t1\|\.t0 <= frame && frame <= " \
  --include=*.ts --include=*.tsx features/match-replay | grep -v "\.test\." | wc -l
→ 10
```
Les 10 sites sont exactement ceux cités (`bombCarrierLayer.ts:82,103`, `objectiveMark.ts:110,144`,
`riftStations.ts:153`, `skullCarrierLayer.ts:79`, `skullPresence.ts:64`, `vehiclesLayer.ts:395,499`,
`vipCrownLayer.ts:75`), en 2 orthographes (7 + 3). Aucune canonique `covers()` / `spansAt()` n'existe.
Aucun garde-rail. Les 3 `…ActiveAt` (`vipCrownLayer.ts:69-80`, `skullCarrierLayer.ts:73-84`,
`bombCarrierLayer.ts:76-87`) ne diffèrent, au `diff`, que par le nom de la fonction, le type et le nom de
la variable de boucle — la ligne de prédicat est identique.

### Ce que l'auditeur n'a pas vu — sa preuve de divergence dit le contraire de ce qu'il en tire

Le constat affirme : « `skullPresence` déclare `carried` à `t1` exactement, **tandis que**
`bombCarrierLayer:103` refuse le sol à ce même `t1` — la règle de frontière est déjà un choix par
calque ». Les deux lignes, lues :

```
skullPresence.ts:64      if (carry.t0 <= frame && frame <= carry.t1) return { state: 'carried' }
bombCarrierLayer.ts:103  if (frame >= c.t0 && frame <= c.t1) return null // portée : jamais au sol en même temps
```

C'est la **même** convention, appliquée cohéremment : borne fermée à `t1`, donc à `t1` l'objet est
encore PORTÉ — et un objet porté n'est, par définition, pas au sol (`return null`). Le « tandis que »
présente un accord comme un désaccord. Vérification étendue : les 10 sites sont tous fermés-fermés
(`>=` … `<=`), et les « 2 orthographes » ne sont qu'un ordre d'opérandes — sémantiquement identiques.
**Il n'y a aucune divergence de convention dans le dossier, actuelle ou latente.**

### Conséquence réelle reformulée

Un prédicat booléen d'une ligne, uniforme sur ses 10 sites, sans foyer ni garde-rail : le coût est
prospectif (une correction de bord à écrire dix fois), il n'y a aucune incohérence en service — ce qui
place le constat au niveau d'un idiome non factorisé (P2), pas d'un risque de contradiction entre
calques (P1).

---

## Bilan : 4 tiennent, 1 réfuté, 1 requalifié

| # | Constat | Verdict |
|---|---|---|
| 1 | Deux formateurs d'horloge (P0) | **RÉFUTÉ** — la sortie `Math.round` n'atteint aucun pixel (`title` sur un `pointer-events-none`, décision documentée `ReplayTimelineTracks.tsx:21`) ; toutes les horloges visibles passent par `formatClock` ; la « doc inversée » de `_scoreCurve.ts:33-35` est exacte (« la carte » = la carte de courbe de match-view, qui appelle bien `formatClockMMSS`). Résidu : au plus P2. |
| 2 | Index vies par slot ×4 (P1) | **TIENT** — 4 blocs md5-identiques, aucune canonique, garde-rail qui ne grep que `livesByXuid`. |
| 3 | `allyTeamFromScoreboard` ×3 + `teamOfXuid` ×2 (P1) | **TIENT** — md5 identiques, `diff` vide, canonique jamais importée par les 3 hooks, aucun garde-rail. |
| 4 | Monde → canvas ×8 (P1) | **TIENT** — 4 corps + 4 closures byte-identiques ; recompte à 34 appels (l'audit sous-estimait à 32). |
| 5 | Pulsation portée ×3 (P1) | **TIENT** — `alphaOf` md5-identique ×3, `ALPHA_SOLID` ×4, `PULSE_*` ×3. Conséquence à réécrire : les 3 glyphes relèvent de 3 modes distincts et ne co-occurrent pas. |
| 6 | « L'intervalle couvre l'image » ×10 (P1) | **TIENT (gravité → P2)** — comptes exacts, mais la divergence de borne alléguée entre `skullPresence.ts:64` et `bombCarrierLayer.ts:103` n'existe pas : même convention fermée à `t1`, et les 10 sites sont uniformes. |
