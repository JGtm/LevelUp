# PLAN — Refonte de la page du rejeu : trois cadres, fiches en rangée, fil sur le plan (2026-11-15)

Statut : **rédigé le 2026-11-15 sur go du user ; maquettes validées le 2026-11-15 ; D-1 à D-16 tranchés par
le user le 2026-11-15 (D-11 corrigé par lui : le Standard reprend la maquette A, sauf la zone de lecture) ;
aucun code écrit ; D-17 tranché (effets gardés) ; D-18 (frise inchangée) validé ; D-19 à D-24 (fiche à l'instant, lot T)
tranchés le 2026-11-16 ; puis le go d'exécution.**

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
  XUID, XP d'équipement. Reportés au backlog (chantier décodeur) : commandes de déplacement, sprints,
  rechargements, visées et sauts en notes de bas de page ; répliques des Spartans ; idées tirées des boucliers
  perdus (badge « os de verre », frags sans bouclier).
- « Totaux du match » ne change pas : chiffres officiels de fin de match.
- **D-24 — notes de bas de page anticipées** (message du user du 2026-11-16 : « on anticipe l'UI mais le travail
  côté décodeur va aller dans le backlog »). La fiche réserve, sous « Totaux du match », une section « En
  passant » : sprints, rechargements, visées, sauts, part du temps en mouvement, à l'instant. T.5 en fixe la
  place et l'ordre dans la mise en page du corps de fiche, sans aucun code de lecture : le document de rejeu ne
  porte pas ces données et son contrat refuse toute clé inconnue, donc une lecture écrite maintenant serait du
  code mort (règle 7). La lecture et le rendu arrivent avec le chantier décodeur du backlog (« Commandes et
  gestes du joueur »).
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
| P0 | 0.3 audit de conception | **Fable** | go | lecture seule ; découpage des composants, risques, écarts à ce plan |
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
| P10 À l'instant | T.0 mesures sur le parc | Sonnet | 0.3 | lecture des documents de rejeu déjà générés, AUCUNE génération |
| P10 | T.1 compteurs exacts à l'instant (pur) | Sonnet | 0.3 | `model/playerInstantStats.ts` |
| P10 | T.2 répartition des éliminations (pur) | Haiku | T.1 | `model/killBreakdown.ts` |
| P10 | T.3 score, rang et estimations (pur) | Sonnet | T.0, T.1 | gardes de suffisance et de score complet |
| P10 | T.4 grenades lancées, ramassages (pur) | Haiku | T.1 | |
| P10 | T.5 rendu dans le corps de fiche | Sonnet | D.1, T.2, T.3, T.4 | seul écrivain de `ReplayPlayerPanelBody.tsx` après D.1 |
| P8 Clôture | R.1 revue finale | **Fable** | lots A à G, S et T clos | adversarial-review du diff complet |
| P8 | R.2 corrections retenues | Sonnet ; Opus si l'écart est un algorithme | R.1 | une seule ronde |
| P8 | R.3 journal, mémoire, registre des reports | pilote | R.2 | |

Vagues :
- **Dès le go** : 0.1 (Haiku), 0.2 (Sonnet), 0.3 (Fable).
- **Après 0.3** : A.1, S.2, C.1, E.1, D.1, T.0, T.1 (sept Sonnet), B.1, F.1 (deux Haiku) en parallèle ;
  puis T.2 et T.4 (Haiku) dès T.1, T.3 dès T.0 et T.1.
- **Puis** : A.2 / A.3, E.2 → E.3 → E.4 ; S.1 dès A.3, S.3 dès C.1 et S.2, S.4 ; B.2 dès A.3 et S.2 ;
  C.2 dès A.3 et E.1.
- **Fin** : D.2, D.3, T.5, C.3, F.2, F.3, G.1 ; R.1, R.2, R.3.

## 1 ter. Points de contact avec le plan 3D

- **`ReplayCanvas.tsx`.** Le plan 3D extrait `buildScene` et `draw` du fichier (D.2, piste P3). E.1 ne touche
  que le rendu JSX (slots), pas la scène. Un seul écrivain à la fois : si D.2 du plan 3D est en cours, E.1
  attend son intégration et repart du fichier intégré ; sinon E.1 passe d'abord et D.2 repart de lui.
- **Amendement du plan 3D, item C.5** (à porter dans ce plan-là par R.3) : le commutateur 2D / 3D ne va pas
  DANS le panneau de `ReplayViewMenu.tsx` mais dans l'emplacement en superposition posé par F.3 (D-9).
- **Amendement du plan 3D, D-10 (HUD)** : en Cinéma et en Plein cadre, les fiches et le fil restent à l'écran
  (D-2, D-4) ; le HUD 3D n'a donc à reprendre ni le fil ni le score, dans aucun cadre.
- **Nord de la mini-carte 3D (D-7 du plan 3D)** : c'est l'orientation EFFECTIVE de la 2D, Auto comprise (D-14).
- **D-8 du plan 3D** reste vrai : la 3D prend la place de la toile 2D, les trois cadres l'accueillent tels quels.

## 2. Ce qui existe (relu sur pièces, `origin/release/v8.2.0` au 2026-11-15, `db11e95da1` ; À RELIRE sur `release/v8.3.0` au go, une fois la 8.2.0 fusionnée dans main)

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

- [ ] 0.1 Témoins « avant » sur `feat/rejeu-refonte-ui` avant tout code : fixations DOM (vitest, rendu jsdom
      avec le document de test `test/testDoc.ts`) de la ZONE DE LECTURE du Standard (`ReplayTransport` avec sa
      frise à pistes, et `ReplayZoomControl`), qui ne doit pas changer (D-8), et des lignes du fil, dont seul le
      lieu doit changer (D-5) ; rangées sous `apps/web/src/features/match-replay/ui/__fixtures__/` avec le suffixe
      `.avant.html`. Gate : les fixations existent et un test les relit à l'identique.
- [ ] 0.2 M-1 : script de mesure (page de démonstration du rejeu, toile à 720 puis 1080 px de haut sur une
      carte haute et une carte large) qui relève le tas JS au pic et la durée des cuissons au redimensionnement.
      L'agent l'écrit ; le user le lance à son écran ; résultats dans `.ai/annexes_refonte_ui_rejeu/mesure_m1.md`.
- [ ] 0.3 Audit de conception (Fable, lecture seule) : relit ce plan contre le code, propose le découpage des
      composants par cadre sans franchir 500 lignes ni grossir `ReplayCanvas.tsx`, liste les risques (Échap,
      export vidéo qui suit la mise en page, couches z, portails). Écarts au plan rapportés au user avant A.1.
- **Gate 0** : 0.1 et 0.3 rendus ; écarts de 0.3 tranchés par le user.

## 4. Lot A — les cadres

- [ ] A.1 `model/replayFrame.ts`, pur : type `ReplayFrame = 'standard' | 'cinema' | 'plein-cadre'`, transitions
      (bascule Cinéma, bascule Plein cadre, Échap), état initial `standard` à chaque montage (aucun stockage,
      même règle que le plein écran actuel : « une manière de regarder, pas un endroit »). Tests unitaires de
      toutes les transitions.
- [ ] A.2 `useReplayFullscreen.ts` scindé en deux gestes : Plein cadre = la superposition seule ; plein écran
      système = le natif seul, qui peut s'ajouter à n'importe quel cadre (en Standard il entre d'abord en Plein
      cadre, comme aujourd'hui). `fullscreenchange` reste la source de vérité du natif. Touches : `t` Cinéma,
      `f` Plein cadre, Échap selon les couches (`REPLAY_ESCAPE_LAYER_ATTR`). Export en cours : aucune bascule,
      comme aujourd'hui. Tests des deux gestes et des touches ; le garde-rail des couches d'Échap reste vert.
- [ ] A.3 Page découpée : `ReplayPage.tsx` ne garde que chargement, modèle et choix du cadre ; la disposition de
      chaque cadre va dans `ui/frames/ReplayStandardFrame.tsx`, `ui/frames/ReplayCinemaFrame.tsx` (Cinéma et
      Plein cadre, qui partagent la structure D-2). Standard : disposition d'aujourd'hui déplacée telle quelle
      dans son fichier (le lot S la refait), zone de lecture identique aux témoins 0.1. Cinéma : en-tête de
      page réduit au retour, au titre et au lien vers la fiche du match (tableau K). Plein cadre : sans navigation de l'app, une barre : Quitter / Échap,
      titre du match. Fichiers sous 500 lignes.
- **Gate A** : `npm --prefix apps/web run typecheck`, `npm --prefix apps/web run test:run -- src/features/match-replay`,
  `npm --prefix apps/web run lint`, `npm --prefix apps/web run lint:colors` verts ; fixations Standard égales
  aux témoins, hors différences listées dans le compte rendu.

## 5. Lot B — fiches en rangée (Cinéma, Plein cadre)

- [ ] B.1 `model/rowLayout.ts`, pur : `rowFits({ camps, placesParCamp, largeurCorps, gabarit, largeurScore })`
      selon D-12 ; la largeur vient de `GABARIT_COMPACT` et des écarts de la rangée, aucun littéral de largeur
      de fiche. Tests : 4v4 à 1440 px (tient), 5v5 à 1440 px, 4v4 à 1280 px, Grande équipe, FFA.
- [ ] B.2 `ui/ReplayTeamsRow.tsx` : `[fiches du camp de gauche] [bandeau de score] [fiches du camp de droite]` ;
      fiches = `ReplayPlayerCard` au gabarit compact (bouclier et santé en deux barres, grenades, équipement,
      tuiles d'attente comprises) ; `ReplayScoreBanner` au centre, à la largeur de sa cellule. Clic sur une fiche :
      `playerPanel.onCard` (ouvre D.3). Repli : quand `rowFits` est faux, le cadre rend la colonne du Standard
      (`ReplayRosterColumn`, lot S).
      Les cotes nouvelles éventuelles vont dans `cardGabarit.ts` (garde-rail existant). Tests : rendu 4v4, repli,
      tuile du joueur suivi, joueur mort.
- **Gate B** : commandes du Gate A ; `cardGabarit.guard.test.ts` vert.

## 5 bis. Lot S — le Standard selon le tableau A

- [ ] S.1 `ReplayStandardFrame.tsx` : contenu borné et centré (constante nommée de largeur, D-11) ; en-tête en
      trois cellules : retour et titre du match à gauche, `ReplayScoreBanner` resserré au centre (environ 440 px,
      manches et barre de prise de zone comprises, sans changer sa lecture `readScoreBanner`), lien vers la fiche
      du match à droite. Le bandeau de score ne coiffe plus le terrain en Standard. Tests : en-tête rendu, score
      présent, score absent en FFA (rien n'est rendu, comme aujourd'hui).
- [ ] S.2 `ui/ReplayRosterRow.tsx` et `ui/ReplayRosterColumn.tsx` : une ligne par place (nom ; bouclier et santé
      en deux barres via `VitalityBar` ; « Réapparition dans … » à la place des barres quand le joueur est mort ;
      arme en main ; grenades ; équipement ; F / M / A), en-tête de colonnes Arme / Gr. / Éq. / F / M / A ; places
      libres et occupants pas encore apparus rendus comme lignes d'attente (même règle que `ReplaySeatWaiting`) ;
      la ligne du joueur suivi se déplie (armes et munitions, grenades, charge de l'équipement). Lectures reprises
      de `model/playerCardReadings.ts`, aucune recalculée ; cotes dans `cardGabarit.ts` (troisième gabarit
      « ligne » si 0.3 le recommande). Clic sur une ligne : `playerPanel.onCard`. Tests : 4v4, joueur mort, place
      libre, joueur suivi déplié, Grande équipe (la part des équipes défile).
- [ ] S.3 Fil de la colonne : `ReplayKillFeed` rendu au format d'une ligne aux colonnes fixes (instant | tueur |
      arme | victime | lieu de la victime, C.1), sans fond ni bordure, plus récente en tête, défilant, titre = menu
      de filtre actuel. Les formes non-élimination du fil (objectifs, entrées et sorties, prises) gardent leur
      forme actuelle sur une ligne. Fixations des lignes comparées aux témoins 0.1 : seul le lieu change.
- [ ] S.4 Montage : la grille du Standard devient carte | colonne de 360 px (équipes puis fil, le fil garde au
      moins 5 lignes) ; la zone de lecture et le zoom restent ceux d'aujourd'hui ; les sections sous le lecteur
      (carte de chaleur, usage de l'équipement) inchangées. `ReplayTeams` et la colonne de 30 rem sortent du
      Standard ; ce qui n'a plus d'appelant est supprimé avec ses tests (règle 0 code mort), la fixation
      `replayTeams.4v4.html` suit le composant qui reste (rangée du lot B).
- **Gate S** : commandes du Gate A ; zone de lecture égale aux témoins 0.1 ; aucun fichier au-delà de 500 lignes.

## 6. Lot C — le fil sur le plan

- [ ] C.1 Lieu de la victime : `model/victimZone.ts` (ou extension de `killerZone.ts` si 0.3 le recommande),
      même règle de nommage (a)-(d) appliquée à la position de la VICTIME à l'instant du frag (dernier point de
      sa vie, même résolveur de porteur, même règle de `z`). Le nommage du point ne change pas : le jumeau Go et
      `nommage_zone_cas.json` restent tels quels. `useKillerZones` devient `useFeedZones` (lieu de la victime),
      l'ancien hook est supprimé avec ses imports (règle 0 code mort). Tests : victime à pied, victime en
      véhicule, victime sans position, bot.
- [ ] C.2 `ui/ReplayFeedOverlay.tsx` : les 5 dernières lignes à l'instant lu du fil FILTRÉ (même filtre que le
      Standard, `useFeedFilter`), plus récente en bas, sans en-tête ni menu, sans fond, ombre portée du texte
      (token, pas de couleur en dur), colonnes fixes partagées avec `ReplayKillLineParts`. Ancré en bas à gauche
      du BLOC de la toile (slot E.1), donc dans la marge d'une carte haute et sur le plan d'une carte large. La
      légende de la carte de chaleur passe en haut à gauche du bloc dans ces cadres. Tests : 0, 3, 5 et 8 lignes
      passées ; ordre ; filtre appliqué.
- [ ] C.3 Option « Fil des éliminations » (affiché par défaut) dans le panneau Vue, retenue dans
      `replayPreferences.ts` (clé nouvelle, garde-rail `replayPreferences.guard.test.ts`). Sans effet en
      Standard (la colonne y garde son fil). Libellés fr et en.
- **Gate C** : commandes du Gate A ; `killerZone.test.ts` / tests du nouveau module verts ; aucun test Go touché
  (`git diff --stat -- apps/go-api` vide).

## 7. Lot D — la fiche détaillée (J en Standard, L en Cinéma et Plein cadre)

- [ ] D.1 `ui/ReplayPlayerPanelBody.tsx` : contenu commun extrait de `ReplayPlayerPanel` (Suivre / Seul /
      Masquer, « À l'instant », « Totaux du match »), sans changer son rendu (fixation avant / après). Tests de
      comportement existants verts sans modification.
- [ ] D.2 Fiche du Standard (tableau J, D-6) : clic sur une ligne de la colonne, la colonne devient la fiche via
      `ReplayFeedOrPanel` (le fil reste monté dessous, mécanisme conservé). En-tête « Fiche » avec retour
      « Équipes » (couche `REPLAY_ESCAPE_LAYER_ATTR` : Échap rend la colonne avant de quitter un cadre) ;
      sélecteur des joueurs en rectangles arrondis ; `ReplayPlayerPanelBody` ; section « Éliminations » = les
      2 dernières lignes du fil du match (tous joueurs, filtre courant, le joueur ouvert mis en avant) au format
      d'une ligne de S.3, lien « Fil complet » qui rend la colonne ; frise à pistes inchangée
      (D-18). Tests : ouverture depuis une ligne, changement par le sélecteur, retour Équipes, Échap, 2 dernières
      lignes du fil, frise égale aux témoins 0.1 fiche ouverte.
- [ ] D.3 `ui/ReplayPlayerPopover.tsx` (tableau L, Cinéma et Plein cadre) : `ReplayPlayerPanelBody` dans un
      panneau ancré sous la fiche cliquée (`useAnchoredPanel`), portail, `REPLAY_ESCAPE_LAYER_ATTR`, croix,
      défilement interne. Lecture non interrompue, aucun voile. Clic sur une autre fiche : change de joueur. La
      fiche et le pion du joueur ouvert sont cerclés (token de focus). Tests : ouverture depuis la rangée et
      depuis la colonne de repli, changement de joueur, Échap ferme la fenêtre puis le cadre, Suivre / Seul /
      Masquer.
- **Gate D** : commandes du Gate A ; `escapeLayers.guard.test.ts` et `playerFocusWiring.guard.test.ts` verts.

## 7 bis. Lot T — la fiche à l'instant (D-19 à D-23)

Toutes les lectures sont pures, à l'image lue, pour le joueur de la fiche, bots compris (un bot a des vies et des
éliminations). Aucune requête nouvelle, aucun changement du document de rejeu ni du décodeur. Les formules
existantes se réutilisent (KDA du jeu, ADR 0006 ; catégorie « arme lourde » des niveaux d'armes du document) : 0.3
les localise, rien n'est recalculé à côté.

- [ ] T.0 Mesures sur le parc, en lisant les documents de rejeu DÉJÀ générés (aucune génération) : part des matchs
      dont le score par action est déclaré complet ; part des joueurs dont la lecture des tirs est jugée
      suffisante ; sur trois films, boucliers tombés à zéro contre dégâts subis (plausibilité de D-21, recharges
      comprises). Résultats dans `.ai/annexes_refonte_ui_rejeu/mesure_t0.md`. Une part trop faible pour valoir
      l'affichage est rapportée au user avant T.3.
- [ ] T.1 `model/playerInstantStats.ts` : à l'image lue, FDA, meilleure série et série en cours, tirs à la tête
      (une élimination dont le coup fatal n'est pas mesuré ne compte ni oui ni non), parfaits (médaille datée),
      réapparitions, durée de la vie en cours (absente quand le joueur est mort), moyenne des vies terminées
      jusqu'à l'instant, boucliers perdus (D-21 : passages à zéro du bouclier du joueur, une recharge puis une
      nouvelle chute comptent deux), médailles gagnées. Tests : début de match, joueur mort, bot, tir à la tête
      non mesuré, deux boucliers perdus dans une vie, fin de match égale au tableau de score sur les valeurs
      exactes (fixation d'un match réel).
- [ ] T.2 `model/killBreakdown.ts` (D-20) : éliminations jusqu'à l'instant par catégorie (arme lourde, mêlée,
      grenade, autres armes) puis par arme, nombre et part en pourcentage ; une élimination sans arme résolue
      compte dans « autres » sous un libellé neutre, jamais sous l'arme d'une voisine. Tests : parts qui somment à
      100, catégorie vide absente, arme inconnue.
- [ ] T.3 Score, rang et estimations (D-22) : score à l'instant = somme des points par action quand le document
      déclare le score complet ; rang dans le lobby seulement si tous les joueurs du lobby sont complets ;
      précision, tirs, touches, dégâts infligés et subis, et par arme tirs et dégâts, pour un joueur à lecture
      suffisante uniquement. Tests : score incomplet (pas de ligne), joueur insuffisant (pas de ligne), joueur
      suffisant (valeurs), rang à égalité.
- [ ] T.4 Grenades lancées par type et ramassages (armes, grenades, équipement) jusqu'à l'instant (D-23). Tests :
      aucun lancer (section absente), types mêlés.
- [ ] T.5 Rendu dans `ReplayPlayerPanelBody` (D.1), donc dans J et dans L : « À l'instant » garde son bloc actuel
      (bouclier, santé, armes, grenades, équipement) puis ajoute les compteurs de T.1, la répartition de T.2, les
      lignes de T.3 (mention « estimation » sur les valeurs estimées) et les sections de T.4 ; « Totaux du match »
      inchangé ; l'ordre des sections du corps de fiche laisse la place de « En passant » en dernier (D-24), rien
      n'y est rendu. Libellés fr et en, sans anglicisme (« série », « tirs à la tête », « répartition des
      éliminations ») ; tokens de couleur uniquement. Tests de rendu : joueur complet, joueur insuffisant, bot,
      défilement de la fiche.
- **Gate T** : commandes du Gate A ; fixation de fin de match de T.1 égale au tableau de score.

## 8. Lot E — la barre de lecture compacte

- [ ] E.1 Slots de `ReplayCanvas.tsx` : `transport` (élément fourni par le cadre : la barre actuelle en Standard,
      la compacte ailleurs) et `blockOverlays` (posés dans le bloc, à côté de la légende et du zoom). Le fichier
      ne grossit pas : la différence nette de lignes est nulle ou négative (le choix de barre et de zoom sort du
      fichier). Zone de lecture du Standard identique aux témoins.
- [ ] E.2 Extraire de `ReplayTransport.tsx` ce que la barre compacte partage : boutons de capture et d'export
      (`ui/ReplayCaptureButtons.tsx`, avec la règle de l'élargissement pendant l'export), boutons de cadre
      (`ui/ReplayFrameButtons.tsx`, D-10, remplace `FullscreenButton`). Zone de lecture du Standard identique aux
      témoins hors les icônes de cadre.
- [ ] E.3 `ui/ReplayTransportCompact.tsx` : une ligne. À gauche caméra (icône seule, E.4) et son ; puis lecture,
      −10, +10, temps, frise compacte (repères d'éliminations et de morts du joueur regardé et de ses alliés,
      bande de dominance, lues dans le même objet `timeline` que `ReplayTimelineTracks`), vitesse, zoom − / +
      (la commande de zoom flottante n'est pas rendue dans ces cadres) ; à droite capture, export, boutons de
      cadre. Raccourcis inchangés. Tests : commandes présentes, frise compacte alimentée par le même fil que la
      frise à pistes, export en cours neutralise la lecture.
- [ ] E.4 Menu caméra (D-13) : « Suivre » (liste des joueurs, même action que Suivre de la fiche, plus « Aucun ») et
      « Point de vue » (`useReplayViewpoint.select`). Tests des deux sections.
- **Gate E** : commandes du Gate A ; fixations Standard égales aux témoins hors icônes de cadre.

## 9. Lot F — plan tourné et superpositions

- [ ] F.1 `autoTurns(carte, boîte)` dans `lib/replay/orientation.ts`, pur : 0 ou 1 quart de tour selon le gain
      de surface (seuil 25 %, retour sous 20 %, D-14). Tests sur Detachment (1526 × 1666), Ronin (659 × 1288),
      Nadair (2133 × 947) dans les boîtes du Standard et du Cinéma à 1440 × 900.
- [ ] F.2 Orientation « Auto » par défaut (D-14) : `useReplayOrientation` distingue « aucun choix » (Auto) d'un
      choix retenu ; section Vue avec « Auto » ; bouton « Plan tourné » dans la barre COMPACTE quand l'Auto a
      tourné, qui retient 0° (en Standard, la barre ne change pas, D-8 : le retour passe par la section Vue). Une carte déjà retenue garde son choix. Tests : Auto, choix retenu, bouton, hystérésis.
- [ ] F.3 Emplacement 2D / 3D en superposition à gauche du bouton « Vue », dans le coin haut droit de la toile :
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
- [ ] R.2 Corrections retenues, une seule ronde.
- [ ] R.3 `make gate-push` vert ; entrée `.ai/thought_log.md` ; amendements C.5 et D-10 portés dans
      `.ai/PLAN_REJEU_3D_2026-11-15.md` ; `[!]` reportés dans `.ai/REGISTRE_REPORTS.md` ; mémoire du chantier.
- [ ] R.4 Validation à l'écran par le user (artefact de vérifications), témoins nommés par lui : Standard
      selon A (zone de lecture inchangée), Cinéma 4v4 sur Detachment, Cinéma sur Nadair (fil sur le plan), Plein cadre, fiche du Standard (J), fiche en fenêtre (L),
      Grande équipe en repli, carte tournée. La fusion dans `release/v8.3.0` attend cette validation.

Chaînes UI : toute chaîne nouvelle existe en fr et en (`i18n/fr.ts`, `i18n/en.ts`, typage
`Record<ReplayLocale, ReplayText>`), sans anglicisme côté fr. Couleurs : tokens uniquement (skill
`color-tokens`). Aucune requête nouvelle, aucune route, aucun code Go : `lib/query/keys.ts`, `openapi.yaml` et
`generated.ts` ne bougent pas.

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
