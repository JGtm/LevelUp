# Retours des contrôles v8 (2026-10-21) — tri, NON VALIDÉ

Source : fiche interactive https://claude.ai/artifact/BM28Ax8NZ3HFFZmJzWcJBy (base `resultats`,
relue le 2026-10-21 vers 10:15), étape 3.2 de `.ai/PLAN_RELEASE_V8_2026-10-17.md`. Contrôle n° 43
ajouté à la fiche le 21/10 à la demande de l'utilisateur.

Statut : **tri seulement**. Aucun lot n'est lancé tant que l'utilisateur n'a pas validé le
découpage (règle « remarque n'est pas validation »). Un correctif du décodeur fait remonter le
schéma de rejeu : recuisson locale + rsync (séquence de la release), à prévenir.

## 0. Bilan de la fiche

- 49 contrôles : 39 OK (dont 16 avec un bémol), 5 KO (T2, n° 7, 24, 34, 43), 3 sans statut
  (T1, T3, n° 38). Aucun « v8.1 ».
- KO n° 34 = contrôle périmé (la vue du match a désormais 4 onglets) ; reste à vérifier que
  l'ancien lien `?tab=details` mène à un onglet existant.

## 1. Lots proposés (ordre de priorité)

### L1 — Sessions d'escouade éclatées (T2, priorité très haute de l'utilisateur)

**Cause établie le 21/10 sur copie des bases** (serveur local arrêté 2 min, accord de
l'utilisateur) : les 9 matchs du 17/10 (JGtm + Madina97294 + Chocoboflor + un 4e joueur qui
change) étaient UNE session (n° 207, 9 matchs) tant que seule la synchro écrivait. Le recalcul
complet du 19/10 18:38 (2.10b) a réécrit 1 122 à 1 129 : une session par match, sauf les deux
matchs où le 4e joueur était le même. Les deux chemins n'utilisent pas la même règle :
- synchro (`engine_postsync_scoring.go`) : `analysis.DefaultSessionOptions()` en dur, mode
  « amis seulement », réglage ignoré ;
- recalcul admin (`settings_sessions_recalc.go` -> `sessionComputeOptionsFor`) : lit le réglage,
  et le `app_settings.json` local porte `"session_team_change_mode": "group"` (« Changement de
  groupe » : rupture dès qu'un coéquipier quelconque change).
Effet : tout l'historique de JGtm est passé d'environ 209 à 1 131 sessions. La prod a reçu les
bases locales le 20/10 : même état. Le recalcul a eu lieu pour Halo 5 (sessions jamais calculées
avant, plan backlog du 16/10) ; l'utilisateur en a pris note le 21/10.
Correctif proposé : une seule source d'options pour les deux chemins (+ garde-rail), réglage
remis à « Amis seulement » (local, prod à lire), recalcul des sessions en local puis en prod,
contrôle page Escouade (session du 17/10 = 9 matchs) et Sessions. Décision attendue sur le
réglage.
### L2 — Vue du match et tableaux (web, sans décodeur)

- n° 43 : bloc Médias = vignettes posées à nu sous le titre de section quand il y a des médias
  (`MatchMediaTab.tsx`, seul l'état vide est dans une carte). Attendu : un bloc au format des
  autres (titre de bloc, tous les médias dans ce bloc).
- n° 34 : ancien lien `?tab=details` -> onglet existant (vérifier, corriger si besoin).
- n° 33 : icône de rejeu pas toujours centrée dans les tableaux (Explorer).
- n° 13 (libellé) : « ne reviendra plus » -> « a quitté la partie ».

### L3 — Rejeu : sons et événements

- n° 7 (KO) : pas de son « manche terminée » (témoin d9781168).
- n° 24 (KO) : rejeu E vers 0:10, beaucoup de lancers de grenades, aucune explosion.
- n° 28 : effet du translocateur sur la fiche absent à la pose, présent à la 1re téléportation.
- n° 13 : suicide affiché par un point médian au lieu de l'icône tête de mort.
- n° 17 : sons de véhicule qui continuent quand on met en pause.

### L4 — Rejeu : véhicules

- n° 11 : tirs du Wasp depuis sa pointe avant ; explosion du Ghost (Banished) en rouge, comme tous
  les véhicules Banished.
- n° 17 : rotations des véhicules saccadées (7b0d89c4), à fluidifier.
- n° 29 : taille maximale des véhicules sur petit écran.

### L5 — Rejeu : BTB

- n° 8 : 4f77afc1, « places libres » et 13 fiches dans l'équipe Eagle (impossible). Règle des
  places finies par équipe déjà tenue en 4v4, à étendre au BTB.

### L6 — Rejeu : lectures à instruire avant toute correction

- n° 3 : Satoru JZ ramasse le surbouclier (son, fiche, compte à rebours du socle) avant d'être
  passé sur le socle : décalage de temps ou de position à mesurer.
- n° 12 : 0d094c9b, AfterHoney65177 commence le match avec le surbouclier selon sa fiche.
- n° 10 : calage des médias sur la frise : 72b0a25e (la 1re capture finit par le triple frag mais
  la vidéo démarre à l'instant de la médaille) ; bfecd02b (le média de JGtm finit 1-2 s après la
  mort de Chocoboflor à 2:15).
- n° 3 (interface) : power-up sur socle et emplacement vide n'ont pas le code couleur des armes
  spéciales (blanc bordé de jaune sur socle, jaune quand vide) — à aligner.

### L7 — Questions ouvertes à l'utilisateur (aucun code avant réponse)

- T3 : « Composition exacte » mal nommé ; usage attendu = garder la session quand un ami
  plante et revient. Proposer 2-3 libellés concis et factuels + le comportement par défaut.
- n° 38 : « Temps mort » en pied de fiche : rien ne s'affiche ; expliquer ce que la ligne devait
  montrer, puis garder ou retirer.
- n° 12 : classé abandonné par un joueur -> MMR à 0 pour tous : exclure ces matchs des
  statistiques de classement ? (décision produit)
- T1 : les trois points laissés ouverts le 19/10 (bouton rejeu sans garde de capacité,
  Citations basculées sur le nom du titre, assistances attendues Halo 5 dans la vue du match).
- n° 11 : les instants indiqués dans la fiche (7:19, 7:35) ne correspondaient pas ; rien à
  corriger côté application.

## 2. Notion

Lignes des encadrés « Contrôles visuels / techniques » rayées le 21/10 pour les contrôles OK
(demande de l'utilisateur) ; les KO et sans statut restent visibles jusqu'à leur correctif.

## 3. Réponses de l'utilisateur (2026-10-21)

- Découpage validé (« ok avec le reste »), avec trois précisions :
  - **Bloc Médias** : le bloc s'affiche TOUJOURS, avec son titre de bloc, et contient soit tous
    les médias, soit le message « aucune capture » (comme en v7.3.2, où titre et contenu
    vivaient dans la même carte). Aujourd'hui le titre est un titre de section hors du bloc.
  - **Présence au rejeu** : « ne reviendra plus » -> « a quitté la partie » ET « entre en partie »
    -> « a rejoint la partie » (mêmes libellés que la présence lue dans l'API). Réserve à
    rappeler : sur un mode à manches, le film ne distingue pas un départ d'une élimination
    définitive (infobulle actuelle).
  - **Suicide au kill-feed** : icône `static/weapons-assets/halo_infinite/jeu/killfeed-61.png`.
- Recalcul des sessions Halo 5 : pris en note.
- Arrêt du serveur local accordé (aucune autre session ne l'utilisait, confirmé par levelup-3a).

## 4. Journal L1 (sessions)

- 2026-10-21 : réglage « Composition de l'équipe » remis à `friends` en local (`app_settings.json`) et en prod
  (copie de sauvegarde `/tmp/app_settings.json.bak-20261021` sur le VPS). PIÈGE : `sed -i` sur l'hôte remplace
  l'inode et le conteneur (montage d'un fichier seul) garde l'ancien ; le fichier vu par le conteneur a été réécrit
  sur place (`cat >`), les deux vues sont identiques. Le prochain déploiement recrée le conteneur et réaligne les
  inodes ; d'ici là, un enregistrement des réglages depuis l'app prod s'écrirait dans l'ancien inode et serait perdu
  au déploiement.
- `CutoffHour` n'est lu nulle part : les options du recalcul avec `friends` = celles de la synchro. Le recalcul
  peut donc réparer les données avec le binaire actuel, sans attendre le correctif.
- Recalcul : route admin seulement (bouton Réglages > Analyse), il faut une session admin connectée (local et prod).
- Correctif du code (une seule source d'options + garde-rail) : exécutant sur `fix/sessions-regle-unique`
  (worktree `LevelUp-wt-sessions`), en cours.
- 2026-10-21 12:48 (local et prod, bouton lancé par l'utilisateur) : recalcul en mode `friends` pour tous les profils
  des deux titres, aucun échec au journal (profils sans match ignorés). Vérifié en prod sur copie de la base de JGtm :
  la session du 17/10 = 1 session de 9 matchs ; 219 sessions pour 1 182 matchs Infinite (contre 1 131 avant).
  Local : mêmes comptes de matchs mis à jour que la prod, joueur par joueur.

## 5. Journal L2-L4 rendu web (`fix/controles-v8-web`, poussée le 21/10)

- Faits (exécutant Sonnet, tests verts : tsc, vitest complet 10 231 tests, Go `mappings`) : bloc Médias en carte à titre
  toujours affichée ; icône de rejeu centrée (Explorer, Synergies) ; présence « a rejoint / a quitté la partie » ;
  pause = tous les sons longs coupés (moteurs relancés à tort par glissé / pas d'image / saut) ; effet de fiche à la
  pose de la faille (`riftPoses`) ; Wasp tire du nez ; explosions rouges des véhicules Banished ; cap interpolé
  (plus court chemin) ; plafond de taille des véhicules à 12 % du côté court du canevas.
- `?tab=details` : déjà redirigé vers l'onglet Chronologie, testé : rien à faire.
- [!] Suicide : le client affiche déjà `killfeed-61.png` pour un suicide typé (b1ad85eb : 2) ; 396cfc92 ne publie
  aucune mort neutre typée -> transmis au lot décodeur (`fix/controles-v8-rejeu`).
- Reste : E2E Playwright en local avant fusion, gate visuel de l'utilisateur.
- E2E Playwright local (21/10, démo synthétique sur API :8021 + vite :5181, `fix/controles-v8-web`) : 58 réussis, 17
  ignorés, 32 en échec, TOUS dans des specs qui appellent l'API en dur sur `localhost:8000` (le serveur réel, hors
  démo -> 403). Aucun échec dans une spec d'écran. Cause d'environnement ; la CI de la PR vers `main` rejoue tout.

## 6. Journal L6 n° 10 (calage des médias, `fix/medias-calage`, 21/10)

- [x] Règle par source (table unique `captureTimeSources`) : « Replay » OBS, « record that » Game Bar et console Xbox,
  Instant Replay ShadowPlay et date de fichier = FIN de la capture ; enregistrement OBS / ShadowPlay = DÉBUT.
- [x] Recalcul des médias déjà indexés : par le serveur à chaque indexation (sans arrêt) ; simulation et écriture hors
  serveur par `levelup media-capture-times`. NON lancé sur les vraies données (superviseur).
- [!] Écart résiduel bfecd02b (~43 s, clip réellement 1:52 → 2:20 au chrono du jeu, posé 2:34 → 3:03) : horloge du PC de
  capture en avance (mesuré sur 6 clips, +2 à +43 s selon la date), hors de portée de la règle. 72b0a25e : 2-3 s.
