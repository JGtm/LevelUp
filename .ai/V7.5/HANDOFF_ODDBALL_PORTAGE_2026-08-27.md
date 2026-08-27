# HANDOFF — Portage du crâne d'Oddball (reprise post-v7.5)

> Écrit le 2026-08-27 à la clôture du chantier « encadré Notion REPLAY 2D ». Destiné à
> l'agent qui reprendra LE PORTAGE (qui tient le crâne, quand). Cinq campagnes de mesure
> ont été menées, toutes à protocole commité AVANT mesure — ce document est leur somme :
> ce qui est ACQUIS (ne pas re-prouver), ce qui a été RÉFUTÉ (ne pas rejouer), et les
> pistes NON MESURÉES (le vrai travail). Tout est sur `feat/v75`.

## 0. État publié (ne pas casser)

- Le calque du crâne LIBRE est en production : clé de document `objectiveObjects`
  (schéma 21, contrat 39), une entrée par vie libre (`t0`, `t1`, `pts` — positions image
  par image), rendu = glyphe boule, encre neutre. Producteur :
  `internal/analysis/replay/build_objectives_live.go` ; rendu web :
  `useReplayObjectiveObjects.ts` + son calque.
- Deux REFUS y sont gardés par tests : rien n'est dessiné pendant les portages, et pas
  de prolongation au-delà de `t1`. Toute publication du PORTEUR devra les remplacer
  proprement, pas les contourner.
- Identité du crâne : mot MPP `0x0017592C`, entrée `[[objective_objects]]` famille
  `ball` du manifeste (`config/titles/halo_infinite/mappings/replay_labels.toml`),
  exclusion des socles d'armes VOULUE et gardée
  (`ground_weapon_flag_exclusion_test.go` étendu).

## 1. Les ACQUIS — mesurés, à réutiliser tels quels

1. **L'oracle est officiel et indépendant du film** : `match_objective_stats_latest`
   porte PAR JOUEUR `time_as_skull_carrier_seconds`, `skull_grabs`,
   `skull_scoring_ticks`. **Un tic de score de crâne = une seconde de portage**
   (écart 3-4 % mesuré). C'est LE juge de paix ; le seuil du gate historique :
   recouvrement ≥ 80 % par joueur ET porteur principal identifié sur ≥ 3/4 films.
2. **La primitive de proximité discrimine** : distribution bimodale q25 0,20-0,43 m
   (quelqu'un est là) contre q75 5,5-7,9 m (personne), médiane 0,77 m. Le seuil ~1,5 m
   se CONSTATE, il ne se règle pas.
3. **« Mourir, c'est lâcher » : 91,7 %** (22/24 : mort du porteur → naissance de vie
   libre < 3 m dans les 3 s).
4. **Le sommeil est RÉFUTÉ** : l'objet BOUGE entre son silence et sa renaissance
   (distances médianes 3,95-9,35 m ; un objet endormi renaîtrait à zéro). Pendant un
   « trou », le crâne est PORTÉ.
5. **Le signal est SPATIAL** — le contrôle le plus propre du chantier : décaler le lieu
   de repos de 12 m et rejouer toute la chaîne effondre le signal à 0,0-3,3 % (contre
   64,9-79,8 % au vrai lieu). La voie n'est pas fausse ; elle est incomplète.
6. **Précondition de pont OBLIGATOIRE** : ≥ 50 % de slots de bipède nommés par le pont
   d'identité, sinon le film sort du corpus (leçon `51ebbc0f` : 9/84 slots → mesures
   sans valeur). L'écrire dans le protocole AVANT, pas après.
