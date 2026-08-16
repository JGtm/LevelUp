# Plan — L'etat ACTIF d'un equipement : lire les changements d'etat dans le film

> Ecrit le 2026-08-16, a la demande de l'utilisateur : « j'aimerais que tu regardes plus en
> profondeur si on peut lire le composant actif et inactif, normalement on peut lire tous
> les changements d'etat dans le film ». Branche `feat/v75`, contrat `plan-execution`.
> AUCUN rendu, AUCUN son, AUCUNE ligne de production : le livrable est un ensemble de
> VERDICTS CHIFFRES PAR FAMILLE, avec instruments versionnes.

## La question, reformulee — et pourquoi elle a une reponse par FAMILLE

L'utilisateur a raison sur le principe : le film est le flux de replication, tout changement
d'etat TRANSMIS y est. Mais « actif/inactif » n'est pas UN composant : chaque famille
d'equipement a son canal candidat, et trois des quatre n'ont JAMAIS ete interroges — leurs
deserialiseurs sont portes et JETTENT leurs valeurs, exactement le defaut d'`i48` (corrige
le 14/08) et de `ti=37` (mesure le 15/08).

| famille | rang(s) | canal candidat de l'etat | grammaire | etat au 2026-08-16 |
|---|---|---|---|---|
| **Camouflage actif** | 8 (fam. A) | **`i28 unit-active-camo-state`** — LE composant NOMME | R(3) + porte + **dequant R(12)** (fraction) + 6 x (R1+optR12) | porte (`unit_weaponstate.go:668`), valeurs JETEES |
| **Surbouclier** | 9 (fam. A) | `i5 object-shield-vitality` — le bouclier DEJA publie par les fiches (`Point.sh`) | deja decode | semantique du champ a VERIFIER (normalise ? clampe ?) |
| **Deployables** (mur 19/2, capteur 22/12) | 19, 22 (fam. B) | naissance d'entite `ti=37` + transitions `equipment-activated` + **`i57` branche v==1** | i57 : R(2) ; si v==1 : **R(2)+R(24)** | i57 : 75 lectures v==1 sur `000d5950` — la SEULE branche qui paie 24 bits, jamais publiee |
| **Mobilite** (grappin 4/20, propulseur 5/21, repulseur 6) | 4, 5, 6, 20, 21 | episodes `i54 biped-mobility-action` (67 episodes dates, ~0,6 s) **x identite `i48` de la MEME vie** | i54 date deja | **le croisement i54 x i48 n'a JAMAIS ete fait** — c'etait la reprise n°1 du registre |

## Decisions prealables (tranchees, ne pas re-arbitrer)

1. **`i27 charges-remaining` : EN RESERVE, ne PAS l'executer** — decision utilisateur du
   16/08 (« on va la garder en tete si jamais on trouve rien d'autre »). Ce plan explore
   l'etat actif ; le compteur de charges reste le repli documente au registre.
2. **Ne pas rejouer les coincidences +/-1 s avec `i56`** — vice de conception identifie :
   `i56` est transmis ~1,6 fois par vie (2 088 lectures / 1 275 slots, mesure du 15/08).
   L'horodatage d'une LECTURE clairsemee n'est pas l'instant du changement : une fenetre
   serree etait structurellement condamnee, ce qui explique a la fois le 4,5 % (1 film) et
   le 1,2 % (12 films) sans dire si `i54` est ou non un usage. Pour un canal clairseme,
   tester l'ORDRE DANS LA VIE (la chute survient APRES l'episode, meme slot), pas une
   fenetre.
3. **Ghidra (projet `HI.rep`) = repli de NOMMAGE uniquement**, si les correlations ne
   nomment pas les valeurs (les 8 de `activated`, les 4 de `i57`). L'export `.c` du 31/07
   ne contient pas les fonctions equipement (verifie le 16/08 : 0 des 6 desers `ti=37`).
   Offline-pur : lecture du fichier, jamais de runtime.
4. **Publication par hook sur le deser de production** (patron `ability_rank.go` /
   `equipment_state.go`) — on ne relit jamais les bits a cote du deserialiseur.
5. **Un seul decodage filmdec par process** ; instruments gardes par variable
   d'environnement, `t.Skip` verifie, saut CI.

## Corpus

Les 12 films du lot `d4be4ab95` (modes verifies en base le 15/08). Par phase :

- Camouflage / surbouclier (famille A, porteurs mesures par `i48` le 14/08) :
  `084a804d` (rang 8 : 10 lectures ; rang 9 : 8), `06dfe6d9` (8:2, 9:2, 11:3), `00ba2e1c`.
