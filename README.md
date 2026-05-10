# Deadline Bot

A Telegram bot for tracking university deadlines. Built as an MVP for the Software Development Case Study course at AITU.

## What it does

- Add deadlines through a step-by-step conversational flow
- Pick dates via an inline calendar - no manual typing
- List all upcoming deadlines sorted by date
- Update or delete deadlines through button menus
- Sends reminders at 24h and 3h before each deadline

## Stack

- Go 1.25
- PostgreSQL
- [go-telegram-bot-api v5](https://github.com/go-telegram-bot-api/telegram-bot-api)

---

## Local setup

### 1. Get a bot token

1. Open Telegram, find `@BotFather`
2. Send `/newbot`, pick a name and username (must end in `bot`)
3. Copy the token

### 2. Install Go

Download from https://go.dev/dl/ - version 1.25 or newer.

### 3. Start PostgreSQL

The easiest way is Docker:

```bash
docker run --name pg -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=deadlines -p 5432:5432 -d postgres:16
```

Or if you have PostgreSQL installed locally, create the database in pgAdmin or via CLI:

```bash
psql -U postgres -c "CREATE DATABASE deadlines;"
```

### 4. Configure environment

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

```env
TELEGRAM_BOT_TOKEN=your_token_here
DATABASE_URL=postgres://postgres:your_password@localhost:5432/deadlines?sslmode=disable
```

### 5. Run

```bash
go mod tidy
go run .
```

You should see `Bot started: @your_bot_name`. The database table is created automatically on first run.

---

## Commands

| Command | Description |
|---|---|
| `/start` | Show welcome message and command list |
| `/add` | Add a new deadline (step-by-step) |
| `/list` | Show all upcoming deadlines sorted by date |
| `/update` | Edit a deadline (pick from list, then change text or date) |
| `/delete` | Deletes your deadlines (asks for confirmation) |

## How it works

**`/add`**
1. Bot asks for the deadline description
2. You type it (e.g. "Math assignment")
3. Bot shows an inline calendar - tap a day to confirm

**`/update`**
1. Bot shows your deadlines as buttons - tap the one to edit
2. Choose what to change: text or date
3. Type new text or pick a new date from the calendar

**`/delete`**
1. Bot asks for confirmation with Yes / Cancel buttons

---

## Reminders

The bot checks every hour and sends reminders:
- **24 hours** before the deadline
- **3 hours** before the deadline

Each reminder is sent once. If delivery fails, the bot retries up to 3 times and will try again on the next hourly check.

---

## Database schema

```sql
CREATE TABLE deadlines (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL,
    text         TEXT        NOT NULL,
    deadline_at  TIMESTAMPTZ NOT NULL,
    reminded_24h BOOLEAN     DEFAULT FALSE,
    reminded_3h  BOOLEAN     DEFAULT FALSE,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Notes

- No authentication - Telegram `user_id` is the only identifier
- Past deadlines are never shown in `/list`
- Past days in the calendar are shown as `·` and are not selectable
- `/delete` removes your deadlines (no undo)
- Updating a deadline resets reminder flags so you get reminders again
