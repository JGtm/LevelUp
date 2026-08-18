# Plan — Retours de la planche R2 (bilan utilisateur du 2026-08-18) -> lots

> Ecrit le 2026-08-18 par la session de pilotage. Le bilan verbatim est en annexe. Contrat
> `plan-execution`. Chaque lot = un worktree frere web `../LevelUp-wt-<nom>` (branche `wt/<nom>`),
> `npm ci` dans le frere, fusion `--no-ff` par le superviseur, gates rejoues sur l'arbre fusionne.
> **Prealable de TOUS les lots web : la fusion `wt/registre-film` de l'autre session (schemas 12/13,
> `objectives.go`, `objectivesLayer.ts`) est POSEE sur `feat/v75`** — les lots partent de ce HEAD.

## Verdicts VALIDES (rien a faire, consignes) : A2, A3, A5, A6, B1, B3, B4, B5, B7, D0, D5 (son),
E1, F1. Questions ouvertes : **frag = 3,335 s GARDE** (D3 « Parfait » avec la re-coupe ; l'utilisateur
n'a pas repondu a la question isolee — a confirmer d'un mot) ; **mur = arc concave GARDE**, sans repli
rond (W1).

## Decisions tranchees (superviseur, a partir du bilan)

1. Toute couleur = token semantique (`color-tokens`), jamais un hex ; « orange dore » du mur = le token
   le plus proche existant (`legendary` est or ; `warning` est orange — l'agent PROPOSE les deux sur la
   planche, l'utilisateur tranche) ; joueur actif : NE PAS reposer sur la couleur seule (a11y) —
   contour double + halo, couleur au token `accent`/`success` en PLUS, proposition planche.
2. Aucune valeur d'ecran inventee : chaque proposition visuelle va sur la planche (recette
   `assemble.cjs`) avant d'entrer en production ; les items VALIDES ne bougent pas.
3. Sons : egalisation = mesure LUFS integree par fichier (`ffmpeg -af loudnorm` en analyse) puis
   normalisation a une cible unique (-16 LUFS, plafond -1 dBTP) pour TOUS les sons (armes, grenades,
   melee, equipements, poses), fichiers regeneres depuis les sources livrees (branche utilisateur
   fusionnee), durees inchangees ; publier avant/apres par fichier.
4. Lancers vs explosions : mesurer le recouvrement reel (timeline) ; si un lancer et son explosion se
   chevauchent a < 0,3 s : garder l'explosion seule ; sinon les deux.
5. Socles (W4) : le calque `weaponPads` (schema 11, note UI du plan item 6) — icone de l'arme sur le
   socle, taille adaptative : armes de puissance (sniper, epee, marteau, roquettes, power-ups) grande,
   armes classiques petite ; etat plein / vide / incertain ; compte a rebours si cycle etabli ; objets
   laches JAMAIS dessines ; ratelier vs socle au sol : NON distinguable par la donnee (position seule)
   — ecrit, pas devine.

## Lots

### R2-V — visuels canvas (worktree `wt/r2-visuels`) — CLOS le 2026-08-18
- [x] V1 (A1) trainee en OPTION (bascule tiroir « Trainee », cle `replay-show-trail`, defaut ALLUME) ;
      joueur de la page distinct par la FORME d'abord — DOUBLE contour + halo, encre au token
      `success` (le vert demande) qui vient EN PLUS, le noyau gardant la couleur d'equipe.
      Commit `4a98073a2`. Proposition planche `R2-1` : forme seule / `success` / `info` / avant.
      Extractions rendues necessaires par les deux cliquets de taille : `useReplayInks.ts`,
      `replayAimCone.ts`, `replayProjectiles.ts`, `i18nContract.ts` (ReplayCanvas 861 -> 824,
      replayMarkers 589 -> 470, i18n 505 -> 317).
- [x] V2 (A8) MESURE PUBLIEE + bascule de portee. Commit `d8afc10ed`.
      **La premisse etait fausse et la mesure le dit** : la carte du 16/08 etait DEJA celle de tout
      le match (`accumulatePresence` sans borne, calque cuit une fois). Il n'existait aucun mode
      « au fur et a mesure ». Ce lot AJOUTE la portee `live` (bornee a l'image courante, recuisson
      toutes les 2 s de match, 6-20 ms mesurees) et la bascule a deux valeurs ; `match` reste le
      defaut. Accentuation = proposition planche `R2-2`, chiffree sur `000d5950` (11 016 cellules
      peintes sur 15 232) : aujourd'hui 50,0 % au plancher et 14,6 % a alpha >= 0,30 ; p40->p95
      +1,4 pt ; alpha max 0,75 +7,1 pt ; les deux +11,0 pt. **Le plafond d'opacite fait cinq fois
      plus que le quantile** — l'utilisateur tranche.
