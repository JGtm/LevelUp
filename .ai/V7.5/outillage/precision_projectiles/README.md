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
