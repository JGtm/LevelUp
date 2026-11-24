# Rapport L.1 (2026-11-24) - decodeur v1.6.0, tete eb0ae98

Hote : LevelUp-wt-dec160, fast-forward sur origin/chore/830-decodeur-v150 (aab869c44e), aucun conflit.

## Gates
| Gate | Resultat |
|---|---|
| Build / vet / vet -tags=research (clone neuf, GOWORK=off) | vert |
| G-clone sans racine (go test -count=1 -v -p 1 ./...) | 4444 PASS / 434 SKIP / 0 FAIL, 35 paquets ok (= reference de l'integrateur) |
| G-clone avec racine (dossier principal, lecture seule) | 4576 PASS / 297 SKIP / 14 FAIL (sous-tests inclus). Rouges : 4 TestContractFixtures* (attendu) et TestTableStatborgConcordeAvecNamedStatSlots (NON prevu : .ai/refs/TABLE_STATS_STATBORG.tsv du dossier principal a 48 lignes, celui de LevelUp-wt-dec160 en a 56 ; le test passe sur la racine wt-dec160). Les 4 TestContractFixtures passent sur la racine de l'integrateur (integ/racine). 297 SKIP au lieu de 301 : non investigue |
| G-rev | 10 PASS + 43 PASS, vert |
| G-compat (gorelease@v0.0.0-20261121134650-e1094eeca1b5, -base=v1.5.0, GOWORK=off) | exit 1, SEULES incompatibilites : SchemaDesFaits 30->31, SchemaVersion 126->127 (admises) |
| G-facade (hote, dec160.work) | vert, aucune adaptation de l'hote |
| G-equiv-ref | avant refigeage 0/20 identiques (attendu) ; apres refigeage 20 identiques, 0 different. Controle : le meme harnais sur un clone v1.5.0 rend 20 identiques contre les references d'origine ; la tete rejouee deux fois rend des TSV identiques a l'octet |
| G-equiv-2p | 20 artefacts IDENTIQUES a l'octet, 0 divergent ; 20 ecarts a classer, tous "kill" (ecart attendu killsource) |
| Registre des replis vs FORMAT.md | ROUGE documentaire : registre = 128 entrees (127 true, 1 false), FORMAT.md l. 896 dit "127 declarations". Cause : repli_tireur_par_le_siege_de_la_place (apport 15) ajoute sans mise a jour de FORMAT.md |
| G-corpus (--reference=base --base aab869c44e --strict, replace local, GOWORK=off, 380 s) | ROUGE : 19 temoins = 4 ok, 12 PERTE, 3 FAUX (voir plus bas) |

Sorties brutes dans dec-v160-recup/l1/ : g_*.txt, comparaison_par_etape.txt, jdiff.txt, g_corpus_pertes.txt, gate_work_corpus/gate.log et gate.json.

## G-equiv-ref : attribution (references v1.5.0 contre tete)
Methode : (a) comparaison par nom d'etape ; (b) dump des valeurs d'etape en v1.5.0 et en tete par un outil jetable, diff champ par champ. Aucune etape disparue ; 10 etapes neuves sur les 20 films.

| Etape (films) | Changement observe | Attribution |
|---|---|---|
| repliques(.stats), gestures(.stats), drawnWeapons(.stats), projectileAttaches(.stats), appearances(.stats) : neuves, 20/20 | etapes neuves | apports 1, 2, 11, 17, 21 |
| artifact (20) | octets | schema 127 |
| controlHolds (18), controlHolds.stats (20) | verifie : champ Kind, longueurs (956->1465 sur 000d5950), stats gagne Move/Crouch/Jump/Gaps/StepGaps/StepUS ; 3 films sans lecture de controle : forme seule | apport 3 |
| reloads (16) | verifie : champ Copies ajoute, comptes egaux | apport 2 |
| grenades (20) | verifie : champs Life et HasLife, comptes egaux | apport 12 |
| vehicles (10) | digest seul (champ interne) ; les 10 films changes = exactement les 10 films au journal "aucun slot ti=40" | apport 16 (deduit par concordance des ensembles) |
| flag (5 BTB : 084a804d, 111fa685, 11de8353, 1c4c63c2, e5adf7b2) | verifie : Records en plus (+860, +415, +369, +1443, +460) | apport 5 (tous les sieges) |
| scoreTimeline (11) | verifie : +2 a +5 points par courbe | apport 18 (penalite) ; la chronique annonce "11 films sur 20" : meme nombre |
| eliminations (17 ; 01e1f945 106->107) | 01e1f945 et bcb6d393 : place / victimPlace (bots) ; les 15 autres : digest seul ; les 3 films sans eliminations inchanges | apport 4 (deduit pour les 15) |
| objectives (bcb6d393 84->97) | verifie : 13 lignes ajoutees, aucune retiree | apport 5 (siege qui change d'occupant) |
| movementStates.stats, gameEvents.stats, damages.stats, reloads.stats (53ce4390, 64e8adfa) | verifie : trames arretees 3->0 et 4->0, terminees +3 et +4 | apport 7 (1), genres 27/93/117 lus sur HI_1_13_0 |

Tous les ecarts sont attribues ; 20 films refiges : 20 fichiers replay/testdata/equivalence/<short8>.tsv (3 colonnes, aucun identifiant, pseudo ni chemin, verifie par grep).

## G-corpus : ecarts a presenter (rien inscrit au manifeste)
Statut : ok x4 (0797ce72, 111fa685, e5adf7b2, 11de8353), PERTE x12, FAUX x3. Schema 126 -> 127 des deux cotes ; revisions de grammaire, objectifs et profil changees sur 19/19 (telemetrie). 20 lignes "decision sans effet - a retirer du manifeste" (deja connues, non traitees).

FAUX (le gate tient tout repli nouveau pour FAUX) : repli_tireur_par_le_siege_de_la_place (apport 15), absent de la base : 084a804d 0->2 ; 50247b26 0->1256 ; 4f77afc1 0->6.

PERTE (avant -> apres ; cause probable) :
- Socles et armes au sol, 17 temoins : cause probable apport 12 (les socles d'equipement entrent dans weaponPads ; sur 0a247154 occupancies 38->49, occupations 33->43, powerupOccupations 13->23). Deduit.
  - padDating.uncovered : bcb6d393 4->5, fb1a1a72 14->15, d9781168 24->26, c75f33b8 33->38, bf15f7ab 4->5, 51ebbc0f 16->18, 084a804d 89->143, 60ae07c4 11->18, a349fea8 97->133, a521164d 17->32, 50247b26 25->39, bfecd02b 17->18, 4f77afc1 121->159, 396cfc92 4->5, 0a247154 7->10.
  - padDating.ambiguous : fb1a1a72 13->14, d9781168 4->6, 084a804d 71->84, 60ae07c4 7->9, a349fea8 36->40, 50247b26 8->12, 4f77afc1 60->65.
  - groundWeapons.unknown : d9781168 13->14, c75f33b8 27->32, 51ebbc0f 9->10, 084a804d 127->134, 60ae07c4 5->8, a349fea8 67->77, a521164d 10->12, bfecd02b 11->12, 4f77afc1 91->97.
- coverage.fallbacks/n +1 : 50247b26 45->46, 4f77afc1 40->41 (le repli neuf).
- A REGARDER EN PRIORITE, 084a804d (BTB) : flagCarries.gaugePoints 183->0, gaugeSpans 1->0, gaugePaired 1->0, carrierTeamUnknown 23->41 : la jauge du drapeau disparait. Cause probable apport 5 ; non demontre.
- bcb6d393 : flagCarries.ambiguousReturns 0->1, noBridge 0->1, objectives.noSlot 0->2 (probable apport 5 : 13 lignes d'objectif ajoutees ; deduit).
- 084a804d : shots.v/presents 475->474, vehicles.shots 475->474, vehicles.shotsVehicleWeapon 421->420 (un tir de moins ; probable apport 15 ; non demontre).
- 4f77afc1 : hits.w/presents 2019->2005 ; un joueur, score 100->0 (probable apport 18, penalite de 100 ; deduit).

## Fichiers modifies non commites
- Decodeur dec-v160 (feat/v1.6.0, eb0ae98) : 20 fichiers replay/testdata/equivalence/<short8>.tsv. Rien d'autre.
- Hote LevelUp-wt-dec160 : aucun (merge en fast-forward, HEAD = aab869c44e). go.mod modifie pour G-corpus (replace local) puis restaure ; outil jetable cmd/zz_equiv_dump supprime. Aucune adaptation de l'hote n'a ete necessaire.
- Hors depot : dec-v160-recup/l1/ contient des dumps avec de vrais identifiants (dump_v150/, dump_v160/, gate_work_corpus/gate.log) : ne rien en copier dans un depot public.

## Ecarts au plan
- Pas de -fils (inexistant). G-corpus : go.mod hote avec replace local (valide par le coordinateur). Racine de G-clone : dossier principal en lecture seule + racine integ pour les fixtures 127.
