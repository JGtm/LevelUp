# SPEC — Représentation intermédiaire du film (2026-09-25)

> **Statut : SPÉCIFICATION DE CADRAGE d'un chantier FUTUR — rien n'est planifié ni lancé.**
> Décidée par l'utilisateur le 2026-09-25 (DU-8 du plan
> `.ai/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`) : « tu prépares une spec pour la structure
> intermédiaire ». Le chantier démarrera sur décision, quand son déclencheur (§9) sera mesuré par
> la carte de fermeture (lot J4.0 du plan).
>
> **Pour qui** : l'utilisateur, et les agents qui écriront l'ADR puis le plan d'exécution de ce
> chantier. Les chiffres cités sont mesurés sur la tête de la campagne des retours rejeu
> (`feat/rr-m4b`, `f2b52546b`) sauf mention contraire ; ils se re-mesurent à l'ouverture.

---

## 1. Le problème, en une phrase

La grammaire du décodeur sait lire le film, mais ce qu'elle lit n'existe nulle part comme un
tout : chaque canal (positions, tirs, équipement, états de mouvement…) relance sa propre lecture,
garde ce qui l'intéresse, jette le reste — et là où la grammaire ne sait pas encore traverser, il
cherche des en-têtes bit à bit.

Conséquences mesurées (registre `.ai/AUDIT_DECODEUR_FILM_2026-09-24.md`) :

| Symptôme | Preuve |
|---|---|
| ~40 traversées du film par cuisson ; le marcheur delta du bipède rappelé par 9 sites | audit, faiblesse 6 ; `walkDeltaBipedRecords` |
| Des lecteurs jumeaux d'un même composant qui décident différemment | GA2-1 à GA2-5 ; seconde séquence de balayage dans `sync/killcollector` |
| Une heuristique de balayage qui perd des corps entiers | GB-1 (P0) : filtre `tag == 1`, les corps de génération ≥ 2 sans position |
| Des tests seulement indirects au niveau du format | fermeture d'image-clé, part des vues de contrôle fermées, compteurs de désynchronisation |
| Une provenance qui se perd | la coordonnée de lecture d'un kill disparaît à la sérialisation des faits |
| Des faits persistés dont la fraîcheur dépend de l'appelant | RA1-1, RA1-3 |

## 2. Ce que la représentation intermédiaire EST, et ce qu'elle n'est pas

**Elle est la SORTIE de la grammaire.** La grammaire parcourt le film UNE fois, dans l'ordre du
flux, et range tout ce qu'elle a lu : chaque paquet, chaque vue, chaque record, chaque composant,
avec sa position exacte en bits, et les morceaux qu'elle n'a pas su traverser. La résolution
(vies, identités, trajectoires, événements — la couche `facts`) lit ensuite cette structure, plus
jamais les octets.

**Elle n'est PAS un second parseur.** La grammaire (profil par build, lecteurs de composants portés
depuis l'écrivain du jeu) reste la SEULE autorité sur « quel élément est où ». La structure ne
remplace pas la grammaire : c'est la grammaire qui la remplit.

**Elle n'exige PAS de tout décoder.** Elle distingue deux choses :

