@echo off
setlocal

:: Make sure the CLIENT__COMPILE_SEED variable is set before calling garble
if not defined CLIENT__COMPILE_SEED (
echo ERROR: CLIENT__COMPILE_SEED environment variable is not set.
exit /b 1
)

:: Call garble with the specified options and the value of %CLIENT__COMPILE_SEED%
garble -literals -seed=%CLIENT__COMPILE_SEED% -tiny %*

endlocal