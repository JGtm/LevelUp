# Squad — Chart Helpers (specifications)

> Specifications des helpers TypeScript a implementer dans [apps/web/src/features/squad/charts/_helpers.ts](../apps/web/src/features/squad/charts/_helpers.ts) (a creer en debut de Phase 0).
>
> Reference Python : `src/visualization/theme.py`, `src/ui/streamlit_modern.py`, `src/visualization/_chart_series.py`, `src/ui/date_formats.py`.

---

## 1. Formatters

### 1.1 `formatDuration(seconds, format)`

Convertit secondes -> string lisible. Equivalent `format_duration_dhm` / `format_duration_hms` / `format_mmss` (Python).

```typescript
type DurationFormat = 'auto' | 'mm:ss' | 'h:mm:ss' | 'd:h:mm'

function formatDuration(seconds: number | null, format: DurationFormat = 'auto'): string
```

**Specs** :
- `null` ou `0` -> `'-'`
- `'mm:ss'` : `0:01` a `59:59`
- `'h:mm:ss'` : `0:00:01` a `23:59:59`
- `'d:h:mm'` : `1d 2h 3m` (suffixes localises FR/EN)
- `'auto'` : choisit selon magnitude (`< 1h` -> mm:ss, `< 24h` -> h:mm:ss, sinon d:h:mm).

**Tests** : 6 cas (null, 0, 30, 3600, 86400, 90061).

---

### 1.2 `formatNumber(n, options)`

Equivalent `_fmt(value)` Python (multiples lieux).

```typescript
interface NumberFormatOptions {
  decimals?: number       // defaut: auto (2 si < 10, 1 si < 100, 0 sinon)
  compact?: boolean       // 1.2k / 3.4M
  forceSign?: boolean     // +5.3 / -2.1
}

function formatNumber(n: number | null, options?: NumberFormatOptions): string
```

**Specs** :
- `null` -> `'-'`
- `compact=true` : seuils 1000 -> `1k`, 1_000_000 -> `1M`, 1e9 -> `1G`. Toujours 1 decimale en compact.
- Separateur milliers selon locale (FR : espace insecable, EN : virgule).

**Tests** : 8 cas (null, 0, 1.5, 99.5, 1234, 1234567, -42.6, +42.6 forceSign).

---

### 1.3 `formatKDA(value)`

Toujours 3 decimales (convention Python `_TRIO_METRIC_SPECS` ratio).

```typescript
function formatKDA(value: number | null): string  // ex: '1.245', '-'
```

---

### 1.4 `formatPercent(value, decimals)`

```typescript
function formatPercent(value: number | null, decimals = 1): string  // ex: '53.2%', '-'
```

**Note** : `value` doit etre dans la convention 0-100 (pas 0-1) pour matcher la convention Python.

---

### 1.5 `formatDate(date, locale, format)`

Date locale FR. Equivalent `FMT_DATETIME_FR` (`src/ui/date_formats.py`).

```typescript
type DateFormat = 'short' | 'long' | 'datetime' | 'time-only'

function formatDate(date: Date | string | null, locale: 'fr' | 'en', format: DateFormat = 'datetime'): string
```

**Specs FR** :
- `short` : `26/04/2026`
- `long` : `26 avril 2026`
- `datetime` : `26/04/2026 14:32`
- `time-only` : `14:32`

**Specs EN** :
- `short` : `04/26/2026`
- `long` : `April 26, 2026`
- `datetime` : `04/26/2026 2:32 PM`
- `time-only` : `2:32 PM`

---

## 2. Couleurs et palette

### 2.1 `attributePlayerColor(xuid, palette)`

Mapping deterministe joueur -> couleur (cf. `SQUAD_DESIGN_TOKENS.md` §1).

```typescript
function attributePlayerColor(xuid: string, palette: readonly string[]): string
```

**Specs** :
- Hash deterministe (DJB2 ou similaire) sur `xuid`.
- Modulo `palette.length`.
- **Cas special "moi"** : si `xuid === currentPlayerXuid`, retourne `palette[0]` toujours (a confirmer Phase 0bis).

---

### 2.2 `negativeColor(hex)`

Variante sombre/desaturee pour barres deaths inversees.

```typescript
function negativeColor(hex: string): string
```

**Specs** : extraire la formule exacte depuis `src/visualization/trio.py::_negative_color`. Probable : conversion HSL, multiplie L par 0.6, retour hex.

**Tests** : 8 cas (chaque couleur Okabe-Ito).

---

### 2.3 `outcomeColor(outcome)` et `outcomeMarkerSymbol(outcome)`

