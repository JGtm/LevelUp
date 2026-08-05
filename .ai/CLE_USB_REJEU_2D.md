# CLÉ USB « PNY » — ce qui a été copié pour le rejeu 2D, et pourquoi

> Écrit le 2026-07-31, avant changement de PC. La clé est montée en `E:`.
> Ce fichier est aussi copié à la racine de `E:/LevelUp_rejeu2D/` sous le nom `LISEZ-MOI.md`.

---

## POURQUOI CETTE COPIE EXISTE

Le nouveau PC **n'aura pas Cheat Engine**. Or une partie de ce qui fonde le décodage du rejeu
vient de **captures mémoire** faites avec lui : elles ne se régénèrent pas, elles se
transportent. Le reste (les films, les données de référence, l'artefact) se régénère mais
coûterait des heures de retéléchargement ou exigerait les fichiers du jeu installés.

**La distinction qui commande tout** :

| nature | régénérable sans Cheat Engine ? | conséquence |
|---|---|---|
| captures mémoire (`captures_cheat_engine/`) | **non** | à emporter, irremplaçable |
| films (`data/cache/film_chunks/`) | oui, par retéléchargement | à emporter par confort |
| données de référence de carte | oui, **mais exige le jeu installé** | à emporter |
| artefact de rejeu construit | oui, en 1 minute | à emporter pour démarrer sans rien construire |

---

## CE QUI A ÉTÉ COPIÉ

### `E:/LevelUp_rejeu2D/captures_cheat_engine/` — 164 Mo, 40 entrées

Le contenu de `.ai/V7.5/dumps/` du worktree. **C'est la partie irremplaçable.**

Y figurent notamment la capture de tous les bipeds d'un match (`allbipeds_capture_sample.txt`),
les relevés de largeurs et de plages de composants (`ce_prec_widths_*.bin`,
`ce_prec_ranges_*.bin`), l'oracle de positions (`ce_pos_oracle.csv`), la rosetta
(`ce_pos_rosetta.csv`), le dump du monde (`ce_world_dump.txt`) et le tampon d'image-clé vivant
(`keyframe_buffer_live.bin`).

Ces fichiers sont ce qui a permis d'**établir** le décodage. Ils ne servent plus au
fonctionnement — le rejeu tient hors ligne depuis longtemps — mais ils sont la seule pièce à
conviction si un décodage doit être remis en cause.

### `E:/data/titles/halo_infinite/reference/` — 1,8 Mo — **C'ÉTAIT ABSENT, ET C'EST BLOQUANT**

- `map_quant_bounds.json` (4 Ko) — les bornes de déquantification **par carte**. Sans lui,
  `replay-build` **refuse de produire un artefact** : appliquer les bornes d'une autre carte
  donnerait des coordonnées fausses d'un facteur d'échelle arbitraire, sans que rien ne le
  signale.
- `map_structure/` — le sol reconstruit, un fichier par module. Deux cartes seulement :
  `ridgeline.json` (Cliffhanger) et `sgh_streets.json` (Streets).
- `map_objectives.json` (376 Ko) — les objectifs de carte.

**Ces fichiers exigent les fichiers du jeu installé pour être regénérés** (`cmd/mapquant-build`,
`cmd/mapstruct-build`). C'est pour cela qu'ils comptent plus que les films.

### `E:/data/cache/replays/halo_infinite/000d5950.json` — 2,1 Mo — **REMPLACÉ**

La clé portait une version du 11 juillet de **76 Ko** : l'artefact d'origine, qui ne contenait
que des trajectoires. Celui-ci porte en plus la structure de carte, les tirs, les lancers, les
projectiles, l'inventaire, le roster et la couverture.

### `E:/data/cache/film_chunks/` — 949 films

La clé en avait 942 ; les 7 manquants ont été ajoutés, dont **`9e8fb31b`**, qui est le film de
la capture Cheat Engine — sans lui, la capture n'a rien à quoi se rapporter.

**Les films de référence du chantier** :

| film | ce qu'il démontre |
|---|---|
| `000d5950` | Cliffhanger, Super Fiesta Slayer. **Le film de référence** : toutes les mesures publiées viennent de lui |
| `01e1f945` | Catalyst, Slayer standard. Le témoin qui montre qu'un accord à 99 % ne prouve rien quand tout le monde porte la même arme |
| `64e8adfa` | Catalyst en CTF. Prévu pour un second POC (objectifs, drapeau) |
| `9e8fb31b` | le film de la capture mémoire |
| `9b191a7f` | cité par le guide du chantier voisin |

---

### `E:/data/titles/halo_infinite/warehouse/` — les bases DuckDB — **RAFRAÎCHIES**

**CORRECTION D'UNE ERREUR QUE J'AVAIS ÉCRITE ICI.** J'avais noté que les bases n'étaient pas
nécessaires au rejeu. C'est faux pour la **page** : le canvas tient hors ligne, mais les fiches
d'équipe joignent le film et la base **par xuid** pour obtenir l'équipe et les compteurs du
match (`replay.tsx` appelle `useMatchView`). Sans base, les fiches sont vides.

Deux bases étaient **périmées** sur la clé et ont été remplacées : `metadata.duckdb`
(48 → 52,7 Mo) et `shared_social.duckdb` (19,7 → 22,3 Mo). Les deux autres étaient à jour.
La copie a été faite **API éteinte** — copier un fichier DuckDB pendant qu'un process l'écrit
peut en capturer un état déchiré.

### `E:/LevelUp_rejeu2D/POC_reference_rejeu2D.html` — 4,1 Mo

La page qui sert de **référence d'écran** à tout le chantier. Elle vivait dans un répertoire
temporaire Windows — donc à un cheveu d'être perdue au prochain nettoyage.

### `E:/data/cache/replays/halo_infinite/01e1f945.json` — 1,4 Mo

Le second artefact construit (Catalyst, Slayer). Utile comme **deuxième film** pour le critère
de retrait du garde local, qui exige une confrontation sur deux cartes différentes.

---

## CE QUI N'EST **PAS** SUR LA CLÉ, ET CE QU'IL FAUT EN PENSER

- **Les fichiers du jeu Halo installé** (`D:/SteamLibrary/.../deploy/pc/levels/multi`). Sans
  eux, impossible de produire les bornes ou la structure d'une **nouvelle** carte. C'est la
  seule vraie dépendance qui reste, et elle pèse sur tout le lot « toutes les cartes ».
- **L'historique Git.** Attention, piège : le `.git` de `LevelUp-go-migration` n'est **pas** un
  dépôt, c'est un **fichier pointeur** vers `C:/Users/Guillaume/Downloads/Scripts/LevelUp/.git`,
  lui-même **shallow** (greffe `2eb6811b`, 831 Mo d'objets). Un simple `cp` du dossier ne
  donnera rien d'utilisable. Sur le nouveau PC : cloner depuis GitHub, et recréer les worktrees.
- **Les fichiers non versionnés qu'un clone ne rendra PAS** — c'est le vrai risque, et c'est
  pour cela qu'ils sont sur la clé : les 199 `.mvar` (77 Mo, `.gitignore:258`, dont 2 seulement
  sont suivis) et les quatre `world_dump*` cachés dans `data/cache/film_chunks/` (`.gitignore:105`).

---

## POUR REDÉMARRER SUR L'AUTRE PC

```bash
# 1. Cloner le depot, puis restaurer les donnees depuis la cle
cp -r E:/data/titles/halo_infinite/reference  <depot>/data/titles/halo_infinite/
cp -r E:/data/cache/film_chunks               <depot>/data/cache/
cp -r E:/data/cache/replays                   <depot>/data/cache/

# 2. Verifier que l artefact se reconstruit a l identique (le film vit dans le depot)
cd apps/go-api
CGO_ENABLED=0 LEVELUP_REPO_ROOT=<depot> \
  go run ./cmd/replay-build --map Cliffhanger 000d5950 <depot>/data/cache/film_chunks/000d5950

# Attendu : 99 traces, 29 221 points, 475 tirs, 70 lancers, 439 projectiles, 10 223 emprises.
# Un ecart sur l un de ces chiffres = quelque chose a bouge, et il faut comprendre quoi
# AVANT de continuer.

# 3. Les captures memoire, si un decodage doit etre remis en cause
cp -r E:/LevelUp_rejeu2D/captures_cheat_engine  <depot>/.ai/re_dump
```

---

## TROIS PIÈCES QUI MANQUAIENT, ET QUI TUENT TOUT SI ELLES SE PERDENT

Trouvées par un second audit, le 2026-07-31. Elles ne sont dans **aucun dépôt**.

### `E:/LevelUp_rejeu2D/retro_ingenierie/` — 2,8 Go

- **`HI.rep` + `HI.gpr`** (2,7 Go) — le projet Ghidra. Il porte des **mois** d'analyse :
  fonctions nommées, structures posées, 369 adresses de désérialiseur identifiées. Le refaire
  demanderait de tout recommencer.
- **`HaloInfinite.exe`** (83,7 Mo) — le binaire analysé. Sans lui, le projet Ghidra ne s'ouvre
  sur rien. **Sans ces deux-là, toute résolution statique meurt.**

### `E:/LevelUp_rejeu2D/jeu_deploy_ds/multi/` — 162 Mo, 31 modules

Les modules de niveau du jeu, version serveur dédié. **Ils suffisent à `mapquant-build`** —
c'est-à-dire aux bornes de déquantification de nouvelles cartes, sans avoir le jeu installé.
C'est la dépendance qui bloquait le lot « toutes les cartes ».

### `E:/LevelUp_rejeu2D/scripts_cheat_engine/` + `filmdec_deser_table.lua`

Les 11 scripts de capture existants, **plus un neuf** — voir la section suivante.

### L'OUTILLAGE DE RÉTRO-INGÉNIERIE — ajouté le 2026-07-31

- **`ghidra_12.1_PUBLIC/`** (1,4 Go) avec son lanceur `ghidraRun.bat`. Le projet `HI.rep` ne
  s'ouvre qu'avec la **même version majeure** : emporter le binaire évite une mauvaise surprise.
- **`ghidra-mcp/`** (924 Mo) — l'outillage qui permet d'interroger Ghidra depuis l'agent.

### `E:/data/cache/film_manifests/` — 950 manifests, 119 Mo — **LA MEILLEURE ASSURANCE**

Ils permettent de **re-télécharger n'importe quel film depuis le CDN Azure sans repasser par
l'API Halo** — donc sans dépendre des tokens ni de l'authentification. Pour 119 Mo, c'est le
meilleur rapport de toute la clé.

### `E:/data/cache/replays/halo_infinite/` — trois artefacts

`000d5950` (Cliffhanger, Slayer), `01e1f945` (Catalyst, Slayer) et `64e8adfa` (Catalyst, **CTF**,
construit le 2026-07-31 : 138 traces, 50 144 points, 68 événements d'objectif).

---

## INVENTAIRE FINAL — 5,3 Go dans `LevelUp_rejeu2D/`, plus `E:/data/`

| dossier | taille |
|---|---|
| `retro_ingenierie/` (Ghidra + projet + exe + MCP) | **5,1 Go** |
| `captures_cheat_engine/` | 164 Mo |
| `jeu_deploy_ds/multi/` (31 modules) | 162 Mo |
| `POC_reference_rejeu2D.html` | 4,1 Mo |
| les plans et handoffs (`.md`) | — |
| **`E:/data/`** (films, manifests, bases, référence, artefacts) | ~54 Go |

**51 Go restent libres.**

---

## FAUT-IL REFAIRE DES CAPTURES AVANT DE DÉBRANCHER ? — **OUI, UNE SEULE, ET ELLE PRESSE**

Vérifié fichier par fichier le 2026-07-31.

**Le pipeline du rejeu ne consomme AUCUNE capture mémoire.** `cmd/replay-build` travaille
depuis les seuls chunks du film, et `internal/analysis/replay` n'ouvre que des `chunk_NN.bin`,
deux CSV de géométrie et un JSON de structure. Le verrou historique « l'inventaire dépend de
Cheat Engine » a été **levé le 2026-07-29** : le décodeur est passé hors ligne, avec un
contrôle terrain 8/8 sur les grenades, 8/8 sur les capacités, 7/7 sur chargeur et réserve.

**Aucune des quatre réserves ouvertes du chantier n'exige une capture neuve** : la cellule de
munitions est une question de décodage, `ownersFromLives` se repose sur le prochain film, les
teintes d'effets sont un choix de design, et les emprises orientées exigent le **jeu installé**,
pas Cheat Engine. Même la mesure déclarée prioritaire depuis le 2026-07-27 — relire l'index de
capacité sur 6 bits au lieu de 3 — est explicitement décrite comme **contournant le problème
sans nouvelle capture**.

### LA TABLE DES DÉSÉRIALISEURS — 15 minutes, et le script est écrit

Une donnée au monde ne s'obtient **que** par Cheat Engine : la table
`typeIndex → désérialiseur par composant`. Elle n'est ni dans le film, ni dérivable de
l'exécutable statique — elle est bâtie à l'initialisation du jeu (`DAT_144e61d88`) et vaut 0
tant que le jeu n'a pas démarré.

**Le manque n'est PAS comblé là où le chantier veut aller.** Couverture mesurée par archétype,
en croisant le registre du film avec le dispatch du décodeur :

| archétype | couvert | |
|---|---|---|
| `ti=35` biped | 62/64 | quasi complet |
| `ti=37` objets au sol · `ti=41` projectiles · `ti=42` armes au sol | 31/31 · 22/22 · 21/21 | complets |
| `ti=40` véhicules | 32/48 | **trou** |
| `ti=43` dispositifs | 18/41 | **trou** |
| `ti=23` zones | **0/33** | **rien** |
| `ti=11` objectifs | **0/34** | **rien** |

Les deux derniers sont exactement ce que vous voulez lire : **qui porte le drapeau, où en est
une capture, qui tient le crâne**. Sans cette table, aucun de ces composants ne sera lisible —
et sans Cheat Engine, elle est hors d'atteinte **définitivement** sur ce build.

**Aucun script ne la lisait.** Les 11 scripts existants hookent le flux ; la table du biped
avait été lue pointeur par pointeur à la main, ce qui ne passe pas à l'échelle des ~1 067
composants. **Il est écrit maintenant** : `filmdec_deser_table.lua`, à la racine de
`E:/LevelUp_rejeu2D/`.

**Marche à suivre, avant de débrancher :**

1. Lancer Halo Infinite, entrer dans **n'importe quel** film en Théâtre — la table est une
   constante du build, la carte et le mode n'ont aucune importance. Mais il faut être **dans un
   film** : au menu, le registre vaut 0.
2. Cheat Engine → ouvrir `HaloInfinite.exe` → *Show Cheat Table Lua Script* → coller le script
   → Execute.
3. Sortie : `C:\Users\Guillaume\Downloads\deser_table.tsv`. **La copier sur la clé.**

Le script lit uniquement de la mémoire — aucun hook, aucune cave, aucune interception. Il
imprime un contrôle à vérifier : l'archétype 35 doit rendre **64** composants. Si ce compte ne
tombe pas, la structure a bougé et la table entière est suspecte.

**Ce qu'on perd sans elle** : les objectifs vivants, les zones, et la moitié des véhicules et
des dispositifs — pour toujours sur ce build.

---

## POUR ALLER PLUS LOIN

Le plan de finalisation est dans `PLAN_FINALISATION_REJEU_2D.md` : ce qui reste à faire, dans
quel ordre, et ce qui bloque.
