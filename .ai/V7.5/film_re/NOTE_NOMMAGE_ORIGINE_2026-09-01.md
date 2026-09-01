# Le ramassage non-arme est NOMME a 100 % par les fichiers du jeu, les classes sont tranchees, l origine ne l est pas

Date : 2026-09-01. Lot 5, RECHERCHE PURE (aucun fichier de production touche, aucune cuisson).
Worktree `LevelUp-wt-pickup-nommage`, branche `wt/pickup-nommage`, base `6ce0fcc2a`.
Instruments : `analysis/replay/equipment_pickup_{manifest,origin}_research_test.go`,
`analysis/filmdec/equipment_{grammar_audit,palette_section3}_research_test.go`, sous gardes
`BIPED_PICKUP_FILM` (etapes 1-2, addendums A et B) et `PICKUP_FILM` + `PICKUP_MAP` (etape 3).
Sautes sans elles : aucun effet en CI.

## LE RESULTAT EN UNE PHRASE, ET POURQUOI IL ETAIT A PORTEE DEPUIS DEUX SEMAINES

Le `R(32)` des ramassages de classe 2 et 3 est un **GlobalID de tag `eqip`**, et le depot
porte DEJA la table qui le nomme : les 21 lignes `[[equipment_objects]]` de
`config/titles/halo_infinite/mappings/replay_labels.toml`, construites le 2026-08-18 en
remontant la chaine `sofd -> sofa -> {string_id, eqip}` dans les fichiers du jeu installe
(plan `.ai/V7.5/replay2d/PLAN_NOMMAGE_EQIP_TRANSLOCATEUR.md`).

**Le lot 4 a cherche a nommer par correlation statistique ce que le depot savait deja nommer
par la structure du jeu.** Ses deux voies (delta i48, etat des images-cles) plafonnaient a
19-29 % de couverture ; la table du jeu couvre **100 %**. La lecon n'est pas que la correlation
etait mauvaise — elle etait juste, et c'est elle qui VALIDE la table aujourd'hui — c'est qu'on
n'avait pas cherche si la reponse existait ailleurs dans le depot avant de la re-mesurer.

## ETAPE 1 — LA TABLE, ET LES QUATRE JUGES QU'ELLE PASSE

Juges ecrits avant la mesure. `M1` couverture >= 90 % (evenements ET identifiants distincts) ·
`M2` zero chevauchement entre l'espace des identifiants d'equipement et celui des armes, dans
les deux sens · `M3` concordance avec les deux etiquettes que le lot 4 avait acquises par deux
voies independantes · `M4` gain sur la voie delta i48 (19,5 % / 25,0 %).

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| ramassages non-arme | 82 | 36 |
| **resolus par le manifeste** | **82 / 82 = 100 %** | **36 / 36 = 100 %** |
| identifiants distincts resolus | **8 / 8** | **8 / 8** |
| ids de classe ARME presents dans le manifeste d'equipement | **0** | **0** |
| ids de classe NON-ARME presents dans le catalogue d'armes | **0** | **0** |
| concordance avec les etiquettes acquises | **2 / 2** | **2 / 2** |

**Verdicts : M1 TENU · M2 TENU · M3 TENU (2/2) · M4 TENU (100 % contre 19,5-25,0 %).**

### La table, telle que les deux films la remplissent

Les MEMES huit identifiants sur les deux films — ce qui confirme au passage, par une troisieme
voie, que le `R(32)` est bien un identifiant de CATALOGUE et non un handle de partie.

| identifiant | famille | provenance (chaine du jeu) | 000d5950 | 00502e52 | classe |
|---|---|---|---|---|---|
| `0f5716ff` | `grenade_spike` | `gggl_entree` | 21 | 2 | 2 |
| `caaadcb0` | `grenade_plasma` | `gggl_entree` | 16 | 7 | 2 |
| `8c77ffe7` | `grapple` | `sofa_modele` | 13 | 13 | 3 |
| `eef5d48d` | `thruster` | `sofa_modele` | 10 | 3 | 3 |
| `bcabbe43` | `grenade_frag` | `gggl_entree` | 8 | 4 | 2 |
| `aada07f3` | `grenade_dynamo` | `gggl_entree` | 6 | 3 | 2 |
| `72199cba` | `sensor` | `sofa_modele` | 5 | 2 | 3 |
| `8e2dc574` | `wall` | `sofa_modele` | 3 | 2 | 3 |

