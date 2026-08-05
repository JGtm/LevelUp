# PLAN — la « belle image » : porter la chaîne des TRIANGLES en Go

> Écrit le 2026-07-31. Contrat d'exécution : skill `plan-execution`.
> Source de vérité sur la méthode : `V7.5/cartes/HANDOFF_GEOMETRIE_TRIANGLES.md` (2026-07-26). Ce plan ne
> la réécrit pas — il la met en travail ordonné, avec ses réserves.

---

## LE RÉSULTAT ATTENDU — LE VISUEL FAIT FOI  *(ajouté le 2026-08-01, arbitrage utilisateur)*

**La référence est une IMAGE VERSIONNÉE, pas une description** :
`.ai/V7.5/dumps/carte_validee_v1.png` — validée par l'utilisateur le 2026-07-26.
**Aucune étape de ce plan n'est close tant que le rendu produit ne lui ressemble pas.**

Ce que la référence porte, et qu'il faut retrouver — énuméré pour être opposable :

| ce qu'on doit voir | pourquoi c'est le critère |
|---|---|
| l'**architecture réelle** lisible à l'œil — sur la référence Cliffhanger : le fer à cheval **en anneau**, la structure circulaire centrale, les plateformes hexagonales, les deux ponts au sud. **Ces repères sont PROPRES À CLIFFHANGER** : ils servent de témoins de non-régression du portage, jamais de critère pour une autre carte (cf. gate humain, checklist (b)) | une forme attendue qui manque = échec, quel que soit le chiffre de couverture |
| la **distinction de matière** : les zones rocheuses (texture mouchetée, dense) se distinguent des plateformes construites (surfaces lisses, arêtes nettes) | c'est ce qui rend la carte lisible d'un coup d'œil ; un aplat uniforme la rend inutilisable |
| **zéro rectangle** : aucune boîte englobante visible | c'est LA différence entre les deux chaînes — voir un rectangle, c'est voir la mauvaise chaîne |
| une **vue orthographique du dessus**, fond neutre, sans décor ajouté | c'est un fond de carte, pas une illustration |

**RÈGLE D'ARBITRAGE, non négociable** : **le visuel commande, le poids s'adapte.** On choisit
la résolution qui préserve ce rendu, PUIS on résout le poids (découpage en tuiles, compression,
simplification du maillage, niveau de détail). Jamais l'inverse. Un pas choisi pour tenir dans
un budget d'octets, puis un rendu « acceptable » constaté après coup, est le chemin exact par
lequel on retomberait sur une carte en gros blocs — c'est-à-dire sur ce qu'on remplace.

### GATE HUMAIN — un artefact à valider, à chaque image produite

**S'applique à toute étape qui produit une image** (1.2, 3, 4, 5, 6) **et à chaque carte
nouvellement traitée.** Les mesures numériques (82,0 % des positions à moins de 25 cm,
12,9 m² de vide dans le fer à cheval, etc.) restent des garde-fous **nécessaires et non
suffisants** : elles disent que la géométrie est juste, pas que la carte est lisible.

1. **Produire un ARTEFACT DE REVUE** — une page publiée (outil `Artifact`), pas un fichier
   perdu dans un scratchpad. Elle porte, **par carte** : le rendu produit et
   `carte_validee_v1.png` **côte à côte, à la même échelle**, la carte nommée par ses DEUX
   noms (affiché + module), la couverture mesurée, et la checklist ci-dessous en clair.
   Contrainte technique : une page publiée n'atteint aucun hôte externe — les images sont
   **embarquées en `data:` URI**. Si le poids devient déraisonnable, embarquer une version
   réduite pour la revue ET donner le chemin de la pleine résolution à côté.
2. **Rendre la main et ATTENDRE.** L'utilisateur valide **expressément**. Aucune revue
   visuelle ne se déclare passée par la session elle-même ; « ça a l'air correct » n'est pas
   une validation. (Le pilotage de navigateur n'est pas utilisé pour ça : on remet la main
   avec une URL et une checklist.)
