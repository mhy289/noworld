@echo off
cd /d %~dp0
set DB_HOST=127.0.0.1
set DB_PORT=3306
set DB_USER=root
set DB_PASS=123456
set DB_NAME=myworld
set PORT=8080

echo [start] 检测并清理占用端口 %PORT% 的旧进程...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%PORT% ^| findstr LISTENING') do (
    echo [start] 找到占用 %PORT% 的进程 PID=%%a，正在结束...
    taskkill /F /PID %%a
)
timeout /t 1 /nobreak >nul

go run . 
