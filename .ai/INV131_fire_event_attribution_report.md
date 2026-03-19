# INV131/132 — Rapport : Attribution des fire events par joueur

> Investigation multi-sessions : ~2026-03-16 → 2026-03-18
> Scripts expérimentaux : `scripts/experimental/inv131_fire_event_player_attribution.py`,
>   `inv132_b2_pi_formula.py`, `scripts/exp_find_pi_offset.py`, `scripts/exp_b2_pi_correlation.py`
> Fix partiel produit : 2026-03-18 (pov_xuid filter — à remplacer par fire_seq % n_players)
> Voir ERRATUM dans `scripts/experimental/FINDINGS_weapon_extraction_EN*.md`

---

## Symptôme initial

JGtm se voyait attribuer de mauvaises armes sur des matchs avec film POV appartenant à
FreshKalvin203 (match `82f3af9f`). Par exemple : des kills au sniper de JGtm étaient
enregistrés avec l'arme de FreshKalvin203 (ARs, pistols...).

---

## Architecture du protocole film Halo Infinite (rappel)

Le flux REPLICATION_DATA est nibble-shifté : `b[i] = ((raw[i] << 4) | (raw[i+1] >> 4)) & 0xFF`.

Structure d'un fire event dans ce stream :

```
b0=0x0d  b1=0x26  b2=0x00  b3=0x40  [ctr 1B]  [slot 1B]  [weapon 8B]
```

