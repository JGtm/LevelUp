# Containment « lettre de zone » — ce qui est etabli, ce qui ne l'est pas

> Lot 4 de la v7.5, `feat/v75`, 2026-08-08. Repond au point « Containment lettre de zone »
> du `PLAN_MASTER_FILM_KILLFEED_REJEU.md` (§ « Ce qui part en piste post-merge »), qui
> demandait de croiser les formes de zone decodees avec les evenements dates pour inverser
> le « Quelle zone : NON » du chantier d'origine.
>
> Ce document est un RAPPORT DE FAISABILITE CHIFFRE. Il ne decrit pas une feature livree :
> rien n'est persiste ni affiche, et le § « Decision » dit pourquoi.

## 1. Ce que l'etat de l'art a corrige dans le cadrage — a lire en premier

L'enonce du lot designait `match_player_positions` comme « LA dependance critique du
croisement ». **Cette table ne peut pas servir**, pour deux raisons independantes, l'une
et l'autre verifiees sur pieces :

| ce qu'on attendait | ce que la table est reellement |
|---|---|
| les positions des joueurs | **0 ligne** en base (`shared_matches_v2.duckdb`, 2026-08-08) |
| une position par joueur | **match-level, SANS xuid** — la delta-compression bloque l'index par joueur ; `team` est best-effort a -1 (`migration/steps_shared_player_positions.go`) |
| une table du pipeline | ecrite par le SEUL `cmd/diag_weapons_v3 -positions -write`, un CLI de diagnostic |

Le croisement n'en avait pas besoin. La vraie source de positions PAR JOUEUR existait
deja, ailleurs, et elle est meilleure : le pipeline du rejeu 2D.

	film (chunks TYPE_2)
	  -> filmdec.ScanFilmBipedPositions   positions bipeds, ~60 Hz, reechantillonnees a 100 ms
	  -> replay.BuildFromFilm             Track PAR XUID (pont : fil des morts + index de joueur)

La granularite (100 ms) et l'identification (xuid) sont donc toutes les deux acquises. La
dependance manquante n'a pas bloque le lot ; elle a change de nom.

**A retenir pour la suite** : `match_player_positions` est une table de diagnostic vide.
Ne pas la citer comme source de positions.

## 2. Les trois pieces, et ce qui manquait entre elles

| piece | ou elle vit | etat avant le lot |
|---|---|---|
| formes de zone | `data/titles/halo_infinite/reference/map_objectives.json` | versionnee, **aucun lecteur Go** — 34 cartes, 63 zones Bastion, 161 zones Extraction, 100 % avec forme |
| instants + auteur | `analysis/objectiveevents` (`NamedEvents` + `SlotIdentity`) | decode et teste, **aucun appelant de production** |
| positions par joueur | `analysis/replay` (`BuildFromFilm`) | livre, en prod |

