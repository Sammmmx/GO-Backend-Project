package main

import "database/sql"

func GetUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(
		`SELECT id, name, age, date_added FROM users`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Age,
			&user.DateAdded,
		); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func InsertUser(db *sql.DB, name string, age int) (User, error) {
	var user User

	err := db.QueryRow(
		`INSERT INTO users (name, age)
		 VALUES ($1, $2)
		 RETURNING id, name, age, date_added`,
		name, age,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Age,
		&user.DateAdded,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func GetUserByID(db *sql.DB, id int) (User, error) {
	var user User

	err := db.QueryRow(
		`SELECT id, name, age, date_added
		 FROM users
		 WHERE id = $1`,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Age,
		&user.DateAdded,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func DeleteUserByID(db *sql.DB, id int) error {
	_, err := db.Exec(
		`DELETE FROM users WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func UpdateUserByID(db *sql.DB, id int, name string, age int) (User, error) {
	var user User
	err := db.QueryRow(
		`UPDATE users
		 SET name = $1, age = $2
		 WHERE id = $3
		 RETURNING id, name, age, date_added`,
		name, age, id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Age,
		&user.DateAdded,
	)
	if err != nil {
		return User{}, err
	}
	return user, nil
}
