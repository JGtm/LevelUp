# PLAN — remplacer les icones d'armes par celles extraites du jeu

> Ecrit le 2026-08-09, branche `feat/v75-icones`. Fait suite a `ETAT_DE_L_ART_ICONES.md`
> (extraction, nommage, gate humain) et repond a la demande : mapping code -> images,
> remplacement des anciennes images, teinte d'equipe au kill feed.
>
> Contrat d'execution : skill `plan-execution`. Ordre strict, un item = un statut, aucun report
> d'une action executable maintenant.

---

## 0. CE QUI EST DEJA VRAI (etat mesure, pas suppose)

| Fait | Ou |
|---|---|
| 168 PNG extraits du jeu, 3 atlas, zero bloc degrade | `static/weapons-assets/halo_infinite/jeu/` |
| Le lien arme -> icone est LU dans le jeu (`sprite index`), 29/29, auto-valide | `cmd/weapon-icons-build` |
| `index.json` porte deja `index -> arme (weapon_key)` et le nom interne | idem, colonne `arme` |
| Les anciennes icones sont 28 PNG nommes a la main, keyes par `name_en` | `internal/games/halo_infinite/adapter_asset_urls.go:36-76` |
| Le front ne resout RIEN : il consomme `image_url` calcule par le back | `PlayerDetailPanel.tsx:83`, `AssetCard.tsx:44` |
| Le kill feed web n'affiche AUCUNE icone d'arme aujourd'hui | `MatchTugOfWarChart.tsx:240-274` |
| Les couleurs d'equipe existent en tokens et suivent les couleurs in-game du joueur | `team-ally` / `team-enemy`, `theme-provider.tsx:27-30` |

**Trois defauts de l'existant, a corriger par ce chantier** (constates, non supposes) :

1. **La cle de resolution est `weapon_labels.name_en`**, seul reste du domaine arme encore keye
   par un nom EN brut. Il DIVERGE deja de la source canonique : `weapon_labels` dit
   « Mk51 Sidekick », `weapon_names.toml` dit « Mk50 Sidekick ». L'image ne marche aujourd'hui
   que parce qu'elle lit la table heritee.
2. **Un fichier ment sur son contenu** : `Cindershot -> Cremator.png`. Un nom de fichier n'est
   pas une source.
3. **Aucun test ne verifie qu'un PNG existe sur disque** : un renommage passe la CI et casse
   l'UI en silence.

---

## 1. DECISION — le mapping vit dans le CODE, pas en base

**Retenu : `index.json` genere + une table Go derivee, keyee par le TAG `weap`** (voir
l amendement de l etape 1 : `weapon_key` a ete essaye et refute par la mesure).

Pourquoi pas la base :

- Le lien index -> arme est une propriete de l'ENSEMBLE D'ASSETS, versionne avec les PNG. Il ne
  change que quand on re-extrait. Le mettre en base imposerait une migration + un seed pour une
  donnee deja statique et deja versionnee, et creerait une **seconde source de verite** face a
  `index.json` — que rien ne comparerait (le defaut exact des tables recopiees de la page de
  nommage, corrige au commit `5543dc170`).
- Le depot a deja le precedent : `abilities-assets/index.json` et `grenades-assets/index.json`.
- La base garde ce qui lui revient et qu'elle porte deja tres bien : le registre des armes, les
  libelles FR/EN, les identifiants filmshell. **Le plan n'y touche pas.**

Ce qui va en base : **rien**. Ce qui va en code : une table `tag weap -> index d atlas`, GENEREE
depuis `index.json` par la commande d extraction, avec un garde-rail qui echoue si une arme du
registre n a pas d icone ou si un PNG reference manque sur disque.

---

## 2. QUELLE IMAGE POUR QUEL USAGE

| Surface | Atlas | Pourquoi |
|---|---|---|
| Liste d'armes du Match View, tiroir d'assets | **contour** (~330x117) | grande, detaillee, c'est l'identite de l'arme |
| Kill feed, rejeu, futures listes denses | **kill feed** (~110x38) | dessinee pour etre lue petite, et elle couvre vehicules, grenades et pictogrammes que l'atlas d'armes n'a pas |
| Silhouette | aucune pour l'instant | meme index que le contour ; a garder en reserve pour un etat plein (survol, selection) |

