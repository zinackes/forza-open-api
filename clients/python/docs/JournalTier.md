# JournalTier

Palier du Collection Journal FH6 : un niveau (1-7) d'une piste (horizon_festival = wristbands colorés Yellow → Gold ; discover_japan = stamps Visitor → Master Explorer). Atteint à pointsRequired points de collection. Peut débloquer une voiture (rewardCarId) et/ou du contenu (unlocksDescription : Legend Island + The Goliath, poussée des Barn Finds…). Remplace les Accolades de FH5. Champs non sourcés → omis. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**track** | [**JournalTrack**](JournalTrack.md) |  | 
**level** | **int** | Niveau du palier dans la piste (1-7). | 
**color** | [**JournalTierColor**](JournalTierColor.md) |  | [optional] 
**name** | **str** | Nom du palier (couleur du wristband ou rang de stamp, ex. \&quot;Gold\&quot;, \&quot;Master Explorer\&quot;). | 
**points_required** | **int** | Points de collection requis pour atteindre le palier. | [optional] 
**reward_car_id** | **str** | Voiture débloquée par ce palier (réf. /v1/cars). Absent si le palier ne donne pas de voiture. | [optional] 
**unlocks_description** | **str** | Contenu débloqué par le palier (ex. \&quot;Legend Island + The Goliath\&quot;, poussée des Barn Finds). Absent si non sourcé.  | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.journal_tier import JournalTier

# TODO update the JSON string below
json = "{}"
# create an instance of JournalTier from a JSON string
journal_tier_instance = JournalTier.from_json(json)
# print the JSON string representation of the object
print(JournalTier.to_json())

# convert the object into a dict
journal_tier_dict = journal_tier_instance.to_dict()
# create an instance of JournalTier from a dict
journal_tier_from_dict = JournalTier.from_dict(journal_tier_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


