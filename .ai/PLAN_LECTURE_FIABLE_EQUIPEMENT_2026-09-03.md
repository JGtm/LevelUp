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

## Décisions à trancher (user)

- D1 : R2.3 — appliquer un correctif de scanner (change les artefacts futurs et le golden) ?
- D2 : R3.2 — remplacer MaxSpeedMPS=100 par une règle de réancrage mesurée ?
- D3 : R4 — bump de schéma pour `gap` (nouveaux films seulement) ?
- D4 : ce que le rejeu DESSINE une fois la lecture fiable acquise (faille, passage, éclat) —
  revient sur la table seulement quand R1-R3 ont rendu leurs verdicts.

## Découvertes / suites à arbitrer (issues de R1-R2, hors périmètre des lots courants)

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
