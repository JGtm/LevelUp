# Référence — Weapon IDs canoniques + structure fire-event (acurtis, 2026-06-02)

> Source : acurtis (reverse-engineering films Halo Infinite), table mise à jour + structure du weapon event.
> Données de référence pour `metadata.weapon_labels` + le parser d'armes (`internal/analysis/weapon_*`).
> Cf. `.ai/RESEARCH_THEATER_RE.md` pour le contexte film-RE complet.

## Table des weapon IDs (64-bit)

| Arme | weapon_id (hex) | high32 (base) | low32 (variante) |
|------|-----------------|---------------|------------------|
| Bandit Evo | 0x6ACDC44D42C9679F | 6ACDC44D | 42C9679F |
| BR75 | 0x2B1824D542C9679F | 2B1824D5 | 42C9679F |
| Disruptor | 0x84BD29ED42C9679F | 84BD29ED | 42C9679F |
| Cindershot | 0x230447B142C9679F | 230447B1 | 42C9679F |
| CQS48 Bulldog | 0xB619D84A42C9679F | B619D84A | 42C9679F |
| Diminisher of Hope | 0x841AC5E5A730E49F | 841AC5E5 | A730E49F *(var. Gravity Hammer)* |
| Duelist Energy Sword | 0x4FF3937E8978AA7A | 4FF3937E | 8978AA7A *(var. Energy Sword)* |
| Elite Bloodblade | 0x4FF3937E1EC48C7A | 4FF3937E | 1EC48C7A *(var. Energy Sword)* |
| Energy Sword | 0x4FF3937E42C9679F | 4FF3937E | 42C9679F |
| Fuel Rod SPNKr | 0x9D6AAED242C9679F | 9D6AAED2 | 42C9679F |
| Gravity Hammer | 0x841AC5E542C9679F | 841AC5E5 | 42C9679F |
| Heatwave | 0x2AC9C2FF42C9679F | 2AC9C2FF | 42C9679F |
| Infected Energy Sword | 0x0C55765F7A9376A0 | 0C55765F | 7A9376A0 *(base distincte)* |
| MA40 AR | 0x48C19D2D42C9679F | 48C19D2D | 42C9679F |
| MA5K Avenger | 0xF5C335DFE7232C0B | F5C335DF | E7232C0B *(base distincte)* |
| Mangler | 0x80977BA542C9679F | 80977BA5 | 42C9679F |
| M392 Bandit | 0x2FB21C8742C9679F | 2FB21C87 | 42C9679F |
| M41 SPNKr | 0x71AB0A2C42C9679F | 71AB0A2C | 42C9679F |
| Mk51 Sidekick | 0xF408190F42C9679F | F408190F | 42C9679F |
| MLRS-2 Hydra | 0x767DB96D42C9679F | 767DB96D | 42C9679F |
| Mutilator | 0xD791556542C9679F | D7915565 | 42C9679F |
| Mythic Sandwich | 0xB7262CA1C8FB11D0 | B7262CA1 | C8FB11D0 *(var. Sandwich)* |
| Needler | 0xB533957E42C9679F | B533957E | 42C9679F |
| Plasma Pistol | 0xC354294642C9679F | C3542946 | 42C9679F |
| Pulse Carbine | 0x30484EA642C9679F | 30484EA6 | 42C9679F |
| Ravager | 0xC30D87C742C9679F | C30D87C7 | 42C9679F |
| Rushdown Hammer | 0x841AC5E5D8D07CA1 | 841AC5E5 | D8D07CA1 *(var. Gravity Hammer)* |
| S7 Sniper | 0x0A1992BC42C9679F | 0A1992BC | 42C9679F |
| Sandwich | 0x880FE0BC42C9679F | 880FE0BC | 42C9679F |
| Sentinel Beam | 0xA0955E9E42C9679F | A0955E9E | 42C9679F |
| Shock Rifle | 0x9387A8B942C9679F | 9387A8B9 | 42C9679F |
| Shock Rifle (Ranked) | 0x1A22FEE642C9679F | 1A22FEE6 | 42C9679F *(var. Shock Rifle, base distincte)* |
| Skewer | 0x0D20C46942C9679F | 0D20C469 | 42C9679F |
| Stalker Rifle | 0xDAF193C742C9679F | DAF193C7 | 42C9679F |
| Vestige Carbine | 0x3E07021742C9679F | 3E070217 | 42C9679F |
| VK78 Commando | 0xFD98554C42C9679F | FD98554C | 42C9679F |