- **traverser** un composant : connaître son étendue en bits pour atteindre le suivant (un record
  n'a pas de longueur propre : sans cela, le reste du paquet est illisible) ;
- **interpréter** un composant : donner un sens et un type publiés à sa valeur.

La table ECS (`grammar/testdata/ecs_table.tsv`) montre l'écart : **1 067 composants** sur 49
archétypes, **44** servent au produit ; **544 sont portés, dont 502 sans aucun usage produit** —
portés pour être SAUTÉS ; 440 non portés, 51 partiels, 32 lecteurs non câblés. On n'interprétera
jamais tout, et il n'en est pas besoin ; la traversée, elle, se mesure et progresse (vue de
contrôle lue jusqu'au bout sur `81c02726` : 19,5 % des paquets avant la reprise M4b, 76,2 % après
six composants portés, dont un son en boucle et un minuteur d'EMP).

NB : traverser exige souvent de LIRE des valeurs (largeurs variables, portes, comptes imbriqués).
La traversée EST l'exécution du désérialiseur ; interpréter, c'est publier un sens.

## 3. Principes (à inscrire dans l'ADR du chantier)

| # | Principe | Ce qu'il empêche |
|---|---|---|
| P1 | La grammaire est la seule autorité ; la structure est sa sortie, jamais un parseur parallèle. | Deux lecteurs qui divergent (GA2). |
| P2 | Trois états par composant : **interprété** (valeur typée disponible), **délimité** (étendue connue, sens inconnu ou sans intérêt), **infranchissable** (largeur inconnue : le reste de la vue devient une **queue opaque** avec sa cause typée). | Confondre « non lu » et « absent » ; cacher un trou. |
| P3 | La forme suit les niveaux STABLES du format ; ce qui dépend du build vit dans le profil et la grammaire. | Une structure qui change à chaque build. |
| P4 | Les paramètres hors flux sont des entrées explicites, avec leur provenance : **lu**, **calibré** (ex. `IDLowBits`, valeur de runtime qui diffère d'un film à l'autre : 11 sur `000d5950`, 14 sur la capture live), **supposé** (ex. l'octet `+0x818` du véhicule, repli `repli_physique_de_type_de_vehicule_supposee`). | Une hypothèse cachée dans un lecteur. |
| P5 | L'état nécessaire à la lecture fait partie de la structure et s'expose en lecture seule : trois tables d'entités (une par vue), archétype et génération par slot, liaisons inférées marquées. | Un état implicite (D-5) ; une identité par slot seul (GB-1). |
| P6 | Identité d'un record = `types.LifeKey{Slot, Gen}` (J5 du plan), jamais le slot seul. | GB-1, RA2-1/2/3. |
| P7 | Provenance de bout en bout : tout fait résolu (vie, tir, position) porte l'étendue source de la lecture qui le fonde. | Une donnée publiée qu'on ne sait plus rattacher au film. |
| P8 | La récupération est une COUCHE SÉPARÉE : les balayages heuristiques travaillent sur les queues opaques et produisent des records marqués **récupérés**, nommés et comptés (D-10) ; jamais mélangés aux records de la marche. | Une structure fidèle qui cesse d'être testable. |
| P9 | Paresseux et sans copie : la structure porte des étendues sur les tampons des chunks ; les valeurs se décodent à la demande. Parcours en flux par défaut, matérialisation réservée aux tests et aux outils. | L'explosion mémoire (cuisson d'un BTB : 7,9 Go de pic mesuré). |
| P10 | Ordres totaux partout ; aucune sortie ne dépend de l'itération d'une map. | Le déterminisme par chance (faiblesse 9). |

## 4. Modèle de données (forme proposée)

Les niveaux sont ceux que le décodeur connaît déjà, cités avec leur fichier (`film/internal/...`).

```
Film
├── En-tête        build, version majeure, empreinte du registre, profil résolu (clé de build),
│                  paramètres hors flux (valeur + provenance lu | calibré | supposé)
├── Registre       archétypes -> composants dans l'ordre d'itération (chunk_00, grammar/registry.go)
├── Chunks[]       index, type (1 registre et en-tête · 2 jeu · 3 temps forts), étendue en octets,
│                  taille annoncée, finalisé (filmcache/finalise.go)
│   └── Paquets[]  (chunks de type 2) chunk, index, type, horodatage, étendue (source/film.go)
│       ├── Trame delta (type 0)   bit de configuration, puis trois vues DANS L'ORDRE
│       │   ├── Vue A (rang 0, messages)  corps { genre R(7) < 0x7b, étendue, état }   (frame_vue_messages.go)
│       │   ├── Vue B (rang 1, entités)   records[]                                     (frame_records.go)
│       │   │     genre NEW | DELTA | DEL | END (code préfixe R(1), sinon R(2))
│       │   │     handle LifeKey (bas = R(IDLowBits) + base ; génération = R(2))
│       │   │     archétype (R(6) sur NEW ; table d'entités sur DELTA), étendue
│       │   │     NEW : état par défaut (étendue), porte, masque, composants[]
│       │   │     DELTA : sélecteur de base R(1)[R(7)], masque, composants présents[]
│       │   │     DEL : R(32)
│       │   │     composant { index, nom, étendue, état, hypothèses[] }
│       │   ├── Vue C (rang 2, contrôle)  entrées[] { kind R(2), index de joueur, étendue, état }  (frame_vue_controle.go)
│       │   ├── fermeture                  bits consommés contre la longueur, par vue
│       │   └── queue opaque               { étendue, cause typée : composant X du record Y, ArretVueC, débordement… }
│       ├── Image-clé      records d'état COMPLET (pas de masque : tous les composants dans l'ordre du
│       │                  registre), rang de vue (deux bits de tête de l'identifiant), fermeture par record
│       └── Autres types   données de session, marqueurs, CHUNK_END (type 7)
├── Temps forts    chunk de type 3 : fil des morts, kill-feed (records et étendues)
├── État de marche tables d'entités par vue (slot -> archétype, génération, vue), liaisons inférées marquées
└── Diagnostics    fermetures, queues opaques par cause, hypothèses employées, records récupérés (couche P8)
```

Une **étendue** vaut `{chunk, paquet, bit de début, longueur en bits}` : c'est l'unité de la
provenance (P7) et de la fermeture.

## 5. Place dans les couches (ADR 0034)

| Couche | Rôle vis-à-vis de la structure |
|---|---|
| `source` | Inchangée : chunks, paquets, lecteur de bits canonique (seule porte aux octets, D-2). |
| `profile` | Inchangée : largeurs et grammaires par build, paramètres calibrés avec leur provenance. |
| `grammar` | PRODUIT la structure : un marcheur unique (`grammar.Lire`, nom à trancher) qui rend un parcours en flux (itérateurs Go 1.23, `iter.Seq2[Paquet, error]`) ; les lecteurs de composants existants sont ses briques (J6 : un portage par fonction du jeu). |
| types | Les types de la structure vivent dans le paquet feuille `film/types` (comme `LifeKey`, `EquipmentPlacement`) : aucune logique, importables par `grammar` et `facts`. |
| `facts` | CONSOMME la structure (et, en second, la couche de récupération) : vies, identités, tirs, événements. Plus aucun accès aux octets. |
| `replay` | Inchangé dans son rôle : publie depuis les faits ; ne voit pas la structure. |
| Hors de `film/` | Rien de nouveau : `killcollector` reçoit des faits par `decfilm` ; la façade ne grandit pas. |

Révision : la structure est une sortie de `grammar` ; son code est haché par `grammar.Rev` (avec
le périmètre par fermeture des imports de J3). Les faits lus depuis elle ne dépendent plus des
gardes de l'appelant (RA1-1 réglé par construction : les gardes décident de la PUBLICATION, pas de
la lecture).

