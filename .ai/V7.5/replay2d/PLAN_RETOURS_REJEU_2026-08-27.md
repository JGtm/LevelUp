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

- [x] D-1 **Diagnostic sur pieces au navigateur** (`make dev`, preview) : reproduire le
      scroll vide a 1920x1080, 1366x768 et < 1280 px (`resize_window`), noter QUELLE regle
      cree l'exces (candidats releves : empilement fixe > viewport ; `min-h-[12rem]` de
      l'aside sous xl ; `max-h-[80vh]` ; bandes internes `CANVAS_PAD = 24`). Ecrire le
      verdict en commentaire du correctif.
      STATUT : fait EN STATIQUE (aucun serveur/donnees dans cet env — la repro navigateur
      revient a l'utilisateur, cf. D-4). Verdict chiffre : pile rigide 678 px ; fantome =
      depassement <= marge basse (24 px) ; +32 px de vide pur sous xl via `min-h-[12rem]`
      dans le trou « rejeu arrive avant la vue du match » ; `max-h-[80vh]` et
      `xl:items-stretch` innocentes ; sous ~730 px de fenetre c'est la CARTE (546 px) qui
      deborde — scroll legitime, seule issue = hauteur adaptative (D8, en Decouvertes).
- [x] D-2 **Correctif minimal** : `min-h-[12rem] xl:min-h-0` supprimes (32 px de vide mort) ;
      marges de page `p-6` -> `px-6 py-3` (bande de vide au pire des cas 24 -> 12 px) ;
      hauteur de la ligne de rappel RESERVEE (`min-h-6`) si bien que la pile reste a 678 px
      — aucune fenetre qui tenait ne se met a defiler (constat P1 de la revue R1, corrige).
      `CANVAS_HEIGHT`, `AppShell`, mecanisme aside-absolue-des-xl : intacts.
- [x] D-3 **Rappel du match** (D9) : `ReplayMatchRecall.tsx` (presentation pure, 61 L) +
      branchement sous le h1 ; `buildMatchHeadingStr` importe de la page match (aucune 3e
      copie) ; badge playlist ; rien tant que la vue est en vol, hauteur reservee par la
      page (P2 « saut de 20 px » de la revue R1, corrige) ; zero cle i18n nouvelle.
- [x] D-4 **Gate** : `make check-types` VERT ; vitest `src/features/match-replay` 77
      fichiers / 1150 tests VERTS (+6 nouveaux dont l'orphelin symetrique du separateur,
      P2 revue R1) ; vitest `src/routes` 30 VERTS ; ESLint 0 sur les 3 fichiers.
      [!] Verification navigateur + capture : IMPOSSIBLE ici (pas de serveur ni de donnees
      dans l'env d'execution) — reportee a l'utilisateur, liste de controle au rapport de
      lot (1080p 125 %, <1280 px, rappel identique a la page match, arrivee tardive de la
      vue du match sans saut).

JOURNAL LOT D (2026-08-27) : livre par agent Opus, revue adversariale R1 (1 relecteur
frais) = 1 P1 + 2 P2 recevables, 11 conditions tiennent ; les 3 constats corriges (pile
constante par hauteur reservee + py-3 ; test mutant separateur) ; ronde 2 = verification
arithmetique superviseur (deterministe) + gates rejoues. Aucun commit (autorisation
utilisateur attendue).

## LOT A — Socles : visuel, etat incertain, compteur (lourd)

Fichiers : `weaponPadsLayer.ts`, `useReplayWeaponPads.ts`, `ReplayWeaponPadTip.tsx`,
`i18n.ts`/`i18nContract.ts` (infobulle), nouveaux `padPresenceRefine.ts` +
`pads_proximity_research_test.go` (mesure, cote Go, test de recherche sous garde env —
AUCUN code de production Go).

- [x] A-1 **Visuel D1** (`weaponPadsLayer.ts`) : pile compteur/icone/losange ; losange
      toujours plein a l'encre de nature ; halo pointille pour incertain ; suppression de
      l'anneau-bordure A13 (constantes `PAD_BORDER_*` comprises) ; l'icone garde son
      lisere 8 directions ET le gagne aussi pour les images finies (non-masques) : le
      lisere n'exige que la SILHOUETTE — `tintedIconCanvas(im, ink.outline)` la rend pour
      toute image a alpha, la restriction actuelle `outline: null` saute
      (`useReplayWeaponPads.ts`). Zone de survol etendue a la pile (padAt : rayon couvrant
      losange + icone). Tests : `weaponPadsLayer.test.ts` (ordre de pile, etats, survol),
      guard `weaponPads.guard.test.ts` relu et ajuste sur pieces.
- [x] A-2 **Mesure du raffinement D2** (AVANT d'implanter — seuils avant mesure) : test de
      recherche Go sous garde env (patron `ground_weapon_pads_research_test.go`, corpus
      des artefacts locaux) qui rejoue la regle « premiere approche a moins de R » sur les
      occupations ACHEVEES : (a) part des occupations videes SANS aucune approche dans
      [tLow, tHigh] (contradiction) a R = 1,0 / 1,5 / 2,0 m ; (b) distribution de l'offset
      de premiere approche. SEUIL : la regle s'implante si la contradiction <= 5 % au R
      retenu ; sinon NEGATIF ecrit ici et A-3 passe `[!]`. Denominateurs publies dans le
      rapport de lot.
      MESURE FAITE (`pads_proximity_research_test.go` + `_geometry_test.go`, env
      `PADS_PROX_CORPUS`, 30 artefacts, verifiee par revue adversariale independante avec
      reimplantation + sonde de falsification 0/5) : denominateur AUTORITAIRE = les 1064
      `padPickups` publies — contradiction 33,18 % (R=1 m), 32,05 % (1,5 m), 30,92 %
      (2 m) ; ligne de controle par Presence (992) : 28,93/27,82/26,71 %. SATURE avec R
      (23,21 % a 10 m base autoritaire). Distribution BIMODALE (mediane du passage le plus
      proche 0,36 m, p90 107 m). CONCENTRATION : 2 films (`084a804d`, `06dfe6d9`) portent
      ~72-75 % des contradictions ; hors eux ~10-13 % ; mediane par film 6,51 % — le
      MINIMUM de toutes les respécifications reste > 5 %.
- [!] A-3 **Implantation D2 cote client** : REFUSEE par la mesure A-2 (seuil 5 % non tenu
      par aucune lecture — minimum 6,51 %). La these « sans passage possible, l'etat n'a
      pas change » est contredite par la donnee ~1 fois sur 3 (base autoritaire), et ce
      n'est PAS un artefact de mesure (sonde 0/5). CONDITION DE REPRISE : enquete Go sur
      les occupations « achevees a tort » (les 2 films dominants + rabattement `lifeEnd`
      de NoLaterKF + fenetres qui chevauchent la reapparition suivante + ~10 % des
      contradictions resolues par ±6 s de bornes) — si un artefact assaini fait passer un
      corpus sous 5 %, rouvrir cet item. Rien d'implante (`padPresenceRefine.ts` n'existe
      pas), decision utilisateur possible en connaissance de cause.
- [x] A-4 **Compteur D3** : `padRespawnSecondsAt` vise `presence[i+1].t0` (mesure) ; repli
      cycle sur le dernier trou ; libelles infobulle distincts mesure/attendu (`i18n.ts` +
      `i18nContract.ts`, FR **et** EN, parite typee). Tests : trou median (mesure), dernier
      trou avec cycle (attendu), dernier trou sans cycle (rien), socle jamais vide.
- [x] A-5 **Gate** : `make check-types` VERT ; vitest match-replay 79 fichiers /
      1163 tests VERTS ; ESLint 0 sur les fichiers touches ; go vet + gofmt propres ;
      test de recherche : mesure avec env, skip propre sans. Mesure chiffree au
      thought_log. [!] Verification navigateur + capture : env sans serveur/donnees —
      reportee a l'utilisateur (pile lisible aux deux themes, compteur sur chaque trou
      referme, survol sur la vignette, infobulle « vue dans le film » vs « ≈ cycle »).

