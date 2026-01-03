from enum import Enum


class PlayerStatus(str, Enum):
    ACTIVE = "Active"
    LOST = "Lost"

    def __str__(self) -> str:
        return str(self.value)
