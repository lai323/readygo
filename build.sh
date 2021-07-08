
version=$(echo "$(git --no-pager tag)" | sed -n '$p')
go build -o dist/readygo-linux-$version main.go
