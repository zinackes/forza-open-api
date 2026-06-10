# UpgradePartList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[UpgradePart]**](UpgradePart.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.upgrade_part_list import UpgradePartList

# TODO update the JSON string below
json = "{}"
# create an instance of UpgradePartList from a JSON string
upgrade_part_list_instance = UpgradePartList.from_json(json)
# print the JSON string representation of the object
print(UpgradePartList.to_json())

# convert the object into a dict
upgrade_part_list_dict = upgrade_part_list_instance.to_dict()
# create an instance of UpgradePartList from a dict
upgrade_part_list_from_dict = UpgradePartList.from_dict(upgrade_part_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


