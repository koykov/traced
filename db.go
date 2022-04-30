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

func dbTraceList(ctx context.Context, pattern string, limit uint) (r []TraceHeader, err error) {
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
		r = append(r, TraceHeader{
			ID: tid,
			DT: string(time.Unix(ts/1e9, ts%1e9).AppendFormat(nil, time.RFC3339Nano)),
		})
	}
	return
}

func dbTraceTree(ctx context.Context, id string) (msg *TraceTree, err error) {
	query := "select svc, min(ts) as ts from trace_log where tid=? group by svc order by ts"
	var rows *sql.Rows
	if rows, err = dbi.QueryContext(ctx, fmtQuery(query), id); err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	msg = &TraceTree{ID: id}
	for rows.Next() {
		var (
			svc string
			ts  int64
		)
		if err = rows.Scan(&svc, &ts); err != nil {
			return
		}
		msg.Services = append(msg.Services, TraceService{ID: svc})
		svci := &msg.Services[len(msg.Services)-1]
		if err = dbWalkSvc(ctx, id, svci); err != nil {
			return
		}
	}

	return
}

func dbWalkSvc(ctx context.Context, id string, svc *TraceService) error {
	query := "select id, tid, thid, rid, ts, lvl, typ, nm, val from trace_log where tid=? and svc=? order by ts"
	var (
		rows *sql.Rows
		err  error
	)
	if rows, err = dbi.QueryContext(ctx, fmtQuery(query), id, svc.ID); err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	crid := -1
	recIdx := make(map[int]*TraceRecord)
	for rows.Next() {
		var (
			id1, thid, rid, lvl, typ uint
			ts                       int64
			tid, nm, val             string
		)
		if err = rows.Scan(&id1, &tid, &thid, &rid, &ts, &lvl, &typ, &nm, &val); err != nil {
			return err
		}

		if et := traceID.EntryType(typ); et == traceID.EntryAcquireThread || et == traceID.EntryReleaseThread {
			if et == traceID.EntryAcquireThread {
				svc.Threads++
			}
			var thid1 uint64
			if thid1, err = strconv.ParseUint(val, 10, 64); err != nil {
				return err
			}
			rec := TraceRecord{
				ID:       0,
				ThreadID: thid,
				ChildID:  uint(thid1),
				Thread: &TraceRow{
					ID:    id1,
					DT:    string(time.Unix(ts/1e9, ts%1e9).AppendFormat(nil, time.RFC3339Nano)),
					Level: traceID.Level(lvl).String(),
					Type:  traceID.EntryType(typ).String(),
				},
			}
			svc.Records = append(svc.Records, rec)
			continue
		}

		if crid != int(rid) {
			crid = int(rid)
			if _, ok := recIdx[crid]; !ok {
				svc.Records = append(svc.Records, TraceRecord{
					ID:       rid,
					ThreadID: thid,
				})
				recIdx[crid] = &svc.Records[len(svc.Records)-1]
			}
		}
		ri := recIdx[crid]
		ri.Rows = append(ri.Rows, TraceRow{
			ID:    id1,
			DT:    string(time.Unix(ts/1e9, ts%1e9).AppendFormat(nil, time.RFC3339Nano)),
			Level: traceID.Level(lvl).First().String(),
			Type:  traceID.EntryType(typ).String(),
			Name:  nm,
			Value: val,
		})
	}
	if svc.Threads == 0 {
		svc.Threads = 1
	}

	for i := 0; i < len(svc.Records); i++ {
		rec := &svc.Records[i]
		if len(rec.Rows) == 0 {
			continue
		}
		row := &rec.Rows[0]
		if strings.Index(row.Value, "{") == -1 || strings.Index(row.Value, "}") == -1 {
			continue
		}
		applyPlaceholders(rec)
	}
	return nil
}

func dbTraceRecord(ctx context.Context, id string) (rec *TraceRecord, err error) {
	id64, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return
	}

	row := dbi.QueryRowContext(ctx, fmtQuery("select tid, svc, thid, rid from trace_log where id=?"), id64)
	var (
		tid  string
		svc  string
		thid uint
		rid  int
	)
	if err = row.Scan(&tid, &svc, &thid, &rid); len(tid) == 0 || len(svc) == 0 || rid == 0 || err == sql.ErrNoRows {
		return
	}

	query := "select id, thid, ts, lvl, typ, nm, val from trace_log where tid=? and svc=? and rid=? order by ts"
	var rows *sql.Rows
	if rows, err = dbi.QueryContext(ctx, fmtQuery(query), tid, svc, rid); err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	rec = &TraceRecord{
		ID:       uint(id64),
		ThreadID: thid,
	}
	for rows.Next() {
		var (
			id1, thid, lvl, typ uint
			ts                  int64
			nm, val             string
		)
		if err = rows.Scan(&id1, &thid, &ts, &lvl, &typ, &nm, &val); err != nil {
			return
		}
		et := traceID.EntryType(typ)
		if et == traceID.EntryChapter || et == traceID.EntryAcquireThread || et == traceID.EntryReleaseThread {
			rec.ThreadID = thid
			rec.Rows = append(rec.Rows, TraceRow{
				ID:     id1,
				DT:     string(time.Unix(ts/1e9, ts%1e9).AppendFormat(nil, time.RFC3339Nano)),
				Levels: splitLevelLabels(traceID.Level(lvl)),
				Type:   traceID.EntryType(typ).String(),
				Name:   nm,
				Value:  val,
			})
		} else {
			rec.Rows = append(rec.Rows, TraceRow{
				ID:     id1,
				Levels: splitLevelLabels(traceID.Level(lvl)),
				Name:   nm,
				Value:  val,
			})
		}
	}
	applyPlaceholders(rec)

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
