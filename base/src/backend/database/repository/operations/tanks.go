package operations

import (
	"context"
	"fmt"
	"time"
)

const tankColumns = "id, farm_id, type, capacity_liters, created_at"

// Tank e uma linha da tabela tanks. FarmID e a fazenda do tanque.
type Tank struct {
	ID             int64
	FarmID         int64
	Type           string
	CapacityLiters float64
	CreatedAt      time.Time
}

// Tanks tem as operacoes da tabela tanks.
type Tanks struct {
	connection Connection
}

// NewTanks cria as operacoes de tanks usando a conexao.
func NewTanks(connection Connection) *Tanks {
	return &Tanks{connection: connection}
}

// Create insere o tanque e preenche ID e CreatedAt nele.
func (t *Tanks) Create(ctx context.Context, tank *Tank) (err error) {
	defer begin(t.connection, "create tank")(&err)

	db, err := database(t.connection)

	if err != nil {
		return err
	}

	tank.CreatedAt = time.Now().UTC()

	result, err := db.ExecContext(ctx,
		"INSERT INTO tanks (farm_id, type, capacity_liters, created_at) VALUES (?, ?, ?, ?)",
		tank.FarmID, tank.Type, tank.CapacityLiters, tank.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create tank: %w", err)
	}

	tank.ID, err = result.LastInsertId()

	return err
}

// GetByID busca o tanque pelo id.
func (t *Tanks) GetByID(ctx context.Context, id int64) (out *Tank, err error) {
	defer begin(t.connection, "get tank %d", id)(&err)

	db, err := database(t.connection)

	if err != nil {
		return nil, err
	}

	tank, err := scanTank(db.QueryRowContext(ctx, "SELECT "+tankColumns+" FROM tanks WHERE id = ?", id))

	if err != nil {
		return nil, notFound(err)
	}

	return tank, nil
}

// ListByFarm devolve os tanques de uma fazenda.
func (t *Tanks) ListByFarm(ctx context.Context, farmID int64) (out []Tank, err error) {
	defer begin(t.connection, "list tanks of farm %d", farmID)(&err)

	db, err := database(t.connection)

	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, "SELECT "+tankColumns+" FROM tanks WHERE farm_id = ? ORDER BY id", farmID)

	if err != nil {
		return nil, fmt.Errorf("list tanks: %w", err)
	}

	defer rows.Close()

	var tanks []Tank

	for rows.Next() {
		tank, err := scanTank(rows)

		if err != nil {
			return nil, fmt.Errorf("list tanks: %w", err)
		}

		tanks = append(tanks, *tank)
	}

	return tanks, rows.Err()
}

// Update troca os dados do tanque (menos id e created_at).
func (t *Tanks) Update(ctx context.Context, tank *Tank) (err error) {
	defer begin(t.connection, "update tank %d", tank.ID)(&err)

	db, err := database(t.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx,
		"UPDATE tanks SET farm_id = ?, type = ?, capacity_liters = ? WHERE id = ?",
		tank.FarmID, tank.Type, tank.CapacityLiters, tank.ID,
	)

	if err != nil {
		return fmt.Errorf("update tank: %w", err)
	}

	return affected(result)
}

// Delete apaga o tanque pelo id.
func (t *Tanks) Delete(ctx context.Context, id int64) (err error) {
	defer begin(t.connection, "delete tank %d", id)(&err)

	db, err := database(t.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, "DELETE FROM tanks WHERE id = ?", id)

	if err != nil {
		return fmt.Errorf("delete tank: %w", err)
	}

	return affected(result)
}

func scanTank(row scanner) (*Tank, error) {
	var tank Tank

	err := row.Scan(&tank.ID, &tank.FarmID, &tank.Type, &tank.CapacityLiters, &tank.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &tank, nil
}
