# PLAN — Backlog du 24 novembre (2026-11-24)

Statut : **écrit le 2026-11-24 à la demande du user (« Pour moi faut faire 1 à 29 », « Les répliques de
spartan va falloir le faire », « Le 10 faut partir sur une maquette réaliste »). Relu contre la grille
`plan-review` et son complément `levelup-plan`. GO DONNÉ par le user le 2026-11-24 (« ok avec ton plan et tes autres recos », cible 8.3.0 confirmée) ; il lui reste
les voix des répliques (D-8) et la validation de la maquette (M.3).**

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
  le user les tranche en la validant. Reste à lui : verser ou non les voix au dépôt (avant R.3).
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

- [ ] 1.1 Fusionner `origin/main` dans `release/v8.3.0` (2 commits manquants le 2026-11-24 ; un essai passé
      donnait 16 conflits). Chaque conflit résolu est listé sous cette ligne.
- **Gate 1 :** `cd apps/go-api && go build ./... && go vet ./...` ; `make check-types` ; tests des paquets
  en conflit ; poussée ; `gh run list --branch release/v8.3.0 --workflow CI --limit 1` verte.

## 4. Étape 0 — instruction sur pièces (Fable, lecture seule)

Rendu : `.ai/annexes_backlog_1124/etape0.md`. Pour chaque entrée des pistes A à E : le défaut existe-t-il
encore sur la tête de la release (fichier et ligne), sa mesure, les fichiers à toucher, l'effort.

- [ ] 0.1 Relecture des entrées des pistes A à E, une ligne par entrée.
- [ ] 0.2 Entrées peut-être déjà closes : zone du tueur dans le fil (citée livrée en 8.2.0), ménage du disque
      (le script dit épargner les worktrees), deux films à l'ancien schéma et arme des touches sur l'index
      brut (parc régénéré le 2026-11-24). Verdict par entrée.
