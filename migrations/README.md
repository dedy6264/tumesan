# PostgreSQL schema

`001_schema.sql` is the complete schema and demo catalog seed for a fresh PostgreSQL 15+ application database. It enables `pgcrypto` for `gen_random_uuid()` compatibility and replaces the former two-step migration sequence.

Run it once against an empty database:

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/001_schema.sql
```

The script is transactional. It does not drop existing tables or data. The API uses a single default price group for guests and new users; set the single-provider credentials in `provider_config` before enabling provider transactions. Product cost, provider fees, and provider response audit columns are internal database fields.

