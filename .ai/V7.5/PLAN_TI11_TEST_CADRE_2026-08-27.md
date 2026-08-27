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
- [x] T1.3 GATE T1 : **TENU apres cablage des 10 feuilles** (RE-MESURE 2026-08-27, commit
      `4c21a2560`). Etat initial (avant cablage, commit `f91a3a521`) = **0,00 %** ; APRES cablage
      des 10 feuilles resolues (i2/i9 formatted-text, i4 interaction-filter, i6/i7/i8/i10/i11/i32/
      i33), sous le cadre gagnant **C5** (en-tete 108 + mots de taille + etat par defaut +
      **LevelShift**), corr=false : **Oddball 24dbb67d 100,00 %** (115/115), **KOTH 01e1f945
      100,00 %** (25/25), CTF 64e8adfa 83,58 % (168/201), **CUMUL 90,32 %** (308/341), temoin
      record NEW **0,00 %**, tous les cadres faux (C0-C4, C6) a 0,00 %. GATE (>= 85 % sur >= 2
      familles distinctes, temoin <= 20 %) TENU (Oddball + KOTH). Inventaire : 34/34 composants
      ti=11 portes. Log fige `TI11_REMESURE_cadre.log`. **La premisse du T1 (le mur = les feuilles
      NON resolues, contra la branche pre-ecrite du plan) est CONFIRMEE** : le cadre EST juste,
      resoudre+cabler les 10 feuilles a leve 0 -> 90,32 %.

### T2 — VALIDATION SEMANTIQUE (T1 passe) — LE VIVANT N'EST PAS DANS LE KEYFRAME

> T1 TENU : T2 s'execute. VERDICT GLOBAL : le cadre reproduit BIT-EXACT et l'alignement interne
> est CONFIRME (i5 type discrimine correctement : Oddball/CTF-drapeau = 2947879880 « portable » ;
> KOTH/CTF-zone = 1496018944 « zone »), MAIS tous les champs VIVANTS du keyframe ti=11 sont des
> SENTINELLES (803 records, 7 films) : i3 object-reference (LE PORTEUR) = 0xFFFFFFFF null, i15
> parent = null, i16-31 sous-objectifs = null, i12/i13 progression = 0, i1 couleur = gris
> constant. Le keyframe stocke l'ETAT PAR DEFAUT de l'objectif ; le vivant vit dans le flux
> DELTA. AUCUNE PUBLICATION. Log `TI11_T2_porteur.log`.

- [~] T2.1 B1 auto-coherence : techniquement TENU (stabilite i3 **100,0 %**, validite 100,0 %,
      temoin 0,0 %) mais **HOLLOW** — |obs|=1 (un seul i3 = 0xFFFFFFFF null sur 803 records) : un
      null constant est stable et introuvable au hasard PAR CONSTRUCTION. Le gate ne detecte pas
      la degenerescence. Verdict honnete : i3 est une sentinelle null dans le keyframe.
- [!] T2.2 B2 owner : NON TENU. i1 couleur = gris constant (128,128,128,255) dans le keyframe —
      ne porte pas le camp. La couleur/owner vit dans le delta, pas la baseline.
- [!] T2.3 B3 zones : NON TENU. i16-31 = 0xFFFFFFFF null dans le keyframe — ne portent pas
      l'identite de zone. (Blocage aval D1 inchange par ailleurs.)
- [!] T2.4 PORTEUR (le livrable phare) : NON TENU. i3 se LIT bit-exact (atterrissage 100 % sur
      les 5 Oddball) mais VAUT 0xFFFFFFFF (null) — il ne designe AUCUN objet porte dans le
      keyframe. Confrontation a `time_as_skull_carrier_seconds` impossible (rien a confronter). Le
      porteur (le [!] de 5 campagnes) reste OUVERT. Log `TI11_T2_porteur.log`.
- [!] T2.5 PUBLICATION. NON TRAITE : les valeurs vivantes ne sont pas dans le keyframe. Aucune
      publication (schema/contrat/calque/i18n INTACTS). **CONDITION DE REPRISE : appliquer la
      grammaire ti=11 34-feuilles MAINTENANT VALIDEE (cadre C5 + LevelShift) aux records DELTA
      (la sonde R4 balayait deja le delta), pas a la baseline keyframe.** Le lot a resolu le
      CADRE ; le vivant demande le chemin delta.

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
