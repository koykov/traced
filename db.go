package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/koykov/fastconv"
	"github.com/koykov/traceID"
	_ "github.com/lib/pq"
)

var dbi *sql.DB

func dbConnect(addr string) (err error) {
	var di int
	if di = strings.Index(addr, "://"); di == -1 {
		err = fmt.Errorf("couldn't get driver name from DSN '%s'", addr)
		return
	}
	drv := addr[:di]
	if len(drv) == 0 {
		return errors.New("empty DB driver")
	}
	addr = addr[di+3:]
	if dbi, err = sql.Open(drv, addr); err != nil {
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
		_, err = tx.ExecContext(ctx, "insert into trace_log(tid, svc, thid, ts, lvl, typ, nm, val) values($1, $2, $3, $4, $5, $6, $7, $8)",
			msg.ID, msg.Service, row.ThreadID, row.Time, row.Level, row.Type, k, v)
		if err != nil {
			return
		}
	}

	row := tx.QueryRowContext(ctx, "select count(ts) as c from trace_uniq where tid=$1", msg.ID)
	var c int
	if err = row.Scan(&c); err == sql.ErrNoRows {
		mustNotify = true
		err = nil
	}

	_, err = tx.ExecContext(ctx, "insert into trace_uniq(tid, ts) values($1, $2)", msg.ID, time.Now().UnixNano())

	err = tx.Commit()
	return
}

func dbClose() error {
	if dbi == nil {
		return nil
	}
	return dbi.Close()
}
