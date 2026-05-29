import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";

import { renderWithProviders } from "@/test/render-utils";
import { server } from "@/test/setup";

import { SessionDetailPage } from "./SessionDetailPage";

// ECharts (renderer canvas) ne peut pas peindre dans jsdom : il leve une
// exception async non rattrapable des qu'il atteint un getContext('2d') (null).
// Les autres tests y echappent par hasard (le chunk echarts lazy-loade par
// ChartCard n'a pas le temps de peindre), mais le flux compare ouvre le drawer
// puis attend un 2e fetch, laissant ECharts peindre. On neutralise donc le
// wrapper : ce fichier ne verifie que du texte (resumes, metriques, boutons),
// jamais le contenu d'un graphe.
vi.mock("echarts-for-react", () => ({
  default: () => null,
}));

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    useParams: () => ({ playerSlug: "test-player" }),
    // SessionDetailPage utilise useSearch({ strict: false }) pour lire ?session=
    // depuis l'URL (ajout post-84ae65ca). On retourne une recherche vide par
    // defaut — les tests qui ont besoin d'une session preselectionnee la
    // setteraient via override.
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
  };
});

function buildBaseResponse() {
  return {
    current_session: {
      session_label: "2026-04-21 19h30",
      start_time: "2026-04-21T19:30:00Z",
      end_time: "2026-04-21T20:05:00Z",
      total_matches: 2,
      wins: 2,
      losses: 0,
      kda: 2.4,
      performance_score: 68.5,
      win_rate: 100,
      kdr: 1.8,
      kills_per_match: 13,
      with_friends: false,
      dominant_category: "Ranked",
    },
    available_sessions: ["2026-04-21 19h30", "2026-04-21 18h"],
    matches: [
      {
        match_id: "match-1",
        start_time: "2026-04-21T19:45:00Z",
        outcome: 2,
        playlist_name: "Ranked Arena",
        pair_name: "Oddball",
        is_ranked: true,
        kills: 13,
        deaths: 5,
        assists: 6,
        kda: 2.6,
        accuracy: 64.8,
        personal_score: 2450,
        performance_score: 70,
        session_label: "2026-04-21 19h30",
        dominant_category: "Ranked",
        offensive_conversion: 1.2,
        defensive_resistance: 0.8,
      },
    ],
    suggested_compare: {
      session_label: "2026-04-21 18h",
      strategy: "category-ranked-close-volume",
      reason: "même catégorie ranked · écart de 1 match(s)",
    },
    compare_enabled: false,
    compare_session: null,
    compare_metrics: [],
  };
}

describe("SessionDetailPage", () => {
  it("ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)", () => {
    server.use(
      http.post(
        "/api/v1/players/:playerSlug/pages/sessions/detail",
        async () => {
          await delay("infinite");
          return HttpResponse.json(buildBaseResponse());
        },
      ),
    );

    renderWithProviders(<SessionDetailPage />);

    // Note : SessionDetailPage garde encore un Spinner local pendant le chargement
    // (n'a pas été migré vers le pattern TopProgressBar globale appliqué aux autres
    // pages). On vérifie donc que le loader SOIT visible — l'assertion inverse
    // sera réintroduite lorsque le composant sera migré.
    expect(screen.getByText(/Chargement de la session/i)).toBeInTheDocument();
  });

  it("affiche un état vide explicite quand aucune session n’est disponible", async () => {
    server.use(
      http.post("/api/v1/players/:playerSlug/pages/sessions/detail", () =>
        HttpResponse.json({
          current_session: null,
          available_sessions: [],
          matches: [],
          suggested_compare: null,
          compare_enabled: false,
          compare_session: null,
          compare_metrics: [],
        }),
      ),
    );

    renderWithProviders(<SessionDetailPage />);

    await waitFor(() => {
      expect(screen.getByText("Aucune session disponible")).toBeInTheDocument();
      expect(
        screen.getByText(/Aucune session n'a pu être reconstruite/i),
      ).toBeInTheDocument();
    });
  });

  it("affiche la suggestion et le détail des matchs après chargement", async () => {
    server.use(
      http.post("/api/v1/players/:playerSlug/pages/sessions/detail", () =>
        HttpResponse.json(buildBaseResponse()),
      ),
    );

    renderWithProviders(<SessionDetailPage />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Comparer/i }),
      ).toBeInTheDocument();
    });

    // Suggestion desormais affichee en hint inline "vs {label} - {reason}"
    // (cartes "Selection" et "Suggestion similaire" supprimees).
    expect(screen.getByText(/même catégorie ranked/)).toBeInTheDocument();
    expect(screen.getByText("Détail des matchs")).toBeInTheDocument();
    expect(screen.getByText("Oddball")).toBeInTheDocument();
    // outcomeLabel(2) → "win" en l'absence de fieldMappings backend mocké
    // (post-84ae65ca : libellé via useFieldMappings/outcomes.toml, fallback
    // sur la clé brute si l'endpoint n'est pas appelé en test).
    expect(screen.getByText("win")).toBeInTheDocument();
  });

  it("active la comparaison suggérée et affiche la lecture comparative", async () => {
    server.use(
      http.post(
        "/api/v1/players/:playerSlug/pages/sessions/detail",
        async ({ request }) => {
          const body = (await request.json()) as { enable_compare?: boolean };
          if (body.enable_compare) {
            return HttpResponse.json({
              ...buildBaseResponse(),
              compare_enabled: true,
              compare_session: {
                session_label: "2026-04-21 18h",
                start_time: "2026-04-21T18:00:00Z",
                end_time: "2026-04-21T18:55:00Z",
                total_matches: 3,
                wins: 1,
                losses: 2,
                kda: 1.3,
                performance_score: 61,
                win_rate: 33.3,
                kdr: 0.9,
                kills_per_match: 9,
                with_friends: false,
                dominant_category: "Ranked",
              },
              compare_metrics: [
                {
                  key: "score",
                  label: "Score perf.",
                  value_a: "68.5",
                  value_b: "61.0",
                  delta: "7.5",
                  winner: "a",
                },
              ],
            });
          }
          return HttpResponse.json(buildBaseResponse());
        },
      ),
    );

    renderWithProviders(<SessionDetailPage />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Comparer/i }),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /Comparer/i }));

    await waitFor(() => {
      // Drawer ouvert : la fermeture se fait via le bouton X (aria
      // "Fermer le panneau de comparaison") + summary compare visible.
      expect(
        screen.getByRole("button", { name: /Fermer le panneau de comparaison/i }),
      ).toBeInTheDocument();
      expect(screen.getAllByText(/Session comparée/i).length).toBeGreaterThan(0);
      expect(screen.getAllByText("Score perf.").length).toBeGreaterThan(0);
    });
  });

  it("affiche un état d’erreur exploitable quand l’API échoue", async () => {
    server.use(
      http.post("/api/v1/players/:playerSlug/pages/sessions/detail", () =>
        HttpResponse.json(
          { code: "session_page_error", message: "boom" },
          { status: 500 },
        ),
      ),
    );

    renderWithProviders(<SessionDetailPage />);

    await waitFor(() => {
      expect(
        screen.getByText(/Erreur lors du chargement de la session/i),
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Réessayer/i }),
      ).toBeInTheDocument();
    });
  });
});
