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
tous au `.pck` de cette arme (controle croise).

- [ ] Parser des objets HIRC utiles (Event, Action, Random/Sequence Container, Sound)
- [ ] Resolution transitive Event -> ensemble de `.wem`
- [ ] Controle croise : intersection non vide avec les IDs du `.pck` de l'arme

### Etape 3 — Retrouver les noms d'evenements (FNV-1 32 bits)

Gate : au moins le fusil d'assaut, le sniper et le needler ont un evenement de tir nomme.

- [ ] Generateur de candidats : noms de `.pck` x verbes x prefixes usuels Wwise
- [ ] Hachage FNV-1 32 bits minuscule, appariement sur les IDs d'evenements
- [ ] Statuer : couverture obtenue, armes non resolues

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

## Decouvertes (hors perimetre — ne pas traiter ici)

- `cmd/weapon-icons-build/hmod.go` duplique volontairement `internal/himodule` (u32 vs
  48 bits). Si `cmd/weapon-sounds` en devient un 3e consommateur, la regle des 2 copies
  impose de promouvoir le lecteur corrige en package partage.
