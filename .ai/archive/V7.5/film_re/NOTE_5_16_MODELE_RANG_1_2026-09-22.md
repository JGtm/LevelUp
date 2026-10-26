# Note 5.16 — Le modele du rang 1 : la table de datums par slot, et l image-cle qui la porte (2026-09-22)

> Lot 5.16, branche `feat/decfilm-66`, base `e13fafde1`. Cette note est le DOUBLE de la section
> du plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « Post-chantier — lot 5.16 ») : le plan fait
> foi sur l avancement, cette note sur la GRAMMAIRE.
>
> Elle prend la suite de `NOTE_5_15_RANG_1_FILM_DENSE_2026-09-22.md` (§3, les deux grammaires de
> `FUN_1406cd128` ; §6, le temoin de 96 bits ; §8, les maillons restants).

## 1. Le temoin de 96 bits est FERME, et il a suffi de le lire jusqu au bout

Le 5.15 laissait douze paquets de `dad793c7` faisant exactement 96 bits, un record lu, 42 bits de
reste. L anatomie complete, corrigee d une valeur (le 5.15 lisait « slot 1 314 » : c est **1 298**,
`id 0x40000512`) :

```
10100000011110110100001000000000 01000100101000100100100001000001 00000100110000011001110000000000

bit  0        1     bit de configuration du frame-processeur
bit  1        0     terminateur de la vue A vide
bit  2        1     prefixe DELTA
bits 3..15    123   idLow, 13 bits           } record 1 : ti=4, masque 0x1
bits 16..17   1     tag de generation        } i0 `high-frequency` = R(8)
bit  18       0     selecteur de baseline ferme
bits 19..28   0x1   masque (1 + 3 + 6)
bits 29..36         i0 high-frequency, 8 bits  -> fin du record au bit 37
bit  37       1     prefixe DELTA            <- l en-tete que la marche REJETAIT
bits 38..50   1298  idLow
bits 51..52   1     tag
bit  53       0     selecteur de baseline ferme
bits 54..63   0x2   masque = le composant i1
bits 64..87         i1 `splash-message-dynamic` = R(24)  (le compteur de 16 bits est DEDANS)
bits 88..90   000   terminateur de la vue B
bit  91       0     terminateur de la vue C (vide)
bits 92..95   0     bourrage d octet, PROUVE a zero
```

**LE PREMIER RECORD EST JUSTE, ET C EST VERIFIE CHEZ L ECRIVAIN** : `FUN_14076d034` (la variante
FRAME de `high-frequency`) lit exactement **8 bits**, ni plus ni moins. Le bit 37 est donc un vrai
prefixe de record, et le reste n est pas un mauvais cadrage du record precedent.

**CE QUI MANQUAIT EST L ARCHETYPE DU SLOT 1298.** L oracle du lot le trouve sans inventer une
largeur : les candidats sont les 50 archetypes du REGISTRE du film (`chunk_00`, un artefact LU) et
le critere est la FERMETURE DU PAQUET au bit — corps propre, terminateur `000` de la vue B, vue C
portee, reste a bourrage nul. Deux archetypes ferment les douze paquets (`ti=30` et `ti=47`), et
l image-cle du film tranche : **slot 1298 -> `ti=47`**, dont le composant `i1` est
`managed-object-networked-splash-message-dynamic-component`, un `R(24)`.

## 2. La source de l archetype : l image-cle EST le dump de la table de datums

| maillon | adresse | ce qu il ecrit |
|---|---|---|
| la liste des entites d une vue | `FUN_142f2e174` (slot `0x10` de la vtable de vue) | un mot de 32 bits par entite VIVANTE de `vue+0x38`..`vue+0x40` sous le bitmap `vue+0x58` : `vue+8 << 0x1e \| slot & 0x1fff \| genre` (genres `0x800000`, `0x1000000`, `0x1800000`, donc `e >> 0x17 & 0x7f` vaut 1, 2 ou 3) |
| la serialisation d une entree | `FUN_142f2c658` | aiguille sur le genre : `FUN_142f303bc` (1), `FUN_142f304a8` (2), `FUN_142f30610` (3 = l etat complet) |
| l etat complet | `FUN_142f30610` | `FUN_142f2c754(writer, 3, eid, archetype)`, puis `FUN_140769e08(writer, index)` (le selecteur de baseline), puis `FUN_142e35e60` (le corps) — et `archetype = *(int *)(*(vue[0x20] + 0x120) + 4 + slot * 0x18)` |
| l en-tete d un record | `FUN_142f2c754` | `[magic 0xf0c3a57e si HasExtraFields][1 bit de prefixe][2 bits de type si type != 3][drapeau + R(8) d archetype si HasExtraFields]`, l identifiant par `FUN_1406d5110` |

