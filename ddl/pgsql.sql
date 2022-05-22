create table trace_uniq
(
    tid varchar(128) not null,
    ts  bigint
);

create unique index trace_uniq_tid_uindex
    on trace_uniq (tid);

create table trace_log
(
    id   serial
        constraint trace_log_pk
        primary key,
    tid  varchar(128)      not null,
    svc  varchar(128),
    stg  varchar(128),
    mid  integer default 0,
    thid integer default 0 not null,
    rid  integer default 0 not null,
    lvl  smallint,
    typ  smallint,
    nm   varchar(512),
    val  bytea,
    ts   bigint
);

create index trace_log_tid_index
    on trace_log (tid);
