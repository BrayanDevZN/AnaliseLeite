package operations

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const contentColumns = "id, analysis_id, content, file, created_at"

// ErrInvalidContent quando content e file estao os dois preenchidos ou os dois vazios.
var ErrInvalidContent = errors.New("exactly one of content or file must be set")

// Content e uma linha da tabela contents.
// Content guarda os dados em JSON (temperatura e outros) e File guarda os bytes do arquivo.
// Exatamente um dos dois deve estar preenchido; o outro fica nil (NULL).
type Content struct {
	ID         int64
	AnalysisID int64
	Content    []byte
	File       []byte
	CreatedAt  time.Time
}

// Contents tem as operacoes da tabela contents.
type Contents struct {
	connection Connection
}

// NewContents cria as operacoes de contents usando a conexao.
func NewContents(connection Connection) *Contents {
	return &Contents{connection: connection}
}

// Create insere o conteudo e preenche ID e CreatedAt nele.
func (c *Contents) Create(ctx context.Context, content *Content) (err error) {
	defer begin(c.connection, "create content")(&err)

	if err := content.validate(); err != nil {
		return err
	}

	db, err := database(c.connection)

	if err != nil {
		return err
	}

	content.CreatedAt = time.Now().UTC()

	result, err := db.ExecContext(ctx,
		"INSERT INTO contents (analysis_id, content, file, created_at) VALUES (?, ?, ?, ?)",
		content.AnalysisID, content.jsonValue(), content.File, content.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create content: %w", err)
	}

	content.ID, err = result.LastInsertId()

	return err
}

// GetByID busca o conteudo pelo id.
func (c *Contents) GetByID(ctx context.Context, id int64) (out *Content, err error) {
	defer begin(c.connection, "get content %d", id)(&err)

	db, err := database(c.connection)

	if err != nil {
		return nil, err
	}

	content, err := scanContent(db.QueryRowContext(ctx, "SELECT "+contentColumns+" FROM contents WHERE id = ?", id))

	if err != nil {
		return nil, notFound(err)
	}

	return content, nil
}

// ListByAnalysis devolve os conteudos de uma analise.
func (c *Contents) ListByAnalysis(ctx context.Context, analysisID int64) (out []Content, err error) {
	defer begin(c.connection, "list contents of analysis %d", analysisID)(&err)

	db, err := database(c.connection)

	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, "SELECT "+contentColumns+" FROM contents WHERE analysis_id = ? ORDER BY id", analysisID)

	if err != nil {
		return nil, fmt.Errorf("list contents: %w", err)
	}

	defer rows.Close()

	var contents []Content

	for rows.Next() {
		content, err := scanContent(rows)

		if err != nil {
			return nil, fmt.Errorf("list contents: %w", err)
		}

		contents = append(contents, *content)
	}

	return contents, rows.Err()
}

// Update troca os dados do conteudo (menos id e created_at).
func (c *Contents) Update(ctx context.Context, content *Content) (err error) {
	defer begin(c.connection, "update content %d", content.ID)(&err)

	if err := content.validate(); err != nil {
		return err
	}

	db, err := database(c.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx,
		"UPDATE contents SET analysis_id = ?, content = ?, file = ? WHERE id = ?",
		content.AnalysisID, content.jsonValue(), content.File, content.ID,
	)

	if err != nil {
		return fmt.Errorf("update content: %w", err)
	}

	return affected(result)
}

// Delete apaga o conteudo pelo id.
func (c *Contents) Delete(ctx context.Context, id int64) (err error) {
	defer begin(c.connection, "delete content %d", id)(&err)

	db, err := database(c.connection)

	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, "DELETE FROM contents WHERE id = ?", id)

	if err != nil {
		return fmt.Errorf("delete content: %w", err)
	}

	return affected(result)
}

// validate garante que so um entre Content e File esta preenchido.
func (c *Content) validate() error {
	if (c.Content == nil) == (c.File == nil) {
		return ErrInvalidContent
	}

	return nil
}

// jsonValue devolve o JSON como texto (o JSON e salvo como TEXT), ou nil para NULL.
func (c *Content) jsonValue() any {
	if c.Content == nil {
		return nil
	}

	return string(c.Content)
}

func scanContent(row scanner) (*Content, error) {
	var content Content

	err := row.Scan(&content.ID, &content.AnalysisID, &content.Content, &content.File, &content.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &content, nil
}
