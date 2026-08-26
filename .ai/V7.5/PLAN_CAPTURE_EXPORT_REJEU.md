# PLAN — Capture d'image et enregistrement vidéo du rejeu 2D

> Date : 2026-08-26. Branche : `wt/rejeu-capture-image-video` (base `f4fcbfa72`, lignée v7.5).
> Worktree dédié : `C:/Users/Guillaume/Projects/LevelUp-wt-capture-rejeu`.
> **Contrat d'exécution : skill `plan-execution`** (ordre strict, aucun report d'item
> exécutable, chaque item statué `[x]` / `[~]` réf / `[!]` justifié, zéro fix hors
> périmètre — les trouvailles vont dans la section « Découvertes »).
> Skills à invoquer avant d'agir : `frontend-patterns` (tout le lot est du React/TS),
> `color-tokens` (si la moindre couleur apparaît — normalement aucune).

## Objectif et critère de succès

Deux commandes nouvelles sur la page de rejeu 2D :

1. **Capturer l'image** — télécharge un PNG de la scène courante du canvas, à sa
   résolution native (DPR compris).
2. **Enregistrer la vidéo** — capture en direct ce que le canvas montre
   (`canvas.captureStream()` + `MediaRecorder`) et télécharge le fichier à l'arrêt,
   avec le son du rejeu quand il est activé.

Succès = gates techniques verts (typecheck, vitest match-replay, eslint, `make test-web`)
+ gate visuel utilisateur (témoins listés en fin de plan — c'est l'utilisateur qui a la
main sur le navigateur, jamais un agent).

## Décisions produit — TRANCHÉES, ne pas rouvrir en cours d'exécution

1. **Pas de partage de lien** : écarté par l'utilisateur (accès par compte → portée trop
   faible).
2. **Capture image = téléchargement PNG uniquement.** Pas de presse-papiers en v1.
   Résolution = le backing store du canvas (déjà mis à l'échelle DPR).
3. **Vidéo = enregistrement temps réel de ce que l'écran montre.** Un clic démarre
   l'enregistrement (et LANCE la lecture si elle est en pause — un clip figé n'a pas de
   sens) ; un second clic arrête et télécharge. La **fin de lecture** (pause manuelle ou
   fin du film) arrête l'enregistrement et télécharge aussi : un seul chemin de sortie.
4. **Vitesse et scrub restent libres pendant l'enregistrement** : le fichier montre ce
   que l'écran montrait, y compris un passage en 2× — comportement « on filme l'écran »,
   assumé et documenté dans le hint i18n.
5. **Format vidéo** : premier type supporté dans cet ordre exact —
   `video/mp4;codecs=avc1` → `video/mp4` → `video/webm;codecs=vp9` → `video/webm`.
   L'extension du fichier suit le type retenu (`.mp4` / `.webm`).
6. **Son** : la piste audio est câblée AU DÉMARRAGE de l'enregistrement, si et seulement
   si le son du rejeu est alors activé (AudioContext existant). Activer le son en cours
   d'enregistrement ne l'ajoute pas au clip en cours.
7. **Navigateur sans `MediaRecorder`/`captureStream`** : le bouton vidéo ne se rend pas
   (détection de capacité). Le bouton image reste (toBlob est universel).
8. **Noms de fichiers** : `rejeu-<matchId>-<XmYYs>.png|mp4|webm` où `XmYYs` est le temps
   de match de l'instant de capture (début d'enregistrement pour la vidéo). Si aucun
   identifiant de match n'est joignable depuis le composant : repli
   `rejeu-<AAAAMMJJ-HHMMSS>.<ext>`.
9. **UI** : deux boutons icône dans `ReplayTransport`, entre les contrôles de son et le
   bouton réglages. SVG inline `currentColor` (convention du fichier), aucune librairie
   d'icônes, aucune couleur en dur. Libellés (aria-label + title) :
   - FR : « Capturer l'image » ; « Enregistrer la vidéo » / « Arrêter l'enregistrement ».
   - EN : "Capture image" ; "Record video" / "Stop recording".
   Pendant l'enregistrement, le bouton vidéo passe en `variant="default"` avec une icône
   d'arrêt (carré) — même patron d'état que lecture/pause.

