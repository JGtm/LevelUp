# RAPPORT — les MANCHES dans l'API Halo Infinite : mesure sur le corpus complet

> Date : 2026-08-29 · Etape E0 de `.ai/PLAN_SCORE_PAR_MANCHES.md` · Registre date, aucune
> modification de code ni de base.
>
> Question posee : sur quels matchs la victoire se decide-t-elle aux MANCHES, et le score
> rendu par l'API ment-il alors sur le resultat ?

---

## 0. VERDICT EN QUATRE LIGNES

1. La regle candidate du plan — « `RoundsWon+RoundsLost+RoundsTied >= 2` donc on affiche les
   manches » — est **REFUTEE**. Elle attraperait le CTF d'arene (deux mi-temps) et y
   afficherait « 0 - 1 » a la place du vrai score de captures « 2 - 3 ».
2. Le mensonge existe et il est **mesure** : **4 matchs Oddball** ou l'equipe VICTORIEUSE a
   MOINS de points que la perdante. Aujourd'hui l'app affiche ces matchs comme des defaites
   au score.
3. Le discriminant n'est pas une formule universelle mais une **table MESUREE par variante**,
   meme doctrine que `regulation.toml [score_target]`. Seul **Oddball** (3 noms de variante,
   `GameVariantCategory` 18) la rejoint a l'issue de cette mesure.
4. Effet de bord verifie : **le score en base est identique au score de l'API sur 1 942
   matchs sur 1 942** — le backfill du 2026-08-24 tient, aucune derive depuis.

---

## 1. PROTOCOLE

| Etape | Detail |
|---|---|
| Liste | `diag_q shared_matches_v2.duckdb "SELECT match_id, pair_name, game_variant_name, team_0_score, team_1_score FROM match_registry WHERE team_0_score IS NOT NULL"` -> **1 942** matchs (sur 1 948 au registre) |
| Fetch | `cmd/diag_matchstats_dump --gamertag JGtm --rps 4`, 17 lots de 120 -> **1 942 / 1 942 succes, 0 erreur** (~10 min) |
| Extraction | `jq` sur `Teams[]` : `TeamId, Outcome, CoreStats.{Score, RoundsWon, RoundsLost, RoundsTied}`, + `MatchInfo.GameVariantCategory` |
| Analyse | jointure sur `game_variant_name` du registre |

`rounds_total` est pris comme **max(total equipe 0, total equipe 1)** : 4 matchs abandonnes
portent 1 chez un camp et 0 chez l'autre (cf. §4.1). Avec `max`, **aucun** match non-FFA ne
tombe a 0.

---

## 2. AMPLEUR

```
non-FFA          1941      (1 match FFA ecarte : != 2 equipes)
rounds_total >= 2  57      (2,9 %)
rounds_total == 1 1884
rounds_total == 0    0
totaux incoherents   4     (cf. §4.1)
```

### 2.1 Les 9 variantes multi-manches, et ce que dit leur score

| Variante | Cat. | Matchs | dont multi | `Score` == `RoundsWon` | Vainqueur avec MOINS de points | Manche nulle |
|---|---|---|---|---|---|---|
| `Arena:Oddball` | 18 | 13 | 13 | 0 | **3** | 1 |
| `Ranked:Oddball` | 18 | 9 | 9 | 0 | 0 | 0 |
| `Oddball:Arena` | 18 | 4 | 4 | 0 | **1** | 0 |
| `Arena:One Flag CTF` | 15 | 7 | 7 | 5 | 0 | 2 |
| `BTB:One Flag CTF` | 15 | 4 | 4 | 4 | 0 | 0 |
| `Assault:One Bomb` | 41 | 3 | 3 | 3 | 0 | 0 |
| `Arena:Attrition` | 7 | 2 | 2 | 2 | 0 | 0 |
| `CTF:Arena` | 15 | 120 | **13** | 0 | 0 | 13 |
| `Ranked:CTF` | 15 | 4 | **2** | 0 | 0 | 2 |

Trois familles se lisent dans ce tableau :

