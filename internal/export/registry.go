package export

// resourceSpec déclare une ressource exportable et ses formats. Les ressources
// « plates » (une table) sortent en json/jsonl/csv ; les imbriquées (playlist :
// séries + rewards + challenges) en json/jsonl seulement — on n'aplatit pas de
// force une structure imbriquée en CSV.
type resourceSpec struct {
	resource string
	formats  []string
}

// registry énumère les ressources archivées, dans un ordre stable. Le manifeste
// ne liste QUE ce qui est réellement généré ; ajouter une ressource = ajouter une
// entrée ici (et son dump dans store.DumpResource).
var registry = []resourceSpec{
	{"cars", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"playlist", []string{FormatJSON, FormatJSONL}},
	{"tracks", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"pr_stunts", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"events", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"barn_finds", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"treasure_cars", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"mastery", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"journal", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"dlc_packs", []string{FormatJSON, FormatJSONL, FormatCSV}},
	{"manufacturers", []string{FormatJSON, FormatJSONL, FormatCSV}},
}

// ext est l'extension de fichier d'un format.
func ext(format string) string { return format } // json|jsonl|csv coïncident

// contentType est le type MIME d'un format d'archive.
func contentType(format string) string {
	switch format {
	case FormatCSV:
		return "text/csv; charset=utf-8"
	case FormatJSONL:
		return "application/x-ndjson"
	default:
		return "application/json"
	}
}
