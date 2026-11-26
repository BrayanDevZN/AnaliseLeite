package operations

import (
	"context"
	"fmt"
	"time"
)

// photo IS NOT NULL diz se a fazenda tem foto sem trazer os bytes dela (a foto so e lida por GetPhoto).
const farmColumns = "id, user_id, name, city, state, created_at, photo IS NOT NULL"

// Farm e uma linha da tabela farms. UserID e o dono da fazenda.
type Farm struct {
	ID        int64
	UserID    int64
	Name      string
	City      string
	State     string
	CreatedAt time.Time

	// HasPhoto diz se a fazenda tem foto. Os bytes ficam de fora de proposito: as listas e o cache
	// ficariam pesados. Use GetPhoto e SetPhoto.
	HasPhoto bool
}

// Farms tem as operacoes da tabela farms.
type Farms struct {
	connection Connection
}

// NewFarms cria as operacoes de farms usando a conexao.
func NewFarms(connection Connection) *Farms {
	return &Farms{connection: connection}
}

// Create insere a fazenda e preenche ID e CreatedAt nela.
func (f *Farms) Create(ctx context.Context, farm *Farm) (err error) {
	defer begin(f.connection, "create farm")(&err)

	db, err := database(f.connection)

	if err != nil {
		return err
	}

	farm.CreatedAt = time.Now().UTC()

	result, err := db.ExecContext(ctx,
		"INSERT INTO farms (user_id, name, city, state, created_at) VALUES (?, ?, ?, ?, ?)",
		farm.UserID, farm.Name, farm.City, farm.State, farm.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create farm: %w", err)
	}

	farm.ID, err = result.LastInsertId()

	return err
}

// GetByID busca a fazenda pelo id.
func (f *Farms) GetByID(ctx context.Context, id int64) (out *Farm, err error) {
	defer begin(f.connection, "get farm %d", id)(&err)

	db, err := database(f.connection)

	if err != nil {
		return nil, err
	}

	farm, err := scanFarm(db.QueryRowContext(ctx, "SELECT "+farmColumns+" FROM farms WHERE id = ?", id))

	if err != nil {
		return nil, notFound(err)
	}

	return farm, nil
}

// ListByUser devolve as fazendas de um usuario.
func (f *Farms) ListByUser(ctx context.Context, userID int64) (out []Farm, err error) {
	defer begin(f.connection, "list farms of user %d", userID)(&err)

	db, err := database(f.connection)

	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, "SELECT "+farmColumns+" FROM farms WHERE user_id = ? ORDER BY id", userID)

	if err != nil {
		return nil, fmt.Errorf("list farms: %w", err)
	}

	defer rows.Close()

	var farms []Farm

	for rows.Next() {
		farm, err := scanFarm(rows)

		if err != nil {
			return nil, fmt.Errorf("list farms: %w", err)
		}

		farms = append(farms, *farm)
	}

	return farms, rows.Err()
}

// Update troca os dados da fazenda (menos id e created_at).
func (f *Farms) Update(ctx context.Context, farm *Farm) (err error) {
	defer begin(f.connection, "update farm %d", farm.ID)(&err)

	db, err := database(f.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx,
		"UPDATE farms SET user_id = ?, name = ?, city = ?, state = ? WHERE id = ?",
		farm.UserID, farm.Name, farm.City, farm.State, farm.ID,
	)

	if err != nil {
		return fmt.Errorf("update farm: %w", err)
	}

	return affected(result)
}

// Delete apaga a fazenda pelo id.
func (f *Farms) Delete(ctx context.Context, id int64) (err error) {
	defer begin(f.connection, "delete farm %d", id)(&err)

	db, err := database(f.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, "DELETE FROM farms WHERE id = ?", id)

	if err != nil {
		return fmt.Errorf("delete farm: %w", err)
	}

	return affected(result)
}

// GetPhoto devolve os bytes da foto da fazenda, ou nil se ela nao tem foto. ErrNotFound se a fazenda nao existe.
func (f *Farms) GetPhoto(ctx context.Context, id int64) (photo []byte, err error) {
	defer begin(f.connection, "get farm %d photo", id)(&err)

	db, err := database(f.connection)

	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, "SELECT photo FROM farms WHERE id = ?", id).Scan(&photo)

	if err != nil {
		return nil, notFound(err)
	}

	return photo, nil
}

// SetPhoto grava a foto da fazenda. Com photo nil, remove a foto.
func (f *Farms) SetPhoto(ctx context.Context, id int64, photo []byte) (err error) {
	defer begin(f.connection, "set farm %d photo", id)(&err)

	db, err := database(f.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, "UPDATE farms SET photo = ? WHERE id = ?", photo, id)

	if err != nil {
		return fmt.Errorf("set farm photo: %w", err)
	}

	return affected(result)
}

func scanFarm(row scanner) (*Farm, error) {
	var farm Farm

	err := row.Scan(&farm.ID, &farm.UserID, &farm.Name, &farm.City, &farm.State, &farm.CreatedAt, &farm.HasPhoto)

	if err != nil {
		return nil, err
	}

	return &farm, nil
}