- [x] V3 (W1) chaine de cap a trois sources. Commit `65b020804`. Mesure corpus (32 films, 62
      panneaux) : cap de la pose 54/62 (87,1 %), trajectoire (dernier deplacement >= 0,5 m) 8/62
      (12,9 %, TOUS resolus, deplacement max 0,50-0,74 m), visee de la derniere image 0/62 mais
      disponible 8/8. Un cap DEDUIT se trace en pointille. Proposition planche `R2-5` :
      `legendary` vs `warning` vs couleur d'equipe.
- [~] V4 (D3) melee fatale : PROPOSITION SEULE sur la planche (`R2-3`, eclat en croix 400 ms, deux
      tailles 14 px / 20 px). Aucune production : le plan la classait « a proposer », et une valeur
      d'ecran n'entre pas en production sans verdict (decision 2).
- [x] V5 (C1) fil des morts sur UNE ligne. Commit `7a48bba78` : les trois formes de ligne (kill,
      mort neutre, medaille seule) partagent `FEED_ROW` (`flex-nowrap` + `overflow-hidden`) ;
      medailles et assistance sont rentrees dans la rangee, plus aucun bloc `pl-7`.
      **Enquete 1:06 RESOLUE, avec la piece** : c'est une ligne de MEDAILLE SEULE. `highlight_events`
      du match `000d5950-83d9-423f-ab55-d068a7237b9f` porte `medal` a `time_ms = 70542` pour
      `xuid 2533274823110022` (JGtm), `medal_name = "Odin's Raven"` ; aucun kill de JGtm a moins de
      500 ms (`MEDAL_ATTACH_MS`), d'ou sa propre ligne. Horloge : `replayMs = 70542 + t0Ms - originMs`
      = 70542 + 0 - 3604 = 66 938 ms -> `formatClock` 1:06. Calibration verifiee : la mort de JGtm
      a `time_ms = 62936` tombe sur la fin de sa vie slot 522 (frame 593 = 59 300 ms), residu 32 ms.
      Le « symbole rond dans un cercle » : le BADGE de la medaille,
      `static/medals/halo_infinite/87172902.png` (disque vert sombre, corbeau blanc, couronne
      bronze) rendu a 15 px (`MEDAL_PX`) — a cette taille il ne reste que le disque dans son anneau.
      La ligne porte AUSSI le glyphe « moi » (`PlayerMark.tsx`, `<circle r=4.2 fill=none stroke>` +
      `<circle r=2>`), lui aussi un rond dans un cercle. **Ce n'est PAS un sprite killfeed** : le fil
      n'en emploie que deux (`killfeed-62` assistance, pictogrammes de type de mort), et aucun ne
      figure sur une ligne de medaille seule.
- [~] V6 (B2) reapparition compacte : PROPOSITION SEULE sur la planche (`R2-7`) — la validee reste
      au-dessus, la compacte garde le compte a rebours, remplace les deux jauges par une barre qui
      se vide et fait respirer un lisere gauche. Sans production, conformement au plan.
- [~] V7 (A4) Dynamo : DEUX propositions sur la planche (`R2-4`) — variante 1 anneau net qui pulse
      + arcs brises repartant du centre ; variante 2 nappe diffuse sans anneau, arcs qui rebondissent
      sur une bordure invisible. Teinte `electric` inchangee, duree 2,5 s inchangee. La reference de
      l'utilisateur est un GIF externe, non embarquable — l'utilisateur tranche.
- [x] V8 (W3) `repair_field` porte une CROIX DE PHARMACIE qui respire dans son cercle
      (`repairCrossAlpha`, periode 1,8 s, alpha 0,55-0,95, immobile sous mouvement reduit) ; la zone
      et son anneau pointille ne bougent PAS — la respiration ne chiffre aucune cadence. Libelle
      `translocator_beacon` explicite : « Balise du translocateur quantique » / « Quantum translocator
      beacon », et le tiroir dit que c'est le POINT DE RETOUR, pas le « ping ». Commit `527495a32`.
- [!] V8 (suite) « ecran de dissimulation » : **famille ABSENTE du manifeste**, a mesurer cote
      donnees. `config/titles/halo_infinite/mappings/replay_labels.toml` ne nomme que 15 familles et
      aucune ne correspond (ni « ecran », ni « dissimulation », ni « camo screen »). Le seul
      identifiant que le corpus laisse sans nom est `0x4396db42` : 94 poses, 7 films, 24 deployees —
      rien ne l'y rattache aujourd'hui. Visuel PROPOSE sur la planche (`R2-6`, bulle opaque a l'encre
      `muted`, bord flou sur 22 % du rayon), non branche. **Reprise : nommer `0x4396db42` par la
      chaine `sofd -> sofa` AVANT de brancher quoi que ce soit.**