C est exactement l en-tete `[id:32][field:26][ti:6]` que `readKeyframeHeader` / `kfAnchorFromID`
lisent depuis le lot R5. **L image-cle est donc le DUMP de la table de datums que la branche vive
interroge** — la reponse est l hypothese (a) du brief.

### 2.1 Et notre lecture la TRONQUE : la mesure

`WalkKeyframeWorld` n est pas un parseur, c est un BALAYEUR d ancres : il suit la CHAINE des
records (frontiere par frontiere, slots croissants, fenetre de recherche de 120 000 bits) parce
qu il rend aussi leurs POSITIONS. Cette chaine se coupe.

| `dad793c7` | records rendus | dernier slot | bit d arret | payload |
|---|---:|---:|---:|---:|
| chunk 1 | **123** | **122** | **139 754** | 1 028 032 |
| chunk 2 | 187 | 1 345 | 303 622 | 1 043 848 |
| chunks 3 a 5 | 186 | 1 340 | 302 518 | 1 042 992 |

Et l identifiant du slot rejete par le temoin, cherche a TOUTES les positions de bit du payload
du chunk 1 : **une seule occurrence, au bit 279 659**, `field26 = 0`, `ti = 47`. Soit
**139 841 bits APRES** le point d arret — au-dela de la fenetre de 120 000 bits. La table est dans
le film ; la fenetre l a coupee.

### 2.2 Les deux autres hypotheses du brief tombent, par les adresses

**(b) « la branche vive a-t-elle un en-tete de record different ? » NON.** `FUN_1406d3140` (le
lecteur d identifiant de la branche 0, appele avec `param_3 = 7`) et l inline de la branche vive
lisent tous deux la **CLASSE 7** du tableau de couples (base, largeur) a `0x1451f98d0` ; le bit
supplementaire n existe que pour la classe 1 (`FUN_1405d5d08` -> `0x23` ou `0x28` -> couple 4).
Le tableau est **ENTIEREMENT NUL dans l image** (`read_memory 0x1451f98d0`, 80 octets) et n est
ecrit qu au runtime par `FUN_1408f1618` : couples 0 et 1 a `slot - 0x1ff`, couples 7 et 8 a
`slot + 1`, en meme temps que `DAT_144706100 = slot + 1` (lu a `0x1fff` dans l image, d ou les
13 bits).

**(c) « une autre source alimente la table » : les deux types de paquet que le depot ne lit pas
sont NOMMES et ECARTES.** Le repartiteur est `FUN_1428e22c0` (`*param_3` = le type) :

| type | handler | ce qu il fait | `dad793c7` | `bfecd02b` |
|---:|---|---|---:|---:|
| 0 | `FUN_1428e2778` -> `FUN_14298816c` | le frame-processeur (une trame) | 5 370 paq. | 31 232 paq. |
| 1 | `FUN_142989418` | lit **16 octets** et avance un compteur — le bloc est SAUTE | 1 715 095 o. | 9 261 513 o. |
| 2 | (aucun) | l image-cle : consommee hors du repartiteur de lecture | 650 107 o. | 4 593 116 o. |
| 10 | `FUN_142988244` -> `FUN_142991114` | 32 octets dans quatre globales (`DAT_144e51628`…) | 26 850 o. | 251 227 o. |

Aucun des deux ne pose une entite.

