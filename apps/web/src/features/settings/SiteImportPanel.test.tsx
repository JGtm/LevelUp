/**
 * Tests composant — SiteImportPanel (import des données depuis le site, lot L22).
 */
import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { SiteImportPanel } from './SiteImportPanel'
import { getSiteImportText } from './siteImportText'

const API = '/api/v1'
const SERVER_MESSAGE = 'MESSAGE DU SERVEUR : ne jamais afficher'
const TOKEN = 'jeton-' + 'k3'.repeat(14)
const SITE = 'https://levelup.example'

interface Reçu {
  url: string
  body: Record<string, unknown>
}

/** Requêtes POST /import/site reçues, dans l'ordre. */
let reçus: Reçu[] = []

/** Sert GET /import/site ; `phase` peut changer entre deux relectures. */
let statut: Record<string, unknown> = { phase: 'idle', matches: 0 }

function serveurImport(refus?: { status: number; code: string }) {
  server.use(
    http.get(`${API}/import/site`, () => HttpResponse.json(statut)),
    http.post(`${API}/import/site`, async ({ request }) => {
      reçus.push({ url: request.url, body: (await request.json()) as Record<string, unknown> })
      if (refus) {
        return HttpResponse.json(
          { status: refus.status, code: refus.code, message: SERVER_MESSAGE },
          { status: refus.status },
        )
      }
      statut = { phase: 'downloading', matches: 0 }
      return HttpResponse.json(statut, { status: 202 })
    }),
  )
}

function remplir(url = SITE, token = TOKEN) {
  fireEvent.change(screen.getByLabelText('Adresse du site'), { target: { value: url } })
  fireEvent.change(screen.getByLabelText("Jeton d'export"), { target: { value: token } })
}