**Le kill feed n'est pas un remplacement, c'est un ajout** : aucune icone d'arme n'y est affichee
aujourd'hui. A traiter comme une feature, avec son gate visuel propre.

---

## 3. LA TEINTE D'EQUIPE — faisable, avec UNE limite dure

**Verifie sur les pixels** (`killfeed-00.png` : 1638 pixels opaques, 94,7 % blanc pur ; le reste
est l'anticrenelage premultiplie) : **le dessin est porte par l'alpha, en blanc**. C'est le cas
ideal pour une teinte.

**Faisable, en CSS pur et sans nouvel asset** — un masque, pas une image :

```tsx
<span
  style={{
    backgroundColor: tokenCssVar(estAllie ? 'team-ally' : 'team-enemy'),
    maskImage: `url(${url})`, WebkitMaskImage: `url(${url})`,
    maskSize: 'contain', maskRepeat: 'no-repeat', maskPosition: 'center',
  }}
/>
```

Conforme au skill `color-tokens` : aucun hex, token semantique uniquement. Et comme
`theme-provider.tsx:27-30` ecrase `--ac-team-ally` avec la couleur d'outline choisie in-game, la
teinte suit **la vraie couleur d'equipe du joueur** sans code supplementaire.

**LA LIMITE, et elle est structurelle** : le kill feed web est un `scatter` **ECharts**
(`MatchTugOfWarChart.tsx:240-274`). Un symbole ECharts image (`symbol: 'image://...'`) **ne peut
pas etre teinte** — ECharts dessine le bitmap tel quel. Trois issues, par ordre de cout :

| Issue | Cout | Verdict |
|---|---|---|
| Rendre le kill feed en **DOM** (liste d'evenements) au lieu de symboles ECharts | moyen | **RECOMMANDE** — la teinte marche, l'accessibilite et le survol aussi |
| Superposer une couche DOM au chart | eleve | fragile (repositionnement au resize/zoom) |
| Pre-teinter les PNG a la generation | 88 icones x N couleurs | **REFUSE** — les couleurs d'equipe sont configurables par l'utilisateur : le produit cartesien n'existe pas |

A trancher avec l'utilisateur avant l'etape 4 : le kill feed reste-t-il dans le chart, ou
devient-il une liste ?

---

## 4. ETAPES

Chaque etape se clot par : gate passe, items statues `[x]`/`[~]`/`[!]`, entree thought_log,
point d'etape. Aucune ne commence avant que la precedente soit close.

### Etape 1 — la cle devient le TAG `weap` (aucun changement visuel)

**AMENDEMENT DU 2026-08-09 — la mesure a refute la premiere version de cette etape.** Elle
prevoyait de keyer par `weapon_key`. Mesure sur la metadata de prod : **42 etiquettes, 36
identifiants, 29 armes au registre, et 7 etiquettes SANS `weapon_key`** — dont MA5K Avenger et
Mutilator, qui ont une icone aujourd hui. Keyer par `weapon_key` les aurait fait disparaitre.

**La bonne cle etait deja documentee** (§1.1 de l etat de l art) : les **32 bits HAUTS** d un
identifiant filmshell sont le global tag id du `weap`. Verifie sur les donnees reelles, 6/6 :

| arme | weapon_id | tag (32 bits hauts) | index |
|---|---|---|---|
| BR75 | `0x2b1824d5_42c9679f` | `2b1824d5` | 1 |
| Diminisher of Hope (variante) | `0x841ac5e5_a730e49f` | `841ac5e5` | 16 |
| MA5K Avenger (sans `weapon_key`) | `0xf5c335df_e7232c0b` | `f5c335df` | 36 |
| Mutilator (hors registre) | `0xd7915565_42c9679f` | `d7915565` | 37 |
| Sandwich (hors registre) | `0x880fe0bc_42c9679f` | `880fe0bc` | 35 |
| Mythic Sandwich (hors registre) | `0xb7262ca1_c8fb11d0` | `b7262ca1` | 35 |

Cette cle ne depend ni d un nom, ni du registre, ni d une table produit. Elle couvre les armes
enregistrees, leurs VARIANTES et celles qui ne sont PAS au registre — trois cas que ni `name_en`
ni `weapon_key` ne couvraient ensemble.

**FAIT** : `index.json` porte desormais `tags_weap` par index, produit par un balayage de TOUS
les groupes de tags (`cmd/weapon-icons-build/weaptags.go`) — le bloc `UI display info` n est pas
propre au `weap`, et c est ce balayage qui a fait sortir l index 29.

**RESTE A FAIRE** :

- [x] `games/adapter.go` : `WeaponImageURL(weaponID int64) string` — un seul parametre, et il
      porte deja tout (le tag est dans ses 32 bits hauts). **Amende** : une seconde methode
      `WeaponImageIsTinted(weaponID int64) bool` l'accompagne. Mesure : les icones extraites
      sont a **100 % blanches** (masque pur), les deux PNG dessines a la main sont en couleur
      (31,6 % et 55,6 % de pixels colores). Rendre un masque tel quel le rend INVISIBLE en
      theme clair ; teindre une image finie l'aplatit en silhouette. Le front ne peut pas
      deviner lequel il tient — seul l'adapter du titre le sait.
- [x] `halo_infinite/adapter_asset_urls.go` : lookup par tag `weap`, table GENEREE
      (`weapon_icons_gen.go`, 124 tags) ; `name_en` a disparu avec ses 39 alias
- [x] `halo_5/adapter_asset_urls.go` et `synthetic_title_b/adapter.go` : signature suivie.
      H5 s'indexe desormais par `weapon_id` et non par `name_en` (meme raison), via
      `canonical.AssetMeta.WeaponID()` — ParseUint puis conversion, car un weapon_id Infinite
      DEPASSE int64 en decimal (ParseInt echouait sur les deux tiers du referentiel)
- [x] `match_view_converters.go` : **amende** — le plan disait « passer `weaponResolved.weaponKey` ».
      Refute par l'amendement de cette meme etape : la cle est le TAG, pas le `weapon_key`.
      C'est `w.WeaponID` qui est passe, et il etait deja sur la ligne
- [x] `asset_service.go` : idem, via `AssetMeta.WeaponID()`
- [x] Les 3 SENTINELLES — **STATUEES** : table explicite keyee par ID, jamais par nom.
      `0` grenade et `1` melee gardent leur PNG dessine a la main ; `2` vehicule n'a PAS
      d'icone (aucun dessin ne represente « un vehicule » en general, et il n'en avait pas
      non plus avant). Les 3 grenades REELLES du referentiel (Frag/Plasma/Dynamo) rejoignent
      la meme table, keyees par TAG : ce ne sont pas des `weap` (elles vivent en `eqip`+`proj`),
      l'atlas ne les porte donc pas — mesure, pas supposition
- [x] Test : `TestAssetURLAdapter_WeaponImageURL_ResoutParTagWeap` couvre le Sidekick malgre
      la divergence Mk51/Mk50, MA5K Avenger et Mutilator sans `weapon_key`, la variante
      Diminisher of Hope, et Cindershot ≠ Heatwave (defaut n°2)
- **Gate** : `go test ./internal/games/... ./internal/service/...` **vert**, `go vet` **vert**.
  `[!]` **« aucune URL d'image changee »** : IMPOSSIBLE ET CONTRADICTOIRE avec l'ordre recu,
  qui fusionne les etapes 1 et 2 en un seul lot — les URL changent par construction (c'est
  l'objet de l'etape 2). Remplace par la comparaison avant/apres portee par l'artefact de
  revue, arme par arme.

### Etape 2 — les nouvelles images remplacent les anciennes

- [x] Les PNG de `jeu/` deviennent la source servie ; **26 des 28** anciens PNG sont supprimes
      avec la map `name_en`. `Grenade.png` et `Melee.png` SURVIVENT, et c'est une decision,
      pas un oubli : l'atlas extrait ne porte NI grenade NI melee generique (elles ne sont pas
      des `weap`). Les supprimer aurait fait perdre son icone a 5 etiquettes sur 42 — une
      regression visible, la ou la regle « 0 code mort » vise le code inutilise, pas un asset
      encore servi. Elles ne sont plus keyees par nom mais par ID/tag : le defaut n°1 tombe
      quand meme
