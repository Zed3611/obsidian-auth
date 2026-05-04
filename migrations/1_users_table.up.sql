begin;

create extension citext;

create table users(
    id bigserial primary key,
    email citext unique not null,
    password_hash text not null,
    created_at timestamp not null default now(),
    updated_at timestamp not null default now()
);

create function update_updated_at_column()
returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language 'plpgsql';

create trigger set_updated_at
before update on users
for each row
execute function update_updated_at_column();

commit;