JOURNAL LOT A (2026-08-27) : livre par agent Opus ; revues adversariales = 2 relecteurs
frais paralleles (code : 0 P0/P1 + 7 P2, 22 conditions tiennent ; mesure : 2 P1 + 1 P2,
18 conditions tiennent, reproduction independante au chiffre pres, sonde 0/5). Les 10
constats corriges par l'auteur (mutations R1-5/R1-6 prouvees puis restaurees ; denominateur
bascule sur `padPickups` ; temoin recompte ; repartition par film publiee). Ronde 2 :
attendus des relecteurs reproduits exactement + verification superviseur sur pieces.
P0+P1 : 2 -> 0. Aucun commit (autorisation utilisateur attendue).

## LOT B — Drapeau : contour, taille, clignotement, onde de choc ; crane par heritage (moyen)

Fichiers : `flagCarriesLayer.ts`, `useReplayFlagCarries.ts`, nouveau `flagCaptureFx.ts`
(pur), tests ; amendement de `PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md` (crane).

- [x] B-1 **Contour + taille D5** : `drawFlagGlyph` gagne un lisere a l'encre du fond
      (parametre `outline` ajoute au style du calque, servi par `useReplayFlagCarries`
      depuis les tokens deja resolus du canvas — regle color-tokens inchangee : ce fichier
      ne connait aucun token) ; `FLAG_GLYPH_SCALE` 1,2 -> 1,45, cotes et rayon de survol
      ensemble. Tests de trace existants ajustes.
      NOTE REVUE R1 : le lisere du fanion CREUX est depose par ECRETAGE `clip('evenodd')`
      (strictement exterieur) — un trait centre comblait le creux a ~87 % (4,64 px²
      restants sur 21,1 avant lot), alors que le creux est devenu le seul signal de
      `carried_open`. 3 mutations prouvees (clip retire, evenodd->nonzero, restore
      supprime). `ReplayCanvas.tsx` : 1 ligne modifiee en place (0 ajoutee), cliquet
      742/742 intact.
