# HANDOFF — Composants où la marche depuis la fin de la vue A bute (2026-10-06)

> **Pour qui** : un agent frais (Opus) chargé d'un lot de grammaire, et le pilote qui le lance.
> **Origine** : clôture de la campagne de grammaire (`.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`
> §7), décision de l'utilisateur du 2026-10-06 « Non, grammaire d'abord » : quand la marche d'une
> trame partie de la fin de la vue A (E) bute, la cure est de LIRE le composant qui l'arrête, pas de
> reprendre plus loin à une signature (convention refusée). Ligne correspondante au registre
> `.ai/REGISTRE_REPORTS.md`.
> **Statut** : prêt, NON lancé. Le lancement est un geste de l'utilisateur.

## 1. Le problème, en une phrase

Depuis V2 de la vue A (`feat/v75` `2707fdb31`), la vue B d'un film récent commence là où la vue A
finit (loi du lecteur `FUN_142987460` → `FUN_14076a1c4`). Quand la marche bute sur un composant que
le décodeur ne sait pas lire, tout le reste du paquet tombe en queue opaque : aucun canal ne le lit.
Avant V2, une heuristique (signature du slot 123) démarrait parfois plus loin et rattrapait des
records ; elle est retirée parce qu'elle n'est pas la loi du jeu.

## 2. Ce qui est mesuré

### 2.1 Paquets sains perdus par V2, par composant d'arrêt (carte v2, 20 films)

