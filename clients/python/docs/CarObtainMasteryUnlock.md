# CarObtainMasteryUnlock

Perk Car Mastery d'une autre voiture qui débloque cette voiture (effectType car_unlock). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**perk_id** | **str** | Identifiant de la perk (réf. /v1/cars/{id}/mastery). | 
**owner_car_id** | **str** | Voiture dont l&#39;arbre contient la perk (réf. /v1/cars). | 
**perk_name** | **str** |  | [optional] 

## Example

```python
from forza_open_api_client.models.car_obtain_mastery_unlock import CarObtainMasteryUnlock

# TODO update the JSON string below
json = "{}"
# create an instance of CarObtainMasteryUnlock from a JSON string
car_obtain_mastery_unlock_instance = CarObtainMasteryUnlock.from_json(json)
# print the JSON string representation of the object
print(CarObtainMasteryUnlock.to_json())

# convert the object into a dict
car_obtain_mastery_unlock_dict = car_obtain_mastery_unlock_instance.to_dict()
# create an instance of CarObtainMasteryUnlock from a dict
car_obtain_mastery_unlock_from_dict = CarObtainMasteryUnlock.from_dict(car_obtain_mastery_unlock_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


