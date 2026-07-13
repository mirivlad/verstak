@echo off
setlocal
set "ROOT=%~dp0"

if not exist "%ROOT%webview2\msedgewebview2.exe" (
  echo The bundled WebView2 runtime is missing.
  pause
  exit /b 1
)

icacls "%ROOT%webview2" /grant *S-1-15-2-2:(OI)(CI)(RX) /T /C >nul 2>&1
icacls "%ROOT%webview2" /grant *S-1-15-2-1:(OI)(CI)(RX) /T /C >nul 2>&1
start "" "%ROOT%verstak-desktop.exe" %*
