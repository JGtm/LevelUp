# Plan — Retours utilisateur du 2026-08-27 sur le rejeu 2D (socles, drapeau, MA40, page)

> Ecrit le 2026-08-27 (session de planification, branche `feat/v75` occupee — AUCUN code
> touche par cette session). Contrat d'execution : skill `plan-execution` (ordre strict, un
> lot a la fois, statuts `[x]`/`[~]`/`[!]`, zero fix hors perimetre). Grille appliquee a la
> redaction : skill `plan-review`.
>
> Sujet utilisateur (message du 2026-08-27, 5 points) :
> 1. socles de power-up / armes speciales : icone AVEC contour et AU-DESSUS du losange (pas
>    dedans), losange PLEIN, etat « incertain » a etudier (« si pas de joueurs ont pickup
>    c'est que l'etat n'a pas du changer »), compteur pas toujours visible ;
> 2. drapeau : contour, plus gros, CLIGNOTE hors de son socle ; crane : pareil ; effet
>    « mini onde de choc » a la capture ;
> 3. MA40 = arme automatique : un fire event du film = une rafale de 3 balles a l'oreille
>    (comportement par defaut attendu pour les automatiques) ;
> 4. scroll fantome sur la page rejeu (« je peux descendre alors qu'il n'y a rien ») ;
> 5. rappeler carte / mode / date sur la page rejeu, comme sur la page match.

## Objectif et critere de succes

Quatre lots livrables independamment, tous cote web sauf une option du lot C. Succes global :
les 5 retours traites ou statues, verification A l'ECRAN (gate navigateur) et A l'OREILLE
(gate d'ecoute utilisateur pour le lot C), suites `make check-types` + vitest match-replay
vertes, aucune modification de contrat API ni de SchemaVersion (AUCUNE re-cuisson des ~951
artefacts du cache — contrainte de conception, pas un hasard).

## Prerequis et branche

- **PREREQUIS BLOQUANT** : le lot « sons » en cours sur `feat/v75` (objectiveSound,
  replaySoundVariants, wavs, `cmd/weapon-sounds` — working tree non committe au 2026-08-27)
  doit etre committe/fusionne AVANT tout demarrage : les lots A/C/D touchent des fichiers
  qu'il modifie (`i18n.ts`, `replaySound.ts`, `useReplaySound.ts`, `ReplayCanvas.tsx`,
  assets sons). Verifier sur pieces : `git status` propre sur ces chemins.
- Branche d'execution : **`feat/replay-retours-0827`**, creee depuis la base a jour qui
  porte le lot sons (feat/v75 fusionne, ou main si la release est passee). 1 tache =
  1 branche, N commits (un commit par lot minimum).
- Ordre des lots : **D (rapide, page) -> A (socles) -> B (drapeau) -> C (MA40, gate
  d'ecoute)**. D d'abord : c'est le plus petit et il donne le cadre de verification
  navigateur aux deux suivants. Chaque lot est CLOS (gates passes, entree thought_log)
  avant d'ouvrir le suivant.

## Decisions par defaut (a confirmer au lancement, sinon elles s'appliquent telles quelles)

- **D1 — grammaire visuelle du socle** : pile verticale `compteur / icone / losange` (le
  losange reste A la position du socle ; l'icone au-dessus avec un ecart ; le compteur
  au-dessus de l'icone). Le losange est TOUJOURS PLEIN a l'encre de sa nature (A13) ;
  l'ETAT se lit sur : plein opacite 0,95 (present) ; plein attenue 0,55 + halo losange
  POINTILLE (incertain — le pointille garde son sens « non affirme ») ; plein tres attenue
  0,35 sans icone (vide). L'anneau-bordure A13 qui ENFERMAIT l'icone est SUPPRIME (c'est
  lui que l'utilisateur lit comme « l'icone dans le losange ») — regle 7, zero code mort.
- **D2 — raffinement de l'etat incertain** : dans la fenetre [tLow, tHigh), la presence est
  PROLONGEE jusqu'a la premiere image ou UN joueur vivant passe a moins de R metres du
  socle (defaut R = 1,5 m, calibre par la mesure A2). Aucune approche sur toute la
  fenetre : le socle reste PLEIN jusqu'a tHigh (la these utilisateur : sans pickup
  possible, l'etat n'a pas change), l'absence prouvee a tHigh reprenant ensuite ses
  droits. Calcul CLIENT (fonction pure sur tracks + pads du document) — aucun changement
  d'artefact.
- **D3 — compteur** : la cible du compte a rebours devient la PROCHAINE APPARITION MESUREE
  (`presence[i+1].t0` / `spawns`), exacte car le rejeu connait la suite du film ; le cycle
  predictif (`cycle.medianS`) ne sert plus que de repli pour le DERNIER trou (aucune
  apparition suivante) ; ni l'un ni l'autre : rien d'affiche (honnetete conservee).
  Consequence : un compteur visible sur TOUS les trous entre deux occupations, y compris
  les 33 socles sur 57 sans cycle etabli. L'infobulle distingue « mesure » de « attendu ».
- **D4 — clignotement du drapeau** : etats `carried`, `carried_open`, `dropped` (= hors de
  la base) clignotent ; `home` est stable. Periode ~0,9 s, creux d'opacite qui ne descend
  jamais sous 0,35 (le glyphe reste localisable). `prefers-reduced-motion` : PAS de
  clignotement — opacite fixe pleine (regle des autres effets). La respiration actuelle du
  `dropped` est REMPLACEE par ce clignotement (pas les deux).
- **D5 — taille et contour du drapeau** : contour = meme technique que les vignettes de
  socle (forme reposee a l'encre du fond autour du trace, `ink.outline` deja resolue par le
  canvas) ; taille `FLAG_GLYPH_SCALE` 1,2 -> 1,45 (~+20 % de plus, cotes ensemble, ancrage
  inchange, rayon de survol suit). A valider a l'ecran au gate.
- **D6 — onde de choc a la capture** : source = `doc.objectives`, stat `flag_captures`
  UNIQUEMENT ; position = celle de l'AUTEUR a la frame (`posOfPlayerAt`, patron des pulses) ;
  gate `filmClockTrusted` (meme regle que les pulses — un effet muet vaut mieux que faux) ;
  rendu = double anneau qui s'ouvre a l'encre d'equipe (~600 ms) ; reduced-motion : anneau
  statique qui s'eteint. Le crane et les zones ne sont PAS concernes (pas de « capture »
  drapeau chez eux).
- **D7 — rafale MA40** : implantation A LA LECTURE, pas dans l'asset : une table
  `WEAPON_BURST_SPECS` (stem -> {coups: 3, ecartMs}) ; a l'instant de tir, le lecteur
  programme 3 departs du MEME fichier sur UNE SEULE enveloppe/voix logique (le plafond
  SOUND_MAX_VOICES compte 1, pas 3), chaque coup avec SON tirage de variation RANGED —
  c'est le comportement du moteur du jeu (variation par balle), qu'un asset cuit figerait.
  L'alternative « re-cuire l'asset en rafale » est refusee : elle contredirait la doctrine
  « coups reconstitues » (un fichier = UN coup) et gelerait la variation interne. TENSION
  ASSUMEE avec la decision du 2026-08-16 (« la duree est portee par le fichier ») : une
  rafale n'allonge pas un son, elle rejoue un geste — a confirmer au lancement.
  Perimetre initial : `hinf_ma40_ar` SEUL ; extension aux autres automatiques APRES
  l'inventaire C1 et le gate d'ecoute (les votes priment, RECETTE_SONS_ARMES §5).
- **D8 — scroll** : correctif MINIMAL des hauteurs fantomes d'abord (diagnostic D1 du lot) ;
  la refonte « hauteur de canvas responsive » (CANVAS_HEIGHT = 480 constant) est HORS
  PERIMETRE — notee en decouverte si le diagnostic la designe comme seule vraie solution.
- **D9 — rappel du match** : une ligne compacte sous le titre de la page rejeu :
  `« {Mode} sur {Carte} · {date} »` + badge playlist si present — memes champs et meme
  helper que la page match (`matchView.header.map_ui/mode_ui/start_time_label/
  playlist_label` + `buildMatchHeadingStr`), deja en cache (meme query key). Aucune
  nouvelle requete, aucune nouvelle cle i18n (libelles serveur deja localises).

## LOT D — Page rejeu : scroll fantome + rappel carte/mode/date (rapide)

Fichier central : `apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx`.
Etat des lieux (exploration du 2026-08-27) : page = pile de hauteurs fixes ~678 px
(`p-6`, titre h-8, bandeau 36, carte canvas ~546 dont `CANVAS_HEIGHT = 480` constant +
transport) dans le `main.overflow-y-auto` de `AppShell` (`h-screen overflow-hidden`) ;
colonne droite `absolute` a partir de `xl` (ne contribue pas a la hauteur), `min-h-[12rem]`
+ `max-h-[80vh]` sous `xl`.

- [ ] D-1 **Diagnostic sur pieces au navigateur** (`make dev`, preview) : reproduire le
      scroll vide a 1920x1080, 1366x768 et < 1280 px (`resize_window`), noter QUELLE regle
      cree l'exces (candidats releves : empilement fixe > viewport ; `min-h-[12rem]` de
      l'aside sous xl ; `max-h-[80vh]` ; bandes internes `CANVAS_PAD = 24`). Ecrire le
      verdict en commentaire du correctif.
- [ ] D-2 **Correctif minimal** : supprimer la ou les contraintes designees par D-1 (ex. :
      `min-h` inutile quand le contenu existe, marges additionnees), SANS toucher a
      `CANVAS_HEIGHT` ni a `AppShell` (portee = la page). Verification : plus aucun scroll
      quand tout le contenu tient ; scroll legitime conserve sous xl quand fiches + fil
      depassent reellement.
- [ ] D-3 **Rappel du match** (D9) : sous la ligne de titre, afficher
      `buildMatchHeadingStr(header.map_ui, header.mode_ui, locale)` + `start_time_label`
      + badge `playlist_label` — import du helper existant
      (`features/match-view/format.ts`), rendu conditionnel (`matchView` peut etre en vol :
      rien d'affiche tant qu'absent, jamais de squelette invente). Pas de logique metier
      dans le composant : composition de chaines par le helper existant uniquement.
- [ ] D-4 **Gate** : `make check-types` ; `npx vitest run src/features/match-replay
      src/routes` (depuis apps/web) ; ESLint sur les fichiers touches ; verification
      navigateur des trois largeurs de D-1 + capture d'ecran pour l'utilisateur.

## LOT A — Socles : visuel, etat incertain, compteur (lourd)

Fichiers : `weaponPadsLayer.ts`, `useReplayWeaponPads.ts`, `ReplayWeaponPadTip.tsx`,
`i18n.ts`/`i18nContract.ts` (infobulle), nouveaux `padPresenceRefine.ts` +
`pads_proximity_research_test.go` (mesure, cote Go, test de recherche sous garde env —
AUCUN code de production Go).

- [ ] A-1 **Visuel D1** (`weaponPadsLayer.ts`) : pile compteur/icone/losange ; losange
      toujours plein a l'encre de nature ; halo pointille pour incertain ; suppression de
      l'anneau-bordure A13 (constantes `PAD_BORDER_*` comprises) ; l'icone garde son
      lisere 8 directions ET le gagne aussi pour les images finies (non-masques) : le
      lisere n'exige que la SILHOUETTE — `tintedIconCanvas(im, ink.outline)` la rend pour
      toute image a alpha, la restriction actuelle `outline: null` saute
      (`useReplayWeaponPads.ts`). Zone de survol etendue a la pile (padAt : rayon couvrant
      losange + icone). Tests : `weaponPadsLayer.test.ts` (ordre de pile, etats, survol),
      guard `weaponPads.guard.test.ts` relu et ajuste sur pieces.
- [ ] A-2 **Mesure du raffinement D2** (AVANT d'implanter — seuils avant mesure) : test de
      recherche Go sous garde env (patron `ground_weapon_pads_research_test.go`, corpus
      des artefacts locaux) qui rejoue la regle « premiere approche a moins de R » sur les
      occupations ACHEVEES : (a) part des occupations videes SANS aucune approche dans
      [tLow, tHigh] (contradiction) a R = 1,0 / 1,5 / 2,0 m ; (b) distribution de l'offset
      de premiere approche. SEUIL : la regle s'implante si la contradiction <= 5 % au R
      retenu ; sinon NEGATIF ecrit ici et A-3 passe `[!]`. Denominateurs publies dans le
      rapport de lot.
- [ ] A-3 **Implantation D2 cote client** : `padPresenceRefine.ts` (pur) — pour chaque
      occupation, `tLowRefined` = derniere image avant la premiere approche (ou tHigh si
      aucune) ; `padStateAt` consomme les intervalles raffines ; memoisation par document
      dans `useReplayWeaponPads` (une passe sur les fenetres, ~1 200 images x joueurs par
      occupation). L'infobulle dit l'etat raffine. Tests unitaires purs (approche unique,
      approches multiples, aucune approche, fenetre jamais fermee tHigh <= tLow).
- [ ] A-4 **Compteur D3** : `padRespawnSecondsAt` vise `presence[i+1].t0` (mesure) ; repli
      cycle sur le dernier trou ; libelles infobulle distincts mesure/attendu (`i18n.ts` +
      `i18nContract.ts`, FR **et** EN, parite typee). Tests : trou median (mesure), dernier
      trou avec cycle (attendu), dernier trou sans cycle (rien), socle jamais vide.
- [ ] A-5 **Gate** : `make check-types` ; vitest match-replay complet ; ESLint fichiers
      touches ; verification navigateur sur un film avec socles de power-up (Catalyst des
      temoins) : pile lisible aux deux themes, compteur present sur chaque trou, capture
      d'ecran a l'utilisateur. Mesure A-2 : rapport chiffre dans l'entree thought_log.

## LOT B — Drapeau : contour, taille, clignotement, onde de choc ; crane par heritage (moyen)

Fichiers : `flagCarriesLayer.ts`, `useReplayFlagCarries.ts`, nouveau `flagCaptureFx.ts`
(pur), tests ; amendement de `PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md` (crane).

- [ ] B-1 **Contour + taille D5** : `drawFlagGlyph` gagne un lisere a l'encre du fond
      (parametre `outline` ajoute au style du calque, servi par `useReplayFlagCarries`
      depuis les tokens deja resolus du canvas — regle color-tokens inchangee : ce fichier
      ne connait aucun token) ; `FLAG_GLYPH_SCALE` 1,2 -> 1,45, cotes et rayon de survol
      ensemble. Tests de trace existants ajustes.
- [ ] B-2 **Clignotement D4** : fonction pure `flagBlinkAlpha(state, frame, reducedMotion)`
      remplacant `alphaOf` (les deux etats portes + `dropped` clignotent, `home` stable,
      reduced-motion fixe) ; la respiration `BREATH_*` du dropped est SUPPRIMEE (remplacee,
      regle 7). Tests : les 4 etats x reduced-motion, bornes d'opacite (jamais < 0,35).
- [ ] B-3 **Onde de choc D6** : `flagCaptureFx.ts` — `buildFlagCaptureFx(doc)` (pur :
      filtre `flag_captures`, gate `filmClockTrusted`, position auteur via
      `posOfPlayerAt`) + `drawFlagCaptureFx(ctx, fx, view, win, style, reducedMotion)`
      (double anneau ~600 ms). Cable dans `useReplayFlagCarries.paint` (aucune ligne
      ajoutee a `ReplayCanvas` — dette de taille gelee par cliquet). NOTE : ne PAS
      reactiver les pulses retires (`flagPulsesRetired`) — cet effet est distinct (capture
      seule, position de l'auteur, pas « l'element le plus proche »). Tests purs :
      construction (horloge non fiable = vide, stat non-capture ignoree), fenetre d'age.
- [ ] B-4 **Crane** : le calque du crane N'EXISTE PAS ENCORE (aucun `ballCarries` dans le
      document — verifie sur pieces le 2026-08-27) ; il arrive avec l'item 4
      (`PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`, phases 1-3, decision 7). Amender la
      decision 7 de ce plan : le glyphe du crane REUTILISE les helpers du present lot
      (contour, clignotement hors socle, echelle) — ecrire le renvoi explicite, pas une
      copie. Statut attendu ici : `[~]` (couvert par l'item 4, reference posee).
- [ ] B-5 **Gate** : `make check-types` ; vitest match-replay ; ESLint ; verification
      navigateur sur un film CTF temoin (`64e8adfa` ou `530820e5`) : clignotement visible
      aux deux themes, onde a chaque capture, reduced-motion respecte (emulation devtools) ;
      capture d'ecran a l'utilisateur.

## LOT C — MA40 : un fire event = une rafale de 3 balles (moyen, gate d'ecoute)

Fichiers : `replayAudio.ts` (departs multiples sur une enveloppe), `useReplaySound.ts`
(passage du spec de rafale au play), nouvelle table dans `weaponSoundVariations.ts` ou
fichier voisin `weaponBurstSpecs.ts`, tests `replayAudio.test.ts` + logique pure.

- [ ] C-1 **Inventaire ferme des automatiques** (AVANT tout code) : pour chacune des 26
      armes livrees (`WEAPON_SOUND_STEMS`), noter : mode de tir en jeu (auto / semi /
      salve / rayon), contenu actuel de l'asset (1 coup ou salve — ecoute + forme d'onde),
      cadence MESUREE des fire events du film (ecart median intra-rafale par arme, sur les
      artefacts locaux — patron du registre des reports, Cremateur 1 400 ms / Calcineur
      800 ms deja mesures). Sortie : tableau dans le rapport de lot. Candidats attendus au
      comportement rafale : MA40, MA5K Avenger, VK78 Commando, Needler ; cas a trancher a
      l'ecoute : BR75 et Pulse Carbine (salves natives — leur asset contient peut-etre
      deja la salve), Sentinel Beam (rayon : exclu).
