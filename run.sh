#!/bin/sh
if [ -z "$MODULE_PATH" ]; then
	echo "MODULE_PATH env var is not defined"
	exit 1
fi

echo "Running kommentaar for $MODULE_PATH"

cp -r /code /go/src/$MODULE_PATH

config=/config/kommentaar.conf
# check if we should override with env var
if [ -n "$CONFIG_NAME" ]; then
	config=/config/$CONFIG_NAME
fi
echo "Config will be loaded from $config"

exec_path=/go/src/$MODULE_PATH/...
if [ -n "$EXEC_PATH" ]; then
	exec_path=$EXEC_PATH
fi
echo "Kommentaar will be executed against $exec_path"

export GOPATH=/go
export GO111MODULE=off
go install /go/src/github.com/teamwork/kommentaar

cd /go/src/$MODULE_PATH
/go/bin/kommentaar -config $config $exec_path > /output/swagger.yaml
