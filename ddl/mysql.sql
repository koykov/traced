create table trace_uniq
(
    tid varchar(128) not null,
    ts  bigint       not null,
    constraint trace_uniq_trace_id_uindex
        unique (tid)
);

create table trace_log
(
    id   int auto_increment
        primary key,
    tid  varchar(128)  not null,
    svc  varchar(128)  null,
    stg  varchar(128)  null,
    mid  int default 0 null,
    thid int default 0 not null,
    rid  int default 0 not null,
    lvl  smallint      null,
    typ  smallint      null,
    nm   varchar(512)  null,
    val  blob          null,
    ts   bigint        not null
);

create index trace_log_tid_index
    on trace_log (tid);
