begin;

drop trigger set_updated_at on users;

drop function update_updated_at_column;

drop table users;

drop extension citext;

commit;