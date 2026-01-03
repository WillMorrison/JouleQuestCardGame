from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

if TYPE_CHECKING:
    from ..models.asset_mix import AssetMix


T = TypeVar("T", bound="GameLastRoundSnapshot")


@_attrs_define
class GameLastRoundSnapshot:
    """Summary statistics from the last round of the game

    Attributes:
        asset_mix (AssetMix):
        price_volatility (int):
        grid_stability (int):
    """

    asset_mix: AssetMix
    price_volatility: int
    grid_stability: int

    def to_dict(self) -> dict[str, Any]:
        asset_mix = self.asset_mix.to_dict()

        price_volatility = self.price_volatility

        grid_stability = self.grid_stability

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "AssetMix": asset_mix,
                "PriceVolatility": price_volatility,
                "GridStability": grid_stability,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.asset_mix import AssetMix

        d = dict(src_dict)
        asset_mix = AssetMix.from_dict(d.pop("AssetMix"))

        price_volatility = d.pop("PriceVolatility")

        grid_stability = d.pop("GridStability")

        game_last_round_snapshot = cls(
            asset_mix=asset_mix,
            price_volatility=price_volatility,
            grid_stability=grid_stability,
        )

        return game_last_round_snapshot
