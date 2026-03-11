# Revue critique du bug report — Weapon Parser

> Revue du 2026-03-11 | Basée sur `weapon_parser_bug_report.md` + analyse DB exhaustive

---

## Synthèse exécutive

Le rapport identifie 4 bugs réels mais contient des **erreurs de métriques**,
**omet un bug critique** (Bug E), et propose des corrections dont certaines sont
**incomplètes ou inapplicables telles quelles**. Le plan d'exécution est globalement
correct mais l'ordre et les priorités doivent être ajustés.

---

## 1. Métriques réelles vs rapport

Le rapport annonce « 339 kills weapon_id=0 » et « 661 melee » pour JGtm.
Chiffres réels mesurés en DB :

| Métrique | Rapport | DB réelle | Écart |
|----------|:-------:|:---------:|:-----:|
| weapon_id=0 JGtm (tueur) | 339 | **262** (220 high + 42 none) | -77 |
| weapon_id=1 JGtm (tueur) | 661 | **661** (570 high + 91 none) | ✓ |
| weapon_id=NULL JGtm | 1 | **54** | +53 |
| Hex manquants JGtm | ~123 | **119** (5 hex confirmés) | ~ok |
| **Total potentiellement mal classifié** | ~1124 (~19%) | **~1096** (~26.8%) | recalculé |
| Hex distincts inconnus (global) | 55 (rapport) | **279** | +224 |

**Note** : le rapport filtrait sur les matchs de JGtm (254 hex).
En global (tous joueurs), il y a 279 hex distincts inconnus pour 14 884 kills.

Contexte global weapon_kills :

| Catégorie | Kills | % du total |
|-----------|------:|:----------:|
| Armes connues (mappées) | 16 050 | 18.8% |
| Hex inconnus (suffixe 42c9679f, T1 low) | 14 884 | 17.5% |
| Sentinels (melee=1, grenade=0, véhicule=2) | 4 447 | 5.2% |
| NULL (non résolus) | 54 313 | **63.7%** |
| **Total** | **85 247** | 100% |

**63.7% de weapon_id=NULL** — c'est le vrai problème critique, pas les 279 hex.

---

## 2. Analyse bug par bug

### Bug A — Faux positifs `is_grenade` par médaille ✅ CONFIRMÉ (mais mécanisme différent)

**Diagnostic du rapport** : la fenêtre ±500ms cause des faux positifs quand une
médaille grenade d'un autre kill « déborde » sur un kill arme normale.

**Ce que j'ai trouvé** : le rapport attribue correctement le mécanisme, mais
**omet la source principale** de weapon_id=0 confidence=high.

Les 220 kills `weapon_id=0 confidence=high` de JGtm **ne viennent PAS** de la
classification `is_grenade` dans `_weapon_kills_repo.py` (qui produit
`_make_sentinel_result` → `confidence="none"`). Ils viennent de :

**`_inject_missing_sentinels` (Step 4b)** dans `weapon_extraction_service.py:60-108`.

Ce step reclassifie les kills arme **les moins certains** (low → none → medium
→ high+swap → high, par delta_ms desc) en GRENADE/MELEE sentinel pour combler
le déficit entre les agrégats API et les sentinels détectés par médailles.

Conséquence : Step 4b **amplifie** les faux positifs de `is_grenade` en ajoutant
encore plus de kills grenade pour matcher l'API. Si l'API dit 50 grenade_kills
et qu'on en détecte 42 par médaille, Step 4b reclassifie 8 kills arme
supplémentaires en grenade → weapon_id=0.

Les vrais faux positifs médaille sont les 42 kills `weapon_id=0 confidence=none`
(sentinels purs via `is_grenade=True`).

**Verdict** : Bug A est réel mais le mécanisme décrit est incomplet. Le Step 4b
est un amplificateur, pas la cause racine.

### Bug B — Hammer/melee sans fire event ✅ CONFIRMÉ

**Diagnostic du rapport** : correct. Les Hammer/Sword ne génèrent pas de fire event,
et leurs médailles spécifiques ne sont pas dans `MELEE_MEDALS`.

