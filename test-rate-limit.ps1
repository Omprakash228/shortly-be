# PowerShell script for testing rate limiting
# Usage: .\test-rate-limit.ps1

$BASE_URL = "http://localhost:8080"
$API_URL = "$BASE_URL/api/v1"

Write-Host "Testing Rate Limiting..." -ForegroundColor Cyan
Write-Host "========================" -ForegroundColor Cyan
Write-Host ""

# Test 1: URL Shortening (2 req/s, burst 5)
Write-Host "Test 1: URL Shortening Rate Limit (2 req/s, burst 5)" -ForegroundColor Yellow
Write-Host "Sending 10 requests rapidly..."
for ($i = 1; $i -le 10; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$API_URL/shorten" `
            -Method POST `
            -Headers @{
                "Content-Type" = "application/json"
                "X-User-ID" = "test-user-123"
            } `
            -Body (ConvertTo-Json @{
                url = "https://example.com/test$i"
            }) `
            -ErrorAction SilentlyContinue
        
        Write-Host "Request $i`: HTTP $($response.StatusCode)" -ForegroundColor Green
    } catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            Write-Host "Request $i`: HTTP 429 (Rate Limited)" -ForegroundColor Red
        } else {
            Write-Host "Request $i`: Error - $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    Start-Sleep -Milliseconds 100
}
Write-Host ""

# Test 2: Auth endpoints (5 req/s, burst 10)
Write-Host "Test 2: Auth Rate Limit (5 req/s, burst 10)" -ForegroundColor Yellow
Write-Host "Sending 15 login requests rapidly..."
for ($i = 1; $i -le 15; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$API_URL/auth/login" `
            -Method POST `
            -Headers @{
                "Content-Type" = "application/json"
            } `
            -Body (ConvertTo-Json @{
                email = "test@example.com"
                password = "wrongpassword"
            }) `
            -ErrorAction SilentlyContinue
        
        Write-Host "Request $i`: HTTP $($response.StatusCode)" -ForegroundColor Green
    } catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            Write-Host "Request $i`: HTTP 429 (Rate Limited)" -ForegroundColor Red
        } else {
            Write-Host "Request $i`: HTTP $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
        }
    }
    Start-Sleep -Milliseconds 100
}
Write-Host ""

# Test 3: General API (10 req/s, burst 20)
Write-Host "Test 3: General API Rate Limit (10 req/s, burst 20)" -ForegroundColor Yellow
Write-Host "Sending 25 requests to /api/v1/urls..."
for ($i = 1; $i -le 25; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$API_URL/urls" `
            -Method GET `
            -Headers @{
                "X-User-ID" = "test-user-123"
            } `
            -ErrorAction SilentlyContinue
        
        Write-Host "Request $i`: HTTP $($response.StatusCode)" -ForegroundColor Green
    } catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            Write-Host "Request $i`: HTTP 429 (Rate Limited)" -ForegroundColor Red
        } else {
            Write-Host "Request $i`: HTTP $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
        }
    }
    Start-Sleep -Milliseconds 50
}
Write-Host ""

# Test 4: Redirect endpoint (30 req/s, burst 60)
Write-Host "Test 4: Redirect Rate Limit (30 req/s, burst 60)" -ForegroundColor Yellow
Write-Host "Sending 70 requests rapidly..."
for ($i = 1; $i -le 70; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$API_URL/redirect/test123" `
            -Method GET `
            -ErrorAction SilentlyContinue
        
        Write-Host "Request $i`: HTTP $($response.StatusCode)" -ForegroundColor Green
    } catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            Write-Host "Request $i`: HTTP 429 (Rate Limited)" -ForegroundColor Red
        } else {
            Write-Host "Request $i`: HTTP $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
        }
    }
    Start-Sleep -Milliseconds 10
}
Write-Host ""

Write-Host "Rate limit testing complete!" -ForegroundColor Cyan
Write-Host "Expected results:" -ForegroundColor Cyan
Write-Host "- URL Shortening: Should see 429 after ~5-7 requests"
Write-Host "- Auth: Should see 429 after ~10-12 requests"
Write-Host "- General API: Should see 429 after ~20-22 requests"
Write-Host "- Redirect: Should see 429 after ~60-65 requests"

