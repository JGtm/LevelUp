# Revue finale adverse — fond « Relief » et coupe par étage (item R.1)

Date : 2026-11-18. Relecteur à contexte frais, lecture seule, une seule ronde (skill `adversarial-review`).

Relu :
- LevelUp, branche `feat/fonds-relief` (worktree `C:\t\relief-int`, tête `588834ca87`) contre sa base de fusion avec
  `origin/release/v8.3.0` (`f5c0b92cec`), hors `data/` et `generated.ts` : 223 fichiers, +15 244 / −1 111 ;
- brique `halo-infinite-map-images`, `v1.3.1..v1.4.0`, dossier `himap` (39 fichiers, +5 553).

## Contrat du lot

```
Objectif        : un troisième fond « Relief », une couche « Traits du relief » et une coupe par étage dans les
                  trois vues 2D, calculés à la cuisson par la brique, teintés et coupés par le navigateur.
Périmètre INCLUS : brique (relief, sols, traits, maillage et sddt du module partagé), mapfond-build (--relief-seul,
                  relief dans la cuisson complète), mapfond-webp, contrat et routes, bibliothèque web du relief,
                  rejeu, Tactique, Occupation, cerne deux tons des pions, jetons, ADR 0049, guides.
Périmètre EXCLU  : cuisson du parc (E.1), plan par mode (8.3.1), cerne des autres marqueurs que les pions.
Invariants       : Plan = défaut ; fonds, plans et calages publiés inchangés ; aucun joueur masqué dans le rejeu ;
                  relief périmé = relief absent ; couches, PathResolver, i18n, jetons, seuils de CLAUDE.md.
Décisions tranchées : D-1 à D-16, les 12 choix de D.1, conception.md (format, seuils, coupe, suivi), faces en
                  triangles, `.relief.json.gz`, coupe non persistée, données chargées à l'ouverture du rejeu.
Dette assumée    : fichiers déjà au-dessus de 500 lignes (ReplayCanvas, keys.ts, port/services.go), messages
                  d'erreur français des handlers, « mètres » pour des unités du jeu.
```

## Ce qui a été fait, et ce qui ne l'a pas été

- Lu en entier : le service, les handlers, le nommage, l'outil de cuisson, la conversion WebP, les 13 fichiers de
  production du relief dans la brique, les 20 modules de `lib/replay/relief/`, les hooks communs, le branchement
  dans les trois vues, le cerne, les jetons et leur garde-rail, l'ADR 0049.
- Lancé : `node tools/lint-cross-feature-imports.mjs` (0 violation), `node tools/lint-no-hardcoded-colors.mjs`
  (0 violation), ESLint sur les fichiers du relief (0 erreur), `go test ./internal/mapfondsidecar/ ./cmd/mapfond-webp/`
  (verts).
- Contrôlé sur pièces : les 8 sidecars du worktree qui portent un relief (5 commités, 3 non commités) ; l'encodage
  de 3 images publiées (Recharge, Behemoth, Live Fire).
- NON fait : suites Go complètes, vitest, tests `gamefiles`, contrôle à l'écran, Safari. Les tests de la brique
  n'ont pas été relus ligne à ligne.

## Compte

| Sévérité | Nombre |
|---|---|
| P0 | 0 |
| P1 | 3 |
| P2 | 9 |

---

## P1

### P1-1 — `--relief-seul` publie en succès un relief « sans maillage » pour une carte qui a un plan

- Où : `apps/go-api/cmd/mapfond-build/relief_seul.go:48-73` (aucun contrôle), `cuisson_forge.go:84, 87, 90, 93`
  (`maillageForge` rend un maillage nil quand le blob est absent OU illisible), `navmesh.go` (`maillageNatif` rend
  nil sur `ErrSansMaillage`, que la brique rend aussi quand toutes les candidates sont illisibles).
- Déclenchement : lancer `--relief-seul` sur une carte dont le sidecar publié porte un `plan`, depuis un poste où
  `.ai/re_dump/navmesh/<id>.blob` manque ou ne se décode pas. Le plan note que ce dépôt est COPIÉ à la main dans le
  worktree de cuisson : c'est le cas d'E.1 (111 cartes).
