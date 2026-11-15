@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo Остановка старых процессов сервера...
taskkill /F /IM main.exe >nul 2>&1
echo Запуск обновленного портала мастерской «Ясна»...
go run main.go
pause
