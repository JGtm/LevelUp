# RÉFÉRENCE — Ce que le film mesure de l'équipement (2026-09-09)

> **Pourquoi ce fichier existe.** Les faits sur l'équipement étaient éparpillés entre quatre
> sources (le handoff du 2026-09-04, le plan de la vague C, les commentaires de
> `document.go`, le handoff session-usage) et j'ai répondu trois fois de mémoire, trois fois
> à tort, dans la même conversation du 2026-09-09 : « on ne mesure aucun ramassage
> d'équipement » (faux), « on ne sait pas si un camouflage a été utilisé » (faux), et une
> forme différente par page sans raison. **À LIRE AVANT toute affirmation sur l'équipement,
> y compris une réponse en conversation.** Chaque ligne porte sa référence de code ; si le
> code contredit ce fichier, le code fait foi et ce fichier se corrige dans le même commit.

## 1. Les canaux, un par un

Tous vivent dans le DOCUMENT DE REJEU (`internal/analysis/replay/document.go`), donc au grain
MATCH. Ce qui remonte au grain session est une autre affaire — §3.

| Canal | Ce qu'il mesure | Grain | Réserve mesurée |
|---|---|---|---|
| `equipmentEpisodes` | L'état ACTIF : **camouflage** et **surbouclier** seulement. Nombre d'épisodes, durée, frags pendant | Par VIE | Deux familles seulement, « parce que deux seulement sont mesurées — les autres restent sans état plutôt que devinés » (`document.go:147`) |
| `equipmentPlacements` `origin: deployed` | Les DÉPLOIEMENTS, et eux seuls : une pose que le film DÉSIGNE par un événement 103 `EquipmentSpawnedObject` — les panneaux de mur | Par pose | **DÉCISION UTILISATEUR DU 2026-09-15 (lot 1.9.1)** : `deployed` est désormais RÉSERVÉ à ce qu'un 103 désigne. Un objet PORTÉ n'y entre plus jamais — son lâcher volontaire à mi-vie sortait `deployed` et faisait dessiner un geste qui n'a pas eu lieu ; il sort `dropped`. `t1` reste une mise au repos, pas une disparition |
| `equipmentPlacements` `origin: dropped` | Ce qui TOMBE d'un porteur — déployables **et bonus** — quelle qu'en soit la cause | Par pose | **LU, PLUS MESURÉ, DEPUIS LE 2026-09-15 (lot 1.9.1)** : une pose est `dropped` quand le film ÉCRIT une MORT du poseur (tolérance 200 ms, mesurée : max 171,7 ms d'un côté, min 205,3 ms de l'autre, intervalle VIDE de 33,6 ms) OU une PRISE du poseur (`equipmentChanges.taken`, 50 ms — 103 des 108 prises retenues sont à moins d'une ms). **L'étiquette ne distingue pas les deux lâchers** (décision utilisateur) : la CAUSE se lit dans `coverage.placements.byCause`. La fenêtre temporelle de 200 ms qui classait ces poses A ÉTÉ RETIRÉE — elle ne décide plus rien |
| `equipmentChanges` | Les RAMASSAGES (`taken`) et les CONSOMMATIONS (`spent`), datés à la ms | Par VIE (`Slot`) | Les annonces de RÉAPPARITION en sont écartées. Témoin de complétude : ~16 émissions manquées sur 319 sur trois films ; **71 sur 1 954 = 3,63 % sur les 64 artefacts du parc** (mesure E0 du 2026-09-09, §2) |
| `grappleLines` | Les TRACTIONS de grappin — la seule activation de capacité mesurée et attribuée | Par VIE | — |
| `abilityCharges` | Les CHARGES RESTANTES, lues au changement | Par VIE | **Grappin et propulseur SEULEMENT.** Rien n'est transmis au ramassage, donc le maximum n'est pas établissable. Le répulseur n'arme jamais ce canal (négatif mesuré, rapport R11) |
| `abilityLabels` | Nomme les RANGS de capacité, palette propre au match | Match | Une capacité non classée ne reçoit aucun nom. **25,21 % des rangs lus par `equipmentChanges` ne sont pas nommés** — dont 8 artefacts sur 64 SANS table du tout (mesure E0, §2) |
| `padPickups` × `weaponPads` | Les socles d'ARME (famille en 8 hexa) et les socles de BONUS vidés (`powerup_*`) | Match, ramasseur nommé depuis le schéma 30 | Un socle de bonus n'est jamais rattachable à un joueur |

### Le canal des ÉVÉNEMENTS du film, et ce qu'il porte de l'équipement (2026-09-13)

**Un seul événement nommé parle d'un objet d'équipement qui apparaît : le type 103
`EquipmentSpawnedObject`, et il dit « une PIÈCE a été engendrée », jamais « un équipement a été
déployé ».** Sa deuxième référence désigne l'objet créé (index 13 bits, base 512, plus la
génération : 93,6 % de résolution contre 2,2 % au témoin de hasard, dt médian +49 ms, sur
25 films et 931 occurrences). Sur les 5 761 poses publiées du corpus il désigne **216 des 216
poses de panneau de mur** — et **AUCUNE pose d'un appareil PORTÉ**, ni déployée (0 sur 91) ni
lâchée (0 sur 4 853 `dropped`, aux 3 panneaux mal classés près). Conséquences, toutes deux
fermes :

1. **Le 103 ne tire pas à la mort** — l'affirmation inverse, tirée d'un appariement en TEMPS
   SEUL du rapport R5 §3.2, est réfutée.
