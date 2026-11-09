/**
 * Tests composant — DataCard (sauvegarde périodique et plafond mémoire de l'application de bureau, lot L22).
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { dataStatusFixture } from '@/test/handlers'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { DataCard } from './DataCard'
import { SaveStatusBadge } from './SaveStatusBadge'
import { useSaveFeedbackStore } from './saveFeedbackStore'

const API = '/api/v1'
const SERVER_MESSAGE = 'MESSAGE DU SERVEUR : ne jamais afficher'

/** Corps des PATCH /settings reçus, dans l'ordre. */
let patches: Record<string, unknown>[] = []

function serveurPatch(refus?: { status: number; code: string }) {
  server.use(
    http.patch(`${API}/settings`, async ({ request }) => {
      const body = (await request.json()) as Record<string, unknown>
      patches.push(body)
      if (refus) {
        return HttpResponse.json(
          { status: refus.status, code: refus.code, message: SERVER_MESSAGE },
          { status: refus.status },
        )
      }
      return HttpResponse.json({ backup_dir: '', backup_interval: '24h', backup_keep: 7, memory_limit_mb: 0, ...body })
    }),
  )
}

function serveurEtat(etat: Record<string, unknown>) {
  server.use(http.get(`${API}/settings/data`, () => HttpResponse.json(etat)))
}

const DERNIERE = {
  name: 'levelup-2026-11-02T10-00-00Z',
  created_at: '2026-11-02T10:00:00Z',
  bytes: 5_300_000,
  databases: 3,
  duration_ms: 4_200,
  partial: false,
  app_version: '8.1.0',
}

