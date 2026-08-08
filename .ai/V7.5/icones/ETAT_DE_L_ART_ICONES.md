# ETAT DE L'ART — les icones d'armes et du kill feed, extraites du jeu

> Ecrit le 2026-08-08, branche `feat/v75-icones`. Ce document fait foi sur ce qui est **prouve**,
> ce qui est **assume**, et ce qui a ete **refute**. Il execute et clot la phase 1 de
> `../../PLAN_RECHERCHE_ASSETS_ICONES.md` ; la phase 2 (integration web) reste derriere le gate
> visuel de l'utilisateur.
>
> Regeneration : `cd apps/go-api && go run ./cmd/weapon-icons-build` (machine avec Halo Infinite
> installe, cgo requis). Sortie : `static/weapons-assets/halo_infinite/jeu/`.

---

## 0. TL;DR

**Trois atlas d'icones ont ete extraits du jeu, 168 PNG au total.** Deux portent les memes
40 armes dans deux styles (contour et silhouette pleine) ; le troisieme, que j'ai longtemps
appele « sandbox », est **l'atlas du KILL FEED** : 88 icones couvrant armes, vehicules,
grenades lancees et pictogrammes de mort.

**Le lien arme -> icone est LU dans le jeu, pas devine** : champ `sprite index` du bloc
`UI display info` du tag `weap`. **29 armes sur 29**, chacune auto-validee.

**Les noms internes sont CRAQUES** : les tags ne portent que des murmur3, les chaines sont
strippees en release. **21 index nommes cote armes** (dont 10 marques « ? », voir §4) et
**58 cote kill feed**, ces derniers lies a leur index par le jeu lui-meme.

| Atlas | Tag | Images | Dimensions | Nommes |
|---|---|---|---|---|
| Armes, contour | `bc17adf1` | 40 | ~330x117 | 26 par le registre, 21 par le nom interne |
| Armes, silhouette | `e39747c8` | 40 | idem, index par index | idem |
| Kill feed | `0302cad3` | 88 | ~110x38 | 58 par sa table `bitd` |

---

## 1. LA CHAINE, MAILLON PAR MAILLON

Chaque etape a un controle. Aucune n'est postulee.

### 1.1 Ancre d'identite — deja versionnee, zero nouveau catalogue

Les 32 bits HAUTS d'un identifiant filmshell de `internal/games/weapons/registry.go` sont le
**global tag id du `weap`**. Controle croise : `80977ba5` (Mangler) et `d7915565` (Mutilateur)
apparaissent tels quels dans la colonne `detail` de `damagetag/data/labels.tsv`.

### 1.2 Du `weap` a l'atlas

Les 29 `weap` du registre referencent tous **les deux memes** `bitm` : `bc17adf1` et
`e39747c8`. Le troisieme bitm varie mais se PARTAGE par groupes d'armes (Mangler+Needler,
MA40+MA5K, six armes Covenant ensemble) : c'est un **reticule**, pas une icone. Le plugin
`weap.xml` le confirme independamment — le reticule a son propre champ
(`hip fire reticle screen reference`), ailleurs dans le tag.

### 1.3 Du tag aux pixels

L'entree `.module` porte a **`+0x10`** un index dans la table de ressources du module. Il vaut
**0** dans la variante `ds/` (qui n'a pas les pixels) et non nul dans `pc/` — c'est ce zero qui
prouve le champ.

Les descripteurs d'images se lisent comme un **tableau declare** : un compte (u32 juste avant
le premier enregistrement) et un pas regulier de **0x28 = 40 octets**. Le premier
enregistrement est retrouve par signature (dimensions repetees a +0x14), pas code en dur : la
lecture reste robuste aux versions.

### 1.4 Le format — controle arithmetique exact