- [ ] C-2 **Mecanique de lecture D7** : `ReplayAudioPlayer.play` accepte un
      `burst?: { count: number; gapMs: number; drawEach: () => SoundDraw }` — N
      BufferSources programmes sur la MEME chaine gain/enveloppe (1 voix logique au
      plafond SOUND_MAX_VOICES), chaque depart avec son tirage RANGED. `useReplaySound.tick`
      passe le spec quand le stem est dans `WEAPON_BURST_SPECS`. Ecart initial MA40 :
      cadence C-1 / 3, ajuste a l'oreille.
- [ ] C-3 **Table initiale** : `WEAPON_BURST_SPECS = { hinf_ma40_ar: { coups: 3,
      ecartMs: <C-1> } }` — MA40 seul, les autres attendent le vote (D7). Garde-rail : un
      test qui verifie que chaque cle de la table existe dans `WEAPON_SOUND_STEMS` (une
      rafale sur un stem non livre est un bug de table).
- [ ] C-4 **Gate technique** : vitest (mock AudioContext existant `test/fakeAudio.ts` :
      N departs, 1 voix comptee, ecarts respectes, variation par depart distincte) ;
      `make check-types` ; ESLint.
- [ ] C-5 **Gate d'ecoute utilisateur** (bloquant, les votes priment) : A/B sur un film
      dense en MA40 a 1x — rafale contre etat actuel ; l'utilisateur valide l'ecart en ms
      et decide de l'extension aux autres candidats de C-1 (chaque extension = une ligne
      de table + une ligne de rapport, pas un nouveau mecanisme).