## État des lieux (mesuré le 2026-08-26 — RE-VÉRIFIER sur pièces avant de coder)

- Un SEUL `<canvas>` : `ReplayCanvas.tsx:644` ; dessin à l'échelle DPR (l. 376-384),
  `CANVAS_HEIGHT = 480` (l. 90). Tout ce qui se voit est dans ce canvas ; les infobulles
  DOM ne seront PAS dans la capture — accepté.
- Boucle de lecture : `useReplayPlayback.ts` (rAF quand `playing`, arrêt sur l'état
  final). `playing` passe à `false` en fin de film — c'est le signal d'auto-arrêt du
  Lot 2.
- Son : `useReplaySound.ts` (couture React) + `replayAudio.ts` (`AudioContext` né au
  geste utilisateur ; `master.connect(ctx.destination)` l. 113 et l. 137,
  `distFilter.connect(ctx.destination)` l. 149 — les numéros ont pu bouger, re-vérifier).
- Barre de lecture : `ReplayTransport.tsx` (186 L — de la marge sous le seuil de 500).
- i18n de la feature : `i18n.ts` (tables FR/EN) + `i18nContract.ts` (UN champ par
  string, chaque champ porte sa justification en commentaire — respecter ce contrat).
- **Cliquet de taille sur `ReplayCanvas.tsx`** (`placementFamily.guard.test.ts`, huit
  extractions déjà imposées) : le câblage dans ce fichier doit rester MINIMAL — toute la
  logique vit dans de NOUVEAUX fichiers.
- Patron téléchargement existant : `URL.createObjectURL` déjà employé dans
  `queries.ts:93` (match-replay) et `lib/api/client.ts` (export CSV) — s'aligner.
- jsdom n'a NI `MediaRecorder`, NI `canvas.captureStream`, NI `toBlob` fiable → les
  tests mockent (patron : `useReplayPlayback.test.tsx` stubbe déjà rAF).
- Commentaires : suivre le style narratif FR du dossier (en-têtes qui expliquent le
  POURQUOI). Pas d'emojis. Aucune string UI en dur hors `i18n.ts`.

## Lot 0 — Amorce du worktree

- [x] `cd apps/web && npm install` (worktree neuf : pas de `node_modules`) — 503 paquets, code 0
- [x] `npm run typecheck` (apps/web) — vert AVANT toute modification (code 0)
- [x] `npx vitest run src/features/match-replay` — baseline verte (74 fichiers, 1127 tests, code 0)

