# Critique de complétude, campagne de grammaire phase 1

Abréviations : RAPPORT = `RAPPORT_CAMPAGNE_GRAMMAIRE_PHASE1_2026-10-01.md`, PLAN = `PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`, MESURES = `campagne_grammaire_2026-10-01/MESURES_CIBLEES.md`, CARTE = `CARTE_FERMETURE_V2_2026-10-01.md`, ANALYSE = `ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md`. Les chemins sont sous `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire/.ai/`.

## A. Dépendance à J12, et à ce qui bloque J12 (c'est la question de l'utilisateur)

1. **« Aucun lot ne touche les fichiers de J12 » n'est pas prouvé, et des pièces le contredisent.**
   - Passages : PLAN l.266 (§6.0) et RAPPORT §0.7.
   - J12.1 convertit les tris (`sort.Slice*`) avec `go fix`. Or `object_deaths_march.go:78` et `movement_states.go:385` contiennent un `sort.SliceStable`, et les deux sont dans la liste des fichiers de L1.
   - J12.5 retouche les commentaires et les commandes `filmdec` (relevées par grep dans `frame_closure.go`, `type1_datums.go`, `dispatch_player.go`, `dispatch_biped.go`). Ce sont des fichiers de L1, L2 et L3.
   - Les deux branches régénèrent `grammar_rev.golden`. La phase 1 l'a déjà modifié (CARTE §9), ainsi que `frame_closure.golden`.
   - ANALYSE §6 dit l'inverse du plan : « J12.1, J12.3, J12.4 et J12.5 touchent les fichiers centraux (…`world.go`…) et les références d'équivalence ; commencer avant garantit des conflits sur chaque golden ».
   - La branche J12 n'est pas visible localement (aucune ref dans `git branch -a`). Le plan ne nomme ni sa tête ni son commit. L'affirmation n'a donc été confrontée à aucun diff.
2. **Pourquoi la preuve doit attendre l'après-J12 n'est écrit nulle part.**
   - Le journal de `PLAN_SUITE_AUDIT_DECODEUR` (2026-10-01) dit que la preuve G-equiv de J12 est déjà faite : « document publié IDENTIQUE à l'octet » sur 20 films. Seuls quatre digests d'étape bougent.
   - La dépendance tient donc aux conflits de fichiers, pas aux références. Le plan ne pèse pas l'option « prouver maintenant contre les références J11.2, puis reconfirmer après J12 ».
