# HANDOFF — précision projectiles : la piste est CLOSE, une voie de décodeur reste ouverte

> Écrit le 2026-08-08 à la clôture de la journée. Branche `research/v75-precision`
> (11 commits, poussée, worktree `LevelUp-wt-precision`, **rien de mergé, rien en base, aucun
> décodeur touché**). Remplace `.ai/HANDOFF_PRECISION_PROJECTILES.md` du 2026-07-31, qui est
> **consommé** — ne pas repartir de lui, il envoie sur des pistes désormais fermées par mesure.

---

## 1. L'ÉTAT EN CINQ LIGNES

1. **La piste E est CLOSE, verdict NÉGATIF, timebox consommé** (2 sessions, décision #6 du
   master plan). La précision par arme des armes à projectile n'est pas livrable.
2. **Le blocage a été relocalisé** : le dénominateur (tirs par arme) n'a jamais manqué — c'est le
   NUMÉRATEUR (touches par arme) qui est absent, et trois voies de décodage sur trois sont
   fermées par mesure.
3. **Une voie de DÉCODEUR reste ouverte** et elle n'appartient pas à la piste E : rattacher
   l'entité projectile à un joueur par sa **trajectoire**. Elle est vivante (les projectiles
   portent bien des trajectoires dans le flux delta) et son prochain geste est **une lecture de
   binaire dumpé**, pas une recherche.
4. **Quatre corrections ont été portées au dossier** (index, guide, master plan) — dont deux
   affirmations qui étaient **fausses** et qui sont restées en ligne des semaines.
5. **Cinq pièges de méthode ont été payés dans la journée** (§6). Ils valent autant que les
   résultats : trois de mes quatre « écarts » contre le code de production étaient à moi.

---

## 2. CE QUI EST CLOS — ne pas rouvrir sans raison neuve

**Document d'autorité : `.ai/V7.5/VERDICT_PRECISION_PROJECTILES.md`** (§0bis = le verdict en une
page). Cinq hypothèses testées, chacune avec son contrôle :

| hypothèse | verdict | le chiffre qui tranche |
|---|---|---|
| Le compteur de tir 7 bits porte les tirs de projectile invisibles | **RÉFUTÉE** | pas moyen Needler **1,3383** contre BR75 **1,3545**, mêmes matchs. Il mesure la complétude du scan |
| Le record de projectile est une TOUCHE (7ter.101) | **CONTREDITE** | taux de porteur Needler **0,0067** (vaudrait ~1) ; cadence inter-record **83,4 ms**, q90 100,1 = son temps de cycle |
| Les touches de projectile sont dans les records de dégât | **FERMÉE** | filtre d'arme levé : Fiesta **0,1729** des touches API contre **0,8556** en Tactical — 31 400 touches sans porteur |
| Les stats 0x0D / 0x0E arrivent dans le film | **FERMÉE, avec sa CAUSE** | le push existe mais derrière `DAT_1451222c8`, **4 références dans tout le binaire, toutes en LECTURE** |
| Déconvolution du taux de touche par arme (voie neuve) | **INSUFFISANTE** | grain joueur, 8 562 observations, bornée [0,1] : **2 armes de contrôle sur 4** à ±0,03 |

**Ce qui a quand même progressé** : le Needler passe de **0,007** (chiffre naïf du dossier) à
**0,2238** — d'une réponse fausse d'un facteur 30 à une réponse plausible mais **non validable**.
⚠ **Aucune nulle n'a été jouée au grain joueur** : ces chiffres sont INDICATIFS, personne ne doit
les publier.

---

## 3. CE QUI RESTE OUVERT — et c'est du décodeur, pas de la piste E

