# Plan R3 — L'identite des entites `ti=37` : nommer le mur de protection et le capteur de menaces

> Ecrit le 2026-08-17. Lot R3 du `PLAN_RETOURS_PLANCHE_2026-08-16.md` (§R3, priorite
> utilisateur explicite au bilan F1 : « j'aimerais les murs de protection et capteurs de
> menaces [...] sujet a pousser davantage niveau investigation »).
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-ti37`, branche `wt/ti37-identite`,
> base `feat/v75` = `3058afbba`. Lot jumeau R4 (`ti=11`, objectifs vivants) tourne EN
> PARALLELE sur `LevelUp-wt-ti11` : voir « Contrat d'interface avec R4 » ci-dessous.
> Execution sous le contrat du skill `plan-execution` (ordre strict, un item = un statut,
> zero fix hors perimetre, zero report d'une etape executable).

## 1. Objectif

Attribuer un TYPE a chaque entite d'archetype `ti=37` (« equipment » / « item ») du film,
de facon a pouvoir dire, pour un objet pose sur la carte : **c'est un mur de protection**,
**c'est un capteur de menaces**, ou **ce n'est ni l'un ni l'autre**. La POSITION et la
FENETRE DE VIE sont deja acquises (phase 0 du `PLAN_EQUIPEMENT_TI37.md` : 97,2 % de
628 368 echantillons dans l'emprise, 12 films sur 12) ; il ne manque que le NOM.

Le rendu (capteur = zone radar pulsee ; mur = segment pose) est HORS PERIMETRE — decision
utilisateur « on reflechira a l'UI plus tard ». Ce lot publie la DONNEE, il ne dessine rien.

### 1.1 Critere de succes — mesurable, et symetrique

Le lot est un SUCCES si les quatre controles suivants passent, chiffres publies avec leurs
denominateurs, sur le corpus de 5 films de §4 :

| C | controle | seuil |
|---|---|---|
| C1 | un champ lu au record de REFERENCE (image-cle) ou de CREATION de l'entite partitionne les entites `ti=37` | cardinalite des valeurs distinctes <= 5 % du nombre de vies d'objet, sur chacun des 3 films principaux |
| C2 | la valeur est STABLE sur la duree d'une vie d'objet (slot, gen) | >= 99 % des vies a valeur unique |
| C3 | la valeur est GLOBALE, pas un handle : les memes valeurs reapparaissent d'un film a l'autre | >= 3 valeurs communes aux 3 films principaux, et recouvrement >= 60 % des vies |
| C4 | au moins DEUX classes sont NOMMEES par un temoin INDEPENDANT du champ lui-meme (§5.3) | 2 classes, chacune avec son temoin chiffre et son temoin negatif |

Le lot est un **NEGATIF MESURE — livrable a part entiere** si C1, C2 ou C3 echoue : on
publie alors les denominateurs, la liste exhaustive des champs interroges et refutes, et la
ligne de registre avec sa condition de reprise. **Aucun nom n'est pose sans C4.** Un negatif
ecrit vaut mieux qu'un nom devine : c'est la regle qui a deja tranche ce chantier trois fois
(lot du 14/08, `i54`, R(24) d'`i57`).

### 1.2 Ce que le lot ne cherche PAS (perimetre ferme)

- La DATATION de l'activation (`activated`, `charges-remaining`) : refutee / en reserve,
  registre du 2026-08-15, hors de ce lot.
- Le lien objet -> joueur poseur (`equipment-creator` R(5) : ni slot ni index joueur,
  mesure 0.4 du `PLAN_EQUIPEMENT_TI37.md`).
- Le translocateur, le repulseur, le grappin : ils ne posent pas d'objet durable.
- Tout rendu, tout son, toute string UI.

## 2. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17

Chemins relatifs a `apps/go-api/internal/analysis/` sauf mention contraire.

| fait | ou (fichier:ligne, relu ce jour) |
|---|---|
| `EquipmentTypeIndex = 37`, archetype des objets du monde | `filmdec/projectiles.go:85` |
| Les 4 champs delta de `ti=37` sont publies par hook, aucun ne porte d'identite | `filmdec/equipment_state.go:68-108` (hook), verdict 0.6 du `PLAN_EQUIPEMENT_TI37.md` |
| Positions `ti=37` : `ScanFilmWorldObjects(dir, wr, 37)` | `filmdec/projectiles.go:103` |
| En-tete d'un record DELTA d'objet du monde : prefixe(1)+slot(13)+gen(2)+porte(2)+count(3), puis index 6 bits croissants | `filmdec/projectiles.go:254-262`, `matchWorldObjectRecord` `:283` |
| **Le default-state de `ti=37` EST porte bit-exact** — et il lit QUATRE champs de 32 bits, tous JETES | `filmdec/default_state_arch.go:154-167` |
| dont le bloc « multiplayer properties » : `R(9)`, **`R(32)` (FUN_14080d6f0)**, **`R(32)` « variant-name » (porte)**, ..., queue G3 **`R(32)`** | `filmdec/default_state.go:329-359` |
| dont, en propre a `ti=37`, **`R(32)` « ability-enabled-id »** derriere une porte | `filmdec/default_state_arch.go:166` (`consumeGateR(br, 32)`) |
| Le default-state par archetype est joue au record NEW | `filmdec/traverse.go:1027` (`defaultStateDeserByTI`, garde `useArchDefaultStateDeser`) |
| Table keyframe : en-tete `[id:32][field:26][ti:6]` = 64 bits, puis le default-state de l'archetype, puis gate+masque+composants ; walker DURCI valide 249/250 entites | `filmdec/keyframe_world.go:19-27`, `WalkKeyframeWorld` `:153` |
| Le precedent METHODOLOGIQUE : au keyframe, l'identite d'un objet du monde se lit par balayage d'un id 32 bits attribue au record qui le CONTIENT | `filmdec/keyframe_ground_weapons.go` + `filmdec/keyframe_loadout.go:118` (`familiesByRecord`) |
| Le second precedent : l'id 32 bits d'un projectile est le **tag global du groupe `proj` DECALE D'UN BIT A GAUCHE**, trouve au record de CREATION | `filmdec/grenade_events.go:30-49` |
| L'index de tags des `.module` du jeu est DISPONIBLE en Go (GlobalID + fourCC de groupe) | `internal/himodule/module.go:67-83`, `internal/himap/moduleindex.go:38`, racine par `internal/himap/deploy_root.go:39` |
| Palette de capacites par film, rang complet publie (`i48`) | `filmdec/ability_rank.go`, `replay/abilities.go` |
| Rang 19 = mur portatif, rang 22 = capteur de menaces (famille B) — **une seule observation Theater chacun**, report ouvert | `.ai/V7.5/REGISTRE_REPORTS.md` ligne « Rangs 19 et 22 de la famille B » |
| Le report que ce lot attaque, mot pour mot : « trouver dans le record de CREATION de l'entite la reference de definition de l'objet — meme voie que la famille high-32 des armes au sol » | `.ai/V7.5/REGISTRE_REPORTS.md` ligne « Identite de l'objet ti=37 » (2026-08-15) |
| `SchemaVersion = 8` (grappin) | `replay/document.go:75` |

### 2.1 Ce qui est REFUTE — ne pas rejouer

- Les 4 champs delta (`deployed` / `activated` / `creator` / `energy`) ne portent AUCUNE
  identite (13 187 records annonceurs sur 1 097 619, verdict 0.6 du 15/08).
- `equipment-creator` R(5) n'est ni un slot de biped (0 valeur sur 1 328 dans
  `bipedSlotBand`) ni un index de joueur (28 sur un film a 8 joueurs).
- Le R(24) de la branche `v==1` d'`i57` n'est PAS un handle vers l'entite `ti=37`
  (valeurs quasi toutes uniques ; vies vivantes a +/-2 s : 1-3 %).
- Les « naissances » `ti=37` deja mesurees sont des PREMIERS RECORDS DELTA d'une vie
  (`filmdec/i57_handle_test.go:326`, `etat_actif_shared_test.go:217`), PAS des records NEW.
  Le record de CREATION de `ti=37` n'a jamais ete localise — c'est le trou de ce lot.

## 3. Hypotheses, ordonnees par COUT CROISSANT

**H1 — le default-state au KEYFRAME (cout : faible).** `WalkKeyframeWorld` rend deja, pour
chaque image-cle, le bit de debut et le `ti` de chaque record. Pour un record `ti=37`, le
corps commence a `Bit+64` et le premier bloc est `consumeDefaultStateTI37`, deja porte
bit-exact. Il suffit d'un HOOK sur les quatre `R(32)` pour les faire sortir. C'est la voie
la moins chere et la seule qui s'appuie integralement sur du code deja valide.
*Risque connu et a chiffrer : une image-cle toutes les ~20 s — la couverture des objets
ephemeres (un mur vit quelques secondes) sera PARTIELLE. Le chiffrer est un resultat.*

**H2 — le record de CREATION (NEW) dans les paquets delta (cout : moyen).** Meme grammaire
(`ti` sur 6 bits, puis `consumeDefaultStateTI37`, puis gate+masque). Precedent exact :
`grenade_events.go`, ou la creation d'un projectile se reconnait par une constante de 24 bits
= `[5 bits bas de ti][19 bits d'amorce]`. Couverture potentiellement TOTALE (toute entite
nait une fois), au prix d'un detecteur a construire et a controler contre un temoin fantome.

