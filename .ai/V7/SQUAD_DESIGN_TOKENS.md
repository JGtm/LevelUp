# Squad — Design Tokens

> Source de verite pour les couleurs, palettes et symboles utilises dans toutes les visualisations de la page Coequipiers.
>
> Reference Python : `src/config/__init__.py` (OKABE_ITO_PALETTE), `src/visualization/theme.py` (apply_halo_plot_style), `src/ui/components/performance.py` (SCORE_THRESHOLDS).

---

## 1. Palette joueurs — Okabe-Ito (8 couleurs)

Palette pensee pour daltonisme, ordre canonique :

| Index | Nom | Hex | Token CSS suggere | Usage |
|------:|-----|-----|-------------------|-------|
| 0 | Orange | `#E69F00` | `--squad-player-1` | Joueur principal (moi) |
| 1 | Sky Blue | `#56B4E9` | `--squad-player-2` | Coequipier 1 |
| 2 | Bluish Green | `#009E73` | `--squad-player-3` | Coequipier 2 |
| 3 | Yellow | `#F0E442` | `--squad-player-4` | Coequipier 3 |
| 4 | Blue | `#0072B2` | `--squad-player-5` | Reserve |
| 5 | Vermilion | `#D55E00` | `--squad-player-6` | Reserve |
| 6 | Reddish Purple | `#CC79A7` | `--squad-player-7` | Reserve |
| 7 | Black/Gray | `#999999` | `--squad-player-8` | Reserve |

> A confirmer / completer en Phase 0bis : verifier les hex exacts dans `src/config/__init__.py` du repo Python `v7/cockpit`.

### Mapping deterministe joueur -> couleur

L'attribution doit etre **stable entre sessions** pour le meme joueur. Algo recommande :

```typescript
function attributePlayerColor(xuid: string, palette: string[]): string {
  let hash = 0
  for (let i = 0; i < xuid.length; i++) {
    hash = (hash << 5) - hash + xuid.charCodeAt(i)
    hash |= 0
  }
  return palette[Math.abs(hash) % palette.length]
}
```

> A trancher en Phase 0bis : faut-il privilegier "moi = palette[0] toujours" et hash uniquement les coequipiers ? **Probablement oui** pour la coherence UX.

### Variantes "negatives" (deaths inversees)

Pour les barres deaths affichees vers le bas (§4.1 Stats par minute Plotly), Python utilise `_negative_color(color)` qui assombrit/desature. Equivalent TS a definir :

```typescript
function negativeColor(hex: string): string {
  // TODO Phase 0bis : extraire la formule exacte de src/visualization/trio.py::_negative_color
  // probablement : HSL(hex).withLightness(L * 0.6).toHex()
}
```

---

## 2. Couleurs outcome (W / L / T / DNF)

| Outcome | Code | Background heatmap impact | Token semantique | Marker symbol Plotly |
|---------|-----:|---------------------------|------------------|----------------------|
| Win | 2 | `rgba(0,158,115,0.30)` | `--outcome-win` | `circle` |
| Loss | 3 | `rgba(213,94,0,0.30)` | `--outcome-loss` | `cross` |
| Tie | 1 | `rgba(100,100,130,0.15)` | `--outcome-tie` | `diamond` |
| DNF | 4 | `rgba(182,196,214,0.45)` | `--outcome-dnf` | `square-open` |

Source Python : `src/ui/pages/teammates_impact.py` (`_OUTCOME_BG`, `_OUTCOME_BG_TIE`).

---

## 3. Heatmap colorscales

### 3.1 Performance score (0-100)

Diverging du gris au vert-bleu profond. Centre 50 = neutral.

| Stop | Couleur | Token |
|-----:|---------|-------|
| 0 | `#7B1F1F` (rouge sombre) | `--perf-0` |
| 25 | `#C46B6B` | `--perf-25` |
| 50 | `#888888` (neutre) | `--perf-50` |
| 75 | `#4FAFA1` | `--perf-75` |
| 100 | `#1F7A6C` (vert profond) | `--perf-100` |

> A confirmer en Phase 0bis : extraire la colorscale exacte depuis `src/visualization/squad_map_heatmap.py`.

### 3.2 Win rate (0-100)

Sequentiel orange -> vert.

### 3.3 Intensity (0-1 normalise)

Sequentiel cool (bleu clair -> bleu profond).

> A spec en Phase 0bis depuis `src/visualization/match_intensity_heatmap.py`.

