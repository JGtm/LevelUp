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
| `equipmentPlacements` `origin: deployed` | Les DÉPLOIEMENTS d'objets sur la carte, par famille (`wall` / `sensor` / `other`) | Par pose, poseur mesuré | `t1` est une mise au repos, pas une disparition. ~5 % des poses sont `origin: unknown` |
| `equipmentPlacements` `origin: dropped` | Ce qui TOMBE à la mort : déployables **et bonus** | Par pose | Classé `dropped` à < 200 ms et < 1,5 m de la dernière position du porteur. Les deux populations sont séparées par trois ordres de grandeur |
| `equipmentChanges` | Les RAMASSAGES (`taken`) et les CONSOMMATIONS (`spent`), datés à la ms | Par VIE (`Slot`) | Les annonces de RÉAPPARITION en sont écartées. Témoin de complétude : ~16 émissions manquées sur 319 sur trois films ; **71 sur 1 954 = 3,63 % sur les 64 artefacts du parc** (mesure E0 du 2026-09-09, §2) |
| `grappleLines` | Les TRACTIONS de grappin — la seule activation de capacité mesurée et attribuée | Par VIE | — |
| `abilityCharges` | Les CHARGES RESTANTES, lues au changement | Par VIE | **Grappin et propulseur SEULEMENT.** Rien n'est transmis au ramassage, donc le maximum n'est pas établissable. Le répulseur n'arme jamais ce canal (négatif mesuré, rapport R11) |
| `abilityLabels` | Nomme les RANGS de capacité, palette propre au match | Match | Une capacité non classée ne reçoit aucun nom. **25,21 % des rangs lus par `equipmentChanges` ne sont pas nommés** — dont 8 artefacts sur 64 SANS table du tout (mesure E0, §2) |
| `padPickups` × `weaponPads` | Les socles d'ARME (famille en 8 hexa) et les socles de BONUS vidés (`powerup_*`) | Match, ramasseur nommé depuis le schéma 30 | Un socle de bonus n'est jamais rattachable à un joueur |

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

**B — Les équipements DÉPLOYABLES.** On les pose sur le terrain. « Utilisé » = **posé**.

Mur de protection (`wall`), capteur de menaces (`sensor`), écran occultant (`shroud`),
traqueur de menaces (`seeker`), champ de réparation (`field`), balise du translocateur
(`rift`) — tous par `equipmentPlacements` `origin: deployed`, avec le poseur mesuré.

