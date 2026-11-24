# Mesure de la teinte du fond Relief (item 0.3, D-3)

Rédigé le 2026-11-18 sur `release/v8.3.0`. C'est une mesure : aucun code de l'application n'est touché. Les outils de calcul sont des scripts jetables hors dépôt (dossier `scratchpad\teinte` de la session) ; `deltaE`, `simulateCvd`, `contrastRatio` et les cinq palettes sont IMPORTÉS de `apps/web/src/lib/accessibility/` (`colorDistance.ts`, `wcagContrast.ts`, `palettes/`), aucune matrice recopiée. Le relief des quatre cartes témoins vient de `relief.webp` et `relief.json` de la variante (b) de `conception.md` §4.2, décodés selon le §1.

Page de choix pour le user : `.ai/V8.3/annexes_fonds_relief/teinte_choix.html` (images réduites : `.ai/V8.3/annexes_fonds_relief/teinte_images/`).

## 1. Verdict

**Aucune candidate ne tient le seuil de D-3 partout** : 0 combinaison sur 64 (8 candidates × 4 palettes × 2 thèmes) n'a zéro échec sur les encres de marqueur et d'objectif. Même la meilleure de chaque palette laisse 28, 47, 50, 55 cellules sous le seuil (palettes dans l'ordre défaut, Okabe-Ito, Cividis, Tol Bright) sur les 160 du rendu par défaut (cellule = une encre, un palier de sol, une vision ; 8 encres × 5 paliers × 4 visions). Le seuil, tel qu'écrit et appliqué palier par palier, n'est pas atteignable par le choix de la teinte seule avec ces sept préréglages et le gris du thème (§6 : un balayage de 63 teintes libres n'y arrive pas non plus).

Pourquoi, en deux faits mesurés :

1. **Le liseré ne rattrape presque jamais le noyau.** L'encre du liseré (`style.ink`) n'est pas sombre : c'est `floor.edge` = `--muted-foreground` (`#a1a1a1` en sombre, `#636a72` en clair), posée à l'alpha 0,62 (`OUTLINE_ALPHA`). Mélangée à 38 % du sol, elle donne un contraste de 1,00 à 2,36 contre le sol ; sur 1 280 combinaisons (palette, thème, candidate, vision, palier) aucune n'atteint 3:1 : la branche « contour ≥ 3:1 » du seuil ne se déclenche JAMAIS, et le seuil se réduit à « ΔE noyau/sol ≥ 15 ».
2. **Le sol parcourt toute la clarté, le noyau est fixe.** Les cinq paliers (gris 0,36 à 1,0) sont espacés de ΔE 6,5 à 7,8 ; de bout en bout, la rampe couvre 28 ΔE. Un noyau n'est à 15 de chacun des cinq que s'il est loin de toute la rampe par sa teinte ou sa chroma : un rouge saturé sur un sol bleu y arrive en vision normale, un bleu sur un sol bleu non, et la vision simulée défait le reste (rouge sur Vert en protanopie : ΔE 1,7).

Les écarts entre les premières candidates d'une même palette sont petits (1 à 3 points de taux d'échec pondéré) : la page de choix sert à trancher sur pièces, pas ce tableau seul.

### Recommandation par palette (provisoire : aucune ne tient le seuil)

Critère de classement : **taux d'échec pondéré** sur les 20 encres (8 du rendu par défaut : équipes, sans-camp, glyphes, zones, socle ; 12 des modes optionnels : couleurs distinctes par joueur et couleurs d'escouade), chaque cellule pesant la part du sol des 4 cartes qui se trouve à ce palier (0,36 : 36 % ; 0,52 : 36 % ; 0,68 : 19 % ; 0,84 : 7 % ; 1,0 : 2 %), pire des deux thèmes. Une teinte unique par palette (D-3) impose de juger sur les deux thèmes à la fois.

| Palette | Teinte recommandée | Base | Taux d'échec pondéré (défaut / options / total) | Seconde et troisième | Raison |
|---|---|---|---|---|---|
| Par défaut (`default`) | **Graphite** | `#6b7078` | 9,3 % / 16,9 % / 13,9 % | Ardoise (15,1 %), Vert (15,9 %) | meilleur total (13,9 %) ; suivent Ardoise (15,1 %) et Vert (15,9 %) ; meilleure sur le seul rendu par défaut : Violet (7,3 %, mais 32,4 % sur les options) ; signes (anneaux, croix de mort) : 13 cellules en échec sur 40, Ardoise 12 ; même valeur dans les deux thèmes, ce que le gris du thème ne permet pas |
| Okabe-Ito (`okabe-ito`) | **Violet** | `#7559b5` | 21,8 % / 21,2 % / 21,4 % | Ardoise (22,3 %), Vert (23,4 %) | meilleur total (21,4 %) ; suivent Ardoise (22,3 %) et Vert (23,4 %) ; meilleure sur le seul rendu par défaut : Vert (20,6 %, mais 25,3 % sur les options) ; signes (anneaux, croix de mort) : 5 cellules en échec sur 40, Ardoise 12 |
| Cividis (`cividis`) | **Vert** | `#3f8c4f` | 22,0 % / 31,1 % / 27,4 % | Bleu (29,5 %), Violet (29,9 %) | meilleur total (27,4 %) ; suivent Bleu (29,5 %) et Violet (29,9 %) ; meilleure sur le seul rendu par défaut : Ardoise (21,4 %, mais 39,1 % sur les options) ; meilleure sur les seules options : Bleu (29,3 %) ; signes (anneaux, croix de mort) : 12 cellules en échec sur 40, Bleu 17 |
| Tol Bright (`tol-bright`) | **Violet** | `#7559b5` | 21,8 % / 24,2 % / 23,3 % | Ardoise (27,2 %), Vert (27,8 %) | meilleur total (23,3 %) ; suivent Ardoise (27,2 %) et Vert (27,8 %) ; signes (anneaux, croix de mort) : 7 cellules en échec sur 40, Ardoise 14 |

