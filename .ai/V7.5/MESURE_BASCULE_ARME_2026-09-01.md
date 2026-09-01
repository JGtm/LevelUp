# MESURE — témoin de bascule de l'arme du kill (étape A0)

> Date : 2026-09-01. Volet A du plan `.ai/V7.5/PLAN_SOURCE_UNIQUE_ARME_2026-09-01.md`.
> Branche `wt/arme-source-unique`, worktree `LevelUp-wt-arme-source`.
>
> Ce fichier est le témoin de la bascule : il dit, chiffres en main, ce que le remplacement
> de la chaîne `weapon_kills` par la source de dégât change. Il n'argumente pas, il mesure.

## 1. Le seuil d'acceptation — écrit AVANT la lecture des résultats

Cette section a été écrite et enregistrée **avant** le premier lancement de la sonde
(commit du seuil antérieur au commit des résultats, cf. `git log` de la branche). C'est la
condition qui rend la mesure honnête : un seuil rédigé après coup se plie toujours au
résultat obtenu.

> **La bascule est ACCEPTÉE si, et seulement si :**
>
> 1. le résidu « Non attribué » agrégé **DIMINUE** — en valeur absolue et en part du total
>    des frags API du même lot de matchs ;
> 2. **aucune classe d'arme à feu** (`shoulder`, `sidearm`, `heavy`) ne perd plus de **2 %**
>    de ses frags **sans explication nommée** — une explication nommée étant un mécanisme
>    identifié et écrit dans ce fichier (par exemple : « ces frags migrent vers la classe X
>    parce que l'ancienne chaîne recollait sur l'arme tenue les morts d'une arme sans tir »).
>
> Si l'une des deux conditions tombe, la bascule n'est pas acceptée en l'état et l'étape A1
> ne démarre pas avant que l'écart ait reçu une explication écrite.

Corollaires qui ne sont **pas** des critères d'acceptation, mais que la mesure doit tout de
même publier (décision D13 du plan) :

- la concordance entre ce que le kill feed du rejeu sait nommer et ce que le graphe sait
  classer (A0.4) ;
- l'écart de nom entre les deux dictionnaires (A0.5).

## 2. La commande de reproduction, telle quelle

La sonde vit dans `apps/go-api/internal/platform/duckdb/temoin_bascule_arme_probe_test.go`
(+ `temoin_bascule_concordance_test.go`). Build normal, **sautée** sans variable
d'environnement — motif des sondes du dépôt.

Les bases sont ouvertes en **lecture seule**, et sur une **COPIE** : le modèle mono-process
de DuckDB interdit d'ouvrir un fichier qu'un autre process tient en écriture.

```bash
# 1. copier les bases de production locales (aucun serveur ne doit tourner)
cp data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb /tmp/hinf_shared.duckdb
cp data/titles/halo_infinite/warehouse/metadata.duckdb          /tmp/hinf_meta.duckdb

# 2. lancer la sonde
export PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1 CC=/c/msys64/ucrt64/bin/gcc.exe
cd apps/go-api
TEMOIN_ARME_SHARED=/tmp/hinf_shared.duckdb \
TEMOIN_ARME_META=/tmp/hinf_meta.duckdb \
TEMOIN_ARME_MATCHS=200 \
TEMOIN_ARME_SORTIE=../../.ai/V7.5/temoin_arme_sortie.md \
go test ./internal/platform/duckdb/ -run TemoinBasculeArme -v -count=1
```

## 3. Résultats

_Section remplie après le premier lancement. Voir plus bas._
