@echo off
REM create_gopath_test.bat - 创建 GOPATH 测试项目 (Windows 版本)

setlocal

REM 设置默认值
set "GOPATH_ROOT=%~1"
if "%GOPATH_ROOT%"=="" set "GOPATH_ROOT=.\gopath_test"
set "PROJECT_PKG=%~2"
if "%PROJECT_PKG%"=="" set "PROJECT_PKG=mycompany\myproject\typealias"

echo 创建 GOPATH 测试项目...
echo GOPATH: %GOPATH_ROOT%
echo Package: %PROJECT_PKG%

REM 创建目录结构
set "PKG_DIR=%GOPATH_ROOT%\src\%PROJECT_PKG%"
mkdir "%PKG_DIR%" 2>nul

REM 创建测试文件 - 包含重定义类型的结构体
(
echo package typealias
echo.
echo // StructWithTypeAlias 包含重定义类型的结构体
echo type StructWithTypeAlias struct {
echo     ID   int64
echo     flag newType
echo     Name string
echo }
echo.
echo // newType 重定义的类型，底层是 uint8，大小应该是 1
echo type newType uint8
echo.
echo // AnotherType 另一个重定义类型
echo type AnotherType uint16
echo.
echo // StructWithAnother 包含另一种重定义类型的结构体
echo type StructWithAnother struct {
echo     X int64
echo     Y AnotherType
echo     Z bool
echo }
echo.
echo // BadStruct 未优化的结构体，包含重定义类型
echo type BadStruct struct {
echo     A bool
echo     B newType
echo     C int64
echo     D AnotherType
echo     E int32
echo }
echo.
echo // GoodStruct 优化后的结构体
echo type GoodStruct struct {
echo     C int64
echo     D AnotherType
echo     E int32
echo     A bool
echo     B newType
echo }
) > "%PKG_DIR%\typealias.go"

echo.
echo 测试项目创建完成: %PKG_DIR%
echo.
echo 目录结构:
dir /s /b "%GOPATH_ROOT%\*.go"

endlocal
