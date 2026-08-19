# Lot C-ter — volet 3 : LA JAUGE DE CAPTURE EN DIRECT (schema 17)

> Perimetre : CT.3.1, CT.3.2, CT.3.3 du plan `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`.
> Branche `wt/jauge-live`, base `638c4d044`. Gates : `LOTCTER_volet3_gates.log`.
> Temoins : `lotCter/{cuisson_cli,temoin,echelle}_<short8>.log`.
> Acquis repris sans les refaire : `LOTCBIS_PHASE2B.md` (grammaire `ti=13`, appariement,
> intervalles de propriete), volet 1 du meme lot (ce que le tag 3 vaut en KOTH).

## 0. Le resultat, en une phrase

**L'arc de la zone se remplit maintenant a l'image, et la donnee qui le remplit est mesuree** :
sur les deux Bastion du corpus, **36/36 et 34/34 = 100,0 %** des bascules de proprietaire sont
precedees d'un pas MONTANT de la jauge dans [-5 s ; +2 s] (seuil ecrit avant la mesure : 90 % ;
temoin decale de +20 s : 38,9 % et 29,4 %, pour un niveau du hasard de 38,6 % et 40,0 %), la
serie pese **1,562 %** et **1,748 %** de l'artefact — et **le KOTH n'en publie AUCUNE**, parce
que le tag 3 n'y est pas une jauge de capture.

## 1. Ce qui est ecrit — fichiers et lignes

| fichier | L | ce qu'il porte |
|---|---|---|
| `internal/analysis/replay/zone_states_gauge.go` (NEUF) | 163 | LA REGLE de la serie : rien hors rampe, allegee (0,02 ou 1 s), meme echelle, retour a zero de fin de rampe, comptage |
| `internal/analysis/replay/zone_states.go` | 347 | `gaugeProgressOf` : l'echelle du JEU (zero quantifie 8 388 607, unite 83 886 quanta), et pourquoi ce n'est plus l'excursion du match |
| `internal/analysis/replay/zone_states_owner.go` | 431 | la serie posee sur chaque zone de Bastion : TOUTES les rampes du slot, pas seulement celles qu'une capture rattache |
| `internal/analysis/replay/zone_states_hill.go` | 195 | le volet colline, qui ne publie AUCUNE serie et l'ecrit |
| `internal/analysis/replay/document_zones.go` | 262 | la forme (`GaugePoint`, `ZoneState.Gauge`, `ZonesCoverage.GaugePoints`) et la chronique du schema **17** |
| `apps/web/.../zoneStatesLayer.ts` | 306 | `zoneGaugeAt` (escalier pur) et `drawGaugeArc` (3 parametres) ; le sommet statique de v16 n'est plus dessine |
| `apps/web/.../useZoneStates.ts` | 90 | la tenue en frames (`ZONE_GAUGE_HOLD_MS` converti une fois par document) et l'encre du CAPTEUR |
| `contracttest/replay_contract_test.go` | +6 | la chronique « 36 -> 36 » : un sous-champ, pas un champ racine |

Tests : `zone_states_gauge_test.go` (294 L, 7 cas : les TROIS clauses d'allegement avec
contre-epreuve — variation, seconde, et le DERNIER point d'une rampe (revue, §10) —, T strictement
croissant, rien hors rampe, le retour a zero, **et le calque en KOTH qui ne publie rien**),
`zone_gauge_temoin_test.go` (320 L, l'instrument de mesure sous gardes `ZONE_ARTEFACT` /
`ZONE_FILM`), `zoneStatesLayer.test.ts` (275 L, 21 cas).

## 2. La forme publiee (extrait REEL, `7344d24f`, zone 1)

La jauge monte pendant la capture, puis TOMBE A ZERO exactement a la frame ou le proprietaire
bascule — c'est le film qui le dit, pas une convention de l'ecriture :

```json
{"t":1060,"v":0.805},{"t":1062,"v":0.833},{"t":1064,"v":0.862},{"t":1066,"v":0.891},
{"t":1068,"v":0.919},{"t":1070,"v":0.948},{"t":1072,"v":0.976},{"t":1073,"v":0.991},
{"t":1074,"v":0}
```

et l'intervalle qui s'ouvre a cette frame : `{"t0":1074,"t1":1536,"owner":0,
"progress":0.9714494,"active":false}`. Couverture :
`{"method":"captures+geometry","catalog":3,"paired":3,"spans":39,"hillPeriods":0,
"gaugePoints":1701}`.

