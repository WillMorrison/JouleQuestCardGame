from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.game_reason import GameReason
from ..models.game_status import GameStatus

if TYPE_CHECKING:
    from ..models.asset_mix import AssetMix
    from ..models.game_last_round_snapshot import GameLastRoundSnapshot
    from ..models.player import Player


T = TypeVar("T", bound="Game")


@_attrs_define
class Game:
    """
    Attributes:
        status (GameStatus):
        reason (GameReason):  Default: GameReason.NONE.
        round_ (int):
        emissions_counter (int):
        last_round_snapshot (GameLastRoundSnapshot): Summary statistics from the last round of the game
        players (list[Player]):
        takeover_pool (AssetMix):
    """

    status: GameStatus
    round_: int
    emissions_counter: int
    last_round_snapshot: GameLastRoundSnapshot
    players: list[Player]
    takeover_pool: AssetMix
    reason: GameReason = GameReason.NONE

    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        reason = self.reason.value

        round_ = self.round_

        emissions_counter = self.emissions_counter

        last_round_snapshot = self.last_round_snapshot.to_dict()

        players = []
        for players_item_data in self.players:
            players_item = players_item_data.to_dict()
            players.append(players_item)

        takeover_pool = self.takeover_pool.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "Status": status,
                "Reason": reason,
                "Round": round_,
                "EmissionsCounter": emissions_counter,
                "LastRoundSnapshot": last_round_snapshot,
                "Players": players,
                "TakeoverPool": takeover_pool,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.asset_mix import AssetMix
        from ..models.game_last_round_snapshot import GameLastRoundSnapshot
        from ..models.player import Player

        d = dict(src_dict)
        status = GameStatus(d.pop("Status"))

        reason = GameReason(d.pop("Reason"))

        round_ = d.pop("Round")

        emissions_counter = d.pop("EmissionsCounter")

        last_round_snapshot = GameLastRoundSnapshot.from_dict(d.pop("LastRoundSnapshot"))

        players = []
        _players = d.pop("Players")
        for players_item_data in _players:
            players_item = Player.from_dict(players_item_data)

            players.append(players_item)

        takeover_pool = AssetMix.from_dict(d.pop("TakeoverPool"))

        game = cls(
            status=status,
            reason=reason,
            round_=round_,
            emissions_counter=emissions_counter,
            last_round_snapshot=last_round_snapshot,
            players=players,
            takeover_pool=takeover_pool,
        )

        return game
