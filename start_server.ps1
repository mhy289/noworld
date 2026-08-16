# 数据库连接信息（环境变量优先级最高）
$env:DB_HOST = '127.0.0.1'
$env:DB_PORT = '3306'
$env:DB_USER = 'mhy'
$env:DB_PASS = '123456'
$env:DB_NAME = 'myworld'

# 连接池参数（可选，未设置时使用内置默认值：10/5/1h）
# $env:DB_MAX_OPEN_CONNS = '20'
# $env:DB_MAX_IDLE_CONNS = '10'
# $env:DB_CONN_MAX_LIFETIME = '30m'

# 可选：改用 JSON 配置文件（取消注释并复制 db.config.example.json 为 db.config.json）
# $env:DB_CONFIG_FILE = 'd:\goproject\noworld\db.config.json'

$env:PORT = '8080'
$proc = Start-Process -FilePath 'go' -ArgumentList 'run', '.' -WorkingDirectory 'd:\goproject\noworld' -RedirectStandardOutput 'd:\goproject\noworld\server-out.log' -RedirectStandardError 'd:\goproject\noworld\server-err.log' -WindowStyle Hidden -PassThru
Write-Host ("STARTED PID=" + $proc.Id)
