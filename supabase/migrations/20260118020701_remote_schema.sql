drop extension if exists "pg_net";

alter table "mcpist"."users" add column "role" text not null default 'user'::text;


