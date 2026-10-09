# Plan — Retours de la planche R3 (bilan utilisateur du 2026-08-18, apres-midi) -> lots

> Ecrit le 2026-08-18 par la session de pilotage. Bilan verbatim en annexe. Contrat `plan-execution`.
> Un worktree frere web `../LevelUp-wt-r3-visuels` (`wt/r3-visuels`), base `feat/v75` `f80b9dfd7`+.
> Valides sans action : A2, A5, A6, B1, B3, B4, B5, B7, D0-D4, E1, F1, W2, W3. Questions : frag 3,335 s
> GARDEE (D3 « Parfait »), mur = arc GARDE (W1) + couleur `warning` (R2-5 valide).

## Decisions tranchees (superviseur, a partir du bilan)

1. Tout en tokens (`color-tokens`) ; les items VALIDES ne bougent pas ; ce qui est « a proposer » va sur la
   planche (recette scratchpad `assemble.cjs`/`app.js`/`fxbundle`, items ajoutes seulement) ; les items
   valides de la planche sont REPLIES par defaut (verdict `[VALIDE]` -> section repliee, deplie au clic).
2. Production (verdicts rendus) : R2-1 joueur actif = token `success` en plus de la forme ; R2-4 Dynamo =
   variante 2 ; R2-5 mur = token `warning` ; A4 : la nappe Dynamo persiste >= 2,5 s au sol (mesurer la
   duree actuelle, la porter a 2,5 s, sans changer le son) ; A1 : croix de mort plus petites, plus epaisses,
   toujours rouges (token `danger`/`destructive` existant), persistance plus longue (proposer 2 durees sur la
   planche, livrer la plus courte par defaut) ; cercle d'altitude a la couleur du pion du joueur ; A3 : la
   bascule « effet de mort » est dans le tiroir de reglages du rejeu (verifier, sinon l'y mettre), OFF par
   defaut ; A8/R2-2 : heatmap = plus d'OPACITE (pas de quantile) et rampe bleu -> rouge -> violet aux
   extremes rares (tokens : construire la rampe a partir de tokens semantiques existants, jamais d'hex ; si
   aucun token violet n'existe, le dire et proposer) ; R2-3 : melee fatale = ETOILE (forme dessin anime), pas
   une croix ; C1 : REGRESSION — le fil des morts ne defile plus (tout compacte en fin de match, illisible)
   -> restaurer le defilement/la hauteur (V5 « une ligne » ne doit PAS supprimer le scroll vertical de la
   liste), retirer le glyphe « joueur actif » du fil, colorer en `success` le joueur actif et ses amis dans
   le fil ; P1/P2 : socles = UNE seule version : point « disponible » + icone de l'arme EN DESSOUS, blanche
   remplie avec contour noir (tokens : `foreground`/`background` du theme — pas d'hex ; contour = encre du
   fond), compteur de reapparition AU-DESSUS du point, meme style, SEULEMENT le compteur (aucune mediane
   ni ecart) ; B2/R2-7 : variante COMPACTE de la fiche = pas de perte d'info sauf : munitions de l'arme en
   main seulement, zone active du joueur retiree, armes + grenades + equipements sur UNE ligne — livree
   comme OPTION (bascule tiroir), la validee reste le defaut ; R2-6 ecran de dissimulation : proposition
   seule, semi-transparent ou pions au-dessus (famille absente : rien de branche).
3. Sons D5 : le traqueur de menaces prend le SON du capteur de menaces (`EQUIPMENT_PLACEMENT_SOUND_STEMS`) ;
   translocateur et champ de reparation : CHERCHER les fichiers ailleurs que la bibliotheque livree
   (archive Desktop `Halo Infinite - Sons armes/`, manifeste d'extraction, tags `eqip`/`snd!` du jeu si un
   index existe dans `.ai/V7.5/`) — trouve => encode et branche ; sinon negatif ecrit avec les chemins fouilles.

## Lot R3-V (un agent, worktree `wt/r3-visuels`) — CLOS le 2026-08-18

> ORDRE D'EXECUTION : C1 (R3.5) est passe EN PREMIER, sur consigne du superviseur — c'est une
> REGRESSION du lot R2-V, et une regression se repare avant qu'on empile dessus. Le reste a
> suivi l'ordre du plan.

- [x] R3.1 A1 croix de mort + cercle d'altitude ; A3 bascule tiroir ; R2-1 success ; planche : 2 durees.
      Commit `a7aa0e71c`. Croix : demi-taille 5 -> 3,6 px, trait 1,6 -> 2,6 px, encre = token
      `destructive` (plus la couleur d'equipe), persistance 1,5 -> 2,5 s. Anneau d'etage : encre
      du theme -> couleur du pion. **A3 et R2-1 sont VERIFIES SUR PIECES et deja en place** —
      `SHOW_KILL_FX_DEFAULT = false` avec la bascule dans la section Effets du tiroir (deux tests
      la tenaient deja), et `SELF_TOKEN = 'success'` livre par R2-V : aucune ligne a ecrire.
      L'ajout est paye par une EXTRACTION (`useReplayTiming.ts`) : ReplayCanvas 857 -> 812, et le
      cliquet DESCEND a 812.
- [x] R3.2 A4 Dynamo variante 2 en production + persistance >= 2,5 s (mesure avant/apres).
      Commit `576bdae66`. **La fenetre DECLAREE disait deja 2,5 s ; ce que l'ecran montrait
      durait 2,10 s.** Mesure (fonction de production sur un contexte enregistreur, age par age,
      seuil de lisibilite 0,15) : AVANT fenetre 2,50 s / visible 2,10 s / courbe 0,85 -> 0,20 a
      2,0 s ; APRES fenetre 3,00 s / visible 2,80 s / plateau 0,66 jusqu'a 2,25 s. Deux choses
      changent ensemble : la fenetre monte a 3,0 s ET la courbe devient un plateau suivi d'une
      chute. Graphie = variante 2 (nappe diffuse, 9 arcs qui rebondissent, aucun anneau), teinte
      `--replay-fx-electric` (ecart planche/production corrige). Son NON touche.
      Extraction payante : `grenadeRestLayer.ts` ; replayDraw 518 -> 404 L.
- [x] R3.3 A8/R2-2 heatmap opacite + rampe bleu/rouge/violet en tokens.
      Commit `156337632`. Plafond d'opacite 0,55 -> 0,75, quantiles INCHANGES (le levier que la
      mesure R2-V donnait cinq fois plus efficace). Mesure sur le film temoin `01e1f945`
      (11 760 cellules, 7 698 peintes) : cellules a alpha >= 0,30 **21,4 % -> 29,4 % (+8,0 pt)**,
      plancher 50,7 % et saturation 5,1 % inchanges — la signature exacte du levier choisi.
      Rampe a TROIS points, `heatRamp` prend N arrets. **Aucun token violet n'existait** : le
      token `extreme` est AJOUTE au contrat semantique avec ses quatre palettes, sur le modele
      de `legendary` (default `#C026D3`, okabe-ito et cividis `#CC79A7`, tol-bright `#AA3377`).
      Composition dans la source unique du depot : `heatmapRampTokens('intensity')`.
- [x] R3.4 R2-3 melee = etoile ; R2-5 mur = `warning`.
      Commit `90e2adaf8`. Etoile a huit branches alternees, 400 ms, au LIEU DE LA MORT ; elle
      REMPLACE le rendu generique — le corps a corps est justement le cas ou l'axe n'existe
      presque jamais, et la melee fatale se reduisait a un anneau pointille. Mur : token
      `warning`, encre FIXE qui ne passe plus par `inkOf` (il perd le camp de son poseur, et
      l'utilisateur l'a accepte explicitement).
- [x] R3.5 C1 fil des morts : defilement restaure (regression), glyphe actif retire, `success` actif+amis.
      Commit `fea5f2686`, JOUE EN PREMIER. **Cause trouvee** : le `overflow-hidden` pose par V5
      sur chaque rangee met la taille minimale automatique d'un element flex a ZERO (CSS Flexbox
      4.5) — les rangees se sont ecrasees jusqu'a tenir toutes dans la hauteur, plus rien n'a
      deborde, donc plus rien n'a defile. Correctif : `shrink-0`. **Preuve de layout dans un vrai
      navigateur** (40 rangees, carte de 320 px) : AVANT rangee 6,06 px / texte 16 px / liste
      320/320 px / defile=false / contenu rogne=true ; APRES rangee 20,00 px / liste 878/320 px /
      defile=true / rogne=false. Glyphe « moi » retire du fil (il reste aux fiches et a la carte),
      libelle conserve pour les lecteurs d'ecran ; noms marques au token `success`.
- [x] R3.6 P1/P2 socles : version unique (point + icone dessous blanche/contour + compteur dessus).
      Commit `761a131f2`. L'anneau disparait ; le POINT dit le lieu et l'etat (plein / pointille /
      discret), la VIGNETTE passe dessous remplie `--foreground` et cernee `--background` (huit
      reposes — un canvas ne sait pas cerner une image), le COMPTEUR passe au-dessus, meme encre.
      `padCycleFmt` (mediane + ecarts) est SUPPRIME du contrat et des deux langues : plus aucun
      ecran ne le rend. La TAILLE adaptative est GARDEE (regle ecrite d'avance et testee — le
      verdict portait sur la forme, pas sur la hierarchie).
- [x] R3.7 B2/R2-7 fiche compacte en option (bascule) ; R2-6 proposition seule.
      Commit `6b16b8dac`. Bascule « Fiches compactes », ETEINTE par defaut. Trois changements :
      zone retiree, armes + grenades + equipement sur UNE rangee, munitions de la seule arme en
      main. Deux rangees de moins par joueur. **La bascule vit dans le tiroir et les fiches dans
      une autre colonne** : `replayPreferences` gagne un registre d'ABONNES (l'evenement
      `storage` du navigateur ne se declenche QUE pour les autres onglets), et la colonne lit un
      hook etroit `useReplayCompactCards`. Deux extractions payantes :
      `ReplayHeatmapSection.tsx` (tiroir 465 -> 431) et `ReplayVitality.tsx` (fiches 490 -> 420).
      R2-6 : proposition SEULE, reprise et elargie a l'item R3-2 de la planche (cf. R3.9).
