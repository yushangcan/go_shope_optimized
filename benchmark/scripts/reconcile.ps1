param([string]$BaseUrl = $(if ($env:BASE_URL) { $env:BASE_URL } else { 'http://localhost:8081' }), [string]$RequestId = $env:REQUEST_ID)
if ([string]::IsNullOrWhiteSpace($RequestId)) { throw 'Set REQUEST_ID to inspect final asynchronous status.' }
Invoke-RestMethod -Method Get -Uri "$BaseUrl/api/seckill/requests/$RequestId"
