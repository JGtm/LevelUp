# Plan — La ligne du grappin : porter l'ancre, la prouver, la tracer

> Ecrit le 2026-08-16. Demande utilisateur : « pour le grappin pas besoin du son mais on
> pourrait avoir une ligne blanche pour indiquer que le joueur l'utilise et se dirige vers
> le point d'accroche ». Branche `feat/v75`, contrat `plan-execution`.

## Ce qui est etabli (mesure du 16/08, phase E de PLAN_ETAT_ACTIF_EQUIPEMENT)

- `i59 biped-spartan-ability-non-predicted-state`, branche `tag==3` : **210 occurrences**
  comptees sur les films famille B ; **115 des 117 lectures a porteur identifie tombent sur
  des vies rang 20 (grappin)**, 0 autre rang. Les occurrences vont par PAIRES a ~0,15 s.
- Le corps `FUN_142f25e90` n'est PAS porte : le decompile dit « vecteur position +
  quaternions + dequantifications, switch sur un octet d'etat ». **Le decompile est DANS le
  depot** : `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` (section i59). Pas de Ghidra requis.
- Aujourd'hui la marche s'arrete proprement sur ces records (`consumeBipedSpartanAbilityNonPredictedState`,
  `components_biped_ability.go` — le cas commun tag!=3 est bit-exact, le corps tag==3 non).
- Hook existant a etendre : `SetAbilityNonPredictedHook` (`ability_state_hooks.go`).

## Objectif et critere de succes

Une ligne du joueur vers le point d'accroche pendant l'usage du grappin, alimentee par une
POSITION MESUREE dans le film — jamais une position devinee. Pas de son (decision
utilisateur). Si le corps ne rend pas une position fiable, le negatif s'ecrit et rien ne se
trace.

## Phases

### Phase 0 — PORTER le corps, et le PROUVER — CLOSE le 2026-08-16

- [x] 0.1 PORTE — mais PAS depuis le decompile seul : ses hypotheses d'enrobage ont ete
      REFUTEES par la marche (les 4 lectures de tag interne testees n'expliquaient que 2 a
      4 des ~62-162 bits reels du corps), et la grammaire a ete LUE dans les films
      (TestI59AnchorBodyDump / TestI59AnchorTemplate : champs complets bornes par le record
      suivant, consensus par position de bit, paires alignees). Resultat
      (`components_biped_anchor.go`) : `[R(3) interne 1=tir|2=accroche][R(3) drapeaux]
      [X][Y][Z aux largeurs de CARTE][R(7)][gate8 1+8?]` + corps lourd `3 x [porte;
      dir24+mag12] + R(24) + R(9)`. Les largeurs de FEUILLES du decompile (0x18/0xc, 0x18,
      +9, gate8) sont CONFIRMEES ; les deux ac4 de tete n'existent pas dans le flux ; le
      bloc commun (drapeaux + position + R7) manquait aux notes. La POSITION est aux
      largeurs `MapQuantEntry` (prevu par le plan) : +10 bits exactement entre Cliffhanger
      (40) et Bazaar (50), les deux classes — c'est ce qui a identifie le champ. Drapeaux
      != 000 (8/210) et interne 4 : ported=false, desync propre. Deux corrections
      transverses : la queue R(3) d'i59 (param_4=2) n'etait JAMAIS lue offline
      (paramByComponent n'avait que les cles d'i57 — ajoutees) ; garde-rail
      WorldObjectPrecision : entree allowlist datee.
- [x] 0.2 PREUVE PUBLIEE, en plus fort que demande : au lieu de « marche aboutie », l'ecart
      entre la fin de marche et le record biped suivant. AVANT : ecarts 62-175 bits (le
      corps entier manque). APRES : min=p10=p50=p90=0 sur les TROIS films — 000d5950
      48/48, 00502e52 72/72, 07aa428d 33/33 ecarts a ZERO, 0 chevauchement, temoins tag!=3
      a 0 partout (988/713/761). Marche aboutie : 57/57, 100/103, 46/50 (les 7 casses =
      desyncs propres voulues : drapeaux/interne inconnus).
