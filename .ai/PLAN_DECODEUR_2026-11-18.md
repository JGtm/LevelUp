# PLAN — Décodeur de film : justesse et canaux, version v1.5.0 pour la 8.3.0 (2026-11-18)

Statut : **écrit le 2026-11-18, relu contre la grille `plan-review`. Demandé par le user le 2026-11-18 (« on
vise cette version mais ce qu'on peut faire c'est un plan à part oui, tu le traiteras avec plus de rigueur et
de prudence. du coup tu piloteras le plan backlog et le plan décodeur »). Décisions DD-1 à DD-5 tranchées
par le user le 2026-11-18 (« ok avec tes recos, go pour les deux plans »), avec deux compléments : la part
des tirs lus lui paraît faible, et il veut aussi l'audio des répliques. GO D'EXÉCUTION DONNÉ le 2026-11-18.**
Contrat d'exécution : skill `plan-execution`. Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]`
non traité (motif écrit). Aucune case vide à la clôture.
Plan frère : `.ai/PLAN_BACKLOG_2026-11-18.md` (même pilote). Plan maître de la version :
`.ai/PLAN_MAITRE_8_3_0_2026-11-17.md`.

Abréviations : `DEC` = dépôt `JGtm/halo-infinite-film-decoder` (PUBLIC), lu pour l'instruction à sa version
v1.4.1 dans le cache des modules Go ; `LU` = ce dépôt ; « témoin » = libfilmshell (décodeur de Den, MIT,
copie locale `C:\Users\Guillaume\Downloads\Scripts\libfilmshell\`), lu pour LOCALISER, jamais traduit.

## 0. Objectif et critère de succès

Objectif : traiter les entrées du backlog qui demandent une version du décodeur, dans UNE version (v1.5.0),
UNE montée dans LevelUp et UNE régénération du parc, sans dégrader ce qui est juste aujourd'hui.

Critère de succès : chaque entrée de la section 1 a un sort écrit ; chaque correction est prouvée par une
mesure avant et après sur le parc, contre une vérité indépendante ; aucun repli neuf ; références
d'équivalence refigées avec chaque écart attribué à un item ; gate du corpus sans perte ; v1.5.0 publiée avec
ses changements de sortie écrits ; LevelUp monté, CI verte ; parc régénéré une fois.

| Piste | Ce que le user verra |
|---|---|
| A | Le coup d'envoi du rejeu est celui de l'horloge du jeu, sur tous les rejeux (8 n'en avaient pas, 22 sont à contrôler) |
| B | Le détail daté des points n'est plus dit « complet » quand il ne l'est pas (Grande bataille ancienne, Assaut) |
| C | Fiche du joueur : plus de compteur d'éliminations à la grenade, au corps à corps ou à la tête supérieur aux éliminations ; séries aberrantes retirées |
| D | Rien sur le parc actuel (arme d'une touche) ; « données insuffisantes » des tirs selon DD-2 |
| E | Grande bataille : un faux porteur du drapeau de 94 secondes disparaît ; porteurs au-delà de 8 joueurs si la recherche aboutit |
| F | Rien à l'écran sauf 7 murs déployés (selon DD-3) ; instant réel d'un départ si le film le porte |
| G | Bases : vraie lettre des zones si la recherche aboutit |
| H | Fiche du joueur, « En passant » : rechargements, sauts lus au bouton, temps avec une commande de déplacement ; répliques selon DD-4 |
| I | Rien (comptage des temps forts de type « mode ») |
| J | Rien (ménage ; côté LevelUp, retrait de l'ancienne précision par film) |

Version du décodeur : v1.5.0 (ajouts de champs : version mineure). Schéma du document : 125 vers 126, UNE
montée pour toutes les pistes. LevelUp : 8.3.0, par exception à « une montée du décodeur par version »,
décidée par le user le 2026-11-18.

## 1. Instruction : ce que les pièces disent (2026-11-18)

Trois instructions en lecture seule par agents Sonnet (score et compteurs ; sièges et objectifs ; canaux et
publication) sur `DEC` v1.4.1, le témoin et les 1 455 rejeux du poste (schéma 125), lus un à un. Deux
mesures du pilote (`.ai/annexes_backlog_1118/mesure_departs.md`, `mesure_coup_d_envoi.md`). Scripts gardés :
`.ai/annexes_decodeur_1118/sondes/`. Les références `fichier:ligne` de `DEC` sont celles de v1.4.1 :
l'exécutant les rouvre sur la tête du dépôt avant de coder (étape 0.5).

Nature : **bornée** = cause localisée, correction écrite ; **recherche** = cause ou donnée non établie, issue
incertaine.

| Entrée | Constat sur pièces | Nature | Sort |
|---|---|---|---|
| Coup d'envoi lu dans le film ; 8 refus | Le détecteur date le premier mouvement des joueurs et refuse quand un seul part (`DEC/replay/t0_film.go:311-333`). L'horloge de manche du jeu est dans tous les rejeux et date les 1 455 ; accord à 1 s près dans 1 422 cas sur 1 447 (médiane +0,36 s) ; 22 rejeux à environ 17 s d'écart, 8 sans date par le mouvement. Lequel des deux se trompe dans les 22 : NON établi (deux lectures opposées, pilote et agent) | bornée après une mesure | Piste A |
| Verdict « lecture complète » du score | Le verdict ne compare rien au tableau (`DEC/replay/document_score_actions.go:306-307`). Parc : 944 rejeux « complets », 44 avec un joueur dont la somme des points diffère du tableau : 12 Grande bataille du build `HI_1_11_0` sur 16 (les 4 du build suivant sont exacts : l'hypothèse « au-delà de 8 joueurs » est réfutée ; c'est le barème des véhicules de ce build), 4 Assaut (points de bombe hors de l'arbre des scripts du Slayer), et 28 autres dont 24 à un joueur avec exactement 100 points manquants | bornée (verdict) ; recherche (barème du build, 100 points) | Piste B |
| Quatre compteurs nommés faux | Table des compteurs fixe, lue par ancrage de bits (`DEC/internal/grammar/signaux/statborg.go:177`), 954 à 5 000 faux ancrages abandonnés par film. La sélection de la suite se fait AVANT la borne, et à longueur égale garde celle qui finit le plus haut (`DEC/internal/facts/objectives/scoreboard_stats.go:242-265`). Sur 9 507 joueurs, toute valeur finale d'éliminations à la grenade d'au moins 12 vaut exactement 16 (19 joueurs). 27 couples joueur-manche sur 14 026 ont un compteur supérieur aux éliminations. `maxSpree` n'est pas borné : 23 joueurs au-dessus de leurs éliminations, jusqu'à 29 812. Les compteurs de l'API sont chargés par l'oracle (`LU internal/domain/match_oracle.go:29-32`) mais jamais comparés par le banc | bornée (garde, comparaison au banc) ; recherche (origine du 16) | Piste C |
| Arme d'une touche par index brut | Confirmé (`DEC/replay/document_hits.go:157-180`) ; effet nul sur le parc (aucun rejeu à deux humains successifs sur un index) | bornée | Piste D |
| Tirs et touches : lecture suffisante | 46,0 % des joueurs (6 469 sur 14 053). Refus : couverture basse 20,6 %, doublons 9,6 %, les deux 9,8 %, aucune touche 5,4 %, aucun tir 1,2 %, lecture arrêtée 7,4 % (builds 1.8 à 1.10). Fusil d'assaut, BR, Sidekick : 65 à 71 % suffisants ; armes à plasma et à projectiles : 17 à 29 %, par structure (plusieurs messages par touche). Le témoin dit que l'événement de tir ne couvre qu'une partie des tirs, le reste étant des objets projectile | recherche (projectiles) ; décision (seuils) | Piste D |
| Porteurs du drapeau faux en Grande bataille | Bande au-delà de 8 joueurs fermée (`DEC/replay/porteurs_entrees.go:94-97`). Le faux porteur de la bande livrée est trouvé : film `390b1de5`, 94,3 s, drapeau publié « au socle » à 50,8 m du porteur, attribué par le socle le plus proche sans plafond de distance (`DEC/replay/flag_assign.go:230-241`). Une seconde chaîne (marqueur de portage) existe et n'est publiée qu'en compteurs. 19 portages sur 1 921 commencent à plus de 8 m du drapeau, dont 11 dans les 14 films de Grande bataille | recherche (origine de la prise) ; bornée si un veto suffit | Piste E |
| Présence d'un joueur qui part | La fin de présence est un fait LU (listage de session), pas une déduction (`DEC/replay/occupants_presence.go:112-125`). Écart avec la fin du dernier corps : médiane 6,1 s, 62 cas au-delà de 30 s (dont 17 morts suivies d'une longue attente). L'instant réel du départ dans le film : non établi | recherche | Piste F |
| Résumés d'usage : deux cas de bord | (a) place reprise à l'image du lâcher : possible par lecture du code, 0 cas sur 71 798 lâchers ; (b) mur déployé 1 à 3 images après la mort du poseur : 7 cas sur 9 178, crédités au mort par le décodeur, à personne par le site | bornée | Piste F (DD-3) |
| Lettre réelle des zones | Le document ne publie qu'un rang (`letterRank`). Le témoin lit la lettre dans un groupe d'états du marqueur ; le décodeur traverse ces mots sans les garder (`DEC/internal/grammar/components_navpoint_suite.go:53-66`). La jointure marqueur vers zone n'est pas établie | recherche | Piste G |
| Ordre des collines | La suite jouée est déjà dans le document (`zoneStates[].spans`), cohérente sur 48 films ; elle diverge entre matchs d'une même carte. L'ordre de la carte est une donnée de variante, pas du film. Aucun travail du décodeur requis : c'est une demande d'affichage | aucune | Reformulée au backlog (J.6) |
| Temps forts de type « mode » | Lus, non publiés dans le document ; LevelUp les range en base et les lit comme « zone capturée ». Le code de l'événement (octet 59) est dans la base de LevelUp | bornée (décompte) ; recherche (moitié Slayer) | Piste I |
| Commandes, saut, rechargements | Commandes de déplacement et bouton de saut lus dans la vue de contrôle puis jetés (`DEC/internal/grammar/frame_vue_controle.go:257-271`) ; rechargement = genre 38, traversé sans valeur. Volume : environ 500 rechargements par film, 4 à 9 Ko compressés. Le bit du bouton de saut n'est pas validé côté décodeur. Constat en plus : aucun accroupissement ni glissade publié sur 200 rejeux alors que le code les lit | bornée (rechargements) ; bornée après validation (saut, commandes) ; diagnostic (accroupi) | Piste H |
| Répliques des Spartans | Genres 75 et 76 traversés sans valeur. Le témoin lit locuteur, identifiant de son, indice de réplique, sans aucun texte. Le texte ou l'audio est dans les fichiers du jeu : aucune piste au dépôt | recherche (hors décodeur) | Piste H (DD-4) |
| Deux replis tenus par LevelUp | Déclarés dans le registre (`DEC/internal/facts/fallback/`), lus par `LU internal/filmreplis`. Retirer les deux ENTRÉES ne change ni sortie ni révision ; retirer les CONSTANTES exportées serait une rupture de compatibilité. L'inventaire du retrait côté LevelUp (`.ai/annexes_backlog_1117/verification_0_3.md`, section F) est encore exact sur le fond | bornée | Piste J |
| `medalLabels` sans jumeau | Trois types et un champ à déclarer ; aucun changement de sortie ; `DEC/replay/document.go` est à 500 lignes | bornée | Piste J |
| `replay-equiv` ne compare pas le tableau | Confirmé : l'étape `score` hache l'entrée, pas `scoreTimeline` | bornée | Piste J |
| Trajet de Warthog porté par l'humain (2026-11-10) | Non reproduit sur le document v1.4.1 | contrôle | Piste J |
| Pièces montées à plus de 15 m (2026-11-10) | Non reproduit : 0 paire au-delà de 15 m sur 879 comparées | contrôle | Piste J |
| Deux films aux faits d'un ancien schéma | Dépend de la régénération | aucune | Étape L |

## 2. Décisions

### Réponses du user du 2026-11-18

« ok avec tes recos, go pour les deux plans », avec deux compléments :
- **DD-2** : « pour les données des tirs ça me paraît peu, même si je peux accepter, j'avais l'impression
  que le décodeur de Den visait plus haut ». Effet : la piste D ne se borne plus à mesurer. D.2 compare, sur
  les mêmes films, notre lecture, celle du témoin et l'API ; D.3 devient un item ferme qui cherche à relever
  la part des joueurs bien lus (objets projectile, messages multiples par touche), validé contre l'API. Le
  critère d'affichage ne change que sur chiffres montrés au user.
- **DD-4** : « si tu peux trouver l'audio aussi ce serait cool ». Effet : H.5 cherche le texte ET l'audio
  d'une réplique dans les fichiers du jeu.

DD-1, DD-3 et DD-5 : retenues telles que recommandées.

### Texte des décisions, tel que posé (recommandation en tête)

- **DD-1 — ce que la version attend.** Recommandé : la v1.5.0 porte tout ce qui est BORNÉ, et les
  recherches qui aboutissent dans leur budget (une session d'agent chacune). Une recherche qui n'aboutit pas
  sort de la version avec ce qu'elle a écarté, et son entrée du backlog est mise à jour ; elle ne retarde ni
  la version ni la régénération. Autre voie : attendre toutes les recherches.
- **DD-2 — « données suffisantes » des tirs.** Recommandé : ne pas toucher au critère dans cette version ;
  mesurer d'abord contre l'API, par arme, ce que vaudrait un critère propre aux armes à plusieurs messages
  par touche, et te montrer le résultat. Le critère décide de ce que la fiche affiche : il ne se change pas
  sans chiffres.
- **DD-3 — mur déployé juste après la mort de celui qui l'a lancé** (7 cas). Recommandé : crédité au joueur
  mort, comme les équipements lâchés en mourant (ta décision du 2026-11-17) ; le site s'aligne sur le
  décodeur.
- **DD-4 — répliques.** Recommandé : chercher d'abord, dans les fichiers du jeu, le texte d'une réplique
  (recherche bornée à une session). Publier l'événement seulement si le texte se retrouve ; un événement
  sans texte n'aurait rien à afficher. Autre voie : publier l'événement brut dès maintenant.
- **DD-5 — modèle des recherches.** Recommandé : Opus pour quatre recherches dans la grammaire du film
  (origine du 16 des compteurs, origine de la prise du faux porteur, lettre des zones, barème du build
  1.11), Sonnet pour tout le reste, Fable pour l'étape 0, l'arbitrage des portes et la relecture finale.

### Tranché par le pilote (conventions, modifiables par le user)

- Tags et fusions du dépôt du décodeur : faits par le pilote (délégation du user des 2026-11-11 et
  2026-11-13) ; `main` de LevelUp reste un geste du user.
- Ordre des deux plans : le plan du backlog s'intègre le premier (il ne change pas le document de rejeu) ;
  la montée du décodeur vient après. Borne commune : 4 agents d'écriture à la fois, les deux plans confondus.
- Compatibilité : aucune constante ni fonction exportée du décodeur n'est retirée (marquée dépréciée au
  besoin) ; v1.5.0 reste une version mineure.
- Le coup d'envoi : la définition est celle du plan du backlog (piste T) ; ce plan-ci ne fournit que la
  lecture dans le film.
- Budget des fixtures de contrat (43 797 octets de marge) : relevé explicitement, daté, dans le commit qui
  ajoute un canal ; jamais contourné en amincissant un témoin existant.

## 3. Règles d'exécution : rigueur et prudence

- **R-1 — pas de correction sans mesure.** Chaque item qui change une sortie porte : sa mesure AVANT sur le
  parc, la vérité indépendante à laquelle il se compare (API, tableau du film, second canal du film), sa
  mesure APRÈS, et le compte des cas qui changent. Une correction qui améliore un compte et en dégrade un
  autre s'arrête et remonte au pilote.
- **R-2 — la grammaire prime.** Aucune heuristique de remplacement, aucun repli neuf (le banc le compte
  FAUX), aucune table par film ou par carte. Une règle vaut pour le parc et pour les films à venir.
- **R-3 — une recherche est bornée.** Une session d'agent ; sortie obligatoire : cause prouvée et item de
  correction écrit ici, daté, AVANT son code ; ou `[!]` avec ce qui a été écarté. Jamais de correction
  « plausible ».
- **R-4 — une piste, une branche, des écarts attribués.** Clone neuf de `DEC` sous `C:/t/dec/`, un
  worktree par piste (`C:/t/dec-<piste>`), sans jonction. Branche d'intégration `feat/v1.5.0`. Une piste
  n'entre dans l'intégration qu'après une passe d'équivalence (20 films, 12 minutes) dont CHAQUE écart est
  rattaché à un de ses items ; un écart non attribué bloque.
- **R-5 — dépôt public.** Aucun vrai identifiant de joueur dans une fixture, un golden, un test ou un
  message de commit ; contrôle avant chaque commit ; les agents ne poussent pas ; le pilote pousse après
  relecture. Plafond de 500 lignes ; aucune boucle de balayage bit à bit neuve.
- **R-6 — registre des changements de sortie.** `.ai/annexes_decodeur_1118/changements_de_sortie.md`, tenu
  par le pilote à chaque intégration de piste : champ, sens du changement, nombre de rejeux touchés. Il
  devient le corps des notes de version du décodeur.
- **R-7 — un seul schéma, une seule régénération.** 125 vers 126 une fois, une entrée de chronique qui liste
  les pistes ; aucune régénération partielle du parc pendant le plan ; les mesures APRÈS se font par le
  harnais sur un lot de films, pas sur le parc régénéré.
- **R-8 — machine.** Un seul décodage de lot à la fois ; jamais deux films en mémoire ; aucune suite complète
  lancée par un agent (tests ciblés ; les passes d'équivalence, le gate du corpus et les suites complètes
  sont au pilote).
- **R-9 — lecture seule des données.** Vérité de l'API : un export unique, fait à l'étape 0 sur une COPIE de
  la base partagée, serveur arrêté ou copie existante ; aucune base ouverte en écriture ; rien n'est écrit
  dans `data/`.
- **R-10 — lecteurs d'autres plans.** Le plan du fond « Relief » (session levelup-98) lit la hauteur `z`
  des points de piste du document de rejeu et `MatchPositionDTO.z`. Aucune piste de ce plan ne change le
  sens, l'unité ni l'emplacement de ce `z` ; si une piste devait y toucher, levelup-98 est prévenue AVANT
  le tag (demande reçue le 2026-11-18). Borne de machine partagée : levelup-98 tient 2 agents d'écriture au
  2026-11-18 ; ce plan et celui du backlog se partagent les 2 autres tant que c'est le cas, et le pilote lui
  annonce chaque lancement.
- Marche de publication : `.ai/annexes_decodeur_1118/marche_de_publication.md` (19 gestes), rejouée à
  l'étape 0 pour vérifier chaque commande.
- Reprise de session : ce fichier (cases, journal), le registre des changements de sortie, `git log` de
  `feat/v1.5.0` dans le clone, `git worktree list` des deux dépôts, `ListAgents`, puis `.ai/thought_log.md`.

| Piste | Items | Modèle | Effort | Risque |
|---|---|---|---|---|
| 0 témoins, vérités, instruments | 6 | pilote (Fable) ; un agent Sonnet pour les exports et les mesures | moyen | aucun |
| A coup d'envoi par l'horloge | 3 | Sonnet | rapide | faible |
| B verdict du score | 4 | Sonnet (B.1, B.2) ; Opus (B.3, selon DD-5) ; Sonnet (B.4) | moyen | moyen |
| C compteurs nommés | 4 | Sonnet (C.1, C.2, C.4) ; Opus (C.3) | moyen | moyen |
| D tirs et touches | 3 | Sonnet | moyen | faible |
| E drapeau en Grande bataille | 4 | Opus (E.1) ; Sonnet (E.2 à E.4) | lourd | élevé |
| F départs et usage | 3 | Sonnet | moyen | moyen |
| G lettre des zones | 2 | Opus (G.1) ; Sonnet (G.2) | lourd | moyen |
| H canaux neufs | 6 | Sonnet | lourd | moyen |
| I temps forts de type « mode » | 2 | Haiku (I.1) ; Sonnet (I.2) | rapide | faible |
| J ménage et contrôles | 6 | Sonnet (J.1, J.3, J.4) ; Haiku (J.2, J.5, J.6) | moyen | faible |
| L publication, montée, régénération | 7 | pilote ; Sonnet pour les miroirs de LevelUp | lourd | élevé |
| R relecture et clôture | 3 | Fable ; une ronde de corrections Sonnet | moyen | aucun |

Estimation : environ 14 agents Sonnet, 4 Opus, 4 Haiku, 1 Fable, plus les passes du pilote (une passe
d'équivalence par piste intégrée, soit une dizaine de fois 12 minutes ; un gate du corpus de 25 minutes ;
une régénération de 2 h 30 et ses dérivés).

Vagues : étape 0 ; vague 1 = J (elle libère `document.go`), A, C, I ; vague 2 = B, D, F, H ; vague 3 = E, G ;
puis L, puis R. Les recherches Opus (B.3, C.3, E.1, G.1) partent dès l'étape 0 close, une à la fois.

## Étape 0 — témoins, vérités, instruments (pilote ; un agent Sonnet)

- [x] 0.1 (fait le 2026-11-18 : `C:/t/dec/main`, tête `037ec4d` = v1.4.1, branche locale `feat/v1.5.0` ; `C:/t/dec-lu`) Clone neuf de `DEC` à la tête de `main` ; vérifier que `main` = v1.4.1 ; branche `feat/v1.5.0` ;
      espace de travail `GOWORK` ; worktree LevelUp `C:/t/dec-lu` depuis `release/v8.3.0`, sans jonction.
- [x] 0.2 (fait le 2026-11-18, sans jonction : gestes 3 à 10 rejoués ; équivalence 20 identiques sur 20 en 5 min 43 ; tests du clone avec les données de l'hôte : 33 paquets verts en 284 s ; gate du corpus : 19 témoins, ni gain ni perte, mais il juge la version PUBLIÉE, voir le journal ; annexe corrigée geste par geste) Marche de publication rejouée à blanc sur le clone SANS aucun changement : chaque commande de
      l'annexe tourne et rend son résultat attendu (tests avec les données de l'hôte, une passe
      d'équivalence à 20 identiques, gate du corpus sans gain ni perte). Tout écart avec l'annexe y est
      corrigé. C'est le témoin « avant » de tout le plan.
- [x] 0.3 (fait le 2026-11-18 : `C:/t/verite/`, hors dépôt) Vérité de l'API, export unique en lecture sur une copie de la base partagée, dans un dossier hors
      dépôt nommé au journal : par match et par joueur, éliminations, morts, assistances, tirs à la tête,
      corps à corps, grenades, armes lourdes, score personnel, tirs, touches, dégâts, temps de jeu, temps
      comme porteur du drapeau. Aucun identifiant de joueur n'entre au dépôt du décodeur.
- [x] 0.4 (fait le 2026-11-18 : `.ai/annexes_decodeur_1118/avant.md`, jointure à l'API pour 99,7 % des joueurs ; le point des 22 coups d'envoi a été tranché par l'audit du plan du backlog, pas rejoué ici) Mesures AVANT, par piste, sur le parc, dans `.ai/annexes_decodeur_1118/avant.md` : celles de la
      section 1 rejouées par les scripts gardés, plus, contre l'export 0.3 : taux d'accord de chaque compteur
      nommé par build ; somme des points contre score personnel ; temps de porteur ; durée de présence contre
      temps de jeu des partants ; coup d'envoi (horloge, mouvement) contre le premier événement daté du
      match, sur les 22 rejeux en désaccord et sur 50 autres.
- [x] 0.5 (fait le 2026-11-18 : quatorze références rouvertes sur la tête du clone, aucune ne diverge) Chaque référence `fichier:ligne` de la section 1 rouverte sur la tête du clone ; écarts au journal.
- [x] 0.6 (fait le 2026-11-18 : `.ai/annexes_decodeur_1118/instruments.md`, commandes vérifiées sur un film ; aucun harnais de lot n'existe : il se monte à partir de `replay-parc-vies --garder-documents`, `replay-verite` et `replay-diff`, premier item de la première piste qui change une sortie) Instruments : le harnais de mesure sur un lot de films (20 films de l'équivalence et les témoins de
      chaque piste) qui sort, pour une branche donnée, les compteurs des pistes ; il réemploie
      `cmd/replay-verite`, `cmd/replay-diff` et les outils de `DEC/research/` avant d'écrire quoi que ce soit.
- **Gate 0** : 0.2 à 20 identiques ; `avant.md` complet ; la question des 22 coups d'envoi est tranchée par
  0.4 et écrite au journal ; aucune piste ne démarre avant.

## Piste J — ménage et contrôles (Sonnet ; Haiku)

- [x] J.1 `replay/document.go` : extraction pour redescendre sous 460 lignes, sans changement de sortie
      (préalable à tout champ racine neuf). Témoin : golden de forme inchangé.
- [x] J.2 `medalLabels` : champ et trois types déclarés, jumeaux de ceux de LevelUp ; calque « à la
      requête » ; golden de forme refigé ; aucune sortie cuite ne change.
- [x] J.3 Deux replis tenus par l'hôte : les deux entrées du registre retirées, constantes gardées et
      marquées dépréciées ; décompte de `FORMAT.md` corrigé. Aucun changement de sortie.
- [ ] J.4 `cmd/replay-equiv` (dans LevelUp) : étape `scoreTimeline` (joueurs du tableau et leurs compteurs),
      nom inscrit aux étapes observées ; les 20 références refigées dans le clone, 20 identiques ensuite.
- [x] J.5 Commentaire faux de `Stances` (`document.go:168-181`) corrigé ; `FORMAT.md` et `GLOSSARY.md` à
      jour pour ce que cette piste touche.
- [x] J.6 Contrôles : trajet de Warthog du film `4f77afc1` et pièces montées à plus de 15 m, rejoués par le
      harnais sur la tête du clone ; si rien ne se reproduit, entrées closes au backlog avec la mesure ; si
      un défaut se reproduit, item écrit ici avant tout code. Ordre des collines : entrée du backlog
      reformulée en demande d'affichage (aucun travail du décodeur).
- [x] J.7 (fait le 2026-11-18, commit `2b7d7f7` du clone ; cause : un occupant qui monte ne réplique plus, aucune vie ne couvre l'instant et le nom retombait sur le premier occupant nommé du siège ; règle : le trajet porte le joueur du CORPS que les enregistrements de création placent sur le siège ; parc : 4 écarts sur 12 015 trajets, 2 films, `4f77afc1` et `3abf0258` ; mesure après sur `4f77afc1` : 1 trajet perd son nom, 251 inchangés, le repli du premier occupant passe de 283 à 192 ; `3abf0258` à contrôler à l'intégration) (né de J.6) Occupant d'un trajet de véhicule. Sur `4f77afc1`, un trajet de Warthog porte un humain
      alors que la vie qui le porte est celle d'un bot sans nom, et que cet humain n'est présent que plus tard :
      1 trajet sur 252 dans ce film, les 323 pistes nommées étant justes. Le nom du trajet vient de la table
      place vers joueur à l'instant (`replay/vehicle_rides_film.go`, et deux sites jumeaux), pas de la vie qui
      porte le trajet. Cause établie d'abord, fréquence comptée sur les 1 455 rejeux du poste, puis règle
      générale : un trajet porte l'identité de la vie qui le porte. Sortie changée : `vehicleRides[].xuid`.
- [ ] J.8 (né de J.6) Pièce montée rattachée à un porteur qui n'est pas né avec elle. `58d09c44` ne se
      reproduit plus (19 pièces, 0 écart), mais le compteur `coverage.vehicles.turretCarrierBirthMismatch`
      vaut 22 sur le parc, sur 4 films (`56ebfc53` 8, `af43b0c8` 8, `a26dbcdb` 4, `cf5d83d5` 2), contre 44
      montées relevées le 2026-11-16. Cause établie sur ces 4 films, puis règle générale ou, si la lecture ne
      désigne pas le porteur, arrêt et retour au pilote.
      Instruit le 2026-11-18, AUCUN code : dans les 22 cas le voisin refusé EST le bon porteur (même point,
      même image, bonne famille) ; il est refusé par le témoin de naissance à 100 ms (écarts mesurés : 105 à
      225 ms, pièces créées 2 paquets après leur châssis). Effet : la pièce n'a pas de porteur, elle n'est pas
      dessinée et ses 27 trajets d'artilleur ne sont pas reportés. Le composant qui dit le parent des pièces
      (`i40`) n'est jamais écrit pour elles ; leur création annonce un autre composant (`i10`, état du parent),
      que rien ne lit à la création. Décision du pilote : UNE tentative bornée de lecture de `i10` à la
      création (Sonnet, après la vague 1) ; si son handle désigne le châssis sur les 4 films et sur le corpus,
      le porteur devient une lecture et le témoin à 100 ms disparaît ; sinon `[!]`, entrée du backlog mise à
      jour avec ce constat. Aucun élargissement du seuil, aucune règle par voisinage de paquets.
- **Gate J** : `go test ./...` du clone avec les données de l'hôte ; passe d'équivalence, écarts = l'étape
  neuve de J.4 seulement.

## Piste A — coup d'envoi par l'horloge du jeu (Sonnet)

- [ ] A.1 Selon le verdict de 0.4 sur les 22 rejeux. Si l'horloge a raison : le coup d'envoi publié est
      l'instant où l'horloge de manche de la PREMIÈRE manche démarre (première lecture qui court, ramenée à
      la valeur de départ ; horloge qui monte comprise) ; le détecteur de premier mouvement et ses refus sont
      retirés ; la couverture dit la source. Si l'horloge a tort dans ces 22 cas : arrêt, retour au pilote,
      modèle revu avec le user avant tout code.
- [ ] A.2 Tests : horloge arrêtée puis qui descend ; déjà en marche à la première lecture ; qui monte ;
      plusieurs manches ; film sans horloge (aucune date, compté).
- [ ] A.3 Mesure APRÈS : 1 455 rejeux datés ; écart au coup d'envoi d'avant, par rejeu, au registre des
      changements de sortie.
- **Gate A** : équivalence, écarts = `t0FilmMs` et sa couverture ; les lecteurs de LevelUp
  (`sync/replayartifacts/t0film.go`) lisent le même champ, vérifié à la montée (L.4).

## Piste B — verdict du score (Sonnet ; Opus pour B.3)

- [ ] B.1 Le verdict « lecture complète » devient faux quand, pour un joueur au moins, la somme des points
      publiés diffère du score du tableau du film ; la couverture publie le nombre de joueurs en écart. Mesure
      attendue : les 44 rejeux faux passent à « incomplet », aucun rejeu exact ne bascule.
- [ ] B.2 Scripts inconnus des groupes : comptés et publiés en couverture au lieu d'être ignorés en silence
      (cas de l'Assaut).
- [ ] B.3 Recherche (une session) : le barème des véhicules du build `HI_1_11_0` (12 films, 313 joueurs,
      écarts multiples de 25) et la réponse de 100 points hors arbre (24 rejeux). Sortie : une table par build
      PROUVÉE par le tableau, ou `[!]`. Jamais un barème ajusté par régression.
- [ ] B.4 Mesure APRÈS contre le score personnel de l'API (export 0.3).
- **Gate B** : équivalence attribuée ; aucun rejeu exact avant ne devient inexact.

## Piste C — compteurs nommés du tableau (Sonnet ; Opus pour C.3)

- [x] C.1 (fait le 2026-11-18 : mode `-compteurs` de `cmd/replay-verite`, commit de `C:/t/dec-lu` ; la ligne « avant » sur les 1 455 rejeux retrouve les valeurs de départ à tous les chiffres) Le banc compare enfin les compteurs nommés à l'API (`cmd/replay-verite`, oracle déjà chargé) : le
      taux d'erreur par compteur et par build devient une mesure du gate, AVANT toute correction.
- [x] C.2 (fait le 2026-11-18, commit `d0911fb` du clone ; borne = éliminations publiées du même joueur dans la manche ; émission écartée en entier, comptée dans `coverage.score.statsGuarded` ; contrôle sur le parc sans décodage : la garde écartera 50 couples, les 27 et les 23 connus, dont AUCUN n'égale l'API ; aucun joueur n'a un compteur positif sans éliminations lues) Garde, appliquée AVANT la sélection de la suite : un compteur d'éliminations nommé ne dépasse pas
      les éliminations du joueur dans la manche ; `maxSpree` reçoit sa borne. Une valeur écartée est comptée
      et publiée en couverture, jamais remise à zéro en silence. Mesure attendue : les 27 couples violés et
      les 23 séries aberrantes disparaissent ; le taux d'accord avec l'API ne baisse pour aucun compteur.
- [ ] C.3 Recherche (une session) : d'où vient la valeur 16 (19 joueurs, tous builds) et pourquoi la bonne
      émission perd. Sortie : cause prouvée dans la lecture du tableau et correction écrite ici, ou `[!]`.
- [x] C.4 (fait le 2026-11-18 sur 70 films, 922 joueurs : couples violés 27 → 0, séries aberrantes 23 → 0, joueurs à 16 grenades 19 → 4 ; gagnés : série +23, grenades +22, corps à corps +10, tête +3 ; perdus : 0 ; aucun build ne baisse ; à rejouer sur le parc régénéré en L.7) Mesure APRÈS par C.1.
- **Gate C** : équivalence attribuée ; C.1 ne montre aucune baisse.

## Piste D — tirs et touches (Sonnet)

- [ ] D.1 Arme d'une touche et d'un dégât : résolue par le joueur à l'instant, comme l'auteur ; par l'index
      pour un bot (son identifiant est vide). Test : deux humains successifs sur un index. Sortie inchangée
      sur le parc, vérifiée.
- [ ] D.2 Mesure à trois voix, sans code de production, sur les 20 films de l'équivalence et 20 films
      choisis parmi les armes mal lues : par joueur et par arme, tirs, touches et dégâts de notre lecture,
      de celle du témoin (sa sonde, bâtie hors dépôt) et de l'API (export 0.3) ; part des joueurs à 10 % près
      de l'API pour chacune des deux lectures ; ce que donne le compte « événements de tir et objets
      projectile ». Résultat dans `.ai/annexes_decodeur_1118/tirs.md` et présenté au user.
      HYPOTHÈSE DU USER À TRANCHER PAR CETTE MESURE (2026-11-18 : « je me demande si le jeu n'enregistre pas que
      les tirs qui ont atteint leur but, les valeurs sont un peu élevées si c'est ça mais ça ressemble pas mal à
      la précision » ; déjà tentée par le passé) : par joueur et par arme, le nombre d'événements de tir du film
      est comparé AUX DEUX valeurs de l'API, tirs tirés et tirs au but ; rendre, par arme, la part des joueurs
      dont le compte du film est à 10 % des tirs tirés, la part à 10 % des tirs au but, et la distribution du
      rapport film / tirs tirés et film / tirs au but. Si le film suit les tirs au but, le dire, avec ce que cela
      change pour la précision (qui ne se calculerait alors plus par le film seul). Verdict écrit, quel qu'il soit.
- [ ] D.3 Relever la part des joueurs bien lus, par la grammaire seulement : tirs portés par un objet
      projectile ajoutés aux événements de tir s'ils ne se recoupent pas (preuve par D.2) ; messages
      multiples d'une même touche reconnus par ce que le film écrit (même auteur, même victime, même image),
      jamais par une fenêtre de temps. Chaque règle est écrite ici, datée, avant son code. Cible : fixée au
      journal après D.2, en part de joueurs à 10 % près de l'API. Le critère « données suffisantes » ne
      change que sur ces chiffres, montrés au user.
- **Gate D** : équivalence attribuée ; pour aucune arme la part des joueurs à 10 % près de l'API ne baisse.

## Piste E — drapeau en Grande bataille (Opus pour E.1 ; Sonnet)

- [ ] E.1 Recherche (une session) sur `390b1de5` et les 19 portages qui commencent loin du drapeau : d'où
      vient l'événement d'ouverture ; ce que disent la seconde chaîne (marqueur de portage) et les messages
      de jeu du drapeau que le témoin lit. Sortie : cause prouvée, ou `[!]`.
- [ ] E.2 Si E.1 conclut : la correction qu'il prescrit, écrite ici d'abord. À défaut de cause, AUCUN veto
      par distance n'est ajouté.
- [ ] E.3 Ouverture de la bande au-delà de 8 joueurs, seulement si E.1 a rendu juste la bande livrée et si
      le temps de porteur s'accorde à l'API sur les 9 films de Grande bataille à drapeau (export 0.3).
- [ ] E.4 Mesure APRÈS : temps de porteur contre l'API, par joueur, sur les 186 films à drapeau.
- **Gate E** : aucun porteur juste avant ne devient faux ; équivalence attribuée.

## Piste F — départs et résumés d'usage (Sonnet)

- [ ] F.1 Résumés d'usage : deux tests neufs (place reprise à l'image du lâcher ; mur déployé après la mort
      du poseur) ; le crédit suit DD-3. Côté LevelUp, la lecture du site s'aligne à la montée (L.4).
- [ ] F.2 Mesure : durée de présence et « fin du dernier corps moins arrivée » contre le temps de jeu de
      l'API, sur les 1 383 partants. Elle dit lequel des deux instants est le départ réel.
- [ ] F.3 Selon F.2 : si un fait du film date mieux le départ, item écrit ici avant tout code ; sinon `[!]`,
      et la règle d'affichage du plan du backlog (A.1) reste la réponse.
- **Gate F** : équivalence attribuée ; `match_usage_players` : 7 poses changent au plus.

## Piste G — lettre réelle des zones (Opus pour G.1 ; Sonnet)

- [ ] G.1 Recherche (une session), sur 5 films de Bases : l'état du groupe de lettre du marqueur se lit-il,
      et le lien du marqueur à une zone déjà publiée s'établit-il par le film seul. Sortie : oui avec la
      lecture, ou `[!]`.
- [ ] G.2 Si oui : champ `letter` ajouté à côté de `letterRank` (gardé), sur les 77 films de Bases ; accord
      avec le rang mesuré et écrit au registre.
- **Gate G** : équivalence attribuée.

## Piste H — canaux neufs (Sonnet)

- [ ] H.1 Rechargements : événements `{place, image}` lus dans le genre 38, avec leur couverture (lus,
      publiés, sans joueur, hors du match, trames arrêtées) ; doublons mesurés avant d'être traités.
- [ ] H.2 Validation du bouton de saut : sur 20 films, le front montant du bit contre les sauts déjà
      dérivés ; accord écrit au journal. Sous 90 %, H.3 ne publie pas le bouton.
- [ ] H.3 Bouton de saut et commande de déplacement : deux genres de plus dans `stances[]` (bouton enfoncé ;
      commande non nulle), jamais la série brute des valeurs.
- [ ] H.4 Diagnostic : pourquoi ni accroupissement ni glissade ne sont publiés sur 200 rejeux ; correction
      si la cause est une lecture, entrée au backlog sinon.
- [ ] H.5 Répliques, selon DD-4 : recherche bornée (une session), dans les fichiers du jeu installé et hors
      décodeur, du texte ET de l'audio d'une réplique à partir de l'identifiant de son et de l'indice que le
      film porte (point de départ : la chaîne des sons d'armes de `LU apps/go-api/cmd/weapon-sounds`). Si le
      lien s'établit : publication de l'événement par le décodeur, et côté LevelUp une table des répliques
      (texte en français et en anglais, son) faite comme celle des sons d'armes, lue par la fiche du joueur.
      Sinon `[!]` avec ce qui a été écarté.
- [ ] H.6 Côté LevelUp, à la montée : rechargements, sauts et temps en commande dans la section « En
      passant » de la fiche du joueur, dans les deux langues ; rien n'est affiché sans couverture suffisante.
- [ ] H.7 (ajouté le 2026-11-18, demande du user : « chaque frag, imagé dans le killfeed, est taggé d'une
      certaine manière quand on frag au tir à la tête ») Marque de tir à la tête PAR ÉLIMINATION. Aujourd'hui le
      décodeur ne la lit pas ; l'icône du fil de LevelUp vient de la lecture du dégât fatal, partielle. Recherche
      bornée (une session) dans le message d'élimination du film et dans l'enregistrement d'élimination des temps
      forts : un bit ou un champ qui vaut vrai exactement quand l'élimination est à la tête. VÉRITÉ, sans API :
      le compteur daté du tableau (`scoreTimeline.players[].stats.headshots`, 99,85 % d'accord avec l'API) : une
      élimination est à la tête si le compteur du même joueur monte à la même image. Une marque candidate se juge
      sur tout le corpus, élimination par élimination. Sortie : la lecture et un champ par élimination publié avec
      sa couverture, ou `[!]` avec ce qui a été écarté.
      AMENDÉ LE 2026-11-18 PAR LE USER (« on attend d'être sûr pour les tirs à la tête et les frags parfaits, je
      préfère lire ces infos depuis les compteurs. Comme ça l'expérience est complète et cohérente ») : la SOURCE
      voulue de ce qui s'affiche est le compteur du jeu, pas une lecture partielle. La session mesure donc
      D'ABORD l'appariement : pour chaque montée du compteur daté de tirs à la tête, y a-t-il exactement UNE
      élimination du même joueur à la même image (part appariée sans ambiguïté, cas à deux éliminations dans
      l'image, montées sans élimination, par version du jeu) ; de même pour chaque médaille « Parfait » et son
      élimination. Si l'appariement est sûr sur le parc, le champ par élimination se publie depuis lui (complet,
      cohérent avec les totaux de la fiche) ; la marque native du message d'élimination, si elle se trouve, sert
      de contre-preuve. Les icônes du fil de LevelUp (tête, parfait) ne reviennent QU'APRÈS ce verdict : rien
      n'est affiché d'ici là.
- **Gate H** : poids ajouté au document mesuré sur 20 films (cible : moins de 2 %) ; budget des fixtures
  relevé et daté ; équivalence en deux passes pour le canal de faits neuf.

## Piste I — temps forts de type « mode » (Haiku ; Sonnet)

- [x] I.1 (fait le 2026-11-18 : `.ai/annexes_decodeur_1118/temps_forts_mode.md` ; 23 683 événements, 527 matchs, un seul code réel, 10) Décompte par mode et par code de l'événement, depuis la copie de la base de LevelUp ; croisé avec
      les destructions de véhicule et les actions de score du document.
- [x] I.2 (fait le 2026-11-18, sans code : un temps fort « mode » est l'écho d'une ACTION DE SCORE propre au mode, du même joueur, dans la même image (plage de 0 à 40 ms) : 76 % des événements de code 10 appariés. Par mode : zone capturée ou sécurisée en Bases, contrôle de la colline, vol, prise, retour, capture du drapeau et élimination du porteur en Drapeau, contrôle de la balle ; en Slayer : destruction de véhicule (12 noms), aide par IEM, quelques éliminations. Sens inverse : 85 à 100 % des actions de score de ces noms ont leur temps fort. Non appariés : 3 410 en Drapeau, 691 en Total Control (aucune action de score publiée pour ce mode), 303 en Extraction, dont le sens reste inconnu. Le décodeur n'a rien à publier de plus : `scoreActions` nomme déjà l'événement. Entrée du backlog mise à jour : la lecture « zone capturée » de LevelUp est fausse hors Bases) Selon I.1 : ce que la moitié inexpliquée en Slayer recouvre, ou `[!]` avec le décompte ; l'entrée
      du backlog est mise à jour, avec le crédit à nuzzles et à Den si leur lecture est reprise.
- **Gate I** : aucun code de production.

## Étape L — publication, montée, régénération (pilote)

- [ ] L.1 Intégration des pistes dans `feat/v1.5.0`, une à une (règle R-4) ; schéma 126, chronique,
      révisions de couche, goldens, fixtures de contrat (annexe, gestes 3 à 7).
- [ ] L.2 Équivalence finale refigée, 20 identiques ; gate du corpus ; registre des replis ; documentation
      du format (gestes 8 à 11).
- [ ] L.3 PR du décodeur, CI verte, tag v1.5.0, release GitHub dont le corps est le registre des changements
      de sortie (geste 12).
- [ ] L.4 Montée dans LevelUp sur une branche `chore/830-decodeur-v150` : `require`, miroirs Go et web,
      contrat de l'API, fixtures, lecteurs alignés (coup d'envoi, usage, « En passant »), retrait de
      l'ancienne précision par film (inventaire fermé de la section F) et de la liste des champs « servis
      seuls » (gestes 13 à 15).
- [ ] L.5 Gates de LevelUp sans `GOWORK`, suite complète du web, `make gate-push`, CI (geste 16) ; notes de
      version dans les deux langues, changelog, journal, backlog, registre des reports (geste 17).
- [ ] L.6 (porte : go du user) Régénération du parc, une fois, serveur arrêté, puis les dérivés (geste 18).
- [ ] L.7 Mesures APRÈS sur le parc régénéré, comparées à `avant.md` ; vérifications à l'écran inscrites à
      l'artefact des vérifications du user ; gestes de prod inscrits à la liste de la release (geste 19).
- **Gate L** : chaque ligne du registre des changements de sortie retrouvée dans les mesures APRÈS.

## Étape R — relecture et clôture

- [ ] R.1 Relecture adversariale (skill `adversarial-review`, Fable) du diff du décodeur et du diff de la
      montée ; axes imposés : heuristique déguisée, repli neuf, identifiant réel, sortie changée sans ligne
      au registre.
- [ ] R.2 P0 et P1 corrigés en une ronde ; P2 au backlog seulement s'ils valent le coup.
- [ ] R.3 Tous les items statués ; recherches closes ou `[!]` avec leurs pièces ; backlog mis à jour entrée
      par entrée ; worktrees retirés un à un après recherche de jonction.

## Journal d'exécution

- 2026-11-18 — icônes de tir à la tête et de frag parfait dans le fil du rejeu : présentes jusqu'à la 8.2.0,
  perdues à la refonte de l'interface sans décision ni maquette. Le user ne veut PAS les remettre sur une
  lecture partielle : elles attendent le verdict de H.7 (amendé : source = compteurs du jeu). Machine : la
  mémoire est retombée à 0,3 Go libre (éditions de liens Go en parallèle entre sessions) ; agents Go bornés à
  une ou deux tâches à la fois, mesure avant chaque suite large.
- 2026-11-18 — hypothèse du user sur les tirs (le film n'écrirait que les tirs au but) inscrite à D.2 comme
  question à trancher par la mesure à trois voix, plutôt qu'au backlog : la piste D est déjà au plan et D.2 ne
  touche aucun code de production.
- 2026-11-18 — H.7 ajouté (marque de tir à la tête par élimination), accord du user (« ok »). Côté LevelUp, la
  fiche du joueur lira dès maintenant le compteur daté du tableau (plan du backlog, Q.2-g). Page Notion demandée
  par le user, créée en brouillon privé : « Compteurs lus dans le film : ce qu'on a et leur justesse »
  (https://app.notion.com/p/41c7ae87e7a3813299caf95332f7bc32), chiffres repris de `avant.md` et du banc de C.
- 2026-11-18 — GO D'AVANCE du user pour la régénération du parc (L.6), message du 2026-11-18 : « Pour les trucs
  plus tard je te donne mon go d'avance » (en réponse à la liste : régénération des rejeux avec la v1.5.0,
  rattrapage du coup d'envoi sur sa base, fusion dans `main`). La porte de L.6 est ouverte ; le pilote prévient
  avant de lancer (serveur arrêté, environ 2 h 30, mémoire libre vérifiée, pas pendant que le user joue).
  Règle apprise à l'intégration de A : le golden de forme refuse d'être régénéré deux fois sous le même numéro
  de schéma ; tant que le schéma n'est pas PUBLIÉ, on restaure le golden de la version publiée
  (`git show 037ec4d:replay/testdata/document_shape.golden`) puis on passe la porte une fois (à reporter dans
  la marche de publication).
- 2026-11-18 — INTÉGRATION DE J dans `feat/v1.5.0` (`4e33bd9`) : schéma 126, chronique, golden de forme, 8 goldens
  d'assemblage, 8 fixtures de contrat (dans `C:/t/dec-lu/apps/web`, non commitées : elles seront refigées par
  A et C), 20 références d'équivalence. Écarts attribués : `schemaVersion`, l'étiquette `publication-126`, et
  le compteur du repli du premier occupant du siège en baisse sur 8 films à véhicules ; aucun trajet ne change
  sur le corpus ; aucune révision de couche ne bouge ; 20 identiques après refige ; suite du clone verte (33
  paquets). Contrôle du second film touché, `3abf0258` : 3 trajets sur 191 perdent leur nom faux, 188
  inchangés ; 2 changent aussi de borne de fin et 1 de visée, parce qu'un trajet est coupé à la vie suivante de
  SON occupant (`vehicle_rides_next_life.go`) : sous le nom d'un autre joueur il était coupé par la vie de ce
  joueur-là. Mécanisme vérifié sur pièces par le pilote et écrit dans la chronique. Piste A fusionnée
  (`ea2a089`), son intégration est lancée ; plafond de poids des fixtures de contrat à relever à 5 111 808
  octets (décision du pilote, datée : les entrées figées portent désormais les lectures d'horloge). C.3 lancé
  (Opus, recherche bornée).
- 2026-11-18 — piste C : C.1, C.2 et C.4 rendus, relus, commités (`d0911fb` dans le clone, banc dans
  `C:/t/dec-lu`). Fait rougir, à refiger à l'intégration : la révision de la couche objectives, son empreinte
  au golden des révisions, le golden de forme (champ `statsGuarded`) ; aucun golden d'assemblage, aucune
  fixture de contrat. Reste C.3 (recherche Opus) : 4 joueurs à 16 grenades dont la valeur tient dans leurs
  éliminations (`6347cbf9`, `d6a9a42e`, `d8cd155a`, `f795dbb3`).
- 2026-11-18 — piste I close (I.1, I.2), aucun code. Découverte pour LevelUp : 2 330 temps forts « mode » de
  la base n'ont pas de code ; sur les 33 matchs qui ont les deux, ce sont des DOUBLONS des événements de code
  10 (1 284 sur 1 284, même joueur, même instant) ; 43 matchs de juin à septembre 2026 n'ont que ceux-là.
  Piste A commitée dans le clone (`702bfe0`) : la piste de précision par « premier battement » est RÉFUTÉE
  par la mesure (l'horloge se lit par pas d'environ 0,55 s, le coup d'envoi est connu à ce pas près), règle
  inchangée ; lectures d'horloge ajoutées aux entrées figées des 8 goldens (69 Ko). À l'intégration de A, le
  plafond des fixtures de contrat sera dépassé de 11 720 octets (elles gagnent `roundClock` et
  `respawnCountdowns`) : relèvement daté à écrire. Intégration de la piste J lancée (schéma 126).
- 2026-11-18 — J.7 rendu, relu, commité (`2b7d7f7`). Il fait rougir 4 fixtures de contrat et 4 goldens
  d'assemblage (`a521164d`, `11de8353`, `111fa685`, `e5adf7b2`) sur le seul compteur du repli du premier
  occupant du siège, en baisse (82→68, 87→75, 89→84, 125→112) : à refiger à l'intégration, attribué à J.7.
  J.8 instruit sans code (voir l'item). Piste A rendue une première fois : règle codée, 34 films mesurés
  (médiane −0,37 s ; 8 refusés datés ; 6 films à +16,8 s) ; deux compléments demandés avant commit
  (précision quand le film commence horloge en marche ; lectures d'horloge ajoutées aux entrées figées des
  goldens). Vague 1 : pistes C (C.1, C.2) et I lancées.
- 2026-11-18 — piste J rendue, relue sur pièces, commitée dans le clone (`a2d5dbb`, branche `piste/j`, non
  poussée). `document.go` à 456 lignes ; `medalLabels` déclaré, seule l'empreinte de la forme stockée bouge,
  schéma inchangé ; registre à 127 replis, les deux constantes gardées et marquées obsolètes dans
  `decfilm/replis_obsoletes.go` ; gate complet du clone vert (33 paquets), révisions et périmètres sans
  régénération. Contrôles de J.6 : les deux défauts existent encore, items J.7 et J.8 écrits avant tout code ;
  les deux entrées du backlog restent ouvertes et renvoient à ces items. Découverte : le commentaire de
  `StanceCoverage.Absent` parle de trois composants là où quatre sont lus (corrigé avec J.7).
- 2026-11-18 — pistes J (J.1, J.2, J.3, J.5, contrôles de J.6) et A lancées, un agent Sonnet chacune, dans
  deux dossiers de travail du clone (`C:/t/dec-j` branche `piste/j`, `C:/t/dec-a` branche `piste/a`, tous deux
  partis de `feat/v1.5.0`). Aucun agent ne monte le schéma, ne régénère un golden hors de sa porte, ne commite
  ni ne pousse : la piste A rend la liste exacte de ce qu'elle fait rougir, le pilote l'attribue à
  l'intégration. J.4 (étape `scoreTimeline` de l'équivalence, côté LevelUp) reste au pilote.
- 2026-11-18 — ÉTAPE 0 CLOSE (Gate 0 : marche à blanc à 20 identiques ; `avant.md` complet ; coup d'envoi
  tranché). Valeurs de départ à retenir : accord exact des points publiés avec le score de l'API 98,42 % sur
  les rejeux « complets » (69,33 % au build 1.11) ; compteurs nommés 99,63 à 99,97 % d'accord, défaut concentré
  (27 couples, 23 séries, 19 joueurs à 16 grenades que l'API ne dit jamais) ; tirs, touches, dégâts à 10 % de
  l'API pour 59,7, 58,4 et 64,8 % de TOUS les joueurs (Needler : 19,3 et 40,1 %) ; portage du drapeau à 2 s de
  l'API pour 91,7 % des joueurs à 8 ou moins et 85,5 % au-delà ; pour un partant, la présence publiée est plus
  proche du temps de jeu de l'API que la fin du dernier corps dans 91,8 % des cas. Correction de la section 1 :
  accroupissements et glissades SONT publiés, mais dans 7 rejeux seulement (1 129 et 1 039). Les pistes
  attendent des places d'écriture : le plan du backlog passe d'abord, et la session du fond Relief a les deux
  prochaines places.

- 2026-11-18 — marche à blanc rendue. Trois faits qui changent la conduite du plan :
  (1) LE GATE DU CORPUS NE JUGE PAS LE CLONE : il compile contre le décodeur que `go.mod` exige. Pour juger
  `feat/v1.5.0`, la branche de la montée désignera une pseudo-version du décodeur (permise sur une branche de
  travail, ADR 0043), jamais un `replace` commité ; L.2 se lit ainsi. Il faut aussi `GOWORK=off` pour que le
  registre des replis d'avant soit lu.
  (2) LES TESTS DU CLONE SE LANCENT AVEC `GOWORK=off` : le fichier d'espace de travail engendré écrit le chemin
  du clone avec des barres obliques, et `go test` d'un sous-paquet du clone échoue alors (« module not one of
  the workspace modules »).
  (3) SANS JONCTION, l'équivalence lit ses 20 films dans une copie sous `C:/t/dec-lu/data/cache` (676 Mo, ignorée
  de git) et le gate dans un parc de travail sous `C:/t/dec/sorties/parc` ; le dossier principal n'est jamais
  la racine d'une commande qui écrit (verrou de décodage, cache du gate).
  Aussi : un sixième golden de formes (`types/`) manquait à l'annexe ; le verrou des fixtures de contrat est
  `REPLAY_CONTRACT_UPDATE` ; huit décisions du 2026-11-16 de `config/replay_corpus.toml` sont devenues sans
  effet (avertissements du gate), à retirer en L.2. 0.4 (mesures « avant » contre l'export) lancé.

- 2026-11-18 — 0.3 fait (export de vérité sous `C:/t/verite/` : `participants.tsv` 29 107 lignes,
  `registre.tsv` 2 045, `evenements_mode.tsv` 58 ; hors dépôt, copie de la base, original jamais ouvert). La
  question des 22 coups d'envoi (Gate 0) est TRANCHÉE par l'audit du plan du backlog : l'horloge a raison. Dans
  ces 22 rejeux, le troisième joueur à bouger réellement le fait 0,17 à 0,84 s après le départ de l'horloge et
  16,8 à 21,6 s après le « premier mouvement » ; première élimination de l'API 16,1 s après l'horloge contre
  32,5 s ; arrivées des joueurs de l'API 0,2 s après l'horloge. A.1 suit donc sa première branche. À savoir
  pour les pistes : le registre n'a pas de colonne de build ; le code d'un temps fort de mode est `type_hint`
  (10), son détail dans `raw_json`, pas dans `details_json`.
- 2026-11-18 — 0.1 fait : clone neuf `C:/t/dec/main`, tête `037ec4d` = tag `v1.4.1` ; worktree LevelUp de la
  montée `C:/t/dec-lu` (`chore/830-decodeur-v150`, depuis `1a7329e625`), sans jonction. Lancés : la marche à
  blanc (0.2, 0.5, 0.6 ; un agent Sonnet, consigne : aucune jonction, trouver la voie par paramètres) ; l'export
  de vérité de l'API (0.3), fait par l'agent d'audit du coup d'envoi du plan du backlog, sur une copie de la
  base sous `C:/t/verite/` (serveur local à l'arrêt, vérifié). 0.4 attend ces deux retours.
- 2026-11-18 — go du user (« ok avec tes recos, go pour les deux plans ») ; DD-1 à DD-5 retenues ; piste D
  renforcée (comparaison à trois voix, relèvement de la part des joueurs bien lus) ; H.5 étendu à l'audio.
- 2026-11-18 — plan écrit après trois instructions en lecture seule et deux mesures du pilote ; aucun code.
  Écart signalé par un agent d'instruction : un `git fetch` lancé dans la copie locale du témoin, contre la
  consigne de lecture seule ; aucun fichier modifié.

## Découvertes (consignées, non traitées)

- 2026-11-18 (0.2) — `scripts/briques-workspace.sh` écrit le chemin du clone avec des barres obliques : sous
  Windows, `go test` d'un sous-paquet du clone échoue avec le `GOWORK` engendré. À porter au backlog (petit
  correctif d'outillage).
- 2026-11-18 (0.2) — `replay-equiv -deux-passes` signale sur v1.4.1 « 1 écart à classer : kill », déjà présent
  avant tout changement.

- 2026-11-18 — les 5 rejeux sans build n'ont aucun compteur nommé lisible (non instruit).
- 2026-11-18 — sur le film `390b1de5`, l'équipe du porteur est inconnue pour 25 portages sur 25 (carte à
  6 socles : la règle par équipe se tait).
