# RefCount

Valeur d'une facette (code) et son nombre d'occurrences pour le jeu demandé. Pour les facettes énumérées (classes, drivetrains), tous les codes valides sont renvoyés, count compris à 0. Pour les facettes libres (bodyTypes, countries, categories), seules les valeurs présentes le sont. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **str** |  | 
**count** | **int** |  | 

## Example

```python
from forza_open_api_client.models.ref_count import RefCount

# TODO update the JSON string below
json = "{}"
# create an instance of RefCount from a JSON string
ref_count_instance = RefCount.from_json(json)
# print the JSON string representation of the object
print(RefCount.to_json())

# convert the object into a dict
ref_count_dict = ref_count_instance.to_dict()
# create an instance of RefCount from a dict
ref_count_from_dict = RefCount.from_dict(ref_count_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


