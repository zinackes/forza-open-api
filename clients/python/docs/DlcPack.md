# DlcPack

Pack DLC / extension (Car Pass, expansion, standalone).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**kind** | [**DlcKind**](DlcKind.md) |  | 
**released_at** | **datetime** | Date de sortie. Absente si pack annoncé mais pas encore sorti. | [optional] 
**description** | **str** |  | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.dlc_pack import DlcPack

# TODO update the JSON string below
json = "{}"
# create an instance of DlcPack from a JSON string
dlc_pack_instance = DlcPack.from_json(json)
# print the JSON string representation of the object
print(DlcPack.to_json())

# convert the object into a dict
dlc_pack_dict = dlc_pack_instance.to_dict()
# create an instance of DlcPack from a dict
dlc_pack_from_dict = DlcPack.from_dict(dlc_pack_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