- [x] R3.8 D5 sons : traqueur = capteur ; translocateur + champ : recherche, encodage si trouve, sinon negatif.
      Commit `fecbe67e1`. Le traqueur prend `sensor_activate` — emprunt adosse a une PARENTE
      (meme appareil a un mode pres, meme geste de pose), pas a la disponibilite d'un fichier.
      **NEGATIF ECRIT pour la balise du translocateur et le champ de reparation**, avec les
      chemins fouilles : (1) bibliotheque livree, sept dossiers d'equipement, aucun des deux ;
      (2) `Desktop/Halo Infinite - Sons armes/`, 60 categories / 4 651 .wav — archive d'ARMES
      declaree telle par son LISEZ-MOI, recherche recursive sur neuf motifs (`*seeker*`,
      `*transloc*`, `*repair*`, `*quantum*`, `*beacon*`, `*shroud*`, `*equip*`, `*heal*`,
      `*medic*`) = ZERO fichier, manifeste indexe par arme (58 entrees, 0 equipement) ;
      (3) la chaine d'extraction part du tag `weap` et ne connait PAS le groupe `eqip` —
      0 occurrence dans les trois documents de la recette, aucun mode de l'outil Go ;
      (4) `static/sounds/halo_infinite/` : 41 fichiers, aucun. Les sources brutes existent
      (.pck du jeu, 90 170 .wem SANS NOMS, installation Steam presente) mais les nommer demande
      une passe de banks sur le module de 7,24 Go — un chantier, pas une ligne, et hors du
      perimetre d'un lot web.
