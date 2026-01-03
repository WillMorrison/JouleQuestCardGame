from enum import Enum


class PlayerActionAssetType(str, Enum):
    BATTERY = "Battery"
    FOSSIL = "Fossil"
    RENEWABLE = "Renewable"

    def __str__(self) -> str:
        return str(self.value)