describe('DataCard', () => {
  beforeEach(() => {
    patches = []
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    useSaveFeedbackStore.getState().clear()
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('montre la dernière sauvegarde : taille, nombre de bases et durée', async () => {
    serveurEtat({
      ...dataStatusFixture,
      backup: { enabled: true, running: false, same_volume: false, last: DERNIERE },
    })
    renderWithProviders(<DataCard />)
    const ligne = await screen.findByTestId('data-last-backup')
    expect(ligne).toHaveTextContent('5,3 Mo')
    expect(ligne).toHaveTextContent('3 bases')
    expect(ligne).toHaveTextContent('4 s')
  })

  it('sans sauvegarde, le dit', async () => {
    renderWithProviders(<DataCard />)
    expect(await screen.findByText('Aucune sauvegarde pour le moment.')).toBeInTheDocument()
  })

  it('pendant la lecture de l’état, ne dit pas « Aucune sauvegarde » : l’état n’est pas encore connu', async () => {
    server.use(http.get(`${API}/settings/data`, async () => {
      await new Promise((resolve) => setTimeout(resolve, 200))
      return HttpResponse.json(dataStatusFixture)
    }))
    renderWithProviders(<DataCard />)
    expect(await screen.findByText("Lecture de l'état…")).toBeInTheDocument()
    expect(screen.queryByText('Aucune sauvegarde pour le moment.')).not.toBeInTheDocument()
    expect(await screen.findByText('Aucune sauvegarde pour le moment.')).toBeInTheDocument()
  })

  it('quand l’état est indisponible, ne dit pas « Aucune sauvegarde » : on ne sait pas s’il en existe une', async () => {
    server.use(http.get(`${API}/settings/data`, () => HttpResponse.json({ status: 500, code: 'internal_error' }, { status: 500 })))
    renderWithProviders(<DataCard />)
    await waitFor(() => expect(screen.getByTestId('data-last-none')).toHaveTextContent(/indisponible/))
    expect(screen.queryByText('Aucune sauvegarde pour le moment.')).not.toBeInTheDocument()
  })

  it('une fréquence hors de la liste (48 h posée à la main) s’affiche telle quelle, pas « Tous les jours »', async () => {
    server.use(
      http.get(`${API}/settings`, () =>
        HttpResponse.json({ backup_dir: 'D:\\Sauvegardes', backup_interval: '48h', backup_keep: 7, memory_limit_mb: 0 }),
      ),
    )
    renderWithProviders(<DataCard />)
    const liste = (await screen.findByLabelText('Fréquence de la sauvegarde')) as HTMLSelectElement
    await waitFor(() => expect(liste.value).toBe('48h'))
    expect(screen.getByRole('option', { name: 'Toutes les 48 h' })).toBeInTheDocument()
    expect(liste.selectedOptions[0]).toHaveTextContent('Toutes les 48 h')
  })

  it('le plafond mémoire est dit « par base ouverte », avec son explication', async () => {
    renderWithProviders(<DataCard />)
    expect(await screen.findByLabelText('Plafond mémoire par base ouverte (Mo)')).toBeInTheDocument()
    expect(screen.getByText(/vaut pour CHAQUE base ouverte/)).toBeInTheDocument()
  })

  it('un jeu incomplet et un dernier passage en échec sont dits', async () => {
    serveurEtat({
      ...dataStatusFixture,
      backup: {
        enabled: true,
        running: false,
        same_volume: false,
        last: { ...DERNIERE, partial: true },
        last_attempt: { at: '2026-11-02T11:00:00Z', ok: false, error_code: 'nothing_exported' },
      },
    })
    renderWithProviders(<DataCard />)
    expect(await screen.findByText(/Jeu incomplet/)).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(/aucune base n'a pu être copiée/)
  })

  it('avertit (sans erreur) quand le dossier est sur le même disque que les données', async () => {
    serveurEtat({
      ...dataStatusFixture,
      backup: { enabled: true, running: false, same_volume: true },
    })
    renderWithProviders(<DataCard />)
    const avertissement = await screen.findByTestId('data-same-volume')
    expect(avertissement).toHaveTextContent(
      'Ce dossier est sur le même disque que vos données : une panne du disque emporterait les deux.',
    )
    expect(avertissement).toHaveAttribute('role', 'status')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it("pas d'avertissement de disque quand le dossier est sur un autre volume", async () => {
    serveurEtat({
      ...dataStatusFixture,
      backup: { enabled: true, running: false, same_volume: false },
    })
    renderWithProviders(<DataCard />)
    await screen.findByLabelText('Dossier de sauvegarde')
    await screen.findByText('Aucune sauvegarde pour le moment.')
    expect(screen.queryByTestId('data-same-volume')).not.toBeInTheDocument()
  })

  it('saisir un dossier puis quitter le champ envoie le PATCH et confirme par le badge de la page', async () => {
    serveurPatch()
    renderWithProviders(
      <>
        <DataCard />
        <SaveStatusBadge />
      </>,
    )
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    fireEvent.change(champ, { target: { value: '  D:\\Sauvegardes  ' } })
    fireEvent.blur(champ)
    await waitFor(() => expect(patches).toEqual([{ backup_dir: 'D:\\Sauvegardes' }]))
    expect(await screen.findByText('✓ Enregistré')).toBeInTheDocument()
    expect(screen.queryByText('Enregistré.')).not.toBeInTheDocument()
  })

  it('le bouton « Enregistrer » envoie le dossier ; inactif tant que rien ne change', async () => {
    serveurPatch()
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    const bouton = screen.getByRole('button', { name: 'Enregistrer' })
    expect(bouton).toBeDisabled()
    fireEvent.change(champ, { target: { value: 'D:\\Sauvegardes' } })
    expect(bouton).toBeEnabled()
    fireEvent.click(bouton)
    await waitFor(() => expect(patches).toEqual([{ backup_dir: 'D:\\Sauvegardes' }]))
  })

  it('quitter le champ sans rien changer ne renvoie rien', async () => {
    serveurPatch()
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    fireEvent.focus(champ)
    fireEvent.blur(champ)
    expect(patches).toEqual([])
  })

  it('un refus du serveur est écrit en français depuis son code, sous le champ, jamais par son message', async () => {
    serveurPatch({ status: 400, code: 'backup_dir_missing' })
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    fireEvent.change(champ, { target: { value: 'D:\\Nulle part' } })
    fireEvent.blur(champ)
    const alerte = await screen.findByRole('alert')
    expect(alerte).toHaveTextContent("Ce dossier n'existe pas.")
    expect(screen.queryByText(SERVER_MESSAGE)).not.toBeInTheDocument()
  })

  it("un refus du serveur montre aussi le badge d'échec de la page", async () => {
    serveurPatch({ status: 400, code: 'backup_dir_missing' })
    renderWithProviders(
      <>
        <DataCard />
        <SaveStatusBadge />
      </>,
    )
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    fireEvent.change(champ, { target: { value: 'D:\\Nulle part' } })
    fireEvent.blur(champ)
    expect(await screen.findByText('✗ Erreur lors de la sauvegarde')).toBeInTheDocument()
    expect(screen.getByText("Ce dossier n'existe pas.")).toBeInTheDocument()
  })

  it('le même refus est écrit en anglais quand la langue est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveurPatch({ status: 400, code: 'backup_dir_inside_data' })
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Backup folder')
    fireEvent.change(champ, { target: { value: 'C:\\data\\sub' } })
    fireEvent.blur(champ)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The backup folder cannot be inside the LevelUp data folder.',
    )
  })

  it('un code inconnu donne le texte générique, jamais le message du serveur', async () => {
    serveurPatch({ status: 500, code: 'code_inconnu' })
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    fireEvent.change(champ, { target: { value: 'D:\\Sauvegardes' } })
    fireEvent.blur(champ)
    expect(await screen.findByRole('alert')).toHaveTextContent("Le réglage n'a pas pu être enregistré.")
    expect(screen.queryByText(SERVER_MESSAGE)).not.toBeInTheDocument()
  })

  it('choisir une fréquence envoie le PATCH', async () => {
    serveurPatch()
    renderWithProviders(<DataCard />)
    const liste = await screen.findByLabelText('Fréquence de la sauvegarde')
    expect(liste).toHaveValue('24h')
    expect(screen.getAllByRole('option')).toHaveLength(6)
    fireEvent.change(liste, { target: { value: '6h' } })
    await waitFor(() => expect(patches).toEqual([{ backup_interval: '6h' }]))
  })

  it('le nombre de jeux conservés part en entier ; un refus est dit sous son champ', async () => {
    serveurPatch({ status: 400, code: 'backup_keep_invalid' })
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Jeux de sauvegarde conservés')
    expect(champ).toHaveValue(7)
    fireEvent.change(champ, { target: { value: '99' } })
    fireEvent.blur(champ)
    await waitFor(() => expect(patches).toEqual([{ backup_keep: 99 }]))
    expect(await screen.findByRole('alert')).toHaveTextContent('entre 1 et 60')
  })

  it("une saisie qui n'est pas un entier ne part pas : le refus du champ s'affiche tout de suite", async () => {
    serveurPatch()
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Plafond mémoire par base ouverte (Mo)')
    fireEvent.change(champ, { target: { value: '12,5' } })
    fireEvent.blur(champ)
    expect(await screen.findByRole('alert')).toHaveTextContent('0 (défaut)')
    expect(patches).toEqual([])
  })

  it('« Sauvegarder maintenant » est désactivé sans dossier de sauvegarde', async () => {
    renderWithProviders(<DataCard />)
    expect(await screen.findByRole('button', { name: 'Sauvegarder maintenant' })).toBeDisabled()
  })

  it('« Sauvegarder maintenant » lance la sauvegarde quand un dossier est choisi', async () => {
    serveurEtat({ ...dataStatusFixture, backup: { enabled: true, running: false, same_volume: false } })
    let lancees = 0
    server.use(
      http.post(`${API}/settings/data/backup/run`, () => {
        lancees++
        return HttpResponse.json({ started: true }, { status: 202 })
      }),
    )
    renderWithProviders(<DataCard />)
    const bouton = await screen.findByRole('button', { name: 'Sauvegarder maintenant' })
    await waitFor(() => expect(bouton).toBeEnabled())
    fireEvent.click(bouton)
    await waitFor(() => expect(lancees).toBe(1))
    expect(await screen.findByText('Sauvegarde lancée.')).toBeInTheDocument()
  })

  it('pendant une sauvegarde, le bouton est désactivé et le dit', async () => {
    serveurEtat({ ...dataStatusFixture, backup: { enabled: true, running: true, same_volume: false } })
    renderWithProviders(<DataCard />)
    const bouton = await screen.findByRole('button', { name: 'Sauvegarde en cours…' })
    expect(bouton).toBeDisabled()
  })

  it('un refus de lancement (409) est écrit depuis son code', async () => {
    serveurEtat({ ...dataStatusFixture, backup: { enabled: true, running: false, same_volume: false } })
    server.use(
      http.post(`${API}/settings/data/backup/run`, () =>
        HttpResponse.json(
          { status: 409, code: 'backup_already_running', message: SERVER_MESSAGE },
          { status: 409 },
        ),
      ),
    )
    renderWithProviders(<DataCard />)
    const bouton = await screen.findByRole('button', { name: 'Sauvegarder maintenant' })
    await waitFor(() => expect(bouton).toBeEnabled())
    fireEvent.click(bouton)
    expect(await screen.findByRole('alert')).toHaveTextContent('Une sauvegarde est déjà en cours.')
  })

  it('montre la mémoire actuelle et le plafond effectif', async () => {
    renderWithProviders(<DataCard />)
    expect(await screen.findByTestId('data-memory-current')).toHaveTextContent('Mémoire actuelle : 812 Mo')
    expect(screen.getByTestId('data-memory-effective')).toHaveTextContent('Plafond effectif par base : 4096MiB')
  })

  it('une mesure de mémoire indisponible (0) est dite', async () => {
    serveurEtat({ ...dataStatusFixture, memory: { rss_mb: 0, limit_mb: 0, effective_limit: '4096MiB' } })
    renderWithProviders(<DataCard />)
    expect(await screen.findByText('Mémoire actuelle : mesure indisponible.')).toBeInTheDocument()
  })

  it('régler le plafond mémoire envoie le PATCH en entier', async () => {
    serveurPatch()
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Plafond mémoire par base ouverte (Mo)')
    expect(champ).toHaveValue(0)
    fireEvent.change(champ, { target: { value: '2048' } })
    fireEvent.blur(champ)
    await waitFor(() => expect(patches).toEqual([{ memory_limit_mb: 2048 }]))
  })

  it('un plafond refusé par le serveur est écrit depuis son code, sous son champ', async () => {
    serveurPatch({ status: 400, code: 'memory_limit_invalid' })
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Plafond mémoire par base ouverte (Mo)')
    fireEvent.change(champ, { target: { value: '512' } })
    fireEvent.blur(champ)
    expect(await screen.findByRole('alert')).toHaveTextContent('entre 1 024 et 32 768 Mo')
  })

  it('sans la capacité de sauvegarde, la section de sauvegarde n’est pas offerte', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_limit_memory: true }) })
    renderWithProviders(<DataCard />)
    await screen.findByLabelText('Plafond mémoire par base ouverte (Mo)')
    expect(screen.queryByLabelText('Dossier de sauvegarde')).not.toBeInTheDocument()
  })

  it('sans la capacité de mémoire, le plafond n’est pas offert', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_backup_to_folder: true }) })
    renderWithProviders(<DataCard />)
    await screen.findByLabelText('Dossier de sauvegarde')
    expect(screen.queryByLabelText('Plafond mémoire par base ouverte (Mo)')).not.toBeInTheDocument()
  })

  it('sur le site (aucune de ces capacités), la carte n’existe pas', () => {
    useAppShellStore.setState({ capabilities: testCapabilities() })
    const { container } = renderWithProviders(<DataCard />)
    expect(container).toBeEmptyDOMElement()
  })

  it('« Parcourir… » enregistre le dossier choisi ; une annulation ne change rien', async () => {
    serveurPatch()
    const invoke = vi.fn().mockResolvedValueOnce(null).mockResolvedValueOnce('D:\\Choisi')
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
    renderWithProviders(<DataCard />)
    const champ = await screen.findByLabelText('Dossier de sauvegarde')
    const parcourir = screen.getByRole('button', { name: 'Parcourir…' })

    fireEvent.click(parcourir)
    await waitFor(() => expect(invoke).toHaveBeenCalledTimes(1))
    expect(invoke).toHaveBeenCalledWith('pick_folder', { initial: '' })
    expect(champ).toHaveValue('')
    expect(patches).toEqual([])

    await waitFor(() => expect(parcourir).toBeEnabled())
    fireEvent.click(parcourir)
    await waitFor(() => expect(patches).toEqual([{ backup_dir: 'D:\\Choisi' }]))
  })

  it('sans la capacité, pas de bouton « Parcourir… » : le champ reste une saisie', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ ...desktopCapabilities(), can_pick_folder: false }) })
    renderWithProviders(<DataCard />)
    await screen.findByLabelText('Dossier de sauvegarde')
    expect(screen.queryByRole('button', { name: 'Parcourir…' })).not.toBeInTheDocument()
  })

  it('parle anglais quand la langue est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<DataCard />)
    expect(await screen.findByLabelText('Backup folder')).toBeInTheDocument()
    expect(await screen.findByTestId('data-memory-current')).toHaveTextContent('Current memory: 812 MB')
    expect(screen.getByRole('button', { name: 'Back up now' })).toBeInTheDocument()
  })
})
