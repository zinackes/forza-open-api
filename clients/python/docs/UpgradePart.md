# UpgradePart

Pièce du catalogue d'upgrade (générique, indépendante d'une voiture). Les deltas sont relatifs au palier inférieur ; NULL si non sourcés (précision > exhaustivité). Sourcing progressif : voitures populaires d'abord, extension par séries ensuite. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**category** | [**UpgradePartCategory**](UpgradePartCategory.md) |  | 
**name** | **str** |  | 
**level** | **int** | Palier de la pièce (1 &#x3D; premier niveau). Omis si non applicable. | [optional] 
**pi_delta** | **int** | Variation de Performance Index apportée par la pièce. | [optional] 
**weight_delta_kg** | **int** | Variation de poids (kg). Négatif &#x3D; allègement. | [optional] 
**power_delta_hp** | **int** | Variation de puissance (ch). | [optional] 
**torque_delta_nm** | **int** | Variation de couple (Nm). | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.upgrade_part import UpgradePart

# TODO update the JSON string below
json = "{}"
# create an instance of UpgradePart from a JSON string
upgrade_part_instance = UpgradePart.from_json(json)
# print the JSON string representation of the object
print(UpgradePart.to_json())

# convert the object into a dict
upgrade_part_dict = upgrade_part_instance.to_dict()
# create an instance of UpgradePart from a dict
upgrade_part_from_dict = UpgradePart.from_dict(upgrade_part_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


