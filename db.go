package main

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/koykov/fastconv"
	"github.com/koykov/traceID"
	_ "github.com/lib/pq"
)

const (
	defaultQPT = "?"
)

var (
	dbi *sql.DB
	qpt string
)

func dbConnect(dbc *DBConfig) (err error) {
	if len(dbc.Driver) == 0 {
		return errors.New("empty DB driver")
	}
	if len(dbc.Driver) == 0 {
		return errors.New("empty DSN string")
	}
	if qpt = dbc.QPT; len(qpt) == 0 {
		qpt = defaultQPT
	}
	if dbi, err = sql.Open(dbc.Driver, dbc.DSN); err != nil {
		return
	}
	if err = dbi.Ping(); err != nil {
		return
	}
	return
}

func dbFlushMsg(msg *traceID.Message, ctx context.Context) (mustNotify bool, err error) {
	var tx *sql.Tx
	if tx, err = dbi.Begin(); err != nil {
		return
	}
	defer func(tx *sql.Tx, err error) {
		if err != nil {
			_ = tx.Rollback()
		}
	}(tx, err)

	for i := 0; i < len(msg.Rows); i++ {
		row := &msg.Rows[i]
		lo, hi := row.Key.Decode()
		k := fastconv.B2S(msg.Buf[lo:hi])
		lo, hi = row.Value.Decode()
		v := fastconv.B2S(msg.Buf[lo:hi])
		_, err = tx.ExecContext(ctx, fmtQuery("insert into trace_log(tid, svc, thid, ts, lvl, typ, nm, val) values(?, ?, ?, ?, ?, ?, ?, ?)"),
			msg.ID, msg.Service, row.ThreadID, row.Time, row.Level, row.Type, k, v)
		if err != nil {
			return
		}
	}

	row := tx.QueryRowContext(ctx, fmtQuery("select count(ts) as c from trace_uniq where tid=?"), msg.ID)
	var c int
	if err = row.Scan(&c); err == sql.ErrNoRows {
		mustNotify = true
		err = nil
	}

	_, err = tx.ExecContext(ctx, fmtQuery("insert into trace_uniq(tid, ts) values(?, ?)"), msg.ID, time.Now().UnixNano())

	err = tx.Commit()
	return
}

func dbClose() error {
	if dbi == nil {
		return nil
	}
	return dbi.Close()
}

func fmtQuery(query string) string {
	if !strings.Contains(query, "?") || qpt == defaultQPT {
		return query
	}
	buf := make([]byte, 0, len(query)*2)
	p := strings.Index(query, "?")
	var i int
	for {
		buf = append(buf, query[:p]...)
		if len(qpt) == 2 && qpt[1] == 'N' {
			i++
			buf = append(buf, qpt[0])
			buf = strconv.AppendInt(buf, int64(i), 10)
		} else {
			buf = append(buf, qpt...)
		}
		query = query[p+1:]
		if p = strings.Index(query, "?"); p == -1 {
			break
		}
	}
	buf = append(buf, query...)
	return fastconv.B2S(buf)
}
