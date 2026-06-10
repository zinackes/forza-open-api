# CarUpgrade

Upgrade disponible pour une voiture : une pièce du catalogue assortie de ses contraintes d'installation (prérequis, groupe exclusif). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**part** | [**UpgradePart**](UpgradePart.md) |  | 
**requires_part_id** | **str** | Pièce prérequise (réf. upgrade-parts) à monter avant celle-ci. Absent si aucune. | [optional] 
**exclusive_group** | **str** | Groupe exclusif : une seule pièce d&#39;un même groupe peut être montée à la fois (ex. compounds de pneus). Absent si non concerné.  | [optional] 

## Example

```python
from forza_open_api_client.models.car_upgrade import CarUpgrade

# TODO update the JSON string below
json = "{}"
# create an instance of CarUpgrade from a JSON string
car_upgrade_instance = CarUpgrade.from_json(json)
# print the JSON string representation of the object
print(CarUpgrade.to_json())

# convert the object into a dict
car_upgrade_dict = car_upgrade_instance.to_dict()
# create an instance of CarUpgrade from a dict
car_upgrade_from_dict = CarUpgrade.from_dict(car_upgrade_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