Ce qui manquait etait exactement le joint : **aucune fonction `Contains`** n'existait sur
`mapvar.Shape`, et le catalogue de formes n'etait lu par personne. Le temoin du 2026-08-02
(etat de l'art Forge/zones, §Q2) avait bien joue le croisement une fois — mais avec
`cmd/tmp_zonetest`, un outil jetable depuis supprime, sur un seul film et 4 instants.

## 3. Ce que le lot a construit

| fichier | role |
|---|---|
| `analysis/replay/mapvar/containment.go` | `Volume` (forme posee + base orthonormee), `Contains`, `DistanceTo`, `Translate` |
| `analysis/replay/objectives_catalog.go` | lecteur de `map_objectives.json` : `LoadMapObjectives`, `Lookup`, `ZonesOfRole` |
| `analysis/replay/zone_attribution.go` | `AttributeZones` : actions x trajectoires x zones -> zone + couverture |
| `games/halo_infinite/film/filmcache` | la source disque du cache film, centralisee (elle existait en double) |
| `cmd/zone-attribution` | la MESURE : corpus, courbe taux(seuil), deux temoins, diagnostic |

Points de methode que le code porte et qu'il ne faut pas defaire :

- **Le test est en 3D et oriente.** Les zones montent de 1,0 a 4,0 m et descendent de 0,0
  a 1,0 m ; Streets et Cliffhanger ont des couloirs a l'aplomb d'une zone. Ignorer
  `Forward` declarait deja « dedans » 31 % de positions qui sont dehors.
- **Aucune lettre A/B/C n'est inventee.** Le catalogue ne porte aucun nom de zone et
  l'ordre du fichier n'en est pas un. Les zones sont identifiees par leur `InstanceID`
  (identite du JEU) et par un `SpatialRank` derive du tri des positions — **notre** ordre,
  stable d'une execution a l'autre, explicitement pas la lettre du jeu. Etablir la lettre
  demande un temoin externe (releve Theater) ; elle se posera alors dans un champ distinct.
- **La couverture est publiee avec le resultat**, et les compteurs somment au total
  (invariant teste) : une attribution sans son denominateur ne se lit pas.

## 4. Le corpus mesurable, et pourquoi il est petit

La chaine a quatre maillons. Sur les **208 matchs en mode a zones** du registre — c'est la
classification canonique d'`objectiveevents.ObjectiveTypeOf`, qui couvre 124 Total
Control, 83 Strongholds et 1 Land Grab, pas seulement le libelle « Strongholds » :

| maillon | ecarte | reste |
|---|---:|---:|
| film en cache local | 151 | 57 |
| bornes de dequantification de la carte (`map_quant_bounds.json`, 15 cartes) | 45 | — |
| formes de zone au catalogue (`map_objectives.json`, 34 cartes) | 4 | — |
| **les quatre maillons** | | **8** |

Repartition des 57 matchs avec film, par catalogue manquant :

| bornes | formes | matchs | cartes |
|---|---|---:|---|
| oui | oui | **8** | Illusion, Forest, Streets, Forbidden, Cliffhanger |
| non | oui | 8 | Live Fire (+ ranked), Prism, Recharge, Fragmentation Heavies |
| oui | non | 4 | Vagabond, Highpower |
| non | non | 37 | Goliath, High Ground, Origin, Fortitude, The Pit, Snowbound... |

Ce que ca dit sur l'effort a fournir, du moins cher au plus cher :

1. **Fragmentation Heavies : un ALIAS de nom suffit, et c'est verifie.** La carte a bien
   ses 3 zones au catalogue, et ses bornes existent sous `fragmentation` — c'est le nom
   affiche qui ne retombe pas dessus, `filmdec.NormalizeMapName` ne retirant que le
   suffixe ` - ranked`. Que ce soit la MEME geometrie n'est pas une supposition : les
   trois zones de `fragmentation_heavies_map` et de `fragmentation_map` sont **au
   centimetre pres aux memes coordonnees** (26,57 / -10,27 / 6,10 · 51,02 / -7,47 / 3,81 ·
   75,50 / -4,61 / 6,20), memes rayons. Le repere est commun, les bornes s'appliquent.
2. **+8 matchs** avec `cmd/mapquant-build` sur Live Fire, Prism, Recharge — exige les
   fichiers du jeu installe (tags `sbsp`), donc un poste avec Halo Infinite.
3. **Vagabond et Highpower ne se traitent pas pareil**, et la nuance compte :
   - *Vagabond* manque au catalogue d'objectifs : `cmd/mapobj-build` (appel reseau a la
     variante UGC) l'ajouterait. C'est la carte du releve terrain de reference du
     2026-08-02 — **sans elle, l'oracle historique n'est pas rejouable par ce code**.
   - *Highpower* EST au catalogue mais **n'y declare aucune zone de Bastion** : ses trois
     variantes portent 5 `extraction_zone` et 0 `strongholds_zone`. Ce n'est donc pas une
     extraction manquante mais une donnee absente de la variante elle-meme ; la traiter
     demanderait de comprendre ou Halo place les zones de Strongholds sur cette carte.
