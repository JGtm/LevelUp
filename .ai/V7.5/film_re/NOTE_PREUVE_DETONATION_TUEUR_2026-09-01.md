# NOTE — La preuve de l'attribution détonation→tireur contre le VRAI tueur — 2026-09-01

> Clôt la traque de l'attribution des touches explosives. Le Workflow détonation avait laissé
> l'attribution « plausible non prouvée » faute de vérité terrain (harvest de morts grossier :
> 5 morts, 0 kill explosif). Cette note branche le **scan de kills ROBUSTE** (`robustCollectKills`)
> et confronte enfin l'attribution au dead-state. Instrument `deto_preuve_robuste_test.go`
> (garde `LOT1_TRAME_FILM` + `LOT1_SONDE_MAP` + `LOT1_MAXCHUNKS`).

## Le scan robuste résout le manque de morts — mais pas assez
Sur `000d5950` (Fiesta, cliffhanger, 27 chunks) : scan **grossier** (rejeu-par-chunk) = **5 morts** ;
scan **robuste** (marche + localisateur) = **26 morts**. Sur ces 26 : 16 tueur mappé, **7 kills
explosifs appariables** (mort à ≤ 8 u d'une détonation), **5 évaluables** pour la thèse.

## Le résultat terrain (n=5, minuscule)
| Voie | accord au vrai tueur (dead-state EnumB) |
|---|---|
| DÉTONATION→tireur (la thèse) | **4/5 = 80 %** |
| VICTIME→tireur (voie « réfutée » pour le splash) | **4/4 = 100 %** |
| Temporel-récent | 4/5 = 80 % |
| Naissance (oracle interne) | 2/3 = 67 % |
| Témoin spatial (victime décalée 30 u) | 0/1 = 0 % |
| Témoin tireur aléatoire (50 tirages/kill) | 50/350 = **14,3 %** (~1/7 roster) |

Les deux vraies voies battent NETTEMENT le hasard (14 %) et le témoin spatial (0 %). Mais **la
victime→tireur (100 %) égale/dépasse la détonation (80 %)** sur les KILLS.

## La conclusion FONDAMENTALE (pourquoi c'est le bout de la route)
Un **KILL** explosif est souvent un coup **DIRECT** (la roquette frappe la victime en plein → mort) :
la victime est donc **sur l'axe de visée**, et victime→tireur marche pour les kills. Or c'est
précisément le SPLASH **non-fatal** (victime hors axe) qu'on cherchait à attribuer — et il n'a
**AUCUNE vérité terrain** : le film ne grave l'instigateur qu'à la **mort** (dead-state). On ne
peut donc **jamais prouver** l'attribution d'une touche splash non-fatale — il n'existe rien à quoi
la comparer. La preuve par les kills teste le DIRECT, pas le splash.

De plus, pour les KILLS eux-mêmes, `killsource` attribue déjà l'arme/le tueur à 97,6 % — la
géométrie détonation n'apporte donc rien de neuf sur les kills.

## Verdict définitif
- **Attribution explosive non-fatale (splash)** : PLAUSIBLE (géométrie discriminante, bat le
  hasard) mais **INDÉMONTRABLE par construction** (pas de vérité terrain pour le non-fatal). Reste
  un **« estimé »**, à jamais.
- **Kills explosifs** : déjà couverts par `killsource` (97,6 %).
- **Échantillon** : de toute façon data-starved (5 kills explosifs évaluables) ; un harvest plus
  gros ne prouverait que le DIRECT.

**ON BOUCLE** : livrer la précision de la classe à DÉGÂT DIRECT (solide, prouvée) ; les explosifs
non-fatals en « estimé » optionnel, jamais présentés comme mesurés.
