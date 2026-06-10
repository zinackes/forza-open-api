# Tour

Tour FH6 (Tours of Japan) : visite guidée de l'activité Discovery, rapporte des stamps au Collection Journal. Champs non sourcés → omis. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**description** | **str** |  | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.tour import Tour

# TODO update the JSON string below
json = "{}"
# create an instance of Tour from a JSON string
tour_instance = Tour.from_json(json)
# print the JSON string representation of the object
print(Tour.to_json())

# convert the object into a dict
tour_dict = tour_instance.to_dict()
# create an instance of Tour from a dict
tour_from_dict = Tour.from_dict(tour_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


