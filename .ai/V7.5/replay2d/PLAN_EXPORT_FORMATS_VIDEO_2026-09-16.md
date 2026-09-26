# PLAN — Formats video standard de l'export du rejeu (2026-09-16)

Branche : `claude/video-export-standard-dimensions-64927b` (base `origin/feat/v75` = `da258bf76`,
fusion cible `feat/v75`). Worktree dedie. Contrat : skill `plan-execution`.

## Probleme

L'export video (`apps/web/src/features/match-replay/export/`) encode la toile du rejeu telle
qu'affichee : hauteur ~960 x devicePixelRatio (`exportScaleFor`, `useReplayView.ts`), largeur =
largeur de la fenetre x le meme facteur. Deux exports du meme match sortent en dimensions et
proportions differentes selon l'ecran — inutilisable pour le partage et le montage.

## Decisions validees par l'utilisateur (fermes)

- D1 : RENDU DIRECT AU FORMAT CIBLE pendant l'export (pas de bandes noires ajoutees apres coup,
  pas de rognage). Le fichier ne depend plus ni de la fenetre ni du DPR.
- D2 : deux formats 16:9 — `1080p` (1920x1080, defaut) et `720p` (1280x720). Pas de portrait
  9:16, pas de mode « taille de la fenetre », pas de 1440p/4K.
- D3 : pas de H.265 (gain negligeable sur des aplats 2D, lecture partielle Firefox/Discord).
- D4 : choix retenu DANS LE NAVIGATEUR via `settings/replayPreferences.ts` (pas de preference
  serveur). Valeur absente/invalide -> `1080p`.
- D5 : cadence 30 im/s et H.264 High inchanges.

## Etapes

### E1 — Catalogue des formats et geometrie cible (pur, teste)
- [x] Module pur des formats (id, largeur, hauteur ; libelle i18n construit par le dialogue depuis id + dimensions, E4) ; ids stables pour la persistance — `export/exportFormats.ts`.
- [x] Taille LOGIQUE de mise en page + facteur de rendu par format, independants du DPR — `EXPORT_LAYOUT` 960x540, `exportLayoutFor` (x2 / x4/3).
- [x] Tests unitaires (dimensions paires, 16:9 exact, id inconnu -> defaut) — `exportFormats.test.ts`, 6 tests.
- Gate : vitest des fichiers touches.

### E2 — Rendu direct au format cible pendant l'export
- [x] Pendant l'export, la toile est dimensionnee et mise en page au format cible (carte ajustee
      au 16:9, surimpressions/panneaux d'export peints dans ce cadre), DPR exclu. — magasin
      `export/exportLayoutStore.ts` (demandee/appliquee, `canvasPixelRatio`, `waitForExportLayout`) ;
      `useReplayView` remplace l'offre d'ecran par le cadre 960x540 (`frameBounds` elargit la scene
      au 16:9) ; `ReplayCanvas.draw` et `cookLayer` rendent a `canvasPixelRatio` ; `paintExportFrame`
      peint les panneaux dans le cadre logique puis le fond `--card` dessous (bandes 16:9).
- [x] Retour a l'etat normal garanti (finally) apres succes, echec ou annulation. — `requestExportLayout(null)`
      dans le `finally` de `run` ; tests succes, echec, annulation en cours.
- [x] L'affichage a l'ecran pendant l'export ne se deforme pas (ou le comportement est documente). —
      la boite CSS de la toile garde la taille d'ecran (`ReplayView.screen`) et `object-fit: contain`
      y inscrit l'image 16:9 : pas de saut de page, pas de deformation, bandes du fond du bloc si
      l'ecran n'est pas 16:9. Documente dans `useReplayView` et l'en-tete de `useReplayExport`.
      Non verifie dans un navigateur (gate visuel E5).
