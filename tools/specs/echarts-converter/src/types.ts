/**
 * Types TypeScript pour les fichiers YAML de spec graphique.
 * Suit `.ai/charts_specs/_schema.yaml`.
 */

export type ChartKind =
  | 'stacked_bar'
  | 'grouped_bar'
  | 'bar_diverging'
  | 'line'
  | 'scatter'
  | 'scatter_matrix'
  | 'histogram'
  | 'heatmap'
  | 'gauge'
  | 'indicator'
  | 'radar'
  | 'pie'
  | 'sunburst'
  | 'bullet'
  | 'lollipop'
  | 'table_html'
  | 'kpi_row'
  | 'composite_block';

export interface ChartSpec {
  id: string;
  title: string;
  page: string;
  section?: string;
  source_function: string;
  source_helpers?: string[];
  fragment?: boolean;
  chart_kind: ChartKind;
  data: DataSection;
  traces: TraceSpec[];
  heatmap?: HeatmapSection;
  layout: LayoutSection;
  display: DisplaySection;
  controls?: ControlSpec[];
  i18n_keys?: { viz_t?: string[]; t?: string[] };
  interactivity?: InteractivitySection;
  fingerprint?: Record<string, unknown>;
}

export interface DataSection {
  computed_by: 'server' | 'client' | 'hybrid';
  upstream_service?: string | null;
  input_dataframe?: {
    required_columns?: string[];
    optional_columns?: string[];
    polars_dtype_hints?: Record<string, string>;
    [key: string]: unknown;
  };
  call_args?: Record<string, unknown>;
  call_args_semantics?: Record<string, string>;
  filters_applied?: string[];
  transformations?: Array<{
    description: string;
    polars?: string;
    sql_duckdb?: string;
    detail?: string;
    [key: string]: unknown;
  }>;
  bucket_logic?: { rule: string; thresholds: Record<string, string> };
  timezone?: string;
  metrics_definition?: Array<Record<string, unknown>>;
}

export interface TraceSpec {
  id?: string;
  name: string;
  type: string; // go.Bar | go.Scatter | go.Heatmap
  chart_role?: 'primary' | 'secondary' | 'overlay' | 'reference';
  color: string | null;
  color_token?: string;
  direction?: 'positive' | 'negative';
  orientation?: 'h' | 'v' | null;
  opacity?: number;
  width?: number;
  clip?: boolean;
  secondary_y?: boolean;
  fill?: string;
  line?: { width?: number; dash?: string; shape?: string };
  marker?: { size?: number; symbol?: string };
  mode?: string;
  data_transform?: {
    x_expression?: string | null;
    y_expression?: string | null;
    explanation?: string;
  };
  customdata?: {
    fields?: string[];
    schema?: Record<string, string>;
  };
  hovertemplate?: string;
  hover_format_tokens?: Record<string, string>;
  text_data?: {
    source: string;
    template?: string | null;
    empty_when?: string | null;
    position?: string | null;
    font_size?: number | null;
  };
  show_when?: string;
}

export interface HeatmapSection {
  z_field: string;
  x_field: string;
  y_field: string;
  z_range?: [number, number];
  colorscale: {
    type: 'linear' | 'discrete';
    stops: Array<{ value: number; color: string }>;
  };
  nan_treatment?: 'mask' | 'interpolate' | 'zero';
  cell_label?: {
    source?: string;
    empty_when?: string;
    font_size?: number;
  };
  colorbar?: {
    title?: string;
    tickformat?: string;
    visible?: boolean;
  };
  x_labels?: { type: string; values?: string[]; format?: string };
  y_labels?: { type: string; values_fr?: string[]; values_en?: string[]; values?: string[] };
}

export interface LayoutSection {
  inherits?: string;
  height: {
    value?: number | null;
    expression?: string | null;
    branches?: Array<{ when?: string; else?: boolean; height: number }>;
    explanation?: string;
  };
  margin: { l: number; r: number; t: number; b: number };
  margin_override?: { l: number; r: number; t: number; b: number };
  barmode?: string | null;
  bargap?: number | null;
  hovermode?: string | null;
  showlegend?: boolean;
  legend?: LegendConfig | { inherits: string };
  legend_override?: LegendConfig;
  xaxis?: AxisConfig;
  yaxis?: AxisConfig;
  yaxis2?: AxisConfig & { overlaying?: string; side?: string };
  shapes?: ShapeSpec[];
  annotations?: AnnotationSpec[];
}

