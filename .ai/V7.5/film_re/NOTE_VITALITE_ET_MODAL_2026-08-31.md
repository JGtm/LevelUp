# NOTE — Vitalité du film (i4/i5) et « MODAL = RATÉ ? » (2026-08-31)

Deux questions utilisateur tranchées sur pièces. Instruments (garde `LOT1_TRAME_FILM`) :
`internal/analysis/filmdec/lot1_vitalite_source_research_test.go` (Q1) et
`internal/analysis/filmdec/lot1_modal_touche_research_test.go` (Q2). Corpus : films
`000d5950`, `01e1f945`, `00502e52` (`data/cache/film_chunks`). `gofmt` + `go vet` verts.

---

## Q1 — La vitalité du film est-elle une découverte, ou ce que le rejeu affiche déjà ?

### Ce que SONT i4 et i5 (décodage, échelle, plage, fréquence)

- **i4 `object-body-vitality`** (SANTÉ) — `internal/analysis/filmdec/vitality.go`
  (`decodeObjectBodyVitality`, `BodyVitality`). Quantum `R(8)` déquantifié par
  `DequantEndpoint` sur **[-1, +1]** (endpointExact) —
  `internal/analysis/filmdec/quantize_endpoint.go` (`VitalityBodyMin=-1`, `VitalityBodyMax=+1`).
  `HealthFraction` replie la moitié négative sur 0 → **fraction [0, 1]**. La zone négative
  existe dans la sérialisation, sa sémantique n'est PAS établie.
- **i5 `object-shield-vitality`** (BOUCLIER) — même fichier (`decodeObjectShieldVitality`,
  `ShieldVitality`). Quantum `R(8)` déquantifié sur **[0, 4]** ; `ShieldFraction` clampe à 1 →
  **[0, 1]**. Le **surbouclier** vit dans le QUANTUM BRUT (`Shield.Q > OvershieldFullQ = 64`),
  jamais dans la fraction clampée.
- **Capture** : `internal/analysis/filmdec/offline_aim.go` (`componentVitals`,
  `scanRecordDirs`, `readBodyVitalityComponent`/`readShieldVitalityComponent`) — poursuite du
  MÊME record biped que la position, sous l'option `CaptureDirs`.
- **Exposition** : `internal/analysis/filmdec/offline_biped.go` — `BipedPosition.HealthAt()` /
  `ShieldAt()` (embed `componentVitals`), alimentées par `ScanFilmBipedPositions`.

### Fiabilité mesurée (3 films, film entier)

| Film | records biped | Santé i4 : couverture / plage frac | Bouclier i5 : couverture / Q / surbouclier |
|---|---|---|---|
| 000d5950 | 171 849 | 974 (**0,57 %**) · frac [0,016..1,000] · frac>0 100 % | 27 405 (**15,95 %**) · Q [0..64] · surbouclier 0 % |
| 01e1f945 | 173 032 | 1 025 (**0,59 %**) · frac [0..1] · frac>0 99,9 % | 47 959 (**27,72 %**) · Q [0..223] · **surbouclier 6,6 %** (Q>64), 93,4 % dans [0,64] |
| 00502e52 | 182 866 | 886 (**0,48 %**) · frac [0,008..1,000] · frac>0 100 % | 30 037 (**16,43 %**) · Q [0..64] · surbouclier 0 % |

