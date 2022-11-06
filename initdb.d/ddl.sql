create table services
(
    id   serial primary key,
    name varchar(30)
);

create table configs
(
    id         serial primary key,
    service_id int references services (id),
    config     json,
    version    int unique ,
    in_use     bool default false
);