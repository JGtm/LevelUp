# Événements projectile du film — `projectile_detonate` (0xC2/5) et `projectile_impact_effect` (0xC3/6-7)

Date : 2026-08-31 (worktree `wt/trame-proj`). Instrument :
`internal/analysis/filmdec/lot1_projectiles_research_test.go` (+ `_helpers_test.go`), garde
`LOT1_TRAME_FILM`, borné 12 chunks, `-count=1`. Films : `000d5950`, `01e1f945`, `00502e52`.

## Contexte

Trou ouvert par les notes `NOTE_ATTRIBUTION_ARME_TIR` et `NOTE_DAMAGE_SECTION_RESPONSE` : les
armes à PROJECTILE (M41 SPNKr, MLRS Hydra, Skewer, Ravager, Shock Rifle, Mangler, Stalker,
Bulldog, Fuel Rod) n'émettent PAS de `damage_aftermath` (0 % de touches en type 0), et le
type 1 (`damage_section_response`) ne porte pas d'attaquant (réfuté). L'utilisateur a désigné
LES ÉVÉNEMENTS PROJECTILE comme la voie. Question : `projectile_detonate` et
`projectile_impact_effect` permettent-ils d'attribuer les touches des armes à projectile à leur
TIREUR et à leur ARME ?

**Réponse : NON.** Comme le type 1, ces événements ne portent pas le tireur. Négatif chiffré
ci-dessous.

## Ce que l'exe dit (Ghidra, `HaloInfinite.exe`)

Les deux événements sont RÉELS, confirmés par leurs noms de chaîne et leurs descripteurs :

| Événement | Chaîne | Descripteur | Commutateur +0x58 | Lecteur charge +0x68 |
|---|---|---|---|---|
| `projectile_detonate` | `0x143c97858` | `0x143d0bae8` | `0x1408096ec` | `0x1408096f8` |
| `projectile_impact_effect` | `0x143c97838` | `0x143d0bb80` | `0x1408096ec` (partagé) | `0x1410f03b4` |

Filtrage paquet (dispatcher générique `FUN_14080a9d4` : `R(7)` = type, PUIS 3 slots de
référence gardés chacun par 1 bit, PUIS lecteur de charge) :
- `projectile_detonate` : octet **0xC2**, type **5** (`Skip(2)+R(7)==5`).
- `projectile_impact_effect` : octet **0xC3**, types **6 ET 7** (multiplexés par le bit 0 de
  l'octet 1).

**En-tête — UNE seule entité (le point décisif).** Le commutateur de domaine partagé
`0x1408096ec` est `test edx,edx ; jnz <assert> ; lea eax,[rdx+5] ; ret` : il rend
**domaine 5** pour le slot 0 et **`INT3` (assertion)** pour tout slot > 0. Donc dans un flux
valide seul le slot 0 porte une référence ; les bits de présence des slots 1 et 2 valent
toujours 0. C'est la différence STRUCTURELLE avec `damage_aftermath` (dom1/dom1/dom7, DEUX
entités blessé+responsable) : **un événement projectile n'a qu'UNE référence d'entité en
en-tête.**

Le lecteur de handle réel est `FUN_1406d3140` : largeur = `ceil(log2(capacité du pool))`
(`FUN_1406d310c`), lue dans la table RUNTIME `DAT_1451f98d0` — **VIDE dans l'image statique**.
La largeur du domaine 5 est donc une valeur d'exécution, différente d'un film à l'autre. Un
« sonde » n'existe que pour le domaine 1. Grammaire d'un slot présent : `R(1)` présence +
`R(width)` index + `R(2)` génération.

**Charge (au bit près pour les premiers champs, decompilation `FUN_1408096f8` /
`FUN_1410f03b4`) :**
- detonate : `R(6)` (`FUN_140809454`) ; `R(1)` porte ; si porte==0 : `[R(1) g ; si g : R(32)]`
  puis **`variant-name` `R(32)`** (`FUN_14080dec4`, littéral `"variant-name"`) ; suivent une
  direction `R(19)`, un déquantifié `R(5)`, `R(9)`, des drapeaux — non nécessaires ici.
- impact : `R(1)` porte ; si porte==0 : `[R(1) g ; si g : R(32)]` puis `variant-name` `R(32)` ;
  suit.

`variant-name` est le tag de VARIANTE du PROJECTILE, pas de l'arme.

## Ce que le film dit (3 films, mesuré)