3. **La checklist soumise — deux parties, à ne pas confondre.**

   **(a) Critères GÉNÉRAUX, valables pour TOUTE carte** :
   - l'architecture de la carte se lit-elle à l'œil (on reconnaît les lieux) ?
   - la roche / le terrain se distinguent-ils des plateformes construites ?
   - voit-on un seul rectangle (= une boîte englobante ayant fuité) ?
   - l'échelle et l'orientation sont-elles cohérentes avec les positions de joueurs ?
   - la couverture est-elle publiée, et le rendu montre-t-il ce qui manque ?

   **(b) Témoins PROPRES À CHAQUE CARTE** — nommés AVANT la revue, jamais inventés par la
   session. Pour **Cliffhanger / ridgeline**, les témoins validés le 2026-07-26 sont
   l'**anneau du fer à cheval** et les **deux ponts au sud** : ils valent pour cette carte
   **et pour elle seule**, ce sont les repères de non-régression du portage.
   Pour **toute autre carte** (Catalyst, Vagabond, les 12 du catalogue) : demander à
   l'utilisateur **deux ou trois repères qu'il s'attend à reconnaître** sur cette carte-là,
   ou les prendre d'une source externe (carte en jeu, rendu Reclaimer). **Ne jamais réutiliser
   les témoins de Cliffhanger ailleurs** — et ne jamais laisser la session choisir elle-même
   ses témoins après avoir vu son propre rendu : un critère qu'on choisit après coup ne
   protège de rien (même leçon que le seuil du garde local, satisfait « à la lettre » par un
   choix de films).
4. **Consigner la validation** dans le journal de ce plan : date, carte, ce qui a été validé,
   et les réserves éventuelles. **Un `[x]` posé sans validation écrite est invalide** — c'est
   la seule étape du chantier où le juge est l'œil de l'utilisateur, et elle ne se
   court-circuite pas.

**Et pour les cartes autres que Cliffhanger** (Catalyst, Vagabond, les 12 autres) : la barre
est la même. Publier une carte, c'est publier son rendu ET sa couverture — une carte à 40 %
se montre comme telle (§5.4), elle ne se publie pas comme un décor complet.

---

## CE QU'ON A, ET CE QU'ON N'A PAS

**La recette existe, elle marche, et elle est validée.** `C_carte_triangles.png` :
« Sol de Ridgeline reconstruit depuis les TRIANGLES — **10 357 instances / 28,9 M triangles /
0 non résolue** ». Témoin porté sur le rendu : **82,0 % des positions joueur à moins de 25 cm du
sol reconstruit, contre 22,8 % attendus par hasard**. Les deux critères d'acceptation de
l'utilisateur sont tenus : fer à cheval **en anneau**, zone sud reliée par **deux ponts**.

**Ce qui manque n'est pas la méthode, c'est le portage.**

| | chaîne **AABB** | chaîne **TRIANGLES** |
|---|---|---|
| ce qu'elle lit | la boîte englobante de l'instance | le maillage réel |
| ce qu'elle rend | des rectangles | la géométrie |
| où | `cmd/mapstruct-build`, **en Go, en production** | **Python jetable**, hors dépôt |
| cartes | ridgeline + `sgh_streets` | **ridgeline seule** |
| dans le rejeu | **oui, c'est ce qui s'affiche** | non |

### Le dimensionnement, mesuré — ce n'est pas un gros chantier

| | lignes |
|---|---|
| `py/geo2.py` — le parseur rtgo, **le seul vrai code à porter** | **163** |
| `py/himod.py` — lecteur de module | 115, **déjà porté** (`internal/himodule`, 314 L) |
| déjà en Go et réutilisable | `himodule` 314 · `himap/instances.go` 397 · `himap/sbsp.go` 394 |

L'audit avait classé ce chantier « effort élevé, à faire en dernier ». **La mesure le contredit** :
c'est 163 lignes de logique neuve sur des fondations déjà posées.

---

## LA CHAÎNE, DE BOUT EN BOUT

Reprise telle quelle du handoff — chaque offset y est établi.

```
.module (pc/levels/multi/<carte>/<carte>-rtx-new.module)
 └─ tag sbsp : bloc « instanced geometry instances » à l'offset 420 du tag
      record de 320 o (0x140) :
        TransformScale@0 · Matrix4x4 Transform@12
        RuntimeGeoMeshReference@60 · MeshIndex@116 · BoundsIndex@118
 └─ tag rtgo désigné par RuntimeGeoMeshReference
      PerMeshData@16 (record 144 o) · Sections@64 · BoundingBoxes@104
      TotalVertexBufferCount@190 · MeshResourceGroups@196
 └─ champ racine « meshes » : le TagBlock enfant à foff = meshIndex × 60
      └─ records de « LOD render data » de 148 octets
           u16 @0x64 = index du tampon de SOMMETS
           u16 @0x8A = index du tampon d'INDICES
 └─ deux tables de descripteurs : 0x50 pour les sommets, 0x48 pour les indices
      « off » est un OFFSET D'OCTETS dans la CONCATÉNATION des entrées-ressource du tag
      (ce n'est PAS un identifiant à résoudre contre le manifeste)
 └─ sommets : u16 × 4, la 4e composante nulle
 └─ déquantification, puis placement par la transformation de l'instance
```