## 3. LE RECADRAGE : en KOTH, aucune serie — et pourquoi

Le plan reservait la serie « en Bastion comme en KOTH ». **La mesure du volet 1 l'interdit** :
sur une colline, le tag 3 n'est pas la progression de garde mais un **compteur de transfert**
d'environ une seconde (9-10 pas fixes, quelle que soit la duree de la garde) ; la progression de
garde vit dans le canal par joueur (mode B, tag 7), hors de ce calque. Publier cette rampe comme
jauge aurait montre **un arc qui se remplit en une seconde a chaque prise — credible et faux**.

`buildHillStates` ne pose donc aucune serie (`attachHillGauges` et `mergeGaugePoints` supprimes),
et **un test le PROUVE** : `TestZoneStatesCollineNePublieAucuneJauge` echoue si une serie
reapparait ou si `gaugePoints` quitte zero. Sur l'artefact KOTH recuit, `gaugePoints` vaut **0**
et aucune des 6 collines publiees ne porte la cle `gauge`.

## 4. L'ECHELLE change avec le schema 17 — et c'est la mesure qui l'impose

La convention du schema 16 ramenait `progress` sur l'**excursion mesuree** de la zone sur le
match. La jauge en direct l'a mise en defaut (`echelle_7344d24f.log`) : deux des trois slots de
jauge de `7344d24f` portent **une** emission aberrante sous zero (4 057 240 et 8 198 709 quanta,
soit -51,6 et -2,3 unites), et ce plancher unique ecrasait toutes les captures reelles de la zone
dans **[0,981 ; 1]** et **[0,694 ; 1]** — un arc plein, ou aux deux tiers plein, DES LE DEPART
d'une capture. Le sommet par intervalle ne le montrait pas (il vaut ~1 dans les deux echelles).

`gaugeProgressOf` ramene desormais le quantum sur l'echelle du JEU : le deser declare
[-100, +100] sur 24 bits, la jauge y vit sur **[0, +1]** — la valeur au repos est exactement le
zero quantifie (8 388 607, la valeur la plus frequente des sept slots de jauge des trois temoins)
et une capture menee a terme culmine juste sous +1,0 (8 471 108 a 8 472 395 quanta pour 8 472 493
a l'unite). Consequence de contrat : **les `progress` d'un v16 et d'un v17 ne sont pas
comparables** — une raison de plus de re-cuire.

## 5. LE RETOUR A ZERO ferme chaque rampe (et l'escalier client TIENT entre deux points)

Mesure : sur les trois slots de jauge de `7344d24f`, **tous** les pas descendants (18, 18 et 16)
sont des retours au zero exact, et il n'y a **aucun** pas nul. Le canal est une marche
d'escalier : il monte tant qu'on capture, se **tait** tant que la capture est figee, et se remet
a zero d'une seule emission — capture aboutie (une frame apres le sommet) comme abandonnee.

Le producteur publie donc ce point (`appendGaugeReset`), et le client tient la derniere valeur
**jusqu'au point suivant** au lieu de l'effacer au bout d'une seconde. Sans cela, une capture
figee disparaissait de l'ecran : sur `7344d24f`, **15 silences de plus d'une seconde** surviennent
alors que l'arc est visible (v >= 0,05), le plus long durant **29,3 s a 0,921** (zone 1, frames
3 757 -> 4 050) — une zone contestee que le film montre et que l'ancienne extinction cachait.
Seul le DERNIER point d'une serie s'efface une seconde apres son instant : rien ne viendra plus
dire ce que la jauge devient.

## 6. Les chiffres des temoins (CT.3.3, relus SUR l'artefact)

Cuisson par `cmd/replay-build --facts`, un film par processus, `LEVELUP_REPO_ROOT` = worktree,
films lus dans le tronc principal, plafond 3 Go : **peak 194 / 159 / 140 Mo**, **239 / 235 / 224 s**.

| film | mode | zones | points de jauge (par zone) | octets des series | artefact v17 | v16 | ecart |
|---|---|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 3 | **1 701** (479 / 666 / 556) | 34 960 o = **1,562 %** | 2 237 955 o | 2 202 930 o | **+1,590 %** |
| `696a9d7c` | Strongholds | 3 | **1 794** (554 / 623 / 617) | 36 858 o = **1,748 %** | 2 108 358 o | 2 071 392 o | **+1,785 %** |
| `01e1f945` | KOTH | 6 | **0** | 24 o = 0,001 % | 1 816 981 o | 1 816 953 o | **+0,002 %** |

