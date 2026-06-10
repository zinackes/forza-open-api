# ForzathonShopItem

Objet d'une rotation hebdomadaire du Forzathon Shop, achetable contre des Forza Points (qui se reportent d'une semaine à l'autre). Champs non sourcés → omis. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**week_start** | **datetime** | Début de la fenêtre de rotation (reset hebdo Forza | 
**week_end** | **datetime** | Fin de la fenêtre. Absente si non sourcée. | [optional] 
**kind** | [**ForzathonShopKind**](ForzathonShopKind.md) |  | 
**car_id** | **str** | Voiture liée (réf. /v1/cars). Présent uniquement pour les objets kind&#x3D;car identifiés au catalogue. | [optional] 
**name** | **str** |  | 
**fp_cost** | **int** | Coût en Forza Points. Absent si non sourcé. | [optional] 
**description** | **str** |  | [optional] 
**image_url** | **str** |  | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.forzathon_shop_item import ForzathonShopItem

# TODO update the JSON string below
json = "{}"
# create an instance of ForzathonShopItem from a JSON string
forzathon_shop_item_instance = ForzathonShopItem.from_json(json)
# print the JSON string representation of the object
print(ForzathonShopItem.to_json())

# convert the object into a dict
forzathon_shop_item_dict = forzathon_shop_item_instance.to_dict()
# create an instance of ForzathonShopItem from a dict
forzathon_shop_item_from_dict = ForzathonShopItem.from_dict(forzathon_shop_item_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