**Couverture obtenue** : 1 247/1 247 couples (globalID, meshIndex) reliés · 29 683 descripteurs,
**0 hors bornes** · 10 357 instances de bsp=0 · **28,88 M triangles, 0 non résolue**.

Les offsets sont **confirmés par une implémentation tierce** — `Gravemind2401/Reclaimer`,
`Reclaimer.Blam/Blam/HaloInfinite/{RuntimeGeoTag,ScenarioStructureBspTag,ModuleItem}.cs`, qui
exporte des modèles ouvrables dans Blender, donc validée par l'usage.

---

## ÉTAPE 0 — SÉCURISER, AVANT TOUT  ✅ FAIT le 2026-07-31

Les scripts vivaient dans un **répertoire temporaire de session** : `geo2.py`, `himod.py`,
`ooz/ooz.dll` (le décompresseur Kraken), `sc/world.npz` (6,2 M sommets, 9,86 M triangles),
`sc/raster.npz` (champ d'altitude, grille 5 cm), les rendus PNG — **4,3 Go** au total.

- [x] Copiés sur la clé, sous `LevelUp_rejeu2D/scratchpad_recherche/`.
- [ ] **0.1** Verser `py/geo2.py`, `py/himod.py` et les rendus de référence **dans le dépôt**,
      sous un chemin versionné. Un artefact qui fonde une méthode ne doit pas vivre hors de Git.

---

## ÉTAPE 1 — REPRODUIRE AVANT DE PORTER

On ne porte pas un résultat qu'on ne sait pas refaire.

- [ ] **1.1** Rejouer `geo2.py` sur `ridgeline` et retrouver **les mêmes chiffres** : 10 357
      instances, 28,88 M triangles, **0 non résolue**, 29 683 descripteurs **0 hors bornes**.
- [ ] **1.2** Régénérer le champ d'altitude et **refaire le rendu**. Les deux critères
      d'acceptation doivent ressortir : anneau du fer à cheval, deux ponts au sud.
- [ ] **1.3** **Attention — piège daté** : tout résultat géométrique antérieur au 2026-07-26 est
      entaché d'une **erreur médiane de 8,4 cm** (déquantification `i16 + 32768` au lieu de `u16`
      brut). Les `.npz` du scratchpad sont donc suspects **sauf régénération**. Vérifier la date,
      ou refaire.

**GATE 1** : les chiffres tombent, l'image ressort. Sinon, comprendre l'écart avant d'aller plus
loin — porter un décodage qu'on ne reproduit pas serait porter un accident.

---

## ÉTAPE 2 — RÉGLER LES DEUX POINTS OUVERTS  *(avant le portage, pas après)*

Le handoff les nomme explicitement : **la passe de réfutation n'a jamais tourné**.

### 2.1 Le témoin de non-régression n'est pas comparable

| fond | témoin publié |
|---|---|
| boîtes (AABB) | 80,6 % à moins de **5 cm** |
| triangles | 82,0 % à moins de **25 cm** |

**Un seuil cinq fois plus large.** Les deux nombres **ne se comparent pas**, et on ne peut donc
pas affirmer que rien n'a régressé.

- [ ] **2.1** Refaire la mesure des triangles **à 5 cm**, et la comparer à 80,6 %. Publier les
      deux seuils côte à côte. Si le résultat est en dessous, c'est une régression — et il vaut
      mieux le savoir avant d'avoir tout porté.

### 2.2 Les bornes de déquantification : deux lectures possibles, une seule juste

- **Reclaimer** place les bornes dans un bloc `BoundingBoxes` **unique par tag** (84 octets,
  trois paires parfaitement symétriques : ±2,6357 en X, ±1,7432 en Y, ±3,0514 en Z), désigné par
  `BoundsIndex` à +118 de l'instance.
- **Notre reconstruction** lit des bornes **par maillage**.

- [ ] **2.2** Trancher. Les deux produisent des chiffres ; une seule est juste. Le juge :
      l'écart aux bornes connues (**5,8 mm** avec la bonne déquantification) et le rendu final.