- [x] 0.3 CONTROLES PASSES sur 3 films / 3 cartes, denominateurs publies :
      (a) EMPRISE 100 % : 57/57 (Cliffhanger), 100/100 (Bazaar), 46/46 (Illusion) dans
      l'AABB du nuage biped ;
      (b) PORTEE [2,40] u : 53/57, 99/100, 44/46 ; DECROISSANCE d(t+1s)<0,8 d(t) :
      accroche 21/23, 47/47, 17/20 contre temoins melanges 8/23, 14/47, 4/20 ; d mediane
      4,7-5,7 u -> 1,3-2,1 u a t+1s puis REMONTE a t+2s (le joueur atteint l'ancre et
      repart). FIXITE inter-membres |P1-P2| = 0,05-0,07 u quand le joueur bouge de 0,42 u :
      P est un point monde FIXE. L'ancre est prouvee.
      (c) PAIRES : ecart tir->accroche 0,150 s CONSTANT (p10=p50) sur les 3 films ;
      25/48/22 paires ; tirs sans accroche = rates. FENETRE DE RENDU MESUREE : du tir a
      l'arrivee (minimum de distance, mediane ~1 s apres l'accroche, borne 2 s).
- [x] 0.4 Instrument `i59_anchor_test.go` (5 tests), garde `I59A_FILM` (+ `I59A_MAP` si la
      signature de largeurs est ambigue — Chasm/Illusion), saute partout ailleurs, un film
      par processus, LockProcessDecode, precision carte installee et restauree par test.

**Gate 0 : PASSE** — marche a l'ecart zero, (a) 203/203, (b) decroissance massivement
au-dessus des temoins, fixite au pas de quantification. Ce que la mesure ne dit pas : la
semantique des vecteurs dir/mag du corps lourd (jamais presents sur les deux membres d'une
paire -> fixite non calculable ; candidats : velocite/cable), celle du R(24)/R(9) de queue,
et la grammaire des 8 records a drapeaux != 000.

### Phase 1 — PUBLIER — CLOSE le 2026-08-16

- [x] 1.1 Publie : `grappleLines` = `{slot, t0, t1, ax, ay, az}` PAR VIE
      (`replay/grapple_lines.go`). ECART assume vs la lettre du plan (`tMs`) : t0/t1 sur
      L'AXE DES FRAMES (le meme que `Point.T` et `equipmentEpisodes` — la convention du
      document, aucun recalage cote client). L'altitude est GRATUITE (le meme champ la
      porte) : publiee (`az`). FENETRE MESUREE, pas choisie : t0 = le TIR (corps leger
      apparie a <= 0,5 s ; l'accroche seule si le tir n'a pas ete lu — on ne recule pas un
      debut non lu), t1 = l'ARRIVEE — l'argmin PAR TRACTION de la distance
      trajectoire->ancre dans les 2,5 s suivant l'accroche (la mesure 0.3 : minimum ~1 s
      puis remontee). Un tir sans accroche est un RATE : compte, jamais trace.
- [x] 1.2 Schema 7 -> 8 (chronique `document.go` + ratchet `structure_test.go`) ; contrat
      29 -> 30 champs (chronique `contracttest/replay_contract_test.go`, GrappleLine +
      GrappleCoverage decrits) ; OpenAPI regenere (`make openapi-gen`) ; `generated.ts`
      regenere ; frontiere web (`replayNormalize` + `NULLABLE_ARRAYS` + type expose) ;
      golden : fixture d'entrees v5 (`REPLAYINPUTS5`, + GrappleReads — 57 lectures sur
      000d5950) et assemblage refige EXPLIQUE (`renderGrapple` : 25 tractions / 15 vies ·
      32 tirs + 25 accroches · 7 rates · 0 corps casse). Couverture publiee dans le
      document (`Coverage.grapple`).