describe('SiteImportPanel', () => {
  beforeEach(() => {
    reçus = []
    statut = { phase: 'idle', matches: 0 }
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('sans la capacité d’import, rien n’est rendu', () => {
    useAppShellStore.setState({ capabilities: testCapabilities() })
    const { container } = renderWithProviders(<SiteImportPanel />)
    expect(container).toBeEmptyDOMElement()
  })

  it('le jeton est un champ secret, sans mémorisation du navigateur', async () => {
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    const jeton = await screen.findByLabelText("Jeton d'export")
    expect(jeton).toHaveAttribute('type', 'password')
    expect(jeton).toHaveAttribute('autocomplete', 'off')
  })

  it('« Importer » est inactif tant que l’adresse et le jeton ne sont pas saisis', async () => {
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    const bouton = await screen.findByRole('button', { name: 'Importer' })
    expect(bouton).toBeDisabled()
    fireEvent.change(screen.getByLabelText('Adresse du site'), { target: { value: SITE } })
    expect(bouton).toBeDisabled()
    fireEvent.change(screen.getByLabelText("Jeton d'export"), { target: { value: TOKEN } })
    expect(bouton).toBeEnabled()
  })

  it('démarre l’import : le jeton part dans le corps seulement, puis le champ est vidé', async () => {
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    await screen.findByLabelText('Adresse du site')
    remplir(`  ${SITE}  `, `  ${TOKEN}  `)
    fireEvent.click(screen.getByRole('button', { name: 'Importer' }))

    await waitFor(() => expect(reçus).toHaveLength(1))
    expect(reçus[0].body).toEqual({ url: SITE, token: TOKEN })
    // Jamais dans l'URL, ni dans un stockage du navigateur.
    expect(reçus[0].url).not.toContain(TOKEN)
    expect(JSON.stringify({ ...localStorage })).not.toContain(TOKEN)
    expect(JSON.stringify({ ...sessionStorage })).not.toContain(TOKEN)
    await waitFor(() => expect(screen.getByLabelText("Jeton d'export")).toHaveValue(''))
    expect(await screen.findByText("Téléchargement de l'archive…")).toBeInTheDocument()
  })

  it('montre l’écriture des données quand l’import est à cette phase', async () => {
    statut = { phase: 'importing', matches: 0 }
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    expect(await screen.findByText('Écriture des données…')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Importer' })).toBeDisabled()
  })

  it('suit la progression toute seule jusqu’au bilan', async () => {
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    await screen.findByLabelText('Adresse du site')
    remplir()
    fireEvent.click(screen.getByRole('button', { name: 'Importer' }))
    expect(await screen.findByText("Téléchargement de l'archive…")).toBeInTheDocument()

    statut = { phase: 'done', matches: 42, inserted: { matches: 42 } }
    expect(await screen.findByText('42 matchs ajoutés.', {}, { timeout: 6000 })).toBeInTheDocument()
  }, 10_000)

  it('dit le bilan d’un import terminé, au singulier pour un match', async () => {
    statut = { phase: 'done', matches: 1 }
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    expect(await screen.findByText('1 match ajouté.')).toBeInTheDocument()
  })

  it.each([
    ['site_token_invalid', /Le jeton est refusé/],
    ['archive_other_player', /autre joueur/],
    ['site_archive_too_big', /trop volumineuse/],
    ['import_failed', /L'import a échoué/],
  ])('un échec de l’import (%s) est écrit depuis son code', async (code, texte) => {
    statut = { phase: 'failed', error_code: code, matches: 0 }
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    expect(await screen.findByRole('alert')).toHaveTextContent(texte)
  })

  it.each([
    [409, 'import_already_running', /déjà en cours/],
    [409, 'import_no_tracked_player', /Aucun joueur n'est suivi/],
    [409, 'import_player_without_xuid', /pas encore relié/],
  ])('un refus du démarrage (%i %s) est écrit depuis son code, jamais par le message du serveur', async (status, code, texte) => {
    serveurImport({ status, code })
    renderWithProviders(<SiteImportPanel />)
    await screen.findByLabelText('Adresse du site')
    remplir()
    fireEvent.click(screen.getByRole('button', { name: 'Importer' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(texte)
    expect(screen.queryByText(SERVER_MESSAGE)).not.toBeInTheDocument()
    // Le jeton est vidé même quand le démarrage est refusé : il est à usage unique.
    expect(screen.getByLabelText("Jeton d'export")).toHaveValue('')
  })

  it('plusieurs joueurs suivis : demande le gamertag, puis le joint à la requête', async () => {
    serveurImport({ status: 400, code: 'import_player_ambiguous' })
    renderWithProviders(<SiteImportPanel />)
    await screen.findByLabelText('Adresse du site')
    expect(screen.queryByLabelText('Joueur suivi (gamertag)')).not.toBeInTheDocument()
    remplir()
    fireEvent.click(screen.getByRole('button', { name: 'Importer' }))
    const gamertag = await screen.findByLabelText('Joueur suivi (gamertag)')
    expect(screen.getByRole('alert')).toHaveTextContent(/Plusieurs joueurs sont suivis/)

    serveurImport()
    fireEvent.change(gamertag, { target: { value: 'JGtm' } })
    fireEvent.change(screen.getByLabelText("Jeton d'export"), { target: { value: TOKEN } })
    fireEvent.click(screen.getByRole('button', { name: 'Importer' }))
    await waitFor(() => expect(reçus).toHaveLength(2))
    expect(reçus[1].body).toEqual({ url: SITE, token: TOKEN, gamertag: 'JGtm' })
  })

  it('variante facultative : repliée au départ, on peut l’ignorer, un clic la déplie', async () => {
    serveurImport()
    renderWithProviders(<SiteImportPanel variant="optional" />)
    const titre = await screen.findByRole('button', {
      name: 'Vous avez déjà un compte sur le site ? Importer vos données',
    })
    expect(titre).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByLabelText('Adresse du site')).not.toBeInTheDocument()
    fireEvent.click(titre)
    expect(titre).toHaveAttribute('aria-expanded', 'true')
    expect(await screen.findByLabelText('Adresse du site')).toBeInTheDocument()
  })

  it('parle anglais quand la langue est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    statut = { phase: 'done', matches: 3 }
    serveurImport()
    renderWithProviders(<SiteImportPanel />)
    expect(await screen.findByLabelText('Site address')).toBeInTheDocument()
    expect(await screen.findByText('3 matches added.')).toBeInTheDocument()
  })

  it.each(['fr', 'en'] as const)('chaque code d’échec de l’import a son texte en %s', (locale) => {
    const t = getSiteImportText(locale)
    for (const code of [
      'site_url_invalid',
      'site_token_invalid',
      'site_rate_limited',
      'site_refused',
      'site_archive_too_big',
      'archive_invalid',
      'archive_other_player',
      'archive_unknown_table',
      'archive_format_unsupported',
      'import_failed',
      'import_already_running',
      'import_no_tracked_player',
      'import_player_without_xuid',
      'import_player_ambiguous',
    ]) {
      expect(t.errors[code], `${locale}/${code}`).toBeTruthy()
    }
  })
})
