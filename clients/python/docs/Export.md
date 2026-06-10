# Export

Archive statique téléchargeable d'une ressource pour un jeu, dans un format donné. `url` pointe le fichier servi depuis l'edge (cache long + ETag) ; `etag` et `sizeBytes` décrivent ce fichier ; `generatedAt` = instant de régénération de l'archive. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**game** | [**Game**](Game.md) |  | 
**resource** | [**ExportResource**](ExportResource.md) |  | 
**format** | [**ExportFormat**](ExportFormat.md) |  | 
**url** | **str** | URL absolue du fichier d&#39;archive (servi depuis l&#39;edge). | 
**size_bytes** | **int** | Taille du fichier en octets. | 
**etag** | **str** | ETag du fichier (hash de contenu), pour la revalidation conditionnelle. | 
**generated_at** | **datetime** | Instant de régénération de l&#39;archive. | 

## Example

```python
from forza_open_api_client.models.export import Export

# TODO update the JSON string below
json = "{}"
# create an instance of Export from a JSON string
export_instance = Export.from_json(json)
# print the JSON string representation of the object
print(Export.to_json())

# convert the object into a dict
export_dict = export_instance.to_dict()
# create an instance of Export from a dict
export_from_dict = Export.from_dict(export_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