- [x] B-2 **Clignotement D4** : fonction pure `flagBlinkAlpha(state, frame, reducedMotion)`
      remplacant `alphaOf` (les deux etats portes + `dropped` clignotent, `home` stable,
      reduced-motion fixe) ; la respiration `BREATH_*` du dropped est SUPPRIMEE (remplacee,
      regle 7). Tests : les 4 etats x reduced-motion, bornes d'opacite (jamais < 0,35).
- [x] B-3 **Onde de choc D6** : `flagCaptureFx.ts` — `buildFlagCaptureFx(doc)` (pur :
      filtre `flag_captures`, gate `filmClockTrusted`, position auteur via
      `posOfPlayerAt`) + `drawFlagCaptureFx(ctx, fx, view, win, style, reducedMotion)`
      (double anneau ~600 ms). Cable dans `useReplayFlagCarries.paint` (aucune ligne
      ajoutee a `ReplayCanvas` — dette de taille gelee par cliquet). NOTE : ne PAS
      reactiver les pulses retires (`flagPulsesRetired`) — cet effet est distinct (capture
      seule, position de l'auteur, pas « l'element le plus proche »). Tests purs :
      construction (horloge non fiable = vide, stat non-capture ignoree), fenetre d'age.
- [~] B-4 **Crane** (couvert par l'item 4 — reference posee, decision 7 amendee le
      2026-08-27 : heritage par IMPORT de `flagBlinkAlpha`, du lisere et de
      `flagCaptureFx`) : le calque du crane N'EXISTE PAS ENCORE (aucun `ballCarries` dans le
      document — verifie sur pieces le 2026-08-27) ; il arrive avec l'item 4
      (`PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`, phases 1-3, decision 7). Amender la
      decision 7 de ce plan : le glyphe du crane REUTILISE les helpers du present lot
      (contour, clignotement hors socle, echelle) — ecrire le renvoi explicite, pas une
      copie. Statut attendu ici : `[~]` (couvert par l'item 4, reference posee).
- [x] B-5 **Gate** : `make check-types` VERT ; vitest match-replay 80 fichiers /
      1200 tests VERTS ; ESLint 0 (9 fichiers) ; cliquet `placementFamily.guard` VERT
      (742/742). [!] Verification navigateur + capture : env sans serveur/donnees —
      reportee a l'utilisateur (clignotement ~1 s aux deux themes ; fanion de
      `carried_open` et de base vide CREUX AU CENTRE — un petit triangle plein cerne
      serait le symptome du bug corrige en revue ; onde a chaque capture a l'endroit du
      capteur, encre de son camp ; reduced-motion : opacite fixe, anneau sans expansion).

JOURNAL LOT B (2026-08-27) : livre par agent Opus ; revue adversariale R1 (1 relecteur
frais) = 1 P1 (creux du fanion comble par son propre lisere centre — corrige par ecretage
`evenodd`, 3 mutations prouvees) + 1 P2 (2 docs restees a la grammaire « attenue » —
reecrites), 16 conditions tiennent (anti-doublon pulses/ondes dans les deux sens, encre
neutre sans exception, frame fractionnaire, cosinus verifie numeriquement, memoisation,
cliquet). P0+P1 : 1 -> 0. Ronde 2 : verification superviseur sur pieces (clip evenodd
present, 452 L < 500, tests cibles 66 verts). Aucun commit.

## LOT C — MA40 : un fire event = une rafale de 3 balles (moyen, gate d'ecoute)

