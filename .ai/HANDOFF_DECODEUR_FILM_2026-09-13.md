# Handoff — décodeur de film Theater : recherche des 12 et 13 septembre 2026, close

> Écrit le 2026-09-13 à la clôture de cinq phases de rétro-ingénierie lancées à partir du
> document `.ai/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md`. La recherche est CLOSE
> (décision utilisateur) ; la suite est un PLAN (skill `plan-execution`). Ce handoff dit où est
> le travail, ce qui est prouvé, ce qui reste ouvert, et les trois correctifs de production que
> le plan doit absorber. Tout ce qui suit est vérifiable par les commandes des notes.

## 1. Où est le travail

| Quoi | Où | État |
|---|---|---|
| Recherche (5 phases, 21 instruments, 6 notes, relevé Ghidra 8 à 8.6) | branche `wt/section3-chunk00`, worktree `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-section3-chunk00`, tête `654a8f55f` (fusion de `wt/film-residus`) | committé, NON poussé, NON fusionné dans `feat/v75` |
| Branche satellite de la phase 5b | `wt/film-residus`, worktree `LevelUp-wt-film-residus` | fusionnée dans `wt/section3-chunk00` ; branche et worktree à supprimer après vérification |
| Document d'architecture cible, amendé au fil des phases (sections 5 bis et 5 bis.1) | `feat/v75`, `.ai/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md`, tête `0053b6634` | committé, NON poussé |
| Journal | `.ai/thought_log.md` : entrée `[2026-09-12] Architecture cible du décodeur de film` + addendums 1 à 5b sur `feat/v75` ; six entrées détaillées par phase sur `wt/section3-chunk00` | à jour |
| Notes de RE | `.ai/V7.5/film_re/NOTE_SECTION3_CHUNK00_2026-09-12.md`, `NOTE_SECTION3_SLOTS_2026-09-12.md`, `NOTE_EQUIPE_FILM_2026-09-12.md`, `NOTE_PROFIL_PAR_BUILD_2026-09-12.md`, `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`, `NOTE_RESIDUS_CHUNK00_2026-09-13.md` (branche de recherche) | chacune avec résumé numéroté, prouvé / hypothèse / réfuté, commandes de rejeu |

Aucun code de production n'a été modifié. Tous les instruments sont des `*_research_test.go`
sous la garde d'environnement `CHUNK00_FILMS` (liste de répertoires de film séparés par `;`,
chemins Windows `C:/...` obligatoires), sautés en CI. Gates verts sur la branche fusionnée :
`gofmt`, `go vet`, `go test ./internal/analysis/filmdec/`, `go test ./internal/archlint/`
(ratchet des variables de paquet intact : aucune ajoutée).

Pièges d'outillage rencontrés : Ghidra est une instance PARTAGÉE (`HaloInfinite.exe`, base
`0x140000000`, MCP `mcp__ghidra__*` ou HTTP `127.0.0.1:8089`), toujours en lecture seule ; la
shared de production peut être tenue par un autre processus, l'oracle se lit sur la sauvegarde
`data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` en `access_mode=read_only` via
`go run ./cmd/diag_q` ; le cache de films (`data/cache/film_chunks`, 1 351 films, 7 builds de
`HI_1_4_1` à `HI_1_13_0`) ne vit que dans `LevelUp-go-migration`, pas dans les worktrees.

## 2. Ce qui est prouvé (deux chaînes sans étape commune : exécutable + films)

1. **`chunk_00` est un flux tassé au bit, décalé d'UN bit par un booléen à `0x0CB45C`.** Quatre
   sections : registre (à l'octet 8, pas 0), en-tête (table de 123 u32 par type, build en clair,
   identifiant de build et changelist), corps, zéros. Écrivain `FUN_14299b198` / `FUN_14299b278`.
2. **Le film porte la table des joueurs du match** : un enregistrement par slot (32 max), XUID
   en clair à 85 bits, gamertag à `sub+0xc14`, 16 champs fermés ; l'ordre des enregistrements
   est le `player_index` de production. Les écarts « aberrants » sont des slots VACANTS. Lecteur
   par grammaire calibré sur le film : 32 slots sur 1 351 films sur 1 351, sept builds.
3. **La transposition par build de la table des slots est portée par le seul bloc de
   personnalisation** (`sub+0xcc0` : 1 852 / 1 492 / 1 312 / 2 052 octets selon le build), qui
   est RÉSERVÉ mais toujours à zéro : le film ne porte pas les tenues (hypothèse utilisateur
   réfutée, 0 octet non nul sur 81 488).
4. **L'équipe d'un joueur est dans la trame d'état**, pas dans `chunk_00` : composant
   `managed-player-team-designator-component`, archétype ti=9, composant i0, 4 bits à 186 bits
   du début du record d'image-clé, valeur = désignateur + 1 ; `team_id 0` = désignateur 0 =
   `First`. 16/18 puis 9/10 films en accord avec la base, six Grande bataille à 24/24, FFA à 0.
   Jamais plus de deux désignateurs, même en Grande bataille (595 matchs).