---

## 4. Score thresholds (cartes performance)

Source : `src/ui/components/performance.py::SCORE_THRESHOLDS` + `get_score_class`.

| Tier | Borne min | Classe CSS | Token suggere |
|------|----------:|------------|---------------|
| Excellent | 80 | `text-excellent` | `--score-excellent` |
| Good | 65 | `text-good` | `--score-good` |
| Average | 50 | `text-average` | `--score-average` |
| Poor (below_average) | 35 | `text-poor` | `--score-poor` |
| Bad | 0 | `text-bad` | `--score-bad` |
| N/A | null | `text-neutral` | `--score-neutral` |

> A confirmer : extraire les bornes exactes depuis `SCORE_THRESHOLDS` Python.

---

## 5. Records overlay — motif Plotly

Pour les barres "fantomes" hachurees (records par metrique) :

```javascript
{
  type: 'bar',
  marker: {
    color: 'rgba(255,255,255,0)',  // transparent
    line: { color: playerColor, width: 1 },
    pattern: {
      shape: '/',           // hachures diagonales
      fgcolor: playerColor,
      fgopacity: 0.4,
      size: 8,
      solidity: 0.3,
    },
  },
  showlegend: false,
}
```

Source : `src/visualization/_chart_series.py::SquadRecordSet.add_record_overlays`.

---

## 6. Emojis impact (8 roles)

| Role | Emoji | Fallback texte | Token semantique |
|------|------:|----------------|------------------|
| First Blood | ⚡ | `FB` | `--impact-first-blood` |
| Clutch Finisher | 🎯 | `CL` | `--impact-clutch` |
| Dead Weight (last_casualty) | 💀 | `DW` | `--impact-deadweight` |
| Tourist (last_group_kill) | 🐌 | `TR` | `--impact-tourist` |
| First Casualty (first_group_death) | 🪦 | `FC` | `--impact-firstcasualty` |
| Silent Hero | 🛡️ | `SH` | `--impact-silenthero` |
| False Brother | 🗡️ | `FB2` | `--impact-falsebrother` |
| Top Killer | 💥 | `TK` | `--impact-topkiller` |

Source : `src/ui/pages/teammates_impact.py::_EVENT_TO_EMOJI`.

**Roles inverses** (gradient Okabe-Ito inverse pour le ranking) : `last_casualty`, `last_group_kill`, `first_group_death`, `false_brother`. Source : `_IMPACT_INVERTED`.

---

## 7. Mapping CSS Python -> React

| Classe Python | Composant / classe React | Note |
|---------------|--------------------------|------|
| `os-perf-card` | `<Card>` shadcn + `os-perf-card.tsx` | Score performance carte. |
| `os-perf-card--compact` | variante `compact` du composant | Reduction du padding + meta caches. |
| `os-perf-card__label` | `<CardHeader>` | Titre carte. |
| `os-perf-card__score` | `<div class="text-3xl font-bold">` | Score gros chiffre. |
| `os-perf-card__status` | `<div class="text-sm">` | Label qualitatif. |
| `os-perf-card__meta` | `<CardFooter>` | Nb matchs (mode non-compact). |
| `os-perf-card__detail` | `<div class="text-xs text-muted">` | Detail bonus equipe. |
| `os-table` | `<Table>` shadcn | Tableaux historique / armes. |
| `os-table-wrap` | `<div class="overflow-x-auto">` | Wrapper scroll horizontal. |
| `v7-context-toolbar-label` | `<Label>` au-dessus du multiselect | Label `Selectionne au moins un coequipier`. |
| `text-excellent` / `text-good` / ... | classes utilitaires Tailwind ou tokens custom | A definir dans `apps/web/src/styles/tokens.css`. |

---

## 8. A produire en Phase 0bis

- [ ] Verifier les 8 hex Okabe-Ito exacts dans le code Python.
- [ ] Extraire la formule `_negative_color()` exacte.
- [ ] Definir les tokens CSS `--squad-player-*` dans `apps/web/src/styles/`.
- [ ] Definir les colorscales heatmap (extraire stops Plotly Python).
- [ ] Confirmer les bornes `SCORE_THRESHOLDS`.
- [ ] Decider mapping moi/coequipiers (moi toujours palette[0] ou hash partout ?).
- [ ] Implementer `tokenCssVar('squad.player.1')` etc. dans `lib/tokens/`.
