# Reward


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**at_percent** | **int** |  | [optional] 
**type** | **str** |  | 
**item** | **str** |  | 

## Example

```python
from forza_open_api_client.models.reward import Reward

# TODO update the JSON string below
json = "{}"
# create an instance of Reward from a JSON string
reward_instance = Reward.from_json(json)
# print the JSON string representation of the object
print(Reward.to_json())

# convert the object into a dict
reward_dict = reward_instance.to_dict()
# create an instance of Reward from a dict
reward_from_dict = Reward.from_dict(reward_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