3. **La chaîne qui bloque J12 est absente.** La fusion de J12 attend celle de J11, qui attend la vague J11.4 (les recuissons en cours, des dizaines d'heures) et J11.5. Le plan ne donne ni échéance ni ordre. Il ne dit pas non plus que les gates de phase 2 (corpus, banc) se disputeront la machine (RAM, CPU) avec cette vague.
4. **Une montée de `grammar.Rev` en phase 2 périme le parc qu'on est en train de recuire.** ADR 0034 : une révision montée ouvre un backlog `redecoder` / killsource. D7 (PLAN l.211) pose la question « quand recuire », mais pas ce conflit avec la vague J11.4 en cours.
5. **J12.7 touche aussi les livrables de la phase 1.**
   - J12.7 pose le tag `research` sur 218 fichiers et ajoute le ratchet `research_tag_test.go`.
   - `frame_closure_detail.go` et `frame_closure_detail_records.go` sont neufs, sans tag, et n'ont aucun appelant de production (seul `cmd_fermeture` les appelle).
   - `type1_datums.go` (`LireBlocDeDatums`, appelé seulement par des tests et par `research`) est sans tag, alors que L1 veut le faire passer en production.
   - RAPPORT l.48 (« seul L0 n'a aucune dépendance ») ignore ce conflit.

## B. Décisions utilisateur cachées ou déjà prises

6. **D1 (PLAN l.439) repose une question déjà tranchée.** ANALYSE §7 : « Tranchées par l'utilisateur le 2026-10-01 … (2) Oui : l'étape 1 et les lots sans différence … démarrent après la fusion de J12 ».
7. **Le plan de phase 2 contredit l'ordre recommandé sans le dire.**
   - ANALYSE §5.1 et §6 recommandent « Marcheur avant la campagne : les correctifs de grammaire atterrissent une fois, dans un seul marcheur ». La décision a été prise « recommandations retenues ».
   - Le §6 du PLAN fait atterrir L1 à L7 dans les marcheurs actuels.
   - Aucune mention de la représentation intermédiaire, du marcheur unique ni de J11.4 dans le §6 (vérifié par grep).
   - Il faut une décision explicite de l'utilisateur.
8. **La décision ANALYSE §7 (6) « blocs véhicules dans la campagne » est restée ouverte.** Pourtant T6 et T7 ont été instruits, et L2 et L4 la supposent prise. PLAN §3 ne note que la décision de réouverture.

## C. Chiffres incohérents ou mal libellés

9. **−7 751 est une perte brute, pas une perte nette.**
   - RAPPORT l.109 (T1-3) : « perte nette sur HI_1_8_0 à HI_1_11_0 (−7 751 sur le corpus) ». Or −7 751 est la perte BRUTE du corpus ; le net corpus vaut +11 354 (`mc_variantes.tsv`).
   - MESURES §2 parle de perte nette sur 1_8, 1_10 et 1_11 seulement. Sur HI_1_9_0, les paquets sont nets positifs (+337/−264), et seuls les utiles y baissent.
   - PLAN l.320 reprend la même formule.
10. **L'oracle n'est pas une borne au sens écrit.**
    - Le PLAN (§6.0) définit « borne = gain maximal ». Or sur HI_1_12_0 la variante `tete-bloc` fait mieux que l'oracle : 33,1 % d'utiles fermés contre 31,0 % (MESURES §2).
    - Le dénominateur `utiles_lus` bouge d'une variante à l'autre : référence 5 961 028, oracle-NEW 5 917 297, `tete-bloc` 5 818 859.
    - Les pourcentages du déclencheur (RAPPORT §3) peuvent donc monter parce qu'on lit moins. Ce point n'est discuté nulle part.
11. **Les variantes annoncées ne sont pas toutes publiées.** MESURES §1 annonce cinq marches, dont `oracle-bloc+tete-bloc`. Ni MESURES ni RAPPORT ne publient ce dernier résultat (TSV : +20 809 / −7 535 ; HI_1_13_0 +15 595 / −3 055).
12. **La région (iii) est incohérente entre les documents.**
    - RAPPORT l.95 : « presque tout `ti=3 i0` », 511 eid, 13 978 paquets.
    - MESURES §T1-5 et §T7-5 : `ti=3 i0` = 1 134 eid et 27 763 paquets, soit plus que la région entière.
    - Les 13 978 paquets du corpus égalent ceux de HI_1_13_0 seul, ce qui contredit « les autres builds suivent la même hiérarchie ».
13. **La table des régions du RAPPORT omet la ligne « (iii') ouverte »** (89 eid, 13 témoins, 2 381 paquets ; MESURES §T1-5). Sans elle, 1 371 + 1 910 + 1 289 + 511 = 5 081 et non 5 170 liaisons. Les témoins ne somment pas non plus à 239.
14. **Libellés approximatifs (sans gravité).**
    - « Moteur `ti=2` (4 331) » (RAPPORT §0.5) inclut 123 paquets `ti=0`.
    - « 12 112 sur un film HI_1_12_0 » désigne `i35` seul ; `ti=43` vaut 12 127 sur HI_1_12_0.
    - Le contexte de la tâche cite 92,5 % ; la CARTE et le RAPPORT donnent 92,7 % (table corrigée). C'est expliqué dans CARTE §2, mais pas dans le RAPPORT.

## D. Affirmations surqualifiées ou sans pièce

15. **Les vérifications adverses n'ont laissé aucune pièce.**
    - RAPPORT l.9 : « chacune relue par deux vérificateurs adverses ».
    - Aucun fichier de vérification n'existe. Les notes T1 à T8 ne mentionnent aucun vérificateur. Le `thought_log` n'a d'entrées que pour les étapes 1, 4 et 5 (rien pour les étapes 2 et 3).
    - Les verdicts « contesté / confirmé » (T1-6, « correction 3 », « correction 4 ») s'appuient sur des sorties non conservées.
16. **« Le jeu n'écrit jamais un DELTA d'entité non déclarée » est classé « Établi » (RAPPORT l.171) alors que trois points restent ouverts.**
    - D-35, la file de records différés `vue+0x1b320` : « si elle est active en lecture de film, le jeu lui-même rejetterait ». C'est la question ouverte de T1 §8, absente du « Ouvert » du RAPPORT (l.181).
    - Les 4 598 paquets fermés après un rejet.
    - La région (iii'), contraire à l'ordre de l'écrivain.
17. **T1-3 « établi sur HI_1_13_0 » repose sur la tête seule.**
    - L'attendu de T1 (§7 M2) était « proche de 100 % sur HI_1_13_0 ».
    - Mesuré au rang 0 sur HI_1_13_0 : pool 4 à 44 %, pool 0 à 4/145, pool 2 à 12/41.
    - Le « 97,7 % » ne concerne que la tête `(gen+1)&3`, pas le slot. L7 et L1a s'appuient sur la prédiction du slot.
18. **« Masque vide au bloc suivant = entité née et morte » (RAPPORT §0.1, §4.2) est une interprétation, pas une mesure.** La CARTE §4 définit « naissance non lue » comme « allocation vivante ou déjà libérée ». Aucune mesure ne distingue « libérée » de « vivante sans composant ».
19. **D-7 n'est pas reporté dans le RAPPORT §4.** Sur les 266 naissances où le pont masque → archétype résout, le `R(6)` trouvé par M1 le contredit 144 fois (54 %). Or l'oracle-NEW lie avec ce `R(6)`. Ce point pèse sur la fiabilité de la borne de L1 au-delà de l'estimation de 4,7 % de liaisons fausses (tirée du seul taux du témoin).
20. **La limite de M1 n'apparaît pas dans le RAPPORT §0.2 (« on sait où sont les naissances ratées »).** MESURES §5 : « la région (i) n'est pas séparée de la vue A dans les paquets à événements ».
21. **Les paquets gagnés sous l'oracle n'ont pas été passés aux invariants de l'écrivain.** Ces invariants sont pourtant instrumentés (`campagne_invariants_research_test.go`). La part de fermetures factices dans les +29 769 paquets gagnés est inconnue.

## E. Mesures annoncées non faites, alors que la recherche suffisait

22. **Le dénominateur des entrées (item 1.3) reste « supposé » alors que T5 donne la règle.** RAPPORT l.145 : « supposé : au plus une par joueur actif ». T5 établit « kind 0 seulement, un par joueur, index croissants, ≤ 32 ». Un dénominateur indépendant était calculable. L'item 1.3 n'est pas tenu et n'a pas de statut.
23. **L1a (bande OU bloc, plus filtre d'invariants de masque) n'a pas été mesuré**, alors que les deux ingrédients existent dans la sonde. PLAN l.320 exige que « le filtre d'invariants doit annuler ces pertes avant tout gate ». C'était faisable en phase 1.
24. **Le double essai de la fourche `+0x74` a été déclaré impossible à tort.** MESURES §T5-3 le dit « non mesuré, exige un portage », mais L5 (PLAN l.395) prévoit justement un « double essai sous tag research ». C'était faisable en phase 1.
25. **Les A/B de position (T4-C1, C4 à C6, L6) ont été déclarés non mesurables par choix, pas par contrainte.** La sonde a déjà recopié la marche (`campagne_marche_research_test.go`) ; elle aurait pu recopier le lecteur de position. T4-C3 est déclaré « non concluant, à remesurer sous le contexte de production » et aucun item n'en est chargé.
26. **Le témoin utilisateur de T7 n'a pas été mesuré.** `81c02726` (G MONEY : NEW `ti=43` désynchronisé sur `i21`), proposé par T7 §6.3, est disponible dans `film_chunks/`. Il n'est ni dans le corpus de 20 films ni dans le gate de L2 (PLAN l.348 et suivantes : seul `bcb6d393` est prévu).

## F. Populations sans lot ni recherche (pistes non instruites)

27. Sont mesurées puis laissées sans lot, alors que plusieurs pèsent plus que L3, L4 et L5 :
    - « Image-clé incomplète » (D-5) : 87 eid, 13 644 paquets.
    - « Réalloué sous une autre génération » : 12 906 paquets.
    - « Aucune allocation » : 6 451 paquets (dont D-6, `0797ce72` : 1 809).
    - « Slot au-delà du plafond, cadrage faux » : 6 987 paquets (lien possible avec D-24, `IDLowBits` figé à 13, non instruit).
    - Eid introuvables : 2 836 eid, 15 192 paquets.
    - Eid trouvés à plus de 3 paquets : 6 724.
28. **Aucun lot ne couvre la région (iii).** Ce sont des NEW `ti=3 i0` désynchronisés, dont la grammaire n'est pas portée. Pourtant l'oracle-NEW inclut leurs liaisons dans la borne de L1, alors que L1a, L1b et L1c ne les traitent pas.
29. **La borne de L1 n'est pas ventilée par sous-lot.** Les +27 611 paquets mélangent (i), (ii), (iii'), (iii) et « ouverte ». L1b dépend du succès de R-L1 (c), et (iii') de R-L1 (a). Le gain propre à L1a seul est inconnu. Le tableau §6.1 présente pourtant +27 611 comme le gain de L1.
30. **L3 n'est pas instruit pour les vieux builds.** Le bassin `i15` y rend des masques « tout à un » ; la cause n'est pas élucidée et aucune recherche préalable n'est prévue. Le gate « aucune baisse sur aucun build » risque de bloquer.

## G. Lots de phase 2 : gates incomplets

31. **L1 change la marche de killsource, mais le gate ne vérifie pas killsource.**
    - L1 modifie `marchLocateStrict` (`object_deaths_march.go`), qui alimente `ScanMarchFacts` et les morts publiées.
    - Le fichier dit lui-même : « UNE SECONDE COPIE DE CETTE MARCHE EXISTE … `film/facts/killsource/walk.go` ». Cette copie est absente de la liste des fichiers de L1.
    - Le §6.0 n'exige pas l'équivalence ou le delta killsource, alors que le gate de J12 l'exige, ni le backfill killsource qui suivrait en prod.
    - D-17 (`marchViews = 8` contre 3 vues prouvées) n'est pas relié.
32. **L1 n'a pas de gate de performance.** Il fait lire en production le bloc de type 1, soit 343 019 octets par chunk (en-tête de `type1_datums.go`), plus un suivi d'allocateur. Rien n'est prévu côté performance ou mémoire de cuisson.
33. **Le gate se juge par build, plus faiblement qu'avant.** Le §6.0, point 3, demande « aucune baisse … sur AUCUN build ». J11 jugeait par film (« carte ≥ référence sur les 20 films »). Une baisse sur un film peut être compensée à l'intérieur de son build.
34. **Le garde-fou de la carte ne couvre pas les paquets à événements.**
    - La CARTE (le gate de chaque lot) passe par `FrameClosureDetaillee`, qui recopie le pilotage de `decodeFrameParRangs` (`frame_closure_detail.go:186-259`).
    - L1b modifie ce pilotage (lecture de la vue A). Le garde-fou `TestFrameClosureDetailleeRendLaCarteDeFrameClosure` ne tourne que sur deux mini-bobines (`ks_000d5950`, `ks_e5adf7b2`), sans preuve qu'elles contiennent des paquets à événements.
    - La recopie supplémentaire de la sonde (`campagne_marche`) n'est pas mentionnée.
35. **Les prérequis de L4 n'ont ni porteur ni méthode** : identifier `77ef810a`, `4118381d` et `d0b40d0a` (PLAN l.387), et décider D5.
36. **Les gains de L5 et L6 ne sont pas mesurés, mais le tableau §6.1 les range sous « Gain mesuré (borne) ».** L5 : MESURES §T5-3 dit « borne estimée ». L6 : « non mesuré ».

## H. Contrat d'exécution (skill `plan-execution`)

37. **Aucun item du PLAN §2 n'a de statut.** Tous sont `[ ]` (1.1 à 1.7, T1 à T8, 3.1, 4.1, 5.1, 5.2). L'en-tête du plan dit encore « EN COURS ». Le journal §4 s'arrête au lancement.
38. **L'item 1.3 n'est pas atteint** (estimateur tautologique, CARTE §5) et n'est pas marqué `[!]`.
39. **Le Gate 1 n'a aucune trace de résultat.** Aucun document ne consigne `go vet -tags=research`, les tests `-tags=research` des paquets touchés ni `-run 'Closure|Fermeture'`. Seule la mutation du garde-fou est citée (CARTE §9).