**Validation DB** :
- Gravity Hammer : 2 kills en DB (très rare)
- Rushdown Hammer / Diminisher of Hope : 0 kills en DB
- Médaille « Pancake » (Hammer) : 8 occurrences dans highlight_events
- « Back Smack » : 1443 occurrences — c'est la seule médaille melee volumineuse

**Médailles melee présentes dans la DB mais absentes de MELEE_MEDALS** :
- `Whiplash` (500 occ.) — ⚠️ c'est **PAS** une médaille melee pure, c'est un
  « grapple + melee » → à **NE PAS** ajouter inconditionnellement
- `Bulltrue` (159 occ.) — kill d'un joueur qui charge avec une Energy Sword
  → le tueur utilise N'IMPORTE QUELLE arme → **NE PAS** ajouter à MELEE_MEDALS
- `Ninja` (54 occ.) — melee dans le dos après saut → **OUI** c'est melee
- `Pancake` (8 occ.) — kill Hammer → **OUI**, Hammer dans le dos = melee
- `Pineapple Express` (2 occ.) — grenade collée + melee → **AMBIGU**

**Verdict** : Bug B confirmé, mais la correction 2c doit être affinée.
`MELEE_MEDALS` ne doit **pas** inclure `Whiplash` ni `Bulltrue`.
Ajouter uniquement : `Ninja`, `Pancake`.

**Médailles Hammer spécifiques** : le rapport propose « Skullcase » et « Pound Town »
mais ces médailles **n'existent pas** dans les highlight_events. Elles n'existent
peut-être pas dans Halo Infinite ou ont un autre nom. Ne pas les ajouter sans
vérification.

### Bug C — Drop silencieux variantes ✅ CONFIRMÉ (mais impact nul sur les 279 hex)

**Diagnostic du rapport** : correct sur le mécanisme, mais l'impact est mal évalué.

Le drop silencieux dans `_scan_fire_events_bitstring:222` et `scan_formula_a:100`
concerne les fire events avec **suffixe ≠ `42c9679f` ET pas dans WEAPON_ID_MAP**.

Or **100% des 279 hex inconnus ont le suffixe `42c9679f`**. Le filtre suffixe
les laisse PASSER. Ce ne sont pas des « variantes invisibles ».

Le drop silencieux affecte potentiellement les variantes exotiques (Arcane Sentinel,
Needler Pinpoint) qui auraient un suffixe non standard — mais vu que 0% des hex
inconnus a un suffixe différent, l'impact actuel sur les données est **nul**.

Le log DEBUG proposé reste utile pour de la découverte future, pas comme correctif.

**Source réelle des 279 hex** : chemin T1 (Formula A) `_attribute_t1_kills`.
Dans `scan_formula_a`, les octets avec suffixe `42c9679f` sont **acceptés** comme
fire events. Ensuite `_attribute_t1_kills` fait :
```python
wid_int = int.from_bytes(wid_bytes, byteorder="big")
conf = "high" if wid_bytes in WEAPON_ID_MAP else "low"
```
Résultat : un weapon_id inconnu mais enregistré avec confidence=low.

Ces 279 IDs sont vraisemblablement des **skins cosmétiques** d'armes connues.
Halo Infinite a des centaines de skins visuels par arme, et chacun a potentiellement
un ID filmshell distinct avec le même suffixe `42c9679f`.

### Bug D — 5 hex confirmés manquants ✅ CONFIRMÉ

**Validation DB** : les 5 hex existent dans weapon_kills et représentent un volume
significatif **globalement** :

| Hex | Nom supposé | Kills JGtm | Kills global | Matchs global |
|-----|-------------|:----------:|:------------:|:-------------:|
| `91eb16de42c9679f` | Mk51 Sidekick (alt) | 49 | 2479 | 275 |
| `edff0e9642c9679f` | Mk51 Sidekick (alt) | 34 | 656 | 214 |
| `b1eb695e42c9679f` | Mk51 Sidekick (alt) | 23 | 383 | 170 |
| `f951480042c9679f` | Mk51 Sidekick (alt) | 7 | 311 | 136 |
| `f55c4bd242c9679f` | MA40 AR (alt) | 6 | 377 | 104 |
| **Total** | | **119** | **4206** | |

4206 kills globaux récupérés juste en ajoutant ces 5 hex — bon ROI.

