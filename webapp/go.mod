module cloudeng.io/webapp

go 1.16

require (
	cloudeng.io/io v0.0.0-00010101000000-000000000000
	cloudeng.io/os v0.0.0-20210416232737-f41cdfa1ea0b
	github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
)

replace cloudeng.io/os => ../os

replace cloudeng.io/io => ../io
