package operations

// Manage junta as operacoes de todas as tabelas, todas usando a mesma conexao.
type Manage struct {
	Users    *Users
	Farms    *Farms
	Tanks    *Tanks
	Analyses *Analyses
	Contents *Contents
}

// NewManage cria as operacoes de todas as tabelas usando a conexao.
func NewManage(connection Connection) *Manage {
	return &Manage{
		Users:    NewUsers(connection),
		Farms:    NewFarms(connection),
		Tanks:    NewTanks(connection),
		Analyses: NewAnalyses(connection),
		Contents: NewContents(connection),
	}
}
