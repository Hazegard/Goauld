@echo off
setlocal

:: Make sure the CLIENT__COMPILE_SEED variable is set before calling garble
if not defined CLIENT__COMPILE_SEED (
echo ERROR: CLIENT__COMPILE_SEED environment variable is not set.
exit /b 1
)

:: Initialize FLAG as an empty variable
set "FLAG="

:: If TINY == "true", add -tiny
if /i "%TINY%"=="true" (
    set "FLAG=%FLAG% -tiny"
)

:: If LITERALS == "true" AND NO_LITERALS != 1, add -literals
if /i "%LITERALS%"=="true" (
    if not "%NO_LITERALS%"=="1" (
        set "FLAG=%FLAG% -literals"
    )
)

:: Run garble with accumulated flags + seed + all passed arguments
garble -debugdir=out -seed=%CLIENT__COMPILE_SEED% %FLAG% %*

endlocal