**M2 merite qu'on dise ce qu'il vaut et ce qu'il ne vaut pas.** Un manifeste de 21 entrees ne
peut pas couvrir 100 % d'un ensemble par hasard — il n'y a pas de temoin a construire la-dessus.
Le seul risque reel etait que les deux espaces d'identifiants soient en fait CONFONDUS, auquel
cas la table serait ambigue. Les deux inclusions croisees sont nulles : ils sont disjoints.

**Ce que la table apporte que la palette ne pouvait pas.** La palette `famille_b` de
`replay_labels.toml` ne nomme que les rangs 20 et 21 ; les rangs 19 et 22 y sont muets. C'est
exactement pour eux que le lot 4 rendait « rang 19 » sans nom. La chaine du jeu les nomme :
`8e2dc574` = **mur** (`wall`), `72199cba` = **capteur** (`sensor`).

## ETAPE 2 — CLASSE 2 = GRENADES, CLASSE 3 = EQUIPEMENT. TRANCHE.

Une fois les identifiants nommes, la question de la classe n'est plus une correlation mais un
tableau de contingence pur, ou la nature vient des FICHIERS DU JEU et ne depend d'aucune fenetre
temporelle. Le classement grenade / non-grenade derive du seul prefixe `grenade_` du manifeste
— une convention du manifeste (les quatre entrees `gggl_entree`, la liste des grenades du jeu),
pas une interpretation de ce lot.

| | 000d5950 | 00502e52 |
|---|---|---|
| classe 2 : grenade / autre / non resolu | 51 / 0 / 0 -> **100,0 % grenade** | 16 / 0 / 0 -> **100,0 %** |
| classe 3 : grenade / autre / non resolu | 0 / 31 / 0 -> **0,0 % grenade** | 0 / 20 / 0 -> **0,0 %** |
| identifiants repartis sur DEUX classes | **0** | **0** |

**Verdicts : C1 TENU (100,0 % contre 0,0 %) · C2 TENU · C3 TENU (purete parfaite par
identifiant).**

**La chaine qui tranche est celle du jeu, pas celle du film**, et c'est ce qu'il faut retenir :
le lot 4 avait deux juges pris DANS le film (rang i48, compteur de grenades i22), tous deux
tributaires de la densite des emissions ; l'un designait la classe 3 comme equipement a 40-45 %
contre temoin 0,0 %, l'autre ne tranchait pas la classe 2. Le nom, lui, tranche a 100/0.

**Ce que le lot 4 avait vu et n'avait pas su prouver est confirme** : le volet B avait observe
que `bcabbe43` et `caaadcb0` recevaient des etiquettes GRENADE par la voie des images-cles et
les avait publies comme « signal qualitatif, PAS une mesure ». Le manifeste les nomme
`grenade_frag` et `grenade_plasma`, et le rang d'etiquette (0 puis 1) est l'ordre exact des
entrees du `gggl` du jeu. L'observation etait juste.

## ADDENDUM A — « NOS GRAMMAIRES SONT-ELLES COMPLETES ? » : LE DOUTE EST REFUTE

Question posee : « je trouve etrange que le film contienne une grammaire d'armes et pas
d'equipement ». Elle a ete instruite comme une hypothese de plein rang, avec le REGISTRE DE
REPLICATION de `chunk_00` comme oracle de completude — le film porte son propre inventaire
(49 archetypes porteurs, 1 067 couples archetype x composant, confirme corpus entier,
`NOTE_COMPTE_REGISTRE_2026-08-30`).

**L'oracle de « decode » n'est pas la table TSV mais le dispatcheur lui-meme** : l'instrument
appelle `consumeByName`, dont la branche `default` rend `ported = false`. C'est litteralement la
definition de « le decodeur ne connait pas ce composant ». Controle negatif joue AVANT la
mesure : un nom de composant invente est bien refuse.

