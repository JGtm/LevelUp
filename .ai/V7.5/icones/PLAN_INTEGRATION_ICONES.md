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

**Retenu : `index.json` genere + une table Go derivee, keyee par `weapon_key`.**

Pourquoi pas la base :

- Le lien index -> arme est une propriete de l'ENSEMBLE D'ASSETS, versionne avec les PNG. Il ne
  change que quand on re-extrait. Le mettre en base imposerait une migration + un seed pour une
  donnee deja statique et deja versionnee, et creerait une **seconde source de verite** face a
  `index.json` — que rien ne comparerait (le defaut exact des tables recopiees de la page de
  nommage, corrige au commit `5543dc170`).
- Le depot a deja le precedent : `abilities-assets/index.json` et `grenades-assets/index.json`.
- La base garde ce qui lui revient et qu'elle porte deja tres bien : le registre des armes, les
  libelles FR/EN, les identifiants filmshell. **Le plan n'y touche pas.**

Ce qui va en base : **rien**. Ce qui va en code : une table `weapon_key -> index d'atlas`,
GENEREE depuis `index.json` par la commande d'extraction, avec un garde-rail qui echoue si un
`weapon_key` du registre n'a pas d'icone ou si un PNG reference manque sur disque.

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

### Etape 1 — la cle devient `weapon_key` (aucun changement visuel)

- [ ] `games/adapter.go` : `WeaponImageURL(weaponKey, nameEN string) string` (2 params, sous le
      seuil de 5 ; `nameEN` reste pour Halo 5 qui resout par nom depuis la base)
- [ ] `halo_infinite/adapter_asset_urls.go` : lookup par `weapon_key`, `name_en` ignore
- [ ] `halo_5/adapter_asset_urls.go` et `synthetic_title_b/adapter.go` : signature suivie,
      comportement inchange
- [ ] `match_view_converters.go:224` : passer `weaponResolved.weaponKey` (il existe deja,
      `weapon_resolver.go:41`, il n'etait juste pas transmis)
- [ ] `asset_service.go:64-81` : idem
- [ ] Test : une arme dont `name_en` diverge entre `weapon_labels` et `weapon_names.toml`
      (Sidekick) rend la BONNE image
- **Gate** : `go test ./internal/games/... ./internal/service/...` vert, `go vet` vert, aucune
  URL d'image changee (comparaison avant/apres sur les 29 armes)

### Etape 2 — les nouvelles images remplacent les anciennes

- [ ] Les PNG de `jeu/` deviennent la source servie ; les 28 anciens PNG sont SUPPRIMES avec
      leurs entrees de map (regle « 0 code mort » : rien n'est garde « au cas ou », git a
      l'historique)
- [ ] `Needler-2.png` orphelin : supprime
- [ ] Table `weapon_key -> index` GENEREE depuis `index.json` par `cmd/weapon-icons-build`
- [ ] **Garde-rail neuf** : un test qui echoue si (a) un `weapon_key` du registre n'a pas
      d'icone, (b) un PNG reference manque sur disque. C'est le defaut n°3 du §0
- [ ] Les 3 armes sans image (`Vehicle` sentinelle, `Sandwich`, `Mythic Sandwich`) : statuer —
      l'atlas du jeu porte `sandwich` (index 35), donc 2 des 3 se comblent
- **Gate** : garde-rail vert, `make check-types`, revue visuelle utilisateur sur >= 10 armes
  (GATE 1 du plan d'origine)

### Etape 3 — l'atlas kill feed est expose

- [ ] Servir les icones kill feed par `weapon_key` (surface distincte de l'atlas d'armes)
- [ ] Les entrees NON-armes (vehicules, grenades, pictogrammes) : exposees par leur nom interne,
      pas par un `weapon_key` — elles n'en ont pas
- [ ] Les index encore non identifies (25, 63, 75, 77 et les tonneaux non confirmes) ne sont PAS
      exposes : afficher une mauvaise icone est pire qu'aucune
- **Gate** : aucun index sans etiquette servi ; test de non-regression sur la liste exposee

### Etape 4 — la teinte d'equipe au kill feed

Precede d'une decision utilisateur (§3 : chart ou liste).

- [ ] Composant d'icone teintee (masque CSS + `tokenCssVar`), dans `components/`
- [ ] Zero hex, zero classe Tailwind couleur — verifie par `grep`
- [ ] Strings FR **et** EN si un libelle apparait
- **Gate** : `make check-types`, `make test-web`, grep couleurs vide, revue visuelle

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

## 6. DECOUVERTES (a ne pas traiter dans ce chantier)

- `weapon_labels.name_en` diverge de `weapon_names.toml` sur le Sidekick (Mk51 vs Mk50). L'etape
  1 rend l'image insensible a cette divergence, mais **la divergence elle-meme reste** et merite
  son propre correctif.
- Les anciens PNG pesent jusqu'a 453 Ko et ne sont pas optimises.
- `abilities-assets` et `grenades-assets` ont un `index.json` que personne ne consomme.