- Deployables / mobilite (famille B, rangs 19-22 = mur/grappin/propulseur/capteur, releve
  Theater du 27/07, 2 rangs a controle de groupe) : `000d5950` (verite terrain 8 slots :
  3 grappins, 3 propulseurs, 1 mur, 1 capteur), `00502e52`, `07aa428d`.
- Temoins negatifs : `0014603f` (i48 jamais au masque), films Assassin (peu d'equipement).

## Phases

### Phase A — CAMOUFLAGE : `i28` est-il l'etat actif ? — CLOSE le 2026-08-16

- [x] A.1 Hook sur `consumeUnitActiveCamoState` (i28, biped) : `camoStateHook`
      (`ability_state_hooks.go`), instrument `i28_camo_test.go` (garde `I28_FILM`).
      Denominateurs sur 3 films : 13 902 lectures / 13 902 annonces au masque, 0 illisible
      (084a804d : 5 413 · 06dfe6d9 : 5 053 · 00ba2e1c : 3 436). DECOUVERTE : la « fraction
      principale » (R12 sous flag0/flag1) n'est JAMAIS transmise (0 sur 13 902) — le canal
      qui varie est dans la QUEUE (6 x R1+optR12) : queue[1] (quasi binaire 0/4095) et
      queue[2] (oscillation 2048/615, presente PARTOUT — pas un etat d'equipement).
- [x] A.2 CONTROLE PASSE, et il est EXCLUSIF : les transitions de `queue[1]` tombent
      integralement sur les vies rang 8 — 084a804d : 25/225 paires du groupe rang 8
      (11,1 %) contre 0/1335 (autres rangs) et 0/1138 (sans identite) ; 06dfe6d9 : 14/33
      (42,4 %) contre 0/1844 et 0/587. Les 23 lectures minoritaires (4095) tombent TOUTES
      sur des vies rang 8 (16 sur 10 vies · 7 sur 2 vies). Temoin inter-films : `00ba2e1c`
      (0 vie rang 8) = 0 transition et 0 valeur 4095 sur tout le film. `queue[2]` echoue le
      meme controle (transitions dans TOUS les groupes, 21-34 %) : ce n'est pas l'etat camo.
- [x] A.3 COURBE publiee (06dfe6d9, slot 531) : paliers 0 -> 4095 -> 0, cinq episodes dont
      un plateau de 16,2 s (18,23 s -> 34,46 s). PAS de rampe : le canal est un
      INTERRUPTEUR (0 ou 4095), l'activation se date au passage a 4095, la desactivation au
      retour a 0, a la precision de la retransmission pres (le canal n'est retransmis que
      lorsqu'il est au masque).

Gate A : PASSE — **l'etat camo SE LIT** : `i28` queue[1], binaire 0/4095, transitions
exclusivement portees par les vies rang 8 (39 transitions rang 8 sur 2 films, 0 sur 574
autres vies), temoin interne ET temoin inter-films. Ce que la mesure ne dit pas : la
semantique des bornes de dequantification (le quantum brut suffit, 0 contre 4095), et la
nature de queue[2] (oscillation universelle, non correlee au rang).

### Phase B — SURBOUCLIER : le bouclier des fiches suffit-il ? — CLOSE le 2026-08-16

- [x] B.1 SEMANTIQUE VERIFIEE SUR PIECES : `Point.sh` est CLAMPE. La chaine est
      `decodeObjectShieldVitality` (dequant [0, 4], `vitality.go`) ->
      `BipedPosition.ShieldAt()` (`offline_biped.go:99`) -> `ShieldFraction` (clamp a 1.0)
      -> `Point.Sh` (`replay/build.go`, decimateTracks). Tout depassement est ecrase AVANT
      la serialisation : l'artefact ne peut PAS discriminer un surbouclier.
- [x] B.2 OUI, DISCRIMINABLE cote film — regle : **quantum i5 > 64** (64 <-> 1.000 exact,
      dequant endpointExact [0,4] sur 8 bits ; le temoin de forme du rejeu donnait deja
      [0, 64] pour un film sans surbouclier). `084a804d` : groupe rang 9 = 4 187/4 828
      mesures au-dela (86,7 %, q max 223 = 3,498) contre **0**/50 180 dans les deux autres
      groupes. `06dfe6d9` : 902/977 (92,3 %) ; une vie etiquetee [23] depasse a 362/362 —
      TROU DE COUVERTURE d'i48 (une transmission ~unique par vie : l'identite lue a un
      autre instant de la vie), pas un contre-exemple : son profil est celui d'un porteur.
      Temoin inter-films : `00ba2e1c` (0 porteur rang 9) = **0**/36 322 mesures > 64.
- [x] B.3 FAIT — c'est la mesure ci-dessus : `i5` non clampe (champ `Shield` + quantum
      brut `Q`), capture par le MEME balayage de production (`ScanFilmBipedPositions`,
      CaptureDirs), instrument `i5_overshield_test.go` (garde `I5_FILM`).

Gate B : PASSE — **le surbouclier SE LIT cote film, PAS dans l'artefact**. Regle de
detection : `q_i5 > 64` (soit valeur > 1.0) ; episodes dates de 6,2 a 61,6 s, decroissance
par paliers depuis q max 223 (3,498). Ce que la mesure ne dit pas : la datation fine du
RAMASSAGE (les episodes commencent a la premiere mesure de bouclier de la vie), et la
couverture i48 laisse des porteurs non etiquetes (vie [23] ci-dessus).

### Phase C — DEPLOYABLES : trois signaux dates a croiser — CLOSE le 2026-08-16

- [x] C.1 Publie (hook `spartanAbilityHook`, instrument `i57_handle_test.go`, garde
      `I57H_FILM`, masques creux ET denses via `i57MatchDense` reutilise). `000d5950` :
      tags 0:693 · **1:75** · 2:613 · 3:33 — le compte attendu AU CHIFFRE PRES, 1 414
      marches sur 1 414, 0 cassee. `00502e52` : v==1 x37 · `07aa428d` : v==1 x193.
- [x] C.2 HYPOTHESE TOMBEE, et proprement : le R(24) n'est PAS un handle `ti=37`.
      (a) Ses valeurs sont quasi TOUTES UNIQUES (0, 0 et 3 valeurs repetees sur 75/37/193
      lectures — un handle re-reference se repete ; un compteur/horodatage non) ;
      (b) decompositions slot13+gen2 et gen2+slot13 : « vie vivante a ±2 s » = 1/75, 2/37
      et 6/193 (~1-3 %, niveau du hasard) ; (c) pas davantage un slot de bipede (1-3/n).
- [x] C.3 REFUTE par les temoins, aux DEUX echelles. Les naissances `ti=37` sont DENSES
      (4,7-5,3/s — melange bonus/equipements, le 0.6 non tranche de la phase 0) : toute
      fenetre >= 0,5 s sature, temoins decales ±5 s compris (100 % partout). Aux fenetres
      fines (±0,02-0,10 s, echelle du paquet), le reel ne bat PAS ses temoins de facon
      reproductible : `000d5950` ±0,10 s donne 73,7 % contre 52,6/31,6 (1,4-2,3x) mais
      `00502e52` donne 66,7 % contre 80,0/53,3 et `07aa428d` 35,7 % contre 64,3/57,1 —
      le reel passe SOUS le temoin sur 2 films sur 3. Les transitions `activated` sont
      introuvables a l'echelle d'un film (1 · 0 · 0 sur les trois films) et ne co-datent
      avec rien (0/19 a toutes les fenetres).
- [x] C.4 Compte : 2 659 / 2 625 / 2 678 vies par film ; duree mediane 67,8-145,9 s ;
      96-99 % finissent > 5 s avant la fin du film. La FIN est DATABLE par la disparition
      du masque, mais la CAUSE n'est lisible que pour ~1-2 % des vies (dernier record
      porteur d'`item-at-rest` : 21/46/21 vies ; d'`object-dead-state` : 16/5/7).

