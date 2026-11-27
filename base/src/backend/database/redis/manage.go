package redis

// Manage junta a conexao com o Redis e as operacoes dele, todas usando a mesma conexao.
type Manage struct {
	Connection *Connection
	Operations *Operations
}

// NewManage cria as operacoes do Redis usando a conexao.
func NewManage(connection *Connection) *Manage {
	return &Manage{
		Connection: connection,
		Operations: NewOperations(connection),
	}
}

// Close espera as operacoes assincronas terminarem e fecha a conexao.
func (m *Manage) Close() error {
	m.Operations.Wait()

	return m.Connection.Close()
}