Image 0 (333x117) : ressource de **53 372 octets** = en-tete 212 + donnees 72 + **53 088**, et
53 088 = 40 320 + 10 080 + 2 688, soit mip0+mip1+mip2 en blocs 4x4 sur 16 octets. **A l'octet
pres, sur toutes les images.** C'est ce controle qui sert aussi de filet a l'appariement
descripteur <-> ressource.

### 1.5 Le contenu — seul l'alpha porte le dessin

R constant a 255, G constant a 0, B = rampe verticale de teinte appliquee au rendu,
**A = LE DESSIN**. Seul l'alpha est extrait, rendu en blanc sur fond transparent — meme
convention que `static/abilities-assets`.

### 1.6 Quelle image designe quelle arme

Champ **`sprite index`** du bloc **`UI display info`** (sous-arbre `_38 "player interface"`) du
tag `weap`, nomme explicitement par la definition communautaire `weap.xml` :

```
_2  "name"              <- StringID (murmur3)
_2  "alt name " / "description" / "help text" / "icon string id"
_41 "sprite"            <- reference de tag = l'atlas
_6  "sprite index"      <- L'INDEX
_41 "alt sprite"        <- l'atlas silhouette
_6  "alt sprite index" / "damage sprite index"
```

**Auto-validation** : un bloc n'est accepte que si son champ `sprite` porte l'un des deux atlas
connus. Si rien ne matche, la commande le dit au lieu de rendre un index faux.

### 1.7 L'atlas du kill feed et sa table de nommage

