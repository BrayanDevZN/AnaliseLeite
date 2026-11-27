package repository

import (
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// Manage junta a conexao, as migrations e as operacoes das tabelas, todas usando a mesma conexao.
type Manage struct {
	Connection *Connection
	Migrate    *Migrate
	Operations *operations.Manage
}

// NewManage cria o Migrate e as operacoes de todas as tabelas usando a conexao.
func NewManage(connection *Connection) *Manage {
	// um *Connection nil dentro de uma interface nao seria nil, entao o nil e passado direto
	var conn operations.Connection

	if connection != nil {
		conn = connection
	}

	return &Manage{
		Connection: connection,
		Migrate:    NewMigrate(connection),
		Operations: operations.NewManage(conn),
	}
}
