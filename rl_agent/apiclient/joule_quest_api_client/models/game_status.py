from enum import Enum


class GameStatus(str, Enum):
    LOSS = "Loss"
    ONGOING = "Ongoing"
    WIN = "Win"

    def __str__(self) -> str:
        return str(self.value)
