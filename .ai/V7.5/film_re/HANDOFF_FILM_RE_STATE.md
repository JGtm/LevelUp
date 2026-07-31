# État des lieux — Reverse Engineering du format Film (Theater) Halo Infinite

> Document de **handoff** destiné à des reverse engineers externes (dend / acurtis / autres).
> Objectif : exposer (1) ce qui est DÉCODÉ et comment, (2) ce qui est BLOQUÉ et pourquoi,
> (3) les **questions précises** dont la réponse nous débloquerait.
> Journal détaillé (chronologique, §A→§R) : `.ai/RESEARCH_THEATER_RE.md`.
> Code de référence : `internal/analysis/{weaponv3, objectiveevents, objectivescore, positions}`,
> `cmd/diag_weapons_v3`, outils throwaway `tmp_film_explore/`.

---

## 0. Contexte

LevelUp décode les films Theater pour extraire des stats absentes de l'API publique (arme-de-kill,
timeline d'objectifs, score par équipe sur le temps, positions/heatmaps, dominance/comeback).
Tout le travail est **offline, lecture seule** (films téléchargés du CDN, sans auth). On NE cherche
PAS à modifier les films ; uniquement à lire l'état du match.

---

## 1. Structure du conteneur (VALIDÉE)

- Film = N chunks **zlib** (magic `0x78`). Chaque chunk décompressé est une suite de **paquets**.
- **Header de paquet = 16 octets** : `[Type u16 LE][b2 u8][b3 u8][Size u32 LE][Timestamp µs u64 LE]`,
  suivi de `Size` octets de payload. On lit `[16o header][Size payload]` en boucle jusqu'à `Type==7`.
- Types de paquet observés : `0`=FRAME (~1199/chunk @60fps, dt≈16.7ms, chaque FRAME porte un µs absolu),
  `1`=REPLICATION_DATA_START / header, `2`=TYPE_2 (snapshot game-state ~20s), `3`=HIGHLIGHT_EVENTS (footer),
  `6`, `7`=CHUNK_END, `8`=PLAYER_METADATA (~25KB, roster), `10`=TYPE_10 (1199× interleavé), `12`=BOT_METADATA.
- **chunk_00 (Type 1)** = **registre ECS** (~1.97 Mo) : ~264 composants nommés (ASCII) — la « table des
  matières » de tout l'état répliqué (ex. `object-position-component`, `statborg-current-round-value-stat-component`,
  `selectable-zone-data-component`, `managed-objective-*`, `weapon-ammo-component`…).

---

## 2. CE QUI EST DÉCODÉ (méthode + ancre)

Toutes les percées reposent sur une **ancre incidente** (motif binaire stable) qui contourne le schéma
de valeurs ECS. Statut « production » (câblé) ou « validé » (prouvé, non câblé).

