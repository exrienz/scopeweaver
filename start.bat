@echo off
chcp 65001 >nul 2>&1
rem ScopeWeaver Windows supervisor. Arguments are forwarded unchanged.
rem Exit 0 stops; exit 75 applies an update and restarts immediately.
rem Other exits back off from 1 to 60 seconds.
rem The binary handles downloads, SHA-256 checks and atomic replacement.

setlocal enabledelayedexpansion
set "LANG_SEL=%SCOPEWEAVER_LANGUAGE%"
if not defined LANG_SEL set "LANG_SEL=%ARTEX_LANGUAGE%"
call :normalize_language
cd /d "%~dp0"

set "BIN=scopeweaver.exe"
if not exist "%BIN%" (
	call :message missing 1>&2
	exit /b 1
)

set "RESTART_CODE=75"
set "MAX_DELAY=60"
set /a delay=1

:loop
"%BIN%" %*
set "code=!ERRORLEVEL!"

if "!code!"=="0" (
	call :message normal
	exit /b 0
)

if "!code!"=="%RESTART_CODE%" (
	call :message restart
	set /a delay=1
	goto loop
)

call :message failed 1>&2
rem Use ping because timeout fails when console input is redirected.
set /a pings=!delay!+1
ping -n !pings! 127.0.0.1 >nul 2>&1
set /a delay=!delay!*2
if !delay! gtr %MAX_DELAY% set /a delay=%MAX_DELAY%
goto loop

:normalize_language
set "TAG=!LANG_SEL:_=-!"
:trim_leading
if "!TAG:~0,1!"==" " (
  set "TAG=!TAG:~1!"
  goto trim_leading
)
:trim_trailing
if "!TAG:~-1!"==" " (
  set "TAG=!TAG:~0,-1!"
  goto trim_trailing
)
set "LANG_SEL=en"
if /i "!TAG!"=="ko" set "LANG_SEL=ko"
if /i "!TAG:~0,3!"=="ko-" set "LANG_SEL=ko"
if /i "!TAG:~0,2!"=="ko" exit /b
if /i "!TAG!"=="en" exit /b
if /i "!TAG:~0,3!"=="en-" exit /b
if "!TAG!"=="" exit /b
if "!TAG:~-1!"=="-" exit /b
if not "!TAG:--=!"=="!TAG!" exit /b
if not "!TAG: =!"=="!TAG!" exit /b
if not "!TAG:;=!"=="!TAG!" exit /b
if /i "!TAG!"=="ja" set "LANG_SEL=ja"
if /i "!TAG:~0,3!"=="ja-" set "LANG_SEL=ja"
if /i "!TAG!"=="es" set "LANG_SEL=es"
if /i "!TAG:~0,3!"=="es-" set "LANG_SEL=es"
if not "!LANG_SEL!"=="en" exit /b
for %%L in (zh zh-CN zh-SG) do if /i "!TAG!"=="%%L" set "LANG_SEL=zh-Hans"
for %%L in (zh-TW zh-HK zh-MO) do if /i "!TAG!"=="%%L" set "LANG_SEL=zh-Hant"
if not "!LANG_SEL!"=="en" exit /b
if /i not "!TAG:~0,3!"=="zh-" exit /b
set "PARTS=!TAG:~3!"
set "SCRIPT="
:script_part
set "PART="
set "REST="
for /f "tokens=1,* delims=-" %%A in ("!PARTS!") do (
  set "PART=%%A"
  set "REST=%%B"
)
if not "!PART:~3,1!"=="" if "!PART:~4!"=="" (
  if defined SCRIPT if /i not "!SCRIPT!"=="!PART!" exit /b
  set "SCRIPT=!PART!"
)
set "PARTS=!REST!"
if defined PARTS goto script_part
if /i "!SCRIPT!"=="hans" set "LANG_SEL=zh-Hans"
if /i "!SCRIPT!"=="hant" set "LANG_SEL=zh-Hant"
exit /b

:message
if "%~1"=="missing" (
  if "!LANG_SEL!"=="ko" (echo [ScopeWeaver] 실행 파일을 찾을 수 없습니다: !BIN!) else   if "!LANG_SEL!"=="zh-Hans" (echo [ScopeWeaver] 未找到可执行文件：!BIN!) else   if "!LANG_SEL!"=="zh-Hant" (echo [ScopeWeaver] 找不到執行檔：!BIN!) else   if "!LANG_SEL!"=="ja" (echo [ScopeWeaver] 実行ファイルが見つかりません: !BIN!) else   if "!LANG_SEL!"=="es" (echo [ScopeWeaver] No se ha encontrado el ejecutable: !BIN!) else (echo [ScopeWeaver] Executable not found: !BIN!)
  exit /b
)
if "%~1"=="normal" (
  if "!LANG_SEL!"=="ko" (echo [ScopeWeaver] 정상 종료) else   if "!LANG_SEL!"=="zh-Hans" (echo [ScopeWeaver] 已正常退出) else   if "!LANG_SEL!"=="zh-Hant" (echo [ScopeWeaver] 已正常結束) else   if "!LANG_SEL!"=="ja" (echo [ScopeWeaver] 正常終了しました) else   if "!LANG_SEL!"=="es" (echo [ScopeWeaver] Finalizó normalmente) else (echo [ScopeWeaver] Exited normally)
  exit /b
)
if "%~1"=="restart" (
  if "!LANG_SEL!"=="ko" (echo [ScopeWeaver] 다시 시작 요청; 업데이트 적용 중...) else   if "!LANG_SEL!"=="zh-Hans" (echo [ScopeWeaver] 收到重启请求；正在应用更新...) else   if "!LANG_SEL!"=="zh-Hant" (echo [ScopeWeaver] 收到重新啟動要求；正在套用更新...) else   if "!LANG_SEL!"=="ja" (echo [ScopeWeaver] 再起動が要求されました。更新を適用中...) else   if "!LANG_SEL!"=="es" (echo [ScopeWeaver] Se ha solicitado reiniciar; aplicando la actualización...) else (echo [ScopeWeaver] Restart requested; applying update...)
  exit /b
)
if "%~1"=="failed" (
  if "!LANG_SEL!"=="ko" (echo [ScopeWeaver] 종료 코드 !code!; !delay!초 후 다시 시작) else   if "!LANG_SEL!"=="zh-Hans" (echo [ScopeWeaver] 退出码为 !code!；!delay! 秒后重启) else   if "!LANG_SEL!"=="zh-Hant" (echo [ScopeWeaver] 結束代碼為 !code!；!delay! 秒後重新啟動) else   if "!LANG_SEL!"=="ja" (echo [ScopeWeaver] 終了コード !code!。!delay! 秒後に再起動します) else   if "!LANG_SEL!"=="es" (echo [ScopeWeaver] Finalizó con el código !code!; se reiniciará en !delay! s) else (echo [ScopeWeaver] Exited with code !code!; restarting in !delay!s)
  exit /b
)
exit /b
