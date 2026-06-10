# TreasureCar

Treasure Car FH6 : voiture associée à une postcard, conduisible immédiatement après la cutscene de lavage. Mécanique distincte des Barn Finds (pas de stamp ni de restauration). Coordonnées NULL si non sourcées (l'indice de la postcard suffit à localiser in-game). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**car_id** | **str** | Identifiant de la voiture obtenue (réf. /v1/cars). | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**postcard_clue** | **str** | Texte de l&#39;indice (postcard) menant à la voiture. | [optional] 
**location_lat** | **float** | Latitude de l&#39;emplacement si sourcée. | [optional] 
**location_lng** | **float** | Longitude de l&#39;emplacement si sourcée. | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.treasure_car import TreasureCar

# TODO update the JSON string below
json = "{}"
# create an instance of TreasureCar from a JSON string
treasure_car_instance = TreasureCar.from_json(json)
# print the JSON string representation of the object
print(TreasureCar.to_json())

# convert the object into a dict
treasure_car_dict = treasure_car_instance.to_dict()
# create an instance of TreasureCar from a dict
treasure_car_from_dict = TreasureCar.from_dict(treasure_car_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


