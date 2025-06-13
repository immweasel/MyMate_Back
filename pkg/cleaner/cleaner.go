package cleaner

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Clean(pool *pgxpool.Pool) {
	query := `SELECT id, name, about, price_from, price_to, neighborhoods_count, 
	neighborhood_age_from, neighborhood_age_to, sex,
	created_at, created_by_id, up_in_search FROM flat WHERE created_at < NOW() - INTERVAL '1 MONTH'`
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var id int64
		err := rows.Scan(&id)
		if err != nil {
			log.Printf("ERROR|cleaner.Clean:%s", err.Error())
			continue
		}
		imageRows, err := pool.Query(context.Background(), "SELECT id, flat_id, url, filename FROM flat_image WHERE flat_id = $1", id)
		if err != nil {
			log.Printf("ERROR|cleaner.Clean:%s", err.Error())
			continue
		}
		for imageRows.Next() {
			var imageId int64
			var flatId int64
			var url string
			var filename string
			err := imageRows.Scan(&imageId, &flatId, &url, &filename)
			if err != nil {
				log.Printf("ERROR|cleaner.Clean:%s", err.Error())
				continue
			}
			_, err = pool.Exec(context.Background(), "DELETE FROM flat_image WHERE id = $1", imageId)
			if err != nil {
				log.Printf("ERROR|cleaner.Clean:%s", err.Error())
				continue
			}
			if filename != "" {
				err = os.Remove(filepath.Join(".", "media", "flats", strconv.FormatInt(flatId, 10), filename))
				if err != nil {
					log.Printf("ERROR|cleaner.Clean:%s", err.Error())
				}
			}
		}
		_, err = pool.Exec(context.Background(), "DELETE FROM flat WHERE id = $1", id)
		if err != nil {
			log.Printf("ERROR|cleaner.Clean:%s", err.Error())
			continue
		}
	}
}
