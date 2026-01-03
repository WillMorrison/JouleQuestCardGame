from enum import Enum


class PlayerActionType(str, Enum):
    BUILDASSET = "BuildAsset"
    FINISHED = "Finished"
    PLEDGECAPACITY = "PledgeCapacity"
    SCRAPASSET = "ScrapAsset"
    TAKEOVERASSET = "TakeoverAsset"
    TAKEOVERSCRAPASSET = "TakeoverScrapAsset"

    def __str__(self) -> str:
        return str(self.value)