Fichiers : `replayAudio.ts` (departs multiples sur une enveloppe), `useReplaySound.ts`
(passage du spec de rafale au play), nouvelle table dans `weaponSoundVariations.ts` ou
fichier voisin `weaponBurstSpecs.ts`, tests `replayAudio.test.ts` + logique pure.

- [x] C-1 **Inventaire ferme des automatiques** (AVANT tout code) : pour chacune des 26
      armes livrees (`WEAPON_SOUND_STEMS`), noter : mode de tir en jeu (auto / semi /
      salve / rayon), contenu actuel de l'asset (1 coup ou salve — ecoute + forme d'onde),
      cadence MESUREE des fire events du film (ecart median intra-rafale par arme, sur les
      artefacts locaux — patron du registre des reports, Cremateur 1 400 ms / Calcineur
      800 ms deja mesures). Sortie : tableau dans le rapport de lot. Candidats attendus au
      comportement rafale : MA40, MA5K Avenger, VK78 Commando, Needler ; cas a trancher a
      l'ecoute : BR75 et Pulse Carbine (salves natives — leur asset contient peut-etre
      deja la salve), Sentinel Beam (rayon : exclu).
- [x] C-2 **Mecanique de lecture D7** : `ReplayAudioPlayer.play` accepte un
      `burst?: { count: number; gapMs: number; drawEach: () => SoundDraw }` — N
      BufferSources programmes sur la MEME chaine gain/enveloppe (1 voix logique au
      plafond SOUND_MAX_VOICES), chaque depart avec son tirage RANGED. `useReplaySound.tick`
      passe le spec quand le stem est dans `WEAPON_BURST_SPECS`. Ecart initial MA40 :
      cadence C-1 / 3, ajuste a l'oreille.
- [x] C-3 **Table initiale** : `WEAPON_BURST_SPECS = { hinf_ma40_ar: { coups: 3,
      ecartMs: <C-1> } }` — MA40 seul, les autres attendent le vote (D7). Garde-rail : un
      test qui verifie que chaque cle de la table existe dans `WEAPON_SOUND_STEMS` (une
      rafale sur un stem non livre est un bug de table).
- [x] C-4 **Gate technique** : vitest (mock AudioContext existant `test/fakeAudio.ts` :
      N departs, 1 voix comptee, ecarts respectes, variation par depart distincte) ;
      `make check-types` ; ESLint.