**Gate Lot 0** : les trois commandes sortent en code 0. Sinon : STOP, consigner, rendre
la main (le socle est cassé, ce n'est pas ce chantier qui le répare).

## Lot 1 — Capture d'image

- [x] **PRÉALABLE IMPOSÉ PAR LE CLIQUET** (non prévu par le plan, cf. Découvertes D1) :
  `ReplayCanvas.tsx` était à 742 lignes PILE, soit son plafond exact — aucune addition
  n'était possible. Le CADRAGE (fond retenu/écarté, bornes de scène, largeur de dessin,
  amplitude verticale, projection partagée, trame d'altitudes) part dans
  `useReplayView.ts` — neuvième extraction imposée par ce cliquet, noms sortis inchangés
  donc zéro ligne de dessin touchée. 742 -> 706, cliquet abaissé à 706.
- [x] **`replayCapture.ts`** (nouveau, logique pure + DOM minimal) :
  - `buildCaptureFilename(matchId: string | null, matchMs: number | null, ext: string): string`
    — décision 8 ; horloge repli passée en paramètre (pas de `Date.now()` enfoui,
    testabilité).
  - `triggerDownload(blob: Blob, filename: string): void` — `createObjectURL` +
    `<a download>` + `revokeObjectURL` (aligné sur le patron existant).
  - `captureCanvasImage(canvas: HTMLCanvasElement): Promise<Blob | null>` — `toBlob`
    PNG ; `null` (canvas vierge/bloqué) = pas de téléchargement, pas de crash.
- [x] **Identifiant de match** : le document PORTE le sien — `ReplayDocument.matchId`
  (schéma généré l. 9420), conservé par `ReplayDocumentReady` (hors liste `Omit` de
  `replayNormalize.ts`). Rien à faire descendre de la route : `doc.matchId` est joignable
  depuis le composant. Le repli décision 8 reste câblé (identifiant vide ou instant
  illisible), et le choix est commenté dans `useReplayCapture.ts`.
- [x] **i18n** : champ `captureImage` dans `i18nContract.ts` (justification + raison
  écrite du refus d'un hint) + FR « Capturer l'image » / EN "Capture image" dans `i18n.ts`.
- [x] **`ReplayTransport.tsx`** : bouton icône appareil photo (SVG inline `currentColor`),
  aria-label + title. ÉCART : la prop n'est pas `onCaptureImage` mais `capture:
  ReplayCapture`, un objet — patron exact de `sound: ReplaySound`, imposé par le cliquet
  (cf. Découvertes D1 : les lots 2 et 3 doivent tenir sans ajouter UNE ligne au canvas).
- [x] **`ReplayCanvas.tsx`** : câblage minimal — 3 lignes (import, appel du hook, prop).
  Aucune logique dans le composant. ÉCART : le handler vit dans `useReplayCapture.ts`
  (hook annoncé par le Lot 2, né au Lot 1) plutôt qu'en ligne dans le canvas — même
  raison de cliquet.
- [x] **`replayCapture.test.ts`** : nom de fichier (nominal, matchId absent ou vide,
  minutes > 9, secondes < 10, instant illisible, identifiant exotique assaini) ;
  `triggerDownload` : create + ancre nommée + click + revoke (spies) ; `captureCanvasImage`
  rendant `null` (toile muette, `toBlob` absent).
- [x] **`useReplayCapture.test.tsx`** (ajouté au Lot 1 avec le hook) : `toBlob` rendant
  `null` → AUCUN téléchargement ; nom figé sur l'instant lu au clic ; toile non montée →
  ne fait rien, ne lève pas.

**Gate Lot 1** :
```
cd apps/web
npm run typecheck
npx vitest run src/features/match-replay
npx eslint src/features/match-replay --max-warnings 0
```
Trois sorties à 0, puis **commit** `feat(rejeu-capture): capture d'image PNG du canvas`.

**Résultat Lot 1** : `typecheck` = 0 ; `vitest src/features/match-replay` = 0 (76 fichiers,
1141 tests) ; `eslint src/features/match-replay --max-warnings 0` = **1** — 0 erreur, 1
warning `react-refresh/only-export-components` sur `ReplayFeedName.tsx:50`, fichier NON
touché par ce lot (`git diff --name-only` vide dessus) et warning de baseline que la
politique du dépôt assume (`eslint.config.js` le passe délibérément en `warn` : « refactor
possible plus tard »). Le lint du PÉRIMÈTRE du lot (11 fichiers touchés ou créés) sort à 0
avec `--max-warnings 0`. Voir Découvertes D2.

## Lot 2 — Enregistrement vidéo (image seule)

- [x] **`replayRecording.ts`** (nouveau, logique PURE) :
  `pickVideoMimeType(isSupported)` — ordre exact de la décision 5, `null` si rien n'est
  supporté. Ajouts assumés dans le même fichier : `CAPTURE_FPS` (30, avec sa raison écrite),
  `isVideoTypeSupported` (interroge le navigateur sans lever) et `canRecordCanvas`
  (les TROIS conditions de la décision 7 en un seul endroit).
- [x] **`useReplayCapture.ts`** (le hook, né au Lot 1, étendu ici) :
  - [x] état `recording: boolean` ;
  - [x] `startRecording()` : `canvas.captureStream(CAPTURE_FPS)` → `MediaRecorder`, chunks
    sur `dataavailable`, nom FIGÉ au démarrage sur le temps de match ; lecture relancée si
    en pause (décision 3) ;
  - [x] `stopRecording()` : `recorder.stop()` → assemblage et `triggerDownload` dans
    `onstop` (la dernière tranche arrive APRÈS le retour de `stop()`) ; les pistes du flux
    ne se coupent qu'à ce moment-là, sans quoi le clip perdrait sa fin ;
  - [x] **auto-arrêt** sur la RETOMBÉE de `playing` (transition vraie → fausse, jamais
    l'état : au démarrage depuis une pause `playing` vaut encore `false` le temps d'un
    rendu, et lire l'état refermerait le clip aussitôt ouvert) ;
  - [x] `recordingSupported: boolean` — nommé ainsi et non `supported` (l'objet porte aussi
    la capture d'image, qui elle est toujours possible) ; couvre `MediaRecorder`,
    `captureStream` ET l'existence d'un conteneur accepté (décision 7) ;
  - [x] garde de démontage (`liveRef`) : l'unmount referme l'enregistreur sans déposer de
    fichier.
- [x] **`ReplayTransport.tsx`** : bouton REC à état (disque plein / carré d'arrêt, `variant`
  selon `recording`, `aria-pressed`, nom accessible = ce que le CLIC fera) ; rendu seulement
  si `capture.recordingSupported`.
- [x] **i18n** : `recordVideo` / `stopRecording` / `recordHint` (l'infobulle porte les
  décisions 3 et 4) dans `i18nContract.ts` + FR/EN dans `i18n.ts`.
- [x] **`ReplayCanvas.tsx`** : ZÉRO ligne ajoutée — l'appel existant s'ÉTEND
  (`playing`, `play: togglePlay`). Le cliquet reste à 706 sans être touché.
- [x] **`useReplayCapture.test.tsx`** : `pickVideoMimeType` (mp4 prioritaire, repli vp9,
  aucun → null) ; cycle avec `FakeMediaRecorder` (start → `recording`, conteneur retenu,
  stop → un seul download, bonne extension, pistes coupées après la dernière tranche) ;
  repli WebM ; auto-arrêt sur retombée de `playing` ; démarrage sur pause (relance + pas de
  fermeture immédiate) ; `recordingSupported=false` quand l'API manque, l'image marchant
  toujours ; unmount en cours → pas de download.

**Gate Lot 2** : mêmes commandes que le gate Lot 1, sorties à 0, puis **commit**
`feat(rejeu-capture): enregistrement video du rejeu (MediaRecorder)`.

**Résultat Lot 2** : `typecheck` = 0 ; `vitest src/features/match-replay` = 0 (76 fichiers,
1154 tests) ; `eslint` dossier = 1 (même unique warning de baseline sur `ReplayFeedName.tsx`,
cf. D2), `eslint --max-warnings 0` sur les 8 fichiers du périmètre = 0.

## Lot 3 — Le son dans la vidéo

- [ ] **`replayAudio.ts`** : méthode `recordingTrack(): MediaStreamTrack | null` — crée
  une seule fois un `MediaStreamAudioDestinationNode` et y connecte les MÊMES nœuds que
  ceux qui se connectent à `ctx.destination` (master aux deux points + distFilter —
  re-vérifier les sites exacts sur pièces). Le nœud n'existe que si le lecteur existe.
- [ ] **`useReplaySound.ts`** : exposer `recordingTrack: () => MediaStreamTrack | null`
  dans l'interface `ReplaySound` (`null` si son coupé ou lecteur non né — décision 6).
- [ ] **`useReplayCapture.ts`** : au démarrage, si une piste audio est rendue →
  `new MediaStream([pisteVideo, pisteAudio])` avant le `MediaRecorder`.
- [ ] **Tests** : regarder comment les tests existants de `replayAudio`/`useReplaySound`
  mockent Web Audio et s'aligner ; au minimum — le hook assemble un flux à 2 pistes
  quand une piste audio est fournie, 1 piste sinon (piste factice en test).

**Gate Lot 3** : mêmes commandes, sorties à 0, puis **commit**
`feat(rejeu-capture): le son du rejeu rejoint la video enregistree`.

## Lot 4 — Clôture

- [ ] `cd apps/web && npm run lint` (repo web entier) — zéro erreur nouvelle
- [ ] `make check-types` et `make test-web` depuis la racine du worktree — verts
- [ ] Seuils : chaque nouveau fichier ≤ 500 L, chaque fonction ≤ 80 L, ≤ 5 paramètres
- [ ] `git diff f4fcbfa72 -- apps/web` relu : aucune couleur hex / classe Tailwind
  couleur, aucune string UI hors i18n, aucun emoji
- [ ] Entrée datée dans `.ai/thought_log.md` (règle du dépôt : sinon tâche non terminée)
- [ ] Chaque item de CE fichier statué `[x]` / `[~]` / `[!]` — aucune case vide
- [ ] **Commit final** `docs(rejeu-capture): plan statue et journal` (plan + journal)

**Gate Lot 4** : commandes vertes + plan entièrement statué. PAS de push, PAS de merge —
la suite appartient à l'utilisateur après le gate visuel.

## Gate visuel — utilisateur (après les 4 lots, hors périmètre agent)

Témoins à vérifier à la main dans le navigateur :

1. Capture d'image pendant la lecture ET en pause → PNG net (résolution DPR), bien nommé.
2. Clip d'environ 15 s → fichier lisible dans un lecteur local, vitesse fidèle.
3. Enregistrement avec son activé → la piste audio est présente et synchrone.
4. Fin du film pendant un enregistrement → téléchargement automatique.
5. Les deux boutons : libellés FR et EN, focus clavier, thèmes sombre et clair.

## Découvertes (hors périmètre — consigner ici, NE PAS traiter)

**D1 — le cliquet de taille était à ZÉRO marge, pas « de la marge » (Lot 1, TRAITÉ car il
bloquait le gate).** L'état des lieux annonçait le cliquet sans mesurer la marge :
`ReplayCanvas.tsx` faisait 742 lignes pour un plafond de 742. Aucun câblage, si minimal
soit-il, ne pouvait entrer. La doctrine du garde-rail lui-même (« le franchir se corrige en
extrayant, pas en relevant le nombre », huit extractions narrées dans son en-tête) donne la
manœuvre : extraire d'abord. D'où `useReplayView.ts` et le plafond abaissé 742 -> 706.
Conséquence de conception, à ne pas défaire : le canvas ne gagne AUCUNE ligne aux lots 2 et
3 — le hook rend un objet unique et l'appel s'ÉTEND au lieu de s'allonger.

**D2 — le gate `eslint --max-warnings 0` est plus strict que le dépôt.** Le script du dépôt
est `eslint .` sans seuil, et `eslint.config.js` passe délibérément
`react-refresh/only-export-components` en `warn` avec la note « refactor possible plus tard
(split en *.types.ts / *.utils.ts) ». Le journal du 2026-08-26 acte la même lecture
(« eslint 0 erreur (21 warnings pre-existants) »). `ReplayFeedName.tsx:50` porte un de ces
warnings depuis avant ce chantier : le gate littéral du plan ne peut donc pas sortir à 0
sans un refactor hors périmètre que la politique du dépôt a explicitement différé. NON
TRAITÉ. Critère retenu à chaque lot : 0 erreur sur le dossier, ET 0 problème
(`--max-warnings 0`) sur les fichiers du périmètre.

**D3 — collision de nom à surveiller.** `canvasRecording.test.ts` existe déjà dans la
feature et NE PARLE PAS d'enregistrement vidéo : c'est le test du « contexte enregistreur »
(la doublure qui empile les primitives de dessin). Les fichiers de ce chantier s'en
écartent (`replayCapture`, `replayRecording`, `useReplayCapture`). Aucune action.

## Reprise de session

L'avancement = les cases de ce fichier + `git log --oneline` du worktree. Reprendre au
premier item non statué du premier lot non clos. Le contrat reste `plan-execution`.