| Donnée | Méthode / ancre | Statut |
|---|---|---|
| **Kill-feed** killer→victim | jointure temporelle highlight events type-3 (th=20 death ↔ th=50 kill, dt≈0), 99% | prod |
| **player_index ↔ xuid** | 5 bits IMMÉDIATEMENT avant le xuid 64-bit **LE**, recherche **bit-level** dans un chunk gameplay (acurtis) ; remplace l'ordre DB cassé | prod (v3) |
| **Arme-de-kill** | fire events (marqueur 11-bit `0b10100100110`, weapon-id 64-bit = high-32 identité + suffixe `0x42c9679f`), corrélation claim-and-remove ; timing µs via le FRAME contenant ; canon high-32 (~37 armes, suffixe 42c9679f = vraie arme) | v3 shadow (~85-90% HIGH Arena) |
| **Melee (arme réelle)** | §K-bis : ancre octet `0x34/0x35` (préfixe 3-bit `101`), type@+76∈{0x42,0x47,0x60}, weapon-id aux offsets {0x42:88,0x47:86,0x60:101/103} **en bits**, pi = octet@+20 bits0-4 | validé |
| **Grenade (type)** | §C : marqueur object-id `0x4c0c00`, weapon32 @+24bits, allowlist {Frag 0xB0171062, Plasma 0xC0E34C44, Shock 0x3B2567D4, Spike 0x9212E428} ; PAS de pi | validé (détection, pas d'attribution lanceur) |
| **Score (par mode)** | ancre **token 12-bit `0x7B6`** (MSB-first, bit-level, fenêtre payload TYPE_2 bytes [835,912)). Slayer byte813×4 / byte823 ; CTF captures = **burst FRAME re-transmettant la table 6-tiers** (détecteur `tiers==6`, ms-exact) ; KOTH Ranked = meters `t2[token+12]/[+16]` ÷5 ; Strongholds = varint à continuation @token+24bits ×~3.86 (LEADER uniquement) | partiel (cf. §3) |
| **Positions joueurs (keyframe)** | §N : par record full-state, un **comb** (motif bit `1⁸0¹⁶ ×4` = 96 bits) ; position = triplet **float32-LE** @ combStart−273 bits ; per-frame = float32-**BE**, ancre FRAME `0xA07B4200`+TICK | livré (angle A, match-level) |
| **Events objectif** | CTF capture (burst tiers==6) ; Strongholds/KOTH/Oddball = events footer th=10 (équipe via roster, temps ±5-20s) | livré (table match_objective_events) |
| **Immobilité / mort / respawn** | frame-burst de MORT (full-state re-transmis, ~4× baseline) + gap figé + frame-burst de RESPAWN, ±3ms des events | validé |

---

## 3. CE QUI EST BLOQUÉ — le mur du schéma `.module`

Cible commune : les **valeurs des composants ECS** sont répliquées **indexées par handle d'objet**
(pas par nom), et leur **layout binaire est défini par les tag-files `.module`** (hors film, côté moteur /
runtime-tagviewer). Le registre chunk_00 nomme les composants mais n'expose pas d'ancre incidente vers
leurs valeurs. Sans le schéma, on ne sait pas où/comment lire la valeur d'un composant donné.

| Cible bloquée | Composant ECS (nommé dans chunk_00) | Détail du blocage |
|---|---|---|
| **Score par équipe sur le temps** (les DEUX équipes, modes objectif) | `statborg-current-round-value-stat-component` (×28) ; `statborg-finalized-rounds-values-stat-component` | Le film ne stocke près du token que l'**accumulateur du LEADER** (cross-validé : team1 candidate = marqueur structurel de fin, pas un score ; winner-proportional trouvé cv 0.003, loser jamais). Le vrai score live est le statborg, handle-indexé, layout `.module`. |
| **Contrôle / identité de zone** (quelle base A/B/C, owner à T, Strongholds) | `selectable-zone-data-component` (×32) ; `managed-objective-sub-objective-entities-component` | Zone-id non récupérable (§R) ; le slot `b36` = slot joueur, pas zone. Owner par-zone ni co-localisé ni à offset fixe. |
| **Tracks denses per-joueur** (positions des 8, attribuées, par frame) | `object-position-dynamic-precision-component` | Delta-compression : seuls les joueurs qui changent sont ré-émis → l'index P n'est pas à offset fixe + continuité cassée au gap de mort. On isole UN joueur via une ancre d'event (sa mort), pas les 8 génériquement. |
| **Aim / hit location** (tête/corps/raté) | `unit-desired-aiming-vector-component` | L'aim vector existe (préfixe stable par joueur/arme) mais sans impact-balle décodé, pas de hit location. |
| **Vitalité / boucliers / ammo / camo** | `object-*-vitality-component`, `weapon-ammo-component`, etc. | Mêmes valeurs ECS handle-indexées, schema-blocked. |

**Oddball** est un cas à part : un seul crâne → l'accumulateur TYPE_2 est **global** (pas per-équipe) ;
le score per-team n'est qu'une intégrale lossy du temps de possession (gagnant ±1-3%).

**HavokScript** : §O (dend/acurtis) mentionne du bytecode HavokScript (`\x1bLua`) + stats nommées. **NON
retrouvé dans nos chunks cachés** (0 occurrence du magic) — soit dans une partie non mise en cache, soit
l'extraction venait d'une autre source. À clarifier (cf. questions).

---

## 4. L'insight central

> **Tout ce qu'on a cracké reposait sur une ANCRE INCIDENTE** (token `0x7B6`, marqueur `0x4c0c00`,
> comb `1⁸0¹⁶`, ancre FRAME `0xA07B4200`, xuid-adjacence 5-bit, suffixe `0x42c9679f`) qui permet de lire
> une valeur SANS connaître le schéma. **Les cibles restantes n'ont pas d'ancre** : elles vivent dans la
> réplication ECS indexée par handle. Les débloquer ne demande pas « plus d'astuce de scan » mais la
> **reconstruction du schéma `.module`** (mapping component-handle → layout binaire), qui est un projet
> distinct (off-film).

---

## 5. Questions précises pour les reverse engineers

1. **Réplication ECS** : le payload TYPE_2 (snapshot) et les deltas FRAME — quel format de sérialisation ?
   Microsoft **Bond** ? un format custom Saber/343 ? Y a-t-il une table handle→composant + un layout
   par composant qu'on peut reconstruire depuis le film, ou faut-il les tag-files moteur ?
2. **Tag-files `.module` / runtime-tagviewer** : disponibles publiquement ? Comment définissent-ils le
   layout des composants `statborg-current-round-value-stat-component` et `selectable-zone-data-component`
   (types/offsets des champs) ?
3. **chunk_00 (registre)** : comment les **handles/IDs** de composants sont-ils assignés, et comment la
   réplication TYPE_2/FRAME les référence (par index de registre ? par hash de nom ?) ?
4. **statborg** : la structure d'un `statborg-current-round-value-stat-component` est-elle une liste de
   paires (stat-id, valeur) ? L'enum des stat-id (FlagCaptures, PersonalScore, score d'objectif…) est-il
   stable/documenté ?
5. **HavokScript** : où vit le bytecode dans le film (pas dans nos chunks type-1/2/3 cachés) ? Donne-t-il
   le schéma de sérialisation des stats nommées ?
6. **Strongholds** : confirmation que le score live des DEUX équipes n'est PAS dans le snapshot keyframe
   près du token (on n'a trouvé que le leader) — est-il uniquement dans le statborg ECS, ou ailleurs ?

---

## 6. Ce qui a été livré côté produit (indépendant du mur)

- **Events objectif** (`match_objective_events`) backfillés (8140 events / 237 matchs).
- **Dominance / comeback** pour CTF (badges remontada/débâcle/contre-remontada) — backfillés (342 badges,
  dont 187 CTF nouveaux).
- **API** `/matches/{id}/objective-events` + overlay capture CTF (lignes/marqueurs) sur les graphes match-view.
- **Positions joueurs** keyframe (§N) : package décodeur + table `match_player_positions` + heatmap match-view
  (angle A) — **backfill en attente** (volontairement, pending feedback RE).
- **weapon_kills_v3** (shadow, non promu) : pi-fix + µs + canon + melee/grenade.

Branche : `feat/weapon-attribution-v3`. Tout en shadow / additif ; aucune table v2 modifiée.
