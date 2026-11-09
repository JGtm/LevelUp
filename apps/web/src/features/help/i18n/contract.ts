/**
 * contract.ts — Le TYPE du dictionnaire de l'aide : une seule déclaration, que `fr.ts`, `en.ts` et
 * toute langue ajoutée doivent satisfaire (`satisfies HelpText`).
 */

export type HelpTab = 'glossary' | 'release-notes'

export interface GlossaryEntry {
  term: string
  definition: string
  formula?: string
  example?: string
}

export interface GlossarySection {
  title: string
  entries: GlossaryEntry[]
}
export interface HelpText {
  tabs: {
    glossary: string
    releaseNotes: string
  }
  page: {
    title: string
    subtitle: string
  }
  releaseNotes: {
    loading: string
    error: string
    title: string
  }
  glossary: {
    title: string
    search: {
      placeholder: string
      sectionsLabel: string
      emptyTitle: string
      emptyDescription: string
    }
    sections: GlossarySection[]
  }
}
