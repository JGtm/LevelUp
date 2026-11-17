# PLAN — Refonte de la page du rejeu : trois cadres, fiches en rangée, fil sur le plan (2026-11-15)

Statut : **rédigé le 2026-11-15 sur go du user ; maquettes validées le 2026-11-15 ; D-1 à D-16 tranchés par
le user le 2026-11-15 (D-11 corrigé par lui : le Standard reprend la maquette A, sauf la zone de lecture) ;
aucun code écrit ; D-17 tranché (effets gardés) ; D-18 (frise inchangée) validé ; D-19 à D-24 (fiche à l'instant, lot T)
tranchés le 2026-11-16 ; go d'exécution donné par le user le 2026-11-17 (plan maître, DM-4).**

**Amendé le 2026-11-17** par le plan maître `.ai/PLAN_MAITRE_8_3_0_2026-11-17.md` (ordre entre plans, fichiers
partagés, borne machine, conformité aux maquettes) et par le plan `.ai/PLAN_BACKLOG_2026-11-17.md` (items E.1 à
E.4 : sprints et sauts, médailles du fil, mesures par build, garde du score). Les amendements sont marqués
« (amendé le 2026-11-17) » à leur place. Worktrees : `C:/t/ui-<piste>`, sans aucune jonction.

**Conformité aux maquettes** : la grille `.ai/annexes_refonte_ui_rejeu/conformite_maquettes.md` fait partie de
ce plan. Hors de ses 11 écarts voulus, la maquette fait foi. Le brief de tout agent d'interface porte les
fichiers de maquette de son tableau et les lignes de la grille qu'il réalise ; son compte rendu statue chaque
ligne avec sa preuve ; le pilote relit la fixation DOM contre la grille avant de cocher.

Contrat d'exécution : skill `plan-execution` (ordre strict par piste, aucun report, statut de chaque item,
vérification sur pièces, aucun correctif hors périmètre). Les règles d'orchestration de la section 1 bis le
précisent pour ce plan. Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (motif
écrit) ; aucune case vide à la clôture. Un item est clos quand son test est écrit, vert, et relu sur pièces
par le pilote ; un lot est clos quand son gate est passé.

Maquettes de référence : canevas https://claude.ai/artifact/GkqDthVG6jGxx3A5WLfXrr, tableaux A (Standard),
J (fiche détaillée du Standard : la colonne devient la fiche), K (Cinéma), L (fiche détaillée du Cinéma et du
Plein cadre : fenêtre), D (Plein cadre), H (rotation). Les tableaux B, C, E, F, G, I sont des étapes
précédentes et ne font pas foi.

Plan lié : `.ai/PLAN_REJEU_3D_2026-11-15.md`. Les points de contact sont listés en section 1 ter ; deux
items du plan 3D sont amendés par celui-ci (C.5 et D-10).

---

## 0. Objectif et critère de succès

Objectif : la page du rejeu propose trois cadres pour un même lecteur. La mise en page dépend du cadre
(et du nombre de places, D-12), jamais de la carte. Le Standard prend la maquette A ; sa zone de lecture
(frise à pistes, commandes de lecture, capture, export, zoom) reste celle d'aujourd'hui.

| Lot | Ce que le user voit | Critère de succès |
|---|---|---|
| 0 | Rien | Témoins « avant » capturés ; mesure M-1 (mémoire de la toile) écrite ; audit Fable rendu |
| A | Trois icônes de cadre à droite de la barre de lecture ; T, F, Échap | Zone de lecture du Standard identique aux témoins hors icônes de cadre ; Cinéma prend la largeur du corps ; Plein cadre recouvre la fenêtre du navigateur sans plein écran système ; le bouton plein écran système fonctionne seul |
| S | Standard selon la maquette A : page à largeur bornée, score dans l'en-tête, colonne de 360 px (équipes en lignes, puis fil d'une ligne par élimination) | Zone de lecture et zoom identiques aux témoins ; bouclier et santé en deux barres ; Grande équipe lisible |
| B | Cinéma et Plein cadre : 4 fiches, le score, 4 fiches, en une rangée au-dessus du plan | Bouclier et santé en deux barres ; repli en colonne selon D-12 ; aucune couleur en dur |
| C | Fil des éliminations posé en bas à gauche du plan, 5 lignes au plus, sans en-tête | Lieu = victime dans les trois cadres ; lisible sur une carte large ; masquable dans « Vue » |
| D | Standard : clic sur une ligne joueur, la colonne devient la fiche (tableau J). Cinéma et Plein cadre : clic sur une fiche, fenêtre accrochée sous elle (tableau L) | Lecture non interrompue ; Échap ferme la fiche avant le cadre ; même contenu de fiche dans les deux formes |
| T | La fiche détaillée compte à l'instant lu : FDA, séries, tirs à la tête, parfaits, répartition des éliminations, vies, boucliers perdus, médailles ; score et précision quand le match le permet | Chaque valeur exacte égale au tableau de score en fin de match ; une valeur estimée porte la mention « estimation » ; aucune valeur affichée sans lecture suffisante |
| E | Barre de lecture d'une ligne en Cinéma et Plein cadre | Mêmes commandes que le Standard, repères d'éliminations et dominance sur la barre ; zone de lecture du Standard inchangée |
| F | 2D / 3D en superposition à côté de « Vue » ; plan tourné automatiquement quand il y gagne | 2D / 3D invisible tant que la 3D n'existe pas (capacité) ; rotation selon D-7, bouton « Plan tourné » |
| G | Plan plus grand sur grand écran en Cinéma et Plein cadre | Plafond de hauteur relevé seulement si M-1 le permet (D-15) |
| R | Rien de neuf | Revue finale Fable, une ronde de corrections, gate-push vert, validation visuelle du user |

## 1. Décisions

### Tranchées par le user (2026-11-15, canevas et messages du jour)

