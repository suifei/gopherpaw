# 将 docs/copaw_to_gopherpaw_plan.md 中的「计划题词」块输出到控制台（UTF-8）
# 用法: .\scripts\print_plan_prompts.ps1
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$planPath = Join-Path $root "docs\copaw_to_gopherpaw_plan.md"
if (-not (Test-Path $planPath)) { Set-Location $root; $planPath = "docs\copaw_to_gopherpaw_plan.md" }
$content = Get-Content $planPath -Raw -Encoding UTF8
if ($content -match '```\r?\n(1\..*?)```') {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    $null = [Console]::OutputEncoding
    Write-Host "`n======== 计划题词（供 gopherpaw-autopilot 按条或按序执行）========`n" -ForegroundColor Cyan
    Write-Host $Matches[1].Trim()
    Write-Host "`n======== 以上共 11 条 ========`n" -ForegroundColor Cyan
} else {
    Write-Warning "未在计划文档中找到计划题词代码块。"
}
