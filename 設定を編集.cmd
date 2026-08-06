@echo off
chcp 65001 >nul
start "" notepad.exe "%~dp0transfer-config.json"