2. **Le film ne porte aucun signal d'ÉVÉNEMENT de déploiement pour le capteur, le traqueur,
   l'écran occultant et le champ de réparation.** Leur origine ne se lit donc pas sur le 103 —
   et par conséquent **ils ne sortent JAMAIS `deployed`** (décision utilisateur du 2026-09-15).
   Ce qu'on lit d'eux, ce sont leurs LÂCHERS, par deux signaux écrits : la MORT du porteur (fil
   des morts, pont d'identité) et sa PRISE d'équipement (`equipmentChanges.taken`, qui dit qu'il
   a échangé). Les deux publient `dropped` ; la CAUSE va dans `coverage.placements.byCause`.
   **La fenêtre de 200 ms a été RETIRÉE du décodeur** : une pose qu'aucun des trois signaux ne
   couvre sort `unknown` (`byCause.none`), et une pose sans poseur mesuré aussi
   (`byCause.no_owner`).

Détail, méthode, témoins et commandes rejouables :
`.ai/V7.5/RAPPORT_F0_DEPLOIEMENT_103_2026-09-13.md`.

### Négatifs MESURÉS — ne pas les rechercher à nouveau

- **Le répulseur n'est dans AUCUN des neuf canaux jugés.** Cas décisif : un joueur le porte
  68 s, le film annonce lui-même la consommation de sa dernière charge, et le compteur reste
  muet — pendant qu'il compte le grappin de trois autres joueurs du même match.
- **Le propulseur EST mesuré** (`abilityImpulses`, schéma 38, validé 5/5 contre un relevé
  Theater). S'il n'a pas de colonne, c'est une DÉCISION, pas une absence de donnée.
- **On ne compte pas les charges** — décision utilisateur du 2026-09-09.

## 1 bis. DEUX FAMILLES D'ÉQUIPEMENT, DEUX DÉFINITIONS DE « UTILISÉ »

C'est la clé de tout le sujet, et c'est ce que j'avais raté (décision utilisateur du
2026-09-09). Ce n'est PAS « bonus contre déployable » : c'est la manière dont l'objet sert.

**A — Les équipements d'ACTIVATION.** On s'en sert sur soi. « Utilisé » = **activé**.

