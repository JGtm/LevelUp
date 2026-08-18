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

### R2-V — visuels canvas (worktree `wt/r2-visuels`)
- [ ] V1 (A1) traînee en OPTION (bascule tiroir, persistee) ; marqueur du joueur ACTIF distinct
      (contour double + halo, token) — proposition planche puis production.
- [ ] V2 (A8) heatmap : accentuer (echelle p50->p95 -> p40->p95 ou opacite max 0,55 -> 0,75, MESURER la
      lisibilite sur `000d5950`) ; bascule « toute la partie » (heatmap statique de tout le match, un
      bouton) en plus du mode progressif.
- [ ] V3 (W1) mur sans cap : cap = direction de la TRAJECTOIRE du poseur (derniere direction de
      deplacement > 0,5 m avant la pose) ; sinon direction de visee `h` de la derniere image ; JAMAIS
      rond ; couleur : proposition `legendary` vs `warning` sur la planche.
- [ ] V4 (D3) melee fatale : effet visuel special court (a proposer sur la planche : eclat en croix au
      point d'impact, 400 ms) — le son est court et discret.
- [ ] V5 (C1) fil des morts : tout sur UNE ligne (assistance comprise, aucun retour a la ligne) ;
      INVESTIGUER le symbole rond dans un cercle a 1:06 sur `000d5950` pour JGtm (medaille ? sprite
      killfeed non nomme ?) — lire l'artefact et le sprite, repondre avec la piece.
- [ ] V6 (B2) reapparition : proposer sur la planche une variante COMPACTE (sans supprimer la validee).
- [ ] V7 (A4) Dynamo : proposer sur la planche 2 variantes inspirees de la reference utilisateur
      (champ electrique : anneau + arcs brises pulsants) — la reference est un GIF externe, non
      embarque ; l'utilisateur tranche.
- [ ] V8 (W3) `repair_field` = croix de pharmacie qui pulse dans son cercle ; `translocator_beacon` =
      la BALISE du translocateur quantique (pas le ping) — le dire dans le tiroir/infobulle ; « ecran de
      dissimulation » : la famille N'EXISTE PAS dans le manifeste (`replay_labels.toml`) — mesurer si un
      identifiant `other` du corpus lui correspond (nommage `sofd -> sofa`), sinon `[!]` corpus ; visuel
      propose sur la planche si la famille apparait : bulle opaque au token `muted` avec bord flou.
Gate V : typecheck/lint/vitest verts ; planche mise a jour avec les propositions ; aucune couleur en dur.

### R2-S — sons (worktree `wt/r2-sons`) — LOT CLOS le 2026-08-18
- [x] S1 (D4/D5) egalisation LUFS de TOUS les sons (decision 3), avant/apres publie. **41 fichiers
      renormalises a -16 LUFS / plafond -1 dBTP par gain LINEAIRE strict** (equivalence avec
      `loudnorm linear=true` verifiee sur piece). AVANT : etendue 31,58 LU (-9,20 rayon de Sentinelle
      a -40,78 melee), 3 fichiers au-dessus de 0 dBTP. APRES : etendue 7,18 LU, 0 fichier au-dessus
      de -1 dBTP, maximum momentane resserre de 30,10 a 11,60 LU. Durees, frequences, canaux et
      `duration_ts` INCHANGES sur les 41. Le grief mesure (surbouclier a peine audible) est corrige :
      -29,53 -> -16,01 LUFS. Commit `83829c7b7`.
- [x] S2 (D2) recouvrement lancers/explosions mesure (decision 4). **La condition de la decision 4
      n'est PAS remplie : les deux restent.** Piste reelle des 3 temoins : 343 lancers, 16 explosions,
      5 chevauchements a moins de 0,3 s = 31,3 %. Mesure de controle sans kill (fin de vol du
      projectile lie, 296 lancers apparies) : 48,6 %, sous le seuil egalement. Regle epinglee par un
      test unitaire. Commit `d641ca70d`.
- [x] S3 (D1) inventaire : armes a TIR CHARGE et a TIR CONTINU (declares dans le tag `weap`, acquis
      registre) presentes dans le corpus vs sons disponibles (charge : Ravager seul ; continu : rayon
      de Sentinelle) — liste des MANQUANTS avec le nom d'arme, sans inventer de son. **Deux armes du
      registre declarent un second mode de tir : Pistolet a plasma (`hinf_plasma_pistol`, mode 3
      charge, JAMAIS livre) et Ravageur (`hinf_ravager`, mode 2 charge, le .wav livre est le mode 1).
      Le tir continu n'est pas un mode de tag : c'est la nature du mode unique du Rayon de Sentinelle,
      dont le .wav livre est le tir COURT. Aucun fichier cree.** Commit `b87bdd1e2`.
Gate S : durees inchangees, LUFS dans ±1 de la cible, tests `replaySound` verts. **PASSE avec une
reserve ECRITE sur la clause LUFS** : 26 fichiers sur 41 sont dans +/-1 LU ; les 15 autres plafonnent
entre -17,2 et -23,1 LUFS parce que leur facteur de crete (15 a 24 dB) interdit la cible sous -1 dBTP.
C'est l'echappatoire prevue par la decision 3 (« si un fichier ne peut pas atteindre la cible sans
ecretage, le laisser au plus pres ») : les y forcer demanderait un limiteur, donc une retouche du
timbre extrait du jeu — decision d'oreille, non prise ici. Durees : 41/41 identiques a l'echantillon.
Tests : `replaySound.test.ts`, `replaySoundAssets.guard.test.ts`, `useReplaySound.test.tsx` verts.

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
- 2026-08-18 — **lot R2-S CLOS** (worktree frere `wt/r2-sons`, base `3907eb505`) : S1 `83829c7b7`,
  S2 `d641ca70d`, S3 `b87bdd1e2`, plan `docs`. Gates rejoues sur l'arbre du frere apres `npm ci`.
  Les trois items sont statues `[x]`, avec une reserve ecrite au gate S sur la clause « LUFS dans
  +/-1 » (15 fichiers plafonnes par leur facteur de crete — echappatoire de la decision 3).

## Decouvertes (lot R2-S) — notees, NON traitees (regle 7)

1. **L'explosion de grenade ne sonne QUE sur un kill, et c'est la vraie cause du grief D2.** Sur les
   trois temoins : 343 lancers, 16 explosions (4,7 %), et ZERO sur `00162144` qui porte pourtant
   143 lancers. Le film ne publiant aucun evenement de detonation, la seule source d'explosion est
   la vignette d'un kill a la grenade (`KILL_SPRITE_SOUND_STEMS`). Sonner la FIN DE VOL du projectile
   (`buildGrenadeRestFx`, deja calculee pour l'effet visuel) donnerait une explosion a chaque grenade
   — mais ce n'est plus « mesurer le recouvrement », c'est une feature : elle appartient a l'utilisateur.
   Ordre de grandeur du changement : +296 sons sur 3 temoins, dont 48,6 % a moins de 0,3 s de leur
   lancer. Le point est le meme pour l'ecran (A4 « fin de vol des grenades »).
2. **Le tir charge du Ravageur et celui du pistolet a plasma sont RENDUS et archives, mais pas livres**
   (`Desktop/Halo Infinite - Sons armes/`). Les brancher demande d'abord une source qui qualifie le
   MODE d'un tir : les deux candidats (jauge de charge, cadence) sont des NO-GO mesures au registre.
3. **`static/sounds/halo_infinite/hinf_gravity_hammer.wav` est le seul fichier MONO** du dossier
   (les 40 autres sont stereo). Sans effet mesure ici — la normalisation l'a preserve tel quel.