Lecture du taux : part des couples (encre, vision) sous le seuil, pondérée par la surface de chaque palier ; ce n'est pas une part du sol. Les cellules qui échouent, encre par encre, sont au §5 ; l'amendement possible de D-3 au §6.

## 2. Méthode

- **Rampe** (Den, `films-ui` `terrainStyle`) : depuis la base B, `low = mix(noir, B, 0,3)`, `mid = mix(noir, B, 0,78)`, `high = mix(B, blanc, 0,3)` ; gris 0 → low, 0,5 → mid, 1 → high, interpolation linéaire en sRGB. Cinq paliers du sol : gris 0,36, 0,52, 0,68, 0,84, 1,0.
- **Candidates** : les sept préréglages de Den (Bleu `#2f7fc1`, Ardoise `#5a6f8c`, Sarcelle `#1f8a8c`, Vert `#3f8c4f`, Sable `#a58a5c`, Violet `#7559b5`, Graphite `#6b7078`) et le **gris du thème**. Le gris du thème est `--muted-foreground` : c'est le gris neutre que le rejeu emploie déjà (`floor.edge`, `readInk('--muted-foreground')`), et les jetons de surface (`--background`, `--card`, `--muted`) sont trop extrêmes pour servir de base à une rampe qui part de 30 % de noir et monte de 30 % vers le blanc. Valeur sombre `#a1a1a1`, claire `#636a72` (`oklch(0.708 0 0)` et `oklch(0.52 0.015 255)` de `globals.css`, convertis en hex ; un aller-retour par `oklab()` du dépôt contrôle la conversion à moins de 0,8 ΔE).
- **Quatre visions** : normale, protanopie, deutéranopie, tritanopie (`simulateCvd`, Machado 2009, sévérité 1,0). Le sol, le noyau et le liseré composé sont simulés AVANT la mesure.
- **Noyau/sol** : `deltaE(noyau, palier)` (OKLab × 100). **Contour/sol** : contraste WCAG entre le liseré réel (`style.ink` à l'alpha 0,62 composé sur le palier, soit 62 % encre + 38 % palier) et le palier.
- **Seuil de D-3 appliqué à chaque cellule** (une encre, un palier, une vision) : ΔE ≥ 15 OU contraste ≥ 3:1. Une candidate tient si aucune cellule des groupes « marqueur » et « objectif » n'échoue. La rampe elle-même (paliers voisins) est mesurée à part.
- **Encres mesurées** (relevées dans le code, §3) : marqueurs de joueur (équipe allié et adverse, sans-camp, 8 couleurs distinctes, 4 couleurs d'escouade ; liseré `--muted-foreground` à 0,62), objectifs (traits de zone de camp sans liseré ; zone sans camp et glyphes neutres avec liseré `--background` plein ; socle `warning` sans liseré), signes posés sur le marqueur (anneaux `success`, croix de mort `destructive` à 0,9, sans liseré propre : seul le ΔE joue ; groupe rapporté à part, hors verdict).

## 3. Encres réelles relevées

Les jetons de ces rôles ont la même valeur dans les deux thèmes (`paletteForTheme` ne change que les rampes d'impact). Seules les encres de mise en page changent. Les couleurs d'équipe `team-ally` / `team-enemy` peuvent être remplacées par l'utilisateur dans les réglages d'accessibilité (`theme-provider.tsx`) : la garde de D.5 ne verra que les valeurs de palette.

Encres de mise en page (`globals.css`, `canvasInk.ts`) :

| Variable | Sombre | Clair | Rôle dans le rejeu |
|---|---|---|---|
| `--muted-foreground` | `#a1a1a1` | `#636a72` | `style.ink`, liseré du marqueur (alpha 0,62 `OUTLINE_ALPHA`, rayon + 1 px `OUTLINE_PAD`) |
| `--background` | `#0a0a0a` | `#f8fafd` | liseré des glyphes (bombe, drapeau, crâne) et des zones sans camp (`markInk.outline`, `zone.outline`) |
| `--foreground` | `#fafafa` | `#1c232c` | remplissage des marques (`markInk.fill`) |
| `--card` | `#171717` | `#fcfdff` | fond du conteneur du canevas (`bg-card`) |

| Jeton | Par défaut | Okabe-Ito | Cividis | Tol Bright | Usage |
|---|---|---|---|---|---|
| `team-ally` | `#3B82F6` | `#0072B2` | `#0072B2` | `#4477AA` | marqueur et zone : équipe alliée (noyau, `teamColorOf`) |
| `team-enemy` | `#EF4444` | `#D55E00` | `#D55E00` | `#EE6677` | marqueur et zone : équipe adverse |
| `divergent-neutral` | `#60A5FA` | `#888888` | `#888888` | `#888888` | marqueur sans camp ; glyphes neutres (`neutralInk`) |
| `zone-neutral` | `#9CA3AF` | `#888888` | `#888888` | `#888888` | zone, base, socle que personne ne tient |
| `warning` | `#F59E0B` | `#E69F00` | `#B6A855` | `#CCBB44` | socle à fort enjeu, mur de protection |
| `success` | `#10B981` | `#009E73` | `#0072B2` | `#228833` | anneaux du joueur de la page (`selfInk`) |
| `destructive` | `#EF4444` | `#D55E00` | `#D55E00` | `#EE6677` | croix de mort (`deathInk`) |
| `chart-series-1` | `#93C5FD` | `#0072B2` | `#00224E` | `#4477AA` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 1) |
| `chart-series-2` | `#6366F1` | `#E69F00` | `#FEE838` | `#EE6677` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 2) |
| `chart-series-3` | `#4F46E5` | `#56B4E9` | `#3F4A6B` | `#228833` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 3) |
| `chart-series-4` | `#4338CA` | `#D55E00` | `#B6A855` | `#CCBB44` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 4) |
| `chart-series-5` | `#3730A3` | `#009E73` | `#575C6D` | `#66CCEE` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 5) |
| `chart-series-6` | `#10B981` | `#F0E442` | `#928D6B` | `#AA3377` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 6) |
| `chart-series-7` | `#F59E0B` | `#CC79A7` | `#1B3158` | `#BBBBBB` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 7) |
| `chart-series-8` | `#EC4899` | `#BBBBBB` | `#E1C752` | `#000000` | couleurs distinctes par joueur (`SERIES_TOKENS`, rang 8) |
| `squad-player-1` | `#2563EB` | `#0072B2` | `#3F4A6B` | `#0077BB` | couleurs d'escouade (Tactique) |
| `squad-player-2` | `#D97706` | `#B77E00` | `#7C7B78` | `#CC3311` | couleurs d'escouade (Tactique) |
| `squad-player-3` | `#DB2777` | `#8E5374` | `#B6A855` | `#AA4499` | couleurs d'escouade (Tactique) |
| `squad-player-4` | `#0891B2` | `#009E73` | `#FEE838` | `#117733` | couleurs d'escouade (Tactique) |

