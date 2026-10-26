# Note — PlayerGameEventSmall (0xE9, type 82) DECODE — 2026-09-01

Instrument : `internal/analysis/filmdec/playergameevent_0xe9_research_test.go`
(+ `playergameevent_0xe9_helpers_test.go`). Garde `LOT1_TRAME_FILM`, borne 12 chunks,
verrou process, lecture seule. Films : `000d5950` (Fiesta), `01e1f945`, `00502e52`.

## Question

`PlayerGameEventSmall` (~923 k evenements sur le corpus, jamais decode) porte-t-il, COTE
TIREUR, des confirmations de touche / degat inflige attribuees au JOUEUR (+ arme), qui
couvriraient les armes explosives que `damage_aftermath` rate (owner du projectile absent
du film, cf. `NOTE_PROJECTILE_OWNER`) ?

## Verdict

**NON. `PlayerGameEventSmall` est un SAC DE PROPRIETES NOMMEES TYPEES — un evenement de jeu
CATEGORIEL (medaille / score / evenement de mode), PAS un enregistrement de touche.** Sa
charge ne porte NI WeaponID, NI magnitude de degat, NI reference de victime structuree, NI
paire tireur+cible. Il ne couvre donc AUCUNE touche explosive attribuee a un tireur+arme.
La jointure temporelle tir<->impact reste la seule voie pour les touches non fatales.

Confiance : **HAUTE**. La grammaire est bit-exacte (lue dans l'exe + validee par l'oracle de
trame a 3.2-3.4 records/paquet contre un temoin a 0.00 sur les 3 films), et l'absence de tout
champ arme/cible/tireur est mesuree, pas supposee.

## Grammaire, lue dans l'exe (HaloInfinite.exe, image base 140000000)

La table de handlers (0x144724xxx) est indexee par un enum INTERNE, PAS par le type filaire
(fire type 36 est au slot 0x144724DD8 = index 53, damage type 0 au slot 0x144724f80). Le
descripteur `PlayerGameEventSmall` se trouve par la chaine : le thunk de nom
`0x14119e930` (`LEA RAX, ["PlayerGameEventSmall"@0x143c97e80] ; RET`) est reference depuis
`0x143d0ec20`, donc l'objet-descripteur a pour base **`0x143d0ec18`** (methode nom a +0x08).
Alignement des champs confirme contre l'objet fire `0x143d0aca0` (+0x38 identiques, +0x50
meme famille) :

| Champ | fire (0x143d0aca0) | PGES (0x143d0ec18) | Role |
|---|---|---|---|
| +0x58 | 0x14080a048 | **0x142ef7f6c** | fonction de domaine des 3 refs d'en-tete |
| +0x68 | 0x14080c1f8 | **0x14080add8** | lecteur de charge |

**Domaines des 3 refs** (fonction 0x142ef7f6c, calibree contre damage `{1,1,7}` = fonction
0x14080a018 et fire `{1,...}` = 0x14080a048, qui rendent exactement les domaines connus) :
index0 -> **dom0**, index1 -> **dom8**, index2 -> **dom7**. Tous largeur 13, sans sonde
(la sonde est propre au domaine 1). Chaque ref = `[R(1) porte ; si 1 : R(13) index + R(2) gen]`.

**Charge** (`FUN_14080add8` -> `FUN_14080ae70` puis finaliseur `FUN_14080ae28`) :

```
R(32) A          identifiant/type de l'evenement (out[0])
R(8)  B          champ court (out+8)
liste de proprietes :
   R(3) compte (0..7)
   compte x [ R(32) nom-hache + R(3) selecteur + valeur typee ]
       selecteur -> valeur : 0 -> 0 bit (drapeau) · 1 -> R(32) int · 2 -> R(32) string-id ·
                    3 -> R(32) · 4 -> R(1) bool · 5 -> chaine (R(8) jusqu'a 0, max 16 octets) ·
                    6 -> R(32) · 7 -> palette quantifiee a largeur runtime (rare)
bloc "text" optionnel (FUN_14080b034, nom de champ "text"@0x1436f4b68) :
   R(1) porte ; si 1 : R(32) nom, R(3) compte, compte x element (FUN_1407f0ebc) :
       R(3) sous-type -> 0 : 0 bit · 1 : R(1)+[si 0 : R(5) index participant absolu, espace
       killsource FUN_1407f2058/FUN_140e958c4] · 2 : quantifie runtime · 3 : R(32) string_id ·
       autres : R(32)
R(32) masque final (FUN_14080ae28 : 32 x R(1))
```

Primitives : `FUN_14080dec4` = R(32) ; `FUN_1406cf008` = R(1) ; `FUN_14080ef08` = R(3)+valeur ;
`FUN_1407cbc24(...,0x10)` = chaine <=16 octets ; `FUN_14076f91c` (gate du selecteur 7) = drapeau
GLOBAL (DAT_144e61ea0/DAT_145121140), pas un bit de flux — d'ou le caractere runtime du selecteur 7.