Clause de poids (<= +2 % par artefact) : **TENUE** sur les trois.

| film | bascules du proprietaire | precedees d'une MONTEE [-5 s ; +2 s] | temoin (+20 s) | niveau du hasard |
|---|---|---|---|---|
| `7344d24f` | 36 | **36/36 = 100,0 %** (seuil 90 %) | 14/36 = 38,9 % | 38,6 % |
| `696a9d7c` | 34 | **34/34 = 100,0 %** | 10/34 = 29,4 % | 40,0 % |
| `01e1f945` | 0 (mode a colline) | sans objet | — | — |

Le temoin decale et le niveau du hasard sont du meme ordre (38,9 vs 38,6 ; 29,4 vs 40,0) : la
clause ne doit rien a la densite des pas montants, elle mesure bien un ordre temporel.

## 7. Le contrat

- `zoneStates[].gauge` : `[{t, v}]`, T strictement croissant, `v` dans [0, 1] a trois decimales,
  ABSENT quand la zone n'a aucune rampe **et toujours absent sur une colline**.
- `coverage.zones.gaugePoints` : somme des points publies (0 en KOTH).
- `zoneStates[].spans[].progress` : conserve, meme cle, meme sens — **echelle changee** (§4).
- Schema **17**. `wantReplayDocumentFields` reste a 36 (sous-champ, pas un champ racine) ; la
  ligne « 36 -> 36 » de la chronique du contracttest le dit.
- Rendu : un artefact <= 16 n'a plus d'arc du tout (le sommet statique se lisait comme une
  jauge) ; la reprise du backfill se fait par `SchemaVersion`.

## 8. Statut des items

- [x] **CT.3.1** — la serie au document, allegee, `gaugePoints`, contrat, `generated.ts`, goldens,
  garde web. Deux nuances par rapport au libelle du plan, ecrites et testees : la serie n'existe
  QUE sur les modes a zones simultanees (§3), et son echelle est celle du jeu (§4).
- [x] **CT.3.2** — l'arc suit la serie en escalier, a l'encre du camp qui capture (neutre si le
  camp de reference est inconnu) ; plus d'arc sans `gauge` ; `drawGaugeArc` a 3 parametres ;
  `ReplayCanvas.tsx` = 812 L (cliquet tenu). Complete apres coup : l'escalier TIENT entre deux
  points (§5).
- [x] **CT.3.3** — les trois temoins recuits et relus sur l'artefact (§6).

## 9. Decouvertes (notees, NON traitees ici)

1. **`progress` en KOTH n'a plus de sens clair.** Le sommet publie par periode de colline est
   desormais le sommet du COMPTEUR DE TRANSFERT sur l'echelle du jeu — il vaut ~1 des qu'un
   transfert s'acheve. Ce n'est pas faux, c'est peu informatif. Le volet 1 possede ce fichier :
   la question lui revient.
2. **Deux slots de jauge sur cinq ne sont pas des jauges de zone** sur `7344d24f` (slot 1545 :
   des pas de 83 887 quanta exactement, soit une unite entiere toutes les 10 frames, jusqu'a 27
   unites — un compteur, pas une jauge ; slot 1547 : aucun retour a zero). Ils ne sont pas
   apparies et ne publient rien ; leur nature reste a nommer.
3. **Ecarts intra-rampe** : 64 et 57 silences de plus d'une seconde sur les deux Bastion (dont 15
   et 10 avec l'arc visible). Ils sont maintenant TENUS a l'ecran (§5) ; s'ils devaient etre
   distingues d'une capture reellement figee, il faudrait une emission de plus, que le film ne
   porte pas.

## 10. Revue adversariale ronde 1 (2026-08-19)

Verdict du contexte frais sur le volet : **0 P0, 0 P1** — les **25 conditions** du volet tiennent
sur pieces. Quatre constats de FORME, corriges ici. **Aucun changement de comportement** : le seul
fichier de production touche est de la documentation (godoc) ou de la suppression de code mort.