4. Les **37 restants** sont des cartes communautaires / Forge : leur module n'est pas un
   BSP livre, les deux outils ne s'y appliquent pas tels quels.

**KOTH est hors de portee, et pas par manque de corpus** : la colline se DEPLACE en cours
de partie, ce que la variante de carte ne peut par construction pas decrire (etat de l'art
Forge/zones, §Q2(c), volontairement non ouverte). Le meme croisement sur `hill` mesurerait
une colline fixe qui n'existe pas.

## 5. La mesure

Corpus : 8 films, **525 actions de zone** identifiees par xuid (424 `zone_captures`,
101 `zone_secures`), toutes cartes a 3 zones. Commande :

	LEVELUP_REPO_ROOT=<repo> go run ./cmd/zone-attribution

### 5.1 Appartenance stricte : 12,2 %, et le premier temoin ment

| film | carte | identifiees | posees | dedans | dehors | sans position |
|---|---|---:|---:|---:|---:|---:|
| `68e58d18` | Forbidden | 56 | 54 | 2 | 38 | 14 |
| `f6a9d127` | Illusion | 59 | 58 | 7 | 27 | 24 |
| `10ed320d` | Forest | 85 | 85 | 18 | 43 | 24 |
| `28d77409` | Forest | 69 | 64 | 9 | 33 | 22 |
| `415f2c6c` | Illusion | 79 | 78 | 8 | 49 | 21 |
| `6d49207d` | Illusion | 54 | 52 | 4 | 33 | 15 |
| `b974a390` | Streets | 80 | 80 | 7 | 47 | 26 |
| `d4230304` | Cliffhanger | 54 | 54 | 9 | 30 | 15 |
| **total** | | **536** | **525** | **64** | **300** | **161** |

Zero action ambigue : aucune zone de Bastion n'en recouvre une autre sur ces 5 cartes.

Le **temoin spatial** (memes instants, zones translatees de 12 m) tombe a 0,4 % contre
12,2 %. Il a l'air excellent — et il ne prouve rien. C'est exactement le controle
tautologique que ce chantier a deja du retirer une fois : deplacer une zone de 12 m la
sort du sol foule, donc elle n'attrape plus personne, quel que soit le sens de la mesure.

### 5.2 Le temoin qui compte : decaler les INSTANTS, pas les zones

Le controle honnete est temporel : les MEMES actions, les MEMES zones, mais posees 30 s
plus loin dans le match (en enroulant). Un joueur de Bastion tourne autour des zones toute
la partie ; si « etre pres d'une zone » suffisait, le temoin ferait aussi bien.

| seuil (m) | reel | temoin temps | temoin espace | reel / pire temoin |
|---:|---:|---:|---:|---:|
| 0 | 12,2 % | **12,6 %** | 0,4 % | **1,0** |
| 2 | 25,1 % | 21,7 % | 5,0 % | 1,2 |
| 4 | 37,5 % | 33,0 % | 9,5 % | 1,1 |
| 6 | 49,1 % | 43,0 % | 15,4 % | 1,1 |
| 8 | 57,9 % | 50,3 % | 20,0 % | 1,2 |
| 10 | 64,8 % | 58,3 % | 28,0 % | 1,1 |
| 15 | 69,3 % | 65,3 % | 51,8 % | 1,1 |
| 20 | 69,3 % | 66,1 % | 65,5 % | 1,0 |

**Tel quel, le croisement n'apporte AUCUNE information** : a seuil strict le temoin
temporel fait meme legerement mieux (12,6 % contre 12,2 %), et le rapport reste a 1,0-1,2
sur toute la courbe. Un seuil relache ne sauve rien — il monte les deux courbes ensemble.

### 5.3 Ce n'est pas la geometrie, c'est l'HORLOGE

