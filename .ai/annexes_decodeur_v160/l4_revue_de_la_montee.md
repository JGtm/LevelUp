# Revue de la montée du décodeur v1.6.0 (branche `chore/830-decodeur-v160`, `aab869c44e..HEAD`)

Lecture seule. Chaque constat est marqué « vérifié » (relu dans le code ou rejoué) ou « déduit ».
Chemins relatifs à la racine du dépôt.

## Constats recevables

### C1 (P1) : le nom d'un socle de grappin, de répulseur ou de propulseur reste sa clé brute

- Où : `apps/web/src/features/match-replay/layers/useReplayWeaponPads.ts:145` (`padNameFor`), qui délègue à
  `apps/web/src/features/match-replay/model/equipmentUsageColumns.ts:37-44` (`equipmentFamilyLabel`).
- Condition : un socle dont `weaponPads[].weapon` vaut `grapple`, `repulsor` ou `thruster`. Dans
  `layers/equipmentPlacementsLayer.ts:159-161`, `PLACEMENT_RENDER` vaut `null` pour ces trois familles (elles ne se
  posent pas), donc `equipmentFamilyLabel` rend la clé telle quelle.
- Conséquence : l'infobulle du socle affiche « grapple », « repulsor » ou « thruster », en français comme en anglais.
  Les notes de version (`docs/CHANGELOG.md:42`, `docs/FR/CHANGELOG.md:42`) annoncent pourtant qu'un socle
  d'équipement est nommé au lieu d'afficher sa clé.
- Mesure (vérifié) : dans les quatre fixtures du schéma 127 du dépôt, 18 socles d'équipement sur 29 hors bonus sont
  dans ce cas (grapple 9, repulsor 8, thruster 1). Les familles nommées sont wall 3, shroud_screen 7,
  translocator_beacon 1.
- Test : `useReplayWeaponPads.test.ts` ne couvre que `wall` et `sensor`. Aucun test ne rougit sur ces trois familles.

### C2 (P2) : un socle d'équipement est traité comme un râtelier d'arme (encre, son)

Même cause : `padEquipmentFamilyOf` (`model/weaponPadFamilies.ts`) ne connaît que `powerup_overshield` et
`powerup_camo`, et c'est le seul test « est-ce un équipement » des lecteurs de socle.

- `apps/web/src/features/match-replay/sound/padSpawnSound.ts:77` | socle de famille `wall`, `grapple`, `repulsor`,
  `shroud_screen`, etc. | chaque apparition joue le son d'apparition d'une ARME sur socle, pas celui de l'équipement
  (vérifié par lecture).