Source : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/LOT_VA_V2.md` §5 et §5.1. 354 paquets sains
perdus en brut, tous de classe E (3 328 records utiles), contre +53 004 sains gagnés.

| Composant / arrêt | Paquets | Films principaux |
|---|---|---|
| `ti=43 i19 device-position-animation-name-component` | 139 | `396cfc92` 85, `f75e7053` 43, `4f77afc1` 11 |
| `ti=12 i16 managed-navpoint-override-flags` | 61 | `4f77afc1` 39, `f75e7053` 12 |
| vue C : terminateur hors cadre | 44 | `d9781168` 37 |
| vue B : sortie par rejet | 37 | `4f77afc1` 31 |
| `ti=10 i2 managed-object-navpoint-component` | 19 | `d9781168` 10, `51ebbc0f` 5 |
| `ti=45 i0 matchflow-sequence-data-component` | 19 | 10 films |
| vue C : bloc `0xbc` (désalignement) | 12 | `d9781168` 12 |
| `ti=12 i18 managed-navpoint-position-offset` | 7 | 5 films |
| `ti=43 i21 device-position-group-component` | 6 | `4f77afc1` |
| autres (`ti=10 i22`, `ti=35 i59`, `ti=10 i18`, `ti=12 i22`, `ti=43 i35`, `ti=43 i39`, `ti=45 i1`) | 10 | — |

Ce tableau ne compte que les paquets SAINS perdus. Les paquets qui ne fermaient déjà pas et dont la
queue a changé de place n'y sont pas : c'est la mesure 2.2 qui les voit.

### 2.2 Lectures des huit lecteurs de la RI (levelup-57, 20 films, avec et sans V2/V3)

**+806 records gagnés, −51 perdus.** Pertes :

- (a) 32 annonces d'emplacement vide (composant `i45`, Restated) dans trois paquets de mise en place
  des joueurs : `000d5950` 2:712, `696a9d7c` 2:1620, `9f57c612` 2:2130, huit slots chacun. La marche
  depuis E y bute avant la signature. Sans conséquence.
- (b) Quelques anciennes ancres démenties par une trame désormais FERMÉE depuis E (correct) :
  `084a804d` 12:314 slot 727, `d9781168` 35:2112 slot 557.
- (c) Environ 15 records isolés réellement perdus, dans des paquets que la marche depuis E ne mène pas
  au bout : `1c4c63c2` 9:2050 slot 696 (le plus chargé), `1c4c63c2` 57:2084 slot 735, `d9781168`
  10:1384 à 10:1504 slot 548 (`i30`, trois fois), `111fa685` 8:844 slot 616, `64e8adfa` 41:1400 slot
  642, `fb1a1a72` 18:1662 slot 565, `7344d24f` 3:1446 slot 515.
- Mini-bobine versionnée (`bobineFamilles`), paquet 2:712 : avant V2, début par signature au bit 7418
  puis les bipèdes 512 à 519 ; après V2, début lu à E = 5224, la marche lit 1508 (NEW `ti=11`), 2-12,
  52-59, puis bute sur le slot 122 (`ti=45`) à son composant 0, au bit ~7286.

La section 2.2 est tenue par levelup-57 : il la complète ou la corrige directement dans ce fichier.

## 3. Ce qui existe déjà (à lire AVANT Ghidra)

- **`ti=43` (dispositifs, 145 paquets avec `i21`, `i35`, `i39`)** : le lot L2 a porté `i19`..`i40`
  (+ `i18`) fidèlement au jeu, relu par un contrôle. Il a été RETIRÉ le 2026-10-03 parce que le gate 2
  « aucun film en baisse » tombait sur `1c4c63c2` (−447 sains en combinaison). La cause instruite
  (D-L2-12) : un faux en-tête NEW `ti=43` pris comme tête de liste au SECOND RANG de
  `debutParFermeture`. Depuis, LR a changé ce second rang : le repli `repli_debut_de_liste_ferme_au_bit`
  ne lie plus de NEW et ne délie plus de DEL (`LOT_LR.md`). V2 a aussi remplacé ce rang par E sur les
  films ÉGALE. **La reprise de L2 est donc le premier essai**, à re-mesurer sur la tête du moment. Code :
  branche locale `feat/cg-l2` (non poussée). `b12eb7692` porte le code avec les corrections du
  contrôle, `d09fcf989` son retrait. Rapport : `LOT_L2.md` (§9 surtout).
- **Statut dans `grammar/testdata/ecs_table.tsv`** : `ti=12 i16`, `ti=12 i18`, `ti=10 i2`, `ti=45 i0`,
  `ti=45 i1`, `ti=43 i19` et `ti=43 i21` sont tous `non_porte`.
- **`ti=12` (navpoints) et `ti=10` (managed-object)** : les autres composants de ces archétypes ont des
  lecteurs (`grammar/components_navpoint*.go`, `components_managed_object.go`) et des sondes de
  recherche (`navpoint_ti12_*_test.go`, `rtpc_ti10_*_test.go`) : partir de là pour le cadre.
- **`ti=45` (matchflow)** : aucun composant porté ; partir du déserialiseur dans le jeu.
- **Arrêts en vue C et sorties par rejet** : ce ne sont pas des composants. Ils relèvent d'autres
  causes (bloc `0xbc` : L5 sorti ; rejet = slot inconnu du monde). Hors périmètre de ce lot sauf
  preuve contraire.

## 4. Méthode (règles de la campagne, inchangées)

1. Un composant à la fois, par ordre de fréquence : `ti=43` (reprise de L2), puis `ti=12 i16`,
   `ti=45 i0`, `ti=10 i2`, `ti=12 i18`.
2. Lire le déserialiseur DANS LE JEU (Ghidra en lecture seule, HTTP direct `127.0.0.1:8089`, image
   `HaloInfinite.exe` HI_1_13_0). Une largeur « présumée par mesure » n'est admise que pour un composant
   fixe simplement sauté. Toute valeur utilisée ou toute taille variable se lit dans le jeu. Aucun
   réglage par film, par carte ni par version.
3. Porter le lecteur ; un vecteur de test écrit d'après l'écrivain ; `ecs_table.tsv` à jour ;
   `grammar.Rev` monte à chaque lot qui change une sortie, avec empreinte régénérée par la commande
   du dépôt.
4. Prouver composant par composant (§5), puis le lot entier.

## 5. Gates (recette de la campagne)

Kit versionné : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/kit_gates/`
(`denominateurs.tsv` = dénominateur fixe consolidé, `cartes_killsource.tsv` = film → carte pour
`killsource json`). Films : les 19 témoins de `config/replay_corpus.toml` + `1c4c63c2`.

