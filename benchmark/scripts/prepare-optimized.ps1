param([string]$BaseUrl = $(if ($env:BASE_URL) { $env:BASE_URL } else { 'http://localhost:8081' }), [string]$AdminUsername = $env:ADMIN_USERNAME, [string]$AdminPassword = $env:ADMIN_PASSWORD, [int]$Price = 999, [int]$Stock = 1000, [int]$ActivityStock = 100)
$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($AdminUsername) -or [string]::IsNullOrWhiteSpace($AdminPassword)) { throw 'Set ADMIN_USERNAME and ADMIN_PASSWORD first.' }
$login = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/auth/login" -ContentType 'application/json' -Body (@{ username = $AdminUsername; password = $AdminPassword } | ConvertTo-Json)
$headers = @{ Authorization = "Bearer $($login.token)" }; $suffix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$product = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/admin/products" -Headers $headers -ContentType 'application/json' -Body (@{ name = "optimized-$suffix"; description = 'k6 optimized product'; price = $Price; stock = $Stock; status = 'ON_SALE' } | ConvertTo-Json)
$start = [DateTime]::UtcNow.AddMinutes(-1).ToString('o'); $end = [DateTime]::UtcNow.AddHours(1).ToString('o')
$activity = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/admin/seckill/activities" -Headers $headers -ContentType 'application/json' -Body (@{ product_id = $product.id; seckill_price = [Math]::Max(1, $Price - 100); total_stock = $ActivityStock; start_time = $start; end_time = $end; status = 'ACTIVE' } | ConvertTo-Json)
Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/admin/seckill/activities/$($activity.id)/publish" -Headers $headers | Out-Null
Write-Output "PRODUCT_ID=$($product.id)"; Write-Output "ACTIVITY_ID=$($activity.id)"