## 4. Table complète

Lecture : « ΔE noyau/sol min » et « contraste contour/sol min » sont les pires valeurs sur toutes les encres de marqueur et d'objectif et les cinq paliers ; N / P / D / T = vision normale, protanopie, deutéranopie, tritanopie. « ΔE voisins min » : plus petit écart entre deux paliers voisins de la rampe. « Échecs » : cellules sous le seuil de D-3 sur 400 (encres de marqueur et d'objectif) ; « signes » : sur 40. « Taux pondéré » : critère du §1 (20 encres, pondéré par la surface des paliers). Contraste : `—` n'existe pas (toutes les lignes ont au moins un liseré).

### Par défaut, thème sombre

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 5,1 / 2,1 / 2,5 / 0,6 | 1,0 / 1,0 / 1,0 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 127 | 9 | 28,8 % | non |
| Ardoise `#5a6f8c` | 3,5 / 1,9 / 3,7 / 3,6 | 1,1 / 1,0 / 1,1 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 83 | 12 | 14,9 % | non |
| Sarcelle `#1f8a8c` | 5,7 / 1,4 / 1,9 / 0,8 | 1,0 / 1,1 / 1,0 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 92 | 16 | 18,3 % | non |
| Vert `#3f8c4f` | 6,9 / 1,6 / 1,0 / 4,5 | 1,0 / 1,1 / 1,0 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 86 | 18 | 15,8 % | non |
| Sable `#a58a5c` | 7,5 / 0,9 / 1,8 / 6,1 | 1,1 / 1,1 / 1,0 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 77 | 15 | 15,7 % | non |
| Violet `#7559b5` | 8,8 / 3,9 / 0,5 / 4,4 | 1,1 / 1,1 / 1,1 / 1,1 | 6,6 / 6,6 / 6,4 / 6,3 | 111 | 3 | 22,1 % | non |
| Graphite `#6b7078` | 2,7 / 2,9 / 2,4 / 2,6 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 77 | 13 | 13,6 % | non |
| Gris du thème `#a1a1a1` | 2,9 / 2,7 / 0,4 / 2,7 | 1,1 / 1,1 / 1,1 / 1,1 | 6,4 / 6,4 / 6,4 / 6,4 | 86 | 16 | 22,1 % | non |