**GATE 2** : les deux points sont tranchés par une mesure écrite. Ce sont les seules réserves
que le handoff laisse ouvertes — les traîner dans le portage les rendrait dix fois plus chères.

---

## ÉTAPE 3 — LE PORTAGE EN GO

Cible : **`cmd/mapstruct-build`** — c'est lui qui produit l'asset figé, c'est lui qui doit
évoluer. Réutiliser `internal/himodule` (lecteur de module, `dataOffset` 48 bits et drapeau
`UseHd1` **déjà portés**) et `internal/himap` (instances, `LocalToWorld` en convention
vecteur-ligne).

- [ ] **3.1** Porter le parcours du §1 : sbsp → rtgo → `meshes` → LOD render data → descripteurs
      de tampons → sommets.
- [ ] **3.2** **Les trois corrections, en dur dans le code et dans les commentaires** :
      1. sommets en **`u16` brut**, jamais `i16 + 32768` (5,8 mm contre 84 mm d'écart) ;
      2. le chaînage ne passe **pas** par `@0x88`/`@0x8c` — **réfuté, 0,0 % de résolution** :
         c'est un hash 64 bits par maillage. Le vrai chemin est le champ racine `meshes` ;
      3. **retirer** le « critère de validation en or » par l'AABB : il est **tautologique**
         (écart médian 0,0000 m, et il ne départage pas un bon tampon d'un tampon tiré au
         hasard — 0,19 m contre 0,22 m). Il a probablement masqué des chaînages faux.
- [ ] **3.3** Poser **T1** à sa place, comme garde-rail de test :
      > l'indice maximal du tampon d'INDICES doit être **strictement inférieur** au nombre de
      > sommets du tampon apparié.
      **100 %** pour le bon appariement, **5,1 %** pour un LOD voisin. C'est lui qui discrimine.
- [ ] **3.4** Second garde-rail : **0 descripteur hors bornes** sur 29 683.

**GATE 3** : la sortie Go est **identique** à la sortie Python sur ridgeline — mêmes comptes,
même champ d'altitude au centimètre près.

---

## ÉTAPE 4 — LE FORMAT DE SORTIE : un CHAMP D'ALTITUDE, pas des boîtes

**Constat qui simplifie tout** : le client construit **déjà** un champ d'altitude
(`mapFloor.ts`, trame de 25 cm, altitude la plus haute par cellule) — mais il le fabrique **à
partir de boîtes**, ce qui est précisément la partie grossière.

Le Python, lui, produit **directement** un champ d'altitude (`raster.npz`, grille de 5 cm,
`marchable_zmax` / `marchable_sol`).

**Donc : déplacer la rastérisation du client vers le build, et la nourrir de triangles.**

- [ ] **4.1** Le fichier de structure porte un **champ d'altitude** : origine, pas, dimensions,
      et les altitudes. Les emprises AABB restent, en repli, pour les cartes non traitées.
- [ ] **4.2** **Arbitrer le pas — SOUS LA RÈGLE VISUELLE** (cf. « LE RÉSULTAT ATTENDU » en
      tête de plan) : le pas se choisit sur ce qu'il PRÉSERVE du rendu de référence, pas sur
      ce qu'il coûte. Méthode : produire le rendu à plusieurs pas (5, 10, 25 cm), les mettre
      côte à côte avec `carte_validee_v1.png`, retenir le plus grossier qui garde encore
      l'anneau, les ponts et la distinction roche/plateforme — **et si aucun ne tient dans le
      budget, c'est le budget qui cède** (tuiles, compression, niveau de détail), pas le
      rendu. Ordres de grandeur pour situer : à 5 cm sur la zone jouée de Cliffhanger
      (≈ 50 × 55 m), ~1,1 M de cellules ; à 25 cm, ~44 000 ; le fichier actuel fait 677 Ko.
      Publier le pas DANS le fichier plutôt que de le supposer.
- [ ] **4.2bis** **Un champ d'altitude seul peut ne pas suffire.** La référence donne à voir
      des arêtes et une différence de matière que la seule altitude par cellule ne porte pas.
      Si le côte-à-côte de 4.2 montre l'écart, prévoir ce qui le comble — ombrage par la
      normale, canal de pente, ou conservation d'un second canal — **et le décider sur
      l'image**, pas sur une intuition de format. C'est le point où le plan d'origine
      supposait l'équivalence sans l'avoir vérifiée.
