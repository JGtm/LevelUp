/**
 * i18n ex-Lab — réduit au périmètre encore en service (A3.5, DC-9) :
 * le panneau DiagnosticsPanel (rendu dans l'onglet admin Données) et ses
 * briques _labShared. Les sections tabs/page/help/resources/contracts sont
 * parties avec les panneaux supprimés.
 */
export type LabLocale = 'fr' | 'en'

export interface LabText {
  intlLocale: string
  common: {
    notAvailable: string
    retry: string
    readOnly: string
    size: string
    modified: string
    present: string
    absent: string
    filterPrefix: string
    version: string
    fetchedAt: string
    hash: string
    source: string
    asset: string
    type: string
    rawValue: string
    score: string
    id: string
    sprite: string
    status: string
    http: string
    mode: string
    defaultMode: string
    payloadUnavailableTitle: (title: string) => string
    payloadUnavailableDescription: string
    statuses: {
      ok: string
      ko: string
      passed: string
      failed: string
      skipped: string
      divergence: string
    }
  }
  diagnostics: {
    loading: string
    unavailableTitle: string
    unavailableDescription: string
    titleMetric: string
    titleMetricHint: string
    endpointsVerified: string
    passedCount: string
    failedCount: string
    skippedHint: (count: string) => string
    parityReportFile: string
    parityReportTitle: string
    parityReportMissingDescription: string
    reportAbsentTitle: string
    reportAbsentDescription: string
    medalGuardsTitle: string
    medalGuardsDescription: string
    entriesAnalyzed: string
    cardinality: string
    requiredFields: string
    images: string
    overallVerdict: string
    noGuardsTitle: string
    noGuardsDescription: string
  }
}

const FR_TEXT: LabText = {
  intlLocale: 'fr-FR',
  common: {
    notAvailable: 'N/A',
    retry: 'Réessayer',
    readOnly: 'Lecture seule',
    size: 'Taille',
    modified: 'Modifié',
    present: 'Présent',
    absent: 'Absent',
    filterPrefix: 'Filtre',
    version: 'Version',
    fetchedAt: 'Fetch',
    hash: 'Hash',
    source: 'Source',
    asset: 'Asset',
    type: 'Type',
    rawValue: 'brut',
    score: 'Score',
    id: 'ID',
    sprite: 'Sprite',
    status: 'Statut',
    http: 'HTTP',
    mode: 'Mode',
    defaultMode: 'complet',
    payloadUnavailableTitle: (title) => `${title} indisponible`,
    payloadUnavailableDescription:
      'Aucun payload brut n\'est disponible pour cette sélection.',
    statuses: {
      ok: 'OK',
      ko: 'KO',
      passed: 'Passé',
      failed: 'Échec',
      skipped: 'Ignoré',
      divergence: 'Divergence',
    },
  },
  diagnostics: {
    loading: 'Chargement des diagnostics…',
    unavailableTitle: 'Diagnostics indisponibles',
    unavailableDescription:
      'Le rapport de parité ou les garde-fous n\'ont pas pu être chargés.',
    titleMetric: 'Titre',
    titleMetricHint: 'Contexte courant du shell',
    endpointsVerified: 'Endpoints vérifiés',
    passedCount: 'Passés',
    failedCount: 'Échecs',
    skippedHint: (count) => `Ignorés : ${count}`,
    parityReportFile: 'Rapport parity_report.json',
    parityReportTitle: 'Rapport de parité',
    parityReportMissingDescription: 'Aucun rapport JSON n’est disponible.',
    reportAbsentTitle: 'Rapport absent',
    reportAbsentDescription:
      'Générez parity_report.json via le script parity_check.py pour alimenter ce panneau.',
    medalGuardsTitle: 'Guards médailles',
    medalGuardsDescription:
      'Validation locale de waypoint_medals_raw pour le titre courant.',
    entriesAnalyzed: 'Entrées analysées',
    cardinality: 'Cardinalité',
    requiredFields: 'Champs requis',
    images: 'Images',
    overallVerdict: 'Verdict global',
    noGuardsTitle: 'Aucun guard calculé',
    noGuardsDescription:
      'La table waypoint_medals_raw est vide ou indisponible pour ce titre.',
  },
}

const EN_TEXT: LabText = {
  intlLocale: 'en-GB',
  common: {
    notAvailable: 'N/A',
    retry: 'Retry',
    readOnly: 'Read-only',
    size: 'Size',
    modified: 'Modified',
    present: 'Present',
    absent: 'Missing',
    filterPrefix: 'Filter',
    version: 'Version',
    fetchedAt: 'Fetched',
    hash: 'Hash',
    source: 'Source',
    asset: 'Asset',
    type: 'Type',
    rawValue: 'raw',
    score: 'Score',
    id: 'ID',
    sprite: 'Sprite',
    status: 'Status',
    http: 'HTTP',
    mode: 'Mode',
    defaultMode: 'full',
    payloadUnavailableTitle: (title) => `${title} unavailable`,
    payloadUnavailableDescription:
      'No raw payload is available for this selection.',
    statuses: {
      ok: 'OK',
      ko: 'Failed',
      passed: 'Passed',
      failed: 'Failed',
      skipped: 'Skipped',
      divergence: 'Mismatch',
    },
  },
  diagnostics: {
    loading: 'Loading diagnostics…',
    unavailableTitle: 'Diagnostics unavailable',
    unavailableDescription:
      'The parity report or medal guards could not be loaded.',
    titleMetric: 'Title',
    titleMetricHint: 'Current shell context',
    endpointsVerified: 'Verified endpoints',
    passedCount: 'Passed',
    failedCount: 'Failed',
    skippedHint: (count) => `Skipped: ${count}`,
    parityReportFile: 'parity_report.json report',
    parityReportTitle: 'Parity report',
    parityReportMissingDescription: 'No JSON report is available.',
    reportAbsentTitle: 'Report missing',
    reportAbsentDescription:
      'Generate parity_report.json with the parity_check.py script to populate this panel.',
    medalGuardsTitle: 'Medal guards',
    medalGuardsDescription:
      'Local validation of waypoint_medals_raw for the current title.',
    entriesAnalyzed: 'Entries analysed',
    cardinality: 'Cardinality',
    requiredFields: 'Required fields',
    images: 'Images',
    overallVerdict: 'Overall verdict',
    noGuardsTitle: 'No guard result',
    noGuardsDescription:
      'The waypoint_medals_raw table is empty or unavailable for this title.',
  },
}

const LAB_TEXT: Record<LabLocale, LabText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizeLabLocale(locale?: string | null): LabLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getLabText(locale?: string | null): LabText {
  return LAB_TEXT[normalizeLabLocale(locale)]
}
