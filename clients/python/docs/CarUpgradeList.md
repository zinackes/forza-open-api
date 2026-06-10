# CarUpgradeList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[CarUpgrade]**](CarUpgrade.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.car_upgrade_list import CarUpgradeList

# TODO update the JSON string below
json = "{}"
# create an instance of CarUpgradeList from a JSON string
car_upgrade_list_instance = CarUpgradeList.from_json(json)
# print the JSON string representation of the object
print(CarUpgradeList.to_json())

# convert the object into a dict
car_upgrade_list_dict = car_upgrade_list_instance.to_dict()
# create an instance of CarUpgradeList from a dict
car_upgrade_list_from_dict = CarUpgradeList.from_dict(car_upgrade_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


