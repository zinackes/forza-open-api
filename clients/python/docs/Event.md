# Event

Événement / course (route fixe ou circuit).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**type** | [**EventType**](EventType.md) |  | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**start_lat** | **float** |  | [optional] 
**start_lng** | **float** |  | [optional] 
**end_lat** | **float** |  | [optional] 
**end_lng** | **float** |  | [optional] 
**route_geojson** | **Dict[str, object]** | Tracé GeoJSON (LineString) si disponible. | [optional] 
**car_class_restriction** | **str** |  | [optional] 
**length_m** | **int** |  | [optional] 

## Example

```python
from forza_open_api_client.models.event import Event

# TODO update the JSON string below
json = "{}"
# create an instance of Event from a JSON string
event_instance = Event.from_json(json)
# print the JSON string representation of the object
print(Event.to_json())

# convert the object into a dict
event_dict = event_instance.to_dict()
# create an instance of Event from a dict
event_from_dict = Event.from_dict(event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