**`FUN_1428e24bc` est bien un aller-retour d etat, et on voit maintenant pourquoi** : il alloue
`0x1b000` octets, appelle `vtable[0x10]` pour la liste de reference, SERIALISE chaque entree par
`FUN_142f2c658`, retourne le writer en reader, abaisse `DAT_14474cd78`, appelle `vtable[0x40]`,
puis restaure. La branche 0 de `FUN_1406cd128` ne lit donc jamais que ce que la branche vive
vient d ecrire — sa table est complete par construction, et sa garde ne rejette jamais rien.

## 3. La baseline : une VALEUR, pas une largeur (D2 du 5.15, referme)

`FUN_14076cb60`, l iterateur de masque et de composants :

```
FUN_14076cb60(descripteur, args) :
   lecteur = args[5]
   FUN_1406d7610(descripteur, lecteur, &masque)          ; LE MASQUE, avant toute baseline
   extra = FUN_14076cea8()
   decales = 0
   pour i de 0 a descripteur[0x4320] :
      deser = descripteur[i * 8]
      si (*(TLS + 0x238) porte un nom non vide) et
         FUN_1428e1dac(&DAT_144c23178, descripteur[0x474c], nom(deser)) est FAUX :
             decales++ ; continue                        <- LE COMPOSANT NE CONSOMME PAS DE BIT
      si (masque >> ((i - decales) & 0xff)) & 1 :
             niveau = vtable[0x00](deser)                 ; la PRECISION, du DESCRIPTEUR
             si args[4] == 0 : prediction = 0             ; PAS de baseline -> reference NULLE
             sinon           : prediction = vtable[0x48](deser, tampon, &args[3])
             vtable[0x28](deser, lecteur, args, &prediction, niveau)   ; LA LECTURE
             [si extra et R(1)] R(32) ; si != 0x0bcddcba -> « entity component corrupt »
```

La baseline n entre que par `&prediction`, un vecteur de 16 octets que le deserialiseur APPLIQUE.
La largeur vient de `niveau` et des bits. **La baseline change une VALEUR, pas une largeur** :
`decodeDelta`, qui consomme le selecteur (`FUN_1406cdc04` = `R(1)` [+ `R(7)`], sentinelle `0xff` ;
ecrivain `FUN_140769e08`) et jette la baseline, ne DERIVE jamais.

Taux d ouverture du selecteur, mesure (`TestBaseline516Selecteur`) :

| film | records DELTA | mesurables | FERME | OUVERT |
|---|---:|---:|---:|---:|
| `dad793c7` | 5 622 | 5 616 | 5 616 | **0** |
| `bfecd02b` | 175 657 | 174 606 | 174 592 | **14** |

99,992 % sur un film dense. Les 14 records a selecteur ouvert publient leurs valeurs contre une
reference NULLE au lieu de l entree d historique designee (`iVar20 = (*ctx - selecteur) - 1`,
resolue par `FUN_141fda280`) : un ecart de VALEUR sur 14 records, dont le port exigerait de tenir
hors ligne l historique des etats de composants des trames precedentes. Report ecrit.

## 4. Le port, et ce qu il deplace

`keyframe_datums.go` — `TableDeDatums(pay)` lit `slot -> archetype` sous DEUX contraintes, les
deux prises de l ecrivain :

1. les gardes d en-tete de `kfAnchorFromID` (generation non nulle, slot borne, `field26` nul,
   `ti` sous le cap objet de 50) ;
2. la **CROISSANCE DES SLOTS** — `FUN_142f2e174` parcourt sa table par index croissant, donc les
   entrees du payload sont en slots croissants : on retient la plus longue sous-suite croissante,
   et on COMPTE les candidats ecartes.

La contrainte 2 n est pas un reglage. Sans elle la table est SALE, et la mesure le dit : quatre
paquets de plus en DEBORDEMENT sur `bfecd02b` (32 -> 36) et 162 liaisons posees sur `dad793c7` au
lieu de 54. Avec elle, zero debordement de plus et tout le gain.

`LierTableDeDatums` pose ces liaisons par `World.BindDatum` sans ecraser une liaison deja posee
(`Soft`, `GenAny`, vue INCONNUE, sans position : la table de datums ne dit QUE l archetype).
Appelants : `movementStateScanner.lierLeMonde` (production) et `m533bLierMonde` (instruments).