- **D-1 — trois cadres.** Standard (page à largeur bornée, tableau A), Cinéma (lecteur sur la largeur du corps de l'app),
  Plein cadre (le lecteur recouvre la fenêtre du navigateur, sans la navigation de l'app, SANS plein écran
  système). Le plein écran système reste un bouton à part. Noms « Cinéma » et « Plein cadre » validés.
- **D-2 — Cinéma = tableau K, Plein cadre = tableau D** (structure de K sans l'app autour). Message du user :
  « K à la place de C pour le mode cinéma ».
- **D-3 — fiches en rangée.** Eagle à gauche du score, Cobra à droite, sur une seule rangée au-dessus du plan.
  Bouclier et santé en DEUX barres distinctes (message du user, la maquette n'en montrait qu'une). Grenades et
  équipement visibles. Aucun liseré gauche ; couleurs = tokens `team-ally` / `team-enemy` et tokens de l'app.
- **D-4 — fil sur le plan.** En Cinéma et Plein cadre : posé en bas à gauche du plan, sans en-tête, sans fond
  ni bordure, une ligne par élimination aux colonnes fixes (instant | tueur | arme | victime | lieu), 5 lignes
  au plus, la plus récente en bas. Le fil est le seul élément posé sur la carte. Une option de « Vue » le masque
  (accordée par le user : « ok avec ton point »).
- **D-5 — lieu = celui de la victime**, dans les trois cadres. Une ligne du fil qui concerne le joueur suivi
  porte un fond légèrement teinté (token), à la place de l'ancien liseré gauche refusé par le user ; vaut pour
  S.3, C.2 et D.2.
- **D-6 — fiche détaillée : deux formes, un seul contenu.** Contenu commun : Suivre / Seul / Masquer,
  « À l'instant » séparé de « Totaux du match », défilant.
  - **Standard = tableau J** (message du user du 2026-11-15 : « j'avais validé la vue J hors mode cinéma et
    plein écran ») : la colonne de 360 px DEVIENT la fiche ; en-tête « Fiche » avec retour « Équipes » et
    Échap ; sélecteur des joueurs en rectangles arrondis (pas de pastilles rondes) pour changer de joueur ;
    section « Éliminations » = le fil du match réduit à ses 2 dernières lignes (tous joueurs, le joueur ouvert
    mis en avant), lien « Fil complet » qui rend la colonne ;
    frise à pistes inchangée (D-18). Le mécanisme actuel qui garde le fil monté
    sous la fiche (`ReplayFeedOrPanel`) est conservé et étendu, pas supprimé.
  - **Cinéma et Plein cadre = tableau L** : fenêtre non bloquante accrochée sous la fiche cliquée, fermée par
    Échap ou la croix ; pas de sélecteur (la rangée en tient lieu).
- **D-7 — rotation.** Le plan tourne de 90° quand il y gagne au moins 25 % de surface dans le cadre ; bouton
  « Plan tourné » pour revenir (tableau H). La mise en page, elle, ne change pas.
- **D-8 — Standard.** « A est bien » ; « pas besoin de revoir le lecteur et les options de zoom qui sont bien
  positionnés ». Précisé par le user le 2026-11-15 : « les contrôles ne changeaient pas, là où il y avait les
  frises et les contrôles de lecture, export et tout, le reste c'est repris de la maquette ». Donc : la zone de
  lecture actuelle (`ReplayTransport` : frise à pistes `ReplayTimelineTracks`, lecture, ±10 s, vitesse, son,
  capture, export) et la commande de zoom actuelle restent telles quelles en Standard ; tout le reste suit le
  tableau A (D-11).
- **D-9 — superpositions.** Le bouton « Vue » reste en superposition en haut à droite de la toile (déjà le cas,
  `settings/ReplayViewMenu.tsx`). Le commutateur 2D / 3D va en superposition à côté de lui. Les cadres
  s'affichent en icônes, pas en texte.

### Proposées puis validées par le user (2026-11-15, réponse point par point)

- **D-10 — emplacement des icônes de cadre.** À droite de la barre de lecture, dans les trois cadres : Cinéma
  (bascule avec Standard), Plein cadre, plein écran système. L'en-tête de page ne porte plus de sélecteur de
  cadre. Motif : c'est la place des lecteurs vidéo, et le coin haut droit de la carte porte déjà « Vue » et
  2D / 3D.
- **D-11 — Standard, périmètre exact (corrigé par le user).** Le Standard suit le tableau A :
  - page à largeur bornée et centrée (le Cinéma, lui, prend la largeur du corps) ; la borne est une constante
    nommée (proposition de la maquette : 1200 px de contenu) ;
  - en-tête : retour, titre du match, bandeau de score resserré au centre (environ 440 px, manches et barre de
    prise de zone comprises), lien vers la fiche du match ; plus de bandeau de score au-dessus du terrain ;
  - sous l'en-tête : la carte à gauche, sans rien par-dessus hors « Vue », 2D / 3D et le zoom actuel ; UNE
    colonne réservée de 360 px à droite : Eagle, Cobra, puis le fil ;
  - équipes en LIGNES (nom, bouclier et santé en deux barres, arme, grenades, équipement, F / M / A ; temps
    avant réapparition à la place des barres quand le joueur est mort) ; le joueur suivi a sa ligne dépliée
    (armes et munitions, grenades, charge de l'équipement) ;
  - fil de la colonne : une ligne par élimination aux colonnes fixes (instant | tueur | arme | victime | lieu),
    sans fond ni bordure, plus récente en tête ; son titre reste le menu de filtre actuel ; il remplit la place
    sous les équipes et défile ;
  - zone de lecture et zoom : ceux d'aujourd'hui (D-8), plus les icônes de cadre (D-10) ;
  - sous la zone de lecture : les sections actuelles (carte de chaleur, usage de l'équipement) sans changement.
  Grande équipe : les lignes d'équipes défilent dans leur part de la colonne, le fil garde au moins 5 lignes.
- **D-12 — quand la rangée ne tient pas.** La rangée s'affiche si le match a exactement deux camps, chacun de
  5 places au plus, ET si la largeur du corps loge toutes les fiches au gabarit compact (`GABARIT_COMPACT`,
  115 px) plus le score. Sinon (Grande équipe, FFA, fenêtre étroite), le Cinéma et le Plein cadre prennent la
  colonne de 360 px du Standard (équipes en lignes puis fil, lot S) ; la barre compacte et les icônes restent. Règle pure,
  calculée sur les cotes du gabarit, jamais sur un littéral de largeur.
- **D-13 — la caméra de la barre compacte.** L'icône caméra ouvre un menu à deux sections : « Suivre » (la
  caméra, même action que Suivre dans la fiche) et « Point de vue » (par les yeux de qui : `useReplayViewpoint`,
  aujourd'hui dans le menu de la première piste de la frise). En Standard, rien ne change.
- **D-14 — orientation.** Sans choix retenu pour la carte, l'orientation est « Auto » (règle D-7). Le panneau
  « Vue » gagne la position « Auto » devant 0° / 90° / 180° / 270°. « Plan tourné » n'apparaît dans la barre
  que quand l'Auto a tourné le plan ; un clic retient 0° pour cette carte (même stockage qu'aujourd'hui).
  La règle s'évalue à l'entrée dans un cadre et après chaque redimensionnement stabilisé (délai existant de
  `useReplayViewport`), avec une hystérésis : tourne à +25 %, ne revient qu'en dessous de +20 %.
- **D-15 — plafond de hauteur de la toile.** `CANVAS_HEIGHT_CEILING` (720 px, garde-fou mémoire des quatre
  calques cuits) bride le Cinéma et le Plein cadre sur un écran haut. Il n'est relevé (proposition : 1080 px
  en Cinéma et Plein cadre seulement) que si M-1 mesure un coût mémoire acceptable (critère : au plus +150 Mo
  de tas JS au pic, aucun gel visible au redimensionnement). Sinon il reste à 720 px et le lot G se clôt `[!]`
  avec la mesure.
- **D-16 — base.** `release/v8.3.0` (décision du user du 2026-11-15, relayée par la session pilote : la refonte sort de la 8.2.0). Sous-branche
  `feat/rejeu-refonte-ui`, un worktree par agent (`LevelUp-wt-ui-<piste>`, SANS jonction vers `data/`).

- **D-17 — effets des fiches : TOUS gardés** (message du user du 2026-11-15 : « on les garde tous »). Flash à
  la mort, ligne éteinte avec « Réapparition dans … », flash à la réapparition, glissement au changement
  d'arme, équipement qui s'allume à l'usage, bloc hachuré « ÉLIMINÉ » compris. Vaut pour B.2 (fiches reprises
  telles quelles) et S.2 (les lignes reprennent les mêmes effets, adaptés à leur hauteur).

### Tranchées par le user le 2026-11-16 (contenu de la fiche à l'instant)

- **D-19 — valeurs exactes à l'instant**, dans « À l'instant » (lot T) : FDA (formule du jeu), meilleure série et
  série en cours, tirs à la tête (quand le coup fatal est mesuré), éliminations parfaites (médaille datée),
  réapparitions, durée de la vie en cours et moyenne des vies jusqu'à l'instant, médailles du joueur.
- **D-20 — répartition des éliminations** : une seule répartition catégorisée (arme lourde, mêlée, grenade,
  autres armes), chaque catégorie et chaque arme avec son nombre et sa part en pourcentage, à la place de trois
  compteurs séparés (message du user : « un simple breakdown catégorisé pour dénombrer et avec un pourcentage »).
- **D-21 — boucliers perdus** : nombre de fois où le bouclier du joueur est tombé à zéro, dérivé de la valeur de
  bouclier publiée à chaque point (le film n'écrit pas d'événement « bouclier brisé », vérifié le 2026-11-16 dans
  notre décodeur et dans libfilmshell). Seul le bouclier du joueur lui-même compte.
- **D-22 — estimations** : précision, tirs et touches, dégâts infligés et subis, et par arme les tirs et les
  dégâts, à l'instant, SEULEMENT pour un joueur dont la lecture des tirs est jugée suffisante par le document, et
  marqués « estimation ». Score et rang dans le lobby à l'instant SEULEMENT quand le document déclare le score
  complet. Sinon la ligne n'existe pas (aucun texte « non mesuré »).
- **D-23 — repris d'OpenSpartan** : grenades lancées par type, ramassages (armes, grenades, équipement). Écartés :
  rang compétitif à l'instant (fixe pendant le match, il reste dans « Totaux »), Rendement / Résistance, rivaux,
  XUID, XP d'équipement. Reportés au backlog (chantier décodeur) : commandes de déplacement, bouton de saut et
  rechargements en notes de bas de page ; répliques des Spartans ; idées tirées des boucliers perdus (badge
  « os de verre », frags sans bouclier). (amendé le 2026-11-17) Sprints, sauts et visée ne sont PAS reportés :
  le document les publie déjà (`stances` : `sprint` lu, `jumpDerived` calculé ; visée publiée ; instruction du
  backlog du 2026-11-17, 15 656 sprints sur 30 rejeux).
- « Totaux du match » ne change pas : chiffres officiels de fin de match.
- **D-24 — notes de bas de page anticipées** (message du user du 2026-11-16 : « on anticipe l'UI mais le travail
  côté décodeur va aller dans le backlog »). La fiche réserve, sous « Totaux du match », une section « En
  passant » : sprints, rechargements, visées, sauts, part du temps en mouvement, à l'instant. T.5 en fixe la
  place et l'ordre dans la mise en page du corps de fiche. (amendé le 2026-11-17, décision D-3 du plan backlog,
  user : « ok avec tes recos ») Le document porte déjà les sprints et les sauts : « En passant » affiche dès la
  refonte le temps cumulé de sprint et le nombre de sauts, ce dernier marqué « estimation » (item T.7). Les
  rechargements, les commandes de déplacement et la part du temps en mouvement restent sans lecture : rien
  n'est rendu pour eux (une lecture écrite maintenant serait du code mort, règle 7) ; ils arrivent avec le
  chantier décodeur du backlog (« Commandes et gestes du joueur »).
- **D-18 — frise à pistes : rien de nouveau** (ok du user le 2026-11-16 : « la frise à piste c'est pas mal comme
  c'est déjà »). La première piste a déjà son menu de joueur (point de vue) et
  montre ses éliminations et ses morts. La mise en avant des repères dessinée sur la maquette J ferait doublon :
  elle est retirée, la frise reste celle d'aujourd'hui (D-8). Ouvrir une fiche ne change pas la frise.

## 1 bis. Orchestration : pistes parallèles et modèles

Demande du user (2026-11-15) : implémentation parallélisable, **Sonnet** principalement, **Haiku** quand
possible, **Opus** si nécessaire ; **Fable** pour l'étape 0 et la revue finale (règle du 2026-11-15).

Règles d'exécution (elles précisent le contrat `plan-execution`) :
- **Pistes.** Dans une piste, l'ordre est strict : un item est clos et vérifié avant le suivant. Des pistes
  différentes tournent en même temps. Seules les dépendances de la colonne « Attend » comptent.
- **Un agent par worktree**, `LevelUp-wt-ui-<piste>` depuis `feat/rejeu-refonte-ui`, sans jonction vers
  `data/`, `GOCACHE` dédié. Aucun agent n'ouvre de navigateur ni ne lance l'app : les preuves sont des tests
  vitest et des fixations DOM ; la vérification à l'écran revient au user (artefact de vérifications).
- **Agents sans commit ni push.** Le pilote relit le compte rendu sur pièces, commite dans le worktree, intègre
  dans `feat/rejeu-refonte-ui` par rebase, lance les portes.
- **Bornes.** Au plus 4 agents Sonnet ou Opus d'écriture à la fois ; les Haiku s'y ajoutent. Un seul
  `make gate-push` à la fois sur la machine (y compris ceux du plan 3D).
- **Fichiers partagés : un seul écrivain à la fois**, désigné par la table : `ReplayPage.tsx`,
  `ui/ReplayCanvas.tsx`, `ui/ReplayTransport.tsx`, `settings/ReplayViewMenu.tsx`,
  `settings/replayPreferences.ts`, `hooks/useReplayFullscreen.ts`, `hooks/useReplayShortcuts.ts`,
  `i18n/fr.ts` + `i18n/en.ts` (et leur contrat `i18nContract.ts`).
- **Modèles.** Sonnet par défaut. Haiku pour un item mécanique, contrôlé sur pièces par le pilote. Opus
  seulement là où la table le dit, avec son motif. Fable pour 0.3 et R.1. Tout agent porte `model` explicite.

| Piste | Item | Modèle | Attend | Note |
|---|---|---|---|---|
| P0 Préalables | 0.1 témoins « avant » (fixations DOM du Standard) | Haiku | go | sorties capturées, aucun code applicatif |
| P0 | 0.2 M-1 mémoire de la toile à 720 / 1080 px | Sonnet | go | l'agent prépare la mesure, le user la lance à l'écran |
| P0 | 0.3 audit de conception | **Fable** | go | lecture seule ; découpage des composants, risques, écarts à ce plan ; complète la grille de conformité |
| P0 | 0.4 maquette de la fiche enrichie (lot T) | pilote | go | un tableau de plus sur le canevas ; validé par le user avant T.5 (plan maître, DM-3) |
| P1 Cadres | A.1 modèle des cadres (pur) | Sonnet | 0.3 | `model/replayFrame.ts` |
| P1 | A.2 plein cadre / plein écran système | Sonnet | A.1 | seul écrivain de `useReplayFullscreen.ts`, `useReplayShortcuts.ts` |
| P1 | A.3 page découpée par cadre | Sonnet | A.1, 0.1 | seul écrivain de `ReplayPage.tsx` ; zone de lecture identique aux témoins |
| P9 Standard | S.1 page bornée, en-tête avec score | Sonnet | A.3 | seul écrivain de `ui/frames/ReplayStandardFrame.tsx` pendant le lot S |
| P9 | S.2 colonne : équipes en lignes | Sonnet | 0.3 | `ui/ReplayRosterRow.tsx`, `ui/ReplayRosterColumn.tsx` |
| P9 | S.3 fil de la colonne d'une ligne | Sonnet | C.1, S.2 | restyle de `ReplayKillFeed` ; filtre conservé |
| P9 | S.4 montage de la colonne dans le Standard | Sonnet | S.1, S.3 | puis Gate S |
| P2 Rangée | B.1 règle de la rangée (D-12), pure | Haiku | 0.3 | fonction et tests sur les cotes du gabarit |
| P2 | B.2 rangée de fiches + score au centre | Sonnet | B.1, A.3, S.2 | réutilise `ReplayPlayerCard` au gabarit compact ; repli sur `ReplayRosterColumn` |
| P3 Fil | C.1 lieu de la victime (règle pure) | Sonnet | 0.3 | à côté de `killerZone.ts`, même jeu de cas pour le nommage |
| P3 | C.2 fil superposé 5 lignes | Sonnet | C.1, A.3, E.1 | ancré au bloc de la toile (slot E.1) |
| P3 | C.3 option « Fil » dans Vue | Haiku | C.2 | seul écrivain de `ReplayViewMenu.tsx`, `replayPreferences.ts` pendant l'item |
| P4 Fiche | D.1 corps de fiche partagé | Sonnet | 0.3 | extrait de `ReplayPlayerPanel`, rendu inchangé |
| P4 | D.2 fiche du Standard (J) | Sonnet | D.1, S.4 | étend `ReplayFeedOrPanel` ; ne touche pas `ReplayTimelineTracks.tsx` |
| P4 | D.3 fiche en fenêtre (L) | Sonnet | D.1, B.2 | `useAnchoredPanel` ; couche d'Échap |
| P5 Barre | E.1 slots de `ReplayCanvas` (barre, superpositions du bloc) | Sonnet | 0.3 | seul écrivain de `ReplayCanvas.tsx` ; voir 1 ter (D.2 du plan 3D) |
| P5 | E.2 boutons partagés extraits de `ReplayTransport` | Sonnet | E.1 | seul écrivain de `ReplayTransport.tsx` ; zone de lecture identique aux témoins |
| P5 | E.3 barre compacte d'une ligne | Sonnet | E.2, A.1 | frise compacte : repères + dominance |
| P5 | E.4 menu caméra (D-13) | Sonnet | E.3 | |
| P6 Plan | F.1 règle d'orientation Auto (pure) | Haiku | 0.3 | `lib/replay/orientation.ts` |
| P6 | F.2 Auto, « Plan tourné », section Vue | Sonnet | F.1, C.3, E.3 | écrivain de `ReplayViewMenu.tsx` / `replayPreferences.ts` APRÈS C.3 |
| P6 | F.3 emplacement 2D / 3D en superposition | Sonnet | F.2 | invisible sans capacité ; branché par le plan 3D (C.5) |
| P7 Plafond | G.1 plafond par cadre | Sonnet | 0.2 validée, A.1 | `[!]` si M-1 refuse (D-15) |
| P10 À l'instant | T.0 mesures sur le parc | Sonnet | 0.3, régénération du parc par le user | lecture des documents de rejeu déjà générés, AUCUNE génération ; mesures par build du jeu |
| P10 | T.6 médailles du match lues dans le document | Sonnet | 0.3 | Go (libellé et image résolus par le serveur) puis lecture web unique ; `openapi.yaml` et `generated.ts` par le pilote |
| P10 | T.7 sprints et sauts (pur) | Sonnet | 0.3 | relecture du témoin `11de8353` d'abord |
| P10 | T.1 compteurs exacts à l'instant (pur) | Sonnet | 0.3, T.6 | `model/playerInstantStats.ts` ; médailles par la lecture de T.6 |
| P10 | T.2 répartition des éliminations (pur) | Haiku | T.1 | `model/killBreakdown.ts` |
| P10 | T.3 score, rang et estimations (pur) | Sonnet | T.0, T.1 | gardes de suffisance et de score complet |
| P10 | T.4 grenades lancées, ramassages (pur) | Haiku | T.1 | |
| P10 | T.5 rendu dans le corps de fiche | Sonnet | D.1, T.2, T.3, T.4, T.6, T.7, 0.4 validée | seul écrivain de `ReplayPlayerPanelBody.tsx` après D.1 |
| P8 Clôture | R.1 revue finale | **Fable** | lots A à G, S et T clos | adversarial-review du diff complet |
| P8 | R.2 corrections retenues | Sonnet ; Opus si l'écart est un algorithme | R.1 | une seule ronde |
| P8 | R.3 journal, mémoire, registre des reports | pilote | R.2 | |

Vagues :
- **Dès le go** : 0.1 (Haiku), 0.2 (Sonnet), 0.3 (Fable).
- **Après 0.3** : A.1, S.2, C.1, E.1, D.1, T.6, T.7 (Sonnet), B.1, F.1 (Haiku), sous la borne commune de 4
  agents d'écriture du plan maître ; T.1 dès T.6 ; T.2 et T.4 (Haiku) dès T.1 ; T.0 dès la régénération du
  parc ; T.3 dès T.0 et T.1. E.1 repart de `ReplayCanvas.tsx` allégé par l'extraction du moteur 2D (item D.2 du
  plan 3D, fusionné d'abord : plan maître, P-5).
- **Puis** : A.2 / A.3, E.2 → E.3 → E.4 ; S.1 dès A.3, S.3 dès C.1 et S.2, S.4 ; B.2 dès A.3 et S.2 ;
  C.2 dès A.3 et E.1.
- **Fin** : D.2, D.3, T.5, C.3, F.2, F.3, G.1 ; R.1, R.2, R.3.

## 1 ter. Points de contact avec le plan 3D

- **`ReplayCanvas.tsx`.** Le plan 3D extrait `buildScene` et `draw` du fichier (D.2, piste P3). E.1 ne touche
  que le rendu JSX (slots), pas la scène. Un seul écrivain à la fois. (amendé le 2026-11-17, plan maître P-5)
  Ordre fixé : D.2 d'abord, sur sa branche `refactor/rejeu-moteur-2d`, fusionné dans `release/v8.3.0` dès son
  gate ; E.1 repart du fichier intégré.
- (amendé le 2026-11-17) Les deux amendements ci-dessous sont portés dans le plan 3D le 2026-11-17, sans
  attendre R.3 : les maquettes de la 3D se dessinent sur le plan amendé.
- **Amendement du plan 3D, item C.5** : le commutateur 2D / 3D ne va pas
  DANS le panneau de `ReplayViewMenu.tsx` mais dans l'emplacement en superposition posé par F.3 (D-9).
- **Amendement du plan 3D, D-10 (HUD)** : en Cinéma et en Plein cadre, les fiches et le fil restent à l'écran
  (D-2, D-4) ; le HUD 3D n'a donc à reprendre ni le fil ni le score, dans aucun cadre.
- **Nord de la mini-carte 3D (D-7 du plan 3D)** : c'est l'orientation EFFECTIVE de la 2D, Auto comprise (D-14).
- **D-8 du plan 3D** reste vrai : la 3D prend la place de la toile 2D, les trois cadres l'accueillent tels quels.

## 1 quater. Gate 0 (2026-11-17) : décisions du user et amendements de l'audit

Source : `.ai/annexes_refonte_ui_rejeu/audit_conception.md` (audit 0.3, Fable, constats recoupés sur pièces
par le pilote sur quatre points). **Cette section prévaut sur le texte des items quand ils diffèrent.**

### Décisions du user (2026-11-17, réponses à des questions fermées)

- **G0-1 — fiches de la rangée : « Garder l'arme dans la fiche ».** La rangée du Cinéma et du Plein cadre garde
  la fiche compacte actuelle à trois lignes, arme comprise, avec TOUS ses effets (D-17 tient en entier). De la
  maquette K restent : la rangée qui remplit la largeur (parts égales), la bordure à la couleur de suivi sur la
  fiche du joueur suivi, le fond clair et la bordure claire de la fiche ouverte. L'ajout K-4 du 2026-11-17
  (contenu de la tuile de K) est retiré : écart voulu V12 de la grille.
- **G0-2 — « suivi » = le joueur regardé.** C'est le joueur de la page au départ, modifiable. « Suivre » reste
  l'action de caméra, à part, éteinte au départ comme aujourd'hui.
- **G0-3 — ligne du fil : « Médailles gardées ».** Dans la colonne du Standard, les médailles restent sur la
  ligne, entre la victime et le lieu ; l'assistance et le badge « porteur » passent en infobulle. Sur le plan :
  ligne nue, comme la maquette.
- **G0-4 — barre d'une ligne : « Tout reprendre ».** L'interrupteur de lecture automatique, « Recommencer » et
  les repères de médias entrent dans la barre d'une ligne : mêmes commandes dans les trois cadres. Écart voulu
  V13 (la barre est plus chargée que K-10).
- **G0-5 — Grande équipe : « Lignes resserrées ».** En Grande équipe seulement, des lignes plus basses (nom,
  bouclier et santé, F / M / A, sans arme ni grenades) pour voir une vingtaine de joueurs sans défiler. Forme
  sans maquette : montrée au user AVANT le code (item 0.5).
- **G0-5 bis — Grande équipe : fiches compactes, deux joueurs par emplacement (remplace « lignes resserrées »).**
  Retours du user du 2026-11-17 sur le tableau N : « Pourquoi […] déplacerait on le score ? et ne peut on pas
  faire une rangée avec deux joueurs ? on masque les stats FDA […] c'est comme le mode 4v4 mais plus compact »,
  puis « ok pour la maquette N version 2 dans la forme mais […] j'aimais pas les bordure colorée sur la gauche ».
  Décidé : le score ne bouge jamais ; fiche compacte = nom, bouclier, santé, SANS frags, morts ni assistances,
  sans arme, grenades ni équipement (visibles dans la fiche du joueur au clic) ; couleur du camp = pastille
  devant le nom comme K, AUCUNE bordure colorée à gauche. Cinéma et Plein cadre : la rangée de K est gardée,
  6 emplacements de deux fiches empilées par camp (fiche de 87,5 px sur 34 px à 1440 px, rangée de 74 px, soit
  16 px de plus que K). Standard : dans la colonne, 6 lignes de deux fiches côte à côte par camp, fiche sur UNE
  ligne de 24 px (nom puis jauges), puis le fil. Conséquences : B.1 ne refuse plus la Grande équipe en rangée
  (règle propre : 6 emplacements par camp au gabarit compact) et B.2 la dessine ; S.2 rend la Grande équipe en
  paires de fiches compactes dans la colonne, sans défilement ; un gabarit « compact Grande équipe » nommé dans
  `cardGabarit.ts` porte ces cotes. Le drapeau du porteur, la pastille « bot » et « Réapp. 4,2 s » abrégé sont validés avec
  le tableau (« ok bien je valide la maquette », user, 2026-11-17, après retrait du liseré).
- **G0-6 — objectif sous le score : « Rien sous le score ».** L'ajout K-7 du 2026-11-17 est retiré : écart
  voulu V14.

### Retours du user sur la maquette M (2026-11-17, second message)

- **G0-7 — ordre de la ligne du fil** (« pour le fil c'est chrono - zone de callout - tueur - arme - victime ») : instant, lieu, tueur, arme, victime, dans la colonne, sur le plan et au pied de la fiche. Le lieu reste celui de la victime (D-5). Dans la colonne, les médailles suivent la victime (G0-3). Écart voulu V17 ; A-14 et K-9 sont amendés.
- **G0-8 — équipement dans sa propre catégorie** (« manque ramassage et usage/perte équipements ») : la fiche gagne une section « Équipement », par famille : pris, utilisé, perdu (lâché en mourant). Item neuf **T.8** (Sonnet) : lecture pure à l'instant, bâtie sur les lectures d'usage de l'équipement déjà écrites pour la page du match (même définition de « utilisé » par famille) ; lire `.ai/archive/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` AVANT de coder. « Ramassages » ne garde que les armes et les grenades (T.4). **T.8 [x] fait le 2026-11-17** : `model/playerEquipmentStats.ts`, lectures datées partagées avec la grille du match (`model/equipmentGestures.ts`), fin de match égale à la grille sur 8 documents réels ; découverte au backlog : les lâchers sont sous-comptés par la grille (attribution stricte), décision du user attendue.
- **G0-9 — médailles comme sur les tuiles de match** : icône de 32 px avec son halo de rareté, nom dessous, « ×n » (`components/ui/match-card.tsx`, `MedalIcon`). T.6 sert donc aussi l'image et la rareté de chaque médaille.
- **G0-10 — « En passant » est refusé comme titre** (« que je déteste »). Proposé : « Gestes » ; attend l'accord du user.
- **G0-11 — « Totaux du match » et « Répartition des éliminations » se ressemblent trop** : deux formes dessinées dans la version 2 du tableau M (une barre empilée et sa légende, totaux en liste sans barres ; ou un seul tableau par arme, l'instant et le match côte à côte). Le user choisit.
- Reste aussi à choisir sur M : « armes de puissance » (mot actuel de la fiche, dessiné) ou « arme lourde » (D-20).

### Validation de la maquette M par le user (2026-11-17, « Forme 2, Gestes ok, armes de lourdes »)

- **Tableau M validé** (`FicheInstant.dc.html`) : il fait foi pour le contenu de la fiche, en colonne (J) et en fenêtre (L). L'item 0.4 est clos.
- **Forme 2 retenue** : un seul tableau « Éliminations par arme », par catégorie puis par arme, avec trois colonnes : à l'instant, part, match. « Totaux du match » ne répète plus les armes (précision, dégâts, score, éliminations à la tête). G0-11 clos.
- **« Gestes »** remplace « En passant » (D-24, T.5, T.7). G0-10 clos.
- **« Armes lourdes »** est le nom de la catégorie (D-20) ; le libellé « armes de puissance » de la fiche actuelle est remplacé dans la fiche.
- **Barre composite et classes d'armes** (user, 2026-11-17 : « j'aimais bien aussi la barre composite de la forme 1 […] on respectera le code couleur des classes d'armes qu'on utilise partout comme la page Synthèse ») : au-dessus du tableau, une barre empilée des éliminations à l'instant par classe. Les catégories de T.2 sont donc les CLASSES DE FRAG de l'application, pas les quatre catégories de D-20 : jetons `frag-shoulder`, `frag-sidearm`, `frag-heavy`, `frag-melee`, `frag-grenade` et suivants (`lib/accessibility/semantic-tokens.ts`), libellés du manifeste `frags.class.*` (« Arme d'épaule », « Arme de poing », « Lourde », « Mêlée », « Grenade »…). T.2 réutilise le classement existant d'un frag dans sa classe (celui de la page du match et de la Synthèse), sans en écrire un second. Le mot choisi par le user, « armes lourdes », désigne la classe que le manifeste nomme « Lourde » : le libellé du manifeste fait foi. Confirmé par le user le 2026-11-17 : « pour la classe d'arme on garde "lourde" comme c'est déjà là ».
- Ordre des sections de la fiche, de haut en bas : état à l'instant ; depuis le début du match (compteurs, score et rang) ; éliminations par arme ; tirs et dégâts (estimation) ; grenades lancées ; ramassages (armes, grenades) ; équipement (pris, utilisé, perdu par famille) ; médailles ; totaux du match ; gestes ; en colonne seulement, les deux dernières lignes du fil.

### Idée du user renvoyée au backlog (2026-11-17)

- **G0-12 — barre de lecture en surimpression.** Idée du user (« juste une idée ») : en Cinéma et en Plein cadre, poser la barre de lecture sur le plan, comme un lecteur vidéo, et la faire disparaître sans mouvement de souris ; il doute lui-même de l'effacement (« sur ordinateur on bouge pas mal la souris […] un repli ou autre comportement serait mieux »). Proposition du pilote, NON validée : en Plein cadre seulement, la barre d'une ligne se replie en une fine ligne de progression (rail, repères, dominance) au bas du plan pendant la lecture, et se déplie au survol du bas du plan, à toute touche et en pause ; le fil posé sur le plan remonte quand elle est dépliée ; le Cinéma et le Standard gardent leur barre fixe. Décision du user (2026-11-17, « note au backlog, il me faudra une maquette ») : hors de ce plan ; entrée dans `.ai/BACKLOG.md`, reprise sur maquette validée, après la barre d'une ligne.

### Tranché par le pilote sur la règle « la maquette fait foi » (dit au user le 2026-11-17)

- Fil posé sur le plan : éliminations seules (D-4 et K-9), sans le filtre du Standard ; il ne capte pas le
  pointeur et passe SOUS les surcouches de carte (écran de fin).
- Sous le lecteur du Standard : rien (la page n'a aucune section sous le lecteur ; D-11 parlait de sections
  « actuelles » qui n'existent pas).
- Menu de la caméra : petit panneau à deux listes, « Suivre » (avec « Aucun ») et « Point de vue ».
- T.6 suit le patron réel de `service/replay_vehicle_labels.go` : une table `medalLabels` par nom de médaille
  et par langue DANS le document, lue par `pickText` ; la clé de requête du document ne porte pas la langue.
- ~~T.2 lit la catégorie « arme lourde » dans `weaponLabels[].role`, mêlée et grenade dans `killEffects`.~~ FAUX,
  constaté sur pièces le 2026-11-17 à l'exécution de T.2 : `role` ne vaut jamais « lourde » (un fusil de précision
  a le rôle `sniper` et la classe `heavy`), et `killEffects` ne porte que des familles d'effet (la grenade et le
  lance-roquettes y partagent `explosive`). Remplacé par l'item **T.2b** (Sonnet) : le serveur pose `class` sur
  `weaponLabels[]` à la requête, depuis le registre des armes (`weapons.ClassesByKey()`, la source de la
  Synthèse), par le chemin qui pose déjà `role` ; le web en tire la table arme -> classe ; mêlée sans arme et
  grenade rangées comme la Synthèse si le fil le permet, sinon « non classées ». Aucun rejeu régénéré.
  **Révisé le 2026-11-17 au retour de T.2b** : la table tirée de `weaponLabels` est incomplète par construction
  (grenades fragmentation et plasma, armes jamais tenues, mêlée sans arme n'y figurent pas). La classe se pose
  donc PAR FRAG, sur l'événement de frag du contexte du match, là où le serveur pose déjà la clé d'arme
  (`weapon_class`, registre des armes ; mêlée sans arme par une propriété déclarée de la règle du pont) ;
  `weaponLabels[].class` n'est pas livré ; `killBreakdown` lit la classe du frag, sans table.
  **T.2b [x] fait le 2026-11-17** : `weapon_class` sur `MatchHighlightEvent`, posé par `service/match_view_killfeed_weapon.go`
  depuis `games/halo_infinite/killsource_class.go` ; 189 des 193 sources que le pont résout reçoivent une classe.
  RÉSERVE À MESURER EN T.0, avant T.5 : un frag dont la source de dégât n'est pas appariée reste « non classé » ;
  la dernière mesure connue (2026-08-11, tout le parc) ne donnait une source qu'à un tiers des frags. Mesurer sur
  le parc régénéré la part de frags classés des matchs AVEC rejeu ; si elle est basse, la section « Éliminations
  par arme » se décide avec le user (masquée sous un seuil de données, ou affichée avec sa part non classée).

### Amendements techniques de l'audit (par item)

- **Préalables neufs, avant tout item qui les attend :**
  - **I.1** [x] fait le 2026-11-17, `fr.ts` 494 -> 474 lignes (Haiku) `equipmentUsage` sort de `i18n/fr.ts`, `i18n/en.ts` et du contrat vers
    `i18n/equipmentUsage.i18n.ts` (patron de `playerPanel.i18n.ts`), aucun appelant ne change.
  - **I.2** [x] fait le 2026-11-17, libellés relevés sur les maquettes validées (pilote, même commit) sous-dictionnaires montés une fois : `i18n/frames.i18n.ts` (lots A, E, F,
    C.3), `i18n/roster.i18n.ts` (S.2, B.2, sélecteur de J), `i18n/playerInstant.i18n.ts` (« À l'instant », lot
    T). Ensuite aucun lot ne touche `fr.ts`, `en.ts` ni le contrat : un écrivain par sous-dictionnaire.
  - **M.1** [x] fait le 2026-11-17, le joueur suivi est le point de vue de la page (aucun second état) ; `followXuid` survit à la fermeture et au changement de fiche ; tant que E.4 ne câble pas la caméra fiche fermée, elle ne suit que si la fiche du joueur suivi est ouverte (Sonnet, avant S.2, B.2, E.4) joueur suivi indépendant de la fiche dans `model/playerPanelLogic.ts`
    (G0-2) : valeur de départ = joueur de la page ; la caméra « Suivre » reste un état à part. Tests du
    réducteur ; `playerFocusWiring.guard.test.ts` vert.
  - **X.1** [x] fait le 2026-11-17, fixations identiques à l'octet ; cliquet des fonctions longues abaissé à 249 (Sonnet, avant S.2) effets des fiches extraits sans changer le rendu : `hooks/useCardScene.ts`
    (depuis `ui/ReplayTeams.tsx`) et `ui/ReplayCardFx.tsx` (depuis `ui/ReplayPlayerCard.tsx`) ; fixations
    `replayTeams.4v4.html` et `6v6.html` inchangées.
- **A.1** : aucun test de la touche Échap dans le module pur ; Échap en Plein cadre rend le cadre d'où l'on
  vient, Échap en Cinéma ne fait rien.
- **A.2** : la touche T passe par les trois relais de F (`ui/ReplayCanvas.tsx`, `hooks/useReplayTimeline.ts`,
  `hooks/useReplayShortcuts.ts`) ; toute bascule de cadre annonce un redimensionnement ; aucune bascule
  pendant un export, T et boutons neufs compris (test).
- **A.3 : UN gabarit à zones**, la toile montée une seule fois. `ui/frames/ReplayFrameLayout.tsx` (zones
  `tete`, `rangee`, `scene`, `colonne`), `ui/frames/ReplayStage.tsx` (l'unique montage de la toile),
  `ui/frames/ReplayFrameHeader.tsx` (en-têtes A, K, D ; titre de page et badge d'administration),
  `ReplayStandardFrame.tsx` et `ReplayCinemaFrame.tsx` ne fournissent que le contenu des zones. Test
  obligatoire : basculer de cadre ne remonte pas la toile. Le garde `ui/rosterHeight.guard.test.ts` suit le
  code déplacé. Bandeau de régénération, avis « instant non calé » et états de chargement gardent une place
  dans les trois cadres.
- **S.1** : cotes nommées du bandeau de score par emplacement (en-tête, rangée), `readScoreBanner` intact.
- **S.2** : gabarit « ligne » dans `cardGabarit.ts` ; `ui/ReplayRosterRowDetail.tsx` pour le dépli ; lignes
  resserrées de la Grande équipe selon 0.5 (G0-5) ; lignes et tuiles portent `data-replay-player-card` et
  `PRESS_GUARDS`.
- **S.3** : ordre et largeurs de la ligne changent (pas « seul le lieu ») ; les fixations 0.1 du fil servent de
  référence de contenu, pas d'identité ; la liste filtrée sort dans `hooks/useFeedView.ts` ; ligne cliquable et
  ligne choisie gardées ; médailles selon G0-3.
- **S.4** : rien sous le lecteur. **S.5 (neuf) [x] fait le 2026-11-17** (`ReplayTeams`, `ReplayTeamHeader`, `cardDensity`, leurs 7 tests et les fixations 4v4 et 6v6 supprimés, 2 921 lignes ; cas vivants reportés dans cinq fichiers de test ; `GABARIT_NORMAL` et `EliminatedRow` RESTENT, lus par le bloc « À l'instant » ; S.5 bis faite le même soir : cellule de munitions, boîte de grenades, marques d'inventaire et seconde cellule d'arme retirées de la fiche, `ReplayInventoryRow` 460 -> 189 lignes, trois textes sans lecteur retirés, cliquet des fonctions longues à 247 ; reste noté : cinq valeurs de `GABARIT_NORMAL` sans lecteur, à solder en scindant le type des gabarits) : suppression de ce qui n'a plus d'appelant après S.4 et B.2
  (`ui/ReplayTeams.tsx`, gabarit normal, `model/cardDensity.ts` si plus lu, tests et fixations associés), avec
  report des cas de test sur les lignes ; `ui/ReplayPlayerCard.tsx` reste (rangée, G0-1).
- **B.1** : cotes nommées dans `cardGabarit.ts` (largeur minimale de fiche, écart, largeur du score, marges) ;
  attendus écrits : 4v4 à 1440 px tient, 4v4 à 1280 px ne tient pas (1286 px demandés), 5v5 à 1440 px ne tient
  pas (1528 px) ; colonnes de la grille par `style`.
- **C.1** : `model/feedZone.ts` et `hooks/useFeedZones.ts` remplacent `killerZone.ts` et `useKillerZones.ts`.
- **C.2** : attend E.1 ; éliminations seules ; sans pointeur ; sous les surcouches.
- **D.1** : le bloc « À l'instant » de J et L n'existe pas aujourd'hui : **D.0 (neuf)** [x] fait le 2026-11-17 (pour-cent du plein faute de points dans le document, charges en nombre sans barre, santé en encre du thème comme M) : `ui/ReplayPlayerNow.tsx`
  le construit (titre et instant lu, bouclier et santé chiffrés, armes, grenades, équipement) ; la forme des
  « Totaux » change (L-5), dessinée avec 0.4.
- **D.2** : J remplace TOUTE la colonne (pas la seule cellule du fil) ; `ui/ReplayPlayerPanel.tsx` reste le
  fichier qui écoute Échap ; `ui/ReplayPlayerPicker.tsx`, `ui/ReplayFeedExtract.tsx`.
- **D.3** : deux options à `useAnchoredPanel` (côté d'alignement, élément borne) ; le pion cerclé est un
  peintre du moteur (`layers/replayRenderer.ts`), pas du JSX.
- **E.1** : la barre et les superpositions du bloc sont des FONCTIONS DE RENDU qui reçoivent un objet de
  commandes né dans la toile ; `hooks/useReplayView.ts` et `hooks/useReplayViewport.ts` entrent dans les
  fichiers à un seul écrivain (E.1, puis F.2, puis G.1).
- **E.2** : `ui/ReplayTransportIcons.tsx` en plus.
- **E.3** : barre d'une ligne avec les commandes de G0-4 ; les frags de tous les joueurs entrent dans l'objet
  `timeline` ; le curseur compact porte `data-replay-timeline`.
- **E.4** : attend M.1 ; rend le focus après le choix (les raccourcis restent actifs).
- **F.2** : lecteur d'orientation qui distingue « absent » ; l'Auto se résout dans `useReplayView` ; les deux
  gardes qui lisent `turns: orientation.turns` suivent dans le même commit.
- **Section 2** : `ui/ReplayCanvas.tsx` n'est pas sous baseline (498 lignes comptées par le lint avant
  l'extraction du moteur, fusionnée le 2026-11-17 : 640 lignes brutes) ; « sous la zone de lecture » : rien.
- **0.4** : la maquette M tranche aussi le mot (« arme lourde » ou « armes de puissance », écrit aujourd'hui
  dans la fiche) et la forme des « Totaux ».

- [x] 0.5 (ajouté le 2026-11-17, G0-5 ; FORME VALIDÉE par le user le 2026-11-17 sur la version 2 du tableau N, `maquettes/GrandeEquipeLignes.dc.html` : fiches compactes à deux joueurs par emplacement, voir « G0-5 bis » en section 1 quater) Maquette des lignes resserrées de la Grande équipe : un tableau de plus
      sur le canevas (colonne de 360 px, 24 joueurs, fil de 5 lignes) ; validé par le user avant la partie
      « Grande équipe » de S.2.

Ordre qui en découle : I.1 et I.2, puis M.1, X.1, A.1, C.1, D.0, T.6, T.7, B.1, F.1 ; A.3 après A.1 et la
fusion du backlog (bouton « Générer le rejeu ») ; E.1 après A.3 (même zone de la page) ; le reste selon la
table 1 bis.

## 2. Ce qui existe (relu sur pièces, `origin/release/v8.2.0` au 2026-11-15, `db11e95da1` ; À RELIRE sur `release/v8.3.0` au go, une fois la 8.2.0 fusionnée dans main)

Relu le 2026-11-17 sur `release/v8.3.0` (`22a1a951ad`, qui contient tout `main`) : tous les fichiers cités
existent ; `ReplayPage.tsx` 469 lignes, `ui/ReplayCanvas.tsx` 810, `ui/ReplayTransport.tsx` 469,
`ui/ReplayPlayerCard.tsx` 606, `ui/ReplayScoreBanner.tsx` 286, `i18n/fr.ts` 494 et `i18n/en.ts` 489 (tout ajout
de chaîne passe par une extraction en sous-dictionnaire) ; fixations `replayTeams.4v4.html` et
`replayTeams.6v6.html`. L'audit 0.3 fait la relecture de fond.

Chemins relatifs à `apps/web/src/features/match-replay/`.

- **Page** `ReplayPage.tsx` (472 lignes) : grille `xl:grid-cols-[minmax(0,1fr)_minmax(0,30rem)]`. Colonne
  gauche : `ReplayScoreBanner` puis `ReplayCanvas` (avec `mapOverlays` : victoire, manche, bombe). Colonne
  droite : `ReplayTeams` puis `ReplayFeedOrPanel` (fil `ReplayKillFeed` OU panneau `ReplayPlayerPanel`,
  lot 4 du plan maquettes, PR #200). Le fichier frôle le seuil de 500 lignes : tout ajout passe par extraction.
- **Plein écran actuel** `hooks/useReplayFullscreen.ts` : DEUX étages en un seul geste, superposition
  `REPLAY_FULLSCREEN_FRAME` (`fixed inset-0 z-40 ...`) posée sur la grille, PLUS plein écran natif de
  `document.documentElement`. Couches d'Échap : `REPLAY_ESCAPE_LAYER_ATTR`, garde-rail
  `hooks/escapeLayers.guard.test.ts`. Désactivé pendant un export (`useExportActive`). Touche `f` / `F`
  (`hooks/useReplayShortcuts.ts:131`). Aucun mode Cinéma n'existe.
- **Toile** `ui/ReplayCanvas.tsx` (809 lignes, au-delà du seuil, gelée par la baseline) : bloc `relative p-3 pb-0`
  contenant la toile centrée `mx-auto` (largeur de la carte), `ReplayViewMenu` (absolu, coin haut droit de la
  toile), `ReplayHeatmapLegend` (absolu `bottom-2 left-3` du BLOC), `ReplayZoomControl` (absolu
  `bottom-2 right-3` du bloc) ; puis `ReplayTransport` bord à bord. `mapOverlays` se pose dans le conteneur de
  la TOILE (bornes de la carte), pas du bloc.
- **Barre de lecture** `ui/ReplayTransport.tsx` (469 lignes) : `ReplayTimelineTracks` (frise à pistes, horloge
  sous le point), lecture, ±10 s, `ReplaySpeedMenu`, `ReplaySoundControls`, capture image, export vidéo
  (`ReplayExportDialog`, bouton qui s'élargit pendant l'export), `FullscreenButton`.
- **Taille du terrain** `hooks/useReplayViewport.ts` : hauteur libre quantifiée (8 px) et stabilisée (120 ms) ;
  `CANVAS_HEIGHT_CEILING` (720 px) dans `hooks/useReplayView.ts`, garde-fou mémoire des quatre calques cuits.
- **Orientation** `settings/useReplayOrientation.ts` (quarts de tour retenus PAR CARTE sous
  `replay-orientation.<mapId>`, `settings/replayPreferences.ts`), section `settings/ReplayOrientationSection.tsx`
  (0° / 90° / 180° / 270°) ; types dans `lib/replay/orientation.ts` (`QuarterTurns`). Aucune règle automatique.
- **Fiches** `ui/ReplayPlayerCard.tsx` (606 lignes) + `ui/ReplayTeams.tsx` ; cotes dans `model/cardGabarit.ts` :
  `GABARIT_NORMAL` (235 px) et `GABARIT_COMPACT` (115 × 62, Grande équipe), choisis par `cardDensity.ts` seul ;
  garde-rail `ui/cardGabarit.guard.test.ts` ; fixation `ui/__fixtures__/replayTeams.4v4.html`. Jauges
  `ui/ReplayVitality.tsx` : bouclier 5 px au-dessus de la santé 3 px. Places et tuiles d'attente
  (`ReplaySeatWaiting`).
- **Score** `ui/ReplayScoreBanner.tsx` (286 lignes), lecture pure `readScoreBanner` ; rend `null` en FFA ou sans
  compteur.
- **Fil** `ui/ReplayKillFeed.tsx` (plus récente en tête, défile, menu de filtre `ReplayFeedFilterMenu`, lignes
  d'événement), formes dans `ui/ReplayFeedLines.tsx` et `ui/ReplayKillLineParts.tsx` ; fil aligné construit une
  fois par la page (`buildFeedEntries`) et partagé avec la frise. Filtre retenu : `settings/useFeedFilter.ts`.
- **Lieu** `hooks/useKillerZones.ts` + `model/killerZone.ts` : zone du TUEUR, jumeau TS de `NommerZone`
  (`apps/go-api/internal/analysis/tactical/zones.go`), jeu de cas partagé
  `apps/go-api/internal/testfixtures/testdata/cartes/nommage_zone_cas.json`.
- **Fiche détaillée** `ui/ReplayPlayerPanel.tsx`, `hooks/usePlayerPanel.ts`, `i18n/playerPanel.i18n.ts`.
- **Panneau ancré** `settings/useAnchoredPanel.ts` (portail `fixed` sur `body`, recalé sur son déclencheur).
- **i18n** `i18n/i18n.ts` : `REPLAY_TEXT: Record<ReplayLocale, ReplayText>` (parité par typage), `fr.ts`, `en.ts`.
- **App** : la coquille ne borne pas la largeur ; la page du rejeu occupe déjà tout le corps (`px-6`).

## 3. Lot 0 — préalables

- [x] 0.1 (fait le 2026-11-17 : six fixations, 6 tests verts rejoués par le pilote ; la première livraison ne
      couvrait ni le lieu du fil ni le bouton d'export, complétée sur relecture : `replayKillFeed.lieu`,
      `replayTransport.export`, `replayTransport.exportEnCours`) Témoins « avant » sur `feat/rejeu-refonte-ui` avant tout code : fixations DOM (vitest, rendu jsdom
      avec le document de test `test/testDoc.ts`) de la ZONE DE LECTURE du Standard (`ReplayTransport` avec sa
      frise à pistes, et `ReplayZoomControl`), qui ne doit pas changer (D-8), et des lignes du fil, dont seul le
      lieu doit changer (D-5) ; rangées sous `apps/web/src/features/match-replay/ui/__fixtures__/` avec le suffixe
      `.avant.html`. Gate : les fixations existent et un test les relit à l'identique.
- [ ] 0.2 M-1 : script de mesure (page de démonstration du rejeu, toile à 720 puis 1080 px de haut sur une
      carte haute et une carte large) qui relève le tas JS au pic et la durée des cuissons au redimensionnement.
      L'agent l'écrit ; le user le lance à son écran ; résultats dans `.ai/annexes_refonte_ui_rejeu/mesure_m1.md`.
      Préparé le 2026-11-17 sur la branche de mesure `ui/m1` (`eac9627e47`, worktree `C:/t/ui-m1`, jamais
      fusionnée) : plafond et hauteur forcés par deux clés du stockage local (5 lignes marquées dans
      `hooks/useReplayViewport.ts`), script de console, mode d'emploi en 10 étapes et tableau vide ; ligne
      posée dans l'artefact des vérifications. Limite écrite : le tas JavaScript ne voit pas les pixels des
      toiles ; le gestionnaire de tâches du navigateur est relevé en complément. Reste le lancement par le
      user ; sans mesure, G.1 se clôt `[!]` et le plafond reste à 720 px.
- [x] 0.3 (rendu le 2026-11-17 : `annexes_refonte_ui_rejeu/audit_conception.md`, grille complétée de 50
      annotations ; écarts et questions tranchés le jour même, section 1 quater) Audit de conception (Fable, lecture seule) : relit ce plan contre le code, propose le découpage des
      composants par cadre sans franchir 500 lignes ni grossir `ReplayCanvas.tsx`, liste les risques (Échap,
      export vidéo qui suit la mise en page, couches z, portails). Écarts au plan rapportés au user avant A.1.
      (amendé le 2026-11-17) L'audit complète et corrige la grille de conformité contre le code : il compare la
      fiche compacte actuelle à la tuile de K (K-3, K-4), dit si le document porte l'objectif du match (K-7),
      vérifie A-2 et L-4, propose une forme pour chacun des 10 éléments sans dessin, et dit où extraire les
      dictionnaires `i18n/fr.ts` et `i18n/en.ts`.
- [x] 0.4 (validé par le user le 2026-11-17, forme 2 ; voir section 1 quater) (ajouté le 2026-11-17, plan maître DM-3, réponse du user : « Oui, une maquette ») Maquette de la
      fiche enrichie : un tableau de plus sur le canevas, la fiche J et la fenêtre L avec le contenu du lot T
      (compteurs, répartition des éliminations, estimations, grenades et ramassages, médailles, « En passant »).
      Validé par le user avant T.5 ; ses lignes entrent alors dans la grille.
      Dessiné et publié le 2026-11-17 : tableau M du canevas (`FicheInstant.dc.html`, copie dans
      `annexes_refonte_ui_rejeu/maquettes/`), trois panneaux (colonne du Standard pour un joueur lu en entier,
      la même pour une lecture partielle, fenêtre du Cinéma et du Plein cadre) ; ligne posée dans l'artefact
      des vérifications. Reste la validation du user.
- **Gate 0** : 0.1 et 0.3 rendus ; écarts de 0.3 et éléments sans dessin tranchés par le user. 0.4 ne bloque
  que T.5.

## 4. Lot A — les cadres

- [x] A.1 (fait le 2026-11-17 : `ReplayFrameState{frame, returnTo}`, `reduceReplayFrame`, lecteurs `showsAppNavigation`, `usesCompactBar`, `canPlaceCardRow`) `model/replayFrame.ts`, pur : type `ReplayFrame = 'standard' | 'cinema' | 'plein-cadre'`, transitions
      (bascule Cinéma, bascule Plein cadre, Échap), état initial `standard` à chaque montage (aucun stockage,
      même règle que le plein écran actuel : « une manière de regarder, pas un endroit »). Tests unitaires de
      toutes les transitions.
- [x] A.2 (fait le 2026-11-17 : `hooks/useReplayFrame.ts` porte l'état du cadre et annonce le redimensionnement ; le natif pris depuis le Standard ouvre un Plein cadre qui se referme avec lui ; `data-replay-frame` sur la racine ; l'infobulle du plein écran du système ne cite plus F) `useReplayFullscreen.ts` scindé en deux gestes : Plein cadre = la superposition seule ; plein écran
      système = le natif seul, qui peut s'ajouter à n'importe quel cadre (en Standard il entre d'abord en Plein
      cadre, comme aujourd'hui). `fullscreenchange` reste la source de vérité du natif. Touches : `t` Cinéma,
      `f` Plein cadre, Échap selon les couches (`REPLAY_ESCAPE_LAYER_ATTR`). Export en cours : aucune bascule,
      comme aujourd'hui. Tests des deux gestes et des touches ; le garde-rail des couches d'Échap reste vert.
- [x] A.3 (fait le 2026-11-17 : `ui/frames/` — gabarit à cinq zones (une zone `avis` ajoutée pour le bandeau de génération, le chargement, l'erreur et l'absence), montage unique de la toile, en-têtes A, K, D ; test de non-remontage de la toile vérifié mordant ; le groupe de cadres est dans l'en-tête comme A, K et D le dessinent, E.2 le déplacera selon V1 ; la colonne existe en deux copies, Standard et Cinéma, jusqu'aux lots S et B ; `ReplayPage` 473 -> 231 lignes, sa fonction reste au-dessus de 80 lignes ; centrage du Plein cadre à regarder à l'écran) Page découpée : `ReplayPage.tsx` ne garde que chargement, modèle et choix du cadre ; la disposition de
      chaque cadre va dans `ui/frames/ReplayStandardFrame.tsx`, `ui/frames/ReplayCinemaFrame.tsx` (Cinéma et
      Plein cadre, qui partagent la structure D-2). Standard : disposition d'aujourd'hui déplacée telle quelle
      dans son fichier (le lot S la refait), zone de lecture identique aux témoins 0.1. Cinéma : en-tête de
      page réduit au retour, au titre et au lien vers la fiche du match (tableau K). Plein cadre : sans navigation de l'app, une barre : Quitter / Échap,
      titre du match. Fichiers sous 500 lignes. (amendé le 2026-11-17, grille D-1) Les touches T et F sont
      rappelées dans les infobulles des icônes de cadre (E.2).
- **Gate A** : `npm --prefix apps/web run typecheck`, `npm --prefix apps/web run test:run -- src/features/match-replay`,
  `npm --prefix apps/web run lint`, `npm --prefix apps/web run lint:colors` verts ; fixations Standard égales
  aux témoins, hors différences listées dans le compte rendu.

## 5. Lot B — fiches en rangée (Cinéma, Plein cadre)

- [x] B.1 (fait le 2026-11-17 : `rowFits`, `largeurRequiseRangee`, cotes `COTES_RANGEE` ; un écart compté par fiche, soit 6 px de marge par camp) `model/rowLayout.ts`, pur : `rowFits({ camps, placesParCamp, largeurCorps, gabarit, largeurScore })`
      selon D-12 ; la largeur vient de `GABARIT_COMPACT` et des écarts de la rangée, aucun littéral de largeur
      de fiche. Tests : 4v4 à 1440 px (tient), 5v5 à 1440 px, 4v4 à 1280 px, Grande équipe, FFA.
- [x] B.2 (fait le 2026-11-17 : `ui/ReplayTeamsRow.tsx`, `hooks/useCardRowLayout.ts` (largeur mesurée du corps), `dispositionEquipes` dans `model/rowLayout.ts` : rangée ordinaire jusqu'à 5 par camp (4 contre 4 = 1286 px), rangée Grande équipe de 6 à 12 par camp à partir de 1350 px (fiche minimale de 80 px, choix de l'agent), repli sur la colonne en lignes sinon, score alors sur la toile ; camp allié à gauche ; état « suivi » ajouté à la fiche ; au Plein cadre la rangée va bord à bord, sans les 24 px de K : à regarder à l'écran) `ui/ReplayTeamsRow.tsx` : `[fiches du camp de gauche] [bandeau de score] [fiches du camp de droite]` ;
      fiches = `ReplayPlayerCard` au gabarit compact (bouclier et santé en deux barres, grenades, équipement,
      tuiles d'attente comprises) ; `ReplayScoreBanner` au centre, à la largeur de sa cellule. Clic sur une fiche :
      `playerPanel.onCard` (ouvre D.3). Repli : quand `rowFits` est faux, le cadre rend la colonne du Standard
      (`ReplayRosterColumn`, lot S).
      Les cotes nouvelles éventuelles vont dans `cardGabarit.ts` (garde-rail existant). Tests : rendu 4v4, repli,
      tuile du joueur suivi, joueur mort.
      (amendé le 2026-11-17, grille K-3, K-4, K-7) La rangée remplit la largeur du corps : les tuiles se
      partagent la place à parts égales, 115 px n'est que le minimum de la règle B.1. Contenu et ordre de la
      tuile = ceux de K (pastille du camp, nom, F / M / A ; puis bouclier et santé, grenades avec leur nombre,
      équipement ; pas d'arme) ; l'écart avec la fiche compacte actuelle est relevé par 0.3 et tranché à
      Gate 0. Sous le score, la ligne d'objectif du match (« Premier à 50 éliminations »), si le document la
      porte (0.3) ; sinon retour au user avant B.2. Tests en plus : largeur partagée, ligne d'objectif.
- **Gate B** : commandes du Gate A ; `cardGabarit.guard.test.ts` vert.

## 5 bis. Lot S — le Standard selon le tableau A

- [x] S.1 (fait le 2026-11-17 : page bornée à 1200 px, en-tête de 64 px en trois cellules, score de 440 px au centre par `COTES_BANDEAU_SCORE` (emplacements `entete`, `rangee`, `scene`) ; le score reste sur la toile en Cinéma et en Plein cadre ; `ReplayMatchRecall`, sans appelant, supprimé avec sa dérogation d'import ; reste le liseré de bord des barres de score, hérité de la maquette A, à montrer au user) `ReplayStandardFrame.tsx` : contenu borné et centré (constante nommée de largeur, D-11) ; en-tête en
      trois cellules : retour et titre du match à gauche, `ReplayScoreBanner` resserré au centre (environ 440 px,
      manches et barre de prise de zone comprises, sans changer sa lecture `readScoreBanner`), lien vers la fiche
      du match à droite. Le bandeau de score ne coiffe plus le terrain en Standard. Tests : en-tête rendu, score
      présent, score absent en FFA (rien n'est rendu, comme aujourd'hui).
- [x] S.2 (fait le 2026-11-17 : lignes `ReplayRosterRow` / `ReplayRosterColumn` / `ReplayRosterRowDetail`, gabarit « ligne » ; Grande équipe selon G0-5 bis : `ReplayCompactCard` en formes empilée et ligne, seuil `grandeEquipe` dans `model/rowLayout.ts` ; écarts acceptés par le pilote et dits au user : colonnes d'arme, de grenades et d'équipement vides pour un mort, charge en nombre sans barre, « suivi » en couleur d'accent du thème ; non monté avant S.4) `ui/ReplayRosterRow.tsx` et `ui/ReplayRosterColumn.tsx` : une ligne par place (nom ; bouclier et santé
      en deux barres via `VitalityBar` ; « Réapparition dans … » à la place des barres quand le joueur est mort ;
      arme en main ; grenades ; équipement ; F / M / A), en-tête de colonnes Arme / Gr. / Éq. / F / M / A ; places
      libres et occupants pas encore apparus rendus comme lignes d'attente (même règle que `ReplaySeatWaiting`) ;
      la ligne du joueur suivi se déplie (armes et munitions, grenades, charge de l'équipement ; dessin = bloc
      `.det` du tableau H, grille A-10). Lectures reprises
      de `model/playerCardReadings.ts`, aucune recalculée ; cotes dans `cardGabarit.ts` (troisième gabarit
      « ligne » si 0.3 le recommande). Clic sur une ligne : `playerPanel.onCard`. Tests : 4v4, joueur mort, place
      libre, joueur suivi déplié, Grande équipe (la part des équipes défile).
- [x] S.3 (fait le 2026-11-17 : ligne de 28 px, colonnes 30 / 70 / 70 / 18 / 70 du tableau N, ordre instant, lieu, tueur, arme, victime, médailles ; `hooks/useFeedView.ts` ; assistance et porteur en infobulle ; fixations « avant » remplacées par des « après » après preuve de contenu identique ; ÉCARTS de l'agent, acceptés et dits au user : la ligne choisie se lit par un fond, ses liserés de camp sont retirés ; plus de fond bleuté sur une élimination assistée ; horloge sans chasse fixe ; à trancher par le user : le liseré gauche du titre du fil (`hudBand`), la teinte de fond du bandeau d'objectif ; C.2 devra partager le filtre entre deux affichages du fil) Fil de la colonne : `ReplayKillFeed` rendu au format d'une ligne aux colonnes fixes (instant | tueur |
      arme | victime | lieu de la victime, C.1), sans fond ni bordure, plus récente en tête, défilant, titre = menu
      de filtre actuel. Les formes non-élimination du fil (objectifs, entrées et sorties, prises) gardent leur
      forme actuelle sur une ligne. Fixations des lignes comparées aux témoins 0.1 : seul le lieu change.
- [x] S.4 (fait le 2026-11-17 : colonne de 360 px par `COTES_COLONNE_STANDARD`, équipes en lignes puis fil ; plancher du fil de 5 lignes (171 px, réserve de titre estimée et non mesurée) ; en Grande équipe une place sans rendu garde sa cellule ; le bloc arrondi commun carte et colonne de A-5 n'est PAS fait, la toile gardant son propre bloc (V5) : à juger à l'écran) Montage : la grille du Standard devient carte | colonne de 360 px (équipes puis fil, le fil garde au
      moins 5 lignes) ; la zone de lecture et le zoom restent ceux d'aujourd'hui ; les sections sous le lecteur
      (carte de chaleur, usage de l'équipement) inchangées. `ReplayTeams` et la colonne de 30 rem sortent du
      Standard ; ce qui n'a plus d'appelant est supprimé avec ses tests (règle 0 code mort), la fixation
      `replayTeams.4v4.html` suit le composant qui reste (rangée du lot B).
- **Gate S** : commandes du Gate A ; zone de lecture égale aux témoins 0.1 ; aucun fichier au-delà de 500 lignes.

## 6. Lot C — le fil sur le plan

- [x] C.1 (fait le 2026-11-17 : `model/feedZone.ts` (`victimZones`), `hooks/useFeedZones.ts`, libellé `killFeedVictimZone` ; fixation du fil avec lieu : seule l'infobulle change) Lieu de la victime : `model/victimZone.ts` (ou extension de `killerZone.ts` si 0.3 le recommande),
      même règle de nommage (a)-(d) appliquée à la position de la VICTIME à l'instant du frag (dernier point de
      sa vie, même résolveur de porteur, même règle de `z`). Le nommage du point ne change pas : le jumeau Go et
      `nommage_zone_cas.json` restent tels quels. `useKillerZones` devient `useFeedZones` (lieu de la victime),
      l'ancien hook est supprimé avec ses imports (règle 0 code mort). Tests : victime à pied, victime en
      véhicule, victime sans position, bot.
- [x] C.2 (fait le 2026-11-17 : `ui/ReplayFeedOverlay.tsx`, `model/planFeed.ts` (éliminations seules, sans filtre), cotes `model/cotesFilPlan.ts` ; posé quand la rangée de fiches l'est, jamais en repli ni au Standard ; la colonne est alors cachée, le fil complet restant monté ; `KillLine` partagée avec la colonne (prop `onPlan`) ; ombre par le jeton `--scrim`, classes vérifiées par un build ; colonnes 30 / 70 / 70 / 18 / 70 de la ligne du fil, et non les 30 / 78 / 88 / 20 du tableau N) `ui/ReplayFeedOverlay.tsx` : les 5 dernières lignes à l'instant lu du fil FILTRÉ (même filtre que le
      Standard, `useFeedFilter`), plus récente en bas, sans en-tête ni menu, sans fond, ombre portée du texte
      (token, pas de couleur en dur), colonnes fixes partagées avec `ReplayKillLineParts`. Ancré en bas à gauche
      du BLOC de la toile (slot E.1), donc dans la marge d'une carte haute et sur le plan d'une carte large. La
      légende de la carte de chaleur passe en haut à gauche du bloc dans ces cadres. Tests : 0, 3, 5 et 8 lignes
      passées ; ordre ; filtre appliqué. (amendé le 2026-11-17, grille K-9) Les deux lignes les plus anciennes
      s'estompent (70 %, puis 85 %), par des cotes nommées ; test : estompage à 5 lignes, aucun à 3.
- [x] C.3 (fait le 2026-11-17 : clé `replay-show-plan-feed`, vraie par défaut, section du panneau Vue montrée seulement là où le fil du plan peut se poser ; option éteinte, le fil des éliminations n'est affiché nulle part dans ces cadres : choix du user) Option « Fil des éliminations » (affiché par défaut) dans le panneau Vue, retenue dans
      `replayPreferences.ts` (clé nouvelle, garde-rail `replayPreferences.guard.test.ts`). Sans effet en
      Standard (la colonne y garde son fil). Libellés fr et en.
- **Gate C** : commandes du Gate A ; `killerZone.test.ts` / tests du nouveau module verts ; aucun fichier Go
  touché par les commits du lot C (amendé le 2026-11-17 : la branche porte par ailleurs le Go de T.6).

## 7. Lot D — la fiche détaillée (J en Standard, L en Cinéma et Plein cadre)

- [x] D.1 (fait le 2026-11-17 : quatre fixations du panneau posées avant, identiques à l'octet après ; la tête est dans le corps commun, le retour et Échap au conteneur ; seule la partie basse défile, alors que J fait défiler toute la fiche : à trancher en D.2) `ui/ReplayPlayerPanelBody.tsx` : contenu commun extrait de `ReplayPlayerPanel` (Suivre / Seul /
      Masquer, « À l'instant », « Totaux du match »), sans changer son rendu (fixation avant / après). Tests de
      comportement existants verts sans modification.
- [x] D.2 (fait le 2026-11-17 : `ui/ReplayPlayerPicker.tsx`, `ui/ReplayFeedExtract.tsx`, cotes `model/cotesFicheStandard.ts` ; la fiche couvre la colonne, équipes et fil restant montés et masqués ; `ReplayPlayerPanelBody` prend `scroll` interne ou confié au conteneur ; le retour « Équipes » défile avec le contenu, comme J-1 ; l'identité J-4 (nom en 17 px, camp) n'est PAS refaite : le corps commun garde sa tête, à trancher avec T.5) Fiche du Standard (tableau J, D-6) : clic sur une ligne de la colonne, la colonne devient la fiche via
      `ReplayFeedOrPanel` (le fil reste monté dessous, mécanisme conservé). En-tête « Fiche » avec retour
      « Équipes » (couche `REPLAY_ESCAPE_LAYER_ATTR` : Échap rend la colonne avant de quitter un cadre) ;
      sélecteur des joueurs en rectangles arrondis ; `ReplayPlayerPanelBody` ; section « Éliminations » = les
      2 dernières lignes du fil du match (tous joueurs, filtre courant, le joueur ouvert mis en avant) au format
      d'une ligne de S.3, lien « Fil complet » qui rend la colonne ; frise à pistes inchangée
      (D-18). Tests : ouverture depuis une ligne, changement par le sélecteur, retour Équipes, Échap, 2 dernières
      lignes du fil, frise égale aux témoins 0.1 fiche ouverte.
- [x] D.3 (fait le 2026-11-17 : `ui/ReplayPlayerPopover.tsx`, options `align` et `boundRef` de `useAnchoredPanel`, fermeture par `useDismissable` sans clic dehors ; la colonne ne se montre plus quand la rangée est posée ; anneau du pion par `layers/openPlayerRing.ts`, encres `--foreground` et `--background`, peint aussi à l'export ; reprise du score : en repli il est dans l'en-tête (H-2), sur la toile seulement sous 820 px de corps ; ÉCARTS avec L, acceptés et dits au user : fenêtre 8 px sous la fiche, tête et actions fixes avec défilement du bas seulement, ombre et fond par jetons ; D.3 bis faite le même soir : fiche « de cellule », « Retour au fil » et la branche correspondante de `ReplayFeedOrPanel` supprimés ; 21 cas de test reportés sur `ReplayPlayerPanelBody.test.tsx` et la fiche en colonne ; quatre fixations du CORPS seul, `replayPlayerPanelBody.*.html`, égales à l'octet au corps des anciennes : témoins « avant » de T.5 ; montage de page des tests partagé dans `test/replayPageHarness.tsx`) `ui/ReplayPlayerPopover.tsx` (tableau L, Cinéma et Plein cadre) : `ReplayPlayerPanelBody` dans un
      panneau ancré sous la fiche cliquée (`useAnchoredPanel`), portail, `REPLAY_ESCAPE_LAYER_ATTR`, croix,
      défilement interne. Lecture non interrompue, aucun voile. Clic sur une autre fiche : change de joueur. La
      fiche et le pion du joueur ouvert sont cerclés (token de focus). Tests : ouverture depuis la rangée et
      depuis la colonne de repli, changement de joueur, Échap ferme la fenêtre puis le cadre, Suivre / Seul /
      Masquer.
- **Gate D** : commandes du Gate A ; `escapeLayers.guard.test.ts` et `playerFocusWiring.guard.test.ts` verts.

## 7 bis. Lot T — la fiche à l'instant (D-19 à D-23)

Toutes les lectures sont pures, à l'image lue, pour le joueur de la fiche, bots compris (un bot a des vies et des
éliminations). Aucune requête nouvelle, aucun changement du décodeur ; (amendé le 2026-11-17) seule exception
côté serveur : T.6, qui sert les médailles du document avec leur libellé et leur image. Les formules
existantes se réutilisent (KDA du jeu, ADR 0006 ; catégorie « arme lourde » des niveaux d'armes du document) : 0.3
les localise, rien n'est recalculé à côté.

- [ ] T.0 (mesures ajoutées le 2026-11-17 : part des frags classés par match avec rejeu — T.2b ; ramassages natifs, borne inférieure ou non — T.4 ; fixation « fin de match = tableau de score » de T.1 sur un match réel) (amendé le 2026-11-17, item E.4 du plan backlog) Les mesures se font APRÈS la régénération du parc au
      schéma 125 (geste du user, dès la fusion du backlog : plan maître DM-1) et se rendent PAR BUILD du jeu.
      Mesures sur le parc, en lisant les documents de rejeu DÉJÀ générés (aucune génération) : part des matchs
      dont le score par action est déclaré complet ; part des joueurs dont la lecture des tirs est jugée
      suffisante ; sur trois films, boucliers tombés à zéro contre dégâts subis (plausibilité de D-21, recharges
      comprises). Résultats dans `.ai/annexes_refonte_ui_rejeu/mesure_t0.md`. Une part trop faible pour valoir
      l'affichage est rapportée au user avant T.3.
- [ ] T.1 (CODE FAIT le 2026-11-17, `readPlayerInstantSeries` puis `playerInstantStatsAt` ; RESTE, case ouverte : la fixation « fin de match = tableau de score » sur un match RÉEL, aucun document du dépôt ne portant à la fois le calque de score et la feuille ; à faire avec T.0 sur un rejeu régénéré — dépendance : fin de la régénération du parc) `model/playerInstantStats.ts` : à l'image lue, FDA, meilleure série et série en cours, tirs à la tête
      (une élimination dont le coup fatal n'est pas mesuré ne compte ni oui ni non), parfaits (médaille datée),
      réapparitions, durée de la vie en cours (absente quand le joueur est mort), moyenne des vies terminées
      jusqu'à l'instant, boucliers perdus (D-21 : passages à zéro du bouclier du joueur, une recharge puis une
      nouvelle chute comptent deux), médailles gagnées. Tests : début de match, joueur mort, bot, tir à la tête
      non mesuré, deux boucliers perdus dans une vie, fin de match égale au tableau de score sur les valeurs
      exactes (fixation d'un match réel).
- [x] T.2 (fait le 2026-11-17 : `killBreakdownAt`, table arme -> classe reçue en paramètre, entrée « non classée » tenue ; la source de la table est l'item T.2b) `model/killBreakdown.ts` (D-20) : éliminations jusqu'à l'instant par catégorie (arme lourde, mêlée,
      grenade, autres armes) puis par arme, nombre et part en pourcentage ; une élimination sans arme résolue
      compte dans « autres » sous un libellé neutre, jamais sous l'arme d'une voisine. Tests : parts qui somment à
      100, catégorie vide absente, arme inconnue.
- [ ] T.3 Score, rang et estimations (D-22) : score à l'instant = somme des points par action quand le document
      déclare le score complet ; rang dans le lobby seulement si tous les joueurs du lobby sont complets ;
      précision, tirs, touches, dégâts infligés et subis, et par arme tirs et dégâts, pour un joueur à lecture
      suffisante uniquement. Tests : score incomplet (pas de ligne), joueur insuffisant (pas de ligne), joueur
      suffisant (valeurs), rang à égalité. (amendé le 2026-11-17, item E.4 du plan backlog) Garde en plus du
      verdict du document : la somme des points par action ne s'affiche que si elle égale le score personnel de
      l'API (le verdict « complet » est faux en Assaut et en Grande bataille du build `HI_1_11_0`). Test : somme
      différente du score de l'API, pas de ligne.
- [x] T.4 (fait le 2026-11-17 : `playerThrowsPickupsAt` ; lancers par `doc.grenades`, ramassages par `doc.pickups` — une BORNE INFÉRIEURE quand `coverage.pickups.multiEvent`, à dire ou à mesurer en T.0 avant T.5 ; les ramassages de grenades ne se relient à aucun libellé, `grenadeLabels[].family` étant vide) Grenades lancées par type et ramassages (armes, grenades, équipement) jusqu'à l'instant (D-23). Tests :
      aucun lancer (section absente), types mêlés.
- [ ] T.5 Rendu dans `ReplayPlayerPanelBody` (D.1), donc dans J et dans L : « À l'instant » garde son bloc actuel
      (bouclier, santé, armes, grenades, équipement) puis ajoute les compteurs de T.1, la répartition de T.2, les
      lignes de T.3 (mention « estimation » sur les valeurs estimées) et les sections de T.4 ; « Totaux du match »
      inchangé ; l'ordre des sections du corps de fiche met « En passant » en dernier (D-24) ; (amendé le
      2026-11-17) elle rend le temps cumulé de sprint et le nombre de sauts de T.7, rien d'autre ; les médailles
      viennent de T.6 ; la mise en page suit la maquette 0.4. Libellés fr et en, sans anglicisme (« série », « tirs à la tête », « répartition des
      éliminations ») ; tokens de couleur uniquement. Tests de rendu : joueur complet, joueur insuffisant, bot,
      défilement de la fiche.
- [x] T.6 (fait le 2026-11-17 : `medalLabels` posé à la lecture par `service/replay_medal_labels.go`, lecture web unique `model/matchMedals.ts` ; contrat et types régénérés ; reste une découverte au backlog : trois champs de médaille du contexte sans lecteur) (ajouté le 2026-11-17, item E.2 du plan backlog, décision D-2 : seule voie qui règle aussi la prod)
      Médailles du match lues dans `medals` du document de rejeu, pour le fil et pour la fiche : UNE lecture.
      Serveur : libellé et image résolus dans la langue de la requête, sur le patron de
      `service/replay_vehicle_labels.go` ; champ servi déclaré dans `internal/domain/replaydoc` ; textes par
      langue ; aucun SQL dans `service/` ni `api/`. Web : une lecture unique « médailles du match » ; le fil
      SUBSTITUE cette source à l'ancienne, sans fusion avec les événements du match ; la clé de requête porte
      la langue (`lib/query/keys.ts`). `openapi.yaml` et `generated.ts` par le pilote. Tests : Go (libellé
      résolu en fr et en, médaille inconnue journalisée et servie sans libellé inventé), web (fil alimenté par
      le document, match sans `medals` : aucune médaille, aucun repli sur l'ancienne source).
- [x] T.7 (fait le 2026-11-17 : témoin `11de8353` relu sur le rejeu régénéré, verdict cohérent — 1 527 sprints tous dans une vie, aucun pendant la mort, vitesse médiane 2,86 en sprint contre 2,02 hors sprint ; 1 159 sauts dérivés dont 91 % à la hauteur attendue ; deux réserves bornées à 0,4 % : un sprint resté ouvert à l'arrêt, cinq sauts aberrants ; `playerMovementStatsAt`, sauts TOUJOURS estimés, le document n'en publiant qu'une dérivation) (ajouté le 2026-11-17, item E.3 du plan backlog, décision D-3) `model/playerMovementStats.ts`, pur :
      temps cumulé de sprint et nombre de sauts jusqu'à l'instant, pour le joueur de la fiche. Précédé de la
      relecture du témoin `11de8353` (les sprints et les sauts du document y sont comparés à ce que montre le
      rejeu ; un écart est rapporté au user avant le code). Le nombre de sauts porte la mention « estimation ».
      Tests : aucun sprint (ligne absente), sprint en cours à l'instant lu, joueur mort.
- **Gate T** : commandes du Gate A ; fixation de fin de match de T.1 égale au tableau de score ; pour T.6, gate
  Go : `cd apps/go-api && GOWORK=off go build ./... && go vet ./... && go test ./internal/service/...
  ./internal/domain/replaydoc/... ./internal/api/handlers/... ./internal/archlint/...`, puis
  `make generate-types` et `make check-types`.

## 8. Lot E — la barre de lecture compacte

- [x] E.1 (fait le 2026-11-17 : `ui/replayTransportCommands.ts` (objet de commandes typé), emplacements `transport` et `blockOverlays` en fonctions de rendu, `ui/frames/replayFrameSlots.tsx` ; le zoom flottant vit dans `blockOverlays` ; `ReplayCanvas.tsx` 644 -> 642 lignes ; fixations et témoin identiques) Slots de `ReplayCanvas.tsx` : `transport` (élément fourni par le cadre : la barre actuelle en Standard,
      la compacte ailleurs) et `blockOverlays` (posés dans le bloc, à côté de la légende et du zoom). Le fichier
      ne grossit pas : la différence nette de lignes est nulle ou négative (le choix de barre et de zoom sort du
      fichier). Zone de lecture du Standard identique aux témoins.
- [x] E.2 (fait le 2026-11-17 : `ReplayCaptureButtons`, `ReplayTransportIcons`, `ReplayFrameButtons` ; le segment de texte des cadres quitte les en-têtes ; `ReplayTransport.tsx` 445 -> 137 lignes, cliquet des fonctions longues à 248 ; icônes de cadre SANS maquette, formes de l'agent, à montrer au user ; deux reprises confiées à E.3 : sortie du Plein cadre par l'icône alignée sur « Quitter », cadre ramené au Standard sans toile et au changement de match) Extraire de `ReplayTransport.tsx` ce que la barre compacte partage : boutons de capture et d'export
      (`ui/ReplayCaptureButtons.tsx`, avec la règle de l'élargissement pendant l'export), boutons de cadre
      (`ui/ReplayFrameButtons.tsx`, D-10, remplace `FullscreenButton` ; infobulles avec les touches T et F,
      grille D-1). Zone de lecture du Standard identique aux témoins hors les icônes de cadre.
- [x] E.3 (fait le 2026-11-17 : `ui/ReplayTransportCompact.tsx`, `ui/ReplayTimelineCompact.tsx`, cotes `model/cotesBarreCompacte.ts` ; `timeline.fragMarks` (frags de tous les joueurs) construit par `useReplayTimeline` ; emplacement `camera` pour E.4 ; `usesCompactBar` décide de la barre ; deux reprises faites : `leaveFullFrame` chemin unique de sortie, `useReplayFrame(scopeKey)` ramène au Standard sans toile et au changement de match ; ÉCARTS avec K à montrer au user : boutons ronds de la barre complète réutilisés, curseur de volume gardé, capture et export dans leur cartouche, pas de croix de déplacement dans la barre d'une ligne) `ui/ReplayTransportCompact.tsx` : une ligne. À gauche caméra (icône seule, E.4) et son ; puis lecture,
      −10, +10, temps, frise compacte (repères d'éliminations et de morts du joueur regardé et de ses alliés,
      bande de dominance, lues dans le même objet `timeline` que `ReplayTimelineTracks`), vitesse, zoom − / +
      (la commande de zoom flottante n'est pas rendue dans ces cadres) ; à droite capture, export, boutons de
      cadre. Raccourcis inchangés. Tests : commandes présentes, frise compacte alimentée par le même fil que la
      frise à pistes, export en cours neutralise la lecture.
- [x] E.4 (fait le 2026-11-17 : `ui/ReplayCameraMenu.tsx`, deux listes sur la même source d'options que la frise (`rosterKey` ajouté à `ViewpointOption`) ; `ReplaySlotCommands` porte `cameraFollow` ; la toile suit `followXuid` fiche ouverte ou fermée par `useReplayPlayerFocus({followPlayer})`, garde `playerFocusWiring` suivi ; fond d'accent du bouton quand un joueur est suivi, non dessiné : à montrer au user) Menu caméra (D-13) : « Suivre » (liste des joueurs, même action que Suivre de la fiche, plus « Aucun ») et
      « Point de vue » (`useReplayViewpoint.select`). Tests des deux sections.
- **Gate E** : commandes du Gate A ; fixations Standard égales aux témoins hors icônes de cadre.

## 9. Lot F — plan tourné et superpositions

- [x] F.1 (fait le 2026-11-17 : `autoTurns(carte, boite, dejaTourne)`, seuils 25 % pour tourner et 20 % pour revenir ; testé sur une boîte de 1390 x 700) `autoTurns(carte, boîte)` dans `lib/replay/orientation.ts`, pur : 0 ou 1 quart de tour selon le gain
      de surface (seuil 25 %, retour sous 20 %, D-14). Tests sur Detachment (1526 × 1666), Ronin (659 × 1288),
      Nadair (2133 × 947) dans les boîtes du Standard et du Cinéma à 1440 × 900.
- [x] F.2 (fait le 2026-11-17 : `readOrientation` rend un quart de tour ou `null` (Auto), stockage inchangé, valeur retenue gardée ; `hooks/useResolvedTurns.ts` seul appelant d'`autoTurns`, résolu dans `useReplayView` sur la boîte de la toile, jamais sur le cadre d'un export ; `autoRotation` dans l'objet de commandes ; `ui/ReplayRotatedButton.tsx` dans la barre compacte ; « Auto » dans le panneau Vue ; ÉCARTS : l'infobulle n'écrit pas le gain en pour-cent de la maquette H, la règle pure ne le rendant pas ; l'onglet Tactique et « Occupation du terrain » lisent le choix RETENU, une carte en Auto y reste droite) Orientation « Auto » par défaut (D-14) : `useReplayOrientation` distingue « aucun choix » (Auto) d'un
      choix retenu ; section Vue avec « Auto » ; bouton « Plan tourné » dans la barre COMPACTE quand l'Auto a
      tourné, qui retient 0° (en Standard, la barre ne change pas, D-8 : le retour passe par la section Vue). Une carte déjà retenue garde son choix. Tests : Auto, choix retenu, bouton, hystérésis.
- [x] F.3 (fait le 2026-11-17 : `ui/ReplayDimensionSwitch.tsx` à gauche du bouton « Vue », `settings/useReplayDimension.ts`, capacité lue par `replay3dCapabilityOf` sous le nom SUPPOSÉ `can_view_replay_3d`, à confirmer par le plan 3D ; rien n'est rendu aujourd'hui) Emplacement 2D / 3D en superposition à gauche du bouton « Vue », dans le coin haut droit de la toile :
      segment 2D / 3D rendu seulement si la capacité de la 3D est vraie (capacité absente aujourd'hui : rien ne
      s'affiche ; pas de drapeau, l'emplacement n'existe à l'écran que par la capacité). Le plan 3D (C.5) le
      branche. Test : capacité absente, rien ; capacité simulée vraie, segment rendu.
- **Gate F** : commandes du Gate A.

## 10. Lot G — plafond de hauteur

- [ ] G.1 Selon M-1 et D-15 : plafond par cadre (`CANVAS_HEIGHT_CEILING` reste la valeur du Standard ; une valeur
      nommée pour Cinéma et Plein cadre), ou `[!]` avec la mesure si le coût est refusé. Tests de
      `useReplayViewport` mis à jour.
- **Gate G** : commandes du Gate A.

## 11. Revue et clôture

- [ ] R.1 Revue finale (Fable) : skill `adversarial-review` sur le diff complet `release/v8.3.0...feat/rejeu-refonte-ui`.
      (amendé le 2026-11-17) La conformité à la grille des maquettes est un axe de la revue : chaque ligne de
      la grille a sa preuve, aucun écart hors de la liste des écarts voulus.
- [ ] R.2 Corrections retenues, une seule ronde.
- [ ] R.3 `make gate-push` vert ; entrée `.ai/thought_log.md` ; amendements C.5 et D-10 portés dans
      `.ai/PLAN_REJEU_3D_2026-11-15.md` ; `[!]` reportés dans `.ai/REGISTRE_REPORTS.md` ; mémoire du chantier.
- [ ] R.4 Validation à l'écran par le user (artefact de vérifications), témoins nommés par lui : Standard
      selon A (zone de lecture inchangée), Cinéma 4v4 sur Detachment, Cinéma sur Nadair (fil sur le plan), Plein cadre, fiche du Standard (J), fiche en fenêtre (L),
      Grande équipe en repli, carte tournée. La fusion dans `release/v8.3.0` attend cette validation.
      (amendé le 2026-11-17) Support : une page de comparaison, un témoin par tableau, la maquette d'un côté et
      la capture de l'app de l'autre, chaque témoin renvoyant aux lignes de la grille.

Chaînes UI : toute chaîne nouvelle existe en fr et en (`i18n/fr.ts`, `i18n/en.ts`, typage
`Record<ReplayLocale, ReplayText>`), sans anglicisme côté fr. Couleurs : tokens uniquement (skill
`color-tokens`). Aucune requête nouvelle, aucune route, aucun code Go, sauf T.6 (amendé le 2026-11-17) : hors
T.6, `lib/query/keys.ts`, `openapi.yaml` et `generated.ts` ne bougent pas.

## 12. Découvertes

Ce qui est vu en route et hors périmètre se note ici, daté, sans être traité.

## 13. Reprise de session

1. Lire ce plan (statuts des items) et la fin de `.ai/thought_log.md`.
2. `git -C <worktree> log --oneline -10` sur `feat/rejeu-refonte-ui` et chaque `LevelUp-wt-ui-<piste>` ;
   `ListAgents` pour les agents encore actifs.
3. Reprendre le premier item non clos de chaque piste dont la colonne « Attend » est satisfaite.

## 13. Passation (2026-11-15)

- La refonte sort de la 8.2.0 et part dans la 8.3.0 (décision du user du 2026-11-15). L'exécution attend son go,
  dans une nouvelle session.
- Avant le go : relire la section 2 sur
  `release/v8.3.0` (la 8.2.0, qui porte le code inventorié, doit y être fusionnée) ; vérifier l'état du plan 3D
  (point de contact sur `ReplayCanvas.tsx`, section 1 ter).
- Maquettes : canevas https://claude.ai/artifact/GkqDthVG6jGxx3A5WLfXrr (version 26). Copie des tableaux dans
  `.ai/annexes_refonte_ui_rejeu/maquettes/` (les fonds de carte y sont des liens du canevas, ils ne s'affichent
  pas hors de lui). Font foi : A, J, K, L, D, H (titres de `canvas.json`).
- Reprise : lire ce plan en entier, puis le skill `plan-execution` ; commencer par le lot 0.
