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

output_ext="yaml"
output="openapi2-yaml"
if [ -n "$OUTPUT" ]; then
	if [ "$OUTPUT" = "openapi2-json" ] || [ "$OUTPUT" = "openapi2-jsonindent" ]; then
		output_ext="json"
		output="$OUTPUT"
	fi

	if [ "$OUTPUT" = "html" ]; then
		output_ext="html"
		output="$OUTPUT"
	fi
fi

exec_path=/go/src/$MODULE_PATH/...
if [ -n "$EXEC_PATH" ]; then
	exec_path=$EXEC_PATH
fi
echo "Kommentaar will be executed against $exec_path"

export GOPATH=/go
export GO111MODULE=off
go install /go/src/github.com/teamwork/kommentaar

cd /go/src/$MODULE_PATH
/go/bin/kommentaar -config $config -output $output $exec_path > /output/swagger.$output_ext