**Verdict** : Étape 1 du plan est valide et devrait inclure aussi les autres hex
fréquents du top 20 si on peut les identifier (ex: `91833a5a42c9679f` à 773 kills).

---

## 3. Bug manquant dans le rapport

### Bug E — 54 313 kills weapon_id=NULL (63.7%) [CRITIQUE]

**Non mentionné dans le rapport** sauf pour 1 kill JGtm.

Le rapport ne traite que les kills du POV de JGtm. Mais en réalité, **54 313 kills
dans weapon_kills n'ont aucun weapon_id** (63.7% du total). C'est le bug dominant.

Ces NULL viennent de `_match_kill_to_fire_event` qui retourne `weapon_id=None`
quand aucun fire event n'est trouvé dans la fenêtre de 5s. Ça concerne :
- Les joueurs non-POV sans données Formula A (pas dans le film)
- Les joueurs T1 dont le `player_index` n'a pas de snapshot Formula A dans le chunk
- Les kills en fin/début de chunk sans fire event à proximité

L'ampleur (63.7%) montre que le pipeline n'a une bonne couverture que pour le POV
et quelques coéquipiers T1. La majorité des joueurs n'a aucun weapon_id.

**Pas de correction immédiate** possible sans amélioration fondamentale du parsing
(ex: étendre le snapshot T1 avec des heuristiques plus agressives, ou utiliser
les médailles d'armes spécifiques comme hint). Mais c'est important de le documenter
car ça biaise toutes les métriques « armes les plus utilisées ».

### Bug F — Step 4b reclassifie des kills T1 low en grenade/melee high [HAUTE]

`_inject_missing_sentinels` trie les kills par uncertainty descendante et reclassifie
les moins certains. Or les kills T1 avec hex inconnu ont `confidence=low`, ce qui
les place **en premier** dans la queue de reclassification. Résultat : des kills
qui étaient réellement des armes (mais avec un skin inconnu) sont reclassifiés
en grenade/melee.

C'est un **poison** systématique : plus il y a de hex inconnus (279 en T1),
plus Step 4b a de candidats low à reclassifier, et plus il produit de faux
sentinels grenade/melee.

**Correction** : dans `_inject_missing_sentinels`, exclure les kills dont le
weapon_id est un hex avec suffixe `42c9679f` (c'est une arme réelle non mappée,
pas un candidat grenade/melee). Ou mieux : ne reclassifier que les kills avec
`weapon_id IS NULL` ou `confidence='none'`, jamais les `low` qui ont un weapon_id
réel.

---

## 4. Plan de correction révisé

### Priorité 1 — Ajouter les 5+ hex confirmés [TRIVIAL, RISQUE NUL]

Identique à l'Étape 1 du rapport. Également ajouter les hex non confirmés
si le top 20 peut être identifié par corrélation (même suffixe, fréquence élevée).

Les 5 hex seuls récupèrent 4 206 kills globaux. Les 20 premier hex inconnus
récupéreraient ~11 000 kills.

**Action** : ajouter dans `WEAPON_ID_MAP` + `WEAPON_FUSION_MAP` pour fusion
vers le nom canonique.

### Priorité 2 — Corriger `_inject_missing_sentinels` (Step 4b) [HAUTE]

**Fichier** : `src/data/services/weapon_extraction_service.py:60-108`

Ne jamais reclassifier un kill qui a un `weapon_id` réel (> 2) et non-NULL
en sentinel. Filtrer le pool :
```python
pool = iter(
    sorted(
        [r for r in kill_rows
         if r.get("weapon_id") is None  # uniquement les non-résolus
         or r.get("weapon_id") in excluded_ids],  # ou déjà sentinel
        key=_uncertainty_key,
    )
)
```

Alternativement, ne reclassifier que les kills `weapon_id IS NULL` (plus conservateur).

### Priorité 3 — `is_melee` prime sur `is_grenade` [MOYENNE]

Identique à l'Étape 2a du rapport. Ajout simple :
```python
is_melee_val = any(m in MELEE_MEDALS for m in nearby)
is_grenade_val = any(m in GRENADE_MEDALS for m in nearby) and not is_melee_val
```

Appliquer dans les deux méthodes : `load_player_kills_for_match` (L345-346)
et `load_all_kills_for_match` (L404-405).

### Priorité 4 — Enrichir `MELEE_MEDALS` [MOYENNE]

**Ajouter uniquement les médailles validées** :
```python
MELEE_MEDALS: frozenset[str] = frozenset({
    "Pummel", "Assassination", "Back Smack", "Melee", "Quigley",
    "Ninja",     # melee dans le dos après saut — confirmé melee
    "Pancake",   # kill Gravity Hammer — confirmé melee
})
```

**NE PAS ajouter** : `Whiplash` (grapple), `Bulltrue` (anti-sword, arme quelconque),
`Skullcase` / `Pound Town` (n'existent pas dans les données).

### Priorité 5 — Fenêtre grenade à ±300ms [BASSE → À TESTER]

L'Étape 2b du rapport (réduire ±500ms → ±300ms) est **risquée sans données
de validation**. Les grenades Dynamo ont un dégât retardé qui peut provoquer
un kill >300ms après le lancer. Recommandation :

1. Requêter les kills grenade confirmés (médaille grenade + weapon_id=0 confidence=none) et mesurer la distribution réelle des délais médaille→kill
2. Si P95 < 300ms → réduire à ±300ms
3. Si P95 entre 300-500ms → garder ±500ms

**Ne pas changer tant que les données ne valident pas le nouveau seuil.**

### Priorité 6 — Logger les variantes filtrées + fusion massive [BASSE]

L'Étape 3 du rapport (log DEBUG dans `_scan_fire_events_bitstring`) est utile
pour la découverte future seulement. Impact zéro sur les 279 hex actuels
(qui passent déjà le filtre suffixe).

Action complémentaire recommandée : créer un `WEAPON_FUSION_MAP` élargi qui
fusionne les 279 hex vers leur arme canonique par préfixe partagé ou heuristique.
Ça éliminerait le problème des skins cosmétiques d'un coup.

### Priorité 7 — Backfill complet [APRÈS toutes les corrections]

Identique à l'Étape 5 du rapport. Avec `--force-weapon-kills` pour re-processer.

---

## 5. Corrections au plan original

| Étape rapport | Verdict | Modification |
|:---:|---------|-------------|
| 1 | ✅ Correct | Étendre aux top 20 hex si identifiables |
| 2a | ✅ Correct | Inchangé |
| 2b | ⚠️ Risqué | Reporter après validation P95 des délais grenade |
| 2c | ⚠️ Partiellement faux | Retirer Skullcase/Pound Town, ajouter Ninja/Pancake |
| 3 | ✅ Utile mais impact nul immédiat | Inchangé, documenter que c'est préventif |
| 4 | ✅ Correct | Inchangé |
| 5 | ✅ Correct | Inchangé |
| — | ❌ **Manquant** | **Ajouter Priorité 2** : fix `_inject_missing_sentinels` |
| — | ❌ **Manquant** | **Documenter Bug E** : 54K kills NULL (63.7%) |

---

## 6. Ordre d'exécution révisé

```
[1] Priorité 1 — Ajouter 5+ hex dans WEAPON_ID_MAP          (trivial, risque nul)
[2] Priorité 2 — Fix _inject_missing_sentinels               (moyen, risque moyen, test requis)
[3] Priorité 3 — is_melee > is_grenade                       (faible, test requis)
[4] Priorité 4 — MELEE_MEDALS += Ninja, Pancake              (faible, test requis)
[5] Priorité 5 — Fenêtre grenade 500→300ms                   (REPORTER, données à collecter)
[6] Priorité 6 — Logger variantes filtrées                   (faible, risque nul)
[7] Priorité 7 — Backfill re-classification                  (après tout le reste)
```

---

## 7. Fichiers modifiés (révisé)

| Fichier | Priorités | Type |
|---------|-----------|------|
| `src/analysis/_weapon_data.py` | 1, 4 | Ajout mappings + médailles |
| `src/data/services/weapon_extraction_service.py` | 2 | Fix `_inject_missing_sentinels` |
| `src/data/repositories/_weapon_kills_repo.py` | 3 | Fix classification |
| `src/analysis/weapon_parser.py` | 6 | Ajout logs |
| `scripts/backfill_data.py` | 7 | Option `--force-weapon-kills` |
