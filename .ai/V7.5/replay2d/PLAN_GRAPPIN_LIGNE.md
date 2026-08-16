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

### Phase 1 — PUBLIER

- [ ] 1.1 Evenements de grappin PAR VIE dans le document : `{tMs, ax, ay}` (l'altitude ne
      sert pas au rendu 2D ; la publier si elle est gratuite, sinon non). Fenetre de rendu
      derivee de 0.3(c) — MESUREE, pas choisie.
- [ ] 1.2 Convention `SchemaVersion` respectee (chronique dans `document.go` — bump 7 -> 8) ;
      contrat (`contracttest`, `replayContract.test.ts`), OpenAPI + `generated.ts`,
      normalisation web, golden rejoue et explique. Couverture publiee (vies avec
      evenements, par film).
- [ ] 1.3 Entree de DONNEES dans `replay.Options`, absence NON FATALE et loggee. Les
      artefacts TEMOINS locaux sont re-cuits (`000d5950` au moins — 3 porteurs de grappin
      en verite terrain).

### Phase 2 — TRACER

- [ ] 2.1 Ligne joueur -> ancre sur le canvas pendant la fenetre mesuree, meme chaine de
      projection que les tracks (`MondeVersPixel` / calage du fond). La position du joueur
      est CELLE DE LA TRACK a l'instant courant (elle bouge, la ligne suit).
- [ ] 2.2 « Blanche » = TOKEN, jamais un hex : prendre le token neutre le plus clair du
      systeme (voir `canvasInk.ts` et le garde-rail `fxInk.guard.test.ts` — les couleurs du
      canvas passent par les tokens resolus). Epaisseur discrete, sous la densite des
      effets de tir.
- [ ] 2.3 `prefers-reduced-motion` : une ligne statique par frame ne l'enfreint pas par
      construction ; pas d'animation propre.
- [ ] 2.4 Toute string UI eventuelle en FR ET EN (`i18n.ts`).

**Gate 2** : gates web complets (purge `.tmp`, typecheck, lint, vitest — exit 0), zero hex,
gate VISUEL utilisateur (film temoin : `000d5950`).

## Regles dures

- Aucune position devinee ; si le port ne valide pas, rien ne se trace.
- Largeurs de dequantification : celles de la CARTE (`MapQuantEntry`), jamais le defaut.
- Un seul decodage filmdec par process ; AUCUNE base DuckDB en ecriture.
- **JAMAIS de pause d'attente passive** : toute attente = commande bloquante avec timeout.
- Zero fix hors perimetre ; decouvertes au registre.

## Statuts et cloture

`[x]` / `[~]` / `[!]` avec justification ; aucune case vide. Entree `thought_log.md`,
registre, commits sur `feat/v75` (pas de push).
