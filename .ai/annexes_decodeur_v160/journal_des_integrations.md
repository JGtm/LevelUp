# Journal des intégrations du décodeur v1.6.0 (6 intégrations, 2026-11-23)

Rédigé le 2026-11-23 par l'intégrateur, repris tel quel plus bas, précédé d'un tableau de synthèse tiré des pièces
(suites de tests, courses, comparaison à un et quatre fils, compatibilité publique).

Lecture du tableau : « tests passés » compte les lignes `--- PASS` (tests et sous-tests) de la suite complète rejouée avec `-v`.
Les 301 tests sautés sont la référence stable (tests qui lisent le jeu installé). Les quatre tests rouges sont toujours les
mêmes : les fixtures de contrat du dépôt hôte (LevelUp) ne sont pas encore recopiées au schéma 127 ; ils sont verts sur la racine
bac à sable de l'intégrateur. Aucun autre test ne rougit. Courses : paquets `replay` et `internal/grammar` sous le
détecteur de courses, aucun signalement. Identité : les empreintes du document et des faits des 13 films du relevé sont égales à
un fil et à quatre fils (13 films sur 13). Compatibilité publique : `gorelease`, seuls changements dits incompatibles : les
deux constantes de schéma (126 vers 127 pour le document, 30 vers 31 pour les faits) ; tout le reste est ajout.

| N | Tête après l'intégration | Ce qui entre | Tests passés | Sautés | Rouges | Courses | 1 fil = 4 fils |
|---|---|---|---|---|---|---|---|
| 1 | `2ac0a7e` | score (A.7, A.2a, A.2b, A.2c), drapeau (B.2), éliminations de bots, répliques, gestes, schéma 127 et faits 31 | 4 393 | 301 | 4 contrat | aucune | 13 sur 13 |
| 2 | `ddf99f9` | tirs (décalage, dégâts par piste et à zéro point), arme tirée, munitions, socles, lancers sans vol | 4 438 | 301 | 4 contrat | aucune | 13 sur 13 |
| 3 | `2a49ff4` | pleine cadence toujours publiée, santé et bouclier par vie | 4 466 | 301 | 4 contrat | aucune | 13 sur 13 |
| 4 | `46be3cf` | tireur par place de contrôle, visées sans position sur tout film | 4 494 | 301 | 4 contrat | aucune | 13 sur 13 |
| 5 | `f13aacd` | projectiles collés | 4 516 | 301 | 4 contrat | aucune | 13 sur 13 |
| 6 | `7bda46f` | pénalité de score, table des réponses en révision, statistiques VIP, victime ambiguë | 4 538 | 301 | 4 contrat | aucune | 13 sur 13 |

Compatibilité : à la première intégration, `gorelease` signalait en plus des retraits (deux champs `SeatsBeyondBand`, trois
compteurs de `ControlHoldStats`) ; ils ont été conservés et dépréciés (`0cc5366`). À la deuxième, un type de socle d'arme
devenu non comparable a été corrigé (`2ac0a7e`). À la sixième, la structure d'une couche de révision l'était aussi : corrigée
par `7bda46f`. À la tête finale, plus rien d'incompatible hors les deux constantes.