Deux mesures eliminent l'hypothese « les deux reperes ne coincident pas » :

- **ecart vertical median : -0,0 m** (p25 -0,6 · p75 +0,5). Les positions du film et les
  positions de la variante de carte sont dans le meme monde, a la dizaine de centimetres.
- **100 % des joueurs sont a moins de 20 m d'une zone** a l'instant d'une action
  (mediane 3,3 m, max 14,6 m). Personne n'est a l'autre bout de la carte.

Reste le confondant le plus probable : **le statborg replique ses compteurs a son propre
rythme** — intervalles de 1 a 31 s mesures au lot precedent (`HANDOFF_EVENEMENTS_NOMMES_2026-08-01.md`
§5). L'instant lu serait donc POSTERIEUR a l'action, et le joueur aurait bouge entre les
deux. Le test : reculer l'instant des actions et regarder si le taux monte pendant que le
temoin reste plat.

| decalage | reel (0 m) | temoin (0 m) | reel (5 m) | temoin (5 m) |
|---:|---:|---:|---:|---:|
| -10,0 s | 22,7 % | 12,6 % | 49,1 % | 36,6 % |
| **-5,0 s** | **28,6 %** | 11,2 % | 55,0 % | 36,8 % |
| **-3,0 s** | 27,2 % | 11,2 % | **55,2 %** | 35,0 % |
| -2,0 s | 18,9 % | 10,9 % | 54,7 % | 34,9 % |
| -1,0 s | 15,4 % | 11,8 % | 49,7 % | 37,1 % |
| -0,5 s | 13,3 % | 12,4 % | 46,9 % | 38,7 % |
| 0,0 s | 12,2 % | 12,6 % | 44,6 % | 39,0 % |
| +1,0 s | 11,2 % | 12,0 % | 40,2 % | 38,3 % |

**Le temoin est PLAT** (10,9 a 12,6 % a tous les decalages) pendant que le reel passe de
12,2 % a 28,6 %. Le rapport signal/temoin passe de **1,0 a 2,6**. Ce n'est pas un effet de
seuil : c'est une structure temporelle. **L'instant du statborg est en retard de quelques
secondes sur l'action qu'il compte.**

### 5.4 Le controle anti-ajustement : le pic, match par match

Un maximum trouve sur le corpus entier peut n'etre qu'un reglage sur le resultat qu'il
doit prouver. Le controle est donc de chercher le pic INDEPENDAMMENT sur chacun des huit
films — cartes, joueurs et dates differents.

| film | carte | pic | taux au pic | taux a 0 s |
|---|---|---:|---:|---:|
| `10ed320d` | Forest | -3,0 s | **65,9 %** | 21,2 % |
| `b974a390` | Streets | -5,0 s | **58,8 %** | 8,8 % |
| `d4230304` | Cliffhanger | -5,0 s | 40,7 % | 16,7 % |
| `6d49207d` | Illusion | -5,0 s | 21,2 % | 7,7 % |
| `28d77409` | Forest | -0,5 s | 15,6 % | 14,1 % |
| `f6a9d127` | Illusion | **-10,0 s** | 15,5 % | 12,1 % |
| `415f2c6c` | Illusion | **-10,0 s** | 12,8 % | 10,3 % |
| `68e58d18` | Forbidden | **-10,0 s** | 11,1 % | 3,7 % |

Ce que ce tableau etablit, et ce qu'il n'etablit pas :

- **ETABLI — la direction, a l'unanimite.** Les 8 films sur 8 piquent a un decalage
  NEGATIF ; aucun ne pique a 0 ni a +1 s. Sur huit tirages independants, c'est une
  propriete du format, pas un reglage. Le retard existe.
- **PAS ETABLI — sa valeur.** Les pics s'etalent de -0,5 a -10 s, et **trois d'entre eux
  tombent sur -10 s, c'est-a-dire sur la BORNE du balayage** : pour ces trois films le
  vrai maximum est peut-etre au-dela, et l'estimation est tronquee. Il n'y a donc pas de
  constante a appliquer — le balayage devra etre elargi avant de conclure.
