# CE QUE VOUS DEVEZ FAIRE AVANT DE CHANGER DE PC

> Écrit le 2026-07-31. Liste d'actions, dans l'ordre. Chaque étape dit **ce qu'on perd si elle
> n'est pas faite** — parce que sans Cheat Engine, certaines portes se referment définitivement
> sur ce build.
>
> Durée totale estimée : **une heure**, dont 15 minutes vraiment critiques.

---

## POURQUOI L'ORDRE COMPTE

La table des désérialiseurs est **le préalable de tout le reste**. Sans elle, les composants
d'objectif ne sont pas lisibles — et une capture de match à objectif, aussi belle soit-elle,
ne servirait à rien puisqu'on ne saurait pas décoder ce qu'elle contient.

```
1. Table des desers  ──►  2. Décompilation Ghidra  ──►  3. Capture oracle sur un match
   (15 min, jeu lancé)      (hors ligne, sans jeu)        (validation, jeu lancé)
```

Les étapes 1 et 3 exigent le jeu **et** Cheat Engine. L'étape 2 se fait tranquillement après,
sur le nouveau PC, avec le projet Ghidra qui est déjà sur la clé.

---

## ÉTAPE 1 — LA TABLE DES DÉSÉRIALISEURS  ⏱ 15 minutes  ⚠ CRITIQUE

