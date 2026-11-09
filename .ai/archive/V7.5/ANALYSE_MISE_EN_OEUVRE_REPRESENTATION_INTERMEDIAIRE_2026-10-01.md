# ANALYSE — Mise en œuvre de la représentation intermédiaire, port Rust, ordre des sujets restants (2026-10-01)

> **Statut : ANALYSE, rien n'est lancé.** Ce n'est pas le plan d'exécution : celui-ci s'écrira
> après les décisions du §7. Code lu à la tête `feat/v75` = `8b894a677` (plan de suite d'audit J1-J11
> fusionnés, J11.4 et J11.5 restants ; la branche de J12 n'est pas visible dans ce dépôt).
>
> **Pour qui** : l'utilisateur, puis les agents qui écriront l'ADR et le plan du chantier.
>
> **Pièces détaillées** (preuves fichier:ligne, établi / supposé séparés) :
> - `.ai/V7.5/film_re/RAPPORT_IR_CARTOGRAPHIE_GO_2026-10-01.md` — inventaire des traversées, germes,
>   conception Go, 13 lots, vue C, conflits J12, mémoire ;
> - `.ai/V7.5/film_re/RAPPORT_PORT_RUST_2026-10-01.md` — port Rust face à la spec, expérience sur nos
>   10 témoins v41, idées à reprendre.
>
> L'exemple Rust de mesure (`examples/levelup_closure.rs`) et le clone du port vivaient dans le
> scratchpad de la session (éphémère) ; le protocole est au §10 du rapport Rust pour le refaire.

---

## 0. Réponse courte

1. **La spec est la bonne direction, et le port Rust le confirme de façon indépendante** : son auteur a
   abouti à la même séparation « lecture canonique avec étendues et arrêts typés → résolution à part ».
   Mais **son décodeur ferme 5,5 % des trames de nos films v41, contre 66,9 % pour le nôtre**, mesuré
   sur les mêmes 335 960 paquets. Il a supprimé toute récupération, donc il perd le contexte des
   entités (datums, images-clés, table anticipée). Leçon : la couche de récupération séparée, nommée
   et comptée de la spec (P8) est indispensable ; « canonique pur » ne tient pas sur Halo Infinite.
2. **Le déclencheur de la spec (95 % des records utiles fermés) dépend presque entièrement d'une
   seule cause** : « vue C : terminateur hors cadre » pèse 92,5 % des records utiles non fermés sur
   HI_1_13_0. Fermer toutes les autres causes plafonne ce build à 81,3 %. Or cette cause est surtout
   une **fin de vue B fausse** (une naissance d'entité non lue, puis des rejets en cascade), c'est-à-dire
   le **résidu de film dense, clos par décision de l'utilisateur le 2026-09-23**. Sans le rouvrir, le
   déclencheur ne sera jamais atteint.
3. **Mise en œuvre concrète** : le « marcheur unique » de l'étape 1 ne doit PAS être un second marcheur
   parallèle (ce serait un second parseur, contraire à P1). La marche de production existe déjà
   (`ScanMarcheDesTrames`) ; on la **refactore en marcheur**, consommé dès le premier jour par les états
   de mouvement et par la carte de fermeture. La spec doit être corrigée sur six points (§2), dont : une
   seule table d'entités (pas trois), deux phases (images-clés puis trames), et l'ordre de migration
   (créations et positions en dernier, pas en premier).
4. **Ordre recommandé** : J12 fusionné → vague J11.4 → **retrait des replis nuls** (avec les comptes du
   parc donnés par la vague) et **instrument de fermeture v2** → ADR → étape 1 de la représentation
   intermédiaire → **campagne de grammaire** (si l'utilisateur rouvre le résidu dense) en alternance
   avec les lots sans différence → changements de comportement en dernier.
5. **Véhicules et tourelles** : ce qui touche la lecture du film (12 composants `ti=40` non portés,
   l'octet `+0x818`, les blocs `ti=43` qui cachent la 3e montée de G MONEY) appartient à la campagne
   de grammaire. Le reste (règle du Falcon de Behemoth, code mort `turretRidesNotRideable`) est
   indépendant et peut se faire n'importe quand après J12.

---

## 1. État mesuré au 2026-10-01

| Fait | Valeur | Source |
|---|---|---|
| Records utiles fermés, meilleur build (HI_1_13_0) | 80,2 % (anciens builds 0 à 30 %) | `.ai/V7.5/film_re/MESURES_CLOTURE_J11_2026-10-01.md` §2 |
| Part de « terminateur hors cadre » dans les records utiles non fermés | 92,5 % (HI_1_13_0), 91,9 % (corpus) | rapport Go §4.3 |
| Plafond si toutes les autres causes sont fermées | 81,3 % sur HI_1_13_0 | rapport Go §4.3 |
| Parcours des trames delta par cuisson | ≈ 37 (+3 sous garde), dont 22 balayages bit à bit, 3 marches grammaticales | rapport Go §1.3-1.5 |
| Marcheurs grammaticaux de trames delta | 3, incompatibles : 3 vues (mouvement), 8 vues (morts d'objet), 8 vues (killsource) ; `IDLowBits` 13 en dur contre 10..15 calibré | rapport Go §1.6 |
| Marcheur ancré du bipède | 10 exécutions par cuisson (i48 lu deux fois) | rapport Go §1.5 |
| Replis au registre / actifs sur le corpus / nuls | 119 / 53 / 66 | MESURES J11 §3 |
| Artefacts de rejeu cuits localement | 115, tous au schéma 71 (avant J3-J11) | `data/cache/replays/halo_infinite` |
| J4 S1 (« étage de lectures unique ») | une seule implémentation partagée cuisson / collecteur, **pas** une seule passe (six traversées) | rapport Go §1.7 |

La mesure du déclencheur **sous-compte l'utile** : la colonne `product_use` de
`grammar/testdata/ecs_table.tsv` marque « aucun » des composants que le produit publie (états de
mouvement `ti=35 i29/i54/i57/i62`, équipe `ti=9 i0`, véhicules `ti=40 i0..i2`, zones `ti=13 i0..i3`,
socles `ti=42 i0`) : 36 lignes utiles à la tête, contre 44 annoncées par la spec. Elle ne compte pas
non plus le dénominateur des entrées de contrôle utiles.

---

## 2. Corrections de la spec, sur pièces

| # | La spec dit | Le code dit | Preuve |
|---|---|---|---|
| C1 | Trois tables d'entités, une par vue (P5, §4) | **Une seule** table de datums indexée par l'eid complet, la vue en attribut (lot 5.16). Le commentaire de `world.go:32` est resté à l'ancien modèle (à corriger, J12.5) | `grammar/frame_infer.go:104-117` |
| C2 | Une marche, une fois, dans l'ordre du flux | Impossible tel quel : la table anticipée lit les images-clés ultérieures, la bande bipède lit l'image-clé du chunk suivant, le filtre de génération daté dépend de créations postérieures. Cible : **deux phases** (images-clés, puis trames) + préliminaires bornés | `movement_states.go:194-199`, `keyframe_anticipe.go:182`, `offline_biped_band.go:182`, `generations_vivantes.go:42-58` |
| C3 | Une queue opaque à cause typée suffit | « Terminateur hors cadre » est une **fermeture refusée** dont la position fautive est inconnue. La structure doit porter un état « records non prouvés » distinct de la queue opaque à position connue | `frame_closure_classement.go:14-15`, `:94-109` |
| C4 | La récupération est hors de la marche (P8) | Elle vit aussi **dedans** : localisation de la vue B par signature, NEW de tête (`debutDeLaListe`), datums à position libre, liaison par anticipation, élection d'ancre d'image-clé. Il faut les marquer, sinon T1 mesure un mélange | rapport Go §2, correction 4 |
| C5 | Ordre de migration : créations de bipède et positions d'abord | Elles ne peuvent pas migrer à différence nulle : sur `bfecd02b` l'ancrage trouve 162 444 records bipèdes, la marche 97 447. Elles restent dans la récupération ; ordre corrigé au §3.3 | `movement_states.go:16-26` |
| C6 | Types dans `film/types` | `film/types` est haché octet par octet dans quatre périmètres de révision ; un paquet feuille `film/internal/grammar/lecture` ne fait monter que `grammar.Rev` | `revision/perimetre.go:171-173`, `couches.go:20-28` |
| C7 | « 7,9 Go de pic sur un BTB » (§P9) | C'était `51101d1d`, un CTF, par le déroulage `NamedEventsFrom`, corrigé le 2026-09-03 (4,9 s, 0,08 Gio). Pics actuels 0,08 à 1,19 Gio | `.ai/V7.5/MESURES_CUISSON_PERF.md`, `REGISTRE_REPORTS.md:492` |
| C8 | Étape 1 = marcheur « NON consommé » | Un marcheur parallèle non consommé est un second parseur. La marche de production devient le marcheur et a deux consommateurs dès le lot 1.2 (états de mouvement, carte de fermeture, qui recopie aujourd'hui son pilotage) | `frame_closure.go:159-164` |

Ce qui reste juste et que le port Rust confirme : P1, P3, P6, P7, P8, P10, l'étendue en bits comme
unité de provenance, et P9 (le port matérialise tout et coûterait ~1 à 3,5 Go par film pour une
traversée complète, [S] extrapolé de ses 130 octets par champ mesurés).

---

## 3. Mise en œuvre concrète

### 3.1 Forme

- **Types** : paquet feuille `film/internal/grammar/lecture` (aucune logique), ratchet de couche :
  ni `replay` ni `decfilm` ne l'importent. Ordre de grandeur : ~40 octets par record et 12 par
  composant dans une arène réutilisée par paquet, contre ~350 octets alloués aujourd'hui pour un
  record delta à 4 composants (rapport Go §3.2).
- **Marcheur** : `FilmContext.ImagesCles()` et `FilmContext.Trames()` en `iter.Seq2`, plus un
  distributeur `grammar.Distribuer(fc, canaux...)`. L'interprétation reste faite PENDANT la marche par
  les crochets de l'`Observation` existante (`observateur.go`) : c'est la forme paresseuse qui garde
  la sémantique actuelle, donc la différence nulle.
- **En-tête (P4)** : `IDLowBits`, largeurs MPP, disposition i0, `gate15` (aujourd'hui choisi dans
  `killsource` seulement) et l'octet `+0x818` deviennent des paramètres d'en-tête avec leur
  provenance (lu, calibré, supposé). `IDLowBits` a aujourd'hui deux provenances contradictoires : c'est
  la première incohérence que la structure résout.
- **États (P2)** : interprété / délimité / infranchissable, plus « non prouvé » (C3). Les statuts
  `porte` / `partiel` / `non_porte` de `ecs_table.tsv` sont des capacités statiques ; l'état est par
  occurrence.
- **Fermeture (T1)** : l'oracle existe (`vueCFermee`) ; la sortie de vue B devient typée
  (terminateur / rejet hors datum / rejet de vue, compteurs déjà dans `observateur.go:291-303`).
- **Révision** : les lots sans différence régénèrent les empreintes à révision constante (précédent
  J4.0) ; `grammar.Rev` ne monte qu'aux lots de comportement, et chaque montée rouvre le backlog
  killsource du parc.

### 3.2 Idées reprises du port Rust (en Go, sans code Rust)

| Idée | Où | Gain |
|---|---|---|
| Qualifier la fin de vue B et la forme de la vue C dans la carte de fermeture | instrument | Casser la cause n° 1 en causes localisées par composant |
| Partition de couverture par paquet (champs / opaque / bourrage / non lu), dérivée de la structure | T1 bis | Progrès mesurable même sans fermeture |
| Mode borné « indisponible » contre le bourrage à zéro | instrument | Compter ce qui dépend de bits hors payload. Aucune preuve Ghidra du « bourrage du moteur » n'existe dans le dépôt (`source/bits.go:18-25` s'appuie sur l'arbitrage V15) |
| Compte d'événements déclaré du chunk de type 3 | `highlight_events.go` | Oracle de complétude gratuit du kill-feed et du fil des morts (le port trouve 220/220 et 844/844 ; nous ne lisons pas ce compte) |
| Arrêts typés riches (`RuntimeContextUnavailable{champ}`, `GenerationMismatch`, `Rejected{entête}`) | structure | Forme naturelle de P2/P4 |

À ne pas reprendre : la matérialisation, l'état de décodage privé, le refus sans repli de `gate15`
et `+0x818` (il coupe tout le reste du paquet), la clé par version majeure seule, et « trois vues
complètes » pris pour un succès (92 % de faux positifs mesurés).

### 3.3 Lots (détail, fichiers et risques : rapport Go §3.8)

| Lot | Contenu | Preuve | Taille | Change une sortie ? |
|---|---|---|---|---|
| 0 | J12 fusionné, références `replay-equiv` re-figées après J12, ADR (sur l'ADR 0034 réorganisé par J12.5), instrument de fermeture v2 (§5.1) | — | S | non |
| 1.1 | Paquet `grammar/lecture` + ratchet de couche | G-arch, empreinte à révision constante | S | non |
| 1.2 | La marche de production devient `FilmContext.Trames` ; états de mouvement et carte de fermeture en sont les consommateurs ; étendues de record (`HeaderBit` posé) et de vues ; sortie de vue B typée | `replay-equiv` 0, `frame_closure.golden` identique | M | non |
| 1.3 | Phase images-clés `FilmContext.ImagesCles` sur la mémo existante ; `KeyframeClosure` consommateur | `keyframe_closure.golden` identique, G-equiv 0 | M | non |
| 1.4 | Tests T1, T3, T5 (fuzz étendu), T6 (`-race`, job `film-race` de J12.6) | verts | S | non |
| 2.1 | Canaux mouvement + tir continu formalisés | `replay-equiv` 0 | S | non |
| 2.2 | Canaux d'image-clé (armes portées, inventaire, marques, équipes, bandes, générations vivantes, table anticipée, liaison) | `replay-equiv` 0 | M | comptes de replis seulement (déclarés) |
| 2.3 | Canaux de tête de vue A (tirs, 117, lunette, ramassages, 103, véhicules) | `replay-equiv` 0 | M | non |
| 2.4 | Récupération ancrée mutualisée : un ancrage bipède par film au lieu de 10, records marqués « récupérés » | données 0, `coverage.fallbacks` déclaré, banc `b.Loop` | L | comptes seulement |
| 2.5 | Récupération des objets du monde (créations multi-archétypes, pistes sur l'union des bandes) | `replay-equiv` 0 | L | non |
| 2.6 | Statborg descendu de `facts/objectives` vers `grammar` (les faits ne lisent plus d'octet) | `replay-equiv` 0 | M | non |
| 2.7 | **Changements de comportement déclarés** : morts d'objet sur le marcheur unique, canaux delta lus par la marche là où elle couvre mieux, killsource EN DERNIER (contexte partagé avec la cuisson) | corpus gate + banc de vérité, `grammar.Rev`/`killsource.Rev` → backlog | L | oui |
| 3.1 | Retrait des marcheurs redondants, un contexte par cuisson | banc, `replay-equiv` 0 | M | non |
| 3.2 | Mesure de performance | rapport | S | non |

**Mesurer avant 2.4/2.5** : le gain annoncé (20 à 40 % du temps de décodage) est une estimation [S].
Les durées par étape existent déjà (`replay/observe.go`) : une mesure sur trois témoins et un BTB, en
début de chantier, décide si la mutualisation vaut son coût.

---

## 4. Le port Rust (S. Imbleau)

- **Ce que c'est** : ~16 000 lignes Rust, licence MIT OR Apache-2.0, v41 seulement, épinglé sur
  LevelUp `43a01721e` + lecteurs de `d61443ef5` (2026-09-26). Il nous crédite (`CREDIT.md`). Branche
  utile : `simbleau/theater-experiments` ; `feat/serde-serialization` n'apporte rien au décodeur
  (modèles de réponse de l'API, `chunks_exact` → `as_chunks`).
- **Architecture** : la spec, presque à la lettre (P1, P3, P7, P8 par retrait, P10). Différences : pas
  de provenance des paramètres (refus à la place), état de décodage privé, tout matérialisé.
- **Grammaire** : exactement nos composants (544 `porte` + 51 `partiel`, aucun autre) ; même lecture
  de la vue C ; rien de neuf sur les kinds 1/2 ni le bloc secondaire. En retard sur nous pour le
  type 8 (roster opaque chez lui) et la localisation de la liste d'événements.
- **Mesure sur nos 10 témoins v41** : 5,5 % de trames fermées (contre 66,9 %), 9,2 % des bits des
  trames traversés, 0,15 à 0,31 s et 230 à 416 Mio par film (sous-estimés : il s'arrête tôt).
- **Son oracle est notre propre code** (403 465 bornes du Go de `d61443ef5`, contextes injectés) :
  il prouve l'accord de son lecteur avec le nôtre, ni la justesse ni la fermeture. 43 commits ont
  touché `grammar`/`profile` depuis, dont les 14 exceptions datées. Utile seulement comme détecteur de
  régression de lecteur (son harnais Go `reference-contexts-v41_test.go` compile probablement à la
  tête : tous ses symboles existent, adaptateur de manifeste à écrire).
- **Coopération** (question ouverte 6 de la spec, décision utilisateur) : il gagnerait notre oracle de
  fermeture `vueCFermee`, notre contexte de production (table anticipée, `TableDeDatums`,
  `debutDeLaListe`), `DecodeRoster` (type 8), `pickGate15` et la table ECS. Nous gagnerions peu de
  code, mais un second compteur de couverture et un détecteur de dérive.

---

## 5. Les trois sujets restants

### 5.1 Campagne de recherche sur la grammaire du jeu

C'est **le seul levier du déclencheur**. Contenu, par gain :

1. **Instrument de fermeture v2** (recherche seule, aucune sortie de production, S) : ventiler
   « hors cadre » par sortie de vue B, vue C vide ou non, état du slot rejeté dans le bloc de type 1,
   dernier composant lu avant la fin faible ; dénominateur des entrées de contrôle utiles ; colonne
   `product_use` corrigée ; compte déclaré du chunk 3 ; mode borné contre le bourrage. Précédent : sur
   `bfecd02b` (lot 5.21), 0 des 23 325 en-têtes rejetés n'était vivant dans le bloc de type 1.
2. **Résidu de film dense** (DÉCISION UTILISATEUR : clos le 23/09) : le record de naissance des entités
   nées et mortes entre deux images-clés (NOTE 5.26, 801 eid sur `bfecd02b`), le décalage de masque
   (`FUN_14076cb60` teste `i - decales`, NOTE 5.16 D1). C'est ce qui ferait monter la fermeture au-delà
   de 81 %.
3. **Les 14 exceptions datées** (`grammar/lecteur_position_ratchet_test.go`, `exceptionsDuPortage`) :
   la lecture du jeu fait monter certains films et baisser d'autres ; critère de retrait écrit
   (« la grammaire dépendante du build est établie »). Effet sur la fermeture : quelques dizaines de
   paquets chacune.
4. **Composants non portés à poids de fermeture** : `ti=43 i35` (12 113 paquets, première cause sur
   HI_1_12_0), `ti=2 managed-engine-timers` (3 635), puis les composants véhicules (§5.3).

Place : l'instrument (1) avant la représentation intermédiaire ; les lots de recherche (2-4) APRÈS
l'étape 1, pour que leurs correctifs atterrissent dans UN marcheur au lieu de trois, et avant le lot
2.7.

### 5.2 Retrait des 66 replis jamais déclenchés

Classement par couche (registre `facts/fallback/registre_*.go`, sites relevés à la tête) :

| Couche | Nombre | Exemples |
|---|---|---|
| grammaire / profil | 9 | `repli_cadre_de_marche_par_defaut_conserve`, `repli_largeurs_mpp_par_defaut`, `repli_largeurs_monde_par_defaut_conservees`, `repli_i0_porte_et_region_par_defaut`, `repli_index_de_region_largeur_un`, `repli_chunks_apres_trou_abandonnes`, `repli_amorce_grenade_profil_de_reference`, `repli_generation_vivante_inconnue_tag1`, `repli_chunk_de_replication_saute` |
| objectifs (`facts/objectives`, `replaybuild`) | 16 | identités de manche, xuid manquants, zones |
| rejeu : équipement, objectifs, identités, places, véhicules | 30 | porteurs, collines, drapeaux, grappin, `repli_tourelle_montee_loin_du_porteur` |
| killsource + collecteur | 11 | homonymes, index en collision, carte |

- Les 9 replis de grammaire sont des **paramètres hors flux par défaut** (P4) : les retirer avant la
  représentation intermédiaire simplifie son en-tête.
- **Mesure gratuite du parc** : le zéro vaut pour 19 témoins seulement. La vague J11.4 recuira les
  115 artefacts (leur `coverage.fallbacks`) et rejouera killsource sur le parc (ligne « replis de la
  passe ») : le compte du parc sortira sans cuisson supplémentaire.
- Réserves déjà écrites (MESURES J11 §3) : `repli_famille_objectif_vide` n'a pas de compteur branché ;
  `repli_garde_equipement_negatif_a_zero` est mesuré non nul dans la passe usage-summary ;
  `repli_distances_de_touche_desactivees` et `repli_precision_par_arme_passe_sautee` sont nuls par
  construction.
- Place : après la fusion de J12 (mêmes fichiers : `slog`, tris) et après la vague J11.4 ; avant la
  représentation intermédiaire. Taille S-M, sans différence sur le corpus par définition.

### 5.3 Véhicules et tourelles

| Sujet | Touche la lecture du film ? | Où le traiter |
|---|---|---|
| 12 composants `ti=40` non portés (`i30-i33`, `i35`, `i36`, `i38-i42`, `i45-i47` : tourelle auto, état de type, jeu d'armes, air-drop, warp…) | oui (fermeture : ~1 600 paquets) | campagne de grammaire |
| Octet `+0x818` de `vehicle-type-physics` supposé (`repli_physique_de_type_de_vehicule_supposee`, 33 193 lectures sur 7 témoins) | oui — c'est une porte, la règle du 25/09 exige Ghidra (l'écrivain de l'octet à la construction du véhicule) | campagne de grammaire |
| 3e montée de G MONEY dans un Ghost (`81c02726`) : blocs `ti=43 i20-i22` non lus, la lecture déraille et perd l'embarquement | oui | campagne de grammaire (décision utilisateur en attente depuis le 25/09) |
| Falcon de Behemoth affiché à tort | non (règle d'affichage du rejeu) | indépendant, après J12 |
| `turretRidesNotRideable` toujours nul | non (code mort, à la prochaine montée de schéma) | indépendant |
| Destruction publiée (`object-dead-state` de `ti=40`) | déjà fait (`replay/vehicle_end.go`) | — |

---

## 6. Ordre recommandé

```
J12 (en cours) ──► fusion ──► J11.4 vague locale (serveur arrêté, PRÉVENIR) ──► J11.5 vérification visuelle
                                     │
                                     ├─► Replis nuls (comptes du parc) ............................ S-M
                                     ├─► Instrument de fermeture v2 + product_use corrigé .......... S
                                     └─► ADR de la représentation intermédiaire (spec corrigée) ..... S
                                                 │
                                                 ▼
                           RI 1.1 → 1.4 (marcheur = marche de production, zéro différence) ...... M
                                                 │
                    ┌────────────────────────────┴────────────────────────────┐
                    ▼                                                         ▼
     RI 2.1 → 2.6 (zéro différence ; mesure perf               Campagne de grammaire (SI rouverte) :
     avant 2.4/2.5)                                             résidu dense, exceptions datées,
                    │                                           ti=43, ti=40, +0x818
                    └────────────────────────────┬────────────────────────────┘
                                                 ▼
                           RI 2.7 (changements déclarés, killsource en dernier) → 3.1 → 3.2
```

Pourquoi cet ordre :
- **J12 d'abord** : J12.1, J12.3, J12.4 et J12.5 touchent les fichiers centraux de la structure
  (`film_scan*.go`, `world.go`, `film_context.go`, `killsource/decode.go`) et les références
  d'équivalence ; commencer avant garantit des conflits sur chaque golden.
- **Replis avant la structure** : moins de code à faire migrer, et un en-tête plus simple.
- **Marcheur avant la campagne** : les correctifs de grammaire atterrissent une fois, dans un seul
  marcheur, avec la sortie de vue B typée comme instrument.
- **Comportement en dernier** : le lot 2.7 dépend de la fermeture ; sous 81 %, faire lire les canaux
  delta par la marche ferait perdre des records que l'ancrage trouve.

Deux branches en parallèle au plus (consigne du 26/09), un worktree dédié par branche :
`feat/representation-intermediaire` et, si la campagne est rouverte, `feat/campagne-grammaire`.

**Coût estimé** (agents Opus, effort high au plus, hors temps machine des gates de corpus) :

| Bloc | Agents |
|---|---|
| Replis nuls | 1-2 |
| Instrument v2 + ADR | 1-2 |
| RI 1.x | 2-3 |
| RI 2.1-2.6 | 4-6 |
| RI 2.7 + 3.x | 3-4 |
| Campagne de grammaire (recherche, Ghidra, ouverte) | 3-6 |
| **Total** | **~14-23** (la spec en estimait 8-10 pour la RI seule ; les deux phases et la récupération mutualisée l'alourdissent) |

---

## 7. Décisions à prendre par l'utilisateur

**Tranchées par l'utilisateur le 2026-10-01 (questionnaire, recommandations retenues)** :
- (1) **Oui** : le résidu de film dense est rouvert, dans une campagne de recherche sur la grammaire
  du jeu, bornée, qui commence par l'outil de mesure (instrument de fermeture v2).
- (2) **Oui** : l'étape 1 et les lots sans différence (1.1-1.4, 2.1-2.6) démarrent après la fusion de
  J12 ; le lot 2.7 attend la campagne ; mesure des durées par étape avant 2.4/2.5.
- Documents commités sur `feat/v75` sans push.

Ces décisions ne sont pas un GO de lancement : chaque bloc démarre sur un GO explicite et daté. Restent
ouvertes : (3) replis nuls, (4) report des corrections dans l'ADR, (5) coopération Rust, (6) blocs
véhicules dans la campagne.

1. **Rouvrir le résidu de film dense** (clos le 23/09) dans une campagne de grammaire ? Sans cela, le
   déclencheur de la spec est inatteignable (plafond 81,3 % sur le meilleur build).
2. **Ouvrir l'étape 1 de la représentation intermédiaire avant le déclencheur** (la spec le permet :
   aucune sortie ne change) ? Recommandation : oui pour 1.x et 2.1-2.6 ; le lot 2.7 attend la campagne.
3. **Retrait des replis nuls (DU-7)** : sur les comptes du parc après la vague J11.4, classe A
   (45) moins les réserves, classe B (2) selon la mesure, classe C (19) selon leur propre cible ?
4. **Corrections de la spec C1-C8** à reporter dans l'ADR ?
5. **Coopération avec le port Rust** : partager notre oracle de fermeture et notre contexte de
   production, ou non ?
6. **Véhicules** : ouvrir les blocs `ti=43 i20-i22` (G MONEY) et l'octet `+0x818` dans la campagne ?

---

## 8. Limites de cette analyse

- La branche de J12 n'est pas visible dans ce dépôt : les conflits sont déduits de la description du
  plan, pas d'un diff.
- L'expérience Rust compare deux définitions différentes de « fermé » ; l'oracle LevelUp a été appliqué
  à la sortie du port (biais listés au §8.2 du rapport Rust). Les nombres de trames sont identiques des
  deux côtés.
- Les gains de performance (20 à 40 %) et les tailles mémoire proposées sont des estimations [S].
- Le mécanisme « naissance non lue → rejets → vue C lue au mauvais endroit » est cohérent avec les
  notes 5.15, 5.16, 5.26 et la sonde P1-S3, mais sa part exacte dans les 264 757 paquets « hors cadre »
  reste à mesurer (instrument v2).