- **Couverture PARTIELLE et c'est structurel** : le film ne réplique la vitalité que lorsqu'elle
  CHANGE. Santé ≈ **0,5-0,6 %** des records (ne bouge qu'aux dégâts qui percent le bouclier) ;
  bouclier **16-28 %** (change souvent en combat). Ce n'est pas un défaut de décodage.
- **Témoin de forme (décisif)** : les quanta i5 tombent tous dans [0, 255] sérialisable mais se
  concentrent sur **[0, 64]** (bouclier standard) ; le seul film avec surbouclier (01e1f945,
  power-up) sort proprement au-delà de 64 (jusqu'à Q=223 ≈ 3,5). Santé i4 : valeurs dans
  [-1, +1], part exploitable (frac>0) ≈ 100 %.

### VERDICT Q1

**i4/i5 n'est PAS une source distincte : c'est EXACTEMENT ce que le rejeu 2D publie déjà.**
`replay.decimateTracks` (`internal/analysis/replay/build.go`, lignes 647-651) remplit, par point
de trajectoire, `pt.Sh = p.ShieldAt()` et `pt.Hp = p.HealthAt()` — ces méthodes-ci de
`BipedPosition` — à partir du balayage `ScanFilmBipedPositions(..., CaptureDirs=true)`
(build.go:218-219). Les champs publiés sont `Point.Sh` et `Point.Hp`
(`internal/analysis/replay/document_aim.go`), dont la doc porte déjà la couverture (~16 %
bouclier, ~0,6 % santé) et la réserve (« Hp publié mais NON destiné à une barre : à 0,6 % de
couverture, toute barre serait 99 % du temps une valeur périmée »). Le surbouclier a son
canal séparé (`internal/analysis/replay/equipment_episodes.go`, règle `Shield.Q > 64` sur le
quantum brut, pas la fraction clampée).

Donc : la mesure « la santé qui baisse via `ScanFilmBipedPositions` » de l'agent précédent
lisait le **même** signal que le rejeu affiche — ce n'est ni plus précis ni plus fiable, c'est
la donnée déjà câblée. Il n'existe pas, dans le code, de lecture de vitalité concurrente à
i4/i5 : i4/i5 EST la vitalité canonique du film.

### Ne pas confondre avec la « fiche » PRODUIT (match view web)

Surface DIFFÉRENTE, source DIFFÉRENTE : la fiche joueur du produit (pages `features/`) affiche
des AGRÉGATS PAR MATCH de l'API Halo — `DamageDealt`/`DamageTaken` de `CoreStats`
(`internal/sync/transforms.go:323-324`) stockés en colonnes DuckDB
(`internal/sync/schema.go:219-220`), consommés par `internal/analysis/combat_yield.go` &
co (baseline 225 PV effectifs). Ce sont des sommes cumulées par match, PAS une vitalité par
instant. Le rejeu 2D, lui, n'utilise pas ces agrégats : il utilise i4/i5 du film.

---

## Q2 — « MODAL = RATÉ ? »

Le record `action_weapon_fire` (`0xD2`, type 36) est **MODAL** quand il ne porte NI cible NI
composante de dégât en ligne (classifieur de production `modalPostCountsBit`/`modalAimBit`,
`internal/analysis/filmdec/fire_aim_modal.go`). Question : « modal » = tir qui ne touche pas
(raté), ou le coup au but est-il rangé dans un `damage_aftermath` (`0xC0`, type 0) séparé ?

### Méthode

On apparie l'attaquant d'un tir (`ref0`, domaine 1, lu AVANT le champ à polarité contestée de
la grammaire) au responsable d'un `damage_aftermath` (`ref1`, domaine 1, MÊME encodage — les
deux se résolvent avec base 512 vers les slots bipèdes, cf. `lot1_degats_blesse`). Appariement
index brut / index brut, fenêtre ±W. Soins (magnitude négative, Kscale=-1) EXCLUS. Le
discriminant est le **ratio au témoin décalé** (même tireur, fenêtre autour de T+3 s) : un lien
causal le dépasse largement, « modal = raté » (pas de dégât) le mettrait au niveau du témoin.
Le taux ABSOLU n'est PAS un discriminant — il est plafonné par la densité des
`damage_aftermath` dans le flux (ex. 01e1f945 : 30 dégâts pour 491 tirs modaux).

### Mesures (12 chunks/film, W = 250 ms)

| Film | tirs (modaux %) | coups au but | AVANT modal→dégât même-tireur / témoin +3 s | ARRIÈRE dégât→tir modal même-tireur / témoin |
|---|---|---|---|---|
| 000d5950 | 245 (**85,7 %**) | 110 | 16,2 % / 1,0 % = **16,2x** | 18,2 % / 0,9 % = **18,2x** |
| 01e1f945 | 867 (**56,6 %**) | 30 | 3,9 % / 2,4 % = 1,6x | 33,3 % / 6,7 % = **5,0x** |
| 00502e52 | 276 (**79,0 %**) | 121 | 20,6 % / 4,1 % = **5,0x** | 28,1 % / 5,8 % = **4,9x** |

Sweep de fenêtre (000d5950) : modal→dégât même-tireur passe de 9,5 % (60 ms) à 28,1 % (500 ms) —
monotone avec la fenêtre, signature d'un lien temporel réel.

### VERDICT Q2

**« MODAL ≠ RATÉ » — TENU sur les 3 films.** Un tir modal PEUT toucher ; quand il touche, le
coup au but est rangé dans un `damage_aftermath` SÉPARÉ. La coïncidence tir-modal ↔ dégât du
MÊME tireur bat le témoin décalé de **1,6 à 16,2x (avant)** et **4,9 à 18,2x (arrière)** —
« modal = raté » la mettrait au niveau du témoin. L'arrière (« un dégât enregistré vient-il d'un
tir modal du même tireur ? ») est le plus fiable quand les dégâts sont rares (01e1f945).

