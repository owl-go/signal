package mysql

import (
	"database/sql"
	"fmt"

	"log"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type MysqlDriver struct {
	MysqlConfig
	DbCon *sql.DB
}

func IsRecordNotFoundError(err error) bool {
	if err != nil && err.Error() == "sql: no rows in result set" {
		return true
	} else {
		return false
	}
}

func NewMysqlDriver(c MysqlConfig) *MysqlDriver {
	dbDriver := &MysqlDriver{
		MysqlConfig: c,
	}
	dbConn, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql", dbDriver.Username, dbDriver.Password, dbDriver.Host, dbDriver.Port))
	if err != nil {
		log.Fatalf(err.Error())
	}
	_, err = dbConn.Exec("CREATE DATABASE IF NOT EXISTS " + dbDriver.Database + " DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_general_ci;")
	if err != nil {
		log.Fatalf(err.Error())

	}
	dbDriver.Open()
	return dbDriver
}

//Open 连接数据库
func (dbDriver *MysqlDriver) Open() {
	dbConn, _ := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbDriver.Username, dbDriver.Password, dbDriver.Host, dbDriver.Port, dbDriver.Database))
	dbConn.SetMaxOpenConns(200)
	dbConn.SetMaxIdleConns(100)
	dbConn.Ping()
	dbDriver.DbCon = dbConn
}

//关闭数据库
func (dbDriver *MysqlDriver) Close() {
	dbDriver.DbCon.Close()
}

//插入数据
func (dbDriver *MysqlDriver) Insert(table string, keys []string, values [][]interface{}) (int64, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf(err.(string))
		}
	}()
	preSql := "INSERT INTO  `" + table + "` "
	keysSql := ""
	keyLen := len(keys)
	valueSql := ""

	for k, v := range keys {
		if k == 0 {
			keysSql = "(`" + v + "`,"
			valueSql = "(?,"
		} else if k == keyLen-1 {
			keysSql += "`" + v + "`)"
			valueSql += "?)"
		} else {
			keysSql += "`" + v + "`,"
			valueSql += "?,"
		}
	}
	preSql = preSql + keysSql + " VALUES "

	tempValue := make([]interface{}, 0)

	totalValuesSql := ""

	for k, v := range values {
		if k == 0 {
			totalValuesSql = valueSql
		} else {
			totalValuesSql = totalValuesSql + "," + valueSql
		}
		for _, v0 := range v {
			tempValue = append(tempValue, v0)
		}
	}
	insertSql := preSql + totalValuesSql

	smts, err := dbDriver.DbCon.Prepare(insertSql)
	if err != nil {
		return 0, err
	}
	defer smts.Close()

	res, err := smts.Exec(tempValue[:]...)
	if err != nil {
		return 0, err
	}
	nums, err := res.RowsAffected()
	return nums, err
}

//执行原生的sql语句
func (dbDriver *MysqlDriver) Exec(sql string, args ...interface{}) (int64, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf(err.(string))
		}
	}()
	smts, err := dbDriver.DbCon.Prepare(sql)
	defer smts.Close()
	if err != nil {
		return 0, err
	}
	res, err := smts.Exec(args...)
	if err != nil {
		return 0, err
	}
	nums, err := res.RowsAffected()
	return nums, err
}

////查找一条记录
func (dbDriver *MysqlDriver) FindOne(sql string, args ...interface{}) (row *sql.Row) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf(err.(string))
		}
	}()
	row = dbDriver.DbCon.QueryRow(sql, args...)
	return
}

//查询结果集
func (dbDriver *MysqlDriver) Find(sql string, args ...interface{}) (rows *sql.Rows, err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf(err.(string))
		}
	}()
	rows, err = dbDriver.DbCon.Query(sql, args...)
	return
}

//删除
func (dbDriver *MysqlDriver) Delete(sql string, args ...interface{}) (int64, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf(err.(string))
		}
	}()
	smts, err := dbDriver.DbCon.Prepare(sql)
	defer smts.Close()
	if err != nil {
		return 0, err
	}

	res, err := smts.Exec(args...)
	if err != nil {
		return 0, err
	}
	nums, _ := res.RowsAffected()
	return nums, nil
}

//更新
func (dbDriver *MysqlDriver) Update(sql string, args ...interface{}) (int64, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf(err.(string))
		}
	}()
	smts, err := dbDriver.DbCon.Prepare(sql)
	defer smts.Close()
	if err != nil {
		return 0, err
	}
	res, err := smts.Exec(args...)
	if err != nil {
		return 0, err
	}
	nums, _ := res.RowsAffected()
	return nums, nil
}
