# PLAN — Extraction des sons de tir par arme (tags `sbnk`)

> Branche : `feat/extraction-sons-armes` — worktree `LevelUp-wt-replay2d`
> Ouvert le 2026-08-15. Contrat d'execution : skill `plan-execution` (fait foi).

## Probleme

Les `.pck` du jeu (`Sound/win/SFX/`) livrent 90 170 `.wem` sans aucun nom : les banks
Wwise n'y sont pas (0 bank sur 1645 `.pck`) et les noms de tags sont absents des modules
(`stringsSize` = 0 sur les 132 modules, 0 chemin ASCII sur 665 Mo scannes). Un pack par
arme identifie l'ARME de facon certaine, mais rien n'identifie le TIR parmi les 80 a 360
sons du pack. Le tri par duree ne generalise pas : le fusil d'assaut utilise 296 micro
echantillons de 0,08 s en round-robin, le sniper des one-shots de 4,5 s avec queue.

## Hypothese de travail (mesuree, pas postulee)

Les banks Wwise ont ete converties en tags `sbnk` dans les modules. Mesures a l'appui :

- `pc/globals/globals-rtx-new.module` contient **1305 tags `sbnk`** (et 14 228 `snd!`,
  111 `weap`) — a comparer aux 1645 `.pck` du dossier `Sound`.
- Les **40 IDs `.wem` du fusil d'assaut** cherches en clair dans les modules sont tous
  localises dans des tags `sbnk` de ce module.
- Les 1305 `sbnk` sont **tous compresses Oodle** (aucun `cs == us`) : `internal/ooz` est
  obligatoire. Verifie : `go build ./internal/ooz/` passe (msys64 ucrt64 g++).

Si le contenu decompresse est une bank Wwise (`BKHD`/`HIRC`), on remonte
Evenement -> Conteneur -> Son -> `.wem`, et le tir devient identifiable de facon certaine.

## Outillage deja disponible (ne rien reecrire)

| Piece | Emplacement | Role |
|---|---|---|
| `hmod.go` | `cmd/weapon-icons-build/` | lecteur `.module`, `dataOffset` 48 bits, blocs |
| `internal/ooz` | `apps/go-api/internal/ooz/` | decompression Kraken (CGO) |
| `hdr.go` | `cmd/weapon-icons-build/` | header `ucsh` + table de dependances typees |
| `weap.xml` | `cmd/weapon-icons-build/` | definition du tag `weap`, champs nommes |

## Etapes

### Etape 1 — Sonde : format reel du contenu d'un `sbnk`

Gate : `go run ./cmd/weapon-sounds -mode probe` statue OUI/NON sur la presence de
`BKHD`/`HIRC` et designe le `sbnk` portant les `.wem` du fusil d'assaut. PASSE.

- [x] Squelette `cmd/weapon-sounds` qui reutilise `himodule` + `ooz` (pas de copie)
- [x] Decompression des `sbnk` et recherche des signatures Wwise
- [x] Verdict ecrit au journal : bank Wwise VERBATIM

DECISION DE BIFURCATION : sans objet, `HIRC` est present. L'etape 2 est nominale.

### Etape 2 — Parser la hierarchie et produire evenement -> liste de `.wem`

Gate : dump JSON pour `sb_010_wea_un_assaultrifle`, non vide, dont les IDs appartiennent
tous au `.pck` de cette arme (controle croise). PASSE.

- [x] Parser des objets HIRC utiles (Event, Action, Random/Sequence Container, Sound)
- [x] Resolution transitive Event -> ensemble de `.wem`
- [x] Controle croise : 0 `.wem` hors du `.pck`, couverture 359/359

### Etape 3 — Designer l'evenement de TIR parmi les 22

Gate : au moins le fusil d'assaut, le sniper et le needler ont leur evenement de tir
designe, par une preuve qui ne repose pas sur la duree des sons.

Voie A — hachage FNV-1 (TENTEE, INSUFFISANTE SEULE) :