Poids du document compressé (gzip niveau 6), trois films : v1.5.0 329 993 / 840 362 / 7 833 786 octets ; fin de l'intégration 2 :
348 591 / 889 584 / 8 148 765 ; fin de la 3 : 481 783 / 1 223 306 / 11 231 739 ; fin de la 5 : 482 325 / 1 225 879 / 11 368 190,
soit de 45 % à 46 % de plus que la v1.5.0 (pleine cadence et santé et bouclier par vie pèsent l'essentiel). Pic de mémoire du plus
gros film, un fil : 1 726 Mio (3) puis 1 624 Mio (5).

Texte de l'intégrateur :

# Journal d'integration
- a7 : fusion sans conflit ; compile ; tests replay Score verts.
- a2a : fusion sans conflit ; grammar : seul TestGrammarRevSuitLaGrammaire rouge (attendu).
- a2b : fusion sans conflit ; tests cibles verts.
- a2c : fusion sans conflit ; objectives : seul TestObjectivesRevSuitLaSortie rouge (attendu).
- b2d (B.2a, B.2b, B.2d) : fusion sans conflit ; compile, vet ; replay vert ; objectives : seul TestObjectivesRevSuitLaSortie rouge.
- b2c : fusion sans conflit ; tests drapeau verts.
- e1c : fusion sans conflit ; sonde zz_e1c absente de l'arbre (ajoutee puis retiree par la branche) ; tests eliminations/scoreActions verts.
- d3 : fusion automatique sans conflit (canal deja au bon endroit : lire = marcherPourLaCuisson, publier = balayerEtatsDeMouvement) ; replay rouge sur : TestContractFixturesMatchCommitted/bcb6d393 + TestGoldenBuildsAssembly/bcb6d393 (e1c), TestDocumentShapeMatchesGolden, TestCodecCouvreFilmInputs et TestCalquesCouvrentTousLesChampsRacineCuits (d3) : a poser, attendu.
- gestes-a : fusion automatique sans conflit ; compile, vet, tests cibles.
- gestes-a : fusion automatique sans conflit ; compile, vet, tests cibles verts.
- gestes-b : conflits d'UNION (deux canaux neufs au meme endroit) dans movement_states.go, build.go, coverage.go, document.go, film_inputs.go, film_scan_mouvement.go, observe.go, observe_test.go, options.go. Resolution : garder les deux (repliques puis gestes, dans cet ordre partout), canaux de la marche `[sc, tir, jeu, degats, vitalites, rechargements, repliques, gestes]`, etapes observees `repliques, repliques.stats, gestures, gestures.stats, filmScripts`, compte de `ScanMarcheDesTramesAvec` passe a 12 (10 de la tete + 1 + 1 ; les deux branches l'avaient montre a 11 chacune, l'auto-fusion aurait laisse 11) et commentaire du test en consequence. La forme de la tete est tenue (lire = marcherPourLaCuisson rend la valeur ; publier = balayerEtatsDeMouvement observe et verse).
- Poses de l'integrateur : layers.go (chatter, announcements + gardes) ; codec des faits (ordre : rechargements, repliques, gestes, commandes) ; porte de cadence des gestes = ControlHoldStats.PleineCadence() (champ CadenceNonPleine supprime, test reecrit) ; coverage.flagCarries.seatsBeyondBand retire ; SchemaVersion 127, SchemaDesFaits 31 (historique des schemas sorti de filmfacts_fichier.go vers filmfacts_schemas.go) ; grammar.Rev grammar-2026-11-23, objectives.Rev objectives-2026-11-23 ; source.Rev et killsource.Rev inchangees (goldens refiges a revision constante : seules des formes de types y entrent) ; options.go ramene a 500 lignes ; plafonds de taille_test mis a jour (chronique, structure_test).
- Fixtures de contrat : le dossier vit dans le depot hote (LevelUp-wt-dec160, branche chore/830-decodeur-v160), hors de mon perimetre d'ecriture. Regenerees dans une racine bac a sable (recup/integ/racine, copie de reference/ et des fixtures 126) ; a recopier par le mainteneur dans apps/web/src/features/match-replay/test/fixtures/go/ de l'hote (supprimer les 8 fichiers 126, ajouter 8 fichiers 127 et manifest.json).
- Non regenere : replay/testdata/equivalence/*.tsv (corpus local, harnais replay-equiv absent de ce depot).
- Preuves : suite complete (-v) : seuls TestContractFixtures* rouges (fixtures de l'hote, voir plus haut ; verts sur la racine bac a sable) ; 301 lignes SKIP ; race x3 vert ; gorelease : retraits signales (SeatsBeyondBand x2, ControlHoldStats.Holds/WithHole/InnerHoles) ; film-bench 13 films : empreintes 1 fil == 4 fils pour les 13, etapes identiques hors en-tete ; toutes differentes de v1.5.0.

## Seconde integration (depuis 2ac0a7e)
- p-tirs : fusion sans conflit ; rendu golden_degats lit les lignes a points (goldens d'assemblage inchanges).
- p-arme : fusion sans conflit ; compile, vet, tests de gardes verts.
- p-socles : fusion sans conflit. Poses : ligne padCandidates de layers.go, chronique v127 (apports 10 a 12), historique des faits (munitions lues par ammoReads, arme tiree, handle des lancers), chronique de la grammaire (cinquieme apport), FORMAT/GLOSSARY/NOTICE, table ordresFiges (grenade_events.go) et formes de types (DrawnWeaponRead, DrawnWeaponStats), golden des familles de la mini-bobine, equivalence mini-bobine.
- Fixtures d'entrees : pipeline_fixtures.sh (table d'identites 4 + pseudonymisation par xuid depuis les anciennes fixtures) ; recontrole : 0 xuid d'origine (varint) dans les 8 blobs et 3 graines, 0 en clair dans le diff texte, 0 nom hors anciens dans les goldens.
- Plafond des fixtures de contrat : 5 226 571 octets mesures (> 5 111 808) ; releve a 5 Mio, decision proposee.
- Poids document (gzip niveau 6, options minimales du banc) avant/apres : film05 335 683 -> 348 591 ; film13 854 403 -> 889 584 ; film04 7 923 121 -> 8 148 765.

## Troisieme integration (depuis ddf99f9)
- p-cadence : fusion sans conflit.
- p-cadence, vitalites : fusions sans conflit. Poses : lignes vitals de layers.go + layers_test.go (2 modifications), chronique v127 (apports 13 et 14), historique des faits (vitalites), FORMAT/GLOSSARY/NOTICE. Fixtures d'entrees inchangees (le blob d'entrees ne porte pas les vitalites) ; graines refigees ; forme du document refigee.
- Plafond des fixtures de contrat : 11 354 146 octets (10,83 Mio) mesures -> plafond 11 Mio (11<<20).
- Poids gzip du document (film05 / film13 / film04) : v1.5.0 329 993 / 840 362 / 7 833 786 ; fin 2e integration 348 591 / 889 584 / 8 148 765 ; maintenant 481 783 / 1 223 306 / 11 231 739. Pic film04 1 fil, 3 passes : 1726 Mio.

## Quatrieme integration (depuis 2a49ff4)
- p-place (2 commits) puis p-visee : fusions automatiques sans conflit (build_pistes.go fusionne seul).
- Fixtures de contrat (4e integration) : 11 461 910 octets, sous le plafond de 11 Mio (11 534 336) : aucun relevement. Goldens d'assemblage : 4 builds, lignes du bilan de tirs (1c62962). Fixtures d'entrees : 3 films changent (visees sans position, 6995a51).

## Cinquieme integration (depuis 46be3cf)
- p-colles : fusion sans conflit (movement_states.go a exactement 500 lignes, options.go 498). Poses : lignes attachments de layers.go + layers_test.go, chronique (17e apport), historique des faits, chronique de la grammaire (6e apport), FORMAT/GLOSSARY/NOTICE. Fixtures d'entrees et goldens d'assemblage inchanges ; goldens de revision (4), types, forme, graines refiges ; fixtures de contrat 11 461 910 octets inchangees.
- Poids final gzip film05 / film13 / film04 : 482 325 / 1 225 879 / 11 368 190 ; pic film04 1 fil, 3 passes : 1624 Mio.

## Sixieme integration (depuis f13aacd)
- bl-score puis bl-divers : fusions automatiques sans conflit.
- bl-score, bl-divers : fusions automatiques sans conflit. Poses : chroniques (document 18e/19e apports, objectives, grammar 7e apport), FORMAT (phrase de la courbe corrigee) / GLOSSARY. Revisions inchangees ; goldens objectives, grammar (perimetre + rev) refiges ; forme du document refigee. revision.Couche.Associes passe de []string a string (+ PaquetsAssocies()) pour garder la structure comparable (gorelease).
- Controle VIP hors banc (3 films, famille VIP + tableau des scores) : documents identiques a 1 et 4 fils (empreintes dans vip_resultat.txt).