5. **186 se DÉRIVE** : `108 + 32 + largeurEtatParDefaut(ti) + 32`, soit `172 + état(ti)`,
   identique sur les 7 builds. La table d'image-clé est lue par le lecteur d'ÉTAT COMPLET
   `FUN_142e2bfd0`, sans masque de présence. Deux mots de taille (`n1`, `n2`) ferment
   gratuitement ; `n2` est un détecteur de largeur d'état par défaut fausse.
6. **Le pied de film porte l'équipe d'un événement à l'octet 37** (665/665 sur 14 films) ;
   l'octet 55 que la production lit vaut 0 partout.
7. **L'horodatage du match est dans le film** (32 bits à `0x0CB65C × 8 + 1`).

## 3. Les trois correctifs de production que le plan doit absorber (aucun fait)

Par ordre de coût croissant, chacun = changement de comportement, donc corpus gate +
équivalence + révision de couche (règles du document d'architecture, section 7) :

| # | Correctif | Où | Preuve d'appui | Coût |
|---|---|---|---|---|
| 1 | Lire l'équipe d'un événement d'objectif à l'octet 37 du bloc de 60 o du pied, pas à l'octet 55 | `objectiveevents/film.go` (`teamRaw`) | 665/665, `residus_pied_research_test.go` | une ligne + ratchet + gate |
| 2 | Ajouter les cinq états par défaut manquants (ti=14 → 6 bits `V ; R(5)`, 17 → 8 `V ; R(7)`, 21 → 18 `R(18)`, 29 → 1 `V`, 47 → 6 `V ; R(5)`) et corriger le commentaire STUB de ti=14 | `filmdec/default_state_arch.go`, `defaultStateDeserByTI` | `imagecle_oracle_n2_research_test.go`, décompilé (relevé 8.5) | trois entrées ; +10 541 records fermés |
| 3 | Remplacer le modèle de record d'image-clé (en-tête 64 bits + masque, `keyframe_record_walk.go` + `TraverseEntity`) par la boucle d'état complet déjà écrite (`keyframe_fullstate_loop.go`, `WalkKeyframeFullState`) dans les deux consommateurs (`navpoint_radial_scan.go`, `objective_scan.go`) | `filmdec/` | 0/62 686 contre 8 796/62 686, `imagecle_fermeture_research_test.go` | moyen ; voir section 4 |

À y ajouter, issus du document d'architecture : la correction du lecteur de registre (octet 8),
et la lecture de la table des slots et du désignateur d'équipe dans `facts` pour publier
l'équipe réelle et le roster SANS ouvrir de base (aujourd'hui `replay` écrit `Team: -1` en dur
et le rejeu hors ligne perd l'équipe : origine des « sans équipe » impossibles).

## 4. Le problème du record d'image-clé, expliqué

Le film écrit un instantané d'état complet (« image-clé ») toutes les vingt secondes environ,
un record par entité. La production le lit comme un record NEW de la trame delta : en-tête de
64 bits, état par défaut de l'archétype, un bit de porte, puis un MASQUE DE PRÉSENCE qui dit
quels composants sont présents, puis ces composants. Le jeu, lui, lit l'image-clé avec un autre
lecteur (`FUN_142e2bfd0`) : en-tête de 108 bits, un mot de taille, l'état par défaut, un second
mot de taille, puis TOUS les composants dans l'ordre du registre, sans masque. C'est de
l'arithmétique : pour l'archétype joueur, la borne du modèle de production est 151 bits, le
premier composant est à 186 ; aucune donnée ne comble 35 bits.

Conséquence mesurée sur 62 686 records (6 films, 3 builds) : la production ne ferme AUCUN
record (le début du record suivant n'est jamais là où elle le prédit). Elle ne déraille pas
bruyamment : 5,5 % des records désynchronisent, les autres se lisent « jusqu'au bout » et
atterrissent au mauvais bit, sans qu'aucun compteur de production le voie. Le masque relu à la
mauvaise position dit « zéro composant présent », ce qui rend un record de 459 bits vide. C'est
la cause de fond des déraillements des lots R3, R4 et R5 (« le désérialiseur du corps d'un
record d'image-clé n'est résolu nulle part »), et de la borne d'arrêt décidée après R7-e.

Pourquoi personne ne l'a vu : le seul chemin de production qui lit un corps d'image-clé est
l'armement de la bombe en Assaut (`replay/bomb_armings.go` → `ScanNavpointRadial`), qui ne
s'engage pas sur le corpus courant. Le correctif ne répare donc la sortie d'aucune page
aujourd'hui : il OUVRE une voie (l'état complet de toutes les entités toutes les vingt
secondes : positions, équipes, objets du monde, véhicules, armes au sol, sans dépendre du delta),
il n'en répare pas une.

Les pistes, dans l'ordre où elles se prouvent :

1. **Le cadre** : la bonne forme existe déjà dans le dépôt depuis R7-d
   (`keyframe_fullstate_loop.go`, jamais branchée). La brancher à la place de `TraverseEntity`
   dans les deux consommateurs fait passer la fermeture de 0 à 14,0 %, huit archétypes gagnent,
   aucun ne régresse, quatre ferment à 100 % (dont le statborg, ti=6).
2. **Les états par défaut** : `n2` (le second mot de taille, lu APRÈS l'état par défaut) est
   constant si la largeur de l'état par défaut est juste et dispersé sinon. Il a désigné cinq
   largeurs manquantes, confirmées au désassemblage ; trois d'entre elles font fermer 10 541
   records de plus (projection 30,8 %). `n2` devient un test permanent : toute largeur d'état
   par défaut fausse rougit.
3. **Les largeurs de composants** : le reliquat (61,8 % des records) marche jusqu'au bout mais
   n'atterrit pas : cadre juste, une ou plusieurs largeurs de composant fausses. Un composant
   inconnu bloque tout l'archétype (ti=9 bute sur `i4 managed-player-forge-weather-effect-
   overrides-component`). C'est « le mur » des notes de RE : les largeurs se prennent chez le
   lecteur de chaque composant dans l'exe (descripteurs à vtable), archétype par archétype, en
   commençant par ceux que la production consomme (ti=9 joueur, ti=11/12 objectifs, ti=35 bipède,
   ti=42/43 armes). L'oracle de fermeture (le record suivant tombe où la grammaire le prédit)
   suffit pour prouver chaque largeur sans capture live.
4. **Le gate** : un compteur « records d'image-clé fermés / total » par archétype, publié en
   expvar et gelé par un ratchet, pour que le cadre ne puisse plus casser en silence.

### 4 bis. Combler les 62 % : la méthode, composant par composant

Le trou n'est pas un mystère, c'est un inventaire. Les faits mesurés qui le bornent :

- Le registre de `chunk_00` nomme 264 composants distincts ; `filmdec` porte aujourd'hui un
  lecteur pour 103 d'entre eux (mesure `grep case "...-component"`, 2026-09-13). Les 161 autres
  n'ont pas de lecteur, et c'est tout ce qui manque : le cadre est juste, l'ordre est connu.
- Dans une image-clé il n'y a pas de masque : TOUS les composants de l'archétype sont présents,
  dans l'ordre du registre. Un seul composant sans lecteur bloque donc tout ce qui le suit. La
  note 5a nomme le bloquant par archétype (ti=9 bute sur `i4
  managed-player-forge-weather-effect-overrides-component`, 1 679 fois sur 1 679 ; neuf
  archétypes butent sur leur premier ou deuxième composant). Le nombre de composants à porter
  pour débloquer les archétypes utiles est donc bien plus petit que 161 : à inventorier au
  premier pas.
- Le lot R7-d a établi la méthode pour trouver la grammaire d'un composant dans l'exécutable :
  chaque descripteur de composant a une vtable de 10 cases dont exactement deux touchent le flux,
  `+0x18` ÉCRIT et son miroir LIT ; le descripteur se retrouve par dump de `.rdata`
  (`0x143606000..0x144395200`, `read_memory`) sans xref. Quatre ports Go sur cinq y ont été
  confirmés largeur pour largeur : les 103 lecteurs existants (écrits pour le delta) valent pour
  l'image-clé, puisque l'écrivain d'image-clé est le miroir exact du lecteur delta (i60 en R7-d).
- L'oracle ne coûte rien et ne demande aucune capture : la FERMETURE (le record suivant tombe
  exactement où la grammaire le prédit, `imagecle_fermeture_research_test.go`), le mot `n2`
  pour l'état par défaut, et pour un champ de largeur constante le profil de bascule des bits en
  dent de scie (méthode de `i0_layout.go`) comme seconde chaîne.

Le pas, répétable jusqu'à ce que les archétypes utiles ferment à 100 % :

1. **Inventaire** : un instrument qui, par archétype, imprime la liste ordonnée des composants
   avec leur statut (porté / manquant / BLOQUANT = premier manquant) et le taux de fermeture.
   Il devient le ratchet « la couverture d'image-clé par archétype ne descend jamais ».
2. **Priorité** : les archétypes que le rejeu et killsource consomment, dans cet ordre : ti=9
   joueur (un seul bloquant, `i4`), ti=11 et ti=12 objectifs (100 % de désync aujourd'hui),
   ti=35 bipède (52 composants, la majorité déjà portée), ti=42 et ti=43 armes, puis véhicules.
3. **Par composant bloquant** : descripteur par `.rdata`, décompilation de l'écrivain `+0x18`,
   port en fonction pure (profil, bits) dans `filmdec` ; une largeur qui dépend d'une config du
   jeu (comme le quaternion de véhicule de ti=40) devient une entrée de profil, pas une
   constante. Preuve : fermeture de l'archétype avant / après, sur 6 films de 3 builds.
4. **Répéter** jusqu'au prochain bloquant. Beaucoup de composants sont partagés entre
   archétypes : chaque port en débloque souvent plusieurs.

Ordre de grandeur honnête : quelques dizaines de composants pour les six archétypes utiles, pas
161 ; chaque composant est une session de désassemblage courte, la méthode et l'oracle existent.
Ce qui n'est PAS à faire : re-mesurer par statistique ce qu'un écrivain lisible dit en clair.

## 5. Ce qui reste ouvert (recherche), par ordre de valeur

- Le reliquat de 61,8 % des records d'image-clé (largeurs de composants) ; ti=13 muet malgré une
  grammaire bit-exacte ; ti=4 dépendant du build.
- `+200` octets sur `HI_1_4_1` et le build des 5 films sans section d'identification.
- Le libellé affiché d'un désignateur (exige le corpus `gamefiles`) ; le rôle de `b36`, `b40` et
  des octets 39 à 43 du pied ; le rôle des trois listes préfixées de l'enregistrement de slot
  (702 octets, 167 mots, masque de 2 048 bits) ; le jeton de 48 bits.
- Le décalage de 8 octets de `parseRegistry` : CONNU, NON TRANCHÉ, à corriger comme pas à part.
- La sémantique des valeurs 1..6 de la table par type de `chunk_00` (appel virtuel
  `vtable+0x30` : version de sérialisation par type, non prouvé).

## 5 bis. Séquence recommandée : RE ou révision du décodeur d'abord ? (question du 2026-09-13)

Ni l'un avant l'autre en bloc. Trois temps, chacun justifié par le risque :

1. **D'abord les correctifs courts, gate-ables un par un, sans attendre la révision** : l'octet
   37 du pied, les cinq états par défaut manquants, le cadre d'image-clé branché dans ses deux
   consommateurs, le lecteur de registre à l'octet 8, et la lecture de la table des slots et du
   désignateur d'équipe dans le constructeur de rejeu (l'équipe réelle sans base). Chacun tient
   en une branche courte sous corpus gate et équivalence ; aucun n'a besoin du profil ni des
   couches. C'est la valeur visible la plus proche (les « sans équipe », le compact BTB).
2. **Ensuite la révision, pas 0 à 7, à zéro différence** : elle crée le profil (là où les
   largeurs dépendantes du build ou de la config doivent vivre), retire les globales et le
   verrou, pose les empreintes, les goldens par build et le budget de temps. Elle est
   mécanique et interruptible ; sa fenêtre exige qu'aucune branche filmdec ne soit en vol,
   donc pas de port de composant pendant qu'elle court.
3. **Enfin la RE des 62 %, comme charge utile du pas 8, dans la structure révisée** : chaque
   composant porté est une fonction pure (profil, bits) dans `grammar`, une révision de couche,
   un gain attendu au corpus gate. Porter 161 composants AVANT la révision reviendrait à
   grossir les fichiers de 1 380 lignes, à écrire les largeurs dépendantes de la config dans
   des globales ou des littéraux (ce que le ratchet interdit déjà, et ce que le document
   diagnostique comme la maladie), et à faire re-mesurer ensuite « zéro différence » sur un
   décodeur qui bouge.

Ce qui peut courir en parallèle sans gêner : la RE en INSTRUMENTS (tests de recherche, notes,
relevé Ghidra), qui ne touche pas la production ; elle nourrit le pas 8 et doit simplement être
fusionnée ou rebasée avant le pas 5 (déplacement des paquets). L'inventaire des composants et
le ratchet de couverture d'image-clé (section 4 bis, étape 1) se posent au pas 0 : c'est un
gate, pas un port.

## 6. Prochaine étape

Écrire le plan du chantier décodeur (skill `plan-review` puis `plan-execution`) à partir du
document d'architecture, en y intégrant : le pas 0 (fixtures de contrat, goldens par build,
budget de temps), la correction du lecteur de registre, les trois correctifs de la section 3, et
la première table de profil par build (section B.1 de `NOTE_PROFIL_PAR_BUILD_2026-09-12.md`).
Avant cela : fusionner `wt/section3-chunk00` dans `feat/v75` (recherche pure, sans risque de
comportement), supprimer `wt/film-residus` et son worktree, pousser après accord de
l'utilisateur. Un ADR 0034 doit porter les invariants durables (couches, profil, porte unique aux
octets, politique de version inconnue).