Gate V : typecheck OK, lint OK, vitest OK (voir journal) ; planche re-bundlee depuis
`LevelUp-wt-r2-visuels` et augmentee de 7 items `proposition R2` (34 items, fumee 0 erreur en clair
comme en sombre) ; aucune couleur en dur cote `features/` (les hex de la planche sont des
transcriptions de `globals.css`, hors depot).

### R2-S — sons (worktree `wt/r2-sons`)
- [ ] S1 (D4/D5) egalisation LUFS de TOUS les sons (decision 3), avant/apres publie.
- [ ] S2 (D2) recouvrement lancers/explosions mesure (decision 4).
- [ ] S3 (D1) inventaire : armes a TIR CHARGE et a TIR CONTINU (declares dans le tag `weap`, acquis
      registre) presentes dans le corpus vs sons disponibles (charge : Ravager seul ; continu : rayon
      de Sentinelle) — liste des MANQUANTS avec le nom d'arme, sans inventer de son.
Gate S : durees inchangees, LUFS dans ±1 de la cible, tests `replaySound` verts.

### R2-P — calque des socles (worktree `wt/r2-socles`) — decision 5, note UI item 6
- [ ] P1 calque `weaponPads` (icone, taille adaptative, plein/vide/incertain, compte a rebours si cycle),
      bascule tiroir, i18n FR+EN, tokens, tests ; temoins `01e1f945`, `00162144`, `bcb6d393`.
Gate P : gates web verts ; planche : items P sur `01e1f945`.

