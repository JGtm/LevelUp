# damage_section_response (type 1, octet 0xC0) — grammaire percée, enjeu armes lourdes RÉFUTÉ

> 2026-08-31. Worktree `wt/trame-type1`. Instrument
> `apps/go-api/internal/analysis/filmdec/lot1_degats_type1_research_test.go`
> (garde `LOT1_TRAME_FILM`, borné 12 chunks). Corpus témoin : films `000d5950`,
> `01e1f945`, `00502e52`. Adresses = VA Ghidra (ImageBase `0x140000000`).

## 1. Ce qu'on cherchait

Le type **1** partage l'octet **0xC0** avec `damage_aftermath` (type 0) — seul le bit de
type R(7) change (0 → 1). Baseline mesurée (`lot1_attrib_arme_tir`) : les armes
LOURDES/explosif/faisceau (M41 SPNKr, Hydra, Skewer, Ravager, Shock Rifle, Mangler,
Stalker, Bulldog, Fuel Rod) affichent **0 % de touches** en type 0 — elles n'émettent pas
de `damage_aftermath`. Hypothèse : leur dégât passe par le type 1, et si le type 1 porte
attaquant + victime en réfs d'en-tête, on pourrait attribuer ces armes PAR LE TIR et
combler le trou de précision.

## 2. Grammaire du type 1 — lue dans l'exe, au bit près (confiance HAUTE)

Descripteur `0x144724f78` → vtable `0x143d0fa10`. Structure de réception 28 octets.

**Domaines des 3 réfs d'en-tête** (`vtable+0x58` = `FUN_14080a048`, un `switch(i)` sans
lecture de bit, cible `0x14232a4ba`) :

| réf | domaine | largeur `R(w)` | rôle |
|---|---|---|---|
| ref0 (i=0) | **1** (entité, sonde) | 13 ou 9 si sonde | l'unité qui répond au dégât |
| ref1 (i=1) | **8** | 13 | (jamais présente sur le corpus) |
| ref2 (i=2) | **7** | 13 | (jamais présente sur le corpus) |

Comparaison type 0 : dom **1 / 1 / 7** — DEUX entités (blessé + responsable). Type 1 :
dom **1 / 8 / 7** — UNE seule entité (ref0).

**Charge utile** (`vtable+0x68` = `FUN_140968368`), désassemblée au bit :

```
R(5)                          // p0
R(1) g1 ; si g1 == 0 : R(4)   // p1, optionnel, POLARITE INVERSEE (FUN_1409684dc : bit 0 -> present)
R(3)                          // p2 (FUN_1424d0f48, inconditionnel)
R(1) g2 ; si g2 == 1 : R(19)  // p3 : direction (FUN_14076dc04 width 0x13 -> FUN_1406d8288 =
                              //      unpack vecteur UNITE normalisé, 0 bit supplémentaire)
```

Min **10 bits**, max **33 bits**. **AUCUN** tag source R(32), **AUCUNE** magnitude R(5) sur
[0,16], **AUCUNE** seconde entité. Ce n'est PAS un enregistrement de dégât autoritaire :
c'est la **réponse d'une section** (quelle section touchée, depuis quelle direction) — un
retour cosmétique/feedback. Le vrai dégât reste le type 0 (`damage_aftermath`).

## 3. Oracle de trame — grammaire VALIDÉE (chiffres)

Après décodage complet de l'événement (en-tête + charge) et du bit de continuation, la
trame de records doit décoder comme une trame de tick. Témoin = même décodage décalé de +3
bits. Discriminant = PROFONDEUR (records/paquet), pas taux de fermeture.

| film | type 1 | paquets à évt unique | profondeur réelle | profondeur témoin +3b | verdict |
|---|---|---|---|---|---|
| 000d5950 | 55 | 10 | **3.00** / paquet | 0.00 | TENU |
| 01e1f945 | 22 | 4 | **2.75** / paquet | 0.00 | TENU |
| 00502e52 | 42 | 9 | **3.78** / paquet | 0.00 | TENU |

La trame plonge (~3 records/paquet, comme le type 0 à ~2,4) au bon cadrage et s'effondre à
0 au témoin +3 bits. **Grammaire confirmée** (seuil adapté au type rare : ≥2 records/paquet,
≥3× le témoin, ≥4 paquets testables — la plupart des type 1 sont en listes multiples).

## 4. Résolution des réfs (chiffres corpus)

- **ref0 (dom1)** : présente à **100 %** (55/55, 22/22, 42/42), 17-36 entités distinctes,
  résout au biped à la base **512/510** (même bande que la victime du type 0). = l'unité qui
  encaisse le dégât (la **victime**).
- **ref1 (dom8)** et **ref2 (dom7)** : **présentes 0 %** sur les trois films. En pratique le
  type 1 ne porte QUE ref0. **Aucune référence d'attaquant.**
- Charge : p0 ∈ {0 (majoritaire), 2, …}, p1 toujours présent, p2 toujours 0, direction
  présente ~45-67 % (l'angle d'impact quand connu).

## 5. Armes lourdes — RÉFUTÉ (chiffres)

Type 1 ne portant qu'une entité (la victime, ref0), pas d'attaquant, l'attribution PAR LE
TIR (qui appariait attaquant du tir ↔ responsable du dégât) est impossible. Deux tests de
repli contre témoin +3 s :

| film | lien par CLÉ (ref0==attaquant, ±250ms) | coïncidence (n'importe quel type 1, ±250ms) | bilan lourdes coïnc. 250ms |
|---|---|---|---|
| 000d5950 | 3.3 % (témoin 0.4 % → 3.3×) | 14.3 % (témoin 10.6 % → 1.3×) | 10.5 % |
| 01e1f945 | 0.9 % (témoin 0.0 %) | 7.2 % (témoin 5.7 % → 1.3×) | 18.2 % |
| 00502e52 | 1.4 % (témoin 1.1 % → 1.3×) | 11.6 % (témoin 11.6 % → 1.0×) | 14.5 % |

- Le lien par clé est en valeur ABSOLUE dérisoire (≤3,3 %) — bruit d'auto-dégât, pas un
  canal d'attribution.
- La coïncidence est dominée par le taux de base : à fenêtre 2 s des armes NON lourdes
  montent aussi haut que les lourdes (Needler 100 %, Sidekick 100 %, Pulse 85,7 %), donc la
  fenêtre attrape des type 1 épars indépendamment de l'arme.
- Verdict par film : **RATE**. Le type 1 ne récupère PAS la précision des armes lourdes.

**Le trou reste ouvert** : les armes lourdes n'émettent ni type 0 ni un type 1 exploitable
pour l'attribution. Leur dégât, s'il est répliqué, l'est ailleurs (probablement via les
composants ECS du projectile/détonation — types 5 `projectile_detonate` / 6-7 impact, non
sondés ici) ou pas du tout à la maille per-hit. Piste éventuelle, hors périmètre de ce lot.

## 6. Portée

Instrument de recherche seul (garde `LOT1_TRAME_FILM`, sauté en CI). Aucun code de
production touché. `gofmt` + `go vet ./internal/analysis/filmdec/` verts.
