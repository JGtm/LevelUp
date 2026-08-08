# Outillage de recherche — precision par arme des armes a projectile (piste E)

ARCHIVE. Ce code n'est ni compile ni importe par l'application : `apps/go-api/cmd/tmp_*/` est
gitignore par convention du depot (`.gitignore:311`), et cette copie existe pour que la session
du 2026-08-08 soit rejouable. Verdict et chiffres :
`.ai/V7.5/VERDICT_PRECISION_PROJECTILES.md`.

## Restaurer

    cp -r .ai/V7.5/outillage/precision_projectiles/tmp_pjcnt apps/go-api/cmd/
    cd apps/go-api && CGO_ENABLED=0 go build ./cmd/tmp_pjcnt

**CGO n'est PAS requis** : l'outil ne touche pas DuckDB. Les deux references (API et
referentiel d'armes) entrent par CSV, exportes une fois par le CLI `duckdb` en READ_ONLY.

## Les deux CSV d'entree

```sql
-- ref.csv : la reference API, restreinte aux films du cache
ATTACH 'data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS sh (READ_ONLY);
COPY (SELECT substr(r.match_id,1,8) AS pfx, r.match_id, r.pair_name,
             p.xuid, p.gamertag, p.shots_fired, p.shots_hit, p.kills, p.deaths,
             p.team_id, p.outcome
      FROM sh.match_registry r
      JOIN sh.match_participants p ON p.match_id = r.match_id) TO 'ref.csv' (HEADER);

-- weapons.csv : le referentiel de noms
ATTACH 'data/titles/halo_infinite/warehouse/metadata.duckdb' AS md (READ_ONLY);
COPY (SELECT weapon_id, name_en FROM md.weapon_labels) TO 'weapons.csv' (HEADER);
```

`ref.csv` peut porter tout le corpus : l'outil ne traite que les prefixes presents A LA FOIS
sur disque et dans la reference, tries, plafonnes par `-limit`.

## Les quatre mesures

| drapeau | ce qu'il mesure | a quoi il sert |
|---|---|---|
| `-align` | `P(pas = +1)` du compteur 7 bits a -1 / 0 / +1 bit, plus l'histogramme des pas | **GATE DE REPRODUCTION** — a jouer AVANT toute conclusion. Profil attendu (7ter.80) : effondrement des deux cotes |
| `-hdr` | ventilation des records type 105 par classe d'en-tete (bits 8..11) et par moitie basse d'identifiant d'arme | delimiter la population de TIR : les records hors suffixe `42c9679f` ne portent pas de compteur de tir |
| `-arme` | par arme : taux de porteur, pas moyen du compteur, **cadence inter-record** (mediane, q90) | le test de la piste E. La cadence est ce qui tranche « le record est-il un tir ou une touche ? » |
| `-fit` | deconvolution du taux de touche par arme contre les touches API du match, moitiés A/B et nulle permutee | la voie neuve. **Toujours avec `-norm`** : sans normalisation de visibilite les coefficients sont inintelligibles |

```bash
tmp_pjcnt -films <cache> -ref ref.csv -famille STANDARD -limit 12 -align
tmp_pjcnt -films <cache> -ref ref.csv -famille FIESTA   -limit 30 -hdr
tmp_pjcnt -films <cache> -ref ref.csv -armes weapons.csv -famille FIESTA -limit 30 -arme
tmp_pjcnt -films <cache> -ref ref.csv -armes weapons.csv -limit 900 -fit -norm -minshots 5000
tmp_pjcnt -films <cache> -ref ref.csv -famille TACTICAL -limit 22 -out m.csv
```

`-famille` classe par `pair_name` : `FIESTA` / `TACTICAL` / `BTB` / `STANDARD`.

## Trois choses a savoir avant de le relancer

1. **`-limit` n'est pas decoratif.** Le balayage non borne du corpus est une bombe RAM
   documentee. 900 films coutent 45 s et tiennent en memoire parce qu'un seul chunk est
   decompresse a la fois — ne pas remonter les records de tous les films dans une seule
   tranche.
2. **La population de tir est filtree** (`isFire`) : variante longue et moitie basse
   d'identifiant d'arme egale a `42c9679f`. C'est une restriction assumee, pas une correction —
   quelques armes reelles (MA5K) ont une autre moitie basse et sortent de la mesure. Les
   records ecartes portent `P(pas = 0) = 0,63` en mode standard : leur champ 26..32 n'est pas
   un compteur de tir.
3. **Le cout est de 50 ms par film**, deux ordres de grandeur sous le balayage bit a bit,
   parce qu'un record se reconnait a son premier octet (`pay[0]>>1 == 105`) et non par un
   marqueur cherche au bit pres.

## Ajout du 2026-08-08 (session 2) — le grain JOUEUR

| drapeau | ce qu'il mesure |
|---|---|
| `-pigate` | **GATE OBLIGATOIRE AVANT `-joueur`** : confronte le resolveur rapide `resolvePIFast` a celui du depot (`weaponv3.ResolveXuidToPI`), chunk par chunk. Attendu : `desaccord=0`. Il a deja rattrape une erreur reelle (277 accords contre 299 desaccords sur la premiere version) |
| `-joueur` | etape A : reference API par arme sur la population a arme dominante (`-purete`, defaut 0.8) + deconvolution BORNEE dans [0,1] au grain (match x joueur), moities hors echantillon. **Toujours avec `-norm`** |

```bash
tmp_pjcnt -films <cache> -ref ref.csv -limit 3 -pigate           # gate du resolveur
tmp_pjcnt -films <cache> -ref ref.csv -armes weapons.csv -limit 900 -joueur -norm -minshots 8000
```

**Pourquoi `resolvePIFast` existe** : le resolveur du depot relit 64 bits a chaque position de
bit et pour chaque xuid — 60 films ne finissent pas en 10 minutes. La version rapide cherche les
8 octets du xuid (ecriture LITTLE-ENDIAN) par `bytes.Index` dans les 8 versions decalees du
chunk : 898 films en 3 min 10. **Piege, et il a ete pris au gate** : le depot retient la
premiere occurrence EN POSITION DE BIT ; balayer decalage par decalage retient la premiere de
CHAQUE decalage, ce qui n est pas la meme occurrence. Il faut le minimum sur les huit decalages.

**Cout mesure** : `-joueur -norm` sur 900 films = **3 min 10** (appariement compris).

---

## Ajout du 2026-08-08 (session 3) — les DUMPS et la BRANCHE OPAQUE d'i0

Ces deux outils ne servent plus la piste E (close) mais le travail de DECODEUR ouvert par
`.ai/V7.5/film_re/NOTE_I0_TI41_POSITION_PROJECTILE.md` §11.

### `tmp_precdump` — lire les tables de precision dumpees de la memoire

| drapeau | ce qu'il fait |
|---|---|
| `-widths <bin>` | decode `ce_prec_widths_1445cc9e0.bin` : triples de largeurs par niveau, comprimes par plages |
| `-ranges <bin>` | decode `ce_prec_ranges_14462cbe0.bin` : 768 AABB `float[6]`, comprimes par plages |
| `-verify <bin>` | **confronte la table a la forme fermee** `min(26, ceilLog2(min(ceil(40000/(2*step(L))), 2^22)))`, `step(L)=2^(16-L)/120` |
| `-derive` | inversion : quel pas rend quelle largeur, sur les plages `+-100` et `+-20000` |

```bash
tmp_precdump -verify .ai/V7.5/dumps/ce_prec_widths_1445cc9e0.bin
tmp_precdump -ranges .ai/V7.5/dumps/ce_prec_ranges_14462cbe0.bin
```

**Ce qu'il a etabli** : `-verify` rend **32 / 32 niveaux en accord**. La table dumpee n'est que
la loi precalculee — elle confirme le desassemblage, elle n'apporte rien. Et l'entree 0 des
plages EGALE le catalogue `.module` de production au flottant pres. Cout : instantane.

**Correction portee au dossier au passage** : `FUN_1406d310c` est `ceilLog2`, pas `bitLen`.
Les deux ne different que sur les puissances exactes de deux — et c'est ce cas qui decide des
niveaux 17 a 22.

### `tmp_i0hi` — la branche opaque de `object-position-component`

| drapeau | ce qu'il mesure |
|---|---|
| `-part` | part des records `ti=41` sur la branche opaque, et structure des vies (mixtes / purement opaques / encadrables) |
| `-controle` | **GATE OBLIGATOIRE AVANT `-reg`** : la regression rejouee sur la branche BASSE, dont les champs sont connus. Attendu : retrouver `off 3/w 13`, `off 16/w 13`, `off 29/w 14` avec des etendues implicites egales a l'AABB |
| `-reg` | la regression sur la branche opaque, en deux regimes (coordonnee MONDE = plage fixe ; coordonnee NORMALISEE = largeurs de la carte) |
| `-cartes <csv>` | mutualise plusieurs cartes : chaque film prend les bornes de SA carte. Un film sans bornes est ECARTE, jamais decode avec celles d'une autre |

```bash
tmp_i0hi -films <cache> -catalogue data/titles/halo_infinite/reference/map_quant_bounds.json \
         -carte cliffhanger -only <ids> -part -controle
tmp_i0hi -films <cache> -catalogue <json> -cartes film_maps.csv -limit 300 -part -reg
```

Le CSV `-cartes` s'exporte en une requete :

```sql
ATTACH 'data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS s (READ_ONLY);
SELECT substr(match_id,1,8) || ',' || lower(map_name) FROM s.match_registry WHERE map_name IS NOT NULL;
```

**Cout** : ~14 s par film (lecture + marche des chunks). 30 films = 7 min ; l'elagage par film
(`keepMixedLives`, actif des que `-cartes` est passe) borne la memoire — sans lui, 300 films
remontent ~250 Mo d'echantillons.

**LA GRAMMAIRE DE RECORD EST CELLE DE LA PRODUCTION**, reprise mot pour mot de
`filmdec/projectiles.go` (celle qui porte les 70 lancers de grenade sur 70). C'est ce qui
distingue cette mesure de `tmp_i0w`.

### ⚠ `tmp_i0w` — SA MESURE EST REFUTEE, ne pas la citer

`tmp_i0w` annonce (note §8.1) que les deux branches d'i0 sont empruntees **a parts comparables**
(264 contre 277). **C'est faux d'un facteur ~8.** Deux instruments independants disent 6,2 % :
`tmp_i0hi` sur 30 films (9 382 records opaques sur 152 535) et `calib.txt` du cache film, qui
porte `object-position-component:45` a une frequence modale de **0,98** — sans qu'on ait eu
besoin de decoder quoi que ce soit. Le balayage de `tmp_i0w` n'a pas la selectivite de la
grammaire de production : ses records a porte = 1 sont pour l'essentiel des faux positifs.

L'outil reste archive pour la tracabilite de la session 2 ; **ses chiffres ne valent pas**.