## 6. Tests — la raison d'être du chantier

| # | Test | Forme |
|---|---|---|
| T1 | **Fermeture** : pour chaque paquet, vue et record, bits consommés = longueur. | Goldens par build sur les huit mini-bobines (`frame_closure.golden` de J4.0, `keyframe_closure.golden`) ; ratchet « aucune baisse ». |
| T2 | **Composant** : un vecteur de bits construit d'après l'écrivain du jeu (Ghidra) donne la largeur et, s'il est interprété, la valeur. | Obligatoire pour tout composant qui passe à `porte` ; les 544 existants sont couverts par T1. |
| T3 | **Provenance** : tout fait publié cite une étendue qui existe dans la structure. | Test sur les mini-bobines. |
| T4 | **Équivalence de migration** : chaque canal migré rend le même document. | `replay-equiv` à zéro différence, ou changement déclaré au `replay-corpus-gate`. |
| T5 | **Robustesse** : aucune panique, allocations bornées par les octets restants. | Fuzz (harnais `FuzzFilmRecordReaders` étendu). |
| T6 | **Déterminisme** : deux marches → même empreinte ; marches parallèles identiques. | `-race` en CI (job `film-race` de J12.6). |
| T7 | **Oracle externe** (facultatif). | La capture live « Rosette » (bits d'en-tête de record, 54 760 / 54 760 concordants sur `000d5950`) — l'outil qui la rejouait (`cmd/tmp_ecsschema`) n'existe plus, ses données sont à retrouver ; un port tiers (le port Rust en cours) comme second oracle de fermeture. |

## 7. Versions et builds

- Le profil reste clé par build (D-3) ; build inconnu = erreur typée, film mis de côté, jamais le
  profil du voisin (D-4).
- La FORME de la structure ne change pas d'un build à l'autre ; un build nouveau = une ligne de
  profil et des largeurs de composants.
- Avantage attendu : la casse d'un build nouveau se LOCALISE (« à partir du build X, le composant Y
  de l'archétype Z ne se ferme plus »), là où les canaux trouvent aujourd'hui moins de choses sans
  le dire.

## 8. Performance et mémoire

- Une traversée par film au lieu de ~40 : gain attendu, à mesurer (banc sur les mini-bobines et un
  BTB, `b.Loop`).
- Ordre de grandeur des volumes : 54 760 records sur `000d5950` ; 97 447 records de bipède sur
  `bfecd02b`, 315 251 sur `4f77afc1`. Matérialiser chaque composant décodé d'un BTB coûterait des
  centaines de mégaoctets : d'où P9 (flux, étendues, décodage à la demande).
- La persistance reste celle des FAITS (cache `film_facts`) ; persister la structure elle-même ne se
  décide que sur mesure (temps d'une marche BTB, taille compacte).

## 9. Déclencheur et étapes

**Déclencheur** (corrigé le 2026-09-25 avec l'utilisateur ; mesuré par la carte de fermeture,
J4.0 du plan en cours) : sur chaque build du corpus, **≥ 95 % des records et des entrées qui
PORTENT une donnée utile au produit sont fermés au bit près** — la marche atterrit exactement sur
le record suivant ou sur le terminateur de la vue. « Donnée utile » = un composant à usage produit
de la table ECS (colonne `product_use`, 44 composants aujourd'hui) ou une entrée de la vue de
contrôle que le produit lit. Un record fermé prouve du même coup que tout ce qui le précède dans le
paquet a été traversé juste.

Ce n'est PAS « 95 % de tous les paquets » : on n'interprétera jamais tout, et une vue ou un
archétype qui ne porte rien d'utile peut rester opaque sans coût. Le seuil se confirme à l'ADR ;
l'utilisateur peut décider d'ouvrir l'étape 1 plus tôt (elle ne change aucune sortie).

| Étape | Contenu | Preuve | Taille |
|---|---|---|---|
| 0 | Dans le plan en cours : carte de fermeture (J4.0), étage de lectures unique (J4), `LifeKey` (J5), un portage par fonction du jeu (J6), révisions par fermeture des imports (J3). | Plan en cours. | — |
| 1 | Marcheur unique produisant la structure, en flux, NON consommé ; T1, T3, T5, T6. | Zéro différence (`replay-equiv`). | M |
| 2 | Migration des canaux un par un vers la structure (ordre proposé : créations de bipède, positions, états de mouvement et tir continu, tirs, équipement, armes tenues, véhicules, objectifs, killsource) ; le balayage heuristique de chaque canal passe dans la couche de récupération (nommé, compté) ou disparaît si la structure couvre. | T4 par canal ; changements déclarés. | L |
| 3 | Retrait des marcheurs redondants ; une traversée par cuisson ; mesure de performance. | Banc ; zéro différence. | M |
| 4 | Persistance de la structure, SI la mesure la justifie. | Décision sur mesure. | ? |

Estimation grossière : 8 à 10 agents Opus (étape 1 : 1 ; étape 2 : 4 à 6 ; étape 3 : 1 ; revues :
2), hors temps machine des gates de corpus.

## 10. Non-objectifs

- Tout interpréter : seuls les composants utiles au produit reçoivent un interprète.
- Écrire un encodeur (tests aller-retour) : utile, pas requis.
- Changer le document publié : la structure est interne à `film/`.
- Halo 5 : autre format de film, hors sujet.

## 11. Questions ouvertes (à trancher à l'ADR)

1. Nom du marcheur et du type racine (`grammar.Lire` / `types.Lecture` ou équivalent anglais du
   reste du paquet `types`).
2. La table des entités par vue : état de `grammar` exposé en lecture, ou sortie à part entière ?
3. Valeur du seuil du déclencheur (§9) : 95 % des records utiles par build, ou un seuil par
   composant utile ?
4. Persistance de la structure (§8).
5. **Largeurs mesurées** : pour un composant de largeur FIXE qu'on ne fait que sauter, une largeur
   trouvée par la fermeture (essais jusqu'à ce que les paquets se ferment au bit près) et
   confirmée sur tout le corpus de chaque build est-elle acceptée comme valeur PRÉSUMÉE, inscrite
   comme telle, sans passer par l'écrivain du jeu dans Ghidra ? L'ADR 0034 (D-3, règle 2) veut la
   grammaire de l'écrivain, la mesure n'étant qu'un oracle ; la règle utilisateur du 2026-09-21
   exigeait Ghidra pour les états du joueur. Un composant de largeur VARIABLE (champs dépendant du
   contenu, portes, comptes) ne se mesure pas ainsi. Décision de l'utilisateur.
6. Coopération avec le port Rust : échange de cartes de fermeture comme oracle croisé (le partage
   de la table ECS ou du code est une décision de l'utilisateur).

## 12. Projet d'ADR (EN, à reprendre dans `docs/adr/` à l'ouverture du chantier)

> **ADR 00xx — Film intermediate representation (proposed).**
> *Context.* The grammar reads the film, but its output exists only transiently: each channel
> re-walks the film (~40 passes per cooking) and some fall back to bit-by-bit header scans. Grammar
> defects surface downstream, twin readers diverge, identity is keyed by slot only in places, and
> provenance is lost.
> *Decision.* The grammar produces, in one ordered walk, an intermediate representation shaped by
> the stable levels of the format (chunks, packets, frame views, records, components), with exact
> bit spans. Each component is interpreted, delimited or untraversable; untraversable bits form an
> opaque tail with a typed cause. Off-stream parameters are explicit inputs with their provenance
> (read, calibrated, assumed). Records are keyed by (slot, generation). Heuristic recovery is a
> separate, named and counted layer. The representation is streamed and lazy; the facts layer is
> its only consumer; the published document is unchanged. It amends ADR 0034 D-1, D-2, D-6, D-7
> and D-10.
> *Consequences.* One walk per film; closure becomes a per-packet, per-build invariant; a new build
> breaks in a localized, measured way; every published fact can cite its source span. Cost: a
> staged migration of every channel, with zero-difference proofs.