- [!] C-5 **Gate d'ecoute utilisateur** (bloquant, les votes priment) : A/B sur un film
      dense en MA40 a 1x — rafale contre etat actuel ; l'utilisateur valide l'ecart en ms
      et decide de l'extension aux autres candidats de C-1 (chaque extension = une ligne
      de table + une ligne de rapport, pas un nouveau mecanisme).
      REPORT VALIDE (decision/ecoute que seul l'utilisateur possede). Protocole pret :
      films `000d5950` ou `084a804d` a 1x ; A/B en retirant la ligne `hinf_ma40_ar` de
      `weaponBurstSpecs.ts` ; ecarts candidats 33 ms (mesure/3, retenu) / 50 ms
      (compromis) / 100 ms (cadence reelle par balle, mais chevauche le tir suivant) ;
      extension candidate chiffree : pulse_carbine 112,5 ms (le plus legitime — salve de 3
      en jeu, asset a 1 coup), ma5k 88,9, needler 100,0, vk78 181,8 ; NE PAS etendre :
      br75 (asset deja a 3 coups -> 9) ni sentinel_beam (faisceau).

JOURNAL LOT C (2026-08-27) : livre par agent Opus (relance apres une premiere tentative
tombee sur la limite de session, arbre propre verifie). Mesure C-1 : 38 films / 39 749
tirs ; MA40 19 388 tirs, 1 417 salves, intervalle moyen par salve mediane 100,0 ms (p10
80, p90 133) -> ecart de rafale 33 ms ; contre-verification croisee par un second
instrument (`REPLAY_CORPUS`) au chiffre pres ; transitoires des 26 assets mesures (ancre
sniper = 1, br75 = 3, sentinel = 3, seuil stable a 25/35/50 %). DECOUVERTE PRODUIT : la
donnee ne soutient PAS « un fire event = 3 balles » — les tirs MA40 publies arrivent deja
a la cadence reelle de l'arme (100 ms) ; la rafale est une mise en scene tranchee par
l'utilisateur, d'ou le gate d'ecoute decisif. Revue adversariale R1 : 0 P0/P1 + 3 P2
(garde-rail sans la duree du fichier, branche morte, mutant du gain par balle), 19
conditions tiennent, zones interdites (constructeur/setDistance/destination — conflit
origin) intactes verifiees ligne a ligne ; 3 P2 corriges, mutations prouvees. Gates :
tsc vert, vitest match-replay 81 fichiers / 1215 verts, ESLint 0, go vet/gofmt propres,
`make test-web` COMPLET 488 fichiers / 4772 verts. P0+P1 : 0 -> 0. Aucun commit.

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
- (lot D + revue R1, 2026-08-27) SOUS ~730 px de fenetre utile (1366x768, 1080p a 150 %),
  la carte de 546 px deborde quoi qu'on fasse : la seule vraie solution est une hauteur de
  carte adaptative (`CANVAS_HEIGHT` pilote par la hauteur disponible) — designee par le
  diagnostic, HORS perimetre (D8), a decider avec l'utilisateur.
- (revue R1 lot D) la justification « fiches et fil tiennent en ~160 px sans matchView »
  etait fausse (les fiches se construisent depuis le FILM, `rosterLogic.buildPlayers`) —
  la conclusion (borne `min-h-[12rem]` inerte) tient pour une raison plus forte ; sans
  consequence, note pour memoire.
- (lot A, mesure + revue) ENQUETE GO A OUVRIR sur les occupations de socle « achevees a
  tort » cote artefact : 2 films (`084a804d`, `06dfe6d9`) concentrent ~72-75 % des
  contradictions de proximite (distances minimales medianes 18-25 m, 14/34 socles de
  `06dfe6d9` jamais approches a 2 m de tout le film) ; `gwPickupResolve`/NoLaterKF rabattu
  par `lifeEnd` (borne haute = fin de film SANS etre `Never`) ; une fenetre observee
  chevauche la reapparition suivante (`00162144` socle #10 : next t0=2628 < tHigh=2631) ;
  ±60 frames de bornes resolvent ~10 % des contradictions. C'est la CONDITION DE REPRISE
  d'A-3.
- (lot A) `PadStyle.ink` (encre neutre du terrain) n'est plus lu par le calque depuis A13
  (2026-08-26) — champ mort qui traverse le hook et les tests ; a retirer dans un lot
  d'hygiene avec son cablage.
- (revue lot A) la zone de survol des socles est un DISQUE : les flancs extremes d'une
  vignette tres large (rapport ~3) sortent de ~4 px a mi-hauteur — assume et documente
  dans `padAt` ; passer a un test rectangulaire si un retour utilisateur le demande.
- (lot B) `flagCarriesLayer.ts` est a 452/500 lignes et le cliquet de `ReplayCanvas.tsx`
  est SATURE (742/742) : l'implantation du crane (item 4) devra EXTRAIRE avant d'ajouter.
- (lot B) constantes de temps du calque drapeau en IMAGES (`BLINK_PERIOD_FRAMES`,
  `FLAG_CAPTURE_HOLD_FRAMES`, pas de 100 ms documente) quand d'autres effets convertissent
  des ms via `msToFrames` — heterogeneite a resorber dans un lot d'hygiene temporelle.
- (lot B) `PadStyle`/`FlagCarriesStyle` : `drawFlagGlyph` reste exportee sans appelant
  externe — export prepare pour le crane, reference par la decision 7 amendee du plan
  objectifs vivants (ce qui l'empeche d'etre du « au cas ou »).
- (lot C, mesure) l'asset `hinf_br75` porte DEJA 3 transitoires (salve native cuite dans
  le fichier) — contredit la doctrine affichee « un fichier = UN coup » de
  `replaySound.ts` ; a trancher hors lot (et interdit d'extension de rafale : 3x3 = 9).
- (lot C, mesure) la donnee ne soutient pas « un fire event = 3 balles » : les tirs MA40
  publies arrivent deja a la cadence reelle (100 ms/balle). La rafale est une mise en
  scene (decision utilisateur) — si l'ecoute C-5 la refuse, la retirer est UNE ligne.
- (lot C) `hinf_shock_rifle` : cadence par salve 33,3 ms, incompatible semi-auto —
  probablement des degats de chaine electrique comptes comme tirs ; piste de nommage.
- (lot C) famille hors registre `1833A5A8` : 118 tirs a 90 ms (signature d'automatique),
  muette par construction — candidate au lot de nommage des armes.
- (lot C) deux instruments de cadence coexistent (`REPLAY_CORPUS` /
  `WEAPON_CADENCE_CORPUS`) : 2e copie signalee, la 3e exigera un helper partage +
  garde-rail (regle 6). Et la constante 1,2 s (duree des assets d'arme) a desormais 3
  traces (2 proses + 1 constante de test) — un 4e usage exigera une constante partagee.
