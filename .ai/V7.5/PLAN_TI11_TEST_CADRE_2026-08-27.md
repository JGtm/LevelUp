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

- [ ] T0.1 Lire les deux docs de conception (PISTE_A, PISTE_B) + `keyframe_fullstate_loop.go`
      (`WalkKeyframeFullState`, `traverseComponentLoop`), `traverse.go` (`consumeByName`,
      patron ti=10 i1), `default_state_ti42.go` (patron d'etat par defaut d'objet),
      `keyframe_biped_fullstate_test.go` (l'instrument de cadre a transposer),
      `keyframe_world.go` (`WalkKeyframeWorld`, l'oracle d'atterrissage), le format de
      `.ai/V7.5/dumps/kf_capture_sample.txt` (400 frontieres de records).
- [ ] T0.2 Ecrire et COMMITTER `.ai/V7.5/replay2d/registre_film/TI11_PROTOCOLE.md` : corpus
      admis (films a records ti=11 : CTF `24dbb67d`? non — CTF est un autre corpus ; prendre
      des films de CHAQUE famille qui porte des objectifs ti=11 : au moins un CTF, un
      Strongholds/KOTH, un Oddball du corpus 5), les seuils T1/T2 recopies SANS modification,
      les temoins. Citer le hash au CR.

### T1 — CABLAGE + TEST DE CADRE (le gate maitre, ecrit AVANT)

- [ ] T1.1 Etendre le dispatch d'etat complet a `ti=11` : cabler les `case` R(n) de
      `consumeByName` pour i0/i1/i3/i5/i12/i13/i14/i15/i16-31 (grammaires des docs de
      conception, patron ti=10) + un hook nomme par famille (patron `SetManagedObjectHook`).
      AUCUNE autre modification produit (calque/contrat/schema INTACTS a ce stade : on
      MESURE, on ne publie pas).
- [ ] T1.2 Transposer `keyframe_biped_fullstate_test.go` a `ti=11` : atterrissage bit-exact
      des records ti=11 contre `WalkKeyframeWorld` / les frontieres de `kf_capture_sample.txt`.
      Instrument sous garde d'environnement (jamais en CI), un film par processus (filmproc).
- [ ] T1.3 GATE T1 (ECRIT, ne bouge pas) : chainage/atterrissage bit-exact des records ti=11
      >= **85 %** sur >= **2** films de familles distinctes, TEMOIN « record NEW/faux » <= **20 %**.
      Log fige `TI11_T1_cadre.log`.
      - SI RATE : le cadre ne reproduit PAS. STOP mesure. Le mur est le format type-2 (localise
        DEFINITIVEMENT, cf. Lot G / REGISTRE:210). Rediger le verdict [!] chiffre, consigner la
        condition de reprise (levier type-0 `FUN_142e35a58`), NE PAS publier. Fin du lot cote
        natif ; la resolution bascule sur la voie statborg (lot separe).

### T2 — VALIDATION SEMANTIQUE (seulement si T1 passe) et PUBLICATION

- [ ] T2.1 B1 auto-coherence (gratuit) : les GlobalID lus (i3, i16-31) sont-ils des entites
      valides et stables dans le temps ? >= **90 %**.
- [ ] T2.2 B2 owner : confronter i1 (couleur -> camp par clustering RGBA <= 3) au proprietaire
      DEJA publie (`zone_states` 93 %, `hillStates` 88-89 %). Seuil accord global >= **90 %**
      ET >= **85 %** par equipe tenante prise SEPAREMENT (le juge du risque POV enregistreur).
- [ ] T2.3 B3 zones : sur un Bastion (3 zones connues), i16-31 apparie les 3 formes
      (cardinal = 3, appariement >= **90 %**, temoin decale 12 m <= 20 %). Sur Total Control,
      cardinal <= 16, 3 actives par manche, attribution >= **80 %**.
- [ ] T2.4 PORTEUR (le livrable phare) : i3 object-reference -> l'objet porte ; confronter au
      gate historique Oddball `time_as_skull_carrier_seconds` >= **80 %** par joueur ET porteur
      principal sur >= 3/4 (5) films. Log `TI11_T2_porteur.log`.
- [ ] T2.5 SI les gates T2 tiennent : PUBLIER l'etat d'objectif natif — porteur dans
      `objectiveObjects` (cle porteur, contrat +1 ; remplacer proprement les 2 refus testes),
      owner de zone/colline, identite de sous-zone. Rendu au patron `flagCarriesLayer`/
      `zoneStatesLayer`. Triplet schema Go/contrat/web, chronique, i18n FR+EN, re-cuisson
      TEMOINS avec verification de CONTENU. Gates : go test packages touches + contracttest,
      tsc -b (cache purge), vitest match-replay, lint web, parite schema.

## 2. GATES DU LOT
Protocole commite avant mesure (un seul commit) ; logs figes ; T1 joue en premier et
conditionne T2 ; si publication, tous les gates web+contrat verts ; `go vet`/`go build` exit 0 ;
arbre propre ; pas de push. CR : verdict T1 chiffre (le cadre passe-t-il ?), puis T2 par gate,
comptes de temoins, commits, textes journal/registre, decouvertes.

## 3. DECOUVERTES — (vide a l'ouverture)
