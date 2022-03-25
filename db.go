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

func dbFlushMsg(ctx context.Context, msg *traceID.Message) (mustNotify bool, err error) {
	var tx *sql.Tx
	if tx, err = dbi.Begin(); err != nil {
		return
	}
	defer func(tx *sql.Tx, err error) {
		if err != nil {
			_ = tx.Rollback()
		}
	}(tx, err)

	if msg.CheckFlag(traceID.FlagOverwrite) {
		if _, err = tx.ExecContext(ctx, fmtQuery("delete from trace_log where tid = ?"), msg.ID); err != nil {
			return
		}
		if _, err = tx.ExecContext(ctx, fmtQuery("delete from trace_uniq where tid = ?"), msg.ID); err != nil {
			return
		}
	}

	for i := 0; i < len(msg.Rows); i++ {
		row := &msg.Rows[i]
		lo, hi := row.Key.Decode()
		k := fastconv.B2S(msg.Buf[lo:hi])
		lo, hi = row.Value.Decode()
		v := fastconv.B2S(msg.Buf[lo:hi])
		_, err = tx.ExecContext(ctx, fmtQuery("insert into trace_log(tid, svc, thid, rid, ts, lvl, typ, nm, val) values(?, ?, ?, ?, ?, ?, ?, ?, ?)"),
			msg.ID, msg.Service, row.ThreadID, row.RecordID, row.Time, row.Level, row.Type, k, v)
		if err != nil {
			return
		}
	}

	row := tx.QueryRowContext(ctx, fmtQuery("select count(ts) as c from trace_uniq where tid=?"), msg.ID)
	var c int
	if err = row.Scan(&c); c == 0 || err == sql.ErrNoRows {
		mustNotify = true
		if _, err = tx.ExecContext(ctx, fmtQuery("insert into trace_uniq(tid, ts) values(?, ?)"), msg.ID, time.Now().UnixNano()); err != nil {
			return
		}
	}

	err = tx.Commit()
	return
}

func dbListMsg(ctx context.Context, pattern string, limit uint) (r []MessageHeader, err error) {
	if limit == 0 {
		limit = 50
	}
	query := "select * from trace_uniq where tid like ? order by ts desc limit ?"
	pattern = "%" + pattern + "%"

	var rows *sql.Rows
	if rows, err = dbi.QueryContext(ctx, fmtQuery(query), pattern, limit); err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			tid string
			ts  int64
		)
		if err = rows.Scan(&tid, &ts); err != nil {
			return
		}
		r = append(r, MessageHeader{
			ID: tid,
			DT: string(time.Unix(ts/1e9, ts%1e9).AppendFormat(nil, time.RFC3339Nano)),
		})
	}
	return
}

func dbMsgTree(ctx context.Context, id string) (msg *MessageTree, err error) {
	query := "select svc, min(ts) as ts from trace_log where tid=? group by svc order by ts"
	var rows *sql.Rows
	if rows, err = dbi.QueryContext(ctx, fmtQuery(query), id); err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	msg = &MessageTree{ID: id}
	for rows.Next() {
		var (
			svc string
			ts  int64
		)
		if err = rows.Scan(&svc, &ts); err != nil {
			return
		}
		msg.Services = append(msg.Services, MessageService{ID: svc})
		svci := &msg.Services[len(msg.Services)-1]
		svci.Threads = append(svci.Threads, MessageThread{ID: 0})
		thr := &svci.Threads[len(svci.Threads)-1]
		if err = dbWalkThr(ctx, id, svc, thr); err != nil {
			return
		}
	}

	return
}

func dbWalkThr(ctx context.Context, id, svc string, thr *MessageThread) error {
	query := "select id, tid, rid, ts, lvl, typ, nm, val from trace_log where tid=? and svc=? and thid=? order by ts"
	var (
		rows *sql.Rows
		err  error
	)
	if rows, err = dbi.QueryContext(ctx, fmtQuery(query), id, svc, thr.ID); err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	crid := -1
	for rows.Next() {
		var (
			id1, rid, lvl, typ uint
			ts                 int64
			tid, nm, val       string
		)
		if err = rows.Scan(&id1, &tid, &rid, &ts, &lvl, &typ, &nm, &val); err != nil {
			return err
		}

		if crid != int(rid) {
			crid = int(rid)
			thr.Records = append(thr.Records, MessageRecord{ID: rid})
		}
		ri := &thr.Records[len(thr.Records)-1]
		ri.Rows = append(ri.Rows, MessageRow{
			ID:    id1,
			DT:    string(time.Unix(ts/1e9, ts%1e9).AppendFormat(nil, time.RFC3339Nano)),
			Level: traceID.LogLevel(lvl).String(),
			Type:  traceID.EntryType(typ).String(),
			Name:  nm,
			Value: val,
		})

		if typ == uint(traceID.EntryAcquireThread) {
			thid, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return err
			}
			thr.Threads = append(thr.Threads, MessageThread{ID: uint(thid)})
			thr1 := &thr.Threads[len(thr.Threads)-1]
			if err := dbWalkThr(ctx, id, svc, thr1); err != nil {
				return err
			}
		}
	}
	return nil
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
