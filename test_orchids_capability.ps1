# Simple test to check Orchids capabilities

Write-Host "Testing Orchids Agent capabilities..." -ForegroundColor Cyan
Write-Host ""

$payload = '{"model":"claude-opus-4-5","messages":[{"role":"user","content":"Can you generate images or videos? Please answer yes or no and explain how."}],"stream":false}'

Write-Host "Sending request..." -ForegroundColor Yellow

try {
    $response = Invoke-RestMethod -Uri "http://192.168.1.201:3002/v1/chat/completions" `
        -Method Post `
        -Body $payload `
        -ContentType "application/json" `
        -TimeoutSec 120
    
    Write-Host "Success!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Agent Response:" -ForegroundColor Cyan
    Write-Host $response.choices[0].message.content
    Write-Host ""
    
    # Check for tool use
    if ($response.choices[0].message.tool_calls) {
        Write-Host "Tool calls detected!" -ForegroundColor Yellow
        $response.choices[0].message.tool_calls | ConvertTo-Json -Depth 10
    }
    
} catch {
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "This might mean:" -ForegroundColor Yellow
    Write-Host "1. Account not configured properly" -ForegroundColor Gray
    Write-Host "2. Request timeout (agent taking too long)" -ForegroundColor Gray
    Write-Host "3. Network issue" -ForegroundColor Gray
}
