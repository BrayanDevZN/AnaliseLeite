package operations

import (
	"context"
	"fmt"
	"time"
)

const userColumns = "id, name, email, password, role, profile_photo, created_at"

// User e uma linha da tabela users. Password e ProfilePhoto sao opcionais (nil vira NULL).
type User struct {
	ID           int64
	Name         string
	Email        string
	Password     *string
	Role         string
	ProfilePhoto []byte
	CreatedAt    time.Time
}

// Users tem as operacoes da tabela users.
type Users struct {
	connection Connection
}

// NewUsers cria as operacoes de users usando a conexao.
func NewUsers(connection Connection) *Users {
	return &Users{connection: connection}
}

// Create insere o usuario e preenche ID e CreatedAt nele.
func (u *Users) Create(ctx context.Context, user *User) (err error) {
	defer begin(u.connection, "create user")(&err)

	db, err := database(u.connection)

	if err != nil {
		return err
	}

	user.CreatedAt = time.Now().UTC()

	result, err := db.ExecContext(ctx,
		"INSERT INTO users (name, email, password, role, profile_photo, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		user.Name, user.Email, user.Password, user.Role, user.ProfilePhoto, user.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	user.ID, err = result.LastInsertId()

	return err
}

// GetByID busca o usuario pelo id.
func (u *Users) GetByID(ctx context.Context, id int64) (out *User, err error) {
	defer begin(u.connection, "get user %d", id)(&err)

	return u.getOne(ctx, "id = ?", id)
}

// GetByEmail busca o usuario pelo email (usado no login).
func (u *Users) GetByEmail(ctx context.Context, email string) (out *User, err error) {
	defer begin(u.connection, "get user by email")(&err)

	return u.getOne(ctx, "email = ?", email)
}

// List devolve todos os usuarios.
func (u *Users) List(ctx context.Context) (out []User, err error) {
	defer begin(u.connection, "list users")(&err)

	db, err := database(u.connection)

	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, "SELECT "+userColumns+" FROM users ORDER BY id")

	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	defer rows.Close()

	var users []User

	for rows.Next() {
		user, err := scanUser(rows)

		if err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}

		users = append(users, *user)
	}

	return users, rows.Err()
}

// Update troca os dados do usuario (menos id e created_at).
func (u *Users) Update(ctx context.Context, user *User) (err error) {
	defer begin(u.connection, "update user %d", user.ID)(&err)

	db, err := database(u.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx,
		"UPDATE users SET name = ?, email = ?, password = ?, role = ?, profile_photo = ? WHERE id = ?",
		user.Name, user.Email, user.Password, user.Role, user.ProfilePhoto, user.ID,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return affected(result)
}

// Delete apaga o usuario pelo id.
func (u *Users) Delete(ctx context.Context, id int64) (err error) {
	defer begin(u.connection, "delete user %d", id)(&err)

	db, err := database(u.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)

	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return affected(result)
}

func (u *Users) getOne(ctx context.Context, where string, arg any) (*User, error) {
	db, err := database(u.connection)

	if err != nil {
		return nil, err
	}

	user, err := scanUser(db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE "+where, arg))

	if err != nil {
		return nil, notFound(err)
	}

	return user, nil
}

func scanUser(row scanner) (*User, error) {
	var user User

	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Role, &user.ProfilePhoto, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