7. Corpus mesurable : `24dbb67d` (Recharge), `43716616` (Smallhalla), `d9781168`
   (Dredge) — pont sain. `51ebbc0f` : pont cassé. Films-bombes connus (exclus d'office,
   l'exécuteur borné les tue proprement) : `51101d1d`, `a349fea8`.

## 2. Les RÉFUTATIONS — ne pas rejouer

| voie | verdict | pourquoi |
|---|---|---|
| Oracle du score PERSONNEL (D4) | réfuté | dominé par les frags (Δ médian 150 pts vs 0-25) ; le « ~1 pt/s » du plan était faux sur ce canal (9-10,6), juste sur les tics de crâne |
| Ramassage à l'INSTANT du silence (D6) | insuffisant | ne couvre que 45-48 % des ramassages (asymétrie de 26 pts avec les lâchers, captés à 62-74 %) |
| Position périmée / fenêtre élargie (D7-S1) | réfutée AVEC cause | +2 à +4,5 pts à 10 images ; l'objet est À L'ARRÊT quand il se tait (0,00-0,28 m/s) |
| Portages fusionnés comme cause principale (D7-S2) | réfutée | aucun portage reconstruit ne dépasse le plus long portage API |
| Sommeil de réplication (D8) | réfuté | cf. acquis n° 4 |
| PREMIÈRE TRAVERSÉE du lieu de repos (D9, la meilleure) | gate raté | 79,8 / 73,8 / 64,9 % pour 80 exigé ; porteur principal 0/3 |
| Fenêtre « queue » à 22,6 s | PIÈGE documenté | ferait franchir 80,7 % au seul premier film — bruit démontré (n'aide pas les deux autres), choisi après coup. NE PAS recalculer |

## 3. LE PROBLÈME RESTANT, nommé par les chiffres de D9

**Les LONGS portages se fragmentent.** Le plus gros porteur API de chaque film ne reçoit
qu'environ la moitié de son temps (80→53 s, 94→41 s, 116→66 s), pendant que les porteurs
moyens et petits sortent justes ou SUR-attribués (29→45, 10→28, 37→65). Ce biais orienté
est ce qui coule le critère « porteur principal ». Le recouvrement global (65-80 %) est
un symptôme ; la fragmentation est la maladie.

## 4. PISTES NON MESURÉES (hypothèses, pas des faits — à protocoler d'avance)

- **P1 — Instrumenter les vies libres INTÉRIEURES aux longs portages API.** Prendre le
  plus gros porteur de chaque film, l'intervalle où l'API dit qu'il portait, et regarder
  ce que le document publie DEDANS : des vies libres courtes y apparaissent-elles
  (micro-lâchers : bousculade, escalade, saut ?) et à qui la reconstruction attribue-t-elle
  la re-prise ? Hypothèse : le porteur RE-RAMASSE son propre crâne, mais un coéquipier
  proche vole l'attribution à la traversée — ce qui expliquerait À LA FOIS la
  sous-attribution du gros porteur ET la sur-attribution des moyens.
- **P2 — Chaînage même-joueur.** Si P1 confirme : règle « une vie libre < N s dont le
  ramasseur reconstruit est LE MÊME que le porteur précédent, ou dont le précédent
  porteur est à portée » = continuation, pas nouveau portage. À écrire dans le protocole
  AVANT, avec N justifié par la distribution mesurée en P1.
- **P3 — Les événements `th=10` du crâne, jamais élucidés.** Leur compte suit les tics
  dans un rapport 3,1-3,7 ; personne n'a établi ce qu'ils DATENT (D6 les a écartés parce
  que non compris, pas parce que réfutés). S'ils datent des ramassages ou des lâchers,
  ils sont un ancrage temporel exact qui remplace la traversée.
- **P4 — Le statborg du film** (compteurs par joueur côté film) n'a jamais été inventorié
  pour Oddball. Attention : la garde de mode qui protège le pont d'identité (19-22 Go
  hors CTF) — lire son en-tête avant de brancher quoi que ce soit dessus.

## 5. Outillage en place (tout est sur feat/v75)

- Instruments conservés sous garde d'environnement (jamais en CI) :
  `internal/analysis/replay/oddball_crane_d4_test.go`, `oddball_portage_d4_test.go`,
  `oddball_portage_d6_test.go` (+ `_rapport`), `oddball_sonde_d7_test.go`,
  `oddball_sommeil_d8_test.go`, `oddball_traversee_d9_test.go`.
- Mesures brutes figées : `.ai/V7.5/replay2d/registre_film/D4_*.log`, `D6_*.log`/`.json`,
  `D7_sonde_diagnostique.log`, `D8 (dans le log D8)`, `D9 (log D9)`.
- Registre : entrées datées 2026-08-26/27 dans `.ai/V7.5/REGISTRE_REPORTS.md` (fin).
- **RÈGLES NON NÉGOCIABLES héritées du chantier** : tout décodage de film passe par
  l'exécuteur borné `internal/filmproc` (un film = un processus, plafond 2 Gio mesure,
  priorité basse — le garde-rail archlint `no_unbounded_film_loop_test` le force) ;
  protocole COMMITÉ avant toute mesure ; seuils jamais abaissés après coup ; témoins
  spatial ET joueur-aléatoire ; arrêt propre au seuil raté.

## 6. Définition du succès (inchangée)

Recouvrement `time_as_skull_carrier_seconds` ≥ 80 % par joueur ET porteur principal
identifié sur ≥ 3/4 films au pont sain. Si tenu : publication du porteur par vie de
portage dans `objectiveObjects` (prévoir la clé porteur → contrat 39 → 40), rendu
crâne-sur-porteur au patron du drapeau porté, re-cuisson des témoins avec vérification
du CONTENU (recette au registre : bonne version ≠ bonne configuration).
