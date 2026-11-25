# PLAN — Backlog du 24 novembre (2026-11-24)

Statut : **écrit le 2026-11-24 à la demande du user (« Pour moi faut faire 1 à 29 », « Les répliques de
spartan va falloir le faire », « Le 10 faut partir sur une maquette réaliste »). Relu contre la grille
`plan-review` et son complément `levelup-plan`. **EXÉCUTÉ le 2026-11-25, tout est dans `release/v8.3.0`, CI verte ; reste la validation à l'écran du user (artefact des vérifications, chantier « Backlog du 24 novembre (8.3.0) », 17 lignes), due avant `main` ; le plan reste à la racine de `.ai/` jusqu'à elle.** GO DONNÉ par le user le 2026-11-24 (« ok avec ton plan et tes autres recos », cible 8.3.0 confirmée) ; il lui reste
la validation de la maquette (M.3).**

Contrat d'exécution : skill `plan-execution`, complément `levelup-plan`. Statuts : `[x]` fait, `[~]` couvert
ailleurs (référence), `[!]` non traité (motif écrit). Aucune case vide à la clôture. Ordre strict dans une
piste ; un item est clos quand son gate est passé et son résultat écrit sous sa ligne. Zéro correctif hors
périmètre : section 9.

## 0. Objectif et critère de succès

Traiter les entrées du backlog retenues par le user le 2026-11-24, livrer la rationalisation des réglages du
rejeu sur maquette validée, puis l'affichage des répliques des Spartans.

Critère : chaque entrée retenue est corrigée et sort du backlog, ou porte un motif d'abandon écrit ; CI de
`release/v8.3.0` verte ; chaque changement visible a sa ligne dans l'artefact des vérifications.

Cible confirmée par le user le 2026-11-24 : la **8.3.0**.

## 1. Décisions

- **D-1 — périmètre.** User, 2026-11-24 : entrées 1 à 29 de la liste remise ce jour, plus les répliques.
- **D-2 — trajet de Warthog suivi par un bot : écarté.** Consigne du user (« poubelle si l'effort est plus
  grand que faible ») ; l'entrée disait effort moyen, côté décodeur. Entrée retirée du backlog.
- **D-3 — icônes de Dend : retirées du backlog** (user : « on a déjà traité »).
- **D-4 — hors plan.** Chantier précision : reporté par le user. Points du décodeur (drapeau en Grande
  bataille, scores, contrôle du corpus, limites des calques, petits points) : ce sont les restes écrits à la
  clôture du plan du décodeur v1.6.0 le 2026-11-24, non corrigés (vérifié : la règle du drapeau et le compteur
  mal nommé sont dans la v1.6.1 publiée) ; ils restent au backlog, hors de ce plan.
