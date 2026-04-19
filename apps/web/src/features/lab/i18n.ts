export type LabTab = 'resources' | 'contracts' | 'diagnostics'
export type LabLocale = 'fr' | 'en'

interface LabIntroCopy {
  eyebrow: string
  title: string
  description: string
  bullets: string[]
  footer: string
}

interface LabToolCopy {
  title: string
  whatItDoes: string
  interest: string
  capabilities: string[]
}

export interface LabText {
  intlLocale: string
  tabs: Record<LabTab, string>
  page: {
    title: string
    subtitle: string
    currentTitleBadge: string
    accessSubtitle: string
    accessDeniedTitle: string
    accessDeniedDescription: string
  }
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
  help: {
    intro: LabIntroCopy
    selectedToolEyebrow: string
    sections: {
      whatItDoes: string
      interest: string
      capabilities: string
    }
    tools: Record<LabTab, LabToolCopy>
  }
  resources: {
    loading: string
    unavailableTitle: string
    unavailableDescription: string
    currentTitle: string
    currentTitleHint: string
    activeSeason: string
    snapshotsHint: (count: number) => string
    localAssets: string
    localMedals: string
    metadataBaseTitle: string
    activeSeasonFallback: string
    snapshotsTitle: string
    snapshotsDescription: string
    noSnapshotsTitle: string
    noSnapshotsDescription: string
    snapshotPayloadTitle: string
    selectSnapshotTitle: string
    selectSnapshotDescription: string
    assetsTitle: string
    assetsDescription: string
    assetsPlaceholder: string
    noAssetsTitle: string
    noAssetsDescription: string
    selectAssetTitle: string
    selectAssetDescription: string
    rawAssetTitle: string
    medalsTitle: string
    medalsDescription: string
    medalsPlaceholder: string
    noMedalsTitle: string
    noMedalsDescription: string
    selectMedalTitle: string
    selectMedalDescription: string
    rawMedalTitle: string
  }
  contracts: {
    loading: string
    unavailableTitle: string
    unavailableDescription: string
    summaryStatus: string
    summaryHint: string
    fastapiRoutes: string
    goRoutes: string
    missingInGo: string
    methodMismatches: string
    goSpec: string
    fastapiReference: string
    missingRoutesTitle: string
    missingRoutesEmptyTitle: string
    missingRoutesEmptyDescription: string
    extraRoutesTitle: string
    extraRoutesEmptyTitle: string
    extraRoutesEmptyDescription: string
    mismatchTitle: string
    mismatchEmptyTitle: string
    mismatchEmptyDescription: string
    fastapiLabel: string
    goLabel: string
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
  tabs: {
    resources: 'Explorateur',
    contracts: 'Contrats API',
    diagnostics: 'Diagnostics',
  },
  page: {
    title: 'Lab interne',
    subtitle:
      'Explorer les métadonnées Waypoint locales, les contrats API et les diagnostics d\'instance sans sortir du shell React.',
    currentTitleBadge: 'Titre courant',
    accessSubtitle:
      'L\'accès à l\'instance lab est contrôlé par la capacité can_manage_instance.',
    accessDeniedTitle: 'Accès refusé',
    accessDeniedDescription:
      'Cette instance n\'autorise pas l\'exploration interne des ressources et diagnostics.',
  },
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
  help: {
    intro: {
      eyebrow: 'Guide rapide',
      title: 'Comment utiliser ce Lab',
      description:
        'Ce Lab permet d\'inspecter l\'état local de l\'instance sans quitter le shell React. Il sert à explorer les ressources locales, vérifier la parité de migration et lire des diagnostics techniques sans écrire en base.',
      bullets: [
        'Explorateur : inspecter les snapshots, assets et médailles locaux, puis lire leur payload JSON brut.',
        'Contrats API : comparer la spec Go avec la référence FastAPI pour repérer les routes manquantes, supplémentaires ou divergentes.',
        'Diagnostics : consulter le dernier rapport de parité disponible et les garde-fous locaux sur les médailles.',
      ],
      footer:
        'Lecture seule : aucune action ici ne déclenche de sync, de backfill ou d\'écriture en base.',
    },
    selectedToolEyebrow: 'Outil sélectionné',
    sections: {
      whatItDoes: 'Ce que fait l\'outil',
      interest: 'Intérêt',
      capabilities: 'Capacités',
    },
    tools: {
      resources: {
        title: 'Explorateur interne',
        whatItDoes:
          'Cet outil affiche les ressources Waypoint et metadata déjà présentes localement pour le titre courant, puis permet d\'ouvrir leur détail brut.',
        interest:
          'Il sert à comprendre ce qui est réellement stocké sur disque, à vérifier un cache local et à inspecter un payload sans passer par les scripts ou DuckDB à la main.',
        capabilities: [
          'Lister les snapshots, assets et médailles disponibles localement.',
          'Filtrer les assets et médailles par recherche texte.',
          'Afficher le JSON brut de la ressource sélectionnée.',
        ],
      },
      contracts: {
        title: 'Diff de contrats API',
        whatItDoes:
          'Cet outil compare la spec Go du repo avec la référence FastAPI gelée afin de montrer les écarts de surface HTTP.',
        interest:
          'Il permet de suivre la migration endpoint par endpoint, d\'identifier les trous de parité et de voir rapidement si un path existe, manque ou diverge entre les deux implémentations.',
        capabilities: [
          'Compter les routes FastAPI et Go détectées dans les fichiers OpenAPI.',
          'Lister les routes absentes côté Go ou supplémentaires côté Go.',
          'Afficher les divergences de méthodes HTTP pour un même path.',
        ],
      },
      diagnostics: {
        title: 'Diagnostics d\'instance',
        whatItDoes:
          'Cet outil regroupe le dernier rapport de parité disponible et les garde-fous exécutés sur les médailles Waypoint locales.',
        interest:
          'Il aide à juger rapidement l\'état technique de l\'instance : dernière comparaison de parité connue, anomalies de metadata et qualité minimale des données medals.',
        capabilities: [
          'Lire le fichier parity_report.json présent sur disque.',
          'Afficher le résumé des checks passés, échoués ou ignorés.',
          'Montrer les résultats des guards cardinalité, champs requis et images pour les médailles.',
        ],
      },
    },
  },
  resources: {
    loading: 'Chargement des ressources internes…',
    unavailableTitle: 'Explorateur indisponible',
    unavailableDescription:
      'Les métadonnées internes n\'ont pas pu être chargées.',
    currentTitle: 'Titre courant',
    currentTitleHint: 'Contexte title-aware appliqué à l\'API',
    activeSeason: 'Saison active',
    snapshotsHint: (count) => `Snapshots : ${count}`,
    localAssets: 'Assets locaux',
    localMedals: 'Médailles locales',
    metadataBaseTitle: 'Base metadata',
    activeSeasonFallback: 'N/A',
    snapshotsTitle: 'Snapshots Waypoint',
    snapshotsDescription:
      'Historique brut de waypoint_resource_snapshots pour le titre courant.',
    noSnapshotsTitle: 'Aucun snapshot archivé',
    noSnapshotsDescription:
      'La table waypoint_resource_snapshots est vide pour ce titre.',
    snapshotPayloadTitle: 'Payload snapshot',
    selectSnapshotTitle: 'Sélectionnez un snapshot',
    selectSnapshotDescription:
      'Choisissez une ressource pour afficher son payload JSON brut.',
    assetsTitle: 'Assets Waypoint',
    assetsDescription:
      'Recherche dans waypoint_assets_raw et inspection du JSON brut.',
    assetsPlaceholder: 'Filtrer par nom, type ou asset_id',
    noAssetsTitle: 'Aucun asset trouvé',
    noAssetsDescription:
      'Affinez la recherche ou alimentez le cache Waypoint local.',
    selectAssetTitle: 'Sélectionnez un asset',
    selectAssetDescription:
      'La colonne de droite affiche le JSON brut du cache local.',
    rawAssetTitle: 'JSON brut asset',
    medalsTitle: 'Médailles Waypoint',
    medalsDescription:
      'Recherche dans waypoint_medals_raw et inspection du JSON brut.',
    medalsPlaceholder:
      'Filtrer par medal_type, name_id ou description_id',
    noMedalsTitle: 'Aucune médaille trouvée',
    noMedalsDescription:
      'Affinez la recherche ou rechargez les métadonnées Waypoint.',
    selectMedalTitle: 'Sélectionnez une médaille',
    selectMedalDescription:
      'Le détail expose le JSON brut tel qu\'archivé en staging.',
    rawMedalTitle: 'JSON brut médaille',
  },
  contracts: {
    loading: 'Calcul du diff OpenAPI…',
    unavailableTitle: 'Contrats API indisponibles',
    unavailableDescription:
      'Le diff OpenAPI n\'a pas pu être calculé.',
    summaryStatus: 'Statut',
    summaryHint: 'Diff Go vs FastAPI',
    fastapiRoutes: 'Routes FastAPI',
    goRoutes: 'Routes Go',
    missingInGo: 'Manquantes côté Go',
    methodMismatches: 'Divergences HTTP',
    goSpec: 'Spec Go',
    fastapiReference: 'Référence FastAPI',
    missingRoutesTitle: 'Routes absentes côté Go',
    missingRoutesEmptyTitle: 'Aucune route manquante',
    missingRoutesEmptyDescription:
      'Tous les paths FastAPI attendus existent côté Go.',
    extraRoutesTitle: 'Routes supplémentaires côté Go',
    extraRoutesEmptyTitle: 'Aucun extra détecté',
    extraRoutesEmptyDescription:
      'La surface Go n\'introduit pas de route inattendue.',
    mismatchTitle: 'Divergences de méthodes',
    mismatchEmptyTitle: 'Aucune divergence HTTP',
    mismatchEmptyDescription:
      'Les méthodes déclarées côté Go et côté FastAPI sont alignées.',
    fastapiLabel: 'FastAPI',
    goLabel: 'Go',
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
  tabs: {
    resources: 'Explorer',
    contracts: 'API Contracts',
    diagnostics: 'Diagnostics',
  },
  page: {
    title: 'Internal Lab',
    subtitle:
      'Explore local Waypoint metadata, API contracts, and instance diagnostics without leaving the React shell.',
    currentTitleBadge: 'Current title',
    accessSubtitle:
      'Access to the instance lab is controlled by the can_manage_instance capability.',
    accessDeniedTitle: 'Access denied',
    accessDeniedDescription:
      'This instance does not allow internal exploration of resources and diagnostics.',
  },
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
  help: {
    intro: {
      eyebrow: 'Quick start',
      title: 'How to use this Lab',
      description:
        'This Lab lets you inspect the local state of the instance without leaving the React shell. Use it to explore local resources, verify migration parity, and read technical diagnostics without writing to the database.',
      bullets: [
        'Explorer: inspect local snapshots, assets, and medals, then read their raw JSON payloads.',
        'API Contracts: compare the Go spec with the FastAPI reference to spot missing, extra, or mismatched routes.',
        'Diagnostics: inspect the latest parity report and the local medal guards.',
      ],
      footer:
        'Read-only: no action here triggers sync, backfill, or database writes.',
    },
    selectedToolEyebrow: 'Selected tool',
    sections: {
      whatItDoes: 'What it does',
      interest: 'Why it matters',
      capabilities: 'Capabilities',
    },
    tools: {
      resources: {
        title: 'Internal Explorer',
        whatItDoes:
          'This tool exposes the Waypoint and metadata resources already stored locally for the current title, then lets you inspect their raw details.',
        interest:
          'It is useful to verify what is really cached on disk, inspect payloads quickly, and debug resource ingestion without opening DuckDB or running helper scripts manually.',
        capabilities: [
          'List locally available snapshots, assets, and medals.',
          'Filter assets and medals with text search.',
          'Display the raw JSON payload for the selected resource.',
        ],
      },
      contracts: {
        title: 'API Contract Diff',
        whatItDoes:
          'This tool compares the Go OpenAPI spec with the frozen FastAPI reference to highlight HTTP surface differences.',
        interest:
          'It helps track migration progress endpoint by endpoint, identify parity gaps, and quickly understand whether a route is missing, extra, or method-mismatched.',
        capabilities: [
          'Count the FastAPI and Go routes discovered in the OpenAPI files.',
          'List routes missing in Go or extra in Go.',
          'Show HTTP method mismatches for matching paths.',
        ],
      },
      diagnostics: {
        title: 'Instance Diagnostics',
        whatItDoes:
          'This tool combines the latest available parity report with the local guards executed on Waypoint medal metadata.',
        interest:
          'It gives a fast technical health snapshot of the instance: last known parity state, metadata anomalies, and the minimum quality checks applied to medal data.',
        capabilities: [
          'Read the parity_report.json artifact stored on disk.',
          'Summarise passed, failed, and skipped checks.',
          'Display cardinality, required-field, and image guard results for medals.',
        ],
      },
    },
  },
  resources: {
    loading: 'Loading internal resources…',
    unavailableTitle: 'Explorer unavailable',
    unavailableDescription: 'Internal metadata could not be loaded.',
    currentTitle: 'Current title',
    currentTitleHint: 'Title-aware API context applied',
    activeSeason: 'Active season',
    snapshotsHint: (count) => `Snapshots: ${count}`,
    localAssets: 'Local assets',
    localMedals: 'Local medals',
    metadataBaseTitle: 'Metadata database',
    activeSeasonFallback: 'N/A',
    snapshotsTitle: 'Waypoint snapshots',
    snapshotsDescription:
      'Raw history from waypoint_resource_snapshots for the current title.',
    noSnapshotsTitle: 'No archived snapshot',
    noSnapshotsDescription:
      'The waypoint_resource_snapshots table is empty for this title.',
    snapshotPayloadTitle: 'Snapshot payload',
    selectSnapshotTitle: 'Select a snapshot',
    selectSnapshotDescription:
      'Choose a resource to display its raw JSON payload.',
    assetsTitle: 'Waypoint assets',
    assetsDescription:
      'Search waypoint_assets_raw and inspect the raw JSON payload.',
    assetsPlaceholder: 'Filter by name, type, or asset_id',
    noAssetsTitle: 'No asset found',
    noAssetsDescription:
      'Refine the search or populate the local Waypoint cache.',
    selectAssetTitle: 'Select an asset',
    selectAssetDescription:
      'The right column displays the raw JSON from the local cache.',
    rawAssetTitle: 'Raw asset JSON',
    medalsTitle: 'Waypoint medals',
    medalsDescription:
      'Search waypoint_medals_raw and inspect the raw JSON payload.',
    medalsPlaceholder:
      'Filter by medal_type, name_id, or description_id',
    noMedalsTitle: 'No medal found',
    noMedalsDescription:
      'Refine the search or reload Waypoint metadata.',
    selectMedalTitle: 'Select a medal',
    selectMedalDescription:
      'The detail panel exposes the raw JSON archived in staging.',
    rawMedalTitle: 'Raw medal JSON',
  },
  contracts: {
    loading: 'Computing the OpenAPI diff…',
    unavailableTitle: 'API contracts unavailable',
    unavailableDescription: 'The OpenAPI diff could not be computed.',
    summaryStatus: 'Status',
    summaryHint: 'Go vs FastAPI diff',
    fastapiRoutes: 'FastAPI routes',
    goRoutes: 'Go routes',
    missingInGo: 'Missing in Go',
    methodMismatches: 'HTTP mismatches',
    goSpec: 'Go spec',
    fastapiReference: 'FastAPI reference',
    missingRoutesTitle: 'Routes missing in Go',
    missingRoutesEmptyTitle: 'No missing route',
    missingRoutesEmptyDescription:
      'All expected FastAPI paths exist on the Go side.',
    extraRoutesTitle: 'Extra Go routes',
    extraRoutesEmptyTitle: 'No extra route',
    extraRoutesEmptyDescription:
      'The Go surface does not introduce unexpected routes.',
    mismatchTitle: 'Method mismatches',
    mismatchEmptyTitle: 'No HTTP mismatch',
    mismatchEmptyDescription:
      'Methods declared in Go and FastAPI are aligned.',
    fastapiLabel: 'FastAPI',
    goLabel: 'Go',
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
