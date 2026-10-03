/**
 * Tests d'intégration GamertagCombobox.
 *
 * Vérifie les invariants UX :
 *  - Affiche le groupe « Joueurs configurés » avec étoile
 *  - Affiche « Coéquipiers fréquents » quand fournis
 *  - Affiche le message vide unifié quand recherche serveur retourne 0
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary, GamertagSearchResponse } from '@/lib/api/types'

import { GamertagCombobox } from './GamertagCombobox'

function asPlayer(gamertag: string): PlayerSummary {
  return {
    player_slug: gamertag.toLowerCase(),
    gamertag,
    xuid: `xuid-${gamertag}`,
  } as PlayerSummary
}

describe('GamertagCombobox', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      availablePlayers: [asPlayer('AlphaPlayer'), asPlayer('BravoGamer')],
    })
  })

  it('affiche le groupe "Joueurs configurés" au focus', () => {
    renderWithProviders(
      <GamertagCombobox selected={[]} onChange={() => {}} />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    expect(screen.getByText('Joueurs configurés')).toBeInTheDocument()
    expect(screen.getByText('AlphaPlayer')).toBeInTheDocument()
    expect(screen.getByText('BravoGamer')).toBeInTheDocument()
  })

  it('affiche le groupe "Coéquipiers fréquents" quand fournis', () => {
    renderWithProviders(
      <GamertagCombobox
        selected={[]}
        onChange={() => {}}
        frequentOptions={[
          { gamertag: 'CharlieX', xuid: 'xuid-CharlieX', encounter_count: 12, last_seen_at: undefined },
        ]}
      />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    expect(screen.getByText('Coéquipiers fréquents')).toBeInTheDocument()
    expect(screen.getByText('CharlieX')).toBeInTheDocument()
    expect(screen.getByText('12×')).toBeInTheDocument()
  })

  it('affiche le message vide unifié quand 0 résultat serveur', async () => {
    useAppShellStore.setState({ availablePlayers: [] })
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({ query: 'xq', items: [] }),
      ),
    )

    renderWithProviders(
      <GamertagCombobox selected={[]} onChange={() => {}} />,
    )
    const input = screen.getByPlaceholderText(/Rechercher un gamertag/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'xq' } })

    await waitFor(
      () => {
        expect(screen.getByText(/Aucun joueur trouvé pour "xq"/)).toBeInTheDocument()
      },
      { timeout: 2000 },
    )
  })

  it('affiche « Aucun résultat Xbox » quand le repli live n’a rien trouvé', async () => {
    // Contre-revue V72 : après « Rechercher sur Xbox » sans aucun hit, le bouton
    // disparaissait sans aucun retour — l'utilisateur croyait à un échec silencieux.
    useAppShellStore.setState({ availablePlayers: [] })
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({ query: 'GhostGT', items: [] }),
      ),
    )

    renderWithProviders(<GamertagCombobox selected={[]} onChange={() => {}} />)
    const input = screen.getByPlaceholderText(/Rechercher un gamertag/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'GhostGT' } })

    const cta = await screen.findByText('Rechercher sur Xbox', {}, { timeout: 2000 })
    fireEvent.click(cta)

    await waitFor(
      () => expect(screen.getByText('Aucun résultat Xbox')).toBeInTheDocument(),
      { timeout: 2000 },
    )
    // Un seul message « rien trouvé » (pas d'empilement avec le message local).
    expect(screen.queryByText(/Aucun joueur trouvé pour/)).not.toBeInTheDocument()
  })

  // ─── LA PROP `sources` — CE QU'ON NE PROPOSE PAS, ON NE L'INTERROGE PAS ──────────
  //
  // Le filtrage par source s'applique au RESULTAT ; sans desarmer la requete, chaque
  // frappe de deux caracteres partait quand meme vers l'annuaire pour une reponse jetee.

  it('sources=[frequent] : AUCUNE requete annuaire, la ou un combobox libre en emet une', async () => {
    // PREUVE D'UN NON-EVENEMENT : on monte les DEUX comboboxes cote a cote et on tape
    // dans chacun. L'arrivee de la requete du combobox LIBRE prouve que la fenetre de
    // debounce est passee pour les deux — sans ce temoin, « aucune requete » serait vrai
    // simplement parce qu'on a regarde trop tot.
    useAppShellStore.setState({ availablePlayers: [] })
    const vues: string[] = []
    server.use(
      http.get('*/directory/gamertags/search', ({ request }) => {
        vues.push(new URL(request.url).searchParams.get('q') ?? '')
        return HttpResponse.json<GamertagSearchResponse>({ query: 'x', items: [] })
      }),
    )

    renderWithProviders(
      <>
        <GamertagCombobox
          selected={[]}
          onChange={() => {}}
          placeholder="restreint"
          sources={['frequent']}
          frequentOptions={[
            { gamertag: 'CharlieX', xuid: 'xuid-CharlieX', encounter_count: 12, last_seen_at: undefined },
          ]}
        />
        <GamertagCombobox selected={[]} onChange={() => {}} placeholder="libre" />
      </>,
    )
    fireEvent.change(screen.getByPlaceholderText('restreint'), { target: { value: 'aaa' } })
    fireEvent.change(screen.getByPlaceholderText('libre'), { target: { value: 'bbb' } })

    await waitFor(() => expect(vues).toContain('bbb'), { timeout: 2000 })
    expect(vues).not.toContain('aaa')
    // …et la source distante n'est pas rendue non plus.
    expect(screen.queryByText('Autres joueurs')).toBeNull()
  })

  it('affiche les presets (escouades/groupes) et charge le roster au clic', () => {
    let loaded: string[] | null = null
    renderWithProviders(
      <GamertagCombobox
        selected={[]}
        onChange={() => {}}
        onLoadPreset={(gts) => {
          loaded = gts
        }}
        presetGroups={[
          {
            key: 'squads',
            label: 'Mes escouades',
            options: [
              { id: 'sq1', name: 'Ranked', subtitle: 'surtout Classé', gamertags: ['Choco', 'JGtm'] },
            ],
          },
        ]}
      />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    expect(screen.getByText('Mes escouades')).toBeInTheDocument()
    expect(screen.getByText('surtout Classé')).toBeInTheDocument()
    // Clic = charge le roster entier (≠ ajout d'un joueur).
    fireEvent.click(screen.getByText('Ranked'))
    expect(loaded).toEqual(['Choco', 'JGtm'])
  })

  it('affiche le footer de gestion fourni par le parent', () => {
    renderWithProviders(
      <GamertagCombobox
        selected={[]}
        onChange={() => {}}
        footer={<button type="button">Enregistrer la compo</button>}
      />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    expect(screen.getByText('Enregistrer la compo')).toBeInTheDocument()
  })

  it('appelle onClose à la fermeture du popover (clic extérieur)', () => {
    let closed = 0
    renderWithProviders(
      <GamertagCombobox
        selected={[]}
        onChange={() => {}}
        onClose={() => {
          closed += 1
        }}
      />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    fireEvent.mouseDown(document.body) // clic hors du combobox → ferme
    expect(closed).toBe(1)
  })

  it('respecte la limite max', () => {
    let selected = ['AlphaPlayer']
    const onChange = (v: string[]) => {
      selected = v
    }
    renderWithProviders(
      <GamertagCombobox selected={selected} onChange={onChange} max={1} />,
    )
    const input = screen.getByPlaceholderText('') as HTMLInputElement
    expect(input.disabled).toBe(true)
    expect(screen.getByText('1/1')).toBeInTheDocument()
  })

  it('Entrée ajoute le texte exact tapé quand il est hors suggestions', () => {
    let result: string[] | null = null
    renderWithProviders(
      <GamertagCombobox selected={[]} onChange={(v) => { result = v }} />,
    )
    const input = screen.getByPlaceholderText(/Rechercher un gamertag/i)
    fireEvent.focus(input)
    // 'UnknownGuy' n'est dans aucune suggestion (configurés Alpha/Bravo) → saisie libre.
    fireEvent.change(input, { target: { value: 'UnknownGuy' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(result).toEqual(['UnknownGuy'])
  })
})
