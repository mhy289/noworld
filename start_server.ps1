$env:DB_HOST = '127.0.0.1'
$env:DB_PORT = '3306'
$env:DB_USER = 'mhy'
$env:DB_PASS = '123456'
$env:DB_NAME = 'myworld'
$env:PORT = '8080'
$proc = Start-Process -FilePath 'go' -ArgumentList 'run', '.' -WorkingDirectory 'd:\goproject\noworld\backend' -RedirectStandardOutput 'd:\goproject\noworld\server-out.log' -RedirectStandardError 'd:\goproject\noworld\server-err.log' -WindowStyle Hidden -PassThru
Write-Host ("STARTED PID=" + $proc.Id)
