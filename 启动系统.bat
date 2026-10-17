@echo off
chcp 65001 >nul
title Car-System 一键启动
powershell -ExecutionPolicy Bypass -File "%~dp0start-all.ps1"
pause
