module system-metrics

go 1.25.2

require (
	db v0.0.0
	github.com/jackc/pgx/v5 v5.7.6
	github.com/joho/godotenv v1.5.1
	github.com/shirou/gopsutil/v4 v4.25.11
	logger v0.0.0
)

replace db => ../pkg/db

replace logger => ../pkg/logger

require (
	github.com/ebitengine/purego v0.9.1 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)