| archetype | role | declares | consommes | feuille d'identite |
|---|---|---|---|---|
| ti=42 | arme au sol — LA REFERENCE | 21 | **21 (100 %)** | **oui** |
| ti=37 | equipement / objet — LE SUSPECT | 31 | **31 (100 %)** | **oui** |
| ti=35 | bipede porteur | 64 | 63 (98,4 %) | oui |

**Verdicts : A1 TENU (zero composant declare non consomme sur ti=37) · A2 TENU · A3 TENU
(controle negatif) · A4 : pas d'asymetrie.** Identique sur les deux films.

**Le point decisif.** La feuille d'identite d'un objet du monde est
`object-multiplayer-properties-component`, et c'est **LE MEME composant** qui identifie une arme
au sol sur ti=42 et un objet d'equipement sur ti=37. Il est declare des deux cotes, consomme des
deux cotes, et EXPLOITE des deux cotes — c'est de lui que `equipment_creation.go` tire le
GlobalID `eqip` qui vient de nommer 100 % des ramassages. L'asymetrie soupconnee n'existe pas au
niveau de l'archetype.

Le seul refus de ti=35 est `simulation-state-component` (i=60), sans rapport avec l'equipement,
et deja consigne `partiel` dans la table ECS versionnee.

**Seconde passe — le vocabulaire.** Tous les composants du registre dont le nom cite
`equipment`, `grenade`, `inventory`, `pickup`, `ammo` ou `item-` : 23 noms distincts, 98 couples.
Les 67 refus sont tous **hors domaine** : `supply-lines-item-unlocked-component` x64 sur
l'archetype 27, `managed-radialmenu-item-count`, `managed-objective-is-only-one-item-unlocked`,
`vehicle-equipment-turret-parent` sur ti=40 (aucun vehicule dans ce corpus). La racine `item-`
sur-capture ; le negatif est publie tel quel plutot que corrige apres coup en retirant la racine.

**Les trois garde-fous du depot rejoues sur les deux films** : G1 (code <-> table), G2
(film <-> table : 1 067 lignes de registre confrontees a 1 067 lignes de table), G3
(table <-> document). Tous verts.

## ADDENDUM B — LA PALETTE N'EST PAS DANS LA SECTION 3 : NEGATIF AVEC TEMOINS

Hypothese : la table rang -> objet, propre a la variante de mode, pourrait vivre dans la
section 3 de `chunk_00` (~537 ko propres au match, jamais ouverte). Scan **cible**, pas une
exploration : les 21 GlobalID `eqip` du manifeste, cherches comme mots de 32 bits alignes sur
l'octet, en little-endian ET en big-endian.

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| section 3 | `[0x0CB65C, 0x14E8A0)` = 537 156 o | `[0x0CB65C, 0x14E31E)` = 535 746 o |
| **B-POS** chaines UTF-16LE lisibles | `whiteknight2519` @ **0x13CCA6**, `PEINX13` | `LordFlacco9189`, `superjp12345` |
| **B-NEG** plancher du hasard | 0,00050 occ. / valeur (2 000 tirages) | idem |
| identifiants d'EQUIPEMENT trouves | **0 / 21** | **0 / 21** |
| **B-REF** familles d'ARME du film trouvees | **0 / 15** | **0 / 15** |

**VERDICT B1 : NON TENU. Negatif.**

Les trois temoins sont ce qui rend ce negatif lisible :

- **B-POS** est le controle qui autorise a conclure. L'instrument retrouve `whiteknight2519` a
  `0x13CCA6` — **exactement** le decalage que la carte de `chunk_00` documente par une autre
  voie. Il lit donc les bons octets. (`LORD PEINX13` sort tronque en `PEINX13` : mon filtre
  exclut l'espace, c'est la meme chaine.)
- **B-REF** est le controle qui interdit de sur-interpreter. Les familles d'arme reellement
  jouees par le film n'y sont pas non plus. **L'absence ne vise donc pas l'equipement** : elle
  dit que la section 3 n'est pas un catalogue d'objets aligne sur l'octet. C'est coherent avec
  ce que la carte de `chunk_00` en disait (flux bit-packe, aucun pas de structure).

La section 3 reste au chantier trame. Aucun chantier d'exploration n'est ouvert ici.