**Document de travail : `.ai/V7.5/film_re/NOTE_I0_TI41_POSITION_PROJECTILE.md`** (551 lignes,
tenue au fil de l'eau, avec son journal en §5).

**La question** : `object-position-component` (`FUN_14076e29c`) de l'archétype `ti=41`. Le port
Go décode la branche « plage de la carte » (13/13/14) mais consomme la branche « plage par
défaut » comme **59 bits opaques** — l'alignement tient, **aucune position n'en sort**.

**Ce qui est acquis :**

- le projectile **est** une entité répliquée, et le flux delta porte ses trajectoires :
  **1 388 records `ti=41` sur 281 slots en 6 films, soit ~4,9 positions par projectile** ;
- **les deux branches sont empruntées à parts comparables** (264 / 277 sur 8 films) : l'opaque
  concerne la MOITIÉ des records projectile, ce n'est pas un cas marginal ;
- la grammaire de la branche opaque est établie et la forme fermée du dossier est **confirmée
  dans le binaire** : `W = min(26, bitLen(ceil(extent / (2·step))))`, plafond `0x1a` ;
- la plage de cette branche est **`DAT_143b8c6d0` = ± 100** — ⚠ **pas** `DAT_143b8c6b8` (± 20000)
  que citait le dossier, deux entrées voisines à 0x18 (un AABB) d'écart ;
- **mais le pas vient d'une globale de runtime** : les largeurs ne sont **pas** dérivables au
  désassembleur.

**LE PROCHAIN GESTE, ET IL NE DEMANDE NI GHIDRA NI FEU VERT** — c'est une lecture, pas une
recherche :

1. **Décoder `.ai/V7.5/dumps/ce_prec_widths_1445cc9e0.bin`** (4 096 o) et
   `ce_prec_ranges_14462cbe0.bin` (18 432 o) → les largeurs de `ti=41` sur la carte de capture.
   *La globale que je déclarais non dérivable est dumpée depuis juillet.*
2. **Confronter le découpage à `.ai/V7.5/dumps/ce_pos_oracle.csv`** — 46 790 lignes,
   colonnes `eid,slot,bitCursor,x,y,z`. La colonne **`bitCursor`** en fait une pierre de Rosette :
   égalité de position contre une vérité terrain, pas heuristique de forme.
   ⚠ **PORTÉE VÉRIFIÉE : les 32 slots sont dans la bande 528-559 — des BIPÈDES, aucun
   projectile.** L'oracle **calibre l'instrument**, il ne tranche pas la branche opaque.
3. Généraliser par la forme fermée **seulement si (1) et (2) passent**. ⚠ Réserve du dossier
   conservée : les tables dumpées sont un instantané **PAR CARTE** (7ter.54 AXE3 — injectées
   telles quelles, elles effondrent 3 films sur 4). Elles valent comme **oracle**, pas comme
   constante du décodeur.
4. Ensuite seulement : lever la couverture du flux delta (le localisateur de boucle de
   `killsource`, `locateStrict` + repli, 690/690 paquets, **n'est pas exporté** — c'est le verrou
   de volume), puis le chaînage sur `i1`, puis le portage.

**RIEN NE SE PORTE DANS `traverse.go` AVANT (2) ET (4).** Ce composant est sur le chemin de tous
les archétypes qui le portent : une régression y est silencieuse.

---

## 4. CE QUI A ÉTÉ CORRIGÉ DANS LE DOSSIER — vérifier avant de citer l'ancien

| document | correction |
|---|---|
| `ETAT_DE_L_ART_KILLWEAPON.md` | `RoundsCorrected` écrit à **`entry+0x10`**, pas `+0x08` (huit octets, et ils changent le sens) · le `NON TESTE` sur la sommation est **levé** · ligne NEUVE sur la porte `DAT_1451222c8` · la ligne « TIR ou TOUCHE » marquée **CONTESTÉE** avec ses deux mesures · entrée de motifs de grep pour la piste E |
| `GUIDE_WEAPON_SHOTS.md` | §1.1 et §3quater.1 marqués contestés **avec leur conséquence pratique écrite : AUCUNE** pour la porte de publication · §3bis.1 porte « timebox clos, ne pas ré-ouvrir sans raison neuve » |
| `PLAN_MASTER_FILM_KILLFEED_REJEU.md` | décision #6 **close** avec son issue |
| `HANDOFF_PRECISION_PROJECTILES.md` (2026-07-31) | **consommé** — bandeau posé, il renvoie ici |

**Règle appliquée partout** : le contestataire est marqué `[MESURE]` non reproduit par un tiers,
et **les énoncés d'origine sont conservés à côté**. Deux formulations coexistent explicitement,
jamais une substitution silencieuse.

**Reste dû** : faire vérifier par un contexte frais la contradiction de §3 du verdict
(« le record est un TIR pour toutes les armes ») avant de retirer la ligne d'origine de 7ter.101.

---

## 5. L'OUTILLAGE — 5 binaires, archivés et rejouables

`apps/go-api/cmd/tmp_*` est **gitignoré** (`.gitignore:311`) ; les sources sont sous
**`.ai/V7.5/outillage/precision_projectiles/`** avec un README qui porte les requêtes d'export
des deux CSV de référence. `CGO_ENABLED=0`, aucune dépendance DuckDB.

| outil | ce qu'il mesure | coût |
|---|---|---|
| `tmp_pjcnt` | 5 modes : gate d'alignement du compteur · classes d'en-tête · **par arme** (taux de porteur, pas moyen, cadence) · déconvolution grain match · **grain joueur borné + référence API par arme** | 50 ms/film ; grain joueur 900 films = 3 min 10 |
| `tmp_ti41` | recensement des archétypes du monde de **keyframe** | ~12 films/min |
| `tmp_ti41d` | recensement des archétypes du flux **delta** | ~6 films/min |
| `tmp_i0w` | échantillons i0 de `ti=41` : split des branches, profil de bascule, colinéarité | ~8 films/9 min |

**GATE OBLIGATOIRE avant `tmp_pjcnt -joueur`** : `-pigate`, qui confronte le résolveur rapide
`resolvePIFast` à celui du dépôt. Attendu `desaccord=0` — **il a déjà rattrapé une erreur réelle**
(277 accords contre 299 désaccords sur la première version : le dépôt retient la première
occurrence *en position de bit*, la mienne la première *de chaque décalage*).

---

## 6. LES CINQ PIÈGES PAYÉS DANS LA JOURNÉE — ils valent autant que les résultats

1. **Conclure sur une source périmée, deux fois.** « Les compteurs ne sont pas répliqués » —
   démenti par une ligne de l'index que je n'avais pas grepée. « Le binding World n'est pas
   résolu » — je citais un document du 13 juin, antérieur à la résolution du chantier.
   **Règle : greper l'index avant d'ÉCRIRE une conclusion, pas seulement avant d'ouvrir une
   piste ; et vérifier la DATE de ce qu'on cite. Préférer le CODE au document.**
2. **Modéliser des branches qui ne s'exécutent jamais.** Trois occurrences du même motif dans ce
   binaire en une journée (porte du tag des codes 6/7, `DAT_1451222c8`, `FUN_14076f91c`).
   **Règle : pour toute branche vue dans un décompilé, établir si sa condition vient du BITSTREAM
   ou d'une GLOBALE avant de la porter.**
3. **Sur 4 « écarts » annoncés contre le code de production, 3 étaient à moi.** Le port était
   juste ; c'est ma lecture des gardes qui ne l'était pas.
4. **Concevoir un discriminant physique sans le soumettre à qui connaît le jeu.** Ma colinéarité
   supposait des trajectoires longues et droites : elles sont **souvent courtes** (un critère
   relatif se disqualifie sur la population la plus fréquente) et **légèrement courbées** (la
   droite était un modèle faux, pas serré). **Un critère de validation physique se valide par la
   connaissance du terrain AVANT d'être codé.**
5. **Un négatif dont la nulle vaut zéro est un négatif sur l'INSTRUMENT.** T3 a rendu zéro
   partout, nulle comprise — le dossier nomme déjà ce piège en §20.1. **Tant qu'un instrument n'a
   pas montré qu'il sait dire OUI quelque part, ses zéros ne valent rien.**

---

## 7. CE QU'IL NE FAUT PAS REFAIRE

- chercher le **dénominateur** : il n'a jamais manqué, c'est le compte de records ;
- chercher les touches de projectile dans les **records de dégât** (mesuré : 0,1729 contre 0,8556)
  ou dans les **codes 6/7** (ni arme ni tireur, 168 380 observations) ;
- traiter le record de projectile comme une **touche** (surestime la précision d'un facteur ~4) ;
- la **déconvolution au grain match sans normalisation de visibilité** (coefficients
  inintelligibles : MA40 0,53, Sniper 5,04) ;
- **conclure « fermé » depuis l'absence d'un compteur AGRÉGÉ** : c'est le saut qui avait fait
  rater l'arme du kill — l'agrégat manquait, l'information par événement était là ;
- lire des frontières de champ dans un **profil de bascule sur un objet rapide** : la prémisse de
  `DetectI0Layout` (une valeur qui bouge peu entre frames) ne s'applique pas aux projectiles.

---

## 8. ÉTAT GIT

```
branche   research/v75-precision   (poussée, 11 commits depuis origin/main, worktree propre)
merge     AUCUN — et il n'y a rien à merger : que des .ai/ et de l'outillage archivé
base      AUCUNE écriture
prod      AUCUNE action
```

Les 11 commits racontent la journée dans l'ordre, y compris les trois qui corrigent mes propres
conclusions (`f6f631a46`, `4ede8d7d6`, `6c5ad966f`).
