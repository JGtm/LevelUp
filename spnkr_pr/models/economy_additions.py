"""Additions to spnkr/models/economy.py — challenges, reward tracks, spartan points.

These classes follow the exact same conventions as the rest of the economy models:
- PascalCaseModel for all economy responses (the API uses PascalCase keys)
- frozen=True on every model
- tuple[X, ...] for arrays (immutable)
- Date alias for ISO8601Date datetime fields
"""

import datetime as dt

from pydantic import Field

# Re-using the base classes from SPNKr — in the actual PR these imports stay as-is.
from spnkr.models.base import PascalCaseModel


class Date(PascalCaseModel, frozen=True):
    """Datetime wrapper for ISO8601Date fields (mirrors economy.Date)."""

    value: dt.datetime = Field(alias="ISO8601Date")


# ─────────────────────────────────────────────────────────────────────────────
# Challenges
# ─────────────────────────────────────────────────────────────────────────────


class ChallengeReward(PascalCaseModel, frozen=True):
    """A single reward granted on challenge completion.

    Attributes:
        inventory_item_path: Relative path to the inventory item JSON on gamecms_hacs.
        amount: Quantity awarded.
    """

    inventory_item_path: str
    amount: int


class Challenge(PascalCaseModel, frozen=True):
    """An individual active challenge for a player.

    Attributes:
        tracking_id: Unique GUID identifying this challenge instance.
        xp_reward: Bonus XP awarded on completion.
        item_rewards: Cosmetic rewards granted on completion (may be empty).
        threshold: Target value to reach (kills, assists, matches, etc.).
        current_progress: Player's progress toward the threshold.
        is_completed: Whether the challenge has been completed.
        expiration_time: UTC expiry date (None if no expiry, e.g. career challenges).
        participation_challenge: Whether this counts as a participation/easy challenge.
    """

    tracking_id: str
    xp_reward: int
    item_rewards: tuple[ChallengeReward, ...] = Field(default=())
    threshold: int
    current_progress: int
    is_completed: bool
    expiration_time: Date | None = Field(default=None)
    participation_challenge: bool = Field(default=False)


class ChallengeCategory(PascalCaseModel, frozen=True):
    """A group of challenges sharing the same cadence (daily, weekly, etc.).

    Attributes:
        campaign_id: Identifier for the challenge campaign/deck.
        title: Human-readable category title.
        challenges: Ordered list of challenges in this category.
    """

    campaign_id: str
    title: str
    challenges: tuple[Challenge, ...]


class PlayerChallenges(PascalCaseModel, frozen=True):
    """All active challenges for a player.

    Attributes:
        challenge_deck_id: Identifier of the current challenge deck.
        categories: List of challenge categories (daily, weekly, event, etc.).
    """

    challenge_deck_id: str
    categories: tuple[ChallengeCategory, ...]


# ─────────────────────────────────────────────────────────────────────────────
# Reward Tracks (Battlepass / Operations / Events)
# ─────────────────────────────────────────────────────────────────────────────


class PlayerRewardTrackProgression(PascalCaseModel, frozen=True):
    """A player's progression on a single reward track.

    The meaning of ``current_progress`` depends on ``track_type``:
    - ``"Operations"`` / ``"Events"``: raw XP accumulated in the track.
    - ``"CareerRanks"``: integer rank (1–272).

    Attributes:
        reward_track_path: Relative path to the reward track definition JSON.
        track_type: Category of the track: "Operations", "Events", "CareerRanks".
        current_progress: Current progression value (XP or rank number).
        previous_progress: Progression value at last snapshot.
        is_owned: Whether the player has purchased the premium pass for this track.
        date_last_updated: When this progression was last updated server-side.
    """

    reward_track_path: str
    track_type: str
    current_progress: int
    previous_progress: int
    is_owned: bool
    date_last_updated: Date


class PlayerRewardTracksSummary(PascalCaseModel, frozen=True):
    """Summary of all reward track progressions for a player.

    Attributes:
        operation_reward_tracks: Progress on all operations (past and current).
        event_reward_tracks: Progress on all events (empty outside event windows).
        career_rank: Career rank progression. None if not yet initialized.
    """

    operation_reward_tracks: tuple[PlayerRewardTrackProgression, ...]
    event_reward_tracks: tuple[PlayerRewardTrackProgression, ...]
    career_rank: PlayerRewardTrackProgression | None = Field(default=None)


# ─────────────────────────────────────────────────────────────────────────────
# Spartan Points
# ─────────────────────────────────────────────────────────────────────────────


class SpartanPoints(PascalCaseModel, frozen=True):
    """A player's Spartan Points currency balance.

    Attributes:
        balance: Current spendable balance.
        total_earned: Lifetime points earned (None if not returned by the API).
    """

    balance: int
    total_earned: int | None = Field(default=None)