## ETAPE 3 — L'ORIGINE : LE JUGE TEMPOREL DESAMBIGUISE, IL NE TRANCHE PAS

**La reserve, ecrite avant la mesure parce qu'elle peut tout expliquer.** `equipment_placements.go`
etablit — mesure, pas suppose — que **la disparition d'un objet n'est PAS dans le film** (trois
pistes de fin explicite instrumentees, trois echecs). Ce que le decodage rend comme `tEnd` est
l'instant ou l'objet CESSE DE BOUGER, prolonge par les recensements d'images-cles : une BORNE
INFERIEURE de la duree de vie. Le juge temporel de ce lot est donc bati sur une quantite dont le
depot sait deja qu'elle n'est pas une fin.

**L'apport neuf du lot 5 est la SEPARATION DE LA POPULATION.** Le lot 4 mesurait les 82
ramassages non-arme en bloc ; l'etape 1 les nomme, donc on peut enfin separer les 51 grenades
des 31 equipements. Les vies ti=37 de grenade sont des grenades LANCEES, qui explosent et ne se
ramassent pas — les melanger etait une facon sure de noyer le signal.

000d5950 / Cliffhanger (mono-film : la carte de `00502e52` reste inconnue, cf. lot 4). Fenetre
500 ms, rayon 3 m — large a dessein, le but est de desambiguiser dans un rayon genereux, pas de
faire le tri par le rayon.

| population | juge | UN SEUL candidat | plusieurs | aucun |
|---|---|---|---|---|
| classe 2 (grenades), n=51 | spatial (lot 4) | 27,5 % | 43,1 % | 29,4 % |
| classe 2 (grenades), n=51 | **temporel** | 25,5 % | **23,5 %** | 51,0 % |
| classe 3 (equipement), n=31 | spatial (lot 4) | 12,9 % | 41,9 % | 45,2 % |
| classe 3 (equipement), n=31 | **temporel** | **25,8 %** | **16,1 %** | 58,1 % |
| toutes, n=82 | spatial | 22,0 % | 42,7 % | 35,4 % |
| toutes, n=82 | **temporel** | 25,6 % | **20,7 %** | 53,7 % |
| TEMOIN +37 s (n=17) | temporel | **0,0 %** | 0,0 % | 100 % |
| TEMOIN -53 s (n=10) | temporel | **0,0 %** | 0,0 % | 100 % |

**Verdicts : O1 (>= 50 % injectif, temoin < 15 %) NON TENU — 25,6 % · O2 (le juge temporel
desambiguise) TENU — l'ambiguite tombe de 42,7 % a 20,7 %, et de 2,6x sur la classe
equipement (41,9 % -> 16,1 %).**

**Lecture honnete.** Le second juge fait ce qu'on lui demandait — il coupe l'ambiguite de
moitie, et il DOUBLE la part injective sur la classe equipement (12,9 % -> 25,8 %), ce qui
valide la separation de population. Mais il ne monte pas la couverture : les 53,7 % « aucun
candidat » sont exactement ce que la reserve annoncait, un `tEnd` qui n'est pas une fin. **La
refutation du lien objet-au-sol n'est ni levee ni confirmee ; ce lot la precise une seconde
fois : l'ambiguite est reductible, la couverture ne l'est pas par ce chemin.**

**Reserve sur les temoins** : leurs denominateurs sont faibles (17 et 10 sur 82) parce que
l'instant decale n'a souvent aucun echantillon de position a moins de 300 ms. Un temoin a 0,0 %
sur n=17 est un plancher, pas une demonstration.

### O3 — le point d'apparition de la carte : NON TESTABLE, faute de donnees

**Le depot ne declare AUCUN point d'apparition d'equipement ni de grenade.**
`map_weapon_pads.json` ne connait que trois familles d'emplacement : `power` (arme de pouvoir),
`rack` (arme de rateliers) et `powerup`. `map_objectives.json` ne porte que des roles
d'objectif (`flag_spawn`, `hill`, `landgrab_zone`, ...). La branche « la prise vient-elle d'un
point d'apparition ? » ne peut donc pas etre testee en propre.