Largeur du domaine 5 calibrée par film via l'invariant « slots 1 et 2 absents » (le
commutateur asserte au-delà de 0) — c'est l'oracle de cadrage de l'en-tête :

| Film | largeur dom5 (slots 1+2 absents) | detonate / impact | ref0 : distinctes (étendue) | coïncidence GLOBALE (proche / témoin +3 s) | BILAN LOURDES (proche / témoin) |
|---|---|---|---|---|---|
| `000d5950` | **8** (80,8 %) | 183 / 114 | 179 (0..254) | 51,4 % / 36,3 % | 50,6 % / 37,7 % |
| `01e1f945` | **8** (93,2 %) | 57 / 16 | 67 (0..136) | 18,0 % / 23,0 % | 33,3 % / 16,7 % (n=12) |
| `00502e52` | **10** (70,2 %) | 90 / 91 | 118 (1..1001) | 37,3 % / 22,8 % | 41,0 % / 26,2 % |

La largeur VARIE (8, 8, 10) : confirmation empirique que c'est bien une capacité de pool
runtime, pas une constante.

**1. ref0 n'est PAS le tireur.** Le domaine 5 rend des dizaines à ~180 valeurs distinctes,
étalées jusqu'à 254 (w8) ou 1001 (w10), n'atterrissant sur AUCUN slot bipède (l'argmax
lands-on-biped tombe hors de la bande 512). Un index de joueur/owner donnerait ≤ ~16 valeurs
répétées. C'est un **pool d'entités projectile** : ref0 est LE PROJECTILE qui détone/impacte,
pas celui qui l'a tiré.

**2. `variant-name` n'est pas une clé d'arme exploitable.** Quand elle est présente (taux très
variable : 24/297, 2/73, 134/181), elle est distincte à ~96-100 %. Ce n'est donc PAS un tag
catégoriel d'arme (une arme = un tag répété) ; intersection avec l'espace des `variant_name`
de TIR = 0 %. L'événement ne nomme pas l'arme utilement (soit espace projectile sans
catalogue, soit champ non catégoriel).

**3. La coïncidence temporelle tir lourd ↔ événement projectile ne discrimine pas.** Les
événements sont DENSES (73-297 par film sur 12 chunks), le taux de coïncidence de base sature
(témoin +3 s à 23-38 %, fenêtre large 2 s à ~100 % pour presque tout). Le bilan armes lourdes
reste 1,3-2,0× le témoin (sous le seuil 1,5× fixé), et le signal par arme est INCOHÉRENT d'un
film et d'une arme à l'autre : Shock Rifle 0 %, SPNKr 43,8 % = témoin, Skewer 85,7 % vs 28,6 %,
Stalker 100 % vs 40 %. Sur `01e1f945` le taux proche global (18 %) est même INFÉRIEUR au témoin
(23 %). Aucun lien fiable exploitable.

## Verdict

`projectile_detonate` et `projectile_impact_effect` **ne permettent pas** d'attribuer les
touches des armes à projectile à leur tireur ni à leur arme :
- pas de tireur/owner en en-tête (une seule réf, = le projectile lui-même) ;
- pas de victime résoluble en en-tête (une seule réf, non bipède) ;
- `variant-name` non catégorielle → ne nomme pas l'arme ;
- coïncidence temporelle non discriminante (densité, incohérence).

Les DEUX événements se comportent pareil sur ce point (impact ⊂ même en-tête, même
commutateur, `variant-name` identique) — pas de distinction detonate/impact utile à
l'attribution.

**Seule piste résiduelle, NON franchissable hors ligne** : relier le handle projectile (ref0,
domaine 5) au tir qui l'a engendré. Mais (a) le domaine 5 est un espace d'id distinct de
l'attaquant du tir (domaine 1) ET des slots de trajectoire ti=41 (`projectiles.go`), (b) la
liaison handle→slot vit dans une table runtime (comme la largeur), absente de l'image et du
film. Le tir n'expose pas non plus le handle du projectile engendré. Il n'existe donc pas de
pont offline. Le trou de précision des armes à projectile RESTE OUVERT.

Garde-fou respecté : négatif chiffré, rien survendu ; ce qui MARCHE (grammaire d'en-tête,
calibrage de largeur, confirmation que les événements existent et sont décodés) est distingué
de ce qui NE marche pas (attribution).

## Gate

`gofmt` propre, `go vet ./internal/analysis/filmdec/` vert (GOCACHE privé). Instrument sous
garde `LOT1_TRAME_FILM`, borné 12 chunks.