## Ce que ce plan NE fait PAS (perimetre ferme)

- Aucun changement d'artefact, de contrat API, de SchemaVersion, de re-cuisson du cache.
- Pas de refonte de la hauteur du canvas (`CANVAS_HEIGHT`) — decouverte si D-1 la designe.
- Pas de calque crane (item 4), pas de sons de rafale pour d'autres armes sans vote C-5.
- Pas de publication du ramasseur (`PadPickup.XUID` reste null — l'oracle plafonne a
  79,7 %, seuil 90 % non rebaisse, condition de reprise au registre des reports).
- Aucune retouche de `ReplayCanvas.tsx` au-dela du strict cablage si un ink manque (le
  cliquet de taille fait foi).

## Gates transverses de cloture (apres le dernier lot)

- `make check-types` ; `make test-web` (suite complete) ; ESLint 0 sur les fichiers
  touches ; `make gate-push` avant merge vers main (la CI reste le gate d'autorite).
- Si C option Go retenue malgre D7 (asset) : `gofmt`/`go vet ./apps/go-api/cmd/weapon-sounds/...`
  + `replaySoundAssets.guard.test.ts` verte apres regeneration.
- Entree `thought_log.md` PAR LOT (statut, decision, mesures, gates) — l'absence d'entree
  = lot non termine.
- Demander a l'utilisateur avant tout commit (regle 16) ; jamais de push main direct
  (deploiement prod automatique).

## Protocole de reprise de session

Lire : cette page (statuts des cases), la derniere entree `thought_log.md` du chantier,
puis `git log --oneline -10` sur `feat/replay-retours-0827`. Reprendre a la PREMIERE case
non statuee du lot ouvert — jamais deux lots ouverts a la fois. Les decouvertes hors
perimetre se consignent ci-dessous, pas en code.

## Decouvertes (a remplir en execution, ne pas traiter)

- (exploration 2026-08-27, deja connue) `buildMatchHeadingStr` (`match-view/format.ts`) et
  `buildMatchHeading` (`components/ui/match-card.tsx`) sont deux copies de la meme regle —
  3e copie interdite par la regle 6 : si D-3 devait en creer une, centraliser AVANT.
- (exploration 2026-08-27) bandes vides internes du canvas (`CANVAS_PAD = 24` haut/bas
  comptees dans les 480 px) — candidate D-1, sinon decouverte.
