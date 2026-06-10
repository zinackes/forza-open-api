# ForzathonShopList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[ForzathonShopItem]**](ForzathonShopItem.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.forzathon_shop_list import ForzathonShopList

# TODO update the JSON string below
json = "{}"
# create an instance of ForzathonShopList from a JSON string
forzathon_shop_list_instance = ForzathonShopList.from_json(json)
# print the JSON string representation of the object
print(ForzathonShopList.to_json())

# convert the object into a dict
forzathon_shop_list_dict = forzathon_shop_list_instance.to_dict()
# create an instance of ForzathonShopList from a dict
forzathon_shop_list_from_dict = ForzathonShopList.from_dict(forzathon_shop_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