```typescript
type Outcome = 'win' | 'loss' | 'tie' | 'dnf' | 'unknown'

function outcomeColor(outcome: Outcome): string  // tokenCssVar('outcome.win') etc.
function outcomeMarkerSymbol(outcome: Outcome): 'circle' | 'cross' | 'diamond' | 'square-open'
function outcomeBackgroundRgba(outcome: Outcome): string  // pour heatmap impact
```

Source : `_OUTCOME_BG` Python + `SQUAD_DESIGN_TOKENS.md` §2.

---

## 3. Style Plotly

### 3.1 `applyHaloPlotStyle(figure, opts?)`

Applique la theme Halo (police, marges, fond transparent, axes).

```typescript
interface HaloPlotStyleOptions {
  height?: number | null      // null = laisse l'option en place
  forceZeroLine?: boolean     // §4.1 stats par minute
  showGrid?: boolean
}

function applyHaloPlotStyle(
  figure: PlotlyFigurePayload,
  opts?: HaloPlotStyleOptions,
): PlotlyFigurePayload
```

**Specs** : equivalent `apply_halo_plot_style()` (`src/visualization/theme.py`). A extraire en Phase 0bis :
- `paper_bgcolor`, `plot_bgcolor` = transparent.
- `font.family`, `font.size`, `font.color`.
- `xaxis.gridcolor`, `yaxis.gridcolor`.
- Margins `{l, r, t, b}` standards.

Si `forceZeroLine=true` : `yaxis.zeroline=true`, `zerolinecolor='rgba(255,255,255,0.75)'`, `zerolinewidth=2`.

---

### 3.2 `hideLegend(figure)`

Masque la legende (utilise quand le panneau lateral fixe affiche les noms).

```typescript
function hideLegend(figure: PlotlyFigurePayload): PlotlyFigurePayload
```

**Specs** : equivalent `_hide_legend()` Python — modifie `layout.showlegend = false` ET passe `showlegend=false` sur chaque trace.

---

### 3.3 `applyRecordsOverlay(figure, records, palette)`

Ajoute des traces fantomes hachurees pour les records par joueur/metrique.

```typescript
interface RecordOverlayInput {
  playerName: string
  metricName: string
  recordValue: number
  color: string
}

function applyRecordsOverlay(
  figure: PlotlyFigurePayload,
  records: RecordOverlayInput[],
): PlotlyFigurePayload
```

**Specs** : pour chaque record, ajoute un trace `bar` transparent avec `marker.pattern.shape='/'`, `fgopacity=0.4`. Position en x = meme que le trace principal du joueur, y = recordValue. Equivalent `SquadRecordSet.add_record_overlays`.

---

## 4. Helpers de structure

### 4.1 `clampDataframe(rows, max)`

Tri par date desc + slice top N. Utilise pour les "20 dernieres cartes" §3.1.

```typescript
function clampDataframe<T extends { start_time: string }>(rows: T[], max: number): T[]
```

---

### 4.2 `intersectMatchIds(perPlayer)`

Retourne les match_ids communs a tous les joueurs (intersection N-aire).

```typescript
function intersectMatchIds(perPlayer: ReadonlyArray<Set<string>>): Set<string>
```

Equivalent `_collect_friend_match_ids` cote Python. Utile cote frontend pour validation rapide avant de lancer un fetch.

---

### 4.3 `prepareTimeAxis(rows)`

Genere les labels d'axe Y pour les heatmaps match-based : carte si dispo, sinon date.

```typescript
function prepareTimeAxis(rows: Array<{ match_id: string; map_ui?: string; start_time: string }>):
  { labels: string[]; tickvals: number[] }
```

Equivalent `prepare_time_axis()` (`src/visualization/_timeseries_helpers.py`).

---

## 5. A produire en Phase 0bis

- [ ] Cette doc relue + completee avec les valeurs exactes (`_negative_color` formule, `apply_halo_plot_style` options).
- [ ] Decision `currentPlayerXuid` toujours palette[0] OU hash partout.
- [ ] Cas limite `formatDuration` en ms (precision sub-seconde) si necessaire.
- [ ] Verification `formatPercent` convention 0-1 vs 0-100 dans toutes les sources backend.

## 6. Implementation (Phase 0)

Une fois cette doc finalisee, creer le fichier `apps/web/src/features/squad/charts/_helpers.ts` avec les 11 helpers ci-dessus + tests Vitest dans `_helpers.test.ts`.

Couverture minimale :
- 100 % branches `formatDuration` (cas auto + chacun des 3 formats).
- 100 % branches `formatNumber` (compact + decimals + forceSign + null).
- 100 % branches `attributePlayerColor` (hash deterministe verifie).
- 100 % branches `outcomeColor` / `outcomeMarkerSymbol` (5 outcomes).
- Snapshot pour `applyHaloPlotStyle` et `applyRecordsOverlay` (figure avant/apres).
