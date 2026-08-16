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

### Phase B — SURBOUCLIER : le bouclier des fiches suffit-il ?

- [ ] B.1 VERIFIER LA SEMANTIQUE de `Point.sh` sur pieces (normalise 0..1 ? clampe au max
      normal ?) — lire `replayNormalize`/le builder AVANT de mesurer, ne rien supposer.
- [ ] B.2 Sur les porteurs `i48` rang 9 : le bouclier depasse-t-il / sature-t-il d'une
      maniere DISCRIMINABLE d'un bouclier plein normal ? Temoin : vies aux autres rangs.
- [ ] B.3 Si `Point.sh` est clampe : mesurer cote film (`i5 object-shield-vitality`,
      deja decode) au lieu de l'artefact.

Gate B : verdict + la regle de detection si elle existe (« sh > X pendant Y »), sinon le
negatif chiffre.

### Phase C — DEPLOYABLES : trois signaux dates a croiser

- [ ] C.1 Publier la branche v==1 d'`i57` par hook : le R(2) et le **R(24)**, par slot et
      horodatage. 75 occurrences attendues sur `000d5950`.
- [ ] C.2 HYPOTHESE FALSIFIABLE, enoncee avant : le R(24) REFERENCE quelque chose —
      candidat : l'entite `ti=37` posee (handle slot+generation). Controle : croiser ses
      valeurs avec les slots `ti=37` VIVANTS au meme instant (bande de slots du film).
      S'il n'en croise aucun, l'hypothese tombe et on le dit.
- [ ] C.3 Croiser TROIS horloges sur les films famille B : naissances d'entites `ti=37`
      (premier record d'une vie d'objet), transitions `equipment-activated` (81 connues sur
      12 films), lectures v==1 d'`i57` des porteurs de rangs 19/22. Publier les
      co-datations par paires (fenetre large, l'ordre plutot que la fenetre quand un canal
      est clairseme).
- [ ] C.4 Les vies `ti=37` ont-elles une FIN lisible (dead-state, at-rest, disparition du
      masque) qui daterait la desactivation/expiration ? Compter, ne pas presumer.

Gate C : verdict « la pose d'un deployable se date / ne se date pas », et si oui par quel
signal, avec taux de co-datation.

### Phase D — MOBILITE : le croisement jamais fait, `i54` x `i48`

- [ ] D.1 Sur chaque film : episodes `i54` par vie, joints a l'identite `i48` de la MEME
      vie (meme slot). PREDICTION FALSIFIABLE : les episodes se concentrent sur les vies a
      rang de mobilite (4/5/6 fam. A, 20/21 fam. B) ; les vies a rang 1/2/12/23 n'en ont
      pratiquement aucun.
- [ ] D.2 Si la prediction tient : `i54` EST l'evenement d'usage date des equipements de
      mobilite, l'identite venant d'`i48` — publier episodes/vie par rang, et la
      distribution des durees.
- [ ] D.3 Si elle ne tient pas : `i54` est autre chose (glissade, escalade) — l'ecrire, et
      la mobilite reste sans instant d'usage par ce canal.

Gate D : le tableau episodes x rang, et le verdict.

### Phase E — `i59` tag==3 : compter sans decoder

- [ ] E.1 Compter et DATER les occurrences tag==3 d'`i59` (le corps `FUN_142f25e90` —
      position + quaternions — n'est PAS porte : on s'arrete au tag, on ne decode pas le
      corps, la marche du record s'arrete la pour ce record).
- [ ] E.2 Croiser ces instants avec les signaux de la phase C. S'ils co-datent avec les
      poses de deployables, le portage du corps (pose exacte du mur/de l'ancre) devient un
      lot justifie — a inscrire au registre comme reprise, PAS a executer ici.

Gate E : le compte, la co-datation, et la recommandation ecrite.

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