> Le mur publie DEUX poses (l'appareil et ses panneaux) et compte pour UNE : filtre
> `WALL_PANEL_IDS` côté déployé. Un lâcher n'en publie qu'une, rien à dédoublonner.

**Dans les deux cas la question est la même** : servi, ou gâché. Seul le canal du « servi »
change. Le « gâché » est commun aux deux — §2.

## 2. Les trois issues d'un objet, et le canal de chacune

C'est le modèle validé par l'utilisateur le 2026-09-09. Les trois sont exclusives, leur
somme est le nombre d'objets ramassés SUR LA CARTE.

| Issue | Canal | État |
|---|---|---|
| **Utilisé** | Famille A : le canal d'activation de la famille (tableau §1 bis). Famille B : `deployed`. `spent` sert de témoin commun | Lu par la vue match, sauf translocateur et propulseur. **MESURÉ E0** : 416 objets sur 1 223 pris (34,0 %) |
| **Lâché en mourant** | `dropped` | Lu par la vue match. **MESURÉ E0** : 440 sur 1 223 (36,0 %) |
| **Gardé sans l'utiliser** | `taken` sans `spent` ni `dropped` | **À brancher** — le canal existe, aucun écran d'usage ne le lit. **MESURÉ E0** : 337 sur 1 223 (27,6 %), et 30 (2,5 %) restent NON EXPLIQUÉS |

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
| `DroppedObjects` | oui — **TOTAL SEULEMENT.** `DroppedByFamily` existe en mémoire mais **la DDL ne le porte pas** (`usage_summary.go:88`) |
| `GrenadesThrown` | oui (produit, non affiché) |
| `PadPickups` / `PadPickupsByWeapon` | oui — familles d'arme en 8 hexa |
| **`taken` / `spent` (ramassages, consommations)** | **NON** |
| **Gardé jusqu'à la fin du match** | **NON** |

**Conséquence directe, et c'est la seule vraie différence entre les pages :** la vue match
peut tout servir sans recuisson (elle lit le document) ; Sessions, Solo et Escouade ne
peuvent servir aujourd'hui ni la ventilation des lâchers par famille, ni le dénominateur des
ramassages, ni la troisième issue. Les leur donner = nouveau champ de résumé + **recuisson**.

`UsageSummaryRev` (`usage_summary.go:65`) est la clé de reprise du backfill : **changer une
règle d'attribution ici DOIT incrémenter cette révision**, sinon le backfill saute les
matchs à re-résumer.

## 4. Qui lit quoi aujourd'hui

- **Vue match, « Usages d'équipement »** (`features/match-replay/model/equipmentUsageLogic.ts`) —
  lit `grappleLines`, `equipmentEpisodes`, `equipmentPlacements` (deployed ET dropped),
  `grenades`. **Ne lit PAS `equipmentChanges`** : ni exploité, ni exclu, simplement jamais
  branché sur cet écran.
- **`equipmentChanges` est bien vivant ailleurs** : `abilityChargeLogic.ts`,
  `placementTeleport.ts`, `riftStations.ts`, `equipmentChangeSound.ts`. Le brancher sur la
  fiche d'usage n'est donc pas un défrichage.
- **Page Sessions** (`features/session-detail/`) — lit le bloc `usage` de la réponse, donc
  uniquement le tableau du §3.
- **Solo / Synthèse et Escouade** — **aucun bloc d'usage**, ni front ni contrat.

## 5. Décisions en vigueur, et l'amendement en attente

| Réf | Décision | Statut |
|---|---|---|
| D5 (vague C) | Les grenades sortent des blocs d'équipement — « ce ne sont pas des équipements » | Ferme |
| D6 (vague C) | La page Sessions ne prend que des graphes normalisés (parts en %, cadences par 10 min) | Ferme |
| D9 (vague C) | « Déployé » et « lâché » fusionnés en UNE colonne par famille, barre empilée, échelle commune | **CORRIGÉE le 2026-09-09** : les power-ups n'en sortent plus. La règle est « deux définitions de utilisé » (§1 bis), pas « bonus vs déployable » |
| — (2026-09-09) | Les deux lectures — « est-ce que je fais ma part » et « est-ce que je gaspille » — sont COMBINÉES en une barre : sa longueur est ma part de l'équipe, son remplissage est l'issue, le nombre à droite est mon TAUX D'UTILISATION | Ferme, à dessiner |
| — (2026-09-09) | On ne compte pas les charges | Ferme |
| — (2026-09-09) | Un objet gardé jusqu'à la fin du match compte comme NON UTILISÉ | Ferme, **à brancher** |

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
- ~~« Chaque page mérite une forme différente. »~~ → même bloc partout ; seules changent la
  fenêtre observée, la présence de coéquipiers nommés, et ce que la base a déjà cuit (§3).
- ~~« Les bonus doivent sortir de la barre. »~~ → non : ils ont juste une autre définition
  de « utilisé » (§1 bis). Le seul équipement qui doive rester hors barre est le
  **répulseur**, et pour une raison opposée : son usage n'est mesuré nulle part.

## 7. Références

- `internal/analysis/replay/document.go` — la liste des canaux et leurs réserves
- `internal/analysis/replay/document_equipment_changes.go` — `taken` / `spent`
- `internal/analysis/replay/usage_summary.go` — ce qui est persisté au grain session
- `features/match-replay/model/equipmentUsageLogic.ts` — les canaux lus par la fiche
- `.ai/HANDOFF_LECTURE_EQUIPEMENT_2026-09-04.md` — négatifs mesurés, pièges de mesure
- `.ai/PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md` — décisions D5, D6, D9 ; lots C5 et C6