- [ ] **4.3** Côté web, `mapFloor.ts` **se simplifie** : il peint un champ reçu au lieu de
      rasteriser 10 223 boîtes. Garder le chemin AABB tant que des cartes n'ont que ça.
- [ ] **4.4** Les tests existants de `mapFloor` couvrent la rastérisation : les faire porter sur
      le **nouveau** contrat sans perdre ce qu'ils protègent (exclusions plafond/dalle/mobilier,
      étalonnage aux centiles, arêtes).

**GATE 4** : le rejeu affiche le sol par triangles sur Cliffhanger, et le repli AABB ailleurs.
**Revue visuelle** — c'est le seul juge qui compte pour une image.

---

## ÉTAPE 5 — LES 29 AUTRES CARTES

- [ ] **5.1** Passer les **14 cartes du catalogue**. `mapstruct-build` tourne déjà sur les 14 en
      ~2 minutes par la chaîne AABB ; mesurer ce que la chaîne des triangles y donne.
- [ ] **5.2** **Catalyst — la question ouverte.** La chaîne AABB y plafonne à 40-49 %. La chaîne
      des triangles lit le maillage réel, mais **elle passe par le même bloc d'instances** : si
      la géométrie de Catalyst n'est vraiment pas instanciée, elle butera pareil. **À mesurer, et
      c'est le premier test à faire** — c'est une des deux cartes que l'utilisateur veut.
- [ ] **5.3** **Vagabond — un troisième chemin.** Carte **Forge** sur la toile `fo08_wetland` :
      son sol vit dans les **objets placés du `.mvar`**, pas dans le BSP. Ni AABB ni triangles ne
      le donneront tel quel. À traiter à part, avec `mapvar` — qui existe et est testé.
- [ ] **5.4** Publier, par carte, **ce que la chaîne a rendu et ce qu'elle n'a pas rendu**. Une
      carte à 40 % ne se publie pas comme un décor complet.

**GATE 5** : chaque carte traitée a son fichier figé et sa mesure de couverture ; les autres ont
une raison écrite.

---

## ÉTAPE 5bis — LES NOMS DE ZONES (CALLOUTS)  *(ajoutée le 2026-08-01, demande utilisateur)*

**Pourquoi ici et pas au backlog** : c'est le **même tag, les mêmes cartes, la même passe**.
Sortir les callouts séparément obligerait à re-parcourir les modules des mois plus tard. Et
surtout, ils portent le critère (a) du gate visuel — *« l'architecture se lit-elle à l'œil,
reconnaît-on les lieux ? »* : une carte sans ses noms reste un dessin, avec eux elle devient
utilisable.

**Ce qui est déjà établi** (`../V7.5/cartes/INVESTIGATION_MAP_ZONE_CALLOUT_NAMES.md`,
2026-06-26) et qu'il ne faut pas re-chercher :