- [x] `Needler-2.png` orphelin : supprime, et `TestAucunPNGOrphelin` ferme le trou par lequel
      il etait passe (un PNG non reference par une table fait echouer la suite)
- [x] Table GENEREE depuis `index.json` — **amende** : `tag weap -> fichier`, pas
      `weapon_key -> index`, et par une commande DEDIEE `cmd/weapon-icons-table`. Motif :
      `weapon-icons-build` exige les archives du jeu installe ; verifier la fraicheur de la
      table serait alors impossible en CI, et elle divergerait en silence. La commande dediee
      ne lit qu'`index.json`, versionne — `TestTableGenereeEstAJour` la rejoue et compare
      octet a octet
- [x] **Garde-rail neuf** : `TestChaqueArmeDuRegistreAUneIcone` (a — toute famille du registre
      est servie, atlas ou concept declare) et `TestWeaponIconFilesExistentSurDisque` (b —
      chaque PNG reference existe). Defaut n°3 ferme
- [x] Les 3 armes sans image — **STATUEES** : `Sandwich` et `Mythic Sandwich` se comblent
      (tags `880fe0bc` / `b7262ca1` -> index 35), `Vehicle` reste sans icone (cf. etape 1)
- **Gate** : garde-rails **verts**, `make check-types` **vert**, **revue visuelle utilisateur
  PASSEE le 2026-08-11** (« tout est parfait ») sur l'artefact arme par arme, thème clair et
  sombre. GATE 1 du plan d'origine : `[x]`.

