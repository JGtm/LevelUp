import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";

import { renderWithProviders } from "@/test/render-utils";
import { server } from "@/test/setup";

import { SessionDetailPage } from "./SessionDetailPage";

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    useParams: () => ({ playerSlug: "test-player" }),
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
  it("affiche le spinner pendant le chargement initial", () => {
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
      expect(screen.getByText("Suggestion similaire")).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Comparer à la session proche/i }),
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("même catégorie ranked · écart de 1 match(s)"),
    ).toBeInTheDocument();
    expect(screen.getByText("Détail des matchs")).toBeInTheDocument();
    expect(screen.getByText("Oddball")).toBeInTheDocument();
    expect(screen.getByText("Victoire")).toBeInTheDocument();
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
        screen.getByRole("button", { name: /Comparer à la session proche/i }),
      ).toBeInTheDocument();
    });

    fireEvent.click(
      screen.getByRole("button", { name: /Comparer à la session proche/i }),
    );

    await waitFor(() => {
      expect(screen.getByText("Lecture comparative")).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Masquer comparaison/i }),
      ).toBeInTheDocument();
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