## Annexe — bilan utilisateur verbatim (2026-08-18)

    BILAN — effets du rejeu 2D — 2026-08-18
    ## Sur la carte
    [VALIDÉ] A1 Marqueur de vie, traînée, cône de visée, nom (à rejuger) — Parfait. petits bonus : avoir la trainée en option et avoir l'icone du joueur actif qui se démarque de tous les autres aussi (j'aurais bien aimé du vert mais pour l'accessibilité je sais pas si ça peut le faire)
    [VALIDÉ] A2 Éclair de bouche — Parfait
    [VALIDÉ] A3 Effet de mort orienté tueur → victime — Optionnel, désactivé par défaut Parfait sinon
    [À REVOIR] A4 Fin de vol des grenades : explosions et nappe Dynamo (à rejuger) — Pour la dynamo je préfère un truc dans ce genre https://i.pinimg.com/originals/34/9f/78/349f7848966d4e5a1d47535d1d8f00f2.gif
    [VALIDÉ] A5 Ligne du grappin — Parfait
    [VALIDÉ] A6 Zones de callout (à rejuger) — Parfait
    [sans avis] A7 Objectifs (placement) et fond de carte
    [À REVOIR] A8 Carte de chaleur (présence / éliminations) (nouveau) — C'est pas mal mais à accentuer un peu. De plus la heatmap au fur et à mesure est une bonne idée mais j'aimerais qu'en un bouton on voit la heatmap de toute la partie, tu vois ce que je veux dire ? En analyse post match ça permet de voir les zones chaudes directs
    ## Les objets posés sur le terrain
    [À REVOIR] W1 Mur de protection — un ARC concave vers le poseur (nouveau) — Sans cap je préferais qu'on tente de correler la visee ou la trajectoir du joueur, un mur portatif rond serait trop troublant. De plus je préfererais un orange doré pour sa couleur
    [VALIDÉ] W2 Capteur de menaces — la zone, le ping, la marque « révélé » (nouveau) — Parfait
    [À REVOIR] W3 Balise, traqueur, champ de réparation (nouveau) — Une balise je vois pas trop ce que c'est, c'est ce qu'on appelle le "ping" en jeu ? Pour le traqueur ok. Champ de réparation comme c'est un truc pour la santé tu peux mettre une croix comme une croix d epharmacie qui pulse ? avec le cercle autour évidemment. Il y a aussi l'écran de dissimulation, une bulle opaque, dedans on voit pas dehors et dehors on voit pas dedans, tu peux me proposer un visuel ?
    [À REVOIR] W4 Ce qui n'est PAS dessiné : le filtre « déployé » (nouveau) — Pour le moment on affichera pas les objets lâchés. Pour ceux qui sont sur socle il faut trouver un moyen, des icones trop petites seraient inutiles mais des trop grosses risquent de polluer mais les infos sont intéressantes à avoir. Que ce soit socle au sol ou les rateliers aux murs. (en principe les rateliers sont plus frequents et des armes "classiques" pas game changer, sur les socles au sol on a des power up et des armes plus puissantes qui ont un interet très strategique
    ## Les fiches joueur
    [VALIDÉ] B1 Éclat de mort sur la fiche — Parfait
    [VALIDÉ] B2 Éclat de réapparition et compteur (à rejuger) — Parfait, tu pourrais me proposer une version plus compacte ? Sans supprimer celle ci qui est validée, je veux tenter autre chose visuellemnt
    [VALIDÉ] B3 Vitalité : bouclier et vie — Parfait
    [VALIDÉ] B4 Armes portées, arme en main, permutation — Parfait
    [VALIDÉ] B5 Grenades EN IMAGES, capacité, munitions (à rejuger) — Parfait
    [VALIDÉ] B7 Équipement actif — VERRE et ENCADRÉ DORÉ (gate en app) — Parfait
    ## Le fil des morts
    [VALIDÉ] C1 Ligne du fil des morts et MARQUE D'ASSISTANCE (à rejuger) — Parfait mais veiller à bien tout avoir sur une même ligne. Et à noter que Sur le match témoin de Cliffhanger, j'ai un élement dans le killfeed à 1:06 qui concerne JGtm maiss je ne sais pas ce que c'est, peut être une médaille seulement ? Il y a un symbole rond dans un cercle d'affiché, je sais pas ce que c'est.
    ## Les sons
    [VALIDÉ] D0 Règles de la piste sonore (à rejuger)
    [À REVOIR] D1 Tirs par arme (et kills à l'arme) — PLUS AUCUNE MUETTE (à rejuger) — Les sons extrait et packagés du jeu ont été récupérés et merge dans la branche donc attention. Pour les tirs chargés je crois n'avoir que le ravager actuellement, pour le tir continu j'ai le rayon de sentinelle. à toi de me dire s'il en manque
    [À REVOIR] D2 Lancers de grenade — Le problème c'est que les lancers sont joués mais j'ai pass les explosions, à voir si ça ne se marche pas dessus, au pire ne agrder que les explosions
    [À REVOIR] D3 Explosions et coup de mêlée fatal — TOUTES RE-COUPÉES (à rejuger) — Parfait, pour les mélées vu que le son est court et discret on peut afficher un effet visuel spécial ?
    [À REVOIR] D4 Équipements actifs : activation et désactivation (à rejuger) — surbouclier activation à peine audible, y a peut être une egalisation à faire pour touuus les sons non ?
    [VALIDÉ] D5 Sons de POSE d'équipement (nouveau) — Même remarque sur l'égalisation, je pense qu'il faut considérer tous les sons dans l'égalisations, armes, effets, etc
    ## Réglages
    [VALIDÉ] E1 Tiroir de réglages — désormais en OVERLAY (gate en app) — Parfait
    ## Ce que le rejeu ne montre pas (encore), et pourquoi
    [À REVOIR] F1 Refus mesurés — rien n'est affiché sans donnée (à rejuger) — Ok
    ## Les deux questions ouvertes
    - Explosion de frag : garder 3,335 s, ou revenir aux 1,2 s validés le 16/08 ? ->
    - Mur de protection : l'arc concave vers le poseur est-il le bon parti ? ->

## Journal du plan

- 2026-08-18 — plan ecrit ; fusion tierce `wt/registre-film` POSEE (`104f468c6`, schema 13, contrat 34) ;
  lots R2-V, R2-S, R2-P LANCES (worktrees freres `wt/r2-visuels`, `wt/r2-sons`, `wt/r2-socles`, base `104f468c6`).
- 2026-08-18 — **lot R2-V CLOS** (branche `wt/r2-visuels`, base `3907eb505`). Cinq commits de
  production, un par item livre : V1 `4a98073a2`, V2 `d8afc10ed`, V3 `65b020804`, V5 `7a48bba78`,
  V8 `527495a32`. V4, V6, V7 restent des PROPOSITIONS (decision 2 : rien n'entre en production
  sans verdict). Planche re-bundlee depuis le worktree frere et augmentee de sept items
  `proposition R2` (`R2-1` joueur actif, `R2-2` accentuation de la chaleur, `R2-3` melee fatale,
  `R2-4` Dynamo, `R2-5` couleur du mur, `R2-6` ecran de dissimulation, `R2-7` reapparition
  compacte) : 34 items, fumee `smoke.cjs` 0 erreur en clair comme en sombre.
  DECOUVERTES : (a) la carte de chaleur etait DEJA celle de tout le match — il n'y avait pas de
  mode progressif a completer, il fallait le creer ; (b) le mur sans cap est resolu 8/8 par la
  trajectoire, le cercle pointille tombe donc a 0/62 a l'ecran ; (c) le symbole de 1:06 est le
  badge de la medaille « Odin's Raven » rendu a 15 px, pas un sprite killfeed.
  DETTE TRAITEE EN PASSANT (condition des cliquets, pas un fix opportuniste) : quatre extractions
  — `useReplayInks.ts`, `replayAimCone.ts`, `replayProjectiles.ts`, `i18nContract.ts`.
  RESTE : `0x4396db42` (94 poses, 7 films) a nommer cote donnees avant tout branchement d'un
  « ecran de dissimulation ».
