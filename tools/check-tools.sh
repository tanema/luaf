#!/bin/sh
status=0
required_tools=("go" "staticcheck" "golangci-lint")

check() {
	cmd_name=$1
	if ! command -v $cmd_name 2>&1 >/dev/null; then
		echo "❌ $cmd_name not installed"
		status=1
	else
		echo "✅ $cmd_name installed"
	fi
}

for tool in "${required_tools[@]}"; do
	check $tool
done

exit $status
