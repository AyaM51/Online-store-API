@echo off
set DATABASE_HOST=127.0.0.1
set DATABASE_USER=your_user
set DATABASE_PASSWORD=your_password
set DATABASE_PORT=5432
set DATABASE_NAME=toystore

set REDIS_HOST=127.0.0.1
set REDIS_PORT=6379

:: Запуск Go-приложения
go run main.go
