rm e2e_linux_amd64
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build -o e2e_linux_amd64 -v pkg/server/main.go
rcp e2e_linux_amd64 /root/e2e/