### Par défaut, thème clair

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 5,1 / 2,1 / 2,5 / 0,6 | 1,1 / 1,0 / 1,1 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 135 | 9 | 29,1 % | non |
| Ardoise `#5a6f8c` | 3,5 / 1,9 / 3,7 / 3,6 | 1,0 / 1,1 / 1,0 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 91 | 12 | 15,1 % | non |
| Sarcelle `#1f8a8c` | 5,7 / 1,4 / 1,9 / 0,8 | 1,0 / 1,0 / 1,1 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 102 | 16 | 18,7 % | non |
| Vert `#3f8c4f` | 6,9 / 1,6 / 1,0 / 4,5 | 1,0 / 1,0 / 1,1 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 91 | 18 | 15,9 % | non |
| Sable `#a58a5c` | 7,5 / 0,9 / 1,8 / 6,1 | 1,1 / 1,0 / 1,1 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 85 | 15 | 16,2 % | non |
| Violet `#7559b5` | 8,8 / 3,9 / 0,5 / 4,4 | 1,0 / 1,0 / 1,0 / 1,0 | 6,6 / 6,6 / 6,4 / 6,3 | 119 | 3 | 22,3 % | non |
| Graphite `#6b7078` | 2,7 / 2,9 / 2,4 / 2,6 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 85 | 13 | 13,9 % | non |
| Gris du thème `#636a72` | 4,1 / 4,2 / 1,1 / 3,9 | 1,0 / 1,0 / 1,0 / 1,0 | 6,2 / 6,3 / 6,3 / 6,1 | 73 | 10 | 10,5 % | non |

### Okabe-Ito, thème sombre

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 4,5 / 3,4 / 3,5 / 1,5 | 1,0 / 1,0 / 1,0 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 130 | 11 | 29,1 % | non |
| Ardoise `#5a6f8c` | 4,0 / 2,2 / 1,7 / 3,9 | 1,1 / 1,0 / 1,1 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 125 | 12 | 22,0 % | non |
| Sarcelle `#1f8a8c` | 7,6 / 3,5 / 1,9 / 1,3 | 1,0 / 1,1 / 1,0 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 145 | 19 | 33,0 % | non |
| Vert `#3f8c4f` | 5,2 / 0,9 / 2,3 / 3,8 | 1,0 / 1,1 / 1,0 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 120 | 24 | 23,2 % | non |
| Sable `#a58a5c` | 5,4 / 2,0 / 3,1 / 4,4 | 1,1 / 1,1 / 1,0 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 142 | 22 | 30,2 % | non |
| Violet `#7559b5` | 8,6 / 3,8 / 3,2 / 3,0 | 1,1 / 1,1 / 1,1 / 1,1 | 6,6 / 6,6 / 6,4 / 6,3 | 112 | 5 | 21,2 % | non |
| Graphite `#6b7078` | 1,4 / 1,4 / 1,1 / 1,4 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 135 | 16 | 24,8 % | non |
| Gris du thème `#a1a1a1` | 0,6 / 0,6 / 0,6 / 0,6 | 1,1 / 1,1 / 1,1 / 1,1 | 6,4 / 6,4 / 6,4 / 6,4 | 153 | 17 | 45,2 % | non |

### Okabe-Ito, thème clair

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 4,5 / 3,4 / 3,5 / 1,5 | 1,1 / 1,0 / 1,1 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 138 | 11 | 29,4 % | non |
| Ardoise `#5a6f8c` | 4,0 / 2,2 / 1,7 / 3,9 | 1,0 / 1,1 / 1,0 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 133 | 12 | 22,3 % | non |
| Sarcelle `#1f8a8c` | 7,6 / 3,5 / 1,9 / 1,3 | 1,0 / 1,0 / 1,1 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 155 | 19 | 33,4 % | non |
| Vert `#3f8c4f` | 5,2 / 0,9 / 2,3 / 3,8 | 1,0 / 1,0 / 1,1 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 128 | 24 | 23,4 % | non |
| Sable `#a58a5c` | 5,4 / 2,0 / 3,1 / 4,4 | 1,1 / 1,0 / 1,1 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 158 | 22 | 31,1 % | non |
| Violet `#7559b5` | 8,6 / 3,8 / 3,2 / 3,0 | 1,0 / 1,0 / 1,0 / 1,0 | 6,6 / 6,6 / 6,4 / 6,3 | 120 | 5 | 21,4 % | non |
| Graphite `#6b7078` | 1,4 / 1,4 / 1,1 / 1,4 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 143 | 16 | 25,0 % | non |
| Gris du thème `#636a72` | 2,8 / 2,5 / 0,5 / 2,9 | 1,0 / 1,0 / 1,0 / 1,0 | 6,2 / 6,3 / 6,3 / 6,1 | 147 | 16 | 23,0 % | non |

