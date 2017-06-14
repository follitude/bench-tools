package main

import (
	"database/sql"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
	"github.com/pingcap/tidb/_vendor/src/github.com/ngaut/log"
)

const (
	RAND_KIND_NUM   = 0
	RAND_KIND_LOWER = 1
	RAND_KIND_UPPER = 2
	RAND_KIND_ALL   = 3
)

type Conn struct {
	db *sql.DB
}

func check(e error) {
	if e != nil {
		errors.Trace(e)
	}
}

func varcharRanGenerator(maxByte int, kind int ,minLenVarchar int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	kind, kinds, result := kind, [][]int{[]int{48, 10}, []int{97, 26}, []int{65, 26}}, make([]byte, maxByte)
	isAll := kind > 2 || kind < 0
	charNum := r.Intn(maxByte-minLenVarchar) + minLenVarchar
	for i := 0; i < charNum; i++ {
		if isAll {
			kind = r.Intn(3)
		}
		base, scope := kinds[kind][0], kinds[kind][1]
		result[i] = byte(base + r.Intn(scope))
	}
	return "\"" + string(result) + "\""
}

func intRanGenerator() string {
	var num int
	for {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		nums := make([]int, 0)
		num = r.Intn(math.MaxInt32)
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
			}
		}
		if !exist {
			nums = append(nums, num)
			break
		}
	}
	return strconv.Itoa(num)
}

func datetimeGenerotor() string {
	timestamp := time.Now().Unix()
	tm := time.Unix(timestamp, 0)
	res:= tm.Format("'2006-01-02 03:04:05'")
	return res
}

func executeSQL(conn *Conn, sqlStats []string) error {
	if len(sqlStats) == 0 {
		return nil
	}
	if err := executeSQLImp(conn.db, sqlStats); err != nil {
		check(err)
	}
	return nil
}

func executeSQLImp(db *sql.DB, sqlStats []string) error {
	txn, err := db.Begin()
	if err != nil {
		log.Errorf("exec sqls[%-.100v] begin failed %v", sqlStats, errors.ErrorStack(err))
		return err
	}
	for i := range sqlStats {
		log.Debugf("[exec][sql]%-.200v", sqlStats)
		_, err = txn.Exec(sqlStats[i])
		if err != nil {
			log.Errorf("[exec][sql]%-.100v[error]%v", sqlStats, err)
		}
	}
	err = txn.Commit()
	if err != nil {
		log.Errorf("exec sqls[%-.100v] commit failed %v", sqlStats, errors.ErrorStack(err))
		return err
	}
	return nil
}

func connection(host string, user string, password string, port string, dbName string) *Conn {
	db, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+dbName)
	check(err)
	return &Conn{db: db}
}

func dropTable(tableName string) string {
	sqlStat := "DROP TABLE " + tableName + ";\n"
	return sqlStat
}

func createTable(tableName string, fieldName []string, fieldType []string, indexName []string, primaryKey string,
	engine string, charset string) string {
	sqlStat := "CREATE TABLE " + tableName + " (\n"
	for i := 0; i < len(fieldName); i++ {
		sqlStat = sqlStat + " " + fieldName[i] + " " + fieldType[i]
		if i != len(fieldName)-1 || primaryKey != "" {

			if primaryKey == fieldName[i] {
				sqlStat += " NOT NULL AUTO_INCREMENT"
			}
			sqlStat += ",\n"
		} else {
			sqlStat += "\n"
		}
	}
	if primaryKey != "" {
		sqlStat += " PRIMARY KEY (" + primaryKey + ")"
	}
	for i := 0; i < len(indexName); i++ {
		sqlStat += ",\n KEY (" + indexName[i] + ")"
	}

	sqlStat += "\n) ENGINE=" + engine + " DEFAULT CHARSET=" + charset + ";\n"
	return sqlStat
}

func insertTable(tableName string, numItem int, fieldType []string, fieldName []string, minLenVarchar int) string {
	var sqlStat string
	if numItem != 0 {
		sqlStat = "INSERT INTO " + tableName + " VALUES\n"
		for i := 0; i < numItem; i++ {
			sqlStat = sqlStat + "("
			for j := 0; j < len(fieldName); j++ {
				index := 0
				start := strings.Index(fieldType[j], "(")
				end := strings.Index(fieldType[j], ")")
				if -1 != start || -1 != end {
					index, _ = strconv.Atoi(fieldType[j][start+1 : end])
				}
				if strings.HasPrefix(fieldType[j], "int") {
					integer := intRanGenerator()
					max := len(integer)
					if index < max {
						sqlStat += integer[:index]
					} else {
						sqlStat += integer[:max]
					}
				}
				if strings.HasPrefix(fieldType[j], "varchar") {
					sqlStat += varcharRanGenerator(index, RAND_KIND_ALL,minLenVarchar)
				}
				if fieldType[j] == "datetime" {
					sqlStat += datetimeGenerotor()
				}
				if j != len(fieldName)-1 {
					sqlStat += ", "
				}
			}
			if i == numItem-1 {
				sqlStat = sqlStat + ");\n"
			} else {
				sqlStat = sqlStat + "),\n"
			}
		}
	}
	return sqlStat
}

func insert(cfg Config)  {
	tableName := cfg.T.TableName
	fieldType := cfg.T.FieldType
	fieldName := cfg.T.FieldName
	indexName := cfg.T.IndexName
	primaryKey := cfg.T.PrimaryKey
	engine := cfg.T.Engine
	charset := cfg.T.Charset

	host:=cfg.DB.Host
	user:=cfg.DB.User
	password:=cfg.DB.Password
	port:=cfg.DB.Port
	dbName:=cfg.DB.DBName

	numItem:=cfg.WL.NumItem
	minLenVarchar:=cfg.WL.MinLenVarchar

	conn := connection(host, user, password, port, dbName)

	var sqlStats []string
	sqlStat := dropTable(tableName)
	sqlStats = append(sqlStats, sqlStat)
	sqlStat = createTable(tableName, fieldName, fieldType, indexName, primaryKey, engine, charset)
	sqlStats = append(sqlStats, sqlStat)
	sqlStat = insertTable(tableName, numItem, fieldType, fieldName,minLenVarchar)
	sqlStats = append(sqlStats, sqlStat)

	err := executeSQL(conn, sqlStats)
	check(err)
}

func main() {
	var cfg Config
	if _, err := toml.DecodeFile("config.toml", &cfg); err != nil {
		check(err)
		return
	}
	insert(cfg)
}
