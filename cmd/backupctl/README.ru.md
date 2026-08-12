# backupctl

Консольная утилита для настройки резервного копирования БД через
[restic](https://restic.net/). Сервер (любая машина с MySQL/PostgreSQL)
делает дампы БД и хранит их локально в restic-репозитории. Клиент (Mac
или другой сервер) периодически забирает этот репозиторий к себе по
`rsync` поверх SSH. Подключение — только по SSH-ключу, у служебного
пользователя `backup-user` вообще нет пароля.

English version: `backupctl --info` (default) or [README.md](https://github.com/Solo-it/backupctl/blob/main/cmd/backupctl/README.md).

> На русский пока переведены только `--info`/`--help` (флаг `-lang=ru`).
> Остальные сообщения программы (шаги `--init-server`, ошибки) — на
> английском. Хотите помочь с переводом — см. раздел «Contributing» в
> английском README.

## Установка

```bash
# Debian/Ubuntu (apt-репозиторий)
echo "deb [trusted=yes] https://solo-it.github.io/backupctl/apt stable main" | sudo tee /etc/apt/sources.list.d/backupctl.list
sudo apt update && sudo apt install backupctl

# Homebrew (macOS/Linux)
brew install Solo-it/tap/backupctl

# Go
go install github.com/Solo-it/backupctl/cmd/backupctl@latest

# или скачать бинарник/.deb/.rpm со страницы Releases
```

## Команды

| Команда | Где выполняется | Что делает |
|---|---|---|
| `--gen-config` | сервер | генерирует серверный `backup.yaml` |
| `--gen-config-client` | клиент | генерирует клиентский `backup.yaml` |
| `--init-server` | сервер (root) | restic, инструменты дампа, `backup-user` без пароля, репозиторий, дамп БД по расписанию через systemd timer |
| `--add-client-key` | сервер (root) | добавляет публичный SSH-ключ клиента в `authorized_keys` `backup-user` |
| `--init-client-pm2` | клиент (Mac) | SSH-ключ + pull-скрипт + `pm2` с `cron_restart` |
| `--init-client-launchd` | клиент (Mac) | SSH-ключ + pull-скрипт + `launchd` plist |
| `--init-client-systemd` | клиент (Linux) | SSH-ключ + pull-скрипт + systemd timer |
| `--init-client-task-scheduler` | клиент (Windows) | SSH-ключ + pull-скрипт (PowerShell) + Планировщик заданий (schtasks) |
| `--version` / `-v` | — | версия |
| `--help` / `-h` | — | справка по флагам |
| `--info` | — | этот README, встроенный в бинарник |

Все команды, кроме `--gen-config`/`--gen-config-client`, читают конфиг из
`-config backup.yaml` (путь можно переопределить флагом). У `--gen-config`
и `--gen-config-client` все флаги опциональны — есть разумные значения по
умолчанию, минимально можно вообще без флагов.

## Пошагово

### 1. На сервере — сгенерировать конфиг и настроить restic

```bash
backupctl --gen-config -out=backup.yaml \
  -restic-repo=/home/backup-user/backups/restic-repo \
  -backup-user=backup-user \
  -server-schedule=02:00
```

Открыть `backup.yaml` и вписать реальные имена БД в `databases:` —
сгенерированный файл содержит только заглушку `example_db` с комментарием,
какие `type:` вообще поддерживаются (сейчас `mysql`, `postgres`). Это
единственное, что нужно доправить руками.

`restic.password_file` в конфиге — это путь, где будет лежать пароль
restic-репозитория. Сам файл на этом шаге ещё не существует.

```bash
sudo backupctl --init-server
```

(`-config` по умолчанию `backup.yaml` в текущей директории — задавайте
флагом только если файл называется иначе.)

Идемпотентно: повторный запуск не ломает уже настроенный сервер.
`schedule` в `backup.yaml` — это именно UTC, независимо от таймзоны
сервера (`OnCalendar` в systemd unit генерируется с явным суффиксом
`UTC`) — проверить фактическое время следующего запуска:
`systemctl list-timers restic-backup.timer`.

Если `restic` не найден в PATH, `--init-server` сам попробует поставить
его через первый найденный пакетный менеджер (`apt-get`, `dnf`, `yum`,
`apk`, `pacman`, `brew`); если ни одного нет — остановится с понятной
ошибкой и ссылкой на ручную установку. Так же заранее проверяется, что
для каждого `type:` из `databases:` в PATH есть нужный инструмент дампа
(`mysqldump` для `mysql`, `pg_dump` для `postgres`).

**Про пароль restic — это важно, если серверов несколько:**

- **Первый запуск `--init-server` для этого сервера, ничего не задано** —
  генерируется случайный пароль, пишется в `password_file` с правами
  `600` (владелец root), и **один раз печатается в консоль**. Это
  единственный момент, когда пароль видно — сохраните его в свой
  менеджер паролей сразу. Если сервер и диск будут потеряны, у клиента
  останется только зашифрованная копия репозитория (см. шаг 2), а без
  этого пароля расшифровать её нельзя ничем.
- **Хотите один и тот же пароль на нескольких серверах** (проще
  управлять, но компрометация одного пароля = компрометация всех
  репозиториев — компромисс между удобством и изоляцией) — передайте его
  флагом при первом запуске на каждом следующем сервере:
  ```bash
  sudo backupctl --init-server -restic-password="<пароль-с-первого-сервера>"
  ```
  Работает только пока `password_file` ещё не существует — существующий
  файл `--init-server` никогда не перезаписывает.
- **Файл уже существует** (неважно как появился) — `--init-server` его
  не трогает и не печатает, просто сообщает, что пароль уже есть.

После успешного `--init-server` команда сама печатает список следующих
шагов (проверить разовый прогон, настроить клиента, разрешить его ключ) —
дальше можно ориентироваться по этому выводу, не только по README.

### 2. На клиенте — сгенерировать конфиг, ключ и планировщик

```bash
backupctl --gen-config-client -out=backup-client.yaml \
  -client-remote=backup-user@<ip-сервера> \
  -client-remote-repo=/home/backup-user/backups/restic-repo \
  -client-schedule=04:00

backupctl --init-client-launchd -config=backup-client.yaml   # macOS
# или --init-client-pm2 / --init-client-systemd / --init-client-task-scheduler
```

`--init-client-task-scheduler` (Windows) дополнительно требует `rsync` в
PATH — например, из WSL, cwRsync или Git for Windows (Git Bash кладёт
`rsync.exe` в PATH).

Команда сама сгенерирует `ed25519` SSH-ключ (если его ещё нет) и напечатает
публичный ключ вместе с готовой командой для сервера.

### 3. На сервере — разрешить ключ клиента

```bash
sudo backupctl --add-client-key -pubkey="ssh-ed25519 AAAA... backupctl-client"
```

Идемпотентно — повторное добавление того же ключа не дублирует запись в
`authorized_keys`. Ключ ограничен опциями
`no-pty,no-port-forwarding,no-X11-forwarding,no-agent-forwarding` — им
можно только гонять `rsync`, не более.

### 4. Проверить забор бэкапа руками

```bash
~/.local/bin/backup-pull.sh        # launchd
/usr/local/bin/backup-pull.sh      # pm2 / systemd
```

```powershell
& "$env:USERPROFILE\backupctl\backup-pull.ps1"   # Task Scheduler (Windows)
```

## Добавили/убрали БД на сервере — как обновить

Отредактируйте `databases:` в `backup.yaml` (добавьте или уберите
`type`/`names`) и запустите `--init-server` ещё раз:

```bash
sudo backupctl --init-server -config=backup.yaml
```

Ничего разрушительного не произойдёт — `backup-user`, restic-репозиторий
и пароль уже существуют, эти шаги просто подтвердят, что всё на месте.
А вот `/root/backup.sh` (список команд дампа) и systemd unit/timer
перезаписываются каждый раз безусловно — так что новая БД попадёт в
следующий же прогон по расписанию. Отдельной команды "обновить" не
нужно, `--init-server` для этого и предназначен.

## Формат backup.yaml

```yaml
restic:
  repo: /home/backup-user/backups/restic-repo
  password_file: /root/.restic-env    # создаётся автоматически при --init-server

backup_user: backup-user

databases:           # поддерживаемые type: mysql, postgres
  - type: mysql
    names:
      - app_db
      - shop_db
  - type: postgres
    names:
      - analytics

schedule: "02:00"   # UTC, время серверного дампа+бэкапа

client:
  remote: backup-user@192.168.1.10
  remote_repo: /home/backup-user/backups/restic-repo
  local_path: ~/backups/mysql-restic-repo
  schedule: "04:00"   # время, когда клиент забирает бэкап
```

## Тесты

```bash
go test ./...                    # юнит-тесты, чистая логика
go test -tags=integration ./...  # + реальные ssh-keygen/restic/rsync
```

`useradd`/`chown`/`systemctl` реальными тестами не покрыты (нужен root и
меняют систему) — их поведение проверяется только вручную, на тестовой VM.

## Версионность

Версия берётся из git-тега `vX.Y.Z`, релиз собирает
[GoReleaser](https://goreleaser.com/) на каждый такой тег:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

## Лицензия

[MIT](../../LICENSE)