### Cividis, thème sombre

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 4,5 / 3,4 / 3,1 / 1,5 | 1,0 / 1,0 / 1,0 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 104 | 17 | 29,3 % | non |
| Ardoise `#5a6f8c` | 2,4 / 1,5 / 2,5 / 1,8 | 1,1 / 1,0 / 1,1 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 135 | 15 | 31,8 % | non |
| Sarcelle `#1f8a8c` | 7,4 / 1,9 / 1,2 / 2,1 | 1,0 / 1,1 / 1,0 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 126 | 19 | 32,2 % | non |
| Vert `#3f8c4f` | 6,8 / 2,1 / 1,0 / 3,2 | 1,0 / 1,1 / 1,0 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 122 | 12 | 27,2 % | non |
| Sable `#a58a5c` | 2,7 / 2,7 / 1,8 / 3,0 | 1,1 / 1,1 / 1,0 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 129 | 13 | 29,3 % | non |
| Violet `#7559b5` | 5,8 / 3,8 / 3,2 / 3,0 | 1,1 / 1,1 / 1,1 / 1,1 | 6,6 / 6,6 / 6,4 / 6,3 | 116 | 14 | 29,7 % | non |
| Graphite `#6b7078` | 1,4 / 1,4 / 1,7 / 1,3 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 133 | 17 | 31,1 % | non |
| Gris du thème `#a1a1a1` | 1,8 / 1,8 / 1,8 / 1,8 | 1,1 / 1,1 / 1,1 / 1,1 | 6,4 / 6,4 / 6,4 / 6,4 | 132 | 13 | 37,7 % | non |

### Cividis, thème clair

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 4,5 / 3,4 / 3,1 / 1,5 | 1,1 / 1,0 / 1,1 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 112 | 17 | 29,5 % | non |
| Ardoise `#5a6f8c` | 2,4 / 1,5 / 2,5 / 1,8 | 1,0 / 1,1 / 1,0 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 143 | 15 | 32,0 % | non |
| Sarcelle `#1f8a8c` | 7,4 / 1,9 / 1,2 / 2,1 | 1,0 / 1,0 / 1,1 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 136 | 19 | 32,6 % | non |
| Vert `#3f8c4f` | 6,8 / 2,1 / 1,0 / 3,2 | 1,0 / 1,0 / 1,1 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 130 | 12 | 27,4 % | non |
| Sable `#a58a5c` | 2,7 / 2,7 / 1,8 / 3,0 | 1,1 / 1,0 / 1,1 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 145 | 13 | 30,2 % | non |
| Violet `#7559b5` | 5,8 / 3,8 / 3,2 / 3,0 | 1,0 / 1,0 / 1,0 / 1,0 | 6,6 / 6,6 / 6,4 / 6,3 | 124 | 14 | 29,9 % | non |
| Graphite `#6b7078` | 1,4 / 1,4 / 1,7 / 1,3 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 141 | 17 | 31,3 % | non |
| Gris du thème `#636a72` | 2,4 / 2,5 / 2,3 / 2,3 | 1,0 / 1,0 / 1,0 / 1,0 | 6,2 / 6,3 / 6,3 / 6,1 | 149 | 19 | 32,4 % | non |

### Tol Bright, thème sombre

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 1,1 / 1,1 / 1,0 / 0,6 | 1,0 / 1,0 / 1,0 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 135 | 11 | 31,6 % | non |
| Ardoise `#5a6f8c` | 4,0 / 3,0 / 2,6 / 3,9 | 1,1 / 1,0 / 1,1 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 137 | 14 | 27,0 % | non |
| Sarcelle `#1f8a8c` | 7,6 / 3,1 / 2,9 / 1,3 | 1,0 / 1,1 / 1,0 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 144 | 22 | 34,7 % | non |
| Vert `#3f8c4f` | 3,8 / 2,8 / 1,8 / 2,1 | 1,0 / 1,1 / 1,0 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 125 | 24 | 27,5 % | non |
| Sable `#a58a5c` | 5,4 / 3,0 / 0,5 / 3,5 | 1,1 / 1,1 / 1,0 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 118 | 21 | 29,3 % | non |
| Violet `#7559b5` | 8,5 / 2,7 / 2,2 / 3,0 | 1,1 / 1,1 / 1,1 / 1,1 | 6,6 / 6,6 / 6,4 / 6,3 | 119 | 7 | 23,0 % | non |
| Graphite `#6b7078` | 1,4 / 1,4 / 1,7 / 1,4 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 131 | 15 | 27,7 % | non |
| Gris du thème `#a1a1a1` | 0,6 / 0,6 / 0,6 / 0,6 | 1,1 / 1,1 / 1,1 / 1,1 | 6,4 / 6,4 / 6,4 / 6,4 | 151 | 16 | 43,6 % | non |

### Tol Bright, thème clair