- **Oddball (26 matchs, 100 % multi-manches)** — le `Score` est un CUMUL DE POINTS sur toutes
  les manches (jusqu'a 287), sans aucun rapport avec le compte de manches. C'est la seule
  famille ou le score CONTREDIT le resultat.
- **One Flag CTF / One Bomb / Attrition (16 matchs, 100 % multi-manches)** — le `Score` **EST
  DEJA** le compte de manches (`3-1` = trois manches a une). Afficher les manches n'y change
  rien a l'ecran ; seul le mot « manches » manque.
- **CTF d'arene et classe (17 matchs multi sur 124)** — deux MI-TEMPS, pas des manches
  decisives : le `Score` est le total de captures (`2 - 3`) et c'est LUI le score du jeu. Le
  compte de manches y vaut `0 - 1`, ce qui serait un contresens a l'affichage. En prime la
  variante est tantot a une manche tantot a deux : aucune regle par variante n'y tiendrait.

---

## 3. LE MENSONGE, MATCH PAR MATCH (question (c) du plan)

Quatre matchs ou l'equipe qui GAGNE affiche MOINS de points que celle qui perd. Ce sont tous
des Oddball a trois manches :

| Match | Variante | Score API affiche aujourd'hui | Manches | Resultat reel |
|---|---|---|---|---|
| `293a763e` | `Arena:Oddball` | 181 - 186 | 2 - 1 | victoire equipe 0 |
| `ca738284` | `Arena:Oddball` | 176 - 168 | 1 - 2 | victoire equipe 1 |
| `adb93fb7` | `Arena:Oddball` | 277 - 234 | 1 - 1 (+1 nulle) | victoire equipe 1 (cf. §4.2) |
| `d9781168` | `Oddball:Arena` | 191 - 196 | 2 - 1 | victoire equipe 0 |

Aucun match ou le vainqueur soit a EGALITE de points (0 cas).

---

## 4. ANOMALIES ET CAS LIMITES

### 4.1 Quatre matchs aux totaux de manches incoherents entre les deux camps

`27a69918` (`Ranked:Slayer`) 0 vs 1, `e318a8b0` et `f7a18c04` (`BTB Heavies:Slayer`) 1 vs 0,
`d25548e7` (`Firefight:Heroic King of the Hill`) 1 vs 0. Tous a une manche cote vainqueur :
un camp qui abandonne ne se voit rien crediter. **Parade retenue : `max` des deux totaux**,
jamais le total d'un seul camp.

### 4.2 Une manche NULLE existe, et elle casse l'invariant

18 matchs portent `RoundsTied > 0` (13 en CTF d'arene, 2 en One Flag CTF, 1 Oddball, 2 Ranked
CTF). L'invariant « le vainqueur a strictement plus de manches gagnees » tient sur **56 des 57**
matchs multi-manches. L'exception est `adb93fb7` : 1 manche gagnee chacun + 1 manche nulle, et
la victoire va tout de meme a un camp. **Consequence pour l'affichage : a egalite de manches,
le compte de manches ne dit pas le resultat — il faut retomber sur les points.**

### 4.3 Quatre matchs multi-manches sans vainqueur

`af718783`, `ec938fb4` (One Flag CTF), `9f57c612` (One Bomb), `8b512df2` (CTF arene) : les
deux camps portent `Outcome = 1` (match nul). Rien a corriger, mais l'affichage doit tolerer
l'egalite.

### 4.4 Score en base vs score de l'API

**0 ecart sur 1 942 matchs.** La colonne `team_{0,1}_score` est fidele au champ
`CoreStats.Score` du payload : le defaut repare le 2026-08-24 n'est pas revenu.

---

## 5. CE QUE LA MESURE IMPOSE AU PLAN

1. **La detection sera DECLARATIVE et MESUREE**, pas deduite d'une formule : nouvelle section
   `[rounds_decide]` dans `config/titles/{slug}/mappings/regulation.toml`, cle =
   `game_variant_name`, meme doctrine que `[score_target]` (« une variante absente n'est
   jamais flaguee ; ne rien ajouter au juge »).
2. **Contenu initial mesure : les trois variantes Oddball uniquement** — `Arena:Oddball`,
   `Ranked:Oddball`, `Oddball:Arena`. Ce sont les seules ou l'affichage actuel MENT.
   - `One Flag CTF` / `One Bomb` / `Attrition` : non declarees, car leur `Score` est deja le
     compte de manches — les declarer ne changerait rien a l'ecran, et transformerait les 2
     cas a manche nulle (`3 - 3` affiche contre `2 - 2` de manches gagnees) en un affichage
     dont rien ne prouve qu'il soit celui du jeu.
   - `CTF:Arena` / `Ranked:CTF` : exclues, mi-temps et non manches decisives (§2.1).
3. **La regle d'affichage a trois conditions cumulatives** :
   ```
   afficher les manches  <=>  variante declaree dans [rounds_decide]
                              ET rounds_total (max des deux camps) >= 2
                              ET rounds_won(A) != rounds_won(B)
   ```
   Toute condition non remplie -> points (comportement actuel, aucune regression).
4. **Colonnes a persister** : `team_0_rounds_won`, `team_1_rounds_won`, `rounds_total`. La
   manche nulle se deduit (`rounds_total - rw0 - rw1`) ; pas de colonne dediee.
5. **Volumetrie du backfill** : 1 942 matchs a re-lire, ~10 min a `--rps 4`, 0 erreur
   constatee lors de cette mesure — le chemin est valide.

---

## 6. REPRODUCTIBILITE

Les 1 942 payloads et les TSV intermediaires vivent dans le scratchpad de la session (non
versionnes). Pour refaire la mesure :

```bash
go run ./cmd/diag_q <shared.duckdb> "SELECT match_id FROM match_registry WHERE team_0_score IS NOT NULL ORDER BY match_id" > ids.txt
xargs -n 120 diag_matchstats_dump --gamertag JGtm --rps 4 --out dumps/ < ids.txt
jq -r '[.MatchId, (.Teams|length), .MatchInfo.GameVariantCategory,
        (.Teams[]|select(.TeamId==0)|.Outcome), (.Teams[]|select(.TeamId==1)|.Outcome),
        (.Teams[]|select(.TeamId==0)|.Stats.CoreStats|.Score,.RoundsWon,.RoundsLost,.RoundsTied),
        (.Teams[]|select(.TeamId==1)|.Stats.CoreStats|.Score,.RoundsWon,.RoundsLost,.RoundsTied)] | @tsv' dumps/*.json
```
