# meet-up-bot

A Telegram bot for creating and joining event lobbies. Users create lobbies with
a **name**, **place**, **time**, and an optional **Telegram chat link**. Lobbies
are either **public** (anyone can join instantly) or **private** (joining
requires the creator's approval).

## Stack

- [go-telegram/bot](https://github.com/go-telegram/bot) — Telegram Bot API client
- **PostgreSQL** with [sqlc](https://sqlc.dev)-generated, type-safe queries (pgx/v5)
- [golang-migrate](https://github.com/golang-migrate/migrate) migrations, embedded into a `cmd/migrate` binary
- [zap](https://github.com/uber-go/zap) structured logging
- Docker + docker-compose + Makefile for local orchestration

## Layout

```
cmd/bot        bot entrypoint
cmd/migrate    migration runner (up/down/drop/force/version)
db/migrations  SQL migrations (embedded) + schema source for sqlc
db/queries     SQL queries compiled by sqlc
internal/config, logger, storage, telegram
```

## Bot commands

| Command      | What it does                                                          |
|--------------|----------------------------------------------------------------------|
| `/create`    | Wizard to set up a new lobby                                          |
| `/lobbies`   | Browse & join upcoming lobbies (private ones require approval)        |
| `/mylobbies` | Manage your lobbies: edit fields, manage/ban members, approve requests |
| `/settings`  | Choose your language (English / Русский)                             |
| `/cancel`    | Abort the current wizard                                              |
| `/help`      | Show help                                                            |

Lobbies can be **edited** after creation (e.g. add the chat link later) from
`/mylobbies` → 🔍 Details → ✏️ Edit; every approved participant is then notified
of the change.

**Events** carry a structured location — country, city, and an optional address
— plus a time that is interpreted and displayed in each user's own timezone.

**`/settings`** stores per-user preferences (in the `users` table):

- **Language** — English / Русский; all bot text is localized.
- **Timezone** — an IANA name (e.g. `Europe/Kyiv`); event times you enter are
  read in your timezone and shown to each viewer in theirs.
- **City** — surfaces nearby lobbies: `/lobbies` filters to your city.
- **Time filter** — show lobbies happening within the next day / week / month,
  or any time.

**`/lobbies`** lists upcoming lobbies soonest-first, filtered by your city and
time-window settings, paginated 10 per page with ⬅️/➡️ navigation.

**Invite links** — from a lobby's Details, an admin or member can generate a
`🔗 Invite link` (`https://t.me/<bot>?start=join_<id>`). Tapping it opens the
bot and joins the tapper (public) or files a join request (private), exactly
like the Join button.

## Quick start (full Docker stack)

1. `cp .env.example .env` and set `TELEGRAM_BOT_TOKEN` (from [@BotFather](https://t.me/BotFather)).
2. `make up`

`make up` builds the image and starts Postgres, runs migrations as a one-shot
container, then starts the bot — and tails its logs. Stop with `make down`
(`make clean` also drops the DB volume).

## Local development (Go on host, Postgres in Docker)

```sh
cp .env.example .env      # set TELEGRAM_BOT_TOKEN
make dev                  # start DB, migrate, run the bot on your machine
```

Other useful targets: `make sqlc` (regenerate queries), `make migrate-up` /
`make migrate-down`, `make build`. Run `make help` for the full list.