`rejetDeVue` porte desormais les DEUX gardes, et les deux sont COMPTEES :
`Observation.RejetsHorsDatum` (la garde VIVE : le slot n est dans aucune table de datums) puis
`Observation.RejetsDeVue` (le repli : la garde de la branche 0). Le repli mesure **ZERO** sur les
deux films, comme le 5.15.1 (d) l annoncait — il est compte plutot que supprime pour que ce zero
soit vu.

### 4.1 Le tableau (A/B par `MOUV516_DATUMS=0`, `TestGate516`)

| mesure | `dad793c7` avant | apres | `bfecd02b` avant | apres |
|---|---:|---:|---:|---:|
| paquets a reste NUL | 5 341 | **5 354** / 5 365 | 2 884 | 2 884 / 30 387 |
| debordements | 2 | 2 | 32 | 32 |
| records rendus | 5 628 | 5 641 | 176 323 | 176 786 |
| records `ti=35` | 75 | 75 | 129 572 | 129 572 |
| desyncs `ti=35` | 0 | 0 | 4 | 4 |
| rejets hors datum | 15 | **2** | 23 896 | 23 769 |
| rejets de vue (repli) | 0 | 0 | **0** | **0** |
| liaisons de datum posees | 0 | 54 | 0 | 10 |

`replay-equiv -films bcb6d393` sans `-update` : les SIX memes ecarts que le lot 5.14, aux memes
valeurs (`movementStates` 1 737, artefact 1 929 397 octets) — la reference est perimee depuis la
fusion 5.10 (report D1 (5.11)), et ce lot ne la deplace pas d un octet.

### 4.2 Ce que le trou portait, en clair

`dad793c7` : un seul archetype bouge, `ti=47` (30 -> 43 records), et un seul composant,
`managed-object-networked-splash-message-dynamic-component` (30 -> 43 lectures d un `R(24)`).
51 etiquettes de composant lues avant comme apres.

`bfecd02b` : 463 records de plus, tous sur des archetypes DEJA lus — `ti=37` equipement
4 340 -> 4 551, `ti=41` arme 566 -> 696, `ti=42` 2 755 -> 2 804, `ti=38` 19 -> 62, `ti=10`
1 003 -> 1 025, plus quelques unites ailleurs. 207 etiquettes avant comme apres.

**AUCUN CANAL D ETAT DE BIPEDE N APPARAIT** : pas une etiquette de composant de plus, sur aucun
des deux films. `replay.SchemaVersion` reste a **67**.

## 5. Le residu de `bfecd02b`, et sa cause n est PAS un trou de modele

Le chiffre ne bouge pas sur le film dense, et le lot ne le cache pas. Ce qui le dit :

- **632 slots distincts rejetes, dont 526 qu AUCUNE source lue ne declare** (ni image-cle, ni
  NEW) — et leur etendue couvre presque uniformement les treize bits (2 a 8 184, mediane 3 307,
  mesure du 5.15.1 (g)). Or la table de datums du film plafonne autour du slot 1 345 : ces
  identifiants ne sont pas des slots, ce sont des lectures prises a une position FAUSSE.
- **L oracle d archetype generalise le confirme** : sur `bfecd02b`, en resolvant chaque rejet par
  le registre et en relancant, 1 039 paquets ferment sur 26 137 tentes, et les archetypes elus
  sont repartis sur 33 valeurs de `ti` — la signature d un premier gagnant arbitraire, pas d un
  modele. Sur `dad793c7` le meme oracle ferme 13 paquets sur 24 et n elit qu UN archetype.
- Les **21 index de controle epars** de la vue C (D5 du 5.14) sont RE-MESURES a la sortie du lot :
  **inchanges** (29 classes, index 0 a 7 denses, 8 a 31 a 2-27 entrees). Ils restent du bruit du
  residu.

Le trou de `bfecd02b` est donc un DESALIGNEMENT EN AMONT, dans le corps d un record, et il faut le
chercher la ou le corps se lit — pas dans le modele d entites. Le §6 nomme le premier suspect.

## 6. Le maillon suivant, et il a une adresse : le DECALAGE DU MASQUE

