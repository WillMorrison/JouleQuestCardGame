from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.player_action_asset_type import PlayerActionAssetType
from ..models.player_action_type import PlayerActionType
from ..types import UNSET, Unset

T = TypeVar("T", bound="PlayerAction")


@_attrs_define
class PlayerAction:
    """
    Attributes:
        type_ (PlayerActionType):
        player_index (int):
        asset_type (PlayerActionAssetType | Unset):
        cost (int | Unset):
    """

    type_: PlayerActionType
    player_index: int
    asset_type: PlayerActionAssetType | Unset = UNSET
    cost: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        type_ = self.type_.value

        player_index = self.player_index

        asset_type: str | Unset = UNSET
        if not isinstance(self.asset_type, Unset):
            asset_type = self.asset_type.value

        cost = self.cost

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "Type": type_,
                "PlayerIndex": player_index,
            }
        )
        if asset_type is not UNSET:
            field_dict["AssetType"] = asset_type
        if cost is not UNSET:
            field_dict["Cost"] = cost

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        type_ = PlayerActionType(d.pop("Type"))

        player_index = d.pop("PlayerIndex")

        _asset_type = d.pop("AssetType", UNSET)
        asset_type: PlayerActionAssetType | Unset
        if isinstance(_asset_type, Unset):
            asset_type = UNSET
        else:
            asset_type = PlayerActionAssetType(_asset_type)

        cost = d.pop("Cost", UNSET)

        player_action = cls(
            type_=type_,
            player_index=player_index,
            asset_type=asset_type,
            cost=cost,
        )

        return player_action
