# Critique de complétude des recherches préalables R-* (2026-10-02)

Je n'ai rien modifié et je n'ai lancé aucune commande `go`. Les recomptages ont été faits par awk sur les TSV.

Le worktree n'est pas dans l'état décrit par la mission. Sa HEAD n'est pas détachée sur `fe18bf67c` : il est sur la branche `feat/campagne-grammaire` @ `da7c2c764` (post-J12). Il contient 180 fichiers non suivis (176 intégrés et 4 `overlay_campagne.json`), et le plan, le rapport et `thought_log.md` sont modifiés sans être commités.

Chemins abrégés :
- PLAN = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire/.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`
- RAPPORT = `.../.ai/V7.5/film_re/RAPPORT_CAMPAGNE_GRAMMAIRE_PHASE1_2026-10-01.md`
- `CG/` = `.../.ai/V7.5/film_re/campagne_grammaire_2026-10-01/`

## A. Violations ou contournements des décisions FERMES (§3)

1. **L0 est absent de l'ordre proposé, alors que D2 en dépend.**
   - D2 fixe « fermé = reste nul ET aucun invariant violé (lot L0, qui touche l'outil et un fichier lu par la publication) ».
   - PLAN §6.5.4 (l. 1694-1707) ordonne : avant la vague 1, vague 1, vague 2. L0 n'y figure nulle part, ni L0.6 (invariant « sortie de vue B par rejet », 1 008 faux sains mesurés). Les gates de la vague 1 jugeraient donc avec le juge à trois invariants.
   - Contradiction restée sans arbitrage : la table §6.1 (l. 993) et le titre §6.2 (l. 1375) disent encore « carte seulement », et la l. 1390 dit « aucun fichier lu par une cuisson, sauf décision D2 ». D2 est tranchée, mais ni la sortie ni la montée de `grammar.Rev` qu'elle implique ne sont réconciliées.

2. **Le titre de §6.5.4 dit « sans changer les décisions FERMES », mais la composition de la vague 2 change.**
   - La décision FERME D-RI fixe « vague 2 = LU puis L1 et L7 ».
   - §6.5.4 donne LU → LS → L1a → LP → L7 :
     - L1b est retiré (D15) ;
     - LP est ajouté (D14) ;
     - LS n'apparaît pas au §3 (D-67 dit « confié à la campagne », sans trace au §3).
   - Il faut présenter ces points comme un amendement de D-RI à soumettre à l'utilisateur, pas comme une simple révision de recommandation.

3. **§6.3 est périmé face au §3 (décisions cachées ou doublées).**
   - « D-VEH (reste à prendre) » (l. 1494) et le prérequis 4 de L4 « D-VEH : les blocs véhicules entrent-ils dans la campagne ? » (l. 1343) contredisent D-VEH FERME.
   - « D2 : la mesure de fermeture doit-elle exclure… » (l. 1497) est encore posée comme une question.
   - D-RI est encore marquée « Recommandation (soumise à l'utilisateur) » (l. 1454), sans mention « tranchée ».

4. **Brut et sains sont mélangés, contrairement à D1/D2 (indicateur « sans factices »).**
   - §6.5.3 (l. 1691) cite « formats 24-25 : 72 à 84 % par film sous LM » comme indicateur. Or `R_VEH.md` §3.3 (l. 299-301) précise « brut, paquets contredits compris » : `r_veh_delta_synthese.tsv`, colonnes 8, 9, 17 et 18.
   - §6.5.4 (l. 1699-1701) ordonne la vague 1 « dans l'ordre du gain mesuré » avec des chiffres BRUTS (L8 +30 618, L2 +18 105, L9 +1 784, L4a +1 436). En sains, L2 vaut +17 598 seul et +14 876 en marginal sur le corpus. L'ordre n'est d'ailleurs même pas respecté : LM (+80 979) est placé 3e.
   - La ligne L4a de §6.1 (l. 987) donne « +26 843 utiles », qui sont des utiles gagnés BRUTS (`R_VEH.md` §3.1).

## B. Verdicts « prêt » contredits par des mesures existantes

5. **L6b est déclaré prêt, mais la mesure contredit le gate par film.**
   - §6.5.2 (l. 1640-1642) range L6b parmi les lots « Prêts pour la vague 1 … gate par film tenu dans la mesure ».
   - Or `CG/MESURES_BIS_4.md` l. 266 et 289 montrent `flock-position` à −6 records utiles sains sur `60ae07c4` (mesuré), ce qui échoue au gate 2 (« records utiles sains, aucun film »).
   - §6.2 L6b (l. 1270) dit encore « le juge des invariants n'a pas été joué sur ces A/B ». C'est périmé : BIS_4 point 25 l'a joué, et le journal §4 le rapporte.

6. **L6a « étendu au site `ti=41 i0` » est déclaré prêt, mais BIS_4 le contredit.**
   - Le site est `consumeObjectPositionMonde`, c'est-à-dire `world-object-i0` (vérifié dans `lecteur_position_exceptions.go:25-35`).
   - `MESURES_BIS_4.md` l. 244 et 272 : `world-object-i0` a trois films en baisse saine (HI_1_8_0 −30, HI_1_11_0 −6, `1c4c63c2` −4), et 490 de ses 654 gains sont contredits.
   - R-P3 (`R_COMP.md` §4.4, l. 306-317) n'a mesuré ce site que sur les deux films Live Fire, alors que la lecture « au jeu » est globale et ne dépend pas de la carte.
   - §6.5.2 et §6.2 L6a (« à juger sous D4 ») omettent ces trois baisses.

7. **Le gate 3 (killsource) n'est mesuré pour aucun lot de composant déclaré « prêt ».**
   - Lots concernés : L8, L2, L3a, L4a, L6a, LM.
   - Cas critique : L8 change le routage de `high-frequency` (`ti=3 i1`), alors que le localisateur de killsource et LS reposent sur la signature `high-frequency` de `ti=4`.
   - Conflit non relevé : la sonde R-LS identifie l'archétype « par NOM : seul composant `high-frequency` » (`R_LOC.md` l. 65). Cela contredit la règle issue de R-HOM, « routage par archétype ou par table, jamais par nom » (PLAN L8, `R_COMP.md` §5).

8. **La mesure killsource de LS exclut un film du corpus.**
   - PLAN D-67 (l. 507) et §6.2 LS annoncent « 1 080 morts » sur 28 films.
   - `R_LOC.md` l. 30 et 228 : `1c4c63c2` (corpus) et `81c02726` sont exclus, faute de carte connue.
   - Le gate 3 exige les 20 films. `1c4c63c2` est justement le film qui porte 359 signatures sur des slots non liés en début de chunk (D-75, justesse non instruite). Ce trou n'est signalé nulle part dans le plan.

9. **L3a : « 0 sain perdu » masque des requalifications.**
   - PLAN l. 983 et §6.5.1.
   - `CG/r_comp_tsv/r_comp_l3_resume.tsv` sur `1c4c63c2` : 424 gagnés, 237 contredits, Δ sains +147. Donc 40 paquets fermés sains en référence deviennent factices.
   - Sur le corpus : 3 924 − 316 = 3 608 gains sains, contre Δ +3 575, soit 33 paquets requalifiés (calcul awk).
   - Le critère net tient, mais la formule « 0 sain perdu » est inexacte et n'est pas expliquée.

## C. Chiffres incohérents entre note, TSV et plan

10. **Le dénominateur fixe consolidé (§6.5.3, D-68, D12) ne suit pas sa propre règle.** Recalcul awk sur les quatre TSV cités : HI_1_13_0 2 959 104 et corpus 7 176 150 sont exacts, mais leur origine est mal décrite.
    - Pour `d9781168` (316 761), `c75f33b8` (177 211) et `bf15f7ab` (230 562), le maximum vient de `ls+vueA`, l'oracle de la vue A dont 45 % des gains sont factices, et non de LS (`rloc_variantes.tsv`). La phrase « monte surtout par … LS » (l. 1680) est donc fausse.
    - Pour `1c4c63c2` (981 811), le maximum vient de `ls+libre`. Pour `50247b26` (330 775), de `libre` / `ls+libre`. Ce sont des variantes de contrôle ou rejetées (D-77). `R_COMB.md` §1.4 excluait pourtant les « témoins négatifs ».
    - Sans `libre` / `ls+libre`, le corpus vaudrait environ 7 166 438. L'écart est faible, mais la règle n'est pas appliquée uniformément.
    - R-COMP (marches L3 et P3) est omis du consolidé, alors que §6.5 dit « tous les chantiers qui ont lu les films ».

11. **D-71 / RC-4 n'est pas recalculé sur le fixe consolidé que D12 recommande.**
    - D-71 (l. 531) affirme que trois films dépassent 95 %, mais sur le fixe de R-COMB.
    - Sur le consolidé, `c75f33b8` vaut 157 161 / 177 211 = 88,7 % (`r_comb_par_film.tsv`).
    - Il ne reste donc que deux films au-dessus de 95 % : `f75e7053` et `81c02726`.

12. **Le nombre de surcouches rouges sous `campagne_overlay` varie d'un document à l'autre.**

    | Document | Surcouches rouges | Remarque |
    |---|---|---|
    | D-100 (l. 628) et §6.0 l. 883 | 3 : fusion, R-COMB, R-COMP | R-LS s'emploie avec `-tags=research` seul |
    | §6.5.5 (l. 1720) | 4 : + R-LS | |
    | RAPPORT §8 | « quatre sur cinq » | |
    | thought_log | « rouge avec quatre » | |

13. **L6a hors Live Fire est compté de trois façons.**
    - PLAN (l. 981, §6.5.1) : « 9 films sur 13 en baisse ».
    - En net de paquets sains (le critère du gate 2), seuls 7 films baissent (tableau `R_VEH.md` §5.4).
    - `R_VEH.md` §0 point 11 et §5.4 dit encore « 12 films ».

14. **Les notes n'intègrent pas les verdicts adverses que le plan leur attribue.**
    - `R_COMP.md` §0 et §2.2 : « établi … un bit de trop dans `i4` ». Le plan dit NON CONFIRMÉ, et la note se contredit elle-même : son §2.2 indique qu'un bit retiré à l'entrée de `i5` à `i10` ferme 49/49.
    - `R_VEH.md` :
      - §0 point 6 : « lectures fausses … mesuré pour les trois » ;
      - R-VEH-1 : « critère de bascule rempli » ;
      - l. 172 et 194 : `0x8a0` au lieu de `0x890` (`r_veh_ti40_cherche_n2.tsv` donne bien `0x890`).
    - `R_LOC.md` : « 99,5 % » ; `R_NAIS.md` : « 2,3 % » et « un paquet sur seize ».
    - `R_FUSION.md` §3 : « violent un ratchet ».

## D. Affirmations sans preuve conservée, et gates

15. **Les verdicts adverses des R-* ne sont archivés nulle part.**
    - Pour la phase 1, ils sont dans `VERIFICATIONS_ADVERSES.md`, daté du 01/10.
    - Pour les R-*, ils n'existent que dans la colonne « Verdict adverse » de §6.5.1, écrite par l'agent d'intégration. Aucun fichier sous `CG/` ne les porte.
    - Exemple non vérifiable : D-93 « Sans 8/3, `ti=35` baisse sur `111fa685` (15 → 10) ».

16. **Le gate de l'intégration n'a pas été rejoué, et un gate de surcouche est rouge sur l'arbre courant.**
    - §6.5.5 précise « rapportée par l'agent d'intégration, non rejouée ici ».
    - Fait rapporté par D-100 / §6.5.5 et cohérent avec la lecture du code, mais pas rejoué : `r_veh_delta_research_test.go`, tagué `campagne_overlay`, exige des symboles qui n'existent que dans `r_veh_overlay/`.
    - Conséquence : la surcouche canonique de la phase 2 (`r_fusion_overlay_postj12/`) et `mesures_bis2_overlay/` ne compilent plus.
    - L'item de la l. 873, « `mesures_bis2_overlay/` reste pour rejouer à l'identique les chiffres de la phase 1 », est donc faux en l'état.
    - Le journal §4 n'a aucune entrée pour l'intégration des R-* ni pour ce rouge.

17. **D-F3 est déclaré « SOLDÉE » (l. 863) sur une preuve partielle.**
    - La preuve est la carte v2 (marche de référence) rejouée à l'identique.
    - Rien n'a été rejoué après J12 : ni le juge (les sains), ni les oracles, ni les sondes R-* sans surcouche.
    - « Les chiffres valent pour la tête » (§6.5) reste donc supposé au-delà de la référence.

18. **Plusieurs passages sont périmés.**
    - §6.4 (l. 1595-1597) : « Il manque `go test ./internal/archlint/` … à jouer », alors que le journal dit qu'il a été joué. §6.4 propose aussi `[~]` + `[!]` pour 1.3, alors que le §2 porte `[!]`.
    - D-66 (l. 477) : « archlint n'est pas rejoué : item ouvert ».

19. **Deux items du §6.0 n'ont pas de statut.** « R-COMB-2 » (l. 917) et « Part de D-44 » (l. 921) restent en `[ ]`, sans `[!]` ni justification de report (règle 3 de `plan-execution`).

20. **Gate 5 : une copie de la marche n'est pas listée.**
    - `r_nais_marche_research_test.go` (`rnMarcher`) recopie `cmMarcher` avec un crochet.
    - Le gate 5 et les fichiers de L1 ne listent que `campagne_marche_research_test.go` comme recopie à suivre.

## E. Points non instruits qui pèsent sur les lots

21. **D-79 contredit la notion de « borne », qui reste pourtant utilisée.** LS (+18 119) dépasse l'oracle (ii) de BIS_1 (+11 540) sur la même région. Pourtant §6.1 L1 (« borne oracle +28 168 ») et l'argumentaire de D-RI (l. 1486) emploient encore ce chiffre comme une borne. Aucune requalification n'est faite.

22. **D14 recommande LM en vague 1 sans le lier à D6, toujours ouverte.**
    - La largeur MPP de 8 bits est seulement mesurée (aucun exécutable de ces builds).
    - LM et L4a sont mesurés sous le contexte des instruments (`R_VEH.md` §8), alors que `60ae07c4` (Live Fire, formats 24-25) exige le contexte de production pour L6a.
    - La combinaison LM × L6a n'est pas mesurée.

23. **R-L1 (b) généralise au-delà de ce qu'il mesure.** Le verdict « toutes factices » s'appuie sur T3-C1. Le test du début antérieur laisse 196 paquets sur 4 598 inexpliqués (dont 32 « sains »), et le sous-groupe dont le reste fait au plus 8 bits (648 paquets) n'est pas analysé à part (`R_NAIS.md` §2.1-2.2).

24. **Python appelé malgré l'interdit (D-102), sans décision sur la suite.** `python3` a été appelé deux fois (`R_COMP.md` DC-9 ; `R_VEH.md` §8, édition d'une sonde). L'écart est consigné, mais aucune décision ni vérification de la sonde modifiée n'est notée.

25. **Mise en forme du journal.** La nouvelle entrée de `.ai/thought_log.md` est collée au paragraphe précédent : il manque une ligne vide avant `## [2026-10-02]`.