| Candidate | ΔE noyau/sol min (N / P / D / T) | Contraste contour/sol min (N / P / D / T) | ΔE voisins min (N / P / D / T) | Échecs /400 | Échecs signes /40 | Taux pondéré | Tient |
|---|---|---|---|---|---|---|---|
| Bleu `#2f7fc1` | 1,1 / 1,1 / 1,0 / 0,6 | 1,1 / 1,0 / 1,1 / 1,0 | 6,8 / 6,6 / 6,8 / 6,1 | 143 | 11 | 31,8 % | non |
| Ardoise `#5a6f8c` | 4,0 / 3,0 / 2,6 / 3,9 | 1,0 / 1,1 / 1,0 / 1,1 | 6,3 / 6,3 / 6,3 / 6,3 | 145 | 14 | 27,2 % | non |
| Sarcelle `#1f8a8c` | 7,6 / 3,1 / 2,9 / 1,3 | 1,0 / 1,0 / 1,1 / 1,0 | 6,8 / 6,8 / 6,7 / 6,5 | 154 | 22 | 35,1 % | non |
| Vert `#3f8c4f` | 3,8 / 2,8 / 1,8 / 2,1 | 1,0 / 1,0 / 1,1 / 1,0 | 7,0 / 6,9 / 6,9 / 6,9 | 133 | 24 | 27,8 % | non |
| Sable `#a58a5c` | 5,4 / 3,0 / 0,5 / 3,5 | 1,1 / 1,0 / 1,1 / 1,1 | 6,7 / 6,8 / 6,5 / 6,6 | 134 | 21 | 30,2 % | non |
| Violet `#7559b5` | 8,5 / 2,7 / 2,2 / 3,0 | 1,0 / 1,0 / 1,0 / 1,0 | 6,6 / 6,6 / 6,4 / 6,3 | 127 | 7 | 23,3 % | non |
| Graphite `#6b7078` | 1,4 / 1,4 / 1,7 / 1,4 | 1,1 / 1,1 / 1,1 / 1,1 | 6,5 / 6,6 / 6,5 / 6,5 | 139 | 15 | 27,9 % | non |
| Gris du thème `#636a72` | 2,8 / 2,5 / 0,5 / 2,9 | 1,0 / 1,0 / 1,0 / 1,0 | 6,2 / 6,3 / 6,3 / 6,1 | 141 | 16 | 25,7 % | non |

Candidates qui tiennent le seuil partout : **aucune** (0 sur 64 combinaisons candidate × palette × thème). Rampe : les paliers voisins restent à ΔE 6,1 à 6,9 sous les quatre visions pour toutes les candidates, comme le notait D-3 (6,1 à 7,0) : la lisibilité du relief ne départage pas les teintes, le Bleu est le plus bas (6,1, protanopie), le Vert le plus haut (6,9).

## 5. Où échoue la recommandation de chaque palette

Pour la teinte recommandée, les encres de marqueur et d'objectif qui passent sous le seuil, avec les paliers en échec par vision (union des deux thèmes ; `—` : aucun). Les couleurs d'équipe sont en tête.

### Par défaut : Graphite

| Encre | Valeur | N | P | D | T | Pire ΔE |
|---|---|---|---|---|---|---|
| team-ally | `#3B82F6` | — | — | — | 0,68 0,84 1 | 10,6 |
| team-enemy | `#EF4444` | — | 0,36 0,52 0,68 0,84 | 0,84 1 | — | 7,2 |
| zone team-ally | `#3B82F6` | — | — | — | 0,68 0,84 1 | 10,6 |
| zone team-enemy | `#EF4444` | — | 0,36 0,52 0,68 0,84 | 0,84 1 | — | 7,2 |
| divergent-neutral | `#60A5FA` | 1 | 1 | 1 | 1 | 11,6 |
| divergent-neutral (glyphes) | `#60A5FA` | 1 | 1 | 1 | 1 | 11,6 |
| zone-neutral | `#9CA3AF` | 1 | 1 | 1 | 1 | 2,6 |
| chart-series-1 | `#93C5FD` | 1 | — | 1 | 1 | 14,2 |
| chart-series-2 | `#6366F1` | — | — | — | 0,52 0,68 0,84 1 | 10,1 |
| chart-series-3 | `#4F46E5` | — | — | — | 0,36 0,52 0,68 0,84 | 9,5 |
| chart-series-4 | `#4338CA` | — | — | — | 0,36 0,52 0,68 | 8,1 |
| chart-series-5 | `#3730A3` | — | — | — | 0,36 0,52 | 6,8 |
| chart-series-6 | `#10B981` | — | 0,84 1 | 0,84 1 | 0,84 1 | 4,6 |
| chart-series-8 | `#EC4899` | — | 0,52 0,68 0,84 1 | 0,68 0,84 1 | — | 2,4 |
| squad-player-1 | `#2563EB` | — | — | — | 0,52 0,68 0,84 1 | 9,7 |
| squad-player-2 | `#D97706` | — | 0,68 0,84 | — | — | 13,5 |
| squad-player-3 | `#DB2777` | — | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | — | 3,4 |
| squad-player-4 | `#0891B2` | 0,68 0,84 1 | 0,68 0,84 1 | 0,68 0,84 1 | 0,68 0,84 1 | 5,9 |

### Okabe-Ito : Violet

| Encre | Valeur | N | P | D | T | Pire ΔE |
|---|---|---|---|---|---|---|
| team-ally | `#0072B2` | 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 | 3,2 |
| zone team-ally | `#0072B2` | 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 | 3,2 |
| divergent-neutral | `#888888` | 0,68 0,84 1 | 0,68 0,84 1 | 0,68 0,84 1 | 0,68 0,84 1 | 3,0 |
| divergent-neutral (glyphes) | `#888888` | 1 | 1 | 1 | 1 | 5,1 |
| zone-neutral | `#888888` | 1 | 1 | 1 | 1 | 5,1 |
| chart-series-1 | `#0072B2` | 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 | 3,2 |
| chart-series-3 | `#56B4E9` | 1 | 1 | 0,84 1 | 1 | 5,7 |
| chart-series-5 | `#009E73` | — | 1 | 0,84 1 | 0,84 1 | 11,2 |
| chart-series-7 | `#CC79A7` | 0,84 1 | 0,68 0,84 1 | 0,84 1 | 0,84 1 | 5,5 |
| chart-series-8 | `#BBBBBB` | 1 | 1 | 1 | 1 | 12,0 |
| squad-player-1 | `#0072B2` | 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 | 3,2 |
| squad-player-2 | `#B77E00` | — | — | — | 0,84 1 | 12,3 |
| squad-player-3 | `#8E5374` | 0,36 0,52 0,68 0,84 | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 | 0,52 0,68 0,84 | 7,9 |
| squad-player-4 | `#009E73` | — | 1 | 0,84 1 | 0,84 1 | 11,2 |

