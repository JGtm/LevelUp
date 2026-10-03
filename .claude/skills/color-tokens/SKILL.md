# Skill : color-tokens — Système de couleurs sémantiques (apps/web)

## Règle fondamentale

**Aucun hex (`#RRGGBB`) ni classe Tailwind de couleur** (`text-red-*`, `bg-green-*`, etc.)
dans `apps/web/src/features/` ou `apps/web/src/components/`.

Toute couleur sémantique passe par les helpers d'accessibilité.

## APIs selon le contexte

| Contexte | Helper | Import |
|---|---|---|
| JSX (className / style) | `tokenCssVar(token)` | `apps/web/src/lib/accessibility/` |
| Plotly / SVG (valeur hex résolue) | `resolveToken(token, tokens)` | idem |
| Séries de données (N couleurs) | `getSeriesColors(n, tokens[])` | idem |
| Hook React (couleur réactive) | `useColor(token)` | idem |

## Exemples

```tsx
// JSX — variable CSS
style={{ color: tokenCssVar('primary') }}

// Plotly — valeur hex résolue
const color = resolveToken('series1', tokens);

// Série de N couleurs pour un chart
const colors = getSeriesColors(5, tokens);
```

## Fichiers source

```
apps/web/src/lib/accessibility/
  semantic-tokens.ts     — liste de tous les tokens disponibles
  resolveToken.ts        — resolveToken()
  useColor.ts            — useColor()
  applyPalette.ts        — applyPalette()
  palettes/default.ts    — palette par défaut
  palettes/okabe-ito.ts  — palette Okabe-Ito (daltonisme)
  scales/               — makeCategoricalScale, makeOrdinalScale, makeDivergentScale
```

## Tokens sémantiques courants

Consulter `semantic-tokens.ts` pour la liste complète. Exemples typiques :
`primary`, `secondary`, `success`, `warning`, `danger`, `series1`…`seriesN`, `neutral`.

## Stats de combat — famille dédiée (2026-09-17)

Toute couleur qui dit « c'est un frag / une mort / une assistance / ce sens d'assistance »
passe par ces cinq jetons, dans toutes les pages :

| Jeton | Rôle |
|---|---|
| `stat-kills` | frags |
| `stat-deaths` | morts |
| `stat-assists` | assistances (y compris le bonus assistances/3 et la marque du rejeu) |
| `assist-received` | un autre joueur t'assiste (« il te sert ») |
| `assist-given` | tu assistes un autre joueur (« tu le sers ») |

- **Part de dégâts d'une assistance** : jamais une autre teinte. Trois tons de la couleur du
  sens pour les tranches < 25 %, 25-50 %, > 50 % — trois CLARTÉS OKLCH du jeton, montant vers
  le premier plan du thème (`light-dark`, chroma relevée), PAS des opacités (remplacées le
  2026-09-19 : elles noyaient les tons faibles dans la piste ; le jeton lui-même ne peut pas
  s'éclaircir, il est retenu par le contraste clair et la séparation deutéranopie) —
  `features/_shared/assists/assistTierTone.ts`, bornes `domain.AssistTier*MaxPct`.
- **Ne relèvent PAS de la famille** (garder la couleur de leur rôle) : couleur d'équipe par
  camp (`team-*`), couleur de joueur (`squad-player-*`), classe d'arme (`frag-*`), issue du
  match, échelle de qualité, accent de sous-type de frag (frags parfaits, tirs à la tête…).
- **Garde-fous** : `lib/accessibility/combatStatTokens.test.ts` (contraste + écarts, valeurs
  par palette) et `lib/accessibility/combatStatNoBorrow.guard.test.ts` (ratchet : un jeton
  emprunté sur une ligne qui nomme une stat échoue, sauf usage justifié dans `KEPT`).
- **Distance de couleur** dans un test : importer `deltaE` / `simulateCvd` depuis
  `lib/accessibility/colorDistance.ts` (copie de la matrice OKLab interdite par ratchet).

## Exceptions tolérées (avec commentaire justificatif)

| Exception | Localisation |
|---|---|
| Couleurs de rareté Halo (Battlepass) | `rarity.ts` |
| Couleurs structurelles layout SVG (fond de piste, bordure) | composant SVG dédié |
| `liked/rose` (badge like) | composant UI générique |
| `warning/amber` (badge d'état système) | composant UI générique |

## Outils de vérification

```bash
# Chercher des hex directs dans features/
grep -r "#[0-9a-fA-F]\{6\}" apps/web/src/features/

# Chercher des classes Tailwind couleur
grep -r "text-\(red\|green\|blue\|yellow\|amber\|rose\)-" apps/web/src/features/
```
