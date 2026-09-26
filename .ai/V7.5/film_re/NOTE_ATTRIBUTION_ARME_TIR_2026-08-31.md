# NOTE — Attribution PAR LE TIR de la precision et de la distance par arme (film Theater)

Date : 2026-08-31. Branche `wt/trame-attrib`. Instrument :
`apps/go-api/internal/analysis/filmdec/lot1_attrib_arme_tir_research_test.go`
(garde `LOT1_TRAME_FILM`, un film par process, borne a `deltaWitnessChunks` = 12 chunks).
Films : `000d5950`, `01e1f945`, `00502e52`.

## Cadrage (correction d'une sonde precedente)

La sonde `lot1_sonde_precision_*` keyait la precision/distance par le TAG SOURCE du degat
(`damage_aftermath`) — mauvais lien. Le tag source est le projectile/effet (enfant de l'arme),
dans un AUTRE espace d'id : il ne joint pas l'arme sans une table (confirme ici, M5 : 0 %
d'intersection avec le WeaponID sur les 3 films).

L'arme est deja connue PAR LE TIR : `action_weapon_fire` (0xD2, type 36, variante longue) porte
le `WeaponID` 64 bits a des offsets fixes (`fire_events.decodeFireEvent`, prouve). On attribue
donc TOUT par le tir, via le pont de `lot1_modal_touche` :

- ATTAQUANT du tir = ref0 domaine 1 (lu par `lot1RefDom1`).
- RESPONSABLE du degat = ref1 domaine 1 de `damage_aftermath` (0xC0 type 0), MEME espace brut.
- Appariement : meme index d'attaquant + `|ts_degat - ts_tir| <= W`. Sans base.
- BLESSE = ref0 du degat apparie -> slot (base 512) -> position ; distance tireur<->victime.

Seuils ecrits AVANT : `W` = 250 ms (comme modal_touche) ; `OFF` = 3 s (temoin decale) ;
`tol` position = 120 ms. Verdict au RATIO temoin, pas au taux absolu.

## Mesures chiffrees (par film, 12 chunks)

### M1 — Fiabilite du lien tir<->degat (le pont est-il reel ?)

| film | tirs | degats | AVANT tir->degat ±250ms | temoin +3s | ratio | ARRIERE degat->tir | temoin | ratio | verdict |
|---|---|---|---|---|---|---|---|---|---|
| 000d5950 | 245 | 190 | 16.7 % | 0.8 % | **16.7x** | 18.8 % | 2.3 % | **8.0x** | TENU |
| 01e1f945 | 867 | 44 | 4.3 % | 1.5 % | **2.8x** | 45.9 % | 10.8 % | **4.2x** | TENU |
| 00502e52 | 276 | 150 | 17.4 % | 4.7 % | **3.7x** | 25.3 % | 4.7 % | **5.4x** | TENU |

Le pont attaquant-dom1 est REEL sur les 3 films (co-incidence tres au-dessus du hasard, dans
les deux sens). Le MECANISME d'attribution par le tir fonctionne.

### M1bis — Sweep de fenetre (le degat est horodate a l'IMPACT, pas au tir)

Le taux AVANT (tir qui trouve un degat) a 250 ms est un PLANCHER : elargir W le fait monter
bien plus vite que le temoin — le degat arrive avec un DELAI apres le tir (vol du projectile,
attribution du tir soutenu, groupage du record).

| film | 250ms | 500ms | 1000ms | 2000ms | temoin@2s |
|---|---|---|---|---|---|
| 000d5950 | 16.7 % | 29.0 % | 40.8 % | 51.4 % | 18.0 % |
| 01e1f945 | 4.3 % | 7.6 % | 12.1 % | 19.1 % | 17.5 % |
| 00502e52 | 17.4 % | 28.6 % | 39.9 % | 40.6 % | 17.0 % |

A ~1 s le taux double-triple avec un ratio au temoin encore fort (~4-8x). A 2 s le temoin
rejoint (fond de tir dense) : ~1 s est le point de fonctionnement. **Le 250 ms pre-enregistre
etait trop serre.**

### M2 — Precision par arme (touches / tirs, cle WeaponID du tir)

Colonne « 2s » = meme mesure a fenetre 2 s (revele les impacts differes).

000d5950 : Disruptor 29 %->94 % · Needler 40 %->100 % · BR75 33 %->52 % · S7 Sniper 19 %->71 % ·
Pulse Carbine 0 %->76 % · **M41 SPNKr 0 %->0 %** · **Ravager 0 %->6 %** · **Mangler 0 %->0 %** ·
**Stalker 0 %->0 %** · **Shock Rifle 0 %->0 %** · **Bulldog 0 %->0 %** · **Hydra 0 %->0 %**.

00502e52 : VK78 Commando 46 %->100 % · BR75 38 %->76 % · Disruptor 22 %->75 % · Heatwave 21 %->71 % ·
Needler 21 %->46 % · Sidekick 9 %->13 % · S7 Sniper 8 %->17 % · **Shock Rifle 0 %->0 %** ·
**Bulldog 0 %->0 %** · **Skewer 0 %->0 %** · **M41 SPNKr 0 %->0 %** · **Stalker 0 %->0 %**.

01e1f945 : MA40 AR 3 %->19 % · Sidekick 7 %->21 % · **S7 Sniper 0 %->0 %** · **Fuel Rod 0 %->9 %**.

**Deux categories nettes parmi les « 0 % a 250 ms » :**
1. RECUPERABLES en elargissant la fenetre — armes a balle/projectile/precision (Disruptor,
   Needler, Pulse Carbine, BR75, VK78, Heatwave, MA40, S7 Sniper, Sidekick). Leur precision EST
   capturable ; elle atteint des magnitudes plausibles (46-100 %) a 2 s.
2. NON EMISES meme a 2 s — classe lourde/speciale (M41 SPNKr, Hydra, Skewer, Ravager, Shock
   Rifle, Mangler, Stalker, Bulldog). Leur degat ne passe PAS par `damage_aftermath` (type 0) —
   probablement `damage_section_response` (type 1) ou un autre record (explosif/AoE/faisceau).

### M3 — Distance tireur<->victime par arme (m)

Positions au ts du degat, base 512, resolvabilite quasi totale quand les bornes de carte sont
connues (000d5950 : 37/41 ; 00502e52 : 48/48). Film 01e1f945 : distances DESACTIVEES (signature
de largeurs `[15 15 15]` ambigue catalyst/deadlock — limite outil, `LOT1_SONDE_MAP` la leve).

| arme | 000d5950 (cliffhanger) | 00502e52 (bazaar) |
|---|---|---|
| Needler | med 4.5 m (n=13) | med 8.1 m (n=14) |
| Disruptor | med 6.4 m (n=14) | med 10.9 m (n=7) |
| VK78 Commando | — | med 10.9 m (n=11) |
| BR75 | med 10.8 m (n=9) | med 16.2 m (n=8) |
| Sidekick | — | med 13.5 m (n=4) |

Ordonnancement PHYSIQUEMENT SENSE et STABLE entre films : Needler = le plus court, BR75 = parmi
les plus longs. La geometrie capturee est propre. Reserve : aucune distance de sniper capturee
(effectif < 4), le point « sniper mesure long » reste non confirme.

### M4 — Degats sans refs d'en-tete

Sur les 3 films, dans la fenetre bornee : **0 degat totalement sans refs.** Chaque
`damage_aftermath` porte au moins une ref d'en-tete ; deux refs presentes : 67 % (000d5950),
84 % (01e1f945), 100 % (00502e52). Categorie « une ref » : 33 % / 16 % / 0 %.

Consequence : dans ces fenetres, l'exclusion de degats NON-ARME (chute/environnement/DoT) N'EST
PAS un biais — il n'y a quasi rien a exclure. Les sources anonymes anticipees par le cadrage
(`0x00d9dbf765`/`95`) N'apparaissent PAS comme classe sans-refs ici (elles etaient probablement
vues comme tags source sur des degats a refs, pas comme un record anonyme distinct). Le vrai
manque de couverture n'est donc pas l'anonymat des degats, mais la NON-EMISSION par classe
d'arme (M2 cat. 2). Reserve mineure : un degat a « une ref » manquant la ref1 (responsable) ne
peut pas se lier a un tir -> leger sous-comptage cote arriere/precision.

### M5 — Hypothese parent/enfant (secondaire)

Le tag source du degat n'intersecte JAMAIS le WeaponID (moitie basse ni haute) : 0 % sur les 3
films. Le tag source est bien un id d'un AUTRE espace (projectile/effet). Non resoluble vers
l'arme sans une table dediee -> VALIDE le choix d'attribuer par le tir, pas par la source.

## Verdict de viabilite (honnete)

**Le mecanisme d'attribution PAR LE TIR est correct et le lien tir<->degat est reel** (M1, ratio
temoin 2.8-16.7x, TENU partout).