### Cividis : Vert

| Encre | Valeur | N | P | D | T | Pire ΔE |
|---|---|---|---|---|---|---|
| team-ally | `#0072B2` | — | — | — | 0,36 0,52 0,68 0,84 | 3,8 |
| team-enemy | `#D55E00` | — | 0,36 0,52 0,68 0,84 | 0,68 0,84 1 | — | 6,3 |
| zone team-ally | `#0072B2` | — | — | — | 0,36 0,52 0,68 0,84 | 3,8 |
| zone team-enemy | `#D55E00` | — | 0,36 0,52 0,68 0,84 | 0,68 0,84 1 | — | 6,3 |
| divergent-neutral | `#888888` | 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,68 0,84 1 | 4,0 |
| divergent-neutral (glyphes) | `#888888` | 1 | 1 | 1 | 1 | 8,6 |
| zone-neutral | `#888888` | 1 | 1 | 1 | 1 | 8,6 |
| warning | `#B6A855` | 0,84 1 | 0,68 0,84 1 | 0,84 1 | 0,84 1 | 5,8 |
| chart-series-3 | `#3F4A6B` | 0,36 | 0,36 0,52 | 0,36 0,52 | 0,36 0,52 | 3,2 |
| chart-series-4 | `#B6A855` | 0,84 1 | 0,68 0,84 1 | 0,84 1 | 0,84 1 | 5,8 |
| chart-series-5 | `#575C6D` | 0,36 0,52 0,68 | 0,36 0,52 0,68 | 0,36 0,52 0,68 | 0,36 0,52 0,68 | 6,0 |
| chart-series-6 | `#928D6B` | 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,68 0,84 1 | 1,0 |
| chart-series-7 | `#1B3158` | — | — | — | 0,36 | 10,6 |
| chart-series-8 | `#E1C752` | — | 1 | — | — | 12,9 |
| squad-player-1 | `#3F4A6B` | 0,36 | 0,36 0,52 | 0,36 0,52 | 0,36 0,52 | 3,2 |
| squad-player-2 | `#7C7B78` | 0,52 0,68 0,84 1 | 0,36 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 4,1 |
| squad-player-3 | `#B6A855` | 0,84 1 | 0,68 0,84 1 | 0,84 1 | 0,84 1 | 5,8 |

### Tol Bright : Violet

| Encre | Valeur | N | P | D | T | Pire ΔE |
|---|---|---|---|---|---|---|
| team-ally | `#4477AA` | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 | 2,2 |
| team-enemy | `#EE6677` | — | 0,68 0,84 1 | 1 | — | 11,0 |
| zone team-ally | `#4477AA` | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 | 2,2 |
| zone team-enemy | `#EE6677` | — | 0,68 0,84 1 | 1 | — | 11,0 |
| divergent-neutral | `#888888` | 0,68 0,84 1 | 0,68 0,84 1 | 0,68 0,84 1 | 0,68 0,84 1 | 3,0 |
| divergent-neutral (glyphes) | `#888888` | 1 | 1 | 1 | 1 | 5,1 |
| zone-neutral | `#888888` | 1 | 1 | 1 | 1 | 5,1 |
| warning | `#CCBB44` | — | — | — | 1 | 13,1 |
| chart-series-1 | `#4477AA` | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 | 2,2 |
| chart-series-2 | `#EE6677` | — | 0,68 0,84 1 | 1 | — | 11,0 |
| chart-series-3 | `#228833` | — | — | — | 0,52 0,68 0,84 | 10,9 |
| chart-series-4 | `#CCBB44` | — | — | — | 1 | 13,1 |
| chart-series-5 | `#66CCEE` | — | — | 1 | — | 11,8 |
| chart-series-6 | `#AA3377` | 0,52 0,68 | 0,36 0,52 0,68 | 0,52 0,68 0,84 | — | 4,8 |
| chart-series-7 | `#BBBBBB` | 1 | 1 | 1 | 1 | 12,0 |
| squad-player-1 | `#0077BB` | 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 0,52 0,68 0,84 1 | 2,7 |
| squad-player-3 | `#AA4499` | 0,52 0,68 0,84 | 0,36 0,52 0,68 0,84 | 0,52 0,68 0,84 1 | 0,68 0,84 | 2,9 |
| squad-player-4 | `#117733` | — | — | — | 0,36 0,52 0,68 0,84 | 10,0 |