- **D-5 — match Halo 5 incomplet après une panne passagère : abandon sauf si 0.4 dit le contraire (D-13).** Mesuré le 2026-11-24
  sur la base du poste : 3 032 matchs Halo 5, 0 sans participants, 303 sans chronologie (cause non établie :
  panne passagère, ou chronologie jamais servie par l'API). Recommandation : abandonner, sauf si l'item 0.4
  montre que l'API sert encore la chronologie de ces matchs.
- **D-6 — réglages du rejeu : tout revoir, sur une maquette au plus près de la vraie application.** User,
  2026-11-24 : « faut revoir tous les réglages des calques et tout sur la page du replay, avec une maquette qui
  ressemble le plus à la vraie app, car en ajouter encore ça fait vraiment beaucoup et le concept des presets
  est bien mais peuvent sembler rigides pour le user donc faut trouver un compromis ». Donc : prototype
  interactif bâti sur le rendu réel de la page (mêmes cadres, mêmes composants, mêmes couleurs, vrai fond de
  carte), tous les réglages inventoriés, et un compromis à proposer entre préréglages et réglage fin : un
  préréglage est un point de départ que le user ajuste librement, pas un mode fermé. Aucun code avant sa
  validation.
- **D-7 — ordre réglages puis répliques.** L'entrée du backlog pose la rationalisation des réglages avant
  l'affichage des répliques : la piste R attend la validation de la maquette M.
- **D-8 — répliques : la fonction plaît au user (2026-11-24) ; leurs réglages se décident dans la maquette M.**
  Défauts, emplacement et son des répliques sont des réglages parmi les autres : la maquette M les porte et
  le user les tranche en la validant. Voix : versées au dépôt (user, 2026-11-24 : « Oui on verse les voix au depot »).
- **D-9 — base et branches.** Base `release/v8.3.0`, après le report de `main` (étape 1). Branche
  d'intégration `feat/backlog-2026-11-24` ; une sous-branche et un worktree par piste
  (`<poste>/Downloads/Scripts/LevelUp-wt-bl24-<piste>`, sans jonction vers `data/`). Fusion dans
  `release/v8.3.0` par piste, dès son gate (go permanent du user sur la release). `main` : geste du user.

- **D-10 — pièces montées à plus de 15 m de leur porteur : INCLUS (user, 2026-11-24).** Le correctif est dans
  le décodeur : piste F, une version de la brique, une montée dans la 8.3.0, régénération des seuls films
  touchés (le user est prévenu avant, aucune régénération du parc).
- **D-11 — « mètres » : INCLUS (user, 2026-11-24 : « faut mieux le faire »).** La conversion se vérifie à
  l'étape 0 (item 0.6) ; le facteur retenu revient au user s'il n'est pas établi sans doute ; puis toute
  distance affichée est juste (piste B, item B.6).
- **D-12 — retirés du backlog (user, 2026-11-24) :** alerte glib, licence du parseur voisin.
- **D-13 — recommandations retenues par le user (2026-11-24, « ok avec tes autres recos ») :** jalon de
  précision laissé tel quel ; fonds Forge mesurés avec le plan par mode en 8.3.1 ; table des distances de
  touche gardée jusqu'au chantier précision ; D-5 : abandon du correctif Halo 5 sauf si 0.4 montre que
  l'API sert encore la chronologie.

- **D-14 — décisions du user du 2026-11-24 sur l'étape 0 (« ok avec tes recos »).**
  - Mètres : une unité du jeu = 3,048 m (10 pieds). Les valeurs AFFICHÉES sont converties ; les 22 champs
    d'API nommés en mètres ne sont pas renommés, ils sont documentés comme des unités du jeu.
  - Radar de la Tactique : corrigé (le rayon en vrais mètres est comparé à des distances converties) ; les
    comptes « isolé » changent, c'est dit dans les notes de version.
  - Coup d'envoi après la fin du match : la fenêtre du match est bornée par le plus long temps de jeu quand
    la durée du registre est manifestement fausse.
  - Cartes Némésis (B.1) : écarté, entrée retirée du backlog.
  - Pseudos réels (E.4) : limité aux trois noms de tests.

- **D-15 — maquette des réglages validée par le user le 2026-11-24 : variante A (rubriques en onglets).** Citation :
  « hyper stylé !!!! Je valide la variante A elle est ouf !!! ». La variante B (repliable avec recherche)
  n'est pas retenue. Une seule modification demandée : **aucun réglage de langue des répliques**. Les
  répliques sont en français quand l'interface est en français, en anglais dans toute autre langue.
  À coder par la source unique des langues (`config/locales.toml`, repli sur l'anglais), sans comparaison
  de langue en dur. La maquette publiée montre encore ce réglage : il ne se code pas.

- **D-16 — Emprise, part « hors radar » des vies (user, 2026-11-24 : « Ok avec ta reco par rapport aux
  calculs »).** Le calcul est corrigé maintenant (la portée du radar est transmise au décodeur en unités du
  jeu) ; les vies déjà rangées se recalculent à la prochaine passe sur le parc, sans passe lancée exprès.
  Un match sans film garde sa valeur. Item B.7.

## 2. Orchestration

- Pilote : cette session. Elle ne code pas, relit chaque compte rendu sur pièces, intègre, lance la suite web
  complète une fois et lit la CI après chaque poussée.
- Modèles : **Fable** pour l'étape 0 et la revue finale ; **Sonnet** par défaut ; **Haiku** pour les items
  marqués (H). Aucun item Opus.
- Borne : 4 agents d'écriture à la fois.
- Chemin critique : étape 1 (report de `main`), puis étape 0, puis les pistes. La piste M démarre en même
  temps que l'étape 0 (inventaire et maquette, aucun code).
- Fichiers partagés : `features/match-replay` est touché par C, M et R. C fusionne avant l'implémentation de
  M ; R part après M. `features/match-view` : B seule.
- Coût estimé : 1 agent Fable (étape 0), environ 25 agents Sonnet, 4 Haiku, 1 revue finale Fable.

## 3. Étape 1 — report de `main` dans `release/v8.3.0` (pilote)

- [x] 1.1 (2026-11-24, fusion `c1a72cad60`, contrôlée par le pilote : `main` est ancêtre de la release) Fusionner `origin/main` dans `release/v8.3.0` (2 commits manquants le 2026-11-24 ; un essai passé
      donnait 16 conflits). Chaque conflit résolu est listé sous cette ligne.
- **Gate 1 :** `cd apps/go-api && go build ./... && go vet ./...` ; `make check-types` ; tests des paquets
  en conflit ; poussée ; `gh run list --branch release/v8.3.0 --workflow CI --limit 1` verte.

## 4. Étape 0 — instruction sur pièces (Fable, lecture seule)

Rendu : `.ai/annexes_backlog_1124/etape0.md`. Pour chaque entrée des pistes A à E : le défaut existe-t-il
encore sur la tête de la release (fichier et ligne), sa mesure, les fichiers à toucher, l'effort.

- [x] 0.1 (2026-11-24, annexe `etape0.md`) Relecture des entrées des pistes A à E, une ligne par entrée.
- [x] 0.2 (2026-11-24 : les quatre entrées sont closes ; zone du tueur livrée en 8.2.0, ménage du disque corrigé par `84ec5ed295`, contrôlé par le pilote) Entrées peut-être déjà closes : zone du tueur dans le fil (citée livrée en 8.2.0), ménage du disque
      (le script dit épargner les worktrees), deux films à l'ancien schéma et arme des touches sur l'index
      brut (parc régénéré le 2026-11-24). Verdict par entrée.
- [x] 0.3 (2026-11-24, `temoins_avant.md` : comptes SQL capturés et tests de service listés ; les sorties de pages ne sont pas capturées, faute de serveur) Témoins avant : sorties des pages touchées par A et B sur trois matchs (fil, cartes Némésis, onglet
      d'équipe, section Objectifs), rangées dans `annexes_backlog_1124/temoins_avant.md`.
- [x] 0.4 (2026-11-24, sans appel à l'API : au plus 66 matchs sur 3 032 touchés, 2,2 %, surtout des parties à un joueur de 2015-2016 ; abandon) Halo 5 (D-5) : dire si l'API sert encore la chronologie de trois des 303 matchs, par la commande de
      synchronisation existante, sans écrire en base.
- [x] 0.5 (2026-11-24 : trois lignes ajoutées à l'artefact des vérifications) Les trois écarts de maquette du rejeu « à juger à l'écran » : ajoutés à l'artefact des vérifications.
- [x] 0.6 (2026-11-24 : une unité vaut environ 3 m, établi par trois mesures ; 3,0 ou 3,048 non séparés ; 17 affichages web et 22 champs d'API listés) « Mètres » (D-11) : établir la conversion d'une unité du jeu (hypothèse : 10 pieds, soit 3,048 m)
      sur des mesures du jeu (taille d'un Spartan, vitesse de course, portées connues) ; lister chaque
      distance affichée à l'écran et sa source.
- **Gate 0 :** annexe écrite, chaque entrée classée « à faire », « déjà faite » ou « à écarter (motif) » ; le
  user tranche D-5 et les entrées proposées à l'écart.

## 5. Pistes

Gates communs. Go : `go build ./... && go vet ./...`, tests des paquets touchés, `go test ./internal/archlint/...`,
`make go-api-lint`. Web : `make check-types`, `npx vitest run <dossiers touchés> --maxWorkers=2`,
`node tools/lint-cross-feature-imports.mjs`. Toute chaîne d'interface en fr et en en.

### Piste A — données et fil (Go ; `-tags=integration` au gate)

- [x] A.1 (`f245776240`, revue `edab899e0b` : la complétion n'insère que le surplus ; migration `shared_highlight_events_dedupe_v1`, 38 573 lignes retirées sur la base du poste, aucune ligne légitime touchée) Éliminations en double dans `highlight_events` : trouver l'écrivain qui double, corriger, dédoublonner
      par la voie d'écriture prévue (règles anti-ART), compter la prod en lecture seule.
- [x] A.2 (avec A.1 : même défaut ; la citation des captures ne compte qu'en Bases, une fois par joueur et par instant) Temps forts « mode » en double et sans code (76 matchs) : écrivain, repassage, retrait des doubles ;
      sens de ces temps forts lu par le nom de l'action de score.
- [x] A.3 (`062777bb67`, revue `6c96986189`, `e3935f552d`, `5376f0eec0`, `fae8dd8095` : marque « film absent » retirée des films complets, 34 sur 616 ; fenêtre du match bornée par le plus long temps de jeu humain au-delà de 10 s ; Mainmise sur la même fenêtre) Match dit « film absent » alors que son film est archivé ; deux rejeux dont le coup d'envoi tombe
      après la fin du match.
- [x] A.4 (`21039ed233`, revue `26368c3dd0` : résolution sous le segment d'écriture, coup d'envoi « film » contrôlé, compteur des résolutions) Coup d'envoi, trois points : course entre rattrapage et publication, contrôle du coup d'envoi « film »
      contre la fenêtre du match, compteur incrémenté avant l'insertion.
- [x] A.5 (`775068f07c` : tri à départage, révision `lu2` seulement si tout est lu, vue des médailles v2) Médailles du bouclier, trois défauts : tri sans départage, révision posée sans lecture complète,
      vue qui décide « dérivé » sur la présence d'une ligne.
- [x] A.6 (`434ddce143` : un accès refusé sur le verrou est un verrou tenu ; l'instabilité d'origine n'a pas été reproduite, cause déduite) Écriture concurrente du catalogue de cartes perdue (test instable).
- **Gate A :** gates Go, `go test -tags=integration ./internal/persist/... ./internal/sync/... ./internal/migration/...`,
  comptes avant / après des doubles (0 attendu après), revue adversariale du diff de la piste (écritures).

### Piste B — page du match (Go et web)

- [!] B.1 (écarté par le user le 2026-11-24, D-14 : volume nul sur Halo Infinite, 6 éliminations sur Halo 5) Cartes Némésis : victime ambiguë nommée, ou éliminations gardées dans le fil.
- [x] B.2 (`562b0efb26` : tri à départage stable ; la section « némésis » n'existe plus côté serveur) Ordre déterministe des sections « rencontres » et « némésis » de l'onglet d'équipe.
- [x] B.3 (`15f9300c3b`, web seul : six colonnes dans la section Objectifs ; séries temporelles non faites, sens seulement en cumul par période, coût moyen, notées en découverte) Éliminations liées à l'objectif sur la fiche de match, par mode ; séries temporelles si elles y ont
      un sens (règle : Sessions, séries temporelles, Escouade).
- [x] B.4 (`3590da8093` : bloc non publié sans équipe alliée lue, colonne de match à 44 px au moins, à juger à l'écran) Tableau « Impact des coéquipiers » : pas de bloc sans signal d'équipe, largeur minimale des colonnes,
      sept ré-exports sans lecteur retirés.
- [x] B.5 (`c583d1eca3` : champs et fonctions retirés, plus la source d'historique Halo 5 qui ne servait qu'à `LoadMatchSummaries` ; contrat régénéré) Champs servis sans lecteur (`nemesis_duels`, `medal_name`, bloc du golden) et fonctions d'analyse
      sans appelant : supprimés avec leurs tests, `make generate-types`.
- **Gate B :** gates Go et web, témoins 0.3 comparés (seuls les écarts voulus), contrat OpenAPI régénéré.

- [x] B.6 (`98dfbdace2`, `c1695f5ef0`, `01f81e8414`, `002ed9703a` : helper web et jumeau Go avec garde-rails, radar comparé dans une seule unité, 11 familles de distances converties, 22 champs documentés) « Mètres » (D-11) : chaque distance affichée convertie par un seul helper (Go et web), avec son
      garde-rail ; noms internes trompeurs (`metersPerPixel` et voisins) laissés si aucun affichage n'en
      dépend, renommage noté en découverte sinon.

- [x] B.7 (`27a9de07e0` : portée convertie en un point, `radar_m` reste en vrais mètres, rapport médiane sur portée corrigé à la lecture, `PlacementRev` montée ; recalcul par `levelup backfill-killsource`, non lancé) Emprise (D-16) : portée du radar convertie avant d'être transmise au décodeur ; lecteurs de
      `radar_m` et `beyond_ms` relus ; la prochaine passe sur le parc recalcule l'existant.

### Piste C — rejeu (web, un peu de Go)

- [~] C.1 (livrée en 8.2.0 ; le fil affiche le lieu de la victime par décision de la refonte ; étape 0) Zone du tueur à chaque ligne du fil (si 0.2 ne la dit pas livrée).
- [x] C.2 (`83ace65640` : pastille « Collines A › B › C », prochaine après un cycle complet répété ; emplacement à juger à l'écran) Suite des collines jouées, et la prochaine quand elle est connue.
- [x] C.3 (`3a66f1936e` : mêlée, lancers amorcés, accroupis, saut au bouton dans le groupe « Gestes » ; le contrat sert `gestures`) Gestes du joueur dans la fiche (section « En passant ») : champs publiés par le décodeur v1.6.
- [x] C.4 (`0a18f92fb4` : lignes de bot rattachées par place, bots dans les antagonistes ; `xuid` facultatif sur les éliminations et actions de score) Éliminations et actions de score d'un bot désignées par place ; bots rouverts dans les antagonistes.
- [x] C.5 (`042d0df614` : trois fichiers au lieu d'un de 781 lignes, test coupé, une seule mesure de largeur et son garde-rail ; 6 356 tests avant et après) Fichiers au-dessus de 500 lignes (liste des joueurs, test de la page) ; quatre mesures de largeur
      réunies en une, avec son garde-rail.
- [x] C.6 (`2d9a7367ef` : panneau de la vue d'ensemble d'administration) Compte des rejeux à l'ancien schéma dans la vue d'administration.
- **Gate C :** gates web et Go, vérification EN LECTURE sur deux rejeux, lignes ajoutées à l'artefact des
  vérifications.

### Piste D — serveur et production (Go ; `-tags=integration` au gate)

- [x] D.1 (`a1990f6270` puis revue `79dc2f95f0`, `997a98028f`, `f1003466bc` : archive coupée, la source des frags n'écrit rien dans aucun profil ; défaut par la capacité `ArchiveFilmsByDefault` ; gestes de production A1 à A3) Archivage des films : le calcul des frags respecte le réglage ; pas d'archivage par défaut sur le
      site hébergé ; conservé sur le bureau. Le retrait de la tâche du serveur va dans les gestes de production.
- [x] D.2 (`bf3718ed37` puis revue `a5254b8be7`, `1cb7cb33e4`, `7ebc10a000`, `845b5709b6` : sur banc simulé à douze comptes, 665 ms avant, moins de 1 ms après ; temps réel non mesuré) Démarrage : pool de jetons créé vide, comptes résolus en tâche de fond, port ouvert sans attendre.
      Revue adversariale propre (authentification).
- [x] D.3 (`94723378ab` : alerte au franchissement du seuil puis tous les 48 cycles ; avertissement de jeton une fois par épisode) Alerte « zero-insert » qui ignore les matchs rangés par une autre voie ; avertissement de mise en file
      quand aucun compte n'a consenti.
- [!] D.4 (abandon, D-5 et D-13 : au plus 66 matchs sur 3 032 ; une commande de reprise manuelle existe) Halo 5 incomplet : seulement si D-5 le retient.
- **Gate D :** gates Go, `go test -tags=integration ./internal/sync/... ./internal/scheduler/...`, démarrage
  mesuré avant / après sur le poste (temps jusqu'à « serveur prêt »).

### Piste E — hygiène (H pour les items marqués)

- [~] E.1 (déjà corrigé le 2026-11-22, commit `84ec5ed295` ; étape 0) Ménage du disque : épargner worktrees enregistrés et dépôts à commits non poussés (si 0.2 le dit
      encore à faire), journal de ce qui est épargné.
- [x] E.2 (`395b4b82e4` : 23 fichiers, sans effet à l'écran ; le « libellé de garde-rail faux » n'est pas identifié avec certitude, deux messages d'un test de garde corrigés) Commentaires périmés, libellé de garde-rail faux, doc d'une commande (EN et FR).
- [x] E.3 (`ff05b2b2b5` : garde des dates indifférente à l'année ; le balayage `libs/` de la garde des processus enfants est ACTIF, il désigne les briques du cache de modules : gardé, commentaire corrigé ; test à octet nul introuvable et non reproduit sur ce poste) Garde-rail des jetons figé sur 2026 ; garde « processus enfants » qui balaie un dossier disparu ;
      test au nom de fichier invalide sous Windows.
- [x] E.4 (`389c8731ca` : trois noms de tests et leurs 13 lignes de la référence ; D-14) Pseudos réels : trois noms de tests, surnoms dans les tests, fichiers hors `apps/`.
- [x] E.5 (`3fe2325280` : champ retiré de l'émission et des deux manifestes, aucun code ne le lisait) `seed-demo --emit-manifest` n'écrit plus l'identifiant réel du joueur source.
- **Gate E :** gates Go et web, `go test ./...` complet (renommages de tests et référence des tests).

### Piste F — décodeur : pièces montées (D-10)

Dans le dépôt de la brique (`halo-infinite-film-decoder`), sur une branche et un worktree dédiés.

- [x] F.1 (2026-11-24, mesure du parc régénéré : 0 pièce à plus de 15 m sur 2 619, 191 films ; 0 sur les 4 films nommés ; recontrôlé par le pilote sur un film) Instruction : les 22 cas sur 4 films, cause dans le choix du porteur par voisinage de place.
- [~] F.2 (sans objet : défaut corrigé par la v1.5.0 du décodeur, qui lit le porteur dans le film ; F.1) Correctif par une règle générale, mesuré sur les 4 films et sur le banc (aucun cas neuf).
- [~] F.3 (sans objet, F.1 : aucune version de la brique) Version de la brique : empreintes `replay-equiv` refigées, gate du corpus, notes de version qui
      disent le changement de sortie, tag et release GitHub.
- [~] F.4 (sans objet, F.1 : ni montée ni régénération) Montée du `require` dans LevelUp ; régénération des seuls films touchés, le user prévenu avant.
- **Gate F :** tests et gates de la brique verts, 0 pièce montée à plus de 15 m sur les 4 films, CI de la
  brique et de la release vertes.

### Piste M — réglages d'affichage du rejeu (D-6)

- [x] M.1 (2026-11-24 : 66 réglages, 17 surfaces, constats ; annexe `reglages_inventaire.md`) Inventaire de tous les réglages du rejeu (calques, informations sur la carte, vues, préférences),
      où ils vivent, ce qu'ils règlent, leurs défauts. Rendu : `annexes_backlog_1124/reglages_inventaire.md`.
- [x] M.2 (2026-11-24 : publiée, https://claude.ai/artifact/FZKvRLrcujrfnFMQac2kYk ; sources `annexes_backlog_1124/maquette_reglages/` ; vue par le pilote dans un navigateur, sans erreur de console ; lignedans l'artefact des vérifications) Maquette au plus près de la vraie application (D-6) : prototype interactif publié en artefact, tous les réglages, répliques comprises,
      compromis préréglages et réglage fin ; ligne dans l artefact des vérifications.
- [x] M.3 (validée le 2026-11-24 : « Je valide la variante A elle est ouf », D-15) Validation du user (citation consignée ici). Une remarque n'est pas une validation.
- [x] M.4 (`34cce33ae3` à `7c9de6e7de`, fusion `b28e67c5d1` : modèle par table, panneau à cinq rubriques, préréglages souples, Mes vues, accès rapide, point de vue depuis la fiche ; grille `maquette_reglages/conformite.md`, 20 écarts motivés) Implémentation conforme à la maquette, après la fusion de C. Migration des préférences existantes.
- **Gate M :** gates web, grille de conformité à la maquette, vérification EN LECTURE.

### Piste R — répliques des Spartans (après M.3 et D-8)

- [x] R.1 (`0432c6de24`, commande `cmd/chatter-build`, table `static/sounds/halo_infinite/repliques/repliques.json`) Table des textes des répliques par langue (commande de table), sans voix.
- [x] R.2 (`a2510ff5ee` contrat servi, `ff84137829` affichage) Affichage dans le rejeu selon la maquette publiée et les choix D-8 ; réglages rangés dans le panneau
      de M.
- [x] R.3 (`a7089784d1`, 7,66 Mo, français seul) Voix versées au dépôt (D-8) ; les voix anglaises restent à recevoir, hors de ce plan si elles n'arrivent pas.
- **Gate R :** gates Go et web, vérification EN LECTURE sur un rejeu en 1.12 ou 1.13 (répliques nommées),
  ligne dans l'artefact des vérifications.

## 6. Clôture

- [x] Z.1 (2026-11-25, sur la branche réunie : suite web complète, 15 909 tests passés, un garde en dépassement de délai sous charge qui passe seul ; `go test ./...` et intégration de `persist`, `migration`, `sync`, `games`, `platform/duckdb`, `service` : un écart réel trouvé et corrigé, le compte des champs du contrat du rejeu, porté à 81 ; restent trois échecs propres au poste, en découverte) Suite web complète une fois (`npx vitest run`), `go test ./...`, `go test -tags=integration ./...`.
- [x] Z.2 (2026-11-25 : trois relecteurs indépendants, un Fable et deux Sonnet, une ronde ; aucun P0 ; deux P1 sur les préréglages du rejeu, corrigés avec leurs tests ; dix P2 traités ; reports au backlog) Revue finale Fable du diff cumulé (skills `adversarial-review` et `levelup-revue`) ; P0 et P1
      corrigés, P2 au backlog.
- [x] Z.3 (2026-11-25 : listes de livraison passées sur le diff cumulé ; CI de la release verte sur trois runs de suite) Skills `delivery-checklist` et `levelup-livraison` ; CI de la release lue.
- [x] Z.4 (2026-11-25 : backlog, registre, journal, notes de version ; 14 caches de compilation rendus, 33 Go ; 8 worktrees retirés et leurs branches supprimées ; un dossier vide `LevelUp-wt-bl24-a/apps/go-api` reste tenu par un processus de la session) Backlog : entrées traitées passées en « Récemment complété » ; reports au registre ; journal ;
      caches de compilation rendus (`scripts/disk-hygiene.ps1 -Lot bl24-<piste> -Apply`) ; worktrees retirés
      dans un tour à part.

## 7. Blocages connus

- Voix anglaises des répliques : à recevoir.
- `main` : fusion et tag restent au user ; `make gate-push` une seule fois, avant `main`.
- La 8.3.0 attend aussi le rejeu 3D (accord du contributeur).

## 8. Reprise de session

Lire ce fichier (statuts, résultats sous chaque item), l'annexe de l'étape 0, puis `git log --oneline -10` sur
`feat/backlog-2026-11-24` et `ListAgents`. Reprendre au premier item non coché de chaque piste dont la
dépendance est close.

## 9. Découvertes

- (piste B) `raw_json.medal_name` des événements de médaille est encore écrit (flux principal, complétion, passe et commande de rattrapage) sans lecteur.
- (piste B) `MatchViewRepo.LookupMedalMetaByName` sans appelant de production ; `MatchHistoryService.dataAdapter` assigné, jamais lu ; `mapMatchSummaries` de Halo 5 peut-être sans appelant.
- (piste B) Le pied de la grille de valeurs affiche « 0 » pour une colonne dont toutes les valeurs sont absentes.
- (piste B) Le snapshot de surface du contrat est en retard sur des ajouts antérieurs.
- (piste B) Séries temporelles des éliminations d'objectif : possibles en cumul par période, coût moyen.
- (revue du pool) Le plafond du pool n'est pas tenu si un compte arrive pendant le remplissage (sans effet : la production ne pose pas de plafond) ; un retrait de consentement fait pendant une reconnexion peut être ignoré jusqu'au re-scan suivant (motif antérieur) ; deux tests du pool échouent quand on les répète sous détection de courses.
- (pilote) `TestWeaponLabels_ArtefactReelDuPoste` (`internal/service`) échoue sur un poste qui a le rejeu : il lit le fichier compressé comme du texte. Antérieur au plan (test inchangé depuis la 8.2.0), invisible en CI et dans les worktrees, où il se saute.
- (piste C) `MatchNemesisCards.logic.ts` dit encore qu'un bot n'est jamais un antagoniste (page du match) ; deux copies identiques d'un hook de largeur sur la liste datée du garde ; commentaire « 58 clés » périmé dans le schéma du document.
- (piste B.6) Emprise : la part « hors radar » des vies (`beyond_ms`, table `match_life_placement`) est calculée à la synchronisation avec la portée du radar en mètres comparée à des unités ; 304 576 vies sur 1 454 matchs du poste portent la valeur sous-estimée. La portée est passée au décodeur par LevelUp (`sync/killcollector/capture.go`) : le correctif est dans LevelUp, le recalcul de l'existant demande les films. Décision du user attendue.
- (piste B.6) Seuils réglés sur des unités mais nommés « M » : étiquettes corrigées, comportement inchangé ; une dizaine de constantes web gardent un nom trompeur.

## 10. Journal d'exécution

- 2026-11-24 — plan écrit ; go du user le même jour ; pièces montées (piste F) et « mètres » (0.6, B.6) ajoutés à sa demande.
- 2026-11-24 — étape 1 : le report de `main` donne 20 fichiers en conflit (code de synchronisation, contrat,
  CI, notes de version). Résolution confiée à un agent Sonnet dans le worktree `LevelUp-wt-bl24-report`,
  branche `chore/report-main-8-2-3`. L'étape 0 et M.1, en lecture seule, partent en même temps : les entrées
  dont les fichiers sont touchés par le report (Halo 5, synchronisation) seront relues après lui.
- 2026-11-24 — décisions du user : voix des répliques versées au dépôt (D-8) ; entrée du schéma des bases
  joueur retirée du backlog ; entrée des fichiers de variante de carte gardée et reformulée, en attente de la
  réponse de Den sur Cartographer.
- 2026-11-24 — ÉTAPE 0 rendue (annexes `etape0.md` et `temoins_avant.md`). Sur 27 items : 21 à faire, 2 déjà
  faits (C.1, E.1), D.4 abandonné, B.1 proposé à l'écart (volume nul sur Halo Infinite, 6 éliminations sur
  Halo 5), E.2 et E.3 avec un point chacun non localisable. Écarts avec le plan : A.1 et A.2 sont un seul
  défaut (une migration) ; A.3 touche 36 matchs et les deux « coups d'envoi après la fin » viennent d'une
  durée fausse au registre ; B.2 se réduit aux « rencontres » ; B.6 touche aussi un calcul (rayon du radar
  de la Tactique comparé à des distances en unités). Gate 0 en attente des décisions du user : facteur de
  conversion, radar de la Tactique, champs d'API en `_m`, fenêtre du coup d'envoi, écart de B.1, périmètre
  de E.4.
- 2026-11-24 — ÉTAPE 1 close. Fusion faite par un agent dans son worktree, relue par le pilote puis intégrée
  (`c1a72cad60`). Choix de l'agent, retenu : sur l'historique de synchronisation, la forme publiée en 8.2.2 et
  8.2.3 remplace le premier jet de la release (carte « Synchronisation », clé `history_complete_at`, lecteur
  et dépôt de l'état de l'historique), incompatible avec elle ; l'arbre fusionné est identique à `main` sur ce
  sujet (vérifié par `git diff origin/main`). Le premier jet se relit dans `221c50dcbb`. Contrat et types
  régénérés, lots de CI en union. Gate de l'agent : build, vet, tests des paquets touchés, intégration de
  `sync` et `halo_5`, typecheck, vitest ciblé, tous à 0 ; les tests de convergence de `cmd/server` restent
  à la CI. Deux vérifications à l'écran mises à jour dans l'artefact (carte et aide de synchronisation).
  Entrée « Reporter la 8.2.2 » retirée du backlog.
- 2026-11-24 — pistes A, B, C et D lancées (un agent Sonnet et un worktree `LevelUp-wt-bl24-<piste>` chacune,
  branches `bl24/a` à `bl24/d` prises sur la release après le report). Précision de D-9 : chaque branche de
  piste se fusionne directement dans `release/v8.3.0` à son gate, sans branche d'intégration intermédiaire.
  Tenus hors des consignes en attendant le user : B.1, B.6, la partie « durée fausse au registre » de A.3.
  E et F partent quand une place d'agent se libère (borne de 4).
- 2026-11-24 — GATE 0 passée : le user retient les six recommandations (D-14). B.1 écarté ; A.3 (durée du
  registre) et A.4 (fenêtre) débloqués et transmis à l'agent de la piste A ; B.6 part après la piste B, dans
  son worktree.
- 2026-11-24 — M.1 et M.2 closes. Maquette : page entière aux trois cadres, carte animée sur les fonds réels
  d'une carte versionnée, 5 rubriques (Joueurs, Carte, Objets, Fil, Son), préréglages souples (« modifié »,
  retour, Mes vues, vue par défaut, vue par mode), accès rapide épinglable, variantes A (onglets) et B
  (repliable avec recherche), comparaison avec le panneau actuel, répliques intégrées. Commandes visibles à
  l'ouverture du panneau : 29 aujourd'hui, 14 en A, 19 en B. M.3 attend le user.
- 2026-11-24 — M.3 close : variante A validée par le user, sans réglage de langue des répliques (D-15). M.4
  part après la fusion de la piste C (mêmes fichiers).
- 2026-11-24 — PISTE B (B.2 à B.5) rendue, contrôlée par le pilote sur pièces et fusionnée (`0ffd0bd25b`).
  Gate de l'agent : build, vet, tests des paquets touchés, intégration de `platform/duckdb` et `teammates`,
  lint, contrat et types frais, typecheck, vitest ciblé (1 801 tests), tous à 0. Rien n'est vu à l'écran :
  trois lignes à l'artefact des vérifications. B.6 (mètres) repart dans le même worktree.
- 2026-11-24 — PISTE D rendue (D.1 `a1990f6270`, D.2 `bf3718ed37`, D.3 `94723378ab`), en revue adversariale
  avant fusion (deux relecteurs Sonnet). Relecteur du pool : 6 constats recevables, 14 conditions qui
  tiennent, aucun P0 ; 4 retenus à corriger dans le lot (journal de fin de résolution faux, acquisition
  publique endormie, course sur `MarkUnhealthy` exposée à chaque démarrage, tâche différée lancée à l'arrêt),
  2 en découverte.
- 2026-11-24 — PISTE E rendue, contrôlée sur pièces et fusionnée (`0648caa1df`). Gate de l'agent : build,
  vet, tests de 86 paquets, intégration de quatre paquets, lint des couleurs, typecheck, tous à 0. La
  comparaison complète des tests à la référence (`check_test_baseline.sh`, 40 minutes) n'est pas jouée : elle
  fait partie de `make gate-push`, avant `main`. Découvertes : deux autres gardes figées sur « 2026- »
  (`checkpoint_single_site_test.go`, `no_inline_per_minute_test.go`) ; `pkg/duckdbbackup` lance restic hors
  garde ; un pseudo reste dans `.gitignore`, deux scripts de banc et l'ADR 0018.
- 2026-11-24 — PISTE D fusionnée (`2168335b54`) après UNE ronde de revue adversariale (deux relecteurs Sonnet).
  Pool de jetons : 6 constats recevables, 4 corrigés avec leur test (bilan de résolution, acquisition
  endormie, course sur `MarkUnhealthy`, boucle différée à l'arrêt), 2 en découverte. Archivage et journal :
  5 constats recevables ; le « relais » d'archivage est retiré (le client Halo du serveur n'a pas de cache de
  film : il n'évitait aucun second téléchargement et laissait des films sur le disque) ; garde d'architecture
  neuve sur les magasins de réglages qui écrivent ; le cas « durée de conservation sans clé d'archive » va aux
  gestes de production. Aucun P0. Gate de l'agent rejoué après corrections, tout à 0 ; build et vet rejoués
  par le pilote sur la branche intégrée.
- 2026-11-24 — CI rouge après la fusion de la piste B, deux causes : huit tests du code supprimé encore dans la
  référence (retirés, `4eeba506c6`) ; garde de taille des fonctions du front à 244 pour 243 (`SessionMatrix`,
  corrigé par l'agent, `89243f758e`). Leçon : les gardes globaux (`src/lib` côté web, référence des tests
  côté Go) tournent avant chaque fusion de piste, pas seulement les tests ciblés.
- 2026-11-24 — PISTE F close sans code. Le « 22 cas sur 4 films » du backlog était un compteur de voisins
  refusés, pas une distance ; il vaut 0 partout. Sur le parc au schéma 127 : 2 619 pièces montées, 2 614
  porteurs lus dans le film, 0 à plus de 15 m, repli par voisinage jamais déclenché. Aucune version du
  décodeur, aucune montée, aucune régénération. Reste au backlog : 5 pièces sans porteur sur 3 films. Worktree
  du décodeur retiré.
- 2026-11-24 — PISTE C fusionnée (`c7a86b82ee`). Gates du pilote sur la branche intégrée (pistes B, C, D, E
  réunies) : build, vet, contrat frais, `archlint`, `api`, `replaybuild`, `domain`, tests de contrat à 0 ;
  typecheck à 0 ; vitest de `src/lib` (tous les gardes globaux), `match-replay`, `squad`, `match-view`,
  `admin` : 10 899 tests, 0 échec. Un test de `service` échoue sur ce poste pour une cause antérieure
  (découverte). Changements de contrat retenus : champ `gestures`, `xuid` facultatif sur deux types, deux
  plafonds datés montés (80 champs de tête, surface 341). Quatre lignes à l'artefact des vérifications.
- 2026-11-24 — B.6 (mètres) fusionnée. Gates du pilote sur la branche intégrée : build, vet, contrat frais,
  `archlint`, `api`, `analysis`, `domain`, tests de contrat à 0 ; typecheck à 0 ; vitest de `src/lib`, de la
  Tactique, de la fiche de match et des graphes : 4 430 tests, 0 échec. Rien n'est vu à l'écran : une ligne
  à l'artefact des vérifications. Effet attendu à dire dans les notes de version : distances affichées
  multipliées par 3,048 ; plus de morts « isolées » dans la Tactique (rayon ramené d'environ 55 m à 18 m).
- 2026-11-24 — PISTE A rendue (A.1 à A.6), en revue adversariale avant fusion (deux relecteurs Sonnet, une
  ronde). Migration de dédoublonnage vérifiée sur la base réelle : 38 573 lignes retirées, aucune ligne
  légitime touchée. Constats recevables : 4 (écritures) et 7 (coup d'envoi) ; aucun P0 ; un P1 (la complétion
  saute la seconde de deux lignes identiques légitimes). Sept corrections transmises à l'agent : surplus à la
  complétion, forme « document », fonction sans appelant, durée « manifestement fausse » (humains seuls,
  marge de 10 s), test discriminant, fenêtre de la Mainmise, commentaires du compteur. En découverte : 524
  temps de jeu écrêtés sur la base du poste par la révision non publiée du 22/11.
- 2026-11-24 — D-16 : le user retient la recommandation sur l'Emprise ; B.7 lancé dans le worktree de la piste B.
- 2026-11-24 — décision du user : le temps de jeu écrêté (35 matchs, environ 550 joueurs sur le poste) va au backlog, hors du plan. Emprise avant la passe sur le parc : les valeurs anciennes restent affichées (recommandation du pilote, non contestée).
- 2026-11-25 — PISTE A fusionnée (`e706b720ed`) après UNE ronde de revue adversariale : 11 constats recevables,
  aucun P0, un P1 corrigé (surplus à la complétion), six autres corrections faites, un constat au backlog
  (temps de jeu écrêté). À la fusion avec la piste D, `postsync.go` passait à 501 lignes : ses requêtes vont
  dans `postsync_requetes.go`. B.7 (Emprise) fusionnée par-dessus. Gates du pilote sur la branche intégrée :
  build, vet avec et sans intégration, contrat frais, gofmt, tests de `archlint`, `sync`, `analysis`,
  `persist`, `migration`, `service`, `platform/duckdb`, `games`, `mapcatalog`, contrat, et intégration de
  `persist`, `migration`, `sync`, `halo_infinite` : tout à 0, sauf le test de poste déjà en découverte.
  Révisions montées : `kickoff-2026-11-24b`, `perf-2026-11-24b`, `lusr-2026-11-24b`, `lu2`,
  `placement-2026-11-24`. Gestes de production inscrits.
- 2026-11-25 — M.4 rendue, relue par le pilote, trois corrections demandées puis fusion (`b28e67c5d1`) :
  traits du relief gardés par fond (le rendu validé du Plan ne change pas, ni dans la Tactique ni dans
  « Occupation du terrain ») ; aucun réglage partagé avec une autre page n'est écrit par un préréglage ou une
  vue (garde-rail `sharedSettings.guard.test.tsx`, 29 réglages couverts) ; « Prendre le point de vue » dans la
  fiche. Écarts retenus : une vue se rattache à l'un des six enjeux lus du film ; « Réinitialiser » ne touche
  pas aux traits du relief. Gates du pilote sur la branche intégrée : typecheck à 0 ; vitest de `src/lib`,
  `match-replay`, Tactique et fiche de match : 10 339 tests, 0 échec. Trois lignes à l'artefact.
- 2026-11-25 — CI : quatrième cause de rouge, intermittente (`EnvironmentTeardownError` dans un test de la
  toile du rejeu). Cause : l'échec de la requête du manifeste des véhicules journalisait et redessinait après
  le démontage. Corrigé avec son test, vu en échec par mutation (`8c4800dde8`).
- 2026-11-25 — PISTE R fusionnée (`bfab87cce0`) : table des répliques (519 parlées, textes 53 Ko), affichage
  (bulles, icône de voix, fil, familles, fréquence par phrase, réglages dans le panneau), voix françaises
  (519 fichiers, 7,3 Mo) chargées à la demande (4 s d'avance, 16 voix en mémoire au plus, 17 Mo au pire).
  Le document servi porte `chatter` : plafond du contrat à 81.
- 2026-11-25 — REVUE FINALE (Z.2), une ronde, trois relecteurs sur ce qui n'avait pas été relu. Réglages et
  répliques (Fable) : 4 constats recevables, 16 conditions qui tiennent ; deux P1 corrigés (un préréglage ne
  coupait qu'une catégorie de son ; la vue d'ouverture ne valait pas pour les réglages disponibles après
  l'ouverture), plus la réinitialisation, la table des répliques demandée une fois, et les tests de Seul et
  Masquer sur la bulle et la voix. Mètres et page du match (Sonnet) : 4 P2, 18 conditions qui tiennent ; test
  de bout en bout du branchement vers le décodeur, helper sans appelant retiré, garde des distances lues
  brutes, règle des records. Rejeu et contrat (Sonnet) : 4 P2, 17 conditions qui tiennent ; champ sans lecteur
  retiré, garde morte, titre de test, date du compte affichée. Fusions `67ee911d1d`, `420150b776`,
  `c9050ec637`.
- 2026-11-25 — CI de `release/v8.3.0` : cinq causes de rouge trouvées et fermées pendant l'exécution
  (référence des tests, taille d'une fonction, identifiant fictif d'un test, avertissement après démontage,
  fermeture de couche hors du mécanisme commun). Verte sur `99010dcd`, `420150b7`, `67ee911d`. Leçon retenue :
  avant chaque fusion de piste, le pilote lance tous les gardes du front (`npx vitest run guard`) et les
  tests de contrat, pas seulement les tests des dossiers touchés.
- 2026-11-25 — CLÔTURE (Z.4) : aucun item sans statut. Backlog : 19 entrées closes ou écartées, 10 allégées, 8 découvertes ajoutées. Registre des reports alimenté. Notes de version et journal des modifications complétés dans les deux langues. Caches rendus, worktrees retirés.
- 2026-11-25 — recalcul de l'Emprise fait sur le poste à la demande du user (`levelup backfill-killsource`) : 1 456 films en 34 min 41 s, 0 erreur ; 1 453 matchs à la révision `placement-2026-11-24`. L'essai à vide a joué les migrations de la 8.3.0 sur la base du poste : 377 348 temps forts après le retrait des doubles, le compte attendu.
