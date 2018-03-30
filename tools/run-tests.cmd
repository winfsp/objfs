@echo off

pushd %~dp0..

go run tools/authtool.go oauth2 https://login.microsoftonline.com/common/oauth2/v2.0/authorize https://login.microsoftonline.com/common/oauth2/v2.0/token keyring:objfs/onedrive_client_secret ./_test.onedrive_token && ^
go test %*
del _test.onedrive_token

popd
