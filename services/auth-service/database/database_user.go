package database

import "auth-service/domain"

func (d *Database) ListUsers() ([]domain.User, error) {
	query := `SELECT * FROM users ORDER BY created_at DESC`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		user := domain.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
			&user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (d *Database) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := d.db.Exec(query, id)
	return err
}