Le tag **`bitd 8646f61a`** declare un bloc `entries` dont chaque enregistrement est le triplet
`identifier` (StringID) + `bitmap` (reference d'atlas) + `bitmap index`. **85 entrees**, toutes
vers `0302cad3`. Layout deduit de `bitd.xml` : +0 identifier (4), +4 bitmap (28, id a +8),
+32 index (2), +34 padding -> **36 octets par entree**.

### 1.8 Les noms internes — craques, pas lus

Les tags portent une table de chaines, mais elle ne contient **que des couples (index, hash)** :
les textes sont strippes en release. Recette reprise de
`../../ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md` §Q1.0-septies : moissonner les chaines du binaire,
hacher, confronter. Fonction : `mapvar.LabelHash` (murmur3 x86_32, seed 0), **deja dans le
depot et testee** — temoin `stockpile_socket` = 2110778921.

Motif du kill feed : **`killfeed_<nom>`**, trouve en essayant 23 prefixes x 8 suffixes x 83 mots
(14 608 combinaisons) — 4 correspondances immediates, puis 43 en prefixant tout le vocabulaire
du binaire.

Un **vocabulaire curate** (~95 mots) comble ce que la moisson ne rend pas : `fusion_coil`,
`power_seed`, `machine_gun`, `sandwich` ne figurent pas dans le binaire comme jeton isole.
Chaque entree n'est retenue que si son hachage tombe **exactement** sur un StringID cherche :
sur 32 bits, une centaine de candidats rend une collision fortuite negligeable. Le test echoue
bruyamment, il ne devine pas.

**Ce qui a le mieux marche : demander a un humain qui REGARDE les icones.** L utilisateur a
fourni une liste de mots en parcourant la page (`perfect`, `waterfall`, `backsmack`, `vip`,
`stick`, `core`, `kinetic barrel`, `flag`, « le Falcon avec des tourelles differentes »,
« les tourelles fixes ») : **9 correspondances immediates**. Et ces mots ont revele des MOTIFS
— `turret_<type>`, `<engin>_turret`, `<grenade>_grenade_stick`, `<objet>_melee`, `falcon_<arme>`
— dont la declinaison systematique (`patternVocabulary`) en a rendu **3 de plus**. Le kill feed
passe de 43 a **55 noms sur 88**. Aucune moisson automatique n avait donne ca.

**POURQUOI CERTAINS NOMS SORTENT ET PAS D AUTRES — la reponse est LEXICALE, pas structurelle.**
Un hachage ne rend AUCUNE information partielle : un nom sort si et seulement si la chaine
exacte a ete essayee. Il n y a donc ni « presque trouve » ni voisinage a exploiter. Trois
mesures le montrent :

1. **L ordre de la table `bitd` ne groupe rien** : rang 0 -> index 10, rang 1 -> index 32,
   rang 2 -> index 59. Aucune famille a suivre.
2. **L atlas, lui, EST ordonne** (armes 0-25, vehicules et tourelles 26-45, grenades 46-51,
   pictogrammes 52-72, divers 73-87) — ce qui donne le VOISINAGE d un trou, donc de bons
   candidats a essayer, mais pas le nom.
3. **Mesure decisive** : `bulldog`, `sidekick`, `disruptor`, `mangler`, `ravager` et
   `cindershot` SONT dans les chaines du binaire. `killfeed_<ce mot>` a donc bien ete essaye,
   et a echoue. Ce n est pas un defaut de moisson : **le kill feed ne les nomme pas par leur
   nom commercial**.

Ce que ces mots ont en commun, c est ce qui a permis d en debloquer trois de plus. Les noms qui
MARCHENT s ecrivent `<qualifiant>_<classe>` : `battle_rifle`, `assault_rifle`, `plasma_pistol`,
`sniper_rifle`. Decliner ce schema rend **`commando_rifle`, `ma5k_smg`, `plasma_blaster`** —
la ou `commando` et `ma5k` seuls echouaient. Le kill feed passe a **58 sur 88**. Elargi a
17 006 formes, le filon ne rend plus rien : les 30 restants portent un vocabulaire interne
qu aucune declinaison raisonnee n atteint. **Le gate humain reste la seule voie pour eux.**

**COMBIEN SONT ENCORE ATTEIGNABLES — 26, pas 30.** Les 30 index sans nom ne correspondent qu a
**26 identifiants** dans la table `bitd` : quatre images n ont AUCUNE entree de nommage. Aucun
craquage ne les nommera jamais, quel que soit le vocabulaire. C est une borne dure, pas un
manque d effort. (Les trous, dans l ordre : 3, 4, 8, 9, 10, 11, 12, 14, 20, 23, 25 cote armes ;
39-45 cote vehicules et tourelles ; 48, 52, 58, 62, 63, 68, 74, 75, 77, 82, 84, 87 ensuite.)

**Attention au decompte des trous** : le mode `kfgaps` reconstruisait son propre dictionnaire,
plus pauvre que celui de la production — sa liste contenait des SID DEJA resolus (`d253deb2` =
`killfeed_turret_plasma`, index 21). C est un sur-ensemble, donc aucune cible n a ete manquee,
mais viser du bruit fausse le decompte. La liste juste se lit dans `index.json` : un index kill
feed sans `nom_jeu` est un trou, point.

**Calibration avant application** : le #35 est le Sandwich d'apres le registre, et
`LabelHash("sandwich")` retombe sur son StringID. La methode est verifiee sur un cas connu
AVANT d'etre appliquee aux inconnus.

### 1.9 Le decodage BC7

Modes 4, 5 et 6 decodes **exactement** (99 % des blocs). Le mode 7 est **reconstruit** : points
extremes, bits P et index sont lus exactement, et ce qui manque (la partition) est retrouve par
ajustement sur le niveau de mip inferieur, decode lui aussi. Les tables de partition ne sont
volontairement pas recopiees de memoire — une table fausse rendrait une image plausible mais
fausse, et c'est le pire cas.

Les modes 0 a 3 restent en repli mais **ne degradent rien** : ces modes n'ont pas de canal
alpha, leur alpha vaut 255 partout et le repli le rend exactement. Or seul l'alpha est conserve.

**Mesure apres correction : zero bloc degrade** sur les 168 images. `index.json` porte
`bc7_rebuilt_pct`, `bc7_opaque_pct` et `bc7_degraded_pct` par image.

---

## 2. LES TABLES DE CORRESPONDANCE

Regenerees par la commande ; `index.json` fait foi.

### Atlas des armes (40 index)

Un « ? » signale un nom issu d un `weap` NON canonique : son index peut etre perime.

<!-- <<MD_WEAP>> -->
| # | weapon_key (registre) | nom interne craque |
|---|---|---|
| 00 | hinf_ma40_ar | assault_rifle |
| 01 | hinf_br75 | battle_rifle |
| 02 | hinf_vk78_commando | — |
| 03 | hinf_sidekick | — |
| 04 | hinf_cqs48_bulldog | — |
| 05 | hinf_s7_sniper | sniper_rifle |
| 06 | hinf_m41_spnkr | — |
| 07 | hinf_hydra | — |
| 08 | — | machine_gun ? |
| 09 | hinf_needler | needler |
| 10 | hinf_stalker_rifle | — |
| 11 | hinf_plasma_pistol | plasma_pistol |
| 12 | — | plasma_turret ? |
| 13 | hinf_pulse_carbine | — |
| 14 | hinf_energy_sword | energy_sword |
| 15 | hinf_ravager | — |
| 16 | hinf_gravity_hammer | gravity_hammer |
| 17 | hinf_skewer | skewer |
| 18 | hinf_mangler | — |
| 19 | — | — |
| 20 | hinf_cindershot | heatwave |
| 21 | hinf_heatwave | — |
| 22 | hinf_sentinel_beam | sentinel_beam |
| 23 | hinf_shock_rifle | — |
| 24 | hinf_disruptor | — |
| 25 | — | skull ? |
| 26 | — | flag ? |
| 27 | — | fusion_coil ? |
| 28 | — | power_seed ? |
| 29 | — | — |
| 30 | — | — |
| 31 | — | shade_turret ? |
| 32 | — | — |
| 33 | hinf_bandit | bandit \| bandit_evo |
| 34 | — | ball \| bomb ? |
| 35 | — | mythic_sandwich \| sandwich ? |
| 36 | hinf_ma5k_avenger | — |
| 37 | — | mutilator ? |
| 38 | hinf_fuel_rod_spnkr | — |
| 39 | hinf_vestige_carbine | — |
<!-- <</MD_WEAP>> -->

### Atlas du kill feed (88 index)

Noms lus dans la table `bitd`, donc lies a leur index par le jeu lui-meme.

<!-- <<MD_KF>> -->
| # | nom | # | nom | # | nom | # | nom |
|---|---|---|---|---|---|---|---|
| 00 | battle_rifle | 01 | commando_rifle | 02 | assault_rifle | 03 | — |
| 04 | — | 05 | turret_chaingun | 06 | hydra | 07 | sniper |
| 08 | — | 09 | — | 10 | — | 11 | — |
| 12 | — | 13 | gravity_hammer | 14 | — | 15 | skewer |
| 16 | plasma_blaster | 17 | plasma_pistol | 18 | sword | 19 | needler |
| 20 | — | 21 | turret_plasma | 22 | heatwave | 23 | — |
| 24 | sentinelbeam | 25 | — | 26 | warthog | 27 | rockethog |
| 28 | mongoose | 29 | gungoose | 30 | pelican | 31 | scorpion |
| 32 | wasp | 33 | wraith | 34 | phantom | 35 | banshee |
| 36 | ghost | 37 | chopper | 38 | scorpion_turret | 39 | — |
| 40 | — | 41 | — | 42 | — | 43 | — |
| 44 | — | 45 | — | 46 | frag_grenade | 47 | plasma_grenade |
| 48 | — | 49 | spike_grenade | 50 | plasma_grenade_stick | 51 | spike_grenade_stick |
| 52 | — | 53 | bomb_melee | 54 | callout | 55 | environment |
| 56 | repulsor | 57 | ricochet | 58 | — | 59 | ball |
| 60 | splatter | 61 | suicide | 62 | — | 63 | — |
| 64 | headshot | 65 | melee | 66 | back_smack | 67 | flag_melee |
| 68 | — | 69 | player_left | 70 | player_joined | 71 | player_rejoined |
| 72 | grappleshot | 73 | bandit | 74 | — | 75 | — |
| 76 | sandwich | 77 | — | 78 | waterfall | 79 | quantum |
| 80 | ma5k_smg | 81 | mutilator | 82 | — | 83 | perfect |
| 84 | — | 85 | falcon | 86 | falcon_chaingun | 87 | — |
<!-- <</MD_KF>> -->
---

## 3. CE QUI A ETE REFUTE (ne pas re-tenter)

| Piste | Verdict | Mesure |
|---|---|---|
| Index d'icone = petit entier a offset CONSTANT dans le corps du `weap` | **MORTE** | 0 candidat sur 29 armes (criteres poses avant de regarder : valeur < 40, >= 50 % distinctes). Le corps est un arbre de structures, les offsets bougent |
| Appariement automatique par SILHOUETTE contre les icones dessinees du depot | **MORTE** | marges de 0,00 a 0,10 pour des scores de 0,44 a 0,90. Corriger le rapport d'aspect ecrase n'a rien change : des armes toutes « longues et horizontales » se ressemblent trop une fois remplies |
| Appariement par SILHOUETTE **entre les deux atlas du JEU** (silhouette <-> kill feed) | **MORTE** | la refutation precedente portait sur un autre trace (icones dessinees du depot) ; celle-ci porte sur la MEME source, seule la resolution change — donc le verdict est general. Controle pose AVANT de conclure : 13 paires connues des deux cotes (`battle_rifle` contour 1 <-> kf 0, `needler` 9 <-> 19...). Resultat **2/13**, marges de 0,01 a 0,07 pour des scores de 0,45 a 0,72. Aspect ecrase : 1/13. Aspect preserve : 2/13. Le matcheur n a pas ete publie |
| Moisson des chaines **UTF-16LE** du binaire | **MORTE** | vrai trou de methode (la moisson ASCII coupe `s.i.d.e.k.i.c.k` a chaque caractere) mais gisement vide : **1749 chaines** seulement, 0 correspondance sur les 26 SID cherches. Le binaire est quasi entierement ASCII |
| Vocabulaire « designations » (MK50, CQS48, M41, S7, VK78) et mots vus a l ecran (scrap, coil, fall, turret, vestige) | **MORTE** | 26 166 formes declinees, **0 correspondance** sur les 26 vrais SID. Les designations ne sont pas ce que le kill feed hache |
| Appariement par RANG plugin <-> tag (la mecanique de `himap` pour `sbsp`) | **MORTE** | tombe UNE UNITE a cote — le bloc obtenu portait des references `mode`/`jmad`/`aset`. Un `+1` correctif ne serait garanti par aucune mesure d'une version a l'autre |
| Sonde de recalage plus profonde (24 au lieu de 6) | **PIRE** | la corruption remonte de l'index 73 a l'index 42 et emporte grenades et pictogrammes : 84 images appariees dont 42 lisibles, contre 85 dont 73 |
| Densite de transitions d'opacite comme detecteur d'image corrompue | **NE SEPARE PAS** | l'icone d'explosion, parfaitement legitime, sort en tete du classement de bruit (0,1437) devant les images reellement rayees |
| `gggl -> eqip -> bitm` pour les grenades | **RIEN** | les 8 equipements lancables declares ne referencent aucun bitmap |
| `cusc` (l'autre referenceur de l'atlas kill feed) | **SANS OBJET** | composition d'UI generique (components, long/string_id properties), aucun champ `sprite index`. Le `bitd` a repondu mieux |
| `HaloInfinite.exe` a la racine de l'installation | **PIEGE** | c'est un lanceur de 3,9 Mo : **0 nom craque**. Le binaire du jeu est sous `game/` et pese 80 Mo (408 525 chaines) |
| Les paquets `.cms` comme vocabulaire (`gamecms.cms`, 493 Mo) | **RIEN, ET DANGEREUX** | 6 068 000 chaines pour **une seule** correspondance — `killfeed_3O1\`, du bruit. A ce volume l'esperance de collision fortuite vaut ~0,2 : l'argument « une egalite de hachage vaut certitude » NE TIENT PLUS. C'est ce qui a motive le filtre `plausibleIdent` (`[a-z0-9_]+`) |
| Elargir le vocabulaire par combinaison (456 formes : prefixes x suffixes x mots) | **EPUISE** | +1 seul nom (`splatter`). Les 41 identifiants restants du kill feed emploient des mots hors de tout vocabulaire devinable — il faut une source, pas plus d'essais |

---

## 4. RESERVES ET DIVERGENCES OUVERTES

- **Index 20 des armes** : le jeu le nomme `heatwave` alors que le registre y lit
  `hinf_cindershot` — et les deux viennent du MEME tag canonique `230447b1`. Soit le registre
  etiquette mal ce tag, soit le nom interne est un reliquat de developpement. Le depot a deja
  une confusion documentee dans ce coin (`Cremator.png` est en realite le Cindershot). **A
  trancher a l'oeil** sur les icones 20 et 21.
- **Tags de campagne a index perime** : un `weap` legacy baptisait l'index 7 « shotgun » la ou
  le registre lit la Hydra. Filtre pose — un index revendique par le registre n'accepte que le
  nom d'un tag canonique. Les conflits restants sont donc signifiants.
- **Noms marques « ? » (10 index cote armes)** : ils viennent d un `weap` NON canonique, et
  les tags de campagne portent des index PERIMES. **Cas avere signale par l utilisateur** :
  l index 31 se faisait appeler `shade_turret` alors que l image est une caisse. Les noms
  restent publies — mais marques, jamais servis comme des faits. A l inverse `fusion_coil`
  (27), `power_seed` (28) et `machine_gun` (8) collent a l image : le marquage dit le DOUTE,
  pas la faute.
- **8 index sans nom ni cle** cote armes, **33** cote kill feed. Le levier est le vocabulaire
  curate : chaque mot ajoute est teste, jamais affiche sans preuve.
- **Les 3 grenades du registre** n'ont pas de bloc `UI display info` : ce ne sont pas des `weap`
  (elles vivent en `eqip` + `proj` declares par `gggl`). Leurs icones existent en revanche dans
  l'atlas du kill feed (`frag_grenade` 46, `plasma_grenade` 47, `spike_grenade` 49).
- **Halo 5** : non couvert. Le jeu n'est pas installe sur ce poste et son format d'archives
  differe ; la voie a explorer est l'API de metadonnees officielle H5, pas ces archives.
- **Definition ancienne du reticule** : les groupes de partage du 3e bitm (Mangler+Needler,
  MA40+MA5K...) n'ont pas ete verifies contre `weap.xml`. Rien n'en depend.

---

## 5. OU EST QUOI

| Quoi | Ou |
|---|---|
| Le binaire de regeneration | `apps/go-api/cmd/weapon-icons-build/` |
| Definition de tag `weap` | `apps/go-api/cmd/weapon-icons-build/weap.xml` (embarquee par `go:embed`) |
| Les 168 PNG + `index.json` | `static/weapons-assets/halo_infinite/jeu/` |
| Page de nommage (hors app) | `NOMMAGE_ICONES.html` (ce dossier) |
| Planches-contact | `planche_contour.png`, `planche_silhouette.png`, `planche_killfeed.png` |
| Fonction de hachage | `internal/analysis/replay/mapvar/hash.go` (`LabelHash`) |
| Plugins de tags (source) | `Gamergotten/Infinite-runtime-tagviewer`, `Plugins/*.xml` — 479 definitions |

**Rien n'est branche cote web.** Ni `apps/web/` ni `adapter_asset_urls.go` ne sont touches :
l'integration attend le gate visuel (decision #4 du master plan).
