# Handoff — Emprise et cartes déplacées : clôture (2026-09-27)

Plan : `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md` (section L6). Maquettes de référence :
`.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` (onglet) et
`.ai/V7.5/MAQUETTE_TRI_CARTES_DEPLACEES_2026-09-26.html` (cartes déplacées).

## 1. État

| Étape | État |
|---|---|
| L0 à L5 (code) | fait, sur `feat/v75` |
| L6.1 revue adversariale | close (ronde 1 : R1-R13 ; ronde 2 : 0 constat ; R14-R15) |
| L6.2 rattrapage local | fait (voir §3) |
| L6.3 fusion + CI | fait : `feat/v75` = `0cae929af`, CI verte au niveau job (runs 36335594113, 36335588369) |
| L6.4 gate visuel | **à faire par l'utilisateur** |
| L6.5 rattrapage prod | **à faire par l'utilisateur, à la main** (§3) |
| L6.6 référence équipement §4 | à faire |
| L7 véhicules | plan détaillé à écrire puis soumettre |

Commit local non poussé : `d7c40151e` sur `wt/emprise` (L6.3 statué au plan, journal) — et ce
handoff. Les pousser avec L6.6 (le pre-push complet prend environ 12 minutes).

Worktree : `C:\Users\Guillaume\Projects\LevelUp-wt-emprise`, branche `wt/emprise`, alignée sur
`feat/v75` plus les commits ci-dessus. Dossier principal : avancé à `0cae929af` ; les
modifications non commitées d'autres sessions y sont intactes (`CLAUDE.md`, les deux
`CONTRIBUTING`, entrées de journal réappliquées en fin de fichier). Sauvegarde de l'état
d'avant : scratchpad de la session, dossier `principal_backup`.

## 2. Gate visuel (L6.4)

