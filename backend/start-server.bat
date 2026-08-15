@echo off
cd /d %~dp0
set DB_HOST=127.0.0.1
set DB_PORT=3306
set DB_USER=root
set DB_PASS=123456
set DB_NAME=myworld
set PORT=8080
go run . 