### Etape 3 — l'atlas kill feed est expose

`[x]` **ROUVERTE ET FERMEE LE 2026-08-11** (lot « arme du kill »). La refutation ci-dessous
reste JUSTE sur les deux liens qu'elle examinait — mais elle en avait manque un TROISIEME,
deja versionne dans ce meme dossier et jamais consulte : la passe humaine
`NOMMAGE_GATE_2026-08-09.tsv`, qui donne `index kill feed -> weapon_key` pour 26 vignettes et
un libelle humain pour les autres. Elle CORRIGE precisement le piege des noms internes
(l'index 22, nomme `heatwave` par le jeu, porte `hinf_cindershot`), et
`config/titles/halo_infinite/mappings/weapon_names.toml` la corrobore independamment
(Cremateur = Cindershot, Calcineur = Heatwave).

Le pont vit dans `internal/games/halo_infinite/film/killicon` : 36 regles versionnees,
resolution indexee par TAG a l'initialisation, garde-rail de corroboration par le registre
d'armes. Detail : entree du 2026-08-11 au thought log.

Ce qui suit est conserve tel quel — c'est la trace de ce qui avait ete mesure et pourquoi
l'etape avait ete fermee une premiere fois :

Elle supposait un lien `index kill feed -> arme`. Il n'existe pas :

1. `index.json` porte bien `tags_weap` sur les entrees `killfeed`, mais ce champ est indexe
   par `sprite index`, qui n'a de sens que pour les DEUX atlas d'armes. L'atlas du kill feed
   a sa propre numerotation : son index 0 est le Battle Rifle la ou le contour 0 est le fusil
   d'assaut. Le champ est un artefact de generation (`weapon-icons-build/main.go` l'ecrit sur
   les trois styles alors qu'il ne le calcule que pour les armes) — le lire donnerait une
   icone fausse pour CHAQUE arme.
2. Le seul autre lien serait le nom interne, et l'etat de l'art le refute explicitement :
   `heatwave` = Cremateur, `plasma_blaster` = Fusil traqueur, `shade_turret` = un bidon.

Rouvrir cette etape demande d'abord de MESURER un lien arme -> index kill feed. En attendant,
la surface produit utilise l'atlas `contour`, dont le lien est lu dans le jeu. Consigne au
`REGISTRE_REPORTS.md`.

### Etape 4 — la teinte d'equipe au kill feed

- [x] Composant d'icone teintee (masque CSS + token) : `components/ui/WeaponIcon.tsx`, avec
      son test. Le MODE de rendu vient du back (`image_tinted`), jamais du titre ni de la
      forme de l'URL — meme discipline que `MedalIcon`. Branche sur `PlayerDetailPanel`
      (armes du scoreboard) et `AssetCard` (tiroir d'assets)
- [x] Zero hex, zero classe Tailwind couleur — verifie par `grep` ET par le test du composant
- [~] Strings FR **et** EN : aucun libelle neuf n'apparait. Le `aria-label` reprend le libelle
      d'arme deja localise par le back
- `[x]` **APPLICATION AU KILL FEED — FAITE LE 2026-08-11** (lot « arme du kill »). Le
      prealable pose ici (« un chemin mesure `jpt! -> weap` ») etait un DETOUR : le tag
      `weap` n'est pas necessaire pour designer une vignette, le NOM suffit, et
      `jpt! -> nom` est deja resolu a 97,6 % par `damagetag`. Exiger le `weap` revenait a
      se donner une contrainte que la mesure ne demandait pas (11 lignes sur 114 le
      portent, d'ou le blocage apparent).
      Livre : `MatchHighlightEvent` porte `weapon_key` / `weapon_label` /
      `weapon_image_url` / `weapon_image_tinted` + `actor_team_id` ; le feed passe en DOM
      (`features/match-view/MatchKillFeed.tsx`) et teinte l'icone a la couleur d'identite
      de l'equipe du tueur — meme cascade que l'en-tete du scoreboard.
- **Gate** : `make check-types` **vert**, `make test-web` **vert**, grep couleurs **vide**,
  revue visuelle **PASSEE le 2026-08-11** (cf. etape 2).

---

## 5. CE QUI N'EST PAS DANS CE PLAN

- **Le rejeu 2D** : il represente l'arme par une FORME et refuse deliberement les tokens
  d'equipe (`ReplayTeams.tsx:102-110` — « ce rejeu se regarde de l'exterieur, pas depuis un
  joueur »). Y poser des icones et des couleurs d'equipe contredirait une decision ecrite : ce
  serait un autre sujet, a rouvrir explicitement.
- **Halo 5** : ses icones sont des URL CDN en base, peuplees par l'API officielle. Rien a
  remplacer. Le plan garde la signature commune, pas le comportement.
- **La question de licence** : redistribuer des assets extraits du jeu dans un produit deploye
  est une decision de l'utilisateur, deja notee comme prealable dans
  `.ai/PLAN_RECHERCHE_ASSETS_ICONES.md:88`. Elle n'est pas tranchee ici.

---

## 5 bis. RENVOYE AU LOT DE FIN DE v7.5 (decisions utilisateur du 2026-08-11)

Trois points, a traiter avec les cartes v2 dans le dernier lot avant le tag. Detail et
condition de reprise : `.ai/V7.5/REGISTRE_REPORTS.md`.

1. **Armes qui partagent une icone** — 5 index servent 13 etiquettes, et c'est LE JEU qui ne
   les distingue pas (meme `sprite index`, voire un seul tag `weap` pour trois etiquettes).
   Chercher une image specifique AILLEURS que dans l'atlas d'armes ; a defaut, ne presenter
   que la variante de BASE.
2. **Grenades** — verifier qu'il n'existe vraiment aucune image, en cherchant la ou l'objet
   est declare (`eqip`/`proj`/`gggl`) et non dans le `weap`. Dernier trou consequent.
3. **La teinte alliee/ennemie ne concerne QUE les icones du kill feed**, pas les grandes
   images. L'etat livre y est conforme : aucune couleur d'equipe n'est appliquee.

---

## 6. DECOUVERTES (a ne pas traiter dans ce chantier)

- `weapon_labels.name_en` diverge de `weapon_names.toml` sur le Sidekick (Mk51 vs Mk50). L'etape
  1 rend l'image insensible a cette divergence, mais **la divergence elle-meme reste** et merite
  son propre correctif.
- **`index.json` publie `tags_weap` sur les entrees `killfeed` alors que ce champ ne vaut que
  pour les atlas d'armes** (`weapon-icons-build/main.go:220` l'ecrit pour les trois styles ;
  `weaptags.go` ne le calcule que depuis le `sprite index`). Le consommateur s'en protege en
  filtrant sur le style `contour`, et `TestSeulLAtlasContourEstLu` le verrouille — mais
  l'artefact versionne porte toujours un champ faux, piege pour le prochain lecteur. Non
  corrige ici : le corriger demande de REGENERER `index.json`, donc une machine avec le jeu
  installe. Consigne au registre des reports.
- Les anciennes icones pesaient jusqu'a 453 Ko ; les nouvelles font ~8 Ko. Le poids servi par
  le tiroir d'assets chute d'environ deux ordres de grandeur — effet de bord favorable, non
  mesure finement.
- Les anciens PNG pesent jusqu'a 453 Ko et ne sont pas optimises.
- `abilities-assets` et `grenades-assets` ont un `index.json` que personne ne consomme.