`make dev` depuis le dossier principal, Escouade, soirée du 22/09, maquettes à côté :
- onglet **Emprise** (l'ancienne adresse Usages y redirige) : piste camp contre camp, contrôle
  des ressources, au fil de la session, répartition des prises (fiches), match par match,
  production, rendement, habitude ;
- **Contributions** : Répartition des frags, Outils de destruction, les quatre cartes d'objectif ;
- **Dynamique** : Écart cumulé au FDA attendu.

L'utilisateur nomme les cartes validées et celles à reprendre. Toute reprise = un commit sur
`wt/emprise`, gate web (`lefthook run pre-push` compris), puis fusion dans `feat/v75`.

## 3. Rattrapage des bases (L6.5)

### Ce qui a été écrit en local

Une seule base, une seule table :
`data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`, table append-only
`match_pad_pickups_by_tier` (lue par la vue `_latest`). `levelup backfill-pad-tiers` : 12 matchs
écrits, 99 déjà présents, 0 échec. `backfill-usage-summary` n'a rien écrit (0 écrit, 17 à jour) :
rien à reporter.

Pourquoi il en faut un en prod : le correctif des niveaux de socle (L4) ne répare que les passes
futures. Les matchs dérivés pendant la panne portent leur marque et ne seront jamais repris
d'eux-mêmes ; sans rattrapage, l'Emprise les affiche « non classé ».

### ATTENTION avant d'écraser la base de prod par la copie locale

- **La base locale s'arrête au 22/09** : dernier match 2026-09-22 22:29 (+02), 9 170 matchs dans
  `match_registry`. La prod synchronise en continu : tout match postérieur présent en prod
  (et toute autre écriture faite en prod depuis) serait **perdu** par l'écrasement.
- La base locale a été compactée le 27/09 par une autre session (1 264 → 350 Mio, vues `_latest`
  identiques) et contient aussi le rattrapage killsource du 27/09 (autre chantier) : la copie les
  emporterait.
- Le code du chantier n'est en prod qu'après la fusion `feat/v75` → `main` (sortie v7.5) : un
  rattrapage fait avant ce déploiement serait refait sur une base que la prod réécrit.

Vérification à faire avant la copie, sur la prod (serveur arrêté ou en lecture) :

```sql
SELECT max(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')), count(*) FROM match_registry;
```

Si la prod a plus récent que le 22/09 22:29 ou plus de 9 170 matchs, **ne pas écraser**.

### Voie recommandée (sans copie)

Après le déploiement de v7.5, sur le VPS : arrêter le serveur (un seul écrivain par base), puis
`levelup backfill-pad-tiers` (reprenable, aucun décodage de film, un artefact à la fois ; lit les
artefacts de rejeu déjà rangés de la prod), puis relancer le serveur. `--dry-run` d'abord pour
voir la liste. `backfill-usage-summary` n'est pas nécessaire.

### Si copie malgré tout

Serveur de prod arrêté ; copier `shared_matches_v2.duckdb` **et** supprimer / ne pas laisser
traîner un `shared_matches_v2.duckdb.wal` de l'ancienne base à côté ; relancer ; vérifier que
`match_pad_pickups_by_tier_latest` compte au moins 111 matchs.

## 4. Autres coordinations

- levelup-ac lira `shared_matches_v2.duckdb` en local (gate `replay-corpus-gate`, lecture seule,
  serveur arrêté) dans l'heure qui suit 20:00 le 27/09 ; il a l'accord de l'utilisateur pour
  arrêter/relancer le serveur local.
- Découvertes non traitées : section « Découvertes » du plan (L6.1 : ligne « frags obtenus avec »
  d'un match au camp inconnu ; `backfill-usage-summary` : 111 artefacts au schéma périmé, recuisson
  hors chantier).
- Leçon : les gates des lots web doivent jouer `lefthook run pre-push` ; deux garde-rails
  (`lint-no-hardcoded-fields`, `lint-contract-ratchet`) n'ont échoué qu'au push.

## 5. Décision du 2026-09-27 : la riposte et l'isolement remplacés par deux graphes

Décision utilisateur : le bloc « Groupés ou isolés » (nuage d'isolement, riposte, « Isolement,
soirée après soirée », mis de côté au §0 du plan) n'est PAS repris. Il est remplacé par les
deux graphes de l'artefact « Écart à l'équipe »
(https://claude.ai/artifact/TtJstMS6cBuCo4jP7tyRzo, proposition 2) :
**« Placement et rendement de chaque vie »** et **« Part des vies par placement »**.
La mise en forme y a été réfléchie avec l'utilisateur : la PORTER telle quelle, ne rien
réinventer (titres, libellés, ordre, couleurs, légendes, infobulles ci-dessous sont les siens).

**Emplacement retenu** : onglet Emprise, à la place réservée au bloc « Groupés ou isolés »
(le nuage pleine largeur, la barre des quarts dessous). La carte Riposte de Synergies (« Frags non
ripostés », `SquadRiposteCard` + `SquadRiposteMatricePanel`) disparaît avec ses tests et son
champ de contrat (règle « 0 code mort ») : l'artefact le dit, « celui-ci prend sa place ».

### Grandeur et unité

- Unité = la **vie** (apparition → mort ou fin de manche), une ligne de la table des vies
  (`persist/lives_persister.go` : début, fin, cause de fin).
- Par vie : **distance médiane au coéquipier vivant le plus proche pendant la vie, en portées de
  radar** de la variante (`regulation.toml [radar_range_m]`, 18 m Arène, 24 m BTB ; match sans
  portée connue hors de l'univers), **part du temps de la vie hors radar** (infobulle seulement),
  **frags dans la vie**, **durée**.
- Exige un **balayage des trajectoires du film au sync**, agrégé par vie (n'existe pas : c'est le
  gros du lot, côté Go, écrit sous les règles anti-ART — table append-only + vue `_latest`).
- Pièges tranchés dans l'artefact : instants où l'équipe est à terre, où le joueur porte
  l'objectif, où il est en véhicule → hors du dénominateur, comptés à part et publiés, jamais
  mélangés. N et échantillon faible publiés.
- Quarts : **isolé** = distance médiane ≥ 1,0 portée de radar ; **rentable** = au moins un frag
  dans la vie.

### « Placement et rendement de chaque vie » (nuage, hauteur 420)

- Un point par vie, couleur du joueur (couleurs d'escouade), liseré 1 px couleur de carte.
  Taille = durée : `6 + min(durée, 90 s) / 9`.
- X : « distance médiane au coéquipier le plus proche pendant la vie, en portées de radar »,
  titre d'axe centré sous l'axe, bornes 0 à 2, pas de 0,25, libellés à deux décimales avec virgule.
- Y : « frags dans la vie », −0,5 à 5,5, pas de 1, libellés négatifs masqués ; léger décalage
  vertical aléatoire (±0,25) pour décoller les points de même compte (affichage seulement).
- Repère du radar : trait vertical pointillé à 1,0, couleur d'accent, étiquette « portée du radar »
  en haut à l'intérieur (11 px) ; PAS de zone « isolé » teintée sur ce graphe.
- Frontière horizontale pointillée (gris discret) à 0,5 : « au moins un frag dans la vie ».
- Quatre quarts nommés, texte gris discret 12 px, titre en capitales + sous-titre :
  - haut gauche « À PORTÉE ET RENTABLE » / « sûr » ;
  - haut droite « ISOLÉ ET RENTABLE » / « flanqueur, surveiller la régularité » ;
  - bas gauche « À PORTÉE ET COÛTEUX » / « duel à travailler, pas le placement » ;
  - bas droite « ISOLÉ ET COÛTEUX » / « vie donnée pour rien, seul » — le SEUL quart teinté
    (couleur du quart coûteux à 10 %, son texte dans cette couleur) : c'est celui que la vue
    existe pour trouver. Les quarts ne se classent pas du bon au mauvais.
- **Gros point par joueur** : médiane X × médiane des frags de ses vies, taille
  `18 + min(nombre de vies, 200) / 10`, couleur du joueur, cerclé 2 px couleur de texte, au-dessus
  du semis.
- Infobulle d'une vie : « **Joueur** · une vie de m:ss » / « distance médiane X radar · P % de la
  vie hors radar · k frag(s) ». Infobulle du gros point : « **Joueur** · N vies » / « médiane X
  radar · k frag(s) par vie » / « P % des vies isolées et sans frag ».
- Légende en bas, centrée, un item par joueur ; un clic isole le semis ET le gros point du joueur
  (même nom de série).

### « Part des vies par placement » (barres, hauteur 230)

- Une barre horizontale empilée à 100 % par joueur (premier joueur en haut), épaisseur 22,
  séparateur 1 px couleur de carte entre segments. Axe X de 0 à 100 %, noms des joueurs en gras.
- Quatre segments, dans cet ordre et ces couleurs — l'échelle de niveaux de l'app, validée
  daltonisme dans les palettes ; jamais une même teinte à deux opacités :
  « à portée et rentable » `perf-tier-1`, « isolé et rentable » `perf-tier-2`,
  « à portée et coûteux » `perf-tier-4`, « isolé et coûteux » `perf-tier-5`
  (`perf-tier-3` n'est pas utilisé ; mêmes couleurs pour les quarts du nuage).
- Valeur « v % » écrite DANS le segment (11 px, gras, encre sombre), seulement à partir de 8 %.
- Infobulle : « **Joueur** · N vies » puis une ligne par quart « nom : v % ».
- Légende en bas, centrée.

### Règles de rendu à tenir (goûts de l'utilisateur, déjà appliqués à l'Emprise)

Légendes en bas et centrées ; graphe centré verticalement dans son bloc ; valeurs dans les barres,
pas de texte en bout de ligne ; titres factuels (ceux ci-dessus) ; FR et EN ; jetons de couleur
seulement (skill `color-tokens`).

### Suite

Un plan dédié (balayage Go puis cartes), revu avec `plan-review`, à soumettre avant tout code. Il
passe après L6.6 ; son ordre par rapport à L7 (véhicules) est à fixer avec l'utilisateur, les deux
lots touchant au film au sync.
