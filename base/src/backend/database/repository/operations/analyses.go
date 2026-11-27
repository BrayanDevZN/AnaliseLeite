package operations

import (
	"context"
	"fmt"
	"time"
)

const analysisColumns = "id, farm_id, tank_id, created_at, kind, model, approved, rejected"

// Tipos de analise.
const (
	KindSample = "sample"
	KindFile   = "file"
)

// Analysis e uma linha da tabela analyses. TankID e opcional (nil vira NULL).
type Analysis struct {
	ID        int64
	FarmID    int64
	TankID    *int64
	CreatedAt time.Time

	// Kind e KindSample (uma amostra) ou KindFile (um arquivo); vazio vale KindSample.
	Kind string

	// Model e o modelo usado ("" e o modelo padrao do pipeline).
	Model string

	// Approved e Rejected sao quantas amostras o modelo aprovou e reprovou: uma amostra da 1 e 0 (ou 0 e 1),
	// um arquivo da uma contagem por linha. As analises antigas, de antes de guardarmos o resultado, ficam com 0 e 0.
	Approved int
	Rejected int
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

	if analysis.Kind == "" {
		analysis.Kind = KindSample
	}

	result, err := db.ExecContext(ctx,
		"INSERT INTO analyses (farm_id, tank_id, created_at, kind, model, approved, rejected) VALUES (?, ?, ?, ?, ?, ?, ?)",
		analysis.FarmID, analysis.TankID, analysis.CreatedAt, analysis.Kind, analysis.Model, analysis.Approved, analysis.Rejected,
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

	err := row.Scan(&analysis.ID, &analysis.FarmID, &analysis.TankID, &analysis.CreatedAt, &analysis.Kind, &analysis.Model, &analysis.Approved, &analysis.Rejected)

	if err != nil {
		return nil, err
	}

	return &analysis, nil
}

// UserAnalysis e uma analise com o nome da fazenda dela, para a lista "minhas analises".
type UserAnalysis struct {
	Analysis
	FarmName string
}

// ListByUser devolve as analises de todas as fazendas de um usuario, da mais nova para a mais antiga, e o total
// (para paginar). limit e offset escolhem a pagina.
func (a *Analyses) ListByUser(ctx context.Context, userID int64, limit, offset int) (items []UserAnalysis, total int, err error) {
	defer begin(a.connection, "list analyses of user %d", userID)(&err)

	db, err := database(a.connection)

	if err != nil {
		return nil, 0, err
	}

	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM analyses a JOIN farms f ON f.id = a.farm_id WHERE f.user_id = ?", userID,
	).Scan(&total)

	if err != nil {
		return nil, 0, fmt.Errorf("count analyses: %w", err)
	}

	rows, err := db.QueryContext(ctx,
		`SELECT a.id, a.farm_id, a.tank_id, a.created_at, a.kind, a.model, a.approved, a.rejected, f.name
		 FROM analyses a JOIN farms f ON f.id = a.farm_id
		 WHERE f.user_id = ? ORDER BY a.id DESC LIMIT ? OFFSET ?`, userID, limit, offset,
	)

	if err != nil {
		return nil, 0, fmt.Errorf("list analyses: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var item UserAnalysis

		err := rows.Scan(&item.ID, &item.FarmID, &item.TankID, &item.CreatedAt, &item.Kind, &item.Model, &item.Approved, &item.Rejected, &item.FarmName)

		if err != nil {
			return nil, 0, fmt.Errorf("list analyses: %w", err)
		}

		items = append(items, item)
	}

	return items, total, rows.Err()
}

// Totals sao os totais de um usuario, para o dashboard.
type Totals struct {
	Farms    int
	Tanks    int
	Analyses int
	Approved int
	Rejected int
}

// Totals soma as fazendas, os tanques, as analises e as amostras aprovadas e reprovadas de um usuario.
func (a *Analyses) Totals(ctx context.Context, userID int64) (totals Totals, err error) {
	defer begin(a.connection, "get totals of user %d", userID)(&err)

	db, err := database(a.connection)

	if err != nil {
		return Totals{}, err
	}

	err = db.QueryRowContext(ctx,
		`SELECT
			(SELECT COUNT(*) FROM farms WHERE user_id = ?),
			(SELECT COUNT(*) FROM tanks t JOIN farms f ON f.id = t.farm_id WHERE f.user_id = ?),
			COUNT(a.id), COALESCE(SUM(a.approved), 0), COALESCE(SUM(a.rejected), 0)
		 FROM analyses a JOIN farms f ON f.id = a.farm_id WHERE f.user_id = ?`, userID, userID, userID,
	).Scan(&totals.Farms, &totals.Tanks, &totals.Analyses, &totals.Approved, &totals.Rejected)

	if err != nil {
		return Totals{}, fmt.Errorf("get totals: %w", err)
	}

	return totals, nil
}

// Activity e uma analise resumida (quando e quantas amostras), para agrupar por dia.
type Activity struct {
	CreatedAt time.Time
	Approved  int
	Rejected  int
}

// Activity devolve as analises de um usuario feitas a partir de since.
func (a *Analyses) Activity(ctx context.Context, userID int64, since time.Time) (points []Activity, err error) {
	defer begin(a.connection, "get activity of user %d", userID)(&err)

	db, err := database(a.connection)

	if err != nil {
		return nil, err
	}

	// o filtro do SQL e folgado (um dia a mais); o corte exato e feito aqui, sem depender de comparar texto de data
	rows, err := db.QueryContext(ctx,
		`SELECT a.created_at, a.approved, a.rejected FROM analyses a JOIN farms f ON f.id = a.farm_id
		 WHERE f.user_id = ? AND a.created_at >= ? ORDER BY a.created_at`, userID, since.Add(-24*time.Hour),
	)

	if err != nil {
		return nil, fmt.Errorf("get activity: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var point Activity

		if err := rows.Scan(&point.CreatedAt, &point.Approved, &point.Rejected); err != nil {
			return nil, fmt.Errorf("get activity: %w", err)
		}

		if !point.CreatedAt.Before(since) {
			points = append(points, point)
		}
	}

	return points, rows.Err()
}