**Precision par arme — viable SOUS CONDITIONS, pas universelle :**
- La fenetre de 250 ms sous-compte massivement : le degat est horodate a l'impact, differe du
  tir. A ~1 s (ratio temoin encore ~4-8x) la capture atteint des niveaux plausibles pour les
  armes a balle/projectile (~2/3 de l'arsenal teste).
- MAIS une classe entiere (explosifs/roquettes/faisceaux/lourds : SPNKr, Hydra, Skewer, Ravager,
  Shock Rifle, Mangler, Stalker, Bulldog) reste a ~0 % meme a 2 s -> son degat n'est PAS dans
  `damage_aftermath`. Pour ces armes, la precision film serait fausse (0/sous-comptee).
- Les taux ABSOLUS restent des captures, a RECALER sur le total API (le film ne voit qu'une
  partie des degats). La FORME relative n'est fiable qu'A L'INTERIEUR de la classe « balle », pas
  entre classes.

**Distance par arme — plus robuste que la precision** (une seule resolution suffit a esquisser
une distribution). Ou l'arme est capturee, l'ordonnancement est physiquement sense et stable
entre films. Herite des memes trous de couverture (armes non emises = pas de distance) et de la
limite d'outil sur les cartes a signature ambigue.

**Reserves reelles a porter dans toute feature :**
1. Fenetre d'impact : fixer W a ~1 s (pas 250 ms) et documenter le ratio au temoin.
2. Trou de couverture par classe : `damage_aftermath` (type 0) ne couvre pas les explosifs/
   faisceaux/lourds. Avant d'annoncer « precision par arme », percer le record de la classe 2
   (probable type 1 `damage_section_response`) ou restreindre la feature aux armes capturees.
3. Echantillon film-seul : recaler les taux sur la precision GLOBALE de l'API par arme ; le film
   n'apporte que la FORME (par distance, et entre armes de la meme classe).
4. Cartes a signature ambigue : renseigner `LOT1_SONDE_MAP` pour les distances (catalyst/deadlock).

## Prochaine etape suggeree (hors perimetre de cette sonde)

Percer `damage_section_response` (0xC0 type 1) pour recuperer les degats de la classe lourde,
puis re-mesurer la precision par arme a W ~1 s sur un corpus plus large que 12 chunks.