**Nuance honnête, chiffrée** : le taux ABSOLU reste partiel (16-21 % des tirs modaux ont un
dégât même-tireur à ±250 ms, 28-33 % à ±500 ms) pour deux raisons — (a) le `damage_aftermath`
est SOUS-répliqué dans le flux delta échantillonné ; (b) une part des tirs modaux sont de VRAIS
ratés. On ne peut donc PAS dire « tout tir modal touche » ; on tranche que **la partie « coup au
but » d'un tir n'est JAMAIS dans le tir modal lui-même — elle vit dans un `damage_aftermath`**.

### Découverte à noter (NON traitée — hors périmètre)

L'en-tête de `internal/analysis/filmdec/fire_events.go` affirme « il n'y a pas de record de tir
manqué : "touché" est une propriété de tous les records lus ici ». Cette affirmation est en
TENSION avec les mesures : 57-86 % des tirs sont modaux mais seuls ~16-20 % ont un dégât
même-tireur à ±250 ms. Le flux est cohérent avec un `action_weapon_fire` émis à CHAQUE tir
(touché OU raté), le détail du dégât vivant dans le `0xC0`. À revérifier si le sujet revient ;
non corrigé ici (règle : zéro fix hors périmètre).

---

## Emplacements de code (récapitulatif)

- Vitalité i4/i5 : `filmdec/vitality.go`, `filmdec/quantize_endpoint.go`, `filmdec/offline_aim.go`,
  `filmdec/offline_biped.go` (`HealthAt`/`ShieldAt`).
- Vitalité affichée par le rejeu : `replay/build.go:218-219,647-651`, `replay/document_aim.go`
  (`Point.Hp`/`Point.Sh`), `replay/equipment_episodes.go` (surbouclier, quantum brut).
- Fiche PRODUIT (agrégat API, autre surface) : `sync/transforms.go:323-324`, `sync/schema.go:219-220`.
- Modal : `filmdec/fire_aim_modal.go` (`modalAimBit`), `filmdec/fire_events.go`.
- Damage aftermath : `filmdec/lot1_degats_research_test.go` (`lot1DecodeDamageAftermath`),
  `filmdec/lot1_degats_blesse_research_test.go` (ref0=blessé, ref1=responsable).
- Instruments de cette note : `filmdec/lot1_vitalite_source_research_test.go`,
  `filmdec/lot1_modal_touche_research_test.go`.