- **PAS ETABLI — que la correction suffise.** Meme a son propre pic, la moitie du corpus
  reste sous 22 %. Deux films seulement (Forest, Streets) montent au-dessus de 55 %.

Enfin, l'ecart n'est pas non plus explique par la semantique de la statistique : au seuil
strict, `zone_secures` fait 16,8 % (17/101) et `zone_captures` 11,1 % (47/424). Une
recompense d'equipe attribuee a des coequipiers absents de la zone jouerait dans ce sens,
mais elle ne rend pas compte de l'ecart observe.

## 6. Decision : RIEN N'EST PERSISTE, et voici pourquoi

Le lot autorisait la persistance « uniquement si le resultat est fiable ET consommable ».
Il ne l'est ni l'un ni l'autre, sur trois criteres independants :

1. **Le taux.** 12,2 % d'appartenance stricte sans correction, 28,6 % au meilleur decalage
   global. Une colonne « zone de l'action » remplie une fois sur quatre, et vide le reste
   du temps, n'est pas consommable par une UI — et une valeur presente n'y serait pas
   distinguable d'une valeur sure.
2. **La correction n'est pas etablie.** Le retard d'horloge existe (8/8), mais sa valeur
   ne l'est pas, et trois films piquent sur la borne du balayage. Persister maintenant
   figerait dans la base un decalage arbitraire — exactement le genre de constante ajustee
   que ce chantier a deja du retirer ailleurs.
3. **Il n'y a AUCUN oracle de justesse.** Tout ce qui est mesure ici est un TAUX
   D'ATTRIBUTION : « une zone a-t-elle ete rattachee ». Rien ne dit que c'est la BONNE.
   Le seul oracle de ce type dans le chantier est le releve terrain du 2026-08-02, sur
   Vagabond — carte absente du catalogue d'objectifs, donc non rejouable par ce code
   (cf. §4).

Aucune table n'a donc ete creee, aucune migration ecrite, aucun champ ajoute a un document
publie. La recette ADR 0026 ne s'applique pas : il n'y a pas de nouvelle table.

**Ce qui EST livre** : le croisement lui-meme, pur, teste et branchable (§3), et l'outil de
mesure qui rend ce document reproductible en une commande. La prochaine tentative ne
repart pas de zero — elle repart d'ici.

## 7. Ce qui reste ouvert, par ordre de rendement

1. **Elargir le balayage d'horloge au-dela de -10 s** (`clockOffsets` dans
   `cmd/zone-attribution/report.go`). Trois films sur huit piquent sur la borne : la mesure
   est tronquee, et c'est le seul point ou une heure de calcul peut changer le verdict.
2. **Chercher a quoi le retard se correle.** S'il suit la cadence de replication du
   statborg (mesurable film par film, `HANDOFF_EVENEMENTS_NOMMES_2026-08-01.md` §5), la
   correction devient une fonction du film et non une constante — et le taux deviendrait
   peut-etre exploitable.
3. **Rendre l'oracle de Vagabond rejouable** : `cmd/mapobj-build` sur cette carte remet en
   service le seul releve terrain qui dise QUELLE zone, et pas seulement combien.
4. **Elargir le corpus** : l'alias « Heavies » (gratuit, §4.1) puis `cmd/mapquant-build`
   sur Live Fire / Prism / Recharge (+8 matchs). A 8 films, un taux par match a 3 points
   de pourcentage pres ne se distingue pas du bruit.
5. **La lettre A/B/C reste hors de portee du hors-ligne** : ni la variante de carte ni le
   film ne la portent. Elle demande un releve Theater. Tant qu'elle n'est pas etablie,
   `Zone.SpatialRank` est un rang stable, pas un nom.

**KOTH n'est PAS dans cette liste** : colline mobile, hors de portee par construction (§4).