## Mesures (3 films, 12 chunks)

| mesure | 000d5950 | 01e1f945 | 00502e52 |
|---|---|---|---|
| type 82 · type 83 | 255 · 0 | 364 · 0 | 224 · 0 |
| charge inexacte (sel. 7 runtime) | 0 | 5 | 0 |
| champ A : distinct / total | 17 / 255 (6.7 %) | 31 / 364 (8.5 %) | 16 / 224 (7.1 %) |
| proprietes/evenement | 0 partout | 0 x294, 1 x70 | 0 partout |
| noms de propriete distincts | 0 | 2 | 0 |
| bloc "text" present | 7 (2.7 %) | 12 (3.3 %) | 4 (1.8 %) |
| refs d'en-tete presentes (ref0/1/2) | 0 % | 0 % | 0 % |
| **ORACLE DE TRAME (records/paquet)** | **3.37** (temoin 0.00) | **3.16** (temoin 0.00) | **3.34** (temoin 0.01) |
| VERDICT cadrage bit-exact | TENU | TENU | TENU |
| A intersecte un WeaponID | 0 / 255 | 0 / 364 | 0 / 224 |
| noms de propriete = WeaponID | — | 0 / 2 | — |
| coincidence evenement<->tir ±250ms | 22.0 % (temoin 24.7 %) | 47.3 % (temoin 43.7 %) | 24.1 % (temoin 29.9 %) |

Lecture :

1. **Cadrage bit-exact PROUVE.** L'oracle de trame (meme juge que damage_aftermath) rend
   3.2-3.4 records/paquet au bon cadrage contre 0.00 au temoin decale de +3 bits : la charge
   entiere se consomme au bit pres. Les 5 evenements « inexacts » de 01e1f945 sont des
   proprietes a selecteur 7 (palette runtime), proprement exclus de l'oracle.

2. **Champ A = un enum d'evenement.** 16-31 valeurs distinctes seulement (~7-8 %), avec un
   noyau partage entre films (606 dominant, puis 519, 607, 320, 322, 325) et quelques valeurs
   propres au film (800, 805, 343). C'est un TYPE d'evenement categoriel, pas une donnee
   par-coup. (Temoin de bruit arme : ~79-81 % de distinct — ici 7 %.)

3. **Pas de charge structuree de touche.** Les proprietes sont quasi toujours absentes
   (0 propriete sur 000d5950 et 00502e52 ; 70/364 a une seule propriete sur 01e1f945, de type
   int32 hache ou palette). Aucune n'est un WeaponID, une magnitude ou une cible.

4. **Aucune attribution d'arme.** Ni A ni aucun nom de propriete n'intersecte l'ensemble des
   WeaponID des tirs (bas ou haut 32 bits) sur aucun film : 0 sur 843 evenements cumules.

5. **Aucune correlation avec les tirs.** La coincidence evenement<->tir est AU NIVEAU ou
   SOUS le temoin decale sur les 3 films : ces evenements ne suivent pas le combat coup par
   coup. (Le pic a 47 % sur 01e1f945 = densite de tirs elevee, egale par le temoin — non
   discriminant.)

6. **Emetteur non porte par l'en-tete.** Les 3 refs d'en-tete (dom0/8/7) sont absentes a
   100 % sur ces films arene : l'evenement n'est pas rattache a une entite joueur par sa
   porte. Le seul lien vers un participant est le sous-type « text » #1 (index R(5), espace
   killsource) — present sur 2-3 % des evenements seulement, cote annotation.

7. **type 83 (TeamGameEvent) = 0** sur ces 3 films arene : seul le type 82 est emis.

## Consequence

`PlayerGameEventSmall` est le canal des evenements de jeu du joueur (medailles, score,
notifications de mode) sous forme de proprietes nommees generiques — exactement le scenario
que le garde-fou anticipait. Il ne fournit PAS de touche exploitable, et surtout pas
d'attribution tireur+arme couvrant les explosifs. Les canaux de touche restent :
`damage_aftermath` (0xC0, tir direct apparie au tireur) et la jointure de vol tir<->impact
pour les projectiles (cf. `NOTE_TOUCHES_EXPLOSIVES`, `NOTE_PROJECTILE_OWNER`).

Piste NON poursuivie (hors sujet touches) : nommer les ~17-31 valeurs du champ A (enum
d'evenement) donnerait un signal produit distinct (medailles/score par joueur), a condition
de resoudre l'enum dans l'exe et de relier l'emetteur — chantier separe.

## Reproduire

```
export GOCACHE=".../.gocache-e9" ; export CGO_ENABLED=0
export LOT1_TRAME_FILM=".../data/cache/film_chunks/000d5950"
go test ./internal/analysis/filmdec/ -run '^TestPlayerGameEventSmall$' -count=1 -v -timeout 30m
```
