package main


type Config struct {
	T  TableInfo `toml:"table"`
	DB Database `toml:"database"`
	WL  Workload `toml:"workload"`
}

type TableInfo struct {
	TableName  string
	FieldType  []string
	FieldName  []string
	IndexName  []string
	PrimaryKey string
	Engine     string
	Charset    string
}

type Database struct {
	Host     string
	User     string
	Password string
	Port     string
	DBName	 string
}

type Workload struct {
	NumItem int
	MinLenVarchar int
}