# Story

Story FH6 : mission narrative de Discover Japan, rapporte des stamps au Collection Journal. Champs non sourcés → omis. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**description** | **str** |  | [optional] 
**chapters_count** | **int** | Nombre de chapitres. Absent si non sourcé. | [optional] 
**reward_description** | **str** | Récompense de complétion (voiture | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.story import Story

# TODO update the JSON string below
json = "{}"
# create an instance of Story from a JSON string
story_instance = Story.from_json(json)
# print the JSON string representation of the object
print(Story.to_json())

# convert the object into a dict
story_dict = story_instance.to_dict()
# create an instance of Story from a dict
story_from_dict = Story.from_dict(story_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