Gate C : PASSE — verdict : **la pose d'un deployable NE SE DATE PAS par ces canaux**.
`equipment-activated` est trop rare (1 transition sur 3 films), le R(24) d'i57 ne
reference ni les entites ti=37 ni les bipedes (valeurs uniques, semantique de type
compteur/horodatage), et la co-datation v==1 <-> naissances est un artefact de densite
(temoins au niveau du reel). Ce que la mesure ne dit pas : ce que le R(24) encode
(candidat : compteur/tick — non etabli), et si un sous-ensemble des naissances ti=37
correspond aux deployables (aucun champ d'identite — resultat 0.6 de la phase 0 confirme).

### Phase D — MOBILITE : le croisement jamais fait, `i54` x `i48` — CLOSE le 2026-08-16

- [x] D.1 Fait sur LES 12 FILMS (instrument `i54_rank_cross_test.go`, garde `I54X_FILM`,
      hook `mobilityActionHook`, jointure PAR VIE — aucune fenetre, decision n°2
      respectee). Denominateurs : 2 519 242 records delta biped, 21 188 masque∋i54,
      21 187 lus, 1 illisible, 20 595 flag1==1. LA PREDICTION TOMBE : episodes/vie
      MOBILITE **0,55** (445 vies, 244 episodes) contre AUTRES RANGS **0,45** (392 vies,
      177 episodes) — ratio 1,2, et AUTRES > MOBILITE sur 4 films sur 12. Rien de
      comparable a l'exclusivite du camo (39 transitions rang 8 contre 0). Temoin
      `0014603f` (i48 jamais au masque) : 631 lectures flag1==1 quand meme — le canal vit
      sans aucun equipement identifie.
- [~] D.2 Sans objet : la prediction D.1 est tombee (couvert par D.3).
- [x] D.3 ECRIT : `i54` est une action de mobilite GENERIQUE (glissade/escalade — durees
      mediane 0,62 s, p90 1,7 s, coherentes avec la mesure du 13/08), pas l'evenement
      d'usage d'un equipement. La mobilite reste sans instant d'usage par ce canal — MAIS
      le grappin recoit le sien par la phase E (i59 tag==3). Biais publie : le groupe
      « sans identite i48 » est construit sur l'union (vies identifiees ∪ vies a episodes)
      — il ne contient que des vies a episodes, son taux « avec episode » de 100 % ne se
      compare pas aux deux autres groupes.

Gate D : PASSE — tableau episodes x rang publie par film (12 logs), verdict : **`i54`
n'est PAS l'usage d'equipement**, prediction refutee chiffree.

### Phase E — `i59` tag==3 : compter sans decoder — CLOSE le 2026-08-16

- [x] E.1 Compte et date (instrument `i59_tag3_test.go`, garde `I59_FILM`, hook
      `abilityNonPredictedHook`, marche arretee A i59 — le corps n'est jamais parcouru).
      3 films famille B : `000d5950` 57 tag==3 (masque∋i59 1342, lus 1309, illisibles 33
      — les 33 sont les records ou i57 tag==3 casse la marche AVANT i59) · `00502e52` 103
      (1131/1077/54) · `07aa428d` 50 (1065/1038/27). Tous dates et attribues a leur slot.
- [x] E.2 CROISE — et la DECOUVERTE n'est pas celle attendue : les instants tag==3 ne
      co-datent PAS avec les signaux de la phase C (naissances ti=37 : reel ≈ temoins
      decales aux fenetres fines, 1,2-2x au mieux a ±0,02 s sans reproductibilite ;
      transitions `activated` : 0 co-datation, 1 seule transition sur 3 films). MAIS le
      croisement avec `i48` — le meme que la phase D — donne la semantique : **115 des 117
      lectures tag==3 a porteur identifie tombent sur des vies rang 20 = GRAPPIN** (46+2x0
      sur 000d5950, 45+2 [19] sur 00502e52, 24 sur 07aa428d ; les 93 lectures restantes
      sont sur des vies SANS identite i48 — le trou de couverture connu). Motif en PAIRES
      a ~0,15 s d'ecart (debut/fin d'un tir de grappin, vraisemblance non prouvee).
      `i59 tag==3` est l'evenement du GRAPPIN, pas une pose de deployable.

Gate E : PASSE — compte publie, co-datation C refutee, RECOMMANDATION : le portage du
corps `FUN_142f25e90` (position + quaternions) est un lot JUSTIFIE comme **ancre du
grappin** (candidat : point d'accroche + orientation), PAS comme pose de deployable —
inscrit au REGISTRE_REPORTS comme reprise, non execute ici (interdit du plan).

## Ce que ce plan ne fait PAS

- Aucun rendu, aucun son, aucun champ de document : c'est un lot de MESURE.
- Pas de hook `i27 charges-remaining` (decision n°1).
- Pas de portage du corps d'`i59` (phase E compte, elle ne decode pas).
- Pas de Ghidra sauf repli de nommage documente (decision n°3).

## Statuts et cloture

`[x]` / `[~]` (reference) / `[!]` (justification ecrite). Aucune case vide. Entree datee au
`.ai/thought_log.md` ; verdicts reportes dans `PLAN_EQUIPEMENT_TI37.md` (vision globale) ;
reports au `.ai/V7.5/REGISTRE_REPORTS.md` avec condition de reprise.

## Protocole de reprise

1. Lire ce fichier ; les gates disent ou en est le chantier.
2. `.ai/thought_log.md`, entree la plus recente « equipement ».
3. Verifier sur pieces avant de coder (le code a pu bouger).