- [x] 1.3 `Options.GrappleReads` (entree de DONNEES) ; `ScanFilmGrappleReads` dans
      `BuildFromFilm`, absence NON FATALE avec warn + stats logguees. Artefact temoin
      re-cuit : `000d5950.json` au schema 8 (25 tractions ; 00502e52 et 07aa428d n'ont
      pas d'artefact local — rien a re-cuire pour eux, le re-build de masse reste
      l'affaire du registre).

### Phase 2 — TRACER — CLOSE le 2026-08-16 (reste le gate VISUEL utilisateur)

- [x] 2.1 `grappleLayer.ts` : ligne position-COURANTE du joueur (`positionAt`, la meme
      interpolation que le marqueur) -> ancre, projetee par `worldToCanvas` (la chaine des
      tracks). Dessinee entre les trajectoires et les effets de tir (elle se lit sur la
      trajectoire sans couvrir les evenements). Teste (`grappleLayer.test.ts` : jointure,
      fenetre stricte, geometrie interpolee, encre de l'appelant).
- [x] 2.2 « Blanche » = `readInk('--foreground')` — l'encre la plus claire du theme sombre,
      re-resolue au changement de theme, ZERO hex (le lint et le garde-rail des couleurs
      ne voient aucune valeur). Epaisseur 1,25 px, alpha 0,85, aucun halo — sous la densite
      des effets de tir (2-3 px + halo) ; petit disque (r=2) au point d'accroche.
- [x] 2.3 Statique par frame, aucune animation propre : reduced-motion respectee par
      construction.
- [~] 2.4 AUCUNE string UI nouvelle (pas de bouton, pas de legende) : rien a traduire —
      couvert par l'absence.

**Gate 2** : gates web PASSES (purge `.tmp`, typecheck exit 0, lint 0 erreur, vitest
424/424 fichiers — 3813 tests), zero hex. RESTE LE GATE VISUEL UTILISATEUR : film temoin
`000d5950` (artefact re-cuit, 25 tractions jouables — slots 516/530/561... des 15 vies).

## Regles dures

- Aucune position devinee ; si le port ne valide pas, rien ne se trace.
- Largeurs de dequantification : celles de la CARTE (`MapQuantEntry`), jamais le defaut.
- Un seul decodage filmdec par process ; AUCUNE base DuckDB en ecriture.
- **JAMAIS de pause d'attente passive** : toute attente = commande bloquante avec timeout.
- Zero fix hors perimetre ; decouvertes au registre.

## Statuts et cloture

`[x]` / `[~]` / `[!]` avec justification ; aucune case vide. Entree `thought_log.md`,
registre, commits sur `feat/v75` (pas de push).

## Decouvertes (notees, non traitees — regle 7 du contrat)

- Drapeaux i59 != 000 (8/210) et valeur interne 4 : grammaire non etablie, desync propre
  comptee — AU REGISTRE avec condition de reprise.
- Semantique des deux vecteurs dir24+mag12 du corps lourd (et du R(24)+R(9) de queue) :
  non identifiee, non necessaire a l'ancre — AU REGISTRE (piste : DAT_143cd839c, ou
  correlation direction cubemap / deplacement du porteur).
- La queue R(3) d'i59 n'etait JAMAIS lue offline (paramByComponent sans cle i59) —
  CORRIGE dans ce lot (transverse, bloquait la preuve de marche).
- Flake vitest hors perimetre (`PalmaresRelationsPage`, 1 echec isole, 2 passes vertes) —
  AU REGISTRE.

## Journal

- 2026-08-16 : phase 0 executee et commitee (`020b95eab`). Le decompile s'est revele
  faux sur l'enrobage (tag interne, deux ac4) : grammaire LUE dans le flux (dump +
  consensus par classe de longueur, 2 cartes aux largeurs d'axe differentes — c'est le
  differentiel +10 bits Cliffhanger->Bazaar qui a identifie la position a largeurs de
  carte). Preuve de marche renforcee : ecart au record suivant = 0 partout.
- 2026-08-16 : phases 1 et 2 executees. Schema 8, golden v5, artefact temoin re-cuit,
  ligne au canvas (`--foreground`, 1,25 px). Gates Go et web passes. Reste le gate
  VISUEL utilisateur sur `000d5950`.