## Structure des ids → canonicalisation (173 distincts → ~60 base)

- **MAJ acurtis (clé)** : *« Weapon variants (e.g., S7 Flexfire) seem to share the same value as the core weapon. »*
  → Les **variantes de gameplay courantes (Flexfire, Tactical…) partagent l'id EXACT de l'arme core** : elles ne
  créent AUCUN id distinct. Donc l'inflation à 173 NE vient PAS des variantes normales mais de : (1) les **upgrades
  Super Fiesta** (le mode introduit des armes améliorées qui ~doublent la liste — ids distincts), (2) les **variantes
  nommées spéciales** ci-dessous (suffixe ≠ 42C9679F), (3) le **bruit** (queue ≤2 kills = mauvais champ lu).
- **VÉRIFIÉ (workflows v3, 2026-06-02)** : l'inflation 173→~37 n'est PAS due aux variantes Super Fiesta — c'est du
  **bruit du parser FormulaA**. Les 137 ids non-mappés (suffixe 42c9679f) totalisent 1 210 kills (1%), dont **99.6%
  via `attribution_path='formula_a'`** (low/medium), high-32 aléatoires. Cause : `ScanFormulaA` (weapon_scanner.go:134-153)
  + `weapon_correlation.go:239` acceptent N'IMPORTE quels 8 octets précédant le suffixe `42c9679f` SANS valider le
  high-32 contre `WeaponBytesMap` (les suffixes non-communs SONT validés ligne 144-149, le commun NON). → FormulaA
  scrape des handles d'objets/instances voisins, pas des weapon-type ids. **Règle : identité = high-32 ; valider le
  high-32 contre un set d'armes connues même pour le suffixe commun ; les 137 ids → NULL/none honnête.** High-32 fold
  ne récupère qu'1 id réel (Sentinel Beam variant `a0955e9e2164b3cf`). Le seed 42 labels est exhaustif (0 vraie arme cachée).

## v3 — plan source-first (vérifié contre code + DB 109k lignes, 2026-06-02)

**Verdict signal** : PAS de champ direct d'arme-de-mort dans le film (le burst full-state de mort = loadout de la
VICTIME, pas la source : killer-weapon présent 6.4% unique, 0/82 sur un 2e match ; pas de composant ECS killed-by ;
pas de télémétrie MaelstromEvent). → la corrélation fire-event reste le seul signal d'arme ; la v3 la DURCIT + ajoute
melee/grenade depuis leurs marqueurs film.

**Découverte clé (code mort)** : `weapon_correlation.go:71-74` route IsMelee/IsGrenade→sentinel, mais
`backfill_weapons.go:352/421` dérive ces flags de `event_type LIKE '%melee%/%grenade%'` — or `event_type` est TOUJOURS
`'kill'` (132 056 lignes, zéro variante). Donc **tous les kills melee+grenade tombent en NULL** (bucket 14 287 NULL).
Les 2 926 lignes sentinelles en DB sont legacy.

**Plan priorisé** :
- **P1 (ROI max, vérifié)** — sourcer grenade+melee depuis les MARQUEURS FILM, pas event_type. Nouveau
  `analysis/weapon_melee_scanner.go` (décodeur §K-bis : ancre 0x34/0x35, type@+76 ∈{42,47,60}, weapon-id aux offsets
  {0x42:[88],0x47:[86],0x60:[101,103]}, pi=octet@+20 bits 0-4 — donne la VRAIE arme : épée/marteau/crosse-pistolet) +
  `analysis/weapon_grenade_scanner.go` (§C : marqueur 0x4c0c00, weapon@+24, allowlist 4 grenades, scan TOUS les chunks
  type-2). Consommés comme attributions HIGH keyées (pi,time_ms) AVANT la corrélation fire. Supprimer la dérivation
  event_type morte. → sort le slice melee+grenade du NULL.
- **P2 (levier vers 90%, à mesurer)** — timestamps µs-précis via l'en-tête de paquet 16 octets (§L), remplaçant le
  bucketing grossier par frame-marker (weapon_scanner.go TimestampEstimator). Vise les **6 321 lignes fire-event soft
  <2s** → high. **Taux de récupération NON vérifié = le lever dominant ; passe de confirmation sur 10-20 matchs requise
  avant de promettre 90%.**
- **P3 (vérifié)** — rejet du bruit à la source + canon high-32. `ScanFormulaA` : exiger high-32 ∈ armes connues même
  pour le suffixe commun. Fold high-32 (récupère Sentinel Beam variant). → 1 207 lignes junk → NULL ; 174→~37 armes.
