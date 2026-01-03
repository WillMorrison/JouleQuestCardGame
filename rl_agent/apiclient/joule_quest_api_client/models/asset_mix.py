from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="AssetMix")


@_attrs_define
class AssetMix:
    """
    Attributes:
        renewables (int):  Default: 0.
        batteries_arbitrage (int):  Default: 0.
        batteries_capacity (int):  Default: 0.
        fossils_wholesale (int):  Default: 0.
        fossils_capacity (int):  Default: 0.
    """

    renewables: int = 0
    batteries_arbitrage: int = 0
    batteries_capacity: int = 0
    fossils_wholesale: int = 0
    fossils_capacity: int = 0

    def to_dict(self) -> dict[str, Any]:
        renewables = self.renewables

        batteries_arbitrage = self.batteries_arbitrage

        batteries_capacity = self.batteries_capacity

        fossils_wholesale = self.fossils_wholesale

        fossils_capacity = self.fossils_capacity

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "Renewables": renewables,
                "BatteriesArbitrage": batteries_arbitrage,
                "BatteriesCapacity": batteries_capacity,
                "FossilsWholesale": fossils_wholesale,
                "FossilsCapacity": fossils_capacity,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        renewables = d.pop("Renewables")

        batteries_arbitrage = d.pop("BatteriesArbitrage")

        batteries_capacity = d.pop("BatteriesCapacity")

        fossils_wholesale = d.pop("FossilsWholesale")

        fossils_capacity = d.pop("FossilsCapacity")

        asset_mix = cls(
            renewables=renewables,
            batteries_arbitrage=batteries_arbitrage,
            batteries_capacity=batteries_capacity,
            fossils_wholesale=fossils_wholesale,
            fossils_capacity=fossils_capacity,
        )

        return asset_mix