- Conséquence : `calculeRelief` reçoit un maillage nil, écrit un relief `sans-maillage` (zone = emprise de la
  matière, ni étage ni coupe), `noteResultat` compte « ajouté », code de sortie 0. Le seul signal est une ligne
  `Info` avec `degradations=[sans-maillage]`. Une fois servi, le rejeu affiche « Carte sans maillage de
  navigation » pour une carte qui en a un, et la coupe disparaît des trois vues. D-7 fixe pourtant les cartes sans
  maillage à deux (les Thunderhead).
- Correction : dans `refaitLeRelief`, mettre la carte en échec quand le sidecar publié porte un plan
  (`s.Plan != nil`) et que la cuisson en mémoire n'a pas de maillage.

### P1-2 — Aucun garde-rail sur les reliefs publiés, alors qu'un relief mort est muet par construction

- Où : `apps/go-api/internal/mapfondsidecar/images_publiees_test.go:36` ne couvre que l'image plan
  (`TestChaqueImagePlanSuperposeAuFond`) ; `cmd/mapfond-build/sans_plan_test.go:70` ne couvre que les plans.
- Déclenchement : un sidecar publié dont `reliefData` ou `reliefImage` nomme un fichier absent, dont le calage
  recopié n'est plus celui du sidecar, ou dont l'image n'a pas les dimensions du calage (copie manuelle incomplète
  depuis le dossier temporaire de cuisson, conversion WebP oubliée ou interrompue, fond recuit ailleurs).
- Conséquence : le service répond 404 et journalise un `Warn` par requête, le web retombe sur le Plan « sans rien
  dire » (D-1). Rien ne rougit en CI, rien ne se voit à l'écran : la carte a simplement perdu son relief en
  production. Le lot E.1 va écrire environ 220 fichiers et réécrire 109 sidecars.
- Correction : un test sur le dossier publié, frère de `TestChaqueImagePlanSuperposeAuFond` : pour chaque sidecar
  à relief, les deux fichiers existent, les données se décompressent, `schemaVersion` et calage sont ceux du
  sidecar, l'image a les dimensions du calage, et l'ensemble des reliefs `sans-maillage` est exactement celui des
  fonds sans plan.
- État mesuré aujourd'hui : les 8 reliefs du worktree passent ces contrôles (voir « Conditions qui tiennent »).

### P1-3 — Code sans appelant de production, entretenu par des tests verts (règle 7, anti-pattern 1)

Chaque symbole ci-dessous n'est lu que par un test (`grep` sur `apps/web/src` et `cmd/mapfond-build`, hors tests).

| Symbole | Où | Seul lecteur |
|---|---|---|
| `heightGreyByte` | `apps/web/src/lib/replay/relief/shading.ts:66` | `shading.test.ts` |
| `stackColumn` | `lib/replay/relief/floorStack.ts:50` (réexporté par `index.ts`) | `floorStack.test.ts`, `slice.test.ts` |
| `FloorStack.truncated` | `lib/replay/relief/floorStack.ts:35, 64` | `floorStack.test.ts` |
| `ReliefComputer.mode()` et l'option `forceInline` | `lib/replay/relief/client.ts:187, 39` | `client.test.ts` |
| branche `image` de `meshShareOf` | `lib/replay/relief/floors.ts:70-80` | aucun appelant ne passe `image` ; la brique écrit toujours `meshShare` |
| `reliefFond.Traits` | `apps/go-api/cmd/mapfond-build/relief_fond.go:43, 90` | aucun, ni production ni test |

- Conséquence : la règle « ce qui n'a pas d'appelant se supprime avec ses tests » n'est pas tenue ; la branche morte
  de `meshShareOf` est en outre le SEUL lecteur web du drapeau `SOL_MAILLAGE` (voir P2-3).
