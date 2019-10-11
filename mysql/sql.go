package mysql

import (
	"database/sql"

	"github.com/myadamtest/gobase/logkit"
)

func (cli *Client) doWrite(db *sql.DB, sqlstr string, args ...interface{}) (int64, int64, error) {
	logkit.Debugs("[SQL]", "doWrite", sqlstr, args)
	result, err := db.Exec(sqlstr, args...)
	if err != nil {
		logkit.Error(err.Error())
		return 0, 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logkit.Error(err.Error())
		return 0, 0, err
	}

	num, err := result.RowsAffected()
	if err != nil {
		logkit.Error(err.Error())
		return 0, 0, err
	}

	return id, num, nil
}

// 插入数据
func (cli *Client) Insert(sqlstr string, args ...interface{}) (int64, error) {
	db, err := cli.GetWrite()
	if err != nil {
		logkit.Error(err.Error())
		return 0, err
	}

	id, _, err := cli.doWrite(db, sqlstr, args...)

	return id, err
}

// 更新数据
func (cli *Client) Update(sqlstr string, args ...interface{}) (int64, error) {
	db, err := cli.GetWrite()
	if err != nil {
		logkit.Error(err.Error())
		return 0, err
	}

	_, num, err := cli.doWrite(db, sqlstr, args...)

	return num, err
}

// 删除数据
func (cli *Client) Delete(sqlstr string, args ...interface{}) (int64, error) {
	db, err := cli.GetWrite()
	if err != nil {
		logkit.Error(err.Error())
		return 0, err
	}

	_, num, err := cli.doWrite(db, sqlstr, args...)

	return num, err
}

func (cli *Client) readRow(db *sql.DB, sqlstr string, args ...interface{}) (map[string]string, error) {
	logkit.Debugs("[SQL]", "readRow", sqlstr, args)
	rows, err := db.Query(sqlstr, args...)

	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	ret := make(map[string]string)

	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		var value string
		for i, col := range values {
			if col == nil {
				value = "" //把数据表中所有为null的地方改成“”
			} else {
				value = string(col)
			}

			ret[columns[i]] = value
		}

		break
	}

	rows.Close()

	return ret, err
}

func (cli *Client) readRows(db *sql.DB, sqlstr string, args ...interface{}) ([]map[string]string, error) {
	logkit.Debugs("[SQL]", "readRows", sqlstr, args)
	rows, err := db.Query(sqlstr, args...)

	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	var rets = make([]map[string]string, 0)

	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		var ret = make(map[string]string) //这里要注意(对语法的理解)

		var value string
		for i, col := range values {
			if col == nil {
				value = "" //把数据表中所有为null的地方改成“”
			} else {
				value = string(col)
			}

			ret[columns[i]] = value
		}

		rets = append(rets, ret)
	}

	return rets, err
}

// 取一行数据
func (cli *Client) FetchRow(sqlstr string, args ...interface{}) (map[string]string, error) {
	db, err := cli.GetRead()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	return cli.readRow(db, sqlstr, args...)
}

// 取多行数据
func (cli *Client) FetchRows(sqlstr string, args ...interface{}) ([]map[string]string, error) {
	db, err := cli.GetRead()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	return cli.readRows(db, sqlstr, args...)
}

// 从master取一行数据
func (cli *Client) FetchRowForMaster(sqlstr string, args ...interface{}) (map[string]string, error) {
	db, err := cli.GetWrite()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	return cli.readRow(db, sqlstr, args...)
}

// 从master取多行数据
func (cli *Client) FetchRowsForMaster(sqlstr string, args ...interface{}) ([]map[string]string, error) {
	db, err := cli.GetWrite()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}

	return cli.readRows(db, sqlstr, args...)
}

func (cli *Client) GetWrite() (*sql.DB, error) {
	if cli.writeDB == nil {
		return nil, ErrNoUseableDB
	}
	db, err := cli.writeDB.Get()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	return db.DB, nil
}

func (cli *Client) GetRead() (*sql.DB, error) {
	if cli.readDB == nil {
		return cli.GetWrite()
	}
	db, err := cli.readDB.Get()
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	return db.DB, nil
}
