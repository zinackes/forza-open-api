# Error

Erreur au format RFC 9457 (application/problem+json). Toutes les réponses d'erreur (400, 401, 404, 429, 5xx) partagent ce format. Le champ `type` porte un code stable (URN, indépendant de l'host) :   - urn:forza-open-api:problem:validation — requête invalide (400) ;   - urn:forza-open-api:problem:unauthorized — clé API absente/invalide (401) ;   - urn:forza-open-api:problem:not-found — ressource introuvable (404) ;   - urn:forza-open-api:problem:rate-limited — quota dépassé (429) ;   - urn:forza-open-api:problem:internal — erreur interne (500). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type** | **str** | Code d&#39;erreur stable (URN). Voir la liste dans la description du schéma. Référence stable dans le temps, dissociée du statut HTTP.  | [optional] [default to 'about:blank']
**title** | **str** |  | [optional] 
**status** | **int** |  | [optional] 
**detail** | **str** |  | [optional] 
**instance** | **str** | Chemin de la requête à l&#39;origine de l&#39;erreur (ex. /v1/cars/ghost). | [optional] 

## Example

```python
from forza_open_api_client.models.error import Error

# TODO update the JSON string below
json = "{}"
# create an instance of Error from a JSON string
error_instance = Error.from_json(json)
# print the JSON string representation of the object
print(Error.to_json())

# convert the object into a dict
error_dict = error_instance.to_dict()
# create an instance of Error from a dict
error_from_dict = Error.from_dict(error_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