- Correction : supprimer ces symboles et leurs tests, ou leur donner un lecteur (par exemple journaliser
  `rf.Traits` dans `journaliseLeRelief`, que l'item B.1 demandait de journaliser par carte).

---

## P2

### P2-1 — La route de l'image sert un relief périmé que la route des données refuse

- Où : `apps/go-api/internal/service/replay_map_background_relief.go:103-116` (`readReliefImage` ne passe pas par
  `donneesReliefAuCalage`) ; le contrat dit l'inverse : `internal/port/services.go:227-228` et ADR 0049 D-15.
- Déclenchement : données au calage différent du sidecar ; `GET …/background.png?variante=relief`.
- Conséquence : 200 avec l'image des hauteurs d'un ancien calage, quand `background.relief.json` répond 404. Sans
  effet dans le web d'aujourd'hui (l'image n'est demandée qu'après le succès des données,
  `lib/replay/reliefQueries.ts:66`), mais le service ne tient pas ce que son port et l'ADR écrivent.
- Correction : faire lire et contrôler les données par `readReliefImage` avant de servir l'image, ou restreindre la
  phrase du port et de l'ADR aux seules données.

### P2-2 — Une carte à maillage sans étage détecté est dite « sans maillage » et perd son curseur

- Où : `apps/web/src/lib/replay/reliefViewLogic.ts:55`.
- Déclenchement : `floors.levels` vide avec `min` et `max` présents. La brique rend ce cas quand aucun pic ne
  dépasse 3 % des échantillons (`relief_sols_etages.go:71`) : des hauteurs de sol étalées à peu près uniformément
  sur plus de 16,7 u (0,5 / 0,03). Aucune des 8 cartes cuites n'y est ; les grandes cartes naturelles d'E.1 peuvent
  l'être.
- Conséquence : le rejeu affiche le bloc « Carte sans maillage de navigation », ce qui est faux, et les trois vues
  retirent le curseur, que l'ADR 0049 D-13 désigne comme « la bonne commande » sur terrain naturel.
- Correction : un état `ready` sans pastille (« Tout » et le curseur seuls) quand le maillage existe et que
  `levels` est vide.

### P2-3 — Trois données servies n'ont pas de lecteur de production, et l'ADR dit le contraire

- Où : drapeau `SOL_MAILLAGE` (bit 1 du canal B) : validé par `decodeImage.ts:76`, lu seulement par la branche
  morte de `floors.ts:70-80` ; codes `hauteurs-ecretees` et `pile-tronquee` : décodés par `decodeData.ts:37`,
  jamais lus ni traduits.
- Écart avec l'ADR 0049 : D-8 dit que l'infobulle « est le lecteur du drapeau » (elle lit `floors.meshShare`,
  `useReliefView.ts:167`) ; D-6 dit que le client « traduit » les codes de dégradation (seul `sans-maillage` est lu).
- Conséquence : règle 18 non tenue pour ces trois valeurs ; une carte aux hauteurs écrêtées ou à la pile tronquée
  s'affiche sans que rien ne le dise.
- Correction : décision de contrat à prendre par le pilote (le format est celui d'une brique publiée) : soit
  corriger les deux phrases de l'ADR et inscrire l'exception, soit journaliser les deux codes côté web à la lecture.

### P2-4 — Le masque de la coupe alloue un canvas et 11 Mo à chaque pas du curseur

- Où : `apps/web/src/lib/replay/relief/reliefBackground.ts:39-46` (`maskCanvas` appelle `canvasOfSize(null, …)` et
  `new Uint8ClampedArray(width * height * 4)` à chaque appel).
- Déclenchement : fond Plan ou Rendu, coupe active, glisser le curseur de hauteur (Behemoth : 1601 × 1776).
- Conséquence : un canvas plein cadre et un tampon de 11,4 Mo par pas, sur le fil principal, alors que l'en-tête du
  fichier pose la règle inverse (« un curseur qui glisse ne doit pas allouer 11 Mo à chaque pas »). Saccades
  probables sur téléphone.
- Correction : garder le canvas du masque et son `ImageData` dans `ReliefCanvases`, comme les deux autres.

### P2-5 — L'infobulle dit « ombré du nord-ouest », la lumière vient du haut à gauche de l'écran

- Où : `apps/web/src/features/match-replay/i18n/relief.i18n.ts:53, 78` ; `apps/web/src/components/map-relief/text.ts:43, 62`.
- Déclenchement : carte tournée d'un quart de tour ou plus (orientation retenue ou automatique).
- Conséquence : D-15 garde la lumière en haut à gauche de l'ÉCRAN (`shading.ts`, `lightForTurns`) ; sur une carte
  tournée de 180°, elle vient du sud-est de la carte et le libellé est faux, dans les deux langues.
- Correction : « ombré depuis le haut à gauche » / « shaded from the upper left ».

### P2-6 — Le défilement au-delà de six pastilles est inatteignable

- Où : `apps/web/src/features/match-replay/settings/ReplayFloorControls.tsx:31, 84` ;
  `apps/web/src/components/map-relief/ReliefPills.tsx:21, 94`.
- Déclenchement : aucun. La brique plafonne les étages à cinq (`himap/relief_sols.go:47`, `MaxEtages = 5`), donc
  `chips.length > 6` est toujours faux.
- Conséquence : le choix 11 de D.1 est codé et testé (`ReplayFloorControls.test.tsx`, `ReliefPills.test.tsx`) mais
  ne peut pas se produire ; les tests donnent une couverture qui n'existe pas à l'écran.
- Correction : décision du pilote : lever le plafond de la brique, ou retirer la branche et ses tests.

### P2-7 — La brique publie pour tout module partagé des mesures propres à Live Fire

- Où : brique, `himap/sddt_partage.go:31-35` (constantes) et `:39-45` (`sddtPartageIgnore`).
- Déclenchement : une seconde carte dont la géométrie vit sous `pc/globals/` (levier `moduleGeometrie`).
- Conséquence : son sidecar recevrait « 42 volumes sous le sol joué, 84 % de leurs cellules… » et « il retire
  0,4 m2 du praticable », chiffres mesurés sur Live Fire seule, et la frontière comme l'eau seraient ignorées sans
  mesure. Par ailleurs la branche « module partagé » de `ModuleVariante` (`himap/sddt.go:476`) n'a plus d'appelant
  de production : ses deux appelants (`cuisson.go:497, 522`) sortent avant par `sddtPartageIgnore`.
- Correction : énoncer la règle sans chiffre dans la dégradation (« sddt d'un module partagé non appliqué : il
  décrit plusieurs arènes »), et garder la mesure de Live Fire dans le commentaire.

### P2-8 — Un démontage pendant le chargement journalise un faux « repli sur le plan »

- Où : `apps/web/src/lib/replay/useReliefRender.ts:78-80`.
- Déclenchement : quitter la vue (ou changer de carte) pendant `readReliefImage` / `computer.load` ; `dispose()`
  ferme le transport, `load` rejette (« calculateur fermé »).
- Conséquence : `console.warn("[replay-relief] chargement du relief impossible, repli sur le plan : …")` alors
  qu'aucun repli n'a lieu ; en développement sous StrictMode, à chaque montage. Le journal des replis, seul signal
  d'un relief illisible, s'en trouve bruité.
- Correction : ne journaliser que si `alive` est encore vrai.

### P2-9 — La zone tenue choisie n'est pas rapprochée de la coupe dans les réapparitions (à vérifier à l'écran)

- Où : `apps/web/src/features/tactical/TacticalReapparitionsPlan.tsx:116` (`PiluleZoneTenue` reçoit
  `zonesProposees`, non filtrées) contre `:142, 152` (teinte et étiquettes filtrées par `zonesCoupees`).
- Déclenchement : lecture des réapparitions, une zone tenue choisie à un étage, puis une coupe sur un autre étage.
- Conséquence : la zone choisie reste nommée dans la pilule et pilote la lecture, mais sa teinte et son étiquette
  ont quitté le plan. Constat tiré de la seule lecture du code : à confirmer au Gate D avant d'en faire un item.
- Correction : garder à l'écran la zone choisie quelle que soit la coupe, ou le dire dans la pilule.

---

## Conditions vérifiées qui tiennent

Données et format (axe a) :
1. Encodage : `q = R + 256 G`, pas 1/256 u, `z0` = plancher du minimum, A dans {0, 255}, A = 0 implique R = G = B = 0,
   bits 2 à 7 de B nuls. Vérifié dans `relief_image.go` et sur 3 images publiées (alphas {0, 255}, B ≤ 3, aucun
   pixel inconnu coloré, aucune zone hors hauteur connue, part du maillage identique au JSON à 10⁻⁴, centiles
   identiques à `height.low` / `high`).
2. Projection : centre du pixel publié, plus proche voisin, jamais d'interpolation ; échelle différente = erreur
   typée, rien n'est écrit ; décalage et matière perdue journalisés.
3. Demi-pixel : la brique (`pose`, `rasterise`), les traits (rupture entre deux colonnes à `x + 0,5`) et le web
   (`pixelToWorld`, `worldToPixel`, `kept`) partagent la même convention.
4. Déterminisme des traits : identifiants de nœuds entiers, tri par (nœud, segment), aucun parcours de map, tri
   final total des polylignes.
5. Marche des carrés : niveau `bas ≤ t < haut`, un coin posé sur `t` compte dessous, aucune division par zéro
   dans `positionDeCourbe` ; identifiants (arête, niveau) sans collision.
6. Relief périmé : égalité exacte du calage (les flottants font l'aller-retour JSON sans perte), version de schéma,
   décompression bornée à 32 Mio décompressés, tout manquement journalisé et rendu en 404.
7. Les 8 sidecars à relief du worktree : fichiers présents, calage identique, dimensions de l'image égales au
   calage, `schemaVersion` 1, tous avec un plan et sans dégradation.
8. `--relief-seul` relit et réécrit le sidecar par `mapfondsidecar` seul ; `--plan-seul` et `mapfond-webp
   -convertir` conservent les deux champs du relief ; une cuisson complète sans relief retire les trois fichiers.

Routes (axe b) :
9. Noms : `filepath.Base`, refus de `/` et `\`, forme `<clé>.relief.json.gz` exigée ; `map_id` validé avant le
   service ; variante inconnue, vide ou de casse différente = 400 ; routes sous l'ownership joueur ; ETag et
   `Cache-Control` centralisés avec garde-rail.
10. Le fichier de données ne finit pas en `.json` : l'index des fonds du décodeur ne le prend pas pour un sidecar ;
    l'empaquetage du bureau (`git ls-files data/titles/*/reference/*`) l'emporte.

Web (axe c) :
11. Repli : relief absent, illisible ou en panne = Plan, puis Rendu ; l'image de base chargée est le plan ; la
    pilule ne propose plus « Relief ».
12. L'image des hauteurs n'est pas partagée entre deux chargements : le cache garde le `Blob`, le décodage est
    refait avant chaque cession au worker (pas de tampon détaché relu).
13. Rejeu : la coupe ne lit ni piste ni filtre de joueur (garde-rail présent et discriminant).
14. Suivi : sol porteur, seuil 0,5 u, délai 300 ms en temps de rejeu, prise immédiate à la réapparition et au saut
    de lecture, coupe figée pendant la mort, choix manuel qui éteint le suivi, identité du joueur ouvert stable
    (pas de remise à zéro à chaque rendu).
15. Occupation : positions jugées sur leur sol porteur, compte affiché, porte du bloc jugée avant la coupe ;
    environ 200 positions par match, coût négligeable même sur 14 000 faces.
16. Tactique : seules les zones nommées sont filtrées (deux lectures), les quatre autres calques le disent.
17. Lumière : les quatre directions de `lightForTurns` donnent bien le haut à gauche de l'écran pour 0 à 3 quarts
    de tour horaires.

Couleurs et règles (axes d, f) :
18. Aucune couleur en dur dans `features/` ni `components/` ; la rampe a une seule source (`reliefRamp.ts`) ; le
    garde-rail mesure le cerne contre la rampe, le Plan, le Rendu et la carte du thème, sous quatre visions.
19. Libellés présents en français et en anglais par typage ; query keys dans `lib/query/keys.ts` ; aucun import
    de feature depuis `lib/` ; les deux exceptions datées de D.4 sont retirées.
20. Fichiers neufs sous 500 lignes, fonctions neuves à 5 paramètres au plus côté LevelUp ; inscriptions archlint
    datées et motivées ; crédit à Cartographer présent dans le README de la brique et dans l'ADR.

## Notes hors constat

- Le worktree porte trois reliefs non commités (`btb_drydock`, `btb_exiled`, `forest`), image encore en `.png` :
  cohérents au contrôle, hors du diff relu.
- La pile des sols n'est pas fusionnée par la même règle des deux côtés (brique : le plus haut gagne ; web : le
  premier rencontré). L'écart est borné par la fusion (0,4 u) et la conception le mesure à un pixel sur 2,8
  millions : aucun effet visible attendu, mais la phrase de `relief_sols.go` (« la cuisson et le web voient les
  mêmes sols ») est plus forte que le code.
- À l'export vidéo, le suivi pose la coupe par un état React puis un calcul du worker : le retard n'est pas borné
  à une image quand l'export avance plus vite que le calcul. Limite déjà notée par le pilote en D.6.