- [ ] 0.3 Témoins avant : sorties des pages touchées par A et B sur trois matchs (fil, cartes Némésis, onglet
      d'équipe, section Objectifs), rangées dans `annexes_backlog_1124/temoins_avant.md`.
- [ ] 0.4 Halo 5 (D-5) : dire si l'API sert encore la chronologie de trois des 303 matchs, par la commande de
      synchronisation existante, sans écrire en base.
- [ ] 0.5 Les trois écarts de maquette du rejeu « à juger à l'écran » : ajoutés à l'artefact des vérifications.
- [ ] 0.6 « Mètres » (D-11) : établir la conversion d'une unité du jeu (hypothèse : 10 pieds, soit 3,048 m)
      sur des mesures du jeu (taille d'un Spartan, vitesse de course, portées connues) ; lister chaque
      distance affichée à l'écran et sa source.
- **Gate 0 :** annexe écrite, chaque entrée classée « à faire », « déjà faite » ou « à écarter (motif) » ; le
  user tranche D-5 et les entrées proposées à l'écart.

## 5. Pistes

Gates communs. Go : `go build ./... && go vet ./...`, tests des paquets touchés, `go test ./internal/archlint/...`,
`make go-api-lint`. Web : `make check-types`, `npx vitest run <dossiers touchés> --maxWorkers=2`,
`node tools/lint-cross-feature-imports.mjs`. Toute chaîne d'interface en fr et en en.

### Piste A — données et fil (Go ; `-tags=integration` au gate)

- [ ] A.1 Éliminations en double dans `highlight_events` : trouver l'écrivain qui double, corriger, dédoublonner
      par la voie d'écriture prévue (règles anti-ART), compter la prod en lecture seule.
- [ ] A.2 Temps forts « mode » en double et sans code (76 matchs) : écrivain, repassage, retrait des doubles ;
      sens de ces temps forts lu par le nom de l'action de score.
- [ ] A.3 Match dit « film absent » alors que son film est archivé ; deux rejeux dont le coup d'envoi tombe
      après la fin du match.
- [ ] A.4 Coup d'envoi, trois points : course entre rattrapage et publication, contrôle du coup d'envoi « film »
      contre la fenêtre du match, compteur incrémenté avant l'insertion.
- [ ] A.5 Médailles du bouclier, trois défauts : tri sans départage, révision posée sans lecture complète,
      vue qui décide « dérivé » sur la présence d'une ligne.
- [ ] A.6 Écriture concurrente du catalogue de cartes perdue (test instable).
- **Gate A :** gates Go, `go test -tags=integration ./internal/persist/... ./internal/sync/... ./internal/migration/...`,
  comptes avant / après des doubles (0 attendu après), revue adversariale du diff de la piste (écritures).

### Piste B — page du match (Go et web)

- [ ] B.1 Cartes Némésis : victime ambiguë nommée, ou éliminations gardées dans le fil.
- [ ] B.2 Ordre déterministe des sections « rencontres » et « némésis » de l'onglet d'équipe.
- [ ] B.3 Éliminations liées à l'objectif sur la fiche de match, par mode ; séries temporelles si elles y ont
      un sens (règle : Sessions, séries temporelles, Escouade).
- [ ] B.4 Tableau « Impact des coéquipiers » : pas de bloc sans signal d'équipe, largeur minimale des colonnes,
      sept ré-exports sans lecteur retirés.
- [ ] B.5 Champs servis sans lecteur (`nemesis_duels`, `medal_name`, bloc du golden) et fonctions d'analyse
      sans appelant : supprimés avec leurs tests, `make generate-types`.
- **Gate B :** gates Go et web, témoins 0.3 comparés (seuls les écarts voulus), contrat OpenAPI régénéré.

- [ ] B.6 « Mètres » (D-11) : chaque distance affichée convertie par un seul helper (Go et web), avec son
      garde-rail ; noms internes trompeurs (`metersPerPixel` et voisins) laissés si aucun affichage n'en
      dépend, renommage noté en découverte sinon.

### Piste C — rejeu (web, un peu de Go)

- [ ] C.1 Zone du tueur à chaque ligne du fil (si 0.2 ne la dit pas livrée).
- [ ] C.2 Suite des collines jouées, et la prochaine quand elle est connue.
- [ ] C.3 Gestes du joueur dans la fiche (section « En passant ») : champs publiés par le décodeur v1.6.
- [ ] C.4 Éliminations et actions de score d'un bot désignées par place ; bots rouverts dans les antagonistes.
- [ ] C.5 Fichiers au-dessus de 500 lignes (liste des joueurs, test de la page) ; quatre mesures de largeur
      réunies en une, avec son garde-rail.
- [ ] C.6 Compte des rejeux à l'ancien schéma dans la vue d'administration.
- **Gate C :** gates web et Go, vérification EN LECTURE sur deux rejeux, lignes ajoutées à l'artefact des
  vérifications.

### Piste D — serveur et production (Go ; `-tags=integration` au gate)

- [ ] D.1 Archivage des films : le calcul des frags respecte le réglage ; pas d'archivage par défaut sur le
      site hébergé ; conservé sur le bureau. Le retrait de la tâche du serveur va dans les gestes de production.
- [ ] D.2 Démarrage : pool de jetons créé vide, comptes résolus en tâche de fond, port ouvert sans attendre.
      Revue adversariale propre (authentification).
- [ ] D.3 Alerte « zero-insert » qui ignore les matchs rangés par une autre voie ; avertissement de mise en file
      quand aucun compte n'a consenti.
- [ ] D.4 Halo 5 incomplet : seulement si D-5 le retient.
- **Gate D :** gates Go, `go test -tags=integration ./internal/sync/... ./internal/scheduler/...`, démarrage
  mesuré avant / après sur le poste (temps jusqu'à « serveur prêt »).

### Piste E — hygiène (H pour les items marqués)

- [ ] E.1 Ménage du disque : épargner worktrees enregistrés et dépôts à commits non poussés (si 0.2 le dit
      encore à faire), journal de ce qui est épargné.
- [ ] E.2 (H) Commentaires périmés, libellé de garde-rail faux, doc d'une commande (EN et FR).
- [ ] E.3 Garde-rail des jetons figé sur 2026 ; garde « processus enfants » qui balaie un dossier disparu ;
      test au nom de fichier invalide sous Windows.
- [ ] E.4 (H) Pseudos réels : trois noms de tests, surnoms dans les tests, fichiers hors `apps/`.
- [ ] E.5 `seed-demo --emit-manifest` n'écrit plus l'identifiant réel du joueur source.
- **Gate E :** gates Go et web, `go test ./...` complet (renommages de tests et référence des tests).

### Piste F — décodeur : pièces montées (D-10)

Dans le dépôt de la brique (`halo-infinite-film-decoder`), sur une branche et un worktree dédiés.

- [ ] F.1 Instruction : les 22 cas sur 4 films, cause dans le choix du porteur par voisinage de place.
- [ ] F.2 Correctif par une règle générale, mesuré sur les 4 films et sur le banc (aucun cas neuf).
- [ ] F.3 Version de la brique : empreintes `replay-equiv` refigées, gate du corpus, notes de version qui
      disent le changement de sortie, tag et release GitHub.
- [ ] F.4 Montée du `require` dans LevelUp ; régénération des seuls films touchés, le user prévenu avant.
- **Gate F :** tests et gates de la brique verts, 0 pièce montée à plus de 15 m sur les 4 films, CI de la
  brique et de la release vertes.

### Piste M — réglages d'affichage du rejeu (D-6)

- [ ] M.1 Inventaire de tous les réglages du rejeu (calques, informations sur la carte, vues, préférences),
      où ils vivent, ce qu'ils règlent, leurs défauts. Rendu : `annexes_backlog_1124/reglages_inventaire.md`.
- [ ] M.2 Maquette au plus près de la vraie application (D-6) : prototype interactif publié en artefact, tous les réglages, répliques comprises,
      compromis préréglages et réglage fin ; ligne dans l artefact des vérifications.
- [ ] M.3 Validation du user (citation consignée ici). Une remarque n'est pas une validation.
- [ ] M.4 Implémentation conforme à la maquette, après la fusion de C. Migration des préférences existantes.
- **Gate M :** gates web, grille de conformité à la maquette, vérification EN LECTURE.

### Piste R — répliques des Spartans (après M.3 et D-8)

- [ ] R.1 Table des textes des répliques par langue (commande de table), sans voix.
- [ ] R.2 Affichage dans le rejeu selon la maquette publiée et les choix D-8 ; réglages rangés dans le panneau
      de M.
- [ ] R.3 Voix : selon D-8 ; les voix anglaises restent à recevoir, hors de ce plan si elles n'arrivent pas.
- **Gate R :** gates Go et web, vérification EN LECTURE sur un rejeu en 1.12 ou 1.13 (répliques nommées),
  ligne dans l'artefact des vérifications.

## 6. Clôture

- [ ] Z.1 Suite web complète une fois (`npx vitest run`), `go test ./...`, `go test -tags=integration ./...`.
- [ ] Z.2 Revue finale Fable du diff cumulé (skills `adversarial-review` et `levelup-revue`) ; P0 et P1
      corrigés, P2 au backlog.
- [ ] Z.3 Skills `delivery-checklist` et `levelup-livraison` ; CI de la release lue.
- [ ] Z.4 Backlog : entrées traitées passées en « Récemment complété » ; reports au registre ; journal ;
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

(vide)

## 10. Journal d'exécution

- 2026-11-24 — plan écrit ; go du user le même jour ; pièces montées (piste F) et « mètres » (0.6, B.6) ajoutés à sa demande.
- 2026-11-24 — étape 1 : le report de `main` donne 20 fichiers en conflit (code de synchronisation, contrat,
  CI, notes de version). Résolution confiée à un agent Sonnet dans le worktree `LevelUp-wt-bl24-report`,
  branche `chore/report-main-8-2-3`. L'étape 0 et M.1, en lecture seule, partent en même temps : les entrées
  dont les fichiers sont touchés par le report (Halo 5, synchronisation) seront relues après lui.