**Sans elle, on perd définitivement** : les objectifs vivants (0 composant sur 34 lisible
aujourd'hui), les zones (0 sur 33), la moitié des véhicules (32/48) et des dispositifs (18/41).
C'est-à-dire *exactement* ce que vous voulez afficher : qui porte le drapeau, où en est une
capture, qui tient le crâne.

**À faire :**

1. Lancer Halo Infinite, entrer dans **n'importe quel** film en mode Théâtre.
   La carte et le mode n'ont **aucune importance** — la table est une constante du build.
   Mais il faut être **dans un film** : au menu, le registre vaut 0 et le script vous le dira.
2. Cheat Engine → ouvrir le processus `HaloInfinite.exe`.
3. *Table* → *Show Cheat Table Lua Script* → coller `filmdec_deser_table.lua`
   (racine de `E:/LevelUp_rejeu2D/`, aussi dans le dépôt sous `tools/ce/`) → **Execute**.
4. **Vérifier le contrôle imprimé** : l'archétype 35 doit rendre **64 composants**. Si ce
   compte ne tombe pas, la structure a bougé et toute la table est suspecte — dites-le-moi
   plutôt que de la garder.
5. Copier la sortie `C:\Users\Guillaume\Downloads\deser_table.tsv` sur la clé, dans
   `E:/LevelUp_rejeu2D/`.

Le script ne fait que **lire de la mémoire** : aucun hook, aucune cave, aucune interception de
flux. C'est la plus sûre des captures.

---

## ÉTAPE 2 — UN MATCH À OBJECTIF SUR CATALYST  ⏱ 20 minutes  ⚠ IMPORTANT

**Pourquoi Catalyst précisément**, et pas une autre carte :

- c'est une des **14 cartes déjà cataloguées** (bornes de déquantification présentes), donc on
  pourra construire un artefact de rejeu à partir du film ;
- vous dites qu'on y trouve **surbouclier et camouflage actif** — et c'est le coup double : ces
  deux équipements manquent à notre table de capacités, qui n'en connaît que 4 sur 11 ;
- on a déjà un film CTF sur Catalyst (`64e8adfa`) qui porte **68 événements d'objectif** quand
  un Slayer en porte zéro. La carte est donc un terrain déjà mesuré.

**Ce qu'on perd sans cette capture** : l'oracle qui permettra de *valider* le décodage des
objectifs une fois les désérialiseurs portés. On pourrait décoder sans, mais sans rien pour
falsifier — et ce chantier a pour règle de ne rien publier qu'on ne puisse contredire.

**À faire :**

1. Jouer (ou trouver dans votre historique) un match **à objectif sur Catalyst** — CTF,
   Strongholds ou Total Control — où **surbouclier et camouflage** apparaissent.
2. Le rejouer en Théâtre, Cheat Engine ouvert, avec le script de capture continue
   `filmdec_full_capture.lua` (ou `filmdec_delta_capture.lua`).
3. Noter **l'identifiant du film** (8 caractères hexadécimaux) et le copier sur la clé avec sa
   capture.
4. **Important** : noter aussi, à l'œil, *qui* prend le surbouclier et *quand*, *qui* prend le
   camouflage, et les instants de capture de zone ou de drapeau. **C'est le relevé terrain** —
   sans lui, la capture est une masse de bits qu'on ne peut confronter à rien.

### Sur « Strongholds sur Nomad » — Nomad, c'est Vagabond

Vous avez raison sur le nom. Et on a déjà un film dessus : **`7344d24f`** (26 Mo), documenté
comme Vagabond. Mais deux obstacles, tous deux vérifiés :

1. **Vagabond n'est pas dans le catalogue de bornes** (14 cartes). Sans ses bornes de
   déquantification, `replay-build` refuse de produire un artefact — c'est un garde-fou
   délibéré, les bornes d'une autre carte donneraient des coordonnées fausses d'un facteur
   d'échelle arbitraire.
2. **Aucun module de `deploy/ds` ne porte visiblement son nom.** Les 31 modules copiés sont
   `catalyst`, `chasm`, `ridgeline`, `forest`, les `ctf_*`, `btb_*`, `sgh_*`, `va_*`, plus des
   `fo*` qui ressemblent à des toiles Forge. Son module reste donc **à identifier**, par la
   méthode du `level_id` qui a déjà résolu 21 niveaux (21/21, 0 sur 2 témoins).

**Conséquence pratique** : une capture sur Vagabond reste utile pour la **table des capacités**,
qui ne dépend pas de la carte — mais on ne pourra pas en tirer un rejeu tant que son module
n'est pas identifié. **Si vous avez le choix, préférez Catalyst**, où tout est déjà en place.

*(Note : les documents affirment que `7344d24f` porte un `world_dump` Cheat Engine. Vérifié sur
disque : il n'y en a que sur `000d5950` et `9e8fb31b`. L'affirmation est fausse.)*

---

## ÉTAPE 3 — SI VOUS AVEZ ENCORE DU TEMPS  ⏱ 15 minutes chacune  ○ CONFORT

Par ordre de valeur décroissante :

| capture | ce qu'elle apporte |
|---|---|
| **KOTH ou Oddball**, n'importe quelle carte cataloguée | un second mode à objectif : le témoin qui distingue « ce qui est propre au CTF » de « ce qui vaut pour tous les objectifs » |
| **Un match où vous mourez avec un marteau ou une épée entamés** | de quoi confronter le compteur de charges dérivé (10 pour le marteau, 7 pour l'épée) à ce que vous voyez à l'écran |
| **Un match sur une carte non cataloguée** que vous jouez souvent | pour élargir le catalogue plus tard |

Aucune de ces trois n'est bloquante. La première seule mérite qu'on s'y attarde.

---

## CE QU'IL NE SERT À RIEN DE CAPTURER

Autant le dire, pour ne pas y passer la soirée :

- **Les médailles.** L'API les lit très bien, et les visuels sont un mapping à reprendre de nos
  pages existantes. Le film porte bien un code de médaille qui se perd à l'écriture, mais c'est
  une redondance, pas un manque. *(Mon audit avait surestimé ce point.)*
- **L'arme par kill, l'assistant, la part de dégâts.** C'est le domaine du chantier voisin, et
  son décodeur est livré. Deux décodeurs du même fait divergeraient.
- **Les positions, les trajectoires, l'inventaire.** Le décodage tient hors ligne depuis le
  29 juillet, contrôle terrain 8/8.

---

## RÉCAPITULATIF — LA LISTE, SANS LE RESTE

- [ ] **1.** Lancer le jeu → un film en Théâtre → Cheat Engine → `filmdec_deser_table.lua` →
      vérifier « archétype 35 = 64 composants » → copier le `.tsv` sur la clé. **15 min.**
- [ ] **2.** Un match à objectif sur **Catalyst** avec surbouclier et camouflage → capture
      continue → noter l'identifiant du film **et le relevé à l'œil**. **20 min.**
- [ ] **3.** *(optionnel)* Un KOTH ou un Oddball sur une carte cataloguée. **15 min.**
- [ ] **4.** Vérifier que la clé porte bien : `deser_table.tsv`, les nouvelles captures, et les
      identifiants de film correspondants dans `data/cache/film_chunks/`.

Le reste — le projet Ghidra, l'exécutable, les 31 modules de niveau, les 949 films, les bases,
les captures existantes — **est déjà sur la clé**.