- **P4 (vérifié mort)** — retirer `ReconcileAPIAggregates` du pipeline (jamais appelé ; reconciled_as 0/109k). Garder
  swap_detected en diagnostic. Optionnel : loadout-victime du burst de mort en filtre NÉGATIF.
- **P5** — re-backfill (`weapon_kills` DELETE-then-INSERT par match, sweep `--weapons` existant sur 1 141 matchs) ;
  ajouter `attribution_path ∈ {melee_event, grenade_marker}`.
- **À vérifier aussi (wiobqkqj1)** : `player_index` est PRÉSUMÉ = ordre DB (`ORDER BY team_id, rank`), non validé contre
  le pi du film ; notre RE suggère que l'hypothèse est fausse (fire-pi=2=…0022=team0/b36=1 ≠ team*4+b36). Un mauvais
  ordre mis-attribue silencieusement les 81% fire_event. → valider/corriger le mapping pi (risque correctness majeur).

**Estimation confidence** : 71.5% aujourd'hui → plancher conservateur ~81-82% (melee+grenade+canon) ; **~88-90%
ATTEIGNABLE si** le timing µs récupère le gros des fire-events soft — non garanti par les données seules.

**Branche cible** : `feat/weapon-attribution-v3`. Câblage en attente du GO utilisateur.
- **Suffixe `42C9679F` = arme standard.** Le high-32 identifie l'arme. La plupart des variantes retombent déjà ici.
- **Suffixe ≠ 42C9679F = variante.** Deux cas :
  - **Même high-32 que la base** → fold vers la base : `841AC5E5` (Gravity Hammer → Diminisher of Hope, Rushdown Hammer), `4FF3937E` (Energy Sword → Duelist, Elite Bloodblade). Variantes cosmétiques/stats du MÊME type d'arme.
  - **High-32 distinct** → arme/variante à part entière : Infected Energy Sword `0C55765F`, MA5K Avenger `F5C335DF`, Mythic Sandwich `B7262CA1`, Shock Rifle Ranked `1A22FEE6`. À mapper individuellement vers leur base narrative (Energy Sword, MA5K=AR-like, Sandwich, Shock Rifle).
- **Super Fiesta** : le mode introduit des **variantes améliorées qui ~DOUBLENT la liste** → la majorité des 138 ids non-mappés de 000d5950 (un match Super Fiesta) = ces variantes upgradées. Canonicaliser par high-32 (où partagé) ou les ajouter comme variantes étiquetées de leur base.
- Règle pratique pour « arme utilisée » (narratif) : `base_weapon = lookup(high32)`, sauf les bases distinctes ci-dessus à mapper explicitement. Le 64-bit complet conserve la variante si besoin.

## Structure du WEAPON (fire) EVENT — bits utiles (acurtis)

Après le weapon_id (64-bit), dans le fire event :
- **Compteur de tirs** : précis, incrémente **0→127 puis reset** (par arme).
- **2e bit après le weapon_id = burst-final** : `0` pour les tirs intermédiaires d'un burst, `1` pour le **dernier tir du burst** (BR75, Shock Rifle → séquence `0-0-1`).
- **3e bit après le weapon_id ≈ hit/miss** : `0`=hit, `1`=miss — **fiable à ~majorité mais pas 100%** ; acurtis confirme avec **un autre bit un peu plus loin**.
- **Sentinel Beam** : tir continu (pas de bursts discrets) → difficile ; weapon_id obtenu via l'investigation des melee events.
- **Tirs après la fin du match** : présents dans le film mais **PAS dans le payload de stats API** (`ShotsHit`/`ShotsFired`) → contribue aux écarts film vs stats lors de la vérification.

## Implication v3 (fiabiliser la source)

Le signal direct d'arme-de-kill = **le dernier tir TOUCHÉ (hit) avant la mort de la victime**, identifié via :
1. le weapon_id du fire event (canonicalisé high-32),
2. le bit **hit** (3e + bit de confirmation) → ne retenir que les tirs qui touchent,
3. le bit **burst-final** → le tir qui porte le coup fatal.

→ attribution **directe, haute confiance**, sans deviner/fallback. Reste à câbler : grenade (`0x4c0c00`) + melee (`~0xd340`) pour les kills sans tir. Médailles écartées (pool trop faible), aim reporté (besoin du point d'impact).
