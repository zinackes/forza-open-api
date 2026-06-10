# MetaGame

Volumes et fraîcheur des données d'un jeu supporté. carCount = voitures au catalogue (0 si rien n'est encore ingéré). catalogUpdatedAt / playlistUpdatedAt = dernier timestamp d'ingestion (journal data_changes) pour les ressources car / series ; absents si jamais ingéré pour ce jeu. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**game** | [**Game**](Game.md) |  | 
**car_count** | **int** |  | 
**catalog_updated_at** | **datetime** | Dernière ingestion du catalogue (voitures) pour ce jeu. Absent si jamais ingéré. | [optional] 
**playlist_updated_at** | **datetime** | Dernière ingestion de la playlist (séries) pour ce jeu. Absent si jamais ingérée. | [optional] 

## Example

```python
from forza_open_api_client.models.meta_game import MetaGame

# TODO update the JSON string below
json = "{}"
# create an instance of MetaGame from a JSON string
meta_game_instance = MetaGame.from_json(json)
# print the JSON string representation of the object
print(MetaGame.to_json())

# convert the object into a dict
meta_game_dict = meta_game_instance.to_dict()
# create an instance of MetaGame from a dict
meta_game_from_dict = MetaGame.from_dict(meta_game_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


