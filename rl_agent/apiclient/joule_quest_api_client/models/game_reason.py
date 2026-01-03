from enum import Enum


class GameReason(str, Enum):
    CARBONEMISSIONSEXCEEDED = "CarbonEmissionsExceeded"
    GRIDUNSTABLE = "GridUnstable"
    INSUFFICIENTGENERATION = "InsufficientGeneration"
    NOACTIVEPLAYERS = "NoActivePlayers"
    NONE = "None"
    UNOWNEDTAKEOVERASSETS = "UnownedTakeoverAssets"

    def __str__(self) -> str:
        return str(self.value)
