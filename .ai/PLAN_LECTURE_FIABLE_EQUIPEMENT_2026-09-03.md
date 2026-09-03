# PLAN — Lecture fiable des usages d'équipement (2026-09-03)

> Chantier commandé par l'utilisateur le 2026-09-03 (« je ne veux pas de choix ou de règle
> arbitraire, je veux de la lecture fiable ») après le diagnostic translocateur du même jour
> (thought_log 2026-09-03, deux entrées ; encart Lot 3 du
> `PLAN_RETOURS_REJEU_MATCHVIEW_2026-09-02.md`). Exécution PILOTÉE : agents de recherche,
> supervision par la session principale. Worktree `LevelUp-wt-lecture-equipement`, branche
> `wt/lecture-equipement`. Contrat : skill `plan-execution`.
>
> **Doctrine du chantier — pas d'heuristique.** Le lot « saut spatial » proposé la veille est
> ABANDONNÉ sur décision user : une téléportation peut faire 50 cm, tout seuil de distance
> invente une règle. Ce chantier cherche les ENREGISTREMENTS du film eux-mêmes ; un résultat
> négatif mesuré vaut mieux qu'une détection devinée.

## Constats fondateurs (mesurés, 2026-09-03)

1. Match Dynasty `1b2d9e08` : DEUX translocations réelles, toutes deux invisibles aux effets.
   - JGtm slot 535 : saut 22,1 m en 1 frame à t=1762. Sa PRISE du translocateur n'a jamais
     été décodée — le témoin du canal l'a comptée (`coverage.equipmentChanges.counterJumps=1,
     missedEstimate=1`) ; son `spent` (t=1851) porte donc `from=4` (grappin). UNE émission
     manquée aveugle le filtre `from` ET le garde-porteur d'un coup.
   - SpiffyDart86537 slot 560 : saut 7,8 m à t=3420, mort à 3432 — aucune émission
     d'épuisement (mort pendant l'effet).
2. Parc (67 films, auto-mesure du canal i48) : 3 158 émissions décodées, 165 manquées entre
   deux vues (5,0 %), 114 vies sur 2 503 (4,6 %) amputées de leur début. En famille A (rangs
   1-12, hors fenêtre keyframe 16-23), AUCUNE resynchronisation : un manque corrompt
   `from`/état porteur jusqu'à la fin de la vie.
3. La faille ACTIVÉE n'existe dans aucun canal décodé : 16 poses `translocator_beacon` parc =
   15 dropped + 1 unknown, 0 deployed ; les ~12 téléportations réelles connues n'ont AUCUNE
   entité correspondante (ni beacon, ni `other`). `0x730dc70f` observé = l'objet ramassable.
4. Piège de production établi (`filmdec/translocateur_test.go`) : `MaxSpeedMPS = 100` — le
   décodeur de production REJETTE les positions à plus de 100 m/s, or une téléportation fait
   200-400 m/s. La lecture de prod jette structurellement des arrivées de translocation.
   **NUANCÉ PAR R3 (03/09)** : la cascade n'existe pas (réancrage aveugle après 3 rejets) ;
   le coût réel est 1-3 échantillons bruts par téléportation, jamais l'arrivée — voir le
   statut R3.1 et le rapport R3.

## Ancres de vérité terrain (à réutiliser telles quelles)

Conversion : `US = (originMs + t_frame × frameIntervalMs) × 1000`. Dynasty : originMs=9062,
frameIntervalMs=100, bounds monde `-11.45,104.54,73.91,19.72,153.51,82.53`
(format TRANSLOC_BOUNDS : minX,minY,minZ,maxX,maxY,maxZ).

| Ancre | Film | Slot | Fait mesuré | Où chercher |
|---|---|---|---|---|
| A1 | `1b2d9e08` | 535 (JGtm) | saut 22,1 m à t=1762 : (2.79,152.17)→(17.34,135.50) ; taken r=4 @1378 ; spent from=4 @1851 | faille attendue près de (17.34,135.50), posée dans [1378,1762] ; émission i48 manquée dans [1378,1851] |
| A2 | `1b2d9e08` | 560 | saut 7,8 m à t=3420 : (11.32,116.69)→(18.34,120.19) ; taken r=11 @3191 ; mort @3432 | faille près de (18.34,120.19), posée dans [3191,3420] |
| A3 | `a0c36016` | 534 | spent r=11 @1800 ET saut 16,25 m @1801 | la coïncidence parfaite spent/saut |
| A4 | `a0c36016` | 623 | saut 20,9 m @6555, spent @6719 (164 frames après) | multi-usages : le spent date l'épuisement |
| A5 | `4577fcc4` | 514/579/591 | spent @975/@4260/@5362 ; sauts 8-14,7 m @5113-5220 (slot 591) | idem |
| A6 | `f2966f08` / `faff9935` | 523@1014 / 533@1499, 551@2333 | spent isolés | généralisation |

Films sur disque : `LevelUp-go-migration/data/cache/film_chunks/{id8}/` (vérifié le 03/09 :
les 5 témoins présents, 25-34 Mo). Artefacts : `data/cache/replays/halo_infinite/{id8}.json`.

## Questions de recherche (lots)

### R1 — L'entité de la faille activée existe-t-elle dans le film ?

L'identité `eqip 0x730dc70f` vient de la STRUCTURE du jeu (chaîne sofd→sofa→eqip,
`PLAN_NOMMAGE_EQIP_TRANSLOCATEUR.md`), jamais d'une observation d'ACTIVATION. Personne n'a
encore cherché « un objet créé à la position de l'ancre pendant la fenêtre ».

- [x] R1.1 FAIT le 03/09 — NÉGATIF MESURÉ sur trois canaux (créations NEW ti 36-43, deltas
      d'objet du monde, recensement d'images-clés tous ti) : la faille activée n'est PAS une
      entité répliquée lisible. Rapport : `RAPPORT_R1_FAILLE_ACTIVATION_2026-09-03.md` §1-3.
- [x] R1.2 FAIT — POSITIF : la téléportation est l'événement type 117
      `EquipmentTranslocatorTeleportEffects` (nom sourcé de l'exe), tête de paquet `0xFA`,
      ref0 = slot du bipède (porte 1 + index 8 bits base 512 + gen 2), émis 5-80 ms avant la
      discontinuité. Gestes sonores non croisés `[~]` : l'événement date déjà le geste
      exactement (rapport §5). Piège d'horloge MOTEUR vs FILM documenté (rapport §0).
- [x] R1.3 FAIT — précision 18/18 sur 5 films (chaque événement = discontinuité vérifiée
      3,24-25,36 m), rappel 8/8 sur les téléportations mesurées ; 7/10 spent couverts, les 3
      absences documentées non départagées (expiration sans usage, ou événement hors TÊTE de
      liste — le scanner ne lit que le premier événement par paquet). Rapport §4.3.

### R2 — L'émission i48 manquée : perte du film ou perte du scanner ?

- [x] R2.1 FAIT le 03/09 — SCANNER, PROUVÉ : l'émission manquante existe (chunk 9, paquet
      134, bit 3055), record SANS i0, masque [17 25 48 56], marche des désers de production
      OK → compteur 6 = la prédiction, rang 11 = le translocateur. Garde fautive unique :
      `ascendingFromZero`. Rapport : `RAPPORT_R2_I48_MANQUES_2026-09-03.md` §2.
- [x] R2.2 FAIT — 13 sauts classés (8 films) : 10/13 SCANNER, 3/13 FILM. Deux formes
      rejetées par la production : record sans i0, et masque dense R(64) (bit k = composant
      63−k). Récupérable : 77 % libéral / 62 % conservateur. Rapport §3-4.
- [x] R2.3 FAIT — correctif PROPOSÉ, non appliqué (décision D1) : récupération GATÉE PAR LE
      TÉMOIN (re-balayage des seules fenêtres de saut, candidat accepté seulement si son
      compteur comble exactement le saut, étiquette `recovered`). Le relâchement
      inconditionnel est RÉFUTÉ par contrôle négatif (+800 fausses acceptations sur 10
      films, chaînes de compteur détruites). Risque résiduel ~1-2 % par saut. Rapport §5-6.

### R3 — Le filtre MaxSpeedMPS=100 : que perd la production ?

- [x] R3.1 FAIT le 03/09 — LA PRÉMISSE ÉTAIT À MOITIÉ FAUSSE : pas de cascade (`DropTeleports`
      réancre aveuglément après 3 rejets, offline_biped.go:144). Coût réel sur les 18
      téléportations : 1-3 échantillons bruts perdus (49 ms médian, 149 ms pire), arrivée
      JAMAIS perdue ; artefact publié : 18/18 présentes à +0/+1 frame (la grille de 100 ms,
      pas le filtre), seul effet imputable : déplacement 0,00-0,20 m du premier point.
      Découverte latérale : les transitions de manche subissent le même rejet (17 positions
      réelles sur 3 instants). Rapport : `RAPPORT_R3_FILTRE_VITESSE_2026-09-03.md`.
- [x] R3.2 FAIT — comparaison chiffrée : OPTION A (filtre levé à ±200 ms d'un événement 117
      du même slot) couvre 51/51 rejets à tort, 0 fausse exemption, invariance prouvée sur
      les films sans translocateur ; OPTION B (réancrage corroboré) couvre autant mais
      modifie les films témoins → EXCLUE par le cahier des charges. Recommandation : A,
      via `ScanFilmTranslocatorTeleports` — en notant que le gain visible artefact est
      ≤ 0,20 m ; le vrai gain est l'exactitude des positions brutes pour l'aval (D4).
      Décision D2, rien appliqué. Rapport §3-4.

### R4 — Publier l'incertitude : le marqueur `gap` par émission (APRÈS R2)

- [ ] R4.1 Si R2 laisse des manques incompressibles : publier sur chaque `EquipmentChange`
      le saut de compteur constaté depuis l'émission précédente de la vie (`gap: n>0`) —
      schéma +1, nouveaux films seulement, aucune re-cuisson. C'est de la lecture honnête,
      pas une heuristique : le film se mesure lui-même.
- [ ] R4.2 Consommateurs : un `from` sous `gap` n'est plus une identité fiable — les filtres
      (`spentTranslocations`, vignette) le traitent comme inconnu, pas comme faux.

## Protocole de pilotage

- Statuts d'item : fait `[x]` / couvert ailleurs `[~]` (référence) / non traité `[!]`
  (justification écrite). AUCUNE case vide à la clôture d'un lot ; lot N clos (rapport rendu
  + items statués) avant d'ouvrir un lot qui en dépend.
- Efforts : R1 lourd (instrument neuf + décodages) ; R2 moyen ; R3 moyen (instrument
  existant) ; R4 rapide (mécanique, mais GATÉ par la décision D3).
- Gate d'un lot de recherche = son RAPPORT : dénominateurs complets, résultats positifs ET
  négatifs, et pour chaque mesure la COMMANDE EXACTE rejouable (env + go test -run) — une
  mesure qu'on ne peut pas rejouer n'existe pas. En plus : `go vet ./...` vert sur le
  package touché par tout instrument ajouté.
- Reprise de session : ce fichier (statuts) + `.ai/thought_log.md` (dernière entrée) +
  les rapports `RAPPORT_R*` déjà rendus.
- Découvertes hors périmètre : consignées en fin de ce fichier, JAMAIS traitées dans le lot.
- Agents de recherche en parallèle : R1 et R2 (indépendants). R3 après R1 (réutilise
  l'instrument translocateur ; éviter deux décodages lourds concurrents du même film). R4
  après verdict R2 ET décision D3.
- Chaque agent : LECTURE SEULE de la production, instruments en `*_research_test.go` gardés
  par variable d'environnement (patron `TRANSLOC_FILM`), `CGO_ENABLED=0`, UN décodage filmdec
  par process (`LockProcessDecode`), balayages BORNÉS (chunks/fenêtres — le balayage sans
  filtre d'un film entier tue le process, mesuré le 2026-08-18). Jamais d'ouverture des
  DuckDB (serveur actif). Aucun commit par les agents — consolidation par le superviseur.
- Rapports : `.ai/V7.5/replay2d/RAPPORT_R<н>_<sujet>_2026-09-XX.md` avec dénominateurs,
  positifs ET négatifs, chemins fichier:ligne des instruments.
- Gates chantier : `go vet ./...` + tests du package touché ; aucun garde-rail affaibli ;
  thought_log à chaque étape franchie ; delivery-checklist avant tout commit.

## Décisions — TRANCHÉES le 2026-09-03 (user : « ok à tout ; la solution la plus propre,
## solide et pérenne »)

- D1 OK : récupération gatée par le témoin de compteur, étiquette `recovered`.
- D2 OK : option A — filtre levé à ±200 ms d'un événement 117 du même slot (jamais ailleurs).
- D3 OK : marqueur `gap` par émission (nouveaux films seulement, aucune re-cuisson).
- D4 OK : publication `translocations[]` + rendu (éclat daté à l'événement, lien du
  va-et-vient, faille APRÈS le premier échange, fin mesurée — jamais à demeure).

## Phase de production (après R1-R3 ; commits lot par lot sur cette branche)

### Lot P1 — Go, schéma 38 : la lecture fiable entre en production

- [x] P1.0 FAIT le 03/09 — ORDRE CONFIRMÉ : `bit k = composant 63−k` (msb=true) sur 40 témoins
      SUPPLÉMENTAIRES au profil conservateur (gardes réduites à `dense`), 12 films nouveaux
      (bfcd1175 x4, 4f77afc1 x5, bf2a9f05 x5, 58801bc5 x4, 9ffce8ef x4, 8a485699/9b4dba45/
      d1dfbc02/e1259a69 x3, 0797ce72/30724141/879a4dba x2), tous au compteur prédit, 38/40 dans
      la famille structurelle [0 1 (5) 17 21 25 (26|28) 48 (56|58...)]. UN SEUL contre-signal
      msb=false propre (bfcd1175 slot 528, fenêtre de 364 s / 21 812 paquets, masque hors
      famille, compteur à 3 chances sur 8 de tomber dans la prédiction) = plancher de bruit
      R2 §5 — et il est ILLISIBLE par la forme de production (relu msb=true : idx[0]=3 ≠ 0,
      rejeté par construction). La forme dense ENTRE en récupération, restreinte à msb=true.
      Bilan additionnel (10 films, 58 sauts) : 42 SCANNER / 16 FILM, 82 % des émissions
      retrouvées. Commande rejouable (depuis apps/go-api ; <b> = film_chunks) :
      `CGO_ENABLED=0 I48M_PARC="<b>/9ffce8ef,<b>/4f77afc1,<b>/0797ce72,<b>/8a485699,
      <b>/bf2a9f05,<b>/d1dfbc02,<b>/30724141,<b>/879a4dba,<b>/58801bc5,<b>/e1259a69"
      I48M_MAXJUMPS=80 go test ./internal/analysis/filmdec/ -run '^TestI48ManquesParc$' -v`
      (+ lots bfcd1175 et 9b4dba45 joués séparément, mêmes env).
- [x] P1.1 (D1) FAIT le 03/09 — post-passe `equipment_recovery.go` dans
      `ScanFilmEquipmentChanges` (aucune garde du strict affaiblie) : fenêtres de saut
      seules, formes « sans i0 » (en-tête production intact, indices croissants, premier
      != 0) et « dense msb=true » (i0 absolu + région exigés), candidat accepté SEULEMENT
      au compteur prédit, sans doublon (ambiguïté = fenêtre rejetée), dans l'ordre, sans
      créer ni répétition ni nouveau saut (règle du trou unique) ; `Recovered` publié
      (filmdec + document + coverage). `livesFirstOffSpec` COUVERT (fenêtre [naissance,
      première émission], compteur virtuel 4 = même arithmétique) : MESURÉ SAIN sur les
      9 témoins — 13 récupérées au total, chaque récupération mi-vie = exactement un hit
      conservateur R2, chaque perte FILM/profil bruité laissée en résiduel, 0 répétition
      créée, 5 têtes de vie combles à la prédiction exacte (1b2d9e08:531, 0a44c6cc:533,
      28c9b538:512 x2, c75f33b8:543). Preuve : TestP1RecuperationDynasty (gaté P1_FILM) —
      slot 535 recovered c6 rang 11 @5700481685us, spent from=11, chaînes closes ; tests
      unitaires equipment_recovery_test.go (fenêtres, témoin de compteur, ordre dense).
- [x] P1.2 (D3) FAIT — `Gap` par émission (saut RÉSIDUEL après récupération, 0 = chaîne
      saine, première émission = 0, tête au coverage LivesFirstOffSpec) ; publié
      `equipmentChanges[].gap` (omitempty) + `coverage.equipmentChanges.recovered`.
      Preuve : TestBuildEquipmentChangesPublieRecuperationEtGap + TestCounterGap.
- [x] P1.3 (D4) FAIT — `filmdec.ScanFilmTranslocatorTeleports` (patron zoom_events.go,
      tête 0xFA, type 117, ref0 8 bits base 512, refs 1-2 non lues) + calque
      `translocations[]` = {slot, t} (avant-origine et sans-piste écartés et comptés,
      `coverage.translocations` toujours publié). Aucun rang, aucune heuristique. Preuve :
      TestDecodeTranslocHead(+Refuse), TestBuildTranslocations ; Dynasty : 3/3 événements
      lus et publiés {535@1761, 560@3261, 560@3419}.
- [x] P1.4 (D2) FAIT — `DropTeleportsExcept` + `TeleportExemptions` (±200 000 µs, horloge
      des paquets = celle des événements, aucun décalage), branché dans
      ScanFilmBipedPositions via ScanFilmOptions.TeleportExemptions ; BuildFromFilm scanne
      les événements AVANT les positions. Invariance : unitaire
      (TestDropTeleportsInvarianceSansEvenement, nil/vide/construit-de-rien identiques) ET
      sur pièces (TestP1InvarianceSansTete117 : 7344d24f, 0 tête 117, 188 979 échantillons
      bit à bit identiques). Effet Dynasty : 7 échantillons ré-acceptés (3+1+3 = les 51/51
      de R3 pour ce film), 0 hors fenêtre (TestP1ExemptionVitesseDynasty).
- [x] P1.5 FAIT — SchemaVersion 38 + chronique (document.go, structure_test.go), golden
      réassemblé par la recette (fixture REPLAYINPUTS11 : + Translocations, parité
      decodeFilmInputs avec l'exemption ; assembly : schéma 38 + section TRANSLOCATEUR
      figée à zéro avec ses compteurs, phrase gardée par golden_guard), contracttest
      (49→50 champs racine, Translocation/TranslocationCoverage dans replaySchemas),
      openapi.yaml régénéré (`make openapi-gen`, +62 lignes : recovered/gap/translocations
      + coverage), `make generate-types` (+29 lignes generated.ts) + frontière web comblée
      (replayNormalize.ts + replayContract.test.ts : `translocations` énuméré et comblé —
      partie MÉCANIQUE du garde de contrat, la consommation reste P2), `make check-types`
      exit 0. Gates : go test filmdec+replay+replaybuild+contracttest exit 0 (0 --- FAIL),
      go vet exit 0, vitest replayContract 13/13.

Gate P1 : `CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/...`
+ replaybuild + contracttest + `go vet ./...` (packages touchés) + `make check-types` (types
régénérés) ; aucun garde-rail affaibli ; thought_log.

REVUE ADVERSARIALE RONDE 1 (03/09, 2 relecteurs aveugles) : 8 constats recevables, 0 jeté,
corrections F1-F6 appliquées — F1 P0 garde de borne forme sans-i0 (panic queue de paquet,
repro figée en test) ; F2 helper unique `counterStep` + garde-rail grep ; F3 départage par
offset de bit au paquet frontière + VERROU FINAL sur la chaîne publiée
(pruneRecoveredViolations) ; F4 fusion extraite pure + tests tueurs de mutations ; F5
fixtures multi-îlots pour `covers` ; F6 invariance D2 prouvée contre une implémentation de
RÉFÉRENCE figée (l'ancienne comparaison était circulaire), commentaires de prod corrigés.
Gates rejoués verts, Dynasty re-cuit : constats (a)(b)(c) inchangés au chiffre près.
Détail : thought_log entrée « P1 — revue adversariale ronde 1 ».

REVUE RONDE 2 (03/09, relecteur frais, corrections seules) : 6/6 corrections FERMÉES
(mutations rejouées et tuées : garde P0 — panic reproduite sans elle —, gap=0,
comparateur inversé, ts[0], référence figée vérifiée contre d28a3aa6a ligne à ligne) ;
17 conditions tiennent. UN défaut nouveau, P2 CONSIGNÉ (pas corrigé dans ce diff, borne
de boucle) : voir Découvertes « verrou tête partielle ». P0+P1 : 7 → 0. LOT MERGEABLE.

### Lot P2 — Web : consommer la lecture fiable (après P1)

- [ ] P2.1 Éclat de fiche daté par `translocations[]` (l'événement, plus jamais le spent) ;
      `spentTranslocations` retiré ou réduit au repli des artefacts < 38 (kill-switch daté).
- [ ] P2.2 Lien du va-et-vient sur la carte : from/to dérivés de la discontinuité de piste à
      la frame de l'événement ; `riftTeleports` (heuristique spatiale) retiré (CLAUDE.md n7)
      — les artefacts < 38 gardent le comportement actuel via le repli.
- [ ] P2.3 Faille dessinée APRÈS le premier échange (position = point de départ du saut,
      forme `drawRift` existante), déplacée à chaque échange, éteinte à fin mesurée
      (spent/`recovered`, mort du porteur) — jamais à demeure (point user expiration).
- [ ] P2.4 `from` sous `gap` traité comme inconnu par les consommateurs (vignette comprise).
- [ ] P2.5 i18n FR/EN si libellé ; tests logic ; plafonds de taille.

Gate P2 : `make check-types && make test-web` + gate visuel user sur un film re-cuit témoin.

### Lot R5 — Recherche : les événements des AUTRES équipements (parallèle)

- [ ] R5.1 Inventaire des types d'événements nommés (annexe A de la grammaire, table
      chunk00) : lesquels parlent d'équipement/usage (répulseur, propulseur, camo,
      surbouclier, écran, capteur, réparation…) ?
- [ ] R5.2 Recensement des têtes par type sur ≥ 5 films et CROISEMENT avec les usages déjà
      mesurés par ailleurs (grappleLines, kills répulseur, épisodes camo/surbouclier, poses
      deployed datées) — précision/rappel par type, comme R1 l'a fait pour 117.
- [ ] R5.3 Rapport : quels équipements gagnent un événement d'usage fiable, lesquels
      restent sans signal — et ce que la LISTE COMPLÈTE (vs tête seule) rapporterait.

### Lot R6 — Ghidra : charge du 117 et émetteurs (commandé oralement le 03/09, user : « si
### t'as besoin de ghidra tu peux utiliser ») — FAIT 03/09

- [x] R6.A Charge utile du 117 DÉCODÉE ET VALIDÉE 18/18 (écarts 0,00-0,26 m, 5 films) :
      `[R(1); R(32) effet = 0xA1344FC2 constant]` + DEUX positions quantifiées — A = DÉPART,
      B = ARRIVÉE du saut. Quantification aux bornes vraies (`map_quant_bounds.json`) ; la
      formule des largeurs lue dans l'exe reproduit exactement les axisWidths du catalogue.
      PIÈGE sourcé : porte région du vecteur INVERSÉE (bit 0 → index région + bornes carte).
      Le « mot 0x42689F84 » du rapport R1 était un artefact de fenêtre décalé d'un bit.
      LA POSE RESTE ILLISIBLE : la charge ne porte rien d'autre — c'est le verdict final du
      canal. Rapport : `RAPPORT_R6_GHIDRA_EVENTS_2026-09-03.md`.
- [x] R6.B Émetteurs des types 30/42/43/48/93/104 introuvables statiquement (réception
      fonctionnelle, diffusion non filtrée par type → l'absence vient de l'ÉMISSION) ;
      sondage liste via grammaires fermées : 0 occurrence des 13 types cibles derrière 597
      têtes marchées. Verdict : « aucune trace ; indéterminé formellement, faisceau vers
      jamais émis en multi » (42/43/93 gestes IA/campagne, 48/31/119 requêtes C2S, 104
      notification ciblée). Répulseur/propulseur : le film ne les enregistre pas — clos.
- Conséquence produit : **P1bis** (à exécuter après la ronde 2 de revue, AVANT toute
  cuisson) — `translocations[]` gagne les positions from/to décodées de la charge (le
  schéma 38 n'est déployé nulle part : on l'enrichit avant sa première cuisson, pas de 39).
  Le lien du va-et-vient (P2.2) se lira alors de l'événement seul, sans dérivation de
  discontinuité de piste.

### Re-cuisson

Un film témoin d'abord (Dynasty `1b2d9e08`), sur accord explicite user, après P1+P2 —
jamais de cuisson de masse sans décision séparée.

## Découvertes / suites à arbitrer (issues de R1-R2, hors périmètre des lots courants)

- P2 revue ronde 2 (03/09) — VERROU TÊTE PARTIELLE : `equipment_changes.go:271`
  (`pruneRecoveredViolations`) / cause `:290` (`chainViolations` ignore le compteur virtuel
  de tête `equipRecoveryHeadCounter`) : une récupérée de TÊTE de vie partielle non contiguë
  à la première stricte (ex. c5 seul retrouvé pour fromC=4, première stricte c7) est
  acceptée par le témoin puis RETIRÉE par le verrou (sonde exécutée : out=2, Recovered=0).
  Perte conservatrice (aucune donnée fausse), périmètre étroit. Piste : amorcer
  `chainViolations` avec le compteur virtuel de tête. À CORRIGER EN P1bis avec son test.
- R6 latéral (03/09) — `unit_zoom` (type 21) apparaît EN POSITION 2 de la liste
  d'événements (5 occurrences) : le canal zoom de production ne lit que les TÊTES → il
  sous-compte. Dette du canal zoom existant, hors chantier — à cadrer séparément.
- R6 latéral — un événement type 15 `Script` suit chaque 117 ; `action_weapon_fire` vu en
  position 2 (20 occ.). Matière pour le chantier « liste complète » (PLAN_PERCER_TRAME_FILM).
- R6 environnement — GhidraMCP headless : piège JDK 25 (`Selector.open()` échoue ;
  contournement documenté) + commande complète : RAPPORT_R6 §0 et annexe C.1.

- P1 (03/09) — VALIDATION DYNASTY 1b2d9e08, artefact re-cuit schéma 38 (worktree,
  data/cache/replays du worktree, films lus en lecture seule) : avant = 31 changements,
  1 saut / 1 manquante / 1 tête hors norme ; après = 33 changements (+2 récupérées),
  0/0/0 résiduel, 0 gap non nul. (a) slot 535 : {t:1378 taken r4} → {t:1520 taken r11
  from=4 recovered:true} → {t:1851 spent from=11}. (b) translocations = [{535,1761},
  {560,3261},{560,3419}] (l'événement précède l'arrivée : arrivées publiées 1762/3262/3420).
  (c) exemption : 7 échantillons bruts ré-acceptés dans les fenêtres ±200 ms, 0 hors
  fenêtre ; les POINTS PUBLIÉS (34 114) sont identiques à l'ancien artefact — conforme à
  R3 §5 (« le bénéfice visible artefact est <= 0,20 m ; le vrai gain est d'amont ») : sur
  ce film, les échantillons ré-acceptés tombent dans des frames dont le premier
  échantillon publié ne change pas. Le gain est dans les positions BRUTES (aval :
  appariements, marqueurs d'usage).
- P1 (03/09) — la frontière web (replayNormalize/replayContract) a reçu `translocations`
  dans ce lot (exigence du garde de compilation, gate check-types) ; P2.1-P2.4 partent
  donc d'une frontière déjà comble — il leur reste la CONSOMMATION (éclat, va-et-vient,
  faille, `from` sous gap).

- Décodage de la LISTE COMPLÈTE d'événements par paquet (le scanner actuel ne lit que la
  TÊTE) : lèverait la seule réserve du rappel R1 (3 spent sans événement), et vaut pour tous
  les types (zoom compris).
- Charge utile de l'événement 117 (`FUN_140f04fb8`, 28 octets, mot constant `0x42689F84`) :
  les positions de départ/arrivée y sont probablement — RE exe (Ghidra) si on les veut.
- Sémantique établie : le translocateur est un VA-ET-VIENT (la balise échange sa position
  avec le joueur à chaque usage) — toute UI « faille posée fixe » serait fausse.
- Brique de production candidate : `ScanFilmTranslocatorTeleports` sur le modèle de
  `zoom_events.go` (~60 lignes) — date chaque téléportation sans aucune heuristique. À
  cadrer avec D4 (schéma +1).
- Cas `livesFirstOffSpec` (manques AVANT la première émission d'une vie) : même mécanique de
  récupération gatée, fenêtre [naissance, première émission] — non mesuré par R2.
- EXPIRATION DE LA FAILLE (point user du 03/09) — première mesure, 8 fins appariées
  (spent croisé au dernier événement 117 du même slot, 5 films) : DEUX modes de fin.
  (a) épuisement PAR l'usage final : spent quasi simultané du dernier saut (2/8 : ~0 s et
  1,2 s) ; (b) expiration APRÈS le dernier échange : 9,0 / 12,6 / 14,2 / 15,3 / 16,5 /
  16,5 s (6/8). La mort du porteur clôt aussi (Dynasty slot 560, mort 1,2 s après son
  échange — aucun spent). Conséquences : le spent n'est PAS l'instant d'usage (jusqu'à
  16,5 s de retard mesuré) ; et une UI de faille ne doit JAMAIS la dessiner à demeure —
  fin = spent (récupéré R2/R4), ou mort du porteur. La POSE reste illisible (ni entité ni
  événement) : entre la pose et le premier échange, la position de la balise n'est connue
  d'aucun canal — la charge utile de l'événement 117 (`FUN_140f04fb8`) est la seule piste
  pour la combler. Échantillon N=8 : à consolider si D4 dessine la faille.

## Hors périmètre (noté, non traité)

- Résolution de camp `team: -1` sur les tracks Dynasty (vu au diagnostic du 03/09).
- Vignette famille A (dégradation honnête) : dépend de R2/R4, cadrage après verdicts.
- Le canal weaponChanges partage sans doute des trous analogues (complétude jamais prouvée,
  cf. `changeRefine.ts` en-tête) — même méthode applicable, chantier distinct.