Dans `FUN_14076cb60` le bit de masque teste est **`i - decales`**, et `decales` compte les
composants que `FUN_1428e1dac(&DAT_144c23178, descripteur[0x474c], nom)` ECARTE quand
`*(TLS + 0x238)` porte un nom de contexte non vide. **Le depot teste le bit `i` BRUT**
(`traverseComponentLoop`, `traverse.go`).

Autrement dit : les indices du masque ne sont pas ceux de l archetype, ce sont ceux des composants
que le contexte RETIENT. Et `&DAT_144c23178` est exactement l objet que le repartiteur de paquets
interroge pour le **type 8** (`FUN_1428e1e94(&DAT_144c23178)` puis
`FUN_142987bd4(session, paquet, …)`) — un type qui pese **700 868 octets sur `bfecd02b`** contre
20 622 sur `dad793c7`.

Un film dense pourrait donc porter une TABLE DE COMPATIBILITE de composants que le depot ne lit
pas, et dont le decalage deplacerait tous les bits de masque d un archetype. C est le premier
suspect du residu du §5, et c est une adresse, pas une absence.

## 7. Les maillons restants, par adresse

| maillon | adresse | ce qu il ouvre |
|---|---|---|
| **le decalage du masque par composant ecarte** | `FUN_14076cb60` (`i - decales`), filtre `FUN_1428e1dac(&DAT_144c23178, ti, nom)`, alimente par le paquet de **type 8** (`FUN_1428e1e94` / `FUN_142987bd4`) | le residu de `bfecd02b` : 23 769 rejets sur des identifiants qui ne sont pas des slots |
| l historique de baseline | `FUN_1406cdc04` -> `FUN_141fda280` -> bloc d arguments de `FUN_14076cb60` (`args[3]`, `args[4]`) | la VALEUR juste sur 14 records de `bfecd02b` |
| le predicat de mode du NEW | `FUN_1408f18d0()` (sans argument), `*(int *)(DAT_144c1cfa8 + 4)` | ce qui decide qu un record NEW est LU ou que la liste sort avec le code 2 |
| le second puits de la vue B | `vue[0x1b320]`, predicat `FUN_142f2b5c4`, publication `FUN_142f29538` | ou vont les records qu un film dense pourrait router ailleurs |
| les BITS D ACTION du controle | `FUN_1406d025c` | inchange depuis le 5.14 (ouvert 108 fois sur `bfecd02b`) |
| la table des 123 types de message de la vue A | `*(obj[0x18] + 0x210 + genre * 8)` | inchange depuis le 5.14 (vue A vide sur les deux temoins) |

## 8. Les instruments

Tous `_test.go` sous `//go:build research`, paquet `grammar`, variables `MOUV511_FILM`,
`MOUV511_BORNES`, `MOUV511_CARTE`, plus l A/B `MOUV516_DATUMS=0`.

| test | role |
|---|---|
| `TestTemoin516Anatomie` | le dump integral des paquets de 96 bits, l en-tete rejete, et ce que les deux lectures de l image-cle disent de son slot |
| `TestTemoin516Archetype` | l ORACLE : quels archetypes du registre ferment le paquet au bit |
| `TestTemoin516Generalise` | le meme oracle, a tous les paquets fautifs, avec relance |
| `TestTemoin516ImageCle` | balayeur filtre contre marcheur deterministe, et le croisement avec les slots rejetes |
| `TestTemoin516Population` | les types de paquet, les slots des images-cles, des NEW, des rejets |
| `TestTemoin516Ou` | quel chunk declare un slot rejete, et avec quel archetype |
| `TestTemoin516ArretImageCle` | ou s arrete le balayeur d image-cle, et les bits a l arret |
| `TestTemoin516Chercher` | la recherche d un identifiant a toutes les positions du payload |
| `TestTemoin516Datums` | la table de datums, ses conflits, et son effet sur le gate |
| `TestBaseline516Selecteur` | le taux d ouverture du selecteur de baseline |
| `TestGate516` | LE GATE du lot : fermeture, debordements, fantomes, `ti=35`, rejets ventiles |
| `TestGate516Contenu` | records et composants par archetype — le tableau avant/apres |