- les noms ne sont **PAS dans le film** (le rejeu ne réplique que le dynamique) ;
- ils ne sont **PAS dans la variante `ds/`** — celle qu'on utilise pour la géométrie ;
- ils sont dans la variante **`any/`** du module de carte (tag scénario / placements
  gameplay), **même espace monde que la géométrie**, lisibles avec l'infra existante
  (`himodule` + `ooz` + parcours d'arbre de tag). Verdict de l'investigation : « mini-chantier
  neuf mais **faible risque** ».
- **`callout_zones_ridgeline_clipped.json`** existe déjà (Cliffhanger, 16 libellés FR), c'est
  ce que le POC affiche — il sert de **témoin de non-régression** pour la voie générale.

**Conséquence matérielle, traitée le 2026-08-01** : la clé ne portait que `ds/levels/multi`.
Les 31 modules de `any/levels/multi` (0,90 Go) y ont été copiés — sans eux, ce lot serait
**mort avec le PC**, comme la palette Forge l'aurait été.

- [ ] **5bis.1** Lire les zones nommées d'une carte depuis la variante `any/`, et retrouver
      les **16 libellés de Cliffhanger** (témoin de non-régression, comparer au JSON existant).
- [ ] **5bis.2** Étendre aux cartes traitées par l'étape 5. **Règle valable pour toutes les
      cartes** (contrainte utilisateur) — pas d'ajustement carte par carte.
- [ ] **5bis.3** **Découper par ÉTAGE** (on a `top`/`bottom` par prisme) et **conserver le
      polygone BRUT à côté du découpé** : le brut est la donnée du jeu, le découpé notre
      interprétation. Les deux voyagent ensemble, jamais l'un sans l'autre.
- [ ] **5bis.4** Publier la couverture par carte : combien de zones nommées, combien sans nom.
      Une zone sans nom reste **sans nom** — jamais un libellé approchant.
- [ ] **5bis.5** i18n FR + EN pour les libellés exposés à l'écran ; aucun nom en dur côté Go.

**GATE 5bis** : le gate humain s'applique — l'artefact de revue montre les noms **posés sur la
carte**, et la checklist (b) devient, pour Cliffhanger, « retrouve-t-on les 16 zones du POC,
au bon endroit ? ».

---

## ÉTAPE 6 — LES RETOUCHES VISUELLES

L'utilisateur a annoncé vouloir intervenir sur les **couleurs et motifs**. À ne faire qu'ici,
une fois la géométrie juste.

- [ ] **6.1** L'échelle d'altitude actuelle est étalonnée aux **centiles 2/98** des altitudes
      reconstruites — à revoir avec de vraies altitudes de triangles, plus fines que des boîtes.
- [ ] **6.2** Tokens sémantiques uniquement, thèmes clair et sombre.
- [ ] **6.3** Les arêtes : aujourd'hui elles viennent des ruptures d'altitude entre cellules
      (seuil 45 cm). Avec des triangles, on peut faire mieux — mais ce n'est pas obligatoire.

---

## PORTES DÉFINITIVEMENT FERMÉES — ne pas les rouvrir

Le handoff les a fermées **avec un chiffre**. Les rouvrir coûterait des jours pour rien.

| piste | pourquoi |
|---|---|
| le compagnon `.module_hd1` | **Preuve d'absence** : trois égalités arithmétiques exactes montrent que les `rtgo` sont **toutes** dans le module principal, **zéro** déportée. Le compagnon ne contient que **59 tags `bitm`** (textures) |
| « int16×4 à 4e composante nulle » comme détecteur de sommets | **tautologique** : un bloc DXT1 plat rend `w == 0`. Une **texture** atteint 0,915 |
| la saturation de plage comme critère de maillage | **tautologique** : span médian 0,888 pour les vrais tampons, **0,890** pour des plages tirées au hasard |
| la collision `scgt` lue en `float32` brut | **faux positif** : 95-97 % des points à moins d'un mètre de l'origine — le nuage est une croix. *Leçon générale : toujours **DESSINER** un résultat, jamais seulement le compter* |
| les boîtes, alignées **comme** orientées | **0,00 m² de vide** dans le fer à cheval dans les deux cas. L'anneau vit dans les triangles, et nulle part ailleurs |
| la collision comme voie de secours | secondaire : les triangles de rendu suffisent. Et **195 des 552 modèles vivent dans des modules GLOBAUX partagés** — toute chaîne « une carte = un module » y serait incomplète |

---

## CE QUE LE RÉSULTAT VAUT DÉJÀ — la barre à ne pas descendre

| mesure | valeur | référence |
|---|---|---|
| vide dans la zone du fer à cheval | **12,9 m²** | 0,00 m² en boîtes |
| désertion du disque central | **×63,8** | ×64 mesurée sur les seules trajectoires |
| rang du centre réel | **99,7 %** | sur 5 862 disques tirés au hasard |
| centroïde du vide | **0,39 m** du centre attesté | — |
| positions joueur sous 25 cm du sol | **82,0 %** | 22,8 % au hasard |

**Et une nuance à retenir** : le trou du fer à cheval **n'est pas dans le plancher**. Sous le
disque de rayon 1 m, la surface projetée des triangles est couverte à **100,0 %**. L'obstacle est
un **bloc plein** de z = −0,8 à +2,0 m, et le sol de l'étage inférieur s'arrête net à son
aplomb. Ce que l'utilisateur voit est juste ; sa cause est un **volume**, pas une absence.

---

## PROTOCOLE DE REPRISE

1. Lire `V7.5/cartes/HANDOFF_GEOMETRIE_TRIANGLES.md` **en entier** — 163 lignes, et chaque offset y est
   établi. *(Ne pas le citer sans l'ouvrir : c'est l'erreur qui a coûté cet aller-retour.)*
2. Étape 0.1, puis 1, puis 2. **Ne pas porter avant d'avoir réglé les deux points ouverts.**
3. Les garde-fous sont déjà mesurés : **T1 à 100 %**, **0 descripteur hors bornes**.
