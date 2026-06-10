# Meta

Métadonnées du service : jeux supportés, volumes et fraîcheur des données. Pensé pour les consommateurs (sélecteur de jeu, indicateur « data à jour ? ») et le dogfooding. games liste un MetaGame par valeur de l'enum Game (les jeux sans données apparaissent à 0). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**games** | [**List[MetaGame]**](MetaGame.md) | Jeux supportés et leurs volumes / fraîcheur (un par valeur de l&#39;enum Game). | 
**data_version** | **str** | Version du jeu de données publiée (tampon opérateur, ex. snapshot wiki daté). Absente si non renseignée.  | [optional] 
**generated_at** | **datetime** | Instant de calcul de la réponse (permet d&#39;estimer l&#39;âge des données côté client). | 

## Example

```python
from forza_open_api_client.models.meta import Meta

# TODO update the JSON string below
json = "{}"
# create an instance of Meta from a JSON string
meta_instance = Meta.from_json(json)
# print the JSON string representation of the object
print(Meta.to_json())

# convert the object into a dict
meta_dict = meta_instance.to_dict()
# create an instance of Meta from a dict
meta_from_dict = Meta.from_dict(meta_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


