
# Build
```bash
brew install FiloSottile/musl-cross/musl-cross
szCurName=`go env GOHOSTOS`_`go env GOARCH`
go build -o $szCurName pkg/server/main.go
rm e2e_linux_amd64
# CGO_ENABLED=0 编译结果才能运行
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build -o e2e_linux_amd64 -v pkg/server/main.go
docker run --rm -it -v $PWD:/app golang:latest /bin/bash -c "cd /app/; CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags \"-linkmode external -extldflags '-static' -s -w\" -o e2e_linux_amd64 pkg/server/main.go; exit"

export rip=e2e.51pwn.com
export rport=22

rcp e2e_linux_amd64 /root/e2e/
ssh -i ~/.ssh/id_rsa -C -p ${rport} root@${rip} 
```


# server
non-BSD
```
sysctl -w net.core.rmem_max=2500000
```
BSD
```
sysctl -w kern.ipc.maxsockbuf=3014656
```
/etc/sysctl.conf
```
kern.ipc.maxsockbuf=3014656
```