| Famille | Canal qui mesure l'activation | État |
|---|---|---|
| Camouflage | `equipmentEpisodes` (i28 queue[1], interrupteur mesuré) | mesuré |
| Surbouclier | `equipmentEpisodes` (i5 non clampé, règle q > 64) | mesuré |
| Grappin | `grappleLines` (tractions, fenêtre datée + point d'accroche) | mesuré |
| Translocateur | `translocations` (événement du film type 117, jamais déduit d'un seuil) | mesuré |
| Propulseur | `abilityImpulses` (schéma 38, validé 5/5 contre relevé Theater) | mesuré |
| **Répulseur** | **AUCUN** | **NÉGATIF MESURÉ** — 9 canaux fouillés (événements, i57/i59, tag 3, poses, i48, i54, i56, masque bipède, entité ti=37). Une colonne dirait « 0 utilisation » là où la vérité est « non mesuré » |

**B — Les équipements DÉPLOYABLES.** On les pose sur le terrain. « Utilisé » = **la charge
consommée** (`equipmentChanges` `spent`) — SAUF pour le mur, « utilisé » = **posé**.

Mur de protection (`wall`), capteur de menaces (`sensor`), écran occultant (`shroud`),
traqueur de menaces (`seeker`), champ de réparation (`field`), balise du translocateur
(`rift`).

> **CORRIGÉ LE 2026-09-10 (lot 5.5, rapport E0 question 5).** Cette ligne disait
> « tous par `equipmentPlacements` `origin: deployed` » : c'était faux pour tous sauf un.
> Une pose `deployed` sur un objet PORTÉ (`kind = "carried"` au manifeste) mesure un
> **lâcher volontaire à mi-vie** — l'objet qui tombe parce que son porteur en ramasse un
> autre — et non un déploiement ; `equipmentOrigin` ne pose qu'une question, « cette
> création est-elle à la fin d'une vie ? », et un échange à mi-vie y répond non. Mesure
> décisive : sur **202 consommations de charge** de ces familles, **ZÉRO** n'est couverte
> par une pose de la même famille du même joueur à moins de 2 s. Le **mur** est à **84 %**
> parce qu'il est le SEUL équipement du manifeste qui **engendre une pièce distincte**
> (ses panneaux, `kind = "deployed"`) : son `spent` tombe sur la pose de panneau (149 sur
> 242) et JAMAIS sur la création de l'appareil porté (0 sur 31). La frontière n'est donc
> pas une opinion : c'est le champ `kind` du manifeste
> (`config/titles/halo_infinite/mappings/replay_labels.toml`), transcrit côté Go en
> `usageFamiliesWithSpawnedPiece` et recollé au manifeste par un garde-rail.
> **Réserve** : `spent` sous-compte à son tour — là où les deux canaux existent (le mur),
> 118 consommations pour 252 poses de panneau. Ces familles passent de « aucune mesure »
> à « une mesure partielle », pas à la vérité.

> Le mur publie DEUX poses (l'appareil et ses panneaux) et compte pour UNE : filtre
> `WALL_PANEL_IDS` côté déployé. Un lâcher n'en publie qu'une, rien à dédoublonner.
> **Et c'est pourquoi son rapport déploiements/lâchers vaut 1:1 quand toutes les autres
> familles sont entre 1:5,3 et 1:13,0** (correctif E0 n° 1, documentaire) : sous la clé
> `wall`, le NUMÉRATEUR additionne 242 poses de panneaux et 31 créations de l'appareil,
> pendant que le DÉNOMINATEUR est presque entièrement l'appareil porté (236 lâchers sur
> 241) — un panneau n'existe qu'une fois déployé, il ne peut pas être lâché à la mort ;
> l'appareil seul est à **1:7,6**, en plein dans le couloir des autres familles. Ce n'est
> PAS une double publication du même geste (médiane 17 s entre une pose d'appareil et la
> pose de panneau la plus proche). Ce rapport ne se compare donc à aucun autre : ses deux
> côtés ne portent pas sur le même objet.

**Dans les deux cas la question est la même** : servi, ou gâché. Seul le canal du « servi »
change. Le « gâché » est commun aux deux — §2.

## 2. Les trois issues d'un objet, et le canal de chacune

C'est le modèle validé par l'utilisateur le 2026-09-09. Les trois sont exclusives, leur
somme est le nombre d'objets ramassés SUR LA CARTE.

| Issue | Canal | État |
|---|---|---|
| **Utilisé** | Famille A : le canal d'activation de la famille (tableau §1 bis). Famille B : `spent` — **SAUF le mur, qui reste sur `deployed`** (corrigé le 2026-09-10, lot 5.5 : voir l'encadré du §1 bis) | Lu par la vue match, sauf translocateur et propulseur. **MESURÉ E0** : 416 objets sur 1 223 pris (34,0 %) — chiffre d'AVANT le correctif ; côté résumé de session, `us6` fait passer le translocateur de 0 à 11 « utilisés » au parc, le traqueur de 3 à 5, l'écran de 7 à 10 |
| **Lâché en mourant** | `dropped` | Lu par la vue match. **MESURÉ E0** : 440 sur 1 223 (36,0 %) |
| **Gardé sans l'utiliser** | `taken` sans `spent` ni `dropped` | **BRANCHÉ** (vue match E2 ; résumé de session E3, révision `us4`) — dérivé, jamais lu d'un canal : `max(0, taken - utilisé - lâché)`, clampé. **MESURÉ E0** : 337 sur 1 223 (27,6 %), et 30 (2,5 %) restent NON EXPLIQUÉS |

### Mesure E0 du 2026-09-09 — ce que le canal vaut sur le parc

Instrument jetable (`replay/e0_gachis_research_test.go`, supprimé après mesure), 64 artefacts
du parc local au schéma 50 recuits le 2026-09-09. Sortie brute intégrale : journal de
`.ai/PLAN_EQUIPEMENT_GACHIS_2026-09-09.md`. Les cinq mesures :

| # | Mesure | Résultat |
|---|---|---|
| 1 | **Volumes** `equipmentChanges` | 1 880 changements publiés sur 64/64 artefacts : **1 422 `taken`** (75,6 %), **458 `spent`** (24,4 %), **aucun autre `Kind`**. 1 372 slots distincts porteurs (un slot = une VIE) |
| 2 | **Émissions manquées** (témoin de rotation) | **71 sur 1 954** publiées + manquées = **3,63 %**. 58 sauts de compteur, 95 émissions récupérées (schéma 38), 96 vies dont la première émission est hors norme, 1 répétition. 1 096 annonces de réapparition écartées |
| 3 | **Rangs de palette NON nommés** | **453 sur 1 797 = 25,21 %**. Deux causes disjointes : **148** (8,2 % des lectures) parce que **8 artefacts sur 64 n'ont AUCUNE table `abilityLabels`** — palette du film non classée ; **305** parce que le rang n'est établi nulle part (19 : 167 lectures, 10 : 95, 22 : 90, 32 : 1). Aucun rang nommé par le manifeste ne manque à la table d'un film qui en a une (0/453) |
| 4 | **Slots non rattachés à un joueur** | **21 événements sur 1 880 = 1,12 %** ; **17 slots porteurs sur 1 372 = 1,24 %**. Jointure par le registre d'identité PUBLIÉ dans l'artefact (`identity.bipedSlots`, bornes par vie) — les 64 artefacts portent tous la section |
| 5 | **Écart de l'identité `taken ≈ utilisé + lâché + gardé`** | Sur 488 couples (joueur, famille) : **médiane 0,00 %**, pire cas à dénominateur substantiel (≥ 5 prises) **20,00 %**. Toutes familles confondues : **30 fenêtres sur 1 223 (2,45 %)** se ferment par un `spent` qu'aucun canal d'usage ne voit. Pires familles : traqueur 20,7 % (6/29), capteur 13,9 % (5/36), champ de réparation 9,1 % (2/22) |

**Ventilation des issues par famille** (fenêtres de `taken`, une pose = une charge donc
dédoublonnée par fenêtre) :

| Famille | Pris | Utilisé | Lâché | Gardé | Non expliqué |
|---|---|---|---|---|---|
| répulseur | 375 | **0** | 210 | 161 | 4 (1,1 %) |
| grappin | 306 | 145 | 96 | 65 | 0 |
| mur | 128 | 50 | 54 | 24 | 0 |
| propulseur | 111 | 43 | 35 | 28 | 5 (4,5 %) |
| camouflage | 96 | 80 | 6 | 5 | 5 (5,2 %) |
| surbouclier | 82 | 73 | 6 | 3 | 0 |
| translocateur | 38 | 16 | 6 | 13 | 3 (7,9 %) |
| capteur | 36 | 4 | 11 | 16 | 5 (13,9 %) |
| traqueur | 29 | 3 | 12 | 8 | 6 (20,7 %) |
| champ de réparation | 22 | 2 | 4 | 14 | 2 (9,1 %) |
| **total** | **1 223** | **416** | **440** | **337** | **30 (2,45 %)** |

> Le **répulseur** est la démonstration du négatif mesuré : 375 objets pris, **zéro** classé
> « utilisé » — non parce qu'il ne sert jamais, mais parce qu'aucun canal ne le mesure. C'est
> la raison de la décision P4 (pas de ligne pour lui) ; une ligne dirait « 0 utilisation ».

> **Réserve de couverture des poses, remesurée** : 681 poses d'origine inconnue sur 11 438 =
> **5,95 %** (820 déployées, 9 937 lâchées). Le chiffre « ~5 % » du §1 tient.

**Ce que le total NE couvre pas.** 1 223 fenêtres pour 1 422 `taken` : les 199 restants
portent un rang sans famille connue ou tombent sur une vie non nommée. Et hors de toute
fenêtre de `taken`, le parc porte **2 024 usages et 9 495 lâchers** — l'équipement de
RÉAPPARITION, qui n'est jamais `taken` par construction. Le dénominateur des trois issues
est donc bien « objets ramassés SUR LA CARTE », jamais « équipement porté ».

**Deux pièges d'unité, tranchés :**

1. Pour un **déployable**, une pose est une CHARGE, pas un objet : un capteur pris une fois
   et lancé quatre fois donne 4 poses pour 1 objet. Le total d'une barre déployé/lâché
   n'est donc PAS un compte de ramassages — sauf à le prendre dans `taken`.
2. Pour un **équipement d'activation**, une activation est un objet : le rapport est 1:1.
   C'est pourquoi la rédaction initiale de la décision D9 (exclure les bonus de la barre)
   était trop large — **corrigée le 2026-09-09**, voir §5.

## 2 bis. LES ARMES SPÉCIALES SUIVENT LA MÊME GRAMMAIRE

Question posée le 2026-09-09 : peut-on lire une arme de socle comme un équipement ? **Oui**,
les trois canaux existent — mais « utilisé » y a une troisième définition : **avoir tiré**.

| Issue | Canal | Réserve |
|---|---|---|
| Prise | `padPickups` (ramasseur nommé depuis le schéma 30), `pickups` (événement natif `biped_pickup`, attribué), `weaponChanges` (qualifie prise / lâcher / échange) | Les trois se recoupent : là où deux voient la même prise, ils s'accordent (21/21 et 11/12, à moins de 500 ms) |
| Utilisée | `shots` — l'arme a tiré au moins une fois | Une arme prise et jamais tirée est un gâchis net |
| Lâchée | `weaponChanges` sur un lâcher — et la frame jusqu'à laquelle l'arme reste montrable au sol | Les ré-annonces d'une arme déjà portée au spawn sont écartées |

Rien de tout cela n'est persisté au grain session : voir §3.

## 3. Ce qui remonte au grain SESSION — et ce qui ne remonte pas

Le résumé persisté est `UsagePlayerSummary` (`internal/analysis/replay/usage_summary.go:67`).
**C'est lui, et lui seul, qui alimente la page Sessions.** Un canal absent d'ici n'existe pas
pour Sessions, Solo et Escouade, quoi qu'en dise le document de rejeu.

| Grandeur | Persistée ? |
|---|---|
| `GrapplePulls` | oui |
| `CamoEpisodes` / `CamoMS` / `CamoKills` | oui |
| `OvershieldEpisodes` / `OvershieldMS` / `OvershieldKills` | oui |
| `DeployedByFamily` | oui — **ventilé par famille** |
| `DroppedObjects` | oui — total ; `DroppedByFamily` **persistée depuis `us4`** (E3, 2026-09-09) |
| `GrenadesThrown` | oui (produit, non affiché) |
| `PadPickups` / `PadPickupsByWeapon` | oui — familles d'arme en 8 hexa |
| **`taken` / `spent` (ramassages, consommations)** | **oui depuis `us4`** — `TakenByFamily` / `SpentByFamily` |
| **Gardé jusqu'à la fin du match** | **oui depuis `us4`** — `KeptByFamily`, DÉRIVÉ (`max(0, taken - utilisé - lâché)`) |

**Mise à jour du 2026-09-09 (étape E3 du plan équipement).** Les quatre lignes en gras
ci-dessus étaient toutes « NON » jusqu'à ce jour : c'est ce que l'étape E3 a changé. Le
résumé porte désormais les trois issues, ventilées par famille, dans les colonnes
`taken_json` / `spent_json` / `kept_json` / `dropped_json` de `match_usage_players`
(migration `shared_match_usage_players_outcomes_v1`), et l'agrégat de session les publie sur
des grandeurs `equipment_<famille>` avec deux taux de référence qui EXCLUENT le joueur (P7).

**Trois points à ne pas re-découvrir :**

1. **Les quatre ventilations parlent le vocabulaire des POSES** (`powerup_camo`, jamais
   `camo`) — le web, lui, nomme un bonus par son ÉPISODE dans `kept` et par sa POSE dans
   `dropped`, et ponte les deux (`droppedFamilyOf`). Côté Go il n'y a qu'une clé.
2. **La jointure rang -> famille se fait sur la FAMILLE PUBLIÉE PAR LE DOCUMENT**
   (`abilityLabels[].family`), depuis le schéma 51 (lot 4.3 du 2026-09-10). Elle se faisait
   sur la RACINE DU LIBELLÉ, faute de mieux : le résumé est une fonction pure du document
   DÉJÀ CUIT et la palette du manifeste n'y était plus en main. Le document publie désormais
   la table du manifeste telle quelle, et la reconstruction par racine a disparu du Go
   (`equipmentOutcomeStems` supprimée). Ce qui reste écrit dans `usage_summary_outcomes.go`
   est le PÉRIMÈTRE du bilan (`equipmentOutcomeFamilies`), une décision produit — le
   répulseur, le grappin et le propulseur ont une famille et n'ont pas de ligne d'issue.
   Les deux bonus ont reçu leur `family` au manifeste dans le même lot (rangs 8 et 9,
   `powerup_camo` / `powerup_overshield`).
3. **Une recuisson reste nécessaire** pour que ces colonnes se remplissent sur les matchs
   déjà résumés : `UsageSummaryRev` est passée à `us4` le 2026-09-09, à `us5` le 2026-09-10
   (bascule sur la famille publiée) puis à **`us6`** le même jour (le « utilisé » d'un
   déployable sans pièce engendrée passe aux consommations — §1 bis), ce qui suffit à faire
   reprendre chaque match par `levelup backfill-usage-summary` (sans `--force`) — mais tant
   que cette passe n'a pas tourné, les colonnes sont vides et les grandeurs `equipment_*`
   absentes. **La recuisson des ARTEFACTS précède celle des résumés** : un artefact au
   schéma 50 ne porte aucune famille, donc son résumé us5 ne classerait rien. **`us6`, lui,
   n'exige AUCUNE recuisson d'artefact** : `spent` et sa famille sont publiés depuis le
   schéma 51. La chronique complète des révisions vit dans
   `internal/analysis/replay/usage_summary_chronicle.go`.

**Ce qui reste vrai :** la vue match peut tout servir sans recuisson (elle lit le document) ;
Sessions, Solo et Escouade lisent la base et ne voient donc jamais mieux que la dernière
passe de résumé.

`UsageSummaryRev` (`usage_summary.go:65`) est la clé de reprise du backfill : **changer une
règle d'attribution ici DOIT incrémenter cette révision**, sinon le backfill saute les
matchs à re-résumer.

## 4. Qui lit quoi aujourd'hui

> **Mis à jour le 2026-10-06 (lot L8.2 de `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`) :**
> l'onglet « Usages » des Séries temporelles est devenu l'Emprise solo, le bloc
> `equipment_usage` est supprimé (code et contrat). Mise à jour précédente : 2026-09-28 (lot L6.6
> de `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md`). Chemins Go relatifs à
> `apps/go-api/internal/`, chemins web à `apps/web/src/`.

Deux sortes de lecteurs. Ceux du DOCUMENT DE REJEU (vue match, rejeu 2D) lisent le film et
voient tout, sans recuisson. Ceux de la BASE (Sessions, Séries temporelles, Escouade) lisent
le résumé persisté (§3) et les niveaux de socle (§7) : jamais mieux que la dernière passe.

### Lecteurs du document (grain match)

- **Vue match, onglet Arsenal** : `features/match-view/MatchViewTabArsenal.tsx:181` monte
  `MatchEquipmentUsageSection` (`features/match-replay/`), qui lit le document par
  `useMatchReplay`. Le calcul vit dans `features/match-replay/model/equipmentUsageLogic.ts` :
  `grappleLines` (l. 335), `equipmentEpisodes` (l. 337), `equipmentPlacements` deployed ET
  dropped (l. 343), `grenades` (l. 352), et `equipmentChanges` `taken` (l. 355), qui sert au
  seul calcul du GARDÉ (`deriveKeptFromTaken`, `equipmentKeptLogic.ts:208`).
- **Le « utilisé » de la vue match** passe par `usageUsedOf` (`equipmentKeptLogic.ts:115`),
  qui nourrit aussi les colonnes (`equipmentUsageColumns.ts:176`). La liste des familles
  affichées passe par `familyHasAnyTrace` (`equipmentKeptLogic.ts:142`, appelée
  `equipmentUsageLogic.ts:434`) : une famille seulement CONSOMMÉE ouvre sa colonne
  (correction C2 du 2026-09-10).
- **Socles de la vue match** : `MatchPadControlSection` (`MatchViewTabArsenal.tsx:188`) lit
  `padPickups` (`model/padControlLogic.ts:184`).
- **Rejeu 2D** : `equipmentChanges` y est lu par `model/abilityChargeLogic.ts:137`,
  `model/placementTeleport.ts:210`, `model/riftStations.ts:118` et
  `sound/equipmentChangeSound.ts:39` (tous sous `features/match-replay/`).

### Lecteurs de la base (grain session et périmètre)

Tous passent par `platform/duckdb/session_usage_repo.go`, sur les vues `_latest` :
`match_usage_films_latest` (l. 59 : `duration_ms`, `powerup_pickups_json`),
`match_usage_players_latest` (l. 103 : épisodes, poses, les quatre ventilations d'issue),
`match_participants` (l. 179, camps) et `match_pad_pickups_by_tier_latest` (l. 243).

- **Page Sessions** : blocs `emprise` / `compare_emprise` de `POST …/pages/sessions/detail`
  (`domain/session_page.go:178`, `SoloEmpriseBlock` sans `maps`), produits par
  `service/solo_emprise_block.go:45` (`buildSoloEmpriseBlock`, appelé par
  `session_page_emprise.go:20`) sur le résumé d'usage lu UNE fois par session
  (`session_page_blocks.go:130`, `lireUsageDeSession`), partagé avec `formes_retenues` et
  l'effectif de camp de la coordination. Même carte « Équipement pris, et ce que j'en ai fait »
  que les Séries temporelles (ci-dessous), montée par
  `features/session-detail/useSessionEmpriseCards.tsx`, en vue compacte dans le tiroir de
  comparaison. L'ancien bloc `usage` (grandeurs `equipment_<famille>`, `sessionusage.ComputeUsage`,
  `service/session_page_usage.go`) est supprimé (plan `PLAN_SESSIONS_EMPRISE_2026-10-06`, S5,
  2026-10-06).
- **Séries temporelles, onglet Usages (Emprise solo)** : bloc `emprise` de
  `POST …/pages/timeseries` (`domain/timeseries.go:348`, `SoloEmpriseBlock` = le bloc de
  l'Escouade plus `maps` et `equipment`), produit par `service/timeseries_service_emprise.go:61`
  (`attachEmprise`, appelé `timeseries_service_sections.go:112`) sur le résumé d'usage lu UNE
  fois par `squadagg.LireUsage` (`timeseries_service_sections.go:127`), partagé avec
  `formes_retenues`. Les ressources (bonus, armes spéciales, râteliers) suivent les mêmes règles
  que l'Emprise de l'Escouade (ci-dessous). La carte **« Équipement pris, et ce que j'en ai
  fait »** lit `analysis/squademprise/equipment.go:45` (`BuildEquipment`) : servi / gardé /
  lâché par `sessionusage.PlayerOutcomeCounts` (l. 88, donc `equipmentUsedOf` : mur = posé,
  autres = charge consommée), pour MOI et pour le reste de MON camp, sur les matchs mesurés à
  camp connu ; le grappin et le propulseur, hors bilan, y portent une ligne « non mesuré » avec
  mes seuls lâchers (`equipmentusage.EquipmentUnmeasuredLineFamilies`, l. 26) ; le répulseur n'a
  aucune ligne. `formes_retenues` n'y porte plus que l'objectif (`domain/timeseries.go:373`).
  Web : `features/timeseries/TimeseriesPage.usages.tsx` monte `EquipmentOutcomesCard` (l. 117)
  et les cartes d'objectif (l. 108). Le bloc `equipment_usage` (« servi ou gâché » de période)
  est **supprimé** du code et du contrat depuis le 2026-10-06.
- **Synthèse** : **aucun bloc d'équipement** ; rien sous `features/synthesis/` ne lit
  l'équipement.
- **Escouade, onglet Emprise** (lots L4-L5, remplace l'ancien onglet Usages) : bloc
  `squad_emprise` de `/pages/teammates` (`domain/teammates.go:608`). Service
  `service/teammates/teammates_service_emprise.go` : résumé d'usage par `squadagg.LireUsage`
  (lecture partagée avec `formes_retenues`, `teammates_service_usage.go:77`), niveaux de socle
  (l. 144), et frags aux armes spéciales de la FEUILLE DE MATCH, hors film
  (`match_participants.power_weapon_kills`, `platform/duckdb/squad_emprise_repo.go:45-47`).
  Calcul pur `analysis/squademprise/` : les BONUS (camouflage, surbouclier :
  `sessionusage.PowerupFamilies`) y lisent prises, gardés et lâchés
  (`sessionusage.PlayerOutcomeCounts`), temps d'effet et frags pendant l'effet
  (`sessionusage.PowerupEffect` : `camo_ms` / `overshield_ms`) et socles vidés
  (`powerup_pickups_json`) — `squademprise/match.go:141` ; les ARMES SPÉCIALES et les
  RÂTELIERS y sont les niveaux `puissance` et `terrain` des socles (`match.go:170`,
  `input.go:103`). Aucun déployable, ni grappin, ni propulseur. Web :
  `features/squad/emprise/useEmpriseModels.ts:20`, sept cartes montées par
  `features/squad/SquadEmprisePage.tsx` (l. 90-170) ; l'onglet se retire quand le bloc n'a
  rien à montrer (`SquadLayout.tsx:243`).
- **Escouade, onglet Contributions** : **aucun canal d'équipement**. Il lit `formes_retenues`
  (`domain/teammates.go:597` ; `SquadContributionsPage.tsx:131-134`) pour la seule partie
  objectif de ses matchs (`features/squad/objectif/objectif.logic.ts:114`, `:123`), et ses
  cartes de frags lisent `frag_classes` (l. 61).
- **Escouade, onglet Dynamique** : **aucun canal d'équipement** (`SquadDynamiquePage.tsx`).
- **Escouade, bloc `equipment_usage`** : **retiré de `/pages/teammates` au lot L5.4**
  (`domain/teammates.go:594-596`) ; son seul lecteur était l'ancien onglet Usages.

### Une seule définition du « utilisé » côté Go

- **ÉCART GO / WEB — REFERMÉ LE 2026-09-10.** Le lot 5.7 a aligné le web (`usageUsedOf`) sur
  le résumé `us6`, et la **correction C1 de la revue de la vague 5** a aligné l'AGRÉGAT DE
  SESSION : `equipmentUsedOf` (`analysis/sessionusage/usage_outcomes.go:98`) lit les
  consommations d'un déployable sans pièce engendrée. Avant, un capteur `taken=3, spent=2,
  dropped=1, deployed=0` s'y affichait « utilisé 0 · gardé 0 · lâché 1 » quand la vue match
  affichait « utilisé 2 ». La liste des familles à pièce engendrée n'est pas recopiée :
  `equipmentusage.UsageFamilySpawnsPiece` (`domain/equipmentusage/families.go:139`),
  garde-rail `analysis/sessionusage/usage_outcomes_guard_test.go`.
- **L'Emprise lit la même définition** : `sessionusage.PlayerOutcomeCounts`
  (`usage_outcomes_counts.go:34`) passe par `equipmentOutcomeOf` puis `equipmentUsedOf`
  (`usage_outcomes.go:72-74`).

## 5. Décisions en vigueur, et l'amendement en attente

| Réf | Décision | Statut |
|---|---|---|
| D5 (vague C) | Les grenades sortent des blocs d'équipement — « ce ne sont pas des équipements » | Ferme |
| D6 (vague C) | La page Sessions ne prend que des graphes normalisés (parts en %, cadences par 10 min) | Ferme |
| D9 (vague C) | « Déployé » et « lâché » fusionnés en UNE colonne par famille, barre empilée, échelle commune | **CORRIGÉE le 2026-09-09** : les power-ups n'en sortent plus. La règle est « deux définitions de utilisé » (§1 bis), pas « bonus vs déployable » |
| — (2026-09-09) | Les deux lectures — « est-ce que je fais ma part » et « est-ce que je gaspille » — sont COMBINÉES en une barre : sa longueur est ma part de l'équipe, son remplissage est l'issue, le nombre à droite est mon TAUX D'UTILISATION | Ferme, à dessiner |
| — (2026-09-09) | On ne compte pas les charges | Ferme |
| — (2026-09-09) | Un objet gardé jusqu'à la fin du match compte comme NON UTILISÉ | Ferme, **branché** : vue match (E2), résumé de session (E3, `us4`) |

**Amendement proposé à D9.** Son raisonnement — les bonus ne se déploient jamais, donc leur
barre serait 100 % « lâché » — est juste **si la barre se construit sur le seul canal des
poses**. Mais un bonus activé est mesuré par `equipmentEpisodes`. Les bonus doivent donc
entrer dans la barre, avec le canal des épisodes comme côté « utilisé ». En attente de la
confirmation de l'utilisateur.

## 6. Ce qu'il ne faut plus dire

Trois affirmations fausses à ne pas répéter :

- ~~« On ne mesure aucun ramassage d'équipement. »~~ → `equipmentChanges.taken`.
- ~~« On ne sait pas si un camouflage a été utilisé. »~~ → `equipmentEpisodes`, et l'objet
  non activé tombe au sol à la mort.
- ~~« Chaque page mérite une forme différente. »~~ → même bloc sur les pages qui affichent le
  « servi ou gâché » (Sessions, Séries temporelles) ; seules changent la fenêtre observée, la
  présence de coéquipiers nommés, et ce que la base a déjà cuit (§3). **Corrigé le 2026-09-28
  (chantier Emprise)** : cette ligne disait « même bloc partout ». L'Escouade n'a plus ce bloc
  depuis le lot L5.4 (`equipment_usage` retiré de `/pages/teammates`) ; son onglet Emprise
  pose une autre question, camp contre camp, sur le bloc `squad_emprise` (§4).
- ~~« Les bonus doivent sortir de la barre. »~~ → non : ils ont juste une autre définition
  de « utilisé » (§1 bis). Le seul équipement qui doive rester hors barre est le
  **répulseur**, et pour une raison opposée : son usage n'est mesuré nulle part.
- ~~« Un déployable est utilisé quand il est posé. »~~ (2026-09-10) → seulement le **mur**.
  Pour les autres, une pose `deployed` est un **lâcher volontaire à mi-vie** ; leur usage se
  lit sur les **consommations de charge** (§1 bis).
- ~~« Le type 103 tire aussi à la mort. »~~ (2026-09-13) → non : 4 poses désignées sur 4 853
  `dropped`, dont 3 sont des panneaux de mur. L'affirmation venait d'un appariement en TEMPS
  SEUL (R5 §3.2) ; avec la référence résolue elle tombe.
- ~~« Une pose est un lâcher parce qu'elle tombe aux pieds du mort. »~~ (2026-09-13) → la
  distance ne classe plus rien côté équipement depuis l'item F.1.
- ~~« L'origine d'une pose est une mesure temporelle. »~~ (2026-09-15, lot 1.9.1) → elle se LIT,
  et par trois signaux écrits : l'événement 103 qui désigne la pièce engendrée, la MORT écrite du
  poseur, sa PRISE écrite. **La fenêtre de 200 ms a été retirée du décodeur**, et ses deux
  entrées sont sorties du registre des replis le même jour : une pose dont le film ne dit rien
  sort `unknown`. `coverage.placements.byCause` dit, film par film, ce qui a décidé.
- ~~« Un déployable posé à mi-vie est déployé. »~~ (2026-09-15, décision utilisateur) → non : un
  appareil PORTÉ qui tombe est **lâché** (`dropped`), que son porteur meure ou qu'il ramasse
  autre chose. `deployed` ne désigne plus que ce qu'un événement 103 nomme — les panneaux de mur.

## 7. Références

- `internal/analysis/replay/document.go` — la liste des canaux et leurs réserves
- `internal/analysis/replay/document_equipment_changes.go` — `taken` / `spent`
- `internal/analysis/replay/usage_summary.go` — ce qui est persisté au grain session
- `features/match-replay/model/equipmentUsageLogic.ts` — les canaux lus par la fiche
- `.ai/V7.5/HANDOFF_LECTURE_EQUIPEMENT_2026-09-04.md` — négatifs mesurés, pièges de mesure
- `.ai/V7.5/PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md` — décisions D5, D6, D9 ; lots C5 et C6

## 7. LES NIVEAUX D'ARME (ajouté le 2026-09-14)

> **À LIRE AVANT toute affirmation sur « les armes de base », « les armes de terrain » ou
> « les armes de puissance ».** Le niveau d'une arme n'est PAS une propriété de l'arme : c'est
> une propriété de l'EMPLACEMENT où elle se trouve, sur CETTE carte, plus l'équipement de
> départ de CE match. Répondre de mémoire produit ici exactement l'erreur que ce fichier
> existe pour éviter.

### Les trois sources, et aucune n'est un nom d'arme

| Niveau | Ce qui le mesure | Où |
|---|---|---|
| **Base** | l'arme est dans l'équipement de DÉPART d'une vie du match — canal `loadouts`, **PREMIÈRE émission de chaque `slot`** (un slot = une vie) | film |
| **Terrain** | l'emplacement de la carte qui confirme le socle est un RÂTELIER (`rack`) | `data/titles/{slug}/reference/map_weapon_pads.json`, croisé au socle du match à moins d'un mètre (`replay.BuildMapWeaponPads`) |
| **Puissance** | ce même emplacement est un SOCLE DE PUISSANCE (`power`) | idem |
| **Non classé** | aucun emplacement ne confirme le socle (carte hors référence, ou socle hors rayon) | reste VISIBLE avec son compte, jamais fondu ailleurs |

### Le négatif qui compte : LE RÔLE DE L'ARME N'EST PAS LE NIVEAU

Mesure du 2026-09-14 (76 artefacts, 669 socles confirmés) : **70 socles sur 669 (10,5 %)**
portent une arme dont le rôle canonique est « lourd » — Hydra (`power`), Needler et Sentinel
Beam (`special`), Shock Rifle (`sniper`) — **sur un RÂTELIER**, et c'est nominal : le jeu les y
pose. Juger au rôle produirait donc 10 % de faux niveaux. Le sens inverse (arme de rôle léger
sur un socle de puissance) est à **0,45 %**, et c'est lui, et lui seul, que le garde-rail
journalise (`service/replay_weapon_tier_check.go`, seuil 2 % des socles d'un match).

### Le piège du canal `loadouts`

Le canal **RÉ-ÉMET en cours de vie** après un changement d'arme. Prendre tout le canal au lieu
de la première émission de chaque slot dilue les trois armes de base de **94,4 % à 85,6 %** en
classé et fait monter le S7 Sniper de 0,73 % à 3,15 % — autrement dit, il transforme une arme
de puissance en arme de base. Un seuil de **5 % des vies du match** écarte en plus la queue
d'artefacts dont la première émission publiée arrive après un ramassage.

### Les trois états que les compteurs séparent, et que rien d'autre ne sépare

| État | Ce qui le dit | Ce que ça veut dire |
|---|---|---|
| aucune ligne en base | absence de ligne dans `match_pad_pickups_by_tier_latest` | **non mesuré** — la passe n'a pas eu lieu. Jamais « aucune prise » |
| `pads_total = 0` | colonne de match | le film n'a vu **aucun socle** : le mode n'en allume aucun. Mesuré sur le parc — **les 13 artefacts Super Fiesta sont à zéro socle**, et c'est pourquoi la ligne « Départs aléatoires » n'est visible sur aucune donnée réelle du poste |
| `pads_confirmed = 0` | colonne de match | des socles, mais la carte n'est pas dans la référence : **niveaux non établis**, tout tombe en `non_classe` |

### Modes à départs aléatoires

Le niveau « base » n'y est **jamais** publié (l'équipement de début de vie est tiré au sort).
La liste des modes concernés vit dans `config/titles/{slug}/mappings/regulation.toml`
(`[weapon_tiers] random_start_mode_tokens`), et le jeton s'y cherche comme un **MOT dans le
`pair_name` ENTIER**.

> **NI UN PRÉFIXE, NI UNE CATÉGORIE — mesure du 2026-09-14, faite après deux versions fausses.**
> Les formes que le registre porte réellement sont `Slayer:Arena Super Fiesta` (catégorie
> « Other »), `Slayer:Arena Fiesta` (« Other »), `BTB:Fiesta Slayer` et `BTB:Fiesta CTF`
> (« BTB ») — et ce sont les plus nombreuses (417 matchs Super Fiesta au seul plateau de score).
> La taxonomie du titre prend le préfixe avant le `:` et jette le sous-mode : elle ne les
> reconnaît pas. Sur elles, un niveau « arme de base » était écrit sur des équipements **tirés
> au sort**, sans la note qui l'aurait avoué. Le garde-rail
> `games/halo_infinite/weapon_tiers_categories_guard_test.go` rejoue ces formes, vérifie la
> cohérence avec la taxonomie (tout ce qu'elle sait classer comme aléatoire doit l'être aussi)
> et fige la mesure du négatif.

> **CE QUE L'ÉCRAN EN LIT.** Le caractère aléatoire est **servi** par le serveur — champ
> `weaponTiers.randomStarts` du document de rejeu (vue match) ; le compteur d'agrégat
> `pad_tiers.matches_random_starts` est parti avec le bloc d'usage de Sessions (2026-10-06). Le web n'en tient aucune liste : celle
> qu'il a tenue une semaine avait déjà divergé.

### Où ça vit

| Couche | Fichier |
|---|---|
| Règle pure (title-agnostic) | `internal/analysis/weapontier/` |
| Projection au fil de l'eau | `internal/sync/replayartifacts/padtiers.go` |
| Écriture INSERT-only | `internal/persist/pad_tiers_persister.go` (table `match_pad_pickups_by_tier`, vue `_latest`) |
| Rattrapage du parc | `levelup backfill-pad-tiers` (serveur ARRÊTÉ) |
| Ligne lue par l'Emprise | `internal/analysis/sessionusage/pad_tiers.go` (`PadTierRow`), `internal/analysis/squademprise/` |
| Vue match (résolution à la requête) | `features/match-replay/model/weaponTier.ts` |
