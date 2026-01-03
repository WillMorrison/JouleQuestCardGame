from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.game import Game
    from ..models.player_action import PlayerAction


T = TypeVar("T", bound="GameUpdate")


@_attrs_define
class GameUpdate:
    """
    Attributes:
        id (str): Internal game ID
        possible_actions (list[PlayerAction] | None): The set of actions that may be sent in the next request to
            /g/{GameID}/action
        game (Game):
    """

    id: str
    possible_actions: list[PlayerAction] | None
    game: Game
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        possible_actions: list[dict[str, Any]] | None
        if isinstance(self.possible_actions, list):
            possible_actions = []
            for possible_actions_type_0_item_data in self.possible_actions:
                possible_actions_type_0_item = possible_actions_type_0_item_data.to_dict()
                possible_actions.append(possible_actions_type_0_item)

        else:
            possible_actions = self.possible_actions

        game = self.game.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "ID": id,
                "PossibleActions": possible_actions,
                "Game": game,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.game import Game
        from ..models.player_action import PlayerAction

        d = dict(src_dict)
        id = d.pop("ID")

        def _parse_possible_actions(data: object) -> list[PlayerAction] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                possible_actions_type_0 = []
                _possible_actions_type_0 = data
                for possible_actions_type_0_item_data in _possible_actions_type_0:
                    possible_actions_type_0_item = PlayerAction.from_dict(possible_actions_type_0_item_data)

                    possible_actions_type_0.append(possible_actions_type_0_item)

                return possible_actions_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[PlayerAction] | None, data)

        possible_actions = _parse_possible_actions(d.pop("PossibleActions"))

        game = Game.from_dict(d.pop("Game"))

        game_update = cls(
            id=id,
            possible_actions=possible_actions,
            game=game,
        )

        game_update.additional_properties = d
        return game_update

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