Ce qui a pu etre mesure, et qui ne prouve rien : Cliffhanger (`cliffhanger_ridgeline`) porte
**UN** socle de famille `powerup`. Ramassages non-arme a moins de 3 m : **3 / 82 = 3,7 %**,
temoin decale 1 / 82 = 1,2 %. Non informatif, et c'est un manque de donnees, pas un resultat.

La resolution de carte du catalogue de socles exige une correspondance **unique** (nom public ou
module) et s'abstient sinon : choisir parmi plusieurs candidats serait exactement l'ajustement
que le lot 4 a mesure comme destructeur (une mauvaise carte fait tomber le rapport de 7,2x a
1,8x).

## CE QUI DEVIENT PUBLIABLE, ET CE QUI NE L'EST PAS

- **PUBLIABLE, et c'est le gain du lot** : le NOM et la NATURE de chaque ramassage non-arme.
  La jointure `pickups[].w` -> `[[equipment_objects]]` couvre 100 % des non-armes des deux films
  de reference, sans collision, et concorde 2/2 avec les etiquettes acquises par correlation.
  La publication produit est un lot separe ; deux points l'attendent, tous deux deja consignes :
  la table est indexee sur la forme `0x` + MAJUSCULES du manifeste tandis que `pickups[].w`
  s'ecrit en `%08x` (meme piege de format que le P0 de la ronde 1 du lot 3 — la normalisation se
  fait AU POINT DE JOINTURE), et le manifeste ne porte que la FAMILLE, pas de libelle FR/EN.
- **ACQUIS COMME CONNAISSANCE** : classe 2 = grenades, classe 3 = equipement, a 100/0 sur deux
  films. L'enonce initial du lot 4 etait inverse ; le lot 4 l'avait corrige a moitie (classe 3
  etablie, classe 2 non conclue), ce lot le ferme.
- **REFUTE** : le doute « nos grammaires sont incompletes cote equipement ». ti=37 est consomme
  a 31/31, sa feuille d'identite est la MEME que celle de ti=42, et elle est exploitee.
- **NEGATIF CONSIGNE** : la section 3 de `chunk_00` ne porte pas de catalogue d'objets aligne
  sur l'octet — ni equipement, ni armes.
- **NON PUBLIABLE** : l'ORIGINE d'une prise. Ni le sol (25,6 % d'injectivite) ni le point
  d'apparition (donnees inexistantes).

## Ce qu'il faudrait pour aller plus loin

1. **Le manifeste ne connait que 21 objets** — ceux du corpus qui a servi a le batir. Les huit
   identifiants des deux films de reference y sont tous, mais un film de famille A (rangs 1-12),
   un mode BTB ou un vehicule pourraient en sortir un 22e. Le garde-rail de parite bilaterale
   (`replaylabels/catalog_test.go`, `TestPariteObjetsEquipementDuCorpus`) le verrait ; la
   couverture 100 % de ce lot vaut pour DEUX films, pas pour le corpus.
2. Pour l'origine : le seul chemin restant est une VRAIE fin d'objet dans le film. Trois pistes
   ont deja echoue ; la quatrieme serait l'evenement natif de destruction, s'il existe, a
   chercher dans le registre des 50 blocs du chantier trame.
3. Un catalogue de points d'apparition d'equipement extrait des `.mvar`, sur le modele de
   `cmd/mapopads-build` — c'est la donnee qui manque a O3, et elle est extractible.

## Reproduire

```
cd apps/go-api
BIPED_PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/replay/ -run 'EquipmentPickupManifestNaming|EquipmentPickupClassByManifest' -v
BIPED_PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/filmdec/ -run 'AuditGrammaireEquipement|AuditRegistreComposants|PaletteEquipementDansSection3' -v
ECS_TABLE_FILM='<film1>;<film2>' \
  go test ./internal/analysis/filmdec/ -run 'TestG1TableSuitLeCode|TestG2TableSuitLeRegistreDuFilm|TestG3TableSuitLeDocument' -v
PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 PICKUP_MAP=Cliffhanger \
  go test ./internal/analysis/replay/ -run 'TestEquipmentPickupOrigin$' -v
```

Un film par process, lecture seule, verrou `LockProcessDecode`, aucune cuisson d'artefact.
