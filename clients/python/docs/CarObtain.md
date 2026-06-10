# CarObtain

Vue agrégée « comment obtenir cette voiture » : méthode du catalogue, prix, packs DLC, Barn Find, Treasure Car, paliers Journal et perks Car Mastery qui la débloquent. Listes vides / champs omis si la voie n'existe pas ou n'est pas sourcée (rien d'inventé). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**car_id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**obtain_method** | **str** | Méthode d&#39;obtention du catalogue (ex. autoshow | [optional] 
**value_cr** | **int** | Prix autoshow en crédits. Absent si non sourcé. | [optional] 
**dlc_packs** | [**List[DlcPack]**](DlcPack.md) | Packs DLC contenant la voiture (vide si aucun). | 
**barn_find** | [**BarnFind**](BarnFind.md) | Barn Find donnant la voiture. Absent si la voiture n&#39;en est pas un. | [optional] 
**treasure_car** | [**TreasureCar**](TreasureCar.md) | Treasure Car donnant la voiture. Absente si la voiture n&#39;en est pas une. | [optional] 
**journal_tiers** | [**List[JournalTier]**](JournalTier.md) | Paliers du Collection Journal qui récompensent la voiture (vide si aucun). | 
**mastery_unlocks** | [**List[CarObtainMasteryUnlock]**](CarObtainMasteryUnlock.md) | Perks Car Mastery (car_unlock) qui débloquent la voiture (vide si aucune). | 

## Example

```python
from forza_open_api_client.models.car_obtain import CarObtain

# TODO update the JSON string below
json = "{}"
# create an instance of CarObtain from a JSON string
car_obtain_instance = CarObtain.from_json(json)
# print the JSON string representation of the object
print(CarObtain.to_json())

# convert the object into a dict
car_obtain_dict = car_obtain_instance.to_dict()
# create an instance of CarObtain from a dict
car_obtain_from_dict = CarObtain.from_dict(car_obtain_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


