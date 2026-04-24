create extension if not exists "pgcrypto";

create table if not exists users (
  id uuid primary key,
  auth_user_id uuid not null unique,
  email text,
  full_name text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists workspaces (
  id uuid primary key,
  slug text not null unique,
  name text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists workspace_members (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  role text not null,
  created_at timestamptz not null default now(),
  unique (workspace_id, user_id)
);

create table if not exists marketplaces (
  id uuid primary key,
  code text not null unique,
  name text not null
);

create table if not exists shops (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  marketplace_code text not null,
  shop_key text not null,
  shop_id_external text,
  slug text not null,
  url text,
  name text,
  status text,
  location text,
  shop_type text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (workspace_id, marketplace_code, shop_key)
);

create table if not exists shop_snapshots (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  shop_id uuid not null references shops(id) on delete cascade,
  captured_at timestamptz not null,
  source_type text not null,
  source_capture_id uuid,
  profile_json jsonb not null default '{}'::jsonb,
  reputation_json jsonb not null default '{}'::jsonb,
  catalog_json jsonb not null default '{}'::jsonb,
  performance_json jsonb not null default '{}'::jsonb,
  promotions_json jsonb not null default '{}'::jsonb,
  operations_json jsonb not null default '{}'::jsonb,
  products_json jsonb not null default '[]'::jsonb,
  insights_json jsonb not null default '[]'::jsonb,
  debug_json jsonb not null default '{}'::jsonb,
  is_latest boolean not null default false,
  created_at timestamptz not null default now()
);

create index if not exists idx_shop_snapshots_latest on shop_snapshots (workspace_id, shop_id, is_latest, captured_at desc);
create unique index if not exists uq_shop_snapshots_latest_true on shop_snapshots (workspace_id, shop_id) where is_latest = true;

create table if not exists products (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  shop_id uuid not null references shops(id) on delete cascade,
  marketplace_code text not null,
  product_key text,
  product_id_external text,
  product_identity text not null,
  url text,
  name text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists uq_products_identity
  on products (workspace_id, marketplace_code, shop_id, product_identity);

create table if not exists product_snapshots (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  shop_id uuid not null references shops(id) on delete cascade,
  product_id uuid not null references products(id) on delete cascade,
  captured_at timestamptz not null,
  source_type text not null,
  source_capture_id uuid,
  profile_json jsonb not null default '{}'::jsonb,
  pricing_json jsonb not null default '{}'::jsonb,
  performance_json jsonb not null default '{}'::jsonb,
  inventory_json jsonb not null default '{}'::jsonb,
  promotion_json jsonb not null default '{}'::jsonb,
  review_json jsonb not null default '{}'::jsonb,
  debug_json jsonb not null default '{}'::jsonb,
  is_latest boolean not null default false,
  created_at timestamptz not null default now()
);

create index if not exists idx_product_snapshots_latest on product_snapshots (workspace_id, shop_id, product_id, is_latest, captured_at desc);
create unique index if not exists uq_product_snapshots_latest_true on product_snapshots (workspace_id, product_id) where is_latest = true;

create table if not exists raw_captures (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  marketplace_code text not null,
  shop_key text not null,
  page_url text,
  page_type text,
  source_channel text,
  request_url text,
  http_method text,
  status_code integer,
  operation_name text,
  entity_type text,
  request_headers_json jsonb not null default '{}'::jsonb,
  response_headers_json jsonb not null default '{}'::jsonb,
  request_body_text text,
  response_body_text text,
  request_body_storage_path text,
  response_body_storage_path text,
  request_hash text not null,
  response_hash text not null,
  captured_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_raw_captures_shop_time on raw_captures (workspace_id, marketplace_code, shop_key, captured_at desc);
create index if not exists idx_raw_captures_request_hash on raw_captures (workspace_id, request_hash);

create table if not exists parsed_captures (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  raw_capture_id uuid references raw_captures(id) on delete set null,
  marketplace_code text not null,
  shop_key text not null,
  page_url text,
  page_type text,
  operation_name text,
  entity_type text not null,
  leaf_fields_json jsonb not null default '{}'::jsonb,
  important_fields_json jsonb not null default '{}'::jsonb,
  captured_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_parsed_captures_shop_time on parsed_captures (workspace_id, marketplace_code, shop_key, captured_at desc);

create table if not exists source_templates (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  marketplace_code text not null,
  shop_key text not null,
  template_type text not null,
  fingerprint text not null,
  request_url text not null,
  http_method text not null,
  headers_json jsonb not null default '{}'::jsonb,
  body_text text,
  operation_name text,
  is_active boolean not null default true,
  last_seen_at timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists uq_source_templates_fingerprint
  on source_templates (workspace_id, marketplace_code, shop_key, fingerprint);

create table if not exists ingest_jobs (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  marketplace_code text not null,
  shop_key text not null,
  status text not null,
  items_total integer not null default 0,
  items_processed integer not null default 0,
  error_message text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists analytics_jobs (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  marketplace_code text not null,
  shop_key text not null,
  status text not null,
  reason text,
  error_message text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_shops_workspace_marketplace_shop_key on shops (workspace_id, marketplace_code, shop_key);
create index if not exists idx_products_workspace_shop_marketplace on products (workspace_id, shop_id, marketplace_code);
create index if not exists idx_source_templates_lookup on source_templates (workspace_id, marketplace_code, shop_key, fingerprint);
create index if not exists idx_ingest_jobs_workspace_created_at on ingest_jobs (workspace_id, created_at desc);
create index if not exists idx_analytics_jobs_workspace_created_at on analytics_jobs (workspace_id, created_at desc);

insert into marketplaces (id, code, name)
values
  (gen_random_uuid(), 'tokopedia', 'Tokopedia'),
  (gen_random_uuid(), 'shopee', 'Shopee')
on conflict (code) do nothing;