Validation du banc contre D-3 : rouge `#EF4444` sur la rampe Vert, protanopie : ΔE mini 1,7 (D-3 : 2). Bleu `#3B82F6` sur la rampe Bleu : ΔE mini N 10,0, P 10,2, D 11,4, T 2,4 (D-3 : « entre 11 et 14 dans toutes les visions » ; vrai pour N, P, D sur le palier médian, mais la **tritanopie descend à 2,4** sur le palier le plus sombre, ce que D-3 ne mentionnait pas).

## 6. Constats hors périmètre

1. **Le liseré du marqueur n'est pas sombre.** Le commentaire de `replayMarkers.ts` parle d'un « liseré sombre » et de la planche du 2026-08-16, mais `style.ink` reçoit `floorEdge` (`replayRenderer.ts`, `ink: floorEdge`) = `--muted-foreground` : un gris clair en thème sombre, un gris moyen en thème clair. Sensibilité mesurée sur les 64 combinaisons (cellules de marqueur sous le seuil, sur 19200) : liseré actuel 6548 (34,1 %) ; liseré `#080c12` à 0,62 (la teinte de `--replay-label-stroke`) 3547 (18,5 %) ; liseré `--foreground` 5296 (27,6 %). À trancher avant D.5 : l'encre du liseré pèse du même ordre que la teinte dans le seuil.
2. **La teinte seule ne suffit pas.** Un balayage de 63 bases libres (12 teintes × 5 combinaisons de saturation et de clarté, plus trois gris) laisse au mieux, pour le rendu par défaut seul, Par défaut : 8 cellules en échec (base `#783a59`) ; Okabe-Ito : 22 cellules en échec (base `#783a78`) ; Cividis : 23 cellules en échec (base `#783a78`) ; Tol Bright : 26 cellules en échec (base `#3333cc`), contre 28, 47, 50, 55 pour la meilleure des 8 candidates imposées (même ordre de palettes). Les bases qui font le mieux sont, pour la plupart, les plus sombres (clarté 0,35 ou gris 0,3) : le levier est la clarté de la rampe (limite haute du sol), pas la teinte. Hors liste de D-3 ; rapporté sans le recommander.
3. **Le « gris du thème » demande une valeur par thème** (`#a1a1a1` et `#636a72`). Le seul mécanisme de jeton dépendant du thème est `paletteForTheme` (rampes d'impact) ; D-3 veut une teinte par palette, sans thème. Les sept préréglages n'ont pas ce défaut. Pour mémoire, sur la palette par défaut, le gris du thème est la meilleure candidate du thème clair (10,5 % contre 13,9 % pour Graphite) et dans le bas du classement du thème sombre (22,1 % contre 13,6 %) : c'est cet écart entre thèmes qui le sort du classement.
4. **Les couleurs d'équipe sont réglables par l'utilisateur** (`--ac-team-ally` / `--ac-team-enemy` posés par `theme-provider.tsx`) : le seuil de D-3 ne peut être tenu que pour les valeurs de palette, jamais pour un rouge ou un bleu choisis à la main.
5. **L'encre « sans camp » des trois palettes daltoniennes est un gris moyen** (`divergent-neutral` et `zone-neutral` valent `#888888` dans Okabe-Ito, Cividis et Tol Bright). Un gris moyen est toujours à moins de 15 ΔE d'un des cinq paliers d'une rampe qui traverse la clarté : le marqueur sans camp échoue dans 12 à 16 des 20 cellules (5 paliers × 4 visions) quelle que soit la teinte retenue, et c'est un plancher commun à ces trois palettes (21 à 27 % de taux d'échec pondéré, même pour la meilleure teinte). Tol Bright porte en plus `#000000` et `#BBBBBB` dans ses couleurs distinctes (`chart-series-8`, `chart-series-7`), Okabe-Ito `#BBBBBB` (`chart-series-8`).
6. **Le sol vit dans les deux paliers sombres** : 36 % de la surface des quatre cartes est au gris 0,36 et 36 % au gris 0,52 (Behemoth : 67 % au plus bas). Le palier 1,0 ne représente que 2 % : un seuil appliqué sur toute la rampe pèse autant le palier rare que le palier courant. Le taux pondéré du §1 est une lecture plus proche de l'écran ; le user peut demander que D-3 s'applique sur 0,36 à 0,84 seulement.
7. Le banc reproduit ce que D-3 donnait : rouge sur Vert en protanopie à ΔE 1,7 (D-3 : 2) ; rampe à 6,1 à 6,9 de ΔE entre paliers (D-3 : 6,1 à 7,0).

## 7. Fichiers

- `.ai/V8.3/annexes_fonds_relief/teinte.md` : ce document.
- `.ai/V8.3/annexes_fonds_relief/teinte_choix.html` : page de choix, une teinte par palette parmi les trois premières, 4 cartes témoins teintées, 4 encres d'essai, sélecteurs de palette, de thème et de vision.
- `.ai/V8.3/annexes_fonds_relief/teinte_images/*.gris.png` : le gris ombré (lumière nord-ouest, rampe de clarté `height.low` à `height.high`, formule de la maquette du 2026-11-18) de chacune des quatre cartes, réduit à 600 px de large au plus (ombrage calculé en pleine résolution puis réduit), gris et alpha ; la teinte est appliquée par la page, une seule image par carte pour toutes les teintes.