1. **La clause `!last` n'avait AUCUN test** (`zone_states_gauge.go:98-99`). Elle est pourtant
   vivante sur donnees reelles : sur `7344d24f`, le sommet **0,991** d'une capture n'est publie
   que par elle — le pas precedent vaut 0,976, soit 0,015 de variation (sous 0,02) UNE frame plus
   tard (sous la seconde) ; les deux clauses d'allegement l'ecarteraient, et l'arc s'arreterait a
   0,976. `zone_states_gauge_test.go` porte desormais
   `TestZoneGaugeDernierPointToujoursPublie`, qui rejoue exactement cette rampe (frames
   1060..1073) et verifie EN PLUS que le dernier pas reste sous LES DEUX seuils — sans cette
   garde, une retouche des valeurs rendrait le cas vacant sans que rien ne rougisse.
   **CONTROLE NEGATIF JOUE** : `last := i+1 >= len(ss) || ss[i+1].t > w.t1` remplace par
   `last := false` (la clause devient inerte) donne
   `--- FAIL: TestZoneGaugeDernierPointToujoursPublie` — « 7 points publies pour 8 emissions »,
   serie arretee a `{1072 0.976}` — et c'est le **SEUL rouge de tout le paquet**
   `internal/analysis/replay` (41 s de tests) : la clause n'avait effectivement aucun autre
   gardien. Mutation annulee (`git checkout --`) ; le code de production n'a pas bouge d'un octet.
2. **Godoc restes a l'echelle de v16** (`document_zones.go:123-124` et `:148`). `ZoneState.Gauge`
   annoncait « part de l'excursion mesuree sur ce match » et `GaugePoint.V` « sur l'echelle de la
   zone », alors que le code est passe a l'ECHELLE DU JEU (§4 : `gaugeProgressOf`, 0 = jauge au
   repos, 1 = pleine, ecretee). Les deux godoc disent maintenant ce que le code fait, et renvoient
   a `gaugeProgressOf`. `ZoneSpan.Progress`, verifie au passage, etait deja juste.
3. **`zoneStateAt` etait du code mort** (`zoneStatesLayer.ts:69-71`) : exporte, et documente
   « c'est elle que le rendu appelle a chaque image » — faux, `drawZoneStates` appelle le prive
   `spanStateAt`. Grep de `zoneStateAt` sur TOUT `apps/web` (`node_modules` compris) : sa
   definition, sa doc, et son fichier de tests — **aucun appelant de production**. Aucun reemploi
   prevu non plus : la seule mention au plan est CT.3.2, close, et c'est `zoneGaugeAt` qui l'a
   servie ; le rendu des objectifs vivants passe par `objectivesLayer.ts` (`PLAN_DRAPEAU_OBJET.md`
   §4). Regle « 0 code mort » appliquee : l'export et ses **4 cas dedies** sont SUPPRIMES,
   `spanStateAt` reste l'unique lecture d'etat du calque et herite de la doc — verite comprise sur
   qui l'appelle, et sur le fait qu'on l'exportera le jour ou quelqu'un la lira, plutot que d'en
   garder deux versions. Trois des quatre cas etaient deja couverts par les tests de
   `drawZoneStates` (bornes INCLUSES, « personne ne la tient » = mesure, rien hors intervalle) ;
   le quatrieme — la zone ACTIVE — ne l'etait pas, il est donc REPORTE AU RENDU (« la zone ACTIVE
   est en SURBRILLANCE : remplie sans proprietaire, lisere renforce »), la ou la surbrillance se
   voit vraiment. Bilan : 24 cas -> 21.
4. **Comptes documentes faux.** §1 de ce journal annoncait « 26 cas » pour
   `zoneStatesLayer.test.ts` (reel avant revue : **24**) et la puce CT.3.2 du plan « 22 » puis
   « 26 » ; `replayContract.test.ts:163` annoncait « 47 chemins » pour un tableau de **50**. Tout
   est recompte (`grep -c "  it("`, comptage du tableau) et corrige : **21 cas / 275 L**,
   **50 chemins**, et les longueurs de la table §1 remises a jour
   (`document_zones.go` 260 -> 262, `zoneStatesLayer.ts` 313 -> 306,
   `zone_states_gauge_test.go` 254 L / 6 cas -> 294 L / 7 cas).

Gates de la revue : section « revue ronde 1 » de `LOTCTER_volet3_gates.log`.
