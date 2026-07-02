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
