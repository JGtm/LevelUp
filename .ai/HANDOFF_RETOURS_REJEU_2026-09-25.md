# Handoff — campagne « retours rejeu » (2026-09-23 -> 2026-09-25)

> Écrit le 2026-09-25 à la demande de l'utilisateur, en fin d'une session trop longue. À lire en
> premier par la session qui reprend. Plan détaillé (décisions, lots, découvertes, journal) :
> `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md`.

## 1. Où on en est

- Les correctifs des 9 points signalés le 23/09 sont **fusionnés dans `feat/v75`** (merge `569932b42`,
  tête `3cca6cf47`, poussée). CI de `feat/v75` **verte sur tous les jobs**, E2E Playwright compris
  (run 36171856757). `main` n'est pas touché.
- Les **111 matchs locaux** qui avaient un rejeu sont **republiés au schéma 71** (serveur local non lancé
  pendant l'opération). Commande : `levelup backfill-replay -only-existing` (sans `-only-existing`, elle
  vise les 1 624 films du cache), puis `backfill-usage-summary`, `backfill-pad-tiers --force`,
  `tactical-rasters --backfill`.
- Worktrees et branches de la campagne : **remis à la session de ménage** (levelup-go-migration-b1). Ne
  rien supprimer d'ici.
- Rien n'est à faire en prod dans cette campagne ; la mise en prod passe par la fusion v7.5 -> `main`
  (geste de l'utilisateur).

## 2. Ce que l'utilisateur doit regarder (serveur local : `make dev`)

- **81c02726** : Ghost de G MONEY — éclair rouge et son à chaque rafale (6 frags sur 6 précédés d'une
  rafale).
- Matchs signalés le 23/09 (points 2 à 4 et 6) : le Mongoose de 0:06 ne part plus seul, Madina97294 ne
  « vole » plus à sa première réapparition, la fiche de XxDaemonGamerxX à 2:08 porte ses armes de
  naissance, « Sprint » n'apparaît plus sur les fiches.
- **b1ad85eb** : 4 contre 4 à chaque instant, un partant libère sa place pour son remplaçant.
- **ab526724** : drapeau visible, score et frise du score présents, véhicules de décor masqués.
- Page **Tactique** : le fond ne se recharge plus quand on change de question ou de filtre.
- **Tiroir des assets** : plus de cartes en double.
- Les **sons** ont été validés le 24/09 (lance-grenades du Falcon = sons du Rockethog, LMG et missiles
  du Wasp). Seule nouveauté : le son d'une arme à tir continu est tenu du début à la fin de la rafale
  au lieu d'un coup par tir. Pas de nouvelle validation demandée, juste à entendre en passant.

## 3. Ce qui reste ouvert — expliqué

### 3.1 Falcon de Behemoth — CORRECTION À FAIRE (fait de l'utilisateur, 25/09)

Message de l'utilisateur du 25/09 : **« Il n'y a pas de Falcon jouable sur Behemoth. »**
Origine : le 24/09, l'utilisateur a dit « les Pelican sont toujours du décor, le Falcon ça dépend ». Le
lot M7b a donc retiré le Falcon de la liste « toujours décor » et l'a soumis à la règle générale du
décor (masqué seulement s'il est hors de la zone jouable). Effet de bord : sur Behemoth en Super Fiesta
(match `1cd3848a`), deux Falcon garés DANS la zone jouable, jamais occupés, sont désormais AFFICHÉS.
C'est faux. À faire : une règle générale (pas un cas par carte) qui masque ces Falcon sans masquer les
Falcon réellement pilotés (Launch Site en Super Fiesta, BTB). Mesure et témoins au §4 « M7b » et §8 du
plan.

### 3.2 La troisième montée de G MONEY dans un Ghost (81c02726, vers 4:55-5:12) n'est pas affichée

Ses rafales sont lues, mais on ne voit pas qu'il monte dans le Ghost, donc elles ne sont pas dessinées.
Cause prouvée : juste avant, un objet de la famille « dispositifs » (type 43 dans le film) apparaît, et
le décodeur ne sait pas lire trois de ses blocs de données (appelés i20, i21, i22). La lecture déraille
à cet endroit et perd les paquets suivants, dont celui qui annonce l'embarquement. Réparer = apprendre
au décodeur ces trois blocs (travail de rétro-ingénierie, Ghidra). Décision de l'utilisateur à prendre :
le faire ou non.

### 3.3 Armes à l'apparition : encore des vies sans relevé

C'est le point 4 du 23/09 (fiche sans armes). Le lot M3 lit maintenant les armes données à la naissance.
Mesure sur 22 films : la part des vies dont on ne connaît pas les armes au départ passe de 18,7 % à
10,2 %. Sur les films des versions récentes du jeu (1.12 et 1.13), il reste 1,3 % ; le reste vient des
films de versions plus anciennes, dont les naissances ne se lisent pas encore. L'objectif du plan était
≤ 1 %. Décision de l'utilisateur : accepter, ou ouvrir un lot pour les anciennes versions.

### 3.4 `turretRidesNotRideable` — tâche technique, pas une question

Compteur interne du document de rejeu : « montées sur une tourelle dont le véhicule porteur n'est pas
pilotable ». Il servait au Falcon quand il était classé non jouable ; depuis M7b il vaut toujours 0.
À supprimer à la prochaine montée de schéma (code mort). Aucune décision utilisateur nécessaire.

### 3.5 Données de frags en base (backfill `killsource`) — décision de l'utilisateur

Le décodeur corrigé lit un peu mieux certains frags (8 au lieu de 7 sur un film de test). Les positions
et armes des frags déjà enregistrées en base (utilisées hors rejeu : cartes, armes par frag) viennent
de l'ancien décodeur. Les recalculer = `levelup backfill-killsource --workers 1` sur les 1 624 films du
cache, environ 2 h 45 de machine, serveur arrêté. Non lancé : à faire seulement sur accord.

### 3.6 Renvoyés ailleurs (aucune action ici)

- Ghost de l'index 4 sur Launch Site (8a485699) : une seule rafale lue au lieu d'une douzaine, à cause
  des objets physiques de la carte mal lus. Pris en charge par le jalon J6 de
  `.ai/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md` (autre session).
- Vies sans nom : 23 vies sur 10 matchs après la campagne, 27 avant (même mesure sur le code
  d'avant) : défaut antérieur, légèrement réduit. Plan §8.33.
- Autres découvertes consignées, non traitées : plan §8 (points 27 à 33).

## 4. Leçons de la session

- La campagne a épuisé le quota hebdomadaire de l'utilisateur (67 agents Opus, exécutants de 3 à 6 h).
  Avant tout workflow : annoncer le coût et demander ; effort « high » au plus ; périmètre strictement
  fermé (demande de l'utilisateur du 25/09).
- Ne poser à l'utilisateur que des questions qu'il peut trancher, expliquées en clair ; les tâches
  techniques (comme 3.4) se notent, elles ne se demandent pas.
