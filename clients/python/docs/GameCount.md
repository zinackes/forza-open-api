# GameCount

Jeu disponible et ses volumes (voitures, séries de playlist).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | [**Game**](Game.md) |  | 
**count_cars** | **int** |  | 
**count_series** | **int** |  | 

## Example

```python
from forza_open_api_client.models.game_count import GameCount

# TODO update the JSON string below
json = "{}"
# create an instance of GameCount from a JSON string
game_count_instance = GameCount.from_json(json)
# print the JSON string representation of the object
print(GameCount.to_json())

# convert the object into a dict
game_count_dict = game_count_instance.to_dict()
# create an instance of GameCount from a dict
game_count_from_dict = GameCount.from_dict(game_count_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


