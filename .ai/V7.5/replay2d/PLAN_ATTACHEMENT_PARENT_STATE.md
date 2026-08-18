# Plan — Attachement d'un objet a un autre (`object-parent-state`, i10) : joueur DANS un vehicule, drapeau PORTE

> Ecrit le 2026-08-18 par la session de pilotage, a la demande de l'utilisateur (« dans le code on
> ne sait pas si un joueur est dans un vehicule ; comment les deux composants sont rattaches ? » ;
> « avoir la trajectoire du drapeau plus finement ; quand il sort de la zone jouable il respawn a
> son emplacement »). Contrat `plan-execution`. Mesure d'abord, production ensuite. Worktree FRERE.

## Comment un moteur fait ca (le modele general, et ce qu'on a deja sous la main)

Un objet « attache » (Spartan assis dans un vehicule, drapeau tenu, arme en main) n'est pas relie
par un evenement mais par une RELATION PARENT-ENFANT portee par l'ENFANT : un handle vers l'objet
parent + un point d'attache (siege / marqueur) + eventuellement une transformation locale. Tant
qu'il est attache, l'enfant NE replique PLUS sa position monde (elle se deduit du parent) ; quand il
se detache, il redevient un objet du monde qui replique sa position. C'est exactement ce que nos
mesures ont montre par ricochet : un objet pose « cesse d'emettre sa position » (poses, armes au
sol), et le drapeau porte se lit CHEZ LE PORTEUR (marqueur `0x00010005`, item 4 phase 0).