- [x] R3.9 Planche : items valides replies, propositions ajoutees, fumee 0 erreur ; plan statue.
      Bundle rebati depuis `LevelUp-wt-r3-visuels` (`replayfx.r3.iife.js`, 233,2 ko,
      `rolldown.r3.config.mjs`), page assemblee par `assemble_r3.cjs`. **38 items** (36 + 2),
      60 canvas, fumee `smoke.cjs` **0 erreur en clair ET en sombre**. Les **17 items VALIDES du
      bilan du matin sont REPLIES** (A2 A5 A6 W2 W3 B1 B3 B4 B5 B7 D0 D1 D2 D3 D4 E1 F1) :
      85 px replie contre 420 px deplie, un clic sur le titre deplie (verifie au navigateur,
      466 px, 0 erreur). Deux propositions AJOUTEES, aucun item existant touche :
      **R3-1** duree de la croix de mort (2,5 s livre / 4 s propose, code de PRODUCTION), et
      **R3-2** ecran de dissimulation (bulle semi-transparente / bulle opaque avec les pions
      AU-DESSUS) — la famille reste absente du manifeste, rien n'est branche.

### Gate du lot — TENU, sans reserve
`npm run typecheck` EXIT=0 · `npm run lint` EXIT=0 (0 erreur, 20 avertissements pre-existants
`react-hooks/incompatible-library`, les memes qu'aux gates P et G) ·
`npx vitest run src/features/match-replay src/lib` EXIT=0 (**143 fichiers, 1 756 tests**) ·
`npx vitest run src/components` EXIT=0. Codes de retour : `scratchpad/r3_visuels_gates.log`.

COULEURS : les 25 occurrences hexadecimales sous `features/` + `components/` sont TOUTES
pre-existantes (commentaires, ou `color-allow` justifie) ; aucun fichier touche par ce lot n'en
introduit. Les hex du lot vivent dans `lib/accessibility/palettes/*`, la seule exception
documentee du depot. Aucune classe Tailwind de couleur sous `match-replay/`.

i18n : FR + EN pour toutes les chaines neuves (`cards`, `cardsCompact`, `cardsCompactHint`),
parite tenue par le typage `Record<ReplayLocale, ReplayText>`.

SEUIL DE 500 LIGNES : tenu partout. ReplayCanvas 812 (cliquet descendu de 858 a 812),
ReplayKillFeed 498, ReplayTeams 420, tiroir 431, replayDraw 404, heatmapLayer 474,
replaySound 457, weaponPadsLayer 378. Six extractions au total :
`ReplayFeedName`, `useReplayTiming`, `grenadeRestLayer`, `meleeStar`, `ReplayHeatmapSection`,
`ReplayVitality`.

## Annexe — bilan utilisateur verbatim (2026-08-18, apres-midi)

(colle par le superviseur : voir message utilisateur ; points cles ci-dessus reprennent chaque verdict)

## Journal du plan
- 2026-08-18 — plan ecrit, agent lance (`wt/r3-visuels`).
- 2026-08-18 — **lot R3-V CLOS** (branche `wt/r3-visuels`, base `85ab55648`). Neuf items
  statues `[x]`, huit commits de production, un par item livre : C1 `fea5f2686` (joue EN
  PREMIER — regression), R3.1 `a7aa0e71c`, R3.2 `576bdae66`, R3.3 `156337632`, R3.4
  `90e2adaf8`, R3.6 `761a131f2`, R3.7 `6b16b8dac`, R3.8 `fecbe67e1`, plus le `docs` de
  cloture. Gate tenu sans reserve (detail ci-dessus).

  TROIS DECISIONS PRISES EN COURS D'EXECUTION, toutes ecrites dans le code :
  1. **La nappe Dynamo demandait DEUX changements, pas un.** La consigne disait « mesurer la
     duree actuelle, la porter a 2,5 s » ; la fenetre DECLAREE valait deja 2,5 s. Ce qui
     manquait etait la COURBE : une droite depuis l'instant zero passait sous le seuil de
     lisibilite a 2,10 s, avant sa propre fenetre. Porter la seule fenetre a 2,5 s n'aurait
     rien change a l'ecran. D'ou plateau + chute ET fenetre a 3,0 s.
  2. **La rampe de chaleur emprunte des tokens de STATUT (`info`, `destructive`) plutot que
     trois tokens neufs.** Chaque palette d'accessibilite les remappe deja pour qu'ils restent
     distinguables entre eux : sous Okabe-Ito la rampe devient Sky Blue -> Vermillion ->
     Reddish Purple, trois couleurs de la meme reference CUD. Inventer trois tokens aurait
     duplique ce travail. Seul le VIOLET manquait : `extreme` est le seul ajout au contrat.
  3. **La taille adaptative des socles est GARDEE alors que le verdict dit « une seule
     version ».** « Une seule version » repond aux variantes de FORME que la planche montrait ;
     la hierarchie de taille (arme de puissance contre arme classique) est une regle distincte,
     ecrite d'avance a la demande explicite de l'utilisateur (W4) et tenue par un garde-rail
     sur le registre du titre. La defaire aurait annule un verdict sans en avoir recu un autre.

  DETTE TRAITEE EN PASSANT (condition des cliquets et du seuil, pas un fix opportuniste) :
  six extractions — `ReplayFeedName.tsx`, `useReplayTiming.ts`, `grenadeRestLayer.ts`,
  `meleeStar.ts`, `ReplayHeatmapSection.tsx`, `ReplayVitality.tsx` — et le cliquet du canvas
  qui DESCEND de 858 a 812.

  CODE MORT SUPPRIME : `padCycleFmt` (contrat + FR + EN + sa ligne d'infobulle), plus aucun
  ecran ne le rendait apres le verdict « seulement le compteur ».

## Decouvertes (lot R3-V) — notees, NON traitees (regle 7)

1. **`drawRestHalo` et la nappe Dynamo ignorent la densite d'ecran (`style.k`)**, la ou tous
   les autres calques la respectent : leur rayon est ecrit en pixels de CANEVAS, donc deux fois
   plus petit a l'oeil sur un ecran a forte densite. C'est anterieur a ce lot (le champ `k`
   existe dans `GrenadeRestStyle` et n'est lu que par l'explosion), et le corriger changerait
   la TAILLE apparente des deux effets — une valeur d'ecran, donc un verdict, pas un correctif.
2. **L'infobulle du reglage « Sons » est toujours fausse** (decouverte du lot R2-G, non
   traitee depuis) : elle annonce des sons « coupes a la seconde » et ne mentionne pas les
   explosions. Une phrase a reecrire, hors perimetre de ce lot-ci egalement.
3. **Le glyphe « moi » disparait du fil mais reste sur la carte et les fiches.** C'est ce que
   le verdict demande, et la grammaire D5 (« une seule grammaire sur les trois panneaux ») en
   sort donc ebrechee : deux panneaux sur trois portent la marque. A confirmer d'un mot au
   prochain gate visuel — si la gene revient sur les fiches, la meme regle s'y applique.
