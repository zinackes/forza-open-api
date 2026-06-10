# Track

Tracé indexé (fonde les leaderboards et la carte).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**type** | [**TrackType**](TrackType.md) |  | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**length_m** | **int** | Longueur en mètres. | [optional] 
**surface_mix** | **str** | Répartition des surfaces (texte libre). | [optional] 
**start_lat** | **float** |  | [optional] 
**start_lng** | **float** |  | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 
**created_at** | **datetime** |  | [optional] 
**updated_at** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.track import Track

# TODO update the JSON string below
json = "{}"
# create an instance of Track from a JSON string
track_instance = Track.from_json(json)
# print the JSON string representation of the object
print(Track.to_json())

# convert the object into a dict
track_dict = track_instance.to_dict()
# create an instance of Track from a dict
track_from_dict = Track.from_dict(track_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


