# PLAN — TEST DE CADRE ti=11 (l'experience maitresse des objectifs vivants)

> Ecrit le 2026-08-27 par la session superviseur. Branche `wt/ti11-cadre`, base `f350eb01d`
> (tete du lot C : Live Fire au catalogue, corpus Oddball 4->5). Contrat plan-execution.
> Regles du chantier : filmproc pour tout decodage, protocole COMMITE avant mesure, seuils
> ci-dessous JAMAIS abaisses, temoins obligatoires, arret propre, un seul build Go a la fois.
>
> Documents de conception (dans ce worktree, `.ai/V7.5/replay2d/`) : `PISTE_A_ti11_color.md`
> (i1 couleur/proprietaire) et `PISTE_B_ti11_sous_entites.md` (i16-31 identite de zone). Ils
> portent les grammaires RESOLUES et les protocoles de validation A/B a seuils geles. CE plan
> les orchestre en UNE experience.

## 0. L'ENJEU, en une phrase

Toutes les feuilles de l'archetype `ti=11` (managed-objective) sont RESOLUES et TRIVIALES
(re-verifiees au Ghidra : i0 2xR(7), i1 4xR(8) RGBA, i3 object-ref R(32) +0x40, i5 type R(32)
+0x150, i12/i13 R(32), i14 state R(3) +0x194, i15 parent R(32) +0x198, i16-31 16xR(32) GlobalID
+0x19c+idx*4, borne 16 prouvee par i32 +0x1dc). La SEULE inconnue depuis R7 : le CADRE
d'image-cle (en-tete 108b, ordre des composants) se reproduit-il sur le corps type-2 du film ?
Les feuilles etant triviales, un ECHEC localise DEFINITIVEMENT le mur dans le cadre (ce que
ti=35 ne pouvait pas isoler) ; un SUCCES rend NATIFS, d'un coup : le PORTEUR (i3 = crane Oddball
= le [!] de 5 campagnes, drapeau CTF), le PROPRIETAIRE (i1, repli owner KOTH), l'IDENTITE DE ZONE
(i16-31, Total Control contournant l'ancrage ti=13 casse), et l'ETAT COMPLET (type/progression/
etat) de tous les modes.

## 1. PHASES

> Ordre strict. Items `[x]`/`[~]`/`[!]`. Commits `ti11-cadre(<phase>):`. Jamais `git add -A`.
> Pas de push. thought_log/REGISTRE non touches (textes au CR). Donnees du principal via
> `LEVELUP_REPO_ROOT=c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`. Lectures
> DuckDB par `OpenReadForQuery`. UN SEUL build Go a la fois (aucun autre lot Go ne tourne).

### T0 — Protocole COMMITE avant mesure

- [x] T0.1 Lire les deux docs de conception (PISTE_A, PISTE_B) + `keyframe_fullstate_loop.go`
      (`WalkKeyframeFullState`, `traverseComponentLoop`), `traverse.go` (`consumeByName`,
      patron ti=10 i1), `default_state_ti42.go` (patron d'etat par defaut d'objet),
      `keyframe_biped_fullstate_test.go` (l'instrument de cadre a transposer),
      `keyframe_world.go` (`WalkKeyframeWorld`, l'oracle d'atterrissage), le format de
      `.ai/V7.5/dumps/kf_capture_sample.txt` (400 frontieres de records).
      FAIT : lus + capture.go, components_managed_object.go, components_managed_property.go,
      default_state_arch.go, bitreader.go, ecs_table.tsv (34 composants ti=11 confirmes).
- [x] T0.2 Ecrire et COMMITTER `.ai/V7.5/replay2d/registre_film/TI11_PROTOCOLE.md` : corpus
      admis (films a records ti=11 : CTF `24dbb67d`? non — CTF est un autre corpus ; prendre
      des films de CHAQUE famille qui porte des objectifs ti=11 : au moins un CTF, un
      Strongholds/KOTH, un Oddball du corpus 5), les seuils T1/T2 recopies SANS modification,
      les temoins. Citer le hash au CR.
      FAIT : protocole ecrit ; corpus par famille (Oddball 5 / CTF 64e8adfa,530820e5,53ce4390 /
      Strongholds 696a9d7c,7344d24f,10ed320d / KOTH 01e1f945,606d9844,8076f97f,0a247154), tous
      verifies presents ; seuils recopies verbatim ; 2 temoins figes (record NEW + cadre faux).

### T1 — CABLAGE + TEST DE CADRE (le gate maitre, ecrit AVANT)

- [x] T1.1 Etendre le dispatch d'etat complet a `ti=11` : cabler les `case` R(n) de
      `consumeByName` pour i0/i1/i3/i5/i12/i13/i14/i15/i16-31 (grammaires des docs de
      conception, patron ti=10) + un hook nomme par famille (patron `SetManagedObjectHook`).
      AUCUNE autre modification produit (calque/contrat/schema INTACTS a ce stade : on
      MESURE, on ne publie pas).
      FAIT : `components_managed_objective.go` (9 desers + `SetManagedObjectiveHook`), 9 cases
      dans `consumeByName` (traverse.go:883-909). ecs_table.tsv : 24 lignes ti=11 passees
      `non_porte -> porte` (COMPANION OBLIGATOIRE du cablage, exigee par le garde-rail G1
      `TestG1TableSuitLeCode` — sinon paquet ROUGE ; prescrite PISTE_A §3.3 ; PAS une publication :
      schema/contrat/calque/i18n INTACTS). go build/vet exit 0 ; suite paquet filmdec verte (11s).
- [x] T1.2 Transposer `keyframe_biped_fullstate_test.go` a `ti=11` : atterrissage bit-exact
      des records ti=11 contre `WalkKeyframeWorld` / les frontieres de `kf_capture_sample.txt`.
      Instrument sous garde d'environnement (jamais en CI), un film par processus (filmproc).
      FAIT : `keyframe_objective_fullstate_test.go` (TestTI11Inventory/CadreLanding/
      DiagnosticNoStub/GapProfile), garde `TI11_ROOT`, un film charge a la fois (borne memoire) ;
      lecture keyframe seule (aucun BuildMatch/BuildFromFilm : hors perimetre du garde-rail
      filmproc, comme le harnais biped). Note : `kf_capture_sample.txt` non utilise
      (sparse objectifs, negatif deja publie ti=42) — `WalkKeyframeWorld` fait foi.
- [!] T1.3 GATE T1 : atterrissage bit-exact des records ti=11 = **0,00 %** (0/1227 records, 3
      familles Oddball/CTF/KOTH, toutes variantes de cadre C0-C6, corr off/on), temoin record
      NEW **0,00 %**. GATE NON TENU (0 << 85). Log fige `TI11_T1_cadre.log`. NE PAS publier.
      NUANCE DECISIVE (voir §3 D2) : le plan pre-ecrivait « SI RATE -> le mur est le format
      type-2, echec non imputable aux feuilles (triviales) ». La MESURE REFUTE cette branche :
      la marche COMPLETE (desync=0 en stub) et sous-lit d'un deficit PETIT et souvent CONSTANT
      par film (103b Oddball, 66b KOTH, ~80b median CTF) = exactement les 10 feuilles NON
      RESOLUES stubbees a 0. La premisse du plan « toutes les feuilles ti=11 sont triviales et
      resolues » est FAUSSE (10/34 non resolues). Le mur bit-exact est les FEUILLES NON
      RESOLUES, PAS prouve etre le cadre. Reprise = resoudre+cabler i2/i9/i32 + i4/i6/i7/i8/i10/
      i11/i33 puis re-mesurer (et non le levier type-0 `FUN_142e35a58` seul, ni la voie statborg).

### T2 — VALIDATION SEMANTIQUE (seulement si T1 passe) et PUBLICATION

> T1 NON TENU (0,00 %) : T2 ne s'execute PAS (le plan conditionne T2 au gate T1). Tous les
> items ci-dessous sont `[!]` non traites, cause = gate T1 non tenu.

- [!] T2.1 B1 auto-coherence (gratuit) : les GlobalID lus (i3, i16-31) sont-ils des entites
      valides et stables dans le temps ? >= **90 %**. NON TRAITE : gate T1 non tenu.
- [!] T2.2 B2 owner : confronter i1 (couleur -> camp par clustering RGBA <= 3) au proprietaire
      DEJA publie (`zone_states` 93 %, `hillStates` 88-89 %). Seuil accord global >= **90 %**
      ET >= **85 %** par equipe tenante prise SEPAREMENT. NON TRAITE : gate T1 non tenu.
- [!] T2.3 B3 zones : sur un Bastion (3 zones connues), i16-31 apparie les 3 formes
      (cardinal = 3, appariement >= **90 %**, temoin decale 12 m <= 20 %). Sur Total Control,
      cardinal <= 16, 3 actives par manche, attribution >= **80 %**. NON TRAITE : gate T1 non
      tenu (et blocage aval : §3 D1, les 3 films Strongholds/Bastion du cache rendent 0 record
      ti=11 borne).
- [!] T2.4 PORTEUR (le livrable phare) : i3 object-reference -> l'objet porte ; confronter au
      gate historique Oddball `time_as_skull_carrier_seconds` >= **80 %** par joueur ET porteur
      principal sur >= 3/4 (5) films. Log `TI11_T2_porteur.log`. NON TRAITE : gate T1 non tenu
      (le porteur i3 se LIT mais son slot n'atterrit pas bit-exact ; pas de confrontation valide).
- [!] T2.5 PUBLICATION. NON TRAITE : gate T1 non tenu, aucune publication (schema/contrat/calque/
      i18n INTACTS, conformement au plan).

## 2. GATES DU LOT
Protocole commite avant mesure (un seul commit) ; logs figes ; T1 joue en premier et
conditionne T2 ; si publication, tous les gates web+contrat verts ; `go vet`/`go build` exit 0 ;
arbre propre ; pas de push. CR : verdict T1 chiffre (le cadre passe-t-il ?), puis T2 par gate,
comptes de temoins, commits, textes journal/registre, decouvertes.

## 3. DECOUVERTES

- **D1 — Strongholds : 0 record ti=11 borne.** Les 3 films Strongholds testes (7344d24f,
  696a9d7c, 10ed320d) portent bien l'archetype ti=11 (34 composants) mais `WalkKeyframeWorld`
  n'en reconstruit AUCUN record ti=11 borne (l'oracle ne trouve pas de record ti=11 adjacent
  dans leurs keyframes). Oddball (637 cumules), CTF (501) et KOTH (89) en fournissent
  abondamment. IMPACT : le gate T2.3 (Bastion/Strongholds, 3 zones) serait bloque en aval faute
  de records ti=11 mesurables sur ces films — a instruire (autre film Bastion, ou comprendre
  pourquoi l'oracle ne borne pas les ti=11 Strongholds) AVANT toute reprise T2.3. NON traite
  (hors perimetre : gate T1 non tenu).

- **D2 — La branche « mur = format type-2 » du plan est REFUTEE par la mesure ; la premisse
  « feuilles ti=11 toutes triviales/resolues » est FAUSSE.** L'inventaire montre 10/34 feuilles
  NON resolues (i2 formatted-text, i4 interaction-filter, i6 enabled, i7 priority, i8
  message-type, i9 secondary-formatted-text, i10 is-new, i11 is-only-one, i32 outro-phase
  QUANTIFIE 8b, i33 forced-update). En mode stub (ces 10 a 0 bit), la marche COMPLETE
  (desync=0) mais sous-lit d'un deficit PETIT et souvent CONSTANT par film (103 b sur les 115
  records Oddball ; 66 b sur les 25 KOTH ; ~80 b median CTF, avec dispersion car formatted-text
  est variable). Ce deficit = exactement les feuilles stubbees. Donc : le mur bit-exact n'est
  PAS prouve etre le cadre — la structure cadre+feuilles-cablees consomme un budget de bits a
  ~90 % du reel, le manque etant localise aux feuilles non resolues. RESERVE d'honnetete : un
  deficit petit ne PROUVE pas non plus que le cadre est juste (l'ecart = reel - consomme, et le
  consomme est fixe car les feuilles cablees sont a largeur fixe ; il ne teste pas
  l'alignement interne). Ce qui est ETABLI : (a) 0 % bit-exact, gate non tenu ; (b) l'echec
  n'est PAS attribuable au cadre seul, contra la premisse du plan. CONDITION DE REPRISE
  PRECISE : resoudre au Ghidra i4/i6/i7/i8/i10/i11/i32/i33 (adresses EXE a etablir) + cabler
  i2/i9 (deser `consumeObjectiveFormattedText` deja ecrit, non EXE-verifie) + i32 (8b quant),
  puis RE-mesurer l'atterrissage. Si alors >= 85 % : le cadre reproduit ; sinon seulement le
  levier type-0 `FUN_142e35a58` / la voie statborg redeviennent la piste (l'ordre du plan est
  donc a inverser : feuilles restantes AVANT type-0).

- **D3 — L'empreinte du registre ECS des films du corpus est INCONNUE du code**
  (`empreinte=7307582345129211936` vs connue `7053924395561516366`, WARN au chargement). Le
  film reste decode (34 composants ti=11 lus), mais l'etiquette i0..i33 porte la reserve
  habituelle (dispatch par NOM a l'execution, robuste au decalage d'index). Sans impact sur la
  mesure (dispatch par nom), signale pour la tracabilite.
