# Change

Changement du jeu de données, enregistré par les jobs d'ingestion (ex. voiture ajoutée par le seed du catalogue). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**resource** | **str** | Type de ressource (car | 
**resource_id** | **str** | Identifiant de la ressource changée. | 
**action** | [**ChangeAction**](ChangeAction.md) |  | 
**summary** | **str** | Description courte lisible (ex. nom de la voiture ajoutée). | [optional] 
**occurred_at** | **datetime** |  | 

## Example

```python
from forza_open_api_client.models.change import Change

# TODO update the JSON string below
json = "{}"
# create an instance of Change from a JSON string
change_instance = Change.from_json(json)
# print the JSON string representation of the object
print(Change.to_json())

# convert the object into a dict
change_dict = change_instance.to_dict()
# create an instance of Change from a dict
change_from_dict = Change.from_dict(change_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