Le player_index (pi) dans Section 1 (snapshots d'état arme) est encodé : `pi = packet_byte >> 5`.
Dans les PLAYER_METADATA (type 8), chaque joueur a un pi assigné.

**Invariant POV** : le joueur POV a toujours `meta_pi=0` dans PLAYER_METADATA — il n'y apparaît
pas (seul `pi != 0` est retourné par `detect_pi_from_metadata`). Ses fire events portent aussi
`player_index=0` dans le stream.

---

## Théories testées et résultats

### Théorie A (2026-03-16) — `_build_marker(pi)` encode le player_index dans b1

**Hypothèse** : La fonction `_build_marker(player_index)` génère `byte1 = (pi << 5) | 0x06`.
Pour pi=1 cela donne `b1 = 0x26`. Peut-être que b1 varie selon le joueur ?
Formula Den : `b1 = 0x26 + pi * 32`, donc pi=0 → 0x26, pi=1 → 0x46, etc.

**Test** : Scan du stream nibble-shifté pour les 8 valeurs de b1 possibles
(`0x26, 0x46, 0x66, 0x86, 0xa6, 0xc6, 0xe6, 0x06`) sur le match `82f3af9f`.

**Résultat** : ❌ **INVALIDÉE**
Seul `b1=0x26` (pi=0) produit des fire events valides : 52 events.
Les 7 autres variants (`pi=1..7`) retournent 0 résultat.
→ b1 est FIXE pour tous les joueurs (pas d'encodage pi dans b1).

---

### Théorie B (2026-03-16) — Les top-3-bits de b0 encodent le player_index

**Hypothèse** : En analogie avec Section 1 (`pi = byte >> 5`), les 3 bits hauts de b0
pourraient encoder le pi du tireur.
`b0 = 0x0d` → top-3-bits = `0b000` = 0. `b0 = 0x2d` → top-3-bits = `0b001` = 1, etc.

**Test** : `_test_b0_top3_vs_pi()` dans `exp_find_pi_offset.py`.
Scan des 8 variants de b0 (`0x0d, 0x2d, 0x4d, 0x6d, 0x8d, 0xad, 0xcd, 0xed`) dans
le stream nibble-shifté. Pour les events valides trouvés : extraction des top-3-bits de b0.

**Résultat** : ❌ **INVALIDÉE**
Seul `b0=0x0d` produit des résultats valides (52 events). Les 7 autres = 0.
Top-3-bits de b0 = 0 pour 55/58 events valides analysés.
→ b0=0x0d est FIXE. Le marker complet `(b0=0x0d, b1=0x26)` est identique pour TOUS les joueurs.

---

### Théorie C (2026-03-17) — Le b2_stream (identificateur de flux) encode le joueur

**Hypothèse** : Chaque joueur aurait un "b2_stream" différent (`fire_events["b2_stream"]`),
permettant de grouper les events par joueur via `map_b2_to_player` → `group_events_by_pi`.

**Test** : `scripts/exp_b2_pi_correlation.py` — comparaison global pool vs b2_to_pi
sur le match `82f3af9f`.

**Résultat** : ⚠️ **PARTIELLEMENT VALIDÉE, mais inapplicable en production**
Sur un film POV-only, il n'y a qu'un seul joueur → un seul b2_stream → mapping trivial.
Sur un film all-players (match `147ffd4d`, 1177 events, 9 joueurs), b2_to_pi donne une
distribution multi-joueurs... mais le mapping reste peu fiable car b2_stream peut varier
au cours d'un match (rechargement, changement d'arme).
**Conclusion** : ne résout pas le problème racine (film POV-only = un seul joueur dans le pool).

---

### Théorie D (2026-03-17) — Le fire event counter encode l'appartenance au joueur

**Hypothèse** : Le byte `ctr` (fire_counter, position b4 après le marker) serait un compteur
global monotone ou pourrait être segmenté par joueur.

**Test** : Inspection des séquences de counter sur le match `82f3af9f`.

**Résultat** : ❌ **INVALIDÉE**
Le counter est effectivement monotone croissant et partagé globalement — il ne discrimine
pas les joueurs.

---

### Théorie E (2026-03-16) — `POV_PLAYER_INDEX = 1` est correct

**Hypothèse** : Le code production utilise `scan_fire_events_bitstring(pi=1)`, avec
`POV_PLAYER_INDEX = 1` dans `weapon_parser.py:45`. Ce serait correct si le joueur POV
avait meta_pi=1.

**Test** : Vérification via `detect_pi_from_metadata` sur le match `82f3af9f`.
FreshKalvin203 (xuid=2533274807120322) est-il à meta_pi=1 ?

**Résultat** : ❌ **INVALIDÉE**
`detect_pi_from_metadata` exclut explicitement pi=0 (`if pi != 0`). FreshKalvin203 est
le joueur POV → il n'apparaît PAS dans les résultats METADATA. Son vrai meta_pi est 0.
`_build_marker(1)` donne `b1 = (1<<5)|0x06 = 0x26` **accidentellement** — coïncide avec
le marker réel mais pour la mauvaise raison. Le code marchait car 0x26 est le marker
universel (fixe pour tous), pas parce que pi=1 est correct.

---

## Conclusion de l'investigation

### Fait établi — Le marker `(b0=0x0d, b1=0x26)` est universel

TOUS les fire events de TOUS les joueurs partagent le même marker fixe `0x0d 0x26` dans
le stream nibble-shifté. Le marker n'encode PAS le player_index.

### ⚠️ Invalidé — "Deux types de films"

La théorie "POV-only vs all-players" était fausse. **Tous les films contiennent les fire
events de tous les joueurs** (confirmé acurtis, 2026-03-18). Le match `82f3af9f` avec
58 events apparemment concentrés sur FreshKalvin203 ne constitue pas un "film POV-only"
— le claim-and-remove temporel global attribuait simplement ses events à ses kills par
proximité temporelle.

### ⚠️ Invalidé — "POV player = meta_pi 0 dans les fire events"

La conclusion "les fire events portent tous player_index=1 (POV)" des FINDINGS était une
erreur d'interprétation de byte[1]=0x26. Le player_index réel est dans **byte[2] = fire_seq**.

### Fait établi — Attribution via fire_seq (inv#132)

```
fire_seq (byte[2]) = player_index + life_number * n_players
→ player_index = fire_seq % n_players
```

Validé 7/7 points : b2=1→pi=1, b2=3→pi=3, b2=6→pi=6, b2=27→pi=3, b2=28→pi=4,
b2=34→pi=2, b2=46→pi=6.

### Racine du bug cross-attribution

`correlate_kills_global` utilise un pool global claim-and-remove sans filtrage par joueur.
Sans `fire_seq % n_players`, les events d'un joueur peuvent être réclamés par les kills
d'un autre joueur si leur timing est plus proche.

Match `82f3af9f` — données observées :
- 4 fire events cross-attribués à JGtm (×1), NewFlipBobCat (×2), alpal capone (×1)

---

## Fix appliqué (2026-03-18)

**Principe** : `correlate_kills_global` reçoit `pov_xuid`. Seuls les kills de ce joueur
sont éligibles aux fire events du pool. Les kills des autres joueurs tombent en fallback
Formula A (Section 1 / snapshots d'état arme).

```python
# src/analysis/_global_correlation.py
is_pov_kill = pov_xuid is None or kill["xuid"] == pov_xuid
candidates = [
    (i, ev) for i, ev in enumerate(available)
    if (t_ms - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= t_ms
    and is_pov_kill
]
```

Propagation : `process_match(xuid)` → `_process_match_inner(xuid)` →
`_correlate_all_players(pov_xuid=xuid)` → `correlate_kills_global(pov_xuid=pov_xuid)`.

**Impact film all-players** : `pov_xuid` est toujours passé (non None), donc les fire
events ne seront attribués qu'au joueur POV. Pour les 8 autres joueurs du pool all-players,
les fire events qui leur appartiendraient réellement seront ignorés (mis en compétition
uniquement pour le POV). C'est sous-optimal pour les films all-players mais correct —
Formula A reste un fallback fonctionnel, et la distinction POV-only / all-players n'est
pas encore détectable automatiquement.

**Fichiers modifiés** :
- `src/analysis/_global_correlation.py` (+7L net, fonction 88→86L)
- `src/data/services/weapon_extraction_service.py` (+5L net, module 518→523L)
- `scripts/size_baseline.txt` (ratchet mis à jour)

**Tests** : 4946 passed, 2 skipped, 2 failed pré-existants (weapon_data fusion labels,
sans rapport avec ce fix).

---

## Prochaine étape — Intégration fire_seq % n_players en production

Le fix `pov_xuid` implémenté le 2026-03-18 est une correction partielle (évite la
cross-attribution en restreignant les events au joueur POV, au prix de passer les autres
joueurs en Formula A). Il doit être remplacé par la formule `fire_seq % n_players` :

1. **Remplacer `map_b2_to_player`** (NS cross-ref avec drops) par `dispatch_by_formula`
   (`fire_seq % n_players`, 0 drop) de `inv132_b2_pi_formula.py` dans le pipeline production.

2. **Mapper pi fire_seq → pi Section 1** : les deux systèmes de numérotation sont
   indépendants (fire_seq pi ≠ Section 1 pi). Il faudra une table de correspondance ou
   utiliser PLAYER_METADATA pour relier les deux.

3. **Supprimer le filtre pov_xuid** une fois la dispatch par joueur opérationnelle.

4. **Valider inv132 sur le corpus complet** : lancer `python scripts/experimental/inv132_b2_pi_formula.py`
   pour mesurer l'accord NS/formule et le drop rate sur tous les matchs en cache.
