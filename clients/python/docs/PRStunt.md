# PRStunt

PR Stunt (speed trap, speed zone, drift zone, danger sign).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**type** | [**PRStuntType**](PRStuntType.md) |  | 
**name** | **str** |  | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**lat** | **float** |  | [optional] 
**lng** | **float** |  | [optional] 
**target_score** | **int** | Score cible (3 étoiles) si connu. | [optional] 

## Example

```python
from forza_open_api_client.models.pr_stunt import PRStunt

# TODO update the JSON string below
json = "{}"
# create an instance of PRStunt from a JSON string
pr_stunt_instance = PRStunt.from_json(json)
# print the JSON string representation of the object
print(PRStunt.to_json())

# convert the object into a dict
pr_stunt_dict = pr_stunt_instance.to_dict()
# create an instance of PRStunt from a dict
pr_stunt_from_dict = PRStunt.from_dict(pr_stunt_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