```bash
# depuis apps/go-api, CGO (msys64 ucrt64), une base = feat/v75 au départ du lot, une tête = le lot
go build -tags=research -o fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture
go build -o killsource.exe ./cmd/killsource
go build -o gate.exe ./cmd/replay-corpus-gate
FILMS=bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,084a804d,0797ce72,111fa685,e5adf7b2,60ae07c4,a349fea8,a521164d,11de8353,50247b26,bfecd02b,4f77afc1,396cfc92,f75e7053,1c4c63c2
./fermeture.exe -racine <checkout principal>/data/cache/film_chunks -films $FILMS -sortie <dir> \
  -table internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv -mode v2 \
  -denominateur-fixe <kit>/denominateurs.tsv -paquets -plafond-gib 4 -mpp-declare
./killsource.exe json <id> -carte "<carte>" -cache <checkout principal>/data/cache \
  -catalogue <checkout principal>/data/titles/halo_infinite/reference/map_quant_bounds.json
KILLSOURCE_FIXTURES=<checkout principal>/data/cache/film_chunks go test ./internal/games/halo_infinite/film/internal/facts/killsource/ -run TestGoldenFilms -count=1 -timeout 60m
./gate.exe --reference=base --base=<sha base> --work-root <dir> --keep-work --json <fichier>
```

Critères :

- **Gate 2** : aucun film en baisse de paquets sains (carte base contre tête, `fermeture_films.tsv`),
  jamais assoupli. Toute perte brute est instruite paquet par paquet ; seule exception, une fermeture
  factice retirée (D2).
- Records utiles sains regagnés comptés par composant ; la colonne `cause` de `fermeture_paquets.tsv`
  dit où la marche s'arrête désormais.
- **Gate 3** : `killsource json` identique sur les 19 témoins. S'il change, `killsource.Rev` monte
  seulement si la sortie persistée change (D23).
- Gate de corpus : chaque `FAUX` et chaque `PERTE` est instruit. 0 `MANQUE`.
- Gates de code : gofmt, vet (et `-tags=research`), archlint, golangci-lint 0 issue, mutations
  rouges, `make gate-push` (TMP court dédié), CI verte.

## 6. Garde-fous

- Worktree dédié (branche `feat/<lot>`), jamais le checkout principal, jamais `main`.
- Aucune cuisson en lot sans l'accord de l'utilisateur. Une passe de décodage lourde à la fois sur la
  machine : prévenir la session RI (levelup-57, si elle tourne) avant chaque carte ou gate de corpus.
- Fusion dans `feat/v75` et recuisson du parc : décisions de l'utilisateur.
- Fichiers attendus : déserialiseurs de composants (`grammar/components_*.go`), `ecs_table.tsv`,
  `rev_chronique.go`. Les fichiers de la RI (canal, marche, lecteurs) ne sont pas concernés.
- Découvertes hors périmètre : consignées, pas traitées.

## 7. Coût estimé

Un agent Opus, effort high, borné aux cinq composants du §4. Ordre de grandeur : une demi-journée à
une journée de calcul.

- Lecture Ghidra et portage : de quelques dizaines de minutes à quelques heures par composant.
- Une carte v2 base + tête : environ 15 à 25 min.
- `killsource` et `TestGoldenFilms` : environ 20 min.
- Gate de corpus : environ 20 min.
- `gate-push` : environ 25 min.
- CI : environ 40 min.

Le plus gros risque est `ti=43`, déjà porté mais retiré une fois.

## 8. Prompt de lancement (à remettre à l'agent frais)

> Tu exécutes le handoff `.ai/HANDOFF_COMPOSANTS_BLOQUANTS_VUE_B_2026-10-06.md` sous le contrat du
> skill `plan-execution`. Lis d'abord ce handoff, puis `LOT_VA_V2.md` §5 et `LOT_L2.md` §9. Crée un
> worktree dédié depuis `feat/v75` (branche `feat/grammaire-arrets-vue-b`). Écris un plan court (un item
> par composant, gates du §5), puis traite les composants dans l'ordre du §4 : lecture dans le jeu
> (Ghidra en lecture seule), portage, preuve par la carte v2 et le gate 2 avant de passer au suivant.
> Aucun réglage par film, par carte ni par version. Aucune cuisson en lot, aucune fusion dans
> `feat/v75` : tu rends un rapport et la décision revient à l'utilisateur. Préviens la session RI avant
> chaque passe lourde.
