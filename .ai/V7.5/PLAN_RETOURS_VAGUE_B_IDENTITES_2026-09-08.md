# Plan — Vague B : le P0 des identités interverties

> **2026-09-09 — PLAN ABANDONNÉ, OBJECTIF ATTEINT PAR UNE AUTRE VOIE.** Le sondage E2 puis le
> lot P-décodeur E2 (lien DIRECT corps → joueur lu dans le record de création du bipède, pont par
> morts relégué à la vérification, décision D14) ont établi la cause que la phase 1 devait
> instruire : le pont par morts nommait à tort 7 paires ÉCHANGÉES sur cinq films. Après fusion
> d'E2 dans `feat/v75` et recuisson du parc au schéma 50, la preuve prévue par ce plan a été
> jouée telle quelle : `swap.sh` rend **`malplaces = 0` sur les quatre témoins** (`8bc6074f`,
> `d8b13ec2`, `a4083bd2`, `bf2a9f05`, contre 2 chacun avant) et sur 37 des 38 artefacts testables
> du parc. Résidu : `58864b3c` (3 sur 7, séparation des camps 4,4 / 8,2 m — au seuil de validité
> de l'heuristique), consigné au registre. Les phases 1 et 2 ci-dessous ne sont pas exécutées ;
> conservées pour mémoire. Décision : plan master 2026-09-09, S4 / D5.


> Issu de `.ai/diagnostics/RETOURS_2026-09-08/DIAGNOSTIC_RETOURS_UTILISATEUR_2026-09-08.md`, § point 4.
> **Exécution sous le contrat du skill `plan-execution`.**

## Objectif et critère de succès

Le rejeu **affirme un faux** : sur 7 artefacts sur 30 testables, au moins un joueur est dessiné dans
le camp adverse ; sur 4, c'est une interversion propre de deux joueurs. C'est le seul constat du
diagnostic où l'écran ment.

**Ce plan se déroule en DEUX phases, et la phase 2 n'existe que si la phase 1 confirme.** C'est
délibéré : ma démonstration repose sur une heuristique, et je l'ai déjà eu faux une fois sur ce même
point (première passe : « réfuté », à tort).

**Critère de succès final :** `swap.sh` rend `malplaces = 0` sur tous les artefacts testables, et un
test de non-régression épingle le cas d'Origin.

**Effort** : phase 1 rapide, phase 2 lourde (elle touche le pont d'identités).
**Branche** : `feat/identites-interverties` depuis `feat/v75`.
**Worktree** : `LevelUp-wt-identites` / `wt/identites`.

## L'état de la preuve, à contredire

| artefact | carte | mal placés | fin de leur 1re vie | `publishable` |
|---|---|---|---|---|
| `8bc6074f` | Origin | 2 | 774 / 774 | 0 / 81 |
| `d8b13ec2` | Goliath | 2 | 462 / 462 | **97 / 97** |
| `a4083bd2` | The Pit | 2 | 775 / 776 | **95 / 95** |
| `bf2a9f05` | Bazaar | 2 | 176 / 176 | **96 / 96** |

Thèse : **deux vies qui se terminent au même instant, deux corps, et l'appariement fin de vie ↔
mort du fil tranche à égalité.** Quand les deux morts sont de camps opposés, l'erreur produit
l'interversion propre. Elle se résorbe dès que les fins se séparent.

## Décisions PRISES

| # | Décision | Valeur retenue |
|---|---|---|
| D1 | Le correctif ne passe PAS par `publishable` | Établi : 3 interversions sur 4 surviennent sur des matchs entièrement publiables. Étendre cette porte à la géométrie ne corrigerait rien |
| D2 | Une vie mal nommée est pire qu'une vie non nommée | En cas d'égalité irréductible, **refuser de nommer** plutôt que tirer au sort. Cohérent avec `ClosedContested`, déjà documenté comme « déductions ABANDONNÉES faute d'unicité » |
| D3 | Le départage privilégié | La **position** : le camp spawne groupé, et la vie suivante du même slot tombe du bon côté. Aucune de ces deux informations n'est consultée aujourd'hui |
| D4 | Pas de recuisson dans cette vague | Corriger le pont change les artefacts à cuire, pas ceux déjà cuits. La recuisson est une tâche à part (cf. vague C, § recuisson) |

## Règles d'exécution

Identiques à la vague A (ordre strict, statuts `[x]`/`[~]`/`[!]`, zéro fix hors périmètre, une seule
revue en fin de vague, reprise par la première case non cochée).

**Règle propre à cette vague : la phase 1 peut CLORE le plan.** Si la contre-vérification réfute la
thèse, on s'arrête, on écrit pourquoi, et la phase 2 n'a pas lieu. Ce n'est pas un échec.

---

# PHASE 1 — Contre-vérification adverse (aucun code modifié)

Consigne à donner au contexte frais, **littéralement** : « Voici un constat et sa démonstration. Ta
tâche est d'établir qu'il est FAUX. En cas de doute, conclus réfuté. »

## Étape B1 — Attaquer l'hypothèse de spawn groupé

**Périmètre fermé :**

- [ ] Établir, mode par mode, si « le camp spawne groupé à l'image 0 » tient. Modes présents au
      parc : Slayer, Team Slayer, CTF, Strongholds. **Le contre-exemple attendu** : un mode ou une
      carte où les spawns initiaux sont mêlés (mêlée générale, Fiesta, spawns aléatoires)
- [ ] Vérifier que les **23 artefacts écartés** par `swap.sh` (séparation < 8 m) le sont pour une
      raison légitime et ne cachent pas d'autres interversions
- [ ] Établir si le seuil de 8 m est le bon : le faire varier (4, 8, 16) et regarder si la
      population de « mal placés » bouge

**Gate :** un tableau mode × « hypothèse tient / ne tient pas », et une phrase de conclusion :
l'heuristique est valide sur quel sous-ensemble.

## Étape B2 — Attaquer le sens de l'erreur

**Périmètre fermé :**

- [ ] Établir que c'est le FILM qui se trompe et non le SCOREBOARD. Piste : `match_participants`
      vient de l'API Halo, qui ne se trompe pas sur l'appartenance d'équipe — mais le vérifier
      plutôt que le supposer
- [ ] Chercher une troisième explication : un joueur peut-il légitimement apparaître dans la base
      adverse à l'image 0 ? (rejoignant en cours, spectateur, spawn de secours)
- [ ] Sur `d8b13ec2` (Goliath, Team Slayer, 97/97 publiables), rejouer le raisonnement de bout en
      bout et tenter de le casser

**Gate :** conclusion écrite « la thèse tient » ou « réfutée », avec la preuve.

## Étape B3 — Attaquer la cause proposée

**Périmètre fermé :**

- [ ] Vérifier que la coïncidence des fins de vie est bien **causale** et non corrélée : existe-t-il
      des artefacts avec des fins de vie simultanées et **aucune** interversion ? Combien ?
- [ ] Si oui — et c'est probable — établir ce qui distingue les deux populations. La simultanéité
      est-elle nécessaire, suffisante, ni l'un ni l'autre ?
- [ ] Localiser dans `analysis/replay/` (`closures.go`, `closures_respawn.go`, `identity.go`,
      `owners.go`) **la fonction qui tranche**, et dire si elle tranche vraiment ou si le nommage
      vient d'ailleurs

**Gate :** le nom exact de la fonction fautive, ou la démonstration qu'elle n'existe pas sous cette
forme.

## Porte de sortie de la phase 1

- [ ] **Si B1, B2 et B3 confirment** → la phase 2 s'ouvre, cadrée par ce que B3 a localisé
- [ ] **Si l'un des trois réfute** → écrire le registre de réfutation, mettre à jour le diagnostic,
      **clore le plan**. Le point 4 redevient un constat sans cause établie, et il faudra
      repartir d'une observation

---

# PHASE 2 — Le correctif (ne s'ouvre que si la phase 1 confirme)

> Le périmètre exact dépend de ce que B3 aura localisé. Ce qui suit est le CADRE, à préciser à
> l'ouverture de la phase — et cette imprécision est assumée, elle est la conséquence directe de la
> porte de sortie ci-dessus.

## Étape B4 — Instrumenter avant de corriger

- [ ] Ajouter au pont un compteur des appariements tranchés À ÉGALITÉ (fins de vie simultanées),
      publié dans `coverage.bridge` — aujourd'hui `indexDisagreements: 0` déclare le pont sain alors
      qu'il se trompe
- [ ] `slog.InfoContext(ctx, "...", ...)` sur chaque arbitrage à égalité, AVANT toute dégradation
      (règle n°3 du dépôt : jamais d'erreur avalée)
- [ ] Mesurer sur le parc : combien d'arbitrages à égalité, sur combien d'artefacts

**Gate :** `jq -r '.coverage.bridge' data/cache/replays/halo_infinite/8bc6074f.json` expose le
nouveau compteur, non nul sur ce match.

## Étape B5 — Départager par la position

- [ ] Implémenter le départage retenu (décision D3) dans `internal/analysis/replay/` — algo **pur**,
      testable hors I/O, conformément aux couches du dépôt
- [ ] Cas d'égalité irréductible : **ne pas nommer** (décision D2), incrémenter le compteur de B4
- [ ] Tests unitaires : deux vies finissant à la même image, camps opposés, spawns séparés → le
      départage retient le bon appariement ; spawns non séparables → aucune des deux n'est nommée

**Gate :**
```bash
cd apps/go-api && go test ./internal/analysis/replay/...
```

## Étape B6 — Non-régression sur le parc

- [ ] Recuire les 4 artefacts témoins (`8bc6074f`, `d8b13ec2`, `a4083bd2`, `bf2a9f05`)
- [ ] `bash .ai/diagnostics/RETOURS_2026-09-08/swap.sh <teams.tsv>` → **`malplaces = 0`** sur les 4
- [ ] Vérifier qu'aucun artefact jusque-là sain ne régresse
- [ ] Poser un test de corpus qui épingle Origin : le joueur `2535469190789936` spawne côté `t0`

**Gate :**
```bash
bash .ai/diagnostics/RETOURS_2026-09-08/swap.sh /tmp/teams.tsv | awk -F'\t' 'NR>1 && $3>0'
# doit ne rien rendre
cd apps/go-api && go test -tags=integration ./...
```

---

## Clôture de vague

- [ ] `make go-api-test` puis `go test -tags=integration ./...` verts
- [ ] `swap.sh` à zéro sur le parc recuit
- [ ] **Une** revue adversariale sur le diff complet
- [ ] Entrée `.ai/thought_log.md` — et si la phase 1 a réfuté, l'écrire aussi clairement que si
      elle avait confirmé
- [ ] Mettre à jour le § point 4 du diagnostic avec le verdict

## Découvertes (à remplir — NE PAS TRAITER)

_(vide au démarrage)_
