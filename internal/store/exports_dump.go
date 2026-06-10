// Lecture brute d'une ressource pour un jeu, destinée aux archives
// téléchargeables (internal/export). On délègue la sérialisation à Postgres via
// to_jsonb : chaque ligne sort en objet JSON aux types corrects (entiers,
// numériques, JSONB imbriqué, NULL → null) sans scan colonne par colonne. Le dump
// est le reflet BRUT des tables (colonnes snake_case), pas la vue camelCase du
// contrat : un dataset offline fidèle à la base. Lecture seule, paramétrée.
package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// Dump est le contenu d'une ressource pour un jeu, prêt à sérialiser. Rows : une
// ligne par enregistrement, chacune un objet JSON (to_jsonb). Columns : colonnes
// ordonnées de la table source, pour la projection CSV ; nil pour une ressource
// imbriquée (playlist) qui n'a pas de forme tabulaire (donc pas de CSV).
type Dump struct {
	Columns []string
	Rows    []json.RawMessage
}

// flatDump décrit le dump tabulaire d'une ressource « plate » : la table dont on
// lit l'ordre des colonnes (CSV) et la requête to_jsonb produisant les lignes
// ($1 = game).
type flatDump struct {
	table    string
	rowQuery string
}

// flatDumps mappe chaque ressource « plate » exportable à sa requête. Tables et
// requêtes sont des CONSTANTES internes (jamais d'entrée utilisateur) → aucune
// injection. mastery dérive le jeu via cars (car_mastery_perks n'a pas de colonne
// game) ; le fichier étant déjà par-jeu, l'absence de colonne game est sans perte.
var flatDumps = map[string]flatDump{
	"cars":          {"cars", `SELECT to_jsonb(t) FROM cars t WHERE t.game = $1 ORDER BY t.id`},
	"tracks":        {"tracks", `SELECT to_jsonb(t) FROM tracks t WHERE t.game = $1 ORDER BY t.id`},
	"pr_stunts":     {"pr_stunts", `SELECT to_jsonb(t) FROM pr_stunts t WHERE t.game = $1 ORDER BY t.id`},
	"events":        {"events", `SELECT to_jsonb(t) FROM events t WHERE t.game = $1 ORDER BY t.id`},
	"barn_finds":    {"barn_finds", `SELECT to_jsonb(t) FROM barn_finds t WHERE t.game = $1 ORDER BY t.id`},
	"treasure_cars": {"treasure_cars", `SELECT to_jsonb(t) FROM treasure_cars t WHERE t.game = $1 ORDER BY t.id`},
	"mastery":       {"car_mastery_perks", `SELECT to_jsonb(t) FROM car_mastery_perks t JOIN cars c ON c.id = t.car_id WHERE c.game = $1 ORDER BY t.id`},
	"journal":       {"journal_tiers", `SELECT to_jsonb(t) FROM journal_tiers t WHERE t.game = $1 ORDER BY t.id`},
	"dlc_packs":     {"dlc_packs", `SELECT to_jsonb(t) FROM dlc_packs t WHERE t.game = $1 ORDER BY t.id`},
	"manufacturers": {"manufacturers", `SELECT to_jsonb(t) FROM manufacturers t WHERE t.game = $1 ORDER BY t.name`},
}

// DumpResource renvoie le contenu brut d'une ressource pour un jeu. playlist est
// imbriquée (séries + rewards + challenges, courant + historique) → Columns nil
// (json/jsonl seulement). Les autres sont des dumps tabulaires (Columns pour le CSV).
func (s *Store) DumpResource(ctx context.Context, resource, game string) (Dump, error) {
	if resource == "playlist" {
		return s.dumpPlaylist(ctx, game)
	}
	fd, ok := flatDumps[resource]
	if !ok {
		return Dump{}, fmt.Errorf("dump: ressource inconnue %q", resource)
	}
	cols, err := s.tableColumns(ctx, fd.table)
	if err != nil {
		return Dump{}, err
	}
	rows, err := s.jsonRows(ctx, fd.rowQuery, game)
	if err != nil {
		return Dump{}, err
	}
	return Dump{Columns: cols, Rows: rows}, nil
}

// dumpPlaylist sérialise chaque série du jeu (courante ET historique) avec ses
// rewards et challenges imbriqués (un objet JSON par série). Pas de Columns :
// structure imbriquée, non aplatie en CSV.
func (s *Store) dumpPlaylist(ctx context.Context, game string) (Dump, error) {
	const q = `
SELECT to_jsonb(s) || jsonb_build_object(
    'rewards', COALESCE(
        (SELECT jsonb_agg(to_jsonb(r) ORDER BY r.at_percent NULLS LAST, r.id)
         FROM rewards r WHERE r.series_id = s.id),
        '[]'::jsonb),
    'challenges', COALESCE(
        (SELECT jsonb_agg(to_jsonb(c) ORDER BY c.id)
         FROM challenges c WHERE c.series_id = s.id),
        '[]'::jsonb)
)
FROM series s
WHERE s.game = $1
ORDER BY s.series NULLS LAST, s.week NULLS LAST, s.id`
	rows, err := s.jsonRows(ctx, q, game)
	if err != nil {
		return Dump{}, err
	}
	return Dump{Rows: rows}, nil
}

// tableColumns lit l'ordre des colonnes d'une table via une requête vide
// (FieldDescriptions) — source de vérité de l'ordre pour le CSV. table est une
// constante interne (cf. flatDumps), jamais une entrée utilisateur.
func (s *Store) tableColumns(ctx context.Context, table string) ([]string, error) {
	rows, err := s.DB.Query(ctx, "SELECT * FROM "+table+" WHERE false")
	if err != nil {
		return nil, fmt.Errorf("colonnes de %s: %w", table, err)
	}
	defer rows.Close()
	fds := rows.FieldDescriptions()
	cols := make([]string, len(fds))
	for i := range fds {
		cols[i] = fds[i].Name
	}
	return cols, nil
}

// jsonRows exécute une requête to_jsonb ($1 = game) et collecte chaque ligne comme
// un objet JSON brut (copie défensive : le buffer pgx peut être réutilisé).
func (s *Store) jsonRows(ctx context.Context, query, game string) ([]json.RawMessage, error) {
	rows, err := s.DB.Query(ctx, query, game)
	if err != nil {
		return nil, fmt.Errorf("dump rows: %w", err)
	}
	defer rows.Close()

	out := []json.RawMessage{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan dump row: %w", err)
		}
		out = append(out, json.RawMessage(append([]byte(nil), raw...)))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dump rows: %w", err)
	}
	return out, nil
}
