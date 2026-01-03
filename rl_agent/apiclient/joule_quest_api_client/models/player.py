from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.player_status import PlayerStatus

if TYPE_CHECKING:
    from ..models.asset_mix import AssetMix


T = TypeVar("T", bound="Player")


@_attrs_define
class Player:
    """
    Attributes:
        status (PlayerStatus):
        money (int):
        assets (AssetMix):
    """

    status: PlayerStatus
    money: int
    assets: AssetMix

    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        money = self.money

        assets = self.assets.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "Status": status,
                "Money": money,
                "Assets": assets,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.asset_mix import AssetMix

        d = dict(src_dict)
        status = PlayerStatus(d.pop("Status"))

        money = d.pop("Money")

        assets = AssetMix.from_dict(d.pop("Assets"))

        player = cls(
            status=status,
            money=money,
            assets=assets,
        )

        return player
