package operations

import (
	"context"
	"fmt"
	"time"
)

const analysisColumns = "id, farm_id, tank_id, created_at"

// Analysis e uma linha da tabela analyses. TankID e opcional (nil vira NULL).
type Analysis struct {
	ID        int64
	FarmID    int64
	TankID    *int64
	CreatedAt time.Time
}

// Analyses tem as operacoes da tabela analyses.
type Analyses struct {
	connection Connection
}

// NewAnalyses cria as operacoes de analyses usando a conexao.
func NewAnalyses(connection Connection) *Analyses {
	return &Analyses{connection: connection}
}

// Create insere a analise e preenche ID e CreatedAt nela.
func (a *Analyses) Create(ctx context.Context, analysis *Analysis) (err error) {
	defer begin(a.connection, "create analysis")(&err)

	db, err := database(a.connection)

	if err != nil {
		return err
	}

	analysis.CreatedAt = time.Now().UTC()

	result, err := db.ExecContext(ctx,
		"INSERT INTO analyses (farm_id, tank_id, created_at) VALUES (?, ?, ?)",
		analysis.FarmID, analysis.TankID, analysis.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create analysis: %w", err)
	}

	analysis.ID, err = result.LastInsertId()

	return err
}

// GetByID busca a analise pelo id.
func (a *Analyses) GetByID(ctx context.Context, id int64) (out *Analysis, err error) {
	defer begin(a.connection, "get analysis %d", id)(&err)

	db, err := database(a.connection)

	if err != nil {
		return nil, err
	}

	analysis, err := scanAnalysis(db.QueryRowContext(ctx, "SELECT "+analysisColumns+" FROM analyses WHERE id = ?", id))

	if err != nil {
		return nil, notFound(err)
	}

	return analysis, nil
}

// ListByFarm devolve as analises de uma fazenda, da mais nova para a mais antiga.
func (a *Analyses) ListByFarm(ctx context.Context, farmID int64) (out []Analysis, err error) {
	defer begin(a.connection, "list analyses of farm %d", farmID)(&err)

	db, err := database(a.connection)

	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, "SELECT "+analysisColumns+" FROM analyses WHERE farm_id = ? ORDER BY id DESC", farmID)

	if err != nil {
		return nil, fmt.Errorf("list analyses: %w", err)
	}

	defer rows.Close()

	var analyses []Analysis

	for rows.Next() {
		analysis, err := scanAnalysis(rows)

		if err != nil {
			return nil, fmt.Errorf("list analyses: %w", err)
		}

		analyses = append(analyses, *analysis)
	}

	return analyses, rows.Err()
}

// Update troca os dados da analise (menos id e created_at).
func (a *Analyses) Update(ctx context.Context, analysis *Analysis) (err error) {
	defer begin(a.connection, "update analysis %d", analysis.ID)(&err)

	db, err := database(a.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx,
		"UPDATE analyses SET farm_id = ?, tank_id = ? WHERE id = ?",
		analysis.FarmID, analysis.TankID, analysis.ID,
	)

	if err != nil {
		return fmt.Errorf("update analysis: %w", err)
	}

	return affected(result)
}

// Delete apaga a analise pelo id.
func (a *Analyses) Delete(ctx context.Context, id int64) (err error) {
	defer begin(a.connection, "delete analysis %d", id)(&err)

	db, err := database(a.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, "DELETE FROM analyses WHERE id = ?", id)

	if err != nil {
		return fmt.Errorf("delete analysis: %w", err)
	}

	return affected(result)
}

func scanAnalysis(row scanner) (*Analysis, error) {
	var analysis Analysis

	err := row.Scan(&analysis.ID, &analysis.FarmID, &analysis.TankID, &analysis.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &analysis, nil
}