**H3 — nommer les valeurs par le CATALOGUE de tags du jeu (cout : moyen, DEPEND de H1/H2).**
Les `.module` locaux donnent GlobalID + groupe fourCC. Si une valeur lue est un tag global
(eventuellement decale d'un bit, comme les grenades), le controle est celui de
`grenade_events.go` : le taux d'appartenance au catalogue doit ECRASER le hasard. Ceci NOMME
la classe ; le nom lui-meme reste une donnee de titre (TOML), jamais un litteral Go.

**H4 — la SIGNATURE COMPORTEMENTALE (cout : faible, INDEPENDANTE des precedentes).** Un mur
de protection et un capteur de menaces ne se comportent pas pareil : multiplicite a la
naissance (le mur est un panneau segmente), duree de vie, immobilite, proximite d'un porteur
de rang 19 vs 22. C'est le TEMOIN INDEPENDANT exige par C4 — il ne sert PAS a decider la
partition, il sert a la NOMMER sans circularite.

## 4. Corpus — ferme, et pourquoi ces films

3 films PRINCIPAUX (famille B, riches en rangs 19 et 22, chiffres du registre) + 2 TEMOINS.
Racine (lecture seule) : `C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/`.

| film | role | justification |
|---|---|---|
| `000d5950` | principal + verite terrain Theater | 49 chunks ; 34/82 lectures `i48` en rangs 19/22 ; golden du rejeu |
| `00502e52` | principal | 30 chunks ; 40/65 lectures en rangs 19/22 |
| `07aa428d` | principal | 28 chunks ; 19/42 lectures en rangs 19/22 |
| `0014603f` | TEMOIN NEGATIF | 22 chunks ; `i48` jamais au masque — aucune identite de capacite |
| `00ba2e1c` | TEMOIN a dotation unique | 29 chunks ; les 8 joueurs y portent le MEME equipement (index 7) : une partition correcte doit y etre DEGENEREE |

Presence des 5 verifiee sur disque le 2026-08-17.

## 4bis. Blockers connus, et leur contournement

| blocker | nature | contournement prevu |
|---|---|---|
| Les films vivent dans le depot PRINCIPAL (`LevelUp/data/cache/film_chunks/`), absent du worktree | chemin | lecture seule par chemin absolu ; aucune ecriture, aucun lien |
| H3 exige l'installation locale de Halo Infinite (`himap.DeployRoot`) | ressource externe | si absente : item 3.3 statue `[!]` avec la raison — report VALIDE au sens du contrat (§7 du skill) |
| Un seul decodage `filmdec` par process (hooks globaux) | technique | verrou pris pour tout l'instrument, hook restaure en `defer` (patron `ScanFilmEquipmentState`) |
| Cache Go partage corrompu par deux `go` concurrents (lot jumeau R4 en parallele) | technique | `GOCACHE` isole au worktree + une seule commande `go` a la fois |
| Un paquet exigeant CGO (DuckDB) sur le chemin de build | technique | `CGO_ENABLED=0` sur tous les paquets du film ; si un paquet l'exige, le DIRE au rapport et ne pas insister |

## 5. Phases

Une phase n'ouvre pas tant que la precedente n'est pas CLOSE (gate passe + tous les items
statues + plan mis a jour + commit + push + point d'etape).

**Effort et independance de livraison** — les phases 1, 2 et 3 sont des phases de MESURE :
chacune est livrable seule (un commit `mesure(...)` qui laisse le produit inchange). La
phase 4 est la seule qui touche l'artefact, et elle DEPEND du verdict de la phase 3.

| phase | effort | livrable seule ? |
|---|---|---|
| 1 instrumenter | moyen (un hook + un balayage + un controle d'alignement) | oui |
| 2 mesurer la partition | moyen (rejeu sur 5 films ; lourd si H2 doit s'ouvrir) | oui |
| 3 nommer | moyen | oui |
| 4 publier | moyen | non (depend de 3) |
| 5 clore | rapide | non |

### Phase 1 — INSTRUMENTER : faire sortir les quatre `R(32)` du default-state de `ti=37` — CLOSE le 2026-08-17

- [x] 1.1 Hook de mesure sur le default-state, dans un fichier NEUF
      `filmdec/equipment_identity.go` : un `equipmentIdentityHook func(field EquipIDField,
      value uint64, present bool)` sur le modele EXACT de `equipmentStateHook`
      (`equipment_state.go:68-75`). Les points de publication sont les quatre `R(32)` deja
      lus : `consumeMultiplayerPropertiesBlock` (`default_state.go:331` mpp-id,
      `:333` variant-name, `:355` queue G3) et `consumeDefaultStateTI37`
      (`default_state_arch.go:166` ability-enabled-id). **AUCUNE largeur ne change** — le
      lot fait publier ce que le deser lisait deja (regle du chantier).
- [x] 1.2 Balayage `ScanFilmEquipmentIdentity(dir, layout)` — AMENDE en cours d'execution :
      la marche n'est PAS une copie de `consumeDefaultStateTI37` posee a la main, c'est le
      lecteur de record NEW de PRODUCTION (`TraverseEntity`) rejoue sur les 6 bits de `ti`
      du record d'image-cle (decalage 58, `keyframeRecordTIBit`). Motif : une copie ne
      controle rien, et `TraverseEntity` porte les trois bascules de grammaire que la
      phase 1.3 devait justement trancher. Denominateurs publies : images-cles, records
      `ti=37`, records BORNES par un voisin, marches bit-exactes / ratees / desynchronisees.
- [x] 1.3 **Controle d'alignement — DURCI en ORACLE, et il a TRANCHE (par la negative).**
      L'ecart mesure n'est plus celui du seul default-state mais celui de la marche
      COMPLETE du record (etat par defaut + porte + masque + tous les composants) contre le
      premier bit du record SUIVANT, connu independamment par `WalkKeyframeWorld`. La
      MATRICE des 8 combinaisons de grammaire (`EquipmentIdentityLayouts` : corruption-check
      film x tail terminal x routage du deser d'etat par defaut) est probee sur le meme
      denominateur. **Resultat sur 3 films : 2 marches bit-exactes sur 1 226 records bornes
      (0,16 %), aucune combinaison au-dessus de 0,2 %.** Verdict : la grammaire du record
      d'image-cle `ti=37` N'EST PAS celle du record NEW du chemin delta. Les valeurs lues
      sont donc declarees NON FIABLES et NE SONT PAS publiees — c'est le garde-fou qui joue
      son role.
- [x] 1.4 Instrument versionne `filmdec/equipment_identity_test.go`, garde d'environnement
      `EQUIP_ID_FILM` (saute en CI, patron de `equipment_state_test.go:35-50`). Il publie
      les denominateurs, la matrice, la distribution des ecarts, et n'affiche les valeurs
      que sur les records BIT-EXACTS — donc, sur ce corpus, aucune.

**Gate 1** (a executer, sorties collees au journal de phase) :

```
export GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-ti37/.gocache
cd apps/go-api && CGO_ENABLED=0 go build ./internal/analysis/...
cd apps/go-api && CGO_ENABLED=0 go vet ./internal/analysis/filmdec/
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
cd apps/go-api && CGO_ENABLED=0 EQUIP_ID_FILM=C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/filmdec/ -run '^TestEquipmentIdentity$' -timeout 30m -v
```

Critere : build/vet/test verts ; l'instrument rend un nombre NON NUL de records `ti=37`
lus en image-cle, et la distribution d'ecart de 1.3 est publiee. Si zero record `ti=37` en
image-cle sur les 3 films principaux, H1 est CLOSE NEGATIVE et la phase 2 bascule sur H2.

**GATE 1 : PASSE le 2026-08-17.** Sorties exactes :

```
CGO_ENABLED=0 go build ./internal/analysis/...                          exit 0
CGO_ENABLED=0 go vet  ./internal/analysis/filmdec/                      exit 0
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
    ok levelup/go-api/internal/analysis/filmdec  1.484s
    ok levelup/go-api/internal/analysis/replay  31.355s               exit 0
gofmt -l internal/analysis/filmdec/                                     (vide)
```

L'instrument, sur les TROIS films principaux :

    film       chunks  images-cles  records keyframe  records ti=37  bornes
    000d5950     27        26            7 825            431         415
    00502e52     29        28            8 048            428         408
    07aa428d     27        26           11 191            421         403
    TOTAL                                27 064          1 280       1 226

    matrice des 8 grammaires, marches BIT-EXACTES / bornes :
    000d5950   0/415 pour les 8 combinaisons
    00502e52   1/408 (combinaison [3]), 0 pour les 7 autres
    07aa428d   1/403 (combinaison [1]), 0 pour les 7 autres
    -> 2 / 1 226 = 0,16 %. Aucune grammaire ne tient.

**CE QUE LA PHASE 1 ETABLIT, et ce n'est pas ce qu'elle cherchait.**

1. Les quatre `R(32)` SORTENT : le hook fonctionne, la mecanique de publication est en
   place, et elle est celle du depot (le deser publie, pas un second lecteur).
2. **L'hypothese H1 est REFUTEE dans sa forme naive** : le corps d'un record d'image-cle
   `ti=37` ne se lit PAS comme un record NEW du chemin delta. Zero desynchronisation sur
   1 226 marches (aucun composant non porte) et pourtant zero atterrissage juste : la
   marche s'arrete TOUJOURS TROP TOT, de 557 a 1 104 bits.
3. **L'ecart residuel est STRUCTURE, et c'est la decouverte du jour** : il prend un petit
   jeu de valeurs RECURRENTES, LES MEMES d'un film a l'autre (557, 707, 773, 881, 950,
   1 104 dominent les trois films). Un residu aleatoire ne ferait pas cela. Deux lectures
   possibles, a departager en phase 2 : (a) l'image-cle porte un ETAT COMPLET (tous les
   composants, sans masque epars) — ce qui expliquerait des records de ~750 a 1 150 bits,
   coherent avec les ~2 800 bits mesures pour un record biped d'image-cle
   (`keyframe_loadout.go`) ; (b) la LONGUEUR du record est elle-meme fonction du TYPE de
   l'objet — auquel cas elle est un signal de partition a part entiere, gratuit.
4. Consequence de methode : les valeurs de 2.1 ne se jugent PAS sur ce canal tant qu'un
   ancrage independant ne les valide pas. La phase 2 s'ouvre donc sur l'ancrage, pas sur
   l'histogramme.

### Phase 2 — MESURER la partition (C1, C2, C3) — CLOSE le 2026-08-17

> AMENDEE AVANT EXECUTION par la conclusion 4 de la phase 1 : les valeurs de la voie H1 ne
> sont pas des mesures (aucun record bit-exact), donc la phase 2 s'ouvre sur l'ANCRAGE (H3)
> et non sur l'histogramme des champs d'H1. C'est le report prevu par l'item 2.5, joue
> d'emblee — et il rend H2 SANS OBJET (perimetre ferme, H2 n'est pas ouverte).

- [~] 2.1 SANS OBJET sur les champs d'H1 (aucun n'est une mesure, verdict phase 1). REMPLACE
      par l'ancrage : le catalogue de tags de l'installation locale
      (`any/globals/*.module`, 66 703 GlobalID, 297 groupes) est confronte aux records
      d'image-cle par le balayage DEJA EPROUVE du paquet (`familiesByRecord`, celui des
      armes portees et des armes au sol) — aucun second balayeur ecrit a cote.
- [x] 2.2 C2 — stabilite par vie `(slot, gen)` : **99,6 a 100 % sur les 5 films** (tableau
      ci-dessous). Seuil C2 (>= 99 %) FRANCHI partout.
- [x] 2.3 C3 — globalite : **les 12 memes classes sur les 3 films principaux**, couvrant la
      totalite des vies identifiees sauf une valeur singleton. Le temoin negatif exige par
      l'item est mieux que fourni : le film `0014603f` (aucune identite `i48`) rend **UNE
      SEULE classe sur 100 % de ses 152 records** — un handle local ne ferait jamais cela.
- [x] 2.4 C1, C2, C3 statues en toutes lettres ci-dessous. Les quatre champs d'H1 sont
      ECRITS comme refutes (ils ne sont pas a leur place, ils ne sont donc pas des champs).
- [~] 2.5 H2 n'est PAS ouverte : l'ancrage passe C1-C3, le perimetre se ferme ici.

**GATE 2 : PASSE le 2026-08-17.**

**(a) L'ANCRAGE, et c'est la mesure du lot** (film `000d5950`, ~30 M bits balayes) — nombre
d'occurrences d'un GlobalID de tag du groupe, PAR ARCHETYPE du record porteur :

    groupe (tags)     ti=35 bipede   ti=37 equip   ti=38   ti=41 proj   ti=42 arme sol
    eqip  (116)             0            428          0         0             0
    weap  (192)           310              2          0         0           267
    proj  (283)            23              0          0        20             0
    sofd  (207)           485              0          0         0           267

    Hasard attendu pour `eqip` : 116 tags sur 2^32, ~30 M bits -> ~0,8 occurrence sur TOUTE
    la charge utile. Mesure : 428, et TOUTES dans `ti=37`. Environ 500x le hasard, et une
    exclusivite parfaite.

    LES TROIS AUTRES LIGNES SONT LE CONTROLE, et il n'est impose par rien dans la methode :
    `weap` tombe sur le bipede (armes portees) et sur l'arme au sol — c'est exactement le
    resultat de `keyframe_loadout.go`, retrouve par une chaine independante ; `proj` tombe
    sur le projectile ; `sofd` (palette de capacites) tombe sur le bipede. Chaque groupe de
    tags atterrit sur l'archetype qui le porte semantiquement. Un balayage qui compte des
    motifs de 32 bits ne peut pas fabriquer cela.

**(b) LA PARTITION** — un tag `eqip` par entite, sur les 5 films du corpus :

    film       records ti=37   porteurs d'un tag   vies identifiees   classes   C2 vies uniques
    000d5950        431          427 (99,1 %)            275             13        274 (99,6 %)
    00502e52        428          421 (98,4 %)            249             12        249 (100 %)
    07aa428d        421          418 (99,3 %)            265             13        264 (99,6 %)
    00ba2e1c       1340         1320 (98,5 %)            646             19        645 (99,8 %)
    0014603f        152          152 (100 %)              83              1         83 (100 %)
    TOTAL          2772         2738 (98,8 %)           1318              -              -

**(c) VERDICTS.**

- **C1 PASSE.** Cardinalite / vies : 13/275 (4,7 %), 12/249 (4,8 %), 13/265 (4,9 %) sur les
  trois films principaux — sous le seuil de 5 %. Sur le BTB `00ba2e1c` : 19/646 (2,9 %).
- **C2 PASSE.** 99,6 % · 100 % · 99,6 % · 99,8 % · 100 %. Une entite `ti=37` porte UNE
  classe et la garde toute sa vie.
- **C3 PASSE.** Les 12 classes des films d'arene sont LES MEMES d'un film a l'autre — un
  handle local ne se repete pas d'un match a l'autre. Le BTB en ajoute 7 (une carte BTB
  porte plus de types d'objets), sans en perdre aucune.
- **Les 4 champs de la voie H1 sont REFUTES** : `mpp-r32`, `mpp-variant-name`, `mpp-tail-g3`
  et `ability-enabled-id` ne sont pas lus a leur place (0 record bit-exact sur 1 226), et le
  decalage de 8 bits entre les deux populations de portes de version fabriquait a lui seul
  la moitie de la « cardinalite 14 » de `variant-name`. Ils ne sont pas publies.

**(d) CE QUE LE TEMOIN NEGATIF DONNE EN PRIME, et qui oriente la phase 3.** Sur `0014603f`,
film SANS aucune identite de capacite, la classe `0xa53cd143` couvre **100 % des entites**.
Elle domine aussi les quatre autres films (191, 212, 197, 568 occurrences). Une classe
presente partout, majoritaire partout, et SEULE dans un film sans equipement de joueur,
n'est pas un equipement de joueur : c'est l'objet de monde ordinaire que l'archetype
`item-*` melange aux equipements (verdict 0.6 du 15/08, qui l'avait suppose sans pouvoir le
montrer). La phase 3 nomme donc PAR DIFFERENCE, pas par frequence.

### Phase 3 — NOMMER sans circularite (C4)

Ouverte SEULEMENT si la phase 2 rend une partition. Sinon : `[~]` avec renvoi au negatif.

- [ ] 3.1 Temoin A — **croisement avec la palette `i48`** : pour chaque classe, la
      distribution des rangs `i48` des vies de biped du meme film. Sur `00ba2e1c` (dotation
      unique) la partition des objets deployables doit etre DEGENEREE ; sur `0014603f`
      (aucune identite `i48`) le croisement doit etre VIDE, pas invente.
- [ ] 3.2 Temoin B — **signature comportementale** (H4), independante du champ : par classe,
      duree de vie mediane, multiplicite a la naissance (entites nees a moins de 0,5 s ET
      moins de 3 u l'une de l'autre — un mur segmente en porte plusieurs, un capteur une
      seule), immobilite (etendue des positions sur la vie). Temoin negatif : les memes
      statistiques sur les classes NON deployables.
- [ ] 3.3 Temoin C — **catalogue de tags** (H3), si et seulement si les valeurs ont l'allure
      d'un tag global : taux d'appartenance au catalogue `.module` (avec et sans decalage
      d'un bit), contre le taux attendu par hasard. Si les modules du jeu ne sont pas
      accessibles sur le poste (`himap.DeployRoot`), statuer `[!]` avec la raison — c'est
      une ressource externe, report VALIDE au sens du contrat.
- [ ] 3.4 Verdict C4 : quelles classes sont nommees, par quels temoins, avec quels chiffres.
      Les classes non nommees gardent leur NUMERO (regle du depot : un nom approchant se lit
      comme une certitude).

**Gate 3** : deux classes nommees avec deux temoins independants chiffres et leur temoin
negatif, OU verdict negatif ecrit. Aucun nom sans provenance.

### Phase 4 — PUBLIER la donnee au document de rejeu

Ouverte SEULEMENT si la phase 3 nomme au moins une classe. Sinon `[~]` + registre.

- [ ] 4.1 Lecteur de PRODUCTION dans `filmdec` (le meme fichier `equipment_identity.go`),
      sur le patron de `camo_state.go` / `grapple_state.go` : hook du deser de production,
      lectures localisees slot/gen/chunk/packet/`TimestampUS`, structure de statistiques
      avec denominateurs.
- [ ] 4.2 `replay.Options.EquipmentIdentities` (patron `CamoStates` / `GrappleReads`,
      `replay/build.go:58-65`), peuplee par `BuildFromFilm`, **absence NON FATALE et
      loggee** (`slog.Warn`, convention du fichier).
- [ ] 4.3 Assemblage `replay/equipment_objects.go` (fichier NEUF) : par vie d'objet, le
      TYPE (identifiant stable, jamais un libelle), la position publiee et la fenetre de vie
      [t0, t1] bornee a la fenetre du document. Reutilise `ScanFilmWorldObjects(dir, wr, 37)`
      — aucun nouveau decodage de position. **Test unitaire PUR** `replay/
      equipment_objects_test.go` (entrees synthetiques, aucune I/O, patron
      `equipment_episodes_test.go`) : bornage a la fenetre, vie sans identite NON publiee,
      vie a identites contradictoires NON publiee.
- [ ] 4.4 Champ `ReplayDocument.EquipmentObjects []EquipmentObject`
      (`json:"equipmentObjects,omitempty"`) + `Coverage.EquipmentObjects`. **`SchemaVersion`
      NON TOUCHE** (contrat R4, §6) — la bosse 8 -> 9 revient au superviseur a la fusion.
- [ ] 4.5 Contrat OpenAPI regenere + `generated.ts` + normalisation web
      (`replayNormalize.ts`) : le champ TRAVERSE, il ne se DESSINE pas. Aucun composant de
      rendu touche, aucune string i18n ajoutee.
- [ ] 4.6 Libelles : aucune chaine FR/EN en dur cote Go. Si des noms sont poses, ils entrent
      dans `config/titles/halo_infinite/mappings/replay_labels.toml` — une ligne par classe
      NOMMEE en phase 3, aucune pour les classes numerotees.

**Gate 4** :

```
export GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-ti37/.gocache
cd apps/go-api && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./internal/analysis/...
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/... ./internal/games/...
make check-types && make test-web    (si 4.5 touche le web)
golangci-lint run --new-from-merge-base=origin/main
```

### Phase 5 — CLORE : registre, journal, statuts

- [ ] 5.1 Toutes les cases de ce plan statuees `[x]` / `[~]` / `[!]`.
- [ ] 5.2 Lignes proposees pour `.ai/V7.5/REGISTRE_REPORTS.md` (ecrites EN UNE SEULE FOIS,
      a la fin — contrat R4) : la ligne « Identite de l'objet ti=37 » du 2026-08-15 est
      SORTIE (si succes) ou AMENDEE de ce qui a ete refute (si negatif), plus une ligne par
      report neuf.
- [ ] 5.3 Entree `.ai/thought_log.md` REDIGEE et remise au superviseur (ce lot n'ecrit PAS
      dans le journal : il appartient au superviseur).

## 6. Contrat d'interface avec le lot jumeau R4 (`ti=11`, worktree `LevelUp-wt-ti11`)

R3 et R4 tournent EN PARALLELE sur deux worktrees. Ordre de fusion prevu : **R3 puis R4**.

1. **Fichiers CREES par R3, et eux seuls** : `filmdec/equipment_identity.go`,
   `filmdec/equipment_identity_test.go`, `replay/equipment_objects.go` (+ son test),
   ce plan, et les lignes de registre de 5.2.
2. **Fichiers PARTAGES du decodeur** (`filmdec/default_state.go`,
   `filmdec/default_state_arch.go`, `filmdec/traverse.go`, `replay/build.go`,
   `replay/document.go`, `replay/coverage.go`) : **une ligne d'enregistrement chacun au
   maximum**, jamais de reecriture, jamais de reindentation, jamais de reordonnancement.
3. **`SchemaVersion` : AUCUNE BOSSE.** Le champ ajoute par R3 est
   `ReplayDocument.EquipmentObjects` ; il est optionnel (`omitempty`) donc non cassant en
   lecture, MAIS il exige une bosse 8 -> 9 pour la meme raison que les schemas 7 et 8 : la
   reprise du backfill se fait par `SchemaVersion`, et un artefact sans le champ doit se
   voir comme « a re-cuire », pas comme a jour. **La bosse unique est faite par le
   superviseur a la fusion**, avec le paragraphe de `document.go` correspondant.
4. **Interdits** : aucun run de masse, aucune re-cuisson d'artefact publie sous `data/`,
   aucune ecriture sur une base DuckDB, aucune ecriture hors du worktree R3. Les films sont
   lus en LECTURE SEULE depuis le depot principal.
5. **Cache Go isole** : toutes les commandes portent
   `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-ti37/.gocache` (exclu via
   `.git/worktrees/.../info/exclude`, jamais le `.gitignore` versionne). **Une seule
   commande `go` a la fois** — le cache partage se corrompt sous deux `go` concurrents.

## 7. Regles dures (heritees, elles ont deja tranche ce chantier)

1. **Aucun nom, aucun effet, aucun son sans donnee mesuree.** Une classe non nommee garde
   son numero.
2. **C'est le DESERIALISEUR qui publie**, jamais un second lecteur pose a cote de lui
   (`equipment_state.go:15`).
3. **Offline-pur et universel** : pas de Cheat Engine, pas de capture runtime.
4. **Un seul decodage `filmdec` par process** (hooks globaux ; `LockProcessDecode` pour les
   chemins de production).
5. **Aucune base DuckDB ouverte en ecriture** : le serveur de l'utilisateur tourne.
6. **Zero fix hors perimetre.** Toute decouverte va en §9, pas dans le diff.
7. **Un negatif mesure est un livrable**, publie avec ses denominateurs et ses temoins.

## 8. Statuts d'item et cloture

`[x]` fait et verifie · `[~]` couvert ailleurs (avec la reference) · `[!]` non traite (avec
la justification ecrite). **Aucune case vide a la cloture d'une phase.** Clore une phase =
gate passe, items statues, plan mis a jour, commit sur `wt/ti37-identite` (prefixe
`feat(v7.5-rejeu-ti37):` / `mesure(...)` / `docs(...)`, terminaison
`Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`), `git push -u origin
wt/ti37-identite`, point d'etape.

## 9. Decouvertes — a consigner, NE PAS traiter dans ce lot

- **La grammaire du record d'IMAGE-CLE n'est pas celle du record NEW delta** (phase 1.3,
  1 226 records, 3 films, 8 combinaisons). Le decodeur n'avait jamais eu a la trancher :
  `WalkKeyframeWorld` apprend les largeurs de record EMPIRIQUEMENT et ne decode pas les
  corps. Consequence potentiellement large — elle vaut pour TOUS les archetypes, pas pour
  `ti=37` seul, et elle explique peut-etre pourquoi les positions d'armes au sol au keyframe
  n'ont jamais pu etre lues (`keyframe_ground_weapons.go`). NE PAS traiter ici.
- **La longueur d'un record `ti=37` d'image-cle prend un petit jeu de valeurs recurrentes,
  identiques d'un film a l'autre** (557 / 707 / 773 / 881 / 950 / 1 104 bits d'ecart
  residuel dominants sur les 3 films). Si cette longueur depend du TYPE de l'objet, c'est un
  signal de partition GRATUIT, lisible sans decoder un seul champ. A mesurer en phase 2.
- **`weap` et `sofd` rendent le MEME compte (267) sur `ti=42`**, et `weap` decale rend 267 sur
  `ti=35` : trois coincidences a la meme valeur, sur un balayage ou rien ne l impose. Piste :
  les deux catalogues partagent peut-etre des GlobalID (une entree indexee sous deux groupes),
  ou un meme motif de 32 bits appartient aux deux jeux. Sans effet sur le resultat `eqip`, qui
  est exclusif. NE PAS traiter ici.
- Les deux populations de portes de version de `ti=37` sur `000d5950` — (1,1) 236 records et
  (1,0) 195 records — expliquent EXACTEMENT le decalage de 8 bits observe entre les deux
  valeurs dominantes de `variant-name` (`0x00006808` et `0x00680814` = la meme suite d'octets
  lue a un octet d'ecart). Mesure conservee : c'est le genre de coincidence qui, non
  expliquee, fabrique une fausse identite a 14 classes.

## 10. Journal d'execution

**2026-08-17 — Phase 1 CLOSE.** Instrument ecrit (`filmdec/equipment_identity.go` +
`equipment_identity_test.go`, garde `EQUIP_ID_FILM`), quatre `R(32)` publies par les desers
de production (4 points de publication : 3 dans `consumeMultiplayerPropertiesBlock`, 1 dans
`consumeDefaultStateTI37`), oracle bit-exact construit et matrice des 8 grammaires probee.
Verdict : H1 refutee dans sa forme naive, 2 marches justes sur 1 226. Aucune valeur publiee.
Gate 1 passe (chiffres ci-dessus). Commit `feat(v7.5-rejeu-ti37)` sur `wt/ti37-identite`.

**2026-08-17 — Phase 2 CLOSE.** Ancrage par catalogue de tags : le groupe `eqip`
(116 tags) se concentre a 428 occurrences sur 428 dans les records `ti=37`, zero ailleurs,
contre ~0,8 attendue par hasard. Les trois groupes temoins (`weap`, `proj`, `sofd`)
atterrissent chacun sur leur archetype semantique — controle non impose par la methode.
Partition mesuree sur 5 films : 2 738 records porteurs sur 2 772 (98,8 %), 1 318 vies
identifiees, 12 classes communes aux 3 films d arene, stabilite par vie 99,6 a 100 %.
C1, C2, C3 PASSES. Les 4 champs de la voie H1 sont refutes et non publies.
Commit `mesure(v7.5-rejeu-ti37)` sur `wt/ti37-identite`.

## 11. Protocole de reprise de session

1. Relire le skill `plan-execution`, puis ce fichier de haut en bas : les cases cochees
   disent ou en est le lot ; le journal §10 dit ce qui a ete mesure.
2. Verifier la branche : `git -C C:/Users/Guillaume/Projects/LevelUp-wt-ti37 branch
   --show-current` doit rendre `wt/ti37-identite`.
3. Relire §6 (contrat R4) AVANT de toucher un fichier partage du decodeur.
4. **Verifier sur pieces** avant de coder : rouvrir le fichier et la ligne cible, le code a
   pu bouger (les references de §2 sont datees du 2026-08-17).
5. Reprendre a la premiere case non statuee de la phase courante. Ne pas re-decider ce qui
   est deja tranche.
