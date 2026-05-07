package model

import (
	"database/sql"
	"fmt"
	"strings"
)

type DeviceRepo struct {
	db *sql.DB
}

func NewDeviceRepo(db *sql.DB) *DeviceRepo {
	return &DeviceRepo{db: db}
}

func (r *DeviceRepo) FindDeviceIDsByUserAndSNs(userID int64, sns []string) (map[string]int64, error) {
	if len(sns) == 0 {
		return map[string]int64{}, nil
	}

	placeholders := make([]string, len(sns))
	args := make([]interface{}, len(sns)+1)
	args[0] = userID
	for i := range sns {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = sns[i]
	}

	query := fmt.Sprintf(`
SELECT d.sn, d.id
FROM device d
INNER JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1
WHERE udb.user_id = $1 AND d.sn IN (%s)
`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var sn string
		var id int64
		if err := rows.Scan(&sn, &id); err != nil {
			return nil, err
		}
		result[sn] = id
	}
	return result, nil
}