- [x] Generateur de candidats : noms de `.pck` x verbes x prefixes usuels Wwise
- [x] Hachage FNV-1 32 bits minuscule, appariement sur les IDs d'evenements
- [!] Resultat : **0 nom retrouve sur 18 evenements**. La fonction de hachage est
  PROUVEE correcte (vecteurs de reference + calibrage sur l'ID de bank, cf. journal) :
  c'est la forme des noms d'evenements qui n'est pas devinable. Voie conservee comme
  complement opportuniste, abandonnee comme moyen principal.

Voie B — le graphe de tags (NOUVEAU MOYEN PRINCIPAL) :

- [ ] Localiser les tags `snd!` qui referencent les identifiants d'evenements d'une bank
- [ ] Remonter au tag `weap` qui depend de ces `snd!`
- [ ] Lire l'offset du champ de tir dans `weap.xml` (champs NOMMES) pour designer lequel
      des `snd!` est le tir — c'est la preuve recherchee, sans aucun nom Wwise

### Etape 4 — Livrable

Gate : un fichier par arme listant les `.wem` de tir, et un export `.wav` de ceux-ci.

- [ ] Table arme -> evenement de tir -> `.wem`
- [ ] Croisement avec le `weap` global tag id (cle deja etablie par `weaptags.go`)
- [ ] Export audio des seuls sons de tir

## Journal

### 2026-08-15 — Ouverture

Prerequis verifies sur pieces avant ouverture : `ooz` compile ; 1305 `sbnk` denombres ;
IDs `.wem` du fusil d'assaut localises dans des `sbnk`. Branche creee depuis
`71bdb589f` (worktree en HEAD detache auparavant).

### 2026-08-15 — Etape 1 CLOSE : le `sbnk` est une bank Wwise verbatim

Commande : `go run ./cmd/weapon-sounds -mode probe -limite 1305`.

Mesure sur les 1305 `sbnk` de `pc/globals/globals-rtx-new.module` : **1305 decompresses,
0 echec**, en-tete `ucsh` sur 1305/1305. Signatures : `BKHD` 1299, `HIRC` 1296,
`DIDX` 694, `DATA` 694, `STMG` 3, `STID` 2. La charge utile du tag EST la bank, pas un
format maison — et elle est dans les octets du tag, pas dans le blob de ressources
(`res = 0 o` sur le tag temoin ; la sonde des ressources est restee, elle ne coute rien).

Le `sbnk` du fusil d'assaut est designe sans ambiguite : **`gid 384b727f`** (1 536 586 o)
est le SEUL des 1305 a porter les 6 `.wem` temoins.

Correctif d'ergonomie inclus : `-limite` par defaut passe de 60 a 0 (= tous). L'heuristique
initiale « une bank d'arme est petite, sonder les plus petites » est REFUTEE — la bank du
fusil d'assaut fait 1,5 Mo et etait absente des 60 plus petites, d'ou un premier verdict
« HIRC absent » qui etait un artefact d'echantillonnage.

Consequence pour l'etape 2 : `STID` n'est present que sur 2 banks. La table des noms
d'evenements est donc quasi absente — l'etape 3 (hachage FNV-1) reste bien necessaire.

### 2026-08-15 — Etape 2 CLOSE : hierarchie resolue, 359/359 couverts

Commande : `go run ./cmd/weapon-sounds -mode map -pck <...>sb_010_wea_un_assaultrifle.pck`.

Mesure : `sbnk` gid `384b727f` designe par intersection des IDs (aucun nom en jeu),
642 objets HIRC, 391 Sound resolus, 22 Events, 35 Actions. **Couverture 359/359** `.wem`
du `.pck` atteints depuis un evenement, **0 `.wem` hors du `.pck`**.

DECISION DE CONCEPTION : le parseur ne postule aucun offset. Les trois lectures ambigues
du format HIRC (dependant de la version de Wwise, non exposee) sont ESSAYEES PUIS VALIDEES
contre un ensemble connu : `sourceID` de Sound valide par l'appartenance aux IDs du `.pck` ;
liste d'enfants d'un conteneur retenue seulement si TOUS ses elements sont des objets de la
bank (on garde la plus longue) ; liste d'Actions d'un Event retenue seulement si tous ses
elements sont des objets de type Action. Le controle croise a 0 hors-pck valide l'approche :
une lecture au mauvais offset n'aurait pas survecu.

Profil des 22 evenements : 8 gros (64 a 103 `.wem`) puis une queue a 4 `.wem`. Les gros sont
les candidats tir (round-robin), les petits la mecanique. **Les departager exige les noms**
— c'est l'objet de l'etape 3, elle n'est pas contournable.

### 2026-08-15 — Etape 3 : la voie du hachage echoue, changement de moyen

CORRECTION D'UNE AFFIRMATION DE L'ETAPE 1 : le journal annoncait « STID 2, STMG 3 ». C'est
FAUX — la sonde cherchait ces quatre lettres N'IMPORTE OU dans les octets (`bytes.Contains`),
pas un vrai chunk. Le mode `noms`, qui decode reellement les chunks, ne trouve **aucun**
`STID` sur les 1305 banks. Il n'y a donc aucun nom en clair nulle part dans le jeu, ce qui
est coherent avec `stringsSize` = 0 sur les modules.

Voie A (hachage) : **0 nom retrouve sur 18 evenements**. Deux garde-fous prouvent que
l'echec ne vient ni de la fonction ni d'un bug :

1. `noms_test.go` verifie FNV-1 sur les vecteurs standards ("" -> 811c9dc5, "a" -> 050c5d7e,
   "foobar" -> 31f0b262) et l'insensibilite a la casse ;
2. calibrage sur une donnee dont le nom EST connu — l'identifiant de bank du chunk `BKHD`
   vaut `4f8f2090`, et `fnv1("sb_010_wea_un_assaultrifle")` vaut exactement `4f8f2090`.

La convention est donc bien « FNV-1 32 bits du nom complet en minuscules ». Ce qui manque,
c'est la FORME des noms d'evenements, et elle n'est pas devinable a partir du nom de bank.

DECISION : la voie principale devient le graphe de tags. Le raisonnement : le tag `weap`
porte des champs NOMMES (`weap.xml`, 84 Ko de definitions deja au depot) ; un de ces champs
designe le son de tir ; il pointe vers un `snd!` ; le `snd!` porte un identifiant d'evenement
Wwise. La preuve ne repose alors sur aucun nom Wwise ni sur aucune heuristique acoustique —
elle vient du champ nomme de la definition de tag.

## Decouvertes (hors perimetre — ne pas traiter ici)

- `cmd/weapon-icons-build/hmod.go` duplique volontairement `internal/himodule` (u32 vs
  48 bits). Si `cmd/weapon-sounds` en devient un 3e consommateur, la regle des 2 copies
  impose de promouvoir le lecteur corrige en package partage.