export interface LegendConfig {
  orientation?: 'h' | 'v';
  yanchor?: string;
  y?: number;
  xanchor?: string;
  x?: number;
}

export interface AxisConfig {
  type?: string;
  title?: string | null;
  tickformat?: string | null;
  tickangle?: number;
  range?: [number, number] | null;
  autorange?: 'reversed' | null;
  automargin?: boolean;
  showticklabels?: boolean;
  showgrid?: boolean;
  zeroline?: boolean;
  side?: string | null;
}

export interface ShapeSpec {
  type: string;
  kind: 'vline' | 'hline' | 'rect';
  x?: number;
  y?: number;
  line_width?: number;
  line_color?: string;
  opacity?: number;
  annotation?: unknown;
  explanation?: string;
}

export interface AnnotationSpec {
  text: string;
  x_ref?: string;
  y_ref?: string;
  x_expression?: string;
  y_expression?: string;
  font?: { color?: string };
  showarrow?: boolean;
}

export interface DisplaySection {
  shown_when?: string;
  hidden_when?: string | null;
  empty_state?: {
    type: string;
    message?: string;
    message_per_case?: Array<{ case: string; condition: string; message: string }>;
    triggered_by?: string[];
  };
  config?: string; // PLOTLY_CLEAN_CONFIG | PLOTLY_STATIC_CONFIG
  width_mode?: string;
  filters_widgets?: string[];
  fragment_isolation?: boolean;
  layout_position?: Record<string, unknown>;
  preceded_by?: string[];
}

export interface ControlSpec {
  id: string;
  label?: string;
  widget: string;
  position: string;
  scope: 'page' | 'section' | 'chart';
  options?: Record<string, unknown>;
  effect: string;
  effect_detail?: string;
  api_param_mapping?: Record<string, unknown>;
  requires_refetch?: boolean;
  affects_charts?: string[];
}

export interface InteractivitySection {
  click_target?: string | null;
  drilldown_chain?: string[];
  session_state_writes?: string[];
}

// === Theme default ===

export interface ThemeDefault {
  template: string;
  bgcolor: { paper: string; plot: string };
  font: { color: string; size: number };
  hoverlabel: { bgcolor: string; bordercolor: string };
  axes_default: {
    showgrid: boolean;
    gridcolor: string;
    zeroline: boolean;
    showline: boolean;
    linecolor: string;
  };
  legend_horizontal_bottom: LegendConfig;
  legend_horizontal_top: LegendConfig;
  margin_default: { l: number; r: number; t: number; b: number };
  heights: {
    default: number;
    tall: number;
    compact: number;
    indicator: number;
  };
  palette: Record<string, string | Record<string, string>>;
  special_colors: Record<string, string>;
  plotly_configs: Record<string, Record<string, unknown>>;
}

// === ECharts option (typage minimal — pas exhaustif, juste ce qu'on génère) ===

export interface EChartsOption {
  backgroundColor?: string;
  textStyle?: { color?: string; fontSize?: number };
  title?: {
    text?: string;
    left?: string | number;
    right?: string | number;
    top?: string | number;
    bottom?: string | number;
    textStyle?: { color?: string; fontSize?: number; fontWeight?: string | number };
  };
  tooltip?: Record<string, unknown>;
  legend?: Record<string, unknown> | false;
  grid?: { left?: number; right?: number; top?: number; bottom?: number; containLabel?: boolean };
  xAxis?: Record<string, unknown> | Array<Record<string, unknown>>;
  yAxis?: Record<string, unknown> | Array<Record<string, unknown>>;
  visualMap?: Record<string, unknown>;
  series: Array<Record<string, unknown>>;
  // Métadonnées pour le développeur (pas consommées par ECharts)
  __meta?: {
    spec_id: string;
    chart_kind: ChartKind;
    source_function: string;
    warnings: string[];
    height: number;
  };
}