- [x] 720p : nettete du texte HUD verifiee (mesure du 2026-08-28 : sous-echantillonnage chroma) ;
      approche retenue (rendu direct 1280x720 ou rendu 1920x1080 reduit) justifiee par mesure ou
      par raisonnement ecrit dans le code. — RENDU DIRECT 1280x720 (cadre 960x540 a x4/3), justifie
      par raisonnement dans l'en-tete de `exportFormats.ts` (la perte chroma est une propriete de la
      grille de sortie ; la reduction n'ajoute aucun echantillon de couleur, floute la luminance et
      depend d'une mise a l'echelle non garantie). NON MESURE : a confirmer au gate visuel.
- [x] Tests : l'encodeur recoit exactement 1920x1080 / 1280x720 quel que soit DPR et taille de toile. —
      `useReplayExport.test.tsx` (describe « le format du fichier » : 4 couples format/DPR/toile,
      chaque image poussee, fond, retour, echec d'application) + `hooks/useReplayView.test.ts`.
- Gate : vitest export + hooks concernes ; seuils max-lines respectes (ReplayCanvas est au plafond).

### E2b — Cadrage de l'export (ajoute le 2026-09-16, decision utilisateur)
- D6 : choix « Cadrage » dans le dialogue, AFFICHE SEULEMENT si la carte est zoomee (palier > 1) :
  « Carte entiere » (defaut) / « Cadrage actuel (xN) ». Non memorise (contextuel).
- D7 : zoom et deplacement (croix, molette, clavier, glisser) BLOQUES pendant `prepare`/`encode` ;
  retour garanti a la fin (succes, echec, annulation).
- [x] « Carte entiere » : le clip est rendu a 1x sans modifier le zoom de l'utilisateur, restaure apres l'export. — `ExportLayout.framing` (`whole` par defaut) ; `useReplayView` calcule la fenetre visible au palier 1 quand `framing === 'whole'`, l'etat de `useReplayZoom` n'est jamais ecrit, donc intact au retour (`requestExportLayout(null)` dans le `finally`).
- [x] « Cadrage actuel » : centre + palier conserves dans le cadre 16:9. — `framing: 'current'` : palier et centre de l'ecran appliques a `frameBounds` elargi au 16:9 (teste : centre identique, fenetre 2x plus etroite que la carte entiere a 2x).
- [x] Gestes de cadrage inactifs pendant l'export (infobulles comprises : decouverte 1 soldee). — `lockedZoom` (hooks/useReplayZoom.ts) : `useReplayView` rend un zoom sans geste des que l'export est demande — croix (boutons desactives par `can*`), molette (`zoomAt`), clavier (`zoomIn/zoomOut`), glisser (`canPan`/`panBy`) passent tous par cet objet. Survol : `hoverHandlers` efface et ne distribue plus rien quand `isExportActive()`. La demande est posee en tete du `try` de `run` (des `prepare`).
- [x] Strings FR + EN ; tests (option masquee a 1x, defaut carte entiere, zoom restaure, gestes bloques). — `exportFraming`, `exportFramingWhole`, `exportFramingCurrentFmt` (« Cadrage actuel (1,5x) » / « Current view (1.5x) ») ; palier relaye ReplayCanvas -> useReplayCapture -> `ReplayExport.zoomLevel` (0 ligne ajoutee au canvas). Tests : dialogue (3), useReplayExport (cadrage transmis, tenue des la preparation, palier relaye, retour apres echec/annulation), useReplayView (3), `layers/hoverLayers.test.ts` (1), exportFormats (1).

### E2c — Apparence fixe de l'export (ajoute le 2026-09-16, decision utilisateur)
- D8 : fond de la video TOUJOURS NOIR PUR, quel que soit le theme — via un token semantique dedie
  (pas de hex dans `features/`, skill `color-tokens`).
- D9 : TOUT le rendu de l'export (carte, traits, HUD, panneaux, ecran de fin) utilise les couleurs du
  THEME SOMBRE, quel que soit le theme actif, SANS basculer le theme de la page.
- [x] Token de fond d'export (noir pur) defini et lu par l'export ; `--card` n'est plus peint dessous. — `--replay-export-backdrop: rgb(0 0 0)` dans `styles/globals.css` (meme valeur `:root`, sombre, clair ; patron `--replay-label-stroke`), `InkVar` de `canvasInk.ts`, lu une fois par export dans `run` et peint en `destination-over` (`OverlayPaintContext.backdrop`).
- [x] Encres de l'export resolues dans le theme sombre (overlay, calques cuits, scene), theme de la page intact. — `layers/themeInk.ts` : `readThemeVar` lit, pendant un export, la cascade du theme sombre DANS LES REGLES de la feuille (style en ligne > `:root[data-theme=dark]` > `:root`, `var()` resolus, table construite une fois par export), `getComputedStyle` sinon ; `data-theme` jamais touche. `readInk` et `readFxInk` passent par lui (seuls lecteurs d'encres dependantes du theme ; les `--ac-*` via `resolveToken` sont en ligne et identiques dans les deux themes). `useReplayInks` se recalcule a l'entree et a la sortie (`useExportActive`) : calques cuits (dependances d'encre), vignettes et scene suivent ; panneaux via `readOverlayInk` lu apres la demande. Vignettes : `layers/loadedImage.ts` (`withLoadedImage`, image deja chargee rendue synchroniquement) pour que la reteinte ait lieu dans le meme passage que le changement d'encres ; 4 copies migrees (grenades, armes au sol, socles, vehicules) + garde-rail `loadedImage.guard.test.ts`. Toile visible en rendu sombre pendant l'export : accepte, documente en tete de `useReplayExport.ts`.
- [x] Theme clair actif -> images d'export identiques a celles d'un export en theme sombre (test). — verifie au niveau des ENCRES, seule entree du rendu qui depend du theme (jsdom n'a pas de contexte 2D, aucune comparaison de pixels) : `themeInk.test.ts` injecte le vrai `globals.css` — encres de mise en page et teintes d'effet identiques au theme sombre, `useReplayInks` identique au memo calcule en sombre, retour aux encres claires, `data-theme` reste `light` ; `useReplayExport.test.tsx` : fond `rgb(0 0 0)` sur les 16 images et encres sombres lues pendant l'encodage, page claire intacte apres. Retour apres echec/annulation : `isExportActive()` faux (tests existants), donc lecture `getComputedStyle`.
- [x] `lint:colors` propre ; tests. — 0 violation.

### E3 — Persistance navigateur
- [x] Cle dans `replayPreferences.ts` (lecture validee contre le catalogue, ecriture sous try). — `EXPORT_FORMAT_KEY` = `replay-export-format`, `readExportFormat` (`readStoredChoice` sur `EXPORT_FORMAT_IDS`), `persistExportFormat` (`persistPreference`).
- [x] Garde-rail `replayPreferences.guard.test.ts` vert (aucun acces localStorage direct ailleurs).
- [x] Tests lecture/ecriture/valeur invalide. — describe « le format de l’export video (D4) » (3 tests) + cas storage indisponible.

### E4 — Selecteur dans le dialogue d'export
- [x] Choix du format dans `ReplayExportDialog.tsx`, desactive pendant l'export ; branche sur E3. — groupe radio `FormatChoice` (etat initialise par `readExportFormat`, `persistExportFormat` a chaque choix, `run(bounds, { sound, format })`). « Desactive » : le formulaire n'est pas rendu du tout pendant `prepare`/`encode` (`isExportBusy`), teste.
- [x] Strings FR + EN (contrat `i18nContract.ts`), FR sans anglicisme (« Format », « 1080p »). — `exportFormat`, `exportFormatOptionFmt` (« 1080p (1920 × 1080) ») ; `exportRunningHint` precise que le terrain defile « dans le cadre de la video ».
- [x] Tokens de couleur uniquement ; test du dialogue (selection -> export au bon format, memorise). — classes semantiques (`text-foreground`, `text-muted-foreground`, `accent-primary`), `npm run lint:colors` propre ; describe « le format du fichier » (4 tests).

### E5 — Gates et cloture
- [x] `make check-types`, lint web, `make test-web` (vitest HORS sandbox) verts. — equivalents executes depuis `apps/web` : `npm run typecheck` (tsc -b) exit 0 ; `npm run lint` 0 erreur (25 avertissements, aucun introduit) ; `npx vitest run` exit 0 : 715 fichiers verts + 1 skippe, 7707 tests verts + 17 skippes (skips preexistants : `ReplayTeams.perf.test.tsx`, `SynthesisPage.test.tsx`).
- [~] Journal ci-dessous + `.ai/thought_log.md`. — journal du plan tenu ; l'entree `thought_log` est prise en charge par le pilote du lot (consigne de delegation), pas par l'executeur.
- [x] Gate visuel : valide par l utilisateur le 2026-09-17 (« tout est bon ») — formats 1080p/720p, fond noir et encres sombres, cadrages, gestes bloques pendant l export.

## Decouvertes (non traitees, hors perimetre)

- [2026-09-16] RESOLUE PAR E2b (gestes eteints et survol muet pendant l'export). Rien n'empeche de zoomer / deplacer la carte (molette, clavier, croix, glisser)
  PENDANT un export : le cadrage change alors en cours de clip. Preexistant (l'export d'avant
  lisait deja `canvasView`). Depuis E2, le survol pendant un export projette aussi dans le cadre
  960x540 alors que la toile est affichee en `contain` : les infobulles peuvent viser a cote.
  Sans effet sur le fichier tant qu'on ne zoome pas. Piste : geler les gestes de cadrage pendant
  `prepare`/`encode`.
- [2026-09-16] `src/features/palmares/PalmaresRelationsPage.test.tsx` « rend le badge cross-jeu » a
  depasse le delai de 5 s une fois sur deux passes de la suite complete (vert isole). Flaky sous
  charge, sans lien avec ce lot. Revu apres E2b sur un autre test du meme fichier (« rend les badges
  solid »), meme delai de 5 s, passe suivante verte.

## Journal
- [2026-09-16] E1 close. `export/exportFormats.ts` (catalogue ferme 1080p/720p, cadre logique 960x540, densite x2 / x4/3, raisonnement 720p ecrit en en-tete) + `exportFormats.test.ts`. Gate : `npx vitest run src/features/match-replay/export/exportFormats.test.ts` -> 1 fichier, 6 tests verts.
- [2026-09-16] E2 close. Magasin `export/exportLayoutStore.ts` (remplace `exportRenderScale`), cadrage d'export dans `useReplayView` (+ `screen`), `ReplayCanvas` (densite + boite d'ecran `contain`, 0 logique ajoutee, 472 -> 473 lignes de code / 500), `cookLayer` a `canvasPixelRatio`, boucle d'export au format (attente d'application, encodeur ouvert au format, fond `--card`). Supprimes : `exportRenderScale`, `EXPORT_SUPERSAMPLE`, `EXPORT_TARGET_HEIGHT`, `exportScaleFor` et leurs 3 tests (useReplayViewport.test) + 2 tests du surechantillonnage (useReplayExport.test), remplaces par le describe « le format du fichier » et `hooks/useReplayView.test.ts`. Choix 720p : rendu direct (raisonnement ecrit, non mesure). Gate : `npx vitest run src/features/match-replay/hooks/useReplayView.test.ts src/features/match-replay/export/ src/features/match-replay/hooks/useReplayViewport.test.ts src/features/match-replay/layers/sceneBinding.guard.test.ts src/features/match-replay/test/arborescence.guard.test.ts` -> 14 fichiers, 164 tests verts (avant l'ajout du test d'annulation ; useReplayExport.test relance ensuite : 24 verts) ; eslint des fichiers touches : 0 erreur (2 avertissements preexistants).
- [2026-09-16] E3 close. Cle `replay-export-format` et lecture/ecriture dans `settings/replayPreferences.ts`, tests associes. Gate : `npx vitest run src/features/match-replay/settings/replayPreferences.test.ts src/features/match-replay/settings/replayPreferences.guard.test.ts` -> 2 fichiers, 14 tests verts ; eslint des deux fichiers : 0.
- [2026-09-16] E4 close. Selecteur de format dans `ReplayExportDialog.tsx`, cles i18n FR/EN, tests du dialogue. Gate : `npx vitest run src/features/match-replay/export/ReplayExportDialog.test.tsx` -> 18 tests verts ; eslint dialogue + i18n : 0 erreur (1 avertissement preexistant `isExportBusy`) ; `npm run lint:colors` : 0 violation.
- [2026-09-16] E5 : gates executes. `npm run typecheck` exit 0 ; `npm run lint` 0 erreur / 25 avertissements ; `npx vitest run` 1re passe : 1 echec hors perimetre (`PalmaresRelationsPage.test.tsx`, delai 5 s depasse sous charge ; relance isolee 14/14 verte), 2e passe complete exit 0 (7707 verts, 17 skippes preexistants). `npm run lint:colors` propre. Reste : gate visuel utilisateur ([!]) et entree thought_log (pilote).
- [2026-09-16] Complement E5 : commentaires de `replayVideoEncoder.ts` (decision 1, `visibleRect`) remis a jour (ils decrivaient la toile a taille d ecran x DPR). Reverifie : `npx vitest run src/features/match-replay/export/replayVideoEncoder.test.ts` 17 verts, eslint 0, `npm run typecheck` exit 0.
- [2026-09-16] E2b close (decisions D6/D7). `ExportFraming` + `ExportLayout.framing` (exportFormats.ts), `isExportActive` (exportLayoutStore.ts), `lockedZoom` (useReplayZoom.ts) branche dans `useReplayView` avec la fenetre a 1x en « carte entiere », survol muet dans `hoverLayers.ts`, demande de mise en page avancee en tete du `try` de `run`, `zoomLevel` relaye jusqu'au dialogue, `FramingChoice` affiche seulement a palier > 1, non memorise. ReplayCanvas : 473 lignes de code / 500 (inchange, relais ajoute sur une ligne existante). Gates : vitest cible (`hooks/useReplayView.test.ts`, `layers/hoverLayers.test.ts`, `export/`, `ui/ReplayTransport.test.tsx`, `hooks/useReplayZoom|Drag|Shortcuts.test.ts`, `test/arborescence.guard.test.ts`) -> 17 fichiers, 224 tests verts (x3) ; `npm run typecheck` exit 0 ; `npm run lint` 0 erreur, 25 avertissements, ensemble identique a avant E2b ; `npm run lint:colors` 0 violation ; `npx vitest run` : 1re passe 1 echec `PalmaresRelationsPage.test.tsx` (delai 5 s, decouverte deja notee), 2e passe exit 0 (716 fichiers + 1 skippe, 7717 tests verts + 17 skippes preexistants).
- [2026-09-16] E2c close (D8/D9). Crees : `layers/themeInk.ts` (+ test), `layers/loadedImage.ts` (+ garde-rail). Modifies : `styles/globals.css` (token de fond), `canvasInk.ts`, `fxInk.ts`, `useReplayInks.ts`, `exportLayoutStore.ts` (`exportLayoutRequested`, `useExportActive`), `useReplayExport.ts` (fond noir, en-tete), `useGrenadeIcons.ts`, `useReplayGroundWeapons.ts`, `useReplayWeaponPads.ts`, `useReplayVehicles.ts`, `useReplayExport.test.tsx`. Aucun `[!]` : aucune partie du rendu n'a demande de refonte. Gates : `npx vitest run src/features/match-replay/layers/ src/features/match-replay/export/ src/features/match-replay/hooks/ src/features/match-replay/test/ src/features/match-replay/ui/` -> 103 fichiers + 1 skippe, 1478 verts + 3 skippes (perf preexistant) ; `npm run typecheck` exit 0 ; `npm run lint` 0 erreur, 25 avertissements, ensemble identique ; `npm run lint:colors` 0 violation ; `npx vitest run` exit 0 : 718 fichiers + 1 skippe, 7726 verts + 17 skippes preexistants.
- [2026-09-17] Gate visuel valide par l utilisateur ; plan clos. Fusion vers feat/v75.