- `apps/web/src/features/match-replay/layers/useReplayWeaponPads.ts:372` avec `model/weaponPadFamilies.ts`
  (`padFamilyOf`) | même socle | le socle prend l'encre « classic » des râteliers d'arme, pas celle des équipements
  (vérifié par lecture ; rendu à l'écran non contrôlé).

### C3 (P2) : aucun test ne tient la chaîne du nombre de fils au-delà de ses maillons isolés

Le code est juste aujourd'hui (voir les conditions vérifiées). Ces lignes peuvent disparaître sans qu'un test rougisse :

- `apps/go-api/internal/sync/replayartifacts/buildone.go:179` (recopie de `DecoderThreads` vers la requête de
  l'enfant) : sans elle, toute génération après synchronisation, par action d'administration et par la file du bureau
  retombe à un fil. `buildone_fils_test.go` s'arrête à une fausse fonction de construction.
- `apps/go-api/internal/replaychild/replaychild.go:139` (`AvecFils` dans l'enfant) : `fils_test.go` ne teste que
  l'aller-retour JSON.
- `apps/go-api/internal/api/wire/registry_replay_build.go:72` (action d'administration et file de régénération du
  bureau) et `apps/go-api/internal/sync/replayartifacts/artifacts.go:148` (`NewHook`) : aucun test ne lit la valeur
  posée.
- Conséquence : le bureau reviendrait à un fil en silence, la sortie étant identique. Vérifié : `DecoderThreads`,
  `AvecFils` et `avecFils` n'apparaissent que dans quatre fichiers de test, aucun sur ces lignes.

### C4 (P2) : la table des formes déployées n'a pas de test en production

- `apps/go-api/internal/games/halo_infinite/replaylabels/catalog.go:76` et
  `apps/go-api/internal/games/mappings/loader_replay_labels_equipment.go:240-254` (`parseDeployedEquipment`).
- Condition : retirer la ligne de `catalog.go`, ou inverser le filtre `kind != deployed`.
- Conséquence : aucun test ne rougit. `filmcorpus/catalogue_titre_test.go:95` pose la table lui-même sans passer par
  `replaylabels.Load`. Le décodeur recevrait une table vide et deux murs posés au même endroit pourraient devenir un
  faux socle. Vérifié : aucun autre test ne cite `DeployedEquipment`.

### C5 (P2) : commentaire devenu faux dans un test de conformité

- `apps/go-api/internal/games/halo_infinite/filmprofile/conformite_table21_test.go:73`.
- Condition : lecture. Le compte passe de 9 à 17, le commentaire dit toujours « trois dispositions d'écriture par
  build, sur trois builds » (soit 9).
- Conséquence : le commentaire contredit la valeur qu'il justifie (règle 17 du `CLAUDE.md`). Le compte lui-même est
  légitime : huit lignes ajoutées au catalogue `film_profiles.json`, test vert.

### C6 (P2) : aucune entrée au journal pour ce lot

- `.ai/thought_log.md` : `git diff --stat aab869c44e..HEAD -- .ai` est vide.
- Conséquence : la règle « THOUGHT LOG, règle obligatoire » du `CLAUDE.md` n'est pas tenue sur ces 11 commits.

## Observation non recevable (pas de déclencheur aujourd'hui)

`archlint/film_slog_contexte_test.go:73` et `archlint/film_tri_total_test.go:67` excluent tout `libs/film/cmd`, alors
que la seule commande de la brique est `cmd/film-bench` (vérifié dans le module v1.6.0 ; la v1.5.0 n'avait pas de
`cmd/`). L'exception est plus large que nécessaire, mais rien ne la déclenche à ce jour. Les entrées de `childproc`,
de la table des couches et du suffixe d'OS sont étroites (fichier ou dossier `film-bench`) et leur justification est
vraie.

## Conditions vérifiées qui tiennent : 19

1. Zéro fil demandé vaut un fil dans la brique (`replay/options_vitesse.go`, `poserLesFils`).
2. Une seule construction des options du décodeur en production (`replaybuild/options.go`), qui porte `Vitesse`.
3. Après synchronisation : les trois constructions du crochet passent par `NewHook`, donc par
   `SpawnBuildOneAvecFils` (`cmd/server`, `scheduler`, `service/sync_jobs_engine`).
4. Action d'administration et file de régénération du bureau : même fonction `RunReplayBuild`, valeur posée.
5. Commandes : `backfill-replay` (enfant) et `replay-parc-vies` lisent la capacité ; `replay-build` et
   `replay-worker` prennent `-fils`, un par défaut, comme écrit dans les notes de version.
6. Démo, contrôle d'équivalence : aucun réglage, donc un fil, conforme à la décision.
7. Le site hébergé et la démo donnent 1, le bureau et le développement 4 (test rejoué, vert). La valeur est dérivée
   dans `internal/config` ; aucun consommateur ne lit le profil.
8. La capacité n'est pas exposée au web (pas de donnée servie sans lecteur).
9. Plafond mémoire de l'enfant : 3 Gio, pic annoncé de 1,7 à 1,9 Gio (déduit du commentaire, non mesuré ici).
10. Lignes de bot : les deux encodages de l'API passent par `FromArtifact`, donc par le filtre. Inverser ou retirer
    le filtre fait rougir `TestProjectionCopieChaqueChamp` (classes peuplée et à feuilles zéro ; déduit de la
    structure du test, test vert rejoué).
11. Contrat inchangé : `openapi.yaml` et les types générés ne bougent pas, `xuid` reste obligatoire côté servi.
12. Web, tueur sans identifiant : écarté des deux côtés de la jointure, test rejoué vert.
13. Dégât à zéro : le seul lecteur (`playerInstantEstimates.ts:257`) somme des points, ne compte pas de lignes ; le
    test neuf rougirait si une arme sans tir gagnait une ligne. Aucun lecteur Go de `damage` hors projection.
14. Score qui recule : tous les lecteurs web lisent le palier courant (`scoreAtFrame`), aucun maximum ni contrainte
    de signe au schéma ; côté Go, `Finale()` lit la dernière valeur.
15. Socles d'équipement, côté Go : les prises par joueur (résumé d'usage, paliers d'arme) ne comptent que les socles
    d'arme, car la brique ne nomme un ramasseur que sur une famille hexadécimale ; les lecteurs de bonus n'itèrent que
    camouflage et surbouclier.
16. Calques retirés au chargement web : liste tenue égale à la décision Go par
    `TestCalquesNonServisDuWebSontCeuxDeGo` (vert) ; aucun des huit calques n'est lu dans `apps/web/src`.
17. Garde-rails modifiés : les tests visés d'`archlint` passent ; le plafond de façade 339 correspond à un symbole
    ajouté et lu (`replay.OptionsVitesse`).
18. Attendu changé de 0 à 2 dans `pont_siege_recycle_test.go` : couvert par la décision
    `coverage.objectives.noSlot` du manifeste, acceptée.
19. Dépôt public : les identifiants des fixtures du schéma 127 sont les mêmes que ceux du schéma 126 déjà versionné
    (comparé sur un film), les noms sont neutres ; le diff n'en introduit pas.

## Non vérifié

- Rendu à l'écran (aucun navigateur lancé).
- Résolution du profil pour une commande lancée sur le site hébergé (déduit : `LEVELUP_ENV` déclaré donne un fil ;
  à confirmer par l'environnement réel du conteneur).
- Les 7 tests d'`archlint` et 2 de `filmcorpus` annoncés rouges, et l'équivalence sur 11 films : dette assumée, non
  rejouée.

**En bref**
Aucun défaut de justesse des données ni du contrat de l'API : pas de total faux, pas de ligne rangée sous le mauvais
joueur, le nombre de fils arrive au décodeur sur chaque chemin.
Un défaut visible : l'infobulle d'un socle de grappin, de répulseur ou de propulseur affiche sa clé brute (18 socles
d'équipement sur 29 dans les fixtures), contrairement aux notes de version.
Cinq constats mineurs : son et encre d'arme sur un socle d'équipement, deux trous de test, un commentaire faux, le
journal absent.

Décisions attendues :
- **Corriger C1 avant la fusion, ou corriger la phrase des notes de version** : la promesse écrite n'est tenue que
  pour une minorité des socles d'équipement.
- **Dire si C2 se corrige dans ce lot ou va au backlog** : même cause que C1, effet plus discret.
- **Dire si C3 et C4 (tests manquants) se traitent ici** : le code est juste, mais rien ne le garde.