Dans le film, ce lien a un nom et il est DEJA PARSE : le composant **`object-parent-state-component`
(i10)**, present sur le bipede (ti=35), l'equipement (37), le corps rigide (38), le vehicule (40),
36 et 39 — table ECS `filmdec/testdata/ecs_table.tsv` (« le PARENT de l'entite (porte par un vehicule,
attache a un autre objet). Non mesure. »). Le deser Go `consumeObjectParentState`
(`filmdec/components_object_state.go:47`, miroir de `FUN_140c1e4d0`) lit : porte R(1) ; si porte :
un quant-stat 16 bits (handle du parent presume), R(16), R(16) optionnel, 2 bits, matrice 3 x R(16)
(offset/orientation d'attache), vitesse R(19) optionnelle, R(8), R(1) ; sinon un bloc `1408f0ac4` +
R(11) optionnel ; queue commune. **Tout est CONSOMME et JETE** : la grammaire est la, les valeurs ne
sont pas gardees. Aucune grammaire manquante, aucun Ghidra necessaire pour la premiere mesure ; Ghidra
ne sert que si le sens d'un champ (quel bit = quel handle) resiste a la mesure.

## Decisions tranchees

1. On lit les VALEURS d'i10 (hook de mesure, comme `SetEquipmentCreationHook`), sur le chemin DELTA
   (image par image) — pas d'image-cle, pas de bit-exact.
2. Deux oracles EXACTS existent deja : (a) CTF : `FlagGrabs`/`FlagSteals`/`FlagCaptures`/`FlagCarriersKilled`
   par slot a la ms (item 4) — le drapeau OBJET doit passer porte=1 avec un handle qui resout au slot
   du porteur au grab, et porte=0 a la fin ; (b) vehicules : un film BTB a vehicules (`084a804d`
   Fortitude Heavies ; verifier `ti=40` present) — un bipede a bord doit passer porte=1 avec un handle
   vers un slot `ti=40`, et sa position propre doit cesser d'emettre (ou emettre celle du vehicule).
3. Le drapeau OBJET : chercher d'abord dans les creations `ti=42` ECARTEES par le croisement d'identite
   (mot MPP hors catalogue d'armes) sur les 3 films CTF, pres des positions/instants de drop et de
   `flag_spawn` — s'il y est, on a sa trajectoire propre a l'image (lache, jete, hors zone => nouvelle
   creation au socle) ; sinon balayer les autres archetypes (37, 38) avec le meme oracle.
4. Seuils AVANT mesure : (a) >= 90 % des grabs suivis dans les 2 images d'un passage porte 0->1 de
   l'objet drapeau (ou du porteur : sens a etablir) avec handle -> slot du porteur ; temoin : un autre
   slot au hasard <= 5 % ; (b) vehicules : >= 90 % des periodes « bipede immobile dans le repere du
   vehicule » (position du bipede == position du vehicule a < 1,5 m sur >= 3 s) ont porte=1 avec
   handle -> ce vehicule ; temoin <= 5 %. Sinon negatif ecrit.
5. Production seulement apres les deux mesures : `attachments` [{child slot, parent slot, t0, t1,
   kind}] dans le document (schema +1), le drapeau publie sur SA piste quand il est libre et sur celle
   du porteur quand il est porte, le joueur en vehicule marque (icone) — lot separe, plan amende.

## Phases

- [x] 0.1 Hook de lecture d'i10 (valeurs brutes par record : ti, slot, gen, t, porte, champs) — instrument
      sous garde `ATT_FILM`, lecture seule. UNE edition de production, minimale et declaree :
      `filmdec.SetObjectParentStateHook` + le type `ObjectParentState` dans
      `components_object_state.go` ; le deser garde desormais les valeurs qu'il jetait, AUCUN bit
      lu ne change (memes lectures, meme ordre, memes largeurs ; la sonde est appelee en `defer`
      apres coup). Instrument : `replay/attachement_phase0_socle_test.go` (marche stateful
      `DecodeFrameViews`, rattachement EXACT lecture -> record par `CompResult.StartBit`).
- [x] 0.2 CTF (`64e8adfa`, `530820e5`, `53ce4390`) : le drapeau OBJET — creations `ti=42` ecartees
      (mot MPP hors catalogue) : combien, ou (distance aux `flag_spawn`), quand (distance aux grabs /
      drops de l'oracle) ; ses passages porte 0/1 vs l'oracle ; handle -> slot porteur ; seuil 4(a).
      RESULTAT : le drapeau EST identifie (mot `0x2A392328`, 3 films / 2 cartes, naissance a 0,0 m
      du socle, a 1 ms d'un evenement de l'oracle) ; le seuil 4(a) est REFUTE (0/149 fenetres,
      4 hypotheses x 4 tolerances x 2 sous-ensembles). Instruments : `attachement_phase0_ctf_test.go`
      (hypotheses de champ + porte du porteur), `attachement_phase0_drapeau_test.go` (objet),
      `attachement_phase0_cartes_test.go` (bornes et socles).
- [ ] 0.3 Vehicules (`084a804d` + un second film BTB choisi sur preuve `ti=40` present) : bipedes
      porte=1, handle -> `ti=40`, coincidence de positions ; seuil 4(b) ; temoin.
- [ ] 0.4 Publier au journal du plan (denominateurs, verdicts) ; sens des champs d'i10 etabli ou
      « non etabli » champ par champ.

**Gate 0** : au moins UN des deux oracles tenu (>= 90 %, temoin <= 5 %) => plan de production ecrit
(decision 5) ; aucun => negatif ecrit, condition de reprise = Ghidra sur `FUN_140c1e4d0` (sens des
champs) puis remesure.

## Regles dures

Mesure avant production ; seuils jamais rebaisses ; un seul decodage filmdec par process ; aucune
base ; films par chemin absolu ; JAMAIS `git add -A` ; ni journal ni registre depuis le frere (textes
au CR) ; RE image-cle fermee (rien ici ne la rouvre : chemin delta seulement).

## Decouvertes (hors perimetre — notees, NON traitees)

- **`param_4` d'i10 n'est mesure QUE sur le bipede.** `paramByComponent` (traverse.go) porte le
  vrai `param_4` par composant, releve sur la capture Cheat Engine — mais l'extraction porte sur
  l'archetype BIPEDE (ti=35) seul, et i10 n'y figure pas, donc il retombe sur le defaut 1. Le
  commentaire du code dit lui-meme « i10's keyframe desync is THIS parameter, not a code bug ».
  Un balayage `SetRecordStateParam` par archetype (offline-pur, deja outille) est donc une piste
  de reprise MOINS couteuse que Ghidra, et elle n'a jamais ete essayee pour ti=37/38/40/42.
  NON TRAITEE ici : hors perimetre de la phase 0, qui mesure la grammaire telle qu'elle est.

- **Le pont xuid -> slot de bipede est un pont vers PLUSIEURS slots.** Un slot de bipede vaut une
  VIE : le pont en nomme 92 a 122 pour huit joueurs. Toute table `xuid -> slot` (au singulier)
  perd dix vies sur onze, et si elle est construite par iteration de map, elle en perd une
  DIFFERENTE a chaque execution. Corrige dans l'instrument de la phase 0 (`attOracle.SlotsDe`) ;
  a verifier chez les autres consommateurs de `objBridge.SlotXUID` avant de s'y fier.

- **Les positions d'objet du monde de `ti=40` n'ont jamais ete controlees.** La refutation du
  2026-08-12 portait sur `ti=42` ; le meme decodeur sert `ti=40` sans qu'aucun controle
  d'etalement ni temoin fantome n'ait ete refait. L'item 0.3 pose ce controle ; ce qu'il en dit
  vaut pour tout futur usage cartographique des vehicules.

## Journal du plan

- 2026-08-18 — plan ecrit ; phase 0 lancee (agent Opus, worktree frere `../LevelUp-wt-attache`).

- 2026-08-18 — **ITEM 0.1 : la sonde d'i10 existe, et elle ne coute pas un bit.**
  `filmdec.SetObjectParentStateHook` publie, en UN appel par lecture, les onze champs que
  `consumeObjectParentState` lisait et jetait (porte, quant16, word16, opt16, deux drapeaux,
  matrice 3 x R(16), vitesse R(19), R(8), queue R(6)/R(3)), plus l'archetype, le param_4 et
  les positions de bit de debut et de fin. La grammaire n'a PAS bouge : le diff ne fait
  qu'affecter des valeurs jusque-la jetees, dans le meme ordre et aux memes largeurs, et la
  sonde est appelee en `defer` apres coup. C'est la SEULE edition de production de la phase 0.
  L'instrument (`replay/attachement_phase0_socle_test.go`, garde `ATT_FILM`, lecture seule)
  deroule la marche stateful de production (`DecodeFrameViews`, 4 vues par paquet, World
  reamorce a chaque image-cle) et rattache chaque lecture a son record PAR EGALITE de position
  de bit (`ObjectParentState.StartBit` contre `CompResult.StartBit`), jamais par voisinage :
  les essais de decodage de l'inference de chaine ne trouvent donc aucun record propre qui les
  reclame et sont comptes a part (« orphelines »).

  **Denominateurs (5 films, corpus complet).** i10 est RARE sur le chemin delta — c'est
  attendu d'un composant delta : il n'entre au masque que lorsque l'etat change.

  | film | paquets delta | records (propres) | lectures i10 | orphelines |
  |---|---|---|---|---|
  | `64e8adfa` Catalyst CTF | 50 956 | — | 547 | 12 251 |
  | `530820e5` Catalyst CTF | 29 148 | — | 370 | 6 080 |
  | `53ce4390` Behemoth CTF | 45 856 | 161 058 (120 112) | 997 | 12 774 |
  | `084a804d` Fortitude Heavies | 32 457 | 91 767 (63 497) | 942 | 9 460 |
  | `50247b26` Oasis BTB | 17 919 | 54 730 (38 173) | 444 | 4 150 |

  Reparti par archetype (`53ce4390`) : ti=38 388 lectures, ti=36 221, ti=37 172, ti=42 122,
  ti=41 44, ti=35 30, ti=39 13, ti=43 6, ti=40 1.

- 2026-08-18 — **ITEM 0.1, VOLET DE CONTROLE : le champ candidat ne designe AUCUNE entite
  vivante — le premier resultat negatif de la phase, et le plus solide.**
  Confronter un champ directement a l'oracle CTF empile trois hypotheses (le champ est un
  handle, l'entite lue est le drapeau, le calage d'horloge est bon) ; un echec ne dirait pas
  laquelle a cede. Le controle n'en teste qu'UNE, et le World la tranche exactement : *un
  handle d'entite pointe sur une entite qui EXISTE*. Temoin : un slot decorrele
  (`+4093 mod 8192`), meme World, meme appel.

  Le champ candidat est `Quant16 & 0x1FFF`, et ce n'etait pas un choix arbitraire :
  `readQuantStat(1, 13)` rend `(top2 << 30) | R(13)`, exactement la forme que `readRecordID`
  donne a un identifiant de record (13 bits de slot, 2 bits de generation), et les valeurs
  observees en avaient l'allure (0x40001B20, 0xC000040D, 0x800003C2).

  | film | lectures attachees | handle -> entite vivante | temoin | taux du hasard |
  |---|---|---|---|---|
  | `64e8adfa` | 193 | 19 (9,8 %) | 7 (3,6 %) | 4,4 % |
  | `530820e5` | 144 | 11 (7,6 %) | 12 (8,3 %) | 4,3 % |
  | `53ce4390` | 313 | 21 (6,7 %) | 25 (8,0 %) | 5,4 % |
  | `084a804d` | 305 | 34 (11,1 %) | 25 (8,2 %) | 8,8 % |
  | `50247b26` | 157 | 22 (14,0 %) | 19 (12,1 %) | 9,8 % |

  La mesure et son temoin sont indistinguables, et tous deux collent au taux de remplissage du
  World. Les couples (archetype enfant -> archetype parent) sont disperses sur ti=0, 2, 5, 6,
  9, 10, 11, 14, 15, 17, 21, 22, 32, 34, 37, 38, 40, 43, 47 sans qu'aucun ne se repete plus de
  5 fois : c'est un nuage, pas une relation.

  **RESTREINT AUX PAQUETS BIT-EXACTS** (toutes les vues ont atteint leur marqueur de fin ET
  aucun record n'a desynchronise — la seule preuve d'alignement disponible hors ligne, car un
  record dont tous les composants sont portes « reussit » meme s'il consomme la mauvaise
  largeur), le resultat ne s'ameliore pas, il empire : 3/38 (7,9 %) contre temoin 0/38 sur
  `64e8adfa`, 2/19 (10,5 %) contre 1/19 sur `530820e5`, 1/47 (2,1 %) contre 4/47 (8,5 %) sur
  `53ce4390`, 3/36 (8,3 %) contre 2/36 sur `084a804d`, 1/25 (4,0 %) contre 4/25 (16,0 %) sur
  `50247b26`. Aucun couple enfant->parent n'y apparait plus d'UNE fois. Part des paquets
  bit-exacts : 7,2 a 11,3 % selon le film.

  **DEUXIEME SIGNE, ET IL VISE LA PORTE ELLE-MEME.** Le bit de porte vaut 1 dans 23 a 44 % des
  lectures, UNIFORMEMENT sur les neuf archetypes (ti=35 a ti=43), et le filtre bit-exact ne
  change pas ce taux (23 a 33 %). Un bit qui dirait « je suis attache a quelque chose » serait
  RARE et tres inegal d'un archetype a l'autre. Un tiers constant partout est la signature
  d'un bit qui n'est pas la porte qu'on croit lire — ce qui rejoint ce que le code disait
  deja de ce composant : sa largeur branche sur `param_4`, valeur de RUNTIME, et le seul
  releve dont on dispose (capture CE) porte sur l'archetype BIPEDE uniquement. Rien ne
  garantit que `param_4 = 1` vaille pour ti=37/38/40/42.

- 2026-08-18 — **ITEM 0.2, PREMIER VOLET : les quatre lectures candidates contre l'oracle CTF —
  0 sur 149. Le seuil 4(a) n'est pas approche, il est a zero.**
  Quatre hypotheses de champ (`quant16 & 0x3FFFFFFF`, `quant16 & 0x1FFF`, `word16`, `opt16`),
  quatre tolerances (50, 250, 1000, 2000 ms), deux sous-ensembles (toutes les lectures /
  bit-exactes seules), trois films (82 + 33 + 34 = 149 fenetres de portage) : AUCUNE fenetre
  n'est suivie ni precedee d'une lecture attachee dont le champ designe une vie du porteur —
  **0/149 dans les 32 combinaisons**. Le temoin plafonne a 2/149 (1,3 %). Quand mesure et
  temoin valent tous deux zero, il n'y a pas de signal a comparer : c'est un negatif franc.

  Calage d'horloge mesure par film : 3 885 404 ms, 4 536 727 ms, 364 220 ms ; ecart median
  entre paquets delta : 817, 501, 317 ms — c'est la duree d'une « image » au sens du plan, et
  elle justifie d'avoir publie quatre tolerances plutot que la seule « 2 images ».

  **LE PONT N'EST PAS EN CAUSE, ET C'EST LE POINT DELICAT D'UN NEGATIF.** Une premiere version
  de l'instrument reduisait chaque joueur a UN slot de bipede (`xuid -> slot`), alors qu'un slot
  vaut une VIE : le pont en nomme 122 sur `64e8adfa` pour huit joueurs. Le defaut a ete corrige
  avant de conclure (`attOracle.SlotsDe` : xuid -> TOUS ses slots), et le resultat n'a pas
  bouge d'une unite. Le denominateur de champ le confirme de son cote : sur 193 lectures
  attachees de `64e8adfa`, 6 seulement designent un slot de bipede NOMME avec `quant16 & 0x1FFF`
  et 2 avec `word16` — soit le taux du hasard, pas un appariement manque.

- 2026-08-18 — **ITEM 0.2, SECOND VOLET : LE DRAPEAU EST TROUVE. C'est le resultat POSITIF de la
  phase 0, et il ne doit rien a i10.**
  Les creations `ti=42` ecartees par le croisement d'identite (mot de 32 bits du bloc MPP hors
  du catalogue d'armes) contiennent un mot qui se comporte exactement comme le drapeau de CTF :
  **`0x2A392328`**, present sur les TROIS films et sur les DEUX cartes.

  | film (carte) | ancres | acceptees | au catalogue d'armes | ecartees | mots distincts | `0x2A392328` : creations | dont a moins de 3 m d'un socle | distance min | ecart min a un evenement de drapeau |
  |---|---|---|---|---|---|---|---|---|---|
  | `64e8adfa` (Catalyst) | 18 306 | 674 | 352 | 322 | 174 | 110 | 41 | **0,0 m** | **55 ms** |
  | `530820e5` (Catalyst) | 6 786 | 368 | 239 | 129 | 66 | 41 | 16 | **0,0 m** | **69 ms** |
  | `53ce4390` (Behemoth) | 13 638 | 566 | 359 | 207 | 157 | 46 | 18 | **0,0 m** | **1 ms** |

  C'est le mot le PLUS FREQUENT des ecartees sur les trois films, le seul a naitre exactement
  au socle (0,0 m, la ou le deuxieme candidat `0x1D63A8CD` reste a 5,6-5,9 m et n'existe pas
  sur Behemoth), et le seul dont une creation tombe a la milliseconde d'un evenement de
  drapeau de l'oracle. Le croisement d'identite fonctionne par ailleurs (239 a 359 creations
  par film resolvent au catalogue d'armes), donc « ecartee » veut bien dire « pas une arme ».

  **ET SON i10 NE DIT RIEN.** Sur les slots de ces creations : `64e8adfa` 5/16 lectures a porte
  ouverte pendant un portage contre 11/25 hors portage ; `530820e5` 0/7 contre 1/4 ; `53ce4390`
  0/11 contre 0/2. Le camp « portage » n'est pas plus ouvert que l'autre — il l'est moins. La
  trajectoire propre du drapeau est donc a portee de main par le RECORD DE CREATION, et
  l'attachement au porteur ne l'est pas par i10.

- 2026-08-18 — **ITEM 0.3, PREALABLE : le second film BTB est choisi SUR PREUVE.**
  Une playlist « BTB Heavies » relevee au registre ne dit rien du flux ; des slots `ti=40` dans
  les images-cles, si. Recensement des trois premieres images-cles de huit candidats (tous BTB
  au registre parquet) :

  | film | carte (registre) | archetypes | slots `ti=40` | slots bipede |
  |---|---|---|---|---|
  | `a349fea8` | Fragmentation Heavies | 29 | **40** | 24 |
  | `084a804d` | Fortitude Heavies | 28 | 26 | 24 |
  | `268337cf` | Fragmentation | 29 | 22 | 23 |
  | `4db2574e` | Deadlock | 29 | 21 | 24 |
  | `9dfd3c24` | Deadlock | 30 | 20 | 21 |
  | `50247b26` | Oasis | 27 | 15 | 23 |
  | `13b00e35` | Highpower | 28 | 14 | 24 |
  | `3efe4592` | Nadair | 28 | 12 | 24 |

  `a349fea8` est retenu : c'est le film le plus riche en `ti=40`, sa carte est au catalogue de
  bornes, et c'est un mode a objectif (Total Control) donc un autre regime de jeu que le CTF de
  `084a804d`.

- 2026-08-18 — **ITEM 0.4 : le sens des champs d'i10, champ par champ.**
  La GRAMMAIRE reste ce qu'elle etait : elle est portee, et rien ici ne la conteste. Ce qui est
  etabli, c'est qu'AUCUN de ses champs n'a recu de sens. La colonne « etabli » ne dit pas « faux »,
  elle dit ce que la mesure autorise a affirmer.

  | champ | grammaire (param_4 = 1) | sens etabli ? |
  |---|---|---|
  | porte | `R(1)` | **NON, et l'inverse est suspecte** : ouverte dans 23 a 44 % des lectures, uniformement sur les neuf archetypes, filtre bit-exact compris. Un attachement reel serait rare et tres inegal. |
  | quant16 | sonde `R(1)` + `R(13)` + `R(2)` | **NON** : lu comme `(gen << 30) \| slot`, il ne designe une entite vivante qu'au taux du hasard (2,1 a 14,0 % contre temoin 0 a 16,0 %). Aucun couple enfant->parent ne se repete sur les paquets bit-exacts. |
  | word16 | `R(16)` | **NON** : 1 a 2 lectures sur 144-193 tombent sur un slot de bipede nomme. |
  | opt16 | `R(1)` puis `R(16)` | **NON** : 0 lecture sur un slot de bipede nomme, sur les trois films CTF. |
  | flagA, flagB | `R(1)` x2 | non mesure — aucun oracle de la phase 0 ne les vise. |
  | matrice | `3 x R(16)` | non mesure. Presque toutes valeurs distinctes (49 a 162 valeurs pour 52 a 171 lectures) : compatible avec un offset ou une orientation, mais rien ne le teste. |
  | vitesse | signe `R(1)`, si 0 -> `R(19)` | non mesure. |
  | byte8 | `R(8)` | non mesure — 24 a 48 valeurs distinctes selon l'archetype. |
  | queue | `R(1)` [`R(6)`] `R(1)` `R(3)` | non mesure. |
  | branche libre | bloc `1408f0ac4` (1 ou 16 bits) + `R(11)` optionnel | non mesure. La sonde publie la LARGEUR consommee, pas la valeur : la lire exigerait de toucher `consume1408f0ac4`, qui sert cinq autres composants. |
